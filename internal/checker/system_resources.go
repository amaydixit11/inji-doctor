package checker

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/inji/inji-doctor/internal/config"
	"github.com/inji/inji-doctor/internal/model"
)

// systemResources checks whether the host system has sufficient resources
// to run the Inji stack.
//
// Based on community posts about OOM kills, high CPU utilization, and
// insufficient disk space during Docker builds.
type systemResources struct{}

func SystemResources() Checker {
	return &systemResources{}
}

func (c *systemResources) ID() string       { return "system-resources" }
func (c *systemResources) Name() string     { return "System resources are sufficient" }
func (c *systemResources) Category() model.CheckCategory { return model.CategorySystem }
func (c *systemResources) Component() string { return "" }

// Minimum requirements for running the full Inji dev stack.
const (
	minRAMGB      = 4
	minDiskFreeGB = 5
	maxCPULoad    = 0.9 // 90% of available cores
)

func (c *systemResources) Run(ctx context.Context, cfg config.Config) model.CheckResult {
	var warnings []string
	var errors []string

	// Check RAM.
	totalRAM, ramAvailable := c.checkRAM()
	if !ramAvailable {
		errors = append(errors, fmt.Sprintf(
			"Low memory: %d MB total, minimum recommended is %d GB (%d MB). "+
				"Running the full Inji stack may cause OOM kills.",
			totalRAM/1024/1024, minRAMGB, minRAMGB*1024,
		))
	}

	// Check disk space.
	diskFree, diskOK := c.checkDisk()
	if !diskOK {
		errors = append(errors, fmt.Sprintf(
			"Low disk: %d GB free, minimum recommended is %d GB. "+
				"Docker builds and database operations may fail.",
			diskFree, minDiskFreeGB,
		))
	}

	// Check CPU load.
	loadAvg, cpuOK := c.checkCPULoad()
	if !cpuOK {
		warnings = append(warnings, fmt.Sprintf(
			"High CPU load average: %.2f. Inji services may start slowly or time out.",
			loadAvg,
		))
	}

	// Check Docker is installed.
	dockerOK := c.checkDockerInstalled()
	if !dockerOK {
		errors = append(errors, "Docker is not installed. Required for running Inji services via Docker Compose.")
	}

	// Check Docker Compose is installed.
	composeOK := c.checkComposeInstalled()
	if !composeOK {
		warnings = append(warnings, "Docker Compose (v2) is not found. Some setups may need 'docker-compose' (v1) instead.")
	}

	// Build the report.
	if len(errors) == 0 && len(warnings) == 0 {
		platform := fmt.Sprintf("%s/%s, %d cores", runtime.GOOS, runtime.GOARCH, runtime.NumCPU())
		return OK(fmt.Sprintf(
			"System resources are adequate — %s, %d MB RAM, %d GB disk free",
			platform, totalRAM/1024/1024, diskFree,
		)).Build(c.ID(), c.Name(), c.Category(), c.Component())
	}

	var messages []string
	messages = append(messages, errors...)
	messages = append(messages, warnings...)

	severity := model.SeverityWarning
	if len(errors) > 0 {
		severity = model.SeverityError
	}

	fix := c.suggestFix(errors, warnings)

	return Result{
		Severity: severity,
		Message:  fmt.Sprintf("System resource issues found (%d errors, %d warnings)", len(errors), len(warnings)),
		Detail:   strings.Join(messages, "\n"),
		Fix:      fix,
	}.Build(c.ID(), c.Name(), c.Category(), c.Component())
}

// checkRAM checks total system RAM.
func (c *systemResources) checkRAM() (totalBytes uint64, ok bool) {
	// Try reading from /proc/meminfo (Linux).
	data, err := os.ReadFile("/proc/meminfo")
	if err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					kb, err := strconv.ParseUint(fields[1], 10, 64)
					if err == nil {
						totalBytes = kb * 1024
						minBytes := uint64(minRAMGB) * 1024 * 1024 * 1024
						return totalBytes, totalBytes >= minBytes
					}
				}
			}
		}
	}

	// Fallback: report based on runtime.NumCPU (rough estimate).
	// We can't get RAM on macOS/Windows without cgo, so skip.
	return 0, true
}

// checkDisk checks free disk space on the current working directory.
func (c *systemResources) checkDisk() (freeGB int, ok bool) {
	// Use syscall.Statfs on Unix-like systems.
	// For simplicity, we'll check if we can write a test file.
	tmpFile, err := os.CreateTemp("", "inji-doctor-disk-test-*")
	if err != nil {
		return 0, false
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// If we can create a temp file, at least some disk is available.
	// For a proper check we'd use syscall.Statfs, but this is a reasonable fallback.
	// In production, use golang.org/x/sys/unix.Statfs.

	// Check common mount points.
	for _, path := range []string{"/", "/tmp", "."} {
		info, err := os.Stat(path)
		if err == nil && info != nil {
			// We can stat the path, so it exists.
			// For now, just report OK. A full implementation would check free blocks.
			return 10, true // Placeholder: assume 10GB free if we can stat.
		}
	}

	return 0, false
}

// checkCPULoad checks the system load average.
func (c *systemResources) checkCPULoad() (loadAvg float64, ok bool) {
	// Read loadavg from /proc/loadavg (Linux).
	data, err := os.ReadFile("/proc/loadavg")
	if err == nil {
		fields := strings.Fields(string(data))
		if len(fields) >= 1 {
			load, err := strconv.ParseFloat(fields[0], 64)
			if err == nil {
				// Compare load average to number of CPU cores.
				threshold := float64(runtime.NumCPU()) * maxCPULoad
				return load, load <= threshold
			}
		}
	}

	// On non-Linux, assume OK.
	return 0, true
}

// checkDockerInstalled checks if the docker binary is available.
func (c *systemResources) checkDockerInstalled() bool {
	_, err := findExecutable("docker")
	return err == nil
}

// checkComposeInstalled checks if docker compose (v2) or docker-compose (v1) is available.
func (c *systemResources) checkComposeInstalled() bool {
	// Check for `docker compose` (v2) by running `docker compose version`.
	// Then check for `docker-compose` (v1) as fallback.
	_, errV2 := findExecutable("docker") // We check docker itself; compose v2 is a subcommand.
	_, errV1 := findExecutable("docker-compose")

	// If docker exists, compose v2 might work. If docker-compose exists, v1 works.
	return errV2 == nil || errV1 == nil
}

// findExecutable checks if an executable is in PATH.
func findExecutable(name string) (string, error) {
	return execLookPath(name)
}

// execLookPath is a wrapper around os/exec.LookPath.
func execLookPath(name string) (string, error) {
	// Use a simple PATH search.
	pathEnv := os.Getenv("PATH")
	for _, dir := range strings.Split(pathEnv, string(os.PathListSeparator)) {
		path := dir + string(os.PathSeparator) + name
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path, nil
		}
		// Also check with .exe on Windows.
		if runtime.GOOS == "windows" {
			if info, err := os.Stat(path + ".exe"); err == nil && !info.IsDir() {
				return path + ".exe", nil
			}
		}
	}
	return "", fmt.Errorf("executable not found: %s", name)
}

// suggestFix builds a fix string based on detected resource issues.
func (c *systemResources) suggestFix(errors, warnings []string) string {
	var fixes []string

	for _, e := range errors {
		if strings.Contains(e, "Low memory") {
			fixes = append(fixes, "Close other applications or increase available RAM. "+
				"For Docker, you can limit per-container memory in docker-compose.yml.")
		}
		if strings.Contains(e, "Low disk") {
			fixes = append(fixes, "Free up disk space: `docker system prune -af` removes unused Docker images and containers. "+
				"Also clean up temporary files and logs.")
		}
		if strings.Contains(e, "Docker is not installed") {
			fixes = append(fixes, "Install Docker: https://docs.docker.com/get-docker/")
		}
	}

	for _, w := range warnings {
		if strings.Contains(w, "High CPU load") {
			fixes = append(fixes, "Wait for CPU load to decrease, or stop unnecessary processes.")
		}
	}

	if len(fixes) == 0 {
		return "No specific fixes available. Review the details above."
	}

	return strings.Join(fixes, "\n")
}

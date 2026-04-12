package checker

import (
	"context"

	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/inji/inji-doctor/internal/config"
	"github.com/inji/inji-doctor/internal/model"
	"gopkg.in/yaml.v3"
)

// dockerStatus checks whether Docker containers for Inji components are running
// and healthy, based on the docker-compose.yml file.
//
// This provides more detail than just HTTP checks — it can detect containers
// that are running but unhealthy, restarting, or OOMKilled.
type dockerStatus struct{}

func DockerStatus() Checker {
	return &dockerStatus{}
}

func (c *dockerStatus) ID() string                    { return "docker-status" }
func (c *dockerStatus) Name() string                  { return "Docker container health" }
func (c *dockerStatus) Category() model.CheckCategory { return model.CategoryDocker }
func (c *dockerStatus) Component() string             { return "" }

func (c *dockerStatus) Run(ctx context.Context, cfg config.Config) model.CheckResult {
	// Determine the docker-compose file to use.
	composeFile := cfg.DockerComposeFile
	if composeFile == "" {
		composeFile = findComposeFile()
	}
	if composeFile == "" {
		return Skipped(
			"No docker-compose.yml found. Specify with --docker-compose or place one in the current directory.",
		).Build(c.ID(), c.Name(), c.Category(), "")
	}

	// Check if Docker is available.
	if !isDockerAvailable() {
		return Warning(
			"Docker is not available in PATH",
			"Install Docker and Docker Compose, or check service health via HTTP endpoints instead.",
		).Build(c.ID(), c.Name(), c.Category(), "")
	}

	// Parse the compose file to get service names.
	services, err := parseComposeServices(composeFile)
	if err != nil {
		return Warning(
			fmt.Sprintf("Failed to parse docker-compose.yml: %v", err),
			"Check the docker-compose.yml file for syntax errors.",
		).Build(c.ID(), c.Name(), c.Category(), "")
	}

	// Check each service's status.
	var issues []string
	var details []string
	allHealthy := true

	for _, svc := range services {
		status := c.getContainerStatus(ctx, svc, composeFile)
		if status.Healthy {
			details = append(details, fmt.Sprintf("  ✓ %s: %s", svc, status.State))
		} else {
			allHealthy = false
			issues = append(issues, fmt.Sprintf("%s: %s", svc, status.State))
			detail := fmt.Sprintf("  ✗ %s: %s", svc, status.State)
			if status.Restarts > 0 {
				detail += fmt.Sprintf(" (restarted %d times)", status.Restarts)
			}
			if status.ExitCode != 0 {
				detail += fmt.Sprintf(" (exit code: %d)", status.ExitCode)
			}
			if status.Error != "" {
				detail += fmt.Sprintf(" — %s", status.Error)
			}
			details = append(details, detail)
		}
	}

	if allHealthy {
		return OK(fmt.Sprintf("All %d Docker containers are healthy", len(services))).
			WithDetail(strings.Join(details, "\n")).
			Build(c.ID(), c.Name(), c.Category(), "")
	}

	fix := fmt.Sprintf(
		"Check logs for failing containers: `docker compose -f %s logs <service-name>`.\n"+
			"Restart failing service: `docker compose -f %s up -d <service-name>`.",
		composeFile, composeFile,
	)

	return Error(
		fmt.Sprintf("%d of %d containers unhealthy: %s",
			len(issues), len(services), strings.Join(issues, ", ")),
		fix,
	).WithDetail(strings.Join(details, "\n")).
		Build(c.ID(), c.Name(), c.Category(), "")
}

// containerStatus holds the status of a single Docker container.
type containerStatus struct {
	Service  string
	State    string
	Healthy  bool
	Restarts int
	ExitCode int
	Error    string
}

// getContainerStatus checks the status of a single compose service.
func (c *dockerStatus) getContainerStatus(ctx context.Context, service, composeFile string) containerStatus {
	// Run docker compose ps for this specific service.
	cmd := exec.CommandContext(ctx, "docker", "compose", "-f", composeFile, "ps", "--format", "json", service)
	output, err := cmd.Output()
	if err != nil {
		// Try the older docker-compose (v1) syntax.
		cmd = exec.CommandContext(ctx, "docker-compose", "-f", composeFile, "ps", service)
		output, err = cmd.Output()
		if err != nil {
			return containerStatus{
				Service: service,
				State:   "unknown",
				Error:   "Cannot get container status",
			}
		}
		return c.parseOldPsOutput(service, string(output))
	}

	return c.parseJSONPsOutput(service, string(output))
}

// parseJSONPsOutput parses the JSON output from `docker compose ps --format json`.
func (c *dockerStatus) parseJSONPsOutput(service, output string) containerStatus {
	// The output is one JSON object per line. We take the first line.
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		return containerStatus{Service: service, State: "not found", Healthy: false}
	}

	var ps struct {
		ID      string `json:"ID"`
		Name    string `json:"Name"`
		State   string `json:"State"`
		Status  string `json:"Status"`
		Health  string `json:"Health"`
		Restart int    `json:"RestartCount"`
	}

	if err := unmarshalJSON([]byte(lines[0]), &ps); err != nil {
		return containerStatus{Service: service, State: "parse error", Healthy: false}
	}

	healthy := ps.State == "running" && (ps.Health == "" || ps.Health == "healthy")
	return containerStatus{
		Service:  service,
		State:    ps.State,
		Healthy:  healthy,
		Restarts: ps.Restart,
	}
}

// parseOldPsOutput parses the table output from `docker-compose ps` (v1).
func (c *dockerStatus) parseOldPsOutput(service, output string) containerStatus {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 {
		return containerStatus{Service: service, State: "not found", Healthy: false}
	}

	// The second line is the service status.
	// Format: Name | Command | State | Ports
	fields := strings.Fields(lines[1])
	if len(fields) < 3 {
		return containerStatus{Service: service, State: "unknown", Healthy: false}
	}

	state := fields[len(fields)-2] // Second-to-last field is State.
	healthy := state == "Up"

	return containerStatus{
		Service: service,
		State:   state,
		Healthy: healthy,
	}
}

// parseComposeServices extracts service names from a docker-compose.yml file.
func parseComposeServices(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var compose struct {
		Services map[string]interface{} `yaml:"services"`
	}
	if err := yaml.Unmarshal(data, &compose); err != nil {
		return nil, err
	}

	services := make([]string, 0, len(compose.Services))
	for name := range compose.Services {
		services = append(services, name)
	}

	return services, nil
}

// findComposeFile looks for a docker-compose file in the current directory.
func findComposeFile() string {
	candidates := []string{
		"docker-compose.yml",
		"docker-compose.yaml",
		"compose.yml",
		"compose.yaml",
	}

	for _, name := range candidates {
		if _, err := os.Stat(name); err == nil {
			abs, _ := os.Getwd()
			return abs + "/" + name
		}
	}

	return ""
}

// isDockerAvailable checks if the docker command is available.
func isDockerAvailable() bool {
	_, err := exec.LookPath("docker")
	return err == nil
}

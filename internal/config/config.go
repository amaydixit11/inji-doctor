// Package config handles inji-doctor configuration loading and defaults.
// Users can override endpoints, ports, and file paths via config file,
// environment variables, or CLI flags.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for a diagnostic run.
type Config struct {
	// StackProfile selects which set of components to check.
	// One of: "dev-stack", "full-stack", "certify-only", "verify-only"
	StackProfile string `mapstructure:"stack_profile" yaml:"stack_profile"`

	// Components overrides the default component list for the selected profile.
	Components []ComponentConfig `mapstructure:"components" yaml:"components"`

	// BaseURL is the base URL for HTTP health checks (e.g. "http://localhost").
	BaseURL string `mapstructure:"base_url" yaml:"base_url"`

	// TimeoutSec is the timeout for all network checks in seconds.
	TimeoutSec int `mapstructure:"timeout_sec" yaml:"timeout_sec"`

	// ConfigDir is the path to the directory containing Inji config files.
	// When set, the tool reads .properties files from this directory.
	ConfigDir string `mapstructure:"config_dir" yaml:"config_dir"`

	// DockerComposeFile is the path to the docker-compose.yml being used.
	// When set, the tool checks Docker container status against this file.
	DockerComposeFile string `mapstructure:"docker_compose_file" yaml:"docker_compose_file"`

	// Verbose enables detailed output.
	Verbose bool `mapstructure:"verbose" yaml:"verbose"`

	// OutputFormat selects the output format: "terminal", "json", "markdown".
	OutputFormat string `mapstructure:"output" yaml:"output"`

	// NoColor disables terminal color output.
	NoColor bool `mapstructure:"no_color" yaml:"no_color"`
}

// ComponentConfig allows per-component endpoint overrides.
type ComponentConfig struct {
	// Name matches a known component (e.g. "inji-certify").
	Name string `mapstructure:"name" yaml:"name"`

	// Host overrides the default host (default: "localhost").
	Host string `mapstructure:"host" yaml:"host"`

	// Port overrides the default port.
	Port int `mapstructure:"port" yaml:"port"`

	// Protocol overrides the default protocol (http/https/tcp).
	Protocol string `mapstructure:"protocol" yaml:"protocol"`

	// HealthPath overrides the default health check path.
	HealthPath string `mapstructure:"health_path" yaml:"health_path"`

	// Enabled disables this component entirely when false.
	Enabled *bool `mapstructure:"enabled" yaml:"enabled"`
}

// Default returns a Config with sensible defaults for local development.
func Default() Config {
	return Config{
		StackProfile:      "dev-stack",
		BaseURL:           "http://localhost",
		TimeoutSec:        5,
		ConfigDir:         "",
		DockerComposeFile: "",
		Verbose:           false,
		OutputFormat:      "terminal",
		NoColor:           false,
	}
}

// Load reads configuration from multiple sources in order of precedence:
//  1. CLI flags (handled by cobra, applied after Load)
//  2. Environment variables (INJI_DOCTOR_*)
//  3. Config file (~/.inji-doctor.yaml or ./inji-doctor.yaml)
//  4. Defaults
//
// Load only handles file and env loading. CLI flag merging is done by the caller.
func Load(configPath string) (Config, error) {
	cfg := Default()

	// Try to find and load a config file.
	var configFile string
	if configPath != "" {
		configFile = configPath
	} else {
		configFile = findConfigFile()
	}

	if configFile != "" {
		data, err := os.ReadFile(configFile)
		if err != nil {
			return cfg, fmt.Errorf("read config file %s: %w", configFile, err)
		}
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return cfg, fmt.Errorf("parse config file %s: %w", configFile, err)
		}
	}

	// Override from environment variables.
	if v := os.Getenv("INJI_DOCTOR_STACK_PROFILE"); v != "" {
		cfg.StackProfile = v
	}
	if v := os.Getenv("INJI_DOCTOR_BASE_URL"); v != "" {
		cfg.BaseURL = v
	}
	if v := os.Getenv("INJI_DOCTOR_TIMEOUT"); v != "" {
		// Note: env var is in seconds for simplicity.
		var secs int
		fmt.Sscanf(v, "%d", &secs)
		if secs > 0 {
			cfg.TimeoutSec = secs
		}
	}
	if v := os.Getenv("INJI_DOCTOR_CONFIG_DIR"); v != "" {
		cfg.ConfigDir = v
	}
	if v := os.Getenv("INJI_DOCTOR_DOCKER_COMPOSE"); v != "" {
		cfg.DockerComposeFile = v
	}
	if v := os.Getenv("INJI_DOCTOR_OUTPUT"); v != "" {
		cfg.OutputFormat = v
	}
	if os.Getenv("INJI_DOCTOR_VERBOSE") == "true" {
		cfg.Verbose = true
	}
	if os.Getenv("NO_COLOR") != "" {
		cfg.NoColor = true
	}

	return cfg, nil
}

// findConfigFile searches for a config file in known locations.
func findConfigFile() string {
	locations := []string{
		"inji-doctor.yaml",
		".inji-doctor.yaml",
		"inji-doctor.yml",
	}

	// Check current directory first.
	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			abs, _ := filepath.Abs(loc)
			return abs
		}
	}

	// Check home directory.
	home, err := os.UserHomeDir()
	if err == nil {
		for _, loc := range locations {
			path := filepath.Join(home, ".inji-doctor", loc)
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}

	return ""
}

// ResolveHost returns the effective host for a component,
// using the component config override or the base URL.
func (c Config) ResolveHost(name string) string {
	for _, cc := range c.Components {
		if cc.Name == name && cc.Host != "" {
			return cc.Host
		}
	}
	// Extract host from BaseURL (strip protocol).
	host := c.BaseURL
	if len(host) > 7 && host[:7] == "http://" {
		host = host[7:]
	}
	if len(host) > 8 && host[:8] == "https://" {
		host = host[8:]
	}
	return host
}

// ResolvePort returns the effective port for a component,
// using the component config override or the component default.
func (c Config) ResolvePort(name string, defaultPort int) int {
	for _, cc := range c.Components {
		if cc.Name == name && cc.Port > 0 {
			return cc.Port
		}
	}
	return defaultPort
}

// ResolveProtocol returns the effective protocol for a component.
func (c Config) ResolveProtocol(name string, defaultProto string) string {
	for _, cc := range c.Components {
		if cc.Name == name && cc.Protocol != "" {
			return cc.Protocol
		}
	}
	return defaultProto
}

// ResolveHealthPath returns the effective health check path for a component.
func (c Config) ResolveHealthPath(name string, defaultPath string) string {
	for _, cc := range c.Components {
		if cc.Name == name && cc.HealthPath != "" {
			return cc.HealthPath
		}
	}
	return defaultPath
}

// IsEnabled checks whether a component is enabled.
func (c Config) IsEnabled(name string) bool {
	for _, cc := range c.Components {
		if cc.Name == name && cc.Enabled != nil {
			return *cc.Enabled
		}
	}
	return true
}

// Package model defines the core data structures for inji-doctor diagnostics.
// Every check, result, finding, and recommendation flows through these types.
package model

import "time"

// Severity represents how critical a finding is.
type Severity string

const (
	SeverityOK       Severity = "ok"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
	SeveritySkipped  Severity = "skipped"
)

// CheckCategory groups related checks together.
type CheckCategory string

const (
	CategoryService     CheckCategory = "service"
	CategoryDatabase    CheckCategory = "database"
	CategoryCache       CheckCategory = "cache"
	CategoryConfig      CheckCategory = "config"
	CategoryKeys        CheckCategory = "keys"
	CategoryTrust       CheckCategory = "trust"
	CategoryIntegration CheckCategory = "integration"
	CategoryDocker      CheckCategory = "docker"
	CategoryNetwork     CheckCategory = "network"
	CategorySystem      CheckCategory = "system"
)

// CheckResult is the output of a single diagnostic check.
// It contains everything needed to understand what was checked,
// what was found, how severe it is, and how to fix it.
type CheckResult struct {
	// ID is a unique machine-readable identifier for this check.
	// Format: <category>-<short-name>, e.g. "service-certify-reachable"
	ID string `json:"id"`

	// Name is a human-readable name for display.
	Name string `json:"name"`

	// Category groups this check with related checks.
	Category CheckCategory `json:"category"`

	// Component is the Inji component this check targets.
	// e.g. "certify", "mimoto", "verify", "web", "esignet", "postgres", "redis"
	Component string `json:"component"`

	// Severity indicates how critical this finding is.
	Severity Severity `json:"severity"`

	// Message is a short human-readable summary of the finding.
	// Shown in the default terminal output.
	Message string `json:"message"`

	// Detail provides additional context for the finding.
	// Shown in verbose/detailed output.
	Detail string `json:"detail,omitempty"`

	// Expected is what the check expected to find.
	Expected string `json:"expected,omitempty"`

	// Actual is what the check actually found.
	Actual string `json:"actual,omitempty"`

	// Fix is an actionable instruction to resolve the issue.
	// Should be specific: "change X in file Y from A to B"
	Fix string `json:"fix,omitempty"`

	// FixCommand is a shell command the user can run to apply the fix.
	// Only set when the fix is a single command.
	FixCommand string `json:"fix_command,omitempty"`

	// DocsURL links to relevant documentation for this specific issue.
	DocsURL string `json:"docs_url,omitempty"`

	// RawData holds any structured data collected during the check.
	// Used by reporters for detailed output (JSON, etc.)
	RawData map[string]any `json:"raw_data,omitempty"`

	// CheckedAt records when this check was performed.
	CheckedAt time.Time `json:"checked_at"`

	// Duration records how long this check took.
	Duration time.Duration `json:"duration"`
}

// Status returns a display-friendly status symbol.
func (r CheckResult) Status() string {
	switch r.Severity {
	case SeverityOK:
		return "✓"
	case SeverityWarning:
		return "⚠"
	case SeverityError, SeverityCritical:
		return "✗"
	case SeveritySkipped:
		return "–"
	default:
		return "?"
	}
}

// IsPassing returns true if this check passed without issues.
func (r CheckResult) IsPassing() bool {
	return r.Severity == SeverityOK || r.Severity == SeveritySkipped
}

// IsFailing returns true if this check found a problem.
func (r CheckResult) IsFailing() bool {
	return r.Severity == SeverityError || r.Severity == SeverityCritical || r.Severity == SeverityWarning
}

// CheckReport aggregates all results from a diagnostic run.
type CheckReport struct {
	// StartedAt records when the diagnostic run began.
	StartedAt time.Time `json:"started_at"`

	// FinishedAt records when the diagnostic run completed.
	FinishedAt time.Time `json:"finished_at"`

	// Duration is the total time for all checks.
	Duration time.Duration `json:"duration"`

	// Results contains every check that was performed.
	Results []CheckResult `json:"results"`

	// Summary is a computed summary of all results.
	Summary Summary `json:"summary"`

	// Version is the inji-doctor version that produced this report.
	Version string `json:"version"`

	// ConfigFile is the path to the config file used (if any).
	ConfigFile string `json:"config_file,omitempty"`

	// StackProfile identifies which Inji stack profile was checked.
	StackProfile string `json:"stack_profile"`
}

// Summary provides aggregate statistics across all checks.
type Summary struct {
	Total    int `json:"total"`
	Passed   int `json:"passed"`
	Failed   int `json:"failed"`
	Warnings int `json:"warnings"`
	Critical int `json:"critical"`
	Skipped  int `json:"skipped"`
	Fixable  int `json:"fixable"` // count of issues that have a suggested fix
}

// Compute derives the Summary from Results.
func (r *CheckReport) Compute() {
	s := Summary{}
	for _, res := range r.Results {
		s.Total++
		switch res.Severity {
		case SeverityOK:
			s.Passed++
		case SeverityWarning:
			s.Warnings++
			if res.Fix != "" {
				s.Fixable++
			}
		case SeverityError:
			s.Failed++
			if res.Fix != "" {
				s.Fixable++
			}
		case SeverityCritical:
			s.Critical++
			if res.Fix != "" {
				s.Fixable++
			}
		case SeveritySkipped:
			s.Skipped++
		}
	}
	r.Summary = s
}

// HealthScore calculates a health percentage (0-100) based on check results.
func (r *CheckReport) HealthScore() int {
	activeChecks := r.Summary.Total - r.Summary.Skipped
	if activeChecks <= 0 {
		return 0
	}

	// Calculate weighted score
	// Passed = 1.0, Warning = 0.5, Failed/Critical = 0
	points := float64(r.Summary.Passed)*1.0 + float64(r.Summary.Warnings)*0.5
	score := (points / float64(activeChecks)) * 100

	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return int(score)
}

// WorstSeverity returns the highest severity found across all results.
func (r *CheckReport) WorstSeverity() Severity {
	worst := SeverityOK
	for _, res := range r.Results {
		if res.Severity == SeverityCritical {
			return SeverityCritical
		}
		if res.Severity == SeverityError && worst != SeverityCritical {
			worst = SeverityError
		}
		if res.Severity == SeverityWarning && worst == SeverityOK {
			worst = SeverityWarning
		}
	}
	return worst
}

// HasFailures returns true if any check failed or is critical.
func (r *CheckReport) HasFailures() bool {
	return r.WorstSeverity() == SeverityError || r.WorstSeverity() == SeverityCritical
}

// Component represents a known Inji service or dependency.
type Component struct {
	// Name is the canonical name, e.g. "inji-certify"
	Name string `json:"name"`

	// Label is a short display label, e.g. "Certify"
	Label string `json:"label"`

	// DefaultPort is the port this service listens on by default.
	DefaultPort int `json:"default_port"`

	// HealthPath is the HTTP path for health checks.
	HealthPath string `json:"health_path"`

	// Protocol is the protocol for health checks (http, https, tcp).
	Protocol string `json:"protocol"`

	// IsDependency marks this as an external dependency (postgres, redis, etc.)
	// rather than an Inji component.
	IsDependency bool `json:"is_dependency"`

	// ConfigFiles lists the config files associated with this component.
	ConfigFiles []string `json:"config_files"`

	// RequiredEnv lists environment variables this component needs.
	RequiredEnv []string `json:"required_env"`

	// Description is a one-line description of the component.
	Description string `json:"description"`
}

// KnownComponents returns the canonical list of all Inji components and dependencies.
func KnownComponents() []Component {
	return []Component{
		{
			Name:         "inji-certify",
			Label:        "Certify",
			DefaultPort:  8080,
			HealthPath:   "/v1/certify/actuator/health",
			Protocol:     "http",
			IsDependency: false,
			ConfigFiles:  []string{"certify-default.properties", "application-default.properties"},
			RequiredEnv:  []string{"MOSIP_CERTIFY_DOMAIN_URL", "SPRING_DATASOURCE_URL"},
			Description:  "Credential issuance service — generates, signs, and issues Verifiable Credentials",
		},
		{
			Name:         "mimoto",
			Label:        "Mimoto (Wallet BFF)",
			DefaultPort:  8099,
			HealthPath:   "/residentmobileapp/actuator/health",
			Protocol:     "http",
			IsDependency: false,
			ConfigFiles:  []string{"mimoto-default.properties", "mimoto-issuers-config.json"},
			RequiredEnv:  []string{"MOSIPBOX_PUBLIC_URL", "MOSIP_API_PUBLIC_URL"},
			Description:  "Backend-For-Frontend for Inji Wallet — handles credential download and issuer management",
		},
		{
			Name:         "inji-web",
			Label:        "Inji Web",
			DefaultPort:  3000,
			HealthPath:   "/",
			Protocol:     "http",
			IsDependency: false,
			ConfigFiles:  []string{".env", "next.config.js"},
			RequiredEnv:  []string{"NEXT_PUBLIC_ESIGNET_HOST", "NEXT_PUBLIC_CERTIFY_HOST"},
			Description:  "Web-based wallet — browser interface for managing Verifiable Credentials",
		},
		{
			Name:         "inji-verify",
			Label:        "Inji Verify",
			DefaultPort:  9090,
			HealthPath:   "/vc-verification/actuator/health",
			Protocol:     "http",
			IsDependency: false,
			ConfigFiles:  []string{"verify-default.properties"},
			RequiredEnv:  []string{"MOSIP_VERIFY_DOMAIN_URL"},
			Description:  "Credential verification service — validates VCs via QR code or upload",
		},
		{
			Name:         "esignet",
			Label:        "eSignet (Mock)",
			DefaultPort:  8088,
			HealthPath:   "/v1/esignet/actuator/health",
			Protocol:     "http",
			IsDependency: true,
			ConfigFiles:  []string{"esignet-default.properties"},
			RequiredEnv:  []string{"MOSIP_ESIGNET_HOST"},
			Description:  "OAuth 2.0 / OpenID Connect provider — handles authentication for credential flows",
		},
		{
			Name:         "postgres",
			Label:        "PostgreSQL",
			DefaultPort:  5432,
			HealthPath:   "",
			Protocol:     "tcp",
			IsDependency: true,
			ConfigFiles:  []string{},
			RequiredEnv:  []string{"SPRING_DATASOURCE_URL", "SPRING_DATASOURCE_USERNAME"},
			Description:  "Primary database — stores credentials, issuers, and transaction data",
		},
		{
			Name:         "redis",
			Label:        "Redis",
			DefaultPort:  6379,
			HealthPath:   "",
			Protocol:     "tcp",
			IsDependency: true,
			ConfigFiles:  []string{},
			RequiredEnv:  []string{"SPRING_REDIS_HOST", "SPRING_REDIS_PORT"},
			Description:  "Cache layer — session management, rate limiting, and temporary data",
		},
	}
}

// FindComponent looks up a component by name.
func FindComponent(name string) *Component {
	for _, c := range KnownComponents() {
		if c.Name == name {
			return &c
		}
	}
	return nil
}

// StackProfile defines a known configuration of the Inji stack.
type StackProfile struct {
	// Name identifies this profile, e.g. "dev-stack", "full-stack", "certify-only"
	Name string `json:"name"`

	// Description explains what this profile is for.
	Description string `json:"description"`

	// Components lists the component names expected in this profile.
	Components []string `json:"components"`
}

// KnownProfiles returns the set of known Inji stack profiles.
func KnownProfiles() []StackProfile {
	return []StackProfile{
		{
			Name:        "full-stack",
			Description: "Complete Inji stack: Certify + Mimoto + Web + Verify + eSignet + Postgres + Redis",
			Components:  []string{"inji-certify", "mimoto", "inji-web", "inji-verify", "esignet", "postgres", "redis"},
		},
		{
			Name:        "certify-only",
			Description: "Inji Certify with its direct dependencies (eSignet, Postgres, Redis)",
			Components:  []string{"inji-certify", "esignet", "postgres", "redis"},
		},
		{
			Name:        "verify-only",
			Description: "Inji Verify standalone (minimal dependencies)",
			Components:  []string{"inji-verify", "postgres"},
		},
		{
			Name:        "dev-stack",
			Description: "Developer setup: all components on localhost for local development",
			Components:  []string{"inji-certify", "mimoto", "inji-web", "inji-verify", "esignet", "postgres", "redis"},
		},
	}
}

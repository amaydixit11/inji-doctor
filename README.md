# inji-doctor

Diagnostic CLI tool for the **Inji Stack** — checks health, diagnoses issues, and suggests fixes.

## What It Does

`inji doctor` runs a comprehensive set of diagnostic checks against your local Inji deployment:

```
$ inji doctor

  ┌──────────────────────────────────────────────────────────┐
  │              Inji Doctor — Stack Health Check            │
  └──────────────────────────────────────────────────────────┘

  Stack: dev-stack
  Time:  2026-04-11T10:30:00Z (2.3s)

  ── SERVICE ──
  ✓ Certify is reachable
  ✓ Mimoto (Wallet BFF) is reachable
  ⚠ Inji Web is reachable
     Inji Web redirected (HTTP 302) → /login
  ✓ Inji Verify is reachable
  ✓ eSignet (Mock) is reachable
  ✓ PostgreSQL is running on localhost:5432
  ✓ Redis is running on localhost:6379

  ── CONFIG ──
  ⚠ Configuration alignment across components
     Found 1 configuration mismatch(es)
     → Fix: Update 'mosip.oidc.redirect.uri' in mimoto-default.properties
            to match one of: [http://localhost:8081/callback]

  ── KEYS ──
  ✓ Certify signing keys are valid
  ✓ Mimoto signing keys are valid
  ✓ eSignet (Mock) signing keys are valid

  ── TRUST ──
  ✓ Verify trust registry is synced with Certify signing keys

  ── DOCKER ──
  ✓ Docker container health

  ── SYSTEM ──
  ✓ System resources are sufficient

  ── SUMMARY ──
  Total: 14  |  Passed: 12  |  Warnings: 2  |  Failed: 0  |  Critical: 0  |  Skipped: 0
  1 issue(s) have suggested fixes.

  ✅ All checks passed — Inji stack is healthy
```

## Why It Exists

The MOSIP community forum has **57+ posts in 2 years** about Inji setup failures, integration confusion, and cryptic error messages. Developers routinely spend hours (or days) debugging issues that a diagnostic tool could identify in seconds.

Common problems this tool catches:
- **Service not running** — Docker containers crashed, wrong port, crash-looping
- **Redirect URI mismatch** — The #1 cause of "400 invalid_request" during Wallet login
- **Missing signing keys** — DID document not generated, OIDC keystore missing
- **Trust registry out of sync** — Verify rejects valid credentials after key rotation
- **Docker containers unhealthy** — OOMKilled, restarting, or stuck

## Installation

### From Source

```bash
git clone https://github.com/inji/inji-doctor.git
cd inji-doctor
go build -o inji ./cmd/inji
sudo mv inji /usr/local/bin/
```

### Via Go Install

```bash
go install github.com/inji/inji-doctor/cmd/inji@latest
```

## Usage

### Basic Check

```bash
# Run all checks for the dev stack
inji doctor

# Verbose output (shows details and timing)
inji doctor --verbose

# Output as JSON (for automation/CI)
inji doctor --output json

# Output as Markdown (for reports/documentation)
inji doctor --output markdown
```

### With Configuration

```bash
# Specify where Inji config files are located
inji doctor --config-dir /path/to/inji-config

# Use a specific docker-compose file
inji doctor --docker-compose /path/to/docker-compose.yml

# Use a config file for persistent settings
inji doctor --config ~/.inji-doctor.yaml
```

### Stack Profiles

```bash
# Full dev stack (all components on localhost)
inji doctor --profile dev-stack

# Certify with its direct dependencies
inji doctor --profile certify-only

# Verify standalone
inji doctor --profile verify-only

# List all available profiles
inji doctor --list-profiles
```

### Listing Checks

```bash
# Show all available diagnostic checks
inji doctor --list-checks
```

### Configuration File

Create `~/.inji-doctor.yaml` for persistent settings:

```yaml
# Stack profile to use
stack_profile: dev-stack

# Base URL for HTTP health checks
base_url: http://localhost

# Timeout per check in seconds
timeout_sec: 5

# Directory containing Inji config files
config_dir: /path/to/inji-config

# Docker Compose file for container status checks
docker_compose_file: /path/to/docker-compose.yml

# Output format: terminal, json, markdown
output: terminal

# Enable verbose output
verbose: false
```

## Checks Performed

### Service Checks (per component)
- HTTP/TCP reachability on the expected port
- Health endpoint response code
- Redirect detection and follow-up
- Timeout detection
- DNS resolution

### Configuration Checks
- Redirect URI alignment between Mimoto and eSignet
- Certify domain URL consistency
- Mimoto issuer configuration completeness
- Database connection string validity
- Verify trust registry URL configuration

### Key Checks
- Certify DID document and verification methods
- Mimoto OIDC keystore (p12 file) existence
- eSignet key manager configuration
- Loader path existence for external JARs

### Trust Registry Checks
- Verify trust registry reachability
- Certify signing key extraction
- Trust registry key comparison
- Key synchronization status

### Docker Checks
- Container running status
- Health check status (if configured)
- Restart count detection
- Exit code reporting

### System Checks
- RAM availability (minimum 4 GB recommended)
- Disk space (minimum 5 GB free recommended)
- CPU load average
- Docker and Docker Compose availability

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | All checks passed |
| 1 | Warnings found (stack is functional but has issues) |
| 2 | Errors found (some components are not working) |
| 3 | Critical issues (stack may be non-functional) |
| 127 | Command-line error (invalid flags, etc.) |

## Output Formats

### Terminal (default)
Colored, human-readable output with status symbols and fix suggestions.

### JSON
Machine-readable output for automation, CI/CD, and integration with monitoring systems.

```json
{
  "started_at": "2026-04-11T10:30:00Z",
  "finished_at": "2026-04-11T10:30:02Z",
  "duration": 2300000000,
  "summary": {
    "total": 14,
    "passed": 12,
    "warnings": 2,
    "failed": 0,
    "critical": 0,
    "skipped": 0,
    "fixable": 1
  },
  "results": [...],
  "version": "0.1.0",
  "stack_profile": "dev-stack"
}
```

### Markdown
Documentation-ready output suitable for bug reports, runbooks, or knowledge base articles.

## Contributing

This tool is part of the Inji ecosystem. Contributions are welcome under the Mozilla Public License 2.0.

### Adding a New Check

1. Create a new file in `internal/checker/` implementing the `checker.Checker` interface
2. Add the check to `internal/checker/registry.go` in the `allCheckers()` function
3. Add tests in `internal/checker/`

### Building

```bash
go build ./...
go test ./...
```

## License

Mozilla Public License 2.0 (MPL-2.0)

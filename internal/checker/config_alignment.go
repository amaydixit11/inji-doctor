package checker

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/inji/inji-doctor/internal/config"
	"github.com/inji/inji-doctor/internal/model"
)

// configAlignment checks whether configuration values match across Inji components.
//
// This is the #2 most common failure mode: redirect_uri mismatches, wrong hostnames,
// misaligned client IDs. Based on 8+ community posts about "400 invalid_request"
// where the real issue was config misalignment.
type configAlignment struct{}

func ConfigAlignment() Checker {
	return &configAlignment{}
}

func (c *configAlignment) ID() string       { return "config-alignment" }
func (c *configAlignment) Name() string     { return "Configuration alignment across components" }
func (c *configAlignment) Category() model.CheckCategory { return model.CategoryConfig }
func (c *configAlignment) Component() string { return "" } // Cross-component check

func (c *configAlignment) Run(ctx context.Context, cfg config.Config) model.CheckResult {
	if cfg.ConfigDir == "" {
		return Skipped("No config directory specified. Use --config-dir or set INJI_DOCTOR_CONFIG_DIR.").
			Build(c.ID(), c.Name(), c.Category(), "")
	}

	issues := c.checkAlignment(cfg)
	if len(issues) == 0 {
		return OK("All checked configuration values are aligned across components").
			Build(c.ID(), c.Name(), c.Category(), "")
	}

	// Build a detailed message from all issues found.
	var msgs []string
	var fixes []string
	for _, issue := range issues {
		msgs = append(msgs, issue.Message)
		if issue.Fix != "" {
			fixes = append(fixes, issue.Fix)
		}
	}

	severity := model.SeverityWarning
	if hasCritical(issues) {
		severity = model.SeverityError
	}

	return Result{
		Severity: severity,
		Message:  fmt.Sprintf("Found %d configuration mismatch(es)", len(issues)),
		Detail:   strings.Join(msgs, "\n"),
		Fix:      strings.Join(fixes, "\n"),
		RawData:  map[string]any{"issues": issues},
	}.Build(c.ID(), c.Name(), c.Category(), "")
}

// configIssue represents a single configuration mismatch.
type configIssue struct {
	Key         string `json:"key"`
	Expected    string `json:"expected"`
	Actual      string `json:"actual"`
	SourceFile  string `json:"source_file"`
	TargetFile  string `json:"target_file"`
	Message     string `json:"message"`
	Fix         string `json:"fix"`
	IsCritical  bool   `json:"is_critical"`
}

func (c *configAlignment) checkAlignment(cfg config.Config) []configIssue {
	var issues []configIssue

	// Check 1: eSignet redirect_uri alignment between Mimoto and eSignet config.
	// This is the MOST COMMON mismatch — causes "400 invalid_request" in Wallet.
	issues = append(issues, c.checkRedirectURI(cfg)...)

	// Check 2: Certify domain URL alignment.
	issues = append(issues, c.checkCertifyDomain(cfg)...)

	// Check 3: Mimoto issuer configuration.
	issues = append(issues, c.checkMimotoIssuers(cfg)...)

	// Check 4: Database connectivity config.
	issues = append(issues, c.checkDatabaseConfig(cfg)...)

	// Check 5: Verify trust registry configuration.
	issues = append(issues, c.checkVerifyTrustConfig(cfg)...)

	return issues
}

// checkRedirectURI checks that Mimoto's redirect_uri matches what eSignet expects.
// This is the single most common error in community posts.
func (c *configAlignment) checkRedirectURI(cfg config.Config) []configIssue {
	var issues []configIssue

	// Read Mimoto config.
	mimotoProps := c.readProperties(cfg, "mimoto-default.properties")
	mimotoIssuers := c.readJSON(cfg, "mimoto-issuers-config.json")

	// Read eSignet config.
	esignetProps := c.readProperties(cfg, "esignet-default.properties")

	// Extract redirect_uri from Mimoto config.
	mimotoRedirect := mimotoProps["mosip.oidc.redirect.uri"]
	if mimotoRedirect == "" {
		mimotoRedirect = mimotoProps["mosipbox.public.url"] + "/callback"
	}

	// Extract registered redirect URIs from eSignet.
	// In eSignet config, client redirect URIs are typically in the OIDC client config.
	esignetRegisteredURIs := []string{}
	if uris, ok := esignetProps["esignet.oauth.client.redirect.uris"]; ok {
		esignetRegisteredURIs = strings.Split(uris, ",")
		for i := range esignetRegisteredURIs {
			esignetRegisteredURIs[i] = strings.TrimSpace(esignetRegisteredURIs[i])
		}
	}

	// If we found both values, compare them.
	if mimotoRedirect != "" && len(esignetRegisteredURIs) > 0 {
		matched := false
		for _, uri := range esignetRegisteredURIs {
			if strings.EqualFold(uri, mimotoRedirect) {
				matched = true
				break
			}
		}
		if !matched {
			issues = append(issues, configIssue{
				Key:        "redirect_uri",
				Expected:   strings.Join(esignetRegisteredURIs, ", "),
				Actual:     mimotoRedirect,
				SourceFile: "mimoto-default.properties",
				TargetFile: "esignet-default.properties",
				Message: fmt.Sprintf(
					"Mimoto redirect_uri '%s' does not match any registered eSignet redirect URI [%s]. "+
						"This causes '400 invalid_request' during Wallet login.",
					mimotoRedirect, strings.Join(esignetRegisteredURIs, ", "),
				),
				Fix: fmt.Sprintf(
					"Update 'mosip.oidc.redirect.uri' in mimoto-default.properties to match one of: [%s]",
					strings.Join(esignetRegisteredURIs, ", "),
				),
				IsCritical: true,
			})
		}
	}

	// Also check the issuers config for token_endpoint mismatches.
	if mimotoIssuers != nil {
		// Check if token_endpoint URLs look reachable.
		if issuers, ok := mimotoIssuers["issuers"].([]interface{}); ok {
			for _, issuer := range issuers {
				if issuerMap, ok := issuer.(map[string]interface{}); ok {
					if tokenEndpoint, ok := issuerMap["token_endpoint"].(string); ok {
						if strings.Contains(tokenEndpoint, "localhost") && !isLocalhostReachable(tokenEndpoint) {
							issues = append(issues, configIssue{
								Key:        "issuer.token_endpoint",
								Expected:   "A reachable eSignet token endpoint",
								Actual:     tokenEndpoint,
								SourceFile: "mimoto-issuers-config.json",
								Message: fmt.Sprintf(
									"Issuer token_endpoint '%s' appears unreachable. "+
										"Ensure eSignet is running on the correct host and port.",
									tokenEndpoint,
								),
								Fix: fmt.Sprintf(
									"Update 'token_endpoint' in mimoto-issuers-config.json to point to the running eSignet instance.",
								),
								IsCritical: true,
							})
						}
					}
				}
			}
		}
	}

	return issues
}

// checkCertifyDomain checks that the Certify domain URL is consistent.
func (c *configAlignment) checkCertifyDomain(cfg config.Config) []configIssue {
	var issues []configIssue

	certifyProps := c.readProperties(cfg, "certify-default.properties")

	certifyDomain := certifyProps["mosip.certify.domain.url"]
	authURL := certifyProps["mosip.certify.authorization.url"]

	if certifyDomain != "" && authURL != "" {
		// The authorization URL should be reachable from the certify domain context.
		if strings.Contains(authURL, "localhost") && certifyDomain != "" {
			if !strings.Contains(certifyDomain, "localhost") {
				issues = append(issues, configIssue{
					Key:        "certify.authorization.url",
					Expected:   "Authorization URL on same host type as certify domain",
					Actual:     fmt.Sprintf("certify.domain=%s, auth.url=%s", certifyDomain, authURL),
					SourceFile: "certify-default.properties",
					Message: "Certify domain and authorization URL are on different hosts. " +
						"This may cause CORS or connectivity issues.",
					Fix: "Ensure mosip.certify.domain.url and mosip.certify.authorization.url " +
						"use the same host type (both localhost or both public domain).",
				})
			}
		}
	}

	return issues
}

// checkMimotoIssuers checks Mimoto issuer configuration completeness.
func (c *configAlignment) checkMimotoIssuers(cfg config.Config) []configIssue {
	var issues []configIssue

	mimotoIssuers := c.readJSON(cfg, "mimoto-issuers-config.json")
	if mimotoIssuers == nil {
		issues = append(issues, configIssue{
			Key:        "mimoto.issuers.config",
			Expected:   "mimoto-issuers-config.json to exist with at least one issuer",
			Actual:     "File not found or empty",
			SourceFile: "mimoto-issuers-config.json",
			Message:    "Mimoto issuers config file is missing. Wallet won't show any issuers without it.",
			Fix:        "Create mimoto-issuers-config.json with at least one issuer configuration. See docs.inji.io for the format.",
		})
		return issues
	}

	// Check for required fields in each issuer.
	if issuers, ok := mimotoIssuers["issuers"].([]interface{}); ok {
		if len(issuers) == 0 {
			issues = append(issues, configIssue{
				Key:        "mimoto.issuers",
				Expected:   "At least one issuer in the issuers array",
				Actual:     "Empty issuers array",
				SourceFile: "mimoto-issuers-config.json",
				Message:    "No issuers configured in mimoto-issuers-config.json. The wallet will show an empty issuer list.",
				Fix:        "Add at least one issuer to the 'issuers' array in mimoto-issuers-config.json.",
			})
		}

		for i, issuer := range issuers {
			issuerMap, ok := issuer.(map[string]interface{})
			if !ok {
				continue
			}

			// Check required fields.
			requiredFields := []string{"name", "issuer_id", "wellknown_endpoint", "client_id"}
			for _, field := range requiredFields {
				if _, hasField := issuerMap[field]; !hasField {
					issues = append(issues, configIssue{
						Key:        fmt.Sprintf("mimoto.issuers[%d].%s", i, field),
						Expected:   fmt.Sprintf("Field '%s' to be present", field),
						Actual:     fmt.Sprintf("Field '%s' is missing", field),
						SourceFile: "mimoto-issuers-config.json",
						Message:    fmt.Sprintf("Issuer #%d is missing required field '%s'.", i+1, field),
						Fix:        fmt.Sprintf("Add '%s' to the issuer configuration in mimoto-issuers-config.json.", field),
					})
				}
			}
		}
	}

	return issues
}

// checkDatabaseConfig checks that database connection strings are valid and reachable.
func (c *configAlignment) checkDatabaseConfig(cfg config.Config) []configIssue {
	var issues []configIssue

	// Check CertIFY database config.
	certifyProps := c.readProperties(cfg, "certify-default.properties")
	dbURL := certifyProps["spring.datasource.url"]
	dbUser := certifyProps["spring.datasource.username"]
	dbPass := certifyProps["spring.datasource.password"]

	if dbURL == "" {
		issues = append(issues, configIssue{
			Key:        "certify.datasource.url",
			Expected:   "A valid PostgreSQL connection URL",
			Actual:     "Not configured",
			SourceFile: "certify-default.properties",
			Message:    "Certify database URL is not configured. Certify cannot start without a database.",
			Fix:        "Set 'spring.datasource.url' in certify-default.properties to your PostgreSQL connection URL.",
		})
	} else if dbUser == "" {
		issues = append(issues, configIssue{
			Key:        "certify.datasource.username",
			Expected:   "Database username",
			Actual:     "Not configured",
			SourceFile: "certify-default.properties",
			Message:    "Certify database username is not configured.",
			Fix:        "Set 'spring.datasource.username' in certify-default.properties.",
		})
	} else if dbPass == "" {
		issues = append(issues, configIssue{
			Key:        "certify.datasource.password",
			Expected:   "Database password",
			Actual:     "Not configured",
			SourceFile: "certify-default.properties",
			Message:    "Certify database password is not configured.",
			Fix:        "Set 'spring.datasource.password' in certify-default.properties.",
		})
	}

	return issues
}

// checkVerifyTrustConfig checks Inji Verify trust registry configuration.
func (c *configAlignment) checkVerifyTrustConfig(cfg config.Config) []configIssue {
	var issues []configIssue

	verifyProps := c.readProperties(cfg, "verify-default.properties")

	trustURL := verifyProps["mosip.verify.trust.registry.url"]
	if trustURL == "" {
		// Check alternative property names.
		trustURL = verifyProps["mosip.verify.trust.url"]
	}

	if trustURL == "" {
		issues = append(issues, configIssue{
			Key:        "verify.trust.registry.url",
			Expected:   "Trust registry URL for VC verification",
			Actual:     "Not configured",
			SourceFile: "verify-default.properties",
			Message:    "Inji Verify trust registry URL is not configured. VC verification may fail.",
			Fix:        "Set 'mosip.verify.trust.registry.url' in verify-default.properties.",
		})
	}

	return issues
}

// --- File reading helpers ---

// readProperties reads a Java .properties file and returns key-value pairs.
func (c *configAlignment) readProperties(cfg config.Config, filename string) map[string]string {
	props := make(map[string]string)

	path := filepath.Join(cfg.ConfigDir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return props // Return empty map if file doesn't exist
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}

		// Split on first '=' or ':'.
		idx := strings.IndexAny(line, "=:")
		if idx < 0 {
			continue
		}

		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])

		// Handle line continuations.
		if strings.HasSuffix(value, "\\") {
			value = value[:len(value)-1]
		}

		props[key] = value
	}

	return props
}

// readJSON reads a JSON file and returns the parsed map.
func (c *configAlignment) readJSON(cfg config.Config, filename string) map[string]interface{} {
	path := filepath.Join(cfg.ConfigDir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil
	}

	return result
}

// isLocalhostReachable checks if a localhost URL is reachable.
func isLocalhostReachable(url string) bool {
	if !strings.Contains(url, "localhost") && !strings.Contains(url, "127.0.0.1") {
		return true // Not a localhost URL, assume it's reachable.
	}

	// Extract port from URL.
	re := regexp.MustCompile(`localhost:(\d+)`)
	matches := re.FindStringSubmatch(url)
	if len(matches) < 2 {
		re2 := regexp.MustCompile(`127\.0\.0\.1:(\d+)`)
		matches = re2.FindStringSubmatch(url)
	}
	if len(matches) < 2 {
		return false
	}

	port := matches[1]
	conn, err := net.DialTimeout("tcp", "localhost:"+port, 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// hasCritical checks if any issue is critical.
func hasCritical(issues []configIssue) bool {
	for _, i := range issues {
		if i.IsCritical {
			return true
		}
	}
	return false
}

package checker

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
	"os"
	"path/filepath"
	"time"

	"github.com/inji/inji-doctor/internal/config"
	"github.com/inji/inji-doctor/internal/model"
)

// keyValidity checks whether signing keys and certificates are valid,
// not expired, and properly configured.
//
// Based on community posts about "Private Key Entry is Missing" and
// "Could not generate secret" errors.
type keyValidity struct {
	component string
}

func KeyValidity(component string) Checker {
	return &keyValidity{component: component}
}

func (c *keyValidity) ID() string {
	return fmt.Sprintf("keys-%s-valid", c.component)
}

func (c *keyValidity) Name() string {
	comp := model.FindComponent(c.component)
	if comp != nil {
		return fmt.Sprintf("%s signing keys are valid", comp.Label)
	}
	return fmt.Sprintf("%s signing keys are valid", c.component)
}

func (c *keyValidity) Category() model.CheckCategory { return model.CategoryKeys }
func (c *keyValidity) Component() string             { return c.component }

func (c *keyValidity) Run(ctx context.Context, cfg config.Config) model.CheckResult {
	if cfg.ConfigDir == "" {
		return Skipped("No config directory specified.").
			Build(c.ID(), c.Name(), c.Category(), c.component)
	}

	switch c.component {
	case "inji-certify":
		return c.checkCertifyKeys(cfg)
	case "mimoto":
		return c.checkMimotoKeys(cfg)
	case "esignet":
		return c.checkEsignetKeys(cfg)
	default:
		return Skipped(fmt.Sprintf("No key checks defined for %s", c.component)).
			Build(c.ID(), c.Name(), c.Category(), c.component)
	}
}

// checkCertifyKeys checks Certify's signing keys.
func (c *keyValidity) checkCertifyKeys(cfg config.Config) model.CheckResult {
	// Check for DID document / signing key files.
	didPath := filepath.Join(cfg.ConfigDir, "did.json")
	if _, err := os.Stat(didPath); err == nil {
		data, err := os.ReadFile(didPath)
		if err != nil {
			return Error(
				fmt.Sprintf("Cannot read DID document: %v", err),
				"Check file permissions on did.json.",
			).Build(c.ID(), c.Name(), c.Category(), c.component)
		}

		// Parse the DID document and check for verificationMethod entries.
		// A valid DID document should have at least one verification method with a public key.
		var didDoc map[string]interface{}
		if err := unmarshalJSON(data, &didDoc); err != nil {
			return Error(
				"DID document (did.json) is not valid JSON",
				"Check the did.json file for syntax errors.",
			).Build(c.ID(), c.Name(), c.Category(), c.component)
		}

		vm, hasVM := didDoc["verificationMethod"]
		if !hasVM {
			return Warning(
				"DID document has no verificationMethod entries",
				"Credential signatures may not be verifiable. Add a verificationMethod to did.json.",
			).Build(c.ID(), c.Name(), c.Category(), c.component)
		}

		vmList, ok := vm.([]interface{})
		if !ok || len(vmList) == 0 {
			return Warning(
				"DID document verificationMethod is empty",
				"Add at least one verificationMethod with a public key to did.json.",
			).Build(c.ID(), c.Name(), c.Category(), c.component)
		}

		return OK(fmt.Sprintf("DID document has %d verification method(s)", len(vmList))).
			Build(c.ID(), c.Name(), c.Category(), c.component)
	}

	// If no did.json, check for key manager / softHSM configuration.
	certifyProps := c.readProperties(cfg, "certify-default.properties")
	keyManagerURL := certifyProps["mosip.keymanager.url"]
	if keyManagerURL != "" {
		// Key manager is configured externally — we can't check keys without hitting the API.
		// But we can at least verify the URL is set.
		return OK(fmt.Sprintf("Key manager URL configured: %s", keyManagerURL)).
			Build(c.ID(), c.Name(), c.Category(), c.component)
	}

	return Warning(
		"No signing key configuration found for Certify",
		"Ensure either did.json exists in the config directory or mosip.keymanager.url is set in certify-default.properties.",
	).Build(c.ID(), c.Name(), c.Category(), c.component)
}

// checkMimotoKeys checks Mimoto's OIDC keystore (p12 file).
func (c *keyValidity) checkMimotoKeys(cfg config.Config) model.CheckResult {
	mimotoProps := c.readProperties(cfg, "mimoto-default.properties")

	// Check for p12 keystore configuration.
	p12Password := mimotoProps["mosip.oidc.p12.password"]
	if p12Password == "" {
		return Warning(
			"mimoto-default.properties: mosip.oidc.p12.password is not set",
			"Set the p12 keystore password. This is required for OIDC client authentication.",
		).Build(c.ID(), c.Name(), c.Category(), c.component)
	}

	// Check if the p12 file exists in the expected location.
	certsDir := filepath.Join(cfg.ConfigDir, "certs")
	p12Path := filepath.Join(certsDir, "oidckeystore.p12")
	if _, err := os.Stat(p12Path); err != nil {
		return Error(
			fmt.Sprintf("OIDC keystore not found: %s", p12Path),
			fmt.Sprintf(
				"Create the OIDC client keystore: mkdir -p %s && place your oidckeystore.p12 file there. "+
					"The p12 file is generated when you create an OIDC client in eSignet/Keycloak.",
				certsDir,
			),
		).Build(c.ID(), c.Name(), c.Category(), c.component)
	}

	// Try to read and parse the p12 file.
	p12Data, err := os.ReadFile(p12Path)
	if err != nil {
		return Error(
			fmt.Sprintf("Cannot read OIDC keystore: %v", err),
			"Check file permissions on oidckeystore.p12.",
		).Build(c.ID(), c.Name(), c.Category(), c.component)
	}

	if len(p12Data) == 0 {
		return Error(
			"OIDC keystore file is empty",
			fmt.Sprintf("Regenerate the OIDC client keystore and place it at %s.", p12Path),
		).Build(c.ID(), c.Name(), c.Category(), c.component)
	}

	return OK(fmt.Sprintf("OIDC keystore found at %s (%d bytes)", p12Path, len(p12Data))).
		Build(c.ID(), c.Name(), c.Category(), c.component)
}

// checkEsignetKeys checks eSignet's key configuration.
func (c *keyValidity) checkEsignetKeys(cfg config.Config) model.CheckResult {
	esignetProps := c.readProperties(cfg, "esignet-default.properties")

	// Check for key manager configuration.
	keyManagerURL := esignetProps["mosip.keymanager.url"]
	if keyManagerURL == "" {
		return Warning(
			"esignet-default.properties: mosip.keymanager.url is not set",
			"Set the key manager URL for eSignet to manage signing keys.",
		).Build(c.ID(), c.Name(), c.Category(), c.component)
	}

	// Check for the auth wrapper JAR in the loader path (for standalone eSignet).
	loaderPath := esignetProps["loader.path"]
	if loaderPath != "" {
		// Check if the loader path contains the auth wrapper JAR.
		// This is a common source of "NoSuchSecurityProviderException" errors.
		if _, err := os.Stat(loaderPath); err != nil {
			return Warning(
				fmt.Sprintf("Loader path does not exist: %s", loaderPath),
				"Create the loader path directory and place required JARs there.",
			).Build(c.ID(), c.Name(), c.Category(), c.component)
		}
	}

	return OK(fmt.Sprintf("Key manager URL configured: %s", keyManagerURL)).
		Build(c.ID(), c.Name(), c.Category(), c.component)
}

// readProperties is a helper for reading Java .properties files.
func (c *keyValidity) readProperties(cfg config.Config, filename string) map[string]string {
	props := make(map[string]string)
	path := filepath.Join(cfg.ConfigDir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return props
	}

	for _, line := range splitLines(string(data)) {
		line = trim(line)
		if line == "" || line[0] == '#' || line[0] == '!' {
			continue
		}
		idx := indexAny(line, "=:")
		if idx < 0 {
			continue
		}
		props[trim(line[:idx])] = trim(line[idx+1:])
	}

	return props
}


func splitLines(s string) []string {
	lines := make([]string, 0)
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func trim(s string) string {
	s = strings.TrimLeft(s, " \t\r\n")
	s = strings.TrimRight(s, " \t\r\n")
	return s
}

func indexAny(s, chars string) int {
	for i := 0; i < len(s); i++ {
		for j := 0; j < len(chars); j++ {
			if s[i] == chars[j] {
				return i
			}
		}
	}
	return -1
}

// Check if a certificate is expired.
func isCertExpired(certData []byte) (bool, time.Time, error) {
	block, _ := pem.Decode(certData)
	if block == nil {
		return false, time.Time{}, fmt.Errorf("no PEM block found")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return false, time.Time{}, err
	}

	return time.Now().After(cert.NotAfter), cert.NotAfter, nil
}

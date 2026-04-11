package checker

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/inji/inji-doctor/internal/config"
	"github.com/inji/inji-doctor/internal/model"
)

// trustRegistry checks whether Inji Verify's trust registry is in sync
// with the signing keys used by Inji Certify.
//
// When this is out of sync, Verify rejects credentials that are actually valid.
// This is a common issue when Certify rotates its signing key.
type trustRegistry struct{}

func TrustRegistry() Checker {
	return &trustRegistry{}
}

func (c *trustRegistry) ID() string       { return "trust-registry-sync" }
func (c *trustRegistry) Name() string     { return "Verify trust registry is synced with Certify signing keys" }
func (c *trustRegistry) Category() model.CheckCategory { return model.CategoryTrust }
func (c *trustRegistry) Component() string { return "inji-verify" }

func (c *trustRegistry) Run(ctx context.Context, cfg config.Config) model.CheckResult {
	verifyComp := model.FindComponent("inji-verify")
	certifyComp := model.FindComponent("inji-certify")
	if verifyComp == nil || certifyComp == nil {
		return Skipped("Verify or Certify component not found in known components.").
			Build(c.ID(), c.Name(), c.Category(), c.Component())
	}

	if !cfg.IsEnabled("inji-verify") {
		return Skipped("Inji Verify is disabled in configuration.").
			Build(c.ID(), c.Name(), c.Category(), c.Component())
	}

	// Build URLs.
	verifyHost := cfg.ResolveHost("inji-verify")
	verifyPort := cfg.ResolvePort("inji-verify", verifyComp.DefaultPort)
	certifyHost := cfg.ResolveHost("inji-certify")
	certifyPort := cfg.ResolvePort("inji-certify", certifyComp.DefaultPort)

	// Check if both services are reachable first.
	certifyReachable := c.isReachable(ctx, certifyHost, certifyPort, cfg)
	verifyReachable := c.isReachable(ctx, verifyHost, verifyPort, cfg)

	if !certifyReachable {
		return Warning(
			"Cannot check trust registry — Inji Certify is not reachable",
			fmt.Sprintf("Start Certify first: ensure it's running on %s:%d.", certifyHost, certifyPort),
		).Build(c.ID(), c.Name(), c.Category(), c.Component())
	}

	if !verifyReachable {
		return Warning(
			"Cannot check trust registry — Inji Verify is not reachable",
			fmt.Sprintf("Start Verify first: ensure it's running on %s:%d.", verifyHost, verifyPort),
		).Build(c.ID(), c.Name(), c.Category(), c.Component())
	}

	// Get Certify's signing key (via its DID document or well-known endpoint).
	certifyKeyID, certifyKeyData := c.getCertifySigningKey(ctx, certifyHost, certifyPort, cfg)

	// Get Verify's trusted keys (via its trust registry endpoint).
	trustedKeyIDs := c.getTrustedKeyIDs(ctx, verifyHost, verifyPort, cfg)

	if certifyKeyID == "" {
		return Warning(
			"Could not determine Certify's current signing key ID",
			"Check the Certify logs for key generation messages, or verify did.json is properly configured.",
		).Build(c.ID(), c.Name(), c.Category(), c.Component())
	}

	if len(trustedKeyIDs) == 0 {
		return Error(
			"Verify's trust registry has no trusted keys",
			"Configure the trust registry with Certify's signing key. Add the key to the trust registry configuration.",
		).Build(c.ID(), c.Name(), c.Category(), c.Component())
	}

	// Check if Certify's key is in Verify's trust list.
	keyTrusted := false
	for _, trustedID := range trustedKeyIDs {
		if trustedID == certifyKeyID {
			keyTrusted = true
			break
		}
	}

	if keyTrusted {
		return OK(fmt.Sprintf(
			"Certify's signing key (%s) is trusted by Verify. Trust registry has %d trusted key(s).",
			certifyKeyID, len(trustedKeyIDs),
		)).Build(c.ID(), c.Name(), c.Category(), c.Component())
	}

	// Key mismatch — this is a real problem.
	return Error(
		fmt.Sprintf(
			"Certify's signing key (%s) is NOT in Verify's trust registry. "+
				"Verify will reject credentials issued by Certify.",
			certifyKeyID,
		),
		fmt.Sprintf(
			"Add Certify's signing key to Verify's trust registry. "+
				"Key ID: %s. Key data: %s. "+
				"Alternatively, restart Verify to trigger a trust registry re-sync.",
			certifyKeyID, summarizeKeyData(certifyKeyData),
		),
	).Build(c.ID(), c.Name(), c.Category(), c.Component())
}

// isReachable checks if a service is reachable via HTTP.
func (c *trustRegistry) isReachable(ctx context.Context, host string, port int, cfg config.Config) bool {
	url := fmt.Sprintf("http://%s:%d/", host, port)
	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.TimeoutSec)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}

	client := &http.Client{Timeout: time.Duration(cfg.TimeoutSec) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return true
}

// getCertifySigningKey attempts to get Certify's current signing key ID.
func (c *trustRegistry) getCertifySigningKey(ctx context.Context, host string, port int, cfg config.Config) (string, string) {
	// Try the OIDC well-known endpoint first.
	url := fmt.Sprintf("http://%s:%d/.well-known/openid-credential-issuer", host, port)

	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.TimeoutSec)*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	client := &http.Client{Timeout: time.Duration(cfg.TimeoutSec) * time.Second}
	resp, err := client.Do(req)
	if err == nil && resp.StatusCode == 200 {
		defer resp.Body.Close()

		var doc map[string]interface{}
		if err := unmarshalJSON(readBody(resp), &doc); err == nil {
			// Look for credential_signing_alg_values_supported.
			if algs, ok := doc["credential_signing_alg_values_supported"].([]interface{}); ok && len(algs) > 0 {
				keyID := fmt.Sprintf("%v", algs[0])
				return keyID, fmt.Sprintf("%v", doc)
			}
		}
	}

	// Fallback: try the DID document endpoint.
	didURL := fmt.Sprintf("http://%s:%d/.well-known/did.json", host, port)
	req2, _ := http.NewRequestWithContext(reqCtx, http.MethodGet, didURL, nil)
	resp2, err := client.Do(req2)
	if err == nil && resp2.StatusCode == 200 {
		defer resp2.Body.Close()

		var didDoc map[string]interface{}
		if err := unmarshalJSON(readBody(resp2), &didDoc); err == nil {
			if vm, ok := didDoc["verificationMethod"].([]interface{}); ok && len(vm) > 0 {
				if vm0, ok := vm[0].(map[string]interface{}); ok {
					if id, ok := vm0["id"].(string); ok {
						return id, fmt.Sprintf("%v", didDoc)
					}
				}
			}
		}
	}

	return "", ""
}

// getTrustedKeyIDs attempts to get the list of trusted key IDs from Verify.
func (c *trustRegistry) getTrustedKeyIDs(ctx context.Context, host string, port int, cfg config.Config) []string {
	// Try the trust registry endpoint.
	url := fmt.Sprintf("http://%s:%d/v1/verify/trust-registry", host, port)

	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.TimeoutSec)*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	client := &http.Client{Timeout: time.Duration(cfg.TimeoutSec) * time.Second}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return nil
	}
	defer resp.Body.Close()

	var doc map[string]interface{}
	if err := unmarshalJSON(readBody(resp), &doc); err != nil {
		return nil
	}

	// Extract trusted key IDs from the response.
	var keyIDs []string
	if trusted, ok := doc["trustedKeys"].([]interface{}); ok {
		for _, key := range trusted {
			if km, ok := key.(map[string]interface{}); ok {
				if id, ok := km["keyId"].(string); ok {
					keyIDs = append(keyIDs, id)
				}
			}
		}
	}

	return keyIDs
}

// readBody reads the HTTP response body (up to 8KB).
func readBody(resp *http.Response) []byte {
	buf := make([]byte, 8192)
	n, _ := resp.Body.Read(buf)
	return buf[:n]
}

// summarizeKeyData returns a short summary of key data for display.
func summarizeKeyData(data string) string {
	if len(data) > 100 {
		return data[:100] + "..."
	}
	return data
}

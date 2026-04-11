package checker

import "github.com/inji/inji-doctor/internal/model"

// allCheckers returns the complete list of all diagnostic checks.
//
// This is the single source of truth. When you add a new check, add it here.
// The checks are ordered by priority: services first, then config, then keys,
// then trust, then docker, then system.
func allCheckers() []Checker {
	return []Checker{
		// === SERVICE CHECKS (most important — is it running?) ===
		ServiceReachable("inji-certify"),
		ServiceReachable("mimoto"),
		ServiceReachable("inji-web"),
		ServiceReachable("inji-verify"),
		ServiceReachable("esignet"),
		ServiceReachable("postgres"),
		ServiceReachable("redis"),

		// === CONFIGURATION CHECKS ===
		ConfigAlignment(),

		// === KEY CHECKS ===
		KeyValidity("inji-certify"),
		KeyValidity("mimoto"),
		KeyValidity("esignet"),

		// === TRUST CHECKS ===
		TrustRegistry(),

		// === DOCKER CHECKS ===
		DockerStatus(),

		// === SYSTEM CHECKS ===
		SystemResources(),
	}
}

// ChecksForProfile returns only the checkers relevant to a given stack profile.
func ChecksForProfile(profile string) []Checker {
	var selectedProfile *model.StackProfile
	for _, p := range model.KnownProfiles() {
		if p.Name == profile {
			selectedProfile = &p
			break
		}
	}

	if selectedProfile == nil {
		// Default to dev-stack.
		for _, p := range model.KnownProfiles() {
			if p.Name == "dev-stack" {
				selectedProfile = &p
				break
			}
		}
	}

	if selectedProfile == nil {
		return All() // Fallback: run everything.
	}

	// Filter checkers to only those targeting components in this profile.
	// System-wide checks (empty component) are always included.
	var filtered []Checker
	for _, checker := range All() {
		comp := checker.Component()
		if comp == "" {
			// System-wide check (e.g., docker status, system resources).
			filtered = append(filtered, checker)
			continue
		}
		for _, pc := range selectedProfile.Components {
			if comp == pc {
				filtered = append(filtered, checker)
				break
			}
		}
	}

	return filtered
}

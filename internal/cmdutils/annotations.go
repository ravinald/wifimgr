package cmdutils

// Command annotation keys for controlling initialization behavior.
// Commands declare their requirements using these annotations.
const (
	// AnnotationNoInit indicates a command requires no initialization.
	// Commands with this annotation skip all config and API setup.
	// Examples: encrypt, version, help, completion
	AnnotationNoInit = "wifimgr:no-init"

	// AnnotationNeedsConfig indicates a command needs config file access only.
	// Commands with this annotation get Viper and logging initialized but
	// skip API client creation and credential decryption.
	// Examples: show intent, lint config, init site
	AnnotationNeedsConfig = "wifimgr:needs-config"

	// AnnotationNeedsAPI is the default behavior (no annotation needed).
	// Commands without annotations or with this annotation get full initialization:
	// config loading, API client creation, and credential decryption.
	// Examples: show api, apply, refresh, search
	AnnotationNeedsAPI = "wifimgr:needs-api"
)

// Initialization tier levels
const (
	// TierNoInit - skip all initialization (Tier 0)
	TierNoInit = iota
	// TierConfigOnly - config file access only, no API (Tier 1)
	TierConfigOnly
	// TierFullAPI - full API access including credentials (Tier 2)
	TierFullAPI
)

// GetCommandTier determines the initialization tier for a command based on its annotations.
// Returns TierNoInit (0), TierConfigOnly (1), or TierFullAPI (2).
func GetCommandTier(annotations map[string]string) int {
	if annotations == nil {
		return TierFullAPI // Default to full API access
	}

	// Check for no-init annotation (Tier 0)
	if _, ok := annotations[AnnotationNoInit]; ok {
		return TierNoInit
	}

	// Check for config-only annotation (Tier 1)
	if _, ok := annotations[AnnotationNeedsConfig]; ok {
		return TierConfigOnly
	}

	// Default: full API access (Tier 2)
	return TierFullAPI
}

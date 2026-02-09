package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"

	"github.com/ravinald/wifimgr/internal/encryption"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// LabeledAPIConfig represents a single API configuration with its label.
// This extends the vendors.APIConfig with additional config-specific fields.
type LabeledAPIConfig struct {
	vendors.APIConfig
	ManagedKeys *ManagedKeys `json:"managed_keys,omitempty"`
}

// ValidationWarning captures issues found during config loading.
type ValidationWarning struct {
	Level   string // "api", "site", or "device"
	Site    string // Site name (if applicable)
	Device  string // Device name (if applicable)
	API     string // The invalid API reference
	Message string // Human-readable message
}

// BuildAPIConfigsFromViper constructs APIConfig objects from Viper configuration.
// Config format: api.<label>.* where each label has vendor, url, credentials, etc.
func BuildAPIConfigsFromViper() (map[string]*vendors.APIConfig, []ValidationWarning) {
	configs := make(map[string]*vendors.APIConfig)
	var warnings []ValidationWarning

	// Get all API configurations: api.<label>.*
	apiSection := viper.GetStringMap("api")

	logging.Debugf("Viper api section has %d keys", len(apiSection))

	// Process each API label
	// Note: We read directly from the map because viper.GetString("api.label.field")
	// doesn't work properly after GetStringMap - Viper doesn't register nested keys
	for label, value := range apiSection {
		nested, ok := value.(map[string]interface{})
		if !ok {
			warnings = append(warnings, ValidationWarning{
				Level:   "api",
				API:     label,
				Message: fmt.Sprintf("API %q has invalid structure", label),
			})
			continue
		}

		vendor := getStringFromMap(nested, "vendor")
		if vendor == "" {
			warnings = append(warnings, ValidationWarning{
				Level:   "api",
				API:     label,
				Message: fmt.Sprintf("API %q missing required 'vendor' field", label),
			})
			continue
		}

		// Extract credentials from nested map
		credentials := make(map[string]string)
		if credsMap, ok := nested["credentials"].(map[string]interface{}); ok {
			for k, v := range credsMap {
				if str, ok := v.(string); ok {
					credentials[k] = str
				}
			}
		}

		config := &vendors.APIConfig{
			Label:        label,
			Vendor:       vendor,
			URL:          getStringFromMap(nested, "url"),
			Credentials:  credentials,
			RateLimit:    getIntFromMap(nested, "rate_limit"),
			ResultsLimit: getIntFromMap(nested, "results_limit"),
			CacheTTL:     getCacheTTLFromMap(nested),
		}

		// Apply vendor-specific defaults
		applyVendorDefaults(config)

		// Normalize credential key names based on vendor
		// This allows using "api_key" in config for Mist (maps to api_token)
		normalizeCredentialNames(config)

		logging.Debugf("  Built config for %q: vendor=%s, url=%s, creds=%d fields",
			label, config.Vendor, config.URL, len(config.Credentials))

		configs[label] = config
	}

	// Apply environment variable overrides BEFORE decryption and validation
	// This allows credentials to come from env vars (e.g., WIFIMGR_API_MIST_CREDENTIALS_KEY)
	applyEnvOverrides(configs)

	// Decrypt encrypted credentials (those with "enc:" prefix)
	// This must happen after env overrides so that WIFIMGR_PASSWORD is available
	for _, config := range configs {
		if warns := decryptCredentials(config); len(warns) > 0 {
			warnings = append(warnings, warns...)
		}
	}

	// Validate credentials AFTER env overrides and decryption
	for label, config := range configs {
		if warns := validateCredentials(config); len(warns) > 0 {
			warnings = append(warnings, warns...)
			// Remove invalid configs
			delete(configs, label)
		}
	}

	logging.Debugf("BuildAPIConfigsFromViper returned %d configs", len(configs))

	return configs, warnings
}

// DefaultCacheTTL is the default cache TTL in seconds (1 day).
const DefaultCacheTTL = 86400

// normalizeCredentialNames normalizes credential field names to app-standard names.
// The app uses "api_key" as the standard field; "api_token" is accepted as an alias.
func normalizeCredentialNames(config *vendors.APIConfig) {
	// Normalize api_token -> api_key (app standard is api_key)
	if config.Credentials["api_key"] == "" && config.Credentials["api_token"] != "" {
		config.Credentials["api_key"] = config.Credentials["api_token"]
		delete(config.Credentials, "api_token")
	}
}

// applyVendorDefaults applies vendor-specific default values.
func applyVendorDefaults(config *vendors.APIConfig) {
	switch config.Vendor {
	case "mist":
		if config.URL == "" {
			config.URL = "https://api.mist.com"
		}
		if config.RateLimit == 0 {
			config.RateLimit = 5000 // Mist default
		}
	case "meraki":
		if config.URL == "" {
			config.URL = "https://api.meraki.com"
		}
		// Meraki rate limit: 10 requests/second with token bucket
		// Cap to 10 even if configured higher
		if config.RateLimit == 0 || config.RateLimit > 10 {
			config.RateLimit = 10
		}
	}

	if config.ResultsLimit == 0 {
		config.ResultsLimit = 100
	}

	// Apply default cache TTL if not explicitly set
	// -1 means not set, 0 means never expire
	if config.CacheTTL < 0 {
		config.CacheTTL = DefaultCacheTTL
	}
}

// decryptCredentials decrypts encrypted credential values (those with "enc:" prefix).
// Returns warnings if decryption fails or password is not available.
func decryptCredentials(config *vendors.APIConfig) []ValidationWarning {
	var warnings []ValidationWarning

	for key, value := range config.Credentials {
		if !encryption.IsEncrypted(value) {
			continue
		}

		// Need password to decrypt
		password := encryption.GetPasswordFromEnv()
		if password == "" {
			warnings = append(warnings, ValidationWarning{
				Level:   "api",
				API:     config.Label,
				Message: fmt.Sprintf("API %q credential %q is encrypted but %s not set", config.Label, key, encryption.PasswordEnvVar),
			})
			continue
		}

		// Attempt decryption
		decrypted, err := encryption.Decrypt(value, password)
		if err != nil {
			warnings = append(warnings, ValidationWarning{
				Level:   "api",
				API:     config.Label,
				Message: fmt.Sprintf("API %q failed to decrypt credential %q: %v", config.Label, key, err),
			})
			continue
		}

		// Replace with decrypted value
		config.Credentials[key] = decrypted
		logging.Debugf("Decrypted credential api.%s.credentials.%s", config.Label, key)
	}

	return warnings
}

// validateCredentials checks that required credentials are present.
// All vendors use the normalized field names: api_key, org_id
func validateCredentials(config *vendors.APIConfig) []ValidationWarning {
	var warnings []ValidationWarning

	// All vendors require org_id and api_key (normalized field names)
	switch config.Vendor {
	case "mist", "meraki":
		if config.Credentials["org_id"] == "" {
			warnings = append(warnings, ValidationWarning{
				Level:   "api",
				API:     config.Label,
				Message: fmt.Sprintf("API %q missing required credential 'org_id'", config.Label),
			})
		}
		if config.Credentials["api_key"] == "" {
			warnings = append(warnings, ValidationWarning{
				Level:   "api",
				API:     config.Label,
				Message: fmt.Sprintf("API %q missing required credential 'api_key'", config.Label),
			})
		}
	}

	return warnings
}

// applyEnvOverrides applies environment variable overrides to API configs.
// Environment variables follow the pattern: WIFIMGR_API_<LABEL>_CREDENTIALS_<FIELD>
// Supported fields: KEY (maps to api_key), ORG (maps to org_id), URL
// Note: Label dashes are converted to underscores (e.g., "mist-prod" -> "MIST_PROD")
func applyEnvOverrides(configs map[string]*vendors.APIConfig) {
	for label, config := range configs {
		envPrefix := fmt.Sprintf("WIFIMGR_API_%s_CREDENTIALS_", strings.ToUpper(strings.ReplaceAll(label, "-", "_")))

		// KEY maps to api_key (normalized field name for all vendors)
		if key := os.Getenv(envPrefix + "KEY"); key != "" {
			config.Credentials["api_key"] = key
		}

		// ORG maps to org_id
		if org := os.Getenv(envPrefix + "ORG"); org != "" {
			config.Credentials["org_id"] = org
		}

		// URL overrides the API base URL
		if url := os.Getenv(envPrefix + "URL"); url != "" {
			config.URL = url
		}
	}
}

// getStringFromMap safely extracts a string value from a map[string]interface{}
func getStringFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if str, ok := v.(string); ok {
			return str
		}
	}
	return ""
}

// getIntFromMap safely extracts an int value from a map[string]interface{}
func getIntFromMap(m map[string]interface{}, key string) int {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case int:
			return val
		case int64:
			return int(val)
		case float64:
			return int(val)
		}
	}
	return 0
}

// getCacheTTLFromMap extracts cache_ttl with special handling:
// - Not present: returns -1 (use default)
// - Value 0: returns 0 (never expire)
// - Positive value: returns that value
func getCacheTTLFromMap(m map[string]interface{}) int {
	v, ok := m["cache_ttl"]
	if !ok {
		return -1 // Not set, use default
	}
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	}
	return -1 // Invalid type, use default
}

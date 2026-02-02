package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"

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

		logging.Debugf("  Built config for %q: vendor=%s, url=%s, creds=%d fields",
			label, config.Vendor, config.URL, len(config.Credentials))

		configs[label] = config
	}

	// Apply environment variable overrides BEFORE validation
	// This allows credentials to come from env vars (e.g., WIFIMGR_API_MIST_CREDENTIALS_KEY)
	applyEnvOverrides(configs)

	// Validate credentials AFTER env overrides have been applied
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

// validateCredentials checks that required credentials are present.
func validateCredentials(config *vendors.APIConfig) []ValidationWarning {
	var warnings []ValidationWarning

	switch config.Vendor {
	case "mist":
		if config.Credentials["org_id"] == "" {
			warnings = append(warnings, ValidationWarning{
				Level:   "api",
				API:     config.Label,
				Message: fmt.Sprintf("API %q missing required credential 'org_id'", config.Label),
			})
		}
		if config.Credentials["api_token"] == "" {
			warnings = append(warnings, ValidationWarning{
				Level:   "api",
				API:     config.Label,
				Message: fmt.Sprintf("API %q missing required credential 'api_token'", config.Label),
			})
		}
	case "meraki":
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
// Supported fields: KEY, ORG, URL
// Note: Label dashes are converted to underscores (e.g., "mist-prod" -> "MIST_PROD")
func applyEnvOverrides(configs map[string]*vendors.APIConfig) {
	for label, config := range configs {
		envPrefix := fmt.Sprintf("WIFIMGR_API_%s_CREDENTIALS_", strings.ToUpper(strings.ReplaceAll(label, "-", "_")))

		// KEY maps to api_token (Mist) or api_key (Meraki) based on vendor
		if key := os.Getenv(envPrefix + "KEY"); key != "" {
			switch config.Vendor {
			case "meraki":
				config.Credentials["api_key"] = key
			default: // mist and others use api_token
				config.Credentials["api_token"] = key
			}
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

// GetDefinedAPILabels returns a map of defined API labels for validation.
func GetDefinedAPILabels(configs map[string]*vendors.APIConfig) map[string]bool {
	labels := make(map[string]bool, len(configs))
	for label := range configs {
		labels[label] = true
	}
	return labels
}

// PrintAPIConfigWarnings prints validation warnings to stderr.
func PrintAPIConfigWarnings(warnings []ValidationWarning) {
	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "WARN  Config validation: %s\n", w.Message)
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

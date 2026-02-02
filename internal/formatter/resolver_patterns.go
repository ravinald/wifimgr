package formatter

import (
	"strings"
)

// ResolverPatterns defines which field patterns can be resolved
type ResolverPatterns struct {
	patterns []string
}

// NewResolverPatterns creates a new resolver patterns instance
func NewResolverPatterns() *ResolverPatterns {
	return &ResolverPatterns{
		patterns: []string{
			"site_id",
			"rf_template_id",
			"rfprofileid", // Meraki uses camelCase rfProfileId
			"gateway_template_id",
			"network_template_id",
			"ap_template_id",
			"device_profile_id",
			"wlan_template_id",
			// Add compound patterns
			"alarm_template_id",
			"network_id",
			"sec_policy_id",
			"site_template_id",
		},
	}
}

// IsResolvable checks if a field path matches any resolvable pattern
func (rp *ResolverPatterns) IsResolvable(fieldPath string) bool {
	fieldLower := strings.ToLower(fieldPath)

	for _, pattern := range rp.patterns {
		// Exact match
		if fieldLower == pattern {
			return true
		}

		// Suffix match (e.g., "ap_rf_template_id" matches "rf_template_id")
		if strings.HasSuffix(fieldLower, pattern) {
			return true
		}
	}

	return false
}

// GetSupportedPatterns returns all supported resolution patterns
func (rp *ResolverPatterns) GetSupportedPatterns() []string {
	return append([]string{}, rp.patterns...) // Return copy
}

// AddPattern adds a new resolvable pattern
func (rp *ResolverPatterns) AddPattern(pattern string) {
	pattern = strings.ToLower(pattern)

	// Check if pattern already exists
	for _, existing := range rp.patterns {
		if existing == pattern {
			return
		}
	}

	rp.patterns = append(rp.patterns, pattern)
}

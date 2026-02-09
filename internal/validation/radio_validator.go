// Package validation provides configuration validation utilities.
package validation

import (
	"fmt"
)

// RadioValidator validates radio configuration for APs.
type RadioValidator struct {
	deviceModel string
	vendor      string
}

// NewRadioValidator creates a new radio validator.
func NewRadioValidator(vendor, deviceModel string) *RadioValidator {
	return &RadioValidator{
		vendor:      vendor,
		deviceModel: deviceModel,
	}
}

// ValidateRadioConfig validates the entire radio_config block.
func (v *RadioValidator) ValidateRadioConfig(rc map[string]any) []LintIssue {
	if rc == nil {
		return nil
	}

	var issues []LintIssue

	// Validate each band if present
	if band24, ok := rc["band_24"].(map[string]any); ok {
		issues = append(issues, v.validateBand24(band24)...)
	}

	if band5, ok := rc["band_5"].(map[string]any); ok {
		issues = append(issues, v.validateBand5(band5)...)
	}

	if band6, ok := rc["band_6"].(map[string]any); ok {
		issues = append(issues, v.validateBand6(band6)...)
	}

	if bandDual, ok := rc["band_dual"].(map[string]any); ok {
		issues = append(issues, v.validateBandDual(bandDual)...)
	}

	// Validate band_5_on_24_radio (Mist-specific)
	if band5On24, ok := rc["band_5_on_24_radio"].(map[string]any); ok {
		if v.vendor == "meraki" {
			issues = append(issues, LintIssue{
				Field:      "radio_config.band_5_on_24_radio",
				Message:    "band_5_on_24_radio is Mist-specific, use band_dual for Meraki",
				Suggestion: "Remove band_5_on_24_radio or switch to Mist API",
			})
		} else {
			// Validate as 5GHz settings
			issues = append(issues, v.validateBand5WithPrefix("band_5_on_24_radio", band5On24)...)
		}
	}

	return issues
}

// validateBand24 validates 2.4 GHz band configuration.
func (v *RadioValidator) validateBand24(band map[string]any) []LintIssue {
	return v.validateBandWithRules("band_24", band, "band_24")
}

// validateBand5 validates 5 GHz band configuration.
func (v *RadioValidator) validateBand5(band map[string]any) []LintIssue {
	return v.validateBandWithRules("band_5", band, "band_5")
}

// validateBand5WithPrefix validates 5 GHz settings with a custom field prefix.
func (v *RadioValidator) validateBand5WithPrefix(prefix string, band map[string]any) []LintIssue {
	return v.validateBandWithRules(prefix, band, "band_5")
}

// validateBand6 validates 6 GHz band configuration.
func (v *RadioValidator) validateBand6(band map[string]any) []LintIssue {
	return v.validateBandWithRules("band_6", band, "band_6")
}

// validateBandDual validates dual-band/flex radio configuration.
func (v *RadioValidator) validateBandDual(band map[string]any) []LintIssue {
	var issues []LintIssue
	fieldPrefix := "radio_config.band_dual"

	// Check radio_mode
	radioMode, hasRadioMode := getIntValue(band, "radio_mode")
	if !hasRadioMode {
		// radio_mode is required when band_dual settings are specified
		if hasSettings(band) {
			issues = append(issues, LintIssue{
				Field:      fieldPrefix + ".radio_mode",
				Message:    "radio_mode is required when band_dual settings are specified",
				Suggestion: "Add radio_mode: 24, 5, or 6 depending on vendor",
			})
		}
		return issues
	}

	// Validate radio_mode for vendor
	if !IsValidRadioMode(v.vendor, radioMode) {
		allowedModes := DualBandRadioModes[v.vendor]
		if allowedModes == nil {
			allowedModes = DualBandRadioModes[""]
		}
		issues = append(issues, LintIssue{
			Field:      fieldPrefix + ".radio_mode",
			Message:    fmt.Sprintf("radio_mode %d is not valid for vendor '%s'", radioMode, v.vendor),
			Suggestion: fmt.Sprintf("Use one of: %v", allowedModes),
		})
	}

	// Get the band to validate against based on radio_mode
	targetBand := GetBandForRadioMode(radioMode)
	if targetBand == "" {
		issues = append(issues, LintIssue{
			Field:      fieldPrefix + ".radio_mode",
			Message:    fmt.Sprintf("invalid radio_mode value: %d", radioMode),
			Suggestion: "Use 24 (2.4GHz), 5 (5GHz), or 6 (6GHz)",
		})
		return issues
	}

	// Validate channel
	if channel, ok := getIntValue(band, "channel"); ok {
		if !IsValidChannel(targetBand, channel) {
			validChannels := GetValidChannels(targetBand)
			issues = append(issues, LintIssue{
				Field:      fieldPrefix + ".channel",
				Message:    fmt.Sprintf("channel %d is not valid for radio_mode %d (%s)", channel, radioMode, targetBand),
				Suggestion: fmt.Sprintf("Valid channels: %v (first 10 shown)", truncateSlice(validChannels, 10)),
			})
		}
	}

	// Validate bandwidth
	if bandwidth, ok := getIntValue(band, "bandwidth"); ok {
		if !IsValidBandwidth(targetBand, bandwidth) {
			validBandwidths := GetValidBandwidths(targetBand)
			issues = append(issues, LintIssue{
				Field:      fieldPrefix + ".bandwidth",
				Message:    fmt.Sprintf("bandwidth %d is not valid for radio_mode %d (%s)", bandwidth, radioMode, targetBand),
				Suggestion: fmt.Sprintf("Valid bandwidths: %v", validBandwidths),
			})
		}
	}

	// Validate power
	if power, ok := getIntValue(band, "power"); ok {
		if !IsValidPower(power) {
			issues = append(issues, LintIssue{
				Field:      fieldPrefix + ".power",
				Message:    fmt.Sprintf("power %d is out of range [%d-%d] dBm", power, PowerRange.Min, PowerRange.Max),
				Suggestion: fmt.Sprintf("Set power between %d and %d dBm", PowerRange.Min, PowerRange.Max),
			})
		}
	}

	return issues
}

// validateBandWithRules validates a band configuration against its rules.
func (v *RadioValidator) validateBandWithRules(fieldPrefix string, band map[string]any, bandType string) []LintIssue {
	var issues []LintIssue
	prefix := "radio_config." + fieldPrefix

	// Validate channel
	if channel, ok := getIntValue(band, "channel"); ok {
		if !IsValidChannel(bandType, channel) {
			validChannels := GetValidChannels(bandType)
			issues = append(issues, LintIssue{
				Field:      prefix + ".channel",
				Message:    fmt.Sprintf("channel %d is not valid for %s", channel, bandType),
				Suggestion: fmt.Sprintf("Valid channels: %v (first 10 shown)", truncateSlice(validChannels, 10)),
			})
		}
	}

	// Validate bandwidth
	if bandwidth, ok := getIntValue(band, "bandwidth"); ok {
		if !IsValidBandwidth(bandType, bandwidth) {
			validBandwidths := GetValidBandwidths(bandType)
			issues = append(issues, LintIssue{
				Field:      prefix + ".bandwidth",
				Message:    fmt.Sprintf("bandwidth %d is not valid for %s", bandwidth, bandType),
				Suggestion: fmt.Sprintf("Valid bandwidths: %v", validBandwidths),
			})
		}
	}

	// Validate power
	if power, ok := getIntValue(band, "power"); ok {
		if !IsValidPower(power) {
			issues = append(issues, LintIssue{
				Field:      prefix + ".power",
				Message:    fmt.Sprintf("power %d is out of range [%d-%d] dBm", power, PowerRange.Min, PowerRange.Max),
				Suggestion: fmt.Sprintf("Set power between %d and %d dBm", PowerRange.Min, PowerRange.Max),
			})
		}
	}

	return issues
}

// getIntValue extracts an int from a map, handling float64 JSON values.
func getIntValue(m map[string]any, key string) (int, bool) {
	val, ok := m[key]
	if !ok {
		return 0, false
	}

	switch v := val.(type) {
	case int:
		return v, true
	case float64:
		return int(v), true
	case int64:
		return int(v), true
	default:
		return 0, false
	}
}

// hasSettings checks if a band config has any actual settings (not just disabled).
func hasSettings(band map[string]any) bool {
	for key := range band {
		if key != "disabled" {
			return true
		}
	}
	return false
}

// truncateSlice returns first n elements of a slice.
func truncateSlice(slice []int, n int) []int {
	if len(slice) <= n {
		return slice
	}
	return slice[:n]
}

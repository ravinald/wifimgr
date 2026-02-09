package config

import (
	"fmt"
	"strings"

	"github.com/ravinald/wifimgr/internal/logging"
)

// ExpandDeviceConfig expands template references in a device config.
// Device-specific values override template values.
// The apiLabel is used to select vendor-specific template blocks.
func ExpandDeviceConfig(
	deviceConfig map[string]any,
	siteWLANs []string,
	templates *TemplateStore,
	apiLabel string,
) (map[string]any, error) {
	if templates == nil || templates.IsEmpty() {
		// No templates loaded, return copy of device config
		return copyMap(deviceConfig), nil
	}

	result := make(map[string]any)

	// Determine vendor from API label (e.g., "mist-prod" → "mist")
	vendor := GetVendorFromAPILabel(apiLabel)

	// Step 1: Expand device_template if present
	if templateName, ok := deviceConfig["device_template"].(string); ok {
		if template, found := templates.GetDeviceTemplate(templateName); found {
			expanded := ExpandForVendor(template, vendor)
			result = mergeConfigs(result, expanded)
			logging.Debugf("Expanded device_template '%s' for vendor '%s'", templateName, vendor)
		} else {
			logging.Warnf("Device template '%s' not found", templateName)
		}
	}

	// Step 2: Expand radio_profile if present
	// Radio templates contain band_* fields directly (not wrapped in radio_config)
	// We wrap them into radio_config during expansion
	if profileName, ok := deviceConfig["radio_profile"].(string); ok {
		if template, found := templates.GetRadioTemplate(profileName); found {
			expanded := ExpandForVendor(template, vendor)
			// Wrap radio template into radio_config
			if existingRadio, ok := result["radio_config"].(map[string]any); ok {
				result["radio_config"] = mergeConfigs(existingRadio, expanded)
			} else {
				result["radio_config"] = expanded
			}
			logging.Debugf("Expanded radio_profile '%s' for vendor '%s'", profileName, vendor)
		} else {
			logging.Warnf("Radio profile '%s' not found", profileName)
		}
	}

	// Step 3: Handle WLANs
	// Priority: device wlan > site wlan
	var wlanLabels []string
	if deviceWLANs, ok := deviceConfig["wlan"].([]any); ok && len(deviceWLANs) > 0 {
		// Device has explicit WLANs - use those
		wlanLabels = toStringSlice(deviceWLANs)
		logging.Debugf("Using device-level WLANs: %v", wlanLabels)
	} else if len(siteWLANs) > 0 {
		// Fall back to site WLANs
		wlanLabels = siteWLANs
		logging.Debugf("Using site-level WLANs: %v", wlanLabels)
	}

	// Expand WLAN templates
	if len(wlanLabels) > 0 {
		expandedWLANs, err := expandWLANs(wlanLabels, templates, vendor)
		if err != nil {
			return nil, fmt.Errorf("failed to expand WLANs: %w", err)
		}
		if len(expandedWLANs) > 0 {
			result["wlan"] = expandedWLANs
		}
	}

	// Step 4: Apply device-specific config (overrides templates)
	// Skip template reference fields
	for k, v := range deviceConfig {
		if isTemplateReferenceField(k) {
			continue
		}
		// Deep merge nested maps
		if existingMap, existsAsMap := result[k].(map[string]any); existsAsMap {
			if newMap, isNewMap := v.(map[string]any); isNewMap {
				result[k] = mergeConfigs(existingMap, newMap)
				continue
			}
		}
		result[k] = v
	}

	// Step 5: Ensure radio bands with settings have disabled: false
	if radioConfig, ok := result["radio_config"].(map[string]any); ok {
		result["radio_config"] = ensureRadioEnabled(radioConfig)
	}

	return result, nil
}

// expandWLANs expands a list of WLAN labels into their full configurations
func expandWLANs(labels []string, templates *TemplateStore, vendor string) ([]map[string]any, error) {
	expandedWLANs := make([]map[string]any, 0, len(labels))

	for _, label := range labels {
		template, found := templates.GetWLANTemplate(label)
		if !found {
			logging.Warnf("WLAN template '%s' not found", label)
			continue
		}

		expanded := ExpandForVendor(template, vendor)
		expandedWLANs = append(expandedWLANs, expanded)
		logging.Debugf("Expanded WLAN template '%s' for vendor '%s'", label, vendor)
	}

	return expandedWLANs, nil
}

// ExpandForVendor merges the appropriate vendor block into common config
func ExpandForVendor(template map[string]any, vendor string) map[string]any {
	result := make(map[string]any)

	// Copy non-vendor fields (common fields)
	for k, v := range template {
		if !isVendorBlock(k) {
			result[k] = deepCopy(v)
		}
	}

	// Merge vendor-specific block if present
	vendorKey := vendor + ":"
	if vendorBlock, ok := template[vendorKey].(map[string]any); ok {
		result = mergeConfigs(result, vendorBlock)
		logging.Debugf("Merged vendor block '%s' into template", vendorKey)
	}

	return result
}

// isVendorBlock returns true if the key is a vendor-specific block
func isVendorBlock(key string) bool {
	return strings.HasSuffix(key, ":")
}

// isTemplateReferenceField returns true if the key is a template reference
// These fields contain template names/labels, not actual configuration
func isTemplateReferenceField(key string) bool {
	switch key {
	case "radio_profile", "device_template", "wlan":
		// wlan is special: it's a list of WLAN template labels that get expanded
		// The expanded WLANs are already in result["wlan"] from step 3
		return true
	default:
		return false
	}
}

// GetVendorFromAPILabel extracts the vendor name from an API label
// e.g., "mist-prod" → "mist", "meraki-corp" → "meraki"
func GetVendorFromAPILabel(apiLabel string) string {
	if apiLabel == "" {
		return ""
	}

	// Common patterns: "vendor-env" or just "vendor"
	parts := strings.SplitN(apiLabel, "-", 2)
	vendor := strings.ToLower(parts[0])

	// Validate known vendors
	switch vendor {
	case "mist", "meraki":
		return vendor
	default:
		// Return as-is for unknown vendors
		return vendor
	}
}

// mergeConfigs merges source into dest, with source values taking precedence
// This performs a deep merge for nested maps
func mergeConfigs(dest, source map[string]any) map[string]any {
	result := make(map[string]any)

	// Copy all from dest
	for k, v := range dest {
		result[k] = deepCopy(v)
	}

	// Merge from source (source wins)
	for k, v := range source {
		if destMap, destIsMap := result[k].(map[string]any); destIsMap {
			if srcMap, srcIsMap := v.(map[string]any); srcIsMap {
				// Both are maps, deep merge
				result[k] = mergeConfigs(destMap, srcMap)
				continue
			}
		}
		result[k] = deepCopy(v)
	}

	return result
}

// deepCopy creates a deep copy of a value
func deepCopy(v any) any {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case map[string]any:
		result := make(map[string]any)
		for k, v := range val {
			result[k] = deepCopy(v)
		}
		return result
	case []any:
		result := make([]any, len(val))
		for i, v := range val {
			result[i] = deepCopy(v)
		}
		return result
	default:
		// Primitive types are immutable, return as-is
		return v
	}
}

// copyMap creates a shallow copy of a map
func copyMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	result := make(map[string]any)
	for k, v := range m {
		result[k] = v
	}
	return result
}

// toStringSlice converts []any to []string
func toStringSlice(arr []any) []string {
	result := make([]string, 0, len(arr))
	for _, v := range arr {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

// ensureRadioEnabled ensures that radio bands with settings have disabled: false.
// This is required because vendor APIs treat missing disabled field as "use site defaults"
// while we want explicit device-level configuration to take effect.
func ensureRadioEnabled(radioConfig map[string]any) map[string]any {
	bands := []string{"band_24", "band_5", "band_6", "band_dual"}

	for _, band := range bands {
		if bandConfig, ok := radioConfig[band].(map[string]any); ok {
			// If band has settings beyond just "disabled", ensure disabled: false
			if hasBandSettings(bandConfig) {
				if _, hasDisabled := bandConfig["disabled"]; !hasDisabled {
					bandConfig["disabled"] = false
					logging.Debugf("Auto-set disabled: false for %s", band)
				}
			}
		}
	}

	return radioConfig
}

// hasBandSettings checks if a band config has any actual settings (not just disabled).
func hasBandSettings(bandConfig map[string]any) bool {
	for key := range bandConfig {
		if key != "disabled" {
			return true
		}
	}
	return false
}

// GetSiteWLANLabels extracts WLAN labels from site configuration
func GetSiteWLANLabels(siteConfig map[string]any) []string {
	// Try profiles.wlan first
	if profiles, ok := siteConfig["profiles"].(map[string]any); ok {
		if wlans, ok := profiles["wlan"].([]any); ok {
			return toStringSlice(wlans)
		}
	}

	// Try direct wlan field
	if wlans, ok := siteConfig["wlan"].([]any); ok {
		return toStringSlice(wlans)
	}

	return nil
}

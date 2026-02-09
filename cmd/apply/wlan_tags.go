package apply

import (
	"sort"
	"strings"
)

// wifimgrWLANTagPrefix is the prefix for wifimgr-managed WLAN availability tags.
// Tags follow the pattern "wifimgr-wlan-{label}" where label is the WLAN template label.
const wifimgrWLANTagPrefix = "wifimgr-wlan-"

// generateWLANAvailabilityTag returns the Meraki availability tag for a WLAN template label.
func generateWLANAvailabilityTag(wlanLabel string) string {
	return wifimgrWLANTagPrefix + wlanLabel
}

// isWifimgrManagedTag returns true if the tag was created by wifimgr for WLAN availability.
func isWifimgrManagedTag(tag string) bool {
	return strings.HasPrefix(tag, wifimgrWLANTagPrefix)
}

// APTagMapping stores WLAN-derived availability tags per AP MAC address.
// Built during the WLAN apply phase and consumed by the device update phase
// to inject the correct tags into each AP's configuration.
type APTagMapping struct {
	APToTags map[string][]string // MAC → list of wifimgr-wlan-* tags
}

// Package-level storage (same pattern as currentTemplateStore)
var currentAPTagMapping *APTagMapping

func setAPTagMapping(m *APTagMapping) {
	currentAPTagMapping = m
}

func getAPTagMapping() *APTagMapping {
	return currentAPTagMapping
}

// buildAPTagMapping builds a reverse mapping from wlanToDevices:
// WLAN label → []MAC  becomes  MAC → []tag.
func buildAPTagMapping(wlanToDevices map[string][]string) *APTagMapping {
	m := &APTagMapping{APToTags: make(map[string][]string)}
	for label, macs := range wlanToDevices {
		tag := generateWLANAvailabilityTag(label)
		for _, mac := range macs {
			m.APToTags[mac] = append(m.APToTags[mac], tag)
		}
	}
	// Sort tags per AP for deterministic output
	for mac := range m.APToTags {
		sort.Strings(m.APToTags[mac])
	}
	return m
}

// mergeAPTags computes the final AP tag list by combining user/current tags with
// wifimgr-managed WLAN tags and removing orphaned wifimgr tags.
//
// Logic:
//  1. Start with userConfigTags if non-nil (user explicitly set tags), otherwise
//     use currentTags from the device's existing state.
//  2. Remove all wifimgr-wlan-* tags from the base set (clears orphans).
//  3. Append requiredWifimgrTags.
//  4. Deduplicate and sort.
func mergeAPTags(currentTags, userConfigTags []string, requiredWifimgrTags []string) []string {
	// Determine the base tag set
	var base []string
	if userConfigTags != nil {
		base = make([]string, len(userConfigTags))
		copy(base, userConfigTags)
	} else {
		base = make([]string, len(currentTags))
		copy(base, currentTags)
	}

	// Strip all wifimgr-managed tags from the base
	filtered := make([]string, 0, len(base))
	for _, t := range base {
		if !isWifimgrManagedTag(t) {
			filtered = append(filtered, t)
		}
	}

	// Append required wifimgr tags
	filtered = append(filtered, requiredWifimgrTags...)

	// Deduplicate
	seen := make(map[string]bool, len(filtered))
	result := make([]string, 0, len(filtered))
	for _, t := range filtered {
		if !seen[t] {
			seen[t] = true
			result = append(result, t)
		}
	}

	sort.Strings(result)
	return result
}

// mergeWifimgrTagsForAP injects wifimgr-managed WLAN availability tags into an AP's
// config map. Called during the device update phase for Meraki APs.
//
// Parameters:
//   - mac: the AP's MAC address (used to look up required tags)
//   - apConfig: the desired device config map (may already have "tags")
//   - currentDevice: map representing current device state from the API (has "tags")
func mergeWifimgrTagsForAP(mac string, apConfig map[string]any, currentDevice map[string]any) map[string]any {
	mapping := getAPTagMapping()
	if mapping == nil {
		return apConfig
	}

	requiredTags := mapping.APToTags[mac]

	// Extract user-configured tags (nil means user didn't specify tags)
	var userConfigTags []string
	var userConfigTagsSet bool
	if rawTags, ok := apConfig["tags"]; ok {
		userConfigTagsSet = true
		userConfigTags = toStringSlice(rawTags)
	}

	// Extract current AP tags from device state
	var currentTags []string
	if currentDevice != nil {
		if rawTags, ok := currentDevice["tags"]; ok {
			currentTags = toStringSlice(rawTags)
		}
	}

	var merged []string
	if userConfigTagsSet {
		merged = mergeAPTags(currentTags, userConfigTags, requiredTags)
	} else {
		merged = mergeAPTags(currentTags, nil, requiredTags)
	}

	// Only set tags if there are wifimgr tags to add or the user specified tags
	if len(merged) > 0 || userConfigTagsSet {
		apConfig["tags"] = merged
	}

	return apConfig
}

// toStringSlice converts various tag representations to []string.
func toStringSlice(v any) []string {
	switch val := v.(type) {
	case []string:
		return val
	case []any:
		result := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case string:
		if val == "" {
			return nil
		}
		return []string{val}
	default:
		return nil
	}
}

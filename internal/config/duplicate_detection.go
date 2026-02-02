package config

import (
	"strings"

	"github.com/ravinald/wifimgr/internal/logging"
)

// DuplicateEntry tracks information about a duplicate configuration entry
type DuplicateEntry struct {
	Name     string
	File     string
	Line     int
	Category string // "site_config" or "device_profile"
	Type     string // for device profiles: "ap", "switch", or "gateway"
}

// DuplicateTracker tracks duplicate configuration entries during loading
type DuplicateTracker struct {
	// Track seen entries: category -> type -> name -> first occurrence
	seen map[string]map[string]map[string]DuplicateEntry
	// Track duplicates found
	duplicates []DuplicateEntry
}

// NewDuplicateTracker creates a new duplicate tracker
func NewDuplicateTracker() *DuplicateTracker {
	return &DuplicateTracker{
		seen:       make(map[string]map[string]map[string]DuplicateEntry),
		duplicates: make([]DuplicateEntry, 0),
	}
}

// CheckAndAdd checks if an entry is a duplicate and adds it to tracking
func (dt *DuplicateTracker) CheckAndAdd(category, deviceType, name, file string, line int) bool {
	// Initialize nested maps if needed
	if dt.seen[category] == nil {
		dt.seen[category] = make(map[string]map[string]DuplicateEntry)
	}
	if dt.seen[category][deviceType] == nil {
		dt.seen[category][deviceType] = make(map[string]DuplicateEntry)
	}

	// Check if we've seen this before
	if existing, exists := dt.seen[category][deviceType][name]; exists {
		// Found a duplicate
		duplicate := DuplicateEntry{
			Name:     name,
			File:     file,
			Line:     line,
			Category: category,
			Type:     deviceType,
		}
		dt.duplicates = append(dt.duplicates, duplicate)

		// Log the duplicate detection
		logging.Warn("========================================")
		logging.Warnf("Duplicate %s found!", category)
		logging.Warnf("  Name: %s", name)
		if deviceType != "" {
			logging.Warnf("  Type: %s", deviceType)
		}
		logging.Warnf("  First defined in: %s (around line %d)", existing.File, existing.Line)
		logging.Warnf("  Duplicate found in: %s (around line %d)", file, line)
		logging.Warn("========================================")
		logging.Warn("")

		return true
	}

	// Not a duplicate, add to tracking
	dt.seen[category][deviceType][name] = DuplicateEntry{
		Name:     name,
		File:     file,
		Line:     line,
		Category: category,
		Type:     deviceType,
	}

	return false
}

// GetDuplicates returns all found duplicates
func (dt *DuplicateTracker) GetDuplicates() []DuplicateEntry {
	return dt.duplicates
}

// HasDuplicates returns true if any duplicates were found
func (dt *DuplicateTracker) HasDuplicates() bool {
	return len(dt.duplicates) > 0
}

// EstimateLineNumber attempts to estimate the line number for a given key in JSON data
// This is a rough estimate based on the JSON structure
func EstimateLineNumber(jsonData []byte, keyPath []string) int {
	lines := strings.Split(string(jsonData), "\n")
	currentLine := 1
	depth := 0
	targetDepth := len(keyPath)
	currentPath := 0

	for i, line := range lines {
		currentLine = i + 1
		trimmed := strings.TrimSpace(line)

		// Track JSON depth
		if strings.Contains(trimmed, "{") {
			depth++
		}
		if strings.Contains(trimmed, "}") {
			depth--
		}

		// Look for our key at the current path level
		if currentPath < targetDepth && strings.Contains(trimmed, `"`+keyPath[currentPath]+`"`) {
			// Check if this is a key (has a colon after it)
			if strings.Contains(trimmed, `"`+keyPath[currentPath]+`"`) && strings.Contains(trimmed, ":") {
				currentPath++
				if currentPath == targetDepth {
					return currentLine
				}
			}
		}
	}

	return currentLine
}

// ExtractNameFromJSON attempts to extract a "name" field from a JSON object
func ExtractNameFromJSON(data interface{}) string {
	if m, ok := data.(map[string]interface{}); ok {
		if name, exists := m["name"]; exists {
			if nameStr, ok := name.(string); ok {
				return nameStr
			}
		}
	}
	return ""
}

// TrackDuplicatesInJSON processes a raw JSON structure and tracks duplicates
// This is useful for processing device profiles which have a nested structure
func (dt *DuplicateTracker) TrackDuplicatesInJSON(category string, jsonData map[string]interface{}, file string, rawJSON []byte) {
	// For device profiles structure: config.device_profiles.<type>.<key>
	if category == "device_profile" {
		if config, ok := jsonData["config"].(map[string]interface{}); ok {
			if deviceProfiles, ok := config["device_profiles"].(map[string]interface{}); ok {
				// Process each device type (ap, switch, gateway)
				for deviceType, profiles := range deviceProfiles {
					if profilesMap, ok := profiles.(map[string]interface{}); ok {
						for profileKey, profileData := range profilesMap {
							// Extract name from profile data if available
							name := ExtractNameFromJSON(profileData)
							if name == "" {
								name = profileKey // Use key as fallback
							}

							// Estimate line number
							keyPath := []string{"config", "device_profiles", deviceType, profileKey}
							line := EstimateLineNumber(rawJSON, keyPath)

							dt.CheckAndAdd(category, deviceType, name, file, line)
						}
					}
				}
			}
		}
	}
}

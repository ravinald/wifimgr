package meraki

import (
	"github.com/ravinald/wifimgr/internal/vendors"
)

// MerakiSpecificAPFields lists fields that are Meraki-specific and should go into the meraki: extension block.
// These fields are not part of the common vendor-agnostic schema.
var MerakiSpecificAPFields = map[string]bool{
	// Meraki identifiers
	"serial":                 true,
	"networkId":              true,
	"network_id":             true,
	"productType":            true,
	"model":                  true,
	"firmware":               true,
	"configurationUpdatedAt": true,
	"url":                    true,
	"details":                true,
	"beaconIdParams":         true,

	// Meraki-specific features
	"floorPlanId":     true,
	"floor_plan_id":   true,
	"switchProfileId": true,
	"moveMapMarker":   true,

	// Meraki location fields (different format than common schema)
	"address": true,
}

// MerakiSpecificRadioFields lists radio-related fields specific to Meraki.
var MerakiSpecificRadioFields = map[string]bool{
	"rf_profile_id":      true,
	"rf_profile_name":    true,
	"rfProfileId":        true,
	"twoFourGhzSettings": true,
	"fiveGhzSettings":    true,
	"sixGhzSettings":     true,
	"perSsidSettings":    true,
}

// ExportAPConfig converts raw Meraki AP config data into a structured APDeviceConfig
// with common fields in standard locations and Meraki-specific fields in the meraki: extension block.
func ExportAPConfig(rawConfig map[string]any) *vendors.APDeviceConfig {
	if rawConfig == nil {
		return nil
	}

	config := &vendors.APDeviceConfig{
		Meraki: make(map[string]any),
	}

	// Extract identity fields
	if name, ok := rawConfig["name"].(string); ok {
		config.Name = name
	}
	if notes, ok := rawConfig["notes"].(string); ok {
		config.Notes = notes
	}

	// Meraki uses different tag format - handle as array or comma-separated
	if tags, ok := rawConfig["tags"].([]any); ok {
		config.Tags = toStringSlice(tags)
	} else if tagsStr, ok := rawConfig["tags"].(string); ok {
		// Meraki sometimes returns tags as space-separated string
		if tagsStr != "" {
			config.Meraki["tags_string"] = tagsStr
		}
	}

	// Extract location fields (Meraki uses lat/lng at top level)
	if lat, latOk := rawConfig["lat"].(float64); latOk {
		if lng, lngOk := rawConfig["lng"].(float64); lngOk {
			config.Location = []float64{lat, lng}
		}
	}

	// Meraki floor plan handling
	if floorPlanID, ok := rawConfig["floorPlanId"].(string); ok {
		config.MapID = floorPlanID
	}

	// Extract radio configuration from Meraki format
	config.RadioConfig = exportMerakiRadioConfig(rawConfig)

	// Extract Meraki-specific fields into the meraki: extension block
	for key, value := range rawConfig {
		if MerakiSpecificAPFields[key] {
			config.Meraki[key] = value
		}
	}

	// Handle radio-specific Meraki fields
	for key, value := range rawConfig {
		if MerakiSpecificRadioFields[key] {
			if config.RadioConfig == nil {
				config.RadioConfig = &vendors.RadioConfig{
					Meraki: make(map[string]any),
				}
			}
			if config.RadioConfig.Meraki == nil {
				config.RadioConfig.Meraki = make(map[string]any)
			}
			config.RadioConfig.Meraki[key] = value
		}
	}

	// Clean up empty meraki block
	if len(config.Meraki) == 0 {
		config.Meraki = nil
	}

	return config
}

// exportMerakiRadioConfig extracts radio configuration from Meraki API format.
func exportMerakiRadioConfig(raw map[string]any) *vendors.RadioConfig {
	rc := &vendors.RadioConfig{
		Meraki: make(map[string]any),
	}

	hasRadioConfig := false

	// Extract RF profile ID (Meraki-specific)
	if rfProfileID, ok := raw["rfProfileId"].(string); ok {
		rc.Meraki["rf_profile_id"] = rfProfileID
		hasRadioConfig = true
	}

	// Extract 2.4GHz settings
	if twoFour, ok := raw["twoFourGhzSettings"].(map[string]any); ok {
		rc.Band24 = exportMerakiBandConfig(twoFour)
		rc.Meraki["twoFourGhzSettings"] = twoFour // Keep original for reference
		hasRadioConfig = true
	}

	// Extract 5GHz settings
	if five, ok := raw["fiveGhzSettings"].(map[string]any); ok {
		rc.Band5 = exportMerakiBandConfig(five)
		rc.Meraki["fiveGhzSettings"] = five
		hasRadioConfig = true
	}

	// Extract 6GHz settings
	if six, ok := raw["sixGhzSettings"].(map[string]any); ok {
		rc.Band6 = exportMerakiBandConfig(six)
		rc.Meraki["sixGhzSettings"] = six
		hasRadioConfig = true
	}

	// Handle per-SSID settings (Meraki-specific)
	if perSsid, ok := raw["perSsidSettings"].(map[string]any); ok {
		rc.Meraki["perSsidSettings"] = perSsid
		hasRadioConfig = true
	}

	if !hasRadioConfig {
		return nil
	}

	if len(rc.Meraki) == 0 {
		rc.Meraki = nil
	}

	return rc
}

// exportMerakiBandConfig converts Meraki band settings to common format.
func exportMerakiBandConfig(raw map[string]any) *vendors.RadioBandConfig {
	if raw == nil {
		return nil
	}

	bc := &vendors.RadioBandConfig{
		Meraki: make(map[string]any),
	}

	// Channel
	if channel, ok := raw["channel"].(float64); ok {
		ch := int(channel)
		bc.Channel = &ch
	}

	// Target power (Meraki uses targetPower)
	if power, ok := raw["targetPower"].(float64); ok {
		p := int(power)
		bc.Power = &p
	}

	// Channel width (Meraki uses channelWidth as string like "20", "40", "80")
	if width, ok := raw["channelWidth"].(string); ok {
		bc.Meraki["channel_width"] = width
		// Try to convert to int for common format
		switch width {
		case "20":
			bw := 20
			bc.Bandwidth = &bw
		case "40":
			bw := 40
			bc.Bandwidth = &bw
		case "80":
			bw := 80
			bc.Bandwidth = &bw
		case "160":
			bw := 160
			bc.Bandwidth = &bw
		case "auto":
			bc.Meraki["bandwidth_auto"] = true
		}
	}

	// Min/max bitrate (Meraki-specific)
	if minBitrate, ok := raw["minBitrate"].(float64); ok {
		bc.Meraki["min_bitrate"] = int(minBitrate)
	}
	if maxBitrate, ok := raw["maxBitrate"].(float64); ok {
		bc.Meraki["max_bitrate"] = int(maxBitrate)
	}

	// Valid auto channels (Meraki-specific)
	if validAutoChannels, ok := raw["validAutoChannels"].([]any); ok {
		bc.Channels = toIntSlice(validAutoChannels)
	}

	// RX-SOP (Meraki-specific)
	if rxsop, ok := raw["rxsop"].(float64); ok {
		bc.Meraki["rxsop"] = int(rxsop)
	}

	if len(bc.Meraki) == 0 {
		bc.Meraki = nil
	}

	return bc
}

// Helper functions

func toStringSlice(arr []any) []string {
	result := make([]string, 0, len(arr))
	for _, v := range arr {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func toIntSlice(arr []any) []int {
	result := make([]int, 0, len(arr))
	for _, v := range arr {
		if f, ok := v.(float64); ok {
			result = append(result, int(f))
		}
	}
	return result
}

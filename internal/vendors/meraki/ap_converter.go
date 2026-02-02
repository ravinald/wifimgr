// Package meraki provides Meraki-specific API conversions.
package meraki

import (
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// ToMerakiAPConfig converts a vendor-agnostic APDeviceConfig to Meraki API format.
// It applies Meraki-specific field mappings and extracts the Meraki extension block.
func ToMerakiAPConfig(cfg *vendors.APDeviceConfig) map[string]any {
	if cfg == nil {
		return nil
	}

	result := make(map[string]any)

	// Identity
	if cfg.Name != "" {
		result["name"] = cfg.Name
	}
	if len(cfg.Tags) > 0 {
		// Meraki uses space-separated tags in a single string
		result["tags"] = tagsToMerakiFormat(cfg.Tags)
	}
	if cfg.Notes != "" {
		result["notes"] = cfg.Notes
	}

	// Location - Meraki uses lat/lng directly
	if len(cfg.Location) >= 2 {
		result["lat"] = cfg.Location[0]
		result["lng"] = cfg.Location[1]
	}

	// Radio config
	if cfg.RadioConfig != nil {
		result["radioSettings"] = convertRadioConfigToMeraki(cfg.RadioConfig)
	}

	// LED config - Meraki uses "ledLightsOn" boolean
	if cfg.LEDConfig != nil && cfg.LEDConfig.Enabled != nil {
		result["ledLightsOn"] = *cfg.LEDConfig.Enabled
	}

	// Merge Meraki extension block
	if cfg.Meraki != nil {
		for k, v := range cfg.Meraki {
			result[k] = v
		}
	}

	return result
}

// convertRadioConfigToMeraki converts radio configuration to Meraki format
func convertRadioConfigToMeraki(cfg *vendors.RadioConfig) map[string]any {
	if cfg == nil {
		return nil
	}

	result := make(map[string]any)

	// Per-band settings use different structure in Meraki
	if cfg.Band24 != nil {
		result["twoFourGhzSettings"] = convertBandConfigToMeraki(cfg.Band24)
	}
	if cfg.Band5 != nil {
		result["fiveGhzSettings"] = convertBandConfigToMeraki(cfg.Band5)
	}
	if cfg.Band6 != nil {
		result["sixGhzSettings"] = convertBandConfigToMeraki(cfg.Band6)
	}

	// Handle RF profile from Meraki extension
	if cfg.Meraki != nil {
		if rfProfileID, ok := cfg.Meraki["rf_profile_id"]; ok {
			result["rfProfileId"] = rfProfileID
		}
	}

	return result
}

// convertBandConfigToMeraki converts per-band settings to Meraki format
func convertBandConfigToMeraki(cfg *vendors.RadioBandConfig) map[string]any {
	if cfg == nil {
		return nil
	}

	result := make(map[string]any)

	if cfg.Channel != nil {
		result["channel"] = *cfg.Channel
	}
	if cfg.Power != nil {
		result["targetPower"] = *cfg.Power // Meraki uses "targetPower"
	}
	if cfg.Bandwidth != nil {
		result["channelWidth"] = *cfg.Bandwidth // Meraki uses "channelWidth"
	}

	// Meraki-specific fields from extension
	if cfg.Meraki != nil {
		if minBitrate, ok := cfg.Meraki["min_bitrate"]; ok {
			result["minBitrate"] = minBitrate
		}
		if rxsop, ok := cfg.Meraki["rxsop"]; ok {
			result["rxsop"] = rxsop
		}
	}

	return result
}

// FromMerakiAPConfig converts Meraki API response to vendor-agnostic APDeviceConfig.
// Returns the configuration and a slice of warnings (type assertion failures, unexpected fields, etc.).
func FromMerakiAPConfig(data map[string]any, mac string) (*vendors.APDeviceConfig, []error) {
	if data == nil {
		return nil, nil
	}

	cfg := &vendors.APDeviceConfig{}
	var warnings []error
	logger := logging.GetLogger()

	// Identity - using safe type converters
	if name, err := vendors.SafeString(data, "name", logger); err != nil {
		if fme, ok := err.(*vendors.FieldMappingError); ok {
			fme.Vendor = "meraki"
			fme.DeviceMAC = mac
		}
		warnings = append(warnings, err)
	} else {
		cfg.Name = name
	}

	if tagsStr, err := vendors.SafeString(data, "tags", logger); err != nil {
		if fme, ok := err.(*vendors.FieldMappingError); ok {
			fme.Vendor = "meraki"
			fme.DeviceMAC = mac
		}
		warnings = append(warnings, err)
	} else {
		cfg.Tags = merakiTagsToSlice(tagsStr)
	}

	if notes, err := vendors.SafeString(data, "notes", logger); err != nil {
		if fme, ok := err.(*vendors.FieldMappingError); ok {
			fme.Vendor = "meraki"
			fme.DeviceMAC = mac
		}
		warnings = append(warnings, err)
	} else {
		cfg.Notes = notes
	}

	// Location - Meraki uses separate lat/lng fields
	lat, err1 := vendors.SafeFloat64(data, "lat", logger)
	lng, err2 := vendors.SafeFloat64(data, "lng", logger)
	if err1 != nil {
		if fme, ok := err1.(*vendors.FieldMappingError); ok {
			fme.Vendor = "meraki"
			fme.DeviceMAC = mac
		}
		warnings = append(warnings, err1)
	}
	if err2 != nil {
		if fme, ok := err2.(*vendors.FieldMappingError); ok {
			fme.Vendor = "meraki"
			fme.DeviceMAC = mac
		}
		warnings = append(warnings, err2)
	}
	if lat != nil && lng != nil {
		cfg.Location = []float64{*lat, *lng}
	}

	// LED
	if ledOn, err := vendors.SafeBool(data, "ledLightsOn", logger); err != nil {
		if fme, ok := err.(*vendors.FieldMappingError); ok {
			fme.Vendor = "meraki"
			fme.DeviceMAC = mac
		}
		warnings = append(warnings, err)
	} else if ledOn != nil {
		cfg.LEDConfig = &vendors.LEDConfig{Enabled: ledOn}
	}

	// Radio settings
	if radioSettings, err := vendors.SafeMap(data, "radioSettings", logger); err != nil {
		if fme, ok := err.(*vendors.FieldMappingError); ok {
			fme.Vendor = "meraki"
			fme.DeviceMAC = mac
		}
		warnings = append(warnings, err)
	} else if radioSettings != nil {
		cfg.RadioConfig = parseRadioSettingsFromMeraki(radioSettings)
	}

	// Store any unrecognized Meraki-specific fields in extension block
	cfg.Meraki = extractMerakiExtensions(data)

	return cfg, warnings
}

// parseRadioSettingsFromMeraki parses Meraki radio settings to vendor-agnostic format
func parseRadioSettingsFromMeraki(data map[string]any) *vendors.RadioConfig {
	if data == nil {
		return nil
	}

	cfg := &vendors.RadioConfig{}

	if twoFourGhz, ok := data["twoFourGhzSettings"].(map[string]any); ok {
		cfg.Band24 = parseBandSettingsFromMeraki(twoFourGhz)
	}
	if fiveGhz, ok := data["fiveGhzSettings"].(map[string]any); ok {
		cfg.Band5 = parseBandSettingsFromMeraki(fiveGhz)
	}
	if sixGhz, ok := data["sixGhzSettings"].(map[string]any); ok {
		cfg.Band6 = parseBandSettingsFromMeraki(sixGhz)
	}

	// RF profile ID
	if rfProfileID, ok := data["rfProfileId"].(string); ok {
		if cfg.Meraki == nil {
			cfg.Meraki = make(map[string]any)
		}
		cfg.Meraki["rf_profile_id"] = rfProfileID
	}

	return cfg
}

// parseBandSettingsFromMeraki parses per-band settings from Meraki format
func parseBandSettingsFromMeraki(data map[string]any) *vendors.RadioBandConfig {
	if data == nil {
		return nil
	}

	cfg := &vendors.RadioBandConfig{}

	if channel, ok := data["channel"].(float64); ok {
		c := int(channel)
		cfg.Channel = &c
	}
	if targetPower, ok := data["targetPower"].(float64); ok {
		p := int(targetPower)
		cfg.Power = &p // Map back to vendor-agnostic "power"
	}
	if channelWidth, ok := data["channelWidth"].(float64); ok {
		b := int(channelWidth)
		cfg.Bandwidth = &b // Map back to vendor-agnostic "bandwidth"
	}

	// Store Meraki-specific fields in extension
	cfg.Meraki = make(map[string]any)
	if minBitrate, ok := data["minBitrate"]; ok {
		cfg.Meraki["min_bitrate"] = minBitrate
	}
	if rxsop, ok := data["rxsop"]; ok {
		cfg.Meraki["rxsop"] = rxsop
	}

	// Remove empty extension block
	if len(cfg.Meraki) == 0 {
		cfg.Meraki = nil
	}

	return cfg
}

// extractMerakiExtensions extracts Meraki-specific fields that don't map to common schema
func extractMerakiExtensions(data map[string]any) map[string]any {
	knownFields := map[string]bool{
		"name": true, "tags": true, "notes": true,
		"lat": true, "lng": true, "ledLightsOn": true,
		"radioSettings": true, "serial": true, "mac": true,
		"model": true, "networkId": true,
		// Common status/metadata fields
		"lanIp": true, "firmware": true, "floorPlanId": true,
		"address": true, "beaconIdParams": true,
	}

	extensions := make(map[string]any)
	for k, v := range data {
		if !knownFields[k] {
			extensions[k] = v
		}
	}

	if len(extensions) == 0 {
		return nil
	}
	return extensions
}

// Helper functions

// tagsToMerakiFormat converts a slice of tags to Meraki's space-separated format
func tagsToMerakiFormat(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	result := ""
	for i, tag := range tags {
		if i > 0 {
			result += " "
		}
		result += tag
	}
	return result
}

// merakiTagsToSlice converts Meraki's space-separated tags to a slice
func merakiTagsToSlice(tags string) []string {
	if tags == "" {
		return nil
	}
	// Simple split by space
	result := make([]string, 0)
	current := ""
	for _, c := range tags {
		if c == ' ' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

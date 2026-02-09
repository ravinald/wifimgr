// Package meraki provides Meraki-specific API conversions.
package meraki

import (
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/vendors"
)

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

// parseRadioSettingsFromMeraki parses Meraki radio settings to vendor-agnostic format.
// Uses RadioTranslator to handle flexRadios -> band_dual conversion.
func parseRadioSettingsFromMeraki(data map[string]any) *vendors.RadioConfig {
	if data == nil {
		return nil
	}

	translator := vendors.NewRadioTranslator()
	return translator.FromMeraki(data)
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

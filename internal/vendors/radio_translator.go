// Package vendors provides vendor-agnostic types and translation utilities.
package vendors

import (
	"fmt"
)

// RadioTranslator handles translation of radio configuration between
// wifimgr's unified format and vendor-specific API formats.
type RadioTranslator struct{}

// NewRadioTranslator creates a new RadioTranslator.
func NewRadioTranslator() *RadioTranslator {
	return &RadioTranslator{}
}

// ToMist converts wifimgr RadioConfig to Mist API format.
// Handles band_dual by setting band_24_usage and band_5_on_24_radio.
func (t *RadioTranslator) ToMist(rc *RadioConfig) map[string]any {
	if rc == nil {
		return nil
	}

	result := rc.ToMap()

	// Handle band_dual translation for Mist
	if rc.BandDual != nil && rc.BandDual.Disabled != nil && !*rc.BandDual.Disabled {
		t.translateBandDualToMist(rc.BandDual, result)
	}

	// Remove band_dual from output (it's Mist-translated to other fields)
	delete(result, "band_dual")

	return result
}

// translateBandDualToMist converts band_dual to Mist's band_24_usage and band_5_on_24_radio.
func (t *RadioTranslator) translateBandDualToMist(dual *DualBandConfig, result map[string]any) {
	if dual.RadioMode == nil {
		return
	}

	radioMode := *dual.RadioMode

	switch radioMode {
	case 24:
		// 2.4GHz mode - use default 2.4GHz band settings
		result["band_24_usage"] = "24"
	case 5:
		// 5GHz mode - set band_24_usage to "5" and configure band_5_on_24_radio
		result["band_24_usage"] = "5"

		band5On24 := make(map[string]any)
		if dual.Disabled != nil {
			band5On24["disabled"] = *dual.Disabled
		}
		if dual.Channel != nil {
			band5On24["channel"] = *dual.Channel
		}
		if dual.Power != nil {
			band5On24["power"] = *dual.Power
		}
		if dual.Bandwidth != nil {
			band5On24["bandwidth"] = *dual.Bandwidth
		}

		if len(band5On24) > 0 {
			result["band_5_on_24_radio"] = band5On24
		}
	case 6:
		// 6GHz mode - not typically supported on Mist dual-band radios
		// This is a Meraki flex radio feature, warn but don't fail
	}
}

// ToMeraki converts wifimgr RadioConfig to Meraki API format.
// Handles band_dual by setting flexRadios configuration.
func (t *RadioTranslator) ToMeraki(rc *RadioConfig) map[string]any {
	if rc == nil {
		return nil
	}

	result := make(map[string]any)

	// Standard band mappings
	if rc.Band24 != nil {
		result["twoFourGhzSettings"] = t.bandConfigToMeraki(rc.Band24)
	}
	if rc.Band5 != nil {
		result["fiveGhzSettings"] = t.bandConfigToMeraki(rc.Band5)
	}
	if rc.Band6 != nil {
		result["sixGhzSettings"] = t.bandConfigToMeraki(rc.Band6)
	}

	// Handle band_dual (flex radio) for Meraki
	if rc.BandDual != nil && rc.BandDual.Disabled != nil && !*rc.BandDual.Disabled {
		t.translateBandDualToMeraki(rc.BandDual, result)
	}

	// Handle RF profile from extension
	if rc.Meraki != nil {
		if rfProfileID, ok := rc.Meraki["rf_profile_id"]; ok {
			result["rfProfileId"] = rfProfileID
		}
	}

	return result
}

// translateBandDualToMeraki converts band_dual to Meraki's flex radio settings.
func (t *RadioTranslator) translateBandDualToMeraki(dual *DualBandConfig, result map[string]any) {
	if dual.RadioMode == nil {
		return
	}

	radioMode := *dual.RadioMode

	// Meraki flex radios toggle between 5GHz and 6GHz
	switch radioMode {
	case 5:
		// 5GHz mode - apply settings to fiveGhzSettings for the flex radio
		if _, exists := result["fiveGhzSettings"]; !exists {
			result["fiveGhzSettings"] = make(map[string]any)
		}
		fiveGhz := result["fiveGhzSettings"].(map[string]any)
		t.applyDualBandSettings(dual, fiveGhz)

		// Set flex radio band selection
		result["flexRadioBand"] = "five"

	case 6:
		// 6GHz mode - apply settings to sixGhzSettings for the flex radio
		if _, exists := result["sixGhzSettings"]; !exists {
			result["sixGhzSettings"] = make(map[string]any)
		}
		sixGhz := result["sixGhzSettings"].(map[string]any)
		t.applyDualBandSettings(dual, sixGhz)

		// Set flex radio band selection
		result["flexRadioBand"] = "six"

	case 24:
		// 2.4GHz mode - not typically supported on Meraki flex radios
		// This is a Mist dual-band radio feature
	}
}

// applyDualBandSettings applies band_dual settings to a band config map.
func (t *RadioTranslator) applyDualBandSettings(dual *DualBandConfig, bandConfig map[string]any) {
	if dual.Channel != nil {
		bandConfig["channel"] = *dual.Channel
	}
	if dual.Power != nil {
		bandConfig["targetPower"] = *dual.Power
	}
	if dual.Bandwidth != nil {
		bandConfig["channelWidth"] = fmt.Sprintf("%d", *dual.Bandwidth)
	}
}

// bandConfigToMeraki converts a RadioBandConfig to Meraki format.
func (t *RadioTranslator) bandConfigToMeraki(cfg *RadioBandConfig) map[string]any {
	if cfg == nil {
		return nil
	}

	result := make(map[string]any)

	if cfg.Channel != nil {
		result["channel"] = *cfg.Channel
	}
	if cfg.Power != nil {
		result["targetPower"] = *cfg.Power
	}
	if cfg.Bandwidth != nil {
		result["channelWidth"] = fmt.Sprintf("%d", *cfg.Bandwidth)
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

// FromMist converts Mist API format to wifimgr RadioConfig.
// Handles band_24_usage and band_5_on_24_radio to band_dual conversion.
func (t *RadioTranslator) FromMist(data map[string]any) *RadioConfig {
	if data == nil {
		return nil
	}

	cfg := &RadioConfig{}

	// Parse global settings
	if allowRRM, ok := data["allow_rrm_disable"].(bool); ok {
		cfg.AllowRRMDisable = &allowRRM
	}
	if scanning, ok := data["scanning_enabled"].(bool); ok {
		cfg.ScanningEnabled = &scanning
	}
	if indoor, ok := data["indoor_use"].(bool); ok {
		cfg.IndoorUse = &indoor
	}
	if antGain24, ok := data["ant_gain_24"].(float64); ok {
		cfg.AntGain24 = &antGain24
	}
	if antGain5, ok := data["ant_gain_5"].(float64); ok {
		cfg.AntGain5 = &antGain5
	}
	if antGain6, ok := data["ant_gain_6"].(float64); ok {
		cfg.AntGain6 = &antGain6
	}
	if antennaMode, ok := data["antenna_mode"].(string); ok {
		cfg.AntennaMode = &antennaMode
	}

	// Parse per-band configs
	if band24, ok := data["band_24"].(map[string]any); ok {
		cfg.Band24 = t.parseBandConfig(band24)
	}
	if band5, ok := data["band_5"].(map[string]any); ok {
		cfg.Band5 = t.parseBandConfig(band5)
	}
	if band6, ok := data["band_6"].(map[string]any); ok {
		cfg.Band6 = t.parseBandConfig(band6)
	}

	// Handle band_24_usage and band_5_on_24_radio -> band_dual conversion
	if band24Usage, ok := data["band_24_usage"].(string); ok {
		cfg.Band24Usage = &band24Usage

		if band24Usage == "5" {
			// Dual-band radio in 5GHz mode
			cfg.BandDual = &DualBandConfig{}
			mode := 5
			cfg.BandDual.RadioMode = &mode
			disabled := false
			cfg.BandDual.Disabled = &disabled

			if band5On24, ok := data["band_5_on_24_radio"].(map[string]any); ok {
				cfg.Band5On24Radio = t.parseBandConfig(band5On24)
				// Also populate band_dual
				if ch, ok := getIntFromMap(band5On24, "channel"); ok {
					cfg.BandDual.Channel = &ch
				}
				if pwr, ok := getIntFromMap(band5On24, "power"); ok {
					cfg.BandDual.Power = &pwr
				}
				if bw, ok := getIntFromMap(band5On24, "bandwidth"); ok {
					cfg.BandDual.Bandwidth = &bw
				}
			}
		} else if band24Usage == "24" {
			// Dual-band radio in 2.4GHz mode
			cfg.BandDual = &DualBandConfig{}
			mode := 24
			cfg.BandDual.RadioMode = &mode
			disabled := false
			cfg.BandDual.Disabled = &disabled
		}
	}

	return cfg
}

// FromMeraki converts Meraki API format to wifimgr RadioConfig.
// Handles flexRadios to band_dual conversion.
func (t *RadioTranslator) FromMeraki(data map[string]any) *RadioConfig {
	if data == nil {
		return nil
	}

	cfg := &RadioConfig{}

	// Parse per-band configs
	if twoFourGhz, ok := data["twoFourGhzSettings"].(map[string]any); ok {
		cfg.Band24 = t.parseMerakiBandConfig(twoFourGhz)
	}
	if fiveGhz, ok := data["fiveGhzSettings"].(map[string]any); ok {
		cfg.Band5 = t.parseMerakiBandConfig(fiveGhz)
	}
	if sixGhz, ok := data["sixGhzSettings"].(map[string]any); ok {
		cfg.Band6 = t.parseMerakiBandConfig(sixGhz)
	}

	// Handle flex radio settings
	if flexBand, ok := data["flexRadioBand"].(string); ok {
		cfg.BandDual = &DualBandConfig{}
		disabled := false
		cfg.BandDual.Disabled = &disabled

		switch flexBand {
		case "five":
			mode := 5
			cfg.BandDual.RadioMode = &mode
			// Copy settings from fiveGhzSettings
			if cfg.Band5 != nil {
				cfg.BandDual.Channel = cfg.Band5.Channel
				cfg.BandDual.Power = cfg.Band5.Power
				cfg.BandDual.Bandwidth = cfg.Band5.Bandwidth
			}
		case "six":
			mode := 6
			cfg.BandDual.RadioMode = &mode
			// Copy settings from sixGhzSettings
			if cfg.Band6 != nil {
				cfg.BandDual.Channel = cfg.Band6.Channel
				cfg.BandDual.Power = cfg.Band6.Power
				cfg.BandDual.Bandwidth = cfg.Band6.Bandwidth
			}
		}
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

// parseBandConfig parses a Mist band config from map.
func (t *RadioTranslator) parseBandConfig(data map[string]any) *RadioBandConfig {
	if data == nil {
		return nil
	}

	cfg := &RadioBandConfig{}

	if disabled, ok := data["disabled"].(bool); ok {
		cfg.Disabled = &disabled
	}
	if ch, ok := getIntFromMap(data, "channel"); ok {
		cfg.Channel = &ch
	}
	if channels, ok := data["channels"].([]any); ok {
		cfg.Channels = interfaceSliceToIntSlice(channels)
	}
	if pwr, ok := getIntFromMap(data, "power"); ok {
		cfg.Power = &pwr
	}
	if pwrMin, ok := getIntFromMap(data, "power_min"); ok {
		cfg.PowerMin = &pwrMin
	}
	if pwrMax, ok := getIntFromMap(data, "power_max"); ok {
		cfg.PowerMax = &pwrMax
	}
	if bw, ok := getIntFromMap(data, "bandwidth"); ok {
		cfg.Bandwidth = &bw
	}
	if antennaMode, ok := data["antenna_mode"].(string); ok {
		cfg.AntennaMode = &antennaMode
	}
	if antGain, ok := data["ant_gain"].(float64); ok {
		cfg.AntGain = &antGain
	}
	if preamble, ok := data["preamble"].(string); ok {
		cfg.Preamble = &preamble
	}

	return cfg
}

// parseMerakiBandConfig parses a Meraki band config from map.
func (t *RadioTranslator) parseMerakiBandConfig(data map[string]any) *RadioBandConfig {
	if data == nil {
		return nil
	}

	cfg := &RadioBandConfig{}

	if ch, ok := getIntFromMap(data, "channel"); ok {
		cfg.Channel = &ch
	}
	if targetPower, ok := getIntFromMap(data, "targetPower"); ok {
		cfg.Power = &targetPower
	}
	if channelWidth, ok := getIntFromMap(data, "channelWidth"); ok {
		cfg.Bandwidth = &channelWidth
	}

	// Meraki-specific fields
	cfg.Meraki = make(map[string]any)
	if minBitrate, ok := data["minBitrate"]; ok {
		cfg.Meraki["min_bitrate"] = minBitrate
	}
	if rxsop, ok := data["rxsop"]; ok {
		cfg.Meraki["rxsop"] = rxsop
	}

	if len(cfg.Meraki) == 0 {
		cfg.Meraki = nil
	}

	return cfg
}

// getIntFromMap extracts an int from a map, handling float64 JSON values.
func getIntFromMap(m map[string]any, key string) (int, bool) {
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

// interfaceSliceToIntSlice converts []any to []int.
func interfaceSliceToIntSlice(in []any) []int {
	result := make([]int, 0, len(in))
	for _, v := range in {
		if f, ok := v.(float64); ok {
			result = append(result, int(f))
		}
	}
	return result
}

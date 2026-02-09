package vendors

import (
	"reflect"
	"testing"
)

func TestRadioTranslator_ToMist(t *testing.T) {
	translator := NewRadioTranslator()

	t.Run("nil config", func(t *testing.T) {
		result := translator.ToMist(nil)
		if result != nil {
			t.Errorf("ToMist(nil) should return nil, got %v", result)
		}
	})

	t.Run("band_dual 5GHz mode", func(t *testing.T) {
		mode := 5
		disabled := false
		channel := 149
		power := 12
		bandwidth := 40

		cfg := &RadioConfig{
			BandDual: &DualBandConfig{
				Disabled:  &disabled,
				RadioMode: &mode,
				Channel:   &channel,
				Power:     &power,
				Bandwidth: &bandwidth,
			},
		}

		result := translator.ToMist(cfg)

		// Should have band_24_usage = "5"
		if result["band_24_usage"] != "5" {
			t.Errorf("band_24_usage should be '5', got %v", result["band_24_usage"])
		}

		// Should have band_5_on_24_radio with settings
		band5On24, ok := result["band_5_on_24_radio"].(map[string]any)
		if !ok {
			t.Error("band_5_on_24_radio should be a map")
		} else {
			if band5On24["channel"] != 149 {
				t.Errorf("band_5_on_24_radio.channel should be 149, got %v", band5On24["channel"])
			}
			if band5On24["power"] != 12 {
				t.Errorf("band_5_on_24_radio.power should be 12, got %v", band5On24["power"])
			}
			if band5On24["bandwidth"] != 40 {
				t.Errorf("band_5_on_24_radio.bandwidth should be 40, got %v", band5On24["bandwidth"])
			}
		}

		// Should NOT have band_dual in output
		if _, exists := result["band_dual"]; exists {
			t.Error("band_dual should be removed from Mist output")
		}
	})

	t.Run("band_dual 24GHz mode", func(t *testing.T) {
		mode := 24
		disabled := false

		cfg := &RadioConfig{
			BandDual: &DualBandConfig{
				Disabled:  &disabled,
				RadioMode: &mode,
			},
		}

		result := translator.ToMist(cfg)

		if result["band_24_usage"] != "24" {
			t.Errorf("band_24_usage should be '24', got %v", result["band_24_usage"])
		}
	})
}

func TestRadioTranslator_ToMeraki(t *testing.T) {
	translator := NewRadioTranslator()

	t.Run("nil config", func(t *testing.T) {
		result := translator.ToMeraki(nil)
		if result != nil {
			t.Errorf("ToMeraki(nil) should return nil, got %v", result)
		}
	})

	t.Run("band_dual 6GHz mode", func(t *testing.T) {
		mode := 6
		disabled := false
		channel := 37
		power := 15
		bandwidth := 160

		cfg := &RadioConfig{
			BandDual: &DualBandConfig{
				Disabled:  &disabled,
				RadioMode: &mode,
				Channel:   &channel,
				Power:     &power,
				Bandwidth: &bandwidth,
			},
		}

		result := translator.ToMeraki(cfg)

		// Should have flexRadioBand = "six"
		if result["flexRadioBand"] != "six" {
			t.Errorf("flexRadioBand should be 'six', got %v", result["flexRadioBand"])
		}

		// Should have sixGhzSettings with settings
		sixGhz, ok := result["sixGhzSettings"].(map[string]any)
		if !ok {
			t.Error("sixGhzSettings should be a map")
		} else {
			if sixGhz["channel"] != 37 {
				t.Errorf("sixGhzSettings.channel should be 37, got %v", sixGhz["channel"])
			}
			if sixGhz["targetPower"] != 15 {
				t.Errorf("sixGhzSettings.targetPower should be 15, got %v", sixGhz["targetPower"])
			}
			if sixGhz["channelWidth"] != "160" {
				t.Errorf("sixGhzSettings.channelWidth should be '160', got %v", sixGhz["channelWidth"])
			}
		}
	})

	t.Run("band_dual 5GHz mode", func(t *testing.T) {
		mode := 5
		disabled := false

		cfg := &RadioConfig{
			BandDual: &DualBandConfig{
				Disabled:  &disabled,
				RadioMode: &mode,
			},
		}

		result := translator.ToMeraki(cfg)

		if result["flexRadioBand"] != "five" {
			t.Errorf("flexRadioBand should be 'five', got %v", result["flexRadioBand"])
		}
	})

	t.Run("standard bands", func(t *testing.T) {
		ch24 := 6
		ch5 := 36
		pwr := 12

		cfg := &RadioConfig{
			Band24: &RadioBandConfig{
				Channel: &ch24,
				Power:   &pwr,
			},
			Band5: &RadioBandConfig{
				Channel: &ch5,
				Power:   &pwr,
			},
		}

		result := translator.ToMeraki(cfg)

		twoFour, ok := result["twoFourGhzSettings"].(map[string]any)
		if !ok {
			t.Error("twoFourGhzSettings should be a map")
		} else {
			if twoFour["channel"] != 6 {
				t.Errorf("twoFourGhzSettings.channel should be 6, got %v", twoFour["channel"])
			}
		}

		five, ok := result["fiveGhzSettings"].(map[string]any)
		if !ok {
			t.Error("fiveGhzSettings should be a map")
		} else {
			if five["channel"] != 36 {
				t.Errorf("fiveGhzSettings.channel should be 36, got %v", five["channel"])
			}
		}
	})
}

func TestRadioTranslator_FromMist(t *testing.T) {
	translator := NewRadioTranslator()

	t.Run("nil data", func(t *testing.T) {
		result := translator.FromMist(nil)
		if result != nil {
			t.Errorf("FromMist(nil) should return nil, got %v", result)
		}
	})

	t.Run("band_24_usage 5 with band_5_on_24_radio", func(t *testing.T) {
		data := map[string]any{
			"band_24_usage": "5",
			"band_5_on_24_radio": map[string]any{
				"channel":   float64(149),
				"power":     float64(12),
				"bandwidth": float64(40),
			},
		}

		result := translator.FromMist(data)

		if result.BandDual == nil {
			t.Fatal("BandDual should be set")
		}
		if result.BandDual.RadioMode == nil || *result.BandDual.RadioMode != 5 {
			t.Errorf("BandDual.RadioMode should be 5, got %v", result.BandDual.RadioMode)
		}
		if result.BandDual.Channel == nil || *result.BandDual.Channel != 149 {
			t.Errorf("BandDual.Channel should be 149, got %v", result.BandDual.Channel)
		}
	})

	t.Run("standard bands", func(t *testing.T) {
		data := map[string]any{
			"band_24": map[string]any{
				"channel":  float64(6),
				"power":    float64(5),
				"disabled": false,
			},
			"band_5": map[string]any{
				"channel":   float64(36),
				"power":     float64(12),
				"bandwidth": float64(80),
			},
		}

		result := translator.FromMist(data)

		if result.Band24 == nil {
			t.Fatal("Band24 should be set")
		}
		if result.Band24.Channel == nil || *result.Band24.Channel != 6 {
			t.Errorf("Band24.Channel should be 6, got %v", result.Band24.Channel)
		}

		if result.Band5 == nil {
			t.Fatal("Band5 should be set")
		}
		if result.Band5.Channel == nil || *result.Band5.Channel != 36 {
			t.Errorf("Band5.Channel should be 36, got %v", result.Band5.Channel)
		}
	})
}

func TestRadioTranslator_FromMeraki(t *testing.T) {
	translator := NewRadioTranslator()

	t.Run("nil data", func(t *testing.T) {
		result := translator.FromMeraki(nil)
		if result != nil {
			t.Errorf("FromMeraki(nil) should return nil, got %v", result)
		}
	})

	t.Run("flexRadioBand six", func(t *testing.T) {
		data := map[string]any{
			"flexRadioBand": "six",
			"sixGhzSettings": map[string]any{
				"channel":      float64(37),
				"targetPower":  float64(15),
				"channelWidth": float64(160),
			},
		}

		result := translator.FromMeraki(data)

		if result.BandDual == nil {
			t.Fatal("BandDual should be set")
		}
		if result.BandDual.RadioMode == nil || *result.BandDual.RadioMode != 6 {
			t.Errorf("BandDual.RadioMode should be 6, got %v", result.BandDual.RadioMode)
		}
		if result.BandDual.Channel == nil || *result.BandDual.Channel != 37 {
			t.Errorf("BandDual.Channel should be 37, got %v", result.BandDual.Channel)
		}
	})

	t.Run("standard bands", func(t *testing.T) {
		data := map[string]any{
			"twoFourGhzSettings": map[string]any{
				"channel":     float64(6),
				"targetPower": float64(5),
			},
			"fiveGhzSettings": map[string]any{
				"channel":      float64(36),
				"targetPower":  float64(12),
				"channelWidth": float64(80),
			},
		}

		result := translator.FromMeraki(data)

		if result.Band24 == nil {
			t.Fatal("Band24 should be set")
		}
		if result.Band24.Channel == nil || *result.Band24.Channel != 6 {
			t.Errorf("Band24.Channel should be 6, got %v", result.Band24.Channel)
		}
		if result.Band24.Power == nil || *result.Band24.Power != 5 {
			t.Errorf("Band24.Power should be 5, got %v", result.Band24.Power)
		}

		if result.Band5 == nil {
			t.Fatal("Band5 should be set")
		}
		if result.Band5.Channel == nil || *result.Band5.Channel != 36 {
			t.Errorf("Band5.Channel should be 36, got %v", result.Band5.Channel)
		}
	})
}

func TestDualBandConfig_ToMap(t *testing.T) {
	disabled := false
	radioMode := 5
	channel := 149
	power := 12
	bandwidth := 40

	cfg := &DualBandConfig{
		Disabled:  &disabled,
		RadioMode: &radioMode,
		Channel:   &channel,
		Power:     &power,
		Bandwidth: &bandwidth,
	}

	result := cfg.ToMap()

	expected := map[string]any{
		"disabled":   false,
		"radio_mode": 5,
		"channel":    149,
		"power":      12,
		"bandwidth":  40,
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("ToMap() = %v, want %v", result, expected)
	}
}

func TestDualBandConfig_ToMap_Nil(t *testing.T) {
	var cfg *DualBandConfig
	result := cfg.ToMap()
	if result != nil {
		t.Errorf("ToMap() on nil should return nil, got %v", result)
	}
}

package meraki

import (
	"testing"
)

func TestExportAPConfig(t *testing.T) {
	tests := []struct {
		name       string
		rawConfig  map[string]any
		wantName   string
		wantMeraki bool // expect meraki extension block to be populated
	}{
		{
			name:      "nil config",
			rawConfig: nil,
			wantName:  "",
		},
		{
			name: "basic config with name",
			rawConfig: map[string]any{
				"name": "AP-Lobby-01",
			},
			wantName: "AP-Lobby-01",
		},
		{
			name: "config with meraki-specific fields",
			rawConfig: map[string]any{
				"name":      "AP-Test",
				"serial":    "QXYZ-1234-ABCD",
				"networkId": "N_12345",
				"firmware":  "MR 29.6",
				"model":     "MR46",
			},
			wantName:   "AP-Test",
			wantMeraki: true,
		},
		{
			name: "config with location",
			rawConfig: map[string]any{
				"name": "AP-Location",
				"lat":  37.7749,
				"lng":  -122.4194,
			},
			wantName: "AP-Location",
		},
		{
			name: "config with floor plan",
			rawConfig: map[string]any{
				"name":        "AP-Floor",
				"floorPlanId": "fp123",
			},
			wantName:   "AP-Floor",
			wantMeraki: true,
		},
		{
			name: "config with RF profile",
			rawConfig: map[string]any{
				"name":        "AP-RF",
				"rfProfileId": "rf-profile-123",
			},
			wantName: "AP-RF",
		},
		{
			name: "config with radio settings",
			rawConfig: map[string]any{
				"name": "AP-Radio",
				"twoFourGhzSettings": map[string]any{
					"channel":     float64(6),
					"targetPower": float64(15),
				},
				"fiveGhzSettings": map[string]any{
					"channel":     float64(36),
					"targetPower": float64(17),
				},
			},
			wantName: "AP-Radio",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExportAPConfig(tt.rawConfig)

			if tt.rawConfig == nil {
				if result != nil {
					t.Error("expected nil result for nil config")
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			if result.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", result.Name, tt.wantName)
			}

			if tt.wantMeraki && len(result.Meraki) == 0 {
				t.Error("expected Meraki extension block to be populated")
			}
		})
	}
}

func TestExportMerakiRadioConfig(t *testing.T) {
	tests := []struct {
		name       string
		rawConfig  map[string]any
		wantBand24 bool
		wantBand5  bool
		wantBand6  bool
	}{
		{
			name: "2.4GHz settings only",
			rawConfig: map[string]any{
				"twoFourGhzSettings": map[string]any{
					"channel":     float64(6),
					"targetPower": float64(15),
				},
			},
			wantBand24: true,
		},
		{
			name: "dual band settings",
			rawConfig: map[string]any{
				"twoFourGhzSettings": map[string]any{
					"channel": float64(11),
				},
				"fiveGhzSettings": map[string]any{
					"channel": float64(36),
				},
			},
			wantBand24: true,
			wantBand5:  true,
		},
		{
			name: "tri-band with 6GHz",
			rawConfig: map[string]any{
				"twoFourGhzSettings": map[string]any{"channel": float64(1)},
				"fiveGhzSettings":    map[string]any{"channel": float64(44)},
				"sixGhzSettings":     map[string]any{"channel": float64(5)},
			},
			wantBand24: true,
			wantBand5:  true,
			wantBand6:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := exportMerakiRadioConfig(tt.rawConfig)

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			if tt.wantBand24 && result.Band24 == nil {
				t.Error("expected Band24 to be set")
			}
			if tt.wantBand5 && result.Band5 == nil {
				t.Error("expected Band5 to be set")
			}
			if tt.wantBand6 && result.Band6 == nil {
				t.Error("expected Band6 to be set")
			}
		})
	}
}

func TestExportMerakiBandConfig(t *testing.T) {
	tests := []struct {
		name        string
		rawConfig   map[string]any
		wantChannel int
		wantPower   int
	}{
		{
			name: "basic band config",
			rawConfig: map[string]any{
				"channel":     float64(6),
				"targetPower": float64(15),
			},
			wantChannel: 6,
			wantPower:   15,
		},
		{
			name: "band config with channel width",
			rawConfig: map[string]any{
				"channel":      float64(36),
				"targetPower":  float64(17),
				"channelWidth": "80",
			},
			wantChannel: 36,
			wantPower:   17,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := exportMerakiBandConfig(tt.rawConfig)

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			if result.Channel == nil || *result.Channel != tt.wantChannel {
				t.Errorf("Channel = %v, want %d", result.Channel, tt.wantChannel)
			}

			if result.Power == nil || *result.Power != tt.wantPower {
				t.Errorf("Power = %v, want %d", result.Power, tt.wantPower)
			}
		})
	}
}

func TestMerakiSpecificFieldsExtraction(t *testing.T) {
	rawConfig := map[string]any{
		"name":        "AP-Test",
		"serial":      "QXYZ-1234-ABCD",
		"networkId":   "N_12345",
		"productType": "wireless",
		"model":       "MR46",
		"firmware":    "MR 29.6",
		"url":         "https://dashboard.meraki.com/...",
		"lat":         37.7749,
		"lng":         -122.4194,
	}

	result := ExportAPConfig(rawConfig)

	// Verify Meraki-specific fields are in the meraki block
	if result.Meraki == nil {
		t.Fatal("expected Meraki extension block")
	}

	expectedMerakiFields := []string{
		"serial", "networkId", "productType", "model", "firmware", "url",
	}

	for _, field := range expectedMerakiFields {
		if _, ok := result.Meraki[field]; !ok {
			t.Errorf("expected field %q in Meraki block", field)
		}
	}

	// Verify common fields are NOT in meraki block
	if _, ok := result.Meraki["name"]; ok {
		t.Error("'name' should not be in Meraki block (it's a common field)")
	}

	// Verify location is extracted to common format
	if len(result.Location) != 2 {
		t.Error("expected location to be extracted")
	}
	if result.Location[0] != 37.7749 || result.Location[1] != -122.4194 {
		t.Errorf("unexpected location: %v", result.Location)
	}
}

func TestMerakiChannelWidthConversion(t *testing.T) {
	tests := []struct {
		width         string
		wantBandwidth int
	}{
		{"20", 20},
		{"40", 40},
		{"80", 80},
		{"160", 160},
	}

	for _, tt := range tests {
		t.Run(tt.width, func(t *testing.T) {
			rawConfig := map[string]any{
				"channelWidth": tt.width,
			}

			result := exportMerakiBandConfig(rawConfig)

			if result.Bandwidth == nil || *result.Bandwidth != tt.wantBandwidth {
				t.Errorf("Bandwidth = %v, want %d", result.Bandwidth, tt.wantBandwidth)
			}
		})
	}
}

func TestMerakiAutoChannelWidth(t *testing.T) {
	rawConfig := map[string]any{
		"channelWidth": "auto",
	}

	result := exportMerakiBandConfig(rawConfig)

	if result.Bandwidth != nil {
		t.Error("expected Bandwidth to be nil for 'auto' width")
	}

	if result.Meraki == nil || result.Meraki["bandwidth_auto"] != true {
		t.Error("expected bandwidth_auto in Meraki block")
	}
}

func TestMerakiToStringSlice(t *testing.T) {
	input := []any{"tag1", "tag2", "tag3"}
	result := toStringSlice(input)

	if len(result) != 3 {
		t.Fatalf("expected 3 items, got %d", len(result))
	}

	if result[0] != "tag1" || result[1] != "tag2" || result[2] != "tag3" {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestMerakiToIntSlice(t *testing.T) {
	input := []any{float64(1), float64(6), float64(11)}
	result := toIntSlice(input)

	if len(result) != 3 {
		t.Fatalf("expected 3 items, got %d", len(result))
	}

	if result[0] != 1 || result[1] != 6 || result[2] != 11 {
		t.Errorf("unexpected result: %v", result)
	}
}

package mist

import (
	"testing"
)

func TestExportAPConfig(t *testing.T) {
	tests := []struct {
		name      string
		rawConfig map[string]any
		wantName  string
		wantNotes string
		wantMist  bool // expect mist extension block to be populated
	}{
		{
			name:      "nil config",
			rawConfig: nil,
			wantName:  "",
		},
		{
			name: "basic config with name and notes",
			rawConfig: map[string]any{
				"name":  "AP-Lobby-01",
				"notes": "Lobby access point",
			},
			wantName:  "AP-Lobby-01",
			wantNotes: "Lobby access point",
		},
		{
			name: "config with mist-specific fields",
			rawConfig: map[string]any{
				"name":       "AP-Test",
				"aeroscout":  map[string]any{"enabled": true},
				"centrak":    map[string]any{"enabled": false},
				"org_id":     "org123",
				"site_id":    "site456",
				"created_at": 1234567890,
			},
			wantName: "AP-Test",
			wantMist: true,
		},
		{
			name: "config with radio settings",
			rawConfig: map[string]any{
				"name": "AP-Radio-Test",
				"radio_config": map[string]any{
					"band_24": map[string]any{
						"channel": float64(6),
						"power":   float64(15),
					},
					"band_5": map[string]any{
						"channel": float64(36),
						"power":   float64(17),
					},
				},
			},
			wantName: "AP-Radio-Test",
		},
		{
			name: "config with location",
			rawConfig: map[string]any{
				"name": "AP-Location",
				"latlng": map[string]any{
					"lat": 37.7749,
					"lng": -122.4194,
				},
				"map_id": "map123",
				"x":      100.5,
				"y":      200.75,
			},
			wantName: "AP-Location",
		},
		{
			name: "config with device profile",
			rawConfig: map[string]any{
				"name":             "AP-Profile",
				"deviceprofile_id": "profile123",
			},
			wantName: "AP-Profile",
		},
		{
			name: "config with IP settings",
			rawConfig: map[string]any{
				"name": "AP-IP",
				"ip_config": map[string]any{
					"type":    "static",
					"ip":      "192.168.1.100",
					"netmask": "255.255.255.0",
					"gateway": "192.168.1.1",
					"dns":     []any{"8.8.8.8", "8.8.4.4"},
				},
			},
			wantName: "AP-IP",
		},
		{
			name: "config with tags",
			rawConfig: map[string]any{
				"name": "AP-Tags",
				"tags": []any{"production", "floor-2", "building-a"},
			},
			wantName: "AP-Tags",
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

			if tt.wantNotes != "" && result.Notes != tt.wantNotes {
				t.Errorf("Notes = %q, want %q", result.Notes, tt.wantNotes)
			}

			if tt.wantMist && len(result.Mist) == 0 {
				t.Error("expected Mist extension block to be populated")
			}
		})
	}
}

func TestExportRadioConfig(t *testing.T) {
	tests := []struct {
		name       string
		rawConfig  map[string]any
		wantBand24 bool
		wantBand5  bool
		wantBand6  bool
	}{
		{
			name:      "nil config",
			rawConfig: nil,
		},
		{
			name: "2.4GHz only",
			rawConfig: map[string]any{
				"band_24": map[string]any{
					"channel": float64(6),
					"power":   float64(15),
				},
			},
			wantBand24: true,
		},
		{
			name: "dual band",
			rawConfig: map[string]any{
				"band_24": map[string]any{
					"channel": float64(11),
				},
				"band_5": map[string]any{
					"channel": float64(36),
				},
			},
			wantBand24: true,
			wantBand5:  true,
		},
		{
			name: "tri-band with 6GHz",
			rawConfig: map[string]any{
				"band_24": map[string]any{"channel": float64(1)},
				"band_5":  map[string]any{"channel": float64(44)},
				"band_6":  map[string]any{"channel": float64(5)},
			},
			wantBand24: true,
			wantBand5:  true,
			wantBand6:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := exportRadioConfig(tt.rawConfig)

			if tt.rawConfig == nil {
				if result != nil {
					t.Error("expected nil result for nil config")
				}
				return
			}

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

func TestExportRadioBandConfig(t *testing.T) {
	tests := []struct {
		name        string
		rawConfig   map[string]any
		wantChannel int
		wantPower   int
	}{
		{
			name: "basic band config",
			rawConfig: map[string]any{
				"channel": float64(6),
				"power":   float64(15),
			},
			wantChannel: 6,
			wantPower:   15,
		},
		{
			name: "band config with bandwidth",
			rawConfig: map[string]any{
				"channel":   float64(36),
				"power":     float64(17),
				"bandwidth": float64(80),
			},
			wantChannel: 36,
			wantPower:   17,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := exportRadioBandConfig(tt.rawConfig)

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

func TestMistSpecificFieldsExtraction(t *testing.T) {
	rawConfig := map[string]any{
		"name":                 "AP-Test",
		"notes":                "Test notes",
		"aeroscout":            map[string]any{"enabled": true},
		"centrak":              map[string]any{"enabled": false},
		"use_auto_orientation": true,
		"use_auto_placement":   false,
		"client_bridge":        map[string]any{"enabled": true},
		"org_id":               "org123",
		"site_id":              "site456",
		"id":                   "device789",
		"mac":                  "aabbccddeeff",
	}

	result := ExportAPConfig(rawConfig)

	// Verify Mist-specific fields are in the mist block
	if result.Mist == nil {
		t.Fatal("expected Mist extension block")
	}

	expectedMistFields := []string{
		"aeroscout", "centrak", "use_auto_orientation", "use_auto_placement",
		"client_bridge", "org_id", "site_id", "id", "mac",
	}

	for _, field := range expectedMistFields {
		if _, ok := result.Mist[field]; !ok {
			t.Errorf("expected field %q in Mist block", field)
		}
	}

	// Verify common fields are NOT in mist block
	if _, ok := result.Mist["name"]; ok {
		t.Error("'name' should not be in Mist block (it's a common field)")
	}
	if _, ok := result.Mist["notes"]; ok {
		t.Error("'notes' should not be in Mist block (it's a common field)")
	}
}

func TestToStringSlice(t *testing.T) {
	input := []any{"tag1", "tag2", "tag3"}
	result := toStringSlice(input)

	if len(result) != 3 {
		t.Fatalf("expected 3 items, got %d", len(result))
	}

	if result[0] != "tag1" || result[1] != "tag2" || result[2] != "tag3" {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestToIntSlice(t *testing.T) {
	input := []any{float64(1), float64(6), float64(11)}
	result := toIntSlice(input)

	if len(result) != 3 {
		t.Fatalf("expected 3 items, got %d", len(result))
	}

	if result[0] != 1 || result[1] != 6 || result[2] != 11 {
		t.Errorf("unexpected result: %v", result)
	}
}

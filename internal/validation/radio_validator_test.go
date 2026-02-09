package validation

import (
	"testing"
)

func TestRadioValidator_ValidateBand24(t *testing.T) {
	validator := NewRadioValidator("mist", "")

	tests := []struct {
		name       string
		config     map[string]any
		wantIssues int
	}{
		{
			name: "valid config",
			config: map[string]any{
				"band_24": map[string]any{
					"disabled": false,
					"channel":  float64(6),
					"power":    float64(5),
				},
			},
			wantIssues: 0,
		},
		{
			name: "invalid channel",
			config: map[string]any{
				"band_24": map[string]any{
					"channel": float64(15),
				},
			},
			wantIssues: 1,
		},
		{
			name: "invalid power",
			config: map[string]any{
				"band_24": map[string]any{
					"power": float64(50),
				},
			},
			wantIssues: 1,
		},
		{
			name: "invalid bandwidth for 2.4GHz",
			config: map[string]any{
				"band_24": map[string]any{
					"bandwidth": float64(40),
				},
			},
			wantIssues: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := validator.ValidateRadioConfig(tt.config)
			if len(issues) != tt.wantIssues {
				t.Errorf("ValidateRadioConfig() got %d issues, want %d: %v", len(issues), tt.wantIssues, issues)
			}
		})
	}
}

func TestRadioValidator_ValidateBand5(t *testing.T) {
	validator := NewRadioValidator("mist", "")

	tests := []struct {
		name       string
		config     map[string]any
		wantIssues int
	}{
		{
			name: "valid config",
			config: map[string]any{
				"band_5": map[string]any{
					"disabled":  false,
					"channel":   float64(36),
					"power":     float64(12),
					"bandwidth": float64(80),
				},
			},
			wantIssues: 0,
		},
		{
			name: "invalid channel for 5GHz",
			config: map[string]any{
				"band_5": map[string]any{
					"channel": float64(6), // 2.4GHz channel
				},
			},
			wantIssues: 1,
		},
		{
			name: "valid 160MHz bandwidth",
			config: map[string]any{
				"band_5": map[string]any{
					"bandwidth": float64(160),
				},
			},
			wantIssues: 0,
		},
		{
			name: "invalid 320MHz bandwidth for 5GHz",
			config: map[string]any{
				"band_5": map[string]any{
					"bandwidth": float64(320),
				},
			},
			wantIssues: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := validator.ValidateRadioConfig(tt.config)
			if len(issues) != tt.wantIssues {
				t.Errorf("ValidateRadioConfig() got %d issues, want %d: %v", len(issues), tt.wantIssues, issues)
			}
		})
	}
}

func TestRadioValidator_ValidateBand6(t *testing.T) {
	validator := NewRadioValidator("meraki", "")

	tests := []struct {
		name       string
		config     map[string]any
		wantIssues int
	}{
		{
			name: "valid 6GHz config",
			config: map[string]any{
				"band_6": map[string]any{
					"disabled":  false,
					"channel":   float64(37),
					"power":     float64(15),
					"bandwidth": float64(160),
				},
			},
			wantIssues: 0,
		},
		{
			name: "valid 320MHz for 6GHz",
			config: map[string]any{
				"band_6": map[string]any{
					"bandwidth": float64(320),
				},
			},
			wantIssues: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := validator.ValidateRadioConfig(tt.config)
			if len(issues) != tt.wantIssues {
				t.Errorf("ValidateRadioConfig() got %d issues, want %d: %v", len(issues), tt.wantIssues, issues)
			}
		})
	}
}

func TestRadioValidator_ValidateBandDual(t *testing.T) {
	tests := []struct {
		name       string
		vendor     string
		config     map[string]any
		wantIssues int
	}{
		{
			name:   "valid Mist dual-band 5GHz mode",
			vendor: "mist",
			config: map[string]any{
				"band_dual": map[string]any{
					"disabled":   false,
					"radio_mode": float64(5),
					"channel":    float64(149),
					"power":      float64(12),
					"bandwidth":  float64(40),
				},
			},
			wantIssues: 0,
		},
		{
			name:   "valid Mist dual-band 24GHz mode",
			vendor: "mist",
			config: map[string]any{
				"band_dual": map[string]any{
					"disabled":   false,
					"radio_mode": float64(24),
					"channel":    float64(6),
					"power":      float64(5),
				},
			},
			wantIssues: 0,
		},
		{
			name:   "invalid Mist dual-band 6GHz mode",
			vendor: "mist",
			config: map[string]any{
				"band_dual": map[string]any{
					"disabled":   false,
					"radio_mode": float64(6), // Mist doesn't support 6GHz dual-band
				},
			},
			wantIssues: 1,
		},
		{
			name:   "valid Meraki flex radio 6GHz mode",
			vendor: "meraki",
			config: map[string]any{
				"band_dual": map[string]any{
					"disabled":   false,
					"radio_mode": float64(6),
					"channel":    float64(37),
					"power":      float64(15),
					"bandwidth":  float64(160),
				},
			},
			wantIssues: 0,
		},
		{
			name:   "invalid Meraki flex radio 24GHz mode",
			vendor: "meraki",
			config: map[string]any{
				"band_dual": map[string]any{
					"disabled":   false,
					"radio_mode": float64(24), // Meraki doesn't support 24GHz flex
				},
			},
			wantIssues: 1,
		},
		{
			name:   "missing radio_mode with settings",
			vendor: "mist",
			config: map[string]any{
				"band_dual": map[string]any{
					"channel": float64(36),
					"power":   float64(12),
				},
			},
			wantIssues: 1, // radio_mode required when settings present
		},
		{
			name:   "channel mismatch for radio_mode",
			vendor: "mist",
			config: map[string]any{
				"band_dual": map[string]any{
					"radio_mode": float64(5),
					"channel":    float64(6), // 2.4GHz channel with 5GHz mode
				},
			},
			wantIssues: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewRadioValidator(tt.vendor, "")
			issues := validator.ValidateRadioConfig(tt.config)
			if len(issues) != tt.wantIssues {
				t.Errorf("ValidateRadioConfig() got %d issues, want %d: %v", len(issues), tt.wantIssues, issues)
			}
		})
	}
}

func TestRadioValidator_Band5On24Radio(t *testing.T) {
	tests := []struct {
		name       string
		vendor     string
		config     map[string]any
		wantIssues int
	}{
		{
			name:   "valid Mist band_5_on_24_radio",
			vendor: "mist",
			config: map[string]any{
				"band_5_on_24_radio": map[string]any{
					"channel":   float64(149),
					"power":     float64(12),
					"bandwidth": float64(40),
				},
			},
			wantIssues: 0,
		},
		{
			name:   "invalid Meraki using band_5_on_24_radio",
			vendor: "meraki",
			config: map[string]any{
				"band_5_on_24_radio": map[string]any{
					"channel": float64(149),
				},
			},
			wantIssues: 1, // Mist-specific field
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewRadioValidator(tt.vendor, "")
			issues := validator.ValidateRadioConfig(tt.config)
			if len(issues) != tt.wantIssues {
				t.Errorf("ValidateRadioConfig() got %d issues, want %d: %v", len(issues), tt.wantIssues, issues)
			}
		})
	}
}

func TestRadioValidator_EmptyConfig(t *testing.T) {
	validator := NewRadioValidator("mist", "")

	issues := validator.ValidateRadioConfig(nil)
	if issues != nil {
		t.Errorf("ValidateRadioConfig(nil) should return nil, got %v", issues)
	}

	issues = validator.ValidateRadioConfig(map[string]any{})
	if len(issues) != 0 {
		t.Errorf("ValidateRadioConfig(empty) should return no issues, got %v", issues)
	}
}

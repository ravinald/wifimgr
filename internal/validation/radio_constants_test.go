package validation

import (
	"testing"
)

func TestIsValidChannel(t *testing.T) {
	tests := []struct {
		name    string
		band    string
		channel int
		want    bool
	}{
		// 2.4 GHz valid channels
		{"band_24 channel 1", "band_24", 1, true},
		{"band_24 channel 6", "band_24", 6, true},
		{"band_24 channel 11", "band_24", 11, true},
		{"band_24 channel 12", "band_24", 12, false}, // Not valid in US
		{"band_24 channel 15", "band_24", 15, false},

		// 5 GHz valid channels
		{"band_5 channel 36", "band_5", 36, true},
		{"band_5 channel 44", "band_5", 44, true},
		{"band_5 channel 149", "band_5", 149, true},
		{"band_5 channel 165", "band_5", 165, true},
		{"band_5 channel 14", "band_5", 14, false},   // Not valid for 5GHz
		{"band_5 channel 200", "band_5", 200, false}, // Not valid

		// 6 GHz valid channels
		{"band_6 channel 1", "band_6", 1, true},
		{"band_6 channel 5", "band_6", 5, true},
		{"band_6 channel 233", "band_6", 233, true},
		{"band_6 channel 2", "band_6", 2, false}, // Not on 4-spacing
		{"band_6 channel 3", "band_6", 3, false},

		// Short band names
		{"24 channel 6", "24", 6, true},
		{"5 channel 36", "5", 36, true},
		{"6 channel 1", "6", 1, true},

		// Unknown band
		{"unknown band", "band_99", 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidChannel(tt.band, tt.channel)
			if got != tt.want {
				t.Errorf("IsValidChannel(%q, %d) = %v, want %v", tt.band, tt.channel, got, tt.want)
			}
		})
	}
}

func TestIsValidBandwidth(t *testing.T) {
	tests := []struct {
		name      string
		band      string
		bandwidth int
		want      bool
	}{
		// 2.4 GHz (only 20MHz)
		{"band_24 20MHz", "band_24", 20, true},
		{"band_24 40MHz", "band_24", 40, false},

		// 5 GHz (up to 160MHz)
		{"band_5 20MHz", "band_5", 20, true},
		{"band_5 40MHz", "band_5", 40, true},
		{"band_5 80MHz", "band_5", 80, true},
		{"band_5 160MHz", "band_5", 160, true},
		{"band_5 320MHz", "band_5", 320, false},

		// 6 GHz (up to 320MHz)
		{"band_6 20MHz", "band_6", 20, true},
		{"band_6 40MHz", "band_6", 40, true},
		{"band_6 160MHz", "band_6", 160, true},
		{"band_6 320MHz", "band_6", 320, true},

		// Invalid bandwidth
		{"band_5 50MHz", "band_5", 50, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidBandwidth(tt.band, tt.bandwidth)
			if got != tt.want {
				t.Errorf("IsValidBandwidth(%q, %d) = %v, want %v", tt.band, tt.bandwidth, got, tt.want)
			}
		})
	}
}

func TestIsValidPower(t *testing.T) {
	tests := []struct {
		name  string
		power int
		want  bool
	}{
		{"min power", 1, true},
		{"mid power", 15, true},
		{"max power", 30, true},
		{"below min", 0, false},
		{"above max", 31, false},
		{"negative", -5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidPower(tt.power)
			if got != tt.want {
				t.Errorf("IsValidPower(%d) = %v, want %v", tt.power, got, tt.want)
			}
		})
	}
}

func TestIsValidRadioMode(t *testing.T) {
	tests := []struct {
		name   string
		vendor string
		mode   int
		want   bool
	}{
		// Mist: 24 or 5
		{"mist mode 24", "mist", 24, true},
		{"mist mode 5", "mist", 5, true},
		{"mist mode 6", "mist", 6, false}, // Not valid for Mist

		// Meraki: 5 or 6
		{"meraki mode 5", "meraki", 5, true},
		{"meraki mode 6", "meraki", 6, true},
		{"meraki mode 24", "meraki", 24, false}, // Not valid for Meraki

		// Unknown vendor: allow all
		{"unknown mode 24", "", 24, true},
		{"unknown mode 5", "", 5, true},
		{"unknown mode 6", "", 6, true},

		// Invalid mode
		{"mist mode 7", "mist", 7, false},
		{"meraki mode 0", "meraki", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidRadioMode(tt.vendor, tt.mode)
			if got != tt.want {
				t.Errorf("IsValidRadioMode(%q, %d) = %v, want %v", tt.vendor, tt.mode, got, tt.want)
			}
		})
	}
}

func TestGetBandForRadioMode(t *testing.T) {
	tests := []struct {
		mode int
		want string
	}{
		{24, "band_24"},
		{5, "band_5"},
		{6, "band_6"},
		{0, ""},
		{99, ""},
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.mode)), func(t *testing.T) {
			got := GetBandForRadioMode(tt.mode)
			if got != tt.want {
				t.Errorf("GetBandForRadioMode(%d) = %q, want %q", tt.mode, got, tt.want)
			}
		})
	}
}

func TestBand6ChannelGeneration(t *testing.T) {
	// Verify 6GHz channels are correctly generated
	channels := Band6Channels

	// Should start at 1
	if channels[0] != 1 {
		t.Errorf("Band6Channels should start at 1, got %d", channels[0])
	}

	// Should end at 233
	if channels[len(channels)-1] != 233 {
		t.Errorf("Band6Channels should end at 233, got %d", channels[len(channels)-1])
	}

	// Check spacing is 4
	for i := 1; i < len(channels); i++ {
		diff := channels[i] - channels[i-1]
		if diff != 4 {
			t.Errorf("Band6Channels spacing should be 4, got %d at index %d", diff, i)
		}
	}
}

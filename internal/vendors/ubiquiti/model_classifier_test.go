package ubiquiti

import "testing"

func TestClassifyDeviceType(t *testing.T) {
	tests := []struct {
		model    string
		expected string
	}{
		// APs - dash-separated codes
		{"U6-Pro", "ap"},
		{"U6-LR", "ap"},
		{"U6-Lite", "ap"},
		{"U6-Enterprise", "ap"},
		{"U7-Pro", "ap"},
		{"U7-Pro-Max", "ap"},
		{"UAP-AC-Pro", "ap"},
		{"UAP-nanoHD", "ap"},
		{"UBB-XG", "ap"},

		// APs - space-separated names (Site Manager API format)
		{"U6 LR", "ap"},
		{"U6 Pro", "ap"},
		{"U7 Pro", "ap"},
		{"Nano HD", "ap"},
		{"AC Pro", "ap"},
		{"AC LR", "ap"},
		{"AC Lite", "ap"},

		// Switches - dash-separated codes
		{"USW-Pro-24-PoE", "switch"},
		{"USW-Lite-16-PoE", "switch"},
		{"USW-Enterprise-48-PoE", "switch"},
		{"USL-Pro-24", "switch"},
		{"US-48-500W", "switch"},
		{"US-24-250W", "switch"},

		// Switches - space-separated names
		{"USW 48 PoE", "switch"},
		{"USW Lite 8 PoE", "switch"},
		{"USW Pro 24 PoE", "switch"},

		// Gateways - dash-separated codes
		{"UDM-Pro", "gateway"},
		{"UDM-SE", "gateway"},
		{"UDM-Pro-Max", "gateway"},
		{"UXG-Pro", "gateway"},
		{"UXG-Max", "gateway"},
		{"USG-Pro-4", "gateway"},
		{"UCG-Ultra", "gateway"},
		{"UDR-6", "gateway"},

		// Gateways - space-separated names
		{"UDM Pro", "gateway"},
		{"USG Pro 4", "gateway"},

		// Other
		{"UNVR-Pro", "other"},
		{"UP-Flex", "other"},
		{"UCK-G2-Plus", "other"},
		{"UCK G2", "other"},
		{"G4 Doorbell Pro PoE", "other"},
		{"Chime PoE", "other"},
		{"UNVR", "other"},
		{"", "other"},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := ClassifyDeviceType(tt.model)
			if got != tt.expected {
				t.Errorf("ClassifyDeviceType(%q) = %q, want %q", tt.model, got, tt.expected)
			}
		})
	}
}

func TestClassifyDeviceType_CaseInsensitive(t *testing.T) {
	tests := []struct {
		model    string
		expected string
	}{
		{"u6-pro", "ap"},
		{"u6 lr", "ap"},
		{"nano hd", "ap"},
		{"usw-pro-24-poe", "switch"},
		{"usw 48 poe", "switch"},
		{"udm-pro", "gateway"},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := ClassifyDeviceType(tt.model)
			if got != tt.expected {
				t.Errorf("ClassifyDeviceType(%q) = %q, want %q", tt.model, got, tt.expected)
			}
		})
	}
}

func TestClassifyByShortname(t *testing.T) {
	tests := []struct {
		shortname string
		expected  string
	}{
		// AP shortnames
		{"U7NHD", "ap"},     // Nano HD
		{"U7PG2", "ap"},     // AC Pro
		{"U7LR", "ap"},      // AC LR
		{"UALR6v2", "ap"},   // U6 LR
		{"U6IW", "ap"},      // U6 In-Wall

		// Switch shortnames
		{"USL48PB", "switch"}, // USW 48 PoE
		{"USL8LPB", "switch"}, // USW Lite 8 PoE
		{"USW24P", "switch"},  // USW 24 PoE

		// Gateway shortnames
		{"UDMPRO", "gateway"}, // UDM Pro
		{"USG3P", "gateway"},  // USG Pro 3
		{"UXGPRO", "gateway"}, // UXG Pro

		// Other
		{"UCKG2", "other"},    // Cloud Key G2
		{"UNVR", "other"},     // NVR
		{"", "other"},
	}

	for _, tt := range tests {
		t.Run(tt.shortname, func(t *testing.T) {
			got := classifyByShortname(tt.shortname)
			if got != tt.expected {
				t.Errorf("classifyByShortname(%q) = %q, want %q", tt.shortname, got, tt.expected)
			}
		})
	}
}

func TestClassifyDevice(t *testing.T) {
	tests := []struct {
		name     string
		device   Device
		expected string
	}{
		{
			"model matches directly",
			Device{Model: "U6 LR", Shortname: "UALR6v2"},
			"ap",
		},
		{
			"falls back to shortname",
			Device{Model: "Nano HD", Shortname: "U7NHD"},
			"ap", // "Nano HD" now matches "NANO" prefix after space→dash normalization
		},
		{
			"shortname resolves AC Pro",
			Device{Model: "AC Pro", Shortname: "U7PG2"},
			"ap",
		},
		{
			"switch via model",
			Device{Model: "USW 48 PoE", Shortname: "USL48PB"},
			"switch",
		},
		{
			"other stays other",
			Device{Model: "UCK G2", Shortname: "UCKG2"},
			"other",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyDevice(tt.device)
			if got != tt.expected {
				t.Errorf("classifyDevice() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestIsNetworkDevice(t *testing.T) {
	tests := []struct {
		name     string
		device   Device
		expected bool
	}{
		{"network device", Device{ProductLine: "network"}, true},
		{"network uppercase", Device{ProductLine: "Network"}, true},
		{"protect device", Device{ProductLine: "protect"}, false},
		{"empty product line", Device{ProductLine: ""}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNetworkDevice(tt.device)
			if got != tt.expected {
				t.Errorf("IsNetworkDevice() = %v, want %v", got, tt.expected)
			}
		})
	}
}

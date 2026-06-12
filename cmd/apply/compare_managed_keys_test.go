package apply

import "testing"

// TestCompareDeviceConfigsWithManagedKeys_SubsetSemantics guards the partial-intent
// model: intent declares only the fields it manages, while the API returns the full
// object — read-only echoes (a radio block's serial) and auto/empty sub-blocks. A
// running config that realizes every declared field is not drift, so a Meraki radio
// apply does not read as perpetually divergent after a faithful push.
func TestCompareDeviceConfigsWithManagedKeys_SubsetSemantics(t *testing.T) {
	managed := []string{"name", "radio_settings"}

	running := map[string]any{
		"name": "ap1-1.lab1",
		"radio_settings": map[string]any{
			"fiveGhzSettings":    map[string]any{"channel": 36, "targetPower": 10},
			"serial":             "Q2ZD-BQ32-KPNP", // read-only echo, never in intent
			"twoFourGhzSettings": map[string]any{}, // auto band the API always returns
		},
	}

	tests := []struct {
		name   string
		intent map[string]any
		want   bool // true = needs update (diverges)
	}{
		{
			name: "realized intent is not drift despite extra running-config fields",
			intent: map[string]any{
				"name":           "ap1-1.lab1",
				"radio_settings": map[string]any{"fiveGhzSettings": map[string]any{"channel": 36, "targetPower": 10}},
			},
			want: false,
		},
		{
			name: "a changed managed leaf is drift",
			intent: map[string]any{
				"name":           "ap1-1.lab1",
				"radio_settings": map[string]any{"fiveGhzSettings": map[string]any{"channel": 40, "targetPower": 10}},
			},
			want: true,
		},
		{
			name: "a declared field the API has not realized is drift",
			intent: map[string]any{
				"name":           "ap1-1.lab1",
				"radio_settings": map[string]any{"sixGhzSettings": map[string]any{"channel": 37}},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := compareDeviceConfigsWithManagedKeys(running, tt.intent, managed); got != tt.want {
				t.Errorf("compareDeviceConfigsWithManagedKeys = %v, want %v", got, tt.want)
			}
		})
	}
}

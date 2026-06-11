package vendors

import "testing"

func TestSkipDeviceConfig(t *testing.T) {
	tests := []struct {
		name     string
		opts     RefreshOptions
		siteID   string
		mac      string
		wantSkip bool
	}{
		{
			name:     "no filters fetches everything",
			opts:     RefreshOptions{},
			siteID:   "site-1",
			mac:      "aabbccddeeff",
			wantSkip: false,
		},
		{
			name:     "site match fetches",
			opts:     RefreshOptions{SiteID: "site-1"},
			siteID:   "site-1",
			mac:      "aabbccddeeff",
			wantSkip: false,
		},
		{
			name:     "site mismatch skips",
			opts:     RefreshOptions{SiteID: "site-1"},
			siteID:   "site-2",
			mac:      "aabbccddeeff",
			wantSkip: true,
		},
		{
			name:     "managed contains mac fetches",
			opts:     RefreshOptions{ManagedMACs: map[string]bool{"aabbccddeeff": true}},
			siteID:   "site-1",
			mac:      "aabbccddeeff",
			wantSkip: false,
		},
		{
			name:     "managed missing mac skips",
			opts:     RefreshOptions{ManagedMACs: map[string]bool{"aabbccddeeff": true}},
			siteID:   "site-1",
			mac:      "112233445566",
			wantSkip: true,
		},
		{
			name:     "empty managed set skips everything",
			opts:     RefreshOptions{ManagedMACs: map[string]bool{}},
			siteID:   "site-1",
			mac:      "aabbccddeeff",
			wantSkip: true,
		},
		{
			name:     "site and managed both must match: armed but wrong site skips",
			opts:     RefreshOptions{SiteID: "site-1", ManagedMACs: map[string]bool{"aabbccddeeff": true}},
			siteID:   "site-2",
			mac:      "aabbccddeeff",
			wantSkip: true,
		},
		{
			name:     "site and managed both match fetches",
			opts:     RefreshOptions{SiteID: "site-1", ManagedMACs: map[string]bool{"aabbccddeeff": true}},
			siteID:   "site-1",
			mac:      "aabbccddeeff",
			wantSkip: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := skipDeviceConfig(tt.opts, tt.siteID, tt.mac); got != tt.wantSkip {
				t.Errorf("skipDeviceConfig = %v, want %v", got, tt.wantSkip)
			}
		})
	}
}

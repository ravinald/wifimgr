package netbox

import (
	"testing"
)

func TestDeviceMetadata(t *testing.T) {
	tests := []struct {
		name     string
		metadata *DeviceMetadata
		wantMAC  string
		wantName string
		wantSite string
	}{
		{
			name: "basic metadata",
			metadata: &DeviceMetadata{
				MAC:      "aabbccddeeff",
				Name:     "AP-01",
				SiteID:   "1",
				SiteName: "US-LAB-01",
				Model:    "AP43",
				Serial:   "ABC123",
			},
			wantMAC:  "aabbccddeeff",
			wantName: "AP-01",
			wantSite: "US-LAB-01",
		},
		{
			name: "metadata without site",
			metadata: &DeviceMetadata{
				MAC:    "112233445566",
				Name:   "AP-02",
				Model:  "AP41",
				Serial: "XYZ789",
			},
			wantMAC:  "112233445566",
			wantName: "AP-02",
			wantSite: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.metadata.MAC != tt.wantMAC {
				t.Errorf("MAC = %v, want %v", tt.metadata.MAC, tt.wantMAC)
			}
			if tt.metadata.Name != tt.wantName {
				t.Errorf("Name = %v, want %v", tt.metadata.Name, tt.wantName)
			}
			if tt.metadata.SiteName != tt.wantSite {
				t.Errorf("SiteName = %v, want %v", tt.metadata.SiteName, tt.wantSite)
			}
		})
	}
}

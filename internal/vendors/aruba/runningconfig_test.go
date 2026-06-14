package aruba

import "testing"

const sampleRunningConfig = `version 8.12.0.0
virtual-controller-country US
organization Acme-Cabins
hostname swarm-master
!
wlan ssid-profile AA-Cabin123
 enable
 index 0
 type employee
 essid AA-Cabin123
 opmode wpa2-psk-aes
 wpa-passphrase a1b2c3hashed
 vlan 102
 rf-band 5.0
 dtim-period 1
!
wlan ssid-profile Guest
 disable
 essid Guest-WiFi
 type guest
 opmode opensystem
 hide-ssid
 rf-band all
!
rf dot11a-radio-profile default
 beacon-interval 100
 max-tx-power 18
!
`

func TestParseRunningConfig_Blocks(t *testing.T) {
	blocks := parseRunningConfig(sampleRunningConfig)

	if got := globalValue(blocks, "organization"); got != "Acme-Cabins" {
		t.Errorf("organization = %q, want Acme-Cabins", got)
	}
	if got := globalValue(blocks, "virtual-controller-country"); got != "US" {
		t.Errorf("country = %q, want US", got)
	}
	if got := globalValue(blocks, "hostname"); got != "swarm-master" {
		t.Errorf("hostname = %q, want swarm-master", got)
	}

	if n := len(ssidProfiles(blocks)); n != 2 {
		t.Fatalf("ssidProfiles = %d, want 2", n)
	}
}

func TestExtractWLANs(t *testing.T) {
	wlans := extractWLANs(parseRunningConfig(sampleRunningConfig), "site-1")
	if len(wlans) != 2 {
		t.Fatalf("got %d WLANs, want 2", len(wlans))
	}

	cabin := wlans[0]
	if cabin.ID != "AA-Cabin123" || cabin.SSID != "AA-Cabin123" {
		t.Errorf("cabin id/ssid = %q/%q", cabin.ID, cabin.SSID)
	}
	if !cabin.Enabled {
		t.Error("cabin should be enabled")
	}
	if cabin.Hidden {
		t.Error("cabin should not be hidden")
	}
	if cabin.Band != "5" {
		t.Errorf("cabin band = %q, want 5", cabin.Band)
	}
	if cabin.VLANID != 102 {
		t.Errorf("cabin vlan = %d, want 102", cabin.VLANID)
	}
	if cabin.AuthType != "psk" || cabin.EncryptionMode != "wpa2" {
		t.Errorf("cabin auth/enc = %q/%q, want psk/wpa2", cabin.AuthType, cabin.EncryptionMode)
	}
	if cabin.SiteID != "site-1" {
		t.Errorf("cabin siteID = %q", cabin.SiteID)
	}

	guest := wlans[1]
	if guest.ID != "Guest" || guest.SSID != "Guest-WiFi" {
		t.Errorf("guest id/ssid = %q/%q, want Guest/Guest-WiFi", guest.ID, guest.SSID)
	}
	if guest.Enabled {
		t.Error("guest should be disabled")
	}
	if !guest.Hidden {
		t.Error("guest should be hidden")
	}
	if guest.Band != "all" {
		t.Errorf("guest band = %q, want all", guest.Band)
	}
	if guest.AuthType != "open" {
		t.Errorf("guest auth = %q, want open", guest.AuthType)
	}
}

func TestAuthFromOpmode(t *testing.T) {
	tests := []struct {
		opmode   string
		wantAuth string
		wantEnc  string
	}{
		{"wpa2-psk-aes", "psk", "wpa2"},
		{"wpa3-sae-aes", "sae", "wpa3"},
		{"opensystem", "open", ""},
		{"wpa3-open", "open", "wpa3"},
		{"wpa2-aes", "wpa2-enterprise", "wpa2"},
		{"static-wep", "wep", "wep"},
		{"", "open", ""},
	}
	for _, tt := range tests {
		auth, enc := authFromOpmode(tt.opmode)
		if auth != tt.wantAuth || enc != tt.wantEnc {
			t.Errorf("authFromOpmode(%q) = %q/%q, want %q/%q", tt.opmode, auth, enc, tt.wantAuth, tt.wantEnc)
		}
	}
}

func TestParseRunningConfig_Empty(t *testing.T) {
	if blocks := parseRunningConfig(""); blocks != nil {
		t.Errorf("empty input should yield nil, got %d blocks", len(blocks))
	}
}

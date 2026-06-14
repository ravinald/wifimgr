package aruba

import (
	"testing"

	"github.com/ravinald/wifimgr/internal/vendors"
)

func TestSSIDProfilePayload(t *testing.T) {
	w := &vendors.WLAN{
		ID:             "AA-Cabin123",
		SSID:           "AA-Cabin123",
		Enabled:        true,
		Band:           "5",
		VLANID:         102,
		AuthType:       "psk",
		EncryptionMode: "wpa2",
		PSK:            "secret123",
	}

	payload := ssidProfilePayload(w)
	profile, ok := payload["ssid-profile"].(map[string]any)
	if !ok {
		t.Fatal("payload missing ssid-profile object")
	}

	if profile["action"] != "create" {
		t.Errorf("action = %v, want create", profile["action"])
	}
	if profile["ssid-profile"] != "AA-Cabin123" {
		t.Errorf("ssid-profile = %v", profile["ssid-profile"])
	}
	if profile["opmode"] != "wpa2-psk-aes" {
		t.Errorf("opmode = %v, want wpa2-psk-aes", profile["opmode"])
	}
	if profile["wpa-passphrase"] != "secret123" {
		t.Errorf("passphrase = %v", profile["wpa-passphrase"])
	}
	if profile["rf-band"] != "5.0" {
		t.Errorf("rf-band = %v, want 5.0", profile["rf-band"])
	}
	if profile["enable"] != "yes" {
		t.Errorf("enable = %v, want yes", profile["enable"])
	}

	essid, ok := profile["essid"].(map[string]any)
	if !ok || essid["value"] != "AA-Cabin123" {
		t.Errorf("essid = %v", profile["essid"])
	}
	vlan, ok := profile["vlan"].(map[string]any)
	if !ok || vlan["value"] != "102" {
		t.Errorf("vlan = %v", profile["vlan"])
	}
}

func TestSSIDProfilePayload_Disabled(t *testing.T) {
	w := &vendors.WLAN{ID: "Guest", SSID: "Guest", Enabled: false, Hidden: true, AuthType: "open"}
	profile := ssidProfilePayload(w)["ssid-profile"].(map[string]any)

	if _, hasEnable := profile["enable"]; hasEnable {
		t.Error("disabled WLAN should not set enable")
	}
	if profile["disable"] != "yes" {
		t.Errorf("disable = %v, want yes", profile["disable"])
	}
	if profile["hide-ssid"] != "enable" {
		t.Errorf("hide-ssid = %v, want enable", profile["hide-ssid"])
	}
	if profile["opmode"] != "opensystem" {
		t.Errorf("opmode = %v, want opensystem", profile["opmode"])
	}
}

func TestSSIDDeletePayload(t *testing.T) {
	profile := ssidDeletePayload("Guest")["ssid-profile"].(map[string]any)
	if profile["action"] != "delete" || profile["ssid-profile"] != "Guest" {
		t.Errorf("delete payload = %v", profile)
	}
}

func TestOpmodeRoundTrip(t *testing.T) {
	// auth/enc derived from an opmode should map back to a compatible opmode.
	for _, opmode := range []string{"wpa2-psk-aes", "wpa3-sae-aes", "opensystem"} {
		auth, enc := authFromOpmode(opmode)
		if got := opmodeFromAuth(auth, enc); got != opmode {
			t.Errorf("round trip %q -> (%s,%s) -> %q", opmode, auth, enc, got)
		}
	}
}

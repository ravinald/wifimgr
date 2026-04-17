package cmd

import (
	"encoding/json"
	"testing"

	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/vendors"
)

func TestSlug(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"Corp Net", "corp-net"},
		{"corp-net", "corp-net"},
		{"MX - Av. Ejercito Nacional Mexicano 904", "mx-av-ejercito-nacional-mexicano-904"},
		{"  leading and trailing  ", "leading-and-trailing"},
		{"ALL_CAPS_UNDERSCORES", "all-caps-underscores"},
		{"Scale Guest", "scale-guest"},
		{"UPPER", "upper"},
		{"", ""},
		{"!!!", ""},
		{"a!!!b", "a-b"},
		{"a--b--c", "a-b-c"},
	}
	for _, tt := range tests {
		if got := slug(tt.in); got != tt.want {
			t.Errorf("slug(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestNormalizeAuthType(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", "open"},
		{"open", "open"},
		{"Open", "open"},
		{"open-with-radius", "open"},
		{"open-enhanced", "open"},
		{"psk", "psk"},
		{"wpa2-psk", "psk"},
		{"wpa3-psk", "psk"},
		{"sae", "psk"},
		{"ipsk-without-radius", "ipsk"},
		{"ipsk-with-radius", "ipsk"},
		{"ipsk-with-nac", "ipsk"},
		{"IPSK", "ipsk"},
		{"8021x-radius", "eap"},
		{"8021x-meraki", "eap"},
		{"8021x-google", "eap"},
		{"wpa2-enterprise", "eap"},
		{"WPA3-Enterprise", "eap"},
		{"exotic", "exotic"}, // unknown passes through verbatim
	}
	for _, tt := range tests {
		if got := normalizeAuthType(tt.in); got != tt.want {
			t.Errorf("normalizeAuthType(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestNormalizeBand(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"dual", "dual"},
		{"5", "5"},
		{"Dual band operation", "dual"},
		{"Dual band operation with Band Steering", "dual"},
		{"5 GHz band only", "5"},
		{"2.4 GHz band only", "2.4"},
		{"6 GHz band only", "6"},
		{"All Bands", "all"},
		{"exotic-mode", "exotic-mode"}, // unknown passes through
	}
	for _, tt := range tests {
		if got := normalizeBand(tt.in); got != tt.want {
			t.Errorf("normalizeBand(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestDerivePairwiseFromConfig(t *testing.T) {
	tests := []struct {
		name       string
		cfg        map[string]any
		legacyMode string
		want       []string
	}{
		{
			name: "meraki wpa2 only",
			cfg:  map[string]any{"wpaEncryptionMode": "WPA2 only"},
			want: []string{"wpa2-ccmp"},
		},
		{
			name: "meraki wpa3 transition",
			cfg:  map[string]any{"wpaEncryptionMode": "WPA3 Transition Mode"},
			want: []string{"wpa2-ccmp", "wpa3"},
		},
		{
			name: "meraki wpa3 personal",
			cfg:  map[string]any{"wpaEncryptionMode": "WPA3 Personal"},
			want: []string{"wpa3"},
		},
		{
			name:       "falls back when wpaEncryptionMode missing",
			cfg:        map[string]any{"other": "stuff"},
			legacyMode: "wpa2",
			want:       []string{"wpa2-ccmp"},
		},
		{
			name:       "falls back when wpaEncryptionMode unknown",
			cfg:        map[string]any{"wpaEncryptionMode": "Future Mode"},
			legacyMode: "wpa3",
			want:       []string{"wpa3"},
		},
		{
			name:       "nil config falls through to legacy",
			cfg:        nil,
			legacyMode: "wpa2",
			want:       []string{"wpa2-ccmp"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := derivePairwiseFromConfig(tt.cfg, tt.legacyMode)
			if !equalStrings(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEncryptionModeToPairwise(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"none", nil},
		{"wpa2", []string{"wpa2-ccmp"}},
		{"wpa3", []string{"wpa3"}},
		{"wpa2/wpa3", []string{"wpa2-ccmp", "wpa3"}},
		{"exotic-mode", []string{"exotic-mode"}},
	}
	for _, tt := range tests {
		got := encryptionModeToPairwise(tt.in)
		if !equalStrings(got, tt.want) {
			t.Errorf("encryptionModeToPairwise(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestExtractPortal(t *testing.T) {
	tests := []struct {
		name string
		in   map[string]any
		want *config.PortalConfig
	}{
		{"nil map", nil, nil},
		{"splashPage none", map[string]any{"splashPage": "None"}, nil},
		{"splashPage click-through", map[string]any{"splashPage": "Click-through"}, &config.PortalConfig{Enabled: true, Auth: "Click-through"}},
		{"mist portal disabled zero", map[string]any{"portal": map[string]any{}}, nil},
		{"mist portal enabled", map[string]any{"portal": map[string]any{"enabled": true, "auth": "sponsor"}}, &config.PortalConfig{Enabled: true, Auth: "sponsor"}},
		{"mist auth without enabled still captured", map[string]any{"portal": map[string]any{"auth": "passphrase"}}, &config.PortalConfig{Enabled: false, Auth: "passphrase"}},
		{"neither key", map[string]any{"other": "stuff"}, nil},
	}
	for _, tt := range tests {
		got := extractPortal(tt.in)
		if !portalEqual(got, tt.want) {
			t.Errorf("%s: extractPortal = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

func TestConvertVendorWLANToProfile_Meraki(t *testing.T) {
	w := &vendors.WLAN{
		ID:             "L_123:0",
		SSID:           "Scale Guest",
		Enabled:        true,
		Hidden:         false,
		Band:           "dual",
		VLANID:         10,
		AuthType:       "psk",
		EncryptionMode: "wpa2",
		PSK:            "s3cr3t",
		Config: map[string]any{
			"perClientBandwidthLimitUp":   float64(1000),
			"perClientBandwidthLimitDown": float64(5000),
			"splashPage":                  "Click-through",
		},
	}
	got := convertVendorWLANToProfile(w, true)

	if got.SSID != "Scale Guest" || !got.Enabled || got.VLANID != 10 || got.Band != "dual" {
		t.Errorf("basic fields wrong: %+v", got)
	}
	if got.Auth.Type != "psk" || got.Auth.PSK != "s3cr3t" {
		t.Errorf("auth wrong: %+v", got.Auth)
	}
	if !equalStrings(got.Auth.Pairwise, []string{"wpa2-ccmp"}) {
		t.Errorf("pairwise wrong: %v", got.Auth.Pairwise)
	}
	if got.ClientLimitUp != 1000 || got.ClientLimitDown != 5000 {
		t.Errorf("bandwidth wrong: up=%d down=%d", got.ClientLimitUp, got.ClientLimitDown)
	}
	if got.Portal == nil || !got.Portal.Enabled || got.Portal.Auth != "Click-through" {
		t.Errorf("portal wrong: %+v", got.Portal)
	}

	// With includeSecrets=false, PSK must be stripped.
	gotNoSec := convertVendorWLANToProfile(w, false)
	if gotNoSec.Auth.PSK != "" {
		t.Errorf("PSK leaked when includeSecrets=false: %q", gotNoSec.Auth.PSK)
	}
}

// TestConvertVendorWLANToProfile_MerakiRealWorld mirrors the exact field shape
// a Meraki site produces today (verbose bandSelection string, legacy "wpa"
// encryption mode, iPSK auth type) and checks the output is canonical.
func TestConvertVendorWLANToProfile_MerakiRealWorld(t *testing.T) {
	w := &vendors.WLAN{
		SSID:           "Scale - Office Devices",
		Enabled:        true,
		Band:           "Dual band operation with Band Steering",
		AuthType:       "ipsk-without-radius",
		EncryptionMode: "wpa",
		Config: map[string]any{
			"wpaEncryptionMode": "WPA2 only",
			"splashPage":        "None",
			"bandSelection":     "Dual band operation with Band Steering",
		},
	}
	got := convertVendorWLANToProfile(w, false)

	if got.Band != "dual" {
		t.Errorf("Band = %q, want %q", got.Band, "dual")
	}
	if got.Auth.Type != "ipsk" {
		t.Errorf("Auth.Type = %q, want %q", got.Auth.Type, "ipsk")
	}
	if !equalStrings(got.Auth.Pairwise, []string{"wpa2-ccmp"}) {
		t.Errorf("Auth.Pairwise = %v, want [wpa2-ccmp]", got.Auth.Pairwise)
	}
	if got.Portal != nil {
		t.Errorf("splashPage=None should not produce a Portal, got %+v", got.Portal)
	}
}

func TestConvertVendorWLANToProfile_MistEnterprise(t *testing.T) {
	w := &vendors.WLAN{
		ID:             "uuid-1",
		SSID:           "Corp-Net",
		Enabled:        true,
		Hidden:         true,
		Band:           "5",
		VLANID:         20,
		AuthType:       "wpa2-enterprise",
		EncryptionMode: "wpa2/wpa3",
		RadiusServers: []vendors.RadiusServer{
			{Host: "radius1.example.com", Port: 1812, Secret: "shared-secret"},
		},
		Config: map[string]any{
			"portal": map[string]any{"enabled": true, "auth": "sponsor"},
		},
	}
	got := convertVendorWLANToProfile(w, true)

	if got.Auth.Type != "eap" {
		t.Errorf("expected Auth.Type=eap, got %q", got.Auth.Type)
	}
	if !equalStrings(got.Auth.Pairwise, []string{"wpa2-ccmp", "wpa3"}) {
		t.Errorf("pairwise wrong: %v", got.Auth.Pairwise)
	}
	if len(got.Auth.RADIUSServers) != 1 ||
		got.Auth.RADIUSServers[0].Host != "radius1.example.com" ||
		got.Auth.RADIUSServers[0].Port != 1812 ||
		got.Auth.RADIUSServers[0].Secret != "shared-secret" {
		t.Errorf("radius servers wrong: %+v", got.Auth.RADIUSServers)
	}
	if got.Portal == nil || got.Portal.Auth != "sponsor" {
		t.Errorf("portal wrong: %+v", got.Portal)
	}

	// includeSecrets=false strips RADIUS secret but keeps host/port.
	gotNoSec := convertVendorWLANToProfile(w, false)
	if gotNoSec.Auth.RADIUSServers[0].Secret != "" {
		t.Errorf("RADIUS secret leaked: %q", gotNoSec.Auth.RADIUSServers[0].Secret)
	}
	if gotNoSec.Auth.RADIUSServers[0].Host != "radius1.example.com" {
		t.Errorf("host should survive secret strip: %q", gotNoSec.Auth.RADIUSServers[0].Host)
	}
}

func TestSynthesizeWLANLabels(t *testing.T) {
	wlans := []*vendors.WLAN{
		{SSID: "Scale Guest", Enabled: true, AuthType: "open"},
		{SSID: "Scale Robotics", Enabled: true, AuthType: "psk"},
	}
	labels, profiles := synthesizeWLANLabels(wlans, "mx-mex-904", false)

	if len(labels) != 2 || len(profiles) != 2 {
		t.Fatalf("expected 2 labels/profiles, got %d/%d", len(labels), len(profiles))
	}
	wantLabels := []string{"mx-mex-904--scale-guest", "mx-mex-904--scale-robotics"}
	if !equalStrings(labels, wantLabels) {
		t.Errorf("labels = %v, want %v", labels, wantLabels)
	}
	for _, label := range labels {
		if profiles[label] == nil {
			t.Errorf("no profile for label %q", label)
		}
	}
}

func TestSynthesizeWLANLabels_CollisionSuffix(t *testing.T) {
	// Two SSIDs that slug to the same base — collision handling kicks in.
	wlans := []*vendors.WLAN{
		{SSID: "Corp Net", Enabled: true, AuthType: "open"},
		{SSID: "corp-net", Enabled: true, AuthType: "psk"},
		{SSID: "CORP__NET", Enabled: true, AuthType: "psk"},
	}
	labels, profiles := synthesizeWLANLabels(wlans, "site", false)

	want := []string{"site--corp-net", "site--corp-net-2", "site--corp-net-3"}
	if !equalStrings(labels, want) {
		t.Errorf("labels = %v, want %v", labels, want)
	}
	if len(profiles) != 3 {
		t.Errorf("expected 3 unique profile keys, got %d", len(profiles))
	}
}

func TestSynthesizeWLANLabels_Empty(t *testing.T) {
	labels, profiles := synthesizeWLANLabels(nil, "site", false)
	if labels != nil || profiles != nil {
		t.Errorf("empty input should return nil/nil, got %v/%v", labels, profiles)
	}
}

// Round-trip check: the generated SiteConfigFile + WLANProfileFile should
// unmarshal into the loader's target types without errors. This is the
// contract that was broken before the fix and the one the test guards.
func TestExportRoundTripsThroughLoaderTypes(t *testing.T) {
	wlans := []*vendors.WLAN{
		{SSID: "Scale Guest", Enabled: true, AuthType: "open"},
	}
	labels, profiles := synthesizeWLANLabels(wlans, "mx-mex-904", false)

	// Build a minimal site export using the new shape.
	siteExport := &SiteExportConfig{
		Version: 1,
		Config: SiteExportConfigData{
			Sites: map[string]*SiteConfigData{
				"MX - Av. Ejercito Nacional Mexicano 904": {
					API:        "meraki",
					SiteConfig: map[string]any{"name": "MX - Av. Ejercito Nacional Mexicano 904"},
					WLAN:       labels,
				},
			},
		},
	}
	siteBytes, err := json.Marshal(siteExport)
	if err != nil {
		t.Fatalf("marshal site: %v", err)
	}

	// Unmarshal into the loader's SiteConfigFile — this is the operation that
	// used to fail with "cannot unmarshal object into ... field of type string".
	var loaded config.SiteConfigFile
	if err := json.Unmarshal(siteBytes, &loaded); err != nil {
		t.Fatalf("site file rejected by loader types: %v", err)
	}
	gotSite, ok := loaded.Config.Sites["MX - Av. Ejercito Nacional Mexicano 904"]
	if !ok {
		t.Fatalf("site not present after unmarshal")
	}
	if !equalStrings(gotSite.WLAN, []string{"mx-mex-904--scale-guest"}) {
		t.Errorf("WLAN labels lost in round-trip: %v", gotSite.WLAN)
	}

	// Now the template file.
	templateExport := &config.WLANProfileFile{Version: 1, WLANProfiles: profiles}
	tplBytes, err := json.Marshal(templateExport)
	if err != nil {
		t.Fatalf("marshal template: %v", err)
	}
	var loadedTpl config.WLANProfileFile
	if err := json.Unmarshal(tplBytes, &loadedTpl); err != nil {
		t.Fatalf("template file rejected by loader types: %v", err)
	}
	if _, ok := loadedTpl.WLANProfiles["mx-mex-904--scale-guest"]; !ok {
		t.Errorf("profile key missing after round-trip; got keys: %v", keysOf(loadedTpl.WLANProfiles))
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func portalEqual(a, b *config.PortalConfig) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Enabled == b.Enabled && a.Auth == b.Auth
}

func keysOf(m map[string]*config.WLANProfile) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

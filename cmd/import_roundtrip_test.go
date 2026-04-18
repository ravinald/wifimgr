package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// TestImportRoundTrip_FieldFidelity exercises the design-doc contract at
// docs-internal/design.md:320–335 — "import → register → apply-diff == no-op"
// for the fields the doc says we care about: labels, auth, band, vlan,
// portal, radius.
//
// The flow:
//  1. Seed vendor-normalized WLANs with each auth shape (open, psk, ipsk, eap).
//  2. Run the export pipeline (synthesize labels + render profile to map).
//  3. Assemble an importEnvelope, serialize to disk.
//  4. Load via config.LoadImportFile (the real loader).
//  5. Merge Templates into a fresh TemplateStore (simulates load-time).
//  6. Assert every "field we care about" round-trips unchanged.
//
// Failures here indicate import fidelity loss, which is load-bearing:
// silent drift on an operator's first `apply diff` is exactly what this
// test is supposed to catch.
func TestImportRoundTrip_FieldFidelity(t *testing.T) {
	tests := []struct {
		name   string
		vendor string
		site   string
		wlans  []*vendors.WLAN
	}{
		{
			name:   "mist enterprise mix",
			vendor: "mist",
			site:   "US-LAB-01",
			wlans: []*vendors.WLAN{
				{
					SSID: "corp-net", Enabled: true, Hidden: false, Band: "5",
					VLANID: 20, AuthType: "wpa2-enterprise", EncryptionMode: "wpa2/wpa3",
					RadiusServers: []vendors.RadiusServer{
						{Host: "radius1.example.com", Port: 1812, Secret: "shared"},
					},
					Config: map[string]any{
						"portal": map[string]any{"enabled": true, "auth": "sponsor"},
					},
				},
				{
					SSID: "guest", Enabled: true, Band: "dual",
					VLANID: 99, AuthType: "open",
					Config: map[string]any{
						"portal": map[string]any{"enabled": true, "auth": "Click-through"},
					},
				},
			},
		},
		{
			name:   "meraki ipsk + psk",
			vendor: "meraki",
			site:   "MX-MEX-904",
			wlans: []*vendors.WLAN{
				{
					SSID: "Scale Robotics", Enabled: true, Band: "dual",
					VLANID: 10, AuthType: "psk", EncryptionMode: "wpa2", PSK: "s3cr3t-pass",
				},
				{
					SSID: "Scale - Office Devices", Enabled: true,
					Band:     "Dual band operation with Band Steering",
					AuthType: "ipsk-without-radius", EncryptionMode: "wpa",
					Config: map[string]any{
						"wpaEncryptionMode": "WPA2 only",
						"splashPage":        "None",
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runRoundTripCase(t, tc.vendor, tc.site, tc.wlans)
		})
	}
}

// runRoundTripCase drives the full export → disk → load → merge path for
// one seed and asserts the critical fields survive.
func runRoundTripCase(t *testing.T, vendor, siteName string, seedWLANs []*vendors.WLAN) {
	t.Helper()

	siteSlug := slug(siteName)

	// Step 1+2: export pipeline (no cache accessor needed for this slice).
	labels, profiles := synthesizeWLANLabels(seedWLANs, siteSlug, true /* includeSecrets */)
	if len(labels) != len(seedWLANs) {
		t.Fatalf("label count = %d, want %d", len(labels), len(seedWLANs))
	}

	templatesMap := make(map[string]map[string]any, len(profiles))
	for label, p := range profiles {
		m, err := profileToMap(p)
		if err != nil {
			t.Fatalf("profileToMap(%q): %v", label, err)
		}
		templatesMap[label] = m
	}

	env := &importEnvelope{
		Version: 1,
		Source: &importSourceExport{
			API:  vendor + "-test",
			Site: siteName,
			Kind: "site",
		},
		Config: &siteConfigEnvelope{
			Sites: map[string]*siteObjExport{
				siteName: {
					API:        vendor + "-test",
					SiteConfig: map[string]any{"name": siteName},
					WLAN:       labels,
				},
			},
		},
		Templates: &templatesEnvelope{WLAN: templatesMap},
	}

	// Step 3: serialize to disk.
	dir := t.TempDir()
	path := filepath.Join(dir, siteSlug+".json")
	data, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	// Step 4: load via the real loader (this catches envelope-shape
	// regressions that any upstream rename would introduce).
	loaded, err := config.LoadImportFile(path)
	if err != nil {
		t.Fatalf("LoadImportFile: %v", err)
	}
	if loaded.Config == nil || loaded.Templates == nil {
		t.Fatalf("Config/Templates missing after load: config=%v templates=%v",
			loaded.Config != nil, loaded.Templates != nil)
	}

	// Step 5: merge templates into a fresh store (mirrors production).
	store := config.NewTemplateStore()
	config.MergeImportTemplates(store, loaded)

	// Step 6: assert every WLAN label on the site has a matching template in
	// the store, and that the fields the design doc calls out are intact.
	site, ok := loaded.Config.Sites[siteName]
	if !ok {
		t.Fatalf("site %q missing after load", siteName)
	}
	if len(site.WLAN) != len(seedWLANs) {
		t.Fatalf("site.WLAN labels = %d, want %d", len(site.WLAN), len(seedWLANs))
	}

	for i, label := range site.WLAN {
		tmpl, ok := store.WLAN[label]
		if !ok {
			t.Errorf("label %q missing from template store", label)
			continue
		}
		seed := seedWLANs[i]

		assertStringField(t, label, tmpl, "ssid", seed.SSID)
		assertBoolField(t, label, tmpl, "enabled", seed.Enabled)

		// Band must be normalized to canonical form regardless of the
		// vendor-native dialect in the seed. That normalization is part of
		// the export pipeline and must survive the loader.
		wantBand := normalizeBand(seed.Band)
		if wantBand != "" {
			assertStringField(t, label, tmpl, "band", wantBand)
		}

		// VLAN ID on non-zero seeds must survive.
		if seed.VLANID != 0 {
			assertNumberField(t, label, tmpl, "vlan_id", seed.VLANID)
		}

		// Auth block: type always, PSK when includeSecrets, RADIUS hosts
		// when present.
		auth, ok := tmpl["auth"].(map[string]any)
		if !ok {
			t.Errorf("label %q: auth block missing/wrong shape: %T", label, tmpl["auth"])
			continue
		}
		assertStringField(t, label+".auth", auth, "type", normalizeAuthType(seed.AuthType))
		if seed.PSK != "" {
			assertStringField(t, label+".auth", auth, "psk", seed.PSK)
		}
		if len(seed.RadiusServers) > 0 {
			servers, ok := auth["radius_servers"].([]any)
			if !ok || len(servers) != len(seed.RadiusServers) {
				t.Errorf("label %q: radius_servers missing/wrong len (got %T, %d entries)",
					label, auth["radius_servers"], len(servers))
			} else {
				first, _ := servers[0].(map[string]any)
				if got, _ := first["host"].(string); got != seed.RadiusServers[0].Host {
					t.Errorf("label %q: radius host = %q, want %q",
						label, got, seed.RadiusServers[0].Host)
				}
			}
		}

		// Portal: when the seed carries a portal signal, the loaded template
		// must expose it. We don't reconstruct exact auth strings across the
		// splashPage/portal dialect gap — only assert presence.
		if seedHasPortal(seed) {
			if tmpl["portal"] == nil {
				t.Errorf("label %q: expected portal section, got nil", label)
			}
		}
	}
}

func seedHasPortal(w *vendors.WLAN) bool {
	if w.Config == nil {
		return false
	}
	if p, ok := w.Config["portal"].(map[string]any); ok && len(p) > 0 {
		return true
	}
	if sp, ok := w.Config["splashPage"].(string); ok && sp != "" && sp != "None" {
		return true
	}
	return false
}

func assertStringField(t *testing.T, ctx string, m map[string]any, key, want string) {
	t.Helper()
	got, ok := m[key].(string)
	if !ok {
		t.Errorf("%s: field %q missing or non-string (type %T)", ctx, key, m[key])
		return
	}
	if got != want {
		t.Errorf("%s: field %q = %q, want %q", ctx, key, got, want)
	}
}

func assertBoolField(t *testing.T, ctx string, m map[string]any, key string, want bool) {
	t.Helper()
	got, ok := m[key].(bool)
	if !ok {
		t.Errorf("%s: field %q missing or non-bool (type %T)", ctx, key, m[key])
		return
	}
	if got != want {
		t.Errorf("%s: field %q = %v, want %v", ctx, key, got, want)
	}
}

func assertNumberField(t *testing.T, ctx string, m map[string]any, key string, want int) {
	t.Helper()
	// JSON numbers decode to float64 through map[string]any.
	got, ok := m[key].(float64)
	if !ok {
		t.Errorf("%s: field %q missing or non-number (type %T)", ctx, key, m[key])
		return
	}
	if int(got) != want {
		t.Errorf("%s: field %q = %v, want %d", ctx, key, got, want)
	}
}

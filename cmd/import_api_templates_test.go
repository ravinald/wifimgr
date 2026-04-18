package cmd

import (
	"encoding/json"
	"testing"

	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/vendors"
)

func TestParseImportTemplatesArgs_Defaults(t *testing.T) {
	got, err := parseImportTemplatesArgs([]string{"target", "mist-prod"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.apiLabel != "mist-prod" {
		t.Errorf("apiLabel = %q, want %q", got.apiLabel, "mist-prod")
	}
	if got.templateType != TemplateTypeWLAN {
		t.Errorf("templateType = %q, want default wlan", got.templateType)
	}
	if got.saveMode || got.includeSecrets || got.outputFile != "" {
		t.Errorf("unexpected non-default flags: %+v", got)
	}
}

func TestParseImportTemplatesArgs_Full(t *testing.T) {
	got, err := parseImportTemplatesArgs([]string{"target", "mist-prod", "type", "wlan", "secrets", "save", "file", "import/org.json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.apiLabel != "mist-prod" || got.templateType != TemplateTypeWLAN ||
		!got.saveMode || !got.includeSecrets || got.outputFile != "import/org.json" {
		t.Errorf("parsed args incorrect: %+v", got)
	}
}

func TestParseImportTemplatesArgs_UnsupportedType(t *testing.T) {
	for _, kind := range []string{"rf", "device", "gateway", "all"} {
		if _, err := parseImportTemplatesArgs([]string{"target", "mist-prod", "type", kind}); err == nil {
			t.Errorf("expected error for reserved-but-unimplemented type %q", kind)
		}
	}
}

func TestParseImportTemplatesArgs_InvalidType(t *testing.T) {
	if _, err := parseImportTemplatesArgs([]string{"target", "mist-prod", "type", "banana"}); err == nil {
		t.Error("expected error for invalid type")
	}
}

func TestParseImportTemplatesArgs_MissingTarget(t *testing.T) {
	if _, err := parseImportTemplatesArgs([]string{"type", "wlan"}); err == nil {
		t.Error("expected error when target is missing")
	}
}

func TestParseImportTemplatesArgs_FileWithoutSave(t *testing.T) {
	if _, err := parseImportTemplatesArgs([]string{"target", "mist-prod", "file", "x.json"}); err == nil {
		t.Error("expected error when file is used without save")
	}
}

// TestSynthesizeOrgWLANTemplates_BareSlugs verifies that org-level templates
// get BARE slugs (no site prefix) — the whole point of the separation.
func TestSynthesizeOrgWLANTemplates_BareSlugs(t *testing.T) {
	ws := []*vendors.WLAN{
		{SSID: "Corp Net", Enabled: true, AuthType: "wpa2-enterprise"},
		{SSID: "Guest Wi-Fi", Enabled: true, AuthType: "open"},
	}
	out := synthesizeOrgWLANTemplates(ws, false)
	if len(out) != 2 {
		t.Fatalf("expected 2 templates, got %d", len(out))
	}
	for _, want := range []string{"corp-net", "guest-wi-fi"} {
		if _, ok := out[want]; !ok {
			t.Errorf("missing label %q; got keys: %v", want, mapKeys(out))
		}
	}
}

func TestSynthesizeOrgWLANTemplates_CollisionSuffix(t *testing.T) {
	ws := []*vendors.WLAN{
		{SSID: "Corp Net", Enabled: true, AuthType: "open"},
		{SSID: "corp-net", Enabled: true, AuthType: "psk"},
	}
	out := synthesizeOrgWLANTemplates(ws, false)
	if len(out) != 2 {
		t.Fatalf("expected 2 templates, got %d", len(out))
	}
	for _, want := range []string{"corp-net", "corp-net-2"} {
		if _, ok := out[want]; !ok {
			t.Errorf("missing label %q; got keys: %v", want, mapKeys(out))
		}
	}
}

func TestSynthesizeOrgWLANTemplates_Empty(t *testing.T) {
	if got := synthesizeOrgWLANTemplates(nil, false); got != nil {
		t.Errorf("empty input should return nil, got %v", got)
	}
}

// TestOrgTemplatesRoundTripThroughLoader ensures the template-only envelope is
// directly loadable via config.LoadImportFile.
func TestOrgTemplatesRoundTripThroughLoader(t *testing.T) {
	ws := []*vendors.WLAN{
		{SSID: "Corp Net", Enabled: true, AuthType: "wpa2-enterprise", VLANID: 10},
	}
	env := &importEnvelope{
		Version: 1,
		Source: &importSourceExport{
			API:  "mist-prod",
			Kind: "wlan-templates",
		},
		Templates: &templatesEnvelope{WLAN: synthesizeOrgWLANTemplates(ws, false)},
	}

	data, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var loaded config.ImportFile
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("loader rejected templates envelope: %v", err)
	}
	if loaded.Config != nil {
		t.Errorf("Config should be nil for templates-only import; got %+v", loaded.Config)
	}
	if loaded.Templates == nil || loaded.Templates.WLAN["corp-net"] == nil {
		t.Errorf("corp-net template lost in round-trip; got %+v", loaded.Templates)
	}
	// And the store merge should pick it up.
	store := config.NewTemplateStore()
	config.MergeImportTemplates(store, &loaded)
	if _, ok := store.WLAN["corp-net"]; !ok {
		t.Errorf("MergeImportTemplates did not register corp-net; store=%v", store.ListTemplates())
	}
}

// TestIsTemplatesEmpty covers the early-exit branch used by the "Meraki has
// no org-level WLANs" message.
func TestIsTemplatesEmpty(t *testing.T) {
	if !isTemplatesEmpty(nil) {
		t.Error("nil envelope should be empty")
	}
	if !isTemplatesEmpty(&templatesEnvelope{}) {
		t.Error("zero envelope should be empty")
	}
	if isTemplatesEmpty(&templatesEnvelope{WLAN: map[string]map[string]any{"x": {}}}) {
		t.Error("envelope with WLAN entry should NOT be empty")
	}
}

func mapKeys(m map[string]map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

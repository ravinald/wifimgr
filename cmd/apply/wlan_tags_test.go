package apply

import (
	"strings"
	"testing"

	configPkg "github.com/ravinald/wifimgr/internal/config"
)

func TestGenerateWLANAvailabilityTag(t *testing.T) {
	tests := []struct {
		label string
		want  string
	}{
		{"corp-wifi", "wifimgr-wlan-corp-wifi"},
		{"guest", "wifimgr-wlan-guest"},
		{"iot-network", "wifimgr-wlan-iot-network"},
		{"", "wifimgr-wlan-"},
	}
	for _, tt := range tests {
		got := generateWLANAvailabilityTag(tt.label)
		if got != tt.want {
			t.Errorf("generateWLANAvailabilityTag(%q) = %q, want %q", tt.label, got, tt.want)
		}
	}
}

func TestIsWifimgrManagedTag(t *testing.T) {
	tests := []struct {
		tag  string
		want bool
	}{
		{"wifimgr-wlan-corp-wifi", true},
		{"wifimgr-wlan-guest", true},
		{"wifimgr-wlan-", true},
		{"office-tag", false},
		{"wifimgr-other", false},
		{"", false},
	}
	for _, tt := range tests {
		got := isWifimgrManagedTag(tt.tag)
		if got != tt.want {
			t.Errorf("isWifimgrManagedTag(%q) = %v, want %v", tt.tag, got, tt.want)
		}
	}
}

func TestBuildAPTagMapping(t *testing.T) {
	wlanToDevices := map[string][]string{
		"corp-wifi": {"aa:bb:cc:dd:ee:01", "aa:bb:cc:dd:ee:02"},
		"guest":     {"aa:bb:cc:dd:ee:01"},
		"iot":       {"aa:bb:cc:dd:ee:03"},
	}

	m := buildAPTagMapping(wlanToDevices)

	// AP 01 should have tags for corp-wifi and guest (sorted)
	ap01Tags := m.APToTags["aa:bb:cc:dd:ee:01"]
	if len(ap01Tags) != 2 {
		t.Fatalf("AP 01: expected 2 tags, got %d: %v", len(ap01Tags), ap01Tags)
	}
	if ap01Tags[0] != "wifimgr-wlan-corp-wifi" || ap01Tags[1] != "wifimgr-wlan-guest" {
		t.Errorf("AP 01: unexpected tags %v", ap01Tags)
	}

	// AP 02 should have tag for corp-wifi only
	ap02Tags := m.APToTags["aa:bb:cc:dd:ee:02"]
	if len(ap02Tags) != 1 || ap02Tags[0] != "wifimgr-wlan-corp-wifi" {
		t.Errorf("AP 02: expected [wifimgr-wlan-corp-wifi], got %v", ap02Tags)
	}

	// AP 03 should have tag for iot only
	ap03Tags := m.APToTags["aa:bb:cc:dd:ee:03"]
	if len(ap03Tags) != 1 || ap03Tags[0] != "wifimgr-wlan-iot" {
		t.Errorf("AP 03: expected [wifimgr-wlan-iot], got %v", ap03Tags)
	}

	// AP 04 (not in mapping) should be nil
	if m.APToTags["aa:bb:cc:dd:ee:04"] != nil {
		t.Errorf("AP 04: expected nil, got %v", m.APToTags["aa:bb:cc:dd:ee:04"])
	}
}

func TestBuildAPTagMapping_Empty(t *testing.T) {
	m := buildAPTagMapping(map[string][]string{})
	if len(m.APToTags) != 0 {
		t.Errorf("expected empty mapping, got %v", m.APToTags)
	}
}

func TestMergeAPTags_UserTagsAndWifimgrTags(t *testing.T) {
	current := []string{"office", "wifimgr-wlan-old-ssid", "floor-3"}
	user := []string{"office", "floor-3", "new-tag"}
	required := []string{"wifimgr-wlan-corp-wifi", "wifimgr-wlan-guest"}

	got := mergeAPTags(current, user, required)
	want := []string{"floor-3", "new-tag", "office", "wifimgr-wlan-corp-wifi", "wifimgr-wlan-guest"}

	if !stringSlicesEqual(got, want) {
		t.Errorf("mergeAPTags() = %v, want %v", got, want)
	}
}

func TestMergeAPTags_NoUserTags_PreserveCurrent(t *testing.T) {
	current := []string{"office", "wifimgr-wlan-old-ssid", "floor-3"}
	required := []string{"wifimgr-wlan-corp-wifi"}

	got := mergeAPTags(current, nil, required)
	want := []string{"floor-3", "office", "wifimgr-wlan-corp-wifi"}

	if !stringSlicesEqual(got, want) {
		t.Errorf("mergeAPTags() = %v, want %v", got, want)
	}
}

func TestMergeAPTags_OrphanRemoval(t *testing.T) {
	// Current has wifimgr tags that are no longer needed
	current := []string{"office", "wifimgr-wlan-old-1", "wifimgr-wlan-old-2"}
	required := []string{"wifimgr-wlan-new-1"}

	got := mergeAPTags(current, nil, required)
	want := []string{"office", "wifimgr-wlan-new-1"}

	if !stringSlicesEqual(got, want) {
		t.Errorf("mergeAPTags() = %v, want %v", got, want)
	}
}

func TestMergeAPTags_WifimgrTagsOnly(t *testing.T) {
	required := []string{"wifimgr-wlan-corp-wifi"}

	got := mergeAPTags(nil, nil, required)
	want := []string{"wifimgr-wlan-corp-wifi"}

	if !stringSlicesEqual(got, want) {
		t.Errorf("mergeAPTags() = %v, want %v", got, want)
	}
}

func TestMergeAPTags_EmptyAll(t *testing.T) {
	got := mergeAPTags(nil, nil, nil)
	if len(got) != 0 {
		t.Errorf("mergeAPTags() = %v, want empty", got)
	}
}

func TestMergeAPTags_Deduplication(t *testing.T) {
	user := []string{"office", "office", "floor-3"}
	required := []string{"wifimgr-wlan-corp-wifi", "wifimgr-wlan-corp-wifi"}

	got := mergeAPTags(nil, user, required)
	want := []string{"floor-3", "office", "wifimgr-wlan-corp-wifi"}

	if !stringSlicesEqual(got, want) {
		t.Errorf("mergeAPTags() = %v, want %v", got, want)
	}
}

func TestMergeAPTags_UserRemovesAllWifimgrTags(t *testing.T) {
	// User explicitly sets tags without any wifimgr tags
	// But wifimgr should still add its required tags
	current := []string{"office", "wifimgr-wlan-old"}
	user := []string{"office"} // user removed the wifimgr tag
	required := []string{"wifimgr-wlan-new"}

	got := mergeAPTags(current, user, required)
	want := []string{"office", "wifimgr-wlan-new"}

	if !stringSlicesEqual(got, want) {
		t.Errorf("mergeAPTags() = %v, want %v", got, want)
	}
}

func TestToStringSlice(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want []string
	}{
		{"[]string", []string{"a", "b"}, []string{"a", "b"}},
		{"[]any", []any{"a", "b"}, []string{"a", "b"}},
		{"string", "a", []string{"a"}},
		{"empty string", "", nil},
		{"nil", nil, nil},
		{"int", 42, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toStringSlice(tt.in)
			if !stringSlicesEqual(got, tt.want) {
				t.Errorf("toStringSlice(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

// --- validateWLANAssignments tests ---

func newTemplateStoreWithWLANs(labels ...string) *configPkg.TemplateStore {
	store := configPkg.NewTemplateStore()
	for _, label := range labels {
		store.WLAN[label] = map[string]any{"ssid": label + "-ssid"}
	}
	return store
}

func TestValidateWLANAssignments_AllValid(t *testing.T) {
	siteConfig := SiteConfig{
		Profiles: struct {
			WLAN   []string `json:"wlan,omitempty"`
			Radio  []string `json:"radio,omitempty"`
			Device []string `json:"device,omitempty"`
		}{
			WLAN: []string{"corp-wifi", "guest"},
		},
		WLAN: []string{"corp-wifi"},
		Devices: struct {
			APs      map[string]map[string]any `json:"ap"`
			Switches map[string]map[string]any `json:"switch"`
			WanEdge  map[string]map[string]any `json:"gateway"`
		}{
			APs: map[string]map[string]any{
				"aa:bb:cc:dd:ee:01": {"name": "AP-1", "wlan": []any{"guest"}},
			},
		},
	}
	templates := newTemplateStoreWithWLANs("corp-wifi", "guest")

	err := validateWLANAssignments(siteConfig, templates)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateWLANAssignments_SiteLevelNotInProfiles(t *testing.T) {
	siteConfig := SiteConfig{
		Profiles: struct {
			WLAN   []string `json:"wlan,omitempty"`
			Radio  []string `json:"radio,omitempty"`
			Device []string `json:"device,omitempty"`
		}{
			WLAN: []string{"corp-wifi"},
		},
		WLAN: []string{"corp-wifi", "missing-wlan"},
	}
	templates := newTemplateStoreWithWLANs("corp-wifi")

	err := validateWLANAssignments(siteConfig, templates)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "missing-wlan") {
		t.Errorf("expected error to mention 'missing-wlan', got: %v", err)
	}
	if !strings.Contains(err.Error(), "site-level wlan") {
		t.Errorf("expected error to mention 'site-level wlan', got: %v", err)
	}
}

func TestValidateWLANAssignments_DeviceLevelNotInProfiles(t *testing.T) {
	siteConfig := SiteConfig{
		Profiles: struct {
			WLAN   []string `json:"wlan,omitempty"`
			Radio  []string `json:"radio,omitempty"`
			Device []string `json:"device,omitempty"`
		}{
			WLAN: []string{"corp-wifi"},
		},
		Devices: struct {
			APs      map[string]map[string]any `json:"ap"`
			Switches map[string]map[string]any `json:"switch"`
			WanEdge  map[string]map[string]any `json:"gateway"`
		}{
			APs: map[string]map[string]any{
				"aa:bb:cc:dd:ee:01": {"name": "AP-1", "wlan": []any{"undeclared-wlan"}},
			},
		},
	}
	templates := newTemplateStoreWithWLANs("corp-wifi")

	err := validateWLANAssignments(siteConfig, templates)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "undeclared-wlan") {
		t.Errorf("expected error to mention 'undeclared-wlan', got: %v", err)
	}
	if !strings.Contains(err.Error(), "aa:bb:cc:dd:ee:01") {
		t.Errorf("expected error to mention device MAC, got: %v", err)
	}
}

func TestValidateWLANAssignments_ProfileNotInTemplates(t *testing.T) {
	siteConfig := SiteConfig{
		Profiles: struct {
			WLAN   []string `json:"wlan,omitempty"`
			Radio  []string `json:"radio,omitempty"`
			Device []string `json:"device,omitempty"`
		}{
			WLAN: []string{"corp-wifi", "nonexistent-template"},
		},
	}
	// Only "corp-wifi" exists as a template
	templates := newTemplateStoreWithWLANs("corp-wifi")

	err := validateWLANAssignments(siteConfig, templates)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent-template") {
		t.Errorf("expected error to mention 'nonexistent-template', got: %v", err)
	}
	if !strings.Contains(err.Error(), "no WLAN template") {
		t.Errorf("expected error to mention 'no WLAN template', got: %v", err)
	}
}

func TestValidateWLANAssignments_MultipleErrors(t *testing.T) {
	siteConfig := SiteConfig{
		Profiles: struct {
			WLAN   []string `json:"wlan,omitempty"`
			Radio  []string `json:"radio,omitempty"`
			Device []string `json:"device,omitempty"`
		}{
			WLAN: []string{"corp-wifi", "phantom"},
		},
		WLAN: []string{"undeclared-site"},
		Devices: struct {
			APs      map[string]map[string]any `json:"ap"`
			Switches map[string]map[string]any `json:"switch"`
			WanEdge  map[string]map[string]any `json:"gateway"`
		}{
			APs: map[string]map[string]any{
				"aa:bb:cc:dd:ee:01": {"wlan": []any{"undeclared-device"}},
			},
		},
	}
	templates := newTemplateStoreWithWLANs("corp-wifi")

	err := validateWLANAssignments(siteConfig, templates)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	errMsg := err.Error()
	// Should report all three errors
	if !strings.Contains(errMsg, "undeclared-site") {
		t.Errorf("expected error to mention 'undeclared-site', got: %v", errMsg)
	}
	if !strings.Contains(errMsg, "undeclared-device") {
		t.Errorf("expected error to mention 'undeclared-device', got: %v", errMsg)
	}
	if !strings.Contains(errMsg, "phantom") {
		t.Errorf("expected error to mention 'phantom', got: %v", errMsg)
	}
}

func TestValidateWLANAssignments_EmptyConfig(t *testing.T) {
	siteConfig := SiteConfig{}
	templates := configPkg.NewTemplateStore()

	err := validateWLANAssignments(siteConfig, templates)
	if err != nil {
		t.Errorf("expected no error for empty config, got: %v", err)
	}
}

func TestValidateWLANAssignments_NilTemplates(t *testing.T) {
	siteConfig := SiteConfig{
		Profiles: struct {
			WLAN   []string `json:"wlan,omitempty"`
			Radio  []string `json:"radio,omitempty"`
			Device []string `json:"device,omitempty"`
		}{
			WLAN: []string{"corp-wifi"},
		},
	}

	// nil templates should skip the template-exists check
	err := validateWLANAssignments(siteConfig, nil)
	if err != nil {
		t.Errorf("expected no error with nil templates, got: %v", err)
	}
}

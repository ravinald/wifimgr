package cmd

import (
	"path/filepath"
	"testing"

	"github.com/spf13/viper"

	"github.com/ravinald/wifimgr/internal/config"
)

// armedAPs reloads the allowlist and returns a site's armed AP MACs.
func armedAPs(t *testing.T, path, site string) []string {
	t.Helper()
	f, err := config.LoadInventoryFile(path)
	if err != nil {
		t.Fatalf("reload inventory: %v", err)
	}
	return f.MACsForSite(site, "ap")
}

func TestApplyArming_ArmThenDisarm(t *testing.T) {
	path := filepath.Join(t.TempDir(), "inventory.json")
	viper.Set("files.inventory", path)
	t.Cleanup(func() { viper.Set("files.inventory", "") })

	targets := []armTarget{
		{mac: "683a1e54490f", name: "AP-01", dtype: "ap"},
		{mac: "aabbccddeeff", name: "AP-02", dtype: "ap"},
	}

	if err := applyArming("ZZ-TMP-SITE", targets, true); err != nil {
		t.Fatalf("arm: %v", err)
	}
	if got := armedAPs(t, path, "ZZ-TMP-SITE"); len(got) != 2 {
		t.Fatalf("after arm: %v, want 2 APs", got)
	}

	// Re-arming is idempotent: still two, no duplicates.
	if err := applyArming("ZZ-TMP-SITE", targets, true); err != nil {
		t.Fatalf("re-arm: %v", err)
	}
	if got := armedAPs(t, path, "ZZ-TMP-SITE"); len(got) != 2 {
		t.Fatalf("after re-arm: %v, want 2 APs", got)
	}

	// Disarm one; the other remains.
	if err := applyArming("ZZ-TMP-SITE", targets[:1], false); err != nil {
		t.Fatalf("disarm: %v", err)
	}
	got := armedAPs(t, path, "ZZ-TMP-SITE")
	if len(got) != 1 || got[0] != "aabbccddeeff" {
		t.Fatalf("after disarm: %v, want [aabbccddeeff]", got)
	}
}

func TestArmActionFor(t *testing.T) {
	cases := []struct {
		token   string
		wantArm bool
		wantOK  bool
	}{
		{"managed", true, true},
		{"unmanaged", false, true},
		{"online", false, false},
	}
	for _, c := range cases {
		arm, ok := armActionFor(c.token)
		if arm != c.wantArm || ok != c.wantOK {
			t.Errorf("armActionFor(%q) = (%v,%v), want (%v,%v)", c.token, arm, ok, c.wantArm, c.wantOK)
		}
	}
}

func TestDisplayName(t *testing.T) {
	if got := displayName("AP-01", "mac"); got != "AP-01" {
		t.Errorf("named: got %q", got)
	}
	if got := displayName("", "683a1e54490f"); got != "683a1e54490f" {
		t.Errorf("unnamed should fall back to MAC: got %q", got)
	}
}

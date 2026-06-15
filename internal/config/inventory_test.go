package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeTempInventory(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "inventory.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write temp inventory: %v", err)
	}
	return path
}

func TestLoadInventoryFile_PerSite(t *testing.T) {
	path := writeTempInventory(t, `{
	  "version": 1,
	  "config": {
	    "inventory": {
	      "site": {
	        "US-LAB-01": {"ap": ["aa:bb:cc:dd:ee:ff"], "switch": ["3c:08:cd:2c:3e:d0"], "gateway": []},
	        "US-LAB-02": {"ap": ["11:22:33:44:55:66"]}
	      }
	    }
	  }
	}`)

	f, err := LoadInventoryFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Case-insensitive site match.
	if got := f.MACsForSite("us-lab-01", "ap"); len(got) != 1 || got[0] != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("MACsForSite(us-lab-01, ap) = %v", got)
	}
	if got := f.MACsForSite("US-LAB-01", "switch"); len(got) != 1 {
		t.Errorf("MACsForSite(US-LAB-01, switch) = %v", got)
	}

	// Per-site scoping: a MAC armed for US-LAB-02 must not appear under US-LAB-01.
	lab1 := f.NormalizedSet([]string{"US-LAB-01"}, "")
	if lab1["112233445566"] {
		t.Error("US-LAB-02 MAC leaked into US-LAB-01 scope")
	}
	if !lab1["aabbccddeeff"] || !lab1["3c08cd2c3ed0"] {
		t.Errorf("US-LAB-01 normalized set missing expected MACs: %v", lab1)
	}

	// Union across all armed sites when none specified.
	all := f.NormalizedSet(nil, "ap")
	if !all["aabbccddeeff"] || !all["112233445566"] {
		t.Errorf("all-sites AP set = %v", all)
	}
}

func TestLoadInventoryFile_LegacySchemaFailsLoud(t *testing.T) {
	path := writeTempInventory(t, `{
	  "version": 1,
	  "config": {"inventory": {"ap": ["aa:bb:cc:dd:ee:ff"], "switch": [], "gateway": []}}
	}`)

	_, err := LoadInventoryFile(path)
	if !errors.Is(err, ErrLegacyInventorySchema) {
		t.Fatalf("expected ErrLegacyInventorySchema, got %v", err)
	}
}

func TestArmSiteDevices_CreatesAndNormalizes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "inventory.json")

	// File absent -> created. Mixed input formats -> stored bare hex.
	if err := ArmSiteDevices(path, "ZZ-TMP-SITE",
		[]string{"68:3A:1E:54:49:0F", "aabb.ccdd.eeff"}, nil, nil, ""); err != nil {
		t.Fatalf("ArmSiteDevices create: %v", err)
	}

	f, err := LoadInventoryFile(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	got := f.MACsForSite("ZZ-TMP-SITE", "ap")
	want := []string{"683a1e54490f", "aabbccddeeff"}
	if len(got) != len(want) {
		t.Fatalf("ap MACs = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ap[%d] = %q, want %q (bare hex)", i, got[i], want[i])
		}
	}
}

func TestArmSiteDevices_IdempotentAndPreservesOtherSites(t *testing.T) {
	path := writeTempInventory(t, `{
	  "version": 1,
	  "config": {"inventory": {"site": {
	    "OTHER": {"ap": ["112233445566"], "switch": [], "gateway": []}
	  }}}
	}`)

	arm := func(note string) {
		if err := ArmSiteDevices(path, "ZZ-TMP-SITE", []string{"68:3a:1e:54:49:0f"}, nil, nil, note); err != nil {
			t.Fatalf("arm: %v", err)
		}
	}
	arm("")
	arm("") // re-run must not duplicate

	f, err := LoadInventoryFile(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got := f.MACsForSite("ZZ-TMP-SITE", "ap"); len(got) != 1 || got[0] != "683a1e54490f" {
		t.Errorf("idempotent arm produced %v", got)
	}
	if got := f.MACsForSite("OTHER", "ap"); len(got) != 1 || got[0] != "112233445566" {
		t.Errorf("other site clobbered: %v", got)
	}

	// A note stamps onto the site section.
	arm("managed by Meraki config template Lab-Template")
	f, _ = LoadInventoryFile(path)
	si, _ := f.site("ZZ-TMP-SITE")
	if si.Note == "" {
		t.Error("expected _note to be stamped")
	}
}

func TestLoadInventoryFile_EmptyIsValid(t *testing.T) {
	path := writeTempInventory(t, `{"version": 1, "config": {"inventory": {}}}`)

	f, err := LoadInventoryFile(path)
	if err != nil {
		t.Fatalf("empty inventory should load, got %v", err)
	}
	if len(f.SiteNames()) != 0 {
		t.Errorf("expected no armed sites, got %v", f.SiteNames())
	}
	if len(f.NormalizedSet(nil, "")) != 0 {
		t.Error("expected empty normalized set")
	}
}

func TestDisarmSiteDevices_RemovesAndCounts(t *testing.T) {
	path := writeTempInventory(t, `{
	  "version": 1,
	  "config": {"inventory": {"site": {
	    "ZZ-TMP-SITE": {"ap": ["683a1e54490f", "aabbccddeeff"], "switch": ["3c08cd2c3ed0"], "gateway": []}
	  }}}
	}`)

	// Remove one AP by a differently-formatted MAC; the switch stays put.
	removed, err := DisarmSiteDevices(path, "zz-tmp-site", []string{"68:3A:1E:54:49:0F"}, nil, nil)
	if err != nil {
		t.Fatalf("disarm: %v", err)
	}
	if removed != 1 {
		t.Fatalf("removed = %d, want 1", removed)
	}

	f, _ := LoadInventoryFile(path)
	if got := f.MACsForSite("ZZ-TMP-SITE", "ap"); len(got) != 1 || got[0] != "aabbccddeeff" {
		t.Errorf("ap after disarm = %v, want [aabbccddeeff]", got)
	}
	if got := f.MACsForSite("ZZ-TMP-SITE", "switch"); len(got) != 1 {
		t.Errorf("switch should be untouched, got %v", got)
	}
}

func TestDisarmSiteDevices_PrunesEmptySite(t *testing.T) {
	path := writeTempInventory(t, `{
	  "version": 1,
	  "config": {"inventory": {"site": {
	    "ZZ-TMP-SITE": {"ap": ["683a1e54490f"], "switch": [], "gateway": []},
	    "OTHER": {"ap": ["112233445566"]}
	  }}}
	}`)

	if _, err := DisarmSiteDevices(path, "ZZ-TMP-SITE", []string{"683a1e54490f"}, nil, nil); err != nil {
		t.Fatalf("disarm: %v", err)
	}

	f, _ := LoadInventoryFile(path)
	for _, name := range f.SiteNames() {
		if name == "ZZ-TMP-SITE" {
			t.Error("emptied site should have been pruned")
		}
	}
	if got := f.MACsForSite("OTHER", "ap"); len(got) != 1 {
		t.Errorf("other site clobbered: %v", got)
	}
}

func TestDisarmSiteDevices_KeepsEmptySiteWithNote(t *testing.T) {
	path := writeTempInventory(t, `{
	  "version": 1,
	  "config": {"inventory": {"site": {
	    "ZZ-TMP-SITE": {"ap": ["683a1e54490f"], "switch": [], "gateway": [], "_note": "template-managed"}
	  }}}
	}`)

	if _, err := DisarmSiteDevices(path, "ZZ-TMP-SITE", []string{"683a1e54490f"}, nil, nil); err != nil {
		t.Fatalf("disarm: %v", err)
	}

	f, _ := LoadInventoryFile(path)
	si, ok := f.site("ZZ-TMP-SITE")
	if !ok {
		t.Fatal("site with a note must survive pruning")
	}
	if si.Note != "template-managed" {
		t.Errorf("note lost: %q", si.Note)
	}
}

func TestDisarmSiteDevices_MissingFileNoOp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "inventory.json")
	removed, err := DisarmSiteDevices(path, "ZZ-TMP-SITE", []string{"683a1e54490f"}, nil, nil)
	if err != nil {
		t.Fatalf("missing file should be a no-op, got %v", err)
	}
	if removed != 0 {
		t.Errorf("removed = %d, want 0", removed)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("disarm must not create the file")
	}
}

func TestDisarmSiteDevices_IdempotentUnknownMAC(t *testing.T) {
	path := writeTempInventory(t, `{
	  "version": 1,
	  "config": {"inventory": {"site": {
	    "ZZ-TMP-SITE": {"ap": ["683a1e54490f"], "switch": [], "gateway": []}
	  }}}
	}`)

	removed, err := DisarmSiteDevices(path, "ZZ-TMP-SITE", []string{"00:00:00:00:00:01"}, nil, nil)
	if err != nil {
		t.Fatalf("disarm: %v", err)
	}
	if removed != 0 {
		t.Errorf("removing an absent MAC should report 0, got %d", removed)
	}
	f, _ := LoadInventoryFile(path)
	if got := f.MACsForSite("ZZ-TMP-SITE", "ap"); len(got) != 1 {
		t.Errorf("armed MAC should remain, got %v", got)
	}
}

func TestDisarmSiteDevices_LegacySchemaFailsLoud(t *testing.T) {
	path := writeTempInventory(t, `{
	  "version": 1,
	  "config": {"inventory": {"ap": ["aa:bb:cc:dd:ee:ff"], "switch": [], "gateway": []}}
	}`)

	if _, err := DisarmSiteDevices(path, "ZZ-TMP-SITE", []string{"aa:bb:cc:dd:ee:ff"}, nil, nil); !errors.Is(err, ErrLegacyInventorySchema) {
		t.Fatalf("expected ErrLegacyInventorySchema, got %v", err)
	}
}

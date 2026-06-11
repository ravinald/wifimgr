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

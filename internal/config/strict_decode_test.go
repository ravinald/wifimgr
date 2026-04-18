package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLoadImportFile_RejectsUnknownEnvelopeField verifies that a typo at
// the top level of an ImportFile (e.g. "template" instead of "templates")
// produces a parse error with the offending field name, instead of silently
// dropping the section.
func TestLoadImportFile_RejectsUnknownEnvelopeField(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")

	// "template" (singular) is a common typo for "templates"
	body := `{
		"version": 1,
		"template": {"wlan": {}}
	}`
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	_, err := LoadImportFile(path)
	if err == nil {
		t.Fatal("expected error for unknown envelope field, got nil")
	}
	if !strings.Contains(err.Error(), "unknown top-level field") {
		t.Errorf("error should mention unknown field: %v", err)
	}
	if !strings.Contains(err.Error(), `"template"`) {
		t.Errorf("error should quote the offending field name: %v", err)
	}
}

// TestLoadImportFile_AcceptsKnownEnvelopeFields verifies that valid top-level
// fields (version, source, config, templates) all load without error.
func TestLoadImportFile_AcceptsKnownEnvelopeFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "good.json")

	body := `{
		"version": 1,
		"source": {"api": "mist-test", "site": "US-LAB-01", "kind": "site"},
		"config": {"sites": {}},
		"templates": {"wlan": {}}
	}`
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	imp, err := LoadImportFile(path)
	if err != nil {
		t.Fatalf("LoadImportFile: %v", err)
	}
	if imp.Version != 1 {
		t.Errorf("version = %d, want 1", imp.Version)
	}
	if imp.Source.API != "mist-test" {
		t.Errorf("source.api = %q, want mist-test", imp.Source.API)
	}
}

// TestLoadImportFile_PayloadRemainsPermissive verifies that vendor-snapshot
// fields (serial, model, site_id) in the Config.sites section are accepted.
// These come from real imported files and must not break round-trip.
func TestLoadImportFile_PayloadRemainsPermissive(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "with_snapshot.json")

	// APConfig doesn't declare "serial" but real imports write it. The
	// envelope check must not recurse into this payload.
	body := `{
		"version": 1,
		"config": {
			"sites": {
				"US-LAB-01": {
					"site_config": {"name": "US-LAB-01"},
					"devices": {
						"ap": {
							"aa:bb:cc:dd:ee:ff": {
								"mac": "aa:bb:cc:dd:ee:ff",
								"serial": "FOC12345678",
								"model": "AP43"
							}
						},
						"switch": {},
						"gateway": {}
					}
				}
			}
		}
	}`
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if _, err := LoadImportFile(path); err != nil {
		t.Errorf("expected permissive payload decode to succeed: %v", err)
	}
}

// TestLoadConfig_RejectsUnknownField verifies that a typo in the main
// wifimgr-config.json produces a parse error instead of silently dropping
// the field.
func TestLoadConfig_RejectsUnknownField(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wifimgr-config.json")

	// "logging_level" is a typo; the real key is under a nested logging block.
	body := `{
		"version": 1,
		"logging_level": "debug",
		"files": {
			"config_dir": "./config",
			"site_configs": [],
			"cache": "./cache",
			"inventory": "./inv.json",
			"log_file": "./log",
			"schemas": "./schemas"
		},
		"api": {
			"credentials": {"api_id": "", "api_token": "", "org_id": ""},
			"url": "", "rate_limit": 10, "results_limit": 100
		},
		"display": {
			"sites": {"format":"table","fields":[]},
			"aps": {"format":"table","fields":[]},
			"inventory": {"format":"table","fields":[]},
			"commands": {}
		},
		"logging": {"enable": false, "level": "info", "format": "text", "stdout": true}
	}`
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for unknown top-level field, got nil")
	}
	if !strings.Contains(err.Error(), "logging_level") {
		t.Errorf("error should mention the offending field: %v", err)
	}
}

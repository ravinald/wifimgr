package intent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ravinald/wifimgr/internal/keypath"
)

const sampleConfig = `{
  "version": 1,
  "config": {
    "sites": {
      "us-lab-01": {
        "site_config": {"name": "US-LAB-01"},
        "devices": {
          "ap": {
            "aabbccddeeff": {
              "mac": "aabbccddeeff",
              "name": "AP-01",
              "radio_config": {"band_5": {"channel": 36, "power": 10}}
            }
          }
        }
      }
    }
  }
}`

func writeSample(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "site.json")
	if err := os.WriteFile(path, []byte(sampleConfig), 0600); err != nil {
		t.Fatalf("write sample: %v", err)
	}
	return path
}

func readField(t *testing.T, path string, segments ...string) any {
	t.Helper()
	raw, err := os.ReadFile(path) //nolint:gosec // test-controlled temp path
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		t.Fatalf("parse back: %v", err)
	}
	full := append([]string{"config", "sites", "us-lab-01", "devices", "ap", "aabbccddeeff"}, segments...)
	v, _ := keypath.GetValueAtPath(root, full)
	return v
}

func baseOpts(path string) SetOptions {
	return SetOptions{
		ConfigFilePath: path,
		SiteKey:        "us-lab-01",
		DeviceType:     "ap",
		DeviceName:     "AP-01",
		// SchemaDir empty: engine mutation is exercised without the schema file.
	}
}

func TestSetDeviceFieldsWritesValue(t *testing.T) {
	path := writeSample(t)
	results, err := SetDeviceFields(baseOpts(path), []FieldChange{
		{KeyPath: "radio_config.band_5.channel", Value: int64(149)},
	})
	if err != nil {
		t.Fatalf("SetDeviceFields: %v", err)
	}
	if len(results) != 1 || !results[0].Changed {
		t.Fatalf("expected one changed result, got %+v", results)
	}
	if results[0].MAC != "aabbccddeeff" {
		t.Errorf("resolved MAC = %q, want aabbccddeeff", results[0].MAC)
	}
	if got := readField(t, path, "radio_config", "band_5", "channel"); got != float64(149) {
		t.Errorf("persisted channel = %v, want 149", got)
	}
}

func TestSetDeviceFieldsMultipleAtomic(t *testing.T) {
	path := writeSample(t)
	_, err := SetDeviceFields(baseOpts(path), []FieldChange{
		{KeyPath: "radio_config.band_5.channel", Value: int64(44)},
		{KeyPath: "radio_config.band_5.power", Value: int64(20)},
	})
	if err != nil {
		t.Fatalf("SetDeviceFields: %v", err)
	}
	if got := readField(t, path, "radio_config", "band_5", "channel"); got != float64(44) {
		t.Errorf("channel = %v, want 44", got)
	}
	if got := readField(t, path, "radio_config", "band_5", "power"); got != float64(20) {
		t.Errorf("power = %v, want 20", got)
	}
}

func TestSetDeviceFieldsNoOp(t *testing.T) {
	path := writeSample(t)
	results, err := SetDeviceFields(baseOpts(path), []FieldChange{
		{KeyPath: "radio_config.band_5.power", Value: int64(10)}, // already 10
	})
	if err != nil {
		t.Fatalf("SetDeviceFields: %v", err)
	}
	if results[0].Changed {
		t.Errorf("expected Changed=false for identical value, got %+v", results[0])
	}
}

func TestSetDeviceFieldsErrors(t *testing.T) {
	path := writeSample(t)
	cases := map[string]struct {
		opts    SetOptions
		changes []FieldChange
	}{
		"unknown device": {
			opts:    SetOptions{ConfigFilePath: path, SiteKey: "us-lab-01", DeviceType: "ap", DeviceName: "AP-99"},
			changes: []FieldChange{{KeyPath: "radio_config.band_5.channel", Value: int64(1)}},
		},
		"unsupported type": {
			opts:    SetOptions{ConfigFilePath: path, SiteKey: "us-lab-01", DeviceType: "router", DeviceName: "AP-01"},
			changes: []FieldChange{{KeyPath: "radio_config.band_5.channel", Value: int64(1)}},
		},
		"bad key path": {
			opts:    baseOpts(path),
			changes: []FieldChange{{KeyPath: "radio_config..channel", Value: int64(1)}},
		},
		"no changes": {
			opts:    baseOpts(path),
			changes: nil,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := SetDeviceFields(tc.opts, tc.changes); err == nil {
				t.Fatal("expected an error, got nil")
			}
		})
	}
}

// schemaDir locates the repo's canonical schema set from the package dir.
const repoSchemaDir = "../../config/schemas"

func TestSetDeviceFieldsSchemaRejectsOutOfRange(t *testing.T) {
	if _, err := os.Stat(filepath.Join(repoSchemaDir, "site-config-schema.json")); err != nil {
		t.Skipf("schema not available: %v", err)
	}
	path := writeSample(t)
	opts := baseOpts(path)
	opts.SchemaDir = repoSchemaDir

	// Power 99 dBm is outside the schema's 1-30 range and must be rejected.
	if _, err := SetDeviceFields(opts, []FieldChange{
		{KeyPath: "radio_config.band_5.power", Value: int64(99)},
	}); err == nil {
		t.Fatal("expected schema validation to reject power 99, got nil")
	}
	// A rejected change must not reach disk.
	if got := readField(t, path, "radio_config", "band_5", "power"); got != float64(10) {
		t.Errorf("power changed on disk to %v despite rejection, want 10", got)
	}

	// A valid power passes and persists.
	if _, err := SetDeviceFields(opts, []FieldChange{
		{KeyPath: "radio_config.band_5.power", Value: int64(15)},
	}); err != nil {
		t.Fatalf("valid power 15 rejected: %v", err)
	}
	if got := readField(t, path, "radio_config", "band_5", "power"); got != float64(15) {
		t.Errorf("power = %v after valid set, want 15", got)
	}
}

func TestCoerceValue(t *testing.T) {
	cases := map[string]any{
		"36":    int64(36),
		"true":  true,
		"false": false,
		"2.5":   2.5,
		"auto":  "auto",
	}
	for in, want := range cases {
		if got := CoerceValue(in); got != want {
			t.Errorf("CoerceValue(%q) = %v (%T), want %v (%T)", in, got, got, want, want)
		}
	}
}

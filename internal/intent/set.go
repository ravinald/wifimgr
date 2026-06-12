// Package intent mutates the JSON site config — wifimgr's vendor-neutral
// desired-state store — through validated, schema-checked field writes. It is
// the write counterpart to the read/diff/apply path: callers change intent
// here, then apply pushes it to the vendor. The package never touches a vendor
// API itself, preserving the intent/executor boundary.
package intent

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ravinald/wifimgr/internal/jsonschema"
	"github.com/ravinald/wifimgr/internal/keypath"
)

// SetOptions locates the device whose fields are being written.
type SetOptions struct {
	ConfigFilePath string // absolute path to the site config file
	SiteKey        string // key under config.sites (from config.GetSiteConfigKey)
	DeviceType     string // "ap", "switch", or "gateway"
	DeviceName     string // device "name" field, resolved to its MAC key
	SchemaDir      string // schema dir for validation; empty skips validation
}

// FieldChange is one field write under a device.
type FieldChange struct {
	KeyPath string // dot path under the device, e.g. "radio_config.band_5.channel"
	Value   any    // already-typed value to write
}

// Result reports the outcome of one field write.
type Result struct {
	ConfigFile string
	DeviceType string
	DeviceName string
	MAC        string
	KeyPath    string
	OldValue   any
	NewValue   any
	Changed    bool
}

// deviceTypeKeys maps a normalized device type to the key used under "devices"
// in the site config. Gateways live under "gateway" in the file.
var deviceTypeKeys = map[string]string{
	"ap":      "ap",
	"switch":  "switch",
	"gateway": "gateway",
}

// SetDeviceFields writes one or more fields into a device's config in a site
// file. It loads the file, resolves the device by name to its MAC key, applies
// every change, validates the whole file against the site-config schema, then
// writes it back. Validation runs before the write, so an invalid change never
// reaches disk — and the batch is all-or-nothing, since a rejected file is not
// written at all.
func SetDeviceFields(opts SetOptions, changes []FieldChange) ([]Result, error) {
	if len(changes) == 0 {
		return nil, fmt.Errorf("intent: no field changes given")
	}
	for _, c := range changes {
		if err := keypath.Validate(c.KeyPath); err != nil {
			return nil, fmt.Errorf("intent: invalid key path %q: %w", c.KeyPath, err)
		}
	}
	devicesKey, ok := deviceTypeKeys[opts.DeviceType]
	if !ok {
		return nil, fmt.Errorf("intent: unsupported device type %q", opts.DeviceType)
	}

	raw, err := os.ReadFile(opts.ConfigFilePath) // #nosec G304 -- path from operator-controlled config
	if err != nil {
		return nil, fmt.Errorf("intent: read %s: %w", opts.ConfigFilePath, err)
	}

	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, fmt.Errorf("intent: parse %s: %w", opts.ConfigFilePath, err)
	}

	devices, err := deviceMap(root, opts.SiteKey, devicesKey)
	if err != nil {
		return nil, err
	}

	mac, deviceCfg, err := resolveDeviceByName(devices, opts.DeviceName)
	if err != nil {
		return nil, err
	}

	results := make([]Result, 0, len(changes))
	for _, c := range changes {
		segments := keypath.Parse(c.KeyPath).Segments
		oldValue, _ := keypath.GetValueAtPath(deviceCfg, segments)
		keypath.SetValueAtPath(deviceCfg, segments, c.Value)
		results = append(results, Result{
			ConfigFile: opts.ConfigFilePath,
			DeviceType: opts.DeviceType,
			DeviceName: opts.DeviceName,
			MAC:        mac,
			KeyPath:    c.KeyPath,
			OldValue:   oldValue,
			NewValue:   c.Value,
			Changed:    !equalJSON(oldValue, c.Value),
		})
	}

	if opts.SchemaDir != "" {
		if err := validateSiteConfig(root, opts.SchemaDir); err != nil {
			return nil, err
		}
	}

	stampModified(root, opts.SiteKey)

	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("intent: marshal config: %w", err)
	}
	if err := os.WriteFile(opts.ConfigFilePath, out, 0600); err != nil {
		return nil, fmt.Errorf("intent: write %s: %w", opts.ConfigFilePath, err)
	}

	return results, nil
}

// deviceMap navigates config.sites.<siteKey>.devices.<devicesKey> and returns
// the MAC-keyed device map, erroring with a precise path when a level is absent.
func deviceMap(root map[string]any, siteKey, devicesKey string) (map[string]any, error) {
	config, ok := root["config"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("intent: config file has no 'config' object")
	}
	sites, ok := config["sites"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("intent: config file has no 'config.sites' object")
	}
	site, ok := sites[siteKey].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("intent: site %q not found in config file", siteKey)
	}
	devicesSection, ok := site["devices"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("intent: site %q has no devices", siteKey)
	}
	devices, ok := devicesSection[devicesKey].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("intent: site %q has no %s devices", siteKey, devicesKey)
	}
	return devices, nil
}

// resolveDeviceByName finds the device whose "name" field matches name
// (case-insensitive) and returns its MAC key and config map.
func resolveDeviceByName(devices map[string]any, name string) (string, map[string]any, error) {
	for mac, v := range devices {
		cfg, ok := v.(map[string]any)
		if !ok {
			continue
		}
		if n, _ := cfg["name"].(string); strings.EqualFold(n, name) {
			return mac, cfg, nil
		}
	}
	return "", nil, fmt.Errorf("intent: no device named %q in site config", name)
}

func validateSiteConfig(root map[string]any, schemaDir string) error {
	v := jsonschema.New(schemaDir)
	if err := v.LoadSchema("site-config", "site-config-schema.json"); err != nil {
		return fmt.Errorf("intent: load schema: %w", err)
	}
	ok, err := v.ValidateData("site-config", root)
	if err != nil {
		return fmt.Errorf("intent: schema validation failed: %w", err)
	}
	if !ok {
		return fmt.Errorf("intent: change rejected by schema validation")
	}
	return nil
}

// stampModified records an edit time at the root and on the touched site, so a
// reader can tell intent files apart by recency the same way apply backups do.
func stampModified(root map[string]any, siteKey string) {
	now := time.Now().UTC().Format(time.RFC3339)
	root["last_modified"] = now
	if config, ok := root["config"].(map[string]any); ok {
		if sites, ok := config["sites"].(map[string]any); ok {
			if site, ok := sites[siteKey].(map[string]any); ok {
				site["last_modified"] = now
			}
		}
	}
}

// CoerceValue turns a CLI string argument into a typed JSON value so schema
// validation sees numbers and booleans as their real types, not strings. It
// tries bool, then integer, then float, and falls back to the raw string.
func CoerceValue(s string) any {
	if b, err := strconv.ParseBool(s); err == nil {
		return b
	}
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	return s
}

// equalJSON compares two decoded JSON values for equality so a no-op write is
// reported as unchanged.
func equalJSON(a, b any) bool {
	ab, err1 := json.Marshal(a)
	bb, err2 := json.Marshal(b)
	if err1 != nil || err2 != nil {
		return false
	}
	return string(ab) == string(bb)
}

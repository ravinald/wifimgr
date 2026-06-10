package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"

	"github.com/ravinald/wifimgr/internal/macaddr"
)

// ErrLegacyInventorySchema marks an inventory.json that still uses the global
// layout (config.inventory.ap/switch/gateway). Per-site scoping is the write
// safety boundary, so loading must refuse the old shape outright rather than
// reinterpret a flat list as "every site" — that would silently widen the
// blast radius instead of narrowing it.
var ErrLegacyInventorySchema = errors.New("legacy inventory schema")

// SiteInventory is the armed allowlist for a single site: the MACs the operator
// permits configuration writes against. Scoping per-site means the decision to
// modify a device never depends on the (stale) cached site assignment.
type SiteInventory struct {
	AP      []string `json:"ap"`
	Switch  []string `json:"switch"`
	Gateway []string `json:"gateway"`
}

// InventoryFile is the on-disk shape of inventory.json.
type InventoryFile struct {
	Version  int `json:"version"`
	Metadata struct {
		Description string `json:"description"`
	} `json:"metadata"`
	Config struct {
		Inventory struct {
			Site map[string]SiteInventory `json:"site"`
		} `json:"inventory"`
	} `json:"config"`
}

// InventoryPath resolves the inventory.json path the way every caller needs it:
// Viper first (which carries flag/env overrides), then the loaded config struct.
func InventoryPath(cfg *Config) string {
	path := viper.GetString("files.inventory")
	if path == "" && cfg != nil {
		path = cfg.Files.Inventory
	}
	return path
}

// LoadInventoryFile reads and validates inventory.json. It fails loud on the
// legacy global schema (see ErrLegacyInventorySchema). A missing file is the
// caller's call to handle — they get the os error and decide whether absence is
// fatal (writes) or benign (reads default to "manage nothing").
func LoadInventoryFile(path string) (*InventoryFile, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- path from operator-controlled config
	if err != nil {
		return nil, err
	}

	// Probe the inventory object's keys before committing to a shape. A
	// legacy file keys ap/switch/gateway directly; the new shape keys "site".
	var probe struct {
		Config struct {
			Inventory map[string]json.RawMessage `json:"inventory"`
		} `json:"config"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, fmt.Errorf("inventory: parse %s: %w", path, err)
	}
	inv := probe.Config.Inventory
	if _, hasSite := inv["site"]; !hasSite {
		_, hasAP := inv["ap"]
		_, hasSwitch := inv["switch"]
		_, hasGateway := inv["gateway"]
		if hasAP || hasSwitch || hasGateway {
			return nil, fmt.Errorf("%w: %s uses config.inventory.<type>; migrate to per-site "+
				"config.inventory.site.<SITE>.<type>", ErrLegacyInventorySchema, path)
		}
	}

	var f InventoryFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("inventory: parse %s: %w", path, err)
	}
	return &f, nil
}

// site returns the allowlist for a site name, matched case-insensitively to
// stay consistent with the site index. The bool reports whether the site is
// armed at all.
func (f *InventoryFile) site(siteName string) (SiteInventory, bool) {
	if f == nil || f.Config.Inventory.Site == nil {
		return SiteInventory{}, false
	}
	if si, ok := f.Config.Inventory.Site[siteName]; ok {
		return si, true
	}
	lower := strings.ToLower(siteName)
	for name, si := range f.Config.Inventory.Site {
		if strings.ToLower(name) == lower {
			return si, true
		}
	}
	return SiteInventory{}, false
}

// MACsForSite returns the raw (un-normalized) armed MACs for a site and device
// type. deviceType is "ap", "switch", or "gateway".
func (f *InventoryFile) MACsForSite(siteName, deviceType string) []string {
	si, ok := f.site(siteName)
	if !ok {
		return nil
	}
	switch deviceType {
	case "ap":
		return si.AP
	case "switch":
		return si.Switch
	case "gateway":
		return si.Gateway
	default:
		return nil
	}
}

// SiteNames returns the armed site names (the keys present in the file).
func (f *InventoryFile) SiteNames() []string {
	if f == nil {
		return nil
	}
	names := make([]string, 0, len(f.Config.Inventory.Site))
	for name := range f.Config.Inventory.Site {
		names = append(names, name)
	}
	return names
}

// NormalizedSet returns the union of armed MACs (normalized, uppercase, no
// separators) across the given sites. deviceType "" unions all three types;
// an empty siteNames slice unions every armed site.
func (f *InventoryFile) NormalizedSet(siteNames []string, deviceType string) map[string]bool {
	set := make(map[string]bool)
	if f == nil {
		return set
	}
	if len(siteNames) == 0 {
		siteNames = f.SiteNames()
	}
	types := []string{deviceType}
	if deviceType == "" {
		types = []string{"ap", "switch", "gateway"}
	}
	for _, site := range siteNames {
		for _, t := range types {
			for _, mac := range f.MACsForSite(site, t) {
				if n := macaddr.NormalizeOrEmpty(mac); n != "" {
					set[n] = true
				}
			}
		}
	}
	return set
}

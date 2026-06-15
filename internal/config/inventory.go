package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	// Note carries an operator-facing annotation, set by import when a site holds
	// template/profile-managed devices that direct-to-device push can't fully own.
	// JSON has no comment syntax, so the warning rides as a data field; loaders
	// ignore unknown keys, so it never affects allowlist evaluation.
	Note string `json:"_note,omitempty"`
}

// inventoryDescription is the metadata blurb stamped on a freshly created
// inventory.json so the file explains itself to whoever opens it next.
const inventoryDescription = "Per-site armed allowlist: devices wifimgr may write configuration to"

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

// siteKey returns the map key under which a site is stored, matched
// case-insensitively, plus whether it already exists. Callers writing the file
// reuse the existing key so a case variance doesn't split one site into two
// entries.
func (f *InventoryFile) siteKey(siteName string) (string, bool) {
	if f == nil || f.Config.Inventory.Site == nil {
		return siteName, false
	}
	if _, ok := f.Config.Inventory.Site[siteName]; ok {
		return siteName, true
	}
	lower := strings.ToLower(siteName)
	for name := range f.Config.Inventory.Site {
		if strings.ToLower(name) == lower {
			return name, true
		}
	}
	return siteName, false
}

// SaveInventoryFile writes inventory.json with the same durability shape as the
// site-config writer: parent dirs created, 2-space indent, 0600 perms (the file
// names devices an operator authorizes for writes — not world-readable).
func SaveInventoryFile(path string, f *InventoryFile) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("inventory: mkdir %s: %w", dir, err)
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("inventory: marshal: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("inventory: write %s: %w", path, err)
	}
	return nil
}

// ArmSiteDevices merges the given MACs into a site's allowlist at path, creating
// the file if absent and leaving every other site untouched. MACs are stored as
// canonical lowercase bare hex and de-duplicated, so re-running an import is
// idempotent. A non-empty note is stamped on the site section (see
// SiteInventory.Note). deviceType slices map to ap/switch/gateway.
func ArmSiteDevices(path, siteName string, aps, switches, gateways []string, note string) error {
	f, err := LoadInventoryFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		f = &InventoryFile{Version: 1}
		f.Metadata.Description = inventoryDescription
	}
	if f.Config.Inventory.Site == nil {
		f.Config.Inventory.Site = map[string]SiteInventory{}
	}

	key, _ := f.siteKey(siteName)
	si := f.Config.Inventory.Site[key]
	si.AP = mergeMACs(si.AP, aps)
	si.Switch = mergeMACs(si.Switch, switches)
	si.Gateway = mergeMACs(si.Gateway, gateways)
	if note != "" {
		si.Note = note
	}
	f.Config.Inventory.Site[key] = si

	return SaveInventoryFile(path, f)
}

// DisarmSiteDevices removes the given MACs from a site's allowlist at path,
// leaving every other site untouched. MACs are normalized before comparison so
// callers may pass any spelling. A site whose ap/switch/gateway slices are all
// empty after removal is pruned — unless it carries a Note (see
// SiteInventory.Note), which an operator put there deliberately and a prune
// would silently discard. A missing file is a no-op success: nothing is armed,
// so nothing can be disarmed. Returns the count of MACs actually removed so the
// caller can report "already unmanaged".
func DisarmSiteDevices(path, siteName string, aps, switches, gateways []string) (int, error) {
	f, err := LoadInventoryFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}
	if f.Config.Inventory.Site == nil {
		return 0, nil
	}

	key, ok := f.siteKey(siteName)
	if !ok {
		return 0, nil
	}

	si := f.Config.Inventory.Site[key]
	removed := 0
	si.AP, removed = removeMACs(si.AP, aps, removed)
	si.Switch, removed = removeMACs(si.Switch, switches, removed)
	si.Gateway, removed = removeMACs(si.Gateway, gateways, removed)

	if len(si.AP) == 0 && len(si.Switch) == 0 && len(si.Gateway) == 0 && si.Note == "" {
		delete(f.Config.Inventory.Site, key)
	} else {
		f.Config.Inventory.Site[key] = si
	}

	if removed == 0 {
		return 0, nil
	}
	if err := SaveInventoryFile(path, f); err != nil {
		return 0, err
	}
	return removed, nil
}

// removeMACs returns existing minus the normalized incoming MACs, preserving
// order, and the running removal count incremented by however many it dropped.
func removeMACs(existing, remove []string, count int) ([]string, int) {
	if len(existing) == 0 || len(remove) == 0 {
		return existing, count
	}
	drop := make(map[string]bool, len(remove))
	for _, mac := range remove {
		if n := macaddr.NormalizeOrEmpty(mac); n != "" {
			drop[n] = true
		}
	}
	out := make([]string, 0, len(existing))
	for _, mac := range existing {
		if drop[macaddr.NormalizeOrEmpty(mac)] {
			count++
			continue
		}
		out = append(out, mac)
	}
	return out, count
}

// mergeMACs appends incoming MACs to existing ones as normalized bare hex,
// dropping invalid entries and duplicates while preserving first-seen order.
// Existing entries are normalized too, so an arm rewrites a hand-edited file to
// the canonical form.
func mergeMACs(existing, incoming []string) []string {
	seen := make(map[string]bool, len(existing)+len(incoming))
	out := make([]string, 0, len(existing)+len(incoming))
	for _, src := range [][]string{existing, incoming} {
		for _, mac := range src {
			n := macaddr.NormalizeOrEmpty(mac)
			if n == "" || seen[n] {
				continue
			}
			seen[n] = true
			out = append(out, n)
		}
	}
	return out
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

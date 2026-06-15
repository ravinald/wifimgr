package vendors

import (
	"time"
)

// APICacheMeta contains metadata about a single API's cache.
type APICacheMeta struct {
	Vendor            string         `json:"vendor"`
	OrgID             string         `json:"org_id"`
	LastRefresh       time.Time      `json:"last_refresh"` // last successful refresh
	RefreshDurationMs int64          `json:"refresh_duration_ms"`
	ItemCounts        map[string]int `json:"item_counts"`
	// LastFailure/LastError record the most recent failed refresh. LastError is
	// cleared on the next success so the current state reads cleanly, while
	// LastFailure is retained as history. "Currently failing" is derived, not
	// stored: LastFailure.After(LastRefresh).
	LastFailure time.Time `json:"last_failure"`
	LastError   string    `json:"last_error,omitempty"`
}

// APICacheSiteIndex provides site name to ID lookup for a single API.
type APICacheSiteIndex struct {
	ByName map[string]string `json:"by_name"` // site name -> site ID
	ByID   map[string]string `json:"by_id"`   // site ID -> site name

	// Duplicates records site names that resolve to more than one ID within
	// this API. Mist (and UniFi display names) allow same-named sites in one
	// org; a name -> single-ID map can't represent that, so lookups consult
	// this first and refuse rather than binding to an arbitrary site.
	Duplicates map[string][]string `json:"duplicates,omitempty"` // site name -> all colliding IDs
}

// APICache represents a single API's cache file structure.
// Each API (mist-prod, meraki-corp, etc.) has its own cache file.
// stampIfZero sets RefreshedAt to t only if it is still zero, so freshly-fetched
// objects get this pass's time while objects carried forward from a prior cache keep
// their original (older) timestamp. Promoted to every type that embeds ObjectMeta.
func (m *ObjectMeta) stampIfZero(t time.Time) {
	if m.RefreshedAt.IsZero() {
		m.RefreshedAt = t
	}
}

// stampMap stamps RefreshedAt on the pointer-valued cache maps.
func stampMap[T interface{ stampIfZero(time.Time) }](m map[string]T, t time.Time) {
	for _, v := range m {
		v.stampIfZero(t)
	}
}

// StampFreshObjects sets RefreshedAt = t on every cached object whose RefreshedAt is
// still zero — the objects fetched in this refresh pass. Carried-forward objects
// already carry their original timestamp and are left untouched, keeping per-object
// freshness honest across a site-scoped refresh.
func (c *APICache) StampFreshObjects(t time.Time) {
	for i := range c.Sites.Info {
		c.Sites.Info[i].stampIfZero(t)
	}
	for i := range c.Templates.RF {
		c.Templates.RF[i].stampIfZero(t)
	}
	for i := range c.Templates.Gateway {
		c.Templates.Gateway[i].stampIfZero(t)
	}
	for i := range c.Templates.WLAN {
		c.Templates.WLAN[i].stampIfZero(t)
	}
	for i := range c.Profiles.Devices {
		c.Profiles.Devices[i].stampIfZero(t)
	}
	stampMap(c.Inventory.AP, t)
	stampMap(c.Inventory.Switch, t)
	stampMap(c.Inventory.Gateway, t)
	stampMap(c.Configs.AP, t)
	stampMap(c.Configs.Switch, t)
	stampMap(c.Configs.Gateway, t)
	stampMap(c.WLANs, t)
	stampMap(c.BSSIDs, t)
	stampMap(c.DeviceStatus, t)
	stampMap(c.ClientDetail, t)
}

type APICache struct {
	Version   int               `json:"version"`
	APILabel  string            `json:"api_label"`
	Meta      APICacheMeta      `json:"meta"`
	SiteIndex APICacheSiteIndex `json:"site_index"`

	// Sites data
	Sites struct {
		Info     []SiteInfo `json:"info"`
		Settings []any      `json:"settings,omitempty"` // vendor-specific
	} `json:"sites"`

	// Inventory by device type, keyed by normalized MAC
	Inventory struct {
		AP      map[string]*InventoryItem `json:"ap"`
		Switch  map[string]*InventoryItem `json:"switch"`
		Gateway map[string]*InventoryItem `json:"gateway"`
	} `json:"inventory"`

	// Templates - RF profiles/templates
	// Mist: org-level RF templates that can be applied to sites
	// Meraki: per-network RF profiles (stored with site_id)
	Templates struct {
		RF      []RFTemplate      `json:"rf,omitempty"`
		Gateway []GatewayTemplate `json:"gateway,omitempty"`
		WLAN    []WLANTemplate    `json:"wlan,omitempty"`
	} `json:"templates,omitempty"`

	// Profiles (Mist-specific, empty for Meraki)
	Profiles struct {
		Devices []DeviceProfile `json:"devices,omitempty"`
	} `json:"profiles,omitempty"`

	// WLANs - SSID configurations
	// For Mist: Includes both org-level and site-level WLANs
	// For Meraki: Per-network SSIDs (numbered 0-14)
	// Keyed by ID for fast lookup
	WLANs map[string]*WLAN `json:"wlans,omitempty"`

	// Device configs by MAC
	Configs struct {
		AP      map[string]*APConfig      `json:"ap,omitempty"`
		Switch  map[string]*SwitchConfig  `json:"switch,omitempty"`
		Gateway map[string]*GatewayConfig `json:"gateway,omitempty"`
	} `json:"configs,omitempty"`

	// BSSIDs keyed by normalized BSSID MAC
	BSSIDs map[string]*BSSIDEntry `json:"bssids,omitempty"`

	// DeviceStatus by normalized MAC address
	// Stored separately to allow independent refresh from inventory
	DeviceStatus map[string]*DeviceStatus `json:"device_status,omitempty"`

	// ClientDetail holds extra per-client state that the default search
	// endpoint doesn't expose (e.g. Meraki connected band). Populated only
	// when an operator runs `refresh client site <name>`. Keyed by
	// normalized MAC.
	ClientDetail map[string]*ClientDetail `json:"client_detail,omitempty"`
}

// CrossAPIIndex represents the cross-API index file structure.
// This file is rebuilt after any API cache refresh.
type CrossAPIIndex struct {
	Version        int                 `json:"version"`
	LastRebuilt    time.Time           `json:"last_rebuilt"`
	MACToAPI       map[string]string   `json:"mac_to_api"`        // normalized MAC -> API label
	SiteNameToAPIs map[string][]string `json:"site_name_to_apis"` // site name -> []API labels
}

// CacheStatus represents the status of a cache file.
type CacheStatus int

const (
	CacheOK CacheStatus = iota
	CacheStale
	CacheCorrupted
	CacheMissing
)

// String returns a string representation of CacheStatus.
func (s CacheStatus) String() string {
	switch s {
	case CacheOK:
		return "ok"
	case CacheStale:
		return "stale"
	case CacheCorrupted:
		return "corrupted"
	case CacheMissing:
		return "missing"
	default:
		return "unknown"
	}
}

// NewAPICache creates a new empty API cache with initialized maps.
func NewAPICache(apiLabel, vendor, orgID string) *APICache {
	return &APICache{
		Version:  1,
		APILabel: apiLabel,
		Meta: APICacheMeta{
			Vendor:     vendor,
			OrgID:      orgID,
			ItemCounts: make(map[string]int),
		},
		SiteIndex: APICacheSiteIndex{
			ByName: make(map[string]string),
			ByID:   make(map[string]string),
		},
		Inventory: struct {
			AP      map[string]*InventoryItem `json:"ap"`
			Switch  map[string]*InventoryItem `json:"switch"`
			Gateway map[string]*InventoryItem `json:"gateway"`
		}{
			AP:      make(map[string]*InventoryItem),
			Switch:  make(map[string]*InventoryItem),
			Gateway: make(map[string]*InventoryItem),
		},
		Configs: struct {
			AP      map[string]*APConfig      `json:"ap,omitempty"`
			Switch  map[string]*SwitchConfig  `json:"switch,omitempty"`
			Gateway map[string]*GatewayConfig `json:"gateway,omitempty"`
		}{
			AP:      make(map[string]*APConfig),
			Switch:  make(map[string]*SwitchConfig),
			Gateway: make(map[string]*GatewayConfig),
		},
		WLANs:        make(map[string]*WLAN),
		BSSIDs:       make(map[string]*BSSIDEntry),
		DeviceStatus: make(map[string]*DeviceStatus),
	}
}

// NewCrossAPIIndex creates a new empty cross-API index.
func NewCrossAPIIndex() *CrossAPIIndex {
	return &CrossAPIIndex{
		Version:        1,
		LastRebuilt:    time.Now().UTC(),
		MACToAPI:       make(map[string]string),
		SiteNameToAPIs: make(map[string][]string),
	}
}

// UpdateItemCounts updates the item counts in the cache metadata.
func (c *APICache) UpdateItemCounts() {
	c.Meta.ItemCounts["sites"] = len(c.Sites.Info)
	c.Meta.ItemCounts["inventory_ap"] = len(c.Inventory.AP)
	c.Meta.ItemCounts["inventory_switch"] = len(c.Inventory.Switch)
	c.Meta.ItemCounts["inventory_gateway"] = len(c.Inventory.Gateway)
	c.Meta.ItemCounts["templates_rf"] = len(c.Templates.RF)
	c.Meta.ItemCounts["templates_gateway"] = len(c.Templates.Gateway)
	c.Meta.ItemCounts["templates_wlan"] = len(c.Templates.WLAN)
	c.Meta.ItemCounts["profiles"] = len(c.Profiles.Devices)
	c.Meta.ItemCounts["wlans"] = len(c.WLANs)
	c.Meta.ItemCounts["bssids"] = len(c.BSSIDs)
	c.Meta.ItemCounts["device_status"] = len(c.DeviceStatus)
}

// RebuildSiteIndex rebuilds the site index from the sites data. When two
// distinct site IDs share a name, the name is recorded in Duplicates instead
// of letting the later entry silently overwrite the earlier in ByName.
func (c *APICache) RebuildSiteIndex() {
	c.SiteIndex.ByName = make(map[string]string)
	c.SiteIndex.ByID = make(map[string]string)
	c.SiteIndex.Duplicates = nil

	for _, site := range c.Sites.Info {
		if site.Name == "" || site.ID == "" {
			continue
		}
		c.SiteIndex.ByID[site.ID] = site.Name

		existingID, seen := c.SiteIndex.ByName[site.Name]
		if seen && existingID != site.ID {
			if c.SiteIndex.Duplicates == nil {
				c.SiteIndex.Duplicates = make(map[string][]string)
			}
			if len(c.SiteIndex.Duplicates[site.Name]) == 0 {
				c.SiteIndex.Duplicates[site.Name] = []string{existingID}
			}
			c.SiteIndex.Duplicates[site.Name] = append(c.SiteIndex.Duplicates[site.Name], site.ID)
			continue
		}
		c.SiteIndex.ByName[site.Name] = site.ID
	}
}

// BackfillInventorySiteNames populates InventoryItem.SiteName from
// SiteIndex.ByID for every cached AP, switch, and gateway. Several vendor
// adapters (notably Meraki) only carry the site/network ID on the device
// payload — the human-readable name lives on the sites endpoint. Without
// this backfill, downstream consumers (reset ap, show device, search
// results) display raw IDs like "L_3732358191183298569" instead of the
// configured site name.
//
// Safe to call repeatedly. RebuildSiteIndex must run first so SiteIndex.ByID
// is populated; callers that touch sites or inventory should invoke
// RebuildSiteIndex followed by BackfillInventorySiteNames.
func (c *APICache) BackfillInventorySiteNames() {
	backfill := func(items map[string]*InventoryItem) {
		for _, item := range items {
			if item == nil || item.SiteID == "" {
				continue
			}
			if name, ok := c.SiteIndex.ByID[item.SiteID]; ok && name != "" {
				item.SiteName = name
			}
		}
	}
	backfill(c.Inventory.AP)
	backfill(c.Inventory.Switch)
	backfill(c.Inventory.Gateway)
}

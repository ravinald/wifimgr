package vendors

import (
	"time"
)

// APICacheMeta contains metadata about a single API's cache.
type APICacheMeta struct {
	Vendor            string         `json:"vendor"`
	OrgID             string         `json:"org_id"`
	LastRefresh       time.Time      `json:"last_refresh"`
	RefreshDurationMs int64          `json:"refresh_duration_ms"`
	ItemCounts        map[string]int `json:"item_counts"`
}

// APICacheSiteIndex provides site name to ID lookup for a single API.
type APICacheSiteIndex struct {
	ByName map[string]string `json:"by_name"` // site name -> site ID
	ByID   map[string]string `json:"by_id"`   // site ID -> site name
}

// APICache represents a single API's cache file structure.
// Each API (mist-prod, meraki-corp, etc.) has its own cache file.
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

	// DeviceStatus by normalized MAC address
	// Stored separately to allow independent refresh from inventory
	DeviceStatus map[string]*DeviceStatus `json:"device_status,omitempty"`
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
	c.Meta.ItemCounts["device_status"] = len(c.DeviceStatus)
}

// RebuildSiteIndex rebuilds the site index from the sites data.
func (c *APICache) RebuildSiteIndex() {
	c.SiteIndex.ByName = make(map[string]string)
	c.SiteIndex.ByID = make(map[string]string)

	for _, site := range c.Sites.Info {
		if site.Name != "" && site.ID != "" {
			c.SiteIndex.ByName[site.Name] = site.ID
			c.SiteIndex.ByID[site.ID] = site.Name
		}
	}
}

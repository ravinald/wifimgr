package vendors

import (
	"sync"

	"github.com/ravinald/wifimgr/internal/logging"
)

// Global cache accessor for cross-package access
var (
	globalCacheAccessor   *CacheAccessor
	globalCacheAccessorMu sync.RWMutex
)

// SetGlobalCacheAccessor sets the global cache accessor.
// This should be called by initialization code after the cache manager is created.
func SetGlobalCacheAccessor(accessor *CacheAccessor) {
	globalCacheAccessorMu.Lock()
	defer globalCacheAccessorMu.Unlock()
	globalCacheAccessor = accessor
}

// GetGlobalCacheAccessor returns the global cache accessor.
// Returns nil if not initialized.
func GetGlobalCacheAccessor() *CacheAccessor {
	globalCacheAccessorMu.RLock()
	defer globalCacheAccessorMu.RUnlock()
	return globalCacheAccessor
}

// CacheAccessor provides O(1) lookups across all API caches.
// It maintains pre-built indexes that are rebuilt when cache files change.
type CacheAccessor struct {
	manager *CacheManager
	indexes *CacheIndexes
	mu      sync.RWMutex
}

// CacheIndexes holds pre-built indexes for O(1) lookups across all APIs.
type CacheIndexes struct {
	// Sites - aggregated from all APIs
	SitesByID   map[string]*SiteInfo // site_id -> site (includes API label)
	SitesByName map[string]*SiteInfo // site_name -> site

	// Templates - aggregated from all APIs
	RFTemplatesByID     map[string]*RFTemplate
	RFTemplatesByName   map[string]*RFTemplate
	GWTemplatesByID     map[string]*GatewayTemplate
	GWTemplatesByName   map[string]*GatewayTemplate
	WLANTemplatesByID   map[string]*WLANTemplate
	WLANTemplatesByName map[string]*WLANTemplate

	// Device Profiles - aggregated from all APIs
	DeviceProfilesByID   map[string]*DeviceProfile
	DeviceProfilesByName map[string]*DeviceProfile

	// Inventory - aggregated from all APIs (keyed by normalized MAC)
	DevicesByMAC  map[string]*InventoryItem
	DevicesByName map[string]*InventoryItem

	// Configs - aggregated from all APIs (keyed by normalized MAC)
	APConfigsByMAC      map[string]*APConfig
	SwitchConfigsByMAC  map[string]*SwitchConfig
	GatewayConfigsByMAC map[string]*GatewayConfig

	// Device Status - aggregated from all APIs (keyed by normalized MAC)
	DeviceStatusByMAC map[string]*DeviceStatus

	// WLANs - aggregated from all APIs
	WLANsByID   map[string]*WLAN
	WLANsBySSID map[string][]*WLAN // SSID name -> WLANs (multiple WLANs can have same SSID)
}

// NewCacheAccessor creates a new cache accessor with pre-built indexes.
func NewCacheAccessor(manager *CacheManager) *CacheAccessor {
	ca := &CacheAccessor{
		manager: manager,
		indexes: newCacheIndexes(),
	}
	ca.RebuildIndexes()
	return ca
}

// newCacheIndexes creates empty cache indexes.
func newCacheIndexes() *CacheIndexes {
	return &CacheIndexes{
		SitesByID:            make(map[string]*SiteInfo),
		SitesByName:          make(map[string]*SiteInfo),
		RFTemplatesByID:      make(map[string]*RFTemplate),
		RFTemplatesByName:    make(map[string]*RFTemplate),
		GWTemplatesByID:      make(map[string]*GatewayTemplate),
		GWTemplatesByName:    make(map[string]*GatewayTemplate),
		WLANTemplatesByID:    make(map[string]*WLANTemplate),
		WLANTemplatesByName:  make(map[string]*WLANTemplate),
		DeviceProfilesByID:   make(map[string]*DeviceProfile),
		DeviceProfilesByName: make(map[string]*DeviceProfile),
		DevicesByMAC:         make(map[string]*InventoryItem),
		DevicesByName:        make(map[string]*InventoryItem),
		APConfigsByMAC:       make(map[string]*APConfig),
		SwitchConfigsByMAC:   make(map[string]*SwitchConfig),
		GatewayConfigsByMAC:  make(map[string]*GatewayConfig),
		DeviceStatusByMAC:    make(map[string]*DeviceStatus),
		WLANsByID:            make(map[string]*WLAN),
		WLANsBySSID:          make(map[string][]*WLAN),
	}
}

// RebuildIndexes rebuilds all indexes from the cache files.
func (ca *CacheAccessor) RebuildIndexes() {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	ca.indexes = newCacheIndexes()

	if ca.manager == nil || ca.manager.registry == nil {
		return
	}

	// Get all API labels from the registry
	labels := ca.manager.registry.GetAllLabels()

	for _, apiLabel := range labels {
		cache, err := ca.manager.GetAPICache(apiLabel)
		if err != nil {
			logging.Debugf("[cache-accessor] Failed to load cache for %s: %v", apiLabel, err)
			continue
		}

		ca.indexAPICache(cache, apiLabel)
	}

	logging.Debugf("[cache-accessor] Indexed %d sites, %d RF templates, %d profiles, %d devices",
		len(ca.indexes.SitesByID),
		len(ca.indexes.RFTemplatesByID),
		len(ca.indexes.DeviceProfilesByID),
		len(ca.indexes.DevicesByMAC))
}

// indexAPICache indexes a single API's cache into the aggregate indexes.
func (ca *CacheAccessor) indexAPICache(cache *APICache, apiLabel string) {
	vendor := cache.Meta.Vendor

	// Index sites
	for i := range cache.Sites.Info {
		site := &cache.Sites.Info[i]
		site.SourceAPI = apiLabel
		site.SourceVendor = vendor

		if site.ID != "" {
			ca.indexes.SitesByID[site.ID] = site
		}
		if site.Name != "" {
			ca.indexes.SitesByName[site.Name] = site
		}
	}

	// Index RF templates
	for i := range cache.Templates.RF {
		tmpl := &cache.Templates.RF[i]
		tmpl.SourceAPI = apiLabel
		tmpl.SourceVendor = vendor

		if tmpl.ID != "" {
			ca.indexes.RFTemplatesByID[tmpl.ID] = tmpl
		}
		if tmpl.Name != "" {
			ca.indexes.RFTemplatesByName[tmpl.Name] = tmpl
		}
	}

	// Index Gateway templates
	for i := range cache.Templates.Gateway {
		tmpl := &cache.Templates.Gateway[i]
		tmpl.SourceAPI = apiLabel
		tmpl.SourceVendor = vendor

		if tmpl.ID != "" {
			ca.indexes.GWTemplatesByID[tmpl.ID] = tmpl
		}
		if tmpl.Name != "" {
			ca.indexes.GWTemplatesByName[tmpl.Name] = tmpl
		}
	}

	// Index WLAN templates
	for i := range cache.Templates.WLAN {
		tmpl := &cache.Templates.WLAN[i]
		tmpl.SourceAPI = apiLabel
		tmpl.SourceVendor = vendor

		if tmpl.ID != "" {
			ca.indexes.WLANTemplatesByID[tmpl.ID] = tmpl
		}
		if tmpl.Name != "" {
			ca.indexes.WLANTemplatesByName[tmpl.Name] = tmpl
		}
	}

	// Index device profiles
	for i := range cache.Profiles.Devices {
		profile := &cache.Profiles.Devices[i]
		profile.SourceAPI = apiLabel
		profile.SourceVendor = vendor

		if profile.ID != "" {
			ca.indexes.DeviceProfilesByID[profile.ID] = profile
		}
		if profile.Name != "" {
			ca.indexes.DeviceProfilesByName[profile.Name] = profile
		}
	}

	// Index inventory (APs)
	for mac, item := range cache.Inventory.AP {
		item.SourceAPI = apiLabel
		item.SourceVendor = vendor
		normalizedMAC := NormalizeMAC(mac)
		ca.indexes.DevicesByMAC[normalizedMAC] = item
		if item.Name != "" {
			ca.indexes.DevicesByName[item.Name] = item
		}
	}

	// Index inventory (Switches)
	for mac, item := range cache.Inventory.Switch {
		item.SourceAPI = apiLabel
		item.SourceVendor = vendor
		normalizedMAC := NormalizeMAC(mac)
		ca.indexes.DevicesByMAC[normalizedMAC] = item
		if item.Name != "" {
			ca.indexes.DevicesByName[item.Name] = item
		}
	}

	// Index inventory (Gateways)
	for mac, item := range cache.Inventory.Gateway {
		item.SourceAPI = apiLabel
		item.SourceVendor = vendor
		normalizedMAC := NormalizeMAC(mac)
		ca.indexes.DevicesByMAC[normalizedMAC] = item
		if item.Name != "" {
			ca.indexes.DevicesByName[item.Name] = item
		}
	}

	// Index configs
	for mac, cfg := range cache.Configs.AP {
		cfg.SourceAPI = apiLabel
		cfg.SourceVendor = vendor
		normalizedMAC := NormalizeMAC(mac)
		ca.indexes.APConfigsByMAC[normalizedMAC] = cfg
	}
	for mac, cfg := range cache.Configs.Switch {
		cfg.SourceAPI = apiLabel
		cfg.SourceVendor = vendor
		normalizedMAC := NormalizeMAC(mac)
		ca.indexes.SwitchConfigsByMAC[normalizedMAC] = cfg
	}
	for mac, cfg := range cache.Configs.Gateway {
		cfg.SourceAPI = apiLabel
		cfg.SourceVendor = vendor
		normalizedMAC := NormalizeMAC(mac)
		ca.indexes.GatewayConfigsByMAC[normalizedMAC] = cfg
	}

	// Index device status
	for mac, status := range cache.DeviceStatus {
		normalizedMAC := NormalizeMAC(mac)
		ca.indexes.DeviceStatusByMAC[normalizedMAC] = status
	}

	// Index WLANs
	for id, wlan := range cache.WLANs {
		wlan.SourceAPI = apiLabel
		wlan.SourceVendor = vendor
		ca.indexes.WLANsByID[id] = wlan
		if wlan.SSID != "" {
			ca.indexes.WLANsBySSID[wlan.SSID] = append(ca.indexes.WLANsBySSID[wlan.SSID], wlan)
		}
	}
}

// GetManager returns the underlying cache manager.
func (ca *CacheAccessor) GetManager() *CacheManager {
	return ca.manager
}

// IsInitialized returns true if the accessor has been initialized.
func (ca *CacheAccessor) IsInitialized() bool {
	ca.mu.RLock()
	defer ca.mu.RUnlock()
	return ca.indexes != nil
}

// GetStats returns statistics about the cached data.
func (ca *CacheAccessor) GetStats() map[string]int {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	return map[string]int{
		"sites":           len(ca.indexes.SitesByID),
		"rf_templates":    len(ca.indexes.RFTemplatesByID),
		"gw_templates":    len(ca.indexes.GWTemplatesByID),
		"wlan_templates":  len(ca.indexes.WLANTemplatesByID),
		"device_profiles": len(ca.indexes.DeviceProfilesByID),
		"devices":         len(ca.indexes.DevicesByMAC),
		"ap_configs":      len(ca.indexes.APConfigsByMAC),
		"switch_configs":  len(ca.indexes.SwitchConfigsByMAC),
		"gateway_configs": len(ca.indexes.GatewayConfigsByMAC),
		"device_status":   len(ca.indexes.DeviceStatusByMAC),
		"wlans":           len(ca.indexes.WLANsByID),
	}
}

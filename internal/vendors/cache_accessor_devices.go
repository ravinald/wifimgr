package vendors

import "fmt"

// Config lookups

// GetAPConfigByMAC returns an AP config by its MAC address.
func (ca *CacheAccessor) GetAPConfigByMAC(mac string) (*APConfig, error) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	normalizedMAC := NormalizeMAC(mac)
	cfg, ok := ca.indexes.APConfigsByMAC[normalizedMAC]
	if !ok {
		return nil, fmt.Errorf("AP config not found: %s", mac)
	}
	return cfg, nil
}

// GetSwitchConfigByMAC returns a switch config by its MAC address.
func (ca *CacheAccessor) GetSwitchConfigByMAC(mac string) (*SwitchConfig, error) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	normalizedMAC := NormalizeMAC(mac)
	cfg, ok := ca.indexes.SwitchConfigsByMAC[normalizedMAC]
	if !ok {
		return nil, fmt.Errorf("switch config not found: %s", mac)
	}
	return cfg, nil
}

// GetGatewayConfigByMAC returns a gateway config by its MAC address.
func (ca *CacheAccessor) GetGatewayConfigByMAC(mac string) (*GatewayConfig, error) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	normalizedMAC := NormalizeMAC(mac)
	cfg, ok := ca.indexes.GatewayConfigsByMAC[normalizedMAC]
	if !ok {
		return nil, fmt.Errorf("gateway config not found: %s", mac)
	}
	return cfg, nil
}

// GetAllAPConfigs returns all AP configs from all APIs.
func (ca *CacheAccessor) GetAllAPConfigs() []*APConfig {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	configs := make([]*APConfig, 0, len(ca.indexes.APConfigsByMAC))
	for _, cfg := range ca.indexes.APConfigsByMAC {
		configs = append(configs, cfg)
	}
	return configs
}

// GetAllSwitchConfigs returns all switch configs from all APIs.
func (ca *CacheAccessor) GetAllSwitchConfigs() []*SwitchConfig {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	configs := make([]*SwitchConfig, 0, len(ca.indexes.SwitchConfigsByMAC))
	for _, cfg := range ca.indexes.SwitchConfigsByMAC {
		configs = append(configs, cfg)
	}
	return configs
}

// GetAllGatewayConfigs returns all gateway configs from all APIs.
func (ca *CacheAccessor) GetAllGatewayConfigs() []*GatewayConfig {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	configs := make([]*GatewayConfig, 0, len(ca.indexes.GatewayConfigsByMAC))
	for _, cfg := range ca.indexes.GatewayConfigsByMAC {
		configs = append(configs, cfg)
	}
	return configs
}

// Status lookups

// GetDeviceStatus returns device status by MAC address.
func (ca *CacheAccessor) GetDeviceStatus(mac string) (*DeviceStatus, error) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	normalizedMAC := NormalizeMAC(mac)
	status, ok := ca.indexes.DeviceStatusByMAC[normalizedMAC]
	if !ok {
		return nil, fmt.Errorf("device status not found: %s", mac)
	}
	return status, nil
}

// WLAN lookups

// GetWLANByID returns a WLAN by its ID.
func (ca *CacheAccessor) GetWLANByID(id string) (*WLAN, error) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	wlan, ok := ca.indexes.WLANsByID[id]
	if !ok {
		return nil, fmt.Errorf("WLAN not found: %s", id)
	}
	return wlan, nil
}

// GetWLANsBySSID returns all WLANs with the given SSID name.
// Multiple WLANs can share the same SSID (different sites/networks).
func (ca *CacheAccessor) GetWLANsBySSID(ssid string) []*WLAN {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	wlans, ok := ca.indexes.WLANsBySSID[ssid]
	if !ok {
		return nil
	}
	// Return a copy to prevent external modification
	result := make([]*WLAN, len(wlans))
	copy(result, wlans)
	return result
}

// GetAllWLANs returns all WLANs from all APIs.
func (ca *CacheAccessor) GetAllWLANs() []*WLAN {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	wlans := make([]*WLAN, 0, len(ca.indexes.WLANsByID))
	for _, wlan := range ca.indexes.WLANsByID {
		wlans = append(wlans, wlan)
	}
	return wlans
}

// GetWLANsBySite returns all WLANs for a specific site/network.
func (ca *CacheAccessor) GetWLANsBySite(siteID string) []*WLAN {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	var result []*WLAN
	for _, wlan := range ca.indexes.WLANsByID {
		if wlan.SiteID == siteID {
			result = append(result, wlan)
		}
	}
	return result
}

// GetWLANsByVendor returns all WLANs from a specific vendor.
func (ca *CacheAccessor) GetWLANsByVendor(vendor string) []*WLAN {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	var result []*WLAN
	for _, wlan := range ca.indexes.WLANsByID {
		if wlan.SourceVendor == vendor {
			result = append(result, wlan)
		}
	}
	return result
}

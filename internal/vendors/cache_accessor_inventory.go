package vendors

import "fmt"

// Device lookups

// GetDeviceByMAC returns an inventory item by its MAC address.
func (ca *CacheAccessor) GetDeviceByMAC(mac string) (*InventoryItem, error) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	normalizedMAC := NormalizeMAC(mac)
	item, ok := ca.indexes.DevicesByMAC[normalizedMAC]
	if !ok {
		return nil, fmt.Errorf("device not found: %s", mac)
	}
	return item, nil
}

// GetDeviceByName returns an inventory item by its name.
func (ca *CacheAccessor) GetDeviceByName(name string) (*InventoryItem, error) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	item, ok := ca.indexes.DevicesByName[name]
	if !ok {
		return nil, fmt.Errorf("device not found: %s", name)
	}
	return item, nil
}

// GetAllDevices returns all inventory items from all APIs.
func (ca *CacheAccessor) GetAllDevices() []*InventoryItem {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	devices := make([]*InventoryItem, 0, len(ca.indexes.DevicesByMAC))
	for _, item := range ca.indexes.DevicesByMAC {
		devices = append(devices, item)
	}
	return devices
}

// Type-specific device lookups

// GetAPByMAC returns an AP inventory item by its MAC address.
func (ca *CacheAccessor) GetAPByMAC(mac string) (*InventoryItem, error) {
	device, err := ca.GetDeviceByMAC(mac)
	if err != nil {
		return nil, err
	}
	if device.Type != "ap" {
		return nil, fmt.Errorf("device %s is not an AP (type: %s)", mac, device.Type)
	}
	return device, nil
}

// GetSwitchByMAC returns a switch inventory item by its MAC address.
func (ca *CacheAccessor) GetSwitchByMAC(mac string) (*InventoryItem, error) {
	device, err := ca.GetDeviceByMAC(mac)
	if err != nil {
		return nil, err
	}
	if device.Type != "switch" {
		return nil, fmt.Errorf("device %s is not a switch (type: %s)", mac, device.Type)
	}
	return device, nil
}

// GetGatewayByMAC returns a gateway inventory item by its MAC address.
func (ca *CacheAccessor) GetGatewayByMAC(mac string) (*InventoryItem, error) {
	device, err := ca.GetDeviceByMAC(mac)
	if err != nil {
		return nil, err
	}
	if device.Type != "gateway" {
		return nil, fmt.Errorf("device %s is not a gateway (type: %s)", mac, device.Type)
	}
	return device, nil
}

// GetAllAPs returns all AP inventory items from all APIs.
func (ca *CacheAccessor) GetAllAPs() []*InventoryItem {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	var aps []*InventoryItem
	for _, item := range ca.indexes.DevicesByMAC {
		if item.Type == "ap" {
			aps = append(aps, item)
		}
	}
	return aps
}

// GetAllSwitches returns all switch inventory items from all APIs.
func (ca *CacheAccessor) GetAllSwitches() []*InventoryItem {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	var switches []*InventoryItem
	for _, item := range ca.indexes.DevicesByMAC {
		if item.Type == "switch" {
			switches = append(switches, item)
		}
	}
	return switches
}

// GetAllGateways returns all gateway inventory items from all APIs.
func (ca *CacheAccessor) GetAllGateways() []*InventoryItem {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	var gateways []*InventoryItem
	for _, item := range ca.indexes.DevicesByMAC {
		if item.Type == "gateway" {
			gateways = append(gateways, item)
		}
	}
	return gateways
}

// GetDevicesBySite returns all inventory items for a specific site and device type.
// deviceType can be "ap", "switch", "gateway", or empty for all types.
func (ca *CacheAccessor) GetDevicesBySite(siteID string, deviceType string) []*InventoryItem {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	var devices []*InventoryItem
	for _, item := range ca.indexes.DevicesByMAC {
		if item.SiteID != siteID {
			continue
		}
		if deviceType != "" && item.Type != deviceType {
			continue
		}
		devices = append(devices, item)
	}
	return devices
}

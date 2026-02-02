package apply

import (
	"context"
	"fmt"
	"sync"

	"github.com/ravinald/wifimgr/api"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/macaddr"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// DeviceBatchLoader provides efficient batch device lookups for apply operations
type DeviceBatchLoader struct {
	devices map[string]*api.UnifiedDevice
	mu      sync.RWMutex
}

// NewDeviceBatchLoader creates a new batch loader for a specific site and device type.
// Uses multi-vendor cache for device data.
func NewDeviceBatchLoader(_ context.Context, _ api.Client, siteID string, deviceType string) (*DeviceBatchLoader, error) {
	loader := &DeviceBatchLoader{
		devices: make(map[string]*api.UnifiedDevice),
	}

	accessor := vendors.GetGlobalCacheAccessor()
	if accessor == nil {
		return nil, fmt.Errorf("cache accessor not initialized")
	}

	inventoryItems := accessor.GetDevicesBySite(siteID, deviceType)

	for _, item := range inventoryItems {
		normalizedMAC := macaddr.NormalizeOrEmpty(item.MAC)
		if normalizedMAC == "" {
			continue
		}

		// Get config from cache for this device
		var configMap map[string]any
		switch deviceType {
		case "ap":
			if cfg, err := accessor.GetAPConfigByMAC(normalizedMAC); err == nil && cfg != nil {
				configMap = cfg.Config
			}
		case "switch":
			if cfg, err := accessor.GetSwitchConfigByMAC(normalizedMAC); err == nil && cfg != nil {
				configMap = cfg.Config
			}
		case "gateway":
			if cfg, err := accessor.GetGatewayConfigByMAC(normalizedMAC); err == nil && cfg != nil {
				configMap = cfg.Config
			}
		}

		// Create UnifiedDevice from cache data
		mac := item.MAC
		name := item.Name
		itemSiteID := item.SiteID
		id := item.ID

		device := &api.UnifiedDevice{
			BaseDevice: api.BaseDevice{
				MAC:    &mac,
				Name:   &name,
				SiteID: &itemSiteID,
				ID:     &id,
			},
		}

		// Set DeviceConfig directly for ToConfigMap() compatibility
		if configMap != nil {
			device.DeviceConfig = configMap
		}

		loader.devices[normalizedMAC] = device
	}

	logging.Debugf("Loaded %d %s devices from cache for site %s", len(loader.devices), deviceType, siteID)
	return loader, nil
}

// GetDeviceByMAC retrieves a device by MAC with O(1) lookup
func (d *DeviceBatchLoader) GetDeviceByMAC(mac string) (*api.UnifiedDevice, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	normalizedMAC, err := macaddr.Normalize(mac)
	if err != nil {
		return nil, fmt.Errorf("invalid MAC: %w", err)
	}

	device, exists := d.devices[normalizedMAC]
	if !exists {
		return nil, fmt.Errorf("device not found: %s", mac)
	}
	return device, nil
}

// GetAllDevices returns all devices in the batch
func (d *DeviceBatchLoader) GetAllDevices() []*api.UnifiedDevice {
	d.mu.RLock()
	defer d.mu.RUnlock()

	devices := make([]*api.UnifiedDevice, 0, len(d.devices))
	for _, device := range d.devices {
		devices = append(devices, device)
	}
	return devices
}

// GetDeviceCount returns the number of devices in the batch loader
func (d *DeviceBatchLoader) GetDeviceCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.devices)
}

// HasDevice checks if a device exists in the batch loader
func (d *DeviceBatchLoader) HasDevice(mac string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	normalizedMAC, err := macaddr.Normalize(mac)
	if err != nil {
		return false
	}

	_, exists := d.devices[normalizedMAC]
	return exists
}

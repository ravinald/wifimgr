package apply

import (
	"context"
	"fmt"

	"github.com/ravinald/wifimgr/api"
	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/keypath"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/macaddr"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// Note: This file contains legacy apply code that uses api.Client.
// Where possible, methods have been updated to use the multi-vendor
// cache accessor instead of making direct API calls.

// DeviceUpdater defines the interface for device update operations
type DeviceUpdater interface {
	// GetDeviceType returns the device type (ap, switch, gateway)
	GetDeviceType() string

	// GetConfiguredDevices extracts device MAC addresses from site configuration
	GetConfiguredDevices(siteConfig SiteConfig) []string

	// GetAssignedDevices gets MAC addresses of devices assigned to a site from cache
	GetAssignedDevices(ctx context.Context, client api.Client, siteID string) ([]string, error)

	// FindDevicesInventoryStatus checks device status in inventory and cache
	FindDevicesInventoryStatus(client api.Client, cfg *config.Config, configuredDevices []string) ([]DeviceInventoryStatus, error)

	// UnassignDevices removes devices from a site
	UnassignDevices(ctx context.Context, client api.Client, cfg *config.Config, macs []string) error

	// AssignDevices assigns devices to a site
	AssignDevices(ctx context.Context, client api.Client, cfg *config.Config, macs []string, siteID string) error

	// FindDevicesToAssign identifies devices that need to be assigned to the site
	FindDevicesToAssign(client api.Client, cfg *config.Config, configuredDevices []string, siteID string) ([]string, error)

	// FindDevicesToUpdate identifies devices that need configuration updates
	FindDevicesToUpdate(ctx context.Context, client api.Client, cfg *config.Config, siteConfig SiteConfig, configuredDevices []string, siteID string, apiLabel string) ([]string, error)

	// UpdateDeviceConfigurations applies configuration updates to devices
	UpdateDeviceConfigurations(ctx context.Context, client api.Client, cfg *config.Config, siteConfig SiteConfig, macs []string, siteID string, apiLabel string) error

	// GetDeviceConfigFromSite extracts device-specific config from site configuration
	GetDeviceConfigFromSite(siteConfig SiteConfig, mac string) (map[string]any, bool)

	// ExportCurrentDeviceConfiguration exports current device config for backup
	ExportCurrentDeviceConfiguration(ctx context.Context, client api.Client, mac string, siteID string) (map[string]any, error)

	// SetInventoryChecker sets the inventory checker for reuse across operations
	SetInventoryChecker(checker *InventoryChecker)

	// GetInventoryChecker returns the stored inventory checker
	GetInventoryChecker() *InventoryChecker
}

// DeviceInventoryStatus represents the status of a device in relation to inventory and cache
type DeviceInventoryStatus struct {
	MAC             string
	InCache         bool
	InInventory     bool
	CurrentSiteName string
	CurrentSiteID   string
}

// BaseDeviceUpdater provides common functionality for all device types
type BaseDeviceUpdater struct {
	deviceType       string
	inventoryChecker *InventoryChecker // Shared inventory checker for reuse across operations
}

// NewBaseDeviceUpdater creates a new base device updater
func NewBaseDeviceUpdater(deviceType string) *BaseDeviceUpdater {
	return &BaseDeviceUpdater{
		deviceType: deviceType,
	}
}

// GetDeviceType returns the device type
func (b *BaseDeviceUpdater) GetDeviceType() string {
	return b.deviceType
}

// SetInventoryChecker sets the inventory checker for reuse across operations
func (b *BaseDeviceUpdater) SetInventoryChecker(checker *InventoryChecker) {
	b.inventoryChecker = checker
}

// GetInventoryChecker returns the stored inventory checker
func (b *BaseDeviceUpdater) GetInventoryChecker() *InventoryChecker {
	return b.inventoryChecker
}

// GetAssignedDevices gets MAC addresses of devices assigned to a site from cache
func (b *BaseDeviceUpdater) GetAssignedDevices(_ context.Context, _ api.Client, siteID string) ([]string, error) {
	accessor := vendors.GetGlobalCacheAccessor()
	if accessor == nil {
		return nil, fmt.Errorf("cache accessor not initialized")
	}

	items := accessor.GetDevicesBySite(siteID, b.deviceType)
	assignedDevices := make([]string, 0, len(items))
	for _, item := range items {
		normalizedMAC := macaddr.NormalizeOrEmpty(item.MAC)
		if normalizedMAC != "" {
			assignedDevices = append(assignedDevices, normalizedMAC)
		}
	}
	logging.Debugf("Found %d %s devices assigned to site %s from cache", len(assignedDevices), b.deviceType, siteID)
	return assignedDevices, nil
}

// UnassignDevices removes devices from a site
func (b *BaseDeviceUpdater) UnassignDevices(ctx context.Context, client api.Client, cfg *config.Config, macs []string) error {
	return client.UnassignDevicesFromSite(ctx, cfg.API.Credentials.OrgID, macs)
}

// AssignDevices assigns devices to a site
func (b *BaseDeviceUpdater) AssignDevices(ctx context.Context, client api.Client, cfg *config.Config, macs []string, siteID string) error {
	return client.AssignDevicesToSite(ctx, cfg.API.Credentials.OrgID, siteID, macs, true)
}

// FindDevicesToAssign identifies devices that need to be assigned to the site.
// NOTE: This only returns devices that are in the API inventory to ensure we don't
// try to assign devices that aren't managed by this system.
func (b *BaseDeviceUpdater) FindDevicesToAssign(_ api.Client, _ *config.Config, configuredDevices []string, siteID string) ([]string, error) {
	accessor := vendors.GetGlobalCacheAccessor()
	if accessor == nil {
		return nil, fmt.Errorf("cache accessor not initialized")
	}

	devicesToAssign := make([]string, 0)

	configuredDeviceMap := make(map[string]bool)
	for _, mac := range configuredDevices {
		configuredDeviceMap[mac] = true
	}

	// Get all devices of this type from cache
	var inventoryItems []*vendors.InventoryItem
	switch b.deviceType {
	case "ap":
		inventoryItems = accessor.GetAllAPs()
	case "switch":
		inventoryItems = accessor.GetAllSwitches()
	case "gateway":
		inventoryItems = accessor.GetAllGateways()
	default:
		inventoryItems = accessor.GetAllDevices()
	}

	// Build inventory MAC map for O(1) lookups
	inventoryMACs := make(map[string]bool)
	for _, item := range inventoryItems {
		normalizedMAC := macaddr.NormalizeOrEmpty(item.MAC)
		if normalizedMAC != "" {
			inventoryMACs[normalizedMAC] = true
		}
	}

	// Find devices that need assignment
	for _, item := range inventoryItems {
		normalizedMAC := macaddr.NormalizeOrEmpty(item.MAC)
		if normalizedMAC != "" && configuredDeviceMap[normalizedMAC] {
			isAssignedToWrongSite := item.SiteID != "" && item.SiteID != siteID
			isNotAssigned := item.SiteID == ""

			if isAssignedToWrongSite || isNotAssigned {
				devicesToAssign = append(devicesToAssign, normalizedMAC)
				logging.Debugf("Device %s needs assignment (currently assigned to: %s)",
					normalizedMAC, item.SiteID)
			}
		}
	}

	// Log devices in config but not in inventory
	for mac := range configuredDeviceMap {
		if !inventoryMACs[mac] {
			logging.Warnf("Device %s is in configuration but not in API inventory - cannot assign", mac)
		}
	}

	logging.Debugf("Found %d %s devices to assign from cache", len(devicesToAssign), b.deviceType)
	return devicesToAssign, nil
}

// ExportCurrentDeviceConfiguration exports current device config for backup
func (b *BaseDeviceUpdater) ExportCurrentDeviceConfiguration(_ context.Context, _ api.Client, mac string, _ string) (map[string]any, error) {
	logging.Debugf("Exporting current configuration for %s %s", b.deviceType, mac)

	accessor := vendors.GetGlobalCacheAccessor()
	if accessor == nil {
		return nil, fmt.Errorf("cache accessor not initialized")
	}

	var currentConfig map[string]any
	var err error

	switch b.deviceType {
	case "ap":
		if cfg, e := accessor.GetAPConfigByMAC(mac); e == nil && cfg != nil {
			currentConfig = cfg.Config
		} else {
			err = e
		}
	case "switch":
		if cfg, e := accessor.GetSwitchConfigByMAC(mac); e == nil && cfg != nil {
			currentConfig = cfg.Config
		} else {
			err = e
		}
	case "gateway":
		if cfg, e := accessor.GetGatewayConfigByMAC(mac); e == nil && cfg != nil {
			currentConfig = cfg.Config
		} else {
			err = e
		}
	default:
		err = fmt.Errorf("unknown device type: %s", b.deviceType)
	}

	if err != nil {
		return nil, fmt.Errorf("cannot export configuration for %s %s from cache: %w", b.deviceType, mac, err)
	}

	logging.Debugf("Exported %d configuration fields for %s %s from cache", len(currentConfig), b.deviceType, mac)
	return currentConfig, nil
}

// compareDeviceConfigsWithManagedKeys compares two device configurations considering only managed keys.
// Supports dot-notation paths (e.g., "radio_config.band_24.power") and wildcards (e.g., "port_config.*.vlan_id").
//
// If managedKeys is nil or empty, all fields are compared (backward compatibility).
// If a managed key specifies a parent path, all its children are also managed recursively.
func compareDeviceConfigsWithManagedKeys(current, desired map[string]any, managedKeys []string) bool {
	// Handle _name suffix translations
	translatedFields := map[string]string{
		"deviceprofile_name": "deviceprofile_id",
		"site_name":          "site_id",
	}

	// Helper function to check if a key path is managed
	// Uses keypath.IsKeyManaged which supports:
	// - Direct match (e.g., "name" matches "name")
	// - Parent path match (e.g., "radio_config.band_24" manages "radio_config.band_24.power")
	// - Wildcard match (e.g., "port_config.*.vlan_id" matches "port_config.eth0.vlan_id")
	isManaged := func(path string) bool {
		// If no managed keys specified, all keys are managed
		if len(managedKeys) == 0 {
			return true
		}
		return keypath.IsKeyManaged(path, managedKeys)
	}

	// First check if all desired fields match current
	for key, desiredValue := range desired {
		// Skip internal fields that should be ignored
		if _, isTranslated := translatedFields[key]; isTranslated {
			continue
		}

		// If managed keys are defined, skip fields not in the managed list
		if !isManaged(key) {
			continue
		}

		currentValue, exists := current[key]
		if !exists {
			// Field is missing in current config
			return true
		}

		// For nested maps, do deep comparison if the key is managed
		if desiredMap, ok := desiredValue.(map[string]any); ok {
			if currentMap, ok := currentValue.(map[string]any); ok {
				// Recursively compare nested maps
				if compareNestedMaps(currentMap, desiredMap) {
					return true
				}
			} else {
				// Type mismatch
				return true
			}
		} else {
			// Compare values (simplified comparison for non-map types)
			if fmt.Sprintf("%v", currentValue) != fmt.Sprintf("%v", desiredValue) {
				return true
			}
		}
	}

	// Check if current has fields that aren't in desired (fields to remove)
	for key := range current {
		// Skip status fields that shouldn't be compared
		statusFields := map[string]bool{
			"id": true, "created_time": true, "modified_time": true,
			"connected": true, "adopted": true, "hostname": true, "jsi": true,
			"last_seen": true, "ip": true, "status": true, "version": true,
			"mac": true, "serial": true, "model": true, "type": true,
		}

		if statusFields[key] {
			continue
		}

		// If managed keys are defined, skip fields not in the managed list
		if !isManaged(key) {
			continue
		}

		if _, exists := desired[key]; !exists {
			// Field exists in current but not in desired
			return true
		}
	}

	return false
}

// compareNestedMaps recursively compares nested map structures
func compareNestedMaps(current, desired map[string]any) bool {
	// Check all desired fields exist and match in current
	for key, desiredValue := range desired {
		currentValue, exists := current[key]
		if !exists {
			return true
		}

		// Recursively handle nested maps
		if desiredMap, ok := desiredValue.(map[string]any); ok {
			if currentMap, ok := currentValue.(map[string]any); ok {
				if compareNestedMaps(currentMap, desiredMap) {
					return true
				}
			} else {
				return true
			}
		} else {
			// Compare non-map values
			if fmt.Sprintf("%v", currentValue) != fmt.Sprintf("%v", desiredValue) {
				return true
			}
		}
	}

	// Check if current has fields not in desired
	for key := range current {
		if _, exists := desired[key]; !exists {
			return true
		}
	}

	return false
}

// filterConfigByManagedKeys filters a configuration map to only include managed keys.
// Supports dot-notation paths (e.g., "radio_config.band_24.power") and wildcards
// (e.g., "port_config.*.vlan_id").
//
// If managedKeys is nil or empty, returns the original config (backward compatibility).
// If a managed key refers to a nested map, the entire nested structure is included.
func filterConfigByManagedKeys(config map[string]any, managedKeys []string) map[string]any {
	// If no managed keys specified, return original config
	if len(managedKeys) == 0 {
		return config
	}

	// Use keypath package for path-aware filtering
	// This handles dot-notation (e.g., "radio_config.band_24.power")
	// and wildcards (e.g., "port_config.*.vlan_id")
	result := keypath.FilterMapByManagedKeys(config, managedKeys)

	// If keypath returns nil (empty result), return empty map instead
	if result == nil {
		return make(map[string]any)
	}

	return result
}

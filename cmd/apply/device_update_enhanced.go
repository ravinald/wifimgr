package apply

import (
	"context"

	"github.com/spf13/viper"

	"github.com/ravinald/wifimgr/api"
	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/macaddr"
)

// findDevicesToUnassignWithInventoryCheck finds devices that need to be unassigned from the site
// but only if they exist in inventory (to avoid touching devices not managed by this system).
// If inventoryChecker is provided, it is used for O(1) lookups instead of making API calls.
func findDevicesToUnassignWithInventoryCheck(ctx context.Context, client api.Client, cfg *config.Config,
	assignedDevices, configuredDevices []string, deviceType string, inventoryChecker *InventoryChecker) ([]string, error) {

	devicesToUnassign := make([]string, 0)

	// Create a map of configured devices for quick lookup
	configuredDeviceMap := make(map[string]bool)
	for _, mac := range configuredDevices {
		configuredDeviceMap[mac] = true
	}

	// If we have an inventory checker, use it for O(1) lookups (no API calls)
	if inventoryChecker != nil {
		for _, mac := range assignedDevices {
			if !configuredDeviceMap[mac] && inventoryChecker.IsInInventory(mac) {
				devicesToUnassign = append(devicesToUnassign, mac)
				logging.Debugf("Device %s will be unassigned (in inventory but not in config)", mac)
			} else if !configuredDeviceMap[mac] && !inventoryChecker.IsInInventory(mac) {
				logging.Infof("Device %s is assigned but not in config or inventory - skipping unassign", mac)
			}
		}
		return devicesToUnassign, nil
	}

	// Fallback: Create inventory map from API (for backward compatibility)
	orgID := cfg.API.Credentials.OrgID
	inventory, err := client.GetInventory(ctx, orgID, deviceType)
	if err != nil {
		return nil, err
	}

	// Create a map of devices in inventory (all items are already of correct type)
	inventoryMap := make(map[string]bool)
	for _, item := range inventory {
		if item.MAC != nil {
			normalizedMAC := macaddr.NormalizeOrEmpty(*item.MAC)
			if normalizedMAC != "" {
				inventoryMap[normalizedMAC] = true
			}
		}
	}

	// Also check local inventory file if available
	// Get inventory file path from Viper (since cfg.Files.Inventory might not be populated)
	inventoryPath := viper.GetString("files.inventory")
	if inventoryPath == "" {
		inventoryPath = cfg.Files.Inventory // Fallback to config struct
	}

	invConfig, err := client.GetInventoryConfig(inventoryPath)
	if err == nil && invConfig != nil {
		// Add devices from local inventory file
		var localInventory []string
		switch deviceType {
		case "ap":
			localInventory = invConfig.Config.Inventory.AP
		case "switch":
			localInventory = invConfig.Config.Inventory.Switch
		case "gateway":
			localInventory = invConfig.Config.Inventory.Gateway
		}

		for _, mac := range localInventory {
			normalizedMAC := macaddr.NormalizeOrEmpty(mac)
			if normalizedMAC != "" {
				inventoryMap[normalizedMAC] = true
			}
		}
	}

	// Find devices that are:
	// 1. Assigned to the site (in cache)
	// 2. NOT in the configuration
	// 3. IN the inventory (managed by this system)
	for _, mac := range assignedDevices {
		if !configuredDeviceMap[mac] && inventoryMap[mac] {
			devicesToUnassign = append(devicesToUnassign, mac)
			logging.Debugf("Device %s will be unassigned (in inventory but not in config)", mac)
		} else if !configuredDeviceMap[mac] && !inventoryMap[mac] {
			logging.Infof("Device %s is assigned but not in config or inventory - skipping unassign", mac)
		}
	}

	return devicesToUnassign, nil
}

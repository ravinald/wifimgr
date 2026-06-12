package apply

import (
	"context"

	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/macaddr"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// findDevicesToUnassignWithInventoryCheck finds devices that need to be unassigned from the site
// but only if they exist in inventory (to avoid touching devices not managed by this system).
// If inventoryChecker is provided, it is used for O(1) lookups instead of making API calls.
func findDevicesToUnassignWithInventoryCheck(ctx context.Context, client vendors.Client, _ *config.Config,
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

	// Fallback: build the inventory map through the vendor-agnostic API.
	inventory, err := client.Inventory().List(ctx, deviceType)
	if err != nil {
		return nil, err
	}

	// Create a map of devices in inventory (all items are already of correct type)
	inventoryMap := make(map[string]bool)
	for _, item := range inventory {
		normalizedMAC := macaddr.NormalizeOrEmpty(item.MAC)
		if normalizedMAC != "" {
			inventoryMap[normalizedMAC] = true
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

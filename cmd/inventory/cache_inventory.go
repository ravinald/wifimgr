package inventory

import (
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// getInventoryFromCache retrieves inventory items from the multi-vendor cache for a specific device type
func getInventoryFromCache(cacheAccessor *vendors.CacheAccessor, deviceType string) []*vendors.InventoryItem {
	switch deviceType {
	case "ap":
		return cacheAccessor.GetAllAPs()

	case "switch":
		return cacheAccessor.GetAllSwitches()

	case "gateway":
		return cacheAccessor.GetAllGateways()

	default:
		logging.Warnf("Unknown device type for cache lookup: %s", deviceType)
		return nil
	}
}

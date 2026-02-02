package apply

import (
	"context"
	"fmt"

	"github.com/spf13/viper"

	"github.com/ravinald/wifimgr/api"
	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/macaddr"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// InventoryChecker provides methods to verify devices are in inventory before operations
type InventoryChecker struct {
	apiInventory   map[string]bool                   // MAC -> exists in API inventory
	localInventory map[string]bool                   // MAC -> exists in local inventory file
	inventoryItems map[string]*api.MistInventoryItem // MAC -> full inventory item (for site assignment lookups)
	deviceType     string
	client         api.Client // Reference to client for site name lookups
}

// NewInventoryChecker creates an inventory checker for the specified device type.
// Uses multi-vendor cache for inventory data.
func NewInventoryChecker(_ context.Context, client api.Client, cfg *config.Config, deviceType string) (*InventoryChecker, error) {
	accessor := vendors.GetGlobalCacheAccessor()
	if accessor == nil {
		return nil, fmt.Errorf("cache accessor not initialized")
	}

	checker := &InventoryChecker{
		apiInventory:   make(map[string]bool),
		localInventory: make(map[string]bool),
		inventoryItems: make(map[string]*api.MistInventoryItem),
		deviceType:     deviceType,
		client:         client,
	}

	// Get devices of this type from cache
	var inventoryItems []*vendors.InventoryItem
	switch deviceType {
	case "ap":
		inventoryItems = accessor.GetAllAPs()
	case "switch":
		inventoryItems = accessor.GetAllSwitches()
	case "gateway":
		inventoryItems = accessor.GetAllGateways()
	default:
		inventoryItems = accessor.GetAllDevices()
	}

	// Store inventory from cache
	for _, item := range inventoryItems {
		normalizedMAC := macaddr.NormalizeOrEmpty(item.MAC)
		if normalizedMAC != "" {
			checker.apiInventory[normalizedMAC] = true
			// Convert vendor inventory item to api inventory item for compatibility
			mac := item.MAC
			name := item.Name
			siteID := item.SiteID
			apiItem := &api.MistInventoryItem{
				MAC:    &mac,
				Name:   &name,
				SiteID: &siteID,
			}
			checker.inventoryItems[normalizedMAC] = apiItem
		}
	}
	logging.Debugf("Loaded %d %s devices from cache", len(checker.apiInventory), deviceType)

	// Load local inventory file
	// Get inventory file path from Viper (since cfg.Files.Inventory might not be populated)
	inventoryPath := viper.GetString("files.inventory")
	if inventoryPath == "" {
		inventoryPath = cfg.Files.Inventory // Fallback to config struct
	}

	logging.Infof("Loading local inventory from path: %s", inventoryPath)
	invConfig, err := client.GetInventoryConfig(inventoryPath)
	if err != nil {
		logging.Errorf("Failed to load inventory config: %v", err)
	}
	if err == nil && invConfig != nil {
		logging.Debugf("Inventory config loaded successfully, checking for %s devices", deviceType)
		var localInventory []string
		switch deviceType {
		case "ap":
			localInventory = invConfig.Config.Inventory.AP
			logging.Debugf("Found %d AP devices in inventory config", len(localInventory))
		case "switch":
			localInventory = invConfig.Config.Inventory.Switch
		case "gateway":
			localInventory = invConfig.Config.Inventory.Gateway
		}

		for _, mac := range localInventory {
			normalizedMAC := macaddr.NormalizeOrEmpty(mac)
			if normalizedMAC != "" {
				checker.localInventory[normalizedMAC] = true
				logging.Debugf("Added %s to local inventory (normalized: %s)", mac, normalizedMAC)
			}
		}
		logging.Debugf("Loaded %d %s devices from local inventory", len(checker.localInventory), deviceType)
	} else {
		logging.Warnf("Could not load local inventory configuration: %v", err)
	}

	logging.Infof("Inventory checker initialized: %d devices in API inventory, %d in local inventory",
		len(checker.apiInventory), len(checker.localInventory))

	return checker, nil
}

// IsInInventory checks if a device MAC is in BOTH API and local inventory.
// This is the strict check required for write operations (apply, configure, assign, unassign).
//
// Rationale:
//   - API inventory = devices that exist in the vendor account
//   - Local inventory = fail-safe allowlist of devices we're allowed to modify
//   - We might want to view devices in the API, but NOT write changes unless explicitly allowlisted
//
// For read-only operations, use IsInAPIInventory() instead.
func (ic *InventoryChecker) IsInInventory(mac string) bool {
	normalizedMAC := macaddr.NormalizeOrEmpty(mac)
	if normalizedMAC == "" {
		return false
	}

	// Device must be in BOTH inventories for write operations
	return ic.apiInventory[normalizedMAC] && ic.localInventory[normalizedMAC]
}

// IsInLocalInventory checks if a device MAC is specifically in the local inventory file
func (ic *InventoryChecker) IsInLocalInventory(mac string) bool {
	normalizedMAC := macaddr.NormalizeOrEmpty(mac)
	if normalizedMAC == "" {
		return false
	}
	return ic.localInventory[normalizedMAC]
}

// IsInAPIInventory checks if a device MAC is in the API inventory
func (ic *InventoryChecker) IsInAPIInventory(mac string) bool {
	normalizedMAC := macaddr.NormalizeOrEmpty(mac)
	if normalizedMAC == "" {
		return false
	}
	return ic.apiInventory[normalizedMAC]
}

// FilterByInventory filters a list of MACs to only include those in inventory
func (ic *InventoryChecker) FilterByInventory(macs []string) []string {
	filtered := make([]string, 0, len(macs))
	skipped := 0

	for _, mac := range macs {
		if ic.IsInInventory(mac) {
			filtered = append(filtered, mac)
		} else {
			skipped++
			logging.Debugf("Skipping device %s - not in inventory", mac)
		}
	}

	if skipped > 0 {
		logging.Infof("Filtered out %d devices not in inventory", skipped)
	}

	return filtered
}

// LogInventoryStatus logs the inventory status of a device
func (ic *InventoryChecker) LogInventoryStatus(mac string) {
	normalizedMAC := macaddr.NormalizeOrEmpty(mac)
	if normalizedMAC == "" {
		return
	}

	inAPI := ic.IsInAPIInventory(normalizedMAC)
	inLocal := ic.IsInLocalInventory(normalizedMAC)

	if inAPI && inLocal {
		logging.Debugf("Device %s is in both API and local inventory (WRITE ALLOWED)", normalizedMAC)
	} else if inAPI {
		logging.Debugf("Device %s is in API inventory only (READ-ONLY, not allowlisted for writes)", normalizedMAC)
	} else if inLocal {
		logging.Warnf("Device %s is in local inventory only but NOT in API (cannot operate on device)", normalizedMAC)
	} else {
		logging.Warnf("Device %s is NOT in any inventory", normalizedMAC)
	}
}

// GetSiteAssignment returns the current site assignment for a device from the cached inventory.
// This provides O(1) lookup without additional API calls.
// Returns siteID, siteName, and whether the device was found in inventory.
func (ic *InventoryChecker) GetSiteAssignment(mac string) (siteID, siteName string, found bool) {
	normalizedMAC := macaddr.NormalizeOrEmpty(mac)
	if normalizedMAC == "" {
		return "", "", false
	}

	item, exists := ic.inventoryItems[normalizedMAC]
	if !exists || item == nil {
		return "", "", false
	}

	// Get site ID from inventory item
	if item.SiteID != nil && *item.SiteID != "" {
		siteID = *item.SiteID

		// Try multi-vendor cache first for site name lookup
		if accessor := vendors.GetGlobalCacheAccessor(); accessor != nil {
			if site, err := accessor.GetSiteByID(siteID); err == nil && site != nil {
				siteName = site.Name
			}
		} else if ic.client != nil {
			// Fall back to legacy client's cached site data
			if name, nameFound := ic.client.GetSiteName(siteID); nameFound {
				siteName = name
			}
		}
	}

	return siteID, siteName, true
}

// GetInventoryItem returns the full inventory item for a device MAC.
// This provides O(1) lookup without additional API calls.
func (ic *InventoryChecker) GetInventoryItem(mac string) (*api.MistInventoryItem, bool) {
	normalizedMAC := macaddr.NormalizeOrEmpty(mac)
	if normalizedMAC == "" {
		return nil, false
	}

	item, exists := ic.inventoryItems[normalizedMAC]
	return item, exists
}

package vendors

import (
	"context"
	"fmt"
	"strings"

	"github.com/ravinald/wifimgr/internal/logging"
)

// FetchDeviceConfig fetches a single device's config from the API and updates the cache.
// This is used for on-demand config fetching, especially for Meraki where bulk fetches are expensive.
// Returns the fetched config or nil if the config service is not available.
func (c *CacheManager) FetchDeviceConfig(ctx context.Context, apiLabel, deviceType, mac string) (any, error) {
	normalizedMAC := NormalizeMAC(mac)

	client, err := c.registry.GetClient(apiLabel)
	if err != nil {
		return nil, fmt.Errorf("failed to get client for %s: %w", apiLabel, err)
	}

	cfgSvc := client.Configs()
	if cfgSvc == nil {
		logging.Debugf("[cache] Config service not available for %s", apiLabel)
		return nil, nil
	}

	// Get the device from cache to find its ID and SiteID
	cache, err := c.GetAPICache(apiLabel)
	if err != nil {
		return nil, fmt.Errorf("failed to load cache for %s: %w", apiLabel, err)
	}

	// Find the device in the appropriate inventory
	var item *InventoryItem
	switch deviceType {
	case "ap":
		item = cache.Inventory.AP[normalizedMAC]
	case "switch":
		item = cache.Inventory.Switch[normalizedMAC]
	case "gateway":
		item = cache.Inventory.Gateway[normalizedMAC]
	default:
		return nil, fmt.Errorf("unknown device type: %s", deviceType)
	}

	if item == nil {
		return nil, fmt.Errorf("device %s not found in %s cache", mac, apiLabel)
	}

	if item.ID == "" || item.SiteID == "" {
		return nil, fmt.Errorf("device %s missing ID or SiteID", mac)
	}

	logging.Debugf("[cache] Fetching %s config for device %s (id=%s, site=%s) from %s",
		deviceType, mac, item.ID, item.SiteID, apiLabel)

	// Fetch the config based on device type
	var cfg any
	var fetchedConfig bool
	switch deviceType {
	case "ap":
		apCfg, fetchErr := cfgSvc.GetAPConfig(ctx, item.SiteID, item.ID)
		err = fetchErr
		if err == nil {
			cache.Configs.AP[normalizedMAC] = apCfg
			cfg = apCfg
			fetchedConfig = true
		}
	case "switch":
		swCfg, fetchErr := cfgSvc.GetSwitchConfig(ctx, item.SiteID, item.ID)
		err = fetchErr
		if err == nil {
			cache.Configs.Switch[normalizedMAC] = swCfg
			cfg = swCfg
			fetchedConfig = true
		}
	case "gateway":
		gwCfg, fetchErr := cfgSvc.GetGatewayConfig(ctx, item.SiteID, item.ID)
		err = fetchErr
		if err == nil {
			cache.Configs.Gateway[normalizedMAC] = gwCfg
			cfg = gwCfg
			fetchedConfig = true
		}
	default:
		// This should never happen as deviceType is validated earlier
		return nil, fmt.Errorf("unexpected device type: %s", deviceType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s config for %s: %w", deviceType, mac, err)
	}

	// Save updated cache
	if fetchedConfig {
		if saveErr := c.SaveAPICache(cache); saveErr != nil {
			logging.Warnf("[cache] Failed to save updated cache after fetching config: %v", saveErr)
			// Don't fail the operation, we still have the config
		} else {
			logging.Debugf("[cache] Saved updated config for device %s to cache", mac)
		}
	}

	return cfg, nil
}

// EnsureDeviceConfig ensures that a device's config is in the cache.
// For Meraki, this fetches the config on-demand if not already cached.
// For Mist, configs are typically already in the cache from the refresh.
// Returns true if config was fetched, false if already cached or not needed.
func (c *CacheManager) EnsureDeviceConfig(ctx context.Context, apiLabel, deviceType, mac string) (bool, error) {
	config, err := c.registry.GetConfig(apiLabel)
	if err != nil {
		return false, err
	}

	// For non-Meraki vendors, we assume configs are already in cache
	if config.Vendor != "meraki" {
		return false, nil
	}

	normalizedMAC := NormalizeMAC(mac)

	// Check if config is already in cache
	cache, err := c.GetAPICache(apiLabel)
	if err != nil {
		return false, err
	}

	var hasConfig bool
	switch deviceType {
	case "ap":
		_, hasConfig = cache.Configs.AP[normalizedMAC]
	case "switch":
		_, hasConfig = cache.Configs.Switch[normalizedMAC]
	case "gateway":
		_, hasConfig = cache.Configs.Gateway[normalizedMAC]
	}

	if hasConfig {
		logging.Debugf("[cache] Config for %s %s already in cache", deviceType, mac)
		return false, nil
	}

	// Fetch the config
	logging.Infof("Fetching %s config for %s from Meraki API...", deviceType, mac)
	_, err = c.FetchDeviceConfig(ctx, apiLabel, deviceType, mac)
	if err != nil {
		return false, err
	}

	return true, nil
}

// NormalizeMAC normalizes a MAC address to lowercase without separators.
func NormalizeMAC(mac string) string {
	mac = strings.ReplaceAll(mac, ":", "")
	mac = strings.ReplaceAll(mac, "-", "")
	mac = strings.ReplaceAll(mac, ".", "")
	return strings.ToLower(mac)
}

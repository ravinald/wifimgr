package cmd

import (
	"context"
	"fmt"

	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// RefreshAPICacheForApply refreshes the cache for the specified API before apply operations.
// This ensures we have the latest running config to compare against.
func RefreshAPICacheForApply(ctx context.Context, apiLabel string) error {
	cacheMgr := GetCacheManager()
	if cacheMgr == nil {
		return fmt.Errorf("cache manager not initialized")
	}

	fmt.Printf("Refreshing %s cache to get current running config...\n", apiLabel)
	if err := cacheMgr.RefreshAPI(ctx, apiLabel); err != nil {
		return fmt.Errorf("failed to refresh %s cache: %w", apiLabel, err)
	}
	fmt.Printf("Cache refreshed successfully\n")

	return nil
}

// ResolveAPIForSite determines which API to use for a site based on:
// 1. target keyword override (with warning if different from config)
// 2. Site config's api field
// 3. Cache lookup to find which API has this site
// 4. Default to first available API if only one is configured
func ResolveAPIForSite(siteName string, siteConfig *config.SiteConfig) (string, error) {
	registry := GetAPIRegistry()
	if registry == nil {
		return "", fmt.Errorf("API registry not initialized")
	}

	cacheMgr := GetCacheManager()
	allLabels := registry.GetAllLabels()

	// Determine the expected API from config or cache
	var expectedAPI string

	// First check site config's api field
	if siteConfig != nil && siteConfig.API != "" {
		expectedAPI = siteConfig.API
		// Validate that this API exists
		if !registry.HasAPI(expectedAPI) {
			return "", fmt.Errorf("site config specifies API '%s' which is not configured\nAvailable APIs: %v", expectedAPI, allLabels)
		}
	}

	// If not in config, try to find in cache
	if expectedAPI == "" && cacheMgr != nil {
		for _, label := range allLabels {
			cache, err := cacheMgr.GetAPICache(label)
			if err != nil {
				continue
			}
			// Check if this API has the site
			if _, ok := cache.SiteIndex.ByName[siteName]; ok {
				expectedAPI = label
				break
			}
		}
	}

	// Handle target keyword override
	if apiFlag != "" {
		if expectedAPI != "" && apiFlag != expectedAPI {
			logging.Warnf("target (%s) overrides site's configured API (%s)", apiFlag, expectedAPI)
			fmt.Printf("WARN: Using target %s instead of site's configured API %s\n", apiFlag, expectedAPI)
		}
		return apiFlag, nil
	}

	// Return expected API if found
	if expectedAPI != "" {
		return expectedAPI, nil
	}

	// If only one API is configured, use it as default
	if len(allLabels) == 1 {
		return allLabels[0], nil
	}

	// Multiple APIs but couldn't determine which one
	return "", fmt.Errorf("could not determine which API to use for site '%s'\nSpecify with 'target <label>' or add 'api' field to site config\nAvailable APIs: %v", siteName, allLabels)
}

// ValidateMultiVendorApply checks if an apply operation is valid.
// Returns the resolved API label and any validation error.
func ValidateMultiVendorApply(_ context.Context, siteName string, siteConfig *config.SiteConfig) (string, error) {
	// Resolve which API to use
	apiLabel, err := ResolveAPIForSite(siteName, siteConfig)
	if err != nil {
		return "", err
	}

	// Validate the API exists and has the site
	registry := GetAPIRegistry()
	if !registry.HasAPI(apiLabel) {
		return "", fmt.Errorf("API '%s' is not configured", apiLabel)
	}

	// Verify site exists in the API's cache
	cacheMgr := GetCacheManager()
	if cacheMgr != nil {
		cache, err := cacheMgr.GetAPICache(apiLabel)
		if err == nil {
			if _, ok := cache.SiteIndex.ByName[siteName]; !ok {
				logging.Warnf("Site '%s' not found in %s cache - may need to refresh cache", siteName, apiLabel)
			}
		}
	}

	return apiLabel, nil
}

// IsMultiVendorApplySupported checks if the apply operation is supported
// for the target API's vendor.
func IsMultiVendorApplySupported(apiLabel string) (bool, string) {
	registry := GetAPIRegistry()
	if registry == nil {
		return false, "API registry not initialized"
	}

	vendor, err := registry.GetVendor(apiLabel)
	if err != nil {
		return false, err.Error()
	}

	// Check vendor-specific support
	switch vendor {
	case "mist":
		return true, ""
	case "meraki":
		// Meraki apply is partially supported
		return true, ""
	default:
		return false, fmt.Sprintf("apply not supported for vendor '%s'", vendor)
	}
}

// EnsureDeviceConfigsForSite fetches configs for all devices in a site that will be modified.
// This is used to batch-fetch configs before applying changes to multiple devices.
// For Meraki, this fetches configs on-demand. For Mist, configs are already cached.
//
// Parameters:
//   - ctx: Context for the API calls
//   - apiLabel: The API label
//   - siteName: Site name to find devices
//   - deviceType: "ap", "switch", "gateway", or "all"
//   - macs: List of device MACs that will be modified (empty = all in site)
//
// Returns the number of configs fetched from the API.
func EnsureDeviceConfigsForSite(ctx context.Context, apiLabel, siteName, deviceType string, macs []string) (int, error) {
	cacheMgr := GetCacheManager()
	if cacheMgr == nil {
		return 0, fmt.Errorf("cache manager not initialized")
	}

	registry := GetAPIRegistry()
	if registry == nil {
		return 0, fmt.Errorf("API registry not initialized")
	}

	// Check if this is a Meraki API
	vendor, err := registry.GetVendor(apiLabel)
	if err != nil {
		return 0, err
	}

	// Only Meraki needs on-demand fetching
	if vendor != "meraki" {
		return 0, nil
	}

	// Get site ID from cache
	siteID, err := cacheMgr.GetSiteIDByName(apiLabel, siteName)
	if err != nil {
		return 0, fmt.Errorf("failed to find site %s: %w", siteName, err)
	}

	cache, err := cacheMgr.GetAPICache(apiLabel)
	if err != nil {
		return 0, err
	}

	fetchCount := 0
	macSet := make(map[string]bool)
	for _, mac := range macs {
		macSet[vendors.NormalizeMAC(mac)] = true
	}

	// Determine which device types to process
	types := []string{deviceType}
	if deviceType == "all" {
		types = []string{"ap", "switch", "gateway"}
	}

	for _, devType := range types {
		var inventory map[string]*vendors.InventoryItem
		switch devType {
		case "ap":
			inventory = cache.Inventory.AP
		case "switch":
			inventory = cache.Inventory.Switch
		case "gateway":
			inventory = cache.Inventory.Gateway
		}

		for mac, item := range inventory {
			// Filter by site
			if item.SiteID != siteID {
				continue
			}

			// Filter by MAC list if provided
			if len(macSet) > 0 && !macSet[mac] {
				continue
			}

			fetched, err := cacheMgr.EnsureDeviceConfig(ctx, apiLabel, devType, mac)
			if err != nil {
				logging.Warnf("Failed to fetch config for %s %s: %v", devType, mac, err)
				continue
			}
			if fetched {
				fetchCount++
			}
		}
	}

	if fetchCount > 0 {
		logging.Infof("Fetched %d device configs from Meraki API before apply", fetchCount)
	}

	return fetchCount, nil
}

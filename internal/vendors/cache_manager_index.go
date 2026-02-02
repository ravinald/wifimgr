package vendors

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ravinald/wifimgr/internal/helpers"
	"github.com/ravinald/wifimgr/internal/logging"
)

// RebuildIndex rebuilds the cross-API index from all API cache files.
func (c *CacheManager) RebuildIndex() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	index := NewCrossAPIIndex()

	// Find all API cache files
	apisDir := filepath.Join(c.cacheDir, "apis")
	entries, err := os.ReadDir(apisDir)
	if err != nil {
		if os.IsNotExist(err) {
			c.index = index
			return c.saveIndex()
		}
		return fmt.Errorf("failed to read apis directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		apiLabel := strings.TrimSuffix(entry.Name(), ".json")
		cache, err := c.GetAPICache(apiLabel)
		if err != nil {
			// Log warning but continue
			continue
		}

		// Index MACs
		for mac := range cache.Inventory.AP {
			c.indexMAC(index, mac, apiLabel)
		}
		for mac := range cache.Inventory.Switch {
			c.indexMAC(index, mac, apiLabel)
		}
		for mac := range cache.Inventory.Gateway {
			c.indexMAC(index, mac, apiLabel)
		}

		// Index site names
		for siteName := range cache.SiteIndex.ByName {
			index.SiteNameToAPIs[siteName] = append(index.SiteNameToAPIs[siteName], apiLabel)
		}
	}

	c.index = index
	return c.saveIndex()
}

// indexMAC adds a MAC to the index with collision detection.
func (c *CacheManager) indexMAC(index *CrossAPIIndex, mac, apiLabel string) {
	normalizedMAC := NormalizeMAC(mac)

	if existingAPI, found := index.MACToAPI[normalizedMAC]; found {
		if existingAPI != apiLabel {
			// MAC collision - log error but keep existing mapping
			_, _ = fmt.Fprintf(os.Stderr, "ERROR MAC collision: %s exists in both %q and %q - keeping %q\n",
				normalizedMAC, existingAPI, apiLabel, existingAPI)
			return
		}
	}

	index.MACToAPI[normalizedMAC] = apiLabel
}

// saveIndex writes the cross-API index to disk.
func (c *CacheManager) saveIndex() error {
	indexPath := filepath.Join(c.cacheDir, "index.json")
	data, err := json.MarshalIndent(c.index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	if err := os.WriteFile(indexPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write index: %w", err)
	}

	return nil
}

// FindDeviceByMAC looks up a device across all APIs.
func (c *CacheManager) FindDeviceByMAC(mac string) (*InventoryItem, string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	normalizedMAC := NormalizeMAC(mac)

	apiLabel, found := c.index.MACToAPI[normalizedMAC]
	if !found {
		return nil, "", &DeviceNotFoundError{Identifier: mac}
	}

	cache, err := c.GetAPICache(apiLabel)
	if err != nil {
		return nil, "", err
	}

	// Check each device type
	if item, ok := cache.Inventory.AP[normalizedMAC]; ok {
		item.SourceAPI = apiLabel
		item.SourceVendor = cache.Meta.Vendor
		return item, apiLabel, nil
	}
	if item, ok := cache.Inventory.Switch[normalizedMAC]; ok {
		item.SourceAPI = apiLabel
		item.SourceVendor = cache.Meta.Vendor
		return item, apiLabel, nil
	}
	if item, ok := cache.Inventory.Gateway[normalizedMAC]; ok {
		item.SourceAPI = apiLabel
		item.SourceVendor = cache.Meta.Vendor
		return item, apiLabel, nil
	}

	return nil, "", &DeviceNotFoundError{Identifier: mac, APILabel: apiLabel}
}

// GetSiteAPIs returns all APIs that have a site with the given name.
func (c *CacheManager) GetSiteAPIs(siteName string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.index == nil {
		return nil
	}
	return c.index.SiteNameToAPIs[siteName]
}

// GetSiteIDByName returns the site ID for a given site name in a specific API.
func (c *CacheManager) GetSiteIDByName(apiLabel, siteName string) (string, error) {
	cache, err := c.GetAPICache(apiLabel)
	if err != nil {
		return "", err
	}

	siteID, found := cache.SiteIndex.ByName[siteName]
	if !found {
		return "", &SiteNotFoundError{SiteName: siteName, APILabel: apiLabel}
	}

	return siteID, nil
}

// VerifyAPICache checks the status of an API's cache.
func (c *CacheManager) VerifyAPICache(apiLabel string) (CacheStatus, error) {
	cachePath := c.getAPICachePath(apiLabel)
	metaPath := c.getAPICacheMetaPath(apiLabel)

	// Check if cache file exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return CacheMissing, nil
	}

	// Check file integrity using hash if metadata exists
	if _, err := os.Stat(metaPath); err == nil {
		if err := helpers.VerifyFileIntegrity(cachePath, metaPath); err != nil {
			logging.Warnf("[cache] Integrity verification failed for %s: %v", apiLabel, err)
			return CacheCorrupted, fmt.Errorf("integrity check failed: %w", err)
		}
	}

	cache, err := c.GetAPICache(apiLabel)
	if err != nil {
		return CacheCorrupted, err
	}

	// Get TTL from API config (0 = never expire)
	config, err := c.registry.GetConfig(apiLabel)
	if err != nil {
		// If we can't get config, use default behavior
		logging.Debugf("[cache] Could not get config for %s, assuming default TTL", apiLabel)
	}

	// Check staleness (only if TTL > 0)
	if config == nil || config.CacheTTL > 0 {
		ttl := time.Duration(86400) * time.Second // default 1 day
		if config != nil && config.CacheTTL > 0 {
			ttl = time.Duration(config.CacheTTL) * time.Second
		}
		age := time.Since(cache.Meta.LastRefresh)
		if age > ttl {
			return CacheStale, nil
		}
	}
	// If CacheTTL == 0, cache never expires (on-demand refresh only)

	// Verify item counts match actual data
	actualAPs := len(cache.Inventory.AP)
	if cache.Meta.ItemCounts["inventory_ap"] != actualAPs {
		return CacheCorrupted, nil
	}

	return CacheOK, nil
}

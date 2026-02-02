package vendors

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/ravinald/wifimgr/internal/helpers"
	"github.com/ravinald/wifimgr/internal/logging"
)

// CacheManager manages per-API cache files and the cross-API index.
type CacheManager struct {
	cacheDir string
	registry *APIClientRegistry
	index    *CrossAPIIndex
	mu       sync.RWMutex
}

// RefreshOptions controls the behavior of cache refresh operations.
type RefreshOptions struct {
	// FetchDeviceConfigs determines whether to fetch individual device configs.
	// For Meraki this is expensive due to per-device API calls.
	// Default: false (only fetch on explicit refresh or initial cache creation)
	FetchDeviceConfigs bool
}

// NewCacheManager creates a new cache manager.
func NewCacheManager(cacheDir string, registry *APIClientRegistry) *CacheManager {
	return &CacheManager{
		cacheDir: cacheDir,
		registry: registry,
	}
}

// Initialize creates the cache directory structure and loads the index.
func (c *CacheManager) Initialize() error {
	// Create cache/apis directory
	apisDir := filepath.Join(c.cacheDir, "apis")
	if err := os.MkdirAll(apisDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Load or create index
	return c.loadIndex()
}

// loadIndex loads the cross-API index from disk or creates a new one.
func (c *CacheManager) loadIndex() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	indexPath := filepath.Join(c.cacheDir, "index.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			c.index = NewCrossAPIIndex()
			return nil
		}
		return fmt.Errorf("failed to read index file: %w", err)
	}

	c.index = &CrossAPIIndex{}
	if err := json.Unmarshal(data, c.index); err != nil {
		// Corrupted index, create new one
		c.index = NewCrossAPIIndex()
		return nil
	}

	return nil
}

// GetAPICache loads a single API's cache file.
func (c *CacheManager) GetAPICache(apiLabel string) (*APICache, error) {
	cachePath := c.getAPICachePath(apiLabel)
	metaPath := c.getAPICacheMetaPath(apiLabel)

	// Verify cache integrity if metadata file exists
	if _, err := os.Stat(metaPath); err == nil {
		if err := helpers.VerifyFileIntegrity(cachePath, metaPath); err != nil {
			logging.Warnf("Cache integrity check failed for %s: %v", apiLabel, err)
			// Continue loading but log the warning - the cache may have been
			// modified outside of wifimgr
		}
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &APINotFoundError{APILabel: apiLabel}
		}
		return nil, fmt.Errorf("failed to read cache for %s: %w", apiLabel, err)
	}

	cache := &APICache{}
	if err := json.Unmarshal(data, cache); err != nil {
		return nil, fmt.Errorf("failed to parse cache for %s: %w", apiLabel, err)
	}

	return cache, nil
}

// SaveAPICache saves a single API's cache file.
func (c *CacheManager) SaveAPICache(cache *APICache) error {
	cache.UpdateItemCounts()
	cache.RebuildSiteIndex()

	cachePath := c.getAPICachePath(cache.APILabel)
	metaPath := c.getAPICacheMetaPath(cache.APILabel)

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache: %w", err)
	}

	// Create metadata file for integrity checking
	if err := helpers.CreateFileMetadata(cachePath, metaPath, "api_cache"); err != nil {
		logging.Warnf("Failed to create metadata file for %s: %v", cache.APILabel, err)
		// Don't fail the save operation if metadata creation fails
	}

	return nil
}

// getAPICachePath returns the path for an API's cache file.
func (c *CacheManager) getAPICachePath(apiLabel string) string {
	return filepath.Join(c.cacheDir, "apis", apiLabel+".json")
}

// getAPICacheMetaPath returns the path for an API's cache metadata file.
// Uses dotfile format: .apiLabel.json.meta
func (c *CacheManager) getAPICacheMetaPath(apiLabel string) string {
	return filepath.Join(c.cacheDir, "apis", "."+apiLabel+".json.meta")
}

// CacheExists checks if a cache file exists for the given API.
func (c *CacheManager) CacheExists(apiLabel string) bool {
	cachePath := c.getAPICachePath(apiLabel)
	_, err := os.Stat(cachePath)
	return err == nil
}

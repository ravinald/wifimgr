package vendors

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/ravinald/wifimgr/internal/encryption"
	"github.com/ravinald/wifimgr/internal/helpers"
	"github.com/ravinald/wifimgr/internal/logging"
)

// CacheManager manages per-API cache files and the cross-API index.
type CacheManager struct {
	cacheDir string
	registry *APIClientRegistry
	index    *CrossAPIIndex
	mu       sync.RWMutex

	// labelMus gives each apiLabel its own mutex so concurrent save/refresh
	// for different APIs don't serialize, while save/refresh for the same
	// API is strictly ordered. Combined with WriteFileAtomic this makes
	// in-process concurrent refreshes safe.
	labelMus sync.Map // map[string]*sync.Mutex

	// secretPw caches the encryption password used to protect WLAN secrets at
	// rest. Resolved once per process so a parallel refresh-all prompts at most
	// once; see secretPassword.
	secretPwOnce sync.Once
	secretPw     string
	secretPwErr  error
}

// secretPassword resolves the password used to encrypt WLAN secrets in the
// cache — from WIFIMGR_PASSWORD or an interactive prompt — once per process.
func (c *CacheManager) secretPassword() (string, error) {
	c.secretPwOnce.Do(func() {
		c.secretPw, c.secretPwErr = encryption.GetPasswordOrPrompt(
			"Enter encryption password to protect WLAN secrets in cache: ")
	})
	return c.secretPw, c.secretPwErr
}

// RefreshOptions controls the behavior of cache refresh operations.
type RefreshOptions struct {
	// FetchDeviceConfigs determines whether to fetch individual device configs.
	// For Meraki this is expensive due to per-device API calls.
	// Default: false (only fetch on explicit refresh or initial cache creation)
	FetchDeviceConfigs bool

	// SiteID, when set, scopes per-device config fetches to devices that
	// belong to that site. Org-scoped fetches (sites, inventory, statuses,
	// templates, profiles, WLANs) still happen — they're cheap. Configs
	// for devices in other sites are copied forward from the existing
	// cache so the saved file isn't a regression. Useful on Meraki, where
	// the per-device config endpoint is the expensive part of a refresh.
	SiteID string

	// ManagedMACs, when non-nil, restricts per-device config fetches to the
	// listed (normalized) MACs — the operator's armed inventory. Devices not
	// in the set keep their prior cached config (carried forward), so a
	// managed refresh stays cheap on Meraki without discarding data a full
	// pass collected. nil means no managed filter: fetch every device in scope.
	ManagedMACs map[string]bool
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
	// 0700: cache holds org IDs, full MAC inventory, and site topology — infrastructure
	// enumeration data that other local users have no business reading.
	if err := os.MkdirAll(apisDir, 0700); err != nil {
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

	// Backfill InventoryItem.SiteName for caches written before this code
	// landed (or by adapters that don't populate it). Cheap: just walks
	// three maps already in memory. Newly saved caches already have
	// SiteName set via saveAPICacheLocked, so this is effectively a no-op
	// for fresh files.
	cache.BackfillInventorySiteNames()

	return cache, nil
}

// labelLock returns the per-label mutex, creating it on first use. All
// save/refresh operations for a given apiLabel must hold this mutex.
func (c *CacheManager) labelLock(apiLabel string) *sync.Mutex {
	if m, ok := c.labelMus.Load(apiLabel); ok {
		return m.(*sync.Mutex)
	}
	m, _ := c.labelMus.LoadOrStore(apiLabel, &sync.Mutex{})
	return m.(*sync.Mutex)
}

// SaveAPICache saves a single API's cache file. Safe to call concurrently
// from different goroutines; invocations for the same apiLabel are
// serialized via the per-label mutex.
func (c *CacheManager) SaveAPICache(cache *APICache) error {
	lock := c.labelLock(cache.APILabel)
	lock.Lock()
	defer lock.Unlock()
	return c.saveAPICacheLocked(cache)
}

// saveAPICacheLocked writes the cache assuming the caller already holds the
// per-label lock. Internal use only.
func (c *CacheManager) saveAPICacheLocked(cache *APICache) error {
	cache.UpdateItemCounts()
	cache.RebuildSiteIndex()
	cache.BackfillInventorySiteNames()

	cachePath := c.getAPICachePath(cache.APILabel)
	metaPath := c.getAPICacheMetaPath(cache.APILabel)

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	if err := helpers.WriteFileAtomic(cachePath, data, 0600); err != nil {
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

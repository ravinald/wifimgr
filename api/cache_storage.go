package api

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/spf13/viper"

	"github.com/ravinald/wifimgr/internal/helpers"
	"github.com/ravinald/wifimgr/internal/logging"
)

// CacheManager manages the unified cache system
type CacheManager struct {
	mu          sync.RWMutex
	cache       *Cache
	indexes     *CacheIndexes
	cachePath   string
	metaPath    string
	backupPath  string
	initialized bool
}

// NewCacheManager creates a new cache manager instance
func NewCacheManager(cachePath string) *CacheManager {
	// Create dotfile metadata path by inserting .meta before the filename
	dir := filepath.Dir(cachePath)
	filename := filepath.Base(cachePath)
	metaPath := filepath.Join(dir, "."+filename+".meta")

	return &CacheManager{
		cachePath:  cachePath,
		metaPath:   metaPath,
		backupPath: cachePath + ".backup",
	}
}

// Initialize loads the cache from disk and builds indexes
func (cm *CacheManager) Initialize() error {
	return cm.InitializeWithOptions(false)
}

// InitializeWithOptions loads the cache with optional force mode that skips integrity checks
func (cm *CacheManager) InitializeWithOptions(forceRecreate bool) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.initialized {
		return nil
	}

	// Check if cache file exists
	cacheInfo, err := os.Stat(cm.cachePath)
	if os.IsNotExist(err) || forceRecreate {
		// Initialize empty cache (either file doesn't exist or forcing recreation)
		cm.cache = cm.newEmptyCache()
		cm.indexes = NewCacheIndexes()
		cm.initialized = true
		return nil
	}

	// Check cache TTL status
	if err == nil {
		cacheTTL := viper.GetInt("files.cache_ttl")
		lastModified := cacheInfo.ModTime().UTC()
		if metaInfo, err := os.Stat(cm.metaPath); err == nil {
			lastModified = metaInfo.ModTime().UTC()
		}

		if cacheTTL > 0 {
			expirationTime := lastModified.Add(time.Duration(cacheTTL) * time.Second)
			currentTime := time.Now().UTC()

			if currentTime.After(expirationTime) || currentTime.Equal(expirationTime) {
				// Cache has expired
				timePastExpiry := currentTime.Sub(expirationTime)

				logging.Infof("Cache expired: TTL of %d seconds exceeded by %s. Triggering automatic rebuild.", cacheTTL, timePastExpiry.Round(time.Second))
				fmt.Printf("Cache expired: TTL of %d seconds exceeded by %s. Rebuilding cache automatically...\n", cacheTTL, timePastExpiry.Round(time.Second))

				// Cache has expired, initialize empty cache
				cm.cache = cm.newEmptyCache()
				cm.indexes = NewCacheIndexes()
				cm.initialized = true
				return nil
			} else {
				// Cache is still valid
				timeRemaining := expirationTime.Sub(currentTime)
				cacheAge := currentTime.Sub(lastModified)

				logging.Debugf("Cache TTL check: Cache age %s, TTL %d seconds, time remaining %s",
					cacheAge.Round(time.Second), cacheTTL, timeRemaining.Round(time.Second))
			}
		} else {
			logging.Debugf("Cache TTL check: TTL disabled (value: %d)", cacheTTL)
		}
	}

	// Load cache from disk
	if err := cm.loadFromDisk(); err != nil {
		return fmt.Errorf("failed to load cache from disk: %w", err)
	}

	// Build indexes
	if err := cm.buildIndexes(); err != nil {
		return fmt.Errorf("failed to build cache indexes: %w", err)
	}

	cm.initialized = true
	return nil
}

// newEmptyCache creates a new empty cache structure
func (cm *CacheManager) newEmptyCache() *Cache {
	return &Cache{
		Version: 1,
		Orgs:    make(map[string]*OrgData),
	}
}

// GetCache returns a pointer to the internal cache for reading.
//
// WARNING: The returned pointer provides direct access to internal state.
// Callers MUST NOT modify the returned cache or any data within it.
// For thread-safe access when performing multiple reads, use ReadCache instead.
// The read lock is only held during this method call, not while using the pointer.
func (cm *CacheManager) GetCache() (*Cache, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if !cm.initialized {
		return nil, fmt.Errorf("cache manager not initialized")
	}

	return cm.cache, nil
}

// ReadCache provides thread-safe read access to the cache.
// The callback function is executed while holding the read lock,
// ensuring consistent reads even during concurrent writes.
// Use this method when performing multiple related reads that must be consistent.
func (cm *CacheManager) ReadCache(fn func(*Cache) error) error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if !cm.initialized {
		return fmt.Errorf("cache manager not initialized")
	}

	return fn(cm.cache)
}

// GetIndexes returns a pointer to the internal cache indexes for reading.
//
// WARNING: The returned pointer provides direct access to internal state.
// Callers MUST NOT modify the returned indexes or any data within them.
// For thread-safe access when performing multiple reads, use ReadIndexes instead.
// The read lock is only held during this method call, not while using the pointer.
func (cm *CacheManager) GetIndexes() (*CacheIndexes, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if !cm.initialized {
		return nil, fmt.Errorf("cache manager not initialized")
	}

	return cm.indexes, nil
}

// ReadIndexes provides thread-safe read access to the cache indexes.
// The callback function is executed while holding the read lock,
// ensuring consistent reads even during concurrent writes.
func (cm *CacheManager) ReadIndexes(fn func(*CacheIndexes) error) error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if !cm.initialized {
		return fmt.Errorf("cache manager not initialized")
	}

	return fn(cm.indexes)
}

// ReplaceCache atomically replaces the entire cache and rebuilds indexes
func (cm *CacheManager) ReplaceCache(newCache *Cache) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Validate new cache
	if newCache == nil {
		return fmt.Errorf("new cache cannot be nil")
	}

	if newCache.Version < 1 {
		return fmt.Errorf("unsupported cache version %d", newCache.Version)
	}

	// Replace cache
	cm.cache = newCache

	// Rebuild indexes
	if err := cm.buildIndexes(); err != nil {
		return fmt.Errorf("failed to rebuild indexes: %w", err)
	}

	// Save to disk
	if err := cm.saveToDisk(); err != nil {
		return fmt.Errorf("failed to save cache to disk: %w", err)
	}

	return nil
}

// SaveCache saves the current cache to disk
func (cm *CacheManager) SaveCache() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if !cm.initialized {
		return fmt.Errorf("cache manager not initialized")
	}

	return cm.saveToDisk()
}

// GetCacheStats returns statistics about the cache
func (cm *CacheManager) GetCacheStats() (map[string]any, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if !cm.initialized {
		return nil, fmt.Errorf("cache manager not initialized")
	}

	// Count items across all organizations
	sitesCount := 0
	siteSettingsCount := 0
	rfTemplatesCount := 0
	gwTemplatesCount := 0
	wlanTemplatesCount := 0
	networksCount := 0
	orgWlansCount := 0
	siteWlansCount := 0
	apsCount := 0
	switchesCount := 0
	gatewaysCount := 0
	profilesCount := 0
	profileDetailsCount := 0

	for _, orgData := range cm.cache.Orgs {
		sitesCount += len(orgData.Sites.Info)
		siteSettingsCount += len(orgData.Sites.Settings)
		rfTemplatesCount += len(orgData.Templates.RF)
		gwTemplatesCount += len(orgData.Templates.Gateway)
		wlanTemplatesCount += len(orgData.Templates.WLAN)
		networksCount += len(orgData.Networks)
		orgWlansCount += len(orgData.WLANs.Org)
		siteWlansCount += len(orgData.WLANs.Sites)
		apsCount += len(orgData.Inventory.AP)
		switchesCount += len(orgData.Inventory.Switch)
		gatewaysCount += len(orgData.Inventory.Gateway)
		profilesCount += len(orgData.Profiles.Devices)
		profileDetailsCount += len(orgData.Profiles.Details)
	}

	stats := map[string]any{
		"version":               cm.cache.Version,
		"org_count":             len(cm.cache.Orgs),
		"sites_count":           sitesCount,
		"site_settings_count":   siteSettingsCount,
		"rf_templates_count":    rfTemplatesCount,
		"gw_templates_count":    gwTemplatesCount,
		"wlan_templates_count":  wlanTemplatesCount,
		"networks_count":        networksCount,
		"org_wlans_count":       orgWlansCount,
		"site_wlans_count":      siteWlansCount,
		"aps_count":             apsCount,
		"switches_count":        switchesCount,
		"gateways_count":        gatewaysCount,
		"profiles_count":        profilesCount,
		"profile_details_count": profileDetailsCount,
		"indexes_built":         cm.indexes != nil,
	}

	return stats, nil
}

// isCacheExpired checks if the cache has expired based on the configured TTL
//
//nolint:unused // Used in tests, will be integrated in future cache refresh logic
func (cm *CacheManager) isCacheExpired(fileInfo os.FileInfo) bool {
	cacheTTL := viper.GetInt("files.cache_ttl")
	if cacheTTL <= 0 {
		return false
	}

	// Check if metadata file exists to get more accurate last modified time
	if metaInfo, err := os.Stat(cm.metaPath); err == nil {
		fileInfo = metaInfo
	}

	// Calculate expiration time
	lastModified := fileInfo.ModTime().UTC()
	expirationTime := lastModified.Add(time.Duration(cacheTTL) * time.Second)
	currentTime := time.Now().UTC()

	return currentTime.After(expirationTime) || currentTime.Equal(expirationTime)
}

// loadFromDisk loads the cache from the JSON file
func (cm *CacheManager) loadFromDisk() error {
	// Verify cache integrity first
	if err := cm.verifyCacheIntegrity(); err != nil {
		return fmt.Errorf("cache integrity check failed: %w", err)
	}

	// Read cache file
	data, err := os.ReadFile(cm.cachePath)
	if err != nil {
		return fmt.Errorf("failed to read cache file: %w", err)
	}

	// Parse JSON
	cache := &Cache{}
	if err := json.Unmarshal(data, cache); err != nil {
		return fmt.Errorf("failed to parse cache JSON: %w", err)
	}

	// Validate cache version
	if cache.Version < 1 {
		return fmt.Errorf("cache version %d is not supported, expected 1+", cache.Version)
	}

	cm.cache = cache
	return nil
}

// saveToDisk saves the cache to disk with atomic write
func (cm *CacheManager) saveToDisk() error {
	// Ensure directory exists
	dir := filepath.Dir(cm.cachePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Create backup of existing cache
	if _, err := os.Stat(cm.cachePath); err == nil {
		if err := helpers.CopyFile(cm.cachePath, cm.backupPath); err != nil {
			return fmt.Errorf("failed to create cache backup: %w", err)
		}
	}

	// Set cache version
	cm.cache.Version = 1

	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(cm.cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache to JSON: %w", err)
	}

	// Write to temporary file first
	tempPath := cm.cachePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary cache file: %w", err)
	}

	// Atomic move
	if err := os.Rename(tempPath, cm.cachePath); err != nil {
		return fmt.Errorf("failed to move cache file into place: %w", err)
	}

	// Create metadata file for integrity checking
	if err := cm.createMetadataFile(); err != nil {
		return fmt.Errorf("failed to create metadata file: %w", err)
	}

	// Remove backup on successful save
	_ = os.Remove(cm.backupPath)

	return nil
}

// verifyCacheIntegrity checks cache file integrity using metadata
func (cm *CacheManager) verifyCacheIntegrity() error {
	// Check if both cache and meta files exist
	if _, err := os.Stat(cm.cachePath); os.IsNotExist(err) {
		return fmt.Errorf("cache file does not exist")
	}

	if _, err := os.Stat(cm.metaPath); os.IsNotExist(err) {
		// Meta file doesn't exist, skip integrity check
		return nil
	}

	// Read and verify using existing utils
	return helpers.VerifyFileIntegrity(cm.cachePath, cm.metaPath)
}

// createMetadataFile creates integrity metadata for the cache file
func (cm *CacheManager) createMetadataFile() error {
	return helpers.CreateFileMetadata(cm.cachePath, cm.metaPath, "enhanced_cache")
}

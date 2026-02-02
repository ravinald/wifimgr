package api

import (
	"fmt"

	"github.com/ravinald/wifimgr/internal/logging"
)

// NewCache creates a new unified cache instance
// This is the main entry point for all cache operations
func NewCache(cachePath string, orgID string) (Cacher, error) {
	if cachePath == "" {
		return nil, fmt.Errorf("cache path cannot be empty")
	}
	if orgID == "" {
		return nil, fmt.Errorf("org ID cannot be empty")
	}

	logging.Debugf("Creating unified cache at: %s for org: %s", cachePath, orgID)

	manager := NewCacheManager(cachePath)

	// Try to initialize from existing cache
	if err := manager.Initialize(); err != nil {
		logging.Debugf("Failed to initialize from existing cache: %v, creating new cache", err)

		// Create new cache structure
		cache := &Cache{
			Version: 1,
			Orgs:    make(map[string]*OrgData),
		}

		// Initialize org data
		cache.Orgs[orgID] = createEmptyOrgData()

		// Replace the cache in the manager
		if err := manager.ReplaceCache(cache); err != nil {
			return nil, fmt.Errorf("failed to replace cache: %w", err)
		}

		// Save the new cache
		if err := manager.SaveCache(); err != nil {
			logging.Warnf("Failed to save initial cache: %v", err)
		}
	}

	return &UnifiedCache{
		CacheManager: manager,
		orgID:        orgID,
	}, nil
}

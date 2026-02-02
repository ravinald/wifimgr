package api

import (
	"context"
)

// Local Cache
// ============================================================================

// GetCacheAccessor returns a mock cache accessor
func (m *MockClient) GetCacheAccessor() CacheAccessor {
	// Return a mock cache accessor for testing
	return NewCacheAccessor("./test_cache.json")
}

// ForceRebuildCache forces a rebuild of the cache
func (m *MockClient) ForceRebuildCache(_ context.Context) error {
	// Mock implementation - just return success
	return nil
}

// UpdateCacheForTypes updates the cache for specific device types
func (m *MockClient) UpdateCacheForTypes(_ context.Context, _ []string, _ []string) error {
	// Mock implementation - just return success
	return nil
}

// PopulateDeviceCacheForSite populates the device cache for a specific site
func (m *MockClient) PopulateDeviceCacheForSite(_ context.Context, _ string, _ string) error {
	// Mock implementation - just return success
	return nil
}

// GetDeviceCache returns the device cache instance
func (m *MockClient) GetDeviceCache() *DeviceCache {
	// Initialize device cache if needed
	if deviceCache == nil {
		InitializeDeviceCache()
	}
	return deviceCache
}

// GetConfigDirectory returns the configuration directory
func (m *MockClient) GetConfigDirectory() string {
	return "./config"
}

// GetSchemaDirectory returns the schema directory
func (m *MockClient) GetSchemaDirectory() string {
	return "./config/schemas"
}

// GetInventoryConfig retrieves the inventory configuration from the specified path
func (m *MockClient) GetInventoryConfig(_ string) (*InventoryConfig, error) {
	// For mock client, just return a simple inventory config
	config := &InventoryConfig{
		Version: 1,
		Config: struct {
			Inventory struct {
				AP      []string `json:"ap"`
				Switch  []string `json:"switch"`
				Gateway []string `json:"gateway"`
			} `json:"inventory"`
		}{
			Inventory: struct {
				AP      []string `json:"ap"`
				Switch  []string `json:"switch"`
				Gateway []string `json:"gateway"`
			}{
				AP:      []string{},
				Switch:  []string{},
				Gateway: []string{},
			},
		},
	}

	return config, nil
}

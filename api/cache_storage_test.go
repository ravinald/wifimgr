package api

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheManager_isCacheExpired(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cache_test")
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	cachePath := filepath.Join(tempDir, "test-cache.json")
	manager := NewCacheManager(cachePath)

	// Create a test file
	testFile := filepath.Join(tempDir, "test-file.txt")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	// Test 1: No TTL configured (cache_ttl = 0)
	viper.Set("files.cache_ttl", 0)
	fileInfo, err := os.Stat(testFile)
	require.NoError(t, err)
	assert.False(t, manager.isCacheExpired(fileInfo), "Cache should not be expired when TTL is 0")

	// Test 2: TTL configured but file is fresh (cache_ttl = 3600 seconds)
	viper.Set("files.cache_ttl", 3600) // 1 hour
	assert.False(t, manager.isCacheExpired(fileInfo), "Cache should not be expired for fresh file")

	// Test 3: TTL configured and file is old
	// Modify the file's modification time to be older than TTL
	oldTime := time.Now().Add(-2 * time.Hour)
	err = os.Chtimes(testFile, oldTime, oldTime)
	require.NoError(t, err)

	fileInfo, err = os.Stat(testFile)
	require.NoError(t, err)
	assert.True(t, manager.isCacheExpired(fileInfo), "Cache should be expired for old file")

	// Test 4: Negative TTL (disabled)
	viper.Set("files.cache_ttl", -1)
	assert.False(t, manager.isCacheExpired(fileInfo), "Cache should not be expired when TTL is negative")
}

func TestCacheManager_InitializeWithTTL(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cache_ttl_test")
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	cachePath := filepath.Join(tempDir, "test-cache.json")

	// Test 1: Create a fresh cache
	viper.Set("files.cache_ttl", 60) // 60 seconds TTL

	manager1 := NewCacheManager(cachePath)
	err = manager1.InitializeWithOptions(false)
	require.NoError(t, err)

	// Save some data to cache
	cache := manager1.newEmptyCache()
	err = manager1.ReplaceCache(cache)
	require.NoError(t, err)

	// Verify cache file exists
	_, err = os.Stat(cachePath)
	require.NoError(t, err)

	// Test 2: Initialize again - should load from disk (not expired)
	manager2 := NewCacheManager(cachePath)
	err = manager2.InitializeWithOptions(false)
	require.NoError(t, err)
	assert.True(t, manager2.initialized, "Manager should be initialized")

	// Test 3: Set TTL to 1 second and wait for expiration
	viper.Set("files.cache_ttl", 1)

	// Modify cache file time to be older than TTL
	oldTime := time.Now().Add(-2 * time.Second)
	err = os.Chtimes(cachePath, oldTime, oldTime)
	require.NoError(t, err)

	// Also modify metadata file if it exists
	metaPath := filepath.Join(tempDir, ".test-cache.json.meta")
	if _, err := os.Stat(metaPath); err == nil {
		err = os.Chtimes(metaPath, oldTime, oldTime)
		require.NoError(t, err)
	}

	// Initialize again - should create empty cache due to expiration
	manager3 := NewCacheManager(cachePath)
	err = manager3.InitializeWithOptions(false)
	require.NoError(t, err)
	assert.True(t, manager3.initialized, "Manager should be initialized")

	// Verify it's an empty cache (no orgs)
	assert.Equal(t, 0, len(manager3.cache.Orgs), "Cache should be empty after expiration")
}

func TestCacheManager_ExpiryLoggingFormat(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cache_expiry_logging_test")
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	cachePath := filepath.Join(tempDir, "test-cache.json")

	// Create initial cache with a specific TTL
	viper.Set("files.cache_ttl", 3600) // 1 hour TTL

	manager1 := NewCacheManager(cachePath)
	err = manager1.InitializeWithOptions(false)
	require.NoError(t, err)

	// Save some data to cache
	cache := manager1.newEmptyCache()
	err = manager1.ReplaceCache(cache)
	require.NoError(t, err)

	// Set cache file time to be 2 hours old (1 hour past TTL)
	oldTime := time.Now().Add(-2 * time.Hour)
	err = os.Chtimes(cachePath, oldTime, oldTime)
	require.NoError(t, err)

	// Also modify metadata file if it exists
	metaPath := filepath.Join(tempDir, ".test-cache.json.meta")
	if _, err := os.Stat(metaPath); err == nil {
		err = os.Chtimes(metaPath, oldTime, oldTime)
		require.NoError(t, err)
	}

	// Initialize again - should log expiration with "1h0m0s" format
	manager2 := NewCacheManager(cachePath)
	err = manager2.InitializeWithOptions(false)
	require.NoError(t, err)
	assert.True(t, manager2.initialized, "Manager should be initialized")
	assert.Equal(t, 0, len(manager2.cache.Orgs), "Cache should be empty after expiration")
}

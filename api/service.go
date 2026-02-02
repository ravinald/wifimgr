package api

import (
	"context"
	"sync"
)

// clientInstance is the global client instance
var (
	clientInstance Client
	clientMutex    sync.RWMutex
)

// SetClient sets the global client instance
func SetClient(client Client) {
	clientMutex.Lock()
	defer clientMutex.Unlock()
	clientInstance = client
}

// GetClient returns the global client instance
func GetClient() Client {
	clientMutex.RLock()
	defer clientMutex.RUnlock()
	return clientInstance
}

// GetSiteNameByID retrieves a site name from the global client cache
func GetSiteNameByID(siteID string) (string, bool) {
	client := GetClient()
	if client == nil {
		return "", false
	}

	return client.GetSiteName(siteID)
}

// EnsureSiteCache ensures that the site cache is populated for the given org ID
func EnsureSiteCache(ctx context.Context, orgID string) error {
	client := GetClient()
	if client == nil {
		return nil // No client available
	}

	// GetSites will populate the cache if needed
	_, err := client.GetSites(ctx, orgID)
	return err
}

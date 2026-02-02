package api

import (
	"sync"
	"time"
)

// cacheItem represents an item in the cache with expiration
type cacheItem[T any] struct {
	data    T
	expires time.Time
}

// cache provides a simple generic cache with TTL
type cache[T any] struct {
	items     map[string]cacheItem[T]
	mutex     sync.RWMutex
	ttl       time.Duration
	siteNames map[string]string // Map of site IDs to site names
}

// newCache creates a new cache with the specified TTL
func newCache[T any](ttl time.Duration) *cache[T] {
	return &cache[T]{
		items:     make(map[string]cacheItem[T]),
		siteNames: make(map[string]string),
		ttl:       ttl,
	}
}

// Set adds or updates an item in the cache
func (c *cache[T]) Set(key string, data T) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.items[key] = cacheItem[T]{
		data:    data,
		expires: time.Now().Add(c.ttl),
	}
}

// Get retrieves an item from the cache if it exists and is not expired
func (c *cache[T]) Get(key string) (T, bool) {
	c.mutex.RLock()
	item, found := c.items[key]
	c.mutex.RUnlock()

	if !found {
		var zero T
		return zero, false
	}

	// Check if the item has expired
	if time.Now().After(item.expires) {
		c.mutex.Lock()
		delete(c.items, key)
		c.mutex.Unlock()
		var zero T
		return zero, false
	}

	return item.data, true
}

// Clear removes all items from the cache
func (c *cache[T]) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.items = make(map[string]cacheItem[T])
}

// Delete removes an item from the cache
func (c *cache[T]) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.items, key)
}

// GetSiteName retrieves a site name by its ID from the cache
func (c *cache[T]) GetSiteName(siteID string) (string, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	name, found := c.siteNames[siteID]
	return name, found
}

// SetSiteName adds a site name to the cache, indexed by site ID
func (c *cache[T]) SetSiteName(siteID, siteName string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.siteNames[siteID] = siteName
}

// BuildSiteNameCache builds a map of site IDs to site names from a slice of sites
func (c *cache[T]) BuildSiteNameCache(sites []Site) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Clear existing site names
	c.siteNames = make(map[string]string)

	// Add each site to the map
	for _, site := range sites {
		if site.Id != nil && site.Name != nil {
			c.siteNames[string(*site.Id)] = *site.Name
		}
	}
}

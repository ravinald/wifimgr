package api

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// mockRateLimiter wraps a rate limiter for API requests
type mockRateLimiter struct {
	limiter *rate.Limiter
}

// newMockRateLimiter creates a new rate limiter
func newMockRateLimiter(limit int, duration time.Duration) *mockRateLimiter {
	return &mockRateLimiter{
		limiter: rate.NewLimiter(rate.Every(duration/time.Duration(limit)), limit),
	}
}

// Wait blocks until the rate limiter allows a request
func (r *mockRateLimiter) Wait(ctx context.Context) error {
	return r.limiter.Wait(ctx)
}

// mockCache is a generic cache implementation with expiration
type mockCache[T any] struct {
	data       T
	expiration time.Time
	mu         sync.RWMutex
}

// newMockCache creates a new cache instance
func newMockCache[T any](ttl time.Duration) *mockCache[T] {
	return &mockCache[T]{
		expiration: time.Now().Add(ttl),
	}
}

// Get retrieves data from the cache
func (c *mockCache[T]) Get() (T, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if time.Now().After(c.expiration) {
		var zero T
		return zero, false
	}

	return c.data, true
}

// Set stores data in the cache
func (c *mockCache[T]) Set(data T, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = data
	c.expiration = time.Now().Add(ttl)
}

// Clear clears the cache
func (c *mockCache[T]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	var zero T
	c.data = zero
	c.expiration = time.Time{}
}

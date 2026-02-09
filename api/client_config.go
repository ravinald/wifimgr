package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ravinald/wifimgr/internal/logging"
)

// Configuration-related methods for the mistClient

// SetRateLimit sets the rate limiting configuration
func (c *mistClient) SetRateLimit(limit int, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if limit > 0 {
		c.rateLimiter = newRateLimiter(limit, duration)
		c.config.RateLimit = limit
		c.config.RateDuration = duration
		logging.Debugf("Rate limit set to %d requests per %v", limit, duration)
	} else {
		c.rateLimiter = nil
		c.config.RateLimit = 0
		logging.Debug("Rate limiting disabled")
	}
}

// SetResultsLimit sets the maximum results per API call
func (c *mistClient) SetResultsLimit(limit int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.config.ResultsLimit = limit
	logging.Debugf("Results limit set to %d", limit)
}

// SetDebug enables or disables debug mode
func (c *mistClient) SetDebug(debug bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.debug = debug
	c.config.Debug = debug

	// Setup debug transport if debug is enabled
	setupDebugTransport(c)

	if debug {
		logging.Debug("Debug mode enabled")
	} else {
		logging.Debug("Debug mode disabled")
	}
}

// GetCacheAccessor is deprecated and has been removed.
// Use vendors.GetGlobalCacheAccessor() instead for cache lookups.

// ValidateAPIToken validates the API token by making a self request
func (c *mistClient) ValidateAPIToken(ctx context.Context) (*SelfResponse, error) {
	var self SelfResponse

	// Make the API request to the /api/v1/self endpoint
	err := c.do(ctx, http.MethodGet, "/self", nil, &self)
	if err != nil {
		// Check for specific error types
		if statusErr, ok := err.(interface{ StatusCode() int }); ok && statusErr.StatusCode() == http.StatusUnauthorized {
			return nil, ErrUnauthorized
		}
		return nil, fmt.Errorf("failed to validate API token: %w", err)
	}

	// If we get here, the request was successful
	return &self, nil
}

// GetAPIUserInfo retrieves information about the authenticated user
func (c *mistClient) GetAPIUserInfo(ctx context.Context) (*SelfResponse, error) {
	return c.ValidateAPIToken(ctx)
}

// formatError formats an error message with additional details
func formatError(message string, err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s: %w", message, err)
}

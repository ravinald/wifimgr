package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"

	"github.com/ravinald/wifimgr/internal/logging"
)

// CacheVerificationStatus represents the status of cache verification
type CacheVerificationStatus int

const (
	CacheOK CacheVerificationStatus = iota
	CacheCorrupted
	CacheOutOfDate
	CacheFailed
)

// Global variables for tracking cache integrity
var (
	cacheIntegrityCompromised bool
)

// IsCacheIntegrityCompromised returns whether cache integrity is compromised
func IsCacheIntegrityCompromised() bool {
	return cacheIntegrityCompromised
}

// VerifyCacheIntegrity verifies the integrity of the cache system with retry logic
// This ensures proper initialization timing and prevents false negatives
func VerifyCacheIntegrity(_ context.Context, client Client, orgID string) (CacheVerificationStatus, error) {
	logging.Debugf("Verifying cache integrity for organization %s", orgID)

	if orgID == "" {
		return CacheFailed, fmt.Errorf("organization ID not provided")
	}

	// Simple verification: try to get the cache accessor
	cacheAccessor := client.GetCacheAccessor()
	if cacheAccessor == nil {
		logging.Warnf("Cache verification error: cache accessor is not available for organization %s", orgID)
		cacheIntegrityCompromised = true
		return CacheCorrupted, fmt.Errorf("cache accessor not available")
	}

	// Check if cache is initialized with retry logic to handle timing issues
	status, err := verifyWithRetry(cacheAccessor, orgID)
	if err != nil {
		cacheIntegrityCompromised = true
		return status, err
	}

	// Cache appears to be working
	cacheIntegrityCompromised = false
	logging.Debugf("Cache integrity verification passed for organization %s", orgID)
	return CacheOK, nil
}

// verifyWithRetry implements retry logic for cache initialization checks
func verifyWithRetry(cacheAccessor CacheAccessor, orgID string) (CacheVerificationStatus, error) {
	const maxRetries = 3

	retryDelay := 100 * time.Millisecond

	for attempt := 1; attempt <= maxRetries; attempt++ {
		logging.Debugf("Cache verification attempt %d/%d for organization %s", attempt, maxRetries, orgID)

		// Check if cache is already initialized
		if cacheAccessor.IsInitialized() {
			logging.Debugf("Cache is initialized on attempt %d for organization %s", attempt, orgID)
			return CacheOK, nil
		}

		// Cache should already be initialized by the singleton
		// Just wait a bit and check again if not initialized yet
		if attempt < maxRetries {
			logging.Debugf("Cache not initialized yet, waiting %v before retry (attempt %d/%d)", retryDelay, attempt, maxRetries)
			time.Sleep(retryDelay)
			retryDelay *= 2 // exponential backoff
		}
	}

	// This should not be reached, but added for completeness
	return CacheOutOfDate, fmt.Errorf("cache verification failed after %d attempts", maxRetries)
}

// WrapOutputWithWarning prefixes each line with a warning indicator if cache integrity is compromised
func WrapOutputWithWarning(output string) string {
	if !cacheIntegrityCompromised {
		return output
	}

	// Create the warning symbol (red asterisk)
	warningSymbol := color.New(color.FgRed, color.Bold).Sprint("*")

	// Process each line
	lines := []string{}
	for _, line := range strings.Split(output, "\n") {
		if line != "" {
			// Add warning symbol to non-empty lines
			lines = append(lines, fmt.Sprintf("%s %s", warningSymbol, line))
		} else {
			// Keep empty lines as is
			lines = append(lines, line)
		}
	}

	return strings.Join(lines, "\n")
}

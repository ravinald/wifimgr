package meraki

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ravinald/wifimgr/internal/logging"
)

// RateLimiter implements a token bucket rate limiter for Meraki API calls.
// Meraki allows 10 requests/second with a burst of 10 additional requests.
type RateLimiter struct {
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter configured for Meraki's limits.
// requestsPerSecond: base rate (default 10)
// burstSize: additional burst capacity (default 10)
func NewRateLimiter(requestsPerSecond int, burstSize int) *RateLimiter {
	if requestsPerSecond <= 0 {
		requestsPerSecond = 10
	}
	if requestsPerSecond > 10 {
		requestsPerSecond = 10 // Meraki max
	}
	if burstSize <= 0 {
		burstSize = 10 // Meraki burst allowance
	}

	return &RateLimiter{
		tokens:     float64(requestsPerSecond + burstSize), // Start with full bucket + burst
		maxTokens:  float64(requestsPerSecond + burstSize),
		refillRate: float64(requestsPerSecond),
		lastRefill: time.Now(),
	}
}

// refill adds tokens based on time elapsed since last refill.
func (r *RateLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(r.lastRefill).Seconds()
	r.tokens += elapsed * r.refillRate
	if r.tokens > r.maxTokens {
		r.tokens = r.maxTokens
	}
	r.lastRefill = now
}

// Acquire blocks until a token is available or context is cancelled.
// Returns nil when a token is acquired, or the context error if cancelled.
func (r *RateLimiter) Acquire(ctx context.Context) error {
	for {
		r.mu.Lock()
		r.refill()

		if r.tokens >= 1 {
			r.tokens--
			r.mu.Unlock()
			return nil
		}

		// Calculate wait time for next token
		waitTime := time.Duration((1-r.tokens)/r.refillRate*1000) * time.Millisecond
		if waitTime < 10*time.Millisecond {
			waitTime = 10 * time.Millisecond
		}
		r.mu.Unlock()

		logging.Debugf("[meraki] Rate limit: waiting %v for token", waitTime)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			// Continue to try acquiring
		}
	}
}

// RetryConfig holds configuration for retry behavior.
type RetryConfig struct {
	MaxRetries  int
	BaseBackoff time.Duration
	MaxBackoff  time.Duration
	RetryOn429  bool
	RateLimiter *RateLimiter
}

// DefaultRetryConfig returns the default retry configuration for Meraki.
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:  3,
		BaseBackoff: 1 * time.Second,
		MaxBackoff:  30 * time.Second,
		RetryOn429:  true,
	}
}

// ParseRetryAfter extracts the retry delay from a Retry-After header.
// Returns the duration to wait, or 0 if not present or unparseable.
func ParseRetryAfter(resp *http.Response) time.Duration {
	if resp == nil {
		return 0
	}

	retryAfter := resp.Header.Get("Retry-After")
	if retryAfter == "" {
		return 0
	}

	// Try parsing as seconds (integer)
	if seconds, err := strconv.Atoi(retryAfter); err == nil {
		return time.Duration(seconds) * time.Second
	}

	// Try parsing as HTTP-date
	if t, err := http.ParseTime(retryAfter); err == nil {
		delay := time.Until(t)
		if delay > 0 {
			return delay
		}
	}

	return 0
}

// Is429Error checks if the error indicates a rate limit (429) response.
// This works with the Meraki SDK error format.
func Is429Error(err error) bool {
	if err == nil {
		return false
	}
	// The Meraki SDK returns errors with status codes in the error message
	errStr := err.Error()
	return contains(errStr, "429") || contains(errStr, "Too Many Requests")
}

// contains checks if a string contains a substring (simple helper).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// RetryState tracks the state of a retry operation.
type RetryState struct {
	Attempt   int
	LastError error
	Config    *RetryConfig
}

// NewRetryState creates a new retry state with the given config.
func NewRetryState(config *RetryConfig) *RetryState {
	if config == nil {
		config = DefaultRetryConfig()
	}
	return &RetryState{
		Attempt: 0,
		Config:  config,
	}
}

// ShouldRetry determines if the operation should be retried.
func (s *RetryState) ShouldRetry(err error) bool {
	if err == nil {
		return false
	}

	// Check if we've exceeded max retries
	if s.Attempt >= s.Config.MaxRetries {
		return false
	}

	// Check if it's a 429 error and we should retry on 429
	if s.Config.RetryOn429 && Is429Error(err) {
		return true
	}

	return false
}

// WaitBeforeRetry waits the appropriate amount of time before retrying.
// Uses Retry-After header if available, otherwise exponential backoff.
func (s *RetryState) WaitBeforeRetry(ctx context.Context, resp *http.Response) error {
	s.Attempt++

	// Try to get delay from Retry-After header
	delay := ParseRetryAfter(resp)

	if delay == 0 {
		// Use exponential backoff
		delay = s.Config.BaseBackoff * time.Duration(1<<uint(s.Attempt-1))
		if delay > s.Config.MaxBackoff {
			delay = s.Config.MaxBackoff
		}
	}

	logging.Infof("[meraki] Rate limited (attempt %d/%d), waiting %v before retry",
		s.Attempt, s.Config.MaxRetries, delay)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(delay):
		return nil
	}
}

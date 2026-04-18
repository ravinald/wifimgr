package meraki

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/vendors"
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
//
// Deprecated: use errors.As against *vendors.RateLimitError instead. This
// shim remains for call sites that haven't migrated yet and for the
// narrow case of legacy error strings that predate ClassifyError.
func Is429Error(err error) bool {
	if err == nil {
		return false
	}
	var rl *vendors.RateLimitError
	if errors.As(err, &rl) {
		return true
	}
	// Legacy string-match fallback for errors that haven't gone through
	// ClassifyError yet.
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
//
// Uses errors.As against the wifimgr error taxonomy so that retries are
// driven by the *type* of failure rather than by string matching. Authn
// errors and non-retryable transport errors return false immediately —
// retrying a 401 or a 400 wastes the budget and mistakes the signal.
func (s *RetryState) ShouldRetry(err error) bool {
	if err == nil {
		return false
	}
	if s.Attempt >= s.Config.MaxRetries {
		return false
	}

	// Non-retryable: authN/authZ, explicit non-retryable transport, 4xx
	// NotFoundError. Return without consuming a retry.
	var authErr *vendors.AuthError
	if errors.As(err, &authErr) {
		return false
	}
	var notFound *vendors.NotFoundError
	if errors.As(err, &notFound) {
		return false
	}
	var tErr *vendors.TransportError
	if errors.As(err, &tErr) && !tErr.Retryable {
		return false
	}

	// Retryable: 429 (always, if configured), 5xx, retryable transport.
	var rl *vendors.RateLimitError
	if s.Config.RetryOn429 && errors.As(err, &rl) {
		return true
	}
	var sErr *vendors.ServerError
	if errors.As(err, &sErr) {
		return true
	}
	if errors.As(err, &tErr) && tErr.Retryable {
		return true
	}

	// Backward-compat: errors that haven't been classified yet (raw SDK
	// errors from un-migrated call sites). Fall back to the legacy string
	// match so we don't regress the pre-taxonomy behaviour.
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

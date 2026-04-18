package ubiquiti

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ravinald/wifimgr/internal/logging"
)

// RateLimiter implements a token bucket rate limiter for Ubiquiti Site Manager API.
// Ubiquiti allows ~10,000 requests/minute (~166 req/sec).
type RateLimiter struct {
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter for Ubiquiti's limits.
// requestsPerSecond: base rate (default 166 for ~10,000 req/min)
// burstSize: additional burst capacity (default 50)
func NewRateLimiter(requestsPerSecond int, burstSize int) *RateLimiter {
	if requestsPerSecond <= 0 {
		requestsPerSecond = 166
	}
	if burstSize <= 0 {
		burstSize = 50
	}

	return &RateLimiter{
		tokens:     float64(requestsPerSecond + burstSize),
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
func (r *RateLimiter) Acquire(ctx context.Context) error {
	for {
		r.mu.Lock()
		r.refill()

		if r.tokens >= 1 {
			r.tokens--
			r.mu.Unlock()
			return nil
		}

		waitTime := time.Duration((1-r.tokens)/r.refillRate*1000) * time.Millisecond
		if waitTime < 10*time.Millisecond {
			waitTime = 10 * time.Millisecond
		}
		r.mu.Unlock()

		logging.Debugf("[ubiquiti] Rate limit: waiting %v for token", waitTime)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
		}
	}
}

// RetryConfig holds configuration for retry behavior.
type RetryConfig struct {
	MaxRetries  int
	BaseBackoff time.Duration
	MaxBackoff  time.Duration
}

// DefaultRetryConfig returns the default retry configuration for Ubiquiti.
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:  3,
		BaseBackoff: 1 * time.Second,
		MaxBackoff:  30 * time.Second,
	}
}

// RetryState tracks the state of a retry operation.
type RetryState struct {
	Attempt int
	Config  *RetryConfig
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

// ShouldRetry determines if the operation should be retried based on HTTP status.
func (s *RetryState) ShouldRetry(statusCode int) bool {
	if s.Attempt >= s.Config.MaxRetries {
		return false
	}
	return statusCode == http.StatusTooManyRequests
}

// WaitBeforeRetry waits the appropriate amount of time before retrying.
// Uses Retry-After header if available, otherwise exponential backoff.
func (s *RetryState) WaitBeforeRetry(ctx context.Context, resp *http.Response) error {
	s.Attempt++

	delay := parseRetryAfter(resp)
	if delay == 0 {
		delay = s.Config.BaseBackoff * time.Duration(1<<uint(s.Attempt-1))
		if delay > s.Config.MaxBackoff {
			delay = s.Config.MaxBackoff
		}
	}

	logging.Infof("[ubiquiti] Rate limited (attempt %d/%d), waiting %v before retry",
		s.Attempt, s.Config.MaxRetries, delay)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(delay):
		return nil
	}
}

// parseRetryAfter extracts the retry delay from a Retry-After header.
func parseRetryAfter(resp *http.Response) time.Duration {
	if resp == nil {
		return 0
	}

	retryAfter := resp.Header.Get("Retry-After")
	if retryAfter == "" {
		return 0
	}

	if seconds, err := strconv.Atoi(retryAfter); err == nil {
		return time.Duration(seconds) * time.Second
	}

	if t, err := http.ParseTime(retryAfter); err == nil {
		delay := time.Until(t)
		if delay > 0 {
			return delay
		}
	}

	return 0
}

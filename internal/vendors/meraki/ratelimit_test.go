package meraki

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestNewRateLimiter(t *testing.T) {
	tests := []struct {
		name              string
		requestsPerSecond int
		burstSize         int
		wantMaxTokens     float64
		wantRefillRate    float64
	}{
		{
			name:              "default values for zero input",
			requestsPerSecond: 0,
			burstSize:         0,
			wantMaxTokens:     20, // 10 + 10
			wantRefillRate:    10,
		},
		{
			name:              "caps at 10 requests per second",
			requestsPerSecond: 20,
			burstSize:         10,
			wantMaxTokens:     20, // 10 + 10
			wantRefillRate:    10,
		},
		{
			name:              "uses provided burst size",
			requestsPerSecond: 5,
			burstSize:         15,
			wantMaxTokens:     20, // 5 + 15
			wantRefillRate:    5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := NewRateLimiter(tt.requestsPerSecond, tt.burstSize)

			if rl.maxTokens != tt.wantMaxTokens {
				t.Errorf("maxTokens = %v, want %v", rl.maxTokens, tt.wantMaxTokens)
			}
			if rl.refillRate != tt.wantRefillRate {
				t.Errorf("refillRate = %v, want %v", rl.refillRate, tt.wantRefillRate)
			}
		})
	}
}

func TestRateLimiter_Acquire(t *testing.T) {
	// Create a rate limiter with 10 tokens and 10 burst (20 total)
	rl := NewRateLimiter(10, 10)

	ctx := context.Background()

	// Should be able to acquire 20 tokens immediately (base + burst)
	for i := 0; i < 20; i++ {
		if err := rl.Acquire(ctx); err != nil {
			t.Errorf("Acquire() error on token %d: %v", i+1, err)
		}
	}

	// Verify tokens are exhausted - set tokens to 0 explicitly
	rl.mu.Lock()
	rl.tokens = 0
	rl.lastRefill = time.Now()
	rl.mu.Unlock()

	// The next acquire should need to wait (or we can test with a timeout context)
	shortCtx, cancel := context.WithTimeout(ctx, 5*time.Millisecond)
	defer cancel()

	err := rl.Acquire(shortCtx)
	if err != context.DeadlineExceeded {
		t.Errorf("Expected DeadlineExceeded after exhausting tokens, got: %v", err)
	}
}

func TestRateLimiter_Refill(t *testing.T) {
	// Create a rate limiter with small refill for faster testing
	rl := NewRateLimiter(10, 0)

	ctx := context.Background()

	// Exhaust all tokens
	for i := 0; i < 10; i++ {
		_ = rl.Acquire(ctx)
	}

	// Wait for some tokens to refill (100ms = 1 token at 10/sec)
	time.Sleep(150 * time.Millisecond)

	// Should be able to acquire at least 1 token now
	shortCtx, cancel := context.WithTimeout(ctx, 5*time.Millisecond)
	defer cancel()

	if err := rl.Acquire(shortCtx); err != nil {
		t.Errorf("Expected token to be available after refill, got: %v", err)
	}
}

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		wantSecs int // Expected seconds (approximate)
	}{
		{
			name:     "nil response",
			header:   "",
			wantSecs: 0,
		},
		{
			name:     "integer seconds",
			header:   "5",
			wantSecs: 5,
		},
		{
			name:     "empty header",
			header:   "",
			wantSecs: 0,
		},
		{
			name:     "invalid value",
			header:   "not-a-number",
			wantSecs: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp *http.Response
			if tt.header != "" || tt.name != "nil response" {
				resp = &http.Response{
					Header: http.Header{},
				}
				if tt.header != "" {
					resp.Header.Set("Retry-After", tt.header)
				}
			}

			got := ParseRetryAfter(resp)
			wantDuration := time.Duration(tt.wantSecs) * time.Second

			if got != wantDuration {
				t.Errorf("ParseRetryAfter() = %v, want %v", got, wantDuration)
			}
		})
	}
}

func TestIs429Error(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		want429 bool
	}{
		{
			name:    "nil error",
			err:     nil,
			want429: false,
		},
		{
			name:    "429 in message",
			err:     &testError{msg: "HTTP 429 Too Many Requests"},
			want429: true,
		},
		{
			name:    "Too Many Requests in message",
			err:     &testError{msg: "Error: Too Many Requests"},
			want429: true,
		},
		{
			name:    "other error",
			err:     &testError{msg: "connection refused"},
			want429: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Is429Error(tt.err)
			if got != tt.want429 {
				t.Errorf("Is429Error() = %v, want %v", got, tt.want429)
			}
		})
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestRetryState_ShouldRetry(t *testing.T) {
	config := &RetryConfig{
		MaxRetries:  3,
		BaseBackoff: 100 * time.Millisecond,
		MaxBackoff:  1 * time.Second,
		RetryOn429:  true,
	}

	tests := []struct {
		name       string
		attempt    int
		err        error
		shouldWait bool
	}{
		{
			name:       "no error",
			attempt:    0,
			err:        nil,
			shouldWait: false,
		},
		{
			name:       "429 error first attempt",
			attempt:    0,
			err:        &testError{msg: "HTTP 429"},
			shouldWait: true,
		},
		{
			name:       "429 error at max retries",
			attempt:    3,
			err:        &testError{msg: "HTTP 429"},
			shouldWait: false, // Exceeded max retries
		},
		{
			name:       "non-429 error",
			attempt:    0,
			err:        &testError{msg: "connection refused"},
			shouldWait: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := NewRetryState(config)
			state.Attempt = tt.attempt

			got := state.ShouldRetry(tt.err)
			if got != tt.shouldWait {
				t.Errorf("ShouldRetry() = %v, want %v", got, tt.shouldWait)
			}
		})
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", config.MaxRetries)
	}
	if config.BaseBackoff != 1*time.Second {
		t.Errorf("BaseBackoff = %v, want 1s", config.BaseBackoff)
	}
	if config.MaxBackoff != 30*time.Second {
		t.Errorf("MaxBackoff = %v, want 30s", config.MaxBackoff)
	}
	if !config.RetryOn429 {
		t.Error("RetryOn429 = false, want true")
	}
}

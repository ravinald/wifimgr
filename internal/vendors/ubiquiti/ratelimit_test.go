package ubiquiti

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestNewRateLimiter_Defaults(t *testing.T) {
	rl := NewRateLimiter(0, 0)
	if rl.refillRate != 166 {
		t.Errorf("refillRate = %v, want 166", rl.refillRate)
	}
	if rl.maxTokens != 216 { // 166 + 50
		t.Errorf("maxTokens = %v, want 216", rl.maxTokens)
	}
}

func TestNewRateLimiter_Custom(t *testing.T) {
	rl := NewRateLimiter(100, 20)
	if rl.refillRate != 100 {
		t.Errorf("refillRate = %v, want 100", rl.refillRate)
	}
	if rl.maxTokens != 120 {
		t.Errorf("maxTokens = %v, want 120", rl.maxTokens)
	}
}

func TestRateLimiter_Acquire(t *testing.T) {
	rl := NewRateLimiter(100, 10)
	ctx := context.Background()

	// Should acquire immediately when tokens are available
	err := rl.Acquire(ctx)
	if err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}
}

func TestRateLimiter_AcquireCancelled(t *testing.T) {
	rl := NewRateLimiter(1, 0)
	// Drain all tokens
	rl.tokens = 0
	rl.lastRefill = time.Now()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := rl.Acquire(ctx)
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
}

func TestRetryState_ShouldRetry(t *testing.T) {
	state := NewRetryState(&RetryConfig{
		MaxRetries:  3,
		BaseBackoff: time.Millisecond,
		MaxBackoff:  time.Second,
	})

	// Should retry on 429
	if !state.ShouldRetry(http.StatusTooManyRequests) {
		t.Error("ShouldRetry(429) = false, want true")
	}

	// Should not retry on 500
	if state.ShouldRetry(http.StatusInternalServerError) {
		t.Error("ShouldRetry(500) = true, want false")
	}

	// Should not retry on 200
	if state.ShouldRetry(http.StatusOK) {
		t.Error("ShouldRetry(200) = true, want false")
	}

	// Exhaust retries
	state.Attempt = 3
	if state.ShouldRetry(http.StatusTooManyRequests) {
		t.Error("ShouldRetry(429) after max retries = true, want false")
	}
}

func TestRetryState_WaitBeforeRetry(t *testing.T) {
	state := NewRetryState(&RetryConfig{
		MaxRetries:  3,
		BaseBackoff: time.Millisecond,
		MaxBackoff:  time.Second,
	})

	ctx := context.Background()
	err := state.WaitBeforeRetry(ctx, nil)
	if err != nil {
		t.Fatalf("WaitBeforeRetry failed: %v", err)
	}
	if state.Attempt != 1 {
		t.Errorf("Attempt = %d, want 1", state.Attempt)
	}
}

func TestRetryState_DefaultConfig(t *testing.T) {
	state := NewRetryState(nil)
	if state.Config.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", state.Config.MaxRetries)
	}
	if state.Config.BaseBackoff != time.Second {
		t.Errorf("BaseBackoff = %v, want %v", state.Config.BaseBackoff, time.Second)
	}
	if state.Config.MaxBackoff != 30*time.Second {
		t.Errorf("MaxBackoff = %v, want %v", state.Config.MaxBackoff, 30*time.Second)
	}
}

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		name     string
		resp     *http.Response
		wantZero bool
	}{
		{"nil response", nil, true},
		{"no header", &http.Response{Header: http.Header{}}, true},
		{"seconds value", &http.Response{Header: http.Header{"Retry-After": {"5"}}}, false},
		{"invalid value", &http.Response{Header: http.Header{"Retry-After": {"abc"}}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := parseRetryAfter(tt.resp)
			if tt.wantZero && d != 0 {
				t.Errorf("parseRetryAfter() = %v, want 0", d)
			}
			if !tt.wantZero && d == 0 {
				t.Errorf("parseRetryAfter() = 0, want non-zero")
			}
		})
	}
}

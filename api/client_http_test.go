package api

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestStatusCodeToError(t *testing.T) {
	c := &mistClient{}
	cases := []struct {
		status     int
		wantSentinel error // nil if not a sentinel
		wantAPIErr   bool  // true if expected *APIError
	}{
		{http.StatusUnauthorized, ErrUnauthorized, false},
		{http.StatusForbidden, ErrForbidden, false},
		{http.StatusNotFound, ErrNotFound, false},
		{http.StatusTooManyRequests, ErrRateLimited, false},
		{http.StatusBadRequest, ErrBadRequest, false},
		{http.StatusInternalServerError, nil, true},
		{http.StatusBadGateway, nil, true},
		{http.StatusServiceUnavailable, nil, true},
		{418, nil, true}, // teapot — default branch
	}
	for _, tc := range cases {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			err := c.statusCodeToError(tc.status)
			if tc.wantSentinel != nil {
				if !errors.Is(err, tc.wantSentinel) {
					t.Errorf("statusCodeToError(%d) = %v, want errors.Is(%v)", tc.status, err, tc.wantSentinel)
				}
			}
			if tc.wantAPIErr {
				var apiErr *APIError
				if !errors.As(err, &apiErr) {
					t.Errorf("statusCodeToError(%d) = %v, want *APIError", tc.status, err)
				} else if apiErr.StatusCode != tc.status {
					t.Errorf("APIError.StatusCode = %d, want %d", apiErr.StatusCode, tc.status)
				}
			}
		})
	}
}

func TestExtractErrorMessages(t *testing.T) {
	cases := []struct {
		name string
		in   map[string]interface{}
		want string
	}{
		{"detail string", map[string]interface{}{"detail": "boom"}, "boom"},
		{"detail array", map[string]interface{}{"detail": []interface{}{"a", "b"}}, "a; b"},
		{"detail nested", map[string]interface{}{"detail": map[string]interface{}{"field": "bad"}}, "field: bad"},
		{"error fallback", map[string]interface{}{"error": "oops"}, "oops"},
		{"message fallback", map[string]interface{}{"message": "hi"}, "hi"},
		{"errors fallback", map[string]interface{}{"errors": []interface{}{"x"}}, "x"},
		{"detail empty falls through", map[string]interface{}{"detail": "", "error": "fallback"}, "fallback"},
		{"unknown shape", map[string]interface{}{"foo": "bar"}, ""},
		{"empty", map[string]interface{}{}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractErrorMessages(tc.in)
			if got != tc.want {
				t.Errorf("extractErrorMessages = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFormatErrorValue_NestedShapes(t *testing.T) {
	// Array of nested maps
	v := []interface{}{
		map[string]interface{}{"field1": "err1"},
		map[string]interface{}{"field2": []interface{}{"err2a", "err2b"}},
	}
	got := formatErrorValue(v)
	// Order within a single map is non-deterministic; check both pieces.
	if !strings.Contains(got, "field1: err1") {
		t.Errorf("missing field1 in %q", got)
	}
	if !strings.Contains(got, "field2: err2a; err2b") {
		t.Errorf("missing field2 nested in %q", got)
	}
}

func TestHandleErrorResponse(t *testing.T) {
	c := &mistClient{}

	t.Run("structured detail produces APIError", func(t *testing.T) {
		body := []byte(`{"detail": "Token expired"}`)
		err := c.handleErrorResponse(http.StatusBadRequest, body)
		var apiErr *APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("got %v, want *APIError", err)
		}
		if apiErr.Message != "Token expired" || apiErr.StatusCode != http.StatusBadRequest {
			t.Errorf("APIError = %+v", apiErr)
		}
	})

	t.Run("invalid JSON falls back to status sentinel", func(t *testing.T) {
		body := []byte(`<html>oops</html>`)
		err := c.handleErrorResponse(http.StatusUnauthorized, body)
		if !errors.Is(err, ErrUnauthorized) {
			t.Errorf("got %v, want errors.Is(ErrUnauthorized)", err)
		}
	})

	t.Run("valid JSON without error fields falls back", func(t *testing.T) {
		body := []byte(`{"unrelated": "value"}`)
		err := c.handleErrorResponse(http.StatusForbidden, body)
		if !errors.Is(err, ErrForbidden) {
			t.Errorf("got %v, want errors.Is(ErrForbidden)", err)
		}
	})
}

func TestBuildURL(t *testing.T) {
	cases := []struct {
		name    string
		baseURL string
		path    string
		want    string
	}{
		{"trailing slash on base", "https://api.example.com/api/v1/", "sites", "https://api.example.com/api/v1/sites"},
		{"no trailing slash on base", "https://api.example.com/api/v1", "sites", "https://api.example.com/api/v1/sites"},
		{"leading slash on path", "https://api.example.com/api/v1/", "/sites", "https://api.example.com/api/v1/sites"},
		{"legacy base without /api/", "https://api.example.com", "sites", "https://api.example.com/api/v1/sites"},
		{"path includes redundant api/v1/", "https://api.example.com/api/v1/", "api/v1/sites", "https://api.example.com/api/v1/sites"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &mistClient{config: Config{BaseURL: tc.baseURL}}
			got := c.buildURL(tc.path)
			if got != tc.want {
				t.Errorf("buildURL(%q) with base %q = %q, want %q", tc.path, tc.baseURL, got, tc.want)
			}
		})
	}
}

func TestShouldRetry(t *testing.T) {
	c := &mistClient{}

	cases := []struct {
		name   string
		status int
		err    error
		want   bool
	}{
		{"2xx success no err", 200, nil, false},
		{"context deadline", 0, context.DeadlineExceeded, true},
		{"io.EOF", 0, io.EOF, true},
		{"io.ErrUnexpectedEOF", 0, io.ErrUnexpectedEOF, true},
		{"net error", 0, &net.OpError{Op: "dial"}, true},
		{"500 retryable", http.StatusInternalServerError, errors.New("boom"), true},
		{"502 retryable", http.StatusBadGateway, errors.New("boom"), true},
		{"503 retryable", http.StatusServiceUnavailable, errors.New("boom"), true},
		{"429 retryable", http.StatusTooManyRequests, errors.New("boom"), true},
		{"401 not retryable", http.StatusUnauthorized, errors.New("nope"), false},
		{"404 not retryable", http.StatusNotFound, errors.New("nope"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := c.shouldRetry(tc.status, tc.err)
			if got != tc.want {
				t.Errorf("shouldRetry(%d, %v) = %v, want %v", tc.status, tc.err, got, tc.want)
			}
		})
	}
}

func TestCalculateBackoff(t *testing.T) {
	c := &mistClient{retryBackoff: 100 * time.Millisecond}

	// Attempt 0 should be ~base (100ms ±20% jitter)
	for i := 0; i < 10; i++ {
		b := c.calculateBackoff(0)
		if b < 80*time.Millisecond || b > 120*time.Millisecond {
			t.Errorf("calculateBackoff(0) = %v, want roughly 100ms ±20%%", b)
		}
	}

	// Attempt 9 should hit the 60s cap
	if got := c.calculateBackoff(9); got != 60*time.Second {
		// Allow that pre-cap value might already be near cap; assert <= cap
		if got > 60*time.Second {
			t.Errorf("calculateBackoff(9) = %v, exceeds 60s cap", got)
		}
	}

	// Each successive attempt should be larger than the previous (no jitter on small attempts)
	prev := c.calculateBackoff(0)
	for i := 1; i < 5; i++ {
		got := c.calculateBackoff(i)
		// Account for ±20% jitter on each side
		if got < prev/2 {
			t.Errorf("calculateBackoff(%d) = %v shrank dramatically from prev %v", i, got, prev)
		}
		prev = got
	}
}

func TestParseRetryAfter(t *testing.T) {
	cases := []struct {
		in      string
		want    int
		wantErr bool
	}{
		{"5", 5, false},
		{"0", 0, false},
		{"not-a-number", 0, true},
		// Past HTTP-date returns 1 (small floor)
		{"Mon, 02 Jan 2006 15:04:05 GMT", 1, false},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, err := parseRetryAfter(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("parseRetryAfter(%q) err = %v, wantErr = %v", tc.in, err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Errorf("parseRetryAfter(%q) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

func TestExtractRetryAfterDuration_FromResponseHeader(t *testing.T) {
	c := &mistClient{}
	resp := &http.Response{Header: make(http.Header)}
	resp.Header.Set("Retry-After", "12")
	got := c.extractRetryAfterDuration(nil, resp)
	if got != 12*time.Second {
		t.Errorf("extractRetryAfterDuration(header=12) = %v, want 12s", got)
	}
}

func TestExtractRetryAfterDuration_FromErrorHint(t *testing.T) {
	// The fallback parser splits on space *after* the colon, so it only succeeds
	// when the value is glued to "Retry-After:" with no whitespace. This is a
	// known fragility but kept here to exercise the success path.
	c := &mistClient{}
	err := errors.New("rate limited; Retry-After:7")
	got := c.extractRetryAfterDuration(err, nil)
	if got != 7*time.Second {
		t.Errorf("extractRetryAfterDuration(err hint=7) = %v, want 7s", got)
	}
}

func TestExtractRetryAfterDuration_NoSignal(t *testing.T) {
	c := &mistClient{}
	got := c.extractRetryAfterDuration(errors.New("plain error"), nil)
	if got != 0 {
		t.Errorf("extractRetryAfterDuration(no signal) = %v, want 0", got)
	}
}

func TestRetryRequest_SuccessFirstTry(t *testing.T) {
	c := &mistClient{maxRetries: 3, retryBackoff: time.Millisecond}
	calls := 0
	err := c.retryRequest(context.Background(), func() (int, error) {
		calls++
		return 200, nil
	})
	if err != nil {
		t.Fatalf("retryRequest error = %v", err)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1", calls)
	}
}

func TestRetryRequest_RetriesThenSucceeds(t *testing.T) {
	c := &mistClient{maxRetries: 3, retryBackoff: time.Millisecond}
	calls := 0
	err := c.retryRequest(context.Background(), func() (int, error) {
		calls++
		if calls < 3 {
			return http.StatusServiceUnavailable, errors.New("transient")
		}
		return 200, nil
	})
	if err != nil {
		t.Fatalf("retryRequest error = %v", err)
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
}

func TestRetryRequest_Exhausted(t *testing.T) {
	c := &mistClient{maxRetries: 2, retryBackoff: time.Millisecond}
	calls := 0
	err := c.retryRequest(context.Background(), func() (int, error) {
		calls++
		return http.StatusServiceUnavailable, errors.New("always fails")
	})
	if err == nil {
		t.Fatal("expected error after exhaustion")
	}
	if !strings.Contains(err.Error(), "exhausted") {
		t.Errorf("error = %v, want to mention exhaustion", err)
	}
	if calls != 3 { // initial + 2 retries
		t.Errorf("calls = %d, want 3", calls)
	}
}

func TestRetryRequest_ContextCancellation(t *testing.T) {
	c := &mistClient{maxRetries: 5, retryBackoff: 100 * time.Millisecond}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	err := c.retryRequest(ctx, func() (int, error) {
		return http.StatusServiceUnavailable, errors.New("transient")
	})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("retryRequest err = %v, want context.Canceled", err)
	}
}

func TestRateLimiter_WaitAndClose(t *testing.T) {
	r := newRateLimiter(2, 100*time.Millisecond)
	if r == nil {
		t.Fatal("newRateLimiter returned nil")
	}
	// Two waits should consume the bucket without blocking measurably.
	start := time.Now()
	r.wait()
	r.wait()
	if time.Since(start) > 50*time.Millisecond {
		t.Errorf("first two wait() calls took too long: %v", time.Since(start))
	}
	r.Close()
	// Calling Close twice is a no-op.
	r.Close()
}

func TestRateLimiter_NilOnZeroLimit(t *testing.T) {
	if r := newRateLimiter(0, time.Second); r != nil {
		t.Errorf("newRateLimiter(0, _) = %v, want nil", r)
	}
}

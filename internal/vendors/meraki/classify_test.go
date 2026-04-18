package meraki

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"

	"github.com/ravinald/wifimgr/internal/vendors"
)

// fakeResponse builds a *resty.Response with the given status. Needed
// because resty.Response wraps *http.Response; tests construct one by
// hand to avoid spinning up an httptest server per row.
func fakeResponse(status int, headers http.Header) *resty.Response {
	raw := &http.Response{
		StatusCode: status,
		Header:     headers,
	}
	return &resty.Response{RawResponse: raw}
}

func TestClassifyError(t *testing.T) {
	sdkErr := fmt.Errorf("sdk-side failure")

	tests := []struct {
		name            string
		resp            *resty.Response
		err             error
		wantNil         bool
		wantAuthStatus  int
		wantServerCode  int
		wantRateLimited bool
		wantNotFound    bool
		wantTransport   bool
		wantRetryable   bool
		wantRetryAfter  time.Duration
	}{
		{
			name:    "success: 200 with no error is nil",
			resp:    fakeResponse(http.StatusOK, nil),
			err:     nil,
			wantNil: true,
		},
		{
			name:    "success: 201 is nil even if sdkErr is spurious-nil",
			resp:    fakeResponse(http.StatusCreated, nil),
			err:     nil,
			wantNil: true,
		},
		{
			name:           "auth: 401 becomes AuthError",
			resp:           fakeResponse(http.StatusUnauthorized, nil),
			err:            sdkErr,
			wantAuthStatus: 401,
		},
		{
			name:           "auth: 403 becomes AuthError",
			resp:           fakeResponse(http.StatusForbidden, nil),
			err:            sdkErr,
			wantAuthStatus: 403,
		},
		{
			name:            "rate limit: 429 becomes RateLimitError with retry-after",
			resp:            fakeResponse(http.StatusTooManyRequests, http.Header{"Retry-After": []string{"7"}}),
			err:             sdkErr,
			wantRateLimited: true,
			wantRetryAfter:  7 * time.Second,
		},
		{
			name:            "rate limit: 429 with no header gives zero RetryAfter",
			resp:            fakeResponse(http.StatusTooManyRequests, nil),
			err:             sdkErr,
			wantRateLimited: true,
			wantRetryAfter:  0,
		},
		{
			name:         "not found: 404 becomes NotFoundError",
			resp:         fakeResponse(http.StatusNotFound, nil),
			err:          sdkErr,
			wantNotFound: true,
		},
		{
			name:           "server: 500 becomes ServerError",
			resp:           fakeResponse(http.StatusInternalServerError, nil),
			err:            sdkErr,
			wantServerCode: 500,
		},
		{
			name:           "server: 503 becomes ServerError",
			resp:           fakeResponse(http.StatusServiceUnavailable, nil),
			err:            sdkErr,
			wantServerCode: 503,
		},
		{
			name:          "network: nil resp becomes retryable TransportError",
			resp:          nil,
			err:           sdkErr,
			wantTransport: true,
			wantRetryable: true,
		},
		{
			name:          "client: 400 becomes non-retryable TransportError",
			resp:          fakeResponse(http.StatusBadRequest, nil),
			err:           sdkErr,
			wantTransport: true,
			wantRetryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyError("test-api", "TestOp", tt.resp, tt.err)

			if tt.wantNil {
				if got != nil {
					t.Fatalf("want nil, got %v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("want typed error, got nil")
			}

			if tt.wantAuthStatus != 0 {
				var ae *vendors.AuthError
				if !errors.As(got, &ae) {
					t.Fatalf("want *AuthError, got %T: %v", got, got)
				}
				if ae.Status != tt.wantAuthStatus {
					t.Errorf("AuthError.Status = %d, want %d", ae.Status, tt.wantAuthStatus)
				}
				if ae.APILabel != "test-api" {
					t.Errorf("AuthError.APILabel = %q, want test-api", ae.APILabel)
				}
			}

			if tt.wantRateLimited {
				var rl *vendors.RateLimitError
				if !errors.As(got, &rl) {
					t.Fatalf("want *RateLimitError, got %T: %v", got, got)
				}
				if rl.RetryAfter != tt.wantRetryAfter {
					t.Errorf("RateLimitError.RetryAfter = %v, want %v",
						rl.RetryAfter, tt.wantRetryAfter)
				}
			}

			if tt.wantNotFound {
				var nf *vendors.NotFoundError
				if !errors.As(got, &nf) {
					t.Fatalf("want *NotFoundError, got %T: %v", got, got)
				}
			}

			if tt.wantServerCode != 0 {
				var se *vendors.ServerError
				if !errors.As(got, &se) {
					t.Fatalf("want *ServerError, got %T: %v", got, got)
				}
				if se.Status != tt.wantServerCode {
					t.Errorf("ServerError.Status = %d, want %d", se.Status, tt.wantServerCode)
				}
				if !errors.Is(got, sdkErr) {
					t.Error("ServerError should unwrap to the SDK error")
				}
			}

			if tt.wantTransport {
				var te *vendors.TransportError
				if !errors.As(got, &te) {
					t.Fatalf("want *TransportError, got %T: %v", got, got)
				}
				if te.Retryable != tt.wantRetryable {
					t.Errorf("TransportError.Retryable = %v, want %v",
						te.Retryable, tt.wantRetryable)
				}
			}
		})
	}
}

// TestRetryState_ShouldRetry_TypedErrors exercises the new type-based
// dispatch inside ShouldRetry. The critical properties: auth errors are
// NEVER retried; 429/5xx/retryable-transport ARE retried (within budget).
func TestRetryState_ShouldRetry_TypedErrors(t *testing.T) {
	cfg := &RetryConfig{MaxRetries: 3, RetryOn429: true}

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"auth 401 is never retried", &vendors.AuthError{Status: 401}, false},
		{"auth 403 is never retried", &vendors.AuthError{Status: 403}, false},
		{"404 not-found is never retried", &vendors.NotFoundError{}, false},
		{"non-retryable transport is never retried", &vendors.TransportError{Retryable: false}, false},
		{"rate limit is retried when configured", &vendors.RateLimitError{}, true},
		{"server error is retried", &vendors.ServerError{Status: 503}, true},
		{"retryable transport is retried", &vendors.TransportError{Retryable: true}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewRetryState(cfg)
			if got := s.ShouldRetry(tc.err); got != tc.want {
				t.Errorf("ShouldRetry(%T) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

// TestRetryState_ExhaustsBudget verifies that ShouldRetry returns false
// once Attempt >= MaxRetries, even for a retryable error.
func TestRetryState_ExhaustsBudget(t *testing.T) {
	cfg := &RetryConfig{MaxRetries: 2, RetryOn429: true}
	s := NewRetryState(cfg)
	err := &vendors.RateLimitError{}

	if !s.ShouldRetry(err) {
		t.Fatal("first attempt: want retry, got no retry")
	}
	s.Attempt = 2
	if s.ShouldRetry(err) {
		t.Fatal("attempt == MaxRetries: want no retry, got retry")
	}
}

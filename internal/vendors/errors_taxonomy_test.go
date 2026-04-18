package vendors

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestErrorTaxonomy_AsAndUnwrap verifies that errors.As and errors.Unwrap
// behave correctly for the new taxonomy types. Callers in retry loops rely
// on these properties to classify failures without string matching.
func TestErrorTaxonomy_AsAndUnwrap(t *testing.T) {
	base := fmt.Errorf("underlying network error")

	tests := []struct {
		name      string
		err       error
		checkFunc func(error) bool
	}{
		{
			name: "TransportError matches via errors.As",
			err:  &TransportError{APILabel: "m-1", Op: "List", Status: 0, Retryable: true, Err: base},
			checkFunc: func(e error) bool {
				var te *TransportError
				return errors.As(e, &te) && te.Retryable
			},
		},
		{
			name: "TransportError unwraps to inner error",
			err:  &TransportError{APILabel: "m-1", Op: "List", Err: base},
			checkFunc: func(e error) bool {
				return errors.Is(e, base)
			},
		},
		{
			name: "AuthError matches and is not retryable-classed",
			err:  &AuthError{APILabel: "m-1", Status: 401, Reason: "bad token"},
			checkFunc: func(e error) bool {
				var ae *AuthError
				var te *TransportError
				return errors.As(e, &ae) && !errors.As(e, &te)
			},
		},
		{
			name: "RateLimitError carries RetryAfter",
			err:  &RateLimitError{APILabel: "m-1", RetryAfter: 3 * time.Second},
			checkFunc: func(e error) bool {
				var rl *RateLimitError
				if !errors.As(e, &rl) {
					return false
				}
				return rl.RetryAfter == 3*time.Second
			},
		},
		{
			name: "ServerError matches for 5xx and unwraps",
			err:  &ServerError{APILabel: "m-1", Status: 503, Err: base},
			checkFunc: func(e error) bool {
				var se *ServerError
				return errors.As(e, &se) && se.Status == 503 && errors.Is(e, base)
			},
		},
		{
			name: "NotFoundError matches",
			err:  &NotFoundError{APILabel: "m-1", Resource: "device X"},
			checkFunc: func(e error) bool {
				var nf *NotFoundError
				return errors.As(e, &nf)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.checkFunc(tt.err) {
				t.Errorf("checkFunc returned false for: %v", tt.err)
			}
		})
	}
}

// TestErrorTaxonomy_UserMessages verifies that every taxonomy type produces
// a UserMessage that includes the API label and some remediation hint.
func TestErrorTaxonomy_UserMessages(t *testing.T) {
	cases := []struct {
		name      string
		err       interface{ UserMessage() string }
		mustHave  []string
		mustNotHave []string
	}{
		{
			name:     "AuthError 401 suggests checking token",
			err:      &AuthError{APILabel: "mist-prod", Status: 401, Reason: "bad token"},
			mustHave: []string{"mist-prod", "401", "token"},
		},
		{
			name:     "AuthError 403 suggests checking permissions",
			err:      &AuthError{APILabel: "meraki", Status: 403, Reason: "forbidden"},
			mustHave: []string{"meraki", "403", "permission"},
		},
		{
			name:     "RateLimitError shows retry-after when present",
			err:      &RateLimitError{APILabel: "meraki", RetryAfter: 5 * time.Second},
			mustHave: []string{"meraki", "rate limited", "5s"},
		},
		{
			name:     "ServerError calls out transient",
			err:      &ServerError{APILabel: "mist", Status: 502},
			mustHave: []string{"mist", "502", "transient"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			msg := tc.err.UserMessage()
			for _, want := range tc.mustHave {
				if !strings.Contains(msg, want) {
					t.Errorf("UserMessage missing %q: %s", want, msg)
				}
			}
		})
	}
}

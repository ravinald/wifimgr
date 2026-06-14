package aruba

import (
	"errors"
	"testing"

	"github.com/ravinald/wifimgr/internal/vendors"
)

func intPtr(i int) *int { return &i }

func TestClassifyEnvelope(t *testing.T) {
	tests := []struct {
		name   string
		env    apiEnvelope
		wantOK bool
		check  func(error) bool
	}{
		{
			name:   "success code 0",
			env:    apiEnvelope{Status: "Success", StatusCode: intPtr(0)},
			wantOK: true,
		},
		{
			name: "expired sid",
			env:  apiEnvelope{StatusCode: intPtr(1), Message: "Invalid session id or session id has expired"},
			check: func(err error) bool {
				return isExpiredSession(err)
			},
		},
		{
			name: "invalid api -> not found",
			env:  apiEnvelope{Status: "Failed", StatusCode: intPtr(2), ErrorMessage: "Invalid API /rest/sow-cmd"},
			check: func(err error) bool {
				var e *vendors.NotFoundError
				return errors.As(err, &e)
			},
		},
		{
			name: "config module -> config error",
			env:  apiEnvelope{StatusCode: intPtr(6), Message: "Profile not found"},
			check: func(err error) bool {
				var e *configModuleError
				return errors.As(err, &e)
			},
		},
		{
			name: "internal comm -> server error",
			env:  apiEnvelope{StatusCode: intPtr(7), Message: "Internal communication error"},
			check: func(err error) bool {
				var e *vendors.ServerError
				return errors.As(err, &e)
			},
		},
		{
			name: "failed login (no code)",
			env:  apiEnvelope{Status: "Failed", ErrorMessage: "Login failed"},
			check: func(err error) bool {
				var e *vendors.AuthError
				return errors.As(err, &e)
			},
		},
		{
			name: "rest api disabled notice",
			env:  apiEnvelope{Status: "Success", StatusCode: intPtr(0), Message: "REST API Service is not enabled"},
			check: func(err error) bool {
				var e *restDisabledError
				return errors.As(err, &e)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := classifyEnvelope("aruba-lab", "show-cmd", &tt.env)
			if tt.wantOK {
				if err != nil {
					t.Fatalf("want nil, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("want error, got nil")
			}
			if tt.check != nil && !tt.check(err) {
				t.Fatalf("error did not match expected type: %v", err)
			}
		})
	}
}

func TestClassifyHTTP(t *testing.T) {
	var auth *vendors.AuthError
	if err := classifyHTTP("aruba", "login", 403, []byte("forbidden")); !errors.As(err, &auth) {
		t.Errorf("403 should classify as AuthError, got %v", err)
	}
	var server *vendors.ServerError
	if err := classifyHTTP("aruba", "ssid", 502, []byte("bad gateway")); !errors.As(err, &server) {
		t.Errorf("502 should classify as ServerError, got %v", err)
	}
}

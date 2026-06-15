package vendors

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestClassifyRefreshError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"nil", nil, ""},
		{"deadline", errors.New(`Post "https://x/rest/login": context deadline exceeded`), "connection failure"},
		{"refused", fmt.Errorf("dial tcp: %w", errors.New("connection refused")), "connection failure"},
		{"no such host", errors.New("lookup api.x: no such host"), "connection failure"},
		{"timeout", errors.New("Client.Timeout exceeded while awaiting headers"), "connection failure"},
		{"401", errors.New("GET /sites: 401 Unauthorized"), "auth failure"},
		{"forbidden", errors.New("403 forbidden"), "auth failure"},
		{"other", errors.New("failed to fetch sites: boom"), "failed to fetch sites: boom"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := classifyRefreshError(c.err); got != c.want {
				t.Errorf("classifyRefreshError(%v) = %q, want %q", c.err, got, c.want)
			}
		})
	}
}

func TestRecordRefreshFailureLocked(t *testing.T) {
	cm := NewCacheManager(t.TempDir(), NewAPIClientRegistry())
	if err := cm.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	// Seed a successful cache.
	success := time.Now().Add(-2 * time.Hour)
	cache := NewAPICache("test-api", "aruba", "")
	cache.Meta.LastRefresh = success
	if err := cm.SaveAPICache(cache); err != nil {
		t.Fatalf("SaveAPICache: %v", err)
	}

	// Record a failure.
	cm.recordRefreshFailureLocked("test-api", errors.New("POST /rest/login: context deadline exceeded"))

	got, err := cm.GetAPICache("test-api")
	if err != nil {
		t.Fatalf("GetAPICache: %v", err)
	}
	if !got.Meta.LastRefresh.Equal(success) {
		t.Errorf("LastRefresh changed: got %v, want %v (last success must survive a failure)", got.Meta.LastRefresh, success)
	}
	if got.Meta.LastFailure.IsZero() {
		t.Error("LastFailure not set after failure")
	}
	if !got.Meta.LastFailure.After(got.Meta.LastRefresh) {
		t.Error("LastFailure should be after LastRefresh (currently failing)")
	}
	if got.Meta.LastError != "connection failure" {
		t.Errorf("LastError = %q, want %q", got.Meta.LastError, "connection failure")
	}
}

func TestRecordRefreshFailureLocked_NoPriorCacheIsNoop(t *testing.T) {
	cm := NewCacheManager(t.TempDir(), NewAPIClientRegistry())
	if err := cm.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	// No cache for this label — must not panic or create one.
	cm.recordRefreshFailureLocked("never-seen", errors.New("boom"))
	if _, err := cm.GetAPICache("never-seen"); err == nil {
		t.Error("expected no cache to be created for a never-seen API")
	}
}

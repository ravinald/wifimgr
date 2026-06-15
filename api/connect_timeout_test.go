package api

import (
	"net/http"
	"testing"
	"time"
)

func TestWithConnectTimeout(t *testing.T) {
	c := NewClientWithOptions("key", "https://api.mist.com", "org", WithConnectTimeout(4*time.Second))
	mc, ok := c.(*mistClient)
	if !ok {
		t.Fatalf("expected *mistClient, got %T", c)
	}
	tr, ok := mc.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport (cloned from default), got %T", mc.httpClient.Transport)
	}
	if tr.DialContext == nil {
		t.Error("DialContext not set")
	}
	if tr.TLSHandshakeTimeout != 4*time.Second {
		t.Errorf("TLSHandshakeTimeout = %v, want 4s", tr.TLSHandshakeTimeout)
	}
	// Overall timeout left intact.
	if mc.httpClient.Timeout != 30*time.Second {
		t.Errorf("overall Timeout = %v, want 30s (connect timeout must not cap it)", mc.httpClient.Timeout)
	}
}

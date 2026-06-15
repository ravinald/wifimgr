package ubiquiti

import (
	"net/http"
	"testing"
	"time"
)

func TestWithConnectTimeout(t *testing.T) {
	c := NewClient("key", "https://api.ui.com", WithConnectTimeout(2*time.Second))

	tr, ok := c.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", c.httpClient.Transport)
	}
	if tr.DialContext == nil {
		t.Error("DialContext not set")
	}
	if tr.TLSHandshakeTimeout != 2*time.Second {
		t.Errorf("TLSHandshakeTimeout = %v, want 2s", tr.TLSHandshakeTimeout)
	}
	// Replaces the shared http.DefaultClient (Timeout 0) with a bounded one.
	if c.httpClient == http.DefaultClient {
		t.Error("still using the shared http.DefaultClient")
	}
	if c.httpClient.Timeout != 30*time.Second {
		t.Errorf("overall Timeout = %v, want 30s", c.httpClient.Timeout)
	}
}

package aruba

import (
	"net/http"
	"testing"
	"time"
)

func TestWithConnectTimeout(t *testing.T) {
	c := NewClient("u", "p", "https://10.0.0.1:4343",
		WithInsecureSkipVerify(true),
		WithConnectTimeout(3*time.Second),
	)

	tr, ok := c.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", c.httpClient.Transport)
	}
	if tr.DialContext == nil {
		t.Error("DialContext not set")
	}
	if tr.TLSHandshakeTimeout != 3*time.Second {
		t.Errorf("TLSHandshakeTimeout = %v, want 3s", tr.TLSHandshakeTimeout)
	}
	// The existing transport (and its TLS config) must survive the option.
	if tr.TLSClientConfig == nil || !tr.TLSClientConfig.InsecureSkipVerify {
		t.Error("WithConnectTimeout clobbered the TLS config")
	}
}

func TestWithConnectTimeout_ZeroIsNoop(t *testing.T) {
	c := NewClient("u", "p", "https://10.0.0.1:4343", WithConnectTimeout(0))
	tr := c.httpClient.Transport.(*http.Transport)
	if tr.TLSHandshakeTimeout != 0 || tr.DialContext != nil {
		t.Error("zero timeout should be a no-op")
	}
}

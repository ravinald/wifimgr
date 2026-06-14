package aruba

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHostFromBaseURL(t *testing.T) {
	tests := map[string]string{
		"https://10.0.0.1:4343":  "10.0.0.1",
		"http://127.0.0.1:8080":  "127.0.0.1",
		"10.0.0.5:4343":          "10.0.0.5",
		"https://vc.example.com": "vc.example.com",
	}
	for in, want := range tests {
		if got := hostFromBaseURL(in); got != want {
			t.Errorf("hostFromBaseURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEncodeShowCmd(t *testing.T) {
	if got := encodeShowCmd("show aps"); got != "show%20aps" {
		t.Errorf("encodeShowCmd = %q, want show%%20aps", got)
	}
	if got := encodeShowCmd("show client status 00:11:22:33:44:55"); !strings.Contains(got, "%20") || strings.Contains(got, "+") {
		t.Errorf("encodeShowCmd should use %%20 not +, got %q", got)
	}
}

func TestStripCLIPrefix(t *testing.T) {
	in := "cli output:\nCOMMAND=show aps\n2 Access Points\n---"
	want := "2 Access Points\n---"
	if got := stripCLIPrefix(in); got != want {
		t.Errorf("stripCLIPrefix = %q, want %q", got, want)
	}
}

func TestRedactPath(t *testing.T) {
	got := redactPath("/rest/show-cmd?iap_ip_addr=10.0.0.1&cmd=show%20aps&sid=SECRET")
	if strings.Contains(got, "SECRET") {
		t.Errorf("redactPath leaked sid: %q", got)
	}
}

func TestShowCommand_RoundTrip(t *testing.T) {
	var loggedIn bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/login":
			loggedIn = true
			_, _ = w.Write([]byte(`{"Status":"Success","sid":"abc123"}`))
		case "/rest/show-cmd":
			if r.URL.Query().Get("sid") != "abc123" {
				http.Error(w, "no sid", http.StatusUnauthorized)
				return
			}
			_, _ = w.Write([]byte(`{"Status":"Success","Status-code":0,"Command output":"cli output:\nCOMMAND=show aps\n1 Access Points"}`))
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c := NewClient("admin", "admin", srv.URL, WithHTTPClient(srv.Client()), WithAPILabel("aruba-lab"))
	c.minInterval = 0

	out, err := c.ShowCommand(context.Background(), "show aps")
	if err != nil {
		t.Fatalf("ShowCommand: %v", err)
	}
	if !loggedIn {
		t.Error("expected an implicit login before the show command")
	}
	if !strings.Contains(out, "1 Access Points") {
		t.Errorf("output = %q", out)
	}
	if strings.Contains(out, "cli output:") {
		t.Errorf("CLI preamble not stripped: %q", out)
	}
}

func TestPostObject_ConfigError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/login":
			_, _ = w.Write([]byte(`{"Status":"Success","sid":"s"}`))
		case "/rest/ssid":
			_, _ = w.Write([]byte(`{"Status-code":6,"message":"CLI error: Profile not found"}`))
		default:
			http.Error(w, "nf", http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c := NewClient("admin", "admin", srv.URL, WithHTTPClient(srv.Client()))
	c.minInterval = 0

	err := c.PostObject(context.Background(), "ssid", map[string]any{"ssid-profile": map[string]any{"action": "delete"}})
	if err == nil {
		t.Fatal("expected a config-module error, got nil")
	}
}

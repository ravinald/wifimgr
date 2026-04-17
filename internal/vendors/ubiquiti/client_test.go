package ubiquiti

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestClient_AuthHeader(t *testing.T) {
	var gotHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-API-KEY")
		json.NewEncoder(w).Encode(apiResponse{
			Data:           []any{},
			HTTPStatusCode: 200,
		})
	}))
	defer server.Close()

	client := NewClient("test-api-key", server.URL, WithHTTPClient(server.Client()))
	_, err := client.GetSites(context.Background())
	if err != nil {
		t.Fatalf("GetSites failed: %v", err)
	}

	if gotHeader != "test-api-key" {
		t.Errorf("X-API-KEY header = %q, want %q", gotHeader, "test-api-key")
	}
}

func TestClient_Pagination(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		nextToken := r.URL.Query().Get("nextToken")

		var resp apiResponse
		switch {
		case nextToken == "" && callCount == 1:
			resp = apiResponse{
				Data: []Site{
					{SiteID: "site-1", Meta: SiteMeta{Name: "Site 1"}},
				},
				HTTPStatusCode: 200,
				NextToken:      "page2",
			}
		case nextToken == "page2":
			resp = apiResponse{
				Data: []Site{
					{SiteID: "site-2", Meta: SiteMeta{Name: "Site 2"}},
				},
				HTTPStatusCode: 200,
			}
		default:
			t.Errorf("unexpected request: nextToken=%q, callCount=%d", nextToken, callCount)
			resp = apiResponse{Data: []Site{}, HTTPStatusCode: 200}
		}

		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("key", server.URL, WithHTTPClient(server.Client()))
	sites, err := client.GetSites(context.Background())
	if err != nil {
		t.Fatalf("GetSites failed: %v", err)
	}

	if len(sites) != 2 {
		t.Fatalf("got %d sites, want 2", len(sites))
	}
	if sites[0].SiteID != "site-1" {
		t.Errorf("sites[0].SiteID = %q, want %q", sites[0].SiteID, "site-1")
	}
	if sites[1].SiteID != "site-2" {
		t.Errorf("sites[1].SiteID = %q, want %q", sites[1].SiteID, "site-2")
	}
	if callCount != 2 {
		t.Errorf("callCount = %d, want 2", callCount)
	}
}

func TestClient_429Retry(t *testing.T) {
	var callCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := callCount.Add(1)
		if count <= 2 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		json.NewEncoder(w).Encode(apiResponse{
			Data:           []Site{{SiteID: "site-1"}},
			HTTPStatusCode: 200,
		})
	}))
	defer server.Close()

	client := NewClient("key", server.URL, WithHTTPClient(server.Client()))
	// Use a shorter backoff for testing
	client.retryConfig = &RetryConfig{
		MaxRetries:  3,
		BaseBackoff: 1,
		MaxBackoff:  10,
	}

	sites, err := client.GetSites(context.Background())
	if err != nil {
		t.Fatalf("GetSites failed: %v", err)
	}
	if len(sites) != 1 {
		t.Fatalf("got %d sites, want 1", len(sites))
	}
	if got := callCount.Load(); got != 3 {
		t.Errorf("callCount = %d, want 3 (2 retries + 1 success)", got)
	}
}

func TestClient_429ExhaustedRetries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := NewClient("key", server.URL, WithHTTPClient(server.Client()))
	client.retryConfig = &RetryConfig{
		MaxRetries:  2,
		BaseBackoff: 1,
		MaxBackoff:  10,
	}

	_, err := client.GetSites(context.Background())
	if err == nil {
		t.Fatal("expected error after exhausting retries, got nil")
	}
}

func TestClient_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"Invalid API key"}`))
	}))
	defer server.Close()

	client := NewClient("bad-key", server.URL, WithHTTPClient(server.Client()))
	_, err := client.GetSites(context.Background())
	if err == nil {
		t.Fatal("expected error for 401 response, got nil")
	}
}

func TestClient_GetDevices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/devices" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(apiResponse{
			Data: []HostDeviceGroup{
				{
					HostID:   "host-1",
					HostName: "UDM-Pro",
					Devices: []Device{
						{ID: "dev-1", MAC: "aa:bb:cc:dd:ee:01", Model: "U6-Pro", ProductLine: "network"},
						{ID: "dev-2", MAC: "aa:bb:cc:dd:ee:02", Model: "USW-Pro-24", ProductLine: "network"},
					},
				},
			},
			HTTPStatusCode: 200,
		})
	}))
	defer server.Close()

	client := NewClient("key", server.URL, WithHTTPClient(server.Client()))
	groups, err := client.GetDevices(context.Background())
	if err != nil {
		t.Fatalf("GetDevices failed: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("got %d groups, want 1", len(groups))
	}
	if len(groups[0].Devices) != 2 {
		t.Fatalf("got %d devices, want 2", len(groups[0].Devices))
	}
}

func TestClient_GetHosts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/hosts" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(apiResponse{
			Data: []Host{
				{ID: "host-1", Type: "udm", ReportedState: ReportedState{Name: "UDM-Pro"}},
			},
			HTTPStatusCode: 200,
		})
	}))
	defer server.Close()

	client := NewClient("key", server.URL, WithHTTPClient(server.Client()))
	hosts, err := client.GetHosts(context.Background())
	if err != nil {
		t.Fatalf("GetHosts failed: %v", err)
	}
	if len(hosts) != 1 {
		t.Fatalf("got %d hosts, want 1", len(hosts))
	}
	if hosts[0].ID != "host-1" {
		t.Errorf("hosts[0].ID = %q, want %q", hosts[0].ID, "host-1")
	}
}

package mist

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ravinald/wifimgr/api"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// testClient wraps api.Client with custom search responses for testing.
type testClient struct {
	api.Client
	wirelessResults *api.MistWirelessClientResponse
	wiredResults    *api.MistWiredClientResponse
	err             error
}

func newTestClient() *testClient {
	mockClient := api.NewMockClient(api.Config{
		BaseURL:      "https://api.mist.com",
		APIToken:     "test-token",
		Organization: "test-org",
	})
	return &testClient{
		Client: mockClient,
	}
}

func (tc *testClient) SearchWirelessClients(ctx context.Context, orgID, text string) (*api.MistWirelessClientResponse, error) {
	if tc.err != nil {
		return nil, tc.err
	}
	if tc.wirelessResults != nil {
		return tc.wirelessResults, nil
	}
	// Fall back to the mock's default implementation
	return tc.Client.SearchWirelessClients(ctx, orgID, text)
}

func (tc *testClient) SearchWiredClients(ctx context.Context, orgID, text string) (*api.MistWiredClientResponse, error) {
	if tc.err != nil {
		return nil, tc.err
	}
	if tc.wiredResults != nil {
		return tc.wiredResults, nil
	}
	// Fall back to the mock's default implementation
	return tc.Client.SearchWiredClients(ctx, orgID, text)
}

// TestEstimateSearchCost tests that Mist always returns low cost.
func TestEstimateSearchCost(t *testing.T) {
	s := &searchService{
		client: newTestClient(),
		orgID:  "org123",
	}

	tests := []struct {
		name   string
		text   string
		siteID string
	}{
		{
			name:   "MAC address",
			text:   "aa:bb:cc:dd:ee:ff",
			siteID: "",
		},
		{
			name:   "hostname",
			text:   "laptop-john",
			siteID: "",
		},
		{
			name:   "with site filter",
			text:   "device",
			siteID: "site123",
		},
		{
			name:   "IP address",
			text:   "192.168.1.1",
			siteID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			estimate, err := s.EstimateSearchCost(ctx, tt.text, tt.siteID)

			if err != nil {
				t.Fatalf("EstimateSearchCost() error = %v", err)
			}

			if estimate.APICalls != 1 {
				t.Errorf("APICalls = %d, want 1", estimate.APICalls)
			}

			if estimate.EstimatedDuration != 2*time.Second {
				t.Errorf("EstimatedDuration = %v, want 2s", estimate.EstimatedDuration)
			}

			if estimate.NeedsConfirmation {
				t.Error("NeedsConfirmation = true, want false (Mist is always efficient)")
			}

			if estimate.Description != "Single API call" {
				t.Errorf("Description = %q, want %q", estimate.Description, "Single API call")
			}
		})
	}
}

// TestSearchWirelessClients tests wireless client search.
func TestSearchWirelessClients(t *testing.T) {
	tests := []struct {
		name         string
		text         string
		opts         vendors.SearchOptions
		mockResponse *api.MistWirelessClientResponse
		mockError    error
		wantCount    int
		wantError    bool
	}{
		{
			name: "successful search",
			text: "laptop",
			opts: vendors.SearchOptions{},
			mockResponse: &api.MistWirelessClientResponse{
				Results: []*api.MistWirelessClient{
					{
						MAC:          strPtr("aabbccddeeff"),
						SiteID:       strPtr("site123"),
						LastHostname: strPtr("laptop-01"),
						LastIP:       strPtr("192.168.1.100"),
						LastSSID:     strPtr("Corporate"),
					},
					{
						MAC:          strPtr("112233445566"),
						SiteID:       strPtr("site456"),
						LastHostname: strPtr("laptop-02"),
						LastIP:       strPtr("192.168.1.101"),
						LastSSID:     strPtr("Guest"),
					},
				},
			},
			wantCount: 2,
			wantError: false,
		},
		{
			name: "search with site filter",
			text: "device",
			opts: vendors.SearchOptions{SiteID: "site123"},
			mockResponse: &api.MistWirelessClientResponse{
				Results: []*api.MistWirelessClient{
					{
						MAC:    strPtr("aabbccddeeff"),
						SiteID: strPtr("site123"),
					},
					{
						MAC:    strPtr("112233445566"),
						SiteID: strPtr("site456"), // Different site - should be filtered
					},
				},
			},
			wantCount: 1,
			wantError: false,
		},
		{
			name:         "nil response",
			text:         "test",
			opts:         vendors.SearchOptions{},
			mockResponse: nil,
			wantCount:    0,
			wantError:    false,
		},
		{
			name: "empty results",
			text: "nonexistent",
			opts: vendors.SearchOptions{},
			mockResponse: &api.MistWirelessClientResponse{
				Results: []*api.MistWirelessClient{},
			},
			wantCount: 0,
			wantError: false,
		},
		{
			name:         "API error",
			text:         "test",
			opts:         vendors.SearchOptions{},
			mockResponse: nil,
			mockError:    errors.New("API error"),
			wantCount:    0,
			wantError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := newTestClient()
			tc.wirelessResults = tt.mockResponse
			tc.err = tt.mockError

			s := &searchService{
				client: tc,
				orgID:  "org123",
			}

			ctx := context.Background()
			results, err := s.SearchWirelessClients(ctx, tt.text, tt.opts)

			if tt.wantError {
				if err == nil {
					t.Error("SearchWirelessClients() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("SearchWirelessClients() error = %v", err)
			}

			if results == nil {
				t.Fatal("SearchWirelessClients() returned nil results")
			}

			if len(results.Results) != tt.wantCount {
				t.Errorf("len(Results) = %d, want %d", len(results.Results), tt.wantCount)
			}

			if results.Total != tt.wantCount {
				t.Errorf("Total = %d, want %d", results.Total, tt.wantCount)
			}

			// Verify SourceVendor is set
			for i, client := range results.Results {
				if client.SourceVendor != "mist" {
					t.Errorf("Results[%d].SourceVendor = %q, want %q", i, client.SourceVendor, "mist")
				}
			}
		})
	}
}

// TestSearchWiredClients tests wired client search.
func TestSearchWiredClients(t *testing.T) {
	tests := []struct {
		name         string
		text         string
		opts         vendors.SearchOptions
		mockResponse *api.MistWiredClientResponse
		mockError    error
		wantCount    int
		wantError    bool
	}{
		{
			name: "successful search",
			text: "server",
			opts: vendors.SearchOptions{},
			mockResponse: &api.MistWiredClientResponse{
				Results: []*api.MistWiredClient{
					{
						ClientMAC:     strPtr("aabbccddeeff"),
						SiteID:        strPtr("site123"),
						LastHostname:  strPtr("server-01"),
						LastIP:        strPtr("192.168.1.10"),
						LastDeviceMAC: strPtr("112233445566"),
						LastPortID:    strPtr("ge-0/0/1"),
					},
				},
			},
			wantCount: 1,
			wantError: false,
		},
		{
			name: "search with site filter",
			text: "desktop",
			opts: vendors.SearchOptions{SiteID: "site123"},
			mockResponse: &api.MistWiredClientResponse{
				Results: []*api.MistWiredClient{
					{
						ClientMAC: strPtr("aabbccddeeff"),
						SiteID:    strPtr("site123"),
					},
					{
						ClientMAC: strPtr("112233445566"),
						SiteID:    strPtr("site456"), // Different site - should be filtered
					},
					{
						ClientMAC: strPtr("778899aabbcc"),
						SiteID:    strPtr("site123"), // Same site - should be included
					},
				},
			},
			wantCount: 2,
			wantError: false,
		},
		{
			name:         "nil response",
			text:         "test",
			opts:         vendors.SearchOptions{},
			mockResponse: nil,
			wantCount:    0,
			wantError:    false,
		},
		{
			name: "empty results",
			text: "nonexistent",
			opts: vendors.SearchOptions{},
			mockResponse: &api.MistWiredClientResponse{
				Results: []*api.MistWiredClient{},
			},
			wantCount: 0,
			wantError: false,
		},
		{
			name:         "API error",
			text:         "test",
			opts:         vendors.SearchOptions{},
			mockResponse: nil,
			mockError:    errors.New("connection timeout"),
			wantCount:    0,
			wantError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := newTestClient()
			tc.wiredResults = tt.mockResponse
			tc.err = tt.mockError

			s := &searchService{
				client: tc,
				orgID:  "org123",
			}

			ctx := context.Background()
			results, err := s.SearchWiredClients(ctx, tt.text, tt.opts)

			if tt.wantError {
				if err == nil {
					t.Error("SearchWiredClients() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("SearchWiredClients() error = %v", err)
			}

			if results == nil {
				t.Fatal("SearchWiredClients() returned nil results")
			}

			if len(results.Results) != tt.wantCount {
				t.Errorf("len(Results) = %d, want %d", len(results.Results), tt.wantCount)
			}

			if results.Total != tt.wantCount {
				t.Errorf("Total = %d, want %d", results.Total, tt.wantCount)
			}

			// Verify SourceVendor is set
			for i, client := range results.Results {
				if client.SourceVendor != "mist" {
					t.Errorf("Results[%d].SourceVendor = %q, want %q", i, client.SourceVendor, "mist")
				}
			}
		})
	}
}

// TestConvertWiredClientToVendor tests the wired client converter.
func TestConvertWiredClientToVendor(t *testing.T) {
	tests := []struct {
		name   string
		client *api.MistWiredClient
		want   *vendors.WiredClient
	}{
		{
			name:   "nil client",
			client: nil,
			want:   nil,
		},
		{
			name: "full client with last fields",
			client: &api.MistWiredClient{
				ClientMAC:     strPtr("aabbccddeeff"),
				SiteID:        strPtr("site123"),
				LastHostname:  strPtr("server-01"),
				LastIP:        strPtr("192.168.1.10"),
				LastDeviceMAC: strPtr("112233445566"),
				LastPortID:    strPtr("ge-0/0/1"),
				LastVLAN:      intPtr(100),
				Manufacture:   strPtr("HP"),
			},
			want: &vendors.WiredClient{
				SourceVendor: "mist",
				MAC:          "aabbccddeeff",
				SiteID:       "site123",
				Hostname:     "server-01",
				IP:           "192.168.1.10",
				SwitchMAC:    "112233445566",
				PortID:       "ge-0/0/1",
				VLAN:         100,
				Manufacturer: "HP",
			},
		},
		{
			name: "client with array fields",
			client: &api.MistWiredClient{
				MAC:         strPtr("aabbccddeeff"),
				SiteID:      strPtr("site456"),
				Hostname:    []string{"desktop-01", "desktop-02"},
				IP:          []string{"192.168.1.20", "192.168.1.21"},
				DeviceMAC:   []string{"998877665544"},
				PortID:      []string{"ge-0/0/5"},
				VLAN:        []int{200},
				Manufacture: strPtr("Dell"),
			},
			want: &vendors.WiredClient{
				SourceVendor: "mist",
				MAC:          "aabbccddeeff",
				SiteID:       "site456",
				Hostname:     "desktop-01",
				IP:           "192.168.1.20",
				SwitchMAC:    "998877665544",
				PortID:       "ge-0/0/5",
				VLAN:         200,
				Manufacturer: "Dell",
			},
		},
		{
			name: "minimal client",
			client: &api.MistWiredClient{
				ClientMAC: strPtr("aabbccddeeff"),
				SiteID:    strPtr("site789"),
			},
			want: &vendors.WiredClient{
				SourceVendor: "mist",
				MAC:          "aabbccddeeff",
				SiteID:       "site789",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertWiredClientToVendor(tt.client)

			if tt.want == nil {
				if got != nil {
					t.Errorf("convertWiredClientToVendor() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Fatal("convertWiredClientToVendor() = nil, want non-nil")
			}

			if got.MAC != tt.want.MAC {
				t.Errorf("MAC = %q, want %q", got.MAC, tt.want.MAC)
			}
			if got.SiteID != tt.want.SiteID {
				t.Errorf("SiteID = %q, want %q", got.SiteID, tt.want.SiteID)
			}
			if got.Hostname != tt.want.Hostname {
				t.Errorf("Hostname = %q, want %q", got.Hostname, tt.want.Hostname)
			}
			if got.IP != tt.want.IP {
				t.Errorf("IP = %q, want %q", got.IP, tt.want.IP)
			}
			if got.PortID != tt.want.PortID {
				t.Errorf("PortID = %q, want %q", got.PortID, tt.want.PortID)
			}
			if got.VLAN != tt.want.VLAN {
				t.Errorf("VLAN = %d, want %d", got.VLAN, tt.want.VLAN)
			}
		})
	}
}

// TestConvertWirelessClientToVendor tests the wireless client converter.
func TestConvertWirelessClientToVendor(t *testing.T) {
	tests := []struct {
		name   string
		client *api.MistWirelessClient
		want   *vendors.WirelessClient
	}{
		{
			name:   "nil client",
			client: nil,
			want:   nil,
		},
		{
			name: "full client with last fields",
			client: &api.MistWirelessClient{
				MAC:          strPtr("aabbccddeeff"),
				SiteID:       strPtr("site123"),
				LastHostname: strPtr("laptop-01"),
				LastIP:       strPtr("192.168.1.100"),
				LastAP:       strPtr("112233445566"),
				LastSSID:     strPtr("Corporate"),
				LastVLAN:     intPtr(10),
				Band:         strPtr("5"),
				Manufacture:  strPtr("Apple"),
				LastOS:       strPtr("macOS 14.0"),
			},
			want: &vendors.WirelessClient{
				SourceVendor: "mist",
				MAC:          "aabbccddeeff",
				SiteID:       "site123",
				Hostname:     "laptop-01",
				IP:           "192.168.1.100",
				APMAC:        "112233445566",
				SSID:         "Corporate",
				VLAN:         10,
				Band:         "5",
				Manufacturer: "Apple",
				OS:           "macOS 14.0",
			},
		},
		{
			name: "client with array fields",
			client: &api.MistWirelessClient{
				MAC:         strPtr("aabbccddeeff"),
				SiteID:      strPtr("site456"),
				Hostname:    []string{"phone-01", "phone-02"},
				IP:          []string{"192.168.1.200"},
				AP:          []string{"998877665544"},
				SSID:        []string{"Guest"},
				VLAN:        []int{20},
				Band:        strPtr("2.4"),
				Manufacture: strPtr("Samsung"),
				OS:          []string{"Android 13"},
			},
			want: &vendors.WirelessClient{
				SourceVendor: "mist",
				MAC:          "aabbccddeeff",
				SiteID:       "site456",
				Hostname:     "phone-01",
				IP:           "192.168.1.200",
				APMAC:        "998877665544",
				SSID:         "Guest",
				VLAN:         20,
				Band:         "2.4",
				Manufacturer: "Samsung",
				OS:           "Android 13",
			},
		},
		{
			name: "minimal client",
			client: &api.MistWirelessClient{
				MAC:    strPtr("aabbccddeeff"),
				SiteID: strPtr("site789"),
			},
			want: &vendors.WirelessClient{
				SourceVendor: "mist",
				MAC:          "aabbccddeeff",
				SiteID:       "site789",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertWirelessClientToVendor(tt.client)

			if tt.want == nil {
				if got != nil {
					t.Errorf("convertWirelessClientToVendor() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Fatal("convertWirelessClientToVendor() = nil, want non-nil")
			}

			if got.MAC != tt.want.MAC {
				t.Errorf("MAC = %q, want %q", got.MAC, tt.want.MAC)
			}
			if got.SiteID != tt.want.SiteID {
				t.Errorf("SiteID = %q, want %q", got.SiteID, tt.want.SiteID)
			}
			if got.Hostname != tt.want.Hostname {
				t.Errorf("Hostname = %q, want %q", got.Hostname, tt.want.Hostname)
			}
			if got.IP != tt.want.IP {
				t.Errorf("IP = %q, want %q", got.IP, tt.want.IP)
			}
			if got.SSID != tt.want.SSID {
				t.Errorf("SSID = %q, want %q", got.SSID, tt.want.SSID)
			}
			if got.Band != tt.want.Band {
				t.Errorf("Band = %q, want %q", got.Band, tt.want.Band)
			}
		})
	}
}

// TestSearchServiceImplementsInterface verifies the service implements the interface.
func TestSearchServiceImplementsInterface(t *testing.T) {
	var _ vendors.SearchService = (*searchService)(nil)
}

// Helper functions for pointer creation.
func strPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

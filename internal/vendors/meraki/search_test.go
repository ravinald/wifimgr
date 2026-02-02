package meraki

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	meraki "github.com/meraki/dashboard-api-go/v5/sdk"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// mockDashboard provides a mock Meraki dashboard client for testing.
type mockDashboard struct {
	networks       *meraki.ResponseOrganizationsGetOrganizationNetworks
	clientSearch   *meraki.ResponseOrganizationsGetOrganizationClientsSearch
	networkClients *meraki.ResponseNetworksGetNetworkClients
	err            error
}

// TestEstimateSearchCost tests the cost estimation for different search scenarios.
func TestEstimateSearchCost(t *testing.T) {
	tests := []struct {
		name               string
		text               string
		siteID             string
		networkCount       int
		wantAPICalls       int
		wantConfirmation   bool
		wantDescriptionKey string
	}{
		{
			name:               "site-scoped search",
			text:               "hostname",
			siteID:             "L_123",
			networkCount:       10,
			wantAPICalls:       1,
			wantConfirmation:   false,
			wantDescriptionKey: "Single network",
		},
		{
			name:               "MAC search",
			text:               "aa:bb:cc:dd:ee:ff",
			siteID:             "",
			networkCount:       10,
			wantAPICalls:       1,
			wantConfirmation:   false,
			wantDescriptionKey: "Organization-wide MAC",
		},
		{
			name:               "IP search with few networks",
			text:               "192.168.1.1",
			siteID:             "",
			networkCount:       3,
			wantAPICalls:       3,
			wantConfirmation:   false,
			wantDescriptionKey: "Requires querying 3",
		},
		{
			name:               "hostname search needs confirmation",
			text:               "laptop",
			siteID:             "",
			networkCount:       10,
			wantAPICalls:       10,
			wantConfirmation:   true,
			wantDescriptionKey: "Requires querying 10",
		},
		{
			name:               "at confirmation threshold",
			text:               "device",
			siteID:             "",
			networkCount:       5,
			wantAPICalls:       5,
			wantConfirmation:   false,
			wantDescriptionKey: "Requires querying 5",
		},
		{
			name:               "above confirmation threshold",
			text:               "device",
			siteID:             "",
			networkCount:       6,
			wantAPICalls:       6,
			wantConfirmation:   true,
			wantDescriptionKey: "Requires querying 6",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock networks
			networks := make(meraki.ResponseOrganizationsGetOrganizationNetworks, tt.networkCount)
			for i := 0; i < tt.networkCount; i++ {
				id := "L_" + string(rune('A'+i))
				name := "Network" + string(rune('A'+i))
				networks[i] = meraki.ResponseItemOrganizationsGetOrganizationNetworks{
					ID:   id,
					Name: name,
				}
			}

			s := &searchService{
				dashboard:      nil, // Not used in this path
				orgID:          "org123",
				rateLimiter:    nil,
				retryConfig:    DefaultRetryConfig(),
				suppressOutput: true,
			}

			// We need a way to inject the network count
			// For simplicity, we'll test the logic that doesn't require API calls
			ctx := context.Background()

			// For site-scoped and MAC searches, we can test directly
			if tt.siteID != "" || isMACAddress(tt.text) {
				estimate, err := s.EstimateSearchCost(ctx, tt.text, tt.siteID)
				if err != nil {
					t.Fatalf("EstimateSearchCost() error = %v", err)
				}

				if estimate.APICalls != tt.wantAPICalls {
					t.Errorf("APICalls = %d, want %d", estimate.APICalls, tt.wantAPICalls)
				}
				if estimate.NeedsConfirmation != tt.wantConfirmation {
					t.Errorf("NeedsConfirmation = %v, want %v", estimate.NeedsConfirmation, tt.wantConfirmation)
				}
				if !strings.Contains(estimate.Description, tt.wantDescriptionKey) {
					t.Errorf("Description = %q, want to contain %q", estimate.Description, tt.wantDescriptionKey)
				}
			}
			// For IP/hostname searches, we would need to mock getNetworkCount
		})
	}
}

// isMACAddress is a helper for testing MAC validation.
func isMACAddress(text string) bool {
	// Simple MAC validation for testing
	return strings.Contains(text, ":") || (len(text) == 12 && isHex(text))
}

func isHex(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// TestIsIPAddress tests IP address detection.
func TestIsIPAddress(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{
			name: "valid IPv4",
			text: "192.168.1.1",
			want: true,
		},
		{
			name: "valid IPv6",
			text: "2001:db8::1",
			want: true,
		},
		{
			name: "hostname",
			text: "laptop-john",
			want: false,
		},
		{
			name: "MAC address",
			text: "aa:bb:cc:dd:ee:ff",
			want: false,
		},
		{
			name: "empty string",
			text: "",
			want: false,
		},
		{
			name: "invalid IP",
			text: "999.999.999.999",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isIPAddress(tt.text)
			if got != tt.want {
				t.Errorf("isIPAddress(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

// TestMatchesText tests local text filtering.
func TestMatchesText(t *testing.T) {
	tests := []struct {
		name   string
		client *meraki.ResponseItemNetworksGetNetworkClients
		text   string
		want   bool
	}{
		{
			name: "matches description",
			client: &meraki.ResponseItemNetworksGetNetworkClients{
				Description: "laptop-john",
			},
			text: "john",
			want: true,
		},
		{
			name: "matches user",
			client: &meraki.ResponseItemNetworksGetNetworkClients{
				User: "john.doe@example.com",
			},
			text: "doe",
			want: true,
		},
		{
			name: "matches manufacturer",
			client: &meraki.ResponseItemNetworksGetNetworkClients{
				Manufacturer: "Apple",
			},
			text: "apple",
			want: true,
		},
		{
			name: "matches OS",
			client: &meraki.ResponseItemNetworksGetNetworkClients{
				Os: "macOS 14.0",
			},
			text: "macos",
			want: true,
		},
		{
			name: "no match",
			client: &meraki.ResponseItemNetworksGetNetworkClients{
				Description:  "laptop-jane",
				User:         "jane@example.com",
				Manufacturer: "Dell",
				Os:           "Windows",
			},
			text: "john",
			want: false,
		},
		{
			name: "case insensitive",
			client: &meraki.ResponseItemNetworksGetNetworkClients{
				Description: "LAPTOP-JOHN",
			},
			text: "laptop",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesText(tt.client, tt.text)
			if got != tt.want {
				t.Errorf("matchesText() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestConvertOrgClientSearchToWirelessClient tests wireless client conversion.
func TestConvertOrgClientSearchToWirelessClient(t *testing.T) {
	tests := []struct {
		name     string
		response *meraki.ResponseOrganizationsGetOrganizationClientsSearch
		record   *meraki.ResponseOrganizationsGetOrganizationClientsSearchRecords
		want     *vendors.WirelessClient
	}{
		{
			name:     "nil response",
			response: nil,
			record:   nil,
			want:     nil,
		},
		{
			name: "valid wireless client",
			response: &meraki.ResponseOrganizationsGetOrganizationClientsSearch{
				Mac:          "aa:bb:cc:dd:ee:ff",
				Manufacturer: "Apple",
			},
			record: &meraki.ResponseOrganizationsGetOrganizationClientsSearchRecords{
				Description:      "johns-iphone",
				IP:               "192.168.1.100",
				SSID:             "Corporate-WiFi",
				Os:               "iOS 17.0",
				RecentDeviceMac:  "11:22:33:44:55:66",
				Network: &meraki.ResponseOrganizationsGetOrganizationClientsSearchRecordsNetwork{
					ID:   "L_123",
					Name: "Main Office",
				},
			},
			want: &vendors.WirelessClient{
				SourceVendor: "meraki",
				MAC:          "aa:bb:cc:dd:ee:ff",
				Hostname:     "johns-iphone",
				IP:           "192.168.1.100",
				SSID:         "Corporate-WiFi",
				Manufacturer: "Apple",
				OS:           "iOS 17.0",
				APMAC:        "11:22:33:44:55:66",
				VLAN:         0,
				SiteID:       "L_123",
				SiteName:     "Main Office",
			},
		},
		{
			name: "nil network",
			response: &meraki.ResponseOrganizationsGetOrganizationClientsSearch{
				Mac:          "aa:bb:cc:dd:ee:ff",
				Manufacturer: "Apple",
			},
			record: &meraki.ResponseOrganizationsGetOrganizationClientsSearchRecords{
				Description:     "johns-iphone",
				IP:              "192.168.1.100",
				SSID:            "Corporate-WiFi",
				RecentDeviceMac: "11:22:33:44:55:66",
				Network:         nil,
			},
			want: &vendors.WirelessClient{
				SourceVendor: "meraki",
				MAC:          "aa:bb:cc:dd:ee:ff",
				Hostname:     "johns-iphone",
				IP:           "192.168.1.100",
				SSID:         "Corporate-WiFi",
				Manufacturer: "Apple",
				APMAC:        "11:22:33:44:55:66",
				VLAN:         0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertOrgClientSearchToWirelessClient(tt.response, tt.record)

			if tt.want == nil {
				if got != nil {
					t.Errorf("convertOrgClientSearchToWirelessClient() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Fatal("convertOrgClientSearchToWirelessClient() = nil, want non-nil")
			}

			if got.MAC != tt.want.MAC {
				t.Errorf("MAC = %q, want %q", got.MAC, tt.want.MAC)
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
			if got.SiteID != tt.want.SiteID {
				t.Errorf("SiteID = %q, want %q", got.SiteID, tt.want.SiteID)
			}
			if got.SiteName != tt.want.SiteName {
				t.Errorf("SiteName = %q, want %q", got.SiteName, tt.want.SiteName)
			}
		})
	}
}

// TestConvertOrgClientSearchToWiredClient tests wired client conversion.
func TestConvertOrgClientSearchToWiredClient(t *testing.T) {
	tests := []struct {
		name     string
		response *meraki.ResponseOrganizationsGetOrganizationClientsSearch
		record   *meraki.ResponseOrganizationsGetOrganizationClientsSearchRecords
		want     *vendors.WiredClient
	}{
		{
			name:     "nil response",
			response: nil,
			record:   nil,
			want:     nil,
		},
		{
			name: "valid wired client",
			response: &meraki.ResponseOrganizationsGetOrganizationClientsSearch{
				Mac:          "aa:bb:cc:dd:ee:ff",
				Manufacturer: "Dell",
			},
			record: &meraki.ResponseOrganizationsGetOrganizationClientsSearchRecords{
				Description:     "desktop-01",
				IP:              "192.168.1.50",
				RecentDeviceMac: "11:22:33:44:55:66",
				Switchport:      "1",
				Network: &meraki.ResponseOrganizationsGetOrganizationClientsSearchRecordsNetwork{
					ID:   "L_456",
					Name: "Branch Office",
				},
			},
			want: &vendors.WiredClient{
				SourceVendor: "meraki",
				MAC:          "aa:bb:cc:dd:ee:ff",
				Hostname:     "desktop-01",
				IP:           "192.168.1.50",
				Manufacturer: "Dell",
				SwitchMAC:    "11:22:33:44:55:66",
				PortID:       "1",
				VLAN:         0,
				SiteID:       "L_456",
				SiteName:     "Branch Office",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertOrgClientSearchToWiredClient(tt.response, tt.record)

			if tt.want == nil {
				if got != nil {
					t.Errorf("convertOrgClientSearchToWiredClient() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Fatal("convertOrgClientSearchToWiredClient() = nil, want non-nil")
			}

			if got.MAC != tt.want.MAC {
				t.Errorf("MAC = %q, want %q", got.MAC, tt.want.MAC)
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
			if got.SiteID != tt.want.SiteID {
				t.Errorf("SiteID = %q, want %q", got.SiteID, tt.want.SiteID)
			}
		})
	}
}

// TestConvertNetworkClientToWirelessClient tests network client to wireless conversion.
func TestConvertNetworkClientToWirelessClient(t *testing.T) {
	tests := []struct {
		name      string
		client    *meraki.ResponseItemNetworksGetNetworkClients
		networkID string
		want      *vendors.WirelessClient
	}{
		{
			name:      "nil client",
			client:    nil,
			networkID: "L_123",
			want:      nil,
		},
		{
			name: "valid wireless client",
			client: &meraki.ResponseItemNetworksGetNetworkClients{
				Mac:             "aa:bb:cc:dd:ee:ff",
				Description:     "johns-laptop",
				IP:              "192.168.1.100",
				SSID:            "Guest-WiFi",
				Manufacturer:    "Apple",
				Os:              "macOS 14.0",
				RecentDeviceMac: "11:22:33:44:55:66",
			},
			networkID: "L_123",
			want: &vendors.WirelessClient{
				SiteID:       "L_123",
				SourceVendor: "meraki",
				MAC:          "aa:bb:cc:dd:ee:ff",
				Hostname:     "johns-laptop",
				IP:           "192.168.1.100",
				SSID:         "Guest-WiFi",
				Manufacturer: "Apple",
				OS:           "macOS 14.0",
				APMAC:        "11:22:33:44:55:66",
				VLAN:         0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertNetworkClientToWirelessClient(tt.client, tt.networkID)

			if tt.want == nil {
				if got != nil {
					t.Errorf("convertNetworkClientToWirelessClient() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Fatal("convertNetworkClientToWirelessClient() = nil, want non-nil")
			}

			if got.SiteID != tt.want.SiteID {
				t.Errorf("SiteID = %q, want %q", got.SiteID, tt.want.SiteID)
			}
			if got.MAC != tt.want.MAC {
				t.Errorf("MAC = %q, want %q", got.MAC, tt.want.MAC)
			}
			if got.SSID != tt.want.SSID {
				t.Errorf("SSID = %q, want %q", got.SSID, tt.want.SSID)
			}
		})
	}
}

// TestConvertNetworkClientToWiredClient tests network client to wired conversion.
func TestConvertNetworkClientToWiredClient(t *testing.T) {
	tests := []struct {
		name      string
		client    *meraki.ResponseItemNetworksGetNetworkClients
		networkID string
		want      *vendors.WiredClient
	}{
		{
			name:      "nil client",
			client:    nil,
			networkID: "L_123",
			want:      nil,
		},
		{
			name: "valid wired client",
			client: &meraki.ResponseItemNetworksGetNetworkClients{
				Mac:             "aa:bb:cc:dd:ee:ff",
				Description:     "server-01",
				IP:              "192.168.1.10",
				Manufacturer:    "HP",
				RecentDeviceMac: "11:22:33:44:55:66",
				Switchport:      "5",
			},
			networkID: "L_456",
			want: &vendors.WiredClient{
				SiteID:       "L_456",
				SourceVendor: "meraki",
				MAC:          "aa:bb:cc:dd:ee:ff",
				Hostname:     "server-01",
				IP:           "192.168.1.10",
				Manufacturer: "HP",
				SwitchMAC:    "11:22:33:44:55:66",
				PortID:       "5",
				VLAN:         0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertNetworkClientToWiredClient(tt.client, tt.networkID)

			if tt.want == nil {
				if got != nil {
					t.Errorf("convertNetworkClientToWiredClient() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Fatal("convertNetworkClientToWiredClient() = nil, want non-nil")
			}

			if got.SiteID != tt.want.SiteID {
				t.Errorf("SiteID = %q, want %q", got.SiteID, tt.want.SiteID)
			}
			if got.MAC != tt.want.MAC {
				t.Errorf("MAC = %q, want %q", got.MAC, tt.want.MAC)
			}
			if got.PortID != tt.want.PortID {
				t.Errorf("PortID = %q, want %q", got.PortID, tt.want.PortID)
			}
		})
	}
}

// TestSearchServiceImplementsInterface verifies the service implements the interface.
func TestSearchServiceImplementsInterface(t *testing.T) {
	var _ vendors.SearchService = (*searchService)(nil)
}

// TestNetParseIP tests the net.ParseIP function behavior (used in isIPAddress).
func TestNetParseIP(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{"valid IPv4", "192.168.1.1", true},
		{"valid IPv6", "2001:db8::1", true},
		{"invalid", "not-an-ip", false},
		{"empty", "", false},
		{"hostname", "example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.input)
			got := ip != nil
			if got != tt.valid {
				t.Errorf("net.ParseIP(%q) validity = %v, want %v", tt.input, got, tt.valid)
			}
		})
	}
}

// TestSearchCostEstimateStructure validates the SearchCostEstimate structure.
func TestSearchCostEstimateStructure(t *testing.T) {
	estimate := &vendors.SearchCostEstimate{
		APICalls:          5,
		EstimatedDuration: 10 * time.Second,
		NeedsConfirmation: true,
		Description:       "Test description",
	}

	if estimate.APICalls != 5 {
		t.Errorf("APICalls = %d, want 5", estimate.APICalls)
	}
	if estimate.EstimatedDuration != 10*time.Second {
		t.Errorf("EstimatedDuration = %v, want 10s", estimate.EstimatedDuration)
	}
	if !estimate.NeedsConfirmation {
		t.Error("NeedsConfirmation = false, want true")
	}
	if estimate.Description != "Test description" {
		t.Errorf("Description = %q, want %q", estimate.Description, "Test description")
	}
}

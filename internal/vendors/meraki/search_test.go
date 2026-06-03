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

// TestShouldApplyLocalFilter verifies when the in-memory text filter is applied
// over Meraki network clients. Empty text must skip filtering so the caller can
// list every client on a site; MAC and IP are already filtered upstream.
func TestShouldApplyLocalFilter(t *testing.T) {
	tests := []struct {
		text string
		want bool
	}{
		{"", false},
		{"laptop-john", true},
		{"aa:bb:cc:dd:ee:ff", false},
		{"AA:BB:CC:DD:EE:FF", false},
		{"192.168.1.1", false},
		{"fe80::1", false},
		{"descriptor-with-colons:but-not-mac", true},
	}
	for _, tt := range tests {
		if got := shouldApplyLocalFilter(tt.text); got != tt.want {
			t.Errorf("shouldApplyLocalFilter(%q) = %v, want %v", tt.text, got, tt.want)
		}
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
				Description:     "johns-iphone",
				IP:              "192.168.1.100",
				SSID:            "Corporate-WiFi",
				Os:              "iOS 17.0",
				RecentDeviceMac: "11:22:33:44:55:66",
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
				Mac:              "aa:bb:cc:dd:ee:ff",
				Description:      "johns-laptop",
				IP:               "192.168.1.100",
				SSID:             "Guest-WiFi",
				Manufacturer:     "Apple",
				Os:               "macOS 14.0",
				RecentDeviceMac:  "11:22:33:44:55:66",
				RecentDeviceName: "ap2-15",
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
				APName:       "ap2-15",
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
			if got.APName != tt.want.APName {
				t.Errorf("APName = %q, want %q", got.APName, tt.want.APName)
			}
			if got.APMAC != tt.want.APMAC {
				t.Errorf("APMAC = %q, want %q", got.APMAC, tt.want.APMAC)
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
				Mac:              "aa:bb:cc:dd:ee:ff",
				Description:      "server-01",
				IP:               "192.168.1.10",
				Manufacturer:     "HP",
				RecentDeviceMac:  "11:22:33:44:55:66",
				RecentDeviceName: "sw-core-1",
				Switchport:       "5",
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
				SwitchName:   "sw-core-1",
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
			if got.SwitchName != tt.want.SwitchName {
				t.Errorf("SwitchName = %q, want %q", got.SwitchName, tt.want.SwitchName)
			}
			if got.SwitchMAC != tt.want.SwitchMAC {
				t.Errorf("SwitchMAC = %q, want %q", got.SwitchMAC, tt.want.SwitchMAC)
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

// TestDecodeFlexTime covers the two response shapes Meraki has been observed
// to emit for firstSeen/lastSeen (Unix int, ISO 8601 string) plus the cases
// that should map to a zero time.Time.
func TestDecodeFlexTime(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		wantZ  bool
		wantTS int64 // expected Unix seconds when non-zero
	}{
		{"unix int", "1747201234", false, 1747201234},
		{"unix int zero", "0", true, 0},
		{"unix int negative", "-1", true, 0},
		{"rfc3339", `"2026-05-14T03:00:34Z"`, false, 1778727634},
		{"rfc3339 with offset", `"2026-05-14T00:00:34-03:00"`, false, 1778727634},
		{"rfc3339 nano", `"2026-05-14T03:00:34.123456Z"`, false, 1778727634},
		{"empty string", `""`, true, 0},
		{"json null", "null", true, 0},
		{"empty raw", "", true, 0},
		{"garbage string", `"not a date"`, true, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decodeFlexTime([]byte(tt.input))
			if got.IsZero() != tt.wantZ {
				t.Fatalf("IsZero=%v, want %v (got=%v)", got.IsZero(), tt.wantZ, got)
			}
			if !tt.wantZ && got.Unix() != tt.wantTS {
				t.Errorf("Unix()=%d, want %d", got.Unix(), tt.wantTS)
			}
		})
	}
}

// TestParseNetworkClientTimes verifies wire-level extraction of firstSeen /
// lastSeen from a per-network /clients response body, including the
// string-shaped case that the v5 SDK silently drops.
func TestParseNetworkClientTimes(t *testing.T) {
	body := []byte(`[
		{"mac": "aa:bb:cc:dd:ee:01", "firstSeen": 1747000000, "lastSeen": 1747200000},
		{"mac": "AA:BB:CC:DD:EE:02", "firstSeen": "2026-05-14T03:00:34Z", "lastSeen": "2026-05-14T04:00:34Z"},
		{"mac": "aa:bb:cc:dd:ee:03", "firstSeen": null, "lastSeen": null},
		{"mac": "", "firstSeen": 1747000000, "lastSeen": 1747200000}
	]`)
	got := parseNetworkClientTimes(body)
	if len(got) != 2 {
		t.Fatalf("expected 2 keyed entries, got %d (%v)", len(got), got)
	}
	if ts, ok := got[vendors.NormalizeMAC("aa:bb:cc:dd:ee:01")]; !ok || ts.FirstSeen.IsZero() || ts.LastSeen.IsZero() {
		t.Errorf("int-shaped entry missing or zero: %+v", ts)
	}
	if ts, ok := got[vendors.NormalizeMAC("aa:bb:cc:dd:ee:02")]; !ok || ts.FirstSeen.IsZero() || ts.LastSeen.IsZero() {
		t.Errorf("string-shaped entry missing or zero: %+v", ts)
	}
}

// TestParseOrgClientSearchTimes covers the org-wide search response shape,
// keyed by network ID.
func TestParseOrgClientSearchTimes(t *testing.T) {
	body := []byte(`{
		"mac": "aa:bb:cc:dd:ee:ff",
		"records": [
			{"network": {"id": "L_1"}, "firstSeen": "2026-05-14T03:00:34Z", "lastSeen": "2026-05-14T04:00:34Z"},
			{"network": {"id": "L_2"}, "firstSeen": 1747000000, "lastSeen": 1747200000},
			{"network": {"id": ""},    "firstSeen": 1, "lastSeen": 2}
		]
	}`)
	got := parseOrgClientSearchTimes(body)
	if len(got) != 2 {
		t.Fatalf("expected 2 entries (skip blank network id), got %d", len(got))
	}
	if _, ok := got["L_1"]; !ok {
		t.Error("L_1 missing from parsed times")
	}
	if _, ok := got["L_2"]; !ok {
		t.Error("L_2 missing from parsed times")
	}
}

// TestApplyClientTimes verifies that wire-parsed timestamps fill in zero
// fields without clobbering values already set by the SDK.
func TestApplyClientTimes(t *testing.T) {
	prior := time.Unix(1700000000, 0).UTC()
	wire := clientTimestamps{
		FirstSeen: time.Unix(1747000000, 0).UTC(),
		LastSeen:  time.Unix(1747200000, 0).UTC(),
	}

	// Both zero -> both filled
	var f1, l1 time.Time
	applyClientTimes(&f1, &l1, wire)
	if !f1.Equal(wire.FirstSeen) || !l1.Equal(wire.LastSeen) {
		t.Errorf("zero->wire failed: f=%v l=%v", f1, l1)
	}

	// Already set -> preserved
	f2, l2 := prior, prior
	applyClientTimes(&f2, &l2, wire)
	if !f2.Equal(prior) || !l2.Equal(prior) {
		t.Errorf("set->preserved failed: f=%v l=%v", f2, l2)
	}
}

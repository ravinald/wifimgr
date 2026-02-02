package netbox

import (
	"strings"
	"testing"

	"github.com/ravinald/wifimgr/internal/vendors"
)

func boolPtr(b bool) *bool {
	return &b
}

func TestMapperToDeviceRequest(t *testing.T) {
	cfg := &Config{
		Mappings: DefaultMappings(),
	}
	mapper := NewMapper(cfg, nil)

	// Valid validation result
	validation := &DeviceValidationResult{
		Valid:        true,
		SiteID:       100,
		DeviceTypeID: 200,
		DeviceRoleID: 300,
	}

	item := &vendors.InventoryItem{
		Name:         "AP-LOBBY-01",
		MAC:          "001122334455",
		Serial:       "SN123456",
		Model:        "AP43",
		Type:         "ap",
		SiteName:     "US-LAB-01",
		SourceAPI:    "mist",
		SourceVendor: "juniper",
	}

	req, err := mapper.ToDeviceRequest(item, validation)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if req.Name != "AP-LOBBY-01" {
		t.Errorf("Expected name 'AP-LOBBY-01', got '%s'", req.Name)
	}
	if req.Serial != "SN123456" {
		t.Errorf("Expected serial 'SN123456', got '%s'", req.Serial)
	}
	if req.Site != 100 {
		t.Errorf("Expected site ID 100, got %d", req.Site)
	}
	if req.DeviceType != 200 {
		t.Errorf("Expected device type ID 200, got %d", req.DeviceType)
	}
	if req.Role != 300 {
		t.Errorf("Expected role ID 300, got %d", req.Role)
	}
	if req.Status != "active" {
		t.Errorf("Expected status 'active', got '%s'", req.Status)
	}
}

func TestMapperToDeviceRequestInvalid(t *testing.T) {
	cfg := &Config{
		Mappings: DefaultMappings(),
	}
	mapper := NewMapper(cfg, nil)

	// Invalid validation result
	validation := &DeviceValidationResult{
		Valid:  false,
		Errors: []string{"site not found"},
	}

	item := &vendors.InventoryItem{
		Name: "AP-LOBBY-01",
		MAC:  "001122334455",
	}

	_, err := mapper.ToDeviceRequest(item, validation)
	if err == nil {
		t.Error("Expected error for invalid validation")
	}
}

func TestMapperGenerateDeviceName(t *testing.T) {
	cfg := &Config{
		Mappings: DefaultMappings(),
	}
	mapper := NewMapper(cfg, nil)

	tests := []struct {
		name     string
		item     *vendors.InventoryItem
		contains string
	}{
		{
			name: "uses provided name",
			item: &vendors.InventoryItem{
				Name: "AP-CUSTOM",
				MAC:  "001122334455",
				Type: "ap",
			},
			contains: "AP-CUSTOM",
		},
		{
			name: "generates from MAC when no name",
			item: &vendors.InventoryItem{
				Name: "",
				MAC:  "001122334455",
				Type: "ap",
			},
			contains: "AP-",
		},
		{
			name: "generates from serial as fallback",
			item: &vendors.InventoryItem{
				Name:   "",
				MAC:    "",
				Serial: "SN123",
				Type:   "switch",
			},
			contains: "SWITCH-SN123",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			validation := &DeviceValidationResult{
				Valid:        true,
				SiteID:       1,
				DeviceTypeID: 2,
				DeviceRoleID: 3,
			}

			req, err := mapper.ToDeviceRequest(tc.item, validation)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if !strings.Contains(req.Name, tc.contains) {
				t.Errorf("Expected name to contain '%s', got '%s'", tc.contains, req.Name)
			}
		})
	}
}

func TestMapperToInterfaceRequest(t *testing.T) {
	cfg := &Config{
		Mappings: DefaultMappings(),
	}
	mapper := NewMapper(cfg, nil)

	tests := []struct {
		name         string
		deviceType   string
		expectedType string
		expectedName string
	}{
		// All devices now use the configured eth0 mapping (default: eth0, 1000base-t)
		// Device-specific names can be configured in netbox.mappings.interfaces
		{"AP interface", "ap", "1000base-t", "eth0"},
		{"Switch interface", "switch", "1000base-t", "eth0"},
		{"Gateway interface", "gateway", "1000base-t", "eth0"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			item := &vendors.InventoryItem{
				MAC:  "001122334455",
				Type: tc.deviceType,
			}

			req, err := mapper.ToInterfaceRequest(123, item)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if req.Device != 123 {
				t.Errorf("Expected device ID 123, got %d", req.Device)
			}
			if req.Type != tc.expectedType {
				t.Errorf("Expected type '%s', got '%s'", tc.expectedType, req.Type)
			}
			if req.Name != tc.expectedName {
				t.Errorf("Expected name '%s', got '%s'", tc.expectedName, req.Name)
			}
			if !req.Enabled {
				t.Error("Expected interface to be enabled")
			}
		})
	}
}

func TestMapperToIPAddressRequest(t *testing.T) {
	cfg := &Config{
		Mappings: DefaultMappings(),
	}
	mapper := NewMapper(cfg, nil)

	tests := []struct {
		name     string
		ip       string
		expected string
	}{
		{"IP without CIDR", "192.168.1.1", "192.168.1.1/32"},
		{"IP with CIDR", "192.168.1.1/24", "192.168.1.1/24"},
		{"IPv6 without CIDR", "2001:db8::1", "2001:db8::1/32"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := mapper.ToIPAddressRequest(456, tc.ip)

			if req.Address != tc.expected {
				t.Errorf("Expected address '%s', got '%s'", tc.expected, req.Address)
			}
			if req.AssignedObjectID != 456 {
				t.Errorf("Expected assigned object ID 456, got %d", req.AssignedObjectID)
			}
			if req.AssignedObjectType != "dcim.interface" {
				t.Errorf("Expected type 'dcim.interface', got '%s'", req.AssignedObjectType)
			}
			if req.Status != "active" {
				t.Errorf("Expected status 'active', got '%s'", req.Status)
			}
		})
	}
}

func TestMapperCustomFields(t *testing.T) {
	cfg := &Config{
		Mappings: DefaultMappings(),
	}
	mapper := NewMapper(cfg, nil)

	item := &vendors.InventoryItem{
		ID:           "vendor-uuid-123",
		Name:         "AP-TEST",
		MAC:          "001122334455",
		Type:         "ap",
		SourceAPI:    "mist",
		SourceVendor: "juniper",
	}

	validation := &DeviceValidationResult{
		Valid:        true,
		SiteID:       1,
		DeviceTypeID: 2,
		DeviceRoleID: 3,
	}

	req, err := mapper.ToDeviceRequest(item, validation)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if req.CustomFields == nil {
		t.Fatal("Expected custom fields to be set")
	}

	if req.CustomFields["wifimgr_source_api"] != "mist" {
		t.Errorf("Expected source_api 'mist', got '%v'", req.CustomFields["wifimgr_source_api"])
	}
	if req.CustomFields["wifimgr_source_vendor"] != "juniper" {
		t.Errorf("Expected source_vendor 'juniper', got '%v'", req.CustomFields["wifimgr_source_vendor"])
	}
	if req.CustomFields["wifimgr_vendor_id"] != "vendor-uuid-123" {
		t.Errorf("Expected vendor_id 'vendor-uuid-123', got '%v'", req.CustomFields["wifimgr_vendor_id"])
	}
}

func TestMapperTagConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		tag         string
		expectTags  bool
		expectedTag string
	}{
		{
			name:        "tag configured",
			tag:         "wifimgr-managed",
			expectTags:  true,
			expectedTag: "wifimgr-managed",
		},
		{
			name:       "no tag configured",
			tag:        "",
			expectTags: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{
				Mappings: MappingConfig{
					Tag:          tc.tag,
					DefaultRoles: DefaultMappings().DefaultRoles,
					DeviceTypes:  make(map[string]DeviceTypeMapping),
				},
			}
			mapper := NewMapper(cfg, nil)

			validation := &DeviceValidationResult{
				Valid:        true,
				SiteID:       1,
				DeviceTypeID: 2,
				DeviceRoleID: 3,
			}

			item := &vendors.InventoryItem{
				Name: "AP-TEST",
				MAC:  "001122334455",
				Type: "ap",
			}

			// Test device request
			deviceReq, err := mapper.ToDeviceRequest(item, validation)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tc.expectTags {
				if len(deviceReq.Tags) != 1 {
					t.Errorf("Expected 1 tag, got %d", len(deviceReq.Tags))
				} else if deviceReq.Tags[0].Name != tc.expectedTag {
					t.Errorf("Expected tag '%s', got '%s'", tc.expectedTag, deviceReq.Tags[0].Name)
				}
			} else {
				if len(deviceReq.Tags) != 0 {
					t.Errorf("Expected no tags, got %d", len(deviceReq.Tags))
				}
			}

			// Test interface request
			interfaceReq, err := mapper.ToInterfaceRequest(123, item)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tc.expectTags {
				if len(interfaceReq.Tags) != 1 {
					t.Errorf("Expected 1 tag on interface, got %d", len(interfaceReq.Tags))
				} else if interfaceReq.Tags[0].Name != tc.expectedTag {
					t.Errorf("Expected interface tag '%s', got '%s'", tc.expectedTag, interfaceReq.Tags[0].Name)
				}
			} else {
				if len(interfaceReq.Tags) != 0 {
					t.Errorf("Expected no tags on interface, got %d", len(interfaceReq.Tags))
				}
			}
		})
	}
}

func TestMapperToRadioInterfaceRequests(t *testing.T) {
	cfg := &Config{
		Mappings: MappingConfig{
			Tag: "wifimgr",
		},
	}
	mapper := NewMapper(cfg, nil)

	tests := []struct {
		name           string
		radioConfig    *vendors.RadioConfig
		expectedCount  int
		expectedNames  []string
		expectedTypes  []string
		expectedRFRole string
	}{
		{
			name: "dual band AP",
			radioConfig: &vendors.RadioConfig{
				Band24: &vendors.RadioBandConfig{},
				Band5:  &vendors.RadioBandConfig{},
			},
			expectedCount:  2,
			expectedNames:  []string{"wifi0", "wifi1"},
			expectedTypes:  []string{"ieee802.11n", "ieee802.11ac"},
			expectedRFRole: "ap",
		},
		{
			name: "tri-band AP with WiFi 6E",
			radioConfig: &vendors.RadioConfig{
				Band24: &vendors.RadioBandConfig{},
				Band5:  &vendors.RadioBandConfig{},
				Band6:  &vendors.RadioBandConfig{},
			},
			expectedCount:  3,
			expectedNames:  []string{"wifi0", "wifi1", "wifi2"},
			expectedTypes:  []string{"ieee802.11n", "ieee802.11ac", "ieee802.11ax"},
			expectedRFRole: "ap",
		},
		{
			name: "5GHz only AP",
			radioConfig: &vendors.RadioConfig{
				Band5: &vendors.RadioBandConfig{},
			},
			expectedCount: 1,
			expectedNames: []string{"wifi1"},
			expectedTypes: []string{"ieee802.11ac"},
		},
		{
			name: "disabled 2.4GHz radio",
			radioConfig: &vendors.RadioConfig{
				Band24: &vendors.RadioBandConfig{Disabled: boolPtr(true)},
				Band5:  &vendors.RadioBandConfig{Disabled: boolPtr(false)},
			},
			expectedCount: 2,
			expectedNames: []string{"wifi0", "wifi1"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reqs, err := mapper.ToRadioInterfaceRequests(100, tc.radioConfig, nil)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(reqs) != tc.expectedCount {
				t.Errorf("Expected %d interfaces, got %d", tc.expectedCount, len(reqs))
			}

			for i, req := range reqs {
				if i < len(tc.expectedNames) && req.Name != tc.expectedNames[i] {
					t.Errorf("Expected name '%s', got '%s'", tc.expectedNames[i], req.Name)
				}
				if i < len(tc.expectedTypes) && req.Type != tc.expectedTypes[i] {
					t.Errorf("Expected type '%s', got '%s'", tc.expectedTypes[i], req.Type)
				}
				if req.Device != 100 {
					t.Errorf("Expected device ID 100, got %d", req.Device)
				}
				if req.RFRole != "ap" {
					t.Errorf("Expected RF role 'ap', got '%s'", req.RFRole)
				}
				if len(req.Tags) != 1 || req.Tags[0].Name != "wifimgr" {
					t.Errorf("Expected tag 'wifimgr'")
				}
			}
		})
	}
}

func TestMapperToVirtualWLANInterfaceRequests(t *testing.T) {
	cfg := &Config{
		Mappings: MappingConfig{
			Tag: "wifimgr",
		},
	}
	mapper := NewMapper(cfg, nil)

	radioInterfaces := map[string]int64{
		"wifi0": 1001,
		"wifi1": 1002,
	}

	wlans := []*vendors.WLAN{
		{SSID: "Corporate", Enabled: true, Band: "dual"},
		{SSID: "Guest", Enabled: true, Band: "2.4"},
	}

	wirelessLANIDs := map[string]int64{
		"Corporate": 2001,
		"Guest":     2002,
	}

	reqs := mapper.ToVirtualWLANInterfaceRequests(100, radioInterfaces, wlans, wirelessLANIDs)

	// Corporate WLAN (dual band) = wifi0.0, wifi1.0
	// Guest WLAN (2.4GHz only) = wifi0.1
	// Total = 3 interfaces
	expectedCount := 3
	if len(reqs) != expectedCount {
		t.Errorf("Expected %d virtual interfaces, got %d", expectedCount, len(reqs))
	}

	// Check that all have correct type and parent
	for _, req := range reqs {
		if req.Type != "virtual" {
			t.Errorf("Expected type 'virtual', got '%s'", req.Type)
		}
		if req.Parent == nil {
			t.Error("Expected parent to be set")
		}
		if len(req.Tags) != 1 || req.Tags[0].Name != "wifimgr" {
			t.Errorf("Expected tag 'wifimgr'")
		}
	}

	// Check wireless LAN linkage
	foundCorporateLink := false
	foundGuestLink := false
	for _, req := range reqs {
		if len(req.WirelessLANs) > 0 {
			if req.WirelessLANs[0] == 2001 {
				foundCorporateLink = true
			}
			if req.WirelessLANs[0] == 2002 {
				foundGuestLink = true
			}
		}
	}

	if !foundCorporateLink {
		t.Error("Expected to find Corporate WLAN linked")
	}
	if !foundGuestLink {
		t.Error("Expected to find Guest WLAN linked")
	}
}

func TestMapperGetRadiosForWLAN(t *testing.T) {
	cfg := &Config{
		Mappings: DefaultMappings(),
	}
	mapper := NewMapper(cfg, nil)

	tests := []struct {
		band     string
		expected []string
	}{
		{"2.4", []string{"wifi0"}},
		{"5", []string{"wifi1"}},
		{"6", []string{"wifi2"}},
		{"dual", []string{"wifi0", "wifi1"}},
		{"all", []string{"wifi0", "wifi1"}},
		{"", []string{"wifi0", "wifi1"}},
	}

	for _, tc := range tests {
		t.Run("band_"+tc.band, func(t *testing.T) {
			result := mapper.getRadiosForWLAN(tc.band)
			if len(result) != len(tc.expected) {
				t.Errorf("Expected %d radios, got %d", len(tc.expected), len(result))
			}
			for i, r := range result {
				if i < len(tc.expected) && r != tc.expected[i] {
					t.Errorf("Expected radio '%s', got '%s'", tc.expected[i], r)
				}
			}
		})
	}
}

func TestMapperToWirelessLANRequest(t *testing.T) {
	cfg := &Config{
		Mappings: MappingConfig{
			Tag: "wifimgr",
		},
	}
	mapper := NewMapper(cfg, nil)

	tests := []struct {
		name         string
		wlan         *vendors.WLAN
		expectedAuth string
	}{
		{
			name:         "open network",
			wlan:         &vendors.WLAN{SSID: "Open", AuthType: "open"},
			expectedAuth: "open",
		},
		{
			name:         "PSK network",
			wlan:         &vendors.WLAN{SSID: "Secure", AuthType: "psk"},
			expectedAuth: "wpa-personal",
		},
		{
			name:         "WPA2 enterprise",
			wlan:         &vendors.WLAN{SSID: "Corp", AuthType: "wpa2-enterprise"},
			expectedAuth: "wpa-enterprise",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := mapper.ToWirelessLANRequest(tc.wlan)

			if req.SSID != tc.wlan.SSID {
				t.Errorf("Expected SSID '%s', got '%s'", tc.wlan.SSID, req.SSID)
			}
			if req.AuthType != tc.expectedAuth {
				t.Errorf("Expected auth type '%s', got '%s'", tc.expectedAuth, req.AuthType)
			}
			if req.Status != "active" {
				t.Errorf("Expected status 'active', got '%s'", req.Status)
			}
			if len(req.Tags) != 1 || req.Tags[0].Name != "wifimgr" {
				t.Error("Expected tag 'wifimgr'")
			}
		})
	}
}

func TestMapperSettingsSourceCustomField(t *testing.T) {
	cfg := &Config{
		Mappings: DefaultMappings(),
	}
	mapper := NewMapper(cfg, nil)

	item := &vendors.InventoryItem{
		ID:           "vendor-uuid-123",
		Name:         "AP-TEST",
		MAC:          "001122334455",
		Type:         "ap",
		SourceAPI:    "mist",
		SourceVendor: "juniper",
	}

	validation := &DeviceValidationResult{
		Valid:        true,
		SiteID:       1,
		DeviceTypeID: 2,
		DeviceRoleID: 3,
	}

	req, err := mapper.ToDeviceRequest(item, validation)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check settings_source field
	if req.CustomFields["settings_source"] != "internal" {
		t.Errorf("Expected settings_source 'internal', got '%v'", req.CustomFields["settings_source"])
	}
}

func TestMapperCustomInterfaceMappings(t *testing.T) {
	// Test custom interface name and type mappings
	cfg := &Config{
		Mappings: MappingConfig{
			Interfaces: InterfaceMappings{
				Eth0:   &InterfaceMapping{Name: "mgmt0", Type: "10gbase-t"},
				Radio0: &InterfaceMapping{Name: "wlan0", Type: "ieee802.11ax"},
				Radio1: &InterfaceMapping{Name: "wlan1", Type: "ieee802.11ax"},
			},
		},
	}
	mapper := NewMapper(cfg, nil)

	t.Run("custom eth0 interface", func(t *testing.T) {
		item := &vendors.InventoryItem{
			MAC:  "001122334455",
			Type: "ap",
		}

		req, err := mapper.ToInterfaceRequest(123, item)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if req.Name != "mgmt0" {
			t.Errorf("Expected custom name 'mgmt0', got '%s'", req.Name)
		}
		if req.Type != "10gbase-t" {
			t.Errorf("Expected custom type '10gbase-t', got '%s'", req.Type)
		}
	})

	t.Run("custom radio interfaces", func(t *testing.T) {
		radioConfig := &vendors.RadioConfig{
			Band24: &vendors.RadioBandConfig{},
			Band5:  &vendors.RadioBandConfig{},
		}

		reqs, err := mapper.ToRadioInterfaceRequests(100, radioConfig, nil)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(reqs) != 2 {
			t.Fatalf("Expected 2 radio interfaces, got %d", len(reqs))
		}

		// Check 2.4GHz radio uses custom mapping
		if reqs[0].Name != "wlan0" {
			t.Errorf("Expected radio0 name 'wlan0', got '%s'", reqs[0].Name)
		}
		if reqs[0].Type != "ieee802.11ax" {
			t.Errorf("Expected radio0 type 'ieee802.11ax', got '%s'", reqs[0].Type)
		}

		// Check 5GHz radio uses custom mapping
		if reqs[1].Name != "wlan1" {
			t.Errorf("Expected radio1 name 'wlan1', got '%s'", reqs[1].Name)
		}
	})
}

func TestMapperInvalidInterfaceType(t *testing.T) {
	// Test that invalid interface types are rejected
	cfg := &Config{
		Mappings: MappingConfig{
			Interfaces: InterfaceMappings{
				Eth0: &InterfaceMapping{Name: "eth0", Type: "invalid-type"},
			},
		},
	}
	mapper := NewMapper(cfg, nil)

	item := &vendors.InventoryItem{
		MAC:  "001122334455",
		Type: "ap",
		Name: "AP-TEST-01",
	}

	_, err := mapper.ToInterfaceRequest(123, item)
	if err == nil {
		t.Error("Expected error for invalid interface type")
	}

	// Check it's an InterfaceTypeError
	if _, ok := err.(*InterfaceTypeError); !ok {
		t.Errorf("Expected InterfaceTypeError, got %T", err)
	}
}

func TestMapperRadioInterfaceInvalidType(t *testing.T) {
	// Test that invalid radio interface types are rejected
	cfg := &Config{
		Mappings: MappingConfig{
			Interfaces: InterfaceMappings{
				Radio0: &InterfaceMapping{Name: "wifi0", Type: "not-a-valid-type"},
			},
		},
	}
	mapper := NewMapper(cfg, nil)

	radioConfig := &vendors.RadioConfig{
		Band24: &vendors.RadioBandConfig{},
	}

	_, err := mapper.ToRadioInterfaceRequests(100, radioConfig, nil)
	if err == nil {
		t.Error("Expected error for invalid radio interface type")
	}

	// Check it's an InterfaceTypeError
	if _, ok := err.(*InterfaceTypeError); !ok {
		t.Errorf("Expected InterfaceTypeError, got %T", err)
	}
}

func TestMapperDeviceLevelInterfaceOverrides(t *testing.T) {
	// Test device-level interface overrides take priority over global config
	cfg := &Config{
		Mappings: MappingConfig{
			Interfaces: InterfaceMappings{
				Eth0:   &InterfaceMapping{Name: "global-eth0", Type: "1000base-t"},
				Radio0: &InterfaceMapping{Name: "global-wifi0", Type: "ieee802.11n"},
			},
		},
	}
	mapper := NewMapper(cfg, nil)

	t.Run("device-level eth0 override", func(t *testing.T) {
		item := &vendors.InventoryItem{
			MAC:  "001122334455",
			Type: "ap",
			Name: "AP-TEST",
			NetBox: &vendors.NetBoxDeviceExtension{
				Interfaces: map[string]*vendors.NetBoxInterfaceMapping{
					"eth0": {Name: "device-mgmt0", Type: "10gbase-t"},
				},
			},
		}

		req, err := mapper.ToInterfaceRequest(123, item)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Device-level should override global
		if req.Name != "device-mgmt0" {
			t.Errorf("Expected device-level name 'device-mgmt0', got '%s'", req.Name)
		}
		if req.Type != "10gbase-t" {
			t.Errorf("Expected device-level type '10gbase-t', got '%s'", req.Type)
		}
	})

	t.Run("device-level partial override", func(t *testing.T) {
		// Device-level only overrides name, type should come from global
		item := &vendors.InventoryItem{
			MAC:  "001122334455",
			Type: "ap",
			Name: "AP-TEST",
			NetBox: &vendors.NetBoxDeviceExtension{
				Interfaces: map[string]*vendors.NetBoxInterfaceMapping{
					"eth0": {Name: "device-eth0"}, // Only name, no type
				},
			},
		}

		req, err := mapper.ToInterfaceRequest(123, item)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Name from device, type from global
		if req.Name != "device-eth0" {
			t.Errorf("Expected device-level name 'device-eth0', got '%s'", req.Name)
		}
		if req.Type != "1000base-t" {
			t.Errorf("Expected global type '1000base-t', got '%s'", req.Type)
		}
	})

	t.Run("device-level radio overrides", func(t *testing.T) {
		item := &vendors.InventoryItem{
			MAC:  "001122334455",
			Type: "ap",
			Name: "AP-TEST",
			NetBox: &vendors.NetBoxDeviceExtension{
				Interfaces: map[string]*vendors.NetBoxInterfaceMapping{
					"radio0": {Name: "device-wlan0", Type: "ieee802.11ax"},
				},
			},
		}

		radioConfig := &vendors.RadioConfig{
			Band24: &vendors.RadioBandConfig{},
		}

		reqs, err := mapper.ToRadioInterfaceRequests(100, radioConfig, item)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(reqs) != 1 {
			t.Fatalf("Expected 1 radio interface, got %d", len(reqs))
		}

		// Device-level should override global
		if reqs[0].Name != "device-wlan0" {
			t.Errorf("Expected device-level name 'device-wlan0', got '%s'", reqs[0].Name)
		}
		if reqs[0].Type != "ieee802.11ax" {
			t.Errorf("Expected device-level type 'ieee802.11ax', got '%s'", reqs[0].Type)
		}
	})

	t.Run("no device-level uses global", func(t *testing.T) {
		item := &vendors.InventoryItem{
			MAC:  "001122334455",
			Type: "ap",
			Name: "AP-TEST",
			// No NetBox extension
		}

		req, err := mapper.ToInterfaceRequest(123, item)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Should use global config
		if req.Name != "global-eth0" {
			t.Errorf("Expected global name 'global-eth0', got '%s'", req.Name)
		}
		if req.Type != "1000base-t" {
			t.Errorf("Expected global type '1000base-t', got '%s'", req.Type)
		}
	})
}

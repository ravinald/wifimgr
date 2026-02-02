package netbox

import (
	"testing"

	"github.com/ravinald/wifimgr/internal/vendors"
)

func TestDefaultMappings(t *testing.T) {
	mappings := DefaultMappings()

	// Test default roles have defaults
	if mappings.DefaultRoles["ap"] != "wireless-ap" {
		t.Errorf("Expected ap role to be 'wireless-ap', got '%s'", mappings.DefaultRoles["ap"])
	}
	if mappings.DefaultRoles["switch"] != "access-switch" {
		t.Errorf("Expected switch role to be 'access-switch', got '%s'", mappings.DefaultRoles["switch"])
	}
	if mappings.DefaultRoles["gateway"] != "router" {
		t.Errorf("Expected gateway role to be 'router', got '%s'", mappings.DefaultRoles["gateway"])
	}

	// Test maps are initialized
	if mappings.DeviceTypes == nil {
		t.Error("Expected DeviceTypes map to be initialized")
	}
	if mappings.SiteOverrides == nil {
		t.Error("Expected SiteOverrides map to be initialized")
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
	}{
		{
			name: "valid config",
			config: Config{
				URL:    "https://netbox.example.com",
				APIKey: "test-api-key",
			},
			expectError: false,
		},
		{
			name: "valid config with settings_source=api",
			config: Config{
				URL:            "https://netbox.example.com",
				APIKey:         "test-api-key",
				SettingsSource: "api",
			},
			expectError: false,
		},
		{
			name: "valid config with settings_source=netbox",
			config: Config{
				URL:            "https://netbox.example.com",
				APIKey:         "test-api-key",
				SettingsSource: "netbox",
			},
			expectError: false,
		},
		{
			name: "invalid settings_source",
			config: Config{
				URL:            "https://netbox.example.com",
				APIKey:         "test-api-key",
				SettingsSource: "invalid",
			},
			expectError: true,
		},
		{
			name: "missing URL",
			config: Config{
				APIKey: "test-api-key",
			},
			expectError: true,
		},
		{
			name: "missing API key",
			config: Config{
				URL: "https://netbox.example.com",
			},
			expectError: true,
		},
		{
			name:        "empty config",
			config:      Config{},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.Validate()
			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Did not expect error but got: %v", err)
			}
			// After validation, settings_source should default to "api" if empty
			if !tc.expectError && tc.config.SettingsSource == "" {
				if tc.config.SettingsSource != "api" {
					t.Errorf("Expected SettingsSource to default to 'api', got '%s'", tc.config.SettingsSource)
				}
			}
		})
	}
}

func TestGetDeviceRoleSlug(t *testing.T) {
	cfg := &Config{
		Mappings: MappingConfig{
			DefaultRoles: map[string]string{
				"custom-type": "custom-role",
			},
		},
	}

	tests := []struct {
		deviceType string
		expected   string
	}{
		{"ap", "wireless-ap"},
		{"switch", "access-switch"},
		{"gateway", "router"},
		{"custom-type", "custom-role"},
		{"unknown", "unknown"}, // returns the input if no mapping
	}

	for _, tc := range tests {
		t.Run(tc.deviceType, func(t *testing.T) {
			result := cfg.GetDeviceRoleSlug(tc.deviceType)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestGetDeviceRoleSlugForModel(t *testing.T) {
	cfg := &Config{
		Mappings: MappingConfig{
			DefaultRoles: map[string]string{
				"ap": "default-ap-role",
			},
			DeviceTypes: map[string]DeviceTypeMapping{
				"MR46E": {Slug: "mr46e", Role: "special-ap-role"},
				"MR55":  {Slug: "mr55"}, // No role override
				"MR*":   {Slug: "meraki-generic", Role: "meraki-role"},
			},
		},
	}

	tests := []struct {
		name         string
		deviceType   string
		model        string
		deviceNetBox any
		expected     string
	}{
		{
			name:         "per-device NetBox role override",
			deviceType:   "ap",
			model:        "MR46E",
			deviceNetBox: &vendors.NetBoxDeviceExtension{DeviceRole: "per-device-role"},
			expected:     "per-device-role",
		},
		{
			name:         "nil NetBox extension falls back to model role",
			deviceType:   "ap",
			model:        "MR46E",
			deviceNetBox: nil,
			expected:     "special-ap-role",
		},
		{
			name:         "empty NetBox role falls back to model role",
			deviceType:   "ap",
			model:        "MR46E",
			deviceNetBox: &vendors.NetBoxDeviceExtension{DeviceRole: ""},
			expected:     "special-ap-role",
		},
		{
			name:         "model with role override",
			deviceType:   "ap",
			model:        "MR46E",
			deviceNetBox: nil,
			expected:     "special-ap-role",
		},
		{
			name:         "model without role override uses default",
			deviceType:   "ap",
			model:        "MR55",
			deviceNetBox: nil,
			expected:     "default-ap-role",
		},
		{
			name:         "pattern match with role override",
			deviceType:   "ap",
			model:        "MR36",
			deviceNetBox: nil,
			expected:     "meraki-role",
		},
		{
			name:         "no model uses default",
			deviceType:   "ap",
			model:        "",
			deviceNetBox: nil,
			expected:     "default-ap-role",
		},
		{
			name:         "unknown model uses default",
			deviceType:   "ap",
			model:        "Unknown-AP",
			deviceNetBox: nil,
			expected:     "default-ap-role",
		},
		{
			name:         "hardcoded default when no config",
			deviceType:   "switch",
			model:        "MS120",
			deviceNetBox: nil,
			expected:     "access-switch",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := cfg.GetDeviceRoleSlugForModel(tc.deviceType, tc.model, tc.deviceNetBox)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestGetDeviceTypeSlug(t *testing.T) {
	cfg := &Config{
		Mappings: MappingConfig{
			DeviceTypes: map[string]DeviceTypeMapping{
				"AP43":     {Slug: "juniper-ap43"},
				"MR46":     {Slug: "meraki-mr46"},
				"Juniper*": {Slug: "juniper-generic"},
			},
		},
	}

	tests := []struct {
		model    string
		expected string
	}{
		{"AP43", "juniper-ap43"},
		{"MR46", "meraki-mr46"},
		{"JuniperAP45", "juniper-generic"}, // pattern match
		{"Unknown Model", "unknown-model"}, // default slug format
		{"Model With Spaces", "model-with-spaces"},
	}

	for _, tc := range tests {
		t.Run(tc.model, func(t *testing.T) {
			result := cfg.GetDeviceTypeSlug(tc.model)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestGetSiteSlug(t *testing.T) {
	cfg := &Config{
		Mappings: MappingConfig{
			SiteOverrides: map[string]string{
				"US-LAB-01": "us-lab-01-override",
			},
		},
	}

	tests := []struct {
		siteName string
		expected string
	}{
		{"US-LAB-01", "us-lab-01-override"},
		{"US-LAB-02", "us-lab-02"},         // default slug
		{"Site With Spaces", "site-with-spaces"},
		{"Site_With_Underscores", "site-with-underscores"},
	}

	for _, tc := range tests {
		t.Run(tc.siteName, func(t *testing.T) {
			result := cfg.GetSiteSlug(tc.siteName)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestGetAPIKey(t *testing.T) {
	// Test decrypted key takes precedence
	cfg := &Config{
		APIKey:       "encrypted-key",
		decryptedKey: "decrypted-key",
	}
	if cfg.GetAPIKey() != "decrypted-key" {
		t.Errorf("Expected decrypted key to take precedence")
	}

	// Test falls back to APIKey when no decrypted key
	cfg2 := &Config{
		APIKey: "plain-key",
	}
	if cfg2.GetAPIKey() != "plain-key" {
		t.Errorf("Expected to fall back to APIKey")
	}
}

func TestDefaultInterfaceMappings(t *testing.T) {
	mappings := DefaultInterfaceMappings()

	// Test Ethernet defaults
	if mappings.Eth0 == nil {
		t.Fatal("Expected Eth0 to be set")
	}
	if mappings.Eth0.Name != "eth0" {
		t.Errorf("Expected Eth0.Name to be 'eth0', got '%s'", mappings.Eth0.Name)
	}
	if mappings.Eth0.Type != "1000base-t" {
		t.Errorf("Expected Eth0.Type to be '1000base-t', got '%s'", mappings.Eth0.Type)
	}

	// Test radio defaults
	if mappings.Radio0 == nil || mappings.Radio0.Name != "wifi0" {
		t.Error("Expected Radio0 to be wifi0")
	}
	if mappings.Radio0.Type != "ieee802.11n" {
		t.Errorf("Expected Radio0.Type to be 'ieee802.11n', got '%s'", mappings.Radio0.Type)
	}

	if mappings.Radio1 == nil || mappings.Radio1.Name != "wifi1" {
		t.Error("Expected Radio1 to be wifi1")
	}
	if mappings.Radio1.Type != "ieee802.11ac" {
		t.Errorf("Expected Radio1.Type to be 'ieee802.11ac', got '%s'", mappings.Radio1.Type)
	}

	if mappings.Radio2 == nil || mappings.Radio2.Name != "wifi2" {
		t.Error("Expected Radio2 to be wifi2")
	}
	if mappings.Radio2.Type != "ieee802.11ax" {
		t.Errorf("Expected Radio2.Type to be 'ieee802.11ax', got '%s'", mappings.Radio2.Type)
	}
}

func TestGetInterfaceMapping(t *testing.T) {
	// Test with custom mappings
	cfg := &Config{
		Mappings: MappingConfig{
			Interfaces: InterfaceMappings{
				Eth0:   &InterfaceMapping{Name: "mgmt0", Type: "10gbase-t"},
				Radio0: &InterfaceMapping{Name: "wlan0", Type: "ieee802.11ax"},
			},
		},
	}

	tests := []struct {
		internalID   string
		expectedName string
		expectedType string
	}{
		// Custom mappings
		{"eth0", "mgmt0", "10gbase-t"},
		{"radio0", "wlan0", "ieee802.11ax"},
		// Defaults for non-overridden mappings
		{"eth1", "eth1", "1000base-t"},
		{"radio1", "wifi1", "ieee802.11ac"},
		{"radio2", "wifi2", "ieee802.11ax"},
		// Unknown returns nil
	}

	for _, tc := range tests {
		t.Run(tc.internalID, func(t *testing.T) {
			mapping := cfg.GetInterfaceMapping(tc.internalID)
			if mapping == nil {
				t.Fatal("Expected mapping to not be nil")
			}
			if mapping.Name != tc.expectedName {
				t.Errorf("Expected name '%s', got '%s'", tc.expectedName, mapping.Name)
			}
			if mapping.Type != tc.expectedType {
				t.Errorf("Expected type '%s', got '%s'", tc.expectedType, mapping.Type)
			}
		})
	}

	// Test unknown internal ID returns nil
	if mapping := cfg.GetInterfaceMapping("unknown"); mapping != nil {
		t.Error("Expected nil for unknown internal ID")
	}
}

func TestGetInterfaceMappingDefaults(t *testing.T) {
	// Test with empty config - should return defaults
	cfg := &Config{
		Mappings: MappingConfig{},
	}

	mapping := cfg.GetInterfaceMapping("eth0")
	if mapping == nil {
		t.Fatal("Expected default mapping for eth0")
	}
	if mapping.Name != "eth0" {
		t.Errorf("Expected default name 'eth0', got '%s'", mapping.Name)
	}
	if mapping.Type != "1000base-t" {
		t.Errorf("Expected default type '1000base-t', got '%s'", mapping.Type)
	}
}

func TestDefaultMappingsIncludesInterfaces(t *testing.T) {
	mappings := DefaultMappings()

	// Interface mappings should be initialized with defaults
	if mappings.Interfaces.Eth0 == nil {
		t.Error("Expected Interfaces.Eth0 to be set in DefaultMappings")
	}
	if mappings.Interfaces.Radio1 == nil {
		t.Error("Expected Interfaces.Radio1 to be set in DefaultMappings")
	}
}

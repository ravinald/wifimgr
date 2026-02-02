package netbox

import (
	"testing"

	"github.com/ravinald/wifimgr/internal/vendors"
)

// TestNetBoxExtensionIntegration verifies the complete flow of NetBox device role resolution
// with per-device extension overrides.
func TestNetBoxExtensionIntegration(t *testing.T) {
	cfg := &Config{
		Mappings: MappingConfig{
			DefaultRoles: map[string]string{
				"ap": "default-ap-role",
			},
			DeviceTypes: map[string]DeviceTypeMapping{
				"MR55": {Slug: "mr55", Role: "model-specific-role"},
			},
		},
	}

	t.Run("per-device NetBox role takes highest priority", func(t *testing.T) {
		// Create an inventory item with NetBox extension
		item := &vendors.InventoryItem{
			MAC:      "aabbccddeeff",
			Serial:   "TEST123",
			Model:    "MR55",
			Name:     "AP-01",
			Type:     "ap",
			SiteName: "US-LAB-01",
			NetBox: &vendors.NetBoxDeviceExtension{
				DeviceRole: "per-device-role",
			},
		}

		// Verify priority order: per-device > model-specific > default
		result := cfg.GetDeviceRoleSlugForModel(item.Type, item.Model, item.NetBox)
		if result != "per-device-role" {
			t.Errorf("Expected per-device role 'per-device-role', got '%s'", result)
		}
	})

	t.Run("model-specific role when no per-device override", func(t *testing.T) {
		item := &vendors.InventoryItem{
			MAC:      "aabbccddeeff",
			Serial:   "TEST123",
			Model:    "MR55",
			Name:     "AP-01",
			Type:     "ap",
			SiteName: "US-LAB-01",
			NetBox:   nil, // No per-device override
		}

		result := cfg.GetDeviceRoleSlugForModel(item.Type, item.Model, nil)
		if result != "model-specific-role" {
			t.Errorf("Expected model-specific role 'model-specific-role', got '%s'", result)
		}
	})

	t.Run("default role when no per-device or model override", func(t *testing.T) {
		item := &vendors.InventoryItem{
			MAC:      "aabbccddeeff",
			Serial:   "TEST123",
			Model:    "UnknownModel",
			Name:     "AP-01",
			Type:     "ap",
			SiteName: "US-LAB-01",
			NetBox:   nil,
		}

		result := cfg.GetDeviceRoleSlugForModel(item.Type, item.Model, nil)
		if result != "default-ap-role" {
			t.Errorf("Expected default role 'default-ap-role', got '%s'", result)
		}
	})

	t.Run("empty per-device role falls back to model role", func(t *testing.T) {
		item := &vendors.InventoryItem{
			MAC:      "aabbccddeeff",
			Serial:   "TEST123",
			Model:    "MR55",
			Name:     "AP-01",
			Type:     "ap",
			SiteName: "US-LAB-01",
			NetBox: &vendors.NetBoxDeviceExtension{
				DeviceRole: "", // Empty role should fall back
			},
		}

		result := cfg.GetDeviceRoleSlugForModel(item.Type, item.Model, item.NetBox)
		if result != "model-specific-role" {
			t.Errorf("Expected model-specific role 'model-specific-role', got '%s'", result)
		}
	})
}

// TestValidatorWithNetBoxExtension verifies the validator correctly handles NetBox extensions.
func TestValidatorWithNetBoxExtension(t *testing.T) {
	// Create mock cache with test data
	validator := &Validator{
		config: &Config{
			Mappings: MappingConfig{
				DefaultRoles: map[string]string{
					"ap": "default-ap-role",
				},
				DeviceTypes: map[string]DeviceTypeMapping{
					"MR55": {Slug: "mr55", Role: "model-role"},
				},
			},
		},
		cache: &LookupCache{
			SitesByName:       map[string]int64{"us-lab-01": 1},
			DeviceTypesBySlug: map[string]int64{"mr55": 100},
			DeviceRolesBySlug: map[string]int64{
				"per-device-role": 200,
				"model-role":      201,
				"default-ap-role": 202,
			},
		},
	}

	t.Run("validator uses per-device role when present", func(t *testing.T) {
		item := &vendors.InventoryItem{
			MAC:      "aabbccddeeff",
			Serial:   "TEST123",
			Model:    "MR55",
			Name:     "AP-01",
			Type:     "ap",
			SiteName: "US-LAB-01",
			NetBox: &vendors.NetBoxDeviceExtension{
				DeviceRole: "per-device-role",
			},
		}

		result := validator.ValidateDevice(item)

		if !result.Valid {
			t.Fatalf("Expected valid result, got errors: %v", result.Errors)
		}

		if result.DeviceRoleID != 200 {
			t.Errorf("Expected per-device role ID 200, got %d", result.DeviceRoleID)
		}
	})

	t.Run("validator falls back to model role when no per-device override", func(t *testing.T) {
		item := &vendors.InventoryItem{
			MAC:      "aabbccddeeff",
			Serial:   "TEST123",
			Model:    "MR55",
			Name:     "AP-01",
			Type:     "ap",
			SiteName: "US-LAB-01",
			NetBox:   nil,
		}

		result := validator.ValidateDevice(item)

		if !result.Valid {
			t.Fatalf("Expected valid result, got errors: %v", result.Errors)
		}

		if result.DeviceRoleID != 201 {
			t.Errorf("Expected model role ID 201, got %d", result.DeviceRoleID)
		}
	})
}

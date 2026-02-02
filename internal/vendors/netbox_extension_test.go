package vendors

import (
	"encoding/json"
	"testing"
)

func TestNetBoxDeviceExtension(t *testing.T) {
	t.Run("marshal and unmarshal", func(t *testing.T) {
		item := &InventoryItem{
			MAC:    "aabbccddeeff",
			Serial: "TEST123",
			Model:  "MR55",
			Name:   "AP-01",
			Type:   "ap",
			NetBox: &NetBoxDeviceExtension{
				DeviceRole: "special-ap-role",
			},
		}

		// Marshal to JSON
		data, err := json.Marshal(item)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		// Verify JSON contains netbox field
		var raw map[string]interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("Failed to unmarshal to map: %v", err)
		}

		netboxField, ok := raw["netbox"]
		if !ok {
			t.Error("Expected 'netbox' field in JSON")
		}

		netboxMap, ok := netboxField.(map[string]interface{})
		if !ok {
			t.Error("Expected 'netbox' to be an object")
		}

		if netboxMap["device_role"] != "special-ap-role" {
			t.Errorf("Expected device_role 'special-ap-role', got '%v'", netboxMap["device_role"])
		}

		// Unmarshal back to struct
		var item2 InventoryItem
		if err := json.Unmarshal(data, &item2); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if item2.NetBox == nil {
			t.Fatal("NetBox extension is nil after unmarshal")
		}

		if item2.NetBox.DeviceRole != "special-ap-role" {
			t.Errorf("Expected device_role 'special-ap-role', got '%s'", item2.NetBox.DeviceRole)
		}
	})

	t.Run("omit empty NetBox extension", func(t *testing.T) {
		item := &InventoryItem{
			MAC:    "aabbccddeeff",
			Serial: "TEST123",
			Model:  "MR55",
			Name:   "AP-01",
			Type:   "ap",
			NetBox: nil,
		}

		data, err := json.Marshal(item)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var raw map[string]interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("Failed to unmarshal to map: %v", err)
		}

		if _, ok := raw["netbox"]; ok {
			t.Error("Expected 'netbox' field to be omitted when nil")
		}
	})

	t.Run("omit empty device_role", func(t *testing.T) {
		item := &InventoryItem{
			MAC:    "aabbccddeeff",
			Serial: "TEST123",
			Model:  "MR55",
			Name:   "AP-01",
			Type:   "ap",
			NetBox: &NetBoxDeviceExtension{
				DeviceRole: "",
			},
		}

		data, err := json.Marshal(item)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var raw map[string]interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("Failed to unmarshal to map: %v", err)
		}

		// NetBox field should be present but empty object should be omitted by omitempty
		if netboxField, ok := raw["netbox"]; ok {
			netboxMap, _ := netboxField.(map[string]interface{})
			if _, hasRole := netboxMap["device_role"]; hasRole {
				t.Error("Expected 'device_role' to be omitted when empty")
			}
		}
	})
}

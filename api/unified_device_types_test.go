package api

import (
	"testing"
)

// Compile-time interface compliance checks - no runtime test needed
var _ DeviceMarshaler = &BaseDevice{}
var _ DeviceMarshaler = &UnifiedDevice{}
var _ DeviceMarshaler = &APDevice{}
var _ DeviceMarshaler = &MistSwitchDevice{}
var _ DeviceMarshaler = &MistGatewayDevice{}

func TestBaseDevice_FromMap_ParsesAllFieldTypes(t *testing.T) {
	testData := map[string]interface{}{
		"id":               "device-123",
		"mac":              "001122334455",
		"serial":           "SN123456789",
		"name":             "Test Device",
		"type":             "ap",
		"connected":        true,
		"created_time":     1640995200.0,
		"tags":             []interface{}{"tag1", "tag2"},
		"unknown_field":    "preserved",
	}

	var device BaseDevice
	if err := device.FromMap(testData); err != nil {
		t.Fatalf("FromMap failed: %v", err)
	}

	// String fields
	if device.ID == nil || *device.ID != "device-123" {
		t.Error("ID not parsed")
	}
	if device.MAC == nil || *device.MAC != "001122334455" {
		t.Error("MAC not parsed")
	}

	// Bool field
	if device.Connected == nil || *device.Connected != true {
		t.Error("Connected not parsed")
	}

	// Timestamp conversion (float64 → int64)
	if device.CreatedTime == nil || *device.CreatedTime != 1640995200 {
		t.Error("CreatedTime not parsed")
	}

	// Tags array
	if device.Tags == nil || len(*device.Tags) != 2 || (*device.Tags)[0] != "tag1" {
		t.Error("Tags not parsed")
	}

	// Unknown fields preserved
	if device.AdditionalConfig["unknown_field"] != "preserved" {
		t.Error("Unknown fields should be preserved in AdditionalConfig")
	}
}

func TestBaseDevice_ToMap_NilFieldsExcluded(t *testing.T) {
	device := BaseDevice{
		ID:   StringPtr("test-id"),
		Name: StringPtr("Test Device"),
		// All other fields nil
	}

	result := device.ToMap()

	if result["id"] != "test-id" {
		t.Error("expected id in map")
	}

	// Nil fields should not be present
	nilFields := []string{"mac", "serial", "model", "type", "connected"}
	for _, field := range nilFields {
		if _, exists := result[field]; exists {
			t.Errorf("nil field %s should not be in map", field)
		}
	}
}

func TestBaseDevice_ToConfigMap_ExcludesStatusFields(t *testing.T) {
	device := BaseDevice{
		ID:          StringPtr("test-id"),       // status - excluded
		Connected:   BoolPtr(true),              // status - excluded
		CreatedTime: Int64Ptr(1640995200),       // status - excluded
		Name:        StringPtr("Test Device"),   // config - included
		Magic:       StringPtr("magic-123"),     // config - included
	}

	result := device.ToConfigMap()

	// Config fields should be present
	if result["name"] != "Test Device" {
		t.Error("config field name should be present")
	}
	if result["magic"] != "magic-123" {
		t.Error("config field magic should be present")
	}

	// Status fields should be excluded
	statusFields := []string{"id", "connected", "created_time", "last_seen", "hw_rev"}
	for _, field := range statusFields {
		if _, exists := result[field]; exists {
			t.Errorf("status field %s should be excluded from config map", field)
		}
	}
}

func TestBaseDevice_FromConfigMap_IgnoresStatusFields(t *testing.T) {
	testData := map[string]interface{}{
		"name":         "Config Test Device",
		"magic":        "config-magic-123",
		"id":           "should-be-ignored",
		"connected":    true,
		"created_time": 1640995200,
	}

	var device BaseDevice
	if err := device.FromConfigMap(testData); err != nil {
		t.Fatalf("FromConfigMap failed: %v", err)
	}

	// Config fields should be set
	if device.Name == nil || *device.Name != "Config Test Device" {
		t.Error("config field Name should be set")
	}

	// Status fields should NOT be set
	if device.ID != nil {
		t.Error("status field ID should not be set from config")
	}
	if device.Connected != nil {
		t.Error("status field Connected should not be set from config")
	}
}

func TestUnifiedDevice_FromMap_DetectsDeviceType(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]interface{}
		expected string
	}{
		{"ap device", map[string]interface{}{"type": "ap"}, "ap"},
		{"switch device", map[string]interface{}{"type": "switch"}, "switch"},
		{"gateway device", map[string]interface{}{"type": "gateway"}, "gateway"},
		{"unknown type", map[string]interface{}{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var device UnifiedDevice
			if err := device.FromMap(tt.data); err != nil {
				t.Fatalf("FromMap failed: %v", err)
			}
			if device.DeviceType != tt.expected {
				t.Errorf("DeviceType = %s, want %s", device.DeviceType, tt.expected)
			}
		})
	}
}

func TestUnifiedDevice_FromMap_StoresDeviceSpecificConfig(t *testing.T) {
	testData := map[string]interface{}{
		"id":           "ap-123",
		"type":         "ap",
		"location":     []interface{}{40.7128, -74.0060},
		"radio_config": map[string]interface{}{"band_24": map[string]interface{}{"power": 10}},
	}

	var device UnifiedDevice
	if err := device.FromMap(testData); err != nil {
		t.Fatalf("FromMap failed: %v", err)
	}

	if _, exists := device.DeviceConfig["location"]; !exists {
		t.Error("location should be in DeviceConfig")
	}
	if _, exists := device.DeviceConfig["radio_config"]; !exists {
		t.Error("radio_config should be in DeviceConfig")
	}
}

func TestAPDevice_FromMap_ParsesAPSpecificFields(t *testing.T) {
	testData := map[string]interface{}{
		"id":           "ap-456",
		"type":         "ap",
		"location":     []interface{}{37.7749, -122.4194},
		"orientation":  90.0,
		"height":       3.0,
		"for_site":     true,
		"disable_eth2": true,
		"ble_config":   map[string]interface{}{"beacon_enabled": true},
	}

	var device APDevice
	if err := device.FromMap(testData); err != nil {
		t.Fatalf("FromMap failed: %v", err)
	}

	if device.Location == nil || len(*device.Location) != 2 {
		t.Error("Location not parsed")
	}
	if device.Orientation == nil || *device.Orientation != 90 {
		t.Error("Orientation not parsed")
	}
	if device.Height == nil || *device.Height != 3.0 {
		t.Error("Height not parsed")
	}
	if device.ForSite == nil || *device.ForSite != true {
		t.Error("ForSite not parsed")
	}
	if device.DisableEth2 == nil || *device.DisableEth2 != true {
		t.Error("DisableEth2 not parsed")
	}
	if device.BleConfig == nil {
		t.Error("BleConfig not parsed")
	}
}

func TestAPDevice_ToMap_IncludesAPSpecificFields(t *testing.T) {
	location := []float64{37.7749, -122.4194}
	device := APDevice{}
	device.ID = StringPtr("ap-456")
	device.Type = StringPtr("ap")
	device.Location = &location
	device.Orientation = IntPtr(90)
	device.Height = Float64Ptr(3.0)

	result := device.ToMap()

	if result["id"] != "ap-456" {
		t.Error("id not in map")
	}
	if _, exists := result["location"]; !exists {
		t.Error("location should be in map")
	}
	if result["orientation"] != 90 {
		t.Error("orientation not in map")
	}
	if result["height"] != 3.0 {
		t.Error("height not in map")
	}
}

func TestDevice_EdgeCases(t *testing.T) {
	// Test with empty data
	var device BaseDevice
	if err := device.FromMap(map[string]interface{}{}); err != nil {
		t.Errorf("FromMap should handle empty data: %v", err)
	}

	// Test with nil values
	if err := device.FromMap(map[string]interface{}{"id": nil, "name": nil}); err != nil {
		t.Errorf("FromMap should handle nil values: %v", err)
	}

	// Test with wrong types - should handle gracefully
	if err := device.FromMap(map[string]interface{}{"id": 123, "created_time": "not a number"}); err != nil {
		t.Errorf("FromMap should handle type mismatches gracefully: %v", err)
	}
}

func TestNewUnifiedDeviceFromMap(t *testing.T) {
	testData := map[string]interface{}{
		"id":   "device-789",
		"type": "switch",
		"name": "Test Switch",
	}

	device, err := NewUnifiedDeviceFromMap(testData)
	if err != nil {
		t.Fatalf("NewUnifiedDeviceFromMap failed: %v", err)
	}

	if device.DeviceType != "switch" {
		t.Errorf("DeviceType = %s, want switch", device.DeviceType)
	}
	if device.GetID() == nil || *device.GetID() != "device-789" {
		t.Error("ID not set correctly")
	}
}

func TestGetDeviceTypeFromMap(t *testing.T) {
	tests := []struct {
		data     map[string]interface{}
		expected string
	}{
		{map[string]interface{}{"type": "ap"}, "ap"},
		{map[string]interface{}{"type": "switch"}, "switch"},
		{map[string]interface{}{"type": "gateway"}, "gateway"},
		{map[string]interface{}{}, "unknown"},
		{map[string]interface{}{"type": 123}, "unknown"},
	}

	for _, test := range tests {
		result := GetDeviceTypeFromMap(test.data)
		if result != test.expected {
			t.Errorf("GetDeviceTypeFromMap(%v) = %s, want %s", test.data, result, test.expected)
		}
	}
}

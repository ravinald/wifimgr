package api

import (
	"testing"
)

// Compile-time interface compliance check - no runtime test needed
var _ InventoryMarshaler = (*MistInventoryItem)(nil)

func TestMistInventoryItem_GetMAC_NormalizesFormat(t *testing.T) {
	// This tests actual behavior - MAC normalization
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"colons", "00:11:22:33:44:55", "001122334455"},
		{"already normalized", "001122334455", "001122334455"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &MistInventoryItem{}
			if tt.input != "" {
				item.MAC = StringPtr(tt.input)
			}
			if got := item.GetMAC(); got != tt.expected {
				t.Errorf("GetMAC() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestMistInventoryItem_ToMap_NilFieldsExcluded(t *testing.T) {
	// Test that nil fields don't appear in output
	item := &MistInventoryItem{
		ID:  StringPtr("test-id"),
		MAC: StringPtr("001122334455"),
		// All other fields nil
	}

	result := item.ToMap()

	// Non-nil fields should be present
	if result["id"] != "test-id" {
		t.Error("expected id in map")
	}
	if result["mac"] != "001122334455" {
		t.Error("expected mac in map")
	}

	// Nil fields should not be present
	nilFields := []string{"serial", "name", "model", "sku", "hw_rev", "type", "magic", "hostname"}
	for _, field := range nilFields {
		if _, exists := result[field]; exists {
			t.Errorf("nil field %s should not be in map", field)
		}
	}
}

func TestMistInventoryItem_ToMap_AdditionalConfigMerged(t *testing.T) {
	item := &MistInventoryItem{
		ID: StringPtr("test-id"),
		AdditionalConfig: map[string]interface{}{
			"custom_field": "custom_value",
		},
		Raw: map[string]interface{}{
			"raw_field": "raw_value",
		},
	}

	result := item.ToMap()

	if result["custom_field"] != "custom_value" {
		t.Error("AdditionalConfig fields should be merged")
	}
	if result["raw_field"] != "raw_value" {
		t.Error("Raw fields should be merged")
	}
}

func TestMistInventoryItem_FromMap_ParsesTypes(t *testing.T) {
	data := map[string]interface{}{
		"id":            "test-id",
		"mac":           "001122334455",
		"created_time":  float64(1640995200), // JSON numbers come as float64
		"modified_time": float64(1640995300),
		"connected":     true,
		"jsi":           false,
	}

	item := &MistInventoryItem{
		AdditionalConfig: make(map[string]interface{}),
	}

	if err := item.FromMap(data); err != nil {
		t.Fatalf("FromMap() error = %v", err)
	}

	// Check type conversions work
	if item.ID == nil || *item.ID != "test-id" {
		t.Error("string field not parsed")
	}
	if item.CreatedTime == nil || *item.CreatedTime != int64(1640995200) {
		t.Error("float64 to int64 conversion failed")
	}
	if item.Connected == nil || *item.Connected != true {
		t.Error("bool field not parsed")
	}
}

func TestMistInventoryItem_FromMap_PreservesUnknownFields(t *testing.T) {
	data := map[string]interface{}{
		"id":            "test-id",
		"unknown_field": "unknown_value",
		"nested_field": map[string]interface{}{
			"sub_field": "sub_value",
		},
	}

	item := &MistInventoryItem{
		AdditionalConfig: make(map[string]interface{}),
	}

	if err := item.FromMap(data); err != nil {
		t.Fatalf("FromMap() error = %v", err)
	}

	if item.AdditionalConfig["unknown_field"] != "unknown_value" {
		t.Error("unknown fields should be in AdditionalConfig")
	}
	if item.Raw["unknown_field"] != "unknown_value" {
		t.Error("unknown fields should be in Raw")
	}
}

func TestMistInventoryItem_ToConfigMap_ExcludesStatusFields(t *testing.T) {
	item := &MistInventoryItem{
		ID:           StringPtr("test-id"),         // status - should be excluded
		OrgID:        StringPtr("org-123"),         // status - should be excluded
		CreatedTime:  Int64Ptr(1640995200),         // status - should be excluded
		ModifiedTime: Int64Ptr(1640995300),         // status - should be excluded
		Connected:    BoolPtr(true),                // status - should be excluded
		Adopted:      BoolPtr(true),                // status - should be excluded
		MAC:          StringPtr("001122334455"),    // config - should be included
		Name:         StringPtr("Test Device"),     // config - should be included
		SiteID:       StringPtr("site-456"),        // config - should be included
	}

	result := item.ToConfigMap()

	// Config fields should be present
	if result["mac"] != "001122334455" {
		t.Error("config field mac should be present")
	}
	if result["name"] != "Test Device" {
		t.Error("config field name should be present")
	}

	// Status fields should be excluded
	statusFields := []string{"id", "org_id", "created_time", "modified_time", "connected", "adopted"}
	for _, field := range statusFields {
		if _, exists := result[field]; exists {
			t.Errorf("status field %s should be excluded from config map", field)
		}
	}
}

func TestMistInventoryItem_FromConfigMap_IgnoresStatusFields(t *testing.T) {
	data := map[string]interface{}{
		"mac":           "001122334455",
		"name":          "Test Device",
		"id":            "should-be-ignored",
		"org_id":        "should-be-ignored",
		"created_time":  1640995200,
		"modified_time": 1640995300,
		"connected":     true,
		"adopted":       true,
	}

	item := &MistInventoryItem{
		AdditionalConfig: make(map[string]interface{}),
	}

	if err := item.FromConfigMap(data); err != nil {
		t.Fatalf("FromConfigMap() error = %v", err)
	}

	// Config fields should be set
	if item.MAC == nil || *item.MAC != "001122334455" {
		t.Error("config field MAC should be set")
	}

	// Status fields should NOT be set
	if item.ID != nil {
		t.Error("status field ID should not be set from config")
	}
	if item.OrgID != nil {
		t.Error("status field OrgID should not be set from config")
	}
	if item.CreatedTime != nil {
		t.Error("status field CreatedTime should not be set from config")
	}
}

func TestMistInventoryItem_RoundTrip(t *testing.T) {
	// Test that ToMap → FromMap preserves data
	original := map[string]interface{}{
		"id":            "test-id",
		"mac":           "001122334455",
		"name":          "Test Device",
		"custom_field":  "custom_value",
		"created_time":  float64(1640995200),
	}

	item, err := NewInventoryItemFromMap(original)
	if err != nil {
		t.Fatalf("NewInventoryItemFromMap() error = %v", err)
	}

	result := item.ToMap()

	// Check key fields preserved
	if result["id"] != "test-id" {
		t.Error("id not preserved")
	}
	if result["mac"] != "001122334455" {
		t.Error("mac not preserved")
	}
	if result["custom_field"] != "custom_value" {
		t.Error("custom_field not preserved")
	}
}

func TestNewInventoryItemFromMap(t *testing.T) {
	data := map[string]interface{}{
		"id":   "test-id",
		"mac":  "001122334455",
		"type": "ap",
	}

	item, err := NewInventoryItemFromMap(data)
	if err != nil {
		t.Fatalf("NewInventoryItemFromMap() error = %v", err)
	}

	if item.GetID() != "test-id" {
		t.Error("GetID() failed")
	}
	if item.GetMAC() != "001122334455" {
		t.Error("GetMAC() failed")
	}
	if item.GetType() != "ap" {
		t.Error("GetType() failed")
	}
}

func TestNewInventoryItemFromConfigMap(t *testing.T) {
	data := map[string]interface{}{
		"mac":    "001122334455",
		"type":   "ap",
		"name":   "Test Device",
		"id":     "should-be-ignored",
		"org_id": "should-be-ignored",
	}

	item, err := NewInventoryItemFromConfigMap(data)
	if err != nil {
		t.Fatalf("NewInventoryItemFromConfigMap() error = %v", err)
	}

	if item.GetMAC() != "001122334455" {
		t.Error("config field MAC not set")
	}
	if item.ID != nil {
		t.Error("status field ID should not be set")
	}
}

func TestConvertInventoryItemToNew(t *testing.T) {
	oldItem := InventoryItem{
		Id:     UUIDPtr("test-id"),
		Mac:    StringPtr("001122334455"),
		Type:   StringPtr("ap"),
		Name:   StringPtr("Test Device"),
		SiteId: UUIDPtr("site-456"),
	}

	newItem := ConvertInventoryItemToNew(oldItem)

	if newItem.GetID() != "test-id" {
		t.Error("ID not converted")
	}
	if newItem.GetMAC() != "001122334455" {
		t.Error("MAC not converted")
	}
	if newItem.GetType() != "ap" {
		t.Error("Type not converted")
	}
}

func TestConvertInventoryItemFromNew(t *testing.T) {
	newItem := &MistInventoryItem{
		ID:     StringPtr("test-id"),
		MAC:    StringPtr("001122334455"),
		Type:   StringPtr("ap"),
		Name:   StringPtr("Test Device"),
		SiteID: StringPtr("site-456"),
	}

	oldItem := ConvertInventoryItemFromNew(newItem)

	if oldItem.Id == nil || *oldItem.Id != "test-id" {
		t.Error("Id not converted")
	}
	if oldItem.Mac == nil || *oldItem.Mac != "001122334455" {
		t.Error("Mac not converted")
	}
}

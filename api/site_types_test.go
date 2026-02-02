package api

import (
	"testing"
)

// Compile-time interface compliance check - no runtime test needed
var _ SiteMarshaler = (*MistSite)(nil)

func TestMistSite_ToMap_NilFieldsExcluded(t *testing.T) {
	site := &MistSite{
		ID:   StringPtr("test-id"),
		Name: StringPtr("Test Site"),
		// All other fields nil
	}

	result := site.ToMap()

	// Non-nil fields should be present
	if result["id"] != "test-id" {
		t.Error("expected id in map")
	}
	if result["name"] != "Test Site" {
		t.Error("expected name in map")
	}

	// Nil fields should not be present
	nilFields := []string{"address", "country_code", "timezone", "notes", "latlng"}
	for _, field := range nilFields {
		if _, exists := result[field]; exists {
			t.Errorf("nil field %s should not be in map", field)
		}
	}
}

func TestMistSite_ToMap_IncludesLatlng(t *testing.T) {
	site := &MistSite{
		ID:   StringPtr("test-id"),
		Name: StringPtr("Test Site"),
		Latlng: &MistLatLng{
			Lat: Float64Ptr(37.7749),
			Lng: Float64Ptr(-122.4194),
		},
	}

	result := site.ToMap()

	latlng, ok := result["latlng"].(map[string]interface{})
	if !ok {
		t.Fatal("expected latlng to be a map")
	}
	if latlng["lat"] != 37.7749 {
		t.Errorf("expected lat = 37.7749, got %v", latlng["lat"])
	}
	if latlng["lng"] != -122.4194 {
		t.Errorf("expected lng = -122.4194, got %v", latlng["lng"])
	}
}

func TestMistSite_ToMap_AdditionalConfigMerged(t *testing.T) {
	site := &MistSite{
		ID: StringPtr("test-id"),
		AdditionalConfig: map[string]interface{}{
			"custom_field": "custom_value",
		},
	}

	result := site.ToMap()

	if result["custom_field"] != "custom_value" {
		t.Error("AdditionalConfig fields should be merged")
	}
}

func TestMistSite_FromMap_ParsesTypes(t *testing.T) {
	data := map[string]interface{}{
		"id":            "test-id",
		"name":          "Test Site",
		"created_time":  1234567890.0,
		"modified_time": 1234567891.0,
		"latlng": map[string]interface{}{
			"lat": 37.7749,
			"lng": -122.4194,
		},
	}

	site := &MistSite{
		AdditionalConfig: make(map[string]interface{}),
	}

	if err := site.FromMap(data); err != nil {
		t.Fatalf("FromMap() error = %v", err)
	}

	if site.ID == nil || *site.ID != "test-id" {
		t.Error("string field not parsed")
	}
	if site.CreatedTime == nil || *site.CreatedTime != 1234567890.0 {
		t.Error("float64 field not parsed")
	}
	if site.Latlng == nil || site.Latlng.Lat == nil || *site.Latlng.Lat != 37.7749 {
		t.Error("latlng not parsed")
	}
}

func TestMistSite_FromMap_PreservesUnknownFields(t *testing.T) {
	data := map[string]interface{}{
		"id":            "test-id",
		"unknown_field": "unknown_value",
	}

	site := &MistSite{
		AdditionalConfig: make(map[string]interface{}),
	}

	if err := site.FromMap(data); err != nil {
		t.Fatalf("FromMap() error = %v", err)
	}

	if site.AdditionalConfig["unknown_field"] != "unknown_value" {
		t.Error("unknown fields should be in AdditionalConfig")
	}
}

func TestMistSite_ToConfigMap_ExcludesStatusFields(t *testing.T) {
	site := &MistSite{
		ID:           StringPtr("test-id"),         // status - should be excluded
		OrgID:        StringPtr("org-123"),         // status - should be excluded
		CreatedTime:  Float64Ptr(1234567890.0),     // status - should be excluded
		ModifiedTime: Float64Ptr(1234567891.0),     // status - should be excluded
		Name:         StringPtr("Test Site"),       // config - should be included
		Address:      StringPtr("123 Test St"),     // config - should be included
	}

	result := site.ToConfigMap()

	// Config fields should be present
	if result["name"] != "Test Site" {
		t.Error("config field name should be present")
	}
	if result["address"] != "123 Test St" {
		t.Error("config field address should be present")
	}

	// Status fields should be excluded
	statusFields := []string{"id", "org_id", "created_time", "modified_time"}
	for _, field := range statusFields {
		if _, exists := result[field]; exists {
			t.Errorf("status field %s should be excluded from config map", field)
		}
	}
}

func TestMistSite_FromConfigMap_IgnoresStatusFields(t *testing.T) {
	data := map[string]interface{}{
		"name":          "Test Site",
		"address":       "123 Test St",
		"id":            "should-be-ignored",
		"org_id":        "should-be-ignored",
		"created_time":  1234567890.0,
		"modified_time": 1234567891.0,
	}

	site := &MistSite{
		AdditionalConfig: make(map[string]interface{}),
	}

	if err := site.FromConfigMap(data); err != nil {
		t.Fatalf("FromConfigMap() error = %v", err)
	}

	// Config fields should be set
	if site.Name == nil || *site.Name != "Test Site" {
		t.Error("config field Name should be set")
	}

	// Status fields should NOT be set
	if site.ID != nil {
		t.Error("status field ID should not be set from config")
	}
	if site.OrgID != nil {
		t.Error("status field OrgID should not be set from config")
	}
}

func TestMistSite_RoundTrip(t *testing.T) {
	original := map[string]interface{}{
		"id":           "test-id",
		"name":         "Test Site",
		"address":      "123 Test St",
		"custom_field": "custom_value",
		"latlng": map[string]interface{}{
			"lat": 37.7749,
			"lng": -122.4194,
		},
	}

	site, err := NewSiteFromMap(original)
	if err != nil {
		t.Fatalf("NewSiteFromMap() error = %v", err)
	}

	result := site.ToMap()

	if result["id"] != "test-id" {
		t.Error("id not preserved")
	}
	if result["name"] != "Test Site" {
		t.Error("name not preserved")
	}
	if result["custom_field"] != "custom_value" {
		t.Error("custom_field not preserved")
	}
}

func TestNewSiteFromMap(t *testing.T) {
	data := map[string]interface{}{
		"id":   "test-id",
		"name": "Test Site",
	}

	site, err := NewSiteFromMap(data)
	if err != nil {
		t.Fatalf("NewSiteFromMap() error = %v", err)
	}

	if site.GetID() != "test-id" {
		t.Error("GetID() failed")
	}
	if site.GetName() != "Test Site" {
		t.Error("GetName() failed")
	}
}

func TestNewSiteFromConfigMap(t *testing.T) {
	data := map[string]interface{}{
		"name":         "Test Site",
		"address":      "123 Test St",
		"id":           "should-be-ignored",
		"created_time": 1234567890.0,
	}

	site, err := NewSiteFromConfigMap(data)
	if err != nil {
		t.Fatalf("NewSiteFromConfigMap() error = %v", err)
	}

	if site.GetName() != "Test Site" {
		t.Error("config field Name not set")
	}
	if site.ID != nil {
		t.Error("status field ID should not be set")
	}
}

func TestConvertSiteToNew(t *testing.T) {
	siteID := UUID("test-id")
	oldSite := Site{
		Id:      &siteID,
		Name:    StringPtr("Test Site"),
		Address: StringPtr("123 Test St"),
		Latlng: &LatLng{
			Lat: 37.7749,
			Lng: -122.4194,
		},
	}

	newSite := ConvertSiteToNew(oldSite)

	if newSite.GetID() != "test-id" {
		t.Error("ID not converted")
	}
	if newSite.GetName() != "Test Site" {
		t.Error("Name not converted")
	}
	if newSite.Latlng == nil || *newSite.Latlng.Lat != 37.7749 {
		t.Error("Latlng not converted")
	}
}

func TestConvertSiteFromNew(t *testing.T) {
	newSite := &MistSite{
		ID:      StringPtr("test-id"),
		Name:    StringPtr("Test Site"),
		Address: StringPtr("123 Test St"),
		Latlng: &MistLatLng{
			Lat: Float64Ptr(37.7749),
			Lng: Float64Ptr(-122.4194),
		},
	}

	oldSite := ConvertSiteFromNew(newSite)

	if oldSite.Id == nil || string(*oldSite.Id) != "test-id" {
		t.Error("Id not converted")
	}
	if oldSite.Name == nil || *oldSite.Name != "Test Site" {
		t.Error("Name not converted")
	}
	if oldSite.Latlng == nil || oldSite.Latlng.Lat != 37.7749 {
		t.Error("Latlng not converted")
	}
}

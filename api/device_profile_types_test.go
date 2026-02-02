package api

import (
	"testing"
)

func TestDeviceProfile_FromMap_ParsesFields(t *testing.T) {
	testData := map[string]interface{}{
		"id":               "test-profile-id",
		"name":             "Test Profile",
		"type":             "ap",
		"org_id":           "test-org-id",
		"created_time":     1640995200.0,
		"for_site":         true,
		"additional_field": "some_value",
	}

	var profile DeviceProfile
	if err := profile.FromMap(testData); err != nil {
		t.Fatalf("FromMap failed: %v", err)
	}

	if profile.ID == nil || *profile.ID != "test-profile-id" {
		t.Error("ID not parsed")
	}
	if profile.Name == nil || *profile.Name != "Test Profile" {
		t.Error("Name not parsed")
	}
	if profile.Type == nil || *profile.Type != "ap" {
		t.Error("Type not parsed")
	}
	if profile.ForSite == nil || *profile.ForSite != true {
		t.Error("ForSite not parsed")
	}
	if profile.AdditionalConfig["additional_field"] != "some_value" {
		t.Error("additional fields should be preserved")
	}
}

func TestDeviceProfile_ToMap_IncludesAllFields(t *testing.T) {
	profile := DeviceProfile{
		ID:      StringPtr("test-profile-id"),
		Name:    StringPtr("Test Profile"),
		Type:    StringPtr("ap"),
		ForSite: BoolPtr(true),
		AdditionalConfig: map[string]interface{}{
			"additional_field": "some_value",
		},
	}

	result := profile.ToMap()

	if result["id"] != "test-profile-id" {
		t.Error("id not in map")
	}
	if result["name"] != "Test Profile" {
		t.Error("name not in map")
	}
	if result["additional_field"] != "some_value" {
		t.Error("additional_field not preserved in map")
	}
}

func TestDeviceProfile_RoundTrip(t *testing.T) {
	original := map[string]interface{}{
		"id":               "test-profile-id",
		"name":             "Test Profile",
		"type":             "ap",
		"additional_field": "some_value",
	}

	var profile DeviceProfile
	if err := profile.FromMap(original); err != nil {
		t.Fatalf("FromMap failed: %v", err)
	}

	result := profile.ToMap()

	if result["id"] != "test-profile-id" {
		t.Error("id not preserved")
	}
	if result["additional_field"] != "some_value" {
		t.Error("additional_field not preserved")
	}
}

func TestNewDeviceProfileFromType(t *testing.T) {
	tests := []struct {
		profileType string
		expected    string
	}{
		{"ap", "*api.DeviceProfileAP"},
		{"gateway", "*api.DeviceProfileGateway"},
		{"switch", "*api.DeviceProfileSwitch"},
		{"unknown", "*api.DeviceProfile"},
	}

	for _, test := range tests {
		profile := NewDeviceProfileFromType(test.profileType)
		actualType := getTypeName(profile)
		if actualType != test.expected {
			t.Errorf("NewDeviceProfileFromType(%s) = %s, want %s",
				test.profileType, actualType, test.expected)
		}
	}
}

// Helper function for type checking
func getTypeName(v interface{}) string {
	switch v.(type) {
	case *DeviceProfileAP:
		return "*api.DeviceProfileAP"
	case *DeviceProfileGateway:
		return "*api.DeviceProfileGateway"
	case *DeviceProfileSwitch:
		return "*api.DeviceProfileSwitch"
	case *DeviceProfile:
		return "*api.DeviceProfile"
	default:
		return "unknown"
	}
}

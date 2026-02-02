package vendors

import (
	"strings"
	"testing"
)

func TestSiteNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      *SiteNotFoundError
		contains []string
	}{
		{
			name: "basic error",
			err: &SiteNotFoundError{
				SiteName: "US-LAB-01",
			},
			contains: []string{"US-LAB-01", "not found"},
		},
		{
			name: "with API label",
			err: &SiteNotFoundError{
				SiteName: "US-LAB-01",
				APILabel: "mist-prod",
			},
			contains: []string{"US-LAB-01", "not found", "mist-prod"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()
			for _, s := range tt.contains {
				if !strings.Contains(errStr, s) {
					t.Errorf("Error() should contain %q, got: %s", s, errStr)
				}
			}
		})
	}
}

func TestSiteNotFoundError_UserMessage(t *testing.T) {
	err := &SiteNotFoundError{
		SiteName:     "US-LAB-01",
		APILabel:     "mist-prod",
		SearchedAPIs: []string{"mist-prod", "meraki-prod"},
	}

	msg := err.UserMessage()

	// Should contain remediation advice
	if !strings.Contains(msg, "refresh") || !strings.Contains(msg, "cache") {
		t.Error("UserMessage should contain cache refresh advice")
	}
	if !strings.Contains(msg, "--api") {
		t.Error("UserMessage should mention --api flag")
	}
	if !strings.Contains(msg, "Searched APIs") {
		t.Error("UserMessage should list searched APIs")
	}
}

func TestDuplicateSiteError(t *testing.T) {
	tests := []struct {
		name     string
		err      *DuplicateSiteError
		contains []string
	}{
		{
			name: "across APIs",
			err: &DuplicateSiteError{
				SiteName: "SHARED-SITE",
				APIs:     []string{"mist-prod", "meraki-prod"},
			},
			contains: []string{"SHARED-SITE", "multiple APIs", "mist-prod", "meraki-prod"},
		},
		{
			name: "within single API",
			err: &DuplicateSiteError{
				SiteName:   "DUP-SITE",
				APILabel:   "mist-prod",
				MatchCount: 3,
			},
			contains: []string{"DUP-SITE", "3 matches", "mist-prod"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()
			for _, s := range tt.contains {
				if !strings.Contains(errStr, s) {
					t.Errorf("Error() should contain %q, got: %s", s, errStr)
				}
			}
		})
	}
}

func TestDuplicateSiteError_UserMessage(t *testing.T) {
	// Test cross-API duplicate
	err := &DuplicateSiteError{
		SiteName: "SHARED-SITE",
		APIs:     []string{"mist-prod", "meraki-prod"},
	}

	msg := err.UserMessage()
	if !strings.Contains(msg, "--api") {
		t.Error("UserMessage should mention --api flag")
	}
	if !strings.Contains(msg, "api") && !strings.Contains(msg, "field") {
		t.Error("UserMessage should mention adding api field to config")
	}

	// Test within-API duplicate
	err2 := &DuplicateSiteError{
		SiteName:   "DUP-SITE",
		APILabel:   "mist-prod",
		MatchCount: 3,
	}

	msg2 := err2.UserMessage()
	if !strings.Contains(msg2, "rename") {
		t.Error("UserMessage for in-API duplicates should suggest renaming")
	}
}

func TestAPINotFoundError(t *testing.T) {
	err := &APINotFoundError{
		APILabel: "nonexistent",
	}

	if !strings.Contains(err.Error(), "nonexistent") {
		t.Error("Error() should contain the API label")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Error("Error() should indicate not found")
	}
}

func TestAPINotFoundError_UserMessage(t *testing.T) {
	// With available APIs
	err := &APINotFoundError{
		APILabel:      "typo-api",
		AvailableAPIs: []string{"mist-prod", "meraki-prod"},
	}

	msg := err.UserMessage()
	if !strings.Contains(msg, "Available APIs") {
		t.Error("UserMessage should list available APIs")
	}
	if !strings.Contains(msg, "mist-prod") {
		t.Error("UserMessage should include available API names")
	}

	// Without available APIs
	err2 := &APINotFoundError{
		APILabel:      "test",
		AvailableAPIs: []string{},
	}

	msg2 := err2.UserMessage()
	if !strings.Contains(msg2, "no APIs configured") {
		t.Error("UserMessage should indicate no APIs configured when empty")
	}
}

func TestCapabilityNotSupportedError(t *testing.T) {
	err := &CapabilityNotSupportedError{
		Capability: "search",
		APILabel:   "meraki-prod",
		VendorName: "meraki",
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "search") {
		t.Error("Error() should contain capability name")
	}
	if !strings.Contains(errStr, "meraki-prod") {
		t.Error("Error() should contain API label")
	}
	if !strings.Contains(errStr, "meraki") {
		t.Error("Error() should contain vendor name")
	}
}

func TestCapabilityNotSupportedError_UserMessage(t *testing.T) {
	err := &CapabilityNotSupportedError{
		Capability:  "search",
		APILabel:    "meraki-prod",
		VendorName:  "meraki",
		SupportedBy: []string{"mist"},
	}

	msg := err.UserMessage()
	if !strings.Contains(msg, "only available for") {
		t.Error("UserMessage should indicate which vendors support the capability")
	}
	if !strings.Contains(msg, "mist") {
		t.Error("UserMessage should list supported vendors")
	}
}

func TestMACCollisionError(t *testing.T) {
	err := &MACCollisionError{
		MAC:  "aabbccddeef0",
		APIs: []string{"mist-prod", "meraki-prod"},
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "aabbccddeef0") {
		t.Error("Error() should contain MAC address")
	}
	if !strings.Contains(errStr, "multiple APIs") {
		t.Error("Error() should indicate multiple APIs")
	}
}

func TestDeviceNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      *DeviceNotFoundError
		contains []string
	}{
		{
			name: "basic error",
			err: &DeviceNotFoundError{
				Identifier: "aabbccddeef0",
			},
			contains: []string{"aabbccddeef0", "not found"},
		},
		{
			name: "with API label",
			err: &DeviceNotFoundError{
				Identifier: "aabbccddeef0",
				APILabel:   "mist-prod",
			},
			contains: []string{"aabbccddeef0", "not found", "mist-prod"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()
			for _, s := range tt.contains {
				if !strings.Contains(errStr, s) {
					t.Errorf("Error() should contain %q, got: %s", s, errStr)
				}
			}
		})
	}
}

func TestInvalidAPIConfigError(t *testing.T) {
	err := &InvalidAPIConfigError{
		APILabel: "bad-config",
		Reason:   "missing credentials",
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "bad-config") {
		t.Error("Error() should contain API label")
	}
	if !strings.Contains(errStr, "missing credentials") {
		t.Error("Error() should contain reason")
	}
}

func TestFieldMappingError_UserMessage(t *testing.T) {
	err := &FieldMappingError{
		Vendor:       "mist",
		DeviceMAC:    "5c:5b:35:8e:4c:f9",
		Field:        "radio_config.band_5.power",
		ExpectedType: "integer",
		ActualType:   "string",
		ActualValue:  "high",
	}

	msg := err.UserMessage()

	// Check key elements are present
	expectedParts := []string{
		"Field Type Mismatch",
		"5c:5b:35:8e:4c:f9",
		"radio_config.band_5.power",
		"Expected: integer",
		"Received: high (string)",
		"Suggested Actions",
		"wifimgr refresh cache",
	}

	for _, part := range expectedParts {
		if !strings.Contains(msg, part) {
			t.Errorf("UserMessage() missing expected part %q\nGot:\n%s", part, msg)
		}
	}
}

func TestUnexpectedFieldWarning_UserMessage(t *testing.T) {
	warn := &UnexpectedFieldWarning{
		Vendor:    "mist",
		DeviceMAC: "5c:5b:35:8e:4c:f9",
		Field:     "new_api_field",
		Value:     "test_value",
	}

	msg := warn.UserMessage()

	expectedParts := []string{
		"Unexpected Field",
		"mist API",
		"5c:5b:35:8e:4c:f9",
		"Field: new_api_field",
		"Value: test_value",
		"warning, not an error",
		"Suggested Actions",
	}

	for _, part := range expectedParts {
		if !strings.Contains(msg, part) {
			t.Errorf("UserMessage() missing expected part %q\nGot:\n%s", part, msg)
		}
	}
}

func TestMissingFieldWarning_UserMessage(t *testing.T) {
	warn := &MissingFieldWarning{
		Vendor:    "mist",
		DeviceMAC: "5c:5b:35:8e:4c:f9",
		Field:     "deprecated_field",
	}

	msg := warn.UserMessage()

	expectedParts := []string{
		"Missing Expected Field",
		"mist API",
		"5c:5b:35:8e:4c:f9",
		"Field: deprecated_field",
		"warning, not an error",
		"Suggested Actions",
		"wifimgr refresh cache",
	}

	for _, part := range expectedParts {
		if !strings.Contains(msg, part) {
			t.Errorf("UserMessage() missing expected part %q\nGot:\n%s", part, msg)
		}
	}
}

func TestExampleForType(t *testing.T) {
	tests := []struct {
		typeName string
		want     string
	}{
		{"integer", "15"},
		{"int", "15"},
		{"float", "3.14"},
		{"float64", "3.14"},
		{"string", `"text"`},
		{"boolean", "true"},
		{"bool", "true"},
		{"array", "[1, 2, 3]"},
		{"slice", "[1, 2, 3]"},
		{"object", `{"key": "value"}`},
		{"map", `{"key": "value"}`},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			if got := exampleForType(tt.typeName); got != tt.want {
				t.Errorf("exampleForType(%q) = %q, want %q", tt.typeName, got, tt.want)
			}
		})
	}
}

func TestFieldMappingError_EmptyVendor(t *testing.T) {
	err := &FieldMappingError{
		Vendor:       "",
		DeviceMAC:    "5c:5b:35:8e:4c:f9",
		Field:        "test_field",
		ExpectedType: "string",
		ActualType:   "int",
		ActualValue:  42,
	}

	msg := err.UserMessage()

	// Should default to "API" when vendor is empty
	if !strings.Contains(msg, "API has changed") {
		t.Errorf("UserMessage() with empty vendor should mention 'API', got:\n%s", msg)
	}
}

func TestConfigValidationError_Error(t *testing.T) {
	err := &ConfigValidationError{
		Field:   "radio_config.band_24_usage",
		Message: "field 'band_24_usage' is Mist-specific and not supported by Meraki",
	}

	got := err.Error()
	want := "radio_config.band_24_usage: field 'band_24_usage' is Mist-specific and not supported by Meraki"

	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

// Test that errors implement the error interface
func TestErrorsImplementErrorInterface(t *testing.T) {
	var _ error = &SiteNotFoundError{}
	var _ error = &DuplicateSiteError{}
	var _ error = &APINotFoundError{}
	var _ error = &CapabilityNotSupportedError{}
	var _ error = &MACCollisionError{}
	var _ error = &DeviceNotFoundError{}
	var _ error = &InvalidAPIConfigError{}
	var _ error = &FieldMappingError{}
	var _ error = &UnexpectedFieldWarning{}
	var _ error = &MissingFieldWarning{}
	var _ error = &ConfigValidationError{}
}

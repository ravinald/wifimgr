package netbox

import (
	"strings"
	"testing"
)

func TestValidateInterfaceType(t *testing.T) {
	tests := []struct {
		name      string
		ifaceType string
		wantErr   bool
	}{
		// Valid Ethernet types
		{"1000base-t valid", "1000base-t", false},
		{"10gbase-t valid", "10gbase-t", false},
		{"100base-tx valid", "100base-tx", false},

		// Valid wireless types
		{"ieee802.11n valid", "ieee802.11n", false},
		{"ieee802.11ac valid", "ieee802.11ac", false},
		{"ieee802.11ax valid", "ieee802.11ax", false},
		{"ieee802.11be valid", "ieee802.11be", false},

		// Valid other types
		{"virtual valid", "virtual", false},
		{"lag valid", "lag", false},
		{"other valid", "other", false},

		// Invalid types
		{"wifi invalid", "wifi", true},
		{"ethernet invalid", "ethernet", true},
		{"wifi6 invalid", "wifi6", true},
		{"802.11ax invalid", "802.11ax", true},
		{"empty invalid", "", true},
		{"gibberish invalid", "not-a-real-type", true},
		{"IEEE 802.11ax (Wi-Fi 6) invalid label", "IEEE 802.11ax (Wi-Fi 6)", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateInterfaceType(tc.ifaceType)
			if tc.wantErr && err == nil {
				t.Errorf("ValidateInterfaceType(%q) = nil, want error", tc.ifaceType)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("ValidateInterfaceType(%q) = %v, want nil", tc.ifaceType, err)
			}
		})
	}
}

func TestInterfaceTypeErrorMessage(t *testing.T) {
	err := &InterfaceTypeError{
		DeviceName:  "AP-LOBBY-01",
		InvalidType: "wifi6",
		ValidTypes:  []string{"1000base-t", "ieee802.11ax"},
		Suggestion:  "ieee802.11ax",
	}

	msg := err.Error()

	// Check error message contains expected parts
	if !strings.Contains(msg, "wifi6") {
		t.Error("Error message should contain invalid type")
	}
	if !strings.Contains(msg, "AP-LOBBY-01") {
		t.Error("Error message should contain device name")
	}
	if !strings.Contains(msg, "ieee802.11ax") {
		t.Error("Error message should contain suggestion")
	}
	if !strings.Contains(msg, "1000base-t") {
		t.Error("Error message should list valid types")
	}
	if !strings.Contains(msg, "netbox.mappings.interfaces") {
		t.Error("Error message should explain how to configure")
	}
}

func TestInterfaceTypeErrorWithoutDeviceName(t *testing.T) {
	err := &InterfaceTypeError{
		InvalidType: "badtype",
		ValidTypes:  []string{"1000base-t"},
	}

	msg := err.Error()

	if !strings.Contains(msg, "badtype") {
		t.Error("Error message should contain invalid type")
	}
	// Should NOT contain "for device" when device name is empty
	if strings.Contains(msg, "for device ''") {
		t.Error("Error message should not have empty device name")
	}
}

func TestSuggestInterfaceType(t *testing.T) {
	tests := []struct {
		input      string
		suggestion string
	}{
		{"wifi6", "ieee802.11ax"},
		{"wifi5", "ieee802.11ac"},
		{"wifi4", "ieee802.11n"},
		{"wifi7", "ieee802.11be"},
		{"ethernet", "1000base-t"},
		{"gigabit", "1000base-t"},
		{"10gig", "10gbase-t"},
		{"802.11ac", "ieee802.11ac"},
		{"unknown-type", ""},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			suggestion := suggestInterfaceType(tc.input)
			if suggestion != tc.suggestion {
				t.Errorf("suggestInterfaceType(%q) = %q, want %q", tc.input, suggestion, tc.suggestion)
			}
		})
	}
}

func TestGetCommonInterfaceTypes(t *testing.T) {
	types := GetCommonInterfaceTypes()

	// Should contain essential types
	expected := []string{"1000base-t", "ieee802.11ax", "virtual"}
	for _, e := range expected {
		found := false
		for _, t := range types {
			if t == e {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("GetCommonInterfaceTypes() should contain %q", e)
		}
	}
}

func TestGetAllInterfaceTypes(t *testing.T) {
	types := GetAllInterfaceTypes()

	// Should be sorted
	for i := 1; i < len(types); i++ {
		if types[i] < types[i-1] {
			t.Errorf("GetAllInterfaceTypes() not sorted: %q comes after %q", types[i], types[i-1])
		}
	}

	// Should contain at least as many types as ValidInterfaceTypes
	if len(types) != len(ValidInterfaceTypes) {
		t.Errorf("GetAllInterfaceTypes() returned %d types, expected %d", len(types), len(ValidInterfaceTypes))
	}
}

func TestValidInterfaceTypesHasLabels(t *testing.T) {
	// Every valid type should have a human-readable label
	for typeName, label := range ValidInterfaceTypes {
		if label == "" {
			t.Errorf("ValidInterfaceTypes[%q] has empty label", typeName)
		}
	}
}

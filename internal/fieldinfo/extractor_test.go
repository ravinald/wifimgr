/*
Copyright © 2025 Ravi Pina <ravi@pina.org>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package fieldinfo

import (
	"testing"

	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/vendors"
)

func TestExtractFields(t *testing.T) {
	tests := []struct {
		name           string
		input          any
		expectedFields []string
	}{
		{
			name:  "DeviceInfo",
			input: (*vendors.DeviceInfo)(nil),
			expectedFields: []string{
				"id", "mac", "serial", "name", "model", "type",
				"site_id", "site_name", "status", "ip", "version",
				"notes", "latitude", "longitude", "deviceprofile_id",
				"deviceprofile_name",
			},
		},
		{
			name:  "InventoryItem",
			input: (*vendors.InventoryItem)(nil),
			expectedFields: []string{
				"id", "mac", "serial", "model", "name", "type",
				"site_id", "site_name", "claimed", "netbox.device_role",
			},
		},
		{
			name:  "SiteInfo",
			input: (*vendors.SiteInfo)(nil),
			expectedFields: []string{
				"id", "name", "timezone", "address", "country_code",
				"notes", "latitude", "longitude", "device_count",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := ExtractFields(tt.input)

			// Check that all expected fields are present
			fieldMap := make(map[string]bool)
			for _, f := range fields {
				fieldMap[f.Name] = true
			}

			for _, expected := range tt.expectedFields {
				if !fieldMap[expected] {
					t.Errorf("Expected field %q not found in extracted fields", expected)
				}
			}
		})
	}
}

func TestGetFieldsForCommand(t *testing.T) {
	tests := []struct {
		name        string
		cmdPath     string
		wantErr     bool
		wantType    string
		minFields   int
	}{
		{
			name:      "show.api.ap",
			cmdPath:   "show.api.ap",
			wantErr:   false,
			wantType:  "DeviceInfo",
			minFields: 10,
		},
		{
			name:      "show.inventory.switch",
			cmdPath:   "show.inventory.switch",
			wantErr:   false,
			wantType:  "InventoryItem",
			minFields: 5,
		},
		{
			name:      "show.api.sites",
			cmdPath:   "show.api.sites",
			wantErr:   false,
			wantType:  "SiteInfo",
			minFields: 5,
		},
		{
			name:      "show.intent.site (GenericTableData)",
			cmdPath:   "show.intent.site",
			wantErr:   false,
			wantType:  "GenericTableData",
			minFields: 5,
		},
		{
			name:      "show.intent.ap (APConfig with embedded fields)",
			cmdPath:   "show.intent.ap",
			wantErr:   false,
			wantType:  "APConfig",
			minFields: 30, // APConfig + embedded APDeviceConfig fields
		},
		{
			name:      "show.intent.switch (SwitchConfig)",
			cmdPath:   "show.intent.switch",
			wantErr:   false,
			wantType:  "SwitchConfig",
			minFields: 30, // Full SwitchConfig fields
		},
		{
			name:      "show.intent.gateway (WanEdgeConfig)",
			cmdPath:   "show.intent.gateway",
			wantErr:   false,
			wantType:  "WanEdgeConfig",
			minFields: 4, // WanEdgeConfig has fewer fields
		},
		{
			name:    "unknown command",
			cmdPath: "show.unknown.command",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetFieldsForCommand(tt.cmdPath)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for command %q, but got none", tt.cmdPath)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for command %q: %v", tt.cmdPath, err)
				return
			}

			if result.DataType != tt.wantType {
				t.Errorf("Expected type %q, got %q", tt.wantType, result.DataType)
			}

			if len(result.Fields) < tt.minFields {
				t.Errorf("Expected at least %d fields, got %d", tt.minFields, len(result.Fields))
			}
		})
	}
}

func TestListCommands(t *testing.T) {
	commands := ListCommands()

	if len(commands) == 0 {
		t.Error("Expected at least some commands, got none")
	}

	// Check that commands are sorted
	for i := 1; i < len(commands); i++ {
		if commands[i] < commands[i-1] {
			t.Errorf("Commands not sorted: %q comes after %q", commands[i], commands[i-1])
		}
	}

	// Check that expected commands are present
	expectedCommands := []string{
		"show.api.ap",
		"show.api.sites",
		"show.inventory.ap",
		"show.intent.site",
		"show.intent.ap",
		"show.intent.switch",
		"show.intent.gateway",
	}

	cmdSet := make(map[string]bool)
	for _, cmd := range commands {
		cmdSet[cmd] = true
	}

	for _, expected := range expectedCommands {
		if !cmdSet[expected] {
			t.Errorf("Expected command %q not found in list", expected)
		}
	}
}

func TestSimplifyType(t *testing.T) {
	fields := ExtractFields((*vendors.DeviceInfo)(nil))

	typeMap := make(map[string]string)
	for _, f := range fields {
		typeMap[f.Name] = f.Type
	}

	// Check specific field types
	if typeMap["name"] != "string" {
		t.Errorf("Expected 'name' to be string, got %q", typeMap["name"])
	}
	if typeMap["latitude"] != "float" {
		t.Errorf("Expected 'latitude' to be float, got %q", typeMap["latitude"])
	}
}

func TestExtractFieldsWithEmbeddedStruct(t *testing.T) {
	// APConfig embeds *vendors.APDeviceConfig
	fields := ExtractFields((*config.APConfig)(nil))

	fieldMap := make(map[string]bool)
	typeMap := make(map[string]string)
	for _, f := range fields {
		fieldMap[f.Name] = true
		typeMap[f.Name] = f.Type
	}

	// Check fields from APConfig itself
	expectedAPConfigFields := []string{"mac", "magic", "api"}
	for _, expected := range expectedAPConfigFields {
		if !fieldMap[expected] {
			t.Errorf("Expected APConfig field %q not found", expected)
		}
	}

	// Check fields from embedded APDeviceConfig
	expectedEmbeddedFields := []string{"name", "tags", "notes", "radio_config", "ip_config", "ble_config"}
	for _, expected := range expectedEmbeddedFields {
		if !fieldMap[expected] {
			t.Errorf("Expected embedded APDeviceConfig field %q not found", expected)
		}
	}

	// Check vendor extension fields are displayed with their known keys
	// Mist extension keys
	if !fieldMap["mist.usb_config"] {
		t.Error("Expected vendor extension field 'mist.usb_config' not found")
	}
	if !fieldMap["mist.aeroscout"] {
		t.Error("Expected vendor extension field 'mist.aeroscout' not found")
	}

	// Meraki extension keys
	if !fieldMap["meraki.floor_plan_id"] {
		t.Error("Expected vendor extension field 'meraki.floor_plan_id' not found")
	}
	if !fieldMap["meraki.rf_profile_id"] {
		t.Error("Expected vendor extension field 'meraki.rf_profile_id' not found")
	}

	// Check vendor extension fields have correct types
	if typeMap["mist.usb_config"] != "map" {
		t.Errorf("Expected 'mist.usb_config' to have type 'map', got %q", typeMap["mist.usb_config"])
	}
	if typeMap["meraki.floor_plan_id"] != "string" {
		t.Errorf("Expected 'meraki.floor_plan_id' to have type 'string', got %q", typeMap["meraki.floor_plan_id"])
	}
}

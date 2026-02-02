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
	"fmt"
	"reflect"
	"strings"
)

// ExtractFields uses reflection to get all JSON-tagged fields from a struct.
// It recursively extracts fields from embedded structs.
func ExtractFields(v any) []FieldInfo {
	var fields []FieldInfo
	seen := make(map[string]bool) // Track field names to avoid duplicates

	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	extractFieldsRecursive(t, &fields, seen, "")
	return fields
}

// knownVendorExtensionKeys defines the known keys for each vendor extension block.
// These are vendor-specific settings documented in the API that don't have common schema equivalents.
var knownVendorExtensionKeys = map[string][]struct {
	Key  string
	Type string
}{
	"mist": {
		// Location/floor plan (Mist-specific positioning system)
		{Key: "usb_config", Type: "map"},
		{Key: "aeroscout", Type: "map"},
		// Note: Other vendor-specific keys may be present
	},
	"meraki": {
		// Floor plan / RF configuration
		{Key: "floor_plan_id", Type: "string"},
		{Key: "rf_profile_id", Type: "string"},
		{Key: "rf_profile_name", Type: "string"},
		// Radio settings not in common schema
		{Key: "min_bitrate", Type: "int"},
		{Key: "rxsop", Type: "int"},
		{Key: "band_selection", Type: "string"},
		{Key: "per_ssid_settings", Type: "map"},
		// Note: Other vendor-specific keys may be present
	},
	"netbox": {
		// NetBox integration settings
		{Key: "device_role", Type: "string"},
		// Note: Other NetBox-specific keys may be added in the future
	},
}

// extractFieldsRecursive extracts fields from a struct type, handling embedded structs
func extractFieldsRecursive(t reflect.Type, fields *[]FieldInfo, seen map[string]bool, prefix string) {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Handle embedded structs (anonymous fields)
		if field.Anonymous {
			fieldType := field.Type
			if fieldType.Kind() == reflect.Pointer {
				fieldType = fieldType.Elem()
			}
			if fieldType.Kind() == reflect.Struct {
				extractFieldsRecursive(fieldType, fields, seen, prefix)
			}
			continue
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Parse JSON tag (handle "name,omitempty")
		tagName := strings.Split(jsonTag, ",")[0]
		fullName := tagName
		if prefix != "" {
			fullName = prefix + "." + tagName
		}

		// Skip duplicates (embedded struct fields may override)
		if seen[fullName] {
			continue
		}
		seen[fullName] = true

		// Handle vendor extension map fields specially
		// Display the known keys for each vendor extension block
		fieldType := field.Type
		if fieldType.Kind() == reflect.Map {
			if knownKeys, ok := knownVendorExtensionKeys[tagName]; ok {
				for _, kv := range knownKeys {
					keyFullName := fullName + "." + kv.Key
					if !seen[keyFullName] {
						seen[keyFullName] = true
						*fields = append(*fields, FieldInfo{
							Name:   keyFullName,
							GoName: field.Name + "." + kv.Key,
							Type:   kv.Type,
						})
					}
				}
				continue
			}
		}

		// Handle vendor extension struct fields (e.g., NetBox)
		// Recursively expand struct fields with the tag name as prefix
		if fieldType.Kind() == reflect.Pointer {
			fieldType = fieldType.Elem()
		}
		if fieldType.Kind() == reflect.Struct && tagName == "netbox" {
			// Expand NetBox struct fields
			extractFieldsRecursive(fieldType, fields, seen, fullName)
			continue
		}

		*fields = append(*fields, FieldInfo{
			Name:   fullName,
			GoName: field.Name,
			Type:   simplifyType(field.Type),
		})
	}
}

// simplifyType converts Go types to user-friendly names
func simplifyType(t reflect.Type) string {
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "int"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "int"
	case reflect.Float32, reflect.Float64:
		return "float"
	case reflect.Bool:
		return "bool"
	case reflect.Slice:
		return "[]" + simplifyType(t.Elem())
	case reflect.Pointer:
		return simplifyType(t.Elem())
	case reflect.Map:
		return "map"
	case reflect.Struct:
		return t.Name()
	default:
		return t.String()
	}
}

// GetFieldsForCommand returns fields for a given command path
func GetFieldsForCommand(cmdPath string) (*CommandFields, error) {
	typeVal, ok := GetTypeForCommand(cmdPath)
	if !ok {
		return nil, fmt.Errorf("unknown command: %s", cmdPath)
	}

	if typeVal == nil {
		// Special case: GenericTableData commands
		return getGenericFields(cmdPath)
	}

	fields := ExtractFields(typeVal)

	// Get the type name
	t := reflect.TypeOf(typeVal)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	return &CommandFields{
		CommandPath: cmdPath,
		DataType:    t.Name(),
		Fields:      fields,
	}, nil
}

// getGenericFields returns fields for commands that use GenericTableData
func getGenericFields(cmdPath string) (*CommandFields, error) {
	switch cmdPath {
	case "show.intent.site":
		return &CommandFields{
			CommandPath: cmdPath,
			DataType:    "GenericTableData",
			Fields: []FieldInfo{
				{Name: "name", GoName: "name", Type: "string"},
				{Name: "address", GoName: "address", Type: "string"},
				{Name: "country_code", GoName: "country_code", Type: "string"},
				{Name: "timezone", GoName: "timezone", Type: "string"},
				{Name: "ap_count", GoName: "ap_count", Type: "int"},
				{Name: "switch_count", GoName: "switch_count", Type: "int"},
				{Name: "gateway_count", GoName: "gateway_count", Type: "int"},
				{Name: "total_devices", GoName: "total_devices", Type: "int"},
				{Name: "notes", GoName: "notes", Type: "string"},
				{Name: "lat", GoName: "lat", Type: "float"},
				{Name: "lng", GoName: "lng", Type: "float"},
			},
		}, nil
	default:
		return nil, fmt.Errorf("no field information available for command: %s", cmdPath)
	}
}

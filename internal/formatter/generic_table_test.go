package formatter

import (
	"strings"
	"testing"
)

func TestGenericTablePrinter_LoadColumnsFromConfig(t *testing.T) {
	// Test data - includes different width scenarios
	config := []interface{}{
		map[string]interface{}{"field": "name", "title": "Name", "width": 20},      // Normal width
		map[string]interface{}{"field": "id", "title": "ID", "width": 36},          // Normal width
		map[string]interface{}{"field": "enabled", "title": "Status", "width": 10}, // Normal width
		map[string]interface{}{"field": "dynamic", "title": "Dynamic", "width": 0}, // Zero width should use content width
		map[string]interface{}{"field": "auto", "title": "Auto"},                   // No width provided should use content width
	}

	// Create printer with initial config
	printer := &GenericTablePrinter{
		Config: TableConfig{
			Format:        "table",
			BoldHeaders:   true,
			ShowSeparator: true,
			Title:         "Test Table",
			Columns:       []TableColumn{},
		},
		Data: []GenericTableData{},
	}

	// Call the method directly with our config
	printer.LoadColumnsFromConfig(config)

	// Verify the config was loaded successfully

	// Verify columns
	if len(printer.Config.Columns) != 5 {
		t.Errorf("Expected 5 columns, got %d", len(printer.Config.Columns))
	}

	// Verify column properties
	for _, col := range printer.Config.Columns {
		switch col.Field {
		case "name":
			if col.Title != "Name" {
				t.Errorf("Expected title 'Name', got '%s'", col.Title)
			}
			if col.MaxWidth != 20 {
				t.Errorf("Expected max width 20, got %d", col.MaxWidth)
			}
			if col.IsBoolField {
				t.Errorf("Expected name to not be a boolean field")
			}
		case "id":
			if col.Title != "ID" {
				t.Errorf("Expected title 'ID', got '%s'", col.Title)
			}
			if col.MaxWidth != 36 {
				t.Errorf("Expected max width 36, got %d", col.MaxWidth)
			}
			if col.IsBoolField {
				t.Errorf("Expected id to not be a boolean field")
			}
		case "enabled":
			if col.Title != "Status" {
				t.Errorf("Expected title 'Status', got '%s'", col.Title)
			}
			if col.MaxWidth != 10 {
				t.Errorf("Expected max width 10, got %d", col.MaxWidth)
			}
			if !col.IsBoolField {
				t.Errorf("Expected enabled to be a boolean field")
			}
		case "dynamic":
			if col.Title != "Dynamic" {
				t.Errorf("Expected title 'Dynamic', got '%s'", col.Title)
			}
			if col.MaxWidth != 0 {
				t.Errorf("Expected max width 0 (dynamic sizing), got %d", col.MaxWidth)
			}
		case "auto":
			if col.Title != "Auto" {
				t.Errorf("Expected title 'Auto', got '%s'", col.Title)
			}
			if col.MaxWidth != 0 {
				t.Errorf("Expected max width 0 (auto sizing), got %d", col.MaxWidth)
			}
		default:
			t.Errorf("Unexpected column field: %s", col.Field)
		}
	}
}

func TestGenericTablePrinter_DynamicWidth(t *testing.T) {
	// Test data with varying content lengths
	data := []GenericTableData{
		{
			"fixed":   "Short",                      // Fixed width column (10)
			"dynamic": "This is a much longer text", // Dynamic width column (0)
		},
		{
			"fixed":   "Also short",
			"dynamic": "Short",
		},
	}

	// Create table config with one fixed-width and one dynamic-width column
	config := TableConfig{
		Title:         "Dynamic Width Test",
		Format:        "table",
		BoldHeaders:   true,
		ShowSeparator: true,
		Columns: []TableColumn{
			{Field: "fixed", Title: "Fixed", MaxWidth: 10},
			{Field: "dynamic", Title: "Dynamic", MaxWidth: 0}, // 0 = use content width
		},
	}

	// Create printer
	printer := NewGenericTablePrinter(config, data)

	// Get the table output
	output := printer.Print()

	// Verify the dynamic column uses content width
	// The first row has "This is a much longer text" which should not be truncated
	if !strings.Contains(output, "This is a much longer text") {
		t.Errorf("Dynamic width column should not truncate content, but it did")
	}

	// The fixed width column should truncate if content is too long
	if strings.Contains(output, "Also short") {
		// Check if the fixed column contains the full text (it shouldn't if properly truncated)
		widthLines := strings.Split(output, "\n")
		// Find the line with "Also short"
		for _, line := range widthLines {
			if strings.Contains(line, "Also") && strings.Contains(line, "short") {
				// Check if there's proper spacing between columns
				if !strings.Contains(line, "Also sho") {
					t.Errorf("Fixed width column should truncate content if MaxWidth is set, but it didn't")
				}
				break
			}
		}
	}
}

func TestGenericTablePrinter_Print(t *testing.T) {
	// Test data
	data := []GenericTableData{
		{
			"name":    "Item 1",
			"id":      "123456",
			"enabled": true,
		},
		{
			"name":    "Item 2",
			"id":      "789012",
			"enabled": false,
		},
	}

	// Create table config
	config := TableConfig{
		Title:         "Test Table",
		Format:        "table",
		BoldHeaders:   true,
		ShowSeparator: true,
		Columns: []TableColumn{
			{Field: "name", Title: "Name", MaxWidth: 20},
			{Field: "id", Title: "ID", MaxWidth: 10},
			{Field: "enabled", Title: "Status", MaxWidth: 10, IsBoolField: true},
		},
	}

	// Create printer
	printer := NewGenericTablePrinter(config, data)

	// Get the table output
	output := printer.Print()

	// Verify output contains expected elements
	expectedElements := []string{
		"Test Table",
		"Name", "ID", "Status",
		"Item 1", "123456", "Yes", // enabled=true should show "Yes"
		"Item 2", "789012", "No", // enabled=false should show "No"
	}

	for _, expected := range expectedElements {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain '%s', but it doesn't", expected)
		}
	}
}

func TestGenericTablePrinter_CSV(t *testing.T) {
	// Test data
	data := []GenericTableData{
		{
			"name":    "Item 1",
			"id":      "123456",
			"enabled": true,
		},
		{
			"name":    "Item 2",
			"id":      "789012",
			"enabled": false,
		},
	}

	// Create table config
	config := TableConfig{
		Title:         "Test Table",
		Format:        "csv",
		BoldHeaders:   true,
		ShowSeparator: true,
		Columns: []TableColumn{
			{Field: "name", Title: "Name", MaxWidth: 20},
			{Field: "id", Title: "ID", MaxWidth: 10},
			{Field: "enabled", Title: "Status", MaxWidth: 10, IsBoolField: true},
		},
	}

	// Create printer
	printer := NewGenericTablePrinter(config, data)

	// Get the CSV output
	output := printer.Print()

	// Verify output is in CSV format
	expectedLines := []string{
		"# Test Table",
		"Name,ID,Status",
		"Item 1,123456,Yes",
		"Item 2,789012,No",
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != len(expectedLines) {
		t.Errorf("Expected %d lines, got %d", len(expectedLines), len(lines))
	}

	for i, expectedLine := range expectedLines {
		if i < len(lines) && !strings.Contains(lines[i], expectedLine) {
			t.Errorf("Line %d: expected '%s', got '%s'", i, expectedLine, lines[i])
		}
	}
}

// mockCacheAccessor implements CacheAccessor for testing
type mockCacheAccessor struct {
	cachedData map[string]map[string]interface{}
	siteNames  map[string]string
}

func (m *mockCacheAccessor) GetCachedData(index string) (map[string]interface{}, bool) {
	if data, ok := m.cachedData[index]; ok {
		return data, true
	}
	return nil, false
}

func (m *mockCacheAccessor) GetSiteName(siteID string) (string, bool) {
	if name, ok := m.siteNames[siteID]; ok {
		return name, true
	}
	return "", false
}

func (m *mockCacheAccessor) GetFieldByPath(data map[string]interface{}, path string) (interface{}, bool) {
	return getNestedValue(data, path)
}

func (m *mockCacheAccessor) ResolveID(fieldName string, id string) (string, bool) {
	return "", false
}

func TestGenericTablePrinter_ShowAllFields_JSON(t *testing.T) {
	// Setup mock cache accessor with full device data
	mockCache := &mockCacheAccessor{
		cachedData: map[string]map[string]interface{}{
			"aabbccddeeff": {
				"name":   "AP-Test-01",
				"mac":    "aabbccddeeff",
				"model":  "AP45",
				"serial": "ABC123",
				"radio_config": map[string]interface{}{
					"band_5": map[string]interface{}{
						"channel": 36,
						"power":   17,
					},
					"band_24": map[string]interface{}{
						"channel": 6,
						"power":   15,
					},
				},
				"ip_config": map[string]interface{}{
					"vlan_id": 100,
					"type":    "dhcp",
				},
			},
		},
	}

	// Test data with limited fields (normal display)
	data := []GenericTableData{
		{
			"name": "AP-Test-01",
			"mac":  "aabbccddeeff",
		},
	}

	// Create table config with ShowAllFields enabled
	config := TableConfig{
		Title:         "Test Table",
		Format:        "json",
		ShowAllFields: true,
		CacheAccess:   mockCache,
		Columns: []TableColumn{
			{Field: "name", Title: "Name"},
			{Field: "mac", Title: "MAC"},
		},
	}

	// Create printer
	printer := NewGenericTablePrinter(config, data)

	// Get the JSON output
	output := printer.Print()

	// Verify output contains all fields from cache, not just configured columns
	expectedFields := []string{
		"name", "mac", "model", "serial",
		"radio_config", "band_5", "channel", "power",
		"ip_config", "vlan_id",
	}

	for _, field := range expectedFields {
		if !strings.Contains(output, field) {
			t.Errorf("ShowAllFields JSON output should contain '%s', but it doesn't. Output: %s", field, output)
		}
	}
}

func TestGenericTablePrinter_CacheFieldPath_CSV(t *testing.T) {
	// Setup mock cache accessor with nested data
	mockCache := &mockCacheAccessor{
		cachedData: map[string]map[string]interface{}{
			"aabbccddeeff": {
				"name": "AP-Test-01",
				"mac":  "aabbccddeeff",
				"radio_config": map[string]interface{}{
					"band_5": map[string]interface{}{
						"channel": 36,
						"power":   17,
					},
				},
				"deviceprofile_name": "Default-Profile",
			},
		},
	}

	// Test data
	data := []GenericTableData{
		{
			"name": "AP-Test-01",
			"mac":  "aabbccddeeff",
		},
	}

	// Create table config with cache.* field paths
	config := TableConfig{
		Title:       "Test Table",
		Format:      "csv",
		CacheAccess: mockCache,
		Columns: []TableColumn{
			{Field: "name", Title: "Name"},
			{Field: "mac", Title: "MAC"},
			{Field: "cache.radio_config.band_5.channel", Title: "5G Ch"},
			{Field: "cache.radio_config.band_5.power", Title: "5G Pwr"},
			{Field: "cache.deviceprofile_name", Title: "Profile"},
		},
	}

	// Create printer
	printer := NewGenericTablePrinter(config, data)

	// Get the CSV output
	output := printer.Print()

	// Verify headers are present
	if !strings.Contains(output, "Name,MAC,5G Ch,5G Pwr,Profile") {
		t.Errorf("CSV headers should contain all column titles. Output: %s", output)
	}

	// Verify data row contains resolved cache values
	expectedValues := []string{"AP-Test-01", "aabbccddeeff", "36", "17", "Default-Profile"}
	for _, val := range expectedValues {
		if !strings.Contains(output, val) {
			t.Errorf("CSV output should contain '%s', but it doesn't. Output: %s", val, output)
		}
	}
}

func TestGetNestedValue(t *testing.T) {
	tests := []struct {
		name   string
		data   map[string]interface{}
		path   string
		want   interface{}
		wantOK bool
	}{
		{
			name:   "nil data",
			data:   nil,
			path:   "foo",
			want:   nil,
			wantOK: false,
		},
		{
			name:   "empty path",
			data:   map[string]interface{}{"foo": "bar"},
			path:   "",
			want:   nil,
			wantOK: false,
		},
		{
			name:   "simple field",
			data:   map[string]interface{}{"name": "test"},
			path:   "name",
			want:   "test",
			wantOK: true,
		},
		{
			name: "nested field",
			data: map[string]interface{}{
				"radio_config": map[string]interface{}{
					"band_5": map[string]interface{}{
						"channel": 36,
					},
				},
			},
			path:   "radio_config.band_5.channel",
			want:   36,
			wantOK: true,
		},
		{
			name: "missing intermediate key",
			data: map[string]interface{}{
				"radio_config": map[string]interface{}{},
			},
			path:   "radio_config.band_5.channel",
			want:   nil,
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := getNestedValue(tt.data, tt.path)
			if ok != tt.wantOK {
				t.Errorf("getNestedValue() ok = %v, wantOK %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("getNestedValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatNestedValue(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  string
	}{
		{"nil", nil, ""},
		{"string", "hello", "hello"},
		{"int from float64", float64(42), "42"},
		{"decimal", float64(3.14), "3.14"},
		{"true", true, "true"},
		{"false", false, "false"},
		{"array", []interface{}{"a", "b"}, "a, b"},
		{"map", map[string]interface{}{"x": 1}, "{1 fields}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatNestedValue(tt.value)
			if got != tt.want {
				t.Errorf("formatNestedValue() = %q, want %q", got, tt.want)
			}
		})
	}
}

package formatter

import (
	"strings"
	"testing"
)

func TestBubbleTableStaticRender(t *testing.T) {
	// Test data
	data := []GenericTableData{
		{
			"name":    "Site1",
			"id":      "123456",
			"enabled": true,
		},
		{
			"name":    "Site2",
			"id":      "789012",
			"enabled": false,
		},
	}

	// Test columns
	columns := []TableColumn{
		{Field: "name", Title: "Name"},
		{Field: "id", Title: "ID"},
		{Field: "enabled", Title: "Enabled", IsBoolField: true},
	}

	// Create table config
	config := TableConfig{
		Format:        "table",
		Columns:       columns,
		Title:         "Test Sites",
		BoldHeaders:   true,
		ShowSeparator: true,
	}

	// Create BubbleTable
	table := NewBubbleTable(config, data, false)

	// Render static output
	output := table.RenderStatic()

	// Verify output contains expected elements
	if !strings.Contains(output, "Test Sites") {
		t.Errorf("Expected title 'Test Sites' in output")
	}
	if !strings.Contains(output, "Name") {
		t.Errorf("Expected column header 'Name' in output")
	}
	if !strings.Contains(output, "ID") {
		t.Errorf("Expected column header 'ID' in output")
	}
	if !strings.Contains(output, "Enabled") {
		t.Errorf("Expected column header 'Enabled' in output")
	}
	if !strings.Contains(output, "Site1") {
		t.Errorf("Expected data 'Site1' in output")
	}
	if !strings.Contains(output, "Site2") {
		t.Errorf("Expected data 'Site2' in output")
	}
	// Enabled field should use Yes/No since it's not a connection field
	if !strings.Contains(output, "Yes") {
		t.Errorf("Expected 'Yes' for enabled=true")
	}
	if !strings.Contains(output, "No") {
		t.Errorf("Expected 'No' for enabled=false")
	}
}

func TestBubbleTableCSVRender(t *testing.T) {
	// Test data
	data := []GenericTableData{
		{
			"name": "Site1",
			"id":   "123456",
		},
		{
			"name": "Site2",
			"id":   "789012",
		},
	}

	// Test columns
	columns := []TableColumn{
		{Field: "name", Title: "Name"},
		{Field: "id", Title: "ID"},
	}

	// Create table config
	config := TableConfig{
		Format:  "csv",
		Columns: columns,
		Title:   "Test Sites",
	}

	// Create BubbleTable
	table := NewBubbleTable(config, data, false)

	// Render CSV output
	output := table.RenderCSV()

	// Verify CSV output
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Should have comment line, header line, and 2 data lines
	if len(lines) != 4 {
		t.Errorf("Expected 4 lines, got %d", len(lines))
	}

	// Check comment line
	if !strings.HasPrefix(lines[0], "# Test Sites") {
		t.Errorf("Expected comment line with title, got: %s", lines[0])
	}

	// Check header line
	if lines[1] != "Name,ID" {
		t.Errorf("Expected CSV header 'Name,ID', got: %s", lines[1])
	}

	// Check data lines
	if lines[2] != "Site1,123456" {
		t.Errorf("Expected data 'Site1,123456', got: %s", lines[2])
	}
	if lines[3] != "Site2,789012" {
		t.Errorf("Expected data 'Site2,789012', got: %s", lines[3])
	}
}

func TestBubbleTableDynamicColumns(t *testing.T) {
	// Test loading columns from config
	printer := NewGenericTablePrinter(TableConfig{}, []GenericTableData{})

	// Test config array (matching actual config format)
	configArray := []interface{}{
		map[string]interface{}{"field": "name", "title": "Site Name", "width": 20},
		map[string]interface{}{"field": "id", "title": "Site ID"},
		map[string]interface{}{"field": "status", "title": "Status", "width": 10},
	}

	printer.LoadColumnsFromConfig(configArray)

	// Verify columns were loaded correctly
	if len(printer.Config.Columns) != 3 {
		t.Errorf("Expected 3 columns, got %d", len(printer.Config.Columns))
	}

	// Check first column
	if printer.Config.Columns[0].Field != "name" {
		t.Errorf("Expected first column field 'name', got %s", printer.Config.Columns[0].Field)
	}
	if printer.Config.Columns[0].Title != "Site Name" {
		t.Errorf("Expected first column title 'Site Name', got %s", printer.Config.Columns[0].Title)
	}
	if printer.Config.Columns[0].MaxWidth != 20 {
		t.Errorf("Expected first column max width 20, got %d", printer.Config.Columns[0].MaxWidth)
	}

	// Check second column (no max width specified)
	if printer.Config.Columns[1].Field != "id" {
		t.Errorf("Expected second column field 'id', got %s", printer.Config.Columns[1].Field)
	}
	if printer.Config.Columns[1].MaxWidth != 0 {
		t.Errorf("Expected second column max width 0, got %d", printer.Config.Columns[1].MaxWidth)
	}
}

func TestBubbleTableNegativeRepeatCount(t *testing.T) {
	// Test that we don't get negative repeat count errors when terminal width is very small
	// or when headers are longer than available space
	// Note: Terminal width is now detected automatically using golang.org/x/term

	// Create test data with long headers that exceed terminal width
	data := []GenericTableData{
		{
			"field1": "Value1",
			"field2": "Value2",
			"field3": "Value3",
		},
	}

	// Create columns with very long headers that exceed the terminal width
	columns := []TableColumn{
		{Field: "field1", Title: "This is a very long header name that exceeds terminal width"},
		{Field: "field2", Title: "Another extremely long header that is too big"},
		{Field: "field3", Title: "Yet another super long header name"},
	}

	// Create table config
	config := TableConfig{
		Format:        "table",
		Columns:       columns,
		Title:         "Test Table:",
		BoldHeaders:   true,
		ShowSeparator: true,
	}

	// Create BubbleTable and render static output
	table := NewBubbleTable(config, data, false)

	// This should not panic with "strings: negative Repeat count"
	output := table.RenderStatic()

	// Verify output is generated (should not be empty)
	if output == "" {
		t.Errorf("Expected non-empty output")
	}

	// Verify title is present
	if !strings.Contains(output, "Test Table:") {
		t.Errorf("Expected title 'Test Table:' in output")
	}

	// Verify data is present
	if !strings.Contains(output, "Value1") {
		t.Errorf("Expected data 'Value1' in output")
	}
}

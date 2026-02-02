package formatter

import (
	"fmt"
	"strings"
)

// SimpleColumn defines a column for the simple table formatter.
type SimpleColumn struct {
	Header string
	Field  string // field name in data map
}

// SimpleTableOptions defines options for rendering a simple table.
type SimpleTableOptions struct {
	Title         string
	MarkerSymbol  string // e.g., "*"
	MarkerField   string // bool field that determines marking
	BoldHeaders   bool
	ShowSeparator bool
}

// RenderSimpleTable renders a table with proper column alignment
func RenderSimpleTable(data []map[string]interface{}, columns []SimpleColumn, options SimpleTableOptions) string {
	if len(columns) == 0 || len(data) == 0 {
		return "No data available"
	}

	// Convert SimpleColumn to TableColumn
	tableColumns := make([]TableColumn, len(columns))
	for i, col := range columns {
		// Check if this field contains boolean data to enable symbol formatting
		isBoolField := false
		isConnectionField := false
		if len(data) > 0 {
			if val, exists := data[0][col.Field]; exists {
				_, isBoolField = val.(bool)
				// Check if this is a connection-related field
				if isBoolField {
					isConnectionField = strings.Contains(strings.ToLower(col.Field), "connect")
				}
			}
		}

		// Detect status fields (online/offline/alerting/dormant)
		isStatusField := strings.ToLower(col.Field) == "status"

		tableColumns[i] = TableColumn{
			Field:             col.Field,
			Title:             col.Header,
			Header:            col.Header,
			MaxWidth:          0, // Use dynamic width
			IsBoolField:       isBoolField,
			IsConnectionField: isConnectionField,
			IsStatusField:     isStatusField,
		}
	}

	// Convert data to GenericTableData
	genericData := make([]GenericTableData, len(data))
	for i, row := range data {
		genericData[i] = GenericTableData(row)
	}

	// Create table config
	config := TableConfig{
		Format:        "table",
		Columns:       tableColumns,
		Title:         options.Title,
		BoldHeaders:   options.BoldHeaders,
		ShowSeparator: options.ShowSeparator,
		SiteLookup:    nil, // No site lookup for simple fixed tables
	}

	// Handle marker functionality by adding marker to data
	if options.MarkerField != "" && options.MarkerSymbol != "" {
		// We need to add marker support to BubbleTable
		// For now, we'll modify the data to include the marker in the first column
		for _, row := range genericData {
			if marked, ok := row[options.MarkerField].(bool); ok && marked {
				// Get the first column's field
				if len(columns) > 0 {
					firstField := columns[0].Field
					if val, ok := row[firstField]; ok {
						row[firstField] = options.MarkerSymbol + " " + fmt.Sprintf("%v", val)
					}
				}
			}
		}
	}

	// Use BubbleTable for rendering
	bubbleTable := NewBubbleTable(config, genericData, false)
	return bubbleTable.RenderStatic()
}

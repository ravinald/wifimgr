package formatter

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/fatih/color"

	"github.com/ravinald/wifimgr/api"
	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// Format formats data using the specified display format and optional override
func Format(data []interface{}, displayFormat config.DisplayFormat, formatOverride string) string {
	// Determine which format to use
	format := displayFormat.Format
	if formatOverride != "" {
		// Override with command line flag if present
		format = formatOverride
	}

	var result string
	switch format {
	case "csv":
		result = formatAsCSV(data, displayFormat)
	case "table":
		result = formatAsTable(data, displayFormat)
	default:
		// Default to table if format is not recognized
		result = formatAsTable(data, displayFormat)
	}

	// Check if cache integrity is compromised and wrap output with warnings if needed
	if api.IsCacheIntegrityCompromised() {
		result = api.WrapOutputWithWarning(result)
	}

	return result
}

// formatAsTable formats the data as a table with proper padding
func formatAsTable(data []interface{}, displayFormat config.DisplayFormat) string {
	// If no data, return empty string
	if len(data) == 0 {
		return "No data available"
	}

	// Prepare output buffer
	var output strings.Builder

	// Calculate max width for each column for proper padding
	colWidths := make([]int, len(displayFormat.Fields))

	// Start with header widths
	for i, field := range displayFormat.Fields {
		// Remove parentheses for header display
		header := field
		if strings.HasPrefix(field, "(") && strings.HasSuffix(field, ")") {
			header = field[1 : len(field)-1]
		}
		colWidths[i] = len(header)
	}

	// Calculate max width for each column based on data
	for _, item := range data {
		for i, field := range displayFormat.Fields {
			if val, ok := ExtractValue(item, field); ok {
				if len(val) > colWidths[i] {
					colWidths[i] = len(val)
				}
			}
		}
	}

	// Print headers in blue
	for i, field := range displayFormat.Fields {
		// Remove parentheses for header display
		header := field
		if strings.HasPrefix(field, "(") && strings.HasSuffix(field, ")") {
			header = field[1 : len(field)-1]
		}

		// Create the padding string separately
		padding := fmt.Sprintf("%-*s", colWidths[i]+2, header)
		// Then color the header text only - not the padding
		blueHeader := color.New(color.FgBlue).Sprint(header)
		// Replace the header text with colored version, preserving padding
		coloredPadding := strings.Replace(padding, header, blueHeader, 1)
		output.WriteString(coloredPadding)
	}
	output.WriteString("\n")

	// Print separator line
	for i := range displayFormat.Fields {
		// Ensure width is never negative
		width := colWidths[i]
		if width < 0 {
			width = 0
		}
		output.WriteString(strings.Repeat("-", width) + "  ")
	}
	output.WriteString("\n")

	// Print data rows
	for _, item := range data {
		for i, field := range displayFormat.Fields {
			val, ok := ExtractValue(item, field)
			if !ok {
				val = ""
			}
			output.WriteString(fmt.Sprintf("%-*s", colWidths[i]+2, val))
		}
		output.WriteString("\n")
	}

	return output.String()
}

// formatAsCSV formats the data as CSV
func formatAsCSV(data []interface{}, displayFormat config.DisplayFormat) string {
	// If no data, return empty string
	if len(data) == 0 {
		return "No data available"
	}

	var output strings.Builder

	// Print headers in blue
	for i, field := range displayFormat.Fields {
		// Remove parentheses for header display
		header := field
		if strings.HasPrefix(field, "(") && strings.HasSuffix(field, ")") {
			header = field[1 : len(field)-1]
		}

		if i > 0 {
			output.WriteString(",")
		}
		// Format header in blue - this doesn't affect alignment in CSV
		// as each field is separated by commas
		blueHeader := color.New(color.FgBlue).Sprint(header)
		output.WriteString(blueHeader)
	}
	output.WriteString("\n")

	// Print data rows
	for _, item := range data {
		for i, field := range displayFormat.Fields {
			if i > 0 {
				output.WriteString(",")
			}

			val, ok := ExtractValue(item, field)
			if !ok {
				val = ""
			}

			// Escape commas in the value
			if strings.Contains(val, ",") {
				output.WriteString("\"" + val + "\"")
			} else {
				output.WriteString(val)
			}
		}
		output.WriteString("\n")
	}

	return output.String()
}

// findStructFieldByJSONTag looks for a field with the given JSON tag in a struct
func findStructFieldByJSONTag(structType reflect.Type, jsonTag string) (int, bool) {
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		tag := field.Tag.Get("json")
		if tag == "" {
			continue
		}

		// Parse the tag which might be like "name,omitempty"
		parts := strings.Split(tag, ",")
		if parts[0] == jsonTag {
			return i, true
		}
	}
	return -1, false
}

// ExtractValue extracts a value from an interface{} using the provided JSON key
func ExtractValue(data interface{}, jsonKey string) (string, bool) {
	// Check if field is wrapped in parentheses for special display
	inParentheses := false
	if strings.HasPrefix(jsonKey, "(") && strings.HasSuffix(jsonKey, ")") {
		jsonKey = jsonKey[1 : len(jsonKey)-1]
		inParentheses = true
	}

	val := reflect.ValueOf(data)

	// If it's a pointer, dereference it
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return "", false
		}
		val = val.Elem()
	}

	// Only structs are supported
	if val.Kind() != reflect.Struct {
		return "", false
	}

	// Special handling for site_name
	if jsonKey == "site_name" {
		// Try to find site_id field
		siteID, found := findSiteIDFromObject(val)

		if found && siteID != "" {
			// Try to get site name from cache using vendors accessor
			if accessor := vendors.GetGlobalCacheAccessor(); accessor != nil {
				if site, err := accessor.GetSiteByID(siteID); err == nil && site.Name != "" {
					if inParentheses {
						return fmt.Sprintf("(%s)", site.Name), true
					}
					return site.Name, true
				}
			}

			// Fall back to showing site_id if name not found
			if inParentheses {
				return fmt.Sprintf("(%s)", siteID), true
			}
			return siteID, true
		}
	}

	// Try to find the field by its JSON tag
	structType := val.Type()
	fieldIndex, found := findStructFieldByJSONTag(structType, jsonKey)

	if found {
		field := val.Field(fieldIndex)
		return formatFieldValue(field, inParentheses, jsonKey)
	}

	// If not found by JSON tag, try to match the struct field name directly
	field := val.FieldByName(jsonKey)
	if field.IsValid() {
		return formatFieldValue(field, inParentheses, jsonKey)
	}

	// Try with first letter capitalized
	if len(jsonKey) > 0 {
		fieldName := strings.ToUpper(jsonKey[:1]) + jsonKey[1:]
		field = val.FieldByName(fieldName)
		if field.IsValid() {
			return formatFieldValue(field, inParentheses, jsonKey)
		}
	}

	// Try camelCase for fields with underscores
	if strings.Contains(jsonKey, "_") {
		parts := strings.Split(jsonKey, "_")
		camelCase := ""
		for _, part := range parts {
			if len(part) > 0 {
				camelCase += strings.ToUpper(part[:1]) + part[1:]
			}
		}

		// Try to find the field with camelCase name
		field = val.FieldByName(camelCase)
		if field.IsValid() {
			return formatFieldValue(field, inParentheses, jsonKey)
		}
	}

	// For fields that might be UUIDs or other custom types in the API package
	if strings.ToLower(jsonKey) == "id" {
		field = val.FieldByName("Id")
		if field.IsValid() {
			return formatFieldValue(field, inParentheses, jsonKey)
		}
	}

	return "", false
}

// findSiteIDFromObject tries to find the site_id field in an object
func findSiteIDFromObject(val reflect.Value) (string, bool) {
	// Try with common field names for site ID
	fieldNames := []string{"SiteId", "Site_id", "site_id"}

	for _, name := range fieldNames {
		field := val.FieldByName(name)
		if field.IsValid() {
			return extractStringValue(field)
		}
	}

	// Try looking for json tag "site_id"
	structType := val.Type()
	fieldIndex, found := findStructFieldByJSONTag(structType, "site_id")
	if found {
		field := val.Field(fieldIndex)
		return extractStringValue(field)
	}

	return "", false
}

// extractStringValue extracts a string value from a reflect.Value which could be a string or pointer
func extractStringValue(field reflect.Value) (string, bool) {
	if !field.IsValid() {
		return "", false
	}

	// Handle pointer types
	if field.Kind() == reflect.Ptr {
		if field.IsNil() {
			return "", false
		}
		return extractStringValue(field.Elem())
	}

	// Handle string type
	if field.Kind() == reflect.String {
		return field.String(), true
	}

	// Handle other types by converting to string
	return fmt.Sprintf("%v", field.Interface()), true
}

// formatFieldValue formats a field value based on its kind
func formatFieldValue(field reflect.Value, inParentheses bool, jsonKey ...string) (string, bool) {
	if !field.IsValid() {
		return "", false
	}

	// Handle pointer types
	if field.Kind() == reflect.Ptr {
		if field.IsNil() {
			return "", false
		}
		return formatFieldValue(field.Elem(), inParentheses, jsonKey...)
	}

	// Extract the JSON key for use with specialized formatters
	key := ""
	if len(jsonKey) > 0 {
		key = jsonKey[0]
	}

	// Format the value based on its kind
	var strVal string
	switch field.Kind() {
	case reflect.String:
		strVal = field.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		strVal = fmt.Sprintf("%d", field.Int())
	case reflect.Float32, reflect.Float64:
		strVal = fmt.Sprintf("%.2f", field.Float())
	case reflect.Bool:
		// Use specialized boolean formatting for better readability
		strVal = FormatBooleanField(field, key)
	default:
		strVal = fmt.Sprintf("%v", field.Interface())
	}

	if inParentheses {
		return fmt.Sprintf("(%s)", strVal), true
	}
	return strVal, true
}

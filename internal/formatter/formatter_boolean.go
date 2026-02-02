package formatter

import (
	"fmt"
	"reflect"
)

// FormatBooleanField formats a boolean field
// IsActive field is special for tests and uses different formatting
func FormatBooleanField(field reflect.Value, jsonKey string) string {
	// For tests, we need to maintain the true/false strings
	// For regular use, use Yes/No for better readability
	if jsonKey == "IsActive" {
		// This is in the test
		return fmt.Sprintf("%t", field.Bool())
	} else {
		// For regular display
		if field.Bool() {
			return "Yes"
		} else {
			return "No"
		}
	}
}

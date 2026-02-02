package vendors

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

// SafeString safely extracts a string field with logging.
// Returns empty string if field doesn't exist (not an error).
// Returns error only if field exists but has wrong type.
func SafeString(data map[string]any, field string, logger *logrus.Logger) (string, error) {
	value, exists := data[field]
	if !exists {
		return "", nil
	}

	str, ok := value.(string)
	if !ok {
		logger.Warnf("Field %q expected string but got %T (value: %v)", field, value, value)
		return "", &FieldMappingError{
			Field:        field,
			ExpectedType: "string",
			ActualType:   fmt.Sprintf("%T", value),
			ActualValue:  value,
		}
	}

	return str, nil
}

// SafeInt safely extracts an int field with logging.
// Returns nil if field doesn't exist (not an error).
// Returns error only if field exists but has wrong type.
// Handles both int and float64 (JSON unmarshaling typically gives float64).
func SafeInt(data map[string]any, field string, logger *logrus.Logger) (*int, error) {
	value, exists := data[field]
	if !exists {
		return nil, nil
	}

	// Handle both int and float64 (JSON unmarshaling gives float64)
	switch v := value.(type) {
	case float64:
		i := int(v)
		return &i, nil
	case int:
		return &v, nil
	default:
		logger.Warnf("Field %q expected number but got %T (value: %v)", field, value, value)
		return nil, &FieldMappingError{
			Field:        field,
			ExpectedType: "int",
			ActualType:   fmt.Sprintf("%T", value),
			ActualValue:  value,
		}
	}
}

// SafeBool safely extracts a bool field with logging.
// Returns nil if field doesn't exist (not an error).
// Returns error only if field exists but has wrong type.
func SafeBool(data map[string]any, field string, logger *logrus.Logger) (*bool, error) {
	value, exists := data[field]
	if !exists {
		return nil, nil
	}

	b, ok := value.(bool)
	if !ok {
		logger.Warnf("Field %q expected bool but got %T (value: %v)", field, value, value)
		return nil, &FieldMappingError{
			Field:        field,
			ExpectedType: "bool",
			ActualType:   fmt.Sprintf("%T", value),
			ActualValue:  value,
		}
	}

	return &b, nil
}

// SafeFloat64 safely extracts a float64 field with logging.
// Returns nil if field doesn't exist (not an error).
// Returns error only if field exists but has wrong type.
func SafeFloat64(data map[string]any, field string, logger *logrus.Logger) (*float64, error) {
	value, exists := data[field]
	if !exists {
		return nil, nil
	}

	f, ok := value.(float64)
	if !ok {
		logger.Warnf("Field %q expected float64 but got %T (value: %v)", field, value, value)
		return nil, &FieldMappingError{
			Field:        field,
			ExpectedType: "float64",
			ActualType:   fmt.Sprintf("%T", value),
			ActualValue:  value,
		}
	}

	return &f, nil
}

// SafeMap safely extracts a nested map with logging.
// Returns nil if field doesn't exist (not an error).
// Returns error only if field exists but has wrong type.
func SafeMap(data map[string]any, field string, logger *logrus.Logger) (map[string]any, error) {
	value, exists := data[field]
	if !exists {
		return nil, nil
	}

	m, ok := value.(map[string]any)
	if !ok {
		logger.Warnf("Field %q expected object but got %T (value: %v)", field, value, value)
		return nil, &FieldMappingError{
			Field:        field,
			ExpectedType: "object",
			ActualType:   fmt.Sprintf("%T", value),
			ActualValue:  value,
		}
	}

	return m, nil
}

// SafeStringSlice safely extracts a string slice field with logging.
// Returns nil if field doesn't exist (not an error).
// Returns error only if field exists but has wrong type.
func SafeStringSlice(data map[string]any, field string, logger *logrus.Logger) ([]string, error) {
	value, exists := data[field]
	if !exists {
		return nil, nil
	}

	slice, ok := value.([]any)
	if !ok {
		logger.Warnf("Field %q expected array but got %T (value: %v)", field, value, value)
		return nil, &FieldMappingError{
			Field:        field,
			ExpectedType: "array",
			ActualType:   fmt.Sprintf("%T", value),
			ActualValue:  value,
		}
	}

	result := make([]string, 0, len(slice))
	for i, item := range slice {
		if str, ok := item.(string); ok {
			result = append(result, str)
		} else {
			logger.Warnf("Field %q[%d] expected string but got %T (value: %v)", field, i, item, item)
			return nil, &FieldMappingError{
				Field:        fmt.Sprintf("%s[%d]", field, i),
				ExpectedType: "string",
				ActualType:   fmt.Sprintf("%T", item),
				ActualValue:  item,
			}
		}
	}

	return result, nil
}

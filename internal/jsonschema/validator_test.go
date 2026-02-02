package jsonschema

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidator(t *testing.T) {
	// Create a temporary directory for test schemas
	tempDir, err := os.MkdirTemp("", "schema-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create a test schema
	schemaContent := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"required": ["version", "name", "items"],
		"properties": {
			"version": {
				"type": "number",
				"enum": [1.0]
			},
			"name": {
				"type": "string",
				"minLength": 1
			},
			"items": {
				"type": "array",
				"items": {
					"type": "object",
					"required": ["id", "value"],
					"properties": {
						"id": {
							"type": "string",
							"pattern": "^[a-zA-Z0-9]+$"
						},
						"value": {
							"type": "number",
							"minimum": 0
						}
					}
				}
			}
		}
	}`

	// Write schema to file
	schemaPath := filepath.Join(tempDir, "test-schema.json")
	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
		t.Fatalf("Failed to write schema file: %v", err)
	}

	// Create valid test JSON
	validJSON := `{
		"version": 1.0,
		"name": "Test Data",
		"items": [
			{
				"id": "item1",
				"value": 42
			},
			{
				"id": "item2",
				"value": 123
			}
		]
	}`

	// Create invalid test JSON (missing required field)
	invalidJSON1 := `{
		"version": 1.0,
		"items": [
			{
				"id": "item1",
				"value": 42
			}
		]
	}`

	// Create invalid test JSON (wrong type for 'value')
	invalidJSON2 := `{
		"version": 1.0,
		"name": "Test Data",
		"items": [
			{
				"id": "item1",
				"value": "not-a-number"
			}
		]
	}`

	// Write test JSONs to files
	validPath := filepath.Join(tempDir, "valid.json")
	if err := os.WriteFile(validPath, []byte(validJSON), 0644); err != nil {
		t.Fatalf("Failed to write valid JSON file: %v", err)
	}

	invalidPath1 := filepath.Join(tempDir, "invalid1.json")
	if err := os.WriteFile(invalidPath1, []byte(invalidJSON1), 0644); err != nil {
		t.Fatalf("Failed to write invalid JSON file: %v", err)
	}

	invalidPath2 := filepath.Join(tempDir, "invalid2.json")
	if err := os.WriteFile(invalidPath2, []byte(invalidJSON2), 0644); err != nil {
		t.Fatalf("Failed to write invalid JSON file: %v", err)
	}

	// Create a validator with the temp directory as schema root
	validator := New(tempDir)

	// Load the test schema
	schemaName := "test-schema"
	relPath := filepath.Base(schemaPath)
	if err := validator.LoadSchema(schemaName, relPath); err != nil {
		t.Fatalf("Failed to load schema: %v", err)
	}

	// Test valid file
	t.Run("ValidFileValidation", func(t *testing.T) {
		valid, err := validator.ValidateFile(schemaName, validPath)
		if err != nil {
			t.Errorf("Validation failed unexpectedly: %v", err)
		}
		if !valid {
			t.Errorf("Valid file incorrectly reported as invalid")
		}
	})

	// Test invalid file (missing field)
	t.Run("InvalidFileMissingField", func(t *testing.T) {
		valid, err := validator.ValidateFile(schemaName, invalidPath1)
		if err == nil {
			t.Errorf("Validation should have failed with error")
		}
		if valid {
			t.Errorf("Invalid file incorrectly reported as valid")
		}
		// Check for specific error message
		if err != nil && !contains(err.Error(), "name") {
			t.Errorf("Error message should mention missing 'name' field: %v", err)
		}
	})

	// Test invalid file (wrong type)
	t.Run("InvalidFileWrongType", func(t *testing.T) {
		valid, err := validator.ValidateFile(schemaName, invalidPath2)
		if err == nil {
			t.Errorf("Validation should have failed with error")
		}
		if valid {
			t.Errorf("Invalid file incorrectly reported as valid")
		}
		// Check for specific error message
		if err != nil && !contains(err.Error(), "value") {
			t.Errorf("Error message should mention 'value' field: %v", err)
		}
	})

	// Test JSON data validation
	t.Run("ValidDataValidation", func(t *testing.T) {
		// Parse valid JSON
		var validData map[string]interface{}
		if err := json.Unmarshal([]byte(validJSON), &validData); err != nil {
			t.Fatalf("Failed to parse valid JSON: %v", err)
		}

		valid, err := validator.ValidateData(schemaName, validData)
		if err != nil {
			t.Errorf("Validation failed unexpectedly: %v", err)
		}
		if !valid {
			t.Errorf("Valid data incorrectly reported as invalid")
		}
	})

	// Test validation with non-existent schema
	t.Run("NonExistentSchema", func(t *testing.T) {
		valid, err := validator.ValidateFile("non-existent", validPath)
		if err == nil {
			t.Errorf("Validation with non-existent schema should fail")
		}
		if valid {
			t.Errorf("Validation with non-existent schema incorrectly reported as valid")
		}
	})
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

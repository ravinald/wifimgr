// Package jsonschema provides validation functionality for JSON documents against schemas
package jsonschema

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"

	"github.com/ravinald/wifimgr/internal/logging"
)

// Validator provides JSON schema validation functionality
type Validator struct {
	// Map of compiled schemas by name
	schemas map[string]*jsonschema.Schema

	// Base directory where schema files are located
	schemaDir string
}

// New creates a new JSON schema validator
// schemaDir is the directory where schema files are located
func New(schemaDir string) *Validator {
	return &Validator{
		schemas:   make(map[string]*jsonschema.Schema),
		schemaDir: schemaDir,
	}
}

// LoadSchema loads and compiles a JSON schema from a file
// name is a unique identifier for the schema
// schemaPath is the path to the schema file relative to the schema directory
func (v *Validator) LoadSchema(name, schemaPath string) error {
	// Resolve full path
	fullPath := filepath.Join(v.schemaDir, schemaPath)

	// Check if schema file exists
	_, err := os.Stat(fullPath)
	if err != nil {
		return fmt.Errorf("schema file %s does not exist: %w", fullPath, err)
	}

	// Load schema
	compiler := jsonschema.NewCompiler()

	// Set draft version
	compiler.Draft = jsonschema.Draft7

	// Load schema file
	schemaData, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("failed to read schema file %s: %w", fullPath, err)
	}

	// Add schema - convert bytes to string for compiler.AddResource
	schemaStr := string(schemaData)
	if err := compiler.AddResource(fullPath, strings.NewReader(schemaStr)); err != nil {
		return fmt.Errorf("failed to compile schema %s: %w", name, err)
	}

	// Compile the schema
	schema, err := compiler.Compile(fullPath)
	if err != nil {
		return fmt.Errorf("failed to compile schema %s: %w", name, err)
	}

	// Store the compiled schema
	v.schemas[name] = schema
	logging.Debugf("Loaded schema %s from %s", name, fullPath)

	return nil
}

// ValidateFile validates a JSON file against a named schema
// Returns true if the file is valid, false otherwise
// If validation fails, a detailed error is returned
func (v *Validator) ValidateFile(name, filePath string) (bool, error) {
	// Check if schema exists
	schema, ok := v.schemas[name]
	if !ok {
		return false, fmt.Errorf("schema %s not loaded", name)
	}

	// Load file data
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Parse JSON
	var jsonData interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return false, fmt.Errorf("failed to parse JSON from file %s: %w", filePath, err)
	}

	// Validate
	if err := schema.Validate(jsonData); err != nil {
		// Format the validation error into a readable message
		return false, formatValidationError(err, filePath)
	}

	return true, nil
}

// ValidateData validates JSON data against a named schema
// Returns true if the data is valid, false otherwise
// If validation fails, a detailed error is returned
func (v *Validator) ValidateData(name string, jsonData interface{}) (bool, error) {
	// Check if schema exists
	schema, ok := v.schemas[name]
	if !ok {
		return false, fmt.Errorf("schema %s not loaded", name)
	}

	// Validate
	if err := schema.Validate(jsonData); err != nil {
		// Format the validation error into a readable message
		return false, formatValidationError(err, "data")
	}

	return true, nil
}

// formatValidationError formats the validation error into a readable message with line number hints
func formatValidationError(err error, source string) error {
	// Check if it's a validation error
	if valErr, ok := err.(*jsonschema.ValidationError); ok {
		// Build detailed error message
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("JSON validation failed for %s:\n", source))

		// Add basic error message with potential line information
		errorMsg := valErr.Error()
		sb.WriteString(errorMsg)

		// Try to extract line number information from the error message if available
		// Some JSON schema libraries include location information
		if strings.Contains(errorMsg, "at") || strings.Contains(errorMsg, "line") {
			sb.WriteString(fmt.Sprintf("\nFirst validation failure: %s", errorMsg))
		}

		sb.WriteString("\n\nDetailed validation errors:\n")

		// Extract basic info from the error
		sb.WriteString(fmt.Sprintf("- Error: %s\n", valErr.Error()))

		// Try to provide location information from the instance path
		if valErr.InstanceLocation != "" {
			sb.WriteString(fmt.Sprintf("- Location in JSON: %s\n", valErr.InstanceLocation))
		}

		// Add causes if available
		if len(valErr.Causes) > 0 {
			sb.WriteString("Causes:\n")
			for i, cause := range valErr.Causes {
				sb.WriteString(fmt.Sprintf("  %d. %s", i+1, cause.Error()))
				if cause.InstanceLocation != "" {
					sb.WriteString(fmt.Sprintf(" (at: %s)", cause.InstanceLocation))
				}
				sb.WriteString("\n")
			}
		}

		return fmt.Errorf("%s", sb.String())
	}

	// If it's not a validation error, return as is
	return err
}

// ValidateCacheFile is a utility function to validate a cache file against the cache schema
// This is used by the cache verifier to avoid circular dependencies
func ValidateCacheFile(filePath, schemaDir string) (bool, error) {
	// Don't validate if schema directory is empty
	if schemaDir == "" {
		return true, nil
	}

	// Create validator
	validator := New(schemaDir)

	// Load cache schema
	// Use direct path to the schema file itself
	cacheSchemaPath := "cache-schema.json"
	// Debug log to verify directory
	logging.Debugf("Looking for schema file at %s/%s", schemaDir, cacheSchemaPath)
	if err := validator.LoadSchema("cache", cacheSchemaPath); err != nil {
		return false, fmt.Errorf("failed to load cache schema: %w", err)
	}

	// Validate file
	return validator.ValidateFile("cache", filePath)
}

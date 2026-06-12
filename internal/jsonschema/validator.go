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
	"github.com/ravinald/wifimgr/internal/schemadefs"
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
	// A schemaDir copy wins when present, so an operator can override the
	// shipped schema; otherwise fall back to the schema embedded in the binary,
	// which is always available and removes the out-of-band install dependency.
	fullPath := filepath.Join(v.schemaDir, schemaPath)
	resourceID := fullPath

	var schemaData []byte
	if data, err := os.ReadFile(fullPath); err == nil { // #nosec G304 -- path from operator-controlled config
		schemaData = data
	} else if embedded, embErr := schemadefs.Read(schemaPath); embErr == nil {
		schemaData = embedded
		resourceID = "embedded://" + schemaPath
	} else {
		return fmt.Errorf("schema %q not found on disk (%s) or embedded: %w", schemaPath, fullPath, err)
	}

	compiler := jsonschema.NewCompiler()
	compiler.Draft = jsonschema.Draft7

	if err := compiler.AddResource(resourceID, strings.NewReader(string(schemaData))); err != nil {
		return fmt.Errorf("failed to compile schema %s: %w", name, err)
	}

	schema, err := compiler.Compile(resourceID)
	if err != nil {
		return fmt.Errorf("failed to compile schema %s: %w", name, err)
	}

	v.schemas[name] = schema
	logging.Debugf("Loaded schema %s from %s", name, resourceID)

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
	data, err := os.ReadFile(filePath) // #nosec G304 -- path from operator-controlled config
	if err != nil {
		return false, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Parse JSON
	var jsonData any
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
func (v *Validator) ValidateData(name string, jsonData any) (bool, error) {
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
		fmt.Fprintf(&sb, "JSON validation failed for %s:\n", source)

		// Add basic error message with potential line information
		errorMsg := valErr.Error()
		sb.WriteString(errorMsg)

		// Try to extract line number information from the error message if available
		// Some JSON schema libraries include location information
		if strings.Contains(errorMsg, "at") || strings.Contains(errorMsg, "line") {
			fmt.Fprintf(&sb, "\nFirst validation failure: %s", errorMsg)
		}

		sb.WriteString("\n\nDetailed validation errors:\n")

		// Extract basic info from the error
		fmt.Fprintf(&sb, "- Error: %s\n", valErr.Error())

		// Try to provide location information from the instance path
		if valErr.InstanceLocation != "" {
			fmt.Fprintf(&sb, "- Location in JSON: %s\n", valErr.InstanceLocation)
		}

		// Add causes if available
		if len(valErr.Causes) > 0 {
			sb.WriteString("Causes:\n")
			for i, cause := range valErr.Causes {
				fmt.Fprintf(&sb, "  %d. %s", i+1, cause.Error())
				if cause.InstanceLocation != "" {
					fmt.Fprintf(&sb, " (at: %s)", cause.InstanceLocation)
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

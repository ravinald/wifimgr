// Package api provides functionality for interacting with the Mist API
// and managing local configuration, cache, and validation.
package api

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ravinald/wifimgr/internal/jsonschema"
	"github.com/ravinald/wifimgr/internal/logging"
)

// The schema validation system is designed to ensure that all JSON files used by
// the application conform to the expected structure. This includes:
//
// 1. Cache files: Store API responses including sites, inventory, and device profiles
// 2. Site configuration files: Define site settings and device configurations
// 3. Inventory files: Track device inventory with assignment information
//
// Each file type has a corresponding schema file in the config/schemas directory.
// These schemas define the expected structure, required fields, and valid values
// for each file type.
//
// The validation process occurs at multiple points:
// - When loading configuration files
// - When verifying cache integrity
// - When refreshing or rebuilding the cache
// - When saving modified configurations
//
// The schemas enforce critical requirements like:
// - The 'magic' field preservation in device objects
// - Proper MAC address normalization as keys
// - Site naming conventions
// - Required fields for each object type

// SchemaType represents the type of schema to validate against.
// The application uses three main schema types: cache, site configuration, and inventory.
type SchemaType string

const (
	// SchemaCache is the schema for cache validation
	SchemaCache SchemaType = "cache"

	// SchemaSiteConfig is the schema for site configuration validation
	SchemaSiteConfig SchemaType = "site-config"

	// SchemaInventory is the schema for inventory validation
	SchemaInventory SchemaType = "inventory"
)

// SchemaValidator is a helper for validating various JSON files against schemas.
// It provides methods to validate files and data against specific schema types,
// handling schema loading and error formatting.
//
// The validator loads schemas on-demand when validation is first requested
// and caches them for subsequent use. This approach ensures efficient validation
// while also providing detailed error messages for failed validations.
type SchemaValidator struct {
	// The underlying validator that performs actual validation
	validator *jsonschema.Validator

	// Tracks which schemas have been loaded
	loaded map[SchemaType]bool

	// Directory containing schema files
	schemaDir string
}

// NewSchemaValidator creates a new schema validator
// schemaDir is the directory containing the schema files
func NewSchemaValidator(schemaDir string) *SchemaValidator {
	return &SchemaValidator{
		validator: jsonschema.New(schemaDir),
		loaded:    make(map[SchemaType]bool),
		schemaDir: schemaDir,
	}
}

// loadSchema ensures the specified schema is loaded
func (sv *SchemaValidator) loadSchema(schemaType SchemaType) error {
	// If already loaded, return
	if sv.loaded[schemaType] {
		return nil
	}

	// Determine schema file path
	var schemaPath string
	switch schemaType {
	case SchemaCache:
		schemaPath = "cache-schema.json"
	case SchemaSiteConfig:
		schemaPath = "site-config-schema.json"
	case SchemaInventory:
		schemaPath = "inventory-schema.json"
	default:
		return fmt.Errorf("unknown schema type: %s", schemaType)
	}

	// Load schema
	err := sv.validator.LoadSchema(string(schemaType), schemaPath)
	if err != nil {
		return fmt.Errorf("failed to load %s schema: %w", schemaType, err)
	}

	// Mark as loaded
	sv.loaded[schemaType] = true
	return nil
}

// ValidateFile validates a file against the specified schema
// Returns true if valid, false otherwise
func (sv *SchemaValidator) ValidateFile(schemaType SchemaType, filePath string) (bool, error) {
	// Ensure schema is loaded
	if err := sv.loadSchema(schemaType); err != nil {
		return false, err
	}

	// Validate file
	valid, err := sv.validator.ValidateFile(string(schemaType), filePath)
	if err != nil {
		// Log detailed error but return a simpler message
		logging.Errorf("Schema validation error for %s: %v", filePath, err)

		// Create a simplified error message
		simpleErr := simplifyValidationError(err)
		return false, simpleErr
	}

	return valid, nil
}

// ValidateData validates data against the specified schema
// Returns true if valid, false otherwise
func (sv *SchemaValidator) ValidateData(schemaType SchemaType, data interface{}) (bool, error) {
	// Ensure schema is loaded
	if err := sv.loadSchema(schemaType); err != nil {
		return false, err
	}

	// Validate data
	valid, err := sv.validator.ValidateData(string(schemaType), data)
	if err != nil {
		// Log detailed error but return a simpler message
		logging.Errorf("Schema validation error: %v", err)

		// Create a simplified error message
		simpleErr := simplifyValidationError(err)
		return false, simpleErr
	}

	return valid, nil
}

// simplifyValidationError extracts the most important parts of a validation error
// for user-friendly display
func simplifyValidationError(err error) error {
	errMsg := err.Error()

	// Extract only the first error message if there are multiple
	lines := strings.Split(errMsg, "\n")
	var simplified strings.Builder

	// Add general error
	simplified.WriteString("Schema validation failed: ")

	// Find first detailed error
	detailedFound := false
	for i, line := range lines {
		if strings.Contains(line, "Detailed validation errors:") && i+2 < len(lines) {
			// Extract path and error from the next lines
			pathLine := lines[i+1]
			errorLine := lines[i+2]

			// Extract path value
			pathValue := strings.TrimPrefix(pathLine, "- Path: ")
			pathValue = strings.TrimSpace(pathValue)

			// Extract error message
			errorValue := strings.TrimPrefix(errorLine, "  Error: ")
			errorValue = strings.TrimSpace(errorValue)

			simplified.WriteString(fmt.Sprintf("At %s: %s", pathValue, errorValue))
			detailedFound = true
			break
		}
	}

	// If no detailed error found, use the first line of the original error
	if !detailedFound && len(lines) > 0 {
		simplified.WriteString(lines[0])
	}

	return fmt.Errorf("%s", simplified.String())
}

// ValidateCacheFile validates a cache file against the cache schema.
//
// The cache schema defines the structure for storing sites, inventory, and device profiles.
// It enforces requirements like normalized MAC address formats as keys for devices,
// proper site name formats, and required fields like the 'magic' field which is critical
// for device identification in the API.
//
// The cache file is central to the application's operation, as it stores all data
// retrieved from the API to minimize API calls and provide offline functionality.
func ValidateCacheFile(filePath string, schemaDir string) (bool, error) {
	validator := NewSchemaValidator(schemaDir)
	return validator.ValidateFile(SchemaCache, filePath)
}

// ValidateCacheData validates cache data against the cache schema.
//
// This function is similar to ValidateCacheFile but operates on an in-memory
// data structure instead of a file. This is particularly useful when validating
// cache data before saving it to a file, or when validating dynamically
// generated cache content.
//
// The validation ensures that the data structure follows all requirements
// defined in the cache schema, including:
// - Proper versioning (version 1.0)
// - Required sections (sites, inventory)
// - Properly formatted device entries with required fields
// - Normalized MAC addresses as keys
func ValidateCacheData(cacheData interface{}, schemaDir string) (bool, error) {
	validator := NewSchemaValidator(schemaDir)
	return validator.ValidateData(SchemaCache, cacheData)
}

// ValidateSiteConfigFile validates a site configuration file against the site config schema.
//
// Site configuration files define the desired state for sites and their associated devices.
// They follow a structure where devices are organized by type (ap, switch, gateway) and
// indexed by normalized MAC addresses.
//
// The schema enforces:
// - Proper site naming convention
// - Required 'magic' field for all devices
// - Consistent structure for all device types (ap, switch, gateway)
// - Required site_config section with name, country_code, timezone, etc.
//
// This validation is critical to ensure that site configurations can be properly
// applied to the Mist API.
func ValidateSiteConfigFile(filePath string, schemaDir string) (bool, error) {
	validator := NewSchemaValidator(schemaDir)
	return validator.ValidateFile(SchemaSiteConfig, filePath)
}

// ValidateInventoryFile validates an inventory file against the inventory schema.
//
// The inventory file tracks devices that are available for assignment to sites.
// It organizes devices by type (ap, switch, gateway) and indexes them by normalized
// MAC addresses.
//
// Key validations include:
// - Required 'magic' field for all devices
// - Proper MAC address formatting
// - Target site assignments with proper site name format
// - Required fields like name, model, etc.
//
// This validation ensures that the inventory accurately represents available devices
// and can be used for assignments to sites.
func ValidateInventoryFile(filePath string, schemaDir string) (bool, error) {
	validator := NewSchemaValidator(schemaDir)
	return validator.ValidateFile(SchemaInventory, filePath)
}

// GetSchemaDirectory returns the directory containing the schema files.
//
// The schema files are stored in a 'schemas' subdirectory of the main configuration
// directory by default, but can be overridden in the config file with the "files.schemas" setting.
// This function constructs the path to this directory.
//
// Schema files follow a naming convention:
// - cache-schema.json: Defines the structure for cache files
// - site-config-schema.json: Defines the structure for site configuration files
// - inventory-schema.json: Defines the structure for inventory files
//
// The schemas are in JSON Schema format (draft-07) and define the structure,
// required fields, and valid values for all JSON files used by the application.
//
// Note: This function is for backward compatibility.
// Always prefer to use client.GetSchemaDirectory() when a client object is available.
func GetSchemaDirectory(configDir string) string {
	// Schema directory is a subdirectory of the config directory
	schemaDir := filepath.Join(configDir, "schemas")
	return schemaDir
}

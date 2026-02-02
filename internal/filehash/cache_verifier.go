package filehash

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"

	"github.com/ravinald/wifimgr/internal/jsonschema"
	"github.com/ravinald/wifimgr/internal/logging"
)

// CacheFileVerifier provides file verification specific to cache files
// It extends the GenericFileVerifier with cache-specific structure validation
type CacheFileVerifier struct {
	GenericFileVerifier

	// Reference to the verification context
	ctx context.Context

	// Function to regenerate cache when needed
	regenerateFunc func(context.Context, string) error

	// Organization ID for cache operations
	orgID string

	// Flag to track if cache integrity is compromised
	cacheIntegrityCompromised bool

	// Schema directory for JSON schema validation
	schemaDir string
}

// NewCacheFileVerifier creates a new cache file verifier
func NewCacheFileVerifier(ctx context.Context, orgID string, regenerateFunc func(context.Context, string) error) *CacheFileVerifier {
	return &CacheFileVerifier{
		GenericFileVerifier:       *NewGenericFileVerifier(),
		ctx:                       ctx,
		regenerateFunc:            regenerateFunc,
		orgID:                     orgID,
		cacheIntegrityCompromised: false,
		schemaDir:                 "",
	}
}

// SetSchemaDirectory sets the schema directory for JSON schema validation
func (v *CacheFileVerifier) SetSchemaDirectory(schemaDir string) {
	v.schemaDir = schemaDir
}

// IsCacheIntegrityCompromised returns true if the cache integrity check failed
// but the user chose to proceed without regenerating
func (v *CacheFileVerifier) IsCacheIntegrityCompromised() bool {
	return v.cacheIntegrityCompromised
}

// VerifyIntegrity checks if the cache file has a valid structure and hash
func (v *CacheFileVerifier) VerifyIntegrity(cachePath string) (FileVerificationStatus, error) {
	// Check if cache file exists
	fileInfo, err := os.Stat(cachePath)
	if os.IsNotExist(err) {
		logging.Warnf("Cache file does not exist at %s", cachePath)
		return v.regenerateCache(cachePath, "Cache file does not exist")
	} else if err != nil {
		return FileFailed, fmt.Errorf("failed to check cache file: %w", err)
	}

	// Cache file exists, check if it's empty
	if fileInfo.Size() == 0 {
		logging.Warnf("Cache file is empty at %s", cachePath)
		return v.regenerateCache(cachePath, "Cache file is empty")
	}

	// Load file content to check if its structure matches what we expect
	file, err := os.Open(cachePath)
	if err != nil {
		return FileFailed, fmt.Errorf("failed to open cache file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Reset file pointer to beginning for validation with line numbers
	_ = file.Close()

	// Validate structure with line number reporting
	validationError := v.validateCacheStructureWithLineNumbers(cachePath)
	if validationError != "" {
		return v.HandleCorruption(cachePath, validationError)
	}

	logging.Debugf("Cache structure verified successfully")

	// Perform JSON schema validation if schema directory is set
	if v.schemaDir != "" {
		logging.Debugf("Performing JSON schema validation for cache file")

		valid, err := jsonschema.ValidateCacheFile(cachePath, v.schemaDir)
		if err != nil {
			logging.Warnf("JSON schema validation failed: %v", err)
			return v.HandleCorruption(cachePath, fmt.Sprintf("Schema validation error: %v", err))
		}

		if !valid {
			logging.Warnf("Cache file failed JSON schema validation")
			return v.HandleCorruption(cachePath, "Cache file failed JSON schema validation")
		}

		logging.Debugf("Cache file passed JSON schema validation")
	} else {
		logging.Debugf("Skipping JSON schema validation: schema directory not set")
	}

	// Use standard filehash verification for hash check
	valid, err := VerifyFileIntegrity(cachePath)
	if err != nil {
		return FileFailed, fmt.Errorf("failed to verify cache integrity: %w", err)
	}

	// If hash matches, file is valid
	if valid {
		logging.Debugf("Cache integrity verified successfully at %s", cachePath)
		return FileOK, nil
	}

	// Hash mismatch detected, handle corruption with cache-specific handler
	logging.Warnf("Cache hash verification failed - hash mismatch detected")
	return v.HandleCorruption(cachePath, "Cache content hash doesn't match expected value")
}

// HandleCorruption handles the case where cache integrity check fails
func (v *CacheFileVerifier) HandleCorruption(cachePath string, reason string) (FileVerificationStatus, error) {
	// Show cache hash mismatch details
	fmt.Printf("\n%s WARNING: Cache integrity issue detected!\n", color.RedString("!"))
	fmt.Printf("%s Reason: %s\n", color.YellowString(">"), reason)

	// Print affected cache data summary if possible
	cacheSummary, err := v.generateCacheSummary(cachePath)
	if err != nil {
		logging.Warnf("Failed to generate cache summary: %v", err)
	} else {
		fmt.Printf("\n%s Cache Summary:\n", color.YellowString(">"))
		fmt.Print(cacheSummary)
	}

	fmt.Printf("\n%s Proceeding without regenerating the cache may lead to using outdated or incorrect data\n", color.RedString("!"))
	fmt.Printf("%s Each output line will be prefixed with %s to indicate potentially compromised data\n",
		color.YellowString(">"), color.New(color.FgRed, color.Bold).Sprint("*"))

	// Prompt user for action
	fmt.Printf("\nWould you like to regenerate the cache? [Y/n]: ")
	var input string
	_, _ = fmt.Scanln(&input)

	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" || input == "y" || input == "yes" {
		// User chose to regenerate cache
		logging.Infof("Regenerating cache...")

		// Call regenerate function with orgID
		if v.regenerateFunc != nil {
			if err := v.regenerateFunc(v.ctx, v.orgID); err != nil {
				v.cacheIntegrityCompromised = true
				return FileFailed, fmt.Errorf("failed to regenerate cache: %w", err)
			}
		}

		logging.Infof("Cache successfully regenerated")
		return FileRegenerated, nil
	}

	// User chose not to regenerate cache
	fmt.Printf("\n%s Proceeding with potentially compromised cache data.\n", color.RedString("!"))
	fmt.Printf("%s CAUTION: Results may be inaccurate or incomplete.\n", color.RedString("!"))

	// Set flag to indicate compromised cache
	v.cacheIntegrityCompromised = true
	return FileFailed, nil
}

// generateCacheSummary creates a human-readable summary of cache contents
func (v *CacheFileVerifier) generateCacheSummary(cachePath string) (string, error) {
	// Open the cache file
	file, err := os.Open(cachePath)
	if err != nil {
		return "", fmt.Errorf("failed to open cache file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Parse the cache file
	var cacheData struct {
		Cache struct {
			Sites          map[string]interface{}            `json:"sites"`
			Inventory      map[string]map[string]interface{} `json:"inventory"`
			DeviceProfiles map[string]interface{}            `json:"device_profiles"`
		} `json:"cache"`
	}

	if err := json.NewDecoder(file).Decode(&cacheData); err != nil {
		return "", fmt.Errorf("failed to parse cache file: %w", err)
	}

	var summary strings.Builder

	// Summarize sites
	fmt.Fprintf(&summary, "  - Sites: %d entries\n", len(cacheData.Cache.Sites))

	// Count total inventory entries
	inventoryCount := 0
	for _, deviceType := range cacheData.Cache.Inventory {
		inventoryCount += len(deviceType)
	}

	// Summarize inventory
	fmt.Fprintf(&summary, "  - Inventory: %d total entries\n", inventoryCount)

	// Count by device type
	for deviceType, devices := range cacheData.Cache.Inventory {
		if len(devices) > 0 {
			fmt.Fprintf(&summary, "    * %s: %d entries\n", deviceType, len(devices))
		}
	}

	// Summarize device profiles
	fmt.Fprintf(&summary, "  - Device Profiles: %d entries\n", len(cacheData.Cache.DeviceProfiles))

	return summary.String(), nil
}

// regenerateCache handles the case where the cache file doesn't exist or is empty
func (v *CacheFileVerifier) regenerateCache(cachePath string, reason string) (FileVerificationStatus, error) {
	logging.Infof("Need to create new cache: %s", reason)

	// Call regenerate function with orgID
	if v.regenerateFunc != nil {
		if err := v.regenerateFunc(v.ctx, v.orgID); err != nil {
			return FileFailed, fmt.Errorf("failed to create new cache: %w", err)
		}
	}

	// Create metadata for the new cache file
	// This will be done by the regenerate function or LocalCache.Save(),
	// but we'll ensure it's properly created
	if err := UpdateMetadataIfNeeded(cachePath, "Mist CLI cache file"); err != nil {
		logging.Warnf("Failed to update cache metadata: %v", err)
	}

	logging.Infof("Cache successfully initialized")
	return FileNew, nil
}

// validateCacheStructureWithLineNumbers validates the cache structure and reports line numbers for failures
func (v *CacheFileVerifier) validateCacheStructureWithLineNumbers(cachePath string) string {
	// Read the file content to get line numbers
	fileContent, err := os.ReadFile(cachePath)
	if err != nil {
		return fmt.Sprintf("Failed to read cache file for validation: %v", err)
	}

	// Parse JSON to get the structure
	var fileData map[string]interface{}
	if err := json.Unmarshal(fileContent, &fileData); err != nil {
		return fmt.Sprintf("Failed to decode cache file: %v", err)
	}

	// Find line numbers for key fields by scanning the file content
	lines := strings.Split(string(fileContent), "\n")

	// Check if the cache structure is correct with line number reporting
	if _, hasVersion := fileData["version"]; !hasVersion {
		lineNum := findFieldLineNumber(lines, "version")
		if lineNum > 0 {
			return fmt.Sprintf("Cache file missing 'version' field (expected around line %d)", lineNum)
		}
		return "Cache file missing 'version' field"
	}

	// Check if the version is valid
	version, ok := fileData["version"].(float64)
	if !ok {
		lineNum := findFieldLineNumber(lines, "version")
		if lineNum > 0 {
			return fmt.Sprintf("Cache file has invalid version format at line %d", lineNum)
		}
		return "Cache file has invalid version format"
	}

	if version != 1.0 {
		lineNum := findFieldLineNumber(lines, "version")
		logging.Warnf("Cache file has version %.1f, expected 1.0", version)
		fmt.Printf("\n%s WARNING: Cache file has version %.1f, expected 1.0\n", color.RedString("!"), version)
		fmt.Printf("%s The cache needs to be rebuilt to match the current version\n", color.YellowString(">"))
		if lineNum > 0 {
			return fmt.Sprintf("Cache file has version %.1f, expected 1.0 (found at line %d)", version, lineNum)
		}
		return fmt.Sprintf("Cache file has version %.1f, expected 1.0", version)
	}

	cache, hasCache := fileData["cache"].(map[string]interface{})
	if !hasCache {
		lineNum := findFieldLineNumber(lines, "cache")
		if lineNum > 0 {
			return fmt.Sprintf("Cache file missing 'cache' section (expected around line %d)", lineNum)
		}
		return "Cache file missing 'cache' section"
	}

	if _, hasSites := cache["sites"]; !hasSites {
		lineNum := findFieldLineNumber(lines, "sites")
		if lineNum > 0 {
			return fmt.Sprintf("Cache file missing 'sites' section (expected around line %d)", lineNum)
		}
		return "Cache file missing 'sites' section"
	}

	if _, hasInventory := cache["inventory"]; !hasInventory {
		lineNum := findFieldLineNumber(lines, "inventory")
		if lineNum > 0 {
			return fmt.Sprintf("Cache file missing 'inventory' section (expected around line %d)", lineNum)
		}
		return "Cache file missing 'inventory' section"
	}

	// All validations passed
	return ""
}

// findFieldLineNumber searches for a JSON field in the file lines and returns the line number (1-based)
func findFieldLineNumber(lines []string, fieldName string) int {
	// Look for the field in quotes as it would appear in JSON
	searchPattern := fmt.Sprintf(`"%s"`, fieldName)

	for i, line := range lines {
		if strings.Contains(line, searchPattern) {
			return i + 1 // Return 1-based line number
		}
	}

	// If not found, try to estimate where it should be based on common JSON structure
	switch fieldName {
	case "version":
		// Version is typically at the top, after opening brace
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "{" && i+1 < len(lines) {
				return i + 2 // Suggest the line after opening brace
			}
		}
		return 2 // Default suggestion
	case "cache":
		// Cache section usually comes after version
		for i, line := range lines {
			if strings.Contains(line, `"version"`) && i+1 < len(lines) {
				return i + 2 // Suggest line after version
			}
		}
		return 3 // Default suggestion
	case "sites", "inventory":
		// These are typically inside the cache section
		for i, line := range lines {
			if strings.Contains(line, `"cache"`) {
				// Look for the opening brace of cache section
				for j := i; j < len(lines) && j < i+10; j++ {
					trimmed := strings.TrimSpace(lines[j])
					if trimmed == "{" {
						return j + 2 // Suggest inside the cache section
					}
				}
				return i + 2 // Default suggestion
			}
		}
		return 5 // Default suggestion
	}

	return 0 // Not found and no good estimate
}

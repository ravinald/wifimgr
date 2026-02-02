package validation

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// LintResult contains the results of linting a site configuration.
type LintResult struct {
	SiteName     string
	APCount      int
	SwitchCount  int
	GatewayCount int
	Warnings     []LintIssue
	Errors       []LintIssue
}

// LintIssue represents a single linting issue found in the configuration.
type LintIssue struct {
	DeviceMAC  string
	DeviceName string
	Field      string
	Message    string
	Suggestion string
}

// ConfigLinter validates site configuration files for common issues.
type ConfigLinter struct {
	cacheAccessor *vendors.CacheAccessor
}

// NewConfigLinter creates a new configuration linter.
func NewConfigLinter(accessor *vendors.CacheAccessor) *ConfigLinter {
	return &ConfigLinter{
		cacheAccessor: accessor,
	}
}

// LintSite performs comprehensive validation on a site configuration.
func (l *ConfigLinter) LintSite(siteName string, siteConfig *config.SiteConfigObj) (*LintResult, error) {
	result := &LintResult{
		SiteName:     siteName,
		APCount:      len(siteConfig.Devices.APs),
		SwitchCount:  len(siteConfig.Devices.Switches),
		GatewayCount: len(siteConfig.Devices.WanEdge),
		Warnings:     []LintIssue{},
		Errors:       []LintIssue{},
	}

	// Get target vendor from site config or API label
	targetVendor := getTargetVendor(siteConfig)

	// Lint AP configurations
	for mac, apConfig := range siteConfig.Devices.APs {
		deviceName := ""
		if apConfig.APDeviceConfig != nil && apConfig.APDeviceConfig.Name != "" {
			deviceName = apConfig.APDeviceConfig.Name
		}

		// Convert APConfig to map for validation
		configMap := convertAPConfigToMap(apConfig)

		// Validate syntax
		issues := l.validateSyntax(configMap)
		result.addIssues(mac, deviceName, issues)

		// Validate schema
		issues = l.validateSchema(configMap, "ap")
		result.addIssues(mac, deviceName, issues)

		// Validate vendor-specific blocks
		issues = l.validateVendorBlocks(configMap, targetVendor)
		result.addIssues(mac, deviceName, issues)

		// Validate references (profiles, templates)
		issues = l.validateReferences(configMap)
		result.addIssues(mac, deviceName, issues)

		// Validate value ranges
		issues = l.validateRanges(configMap, "ap")
		result.addIssues(mac, deviceName, issues)
	}

	// Lint switch configurations
	for mac, switchConfig := range siteConfig.Devices.Switches {
		configMap := convertSwitchConfigToMap(switchConfig)

		issues := l.validateSyntax(configMap)
		result.addIssues(mac, switchConfig.Name, issues)

		issues = l.validateSchema(configMap, "switch")
		result.addIssues(mac, switchConfig.Name, issues)

		issues = l.validateVendorBlocks(configMap, targetVendor)
		result.addIssues(mac, switchConfig.Name, issues)

		issues = l.validateRanges(configMap, "switch")
		result.addIssues(mac, switchConfig.Name, issues)
	}

	// Lint gateway configurations
	for mac, gwConfig := range siteConfig.Devices.WanEdge {
		configMap := convertGatewayConfigToMap(gwConfig)

		issues := l.validateSyntax(configMap)
		result.addIssues(mac, gwConfig.Name, issues)

		issues = l.validateSchema(configMap, "gateway")
		result.addIssues(mac, gwConfig.Name, issues)

		issues = l.validateVendorBlocks(configMap, targetVendor)
		result.addIssues(mac, gwConfig.Name, issues)
	}

	return result, nil
}

// validateSyntax checks for basic syntax issues in the configuration.
func (l *ConfigLinter) validateSyntax(configMap map[string]any) []LintIssue {
	var issues []LintIssue

	// Check for nil values in required fields
	requiredFields := []string{"name"}
	for _, field := range requiredFields {
		if val, exists := configMap[field]; !exists || val == nil || val == "" {
			issues = append(issues, LintIssue{
				Field:      field,
				Message:    fmt.Sprintf("Required field '%s' is missing or empty", field),
				Suggestion: fmt.Sprintf("Add a value for '%s'", field),
			})
		}
	}

	return issues
}

// validateSchema checks if the configuration matches expected schema.
func (l *ConfigLinter) validateSchema(configMap map[string]any, deviceType string) []LintIssue {
	var issues []LintIssue

	// Define expected field types for each device type
	expectedTypes := getExpectedFieldTypes(deviceType)

	for field, expectedType := range expectedTypes {
		if val, exists := configMap[field]; exists && val != nil {
			actualType := getTypeString(val)
			if actualType != expectedType {
				issues = append(issues, LintIssue{
					Field:      field,
					Message:    fmt.Sprintf("Field '%s' has incorrect type: expected %s, got %s", field, expectedType, actualType),
					Suggestion: fmt.Sprintf("Convert '%s' to %s", field, expectedType),
				})
			}
		}
	}

	return issues
}

// validateVendorBlocks checks for vendor-specific configuration blocks.
func (l *ConfigLinter) validateVendorBlocks(configMap map[string]any, targetVendor string) []LintIssue {
	var issues []LintIssue

	// Check for Mist-specific fields
	mistFields := []string{"deviceprofile_id", "deviceprofile_name", "radio_config", "ip_config"}
	// Check for Meraki-specific fields
	merakiFields := []string{"tags", "address"}

	if targetVendor == "mist" {
		// Warn if Meraki-specific fields are present
		for _, field := range merakiFields {
			if _, exists := configMap[field]; exists {
				issues = append(issues, LintIssue{
					Field:      field,
					Message:    fmt.Sprintf("Field '%s' is Meraki-specific but target vendor is Mist", field),
					Suggestion: "Remove Meraki-specific fields or change target vendor",
				})
			}
		}
	} else if targetVendor == "meraki" {
		// Warn if Mist-specific fields are present
		for _, field := range mistFields {
			if _, exists := configMap[field]; exists {
				issues = append(issues, LintIssue{
					Field:      field,
					Message:    fmt.Sprintf("Field '%s' is Mist-specific but target vendor is Meraki", field),
					Suggestion: "Remove Mist-specific fields or change target vendor",
				})
			}
		}
	}

	return issues
}

// validateReferences checks if referenced resources (profiles, templates) exist.
func (l *ConfigLinter) validateReferences(configMap map[string]any) []LintIssue {
	var issues []LintIssue

	if l.cacheAccessor == nil {
		return issues
	}

	// Check device profile references
	if profileName, ok := configMap["deviceprofile_name"].(string); ok && profileName != "" {
		if profile, err := l.cacheAccessor.GetDeviceProfileByName(profileName); err != nil || profile == nil {
			issues = append(issues, LintIssue{
				Field:      "deviceprofile_name",
				Message:    fmt.Sprintf("Referenced device profile '%s' not found in cache", profileName),
				Suggestion: "Verify the profile name or refresh the cache",
			})
		}
	}

	if profileID, ok := configMap["deviceprofile_id"].(string); ok && profileID != "" {
		if profile, err := l.cacheAccessor.GetDeviceProfileByID(profileID); err != nil || profile == nil {
			issues = append(issues, LintIssue{
				Field:      "deviceprofile_id",
				Message:    fmt.Sprintf("Referenced device profile ID '%s' not found in cache", profileID),
				Suggestion: "Verify the profile ID or refresh the cache",
			})
		}
	}

	return issues
}

// validateRanges checks if numeric values are within acceptable ranges.
func (l *ConfigLinter) validateRanges(configMap map[string]any, deviceType string) []LintIssue {
	var issues []LintIssue

	// Define valid ranges for common fields
	ranges := map[string]struct {
		min int
		max int
	}{
		"vlan_id": {1, 4094},
	}

	// AP-specific ranges
	if deviceType == "ap" {
		ranges["tx_power"] = struct {
			min int
			max int
		}{1, 20}
		ranges["channel"] = struct {
			min int
			max int
		}{1, 165}
	}

	for field, validRange := range ranges {
		if val, exists := configMap[field]; exists {
			var intVal int
			switch v := val.(type) {
			case int:
				intVal = v
			case float64:
				intVal = int(v)
			default:
				continue
			}

			if intVal < validRange.min || intVal > validRange.max {
				issues = append(issues, LintIssue{
					Field:      field,
					Message:    fmt.Sprintf("Field '%s' value %d is out of valid range [%d-%d]", field, intVal, validRange.min, validRange.max),
					Suggestion: fmt.Sprintf("Set '%s' to a value between %d and %d", field, validRange.min, validRange.max),
				})
			}
		}
	}

	return issues
}

// addIssues adds issues to the result, categorizing them as warnings or errors.
func (r *LintResult) addIssues(mac, deviceName string, issues []LintIssue) {
	for _, issue := range issues {
		issue.DeviceMAC = mac
		issue.DeviceName = deviceName

		// Categorize: errors are for required fields, warnings for everything else
		if strings.Contains(issue.Message, "Required field") {
			r.Errors = append(r.Errors, issue)
		} else {
			r.Warnings = append(r.Warnings, issue)
		}
	}
}

// Helper functions

func getTargetVendor(siteConfig *config.SiteConfigObj) string {
	if siteConfig.API != "" {
		// Parse vendor from API label (format: "vendor-name" or just vendor)
		parts := strings.Split(siteConfig.API, "-")
		if len(parts) > 0 {
			return parts[0]
		}
	}
	return "mist" // default
}

func convertAPConfigToMap(apConfig config.APConfig) map[string]any {
	result := make(map[string]any)

	if apConfig.APDeviceConfig != nil {
		result["name"] = apConfig.APDeviceConfig.Name
		result["notes"] = apConfig.APDeviceConfig.Notes
		// Add other fields as needed
	}

	result["mac"] = apConfig.MAC
	result["magic"] = apConfig.Magic

	return result
}

func convertSwitchConfigToMap(switchConfig config.SwitchConfig) map[string]any {
	result := make(map[string]any)
	result["name"] = switchConfig.Name
	result["notes"] = switchConfig.Notes
	result["role"] = switchConfig.Role
	result["magic"] = switchConfig.Magic
	return result
}

func convertGatewayConfigToMap(gwConfig config.WanEdgeConfig) map[string]any {
	result := make(map[string]any)
	result["name"] = gwConfig.Name
	result["notes"] = gwConfig.Notes
	result["magic"] = gwConfig.Magic
	return result
}

func getExpectedFieldTypes(deviceType string) map[string]string {
	// Common fields across all device types
	types := map[string]string{
		"name":  "string",
		"notes": "string",
		"magic": "string",
	}

	if deviceType == "ap" {
		types["tx_power"] = "int"
		types["channel"] = "int"
		types["vlan_id"] = "int"
	}

	return types
}

func getTypeString(val any) string {
	if val == nil {
		return "null"
	}

	t := reflect.TypeOf(val)
	switch t.Kind() {
	case reflect.Bool:
		return "bool"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "int"
	case reflect.Float32, reflect.Float64:
		return "float"
	case reflect.String:
		return "string"
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.Map, reflect.Struct:
		return "object"
	default:
		return "unknown"
	}
}

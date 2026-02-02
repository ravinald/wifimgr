package validation

import (
	"fmt"
	"strings"
	"time"

	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// CompatibilityResult contains the results of checking API compatibility.
type CompatibilityResult struct {
	SiteName    string
	APIVersion  string
	LastChecked time.Time
	Issues      []CompatibilityIssue
	Compatible  bool
}

// CompatibilityIssue represents a single compatibility issue.
type CompatibilityIssue struct {
	Severity        string   // "warning", "error"
	Field           string   // field name that has the issue
	Message         string   // description of the issue
	AffectedDevices []string // list of device MACs affected
	Action          string   // suggested action to fix
}

// CompatibilityChecker validates that configurations are compatible with the target API.
type CompatibilityChecker struct {
	schemaTracker *vendors.SchemaTracker
	cacheAccessor *vendors.CacheAccessor
}

// NewCompatibilityChecker creates a new compatibility checker.
func NewCompatibilityChecker(tracker *vendors.SchemaTracker, accessor *vendors.CacheAccessor) *CompatibilityChecker {
	return &CompatibilityChecker{
		schemaTracker: tracker,
		cacheAccessor: accessor,
	}
}

// CheckSite performs compatibility validation on a site configuration.
func (c *CompatibilityChecker) CheckSite(siteName string, siteConfig *config.SiteConfigObj) (*CompatibilityResult, error) {
	result := &CompatibilityResult{
		SiteName:    siteName,
		APIVersion:  "unknown",
		LastChecked: time.Now(),
		Issues:      []CompatibilityIssue{},
		Compatible:  true,
	}

	targetVendor := getTargetVendor(siteConfig)

	// Check AP configurations
	apIssues := c.checkDeviceCompatibility(siteConfig.Devices.APs, targetVendor, "ap")
	result.Issues = append(result.Issues, apIssues...)

	// Check switch configurations
	switchIssues := c.checkSwitchCompatibility(siteConfig.Devices.Switches, targetVendor)
	result.Issues = append(result.Issues, switchIssues...)

	// Check gateway configurations
	gatewayIssues := c.checkGatewayCompatibility(siteConfig.Devices.WanEdge, targetVendor)
	result.Issues = append(result.Issues, gatewayIssues...)

	// Determine overall compatibility
	for _, issue := range result.Issues {
		if issue.Severity == "error" {
			result.Compatible = false
			break
		}
	}

	return result, nil
}

// checkDeviceCompatibility checks AP device configurations for compatibility.
func (c *CompatibilityChecker) checkDeviceCompatibility(devices map[string]config.APConfig, vendor, deviceType string) []CompatibilityIssue {
	var issues []CompatibilityIssue

	// Get schema snapshot for this vendor/device type
	var snapshot *vendors.SchemaSnapshot
	if c.schemaTracker != nil {
		snapshot = c.schemaTracker.GetSnapshot(vendor, deviceType)
	}

	for mac, device := range devices {
		// Check for deprecated fields
		deprecated := c.checkDeprecatedFields(device, vendor, deviceType)
		for _, issue := range deprecated {
			issue.AffectedDevices = []string{mac}
			issues = append(issues, issue)
		}

		// Check for schema mismatches if we have a snapshot
		if snapshot != nil {
			schemaIssues := c.checkSchemaCompatibility(device, snapshot, mac)
			issues = append(issues, schemaIssues...)
		}

		// Check for vendor-specific incompatibilities
		vendorIssues := c.checkVendorSpecificIssues(device, vendor, mac)
		issues = append(issues, vendorIssues...)
	}

	return issues
}

// checkSwitchCompatibility checks switch configurations for compatibility.
func (c *CompatibilityChecker) checkSwitchCompatibility(switches map[string]config.SwitchConfig, _ string) []CompatibilityIssue {
	var issues []CompatibilityIssue

	for mac := range switches {
		// Placeholder for switch-specific checks
		// In a full implementation, would check switch-specific fields
		_ = mac
	}

	return issues
}

// checkGatewayCompatibility checks gateway configurations for compatibility.
func (c *CompatibilityChecker) checkGatewayCompatibility(gateways map[string]config.WanEdgeConfig, _ string) []CompatibilityIssue {
	var issues []CompatibilityIssue

	for mac := range gateways {
		// Placeholder for gateway-specific checks
		_ = mac
	}

	return issues
}

// checkDeprecatedFields checks for use of deprecated configuration fields.
func (c *CompatibilityChecker) checkDeprecatedFields(device config.APConfig, _, _ string) []CompatibilityIssue {
	var issues []CompatibilityIssue

	// Check for legacy Config field (APHWConfig)
	if device.Config.LEDEnabled || device.Config.ScanningEnabled || device.Config.IndoorUse {
		issues = append(issues, CompatibilityIssue{
			Severity: "warning",
			Field:    "config",
			Message:  "Legacy 'config' field is deprecated",
			Action:   "Migrate to APDeviceConfig.RadioConfig structure",
		})
	}

	// Check for legacy VlanID field
	if device.VlanID != 0 {
		issues = append(issues, CompatibilityIssue{
			Severity: "warning",
			Field:    "vlan_id",
			Message:  "Top-level 'vlan_id' field is deprecated",
			Action:   "Use APDeviceConfig.IPConfig.VlanID instead",
		})
	}

	return issues
}

// checkSchemaCompatibility checks if device config matches expected schema.
func (c *CompatibilityChecker) checkSchemaCompatibility(device config.APConfig, snapshot *vendors.SchemaSnapshot, mac string) []CompatibilityIssue {
	var issues []CompatibilityIssue

	// Convert device to map for comparison
	configMap := convertAPConfigToMap(device)

	for field := range configMap {
		// Check if field exists in schema
		if _, exists := snapshot.Fields[field]; !exists {
			issues = append(issues, CompatibilityIssue{
				Severity:        "warning",
				Field:           field,
				Message:         fmt.Sprintf("Field '%s' not found in API schema", field),
				AffectedDevices: []string{mac},
				Action:          "Field may not be supported by the API - verify or remove",
			})
		}
	}

	return issues
}

// checkVendorSpecificIssues checks for vendor-specific compatibility issues.
func (c *CompatibilityChecker) checkVendorSpecificIssues(device config.APConfig, vendor, mac string) []CompatibilityIssue {
	var issues []CompatibilityIssue

	configMap := convertAPConfigToMap(device)

	// Mist-specific checks
	if vendor == "mist" {
		// Check for profile references
		if profileName, ok := configMap["deviceprofile_name"].(string); ok && profileName != "" {
			if c.cacheAccessor != nil {
				if profile, err := c.cacheAccessor.GetDeviceProfileByName(profileName); err != nil || profile == nil {
					issues = append(issues, CompatibilityIssue{
						Severity:        "error",
						Field:           "deviceprofile_name",
						Message:         fmt.Sprintf("Device profile '%s' not found", profileName),
						AffectedDevices: []string{mac},
						Action:          "Create the profile or use a different profile name",
					})
				}
			}
		}
	}

	// Meraki-specific checks
	if vendor == "meraki" {
		// Check for Mist-specific fields that won't work with Meraki
		mistOnlyFields := []string{"deviceprofile_id", "deviceprofile_name", "radio_config"}
		for _, field := range mistOnlyFields {
			if _, exists := configMap[field]; exists {
				issues = append(issues, CompatibilityIssue{
					Severity:        "error",
					Field:           field,
					Message:         fmt.Sprintf("Field '%s' is not supported by Meraki API", field),
					AffectedDevices: []string{mac},
					Action:          "Remove Mist-specific fields or switch to Mist API",
				})
			}
		}
	}

	return issues
}

// HasErrors returns true if there are any error-level issues.
func (r *CompatibilityResult) HasErrors() bool {
	for _, issue := range r.Issues {
		if issue.Severity == "error" {
			return true
		}
	}
	return false
}

// HasWarnings returns true if there are any warning-level issues.
func (r *CompatibilityResult) HasWarnings() bool {
	for _, issue := range r.Issues {
		if issue.Severity == "warning" {
			return true
		}
	}
	return false
}

// Summary returns a human-readable summary of the compatibility check.
func (r *CompatibilityResult) Summary() string {
	errorCount := 0
	warningCount := 0

	for _, issue := range r.Issues {
		if issue.Severity == "error" {
			errorCount++
		} else if issue.Severity == "warning" {
			warningCount++
		}
	}

	if r.Compatible {
		if warningCount > 0 {
			return fmt.Sprintf("Compatible with %d warning(s)", warningCount)
		}
		return "Fully compatible"
	}

	return fmt.Sprintf("Not compatible: %d error(s), %d warning(s)", errorCount, warningCount)
}

// GroupByField groups issues by field name for easier reporting.
func (r *CompatibilityResult) GroupByField() map[string][]CompatibilityIssue {
	groups := make(map[string][]CompatibilityIssue)

	for _, issue := range r.Issues {
		field := issue.Field
		if field == "" {
			field = "general"
		}
		groups[field] = append(groups[field], issue)
	}

	return groups
}

// FilterBySeverity returns only issues matching the given severity.
func (r *CompatibilityResult) FilterBySeverity(severity string) []CompatibilityIssue {
	var filtered []CompatibilityIssue
	severityLower := strings.ToLower(severity)

	for _, issue := range r.Issues {
		if strings.ToLower(issue.Severity) == severityLower {
			filtered = append(filtered, issue)
		}
	}

	return filtered
}

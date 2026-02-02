package vendors

import "fmt"

// SiteNotFoundError indicates a site name doesn't exist in the API.
type SiteNotFoundError struct {
	SiteName     string
	APILabel     string
	SearchedAPIs []string
}

func (e *SiteNotFoundError) Error() string {
	if e.APILabel != "" {
		return fmt.Sprintf("site %q not found in API %q", e.SiteName, e.APILabel)
	}
	return fmt.Sprintf("site %q not found", e.SiteName)
}

// UserMessage returns a user-friendly error message with remediation advice.
func (e *SiteNotFoundError) UserMessage() string {
	msg := fmt.Sprintf("Site %q not found", e.SiteName)
	if e.APILabel != "" {
		msg += fmt.Sprintf(" in API %q", e.APILabel)
	}
	if len(e.SearchedAPIs) > 0 {
		msg += fmt.Sprintf("\n\nSearched APIs: %v", e.SearchedAPIs)
	}
	msg += "\n\nTry:\n  - Refresh the cache: wifimgr refresh cache\n  - Check site name spelling\n  - Use --api to target a specific API"
	return msg
}

// DuplicateSiteError indicates multiple sites with the same name.
// This can occur within one API or across multiple APIs.
type DuplicateSiteError struct {
	SiteName   string
	APILabel   string   // Set if duplicates are within one API
	APIs       []string // Set if duplicates are across multiple APIs
	MatchCount int
}

func (e *DuplicateSiteError) Error() string {
	if len(e.APIs) > 0 {
		return fmt.Sprintf("site %q exists in multiple APIs: %v", e.SiteName, e.APIs)
	}
	return fmt.Sprintf("site %q has %d matches in API %q - duplicate site names not supported",
		e.SiteName, e.MatchCount, e.APILabel)
}

// UserMessage returns a user-friendly error message with remediation advice.
func (e *DuplicateSiteError) UserMessage() string {
	if len(e.APIs) > 0 {
		return fmt.Sprintf(`Site "%s" exists in multiple APIs: %v

To resolve this:
  1. Use --api <label> to specify which API to use
  2. Add 'api' field to the site config to set a default
  3. Rename one of the sites to avoid conflicts`,
			e.SiteName, e.APIs)
	}
	return fmt.Sprintf(`Site "%s" has %d matches in API "%s" - skipping site

Duplicate site names within a vendor are not supported.
Please rename one of the sites in the vendor dashboard.`,
		e.SiteName, e.MatchCount, e.APILabel)
}

// APINotFoundError indicates an API label doesn't exist in the registry.
type APINotFoundError struct {
	APILabel      string
	AvailableAPIs []string
}

func (e *APINotFoundError) Error() string {
	return fmt.Sprintf("API %q not found", e.APILabel)
}

// UserMessage returns a user-friendly error message listing available APIs.
func (e *APINotFoundError) UserMessage() string {
	if len(e.AvailableAPIs) == 0 {
		return fmt.Sprintf("API %q not found - no APIs configured", e.APILabel)
	}
	return fmt.Sprintf("API %q not found\n\nAvailable APIs: %v",
		e.APILabel, e.AvailableAPIs)
}

// CapabilityNotSupportedError indicates a vendor doesn't support a requested capability.
type CapabilityNotSupportedError struct {
	Capability  string
	APILabel    string
	VendorName  string
	SupportedBy []string
}

func (e *CapabilityNotSupportedError) Error() string {
	return fmt.Sprintf("%s not supported by %s (%s vendor)",
		e.Capability, e.APILabel, e.VendorName)
}

// UserMessage returns a user-friendly error message.
func (e *CapabilityNotSupportedError) UserMessage() string {
	msg := fmt.Sprintf("%s not supported by %s (%s vendor)",
		e.Capability, e.APILabel, e.VendorName)
	if len(e.SupportedBy) > 0 {
		msg += fmt.Sprintf("\n\nThis feature is only available for: %v", e.SupportedBy)
	}
	return msg
}

// MACCollisionError indicates the same MAC address exists in multiple APIs.
// This should not happen in practice but is detected during cache building.
type MACCollisionError struct {
	MAC  string
	APIs []string
}

func (e *MACCollisionError) Error() string {
	return fmt.Sprintf("MAC %s exists in multiple APIs: %v", e.MAC, e.APIs)
}

// DeviceNotFoundError indicates a device was not found.
type DeviceNotFoundError struct {
	Identifier string // MAC, serial, or name
	APILabel   string
}

func (e *DeviceNotFoundError) Error() string {
	if e.APILabel != "" {
		return fmt.Sprintf("device %q not found in API %q", e.Identifier, e.APILabel)
	}
	return fmt.Sprintf("device %q not found", e.Identifier)
}

// InvalidAPIConfigError indicates an API configuration is invalid.
type InvalidAPIConfigError struct {
	APILabel string
	Reason   string
}

func (e *InvalidAPIConfigError) Error() string {
	return fmt.Sprintf("invalid API configuration for %q: %s", e.APILabel, e.Reason)
}

// FieldMappingError represents a field transformation error during vendor API conversions.
type FieldMappingError struct {
	Vendor       string
	DeviceMAC    string
	DeviceName   string
	Field        string
	ExpectedType string
	ActualType   string
	ActualValue  any
}

func (e *FieldMappingError) Error() string {
	return fmt.Sprintf(
		"%s API field mapping error for device %s: field %q expected %s but got %s (value: %v)",
		e.Vendor, e.DeviceMAC, e.Field, e.ExpectedType, e.ActualType, e.ActualValue,
	)
}

// UserMessage returns a user-friendly error message with remediation advice.
func (e *FieldMappingError) UserMessage() string {
	vendorName := e.Vendor
	if vendorName == "" {
		vendorName = "API"
	}

	deviceID := e.DeviceMAC
	if e.DeviceName != "" {
		deviceID = fmt.Sprintf("%s (%s)", e.DeviceMAC, e.DeviceName)
	}

	return fmt.Sprintf(`Field Type Mismatch for AP %s

  Field: %s
  Expected: %s (e.g., %s)
  Received: %v (%s)

Suggested Actions:
  1. Check if the %s API has changed
  2. Verify your configuration uses correct field types
  3. Run: wifimgr refresh cache`,
		deviceID,
		e.Field,
		e.ExpectedType,
		exampleForType(e.ExpectedType),
		e.ActualValue,
		e.ActualType,
		vendorName,
	)
}

// UnexpectedFieldWarning represents a new field from vendor API that isn't in the converter's known field set.
type UnexpectedFieldWarning struct {
	Vendor     string
	DeviceMAC  string
	DeviceName string
	Field      string
	Value      any
}

func (e *UnexpectedFieldWarning) Error() string {
	return fmt.Sprintf(
		"%s API returned unexpected field for device %s: %q (value: %v) - this may be a new API feature",
		e.Vendor, e.DeviceMAC, e.Field, e.Value,
	)
}

// UserMessage returns a user-friendly warning message with remediation advice.
func (e *UnexpectedFieldWarning) UserMessage() string {
	vendorName := e.Vendor
	if vendorName == "" {
		vendorName = "API"
	}

	deviceID := e.DeviceMAC
	if e.DeviceName != "" {
		deviceID = fmt.Sprintf("%s (%s)", e.DeviceMAC, e.DeviceName)
	}

	return fmt.Sprintf(`Unexpected Field from %s API for AP %s

  Field: %s
  Value: %v

This is a warning, not an error. The %s API returned a field that wifimgr
doesn't recognize. This may be a new API feature.

Suggested Actions:
  1. Check if wifimgr needs updating to support new API features
  2. If this field is critical, please report it as a feature request`,
		vendorName,
		deviceID,
		e.Field,
		e.Value,
		vendorName,
	)
}

// MissingFieldWarning represents an expected field that wasn't in the API response.
type MissingFieldWarning struct {
	Vendor     string
	DeviceMAC  string
	DeviceName string
	Field      string
}

func (e *MissingFieldWarning) Error() string {
	return fmt.Sprintf(
		"%s API missing expected field for device %s: %q - field may have been removed from API",
		e.Vendor, e.DeviceMAC, e.Field,
	)
}

// UserMessage returns a user-friendly warning message with remediation advice.
func (e *MissingFieldWarning) UserMessage() string {
	vendorName := e.Vendor
	if vendorName == "" {
		vendorName = "API"
	}

	deviceID := e.DeviceMAC
	if e.DeviceName != "" {
		deviceID = fmt.Sprintf("%s (%s)", e.DeviceMAC, e.DeviceName)
	}

	return fmt.Sprintf(`Missing Expected Field from %s API for AP %s

  Field: %s

This is a warning, not an error. The %s API didn't include a field that
wifimgr expected. This may indicate the API has changed.

Suggested Actions:
  1. Check if the %s API has deprecated this field
  2. Run: wifimgr refresh cache
  3. If this causes issues, please report it`,
		vendorName,
		deviceID,
		e.Field,
		vendorName,
		vendorName,
	)
}

// exampleForType provides example values for type names in error messages.
func exampleForType(typeName string) string {
	switch typeName {
	case "integer", "int":
		return "15"
	case "float", "float64", "number":
		return "3.14"
	case "string":
		return `"text"`
	case "boolean", "bool":
		return "true"
	case "array", "slice":
		return "[1, 2, 3]"
	case "object", "map":
		return `{"key": "value"}`
	default:
		return typeName
	}
}

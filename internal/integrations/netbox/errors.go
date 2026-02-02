package netbox

import (
	"fmt"
	"strings"
)

// InterfaceTypeError is returned when an invalid interface type is used
type InterfaceTypeError struct {
	DeviceName  string   // Device name (if available)
	InvalidType string   // The invalid interface type that was specified
	ValidTypes  []string // List of valid interface types
	Suggestion  string   // Suggested correction (if available)
}

// Error implements the error interface
func (e *InterfaceTypeError) Error() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("interface type '%s' is not valid", e.InvalidType))
	if e.DeviceName != "" {
		b.WriteString(fmt.Sprintf(" for device '%s'", e.DeviceName))
	}

	b.WriteString("\n\nCommon valid types:\n")
	for _, t := range e.ValidTypes {
		label := ValidInterfaceTypes[t]
		if label != "" {
			b.WriteString(fmt.Sprintf("  - %s (%s)\n", t, label))
		} else {
			b.WriteString(fmt.Sprintf("  - %s\n", t))
		}
	}

	if e.Suggestion != "" {
		b.WriteString(fmt.Sprintf("\nSuggestion: use '%s' instead\n", e.Suggestion))
	}

	b.WriteString("\nConfigure in netbox.mappings.interfaces in wifimgr config file")
	return b.String()
}

// DeviceExistsError is returned when trying to create a device that already exists
type DeviceExistsError struct {
	DeviceName string
	MAC        string
	NetBoxID   int64
}

func (e *DeviceExistsError) Error() string {
	return fmt.Sprintf("device '%s' (MAC: %s) already exists in NetBox with ID %d",
		e.DeviceName, e.MAC, e.NetBoxID)
}

// MissingDependencyError is returned when a required NetBox object is missing
type MissingDependencyError struct {
	DependencyType string // "site", "device_type", "device_role"
	Name           string // The name/slug that was not found
	Suggestion     string // Suggested fix
}

func (e *MissingDependencyError) Error() string {
	msg := fmt.Sprintf("%s '%s' not found in NetBox", e.DependencyType, e.Name)
	if e.Suggestion != "" {
		msg += fmt.Sprintf("\n\n%s", e.Suggestion)
	}
	return msg
}

// InterfaceCreateError is returned when interface creation fails
type InterfaceCreateError struct {
	DeviceName    string
	InterfaceName string
	InterfaceType string
	Cause         error
}

func (e *InterfaceCreateError) Error() string {
	return fmt.Sprintf("failed to create interface '%s' (type: %s) on device '%s': %v",
		e.InterfaceName, e.InterfaceType, e.DeviceName, e.Cause)
}

func (e *InterfaceCreateError) Unwrap() error {
	return e.Cause
}

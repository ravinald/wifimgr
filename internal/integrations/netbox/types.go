package netbox

import "fmt"

// Site represents a NetBox site
type Site struct {
	ID     int64
	Name   string
	Slug   string
	Status string
}

// DeviceType represents a NetBox device type
type DeviceType struct {
	ID           int64
	Manufacturer string
	Model        string
	Slug         string
}

// DeviceRole represents a NetBox device role
type DeviceRole struct {
	ID   int64
	Name string
	Slug string
}

// Device represents a NetBox device
type Device struct {
	ID           int64
	Name         string
	DeviceType   *DeviceType
	Role         *DeviceRole
	Site         *Site
	Serial       string
	Status       string
	PrimaryIP    string
	PrimaryIPv4  string
	PrimaryIPv6  string
	Comments     string
	CustomFields map[string]any
}

// Interface represents a NetBox device interface
type Interface struct {
	ID           int64
	Device       int64
	Name         string
	Type         string
	MACAddr      string
	Enabled      bool
	MTU          int
	Mode         string
	TaggedVLANs  []int64
	UntaggedVLAN int64
	RFRole       string  // "ap" for access point radios
	Parent       *int64  // Parent interface ID for virtual interfaces
	WirelessLANs []int64 // Linked WirelessLAN IDs
}

// InterfaceTemplate represents a NetBox interface template defined on a device type
type InterfaceTemplate struct {
	ID   int64
	Name string
	Type string
}

// InterfaceUpdateRequest represents a request to update an existing interface
type InterfaceUpdateRequest struct {
	Type    string `json:"type,omitempty"`
	Enabled *bool  `json:"enabled,omitempty"`
}

// WirelessLAN represents a NetBox WirelessLAN object
type WirelessLAN struct {
	ID         int64  `json:"id"`
	SSID       string `json:"ssid"`
	Status     string `json:"status"`      // "active", "disabled"
	AuthType   string `json:"auth_type"`   // "open", "wpa-personal", "wpa-enterprise"
	AuthCipher string `json:"auth_cipher"` // "auto", "tkip", "aes"
	VLAN       *int64 `json:"vlan,omitempty"`
	Tags       []Tag  `json:"tags,omitempty"`
}

// IPAddress represents a NetBox IP address
type IPAddress struct {
	ID                 int64
	Address            string
	Status             string
	AssignedObjectType string
	AssignedObjectID   int64
	DNSName            string
}

// Tag represents a NetBox tag
type Tag struct {
	Name string `json:"name"`
}

// DeviceRequest represents a request to create/update a NetBox device
type DeviceRequest struct {
	Name         string         `json:"name"`
	DeviceType   int64          `json:"device_type"`
	Role         int64          `json:"role"`
	Site         int64          `json:"site"`
	Serial       string         `json:"serial,omitempty"`
	Status       string         `json:"status,omitempty"`
	Comments     string         `json:"comments,omitempty"`
	Tags         []Tag          `json:"tags,omitempty"`
	CustomFields map[string]any `json:"custom_fields,omitempty"`
}

// InterfaceRequest represents a request to create/update a NetBox interface
type InterfaceRequest struct {
	Device       int64   `json:"device"`
	Name         string  `json:"name"`
	Type         string  `json:"type"`
	MACAddr      string  `json:"mac_address,omitempty"`
	Enabled      bool    `json:"enabled"`
	Tags         []Tag   `json:"tags,omitempty"`
	RFRole       string  `json:"rf_role,omitempty"`       // "ap" for radio interfaces
	RFChannel    *int    `json:"rf_channel,omitempty"`    // RF channel number
	WirelessLANs []int64 `json:"wireless_lans,omitempty"` // Link to WirelessLAN IDs
	Parent       *int64  `json:"parent,omitempty"`        // Parent interface ID for virtual interfaces
	Description  string  `json:"description,omitempty"`   // Interface description
}

// WirelessLANRequest represents a request to create/update a NetBox WirelessLAN
type WirelessLANRequest struct {
	SSID       string `json:"ssid"`
	Status     string `json:"status,omitempty"`      // "active", "disabled"
	AuthType   string `json:"auth_type,omitempty"`   // "open", "wpa-personal", "wpa-enterprise"
	AuthCipher string `json:"auth_cipher,omitempty"` // "auto", "tkip", "aes"
	VLAN       *int64 `json:"vlan,omitempty"`
	Tags       []Tag  `json:"tags,omitempty"`
}

// IPAddressRequest represents a request to create/update a NetBox IP address
type IPAddressRequest struct {
	Address            string `json:"address"`
	Status             string `json:"status,omitempty"`
	AssignedObjectType string `json:"assigned_object_type,omitempty"`
	AssignedObjectID   int64  `json:"assigned_object_id,omitempty"`
	DNSName            string `json:"dns_name,omitempty"`
	Tags               []Tag  `json:"tags,omitempty"`
}

// ExportResult contains the results of an export operation
type ExportResult struct {
	Created []DeviceExportResult
	Updated []DeviceExportResult
	Skipped []SkippedDevice
	Errors  []ExportError
	Stats   ExportStats
}

// DeviceExportResult represents a successfully exported device
type DeviceExportResult struct {
	Name      string
	MAC       string
	NetBoxID  int64
	Operation string // "created" or "updated"
}

// SkippedDevice represents a device that was skipped during export
type SkippedDevice struct {
	Name   string
	MAC    string
	Reason string
}

// ExportError represents a non-fatal error during export
type ExportError struct {
	DeviceName string
	DeviceMAC  string
	Operation  string // "validate", "create", "update", "interface", "ip"
	Message    string
	Err        error
}

func (e ExportError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s (%s): %s - %v", e.DeviceName, e.DeviceMAC, e.Message, e.Err)
	}
	return fmt.Sprintf("%s (%s): %s", e.DeviceName, e.DeviceMAC, e.Message)
}

// ExportStats contains statistics about the export operation
type ExportStats struct {
	TotalDevices int
	Created      int
	Updated      int
	Skipped      int
	Errors       int
	Duration     string
}

// ExportOptions configures the export operation
type ExportOptions struct {
	SiteName      string // Export specific site, empty for all
	DryRun        bool   // Just validate, don't write
	Force         bool   // Skip confirmation prompt
	IncludeRadios bool   // Create radio interfaces (wifi0/1/2) and WLAN virtual interfaces
}

// ValidationError represents a validation error for a device
type ValidationError struct {
	Field   string
	Message string
}

func (v ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", v.Field, v.Message)
}

package vendors

import "time"

// SiteInfo represents a site/network in a vendor-agnostic way.
// In Mist this maps to a Site, in Meraki this maps to a Network.
type SiteInfo struct {
	// ID is the vendor-specific identifier (UUID for Mist, L_XXXXX for Meraki)
	ID string `json:"id"`

	// Name is the human-readable site name
	Name string `json:"name"`

	// Timezone is the site's timezone (e.g., "America/Los_Angeles")
	Timezone string `json:"timezone,omitempty"`

	Address string `json:"address,omitempty"`

	// CountryCode is the ISO country code (e.g., "US")
	CountryCode string `json:"country_code,omitempty"`

	// Notes contains any additional notes about the site
	Notes string `json:"notes,omitempty"`

	Latitude    float64 `json:"latitude,omitempty"`
	Longitude   float64 `json:"longitude,omitempty"`
	DeviceCount int     `json:"device_count,omitempty"`

	// Provenance tracks where this data came from (set by loader, not serialized)
	SourceAPI    string `json:"-"`
	SourceVendor string `json:"-"`
}

// NetBoxInterfaceMapping defines a device-level interface name and type override.
type NetBoxInterfaceMapping struct {
	Name string `json:"name,omitempty"` // NetBox interface name (e.g., "eth0", "mgmt0")
	Type string `json:"type,omitempty"` // NetBox PHY type (e.g., "1000base-t", "ieee802.11ax")
}

// NetBoxDeviceExtension contains device-level NetBox integration settings.
// These settings override global config for specific devices.
type NetBoxDeviceExtension struct {
	// DeviceRole overrides the default device role for this device
	DeviceRole string `json:"device_role,omitempty"`

	// Interfaces contains per-interface name and type overrides.
	// Keys are internal interface IDs: "eth0", "eth1", "radio0", "radio1", "radio2"
	Interfaces map[string]*NetBoxInterfaceMapping `json:"interfaces,omitempty"`
}

// InventoryItem represents a device in the organization's inventory.
// This is a device that has been claimed but may or may not be assigned to a site.
type InventoryItem struct {
	// ID is the vendor-specific device identifier
	// For Mist: UUID, for Meraki: serial number
	ID string `json:"id,omitempty"`

	// MAC is the device MAC address (normalized to lowercase, no separators)
	MAC string `json:"mac"`

	Serial string `json:"serial"`

	// Model is the device model (e.g., "AP43", "MR46")
	Model string `json:"model"`

	// Name is the device name (may be empty if not configured)
	Name string `json:"name,omitempty"`

	// Type is the normalized device type: "ap", "switch", "gateway"
	Type string `json:"type"`

	// SiteID is the vendor-specific site/network ID (empty if unassigned)
	SiteID string `json:"site_id,omitempty"`

	// SiteName is the human-readable site name (empty if unassigned)
	SiteName string `json:"site_name,omitempty"`

	// Claimed indicates whether the device has been claimed to the organization
	Claimed bool `json:"claimed"`

	// NetBox contains device-level NetBox integration settings
	NetBox *NetBoxDeviceExtension `json:"netbox,omitempty"`

	// Provenance tracks where this data came from (set by loader, not serialized)
	SourceAPI    string `json:"-"`
	SourceVendor string `json:"-"`
}

// DeviceInfo represents a configured device at a site.
// This extends InventoryItem with configuration and status information.
type DeviceInfo struct {
	// ID is the vendor-specific device identifier
	ID string `json:"id"`

	// MAC is the device MAC address (normalized to lowercase, no separators)
	MAC string `json:"mac"`

	Serial string `json:"serial,omitempty"`

	// Name is the configured device name
	Name string `json:"name"`

	// Model is the device model
	Model string `json:"model"`

	// Type is the normalized device type: "ap", "switch", "gateway"
	Type string `json:"type"`

	// SiteID is the vendor-specific site/network ID
	SiteID string `json:"site_id"`

	// SiteName is the human-readable site name
	SiteName string `json:"site_name,omitempty"`

	// Status is the device status: "connected", "disconnected"
	Status string `json:"status"`

	IP        string  `json:"ip,omitempty"`
	Version   string  `json:"version,omitempty"` // firmware/software version
	Notes     string  `json:"notes,omitempty"`
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`

	// DeviceProfileID is the assigned device profile (Mist-specific)
	DeviceProfileID string `json:"deviceprofile_id,omitempty"`

	// DeviceProfileName is the name of the assigned device profile
	DeviceProfileName string `json:"deviceprofile_name,omitempty"`

	// Provenance tracks where this data came from (set by loader, not serialized)
	SourceAPI    string `json:"-"`
	SourceVendor string `json:"-"`
}

// DeviceProfile represents a device configuration template.
// This is primarily a Mist concept.
type DeviceProfile struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"` // "ap", "switch", or "gateway"
	OrgID   string `json:"org_id,omitempty"`
	ForSite bool   `json:"for_site,omitempty"` // true if site-level profile
	SiteID  string `json:"site_id,omitempty"`

	// Provenance tracks where this data came from (set by loader, not serialized)
	SourceAPI    string `json:"-"`
	SourceVendor string `json:"-"`
}

// WiredSearchResults contains results from a wired client search.
type WiredSearchResults struct {
	Results []*WiredClient `json:"results"`
	Total   int            `json:"total"` // may exceed len(Results) if paginated
}

// WiredClient represents a wired client device connected to the network.
type WiredClient struct {
	MAC          string `json:"mac"`
	IP           string `json:"ip,omitempty"`
	Hostname     string `json:"hostname,omitempty"`
	SiteID       string `json:"site_id"`
	SiteName     string `json:"site_name,omitempty"`
	SwitchMAC    string `json:"switch_mac,omitempty"`
	SwitchName   string `json:"switch_name,omitempty"`
	PortID       string `json:"port_id,omitempty"`
	VLAN         int    `json:"vlan,omitempty"`
	Manufacturer string `json:"manufacturer,omitempty"` // from OUI lookup

	// Provenance tracks where this data came from (set by loader, not serialized)
	SourceAPI    string `json:"-"`
	SourceVendor string `json:"-"`
}

// WirelessSearchResults contains results from a wireless client search.
type WirelessSearchResults struct {
	Results []*WirelessClient `json:"results"`
	Total   int               `json:"total"` // may exceed len(Results) if paginated
}

// WirelessClient represents a wireless client device connected to the network.
type WirelessClient struct {
	MAC          string `json:"mac"`
	IP           string `json:"ip,omitempty"`
	Hostname     string `json:"hostname,omitempty"`
	SiteID       string `json:"site_id"`
	SiteName     string `json:"site_name,omitempty"`
	APMAC        string `json:"ap_mac,omitempty"`
	APName       string `json:"ap_name,omitempty"`
	SSID         string `json:"ssid,omitempty"`
	VLAN         int    `json:"vlan,omitempty"`
	Band         string `json:"band,omitempty"`         // "2.4", "5", or "6"
	Manufacturer string `json:"manufacturer,omitempty"` // from OUI lookup
	OS           string `json:"os,omitempty"`

	// Provenance tracks where this data came from (set by loader, not serialized)
	SourceAPI    string `json:"-"`
	SourceVendor string `json:"-"`
}

// DeviceStatus represents the current status of a device.
// This is stored separately from inventory to allow independent refresh.
type DeviceStatus struct {
	// Status is the normalized device status: "online", "offline", "alerting", "dormant"
	Status string `json:"status"`

	// LastReportedAt is the last time the device reported to the cloud
	LastReportedAt time.Time `json:"last_reported_at,omitempty"`

	// IP is the device's LAN IP address
	IP string `json:"ip,omitempty"`

	// PublicIP is the device's public IP address (Meraki only)
	PublicIP string `json:"public_ip,omitempty"`
}

// RFTemplate represents an RF configuration template.
// For Mist: org-level templates that can be applied to sites.
// For Meraki: per-network (site) RF profiles.
type RFTemplate struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	OrgID        string                 `json:"org_id,omitempty"`
	SiteID       string                 `json:"site_id,omitempty"` // Meraki only - RF profiles are per-network
	Config       map[string]interface{} `json:"config,omitempty"`  // full RF profile configuration
	SourceAPI    string                 `json:"-"`
	SourceVendor string                 `json:"-"`
}

// GatewayTemplate represents a gateway configuration template (Mist-specific).
type GatewayTemplate struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	OrgID        string `json:"org_id,omitempty"`
	SourceAPI    string `json:"-"`
	SourceVendor string `json:"-"`
}

// WLANTemplate represents a WLAN configuration template (Mist-specific).
type WLANTemplate struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	OrgID        string `json:"org_id,omitempty"`
	SourceAPI    string `json:"-"`
	SourceVendor string `json:"-"`
}

// WLAN represents a wireless LAN (SSID) configuration.
// For Mist: This can be an org-level WLAN or a site-level WLAN.
// For Meraki: This is a per-network SSID (numbered 0-14).
type WLAN struct {
	ID             string                 `json:"id"`   // UUID (Mist) or SSID number 0-14 (Meraki)
	SSID           string                 `json:"ssid"` // SSID name visible to users
	OrgID          string                 `json:"org_id,omitempty"`
	SiteID         string                 `json:"site_id,omitempty"` // empty for org-level WLANs (Mist only)
	Enabled        bool                   `json:"enabled"`
	Hidden         bool                   `json:"hidden,omitempty"` // true if SSID not broadcast
	Band           string                 `json:"band,omitempty"`   // "2.4", "5", "6", "dual", or "all"
	VLANID         int                    `json:"vlan_id,omitempty"`
	AuthType       string                 `json:"auth_type,omitempty"`       // "open", "psk", "wpa2-enterprise", etc.
	EncryptionMode string                 `json:"encryption_mode,omitempty"` // "wpa2", "wpa3", "wpa2/wpa3", etc.
	PSK            string                 `json:"psk,omitempty"`             // masked in cache for security
	RadiusServers  []RadiusServer         `json:"radius_servers,omitempty"`
	Config         map[string]interface{} `json:"config,omitempty"` // full vendor config for round-trip accuracy
	SourceAPI      string                 `json:"-"`
	SourceVendor   string                 `json:"-"`
}

// RadiusServer represents a RADIUS server configuration for 802.1X authentication.
type RadiusServer struct {
	Host   string `json:"host"`
	Port   int    `json:"port,omitempty"`
	Secret string `json:"secret,omitempty"` // Masked in cache
}

// APConfig represents the full configuration for an access point.
type APConfig struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	MAC          string                 `json:"mac"`
	SiteID       string                 `json:"site_id"`
	Config       map[string]interface{} `json:"config"` // full vendor-specific config
	SourceAPI    string                 `json:"-"`
	SourceVendor string                 `json:"-"`
}

// SwitchConfig represents the full configuration for a switch.
type SwitchConfig struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	MAC          string                 `json:"mac"`
	SiteID       string                 `json:"site_id"`
	Config       map[string]interface{} `json:"config"` // full vendor-specific config
	SourceAPI    string                 `json:"-"`
	SourceVendor string                 `json:"-"`
}

// GatewayConfig represents the full configuration for a gateway/appliance.
type GatewayConfig struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	MAC          string                 `json:"mac"`
	SiteID       string                 `json:"site_id"`
	Config       map[string]interface{} `json:"config"` // full vendor-specific config
	SourceAPI    string                 `json:"-"`
	SourceVendor string                 `json:"-"`
}

// SearchOptions contains options for search operations.
type SearchOptions struct {
	SiteID string // empty for org-wide search
}

// SearchCostEstimate provides information about the cost of a search operation.
// This allows the command layer to warn users before expensive operations.
type SearchCostEstimate struct {
	APICalls          int           // number of API calls required
	EstimatedDuration time.Duration // rough estimate
	NeedsConfirmation bool          // true if exceeds cost threshold
	Description       string        // human-readable explanation
}

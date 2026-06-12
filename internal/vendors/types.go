package vendors

import "time"

// Apply states recorded on a cached object after a push.
const (
	ApplyStateVerified           = "verified"            // pushed 2xx and read-back matched intent
	ApplyStateAppliedUnvalidated = "applied_unvalidated" // pushed 2xx, no read-back (trust mode)
	ApplyStateDivergent          = "divergent"           // pushed 2xx but read-back != intent
)

// ObjectMeta carries per-object cache freshness and apply state. It is embedded in
// every cached object so freshness is tracked uniformly: RefreshedAt is when the
// object was last fetched from the vendor (ground truth), while AppliedAt and
// ApplyState are set only for device configs that apply has pushed. The whole-API
// timestamp (APICacheMeta.LastRefresh) is the baseline; these are per-object deltas.
type ObjectMeta struct {
	RefreshedAt time.Time `json:"refreshed_at,omitzero"` // omitzero: omitempty does not detect a zero time.Time
	AppliedAt   time.Time `json:"applied_at,omitzero"`
	ApplyState  string    `json:"apply_state,omitempty"`
}

// SiteInfo represents a site/network in a vendor-agnostic way.
// In Mist this maps to a Site, in Meraki this maps to a Network.
type SiteInfo struct {
	ObjectMeta // per-object cache freshness + apply state
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

	// BoundToConfigTemplate marks a site whose devices a vendor manages through a
	// bound configuration template (a Meraki network bound to a config template).
	// wifimgr pushes config direct to devices, so a bound site signals that an
	// import can't fully own those devices — import warns rather than silently
	// producing config a template will overwrite. ConfigTemplateName is best-effort
	// (the org-networks list exposes the binding as a flag, not the template name).
	BoundToConfigTemplate bool   `json:"bound_to_config_template,omitempty"`
	ConfigTemplateName    string `json:"config_template_name,omitempty"`

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
	ObjectMeta // per-object cache freshness + apply state
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
	ObjectMeta        // per-object cache freshness + apply state
	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"` // "ap", "switch", or "gateway"
	OrgID      string `json:"org_id,omitempty"`
	ForSite    bool   `json:"for_site,omitempty"` // true if site-level profile
	SiteID     string `json:"site_id,omitempty"`

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
	MAC          string    `json:"mac"`
	IP           string    `json:"ip,omitempty"`
	Hostname     string    `json:"hostname,omitempty"`
	SiteID       string    `json:"site_id"`
	SiteName     string    `json:"site_name,omitempty"`
	SwitchMAC    string    `json:"switch_mac,omitempty"`
	SwitchName   string    `json:"switch_name,omitempty"`
	PortID       string    `json:"port_id,omitempty"`
	VLAN         int       `json:"vlan,omitempty"`
	Manufacturer string    `json:"manufacturer,omitempty"` // from OUI lookup
	FirstSeen    time.Time `json:"first_seen,omitzero"`    // first time client was seen on the network (vendor-supplied)
	LastSeen     time.Time `json:"last_seen,omitzero"`     // most recent sighting (vendor-supplied)

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
	MAC          string    `json:"mac"`
	IP           string    `json:"ip,omitempty"`
	Hostname     string    `json:"hostname,omitempty"`
	SiteID       string    `json:"site_id"`
	SiteName     string    `json:"site_name,omitempty"`
	APMAC        string    `json:"ap_mac,omitempty"`
	APName       string    `json:"ap_name,omitempty"`
	SSID         string    `json:"ssid,omitempty"`
	VLAN         int       `json:"vlan,omitempty"`
	Band         string    `json:"band,omitempty"`         // "2.4", "5", or "6"
	Status       string    `json:"status,omitempty"`       // vendor-supplied state, e.g. "Online" / "Offline"
	Manufacturer string    `json:"manufacturer,omitempty"` // from OUI lookup
	OS           string    `json:"os,omitempty"`
	FirstSeen    time.Time `json:"first_seen,omitzero"` // first time client was seen on the network (vendor-supplied)
	LastSeen     time.Time `json:"last_seen,omitzero"`  // most recent sighting (vendor-supplied)

	// Provenance tracks where this data came from (set by loader, not serialized)
	SourceAPI    string `json:"-"`
	SourceVendor string `json:"-"`
}

// ClientDetail is a cached per-client supplement populated by
// `refresh client site <name>`. It exists because some vendor APIs (notably
// Meraki) don't expose per-client connected band on their primary client
// list endpoint, so we maintain a persistent cache fed by the operator on
// demand. Keyed by normalized MAC.
//
// Connection status (online/offline) is deliberately NOT cached here —
// Meraki's primary search response already carries it live, so re-using
// cached status would invite staleness bugs.
type ClientDetail struct {
	ObjectMeta           // per-object cache freshness + apply state
	MAC        string    `json:"mac"`
	SiteID     string    `json:"site_id,omitempty"`
	Band       string    `json:"band,omitempty"` // "2.4" / "5" / "6"
	FetchedAt  time.Time `json:"fetched_at"`
}

// DeviceStatus represents the current status of a device.
// This is stored separately from inventory to allow independent refresh.
type DeviceStatus struct {
	ObjectMeta // per-object cache freshness + apply state
	// Status is the normalized device status: "online", "offline", "alerting", "dormant"
	Status string `json:"status"`

	// LastReportedAt is the last time the device reported to the cloud
	LastReportedAt time.Time `json:"last_reported_at,omitempty"`

	// IP is the device's LAN IP address
	IP string `json:"ip,omitempty"`

	// PublicIP is the device's public IP address (Meraki only)
	PublicIP string `json:"public_ip,omitempty"`
}

// BSSIDEntry represents a single BSSID and its associated AP, SSID, and radio details.
type BSSIDEntry struct {
	ObjectMeta            // per-object cache freshness + apply state
	BSSID          string `json:"bssid"` // normalized MAC (no separators)
	APName         string `json:"ap_name"`
	APSerial       string `json:"ap_serial"`
	APMAC          string `json:"ap_mac"` // normalized, for DeviceStatus lookup
	SiteID         string `json:"site_id"`
	SiteName       string `json:"site_name"`
	SSIDName       string `json:"ssid_name"`
	SSIDNumber     int    `json:"ssid_number"`
	Band           string `json:"band"` // "2.4", "5", "6"
	Channel        int    `json:"channel"`
	ChannelWidth   int    `json:"channel_width"`
	Power          int    `json:"power"`
	IsBroadcasting bool   `json:"is_broadcasting"`
}

// RFTemplate represents an RF configuration template.
// For Mist: org-level templates that can be applied to sites.
// For Meraki: per-network (site) RF profiles.
type RFTemplate struct {
	ObjectMeta                          // per-object cache freshness + apply state
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
	ObjectMeta          // per-object cache freshness + apply state
	ID           string `json:"id"`
	Name         string `json:"name"`
	OrgID        string `json:"org_id,omitempty"`
	SourceAPI    string `json:"-"`
	SourceVendor string `json:"-"`
}

// WLANTemplate represents a WLAN configuration template (Mist-specific).
type WLANTemplate struct {
	ObjectMeta          // per-object cache freshness + apply state
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
	ObjectMeta                            // per-object cache freshness + apply state
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
	ObjectMeta                          // per-object cache freshness + apply state
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
	ObjectMeta                          // per-object cache freshness + apply state
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
	ObjectMeta                          // per-object cache freshness + apply state
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

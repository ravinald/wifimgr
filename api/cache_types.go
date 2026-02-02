package api

import (
	"time"
)

// CacheMetadata contains metadata about the cache
type CacheMetadata struct {
	Created      time.Time `json:"created"`
	OrgID        string    `json:"org_id"`
	SitesCount   int       `json:"sites_count"`
	DevicesCount int       `json:"devices_count"`
	Version      string    `json:"version"`
	Expired      bool      `json:"expired,omitempty"`
	NeedsRebuild bool      `json:"needs_rebuild,omitempty"`
}

// OrgData represents all data for a single organization
type OrgData struct {
	// Organization stats from /api/v1/orgs/{org_id}/stats
	OrgStats *OrgStats `json:"org_stats,omitempty"`

	// All other data that was previously at top level
	Sites struct {
		Info     []MistSite    `json:"info"`
		Settings []SiteSetting `json:"settings"`
	} `json:"sites"`
	Templates struct {
		RF      []MistRFTemplate      `json:"rf"`
		Gateway []MistGatewayTemplate `json:"gateway"`
		WLAN    []MistWLANTemplate    `json:"wlan"`
	} `json:"templates"`
	Networks []MistNetwork `json:"networks"`
	WLANs    struct {
		Org   []MistWLAN            `json:"org"`
		Sites map[string][]MistWLAN `json:"sites"` // [siteID][]WLAN
	} `json:"wlans"`
	Inventory struct {
		AP      map[string]APDevice          `json:"ap"`      // MAC -> APDevice
		Switch  map[string]MistSwitchDevice  `json:"switch"`  // MAC -> SwitchDevice
		Gateway map[string]MistGatewayDevice `json:"gateway"` // MAC -> GatewayDevice
	} `json:"inventory"`
	Profiles struct {
		Devices []DeviceProfile          `json:"devices"`
		Details []map[string]interface{} `json:"details"`
	} `json:"profiles"`
	Configs struct {
		AP      map[string]APConfig      `json:"ap"`      // MAC -> APConfig
		Switch  map[string]SwitchConfig  `json:"switch"`  // MAC -> SwitchConfig
		Gateway map[string]GatewayConfig `json:"gateway"` // MAC -> GatewayConfig
	} `json:"configs"`
}

// Cache represents the new unified cache structure with orgs at top level
type Cache struct {
	Version int                 `json:"version"`
	Orgs    map[string]*OrgData `json:"orgs"` // [orgID]*OrgData
}

// CacheIndexes provides O(1) lookup indexes for all cached data
type CacheIndexes struct {
	// Organizations
	OrgsByName map[string]*OrgStats
	OrgsByID   map[string]*OrgStats

	// Sites
	SitesByName map[string]*MistSite
	SitesByID   map[string]*MistSite

	// Site Settings
	SiteSettingsBySiteID map[string]*SiteSetting
	SiteSettingsByID     map[string]*SiteSetting

	// Templates
	RFTemplatesByName   map[string]*MistRFTemplate
	RFTemplatesByID     map[string]*MistRFTemplate
	GWTemplatesByName   map[string]*MistGatewayTemplate
	GWTemplatesByID     map[string]*MistGatewayTemplate
	WLANTemplatesByName map[string]*MistWLANTemplate
	WLANTemplatesByID   map[string]*MistWLANTemplate

	// Networks
	NetworksByName map[string]*MistNetwork
	NetworksByID   map[string]*MistNetwork

	// WLANs
	OrgWLANsByName  map[string]*MistWLAN
	OrgWLANsByID    map[string]*MistWLAN
	SiteWLANsByName map[string]map[string]*MistWLAN // [siteID][name]
	SiteWLANsByID   map[string]map[string]*MistWLAN // [siteID][id]

	// Devices - using normalized MAC addresses as keys
	APsByName map[string]*APDevice
	APsByMAC  map[string]*APDevice
	APsBySite map[string][]*APDevice

	SwitchesByName map[string]*MistSwitchDevice
	SwitchesByMAC  map[string]*MistSwitchDevice
	SwitchesBySite map[string][]*MistSwitchDevice

	GatewaysByName map[string]*MistGatewayDevice
	GatewaysByMAC  map[string]*MistGatewayDevice
	GatewaysBySite map[string][]*MistGatewayDevice

	// Device Profiles
	DeviceProfilesByName map[string]*DeviceProfile
	DeviceProfilesByID   map[string]*DeviceProfile

	// Device Profile Details (full details from individual API calls)
	DeviceProfileDetailsByName map[string]*map[string]interface{}
	DeviceProfileDetailsByID   map[string]*map[string]interface{}

	// Device Configurations
	APConfigsByName map[string]*APConfig
	APConfigsByMAC  map[string]*APConfig

	SwitchConfigsByName map[string]*SwitchConfig
	SwitchConfigsByMAC  map[string]*SwitchConfig

	GatewayConfigsByName map[string]*GatewayConfig
	GatewayConfigsByMAC  map[string]*GatewayConfig
}

// MistNetwork represents an organization network with bidirectional data handling
type MistNetwork struct {
	// Core fields
	ID    *string `json:"id,omitempty"`
	Name  *string `json:"name,omitempty"`
	OrgID *string `json:"org_id,omitempty"`

	// Network configuration
	VlanID   *int    `json:"vlan_id,omitempty"`
	Subnet   *string `json:"subnet,omitempty"`
	Gateway  *string `json:"gateway,omitempty"`
	Subnet6  *string `json:"subnet6,omitempty"`
	Gateway6 *string `json:"gateway6,omitempty"`

	// Access control
	InternalAccess *bool `json:"internal_access,omitempty"`
	InternetAccess *bool `json:"internet_access,omitempty"`
	Isolation      *bool `json:"isolation,omitempty"`

	// Advanced settings
	Multicast *bool     `json:"multicast,omitempty"`
	Tenants   *[]string `json:"tenants,omitempty"`

	// Metadata
	CreatedTime  *int64 `json:"created_time,omitempty"`
	ModifiedTime *int64 `json:"modified_time,omitempty"`

	// Raw field removed - all API data mapped to struct fields

	// Additional configuration not covered by struct fields
	AdditionalConfig map[string]interface{} `json:"-"`
}

// MistWLAN represents a WLAN with bidirectional data handling
type MistWLAN struct {
	// Core fields
	ID     *string `json:"id,omitempty"`
	SSID   *string `json:"ssid,omitempty"`
	OrgID  *string `json:"org_id,omitempty"`
	SiteID *string `json:"site_id,omitempty"`

	// Network settings
	VlanID    *int    `json:"vlan_id,omitempty"`
	Interface *string `json:"interface,omitempty"`
	Isolation *bool   `json:"isolation,omitempty"`

	// Authentication
	Auth struct {
		Type       *string   `json:"type,omitempty"`
		PSK        *string   `json:"psk,omitempty"`
		KeyIdx     *int      `json:"key_idx,omitempty"`
		Keys       *[]string `json:"keys,omitempty"`
		Enterprise *struct {
			Radius *struct {
				Host   *string `json:"host,omitempty"`
				Port   *int    `json:"port,omitempty"`
				Secret *string `json:"secret,omitempty"`
			} `json:"radius,omitempty"`
		} `json:"enterprise,omitempty"`
	} `json:"auth,omitempty"`

	// Quality of Service
	QoS struct {
		Class *string `json:"class,omitempty"`
	} `json:"qos,omitempty"`

	// Advanced settings
	Band    *string   `json:"band,omitempty"`
	Enabled *bool     `json:"enabled,omitempty"`
	Hidden  *bool     `json:"hidden,omitempty"`
	ApplyTo *[]string `json:"apply_to,omitempty"`

	// Metadata
	CreatedTime  *int64 `json:"created_time,omitempty"`
	ModifiedTime *int64 `json:"modified_time,omitempty"`

	// Raw field removed - all API data mapped to struct fields

	// Additional configuration not covered by struct fields
	AdditionalConfig map[string]interface{} `json:"-"`
}

// MistRFTemplate represents an RF template with complete API response data
type MistRFTemplate struct {
	// Core fields
	ID    *string `json:"id,omitempty"`
	Name  *string `json:"name,omitempty"`
	OrgID *string `json:"org_id,omitempty"`

	// Antenna gain settings
	AntGain24 *int `json:"ant_gain_24,omitempty"`
	AntGain5  *int `json:"ant_gain_5,omitempty"`
	AntGain6  *int `json:"ant_gain_6,omitempty"`

	// Radio band configurations (complex objects from schema)
	Band24         map[string]interface{} `json:"band_24,omitempty"`
	Band24Usage    *string                `json:"band_24_usage,omitempty"`
	Band5          map[string]interface{} `json:"band_5,omitempty"`
	Band5On24Radio map[string]interface{} `json:"band_5_on_24_radio,omitempty"`
	Band6          map[string]interface{} `json:"band_6,omitempty"`

	// Settings
	CountryCode     *string `json:"country_code,omitempty"`
	ForSite         *bool   `json:"for_site,omitempty"`
	ScanningEnabled *bool   `json:"scanning_enabled,omitempty"`

	// Model-specific configurations
	ModelSpecific map[string]interface{} `json:"model_specific,omitempty"`

	// Metadata
	CreatedTime  *int64 `json:"created_time,omitempty"`
	ModifiedTime *int64 `json:"modified_time,omitempty"`
}

// MistGatewayTemplate represents a gateway template with bidirectional data handling
type MistGatewayTemplate struct {
	// Core fields
	ID    *string `json:"id,omitempty"`
	Name  *string `json:"name,omitempty"`
	OrgID *string `json:"org_id,omitempty"`
	Type  *string `json:"type,omitempty"`

	// Network configuration
	Networks   *map[string]interface{} `json:"networks,omitempty"`
	PortConfig *map[string]interface{} `json:"port_config,omitempty"`

	// Advanced gateway settings
	BGPConfig     *map[string]interface{} `json:"bgp_config,omitempty"`
	VRFConfig     *map[string]interface{} `json:"vrf_config,omitempty"`
	RoutingConfig *map[string]interface{} `json:"routing_config,omitempty"`

	// Metadata
	CreatedTime  *int64 `json:"created_time,omitempty"`
	ModifiedTime *int64 `json:"modified_time,omitempty"`

	// Raw field removed - all API data mapped to struct fields

	// Additional configuration not covered by struct fields
	AdditionalConfig map[string]interface{} `json:"-"`
}

// MistWLANTemplate represents a WLAN template with bidirectional data handling
type MistWLANTemplate struct {
	// Core fields
	ID    *string `json:"id,omitempty"`
	Name  *string `json:"name,omitempty"`
	OrgID *string `json:"org_id,omitempty"`

	// Template settings
	SSID      *string `json:"ssid,omitempty"`
	VlanID    *int    `json:"vlan_id,omitempty"`
	Interface *string `json:"interface,omitempty"`

	// Authentication template
	Auth *map[string]interface{} `json:"auth,omitempty"`

	// QoS template
	QoS *map[string]interface{} `json:"qos,omitempty"`

	// Advanced settings
	Band    *string `json:"band,omitempty"`
	Enabled *bool   `json:"enabled,omitempty"`
	Hidden  *bool   `json:"hidden,omitempty"`

	// Metadata
	CreatedTime  *int64 `json:"created_time,omitempty"`
	ModifiedTime *int64 `json:"modified_time,omitempty"`

	// Raw field removed - all API data mapped to struct fields

	// Additional configuration not covered by struct fields
	AdditionalConfig map[string]interface{} `json:"-"`
}

// NewCacheIndexes creates and initializes a new CacheIndexes struct
func NewCacheIndexes() *CacheIndexes {
	return &CacheIndexes{
		// Organizations
		OrgsByName: make(map[string]*OrgStats),
		OrgsByID:   make(map[string]*OrgStats),

		// Sites
		SitesByName: make(map[string]*MistSite),
		SitesByID:   make(map[string]*MistSite),

		// Site Settings
		SiteSettingsBySiteID: make(map[string]*SiteSetting),
		SiteSettingsByID:     make(map[string]*SiteSetting),

		// Templates
		RFTemplatesByName:   make(map[string]*MistRFTemplate),
		RFTemplatesByID:     make(map[string]*MistRFTemplate),
		GWTemplatesByName:   make(map[string]*MistGatewayTemplate),
		GWTemplatesByID:     make(map[string]*MistGatewayTemplate),
		WLANTemplatesByName: make(map[string]*MistWLANTemplate),
		WLANTemplatesByID:   make(map[string]*MistWLANTemplate),

		// Networks
		NetworksByName: make(map[string]*MistNetwork),
		NetworksByID:   make(map[string]*MistNetwork),

		// WLANs
		OrgWLANsByName:  make(map[string]*MistWLAN),
		OrgWLANsByID:    make(map[string]*MistWLAN),
		SiteWLANsByName: make(map[string]map[string]*MistWLAN),
		SiteWLANsByID:   make(map[string]map[string]*MistWLAN),

		// Devices
		APsByName: make(map[string]*APDevice),
		APsByMAC:  make(map[string]*APDevice),
		APsBySite: make(map[string][]*APDevice),

		SwitchesByName: make(map[string]*MistSwitchDevice),
		SwitchesByMAC:  make(map[string]*MistSwitchDevice),
		SwitchesBySite: make(map[string][]*MistSwitchDevice),

		GatewaysByName: make(map[string]*MistGatewayDevice),
		GatewaysByMAC:  make(map[string]*MistGatewayDevice),
		GatewaysBySite: make(map[string][]*MistGatewayDevice),

		// Device Profiles
		DeviceProfilesByName: make(map[string]*DeviceProfile),
		DeviceProfilesByID:   make(map[string]*DeviceProfile),

		// Device Profile Details
		DeviceProfileDetailsByName: make(map[string]*map[string]interface{}),
		DeviceProfileDetailsByID:   make(map[string]*map[string]interface{}),

		// Device Configurations
		APConfigsByName:      make(map[string]*APConfig),
		APConfigsByMAC:       make(map[string]*APConfig),
		SwitchConfigsByName:  make(map[string]*SwitchConfig),
		SwitchConfigsByMAC:   make(map[string]*SwitchConfig),
		GatewayConfigsByName: make(map[string]*GatewayConfig),
		GatewayConfigsByMAC:  make(map[string]*GatewayConfig),
	}
}

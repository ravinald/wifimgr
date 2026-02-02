package api

import (
	"time"
)

// InventoryConfig represents the structure of the inventory configuration file
type InventoryConfig struct {
	Version  int `json:"version"`
	Metadata struct {
		Description string `json:"description"`
	} `json:"metadata"`
	Config struct {
		Inventory struct {
			AP      []string `json:"ap"`
			Switch  []string `json:"switch"`
			Gateway []string `json:"gateway"`
		} `json:"inventory"`
	} `json:"config"`
	AvailableFields struct {
		AP      []string `json:"ap,omitempty"`
		Switch  []string `json:"switch,omitempty"`
		Gateway []string `json:"gateway,omitempty"`
	} `json:"available_fields,omitempty"`
}

// UUID type for Mist IDs
type UUID string

// Site represents a Mist site
type Site struct {
	// Id is the unique identifier for the site
	Id *UUID `json:"id,omitempty"`

	// Name is the name of the site
	Name *string `json:"name,omitempty"`

	// Address is the full address of the site
	Address *string `json:"address,omitempty"`

	// CountryCode is the country code for the site (for AP config generation)
	CountryCode *string `json:"country_code,omitempty"`

	// Timezone is the timezone of the site
	Timezone *string `json:"timezone,omitempty"`

	// Notes are additional information about the site
	Notes *string `json:"notes,omitempty"`

	// Location of the site
	Latlng *LatLng `json:"latlng,omitempty"`

	// CreatedTime is when the site was created
	CreatedTime *float64 `json:"created_time,omitempty"`

	// ModifiedTime is when the site was last modified
	ModifiedTime *float64 `json:"modified_time,omitempty"`
}

// DeviceProfile types are now defined in device_profile_types.go

// AP represents a Mist access point
type AP struct {
	// Id is the unique identifier for the AP
	Id *UUID `json:"id,omitempty"`

	// Name is the name of the AP
	Name *string `json:"name,omitempty"`

	// Serial is the serial number of the AP
	Serial *string `json:"serial,omitempty"`

	// Mac is the MAC address of the AP
	Mac *string `json:"mac,omitempty"`

	// Magic is the claim code for the device
	Magic *string `json:"magic,omitempty"`

	// Model is the AP model
	Model *string `json:"model,omitempty"`

	// HwRev is the hardware revision of the AP
	HwRev *string `json:"hw_rev,omitempty"`

	// Type is the AP type
	Type *string `json:"type,omitempty"`

	// Tags are labels applied to the AP
	Tags *[]string `json:"tags,omitempty"`

	// Notes are additional information about the AP
	Notes *string `json:"notes,omitempty"`

	// SiteId is the ID of the site the AP belongs to
	SiteId *UUID `json:"site_id,omitempty"`

	// OrgId is the ID of the organization
	OrgId *string `json:"org_id,omitempty"`

	// Location of the AP [latitude, longitude]
	Location *[]float64 `json:"location,omitempty"`

	// Orientation of the AP in degrees
	Orientation *int `json:"orientation,omitempty"`

	// Status of the AP (connected, disconnected, etc.)
	Status *string `json:"status,omitempty"`

	// Connected indicates if the device is connected
	Connected *bool `json:"connected,omitempty"`

	// Led indicates whether LED is enabled on the AP
	Led *bool `json:"led,omitempty"`

	// MapID is the map the AP is placed on
	MapID *string `json:"map_id,omitempty"`

	// TagUUID is the tag UUID
	TagUUID *string `json:"tag_uuid,omitempty"`

	// TagID is the tag ID
	TagID *int `json:"tag_id,omitempty"`

	// EvpnScope is the EVPN scope
	EvpnScope *string `json:"evpn_scope,omitempty"`

	// EvpntopoID is the EVPN topology ID
	EvpntopoID *string `json:"evpntopo_id,omitempty"`

	// StIPBase is the ST IP base
	StIPBase *string `json:"st_ip_base,omitempty"`

	// BundledMac is the bundled MAC address
	BundledMac *string `json:"bundled_mac,omitempty"`

	// CreatedTime is when the AP was created
	CreatedTime *int64 `json:"created_time,omitempty"`

	// ModifiedTime is when the AP was last modified
	ModifiedTime *int64 `json:"modified_time,omitempty"`

	// SKU is the stock keeping unit
	SKU *string `json:"sku,omitempty"`

	// DeviceProfileId is the deviceprofile ID if assigned, null if not assigned
	DeviceProfileId *string `json:"deviceprofile_id,omitempty"`

	// Band24 is the 2.4GHz radio configuration (legacy)
	Band24 *BandConfig `json:"band_24,omitempty"`

	// Band5 is the 5GHz radio configuration (legacy)
	Band5 *BandConfig `json:"band_5,omitempty"`

	// LastSeen is the last time the AP was seen by the Mist cloud
	LastSeen *time.Time `json:"last_seen,omitempty"`

	// RadioConfig contains the radio configuration
	RadioConfig *RadioConfig `json:"radio_config,omitempty"`
}

// BandConfig represents radio configuration for a specific band
type BandConfig struct {
	// Disabled indicates whether the radio is disabled
	Disabled *bool `json:"disabled,omitempty"`

	// Channel is the channel number
	Channel *int `json:"channel,omitempty"`

	// TxPower is the transmission power in dBm
	TxPower *int `json:"tx_power,omitempty"`
}

// RadioConfigBand24 represents the radio configuration for the 2.4GHz band
type RadioConfigBand24 struct {
	// Disabled indicates whether the radio is disabled
	Disabled *bool `json:"disabled,omitempty"`

	// AllowRRMDisable allows the RRM to disable the radio
	AllowRRMDisable *bool `json:"allow_rrm_disable,omitempty"`

	// AntGain is the antenna gain in dB
	AntGain *float64 `json:"ant_gain,omitempty"`

	// AntennaMode is the antenna mode (1x1, 2x2, 3x3, 4x4, default)
	AntennaMode *string `json:"antenna_mode,omitempty"`

	// Bandwidth is the channel width for the 2.4GHz band (20, 40)
	Bandwidth *string `json:"bandwidth,omitempty"`

	// Channel is the primary channel for the band
	Channel *int `json:"channel,omitempty"`

	// Channels is a list of channels for RFTemplates
	Channels *[]int `json:"channels,omitempty"`

	// Power is the TX power of the radio
	Power *int `json:"power,omitempty"`

	// PowerMax is the maximum TX power to use when Power=0
	PowerMax *int `json:"power_max,omitempty"`

	// PowerMin is the minimum TX power to use when Power=0
	PowerMin *int `json:"power_min,omitempty"`

	// Preamble is the preamble type (auto, long, short)
	Preamble *string `json:"preamble,omitempty"`
}

// RadioConfigBand5 represents the radio configuration for the 5GHz band
type RadioConfigBand5 struct {
	// Disabled indicates whether the radio is disabled
	Disabled *bool `json:"disabled,omitempty"`

	// AllowRRMDisable allows the RRM to disable the radio
	AllowRRMDisable *bool `json:"allow_rrm_disable,omitempty"`

	// AntGain is the antenna gain in dB
	AntGain *float64 `json:"ant_gain,omitempty"`

	// AntennaMode is the antenna mode (1x1, 2x2, 3x3, 4x4, default)
	AntennaMode *string `json:"antenna_mode,omitempty"`

	// Bandwidth is the channel width for the 5GHz band (20, 40, 80)
	Bandwidth *string `json:"bandwidth,omitempty"`

	// Channel is the primary channel for the band
	Channel *int `json:"channel,omitempty"`

	// Channels is a list of channels for RFTemplates
	Channels *[]int `json:"channels,omitempty"`

	// Power is the TX power of the radio
	Power *int `json:"power,omitempty"`

	// PowerMax is the maximum TX power to use when Power=0
	PowerMax *int `json:"power_max,omitempty"`

	// PowerMin is the minimum TX power to use when Power=0
	PowerMin *int `json:"power_min,omitempty"`

	// Preamble is the preamble type (auto, long, short)
	Preamble *string `json:"preamble,omitempty"`
}

// RadioConfigBand6 represents the radio configuration for the 6GHz band
type RadioConfigBand6 struct {
	// Disabled indicates whether the radio is disabled
	Disabled *bool `json:"disabled,omitempty"`

	// AllowRRMDisable allows the RRM to disable the radio
	AllowRRMDisable *bool `json:"allow_rrm_disable,omitempty"`

	// AntGain is the antenna gain in dB
	AntGain *float64 `json:"ant_gain,omitempty"`

	// AntennaMode is the antenna mode (1x1, 2x2, 3x3, 4x4, default)
	AntennaMode *string `json:"antenna_mode,omitempty"`

	// Bandwidth is the channel width for the 6GHz band (20, 40, 80, 160)
	Bandwidth *string `json:"bandwidth,omitempty"`

	// Channel is the primary channel for the band
	Channel *int `json:"channel,omitempty"`

	// Channels is a list of channels for RFTemplates
	Channels *[]int `json:"channels,omitempty"`

	// Power is the TX power of the radio
	Power *int `json:"power,omitempty"`

	// PowerMax is the maximum TX power to use when Power=0
	PowerMax *int `json:"power_max,omitempty"`

	// PowerMin is the minimum TX power to use when Power=0
	PowerMin *int `json:"power_min,omitempty"`

	// Preamble is the preamble type (auto, long, short)
	Preamble *string `json:"preamble,omitempty"`

	// StandardPower indicates whether to use standard-power operation with AFC
	StandardPower *bool `json:"standard_power,omitempty"`
}

// RadioConfig represents the radio configuration for an AP
type RadioConfig struct {
	// AllowRRMDisable allows the RRM to disable the radio
	AllowRRMDisable *bool `json:"allow_rrm_disable,omitempty"`

	// AntGain24 is the antenna gain for 2.4G
	AntGain24 *float64 `json:"ant_gain_24,omitempty"`

	// AntGain5 is the antenna gain for 5G
	AntGain5 *float64 `json:"ant_gain_5,omitempty"`

	// AntGain6 is the antenna gain for 6G
	AntGain6 *float64 `json:"ant_gain_6,omitempty"`

	// AntennaMode is the antenna mode for the AP
	AntennaMode *string `json:"antenna_mode,omitempty"`

	// Band24 is the 2.4GHz radio configuration
	Band24 *RadioConfigBand24 `json:"band_24,omitempty"`

	// Band24Usage configures how 2.4GHz radio is used (24, 5, 6, auto)
	Band24Usage *string `json:"band_24_usage,omitempty"`

	// Band5 is the 5GHz radio configuration
	Band5 *RadioConfigBand5 `json:"band_5,omitempty"`

	// Band5On24Radio is the 5GHz configuration for dual 5GHz APs
	Band5On24Radio *RadioConfigBand5 `json:"band_5_on_24_radio,omitempty"`

	// Band6 is the 6GHz radio configuration
	Band6 *RadioConfigBand6 `json:"band_6,omitempty"`

	// FullAutomaticRRM indicates whether to let RRM control everything
	FullAutomaticRRM *bool `json:"full_automatic_rrm,omitempty"`

	// IndoorUse indicates whether an outdoor AP should operate as indoor
	IndoorUse *bool `json:"indoor_use,omitempty"`

	// ScanningEnabled indicates whether scanning radio is enabled
	ScanningEnabled *bool `json:"scanning_enabled,omitempty"`
}

// For legacy compatibility
type RadioConfigBand struct {
	// Disabled indicates whether the radio is disabled
	Disabled *bool `json:"disabled,omitempty"`

	// AllowRRMDisable allows the RRM to disable the radio
	AllowRRMDisable *bool `json:"allow_rrm_disable,omitempty"`

	// Power is the power setting
	Power *int `json:"power,omitempty"`

	// Channel is the channel number
	Channel *int `json:"channel,omitempty"`

	// Bandwidth is the channel bandwidth
	Bandwidth *int `json:"bandwidth,omitempty"`
}

// LatLng represents geographical coordinates
type LatLng struct {
	// Lat is the latitude
	Lat float64 `json:"lat"`

	// Lng is the longitude
	Lng float64 `json:"lng"`
}

// InventoryAssignResponse represents the response from an inventory assign operation
type InventoryAssignResponse struct {
	// Op is the operation performed (assign, unassign)
	Op string `json:"op,omitempty"`

	// Success is the list of MAC addresses that were successfully processed
	Success []string `json:"success,omitempty"`

	// Error is the list of MAC addresses that failed to be processed
	Error []string `json:"error,omitempty"`

	// Reason contains the error reasons for each failed MAC address
	Reason []string `json:"reason,omitempty"`
}

// InventoryItem represents a device in the Mist inventory
type InventoryItem struct {
	// Id is the unique identifier for the device
	Id *UUID `json:"id,omitempty"`

	// Mac is the MAC address of the device
	Mac *string `json:"mac,omitempty"`

	// Serial is the serial number of the device
	Serial *string `json:"serial,omitempty"`

	// Model is the device model
	Model *string `json:"model,omitempty"`

	// SKU is the stock keeping unit
	SKU *string `json:"sku,omitempty"`

	// HwRev is the device hardware revision number
	HwRev *string `json:"hw_rev,omitempty"`

	// Type is the device type (ap, switch, gateway)
	Type *string `json:"type,omitempty"`

	// Magic is the device claim code
	Magic *string `json:"magic,omitempty"`

	// Name is the device name if configured
	Name *string `json:"name,omitempty"`

	// Hostname is the hostname reported by the device
	Hostname *string `json:"hostname,omitempty"`

	// JSI indicates if the device is JSI
	JSI *bool `json:"jsi,omitempty"`

	// Virtual Chassis fields
	ChassisMac    *string `json:"chassis_mac,omitempty"`
	ChassisSerial *string `json:"chassis_serial,omitempty"`
	VcMac         *string `json:"vc_mac,omitempty"`

	// OrgId is the organization ID
	OrgId *UUID `json:"org_id,omitempty"`

	// SiteId is the site ID if assigned, null if not assigned
	SiteId *UUID `json:"site_id,omitempty"`

	// CreatedTime is the inventory created time, in epoch
	CreatedTime *int64 `json:"created_time,omitempty"`

	// ModifiedTime is the inventory last modified time, in epoch
	ModifiedTime *int64 `json:"modified_time,omitempty"`

	// DeviceProfileId is the deviceprofile ID if assigned, null if not assigned
	DeviceProfileId *string `json:"deviceprofile_id,omitempty"`

	// Connected indicates whether the device is connected
	Connected *bool `json:"connected,omitempty"`

	// Adopted indicates whether the switch/gateway is adopted
	Adopted *bool `json:"adopted,omitempty"`
}

// Device represents a universal device in the Mist system
// This type is used to handle all device types (AP, Switch, Gateway) uniformly
type Device struct {
	// Core fields from InventoryItem
	Id           *UUID   `json:"id,omitempty"`
	Mac          *string `json:"mac,omitempty"`
	Serial       *string `json:"serial,omitempty"`
	Name         *string `json:"name,omitempty"`
	Model        *string `json:"model,omitempty"`
	SKU          *string `json:"sku,omitempty"`
	HwRev        *string `json:"hw_rev,omitempty"`
	Type         *string `json:"type,omitempty"`
	Magic        *string `json:"magic,omitempty"`
	Hostname     *string `json:"hostname,omitempty"`
	JSI          *bool   `json:"jsi,omitempty"`
	OrgId        *string `json:"org_id,omitempty"`
	SiteId       *string `json:"site_id,omitempty"`
	CreatedTime  *int64  `json:"created_time,omitempty"`
	ModifiedTime *int64  `json:"modified_time,omitempty"`
	Connected    *bool   `json:"connected,omitempty"`
	Adopted      *bool   `json:"adopted,omitempty"`

	// AP-specific fields
	Location        *[]float64  `json:"location,omitempty"`
	Orientation     *int        `json:"orientation,omitempty"`
	Status          *string     `json:"status,omitempty"`
	Led             *bool       `json:"led,omitempty"`
	MapID           *string     `json:"map_id,omitempty"`
	RadioConfig     interface{} `json:"radio_config,omitempty"`
	Notes           *string     `json:"notes,omitempty"`
	TagUUID         *string     `json:"tag_uuid,omitempty"`
	TagID           *int        `json:"tag_id,omitempty"`
	EvpnScope       *string     `json:"evpn_scope,omitempty"`
	EvpntopoID      *string     `json:"evpntopo_id,omitempty"`
	StIPBase        *string     `json:"st_ip_base,omitempty"`
	BundledMac      *string     `json:"bundled_mac,omitempty"`
	DeviceProfileId *string     `json:"deviceprofile_id,omitempty"`

	// Switch-specific fields
	IP      *string `json:"ip,omitempty"`
	Port    *int    `json:"port,omitempty"`
	Version *string `json:"version,omitempty"`

	// Gateway-specific fields
	MgmtIntf *string `json:"mgmt_intf,omitempty"`

	// Additional device configuration - varies by device type
	Config map[string]interface{} `json:"config,omitempty"`
}

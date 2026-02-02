// Package vendors provides vendor-agnostic types for network device configuration.
// These types use Mist nomenclature as the standard, with vendor-specific extension
// blocks (mist:, meraki:) for vendor-specific settings.
package vendors

// APDeviceConfig represents vendor-agnostic AP configuration.
// Field names follow Mist API conventions. Vendor-specific settings should be
// placed in the Mist or Meraki extension blocks.
type APDeviceConfig struct {
	// Identity
	Name  string   `json:"name,omitempty"`
	Tags  []string `json:"tags,omitempty"`
	Notes string   `json:"notes,omitempty"`

	// Location
	Location    []float64 `json:"location,omitempty"`    // [lat, lng]
	Orientation *int      `json:"orientation,omitempty"` // degrees
	MapID       string    `json:"map_id,omitempty"`
	MapName     string    `json:"map_name,omitempty"` // Alternative to map_id (mutually exclusive)
	X           *float64  `json:"x,omitempty"`
	Y           *float64  `json:"y,omitempty"`
	Height      *float64  `json:"height,omitempty"`

	// Configuration blocks
	RadioConfig  *RadioConfig  `json:"radio_config,omitempty"`
	IPConfig     *IPConfig     `json:"ip_config,omitempty"`
	BLEConfig    *BLEConfig    `json:"ble_config,omitempty"`
	MeshConfig   *MeshConfig   `json:"mesh,omitempty"`
	PortConfig   []PortConfig  `json:"port_config,omitempty"`
	LACPConfig   *LACPConfig   `json:"lacp_config,omitempty"`
	UplinkConfig *UplinkConfig `json:"uplink_port_config,omitempty"`
	IoTConfig    *IoTConfig    `json:"iot_config,omitempty"`
	PowerConfig  *PowerConfig  `json:"pwr_config,omitempty"`
	LEDConfig    *LEDConfig    `json:"led,omitempty"`

	// Hardware flags
	DisableEth1    *bool `json:"disable_eth1,omitempty"`
	DisableEth2    *bool `json:"disable_eth2,omitempty"`
	DisableEth3    *bool `json:"disable_eth3,omitempty"`
	DisableModule  *bool `json:"disable_module,omitempty"`
	PoEPassthrough *bool `json:"poe_passthrough,omitempty"`

	// References - ID or Name (mutually exclusive)
	DeviceProfileID   string `json:"deviceprofile_id,omitempty"`
	DeviceProfileName string `json:"deviceprofile_name,omitempty"`

	// Variables for templating
	Vars map[string]any `json:"vars,omitempty"`

	// Vendor extensions - for vendor-specific settings not in common schema
	Mist   map[string]any `json:"mist,omitempty"`
	Meraki map[string]any `json:"meraki,omitempty"`
}

// RadioConfig contains radio/band configuration for APs.
type RadioConfig struct {
	// Global settings
	AllowRRMDisable  *bool    `json:"allow_rrm_disable,omitempty"`
	FullAutomaticRRM *bool    `json:"full_automatic_rrm,omitempty"`
	IndoorUse        *bool    `json:"indoor_use,omitempty"`
	ScanningEnabled  *bool    `json:"scanning_enabled,omitempty"`
	AntGain24        *float64 `json:"ant_gain_24,omitempty"`
	AntGain5         *float64 `json:"ant_gain_5,omitempty"`
	AntGain6         *float64 `json:"ant_gain_6,omitempty"`
	AntennaMode      *string  `json:"antenna_mode,omitempty"` // "default", "1x1", "2x2", etc.
	Band24Usage      *string  `json:"band_24_usage,omitempty"`

	// Per-band configuration
	Band24         *RadioBandConfig `json:"band_24,omitempty"`
	Band5          *RadioBandConfig `json:"band_5,omitempty"`
	Band5On24Radio *RadioBandConfig `json:"band_5_on_24_radio,omitempty"`
	Band6          *RadioBandConfig `json:"band_6,omitempty"`

	// Vendor extensions
	Mist   map[string]any `json:"mist,omitempty"`
	Meraki map[string]any `json:"meraki,omitempty"`
}

// RadioBandConfig contains per-band radio settings.
type RadioBandConfig struct {
	Disabled        *bool    `json:"disabled,omitempty"`
	Channel         *int     `json:"channel,omitempty"`
	Channels        []int    `json:"channels,omitempty"` // Allowed channel list
	Power           *int     `json:"power,omitempty"`    // dBm
	PowerMin        *int     `json:"power_min,omitempty"`
	PowerMax        *int     `json:"power_max,omitempty"`
	Bandwidth       *int     `json:"bandwidth,omitempty"` // 20, 40, 80, 160
	AntennaMode     *string  `json:"antenna_mode,omitempty"`
	AntGain         *float64 `json:"ant_gain,omitempty"`
	AllowRRMDisable *bool    `json:"allow_rrm_disable,omitempty"`
	Preamble        *string  `json:"preamble,omitempty"` // "short", "long", "auto"

	// Vendor extensions
	Mist   map[string]any `json:"mist,omitempty"`
	Meraki map[string]any `json:"meraki,omitempty"`
}

// IPConfig contains network IP configuration.
type IPConfig struct {
	Type      *string  `json:"type,omitempty"` // "dhcp", "static"
	IP        *string  `json:"ip,omitempty"`
	Netmask   *string  `json:"netmask,omitempty"`
	Gateway   *string  `json:"gateway,omitempty"`
	DNS       []string `json:"dns,omitempty"`
	DNSSuffix []string `json:"dns_suffix,omitempty"`
	VlanID    *int     `json:"vlan_id,omitempty"`
	Mtu       *int     `json:"mtu,omitempty"`

	// Vendor extensions
	Mist   map[string]any `json:"mist,omitempty"`
	Meraki map[string]any `json:"meraki,omitempty"`
}

// BLEConfig contains Bluetooth Low Energy configuration.
type BLEConfig struct {
	Enabled *bool   `json:"enabled,omitempty"`
	Power   *int    `json:"power,omitempty"` // dBm
	Mode    *string `json:"mode,omitempty"`  // "unique", "shared"

	// Beacon types
	IBeacon   *IBeaconConfig   `json:"ibeacon,omitempty"`
	Eddystone *EddystoneConfig `json:"eddystone,omitempty"`

	// Vendor extensions
	Mist   map[string]any `json:"mist,omitempty"`
	Meraki map[string]any `json:"meraki,omitempty"`
}

// IBeaconConfig contains iBeacon-specific settings.
type IBeaconConfig struct {
	Enabled *bool   `json:"enabled,omitempty"`
	UUID    *string `json:"uuid,omitempty"`
	Major   *int    `json:"major,omitempty"`
	Minor   *int    `json:"minor,omitempty"`
	Power   *int    `json:"power,omitempty"` // Override BLE power for iBeacon
}

// EddystoneConfig contains Eddystone beacon settings.
type EddystoneConfig struct {
	Enabled     *bool   `json:"enabled,omitempty"`
	NamespaceID *string `json:"namespace_id,omitempty"`
	InstanceID  *string `json:"instance_id,omitempty"`
	URL         *string `json:"url,omitempty"`
}

// MeshConfig contains wireless mesh networking configuration.
type MeshConfig struct {
	Enabled *bool   `json:"enabled,omitempty"`
	Role    *string `json:"role,omitempty"`  // "root", "node", "spreader"
	Group   *string `json:"group,omitempty"` // Mesh group identifier

	// Vendor extensions
	Mist   map[string]any `json:"mist,omitempty"`
	Meraki map[string]any `json:"meraki,omitempty"`
}

// PortConfig contains Ethernet port configuration.
type PortConfig struct {
	PortID      *string `json:"port_id,omitempty"` // "eth0", "eth1", etc.
	Enabled     *bool   `json:"enabled,omitempty"`
	Mode        *string `json:"mode,omitempty"` // "access", "trunk"
	VlanID      *int    `json:"vlan_id,omitempty"`
	VlanIDs     []int   `json:"vlan_ids,omitempty"` // For trunk mode
	NativeVlan  *int    `json:"native_vlan,omitempty"`
	VoiceVlan   *int    `json:"voice_vlan,omitempty"`
	PoEEnabled  *bool   `json:"poe_enabled,omitempty"`
	SpeedDuplex *string `json:"speed_duplex,omitempty"` // "auto", "100/full", "1000/full"
	Description *string `json:"description,omitempty"`

	// 802.1X port authentication
	PortAuth *PortAuthConfig `json:"port_auth,omitempty"`

	// Vendor extensions
	Mist   map[string]any `json:"mist,omitempty"`
	Meraki map[string]any `json:"meraki,omitempty"`
}

// PortAuthConfig contains 802.1X port authentication settings.
type PortAuthConfig struct {
	Enabled    *bool   `json:"enabled,omitempty"`
	Mode       *string `json:"mode,omitempty"` // "single", "multi", "mac"
	GuestVlan  *int    `json:"guest_vlan,omitempty"`
	FailedVlan *int    `json:"failed_vlan,omitempty"`
}

// LACPConfig contains Link Aggregation Control Protocol configuration.
type LACPConfig struct {
	Enabled     *bool    `json:"enabled,omitempty"`
	PortMembers []string `json:"port_members,omitempty"` // ["eth0", "eth1"]
	LACPMode    *string  `json:"lacp_mode,omitempty"`    // "active", "passive"
	HashMode    *string  `json:"hash_mode,omitempty"`    // "L2", "L3", "L4"

	// Vendor extensions
	Mist   map[string]any `json:"mist,omitempty"`
	Meraki map[string]any `json:"meraki,omitempty"`
}

// UplinkConfig contains uplink port configuration.
type UplinkConfig struct {
	// 802.1X uplink authentication
	Dot1xEnabled *bool   `json:"dot1x_enabled,omitempty"`
	Identity     *string `json:"identity,omitempty"`
	Password     *string `json:"password,omitempty"`

	// Uplink preferences
	Primary   *string `json:"primary,omitempty"`   // "eth0", "wlan"
	Secondary *string `json:"secondary,omitempty"` // Fallback uplink

	// Vendor extensions
	Mist   map[string]any `json:"mist,omitempty"`
	Meraki map[string]any `json:"meraki,omitempty"`
}

// IoTConfig contains IoT/sensor configuration.
type IoTConfig struct {
	Enabled *bool   `json:"enabled,omitempty"`
	Type    *string `json:"type,omitempty"` // Sensor/IoT type

	// Vendor extensions
	Mist   map[string]any `json:"mist,omitempty"`
	Meraki map[string]any `json:"meraki,omitempty"`
}

// PowerConfig contains power/PoE configuration.
type PowerConfig struct {
	Mode      *string `json:"mode,omitempty"` // "auto", "power_supply", "low"
	BaseValue *int    `json:"base_value,omitempty"`

	// Vendor extensions
	Mist   map[string]any `json:"mist,omitempty"`
	Meraki map[string]any `json:"meraki,omitempty"`
}

// LEDConfig contains LED status light configuration.
type LEDConfig struct {
	Enabled    *bool `json:"enabled,omitempty"`
	Brightness *int  `json:"brightness,omitempty"` // 0-100

	// Vendor extensions
	Mist   map[string]any `json:"mist,omitempty"`
	Meraki map[string]any `json:"meraki,omitempty"`
}

// Validate checks APDeviceConfig for configuration errors.
// Returns an error if mutually exclusive fields are both set.
func (c *APDeviceConfig) Validate() error {
	if c == nil {
		return nil
	}

	// Check mutually exclusive ID/Name fields
	if c.DeviceProfileID != "" && c.DeviceProfileName != "" {
		return &ConfigValidationError{
			Field:   "deviceprofile_id/deviceprofile_name",
			Message: "deviceprofile_id and deviceprofile_name are mutually exclusive",
		}
	}

	if c.MapID != "" && c.MapName != "" {
		return &ConfigValidationError{
			Field:   "map_id/map_name",
			Message: "map_id and map_name are mutually exclusive",
		}
	}

	// Validate nested configs
	if c.RadioConfig != nil {
		if err := c.RadioConfig.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// ValidateForVendor checks if configuration is valid for the target vendor API.
// Returns a slice of validation errors if vendor-specific constraints are violated.
func (c *APDeviceConfig) ValidateForVendor(vendor string) []error {
	if c == nil {
		return nil
	}

	var errors []error

	// Check vendor block matches target vendor
	if vendor == "mist" && c.Meraki != nil && len(c.Meraki) > 0 {
		errors = append(errors, &ConfigValidationError{
			Field:   "meraki",
			Message: "configuration contains 'meraki:' block but device targets Mist API",
		})
	}
	if vendor == "meraki" && c.Mist != nil && len(c.Mist) > 0 {
		errors = append(errors, &ConfigValidationError{
			Field:   "mist",
			Message: "configuration contains 'mist:' block but device targets Meraki API",
		})
	}

	// Vendor-specific field validation
	if vendor == "meraki" {
		if c.RadioConfig != nil {
			if c.RadioConfig.Band24Usage != nil {
				errors = append(errors, &ConfigValidationError{
					Field:   "radio_config.band_24_usage",
					Message: "field 'band_24_usage' is Mist-specific and not supported by Meraki",
				})
			}
		}
	}

	return errors
}

// Validate checks RadioConfig for configuration errors.
func (c *RadioConfig) Validate() error {
	if c == nil {
		return nil
	}

	// Check for rf_profile_id/rf_profile_name mutual exclusion in Meraki extension
	if c.Meraki != nil {
		hasID := c.Meraki["rf_profile_id"] != nil && c.Meraki["rf_profile_id"] != ""
		hasName := c.Meraki["rf_profile_name"] != nil && c.Meraki["rf_profile_name"] != ""
		if hasID && hasName {
			return &ConfigValidationError{
				Field:   "radio_config.meraki.rf_profile_id/rf_profile_name",
				Message: "rf_profile_id and rf_profile_name are mutually exclusive",
			}
		}
	}

	return nil
}

// ValidateForVendor checks if radio configuration is valid for the target vendor.
func (c *RadioConfig) ValidateForVendor(vendor string) []error {
	if c == nil {
		return nil
	}

	var errors []error

	// Check vendor extension blocks match target vendor
	if vendor == "mist" && c.Meraki != nil && len(c.Meraki) > 0 {
		errors = append(errors, &ConfigValidationError{
			Field:   "radio_config.meraki",
			Message: "radio_config contains 'meraki:' block but device targets Mist API",
		})
	}
	if vendor == "meraki" && c.Mist != nil && len(c.Mist) > 0 {
		errors = append(errors, &ConfigValidationError{
			Field:   "radio_config.mist",
			Message: "radio_config contains 'mist:' block but device targets Meraki API",
		})
	}

	return errors
}

// ConfigValidationError represents a configuration validation error.
type ConfigValidationError struct {
	Field   string
	Message string
}

func (e *ConfigValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// ToMap converts APDeviceConfig to a map[string]any for API calls.
func (c *APDeviceConfig) ToMap() map[string]any {
	if c == nil {
		return nil
	}

	result := make(map[string]any)

	// Identity
	if c.Name != "" {
		result["name"] = c.Name
	}
	if len(c.Tags) > 0 {
		result["tags"] = c.Tags
	}
	if c.Notes != "" {
		result["notes"] = c.Notes
	}

	// Location
	if len(c.Location) > 0 {
		result["location"] = c.Location
	}
	if c.Orientation != nil {
		result["orientation"] = *c.Orientation
	}
	if c.MapID != "" {
		result["map_id"] = c.MapID
	}
	if c.X != nil {
		result["x"] = *c.X
	}
	if c.Y != nil {
		result["y"] = *c.Y
	}
	if c.Height != nil {
		result["height"] = *c.Height
	}

	// Configuration blocks
	if c.RadioConfig != nil {
		result["radio_config"] = c.RadioConfig.ToMap()
	}
	if c.IPConfig != nil {
		result["ip_config"] = c.IPConfig.ToMap()
	}
	if c.BLEConfig != nil {
		result["ble_config"] = c.BLEConfig.ToMap()
	}
	if c.MeshConfig != nil {
		result["mesh"] = c.MeshConfig.ToMap()
	}
	if len(c.PortConfig) > 0 {
		ports := make([]map[string]any, len(c.PortConfig))
		for i, p := range c.PortConfig {
			ports[i] = p.ToMap()
		}
		result["port_config"] = ports
	}
	if c.LACPConfig != nil {
		result["lacp_config"] = c.LACPConfig.ToMap()
	}
	if c.UplinkConfig != nil {
		result["uplink_port_config"] = c.UplinkConfig.ToMap()
	}
	if c.IoTConfig != nil {
		result["iot_config"] = c.IoTConfig.ToMap()
	}
	if c.PowerConfig != nil {
		result["pwr_config"] = c.PowerConfig.ToMap()
	}
	if c.LEDConfig != nil {
		result["led"] = c.LEDConfig.ToMap()
	}

	// Hardware flags
	if c.DisableEth1 != nil {
		result["disable_eth1"] = *c.DisableEth1
	}
	if c.DisableEth2 != nil {
		result["disable_eth2"] = *c.DisableEth2
	}
	if c.DisableEth3 != nil {
		result["disable_eth3"] = *c.DisableEth3
	}
	if c.DisableModule != nil {
		result["disable_module"] = *c.DisableModule
	}
	if c.PoEPassthrough != nil {
		result["poe_passthrough"] = *c.PoEPassthrough
	}

	// References
	if c.DeviceProfileID != "" {
		result["deviceprofile_id"] = c.DeviceProfileID
	}

	// Variables
	if len(c.Vars) > 0 {
		result["vars"] = c.Vars
	}

	// Vendor extensions
	if len(c.Mist) > 0 {
		for k, v := range c.Mist {
			result[k] = v // Mist extensions merge at top level
		}
	}
	// Meraki extensions are handled by the Meraki converter

	return result
}

// ToMap converts RadioConfig to a map.
func (c *RadioConfig) ToMap() map[string]any {
	if c == nil {
		return nil
	}

	result := make(map[string]any)

	if c.AllowRRMDisable != nil {
		result["allow_rrm_disable"] = *c.AllowRRMDisable
	}
	if c.FullAutomaticRRM != nil {
		result["full_automatic_rrm"] = *c.FullAutomaticRRM
	}
	if c.IndoorUse != nil {
		result["indoor_use"] = *c.IndoorUse
	}
	if c.ScanningEnabled != nil {
		result["scanning_enabled"] = *c.ScanningEnabled
	}
	if c.AntGain24 != nil {
		result["ant_gain_24"] = *c.AntGain24
	}
	if c.AntGain5 != nil {
		result["ant_gain_5"] = *c.AntGain5
	}
	if c.AntGain6 != nil {
		result["ant_gain_6"] = *c.AntGain6
	}
	if c.AntennaMode != nil {
		result["antenna_mode"] = *c.AntennaMode
	}
	if c.Band24Usage != nil {
		result["band_24_usage"] = *c.Band24Usage
	}

	if c.Band24 != nil {
		result["band_24"] = c.Band24.ToMap()
	}
	if c.Band5 != nil {
		result["band_5"] = c.Band5.ToMap()
	}
	if c.Band5On24Radio != nil {
		result["band_5_on_24_radio"] = c.Band5On24Radio.ToMap()
	}
	if c.Band6 != nil {
		result["band_6"] = c.Band6.ToMap()
	}

	// Merge Mist extensions at top level
	if len(c.Mist) > 0 {
		for k, v := range c.Mist {
			result[k] = v
		}
	}

	return result
}

// ToMap converts RadioBandConfig to a map.
func (c *RadioBandConfig) ToMap() map[string]any {
	if c == nil {
		return nil
	}

	result := make(map[string]any)

	if c.Disabled != nil {
		result["disabled"] = *c.Disabled
	}
	if c.Channel != nil {
		result["channel"] = *c.Channel
	}
	if len(c.Channels) > 0 {
		result["channels"] = c.Channels
	}
	if c.Power != nil {
		result["power"] = *c.Power
	}
	if c.PowerMin != nil {
		result["power_min"] = *c.PowerMin
	}
	if c.PowerMax != nil {
		result["power_max"] = *c.PowerMax
	}
	if c.Bandwidth != nil {
		result["bandwidth"] = *c.Bandwidth
	}
	if c.AntennaMode != nil {
		result["antenna_mode"] = *c.AntennaMode
	}
	if c.AntGain != nil {
		result["ant_gain"] = *c.AntGain
	}
	if c.AllowRRMDisable != nil {
		result["allow_rrm_disable"] = *c.AllowRRMDisable
	}
	if c.Preamble != nil {
		result["preamble"] = *c.Preamble
	}

	// Merge Mist extensions
	if len(c.Mist) > 0 {
		for k, v := range c.Mist {
			result[k] = v
		}
	}

	return result
}

// ToMap converts IPConfig to a map.
func (c *IPConfig) ToMap() map[string]any {
	if c == nil {
		return nil
	}

	result := make(map[string]any)

	if c.Type != nil {
		result["type"] = *c.Type
	}
	if c.IP != nil {
		result["ip"] = *c.IP
	}
	if c.Netmask != nil {
		result["netmask"] = *c.Netmask
	}
	if c.Gateway != nil {
		result["gateway"] = *c.Gateway
	}
	if len(c.DNS) > 0 {
		result["dns"] = c.DNS
	}
	if len(c.DNSSuffix) > 0 {
		result["dns_suffix"] = c.DNSSuffix
	}
	if c.VlanID != nil {
		result["vlan_id"] = *c.VlanID
	}
	if c.Mtu != nil {
		result["mtu"] = *c.Mtu
	}

	// Merge extensions
	if len(c.Mist) > 0 {
		for k, v := range c.Mist {
			result[k] = v
		}
	}

	return result
}

// ToMap converts BLEConfig to a map.
func (c *BLEConfig) ToMap() map[string]any {
	if c == nil {
		return nil
	}

	result := make(map[string]any)

	if c.Enabled != nil {
		result["enabled"] = *c.Enabled
	}
	if c.Power != nil {
		result["power"] = *c.Power
	}
	if c.Mode != nil {
		result["mode"] = *c.Mode
	}

	if c.IBeacon != nil {
		ibeacon := make(map[string]any)
		if c.IBeacon.Enabled != nil {
			ibeacon["enabled"] = *c.IBeacon.Enabled
		}
		if c.IBeacon.UUID != nil {
			ibeacon["uuid"] = *c.IBeacon.UUID
		}
		if c.IBeacon.Major != nil {
			ibeacon["major"] = *c.IBeacon.Major
		}
		if c.IBeacon.Minor != nil {
			ibeacon["minor"] = *c.IBeacon.Minor
		}
		if c.IBeacon.Power != nil {
			ibeacon["power"] = *c.IBeacon.Power
		}
		if len(ibeacon) > 0 {
			result["ibeacon"] = ibeacon
		}
	}

	if c.Eddystone != nil {
		eddystone := make(map[string]any)
		if c.Eddystone.Enabled != nil {
			eddystone["enabled"] = *c.Eddystone.Enabled
		}
		if c.Eddystone.NamespaceID != nil {
			eddystone["namespace_id"] = *c.Eddystone.NamespaceID
		}
		if c.Eddystone.InstanceID != nil {
			eddystone["instance_id"] = *c.Eddystone.InstanceID
		}
		if c.Eddystone.URL != nil {
			eddystone["url"] = *c.Eddystone.URL
		}
		if len(eddystone) > 0 {
			result["eddystone"] = eddystone
		}
	}

	// Merge extensions
	if len(c.Mist) > 0 {
		for k, v := range c.Mist {
			result[k] = v
		}
	}

	return result
}

// ToMap converts MeshConfig to a map.
func (c *MeshConfig) ToMap() map[string]any {
	if c == nil {
		return nil
	}

	result := make(map[string]any)

	if c.Enabled != nil {
		result["enabled"] = *c.Enabled
	}
	if c.Role != nil {
		result["role"] = *c.Role
	}
	if c.Group != nil {
		result["group"] = *c.Group
	}

	if len(c.Mist) > 0 {
		for k, v := range c.Mist {
			result[k] = v
		}
	}

	return result
}

// ToMap converts PortConfig to a map.
func (c *PortConfig) ToMap() map[string]any {
	result := make(map[string]any)

	if c.PortID != nil {
		result["port_id"] = *c.PortID
	}
	if c.Enabled != nil {
		result["enabled"] = *c.Enabled
	}
	if c.Mode != nil {
		result["mode"] = *c.Mode
	}
	if c.VlanID != nil {
		result["vlan_id"] = *c.VlanID
	}
	if len(c.VlanIDs) > 0 {
		result["vlan_ids"] = c.VlanIDs
	}
	if c.NativeVlan != nil {
		result["native_vlan"] = *c.NativeVlan
	}
	if c.VoiceVlan != nil {
		result["voice_vlan"] = *c.VoiceVlan
	}
	if c.PoEEnabled != nil {
		result["poe_enabled"] = *c.PoEEnabled
	}
	if c.SpeedDuplex != nil {
		result["speed_duplex"] = *c.SpeedDuplex
	}
	if c.Description != nil {
		result["description"] = *c.Description
	}

	if c.PortAuth != nil {
		portAuth := make(map[string]any)
		if c.PortAuth.Enabled != nil {
			portAuth["enabled"] = *c.PortAuth.Enabled
		}
		if c.PortAuth.Mode != nil {
			portAuth["mode"] = *c.PortAuth.Mode
		}
		if c.PortAuth.GuestVlan != nil {
			portAuth["guest_vlan"] = *c.PortAuth.GuestVlan
		}
		if c.PortAuth.FailedVlan != nil {
			portAuth["failed_vlan"] = *c.PortAuth.FailedVlan
		}
		if len(portAuth) > 0 {
			result["port_auth"] = portAuth
		}
	}

	if len(c.Mist) > 0 {
		for k, v := range c.Mist {
			result[k] = v
		}
	}

	return result
}

// ToMap converts LACPConfig to a map.
func (c *LACPConfig) ToMap() map[string]any {
	if c == nil {
		return nil
	}

	result := make(map[string]any)

	if c.Enabled != nil {
		result["enabled"] = *c.Enabled
	}
	if len(c.PortMembers) > 0 {
		result["port_members"] = c.PortMembers
	}
	if c.LACPMode != nil {
		result["lacp_mode"] = *c.LACPMode
	}
	if c.HashMode != nil {
		result["hash_mode"] = *c.HashMode
	}

	if len(c.Mist) > 0 {
		for k, v := range c.Mist {
			result[k] = v
		}
	}

	return result
}

// ToMap converts UplinkConfig to a map.
func (c *UplinkConfig) ToMap() map[string]any {
	if c == nil {
		return nil
	}

	result := make(map[string]any)

	if c.Dot1xEnabled != nil {
		result["dot1x_enabled"] = *c.Dot1xEnabled
	}
	if c.Identity != nil {
		result["identity"] = *c.Identity
	}
	if c.Password != nil {
		result["password"] = *c.Password
	}
	if c.Primary != nil {
		result["primary"] = *c.Primary
	}
	if c.Secondary != nil {
		result["secondary"] = *c.Secondary
	}

	if len(c.Mist) > 0 {
		for k, v := range c.Mist {
			result[k] = v
		}
	}

	return result
}

// ToMap converts IoTConfig to a map.
func (c *IoTConfig) ToMap() map[string]any {
	if c == nil {
		return nil
	}

	result := make(map[string]any)

	if c.Enabled != nil {
		result["enabled"] = *c.Enabled
	}
	if c.Type != nil {
		result["type"] = *c.Type
	}

	if len(c.Mist) > 0 {
		for k, v := range c.Mist {
			result[k] = v
		}
	}

	return result
}

// ToMap converts PowerConfig to a map.
func (c *PowerConfig) ToMap() map[string]any {
	if c == nil {
		return nil
	}

	result := make(map[string]any)

	if c.Mode != nil {
		result["mode"] = *c.Mode
	}
	if c.BaseValue != nil {
		result["base_value"] = *c.BaseValue
	}

	if len(c.Mist) > 0 {
		for k, v := range c.Mist {
			result[k] = v
		}
	}

	return result
}

// ToMap converts LEDConfig to a map.
func (c *LEDConfig) ToMap() map[string]any {
	if c == nil {
		return nil
	}

	result := make(map[string]any)

	if c.Enabled != nil {
		result["enabled"] = *c.Enabled
	}
	if c.Brightness != nil {
		result["brightness"] = *c.Brightness
	}

	if len(c.Mist) > 0 {
		for k, v := range c.Mist {
			result[k] = v
		}
	}

	return result
}

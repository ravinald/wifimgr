package api

import (
	"fmt"
)

// device_profile_types.go - Device Profile type definitions
// This file contains all device profile related types and structures

// DeviceProfileAssignResult represents the result of assigning a device profile
type DeviceProfileAssignResult struct {
	// Success contains the list of MAC addresses that were successfully assigned
	Success []string `json:"success,omitempty"`

	// Errors contains any errors that occurred during assignment
	Errors map[string]string `json:"errors,omitempty"`
}

// DeviceProfile represents a Mist device profile with core fields
// This serves as the base structure for all device profile types
type DeviceProfile struct {
	// Core fields present in all device profile types
	// ID is the unique identifier for the device profile
	ID *string `json:"id,omitempty"`

	// Name is the name of the device profile
	Name *string `json:"name,omitempty"`

	// Type is the device type this profile applies to (ap, switch, gateway)
	Type *string `json:"type,omitempty"`

	// OrgID is the organization ID
	OrgID *string `json:"org_id,omitempty"`

	// CreatedTime is when the object has been created, in epoch
	CreatedTime *float64 `json:"created_time,omitempty"`

	// ModifiedTime is when the object has been modified for the last time, in epoch
	ModifiedTime *float64 `json:"modified_time,omitempty"`

	// ForSite indicates if this is a site-level profile
	ForSite *bool `json:"for_site,omitempty"`

	// SiteID is the site ID if this is a site-level profile
	SiteID *string `json:"site_id,omitempty"`

	// Additional configuration for any unmapped fields
	AdditionalConfig map[string]interface{} `json:"-"`
}

// DeviceProfileAP represents the complete AP device profile structure
type DeviceProfileAP struct {
	DeviceProfile

	// AP-specific configuration fields
	Aeroscout      map[string]interface{} `json:"aeroscout,omitempty"`
	BleConfig      map[string]interface{} `json:"ble_config,omitempty"`
	DisableEth1    *bool                  `json:"disable_eth1,omitempty"`
	DisableEth2    *bool                  `json:"disable_eth2,omitempty"`
	DisableEth3    *bool                  `json:"disable_eth3,omitempty"`
	DisableModule  *bool                  `json:"disable_module,omitempty"`
	EslConfig      map[string]interface{} `json:"esl_config,omitempty"`
	Height         *float64               `json:"height,omitempty"`
	IotConfig      map[string]interface{} `json:"iot_config,omitempty"`
	IPConfig       map[string]interface{} `json:"ip_config,omitempty"`
	LacpConfig     map[string]interface{} `json:"lacp_config,omitempty"`
	Led            map[string]interface{} `json:"led,omitempty"`
	MapID          *string                `json:"map_id,omitempty"`
	Mesh           map[string]interface{} `json:"mesh,omitempty"`
	Notes          *string                `json:"notes,omitempty"`
	NtpServers     []string               `json:"ntp_servers,omitempty"`
	Orientation    *int                   `json:"orientation,omitempty"`
	PoePassthrough *bool                  `json:"poe_passthrough,omitempty"`
	X              *float64               `json:"x,omitempty"`
	Y              *float64               `json:"y,omitempty"`

	// Additional AP-specific fields from schema and API response
	RadioConfig      map[string]interface{} `json:"radio_config,omitempty"`
	SwitchConfig     map[string]interface{} `json:"switch_config,omitempty"`
	UsbConfig        map[string]interface{} `json:"usb_config,omitempty"`
	UplinkPortConfig map[string]interface{} `json:"uplink_port_config,omitempty"`
	Vars             map[string]string      `json:"vars,omitempty"`
	PortConfig       map[string]interface{} `json:"port_config,omitempty"`
	PwrConfig        map[string]interface{} `json:"pwr_config,omitempty"`
	Centrak          map[string]interface{} `json:"centrak,omitempty"`

	// Additional fields that may be present
	AdditionalConfig map[string]interface{} `json:"-"`
}

// DeviceProfileGateway represents the complete Gateway device profile structure
type DeviceProfileGateway struct {
	DeviceProfile

	// Gateway-specific configuration fields
	AdditionalConfigCmds []string                          `json:"additional_config_cmds,omitempty"`
	BgpConfig            map[string]map[string]interface{} `json:"bgp_config,omitempty"`
	DhcpdConfig          map[string]interface{}            `json:"dhcpd_config,omitempty"`
	DnsOverride          *bool                             `json:"dnsOverride,omitempty"`
	DnsServers           []string                          `json:"dns_servers,omitempty"`
	DnsSuffix            []string                          `json:"dns_suffix,omitempty"`
	ExtraRoutes          map[string]map[string]interface{} `json:"extra_routes,omitempty"`
	ExtraRoutes6         map[string]map[string]interface{} `json:"extra_routes6,omitempty"`
	GatewayMatching      map[string]interface{}            `json:"gateway_matching,omitempty"`
	IdpProfiles          map[string]map[string]interface{} `json:"idp_profiles,omitempty"`
	IPConfigs            map[string]map[string]interface{} `json:"ip_configs,omitempty"`
	NtpOverride          *bool                             `json:"ntpOverride,omitempty"`
	NtpServers           []string                          `json:"ntp_servers,omitempty"`
	PathPreferences      map[string]map[string]interface{} `json:"path_preferences,omitempty"`
	PortConfig           map[string]map[string]interface{} `json:"port_config,omitempty"`
	ServicePolicies      []map[string]interface{}          `json:"service_policies,omitempty"`

	// Additional Gateway-specific fields from schema
	Networks              []map[string]interface{}          `json:"networks,omitempty"`
	OobIPConfig           map[string]interface{}            `json:"oob_ip_config,omitempty"`
	RouterID              *string                           `json:"router_id,omitempty"`
	RoutingPolicies       map[string]map[string]interface{} `json:"routing_policies,omitempty"`
	TunnelConfigs         map[string]map[string]interface{} `json:"tunnel_configs,omitempty"`
	TunnelProviderOptions map[string]interface{}            `json:"tunnel_provider_options,omitempty"`
	VrfConfig             map[string]interface{}            `json:"vrf_config,omitempty"`
	VrfInstances          map[string]map[string]interface{} `json:"vrf_instances,omitempty"`

	// Additional fields that may be present
	AdditionalConfig map[string]interface{} `json:"-"`
}

// DeviceProfileSwitch represents the complete Switch device profile structure
type DeviceProfileSwitch struct {
	DeviceProfile

	// Switch-specific configuration fields
	AclPolicies           []map[string]interface{}          `json:"acl_policies,omitempty"`
	AclTags               map[string]map[string]interface{} `json:"acl_tags,omitempty"`
	AdditionalConfigCmds  []string                          `json:"additional_config_cmds,omitempty"`
	AggregateRoutes       map[string]map[string]interface{} `json:"aggregate_routes,omitempty"`
	AggregateRoutes6      map[string]map[string]interface{} `json:"aggregate_routes6,omitempty"`
	DhcpSnooping          map[string]interface{}            `json:"dhcp_snooping,omitempty"`
	DhcpdConfig           map[string]interface{}            `json:"dhcpd_config,omitempty"`
	DnsServers            []string                          `json:"dns_servers,omitempty"`
	DnsSuffix             []string                          `json:"dns_suffix,omitempty"`
	EvpnConfig            map[string]interface{}            `json:"evpn_config,omitempty"`
	ExtraRoutes           map[string]map[string]interface{} `json:"extra_routes,omitempty"`
	ExtraRoutes6          map[string]map[string]interface{} `json:"extra_routes6,omitempty"`
	IPConfig              map[string]interface{}            `json:"ip_config,omitempty"`
	IotConfig             map[string]map[string]interface{} `json:"iot_config,omitempty"`
	MistNac               map[string]interface{}            `json:"mist_nac,omitempty"`
	Networks              map[string]map[string]interface{} `json:"networks,omitempty"`
	NtpServers            []string                          `json:"ntp_servers,omitempty"`
	OobIPConfig           map[string]interface{}            `json:"oob_ip_config,omitempty"`
	OspfAreas             map[string]map[string]interface{} `json:"ospf_areas,omitempty"`
	OtherIPConfigs        map[string]map[string]interface{} `json:"other_ip_configs,omitempty"`
	PortConfig            map[string]map[string]interface{} `json:"port_config,omitempty"`
	PortMirroring         map[string]map[string]interface{} `json:"port_mirroring,omitempty"`
	PortUsages            map[string]map[string]interface{} `json:"port_usages,omitempty"`
	RadiusConfig          map[string]interface{}            `json:"radius_config,omitempty"`
	RemoteSyslog          map[string]interface{}            `json:"remote_syslog,omitempty"`
	SnmpConfig            map[string]interface{}            `json:"snmp_config,omitempty"`
	StpConfig             map[string]interface{}            `json:"stp_config,omitempty"`
	SwitchMgmt            map[string]interface{}            `json:"switch_mgmt,omitempty"`
	UseRouterIDAsSourceIP *bool                             `json:"use_router_id_as_source_ip,omitempty"`
	VrfConfig             map[string]interface{}            `json:"vrf_config,omitempty"`
	VrfInstances          map[string]map[string]interface{} `json:"vrf_instances,omitempty"`
	VrrpConfig            map[string]interface{}            `json:"vrrp_config,omitempty"`

	// Additional fields that may be present
	AdditionalConfig map[string]interface{} `json:"-"`
}

// DeviceProfileMarshaler defines the common interface for bidirectional device profile
// data transformation between API representations and structured types.
type DeviceProfileMarshaler interface {
	GetID() *string
	GetName() *string
	GetType() *string
	GetOrgID() *string
	GetCreatedTime() *float64
	GetModifiedTime() *float64
	GetForSite() *bool
	GetSiteID() *string
	ToMap() map[string]interface{}
	FromMap(data map[string]interface{}) error
}

// Implement the interface for the base DeviceProfile
func (dp *DeviceProfile) GetID() *string {
	return dp.ID
}

func (dp *DeviceProfile) GetName() *string {
	return dp.Name
}

func (dp *DeviceProfile) GetType() *string {
	return dp.Type
}

func (dp *DeviceProfile) GetOrgID() *string {
	return dp.OrgID
}

func (dp *DeviceProfile) GetCreatedTime() *float64 {
	return dp.CreatedTime
}

func (dp *DeviceProfile) GetModifiedTime() *float64 {
	return dp.ModifiedTime
}

func (dp *DeviceProfile) GetForSite() *bool {
	return dp.ForSite
}

func (dp *DeviceProfile) GetSiteID() *string {
	return dp.SiteID
}

// ToMap converts the device profile to a map for API operations
func (dp *DeviceProfile) ToMap() map[string]interface{} {
	result := make(map[string]interface{})

	// Add typed fields
	if dp.ID != nil {
		result["id"] = *dp.ID
	}
	if dp.Name != nil {
		result["name"] = *dp.Name
	}
	if dp.Type != nil {
		result["type"] = *dp.Type
	}
	if dp.OrgID != nil {
		result["org_id"] = *dp.OrgID
	}
	if dp.CreatedTime != nil {
		result["created_time"] = *dp.CreatedTime
	}
	if dp.ModifiedTime != nil {
		result["modified_time"] = *dp.ModifiedTime
	}
	if dp.ForSite != nil {
		result["for_site"] = *dp.ForSite
	}
	if dp.SiteID != nil {
		result["site_id"] = *dp.SiteID
	}

	// Add additional configuration
	for key, value := range dp.AdditionalConfig {
		result[key] = value
	}

	return result
}

// FromMap populates the device profile from a map
func (dp *DeviceProfile) FromMap(data map[string]interface{}) error {
	if data == nil {
		return fmt.Errorf("data cannot be nil")
	}

	// Initialize additional config if nil
	if dp.AdditionalConfig == nil {
		dp.AdditionalConfig = make(map[string]interface{})
	}

	// Extract known fields
	if id, ok := data["id"].(string); ok {
		dp.ID = &id
	}
	if name, ok := data["name"].(string); ok {
		dp.Name = &name
	}
	if profileType, ok := data["type"].(string); ok {
		dp.Type = &profileType
	}
	if orgID, ok := data["org_id"].(string); ok {
		dp.OrgID = &orgID
	}
	if createdTime, ok := data["created_time"].(float64); ok {
		dp.CreatedTime = &createdTime
	}
	if modifiedTime, ok := data["modified_time"].(float64); ok {
		dp.ModifiedTime = &modifiedTime
	}
	if forSite, ok := data["for_site"].(bool); ok {
		dp.ForSite = &forSite
	}
	if siteID, ok := data["site_id"].(string); ok {
		dp.SiteID = &siteID
	}

	// Store any unknown fields in AdditionalConfig (following APDevice pattern)
	knownFields := map[string]bool{
		"id": true, "name": true, "type": true, "org_id": true,
		"created_time": true, "modified_time": true, "for_site": true, "site_id": true,
	}

	for k, v := range data {
		if !knownFields[k] {
			dp.AdditionalConfig[k] = v
		}
	}

	return nil
}

// ToMap converts the DeviceProfileAP to a map
func (dp *DeviceProfileAP) ToMap() map[string]interface{} {
	// Start with the base profile map
	result := dp.DeviceProfile.ToMap()

	// Add AP-specific fields
	if dp.Aeroscout != nil {
		result["aeroscout"] = dp.Aeroscout
	}
	if dp.BleConfig != nil {
		result["ble_config"] = dp.BleConfig
	}
	if dp.DisableEth1 != nil {
		result["disable_eth1"] = *dp.DisableEth1
	}
	if dp.DisableEth2 != nil {
		result["disable_eth2"] = *dp.DisableEth2
	}
	if dp.DisableEth3 != nil {
		result["disable_eth3"] = *dp.DisableEth3
	}
	if dp.DisableModule != nil {
		result["disable_module"] = *dp.DisableModule
	}
	if dp.EslConfig != nil {
		result["esl_config"] = dp.EslConfig
	}
	if dp.Height != nil {
		result["height"] = *dp.Height
	}
	if dp.IotConfig != nil {
		result["iot_config"] = dp.IotConfig
	}
	if dp.IPConfig != nil {
		result["ip_config"] = dp.IPConfig
	}
	if dp.LacpConfig != nil {
		result["lacp_config"] = dp.LacpConfig
	}
	if dp.Led != nil {
		result["led"] = dp.Led
	}
	if dp.MapID != nil {
		result["map_id"] = *dp.MapID
	}
	if dp.Mesh != nil {
		result["mesh"] = dp.Mesh
	}
	if dp.Notes != nil {
		result["notes"] = *dp.Notes
	}
	if len(dp.NtpServers) > 0 {
		result["ntp_servers"] = dp.NtpServers
	}
	if dp.Orientation != nil {
		result["orientation"] = *dp.Orientation
	}
	if dp.PoePassthrough != nil {
		result["poe_passthrough"] = *dp.PoePassthrough
	}
	if dp.X != nil {
		result["x"] = *dp.X
	}
	if dp.Y != nil {
		result["y"] = *dp.Y
	}
	if dp.RadioConfig != nil {
		result["radio_config"] = dp.RadioConfig
	}
	if dp.SwitchConfig != nil {
		result["switch_config"] = dp.SwitchConfig
	}
	if dp.UsbConfig != nil {
		result["usb_config"] = dp.UsbConfig
	}
	if dp.UplinkPortConfig != nil {
		result["uplink_port_config"] = dp.UplinkPortConfig
	}
	if len(dp.Vars) > 0 {
		result["vars"] = dp.Vars
	}
	if dp.PortConfig != nil {
		result["port_config"] = dp.PortConfig
	}
	if dp.PwrConfig != nil {
		result["pwr_config"] = dp.PwrConfig
	}
	if dp.Centrak != nil {
		result["centrak"] = dp.Centrak
	}

	return result
}

// FromMap populates the DeviceProfileAP from a map
func (dp *DeviceProfileAP) FromMap(data map[string]interface{}) error {
	// First populate the base profile
	if err := dp.DeviceProfile.FromMap(data); err != nil {
		return err
	}

	// Clear the base additional config since we'll handle all fields
	dp.DeviceProfile.AdditionalConfig = make(map[string]interface{})
	if dp.AdditionalConfig == nil {
		dp.AdditionalConfig = make(map[string]interface{})
	}

	// Extract AP-specific fields
	if aeroscout, ok := data["aeroscout"].(map[string]interface{}); ok {
		dp.Aeroscout = aeroscout
	}
	if bleConfig, ok := data["ble_config"].(map[string]interface{}); ok {
		dp.BleConfig = bleConfig
	}
	if disableEth1, ok := data["disable_eth1"].(bool); ok {
		dp.DisableEth1 = &disableEth1
	}
	if disableEth2, ok := data["disable_eth2"].(bool); ok {
		dp.DisableEth2 = &disableEth2
	}
	if disableEth3, ok := data["disable_eth3"].(bool); ok {
		dp.DisableEth3 = &disableEth3
	}
	if disableModule, ok := data["disable_module"].(bool); ok {
		dp.DisableModule = &disableModule
	}
	if eslConfig, ok := data["esl_config"].(map[string]interface{}); ok {
		dp.EslConfig = eslConfig
	}
	if height, ok := data["height"].(float64); ok {
		dp.Height = &height
	}
	if iotConfig, ok := data["iot_config"].(map[string]interface{}); ok {
		dp.IotConfig = iotConfig
	}
	if ipConfig, ok := data["ip_config"].(map[string]interface{}); ok {
		dp.IPConfig = ipConfig
	}
	if lacpConfig, ok := data["lacp_config"].(map[string]interface{}); ok {
		dp.LacpConfig = lacpConfig
	}
	if led, ok := data["led"].(map[string]interface{}); ok {
		dp.Led = led
	}
	if mapID, ok := data["map_id"].(string); ok {
		dp.MapID = &mapID
	}
	if mesh, ok := data["mesh"].(map[string]interface{}); ok {
		dp.Mesh = mesh
	}
	if notes, ok := data["notes"].(string); ok {
		dp.Notes = &notes
	}
	if ntpServers, ok := data["ntp_servers"].([]interface{}); ok {
		dp.NtpServers = make([]string, 0, len(ntpServers))
		for _, server := range ntpServers {
			if s, ok := server.(string); ok {
				dp.NtpServers = append(dp.NtpServers, s)
			}
		}
	}
	if orientation, ok := data["orientation"].(float64); ok {
		o := int(orientation)
		dp.Orientation = &o
	}
	if poePassthrough, ok := data["poe_passthrough"].(bool); ok {
		dp.PoePassthrough = &poePassthrough
	}
	if x, ok := data["x"].(float64); ok {
		dp.X = &x
	}
	if y, ok := data["y"].(float64); ok {
		dp.Y = &y
	}
	if radioConfig, ok := data["radio_config"].(map[string]interface{}); ok {
		dp.RadioConfig = radioConfig
	}
	if switchConfig, ok := data["switch_config"].(map[string]interface{}); ok {
		dp.SwitchConfig = switchConfig
	}
	if usbConfig, ok := data["usb_config"].(map[string]interface{}); ok {
		dp.UsbConfig = usbConfig
	}
	if uplinkPortConfig, ok := data["uplink_port_config"].(map[string]interface{}); ok {
		dp.UplinkPortConfig = uplinkPortConfig
	}
	if vars, ok := data["vars"].(map[string]interface{}); ok {
		dp.Vars = make(map[string]string)
		for k, v := range vars {
			if s, ok := v.(string); ok {
				dp.Vars[k] = s
			}
		}
	}
	if portConfig, ok := data["port_config"].(map[string]interface{}); ok {
		dp.PortConfig = portConfig
	}
	if pwrConfig, ok := data["pwr_config"].(map[string]interface{}); ok {
		dp.PwrConfig = pwrConfig
	}
	if centrak, ok := data["centrak"].(map[string]interface{}); ok {
		dp.Centrak = centrak
	}

	// Store any unknown fields in AdditionalConfig
	knownFields := map[string]bool{
		"id": true, "name": true, "type": true, "org_id": true,
		"created_time": true, "modified_time": true, "for_site": true, "site_id": true,
		"aeroscout": true, "ble_config": true, "disable_eth1": true, "disable_eth2": true,
		"disable_eth3": true, "disable_module": true, "esl_config": true, "height": true,
		"iot_config": true, "ip_config": true, "lacp_config": true, "led": true,
		"map_id": true, "mesh": true, "notes": true, "ntp_servers": true,
		"orientation": true, "poe_passthrough": true, "x": true, "y": true,
		"radio_config": true, "switch_config": true, "usb_config": true,
		"uplink_port_config": true, "vars": true, "port_config": true,
		"pwr_config": true, "centrak": true,
	}

	for k, v := range data {
		if !knownFields[k] {
			dp.AdditionalConfig[k] = v
		}
	}

	return nil
}

// ToMap converts the DeviceProfileGateway to a map
func (dp *DeviceProfileGateway) ToMap() map[string]interface{} {
	// Start with the base profile map
	result := dp.DeviceProfile.ToMap()

	// Add Gateway-specific fields
	// TODO: Implement gateway-specific fields similar to AP

	return result
}

// FromMap populates the DeviceProfileGateway from a map
func (dp *DeviceProfileGateway) FromMap(data map[string]interface{}) error {
	// First populate the base profile
	if err := dp.DeviceProfile.FromMap(data); err != nil {
		return err
	}

	// TODO: Implement gateway-specific field extraction similar to AP

	return nil
}

// ToMap converts the DeviceProfileSwitch to a map
func (dp *DeviceProfileSwitch) ToMap() map[string]interface{} {
	// Start with the base profile map
	result := dp.DeviceProfile.ToMap()

	// Add Switch-specific fields
	// TODO: Implement switch-specific fields similar to AP

	return result
}

// FromMap populates the DeviceProfileSwitch from a map
func (dp *DeviceProfileSwitch) FromMap(data map[string]interface{}) error {
	// First populate the base profile
	if err := dp.DeviceProfile.FromMap(data); err != nil {
		return err
	}

	// TODO: Implement switch-specific field extraction similar to AP

	return nil
}

// NewDeviceProfileFromType creates a specific device profile type based on the type string
func NewDeviceProfileFromType(profileType string) DeviceProfileMarshaler {
	switch profileType {
	case "ap":
		return &DeviceProfileAP{}
	case "gateway":
		return &DeviceProfileGateway{}
	case "switch":
		return &DeviceProfileSwitch{}
	default:
		return &DeviceProfile{}
	}
}

// Verify DeviceProfile types implement DeviceProfileMarshaler at compile time
var (
	_ DeviceProfileMarshaler = (*DeviceProfile)(nil)
	_ DeviceProfileMarshaler = (*DeviceProfileAP)(nil)
	_ DeviceProfileMarshaler = (*DeviceProfileGateway)(nil)
	_ DeviceProfileMarshaler = (*DeviceProfileSwitch)(nil)
)

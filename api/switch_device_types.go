package api

import (
	"fmt"
)

// MistSwitchDevice represents a Mist switch with complete data preservation
type MistSwitchDevice struct {
	BaseDevice

	// Switch-specific typed fields for frequently used configurations
	IP       *string `json:"ip,omitempty"`
	Port     *int    `json:"port,omitempty"`
	Version  *string `json:"version,omitempty"`
	Role     *string `json:"role,omitempty"`
	Managed  *bool   `json:"managed,omitempty"`
	RouterID *string `json:"router_id,omitempty"`

	// Complex configuration stored as maps for flexibility and evolution
	PortConfig     map[string]interface{} `json:"port_config,omitempty"`
	Networks       map[string]interface{} `json:"networks,omitempty"`
	IPConfig       map[string]interface{} `json:"ip_config,omitempty"`
	OobIPConfig    map[string]interface{} `json:"oob_ip_config,omitempty"`
	StpConfig      map[string]interface{} `json:"stp_config,omitempty"`
	VlanConfig     map[string]interface{} `json:"vlan_config,omitempty"`
	L2Config       map[string]interface{} `json:"l2_config,omitempty"`
	L3Config       map[string]interface{} `json:"l3_config,omitempty"`
	RoutingConfig  map[string]interface{} `json:"routing_config,omitempty"`
	SecurityConfig map[string]interface{} `json:"security_config,omitempty"`
	QosConfig      map[string]interface{} `json:"qos_config,omitempty"`
	SnmpConfig     map[string]interface{} `json:"snmp_config,omitempty"`
	SyslogConfig   map[string]interface{} `json:"syslog_config,omitempty"`
	NtpConfig      map[string]interface{} `json:"ntp_config,omitempty"`
	DnsConfig      map[string]interface{} `json:"dns_config,omitempty"`

	// Additional config for unknown fields
	AdditionalConfig map[string]interface{} `json:"-"`
}

// SwitchDeviceInterface defines the interface for switch device operations.
// Note: The "Interface" suffix is retained because there is a struct named MistSwitchDevice.
type SwitchDeviceInterface interface {
	DeviceMarshaler
	GetIP() *string
	GetPort() *int
	GetVersion() *string
	GetRole() *string
	GetManaged() *bool
	GetRouterID() *string
	GetPortConfig() map[string]interface{}
	GetNetworks() map[string]interface{}
	GetIPConfig() map[string]interface{}
}

// Implement SwitchDeviceInterface
func (sw *MistSwitchDevice) GetIP() *string                        { return sw.IP }
func (sw *MistSwitchDevice) GetPort() *int                         { return sw.Port }
func (sw *MistSwitchDevice) GetVersion() *string                   { return sw.Version }
func (sw *MistSwitchDevice) GetRole() *string                      { return sw.Role }
func (sw *MistSwitchDevice) GetManaged() *bool                     { return sw.Managed }
func (sw *MistSwitchDevice) GetRouterID() *string                  { return sw.RouterID }
func (sw *MistSwitchDevice) GetPortConfig() map[string]interface{} { return sw.PortConfig }
func (sw *MistSwitchDevice) GetNetworks() map[string]interface{}   { return sw.Networks }
func (sw *MistSwitchDevice) GetIPConfig() map[string]interface{}   { return sw.IPConfig }

// FromMap populates the MistSwitchDevice from API response data
func (sw *MistSwitchDevice) FromMap(data map[string]interface{}) error {
	// First populate base device fields
	if err := sw.BaseDevice.FromMap(data); err != nil {
		return fmt.Errorf("failed to populate base device fields: %w", err)
	}

	// Parse switch-specific typed fields
	if ip, ok := data["ip"].(string); ok {
		sw.IP = &ip
	}

	if port, ok := data["port"].(float64); ok {
		portInt := int(port)
		sw.Port = &portInt
	}

	if version, ok := data["version"].(string); ok {
		sw.Version = &version
	}

	if role, ok := data["role"].(string); ok {
		sw.Role = &role
	}

	if managed, ok := data["managed"].(bool); ok {
		sw.Managed = &managed
	}

	if routerID, ok := data["router_id"].(string); ok {
		sw.RouterID = &routerID
	}

	// Parse complex configuration objects
	configFields := map[string]*map[string]interface{}{
		"port_config":     &sw.PortConfig,
		"networks":        &sw.Networks,
		"ip_config":       &sw.IPConfig,
		"oob_ip_config":   &sw.OobIPConfig,
		"stp_config":      &sw.StpConfig,
		"vlan_config":     &sw.VlanConfig,
		"l2_config":       &sw.L2Config,
		"l3_config":       &sw.L3Config,
		"routing_config":  &sw.RoutingConfig,
		"security_config": &sw.SecurityConfig,
		"qos_config":      &sw.QosConfig,
		"snmp_config":     &sw.SnmpConfig,
		"syslog_config":   &sw.SyslogConfig,
		"ntp_config":      &sw.NtpConfig,
		"dns_config":      &sw.DnsConfig,
	}

	for fieldName, configPtr := range configFields {
		if configData, ok := data[fieldName].(map[string]interface{}); ok {
			*configPtr = configData
		}
	}

	// Store any unknown fields in AdditionalConfig
	sw.AdditionalConfig = make(map[string]interface{})
	knownFields := map[string]bool{
		// Base device fields
		"id": true, "mac": true, "serial": true, "name": true, "model": true, "type": true,
		"magic": true, "hw_rev": true, "sku": true, "site_id": true, "org_id": true,
		"created_time": true, "modified_time": true, "deviceprofile_id": true,
		"connected": true, "adopted": true, "hostname": true, "notes": true, "jsi": true, "tags": true,

		// Switch-specific typed fields
		"ip": true, "port": true, "version": true, "role": true, "managed": true, "router_id": true,

		// Switch complex config fields
		"port_config": true, "networks": true, "ip_config": true, "oob_ip_config": true,
		"stp_config": true, "vlan_config": true, "l2_config": true, "l3_config": true,
		"routing_config": true, "security_config": true, "qos_config": true, "snmp_config": true,
		"syslog_config": true, "ntp_config": true, "dns_config": true,
	}

	for k, v := range data {
		if !knownFields[k] {
			sw.AdditionalConfig[k] = v
		}
	}

	return nil
}

// ToMap converts the MistSwitchDevice to a map for API operations
func (sw *MistSwitchDevice) ToMap() map[string]interface{} {
	// Start with base device fields
	result := sw.BaseDevice.ToMap()

	// Add switch-specific typed fields
	if sw.IP != nil {
		result["ip"] = *sw.IP
	}
	if sw.Port != nil {
		result["port"] = *sw.Port
	}
	if sw.Version != nil {
		result["version"] = *sw.Version
	}
	if sw.Role != nil {
		result["role"] = *sw.Role
	}
	if sw.Managed != nil {
		result["managed"] = *sw.Managed
	}
	if sw.RouterID != nil {
		result["router_id"] = *sw.RouterID
	}

	// Add complex configuration objects
	if sw.PortConfig != nil {
		result["port_config"] = sw.PortConfig
	}
	if sw.Networks != nil {
		result["networks"] = sw.Networks
	}
	if sw.IPConfig != nil {
		result["ip_config"] = sw.IPConfig
	}
	if sw.OobIPConfig != nil {
		result["oob_ip_config"] = sw.OobIPConfig
	}
	if sw.StpConfig != nil {
		result["stp_config"] = sw.StpConfig
	}
	if sw.VlanConfig != nil {
		result["vlan_config"] = sw.VlanConfig
	}
	if sw.L2Config != nil {
		result["l2_config"] = sw.L2Config
	}
	if sw.L3Config != nil {
		result["l3_config"] = sw.L3Config
	}
	if sw.RoutingConfig != nil {
		result["routing_config"] = sw.RoutingConfig
	}
	if sw.SecurityConfig != nil {
		result["security_config"] = sw.SecurityConfig
	}
	if sw.QosConfig != nil {
		result["qos_config"] = sw.QosConfig
	}
	if sw.SnmpConfig != nil {
		result["snmp_config"] = sw.SnmpConfig
	}
	if sw.SyslogConfig != nil {
		result["syslog_config"] = sw.SyslogConfig
	}
	if sw.NtpConfig != nil {
		result["ntp_config"] = sw.NtpConfig
	}
	if sw.DnsConfig != nil {
		result["dns_config"] = sw.DnsConfig
	}

	// Add additional unknown fields
	for k, v := range sw.AdditionalConfig {
		if _, exists := result[k]; !exists {
			result[k] = v
		}
	}

	return result
}

// ToConfigMap converts switch device to configuration map (for config files)
func (sw *MistSwitchDevice) ToConfigMap() map[string]interface{} {
	// Start with base configuration fields
	result := sw.BaseDevice.ToConfigMap()

	// Add switch-specific configuration fields (exclude runtime/status fields)
	if sw.Role != nil {
		result["role"] = *sw.Role
	}
	if sw.Managed != nil {
		result["managed"] = *sw.Managed
	}
	if sw.RouterID != nil {
		result["router_id"] = *sw.RouterID
	}

	// Add configuration objects (exclude runtime status)
	if sw.PortConfig != nil {
		result["port_config"] = sw.PortConfig
	}
	if sw.Networks != nil {
		result["networks"] = sw.Networks
	}
	if sw.IPConfig != nil {
		result["ip_config"] = sw.IPConfig
	}
	if sw.OobIPConfig != nil {
		result["oob_ip_config"] = sw.OobIPConfig
	}
	if sw.StpConfig != nil {
		result["stp_config"] = sw.StpConfig
	}
	if sw.VlanConfig != nil {
		result["vlan_config"] = sw.VlanConfig
	}
	if sw.L2Config != nil {
		result["l2_config"] = sw.L2Config
	}
	if sw.L3Config != nil {
		result["l3_config"] = sw.L3Config
	}
	if sw.RoutingConfig != nil {
		result["routing_config"] = sw.RoutingConfig
	}
	if sw.SecurityConfig != nil {
		result["security_config"] = sw.SecurityConfig
	}
	if sw.QosConfig != nil {
		result["qos_config"] = sw.QosConfig
	}
	if sw.SnmpConfig != nil {
		result["snmp_config"] = sw.SnmpConfig
	}
	if sw.SyslogConfig != nil {
		result["syslog_config"] = sw.SyslogConfig
	}
	if sw.NtpConfig != nil {
		result["ntp_config"] = sw.NtpConfig
	}
	if sw.DnsConfig != nil {
		result["dns_config"] = sw.DnsConfig
	}

	// Add configuration fields from AdditionalConfig, filtering out status fields
	statusFields := map[string]bool{
		"connected": true, "adopted": true, "last_seen": true, "uptime": true,
		"version": true, "ip": true, "port": true, "status": true, "hw_rev": true, "sku": true,
	}

	for k, v := range sw.AdditionalConfig {
		if !statusFields[k] {
			if _, exists := result[k]; !exists {
				result[k] = v
			}
		}
	}

	return result
}

// FromConfigMap populates switch device from configuration map (from config files)
func (sw *MistSwitchDevice) FromConfigMap(data map[string]interface{}) error {
	// First populate base device configuration
	if err := sw.BaseDevice.FromConfigMap(data); err != nil {
		return fmt.Errorf("failed to populate base device config: %w", err)
	}

	// Parse switch-specific configuration fields (exclude runtime fields)
	if role, ok := data["role"].(string); ok {
		sw.Role = &role
	}

	if managed, ok := data["managed"].(bool); ok {
		sw.Managed = &managed
	}

	if routerID, ok := data["router_id"].(string); ok {
		sw.RouterID = &routerID
	}

	// Parse complex configuration objects
	configFields := map[string]*map[string]interface{}{
		"port_config":     &sw.PortConfig,
		"networks":        &sw.Networks,
		"ip_config":       &sw.IPConfig,
		"oob_ip_config":   &sw.OobIPConfig,
		"stp_config":      &sw.StpConfig,
		"vlan_config":     &sw.VlanConfig,
		"l2_config":       &sw.L2Config,
		"l3_config":       &sw.L3Config,
		"routing_config":  &sw.RoutingConfig,
		"security_config": &sw.SecurityConfig,
		"qos_config":      &sw.QosConfig,
		"snmp_config":     &sw.SnmpConfig,
		"syslog_config":   &sw.SyslogConfig,
		"ntp_config":      &sw.NtpConfig,
		"dns_config":      &sw.DnsConfig,
	}

	for fieldName, configPtr := range configFields {
		if configData, ok := data[fieldName].(map[string]interface{}); ok {
			*configPtr = configData
		}
	}

	// Store any unknown configuration fields in AdditionalConfig
	sw.AdditionalConfig = make(map[string]interface{})
	knownFields := map[string]bool{
		// Base device config fields
		"name": true, "magic": true, "deviceprofile_id": true, "notes": true, "tags": true,

		// Switch-specific config fields
		"role": true, "managed": true, "router_id": true,

		// Switch complex config fields
		"port_config": true, "networks": true, "ip_config": true, "oob_ip_config": true,
		"stp_config": true, "vlan_config": true, "l2_config": true, "l3_config": true,
		"routing_config": true, "security_config": true, "qos_config": true, "snmp_config": true,
		"syslog_config": true, "ntp_config": true, "dns_config": true,
	}

	for k, v := range data {
		if !knownFields[k] {
			sw.AdditionalConfig[k] = v
		}
	}

	return nil
}

// NewSwitchDeviceFromMap creates a new MistSwitchDevice from API response data
func NewSwitchDeviceFromMap(data map[string]interface{}) (*MistSwitchDevice, error) {
	device := &MistSwitchDevice{}
	if err := device.FromMap(data); err != nil {
		return nil, fmt.Errorf("failed to create switch device from map: %w", err)
	}
	return device, nil
}

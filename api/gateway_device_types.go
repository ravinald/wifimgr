package api

import (
	"fmt"
)

// MistGatewayDevice represents a Mist gateway with complete data preservation
type MistGatewayDevice struct {
	BaseDevice

	// Gateway-specific typed fields for frequently used configurations
	MgmtIntf *string `json:"mgmt_intf,omitempty"`
	Version  *string `json:"version,omitempty"`
	IP       *string `json:"ip,omitempty"`

	// Complex configuration stored as maps for flexibility and evolution
	WANConfig      map[string]interface{} `json:"wan_config,omitempty"`
	LANConfig      map[string]interface{} `json:"lan_config,omitempty"`
	SecurityConfig map[string]interface{} `json:"security_config,omitempty"`
	VPNConfig      map[string]interface{} `json:"vpn_config,omitempty"`
	RoutingConfig  map[string]interface{} `json:"routing_config,omitempty"`
	FirewallConfig map[string]interface{} `json:"firewall_config,omitempty"`
	NATConfig      map[string]interface{} `json:"nat_config,omitempty"`
	QosConfig      map[string]interface{} `json:"qos_config,omitempty"`
	DhcpConfig     map[string]interface{} `json:"dhcp_config,omitempty"`
	DnsConfig      map[string]interface{} `json:"dns_config,omitempty"`
	NtpConfig      map[string]interface{} `json:"ntp_config,omitempty"`
	SnmpConfig     map[string]interface{} `json:"snmp_config,omitempty"`
	SyslogConfig   map[string]interface{} `json:"syslog_config,omitempty"`
	TunnelConfig   map[string]interface{} `json:"tunnel_config,omitempty"`
	ClusterConfig  map[string]interface{} `json:"cluster_config,omitempty"`
	PortConfig     map[string]interface{} `json:"port_config,omitempty"`

	// Additional config for unknown fields
	AdditionalConfig map[string]interface{} `json:"-"`
}

// GatewayDeviceInterface defines the interface for gateway device operations.
// Note: The "Interface" suffix is retained because there is a struct named MistGatewayDevice.
type GatewayDeviceInterface interface {
	DeviceMarshaler
	GetMgmtIntf() *string
	GetVersion() *string
	GetIP() *string
	GetWANConfig() map[string]interface{}
	GetLANConfig() map[string]interface{}
	GetSecurityConfig() map[string]interface{}
	GetVPNConfig() map[string]interface{}
}

// Implement GatewayDeviceInterface
func (gw *MistGatewayDevice) GetMgmtIntf() *string                      { return gw.MgmtIntf }
func (gw *MistGatewayDevice) GetVersion() *string                       { return gw.Version }
func (gw *MistGatewayDevice) GetIP() *string                            { return gw.IP }
func (gw *MistGatewayDevice) GetWANConfig() map[string]interface{}      { return gw.WANConfig }
func (gw *MistGatewayDevice) GetLANConfig() map[string]interface{}      { return gw.LANConfig }
func (gw *MistGatewayDevice) GetSecurityConfig() map[string]interface{} { return gw.SecurityConfig }
func (gw *MistGatewayDevice) GetVPNConfig() map[string]interface{}      { return gw.VPNConfig }

// FromMap populates the MistGatewayDevice from API response data
func (gw *MistGatewayDevice) FromMap(data map[string]interface{}) error {
	// First populate base device fields
	if err := gw.BaseDevice.FromMap(data); err != nil {
		return fmt.Errorf("failed to populate base device fields: %w", err)
	}

	// Parse gateway-specific typed fields
	if mgmtIntf, ok := data["mgmt_intf"].(string); ok {
		gw.MgmtIntf = &mgmtIntf
	}

	if version, ok := data["version"].(string); ok {
		gw.Version = &version
	}

	if ip, ok := data["ip"].(string); ok {
		gw.IP = &ip
	}

	// Parse complex configuration objects
	configFields := map[string]*map[string]interface{}{
		"wan_config":      &gw.WANConfig,
		"lan_config":      &gw.LANConfig,
		"security_config": &gw.SecurityConfig,
		"vpn_config":      &gw.VPNConfig,
		"routing_config":  &gw.RoutingConfig,
		"firewall_config": &gw.FirewallConfig,
		"nat_config":      &gw.NATConfig,
		"qos_config":      &gw.QosConfig,
		"dhcp_config":     &gw.DhcpConfig,
		"dns_config":      &gw.DnsConfig,
		"ntp_config":      &gw.NtpConfig,
		"snmp_config":     &gw.SnmpConfig,
		"syslog_config":   &gw.SyslogConfig,
		"tunnel_config":   &gw.TunnelConfig,
		"cluster_config":  &gw.ClusterConfig,
		"port_config":     &gw.PortConfig,
	}

	for fieldName, configPtr := range configFields {
		if configData, ok := data[fieldName].(map[string]interface{}); ok {
			*configPtr = configData
		}
	}

	// Store any unknown fields in AdditionalConfig
	gw.AdditionalConfig = make(map[string]interface{})
	knownFields := map[string]bool{
		// Base device fields
		"id": true, "mac": true, "serial": true, "name": true, "model": true, "type": true,
		"magic": true, "hw_rev": true, "sku": true, "site_id": true, "org_id": true,
		"created_time": true, "modified_time": true, "deviceprofile_id": true,
		"connected": true, "adopted": true, "hostname": true, "notes": true, "jsi": true, "tags": true,

		// Gateway-specific typed fields
		"mgmt_intf": true, "version": true, "ip": true,

		// Gateway complex config fields
		"wan_config": true, "lan_config": true, "security_config": true, "vpn_config": true,
		"routing_config": true, "firewall_config": true, "nat_config": true, "qos_config": true,
		"dhcp_config": true, "dns_config": true, "ntp_config": true, "snmp_config": true,
		"syslog_config": true, "tunnel_config": true, "cluster_config": true, "port_config": true,
	}

	for k, v := range data {
		if !knownFields[k] {
			gw.AdditionalConfig[k] = v
		}
	}

	return nil
}

// ToMap converts the MistGatewayDevice to a map for API operations
func (gw *MistGatewayDevice) ToMap() map[string]interface{} {
	// Start with base device fields
	result := gw.BaseDevice.ToMap()

	// Add gateway-specific typed fields
	if gw.MgmtIntf != nil {
		result["mgmt_intf"] = *gw.MgmtIntf
	}
	if gw.Version != nil {
		result["version"] = *gw.Version
	}
	if gw.IP != nil {
		result["ip"] = *gw.IP
	}

	// Add complex configuration objects
	if gw.WANConfig != nil {
		result["wan_config"] = gw.WANConfig
	}
	if gw.LANConfig != nil {
		result["lan_config"] = gw.LANConfig
	}
	if gw.SecurityConfig != nil {
		result["security_config"] = gw.SecurityConfig
	}
	if gw.VPNConfig != nil {
		result["vpn_config"] = gw.VPNConfig
	}
	if gw.RoutingConfig != nil {
		result["routing_config"] = gw.RoutingConfig
	}
	if gw.FirewallConfig != nil {
		result["firewall_config"] = gw.FirewallConfig
	}
	if gw.NATConfig != nil {
		result["nat_config"] = gw.NATConfig
	}
	if gw.QosConfig != nil {
		result["qos_config"] = gw.QosConfig
	}
	if gw.DhcpConfig != nil {
		result["dhcp_config"] = gw.DhcpConfig
	}
	if gw.DnsConfig != nil {
		result["dns_config"] = gw.DnsConfig
	}
	if gw.NtpConfig != nil {
		result["ntp_config"] = gw.NtpConfig
	}
	if gw.SnmpConfig != nil {
		result["snmp_config"] = gw.SnmpConfig
	}
	if gw.SyslogConfig != nil {
		result["syslog_config"] = gw.SyslogConfig
	}
	if gw.TunnelConfig != nil {
		result["tunnel_config"] = gw.TunnelConfig
	}
	if gw.ClusterConfig != nil {
		result["cluster_config"] = gw.ClusterConfig
	}
	if gw.PortConfig != nil {
		result["port_config"] = gw.PortConfig
	}

	// Add additional unknown fields
	for k, v := range gw.AdditionalConfig {
		if _, exists := result[k]; !exists {
			result[k] = v
		}
	}

	return result
}

// ToConfigMap converts gateway device to configuration map (for config files)
func (gw *MistGatewayDevice) ToConfigMap() map[string]interface{} {
	// Start with base configuration fields
	result := gw.BaseDevice.ToConfigMap()

	// Add gateway-specific configuration fields (exclude runtime/status fields)
	if gw.MgmtIntf != nil {
		result["mgmt_intf"] = *gw.MgmtIntf
	}

	// Add configuration objects (exclude runtime status)
	if gw.WANConfig != nil {
		result["wan_config"] = gw.WANConfig
	}
	if gw.LANConfig != nil {
		result["lan_config"] = gw.LANConfig
	}
	if gw.SecurityConfig != nil {
		result["security_config"] = gw.SecurityConfig
	}
	if gw.VPNConfig != nil {
		result["vpn_config"] = gw.VPNConfig
	}
	if gw.RoutingConfig != nil {
		result["routing_config"] = gw.RoutingConfig
	}
	if gw.FirewallConfig != nil {
		result["firewall_config"] = gw.FirewallConfig
	}
	if gw.NATConfig != nil {
		result["nat_config"] = gw.NATConfig
	}
	if gw.QosConfig != nil {
		result["qos_config"] = gw.QosConfig
	}
	if gw.DhcpConfig != nil {
		result["dhcp_config"] = gw.DhcpConfig
	}
	if gw.DnsConfig != nil {
		result["dns_config"] = gw.DnsConfig
	}
	if gw.NtpConfig != nil {
		result["ntp_config"] = gw.NtpConfig
	}
	if gw.SnmpConfig != nil {
		result["snmp_config"] = gw.SnmpConfig
	}
	if gw.SyslogConfig != nil {
		result["syslog_config"] = gw.SyslogConfig
	}
	if gw.TunnelConfig != nil {
		result["tunnel_config"] = gw.TunnelConfig
	}
	if gw.ClusterConfig != nil {
		result["cluster_config"] = gw.ClusterConfig
	}
	if gw.PortConfig != nil {
		result["port_config"] = gw.PortConfig
	}

	// Add configuration fields from AdditionalConfig, filtering out status fields
	statusFields := map[string]bool{
		"connected": true, "adopted": true, "last_seen": true, "uptime": true,
		"version": true, "ip": true, "status": true, "hw_rev": true, "sku": true,
	}

	for k, v := range gw.AdditionalConfig {
		if !statusFields[k] {
			if _, exists := result[k]; !exists {
				result[k] = v
			}
		}
	}

	return result
}

// FromConfigMap populates gateway device from configuration map (from config files)
func (gw *MistGatewayDevice) FromConfigMap(data map[string]interface{}) error {
	// First populate base device configuration
	if err := gw.BaseDevice.FromConfigMap(data); err != nil {
		return fmt.Errorf("failed to populate base device config: %w", err)
	}

	// Parse gateway-specific configuration fields (exclude runtime fields)
	if mgmtIntf, ok := data["mgmt_intf"].(string); ok {
		gw.MgmtIntf = &mgmtIntf
	}

	// Parse complex configuration objects
	configFields := map[string]*map[string]interface{}{
		"wan_config":      &gw.WANConfig,
		"lan_config":      &gw.LANConfig,
		"security_config": &gw.SecurityConfig,
		"vpn_config":      &gw.VPNConfig,
		"routing_config":  &gw.RoutingConfig,
		"firewall_config": &gw.FirewallConfig,
		"nat_config":      &gw.NATConfig,
		"qos_config":      &gw.QosConfig,
		"dhcp_config":     &gw.DhcpConfig,
		"dns_config":      &gw.DnsConfig,
		"ntp_config":      &gw.NtpConfig,
		"snmp_config":     &gw.SnmpConfig,
		"syslog_config":   &gw.SyslogConfig,
		"tunnel_config":   &gw.TunnelConfig,
		"cluster_config":  &gw.ClusterConfig,
		"port_config":     &gw.PortConfig,
	}

	for fieldName, configPtr := range configFields {
		if configData, ok := data[fieldName].(map[string]interface{}); ok {
			*configPtr = configData
		}
	}

	// Store any unknown configuration fields in AdditionalConfig
	gw.AdditionalConfig = make(map[string]interface{})
	knownFields := map[string]bool{
		// Base device config fields
		"name": true, "magic": true, "deviceprofile_id": true, "notes": true, "tags": true,

		// Gateway-specific config fields
		"mgmt_intf": true,

		// Gateway complex config fields
		"wan_config": true, "lan_config": true, "security_config": true, "vpn_config": true,
		"routing_config": true, "firewall_config": true, "nat_config": true, "qos_config": true,
		"dhcp_config": true, "dns_config": true, "ntp_config": true, "snmp_config": true,
		"syslog_config": true, "tunnel_config": true, "cluster_config": true, "port_config": true,
	}

	for k, v := range data {
		if !knownFields[k] {
			gw.AdditionalConfig[k] = v
		}
	}

	return nil
}

// NewGatewayDeviceFromMap creates a new MistGatewayDevice from API response data
func NewGatewayDeviceFromMap(data map[string]interface{}) (*MistGatewayDevice, error) {
	device := &MistGatewayDevice{}
	if err := device.FromMap(data); err != nil {
		return nil, fmt.Errorf("failed to create gateway device from map: %w", err)
	}
	return device, nil
}

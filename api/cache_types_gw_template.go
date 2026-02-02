package api

// FromMap and ToMap methods for Gateway Template types

// MistGatewayTemplate FromMap and ToMap methods
func (gw *MistGatewayTemplate) FromMap(data map[string]interface{}) error {
	// Core fields
	if id, ok := data["id"].(string); ok {
		gw.ID = &id
	}
	if name, ok := data["name"].(string); ok {
		gw.Name = &name
	}
	if orgID, ok := data["org_id"].(string); ok {
		gw.OrgID = &orgID
	}
	if gType, ok := data["type"].(string); ok {
		gw.Type = &gType
	}

	// Network configuration (complex objects from schema)
	if networks, ok := data["networks"].(map[string]interface{}); ok {
		gw.Networks = &networks
	}
	if portConfig, ok := data["port_config"].(map[string]interface{}); ok {
		gw.PortConfig = &portConfig
	}

	// Advanced gateway settings (complex objects from schema)
	if bgpConfig, ok := data["bgp_config"].(map[string]interface{}); ok {
		gw.BGPConfig = &bgpConfig
	}
	if vrfConfig, ok := data["vrf_config"].(map[string]interface{}); ok {
		gw.VRFConfig = &vrfConfig
	}
	if routingConfig, ok := data["routing_policies"].(map[string]interface{}); ok {
		gw.RoutingConfig = &routingConfig
	}

	// Additional configuration for unmapped fields
	if gw.AdditionalConfig == nil {
		gw.AdditionalConfig = make(map[string]interface{})
	}

	// Store complex fields that don't have specific struct mappings
	complexFields := []string{
		"additional_config_cmds", "dhcpd_config", "dns_servers", "dns_suffix",
		"extra_routes", "extra_routes6", "gateway_matching", "idp_profiles",
		"ip_configs", "ntp_servers", "oob_ip_config", "path_preferences",
		"router_id", "service_policies", "tunnel_configs", "tunnel_provider_options",
		"vrf_instances", "dnsOverride", "ntpOverride",
	}

	for _, field := range complexFields {
		if value, exists := data[field]; exists {
			gw.AdditionalConfig[field] = value
		}
	}

	// Timestamps
	if createdTime, ok := data["created_time"].(float64); ok {
		createdTimeInt := int64(createdTime)
		gw.CreatedTime = &createdTimeInt
	}
	if modifiedTime, ok := data["modified_time"].(float64); ok {
		modifiedTimeInt := int64(modifiedTime)
		gw.ModifiedTime = &modifiedTimeInt
	}

	return nil
}

func (gw *MistGatewayTemplate) ToMap() map[string]interface{} {
	result := make(map[string]interface{})

	// Core fields
	if gw.ID != nil {
		result["id"] = *gw.ID
	}
	if gw.Name != nil {
		result["name"] = *gw.Name
	}
	if gw.OrgID != nil {
		result["org_id"] = *gw.OrgID
	}
	if gw.Type != nil {
		result["type"] = *gw.Type
	}

	// Network configuration
	if gw.Networks != nil {
		result["networks"] = *gw.Networks
	}
	if gw.PortConfig != nil {
		result["port_config"] = *gw.PortConfig
	}

	// Advanced gateway settings
	if gw.BGPConfig != nil {
		result["bgp_config"] = *gw.BGPConfig
	}
	if gw.VRFConfig != nil {
		result["vrf_config"] = *gw.VRFConfig
	}
	if gw.RoutingConfig != nil {
		result["routing_policies"] = *gw.RoutingConfig
	}

	// Timestamps
	if gw.CreatedTime != nil {
		result["created_time"] = *gw.CreatedTime
	}
	if gw.ModifiedTime != nil {
		result["modified_time"] = *gw.ModifiedTime
	}

	// Add additional configuration fields
	for key, value := range gw.AdditionalConfig {
		result[key] = value
	}

	return result
}

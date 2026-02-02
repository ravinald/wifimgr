package api

// FromMap and ToMap methods for Network types

// MistNetwork FromMap and ToMap methods
func (n *MistNetwork) FromMap(data map[string]interface{}) error {
	// Core fields
	if id, ok := data["id"].(string); ok {
		n.ID = &id
	}
	if name, ok := data["name"].(string); ok {
		n.Name = &name
	}
	if orgID, ok := data["org_id"].(string); ok {
		n.OrgID = &orgID
	}

	// Network configuration fields
	if vlanID, ok := data["vlan_id"].(float64); ok {
		vlanIDInt := int(vlanID)
		n.VlanID = &vlanIDInt
	}
	if subnet, ok := data["subnet"].(string); ok {
		n.Subnet = &subnet
	}
	if gateway, ok := data["gateway"].(string); ok {
		n.Gateway = &gateway
	}
	if subnet6, ok := data["subnet6"].(string); ok {
		n.Subnet6 = &subnet6
	}
	if gateway6, ok := data["gateway6"].(string); ok {
		n.Gateway6 = &gateway6
	}

	// Access control fields
	if internalAccess, ok := data["internal_access"].(bool); ok {
		n.InternalAccess = &internalAccess
	}
	if internetAccess, ok := data["internet_access"].(bool); ok {
		n.InternetAccess = &internetAccess
	}
	if isolation, ok := data["isolation"].(bool); ok {
		n.Isolation = &isolation
	}

	// Advanced settings
	if multicast, ok := data["multicast"].(bool); ok {
		n.Multicast = &multicast
	}

	// Tenants field - handle as array of strings
	if tenants, ok := data["tenants"].([]interface{}); ok {
		var tenantStrings []string
		for _, tenant := range tenants {
			if tenantStr, ok := tenant.(string); ok {
				tenantStrings = append(tenantStrings, tenantStr)
			}
		}
		if len(tenantStrings) > 0 {
			n.Tenants = &tenantStrings
		}
	}

	// Timestamps
	if createdTime, ok := data["created_time"].(float64); ok {
		createdTimeInt := int64(createdTime)
		n.CreatedTime = &createdTimeInt
	}
	if modifiedTime, ok := data["modified_time"].(float64); ok {
		modifiedTimeInt := int64(modifiedTime)
		n.ModifiedTime = &modifiedTimeInt
	}

	// Store additional complex fields in AdditionalConfig
	if n.AdditionalConfig == nil {
		n.AdditionalConfig = make(map[string]interface{})
	}

	complexFields := []string{
		"disallow_mist_services", "routed_for_networks", "vpn_access",
	}

	for _, field := range complexFields {
		if value, exists := data[field]; exists {
			n.AdditionalConfig[field] = value
		}
	}

	return nil
}

func (n *MistNetwork) ToMap() map[string]interface{} {
	result := make(map[string]interface{})

	// Core fields
	if n.ID != nil {
		result["id"] = *n.ID
	}
	if n.Name != nil {
		result["name"] = *n.Name
	}
	if n.OrgID != nil {
		result["org_id"] = *n.OrgID
	}

	// Network configuration
	if n.VlanID != nil {
		result["vlan_id"] = *n.VlanID
	}
	if n.Subnet != nil {
		result["subnet"] = *n.Subnet
	}
	if n.Gateway != nil {
		result["gateway"] = *n.Gateway
	}
	if n.Subnet6 != nil {
		result["subnet6"] = *n.Subnet6
	}
	if n.Gateway6 != nil {
		result["gateway6"] = *n.Gateway6
	}

	// Access control
	if n.InternalAccess != nil {
		result["internal_access"] = *n.InternalAccess
	}
	if n.InternetAccess != nil {
		result["internet_access"] = *n.InternetAccess
	}
	if n.Isolation != nil {
		result["isolation"] = *n.Isolation
	}

	// Advanced settings
	if n.Multicast != nil {
		result["multicast"] = *n.Multicast
	}
	if n.Tenants != nil {
		result["tenants"] = *n.Tenants
	}

	// Timestamps
	if n.CreatedTime != nil {
		result["created_time"] = *n.CreatedTime
	}
	if n.ModifiedTime != nil {
		result["modified_time"] = *n.ModifiedTime
	}

	// Add additional configuration fields
	for key, value := range n.AdditionalConfig {
		result[key] = value
	}

	return result
}

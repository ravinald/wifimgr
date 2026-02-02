package api

// FromMap and ToMap methods for WLAN Template types

// MistWLANTemplate FromMap and ToMap methods
func (wt *MistWLANTemplate) FromMap(data map[string]interface{}) error {
	// Core fields
	if id, ok := data["id"].(string); ok {
		wt.ID = &id
	}
	if name, ok := data["name"].(string); ok {
		wt.Name = &name
	}
	if orgID, ok := data["org_id"].(string); ok {
		wt.OrgID = &orgID
	}

	// Template settings
	if ssid, ok := data["ssid"].(string); ok {
		wt.SSID = &ssid
	}
	if vlanID, ok := data["vlan_id"].(float64); ok {
		vlanIDInt := int(vlanID)
		wt.VlanID = &vlanIDInt
	}
	if iface, ok := data["interface"].(string); ok {
		wt.Interface = &iface
	}

	// Authentication template (complex object)
	if auth, ok := data["auth"].(map[string]interface{}); ok {
		wt.Auth = &auth
	}

	// QoS template (complex object)
	if qos, ok := data["qos"].(map[string]interface{}); ok {
		wt.QoS = &qos
	}

	// Advanced settings
	if band, ok := data["band"].(string); ok {
		wt.Band = &band
	}
	if enabled, ok := data["enabled"].(bool); ok {
		wt.Enabled = &enabled
	}
	if hidden, ok := data["hidden"].(bool); ok {
		wt.Hidden = &hidden
	}

	// Timestamps
	if createdTime, ok := data["created_time"].(float64); ok {
		createdTimeInt := int64(createdTime)
		wt.CreatedTime = &createdTimeInt
	}
	if modifiedTime, ok := data["modified_time"].(float64); ok {
		modifiedTimeInt := int64(modifiedTime)
		wt.ModifiedTime = &modifiedTimeInt
	}

	// Store additional complex fields in AdditionalConfig
	if wt.AdditionalConfig == nil {
		wt.AdditionalConfig = make(map[string]interface{})
	}

	// Store all other fields that aren't explicitly mapped
	complexFields := []string{
		"template_id", "apply_to", "wlan_limit_down", "wlan_limit_up",
		"client_limit_down", "client_limit_up", "airwatch", "bonjour",
		"radsec", "dynamic_vlan", "hotspot20", "portal_template",
	}

	for _, field := range complexFields {
		if value, exists := data[field]; exists {
			wt.AdditionalConfig[field] = value
		}
	}

	return nil
}

func (wt *MistWLANTemplate) ToMap() map[string]interface{} {
	result := make(map[string]interface{})

	// Core fields
	if wt.ID != nil {
		result["id"] = *wt.ID
	}
	if wt.Name != nil {
		result["name"] = *wt.Name
	}
	if wt.OrgID != nil {
		result["org_id"] = *wt.OrgID
	}

	// Template settings
	if wt.SSID != nil {
		result["ssid"] = *wt.SSID
	}
	if wt.VlanID != nil {
		result["vlan_id"] = *wt.VlanID
	}
	if wt.Interface != nil {
		result["interface"] = *wt.Interface
	}

	// Authentication and QoS templates
	if wt.Auth != nil {
		result["auth"] = *wt.Auth
	}
	if wt.QoS != nil {
		result["qos"] = *wt.QoS
	}

	// Advanced settings
	if wt.Band != nil {
		result["band"] = *wt.Band
	}
	if wt.Enabled != nil {
		result["enabled"] = *wt.Enabled
	}
	if wt.Hidden != nil {
		result["hidden"] = *wt.Hidden
	}

	// Timestamps
	if wt.CreatedTime != nil {
		result["created_time"] = *wt.CreatedTime
	}
	if wt.ModifiedTime != nil {
		result["modified_time"] = *wt.ModifiedTime
	}

	// Add additional configuration fields
	for key, value := range wt.AdditionalConfig {
		result[key] = value
	}

	return result
}

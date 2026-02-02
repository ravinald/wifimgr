package api

// FromMap and ToMap methods for Site types

// MistSite FromMap - enhanced to map all fields from site.json schema
func (s *MistSite) FromMapEnhanced(data map[string]interface{}) error {
	// Initialize AdditionalConfig if nil
	if s.AdditionalConfig == nil {
		s.AdditionalConfig = make(map[string]interface{})
	}

	// Core fields
	if id, ok := data["id"].(string); ok {
		s.ID = &id
	}
	if name, ok := data["name"].(string); ok {
		s.Name = &name
	}
	if address, ok := data["address"].(string); ok {
		s.Address = &address
	}
	if countryCode, ok := data["country_code"].(string); ok {
		s.CountryCode = &countryCode
	}
	if timezone, ok := data["timezone"].(string); ok {
		s.Timezone = &timezone
	}
	if notes, ok := data["notes"].(string); ok {
		s.Notes = &notes
	}

	// Status fields (read-only)
	if createdTime, ok := data["created_time"].(float64); ok {
		s.CreatedTime = &createdTime
	}
	if modifiedTime, ok := data["modified_time"].(float64); ok {
		s.ModifiedTime = &modifiedTime
	}
	if orgID, ok := data["org_id"].(string); ok {
		s.OrgID = &orgID
	}

	// Latlng coordinates
	if latlngData, ok := data["latlng"].(map[string]interface{}); ok {
		if s.Latlng == nil {
			s.Latlng = &MistLatLng{}
		}
		if lat, ok := latlngData["lat"].(float64); ok {
			s.Latlng.Lat = &lat
		}
		if lng, ok := latlngData["lng"].(float64); ok {
			s.Latlng.Lng = &lng
		}
	}

	// Store additional complex fields in AdditionalConfig based on site.json schema
	complexFields := []string{
		"alarmtemplate_id", "aptemplate_id", "gatewaytemplate_id", "networktemplate_id",
		"rftemplate_id", "secpolicy_id", "sitegroup_ids", "vars",
		"setting", "sle", "stats", "auto_upgrade", "engage",
		"evpn_options", "location", "msp_id", "rogue", "wan_vna",
		"wlan_limit_down", "wlan_limit_up", "wxtag_ids", "wxtunnel_id",
		"skyatp", "wiredvna", "managed_ap_device_limit", "managed_other_device_limit",
	}

	for _, field := range complexFields {
		if value, exists := data[field]; exists {
			s.AdditionalConfig[field] = value
		}
	}

	return nil
}

// ToMapEnhanced converts the MistSite to a complete map preserving all API fields
func (s *MistSite) ToMapEnhanced() map[string]interface{} {
	result := make(map[string]interface{})

	// Core fields
	if s.ID != nil {
		result["id"] = *s.ID
	}
	if s.Name != nil {
		result["name"] = *s.Name
	}
	if s.Address != nil {
		result["address"] = *s.Address
	}
	if s.CountryCode != nil {
		result["country_code"] = *s.CountryCode
	}
	if s.Timezone != nil {
		result["timezone"] = *s.Timezone
	}
	if s.Notes != nil {
		result["notes"] = *s.Notes
	}

	// Status fields
	if s.CreatedTime != nil {
		result["created_time"] = *s.CreatedTime
	}
	if s.ModifiedTime != nil {
		result["modified_time"] = *s.ModifiedTime
	}
	if s.OrgID != nil {
		result["org_id"] = *s.OrgID
	}

	// Latlng coordinates
	if s.Latlng != nil {
		latlngMap := make(map[string]interface{})
		if s.Latlng.Lat != nil {
			latlngMap["lat"] = *s.Latlng.Lat
		}
		if s.Latlng.Lng != nil {
			latlngMap["lng"] = *s.Latlng.Lng
		}
		if len(latlngMap) > 0 {
			result["latlng"] = latlngMap
		}
	}

	// Add all additional configuration fields
	for key, value := range s.AdditionalConfig {
		result[key] = value
	}

	return result
}

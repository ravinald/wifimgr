package api

// FromMap and ToMap methods for RF Template types

// MistRFTemplate FromMap and ToMap methods
func (rf *MistRFTemplate) FromMap(data map[string]interface{}) error {
	// Core fields
	if id, ok := data["id"].(string); ok {
		rf.ID = &id
	}
	if name, ok := data["name"].(string); ok {
		rf.Name = &name
	}
	if orgID, ok := data["org_id"].(string); ok {
		rf.OrgID = &orgID
	}

	// Antenna gain settings
	if antGain24, ok := data["ant_gain_24"].(float64); ok {
		antGain24Int := int(antGain24)
		rf.AntGain24 = &antGain24Int
	}
	if antGain5, ok := data["ant_gain_5"].(float64); ok {
		antGain5Int := int(antGain5)
		rf.AntGain5 = &antGain5Int
	}
	if antGain6, ok := data["ant_gain_6"].(float64); ok {
		antGain6Int := int(antGain6)
		rf.AntGain6 = &antGain6Int
	}

	// Radio band configurations
	if band24, ok := data["band_24"].(map[string]interface{}); ok {
		rf.Band24 = band24
	}
	if band24Usage, ok := data["band_24_usage"].(string); ok {
		rf.Band24Usage = &band24Usage
	}
	if band5, ok := data["band_5"].(map[string]interface{}); ok {
		rf.Band5 = band5
	}
	if band5On24Radio, ok := data["band_5_on_24_radio"].(map[string]interface{}); ok {
		rf.Band5On24Radio = band5On24Radio
	}
	if band6, ok := data["band_6"].(map[string]interface{}); ok {
		rf.Band6 = band6
	}

	// Settings
	if countryCode, ok := data["country_code"].(string); ok {
		rf.CountryCode = &countryCode
	}
	if forSite, ok := data["for_site"].(bool); ok {
		rf.ForSite = &forSite
	}
	if scanningEnabled, ok := data["scanning_enabled"].(bool); ok {
		rf.ScanningEnabled = &scanningEnabled
	}

	// Model-specific configurations
	if modelSpecific, ok := data["model_specific"].(map[string]interface{}); ok {
		rf.ModelSpecific = modelSpecific
	}

	// Timestamps
	if createdTime, ok := data["created_time"].(float64); ok {
		createdTimeInt := int64(createdTime)
		rf.CreatedTime = &createdTimeInt
	}
	if modifiedTime, ok := data["modified_time"].(float64); ok {
		modifiedTimeInt := int64(modifiedTime)
		rf.ModifiedTime = &modifiedTimeInt
	}

	return nil
}

func (rf *MistRFTemplate) ToMap() map[string]interface{} {
	result := make(map[string]interface{})

	// Core fields
	if rf.ID != nil {
		result["id"] = *rf.ID
	}
	if rf.Name != nil {
		result["name"] = *rf.Name
	}
	if rf.OrgID != nil {
		result["org_id"] = *rf.OrgID
	}

	// Antenna gain settings
	if rf.AntGain24 != nil {
		result["ant_gain_24"] = *rf.AntGain24
	}
	if rf.AntGain5 != nil {
		result["ant_gain_5"] = *rf.AntGain5
	}
	if rf.AntGain6 != nil {
		result["ant_gain_6"] = *rf.AntGain6
	}

	// Radio band configurations
	if rf.Band24 != nil {
		result["band_24"] = rf.Band24
	}
	if rf.Band24Usage != nil {
		result["band_24_usage"] = *rf.Band24Usage
	}
	if rf.Band5 != nil {
		result["band_5"] = rf.Band5
	}
	if rf.Band5On24Radio != nil {
		result["band_5_on_24_radio"] = rf.Band5On24Radio
	}
	if rf.Band6 != nil {
		result["band_6"] = rf.Band6
	}

	// Settings
	if rf.CountryCode != nil {
		result["country_code"] = *rf.CountryCode
	}
	if rf.ForSite != nil {
		result["for_site"] = *rf.ForSite
	}
	if rf.ScanningEnabled != nil {
		result["scanning_enabled"] = *rf.ScanningEnabled
	}

	// Model-specific configurations
	if rf.ModelSpecific != nil {
		result["model_specific"] = rf.ModelSpecific
	}

	// Timestamps
	if rf.CreatedTime != nil {
		result["created_time"] = *rf.CreatedTime
	}
	if rf.ModifiedTime != nil {
		result["modified_time"] = *rf.ModifiedTime
	}

	return result
}

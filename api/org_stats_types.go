package api

// OrgStatsSLEUserMinutes represents the user minutes statistics for SLE
type OrgStatsSLEUserMinutes struct {
	OK    *float64 `json:"ok"`
	Total *float64 `json:"total"`
}

// OrgStatsSLE represents the SLE (Service Level Expectation) statistics
type OrgStatsSLE struct {
	Path        *string                 `json:"path"`
	UserMinutes *OrgStatsSLEUserMinutes `json:"user_minutes,omitempty"`
}

// OrgStats represents organization statistics from /api/v1/orgs/{org_id}/stats
type OrgStats struct {
	// Required fields
	AlarmTemplateID        *string        `json:"alarmtemplate_id"`
	AllowMist              *bool          `json:"allow_mist"`
	CreatedTime            *float64       `json:"created_time"`
	ID                     *string        `json:"id"`
	ModifiedTime           *float64       `json:"modified_time"`
	MspID                  *string        `json:"msp_id"`
	Name                   *string        `json:"name"`
	NumDevices             *int           `json:"num_devices"`
	NumDevicesConnected    *int           `json:"num_devices_connected"`
	NumDevicesDisconnected *int           `json:"num_devices_disconnected"`
	NumInventory           *int           `json:"num_inventory"`
	NumSites               *int           `json:"num_sites"`
	OrgGroupIDs            []string       `json:"orggroup_ids"`
	SessionExpiry          *int64         `json:"session_expiry"`
	SLE                    []*OrgStatsSLE `json:"sle"`

	// Additional fields that may be present
	AdditionalFields map[string]interface{} `json:"-"`
}

// ToMap converts OrgStats to a map for API operations
func (os *OrgStats) ToMap() map[string]interface{} {
	result := make(map[string]interface{})

	// Add typed fields
	if os.AlarmTemplateID != nil {
		result["alarmtemplate_id"] = *os.AlarmTemplateID
	}
	if os.AllowMist != nil {
		result["allow_mist"] = *os.AllowMist
	}
	if os.CreatedTime != nil {
		result["created_time"] = *os.CreatedTime
	}
	if os.ID != nil {
		result["id"] = *os.ID
	}
	if os.ModifiedTime != nil {
		result["modified_time"] = *os.ModifiedTime
	}
	if os.MspID != nil {
		result["msp_id"] = *os.MspID
	}
	if os.Name != nil {
		result["name"] = *os.Name
	}
	if os.NumDevices != nil {
		result["num_devices"] = *os.NumDevices
	}
	if os.NumDevicesConnected != nil {
		result["num_devices_connected"] = *os.NumDevicesConnected
	}
	if os.NumDevicesDisconnected != nil {
		result["num_devices_disconnected"] = *os.NumDevicesDisconnected
	}
	if os.NumInventory != nil {
		result["num_inventory"] = *os.NumInventory
	}
	if os.NumSites != nil {
		result["num_sites"] = *os.NumSites
	}
	if len(os.OrgGroupIDs) > 0 {
		result["orggroup_ids"] = os.OrgGroupIDs
	}
	if os.SessionExpiry != nil {
		result["session_expiry"] = *os.SessionExpiry
	}
	if len(os.SLE) > 0 {
		sleArray := make([]map[string]interface{}, len(os.SLE))
		for i, sle := range os.SLE {
			sleMap := make(map[string]interface{})
			if sle.Path != nil {
				sleMap["path"] = *sle.Path
			}
			if sle.UserMinutes != nil {
				userMinutesMap := make(map[string]interface{})
				if sle.UserMinutes.OK != nil {
					userMinutesMap["ok"] = *sle.UserMinutes.OK
				}
				if sle.UserMinutes.Total != nil {
					userMinutesMap["total"] = *sle.UserMinutes.Total
				}
				sleMap["user_minutes"] = userMinutesMap
			}
			sleArray[i] = sleMap
		}
		result["sle"] = sleArray
	}

	// Add additional fields
	for key, value := range os.AdditionalFields {
		result[key] = value
	}

	return result
}

// FromMap populates OrgStats from a map
func (os *OrgStats) FromMap(data map[string]interface{}) error {
	if data == nil {
		return nil
	}

	// Initialize additional fields if nil
	if os.AdditionalFields == nil {
		os.AdditionalFields = make(map[string]interface{})
	}

	// Extract known fields
	if alarmTemplateID, ok := data["alarmtemplate_id"].(string); ok {
		os.AlarmTemplateID = &alarmTemplateID
	}
	if allowMist, ok := data["allow_mist"].(bool); ok {
		os.AllowMist = &allowMist
	}
	if createdTime, ok := data["created_time"].(float64); ok {
		os.CreatedTime = &createdTime
	}
	if id, ok := data["id"].(string); ok {
		os.ID = &id
	}
	if modifiedTime, ok := data["modified_time"].(float64); ok {
		os.ModifiedTime = &modifiedTime
	}
	if mspID, ok := data["msp_id"].(string); ok {
		os.MspID = &mspID
	}
	if name, ok := data["name"].(string); ok {
		os.Name = &name
	}
	if numDevices, ok := data["num_devices"].(float64); ok {
		numDevicesInt := int(numDevices)
		os.NumDevices = &numDevicesInt
	}
	if numDevicesConnected, ok := data["num_devices_connected"].(float64); ok {
		numDevicesConnectedInt := int(numDevicesConnected)
		os.NumDevicesConnected = &numDevicesConnectedInt
	}
	if numDevicesDisconnected, ok := data["num_devices_disconnected"].(float64); ok {
		numDevicesDisconnectedInt := int(numDevicesDisconnected)
		os.NumDevicesDisconnected = &numDevicesDisconnectedInt
	}
	if numInventory, ok := data["num_inventory"].(float64); ok {
		numInventoryInt := int(numInventory)
		os.NumInventory = &numInventoryInt
	}
	if numSites, ok := data["num_sites"].(float64); ok {
		numSitesInt := int(numSites)
		os.NumSites = &numSitesInt
	}
	if orgGroupIDs, ok := data["orggroup_ids"].([]interface{}); ok {
		os.OrgGroupIDs = make([]string, 0, len(orgGroupIDs))
		for _, id := range orgGroupIDs {
			if idStr, ok := id.(string); ok {
				os.OrgGroupIDs = append(os.OrgGroupIDs, idStr)
			}
		}
	}
	if sessionExpiry, ok := data["session_expiry"].(float64); ok {
		sessionExpiryInt := int64(sessionExpiry)
		os.SessionExpiry = &sessionExpiryInt
	}

	// Parse SLE array
	if sleArray, ok := data["sle"].([]interface{}); ok {
		os.SLE = make([]*OrgStatsSLE, 0, len(sleArray))
		for _, sleItem := range sleArray {
			if sleMap, ok := sleItem.(map[string]interface{}); ok {
				sle := &OrgStatsSLE{}
				if path, ok := sleMap["path"].(string); ok {
					sle.Path = &path
				}
				if userMinutesMap, ok := sleMap["user_minutes"].(map[string]interface{}); ok {
					userMinutes := &OrgStatsSLEUserMinutes{}
					if okVal, ok := userMinutesMap["ok"].(float64); ok {
						userMinutes.OK = &okVal
					}
					if totalVal, ok := userMinutesMap["total"].(float64); ok {
						userMinutes.Total = &totalVal
					}
					sle.UserMinutes = userMinutes
				}
				os.SLE = append(os.SLE, sle)
			}
		}
	}

	// Store any unknown fields in AdditionalFields
	knownFields := map[string]bool{
		"alarmtemplate_id": true, "allow_mist": true, "created_time": true, "id": true,
		"modified_time": true, "msp_id": true, "name": true, "num_devices": true,
		"num_devices_connected": true, "num_devices_disconnected": true, "num_inventory": true,
		"num_sites": true, "orggroup_ids": true, "session_expiry": true, "sle": true,
	}

	for k, v := range data {
		if !knownFields[k] {
			os.AdditionalFields[k] = v
		}
	}

	return nil
}

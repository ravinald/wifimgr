package api

// FromMap and ToMap methods for WLAN types

// MistWLAN FromMap and ToMap methods
func (w *MistWLAN) FromMap(data map[string]interface{}) error {
	// Core fields
	if id, ok := data["id"].(string); ok {
		w.ID = &id
	}
	if ssid, ok := data["ssid"].(string); ok {
		w.SSID = &ssid
	}
	if orgID, ok := data["org_id"].(string); ok {
		w.OrgID = &orgID
	}
	if siteID, ok := data["site_id"].(string); ok {
		w.SiteID = &siteID
	}

	// Network settings
	if vlanID, ok := data["vlan_id"].(float64); ok {
		vlanIDInt := int(vlanID)
		w.VlanID = &vlanIDInt
	}
	if iface, ok := data["interface"].(string); ok {
		w.Interface = &iface
	}
	if isolation, ok := data["isolation"].(bool); ok {
		w.Isolation = &isolation
	}

	// Authentication (complex nested structure)
	if authData, ok := data["auth"].(map[string]interface{}); ok {
		if authType, ok := authData["type"].(string); ok {
			w.Auth.Type = &authType
		}
		if psk, ok := authData["psk"].(string); ok {
			w.Auth.PSK = &psk
		}
		if keyIdx, ok := authData["key_idx"].(float64); ok {
			keyIdxInt := int(keyIdx)
			w.Auth.KeyIdx = &keyIdxInt
		}
		if keys, ok := authData["keys"].([]interface{}); ok {
			var keyStrings []string
			for _, key := range keys {
				if keyStr, ok := key.(string); ok {
					keyStrings = append(keyStrings, keyStr)
				}
			}
			if len(keyStrings) > 0 {
				w.Auth.Keys = &keyStrings
			}
		}

		// Enterprise auth
		if enterprise, ok := authData["enterprise"].(map[string]interface{}); ok {
			if w.Auth.Enterprise == nil {
				w.Auth.Enterprise = &struct {
					Radius *struct {
						Host   *string `json:"host,omitempty"`
						Port   *int    `json:"port,omitempty"`
						Secret *string `json:"secret,omitempty"`
					} `json:"radius,omitempty"`
				}{}
			}

			if radius, ok := enterprise["radius"].(map[string]interface{}); ok {
				if w.Auth.Enterprise.Radius == nil {
					w.Auth.Enterprise.Radius = &struct {
						Host   *string `json:"host,omitempty"`
						Port   *int    `json:"port,omitempty"`
						Secret *string `json:"secret,omitempty"`
					}{}
				}

				if host, ok := radius["host"].(string); ok {
					w.Auth.Enterprise.Radius.Host = &host
				}
				if port, ok := radius["port"].(float64); ok {
					portInt := int(port)
					w.Auth.Enterprise.Radius.Port = &portInt
				}
				if secret, ok := radius["secret"].(string); ok {
					w.Auth.Enterprise.Radius.Secret = &secret
				}
			}
		}
	}

	// QoS settings
	if qosData, ok := data["qos"].(map[string]interface{}); ok {
		if class, ok := qosData["class"].(string); ok {
			w.QoS.Class = &class
		}
	}

	// Basic settings
	if band, ok := data["band"].(string); ok {
		w.Band = &band
	}
	if enabled, ok := data["enabled"].(bool); ok {
		w.Enabled = &enabled
	}
	if hidden, ok := data["hidden"].(bool); ok {
		w.Hidden = &hidden
	}

	// Apply To field
	if applyTo, ok := data["apply_to"].([]interface{}); ok {
		var applyToStrings []string
		for _, item := range applyTo {
			if itemStr, ok := item.(string); ok {
				applyToStrings = append(applyToStrings, itemStr)
			}
		}
		if len(applyToStrings) > 0 {
			w.ApplyTo = &applyToStrings
		}
	}

	// Timestamps
	if createdTime, ok := data["created_time"].(float64); ok {
		createdTimeInt := int64(createdTime)
		w.CreatedTime = &createdTimeInt
	}
	if modifiedTime, ok := data["modified_time"].(float64); ok {
		modifiedTimeInt := int64(modifiedTime)
		w.ModifiedTime = &modifiedTimeInt
	}

	// Store additional complex fields in AdditionalConfig
	if w.AdditionalConfig == nil {
		w.AdditionalConfig = make(map[string]interface{})
	}

	// Store all other complex fields that aren't explicitly mapped
	complexFields := []string{
		"acct_immediate_update", "acct_interim_interval", "acct_servers", "airwatch",
		"allow_ipv6_ndp", "allow_mdns", "allow_ssdp", "ap_ids", "app_limit", "app_qos",
		"bonjour", "client_limit_down", "client_limit_up", "coa_servers", "disable_11ax",
		"disable_ht_cc_protection", "disable_uapsd", "dynamic_psk", "dynamic_vlan",
		"fast_dot1x_timers", "portal", "radsec", "roam_mode", "schedule", "wlan_limit_down",
		"wlan_limit_up", "wxtag_ids", "wxtunnel_id", "wxtunnel_remote_id",
	}

	for _, field := range complexFields {
		if value, exists := data[field]; exists {
			w.AdditionalConfig[field] = value
		}
	}

	return nil
}

func (w *MistWLAN) ToMap() map[string]interface{} {
	result := make(map[string]interface{})

	// Core fields
	if w.ID != nil {
		result["id"] = *w.ID
	}
	if w.SSID != nil {
		result["ssid"] = *w.SSID
	}
	if w.OrgID != nil {
		result["org_id"] = *w.OrgID
	}
	if w.SiteID != nil {
		result["site_id"] = *w.SiteID
	}

	// Network settings
	if w.VlanID != nil {
		result["vlan_id"] = *w.VlanID
	}
	if w.Interface != nil {
		result["interface"] = *w.Interface
	}
	if w.Isolation != nil {
		result["isolation"] = *w.Isolation
	}

	// Authentication - reconstruct the nested structure
	authMap := make(map[string]interface{})
	if w.Auth.Type != nil {
		authMap["type"] = *w.Auth.Type
	}
	if w.Auth.PSK != nil {
		authMap["psk"] = *w.Auth.PSK
	}
	if w.Auth.KeyIdx != nil {
		authMap["key_idx"] = *w.Auth.KeyIdx
	}
	if w.Auth.Keys != nil {
		authMap["keys"] = *w.Auth.Keys
	}

	if w.Auth.Enterprise != nil && w.Auth.Enterprise.Radius != nil {
		enterpriseMap := make(map[string]interface{})
		radiusMap := make(map[string]interface{})

		if w.Auth.Enterprise.Radius.Host != nil {
			radiusMap["host"] = *w.Auth.Enterprise.Radius.Host
		}
		if w.Auth.Enterprise.Radius.Port != nil {
			radiusMap["port"] = *w.Auth.Enterprise.Radius.Port
		}
		if w.Auth.Enterprise.Radius.Secret != nil {
			radiusMap["secret"] = *w.Auth.Enterprise.Radius.Secret
		}

		if len(radiusMap) > 0 {
			enterpriseMap["radius"] = radiusMap
			authMap["enterprise"] = enterpriseMap
		}
	}

	if len(authMap) > 0 {
		result["auth"] = authMap
	}

	// QoS settings
	if w.QoS.Class != nil {
		qosMap := map[string]interface{}{
			"class": *w.QoS.Class,
		}
		result["qos"] = qosMap
	}

	// Basic settings
	if w.Band != nil {
		result["band"] = *w.Band
	}
	if w.Enabled != nil {
		result["enabled"] = *w.Enabled
	}
	if w.Hidden != nil {
		result["hidden"] = *w.Hidden
	}
	if w.ApplyTo != nil {
		result["apply_to"] = *w.ApplyTo
	}

	// Timestamps
	if w.CreatedTime != nil {
		result["created_time"] = *w.CreatedTime
	}
	if w.ModifiedTime != nil {
		result["modified_time"] = *w.ModifiedTime
	}

	// Add additional configuration fields
	for key, value := range w.AdditionalConfig {
		result[key] = value
	}

	return result
}

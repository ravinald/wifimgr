package api

import (
	"fmt"
)

// SiteMarshaler defines the interface for bidirectional site data transformation
// between API representations and structured types.
type SiteMarshaler interface {
	GetID() string
	GetName() string
	ToMap() map[string]interface{}
	FromMap(data map[string]interface{}) error
	ToConfigMap() map[string]interface{}
	FromConfigMap(data map[string]interface{}) error
	GetRaw() map[string]interface{}
	SetRaw(data map[string]interface{})
}

// MistSite represents a Mist site with bidirectional data handling
type MistSite struct {
	// Core site fields with typed access
	ID          *string     `json:"id,omitempty"`
	Name        *string     `json:"name,omitempty"`
	Address     *string     `json:"address,omitempty"`
	CountryCode *string     `json:"country_code,omitempty"`
	Timezone    *string     `json:"timezone,omitempty"`
	Notes       *string     `json:"notes,omitempty"`
	Latlng      *MistLatLng `json:"latlng,omitempty"`

	// Status fields (read-only, not included in config)
	CreatedTime  *float64 `json:"created_time,omitempty"`
	ModifiedTime *float64 `json:"modified_time,omitempty"`
	OrgID        *string  `json:"org_id,omitempty"`

	// Template IDs from site.json schema
	AlarmTemplateID   *string `json:"alarmtemplate_id,omitempty"`
	APTemplateID      *string `json:"aptemplate_id,omitempty"`
	GatewayTemplateID *string `json:"gatewaytemplate_id,omitempty"`
	NetworkTemplateID *string `json:"networktemplate_id,omitempty"`
	RFTemplateID      *string `json:"rftemplate_id,omitempty"`
	SecPolicyID       *string `json:"secpolicy_id,omitempty"`
	SiteTemplateID    *string `json:"sitetemplate_id,omitempty"`

	// Site groups and variables
	SiteGroupIDs *[]string              `json:"sitegroup_ids,omitempty"`
	Vars         map[string]interface{} `json:"vars,omitempty"`

	// Complex configuration objects (stored as maps to preserve exact API structure)
	Setting     map[string]interface{} `json:"setting,omitempty"`
	SLE         map[string]interface{} `json:"sle,omitempty"`
	Stats       map[string]interface{} `json:"stats,omitempty"`
	AutoUpgrade map[string]interface{} `json:"auto_upgrade,omitempty"`
	Engage      map[string]interface{} `json:"engage,omitempty"`
	EvpnOptions map[string]interface{} `json:"evpn_options,omitempty"`
	Rogue       map[string]interface{} `json:"rogue,omitempty"`
	WanVNA      map[string]interface{} `json:"wan_vna,omitempty"`
	SkyATP      map[string]interface{} `json:"skyatp,omitempty"`
	WiredVNA    map[string]interface{} `json:"wiredvna,omitempty"`

	// Simple limit fields and additional properties
	MSPID                   *string   `json:"msp_id,omitempty"`
	WlanLimitDown           *int      `json:"wlan_limit_down,omitempty"`
	WlanLimitUp             *int      `json:"wlan_limit_up,omitempty"`
	ManagedAPDeviceLimit    *int      `json:"managed_ap_device_limit,omitempty"`
	ManagedOtherDeviceLimit *int      `json:"managed_other_device_limit,omitempty"`
	WxTagIDs                *[]string `json:"wxtag_ids,omitempty"`
	WxTunnelID              *string   `json:"wxtunnel_id,omitempty"`
	TzOffset                *int      `json:"tzoffset,omitempty"`

	// Additional configuration for any unmapped fields
	AdditionalConfig map[string]interface{} `json:"-"`
}

// MistLatLng represents latitude/longitude coordinates with bidirectional handling
type MistLatLng struct {
	Lat *float64 `json:"lat,omitempty"`
	Lng *float64 `json:"lng,omitempty"`
}

// GetID returns the site ID as a string
func (s *MistSite) GetID() string {
	if s.ID != nil {
		return *s.ID
	}
	return ""
}

// GetName returns the site name
func (s *MistSite) GetName() string {
	if s.Name != nil {
		return *s.Name
	}
	return ""
}

// ToMap converts the site to a map representation suitable for API requests
func (s *MistSite) ToMap() map[string]interface{} {
	result := make(map[string]interface{})

	// Add typed fields
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
	if s.Latlng != nil {
		latlng := make(map[string]interface{})
		if s.Latlng.Lat != nil {
			latlng["lat"] = *s.Latlng.Lat
		}
		if s.Latlng.Lng != nil {
			latlng["lng"] = *s.Latlng.Lng
		}
		if len(latlng) > 0 {
			result["latlng"] = latlng
		}
	}

	// Add status fields
	if s.CreatedTime != nil {
		result["created_time"] = *s.CreatedTime
	}
	if s.ModifiedTime != nil {
		result["modified_time"] = *s.ModifiedTime
	}
	if s.OrgID != nil {
		result["org_id"] = *s.OrgID
	}

	// Add template IDs
	if s.AlarmTemplateID != nil {
		result["alarmtemplate_id"] = *s.AlarmTemplateID
	}
	if s.APTemplateID != nil {
		result["aptemplate_id"] = *s.APTemplateID
	}
	if s.GatewayTemplateID != nil {
		result["gatewaytemplate_id"] = *s.GatewayTemplateID
	}
	if s.NetworkTemplateID != nil {
		result["networktemplate_id"] = *s.NetworkTemplateID
	}
	if s.RFTemplateID != nil {
		result["rftemplate_id"] = *s.RFTemplateID
	}
	if s.SecPolicyID != nil {
		result["secpolicy_id"] = *s.SecPolicyID
	}
	if s.SiteTemplateID != nil {
		result["sitetemplate_id"] = *s.SiteTemplateID
	}

	// Add site groups and variables
	if s.SiteGroupIDs != nil {
		result["sitegroup_ids"] = *s.SiteGroupIDs
	}
	if s.Vars != nil {
		result["vars"] = s.Vars
	}

	// Add complex configuration objects (preserve exact structure)
	if s.Setting != nil {
		result["setting"] = s.Setting
	}
	if s.SLE != nil {
		result["sle"] = s.SLE
	}
	if s.Stats != nil {
		result["stats"] = s.Stats
	}
	if s.AutoUpgrade != nil {
		result["auto_upgrade"] = s.AutoUpgrade
	}
	if s.Engage != nil {
		result["engage"] = s.Engage
	}
	if s.EvpnOptions != nil {
		result["evpn_options"] = s.EvpnOptions
	}
	if s.Rogue != nil {
		result["rogue"] = s.Rogue
	}
	if s.WanVNA != nil {
		result["wan_vna"] = s.WanVNA
	}
	if s.SkyATP != nil {
		result["skyatp"] = s.SkyATP
	}
	if s.WiredVNA != nil {
		result["wiredvna"] = s.WiredVNA
	}

	// Add simple limit fields
	if s.MSPID != nil {
		result["msp_id"] = *s.MSPID
	}
	if s.WlanLimitDown != nil {
		result["wlan_limit_down"] = *s.WlanLimitDown
	}
	if s.WlanLimitUp != nil {
		result["wlan_limit_up"] = *s.WlanLimitUp
	}
	if s.ManagedAPDeviceLimit != nil {
		result["managed_ap_device_limit"] = *s.ManagedAPDeviceLimit
	}
	if s.ManagedOtherDeviceLimit != nil {
		result["managed_other_device_limit"] = *s.ManagedOtherDeviceLimit
	}
	if s.WxTagIDs != nil {
		result["wxtag_ids"] = *s.WxTagIDs
	}
	if s.WxTunnelID != nil {
		result["wxtunnel_id"] = *s.WxTunnelID
	}
	if s.TzOffset != nil {
		result["tzoffset"] = *s.TzOffset
	}

	// Add additional configuration
	for key, value := range s.AdditionalConfig {
		result[key] = value
	}

	// Raw field removed - all data is now in struct fields or AdditionalConfig

	return result
}

// FromMap populates the site from a map representation (e.g., from API response)
func (s *MistSite) FromMap(data map[string]interface{}) error {
	if data == nil {
		return fmt.Errorf("data cannot be nil")
	}

	// Initialize additional config if nil
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

	// Handle latlng
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

	// Status fields
	if createdTime, ok := data["created_time"].(float64); ok {
		s.CreatedTime = &createdTime
	}
	if modifiedTime, ok := data["modified_time"].(float64); ok {
		s.ModifiedTime = &modifiedTime
	}
	if orgID, ok := data["org_id"].(string); ok {
		s.OrgID = &orgID
	}

	// Template IDs
	if alarmTemplateID, ok := data["alarmtemplate_id"].(string); ok {
		s.AlarmTemplateID = &alarmTemplateID
	}
	if apTemplateID, ok := data["aptemplate_id"].(string); ok {
		s.APTemplateID = &apTemplateID
	}
	if gatewayTemplateID, ok := data["gatewaytemplate_id"].(string); ok {
		s.GatewayTemplateID = &gatewayTemplateID
	}
	if networkTemplateID, ok := data["networktemplate_id"].(string); ok {
		s.NetworkTemplateID = &networkTemplateID
	}
	if rfTemplateID, ok := data["rftemplate_id"].(string); ok {
		s.RFTemplateID = &rfTemplateID
	}
	if secPolicyID, ok := data["secpolicy_id"].(string); ok {
		s.SecPolicyID = &secPolicyID
	}
	if siteTemplateID, ok := data["sitetemplate_id"].(string); ok {
		s.SiteTemplateID = &siteTemplateID
	}

	// Site groups
	if siteGroupIDs, ok := data["sitegroup_ids"].([]interface{}); ok {
		var siteGroupIDStrs []string
		for _, id := range siteGroupIDs {
			if idStr, ok := id.(string); ok {
				siteGroupIDStrs = append(siteGroupIDStrs, idStr)
			}
		}
		if len(siteGroupIDStrs) > 0 {
			s.SiteGroupIDs = &siteGroupIDStrs
		}
	}

	// Variables
	if vars, ok := data["vars"].(map[string]interface{}); ok {
		s.Vars = vars
	}

	// Complex configuration objects (preserve exact structure like APDevice)
	configFields := map[string]*map[string]interface{}{
		"setting":      &s.Setting,
		"sle":          &s.SLE,
		"stats":        &s.Stats,
		"auto_upgrade": &s.AutoUpgrade,
		"engage":       &s.Engage,
		"evpn_options": &s.EvpnOptions,
		"rogue":        &s.Rogue,
		"wan_vna":      &s.WanVNA,
		"skyatp":       &s.SkyATP,
		"wiredvna":     &s.WiredVNA,
	}

	for fieldName, configPtr := range configFields {
		if configData, ok := data[fieldName].(map[string]interface{}); ok {
			*configPtr = configData
		}
	}

	// Simple fields
	if mspID, ok := data["msp_id"].(string); ok {
		s.MSPID = &mspID
	}
	if wlanLimitDown, ok := data["wlan_limit_down"].(float64); ok {
		limitInt := int(wlanLimitDown)
		s.WlanLimitDown = &limitInt
	}
	if wlanLimitUp, ok := data["wlan_limit_up"].(float64); ok {
		limitInt := int(wlanLimitUp)
		s.WlanLimitUp = &limitInt
	}
	if managedAPLimit, ok := data["managed_ap_device_limit"].(float64); ok {
		limitInt := int(managedAPLimit)
		s.ManagedAPDeviceLimit = &limitInt
	}
	if managedOtherLimit, ok := data["managed_other_device_limit"].(float64); ok {
		limitInt := int(managedOtherLimit)
		s.ManagedOtherDeviceLimit = &limitInt
	}

	// WxTag IDs
	if wxTagIDs, ok := data["wxtag_ids"].([]interface{}); ok {
		var wxTagIDStrs []string
		for _, id := range wxTagIDs {
			if idStr, ok := id.(string); ok {
				wxTagIDStrs = append(wxTagIDStrs, idStr)
			}
		}
		if len(wxTagIDStrs) > 0 {
			s.WxTagIDs = &wxTagIDStrs
		}
	}

	if wxTunnelID, ok := data["wxtunnel_id"].(string); ok {
		s.WxTunnelID = &wxTunnelID
	}

	if tzOffset, ok := data["tzoffset"].(float64); ok {
		offsetInt := int(tzOffset)
		s.TzOffset = &offsetInt
	}

	// Store any unknown fields in AdditionalConfig (following APDevice pattern)
	knownFields := map[string]bool{
		// Core fields
		"id": true, "name": true, "address": true, "country_code": true,
		"timezone": true, "notes": true, "latlng": true,
		// Status fields
		"created_time": true, "modified_time": true, "org_id": true,
		// Template IDs
		"alarmtemplate_id": true, "aptemplate_id": true, "gatewaytemplate_id": true,
		"networktemplate_id": true, "rftemplate_id": true, "secpolicy_id": true,
		"sitetemplate_id": true,
		// Groups and variables
		"sitegroup_ids": true, "vars": true,
		// Complex config objects
		"setting": true, "sle": true, "stats": true, "auto_upgrade": true,
		"engage": true, "evpn_options": true, "rogue": true,
		"wan_vna": true, "skyatp": true, "wiredvna": true,
		// Simple fields
		"msp_id": true, "wlan_limit_down": true, "wlan_limit_up": true,
		"managed_ap_device_limit": true, "managed_other_device_limit": true,
		"wxtag_ids": true, "wxtunnel_id": true, "tzoffset": true,
	}

	for k, v := range data {
		if !knownFields[k] {
			s.AdditionalConfig[k] = v
		}
	}

	return nil
}

// ToConfigMap converts the site to a map suitable for configuration files (excludes status fields)
func (s *MistSite) ToConfigMap() map[string]interface{} {
	result := make(map[string]interface{})

	// Add only configuration fields (exclude status fields)
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
	if s.Latlng != nil {
		latlng := make(map[string]interface{})
		if s.Latlng.Lat != nil {
			latlng["lat"] = *s.Latlng.Lat
		}
		if s.Latlng.Lng != nil {
			latlng["lng"] = *s.Latlng.Lng
		}
		if len(latlng) > 0 {
			result["latlng"] = latlng
		}
	}

	// Add template IDs (configuration fields)
	if s.AlarmTemplateID != nil {
		result["alarmtemplate_id"] = *s.AlarmTemplateID
	}
	if s.APTemplateID != nil {
		result["aptemplate_id"] = *s.APTemplateID
	}
	if s.GatewayTemplateID != nil {
		result["gatewaytemplate_id"] = *s.GatewayTemplateID
	}
	if s.NetworkTemplateID != nil {
		result["networktemplate_id"] = *s.NetworkTemplateID
	}
	if s.RFTemplateID != nil {
		result["rftemplate_id"] = *s.RFTemplateID
	}
	if s.SecPolicyID != nil {
		result["secpolicy_id"] = *s.SecPolicyID
	}
	if s.SiteTemplateID != nil {
		result["sitetemplate_id"] = *s.SiteTemplateID
	}

	// Add site groups and variables (configuration fields)
	if s.SiteGroupIDs != nil {
		result["sitegroup_ids"] = *s.SiteGroupIDs
	}
	if s.Vars != nil {
		result["vars"] = s.Vars
	}

	// Add complex configuration objects
	if s.Setting != nil {
		result["setting"] = s.Setting
	}
	if s.SLE != nil {
		result["sle"] = s.SLE
	}
	if s.AutoUpgrade != nil {
		result["auto_upgrade"] = s.AutoUpgrade
	}
	if s.Engage != nil {
		result["engage"] = s.Engage
	}
	if s.EvpnOptions != nil {
		result["evpn_options"] = s.EvpnOptions
	}
	if s.Rogue != nil {
		result["rogue"] = s.Rogue
	}
	if s.WanVNA != nil {
		result["wan_vna"] = s.WanVNA
	}
	if s.SkyATP != nil {
		result["skyatp"] = s.SkyATP
	}
	if s.WiredVNA != nil {
		result["wiredvna"] = s.WiredVNA
	}

	// Add simple limit fields (configuration fields)
	if s.MSPID != nil {
		result["msp_id"] = *s.MSPID
	}
	if s.WlanLimitDown != nil {
		result["wlan_limit_down"] = *s.WlanLimitDown
	}
	if s.WlanLimitUp != nil {
		result["wlan_limit_up"] = *s.WlanLimitUp
	}
	if s.ManagedAPDeviceLimit != nil {
		result["managed_ap_device_limit"] = *s.ManagedAPDeviceLimit
	}
	if s.ManagedOtherDeviceLimit != nil {
		result["managed_other_device_limit"] = *s.ManagedOtherDeviceLimit
	}
	if s.WxTagIDs != nil {
		result["wxtag_ids"] = *s.WxTagIDs
	}
	if s.WxTunnelID != nil {
		result["wxtunnel_id"] = *s.WxTunnelID
	}
	if s.TzOffset != nil {
		result["tzoffset"] = *s.TzOffset
	}

	// Add additional configuration (excluding status fields)
	statusFields := map[string]bool{
		"id":            true,
		"created_time":  true,
		"modified_time": true,
		"org_id":        true,
		"stats":         true, // stats is read-only, should be excluded from config
	}

	for key, value := range s.AdditionalConfig {
		if !statusFields[key] {
			result[key] = value
		}
	}

	return result
}

// FromConfigMap populates the site from a configuration map (excludes status fields)
func (s *MistSite) FromConfigMap(data map[string]interface{}) error {
	if data == nil {
		return fmt.Errorf("data cannot be nil")
	}

	// Initialize additional config if nil
	if s.AdditionalConfig == nil {
		s.AdditionalConfig = make(map[string]interface{})
	}

	// Extract typed fields
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

	// Handle latlng
	if latlngData, ok := data["latlng"].(map[string]interface{}); ok {
		s.Latlng = &MistLatLng{}
		if lat, ok := latlngData["lat"].(float64); ok {
			s.Latlng.Lat = &lat
		}
		if lng, ok := latlngData["lng"].(float64); ok {
			s.Latlng.Lng = &lng
		}
	}

	// Template IDs
	if alarmTemplateID, ok := data["alarmtemplate_id"].(string); ok {
		s.AlarmTemplateID = &alarmTemplateID
	}
	if apTemplateID, ok := data["aptemplate_id"].(string); ok {
		s.APTemplateID = &apTemplateID
	}
	if gatewayTemplateID, ok := data["gatewaytemplate_id"].(string); ok {
		s.GatewayTemplateID = &gatewayTemplateID
	}
	if networkTemplateID, ok := data["networktemplate_id"].(string); ok {
		s.NetworkTemplateID = &networkTemplateID
	}
	if rfTemplateID, ok := data["rftemplate_id"].(string); ok {
		s.RFTemplateID = &rfTemplateID
	}
	if secPolicyID, ok := data["secpolicy_id"].(string); ok {
		s.SecPolicyID = &secPolicyID
	}
	if siteTemplateID, ok := data["sitetemplate_id"].(string); ok {
		s.SiteTemplateID = &siteTemplateID
	}

	// Site groups
	if siteGroupIDs, ok := data["sitegroup_ids"].([]interface{}); ok {
		var siteGroupIDStrs []string
		for _, id := range siteGroupIDs {
			if idStr, ok := id.(string); ok {
				siteGroupIDStrs = append(siteGroupIDStrs, idStr)
			}
		}
		if len(siteGroupIDStrs) > 0 {
			s.SiteGroupIDs = &siteGroupIDStrs
		}
	}

	// Variables
	if vars, ok := data["vars"].(map[string]interface{}); ok {
		s.Vars = vars
	}

	// Complex configuration objects (preserve exact structure like APDevice)
	configFields := map[string]*map[string]interface{}{
		"setting":      &s.Setting,
		"sle":          &s.SLE,
		"auto_upgrade": &s.AutoUpgrade,
		"engage":       &s.Engage,
		"evpn_options": &s.EvpnOptions,
		"rogue":        &s.Rogue,
		"wan_vna":      &s.WanVNA,
		"skyatp":       &s.SkyATP,
		"wiredvna":     &s.WiredVNA,
	}

	for fieldName, configPtr := range configFields {
		if configData, ok := data[fieldName].(map[string]interface{}); ok {
			*configPtr = configData
		}
	}

	// Simple fields
	if mspID, ok := data["msp_id"].(string); ok {
		s.MSPID = &mspID
	}
	if wlanLimitDown, ok := data["wlan_limit_down"].(float64); ok {
		limitInt := int(wlanLimitDown)
		s.WlanLimitDown = &limitInt
	}
	if wlanLimitUp, ok := data["wlan_limit_up"].(float64); ok {
		limitInt := int(wlanLimitUp)
		s.WlanLimitUp = &limitInt
	}
	if managedAPLimit, ok := data["managed_ap_device_limit"].(float64); ok {
		limitInt := int(managedAPLimit)
		s.ManagedAPDeviceLimit = &limitInt
	}
	if managedOtherLimit, ok := data["managed_other_device_limit"].(float64); ok {
		limitInt := int(managedOtherLimit)
		s.ManagedOtherDeviceLimit = &limitInt
	}

	// WxTag IDs
	if wxTagIDs, ok := data["wxtag_ids"].([]interface{}); ok {
		var wxTagIDStrs []string
		for _, id := range wxTagIDs {
			if idStr, ok := id.(string); ok {
				wxTagIDStrs = append(wxTagIDStrs, idStr)
			}
		}
		if len(wxTagIDStrs) > 0 {
			s.WxTagIDs = &wxTagIDStrs
		}
	}

	if wxTunnelID, ok := data["wxtunnel_id"].(string); ok {
		s.WxTunnelID = &wxTunnelID
	}

	if tzOffset, ok := data["tzoffset"].(float64); ok {
		offsetInt := int(tzOffset)
		s.TzOffset = &offsetInt
	}

	// Store any additional fields in AdditionalConfig (exclude status fields)
	knownConfigFields := map[string]bool{
		// Core fields
		"name": true, "address": true, "country_code": true,
		"timezone": true, "notes": true, "latlng": true,
		// Template IDs
		"alarmtemplate_id": true, "aptemplate_id": true, "gatewaytemplate_id": true,
		"networktemplate_id": true, "rftemplate_id": true, "secpolicy_id": true,
		"sitetemplate_id": true,
		// Groups and variables
		"sitegroup_ids": true, "vars": true,
		// Complex config objects
		"setting": true, "sle": true, "auto_upgrade": true,
		"engage": true, "evpn_options": true, "rogue": true,
		"wan_vna": true, "skyatp": true, "wiredvna": true,
		// Simple fields
		"msp_id": true, "wlan_limit_down": true, "wlan_limit_up": true,
		"managed_ap_device_limit": true, "managed_other_device_limit": true,
		"wxtag_ids": true, "wxtunnel_id": true, "tzoffset": true,
	}

	statusFields := map[string]bool{
		"id":            true,
		"created_time":  true,
		"modified_time": true,
		"org_id":        true,
		"stats":         true, // stats is read-only, should be excluded from config
	}

	for key, value := range data {
		if !knownConfigFields[key] && !statusFields[key] {
			s.AdditionalConfig[key] = value
		}
	}

	return nil
}

// GetRaw returns the raw API data (reconstructed from struct fields)
func (s *MistSite) GetRaw() map[string]interface{} {
	// Return complete API representation from struct fields
	return s.ToMap()
}

// SetRaw sets the raw API data (populates struct fields from map)
func (s *MistSite) SetRaw(data map[string]interface{}) {
	// Populate struct fields from raw data
	_ = s.FromMap(data)
}

// NewSiteFromMap creates a new site from a map representation
func NewSiteFromMap(data map[string]interface{}) (*MistSite, error) {
	site := &MistSite{
		AdditionalConfig: make(map[string]interface{}),
	}

	if err := site.FromMap(data); err != nil {
		return nil, fmt.Errorf("failed to create site from map: %w", err)
	}

	return site, nil
}

// NewSiteFromConfigMap creates a new site from a configuration map
func NewSiteFromConfigMap(data map[string]interface{}) (*MistSite, error) {
	site := &MistSite{
		AdditionalConfig: make(map[string]interface{}),
	}

	if err := site.FromConfigMap(data); err != nil {
		return nil, fmt.Errorf("failed to create site from config map: %w", err)
	}

	return site, nil
}

// ConvertSiteToNew converts a legacy Site to MistSite
func ConvertSiteToNew(oldSite Site) *MistSite {
	site := &MistSite{
		AdditionalConfig: make(map[string]interface{}),
	}

	// Convert typed fields
	if oldSite.Id != nil {
		id := string(*oldSite.Id)
		site.ID = &id
	}
	if oldSite.Name != nil {
		site.Name = oldSite.Name
	}
	if oldSite.Address != nil {
		site.Address = oldSite.Address
	}
	if oldSite.CountryCode != nil {
		site.CountryCode = oldSite.CountryCode
	}
	if oldSite.Timezone != nil {
		site.Timezone = oldSite.Timezone
	}
	if oldSite.Notes != nil {
		site.Notes = oldSite.Notes
	}
	if oldSite.Latlng != nil {
		site.Latlng = &MistLatLng{
			Lat: &oldSite.Latlng.Lat,
			Lng: &oldSite.Latlng.Lng,
		}
	}

	// Convert status fields
	site.CreatedTime = oldSite.CreatedTime
	site.ModifiedTime = oldSite.ModifiedTime

	// Raw field removed - all data preserved in struct fields

	return site
}

// ConvertSiteFromNew converts a MistSite back to legacy Site
func ConvertSiteFromNew(newSite *MistSite) Site {
	site := Site{}

	// Convert typed fields
	if newSite.ID != nil {
		uuid := UUID(*newSite.ID)
		site.Id = &uuid
	}
	if newSite.Name != nil {
		site.Name = newSite.Name
	}
	if newSite.Address != nil {
		site.Address = newSite.Address
	}
	if newSite.CountryCode != nil {
		site.CountryCode = newSite.CountryCode
	}
	if newSite.Timezone != nil {
		site.Timezone = newSite.Timezone
	}
	if newSite.Notes != nil {
		site.Notes = newSite.Notes
	}
	if newSite.Latlng != nil {
		site.Latlng = &LatLng{}
		if newSite.Latlng.Lat != nil {
			site.Latlng.Lat = *newSite.Latlng.Lat
		}
		if newSite.Latlng.Lng != nil {
			site.Latlng.Lng = *newSite.Latlng.Lng
		}
	}

	// Convert status fields
	site.CreatedTime = newSite.CreatedTime
	site.ModifiedTime = newSite.ModifiedTime

	return site
}

// Verify MistSite implements SiteMarshaler at compile time
var _ SiteMarshaler = (*MistSite)(nil)

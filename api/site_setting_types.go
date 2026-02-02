package api

import (
	"fmt"
)

// SiteSetting represents a Mist site setting with comprehensive field mapping
// This follows the comprehensive FromMap/ToMap pattern to preserve exact API response structure
type SiteSetting struct {
	// Core identification fields
	ID           *string  `json:"id,omitempty"`
	SiteID       *string  `json:"site_id,omitempty"`
	OrgID        *string  `json:"org_id,omitempty"`
	CreatedTime  *float64 `json:"created_time,omitempty"`
	ModifiedTime *float64 `json:"modified_time,omitempty"`
	ForSite      *bool    `json:"for_site,omitempty"`

	// Simple configuration fields
	AdditionalConfigCmds        *[]string `json:"additional_config_cmds,omitempty"`
	GatewayAdditionalConfigCmds *[]string `json:"gateway_additional_config_cmds,omitempty"`
	APUpdownThreshold           *int      `json:"ap_updown_threshold,omitempty"`
	AutoUpgradeLinecard         *bool     `json:"auto_upgrade_linecard,omitempty"`
	DenylistURL                 *string   `json:"blacklist_url,omitempty"` // Mist API field name
	ConfigAutoRevert            *bool     `json:"config_auto_revert,omitempty"`
	DefaultPortUsage            *string   `json:"default_port_usage,omitempty"`
	DeviceUpdownThreshold       *int      `json:"device_updown_threshold,omitempty"`
	DNSServers                  *[]string `json:"dns_servers,omitempty"`
	DNSSuffix                   *[]string `json:"dns_suffix,omitempty"`
	EnableUnii4                 *bool     `json:"enable_unii_4,omitempty"`
	GatewayUpdownThreshold      *int      `json:"gateway_updown_threshold,omitempty"`
	NTPServers                  *[]string `json:"ntp_servers,omitempty"`
	PersistConfigOnDevice       *bool     `json:"persist_config_on_device,omitempty"`
	RemoveExistingConfigs       *bool     `json:"remove_existing_configs,omitempty"`
	ReportGatt                  *bool     `json:"report_gatt,omitempty"`
	SSHKeys                     *[]string `json:"ssh_keys,omitempty"`
	SwitchUpdownThreshold       *int      `json:"switch_updown_threshold,omitempty"`
	TrackAnonymousDevices       *bool     `json:"track_anonymous_devices,omitempty"`
	TuntermMonitoringDisabled   *bool     `json:"tunterm_monitoring_disabled,omitempty"`
	WatchedStationURL           *string   `json:"watched_station_url,omitempty"`
	AllowlistURL                *string   `json:"whitelist_url,omitempty"` // Mist API field name

	// Array fields with complex objects
	ACLPolicies                     []map[string]interface{} `json:"acl_policies,omitempty"`
	DisabledSystemDefinedPortUsages *[]string                `json:"disabled_system_defined_port_usages,omitempty"`
	TuntermMonitoring               []map[string]interface{} `json:"tunterm_monitoring,omitempty"`

	// Complex nested configuration objects (stored as maps to preserve exact API structure)
	ACLTags                map[string]interface{} `json:"acl_tags,omitempty"`
	Analytic               map[string]interface{} `json:"analytic,omitempty"`
	APMatching             map[string]interface{} `json:"ap_matching,omitempty"`
	APPortConfig           map[string]interface{} `json:"ap_port_config,omitempty"`
	AutoPlacement          map[string]interface{} `json:"auto_placement,omitempty"`
	AutoUpgrade            map[string]interface{} `json:"auto_upgrade,omitempty"`
	BLEConfig              map[string]interface{} `json:"ble_config,omitempty"`
	ConfigPushPolicy       map[string]interface{} `json:"config_push_policy,omitempty"`
	CriticalURLMonitoring  map[string]interface{} `json:"critical_url_monitoring,omitempty"`
	DHCPSnooping           map[string]interface{} `json:"dhcp_snooping,omitempty"`
	Engagement             map[string]interface{} `json:"engagement,omitempty"`
	EVPNOptions            map[string]interface{} `json:"evpn_options,omitempty"`
	ExtraRoutes            map[string]interface{} `json:"extra_routes,omitempty"`
	ExtraRoutes6           map[string]interface{} `json:"extra_routes6,omitempty"`
	Flags                  map[string]interface{} `json:"flags,omitempty"`
	Gateway                map[string]interface{} `json:"gateway,omitempty"`
	GatewayMgmt            map[string]interface{} `json:"gateway_mgmt,omitempty"`
	JuniperSRX             map[string]interface{} `json:"juniper_srx,omitempty"`
	LED                    map[string]interface{} `json:"led,omitempty"`
	Marvis                 map[string]interface{} `json:"marvis,omitempty"`
	MistNAC                map[string]interface{} `json:"mist_nac,omitempty"`
	MXEdge                 map[string]interface{} `json:"mxedge,omitempty"`
	MXEdgeMgmt             map[string]interface{} `json:"mxedge_mgmt,omitempty"`
	MXTunnels              map[string]interface{} `json:"mxtunnels,omitempty"`
	Networks               map[string]interface{} `json:"networks,omitempty"`
	Occupancy              map[string]interface{} `json:"occupancy,omitempty"`
	OSPFAreas              map[string]interface{} `json:"ospf_areas,omitempty"`
	PaloAltoNetworks       map[string]interface{} `json:"paloalto_networks,omitempty"`
	PortMirroring          map[string]interface{} `json:"port_mirroring,omitempty"`
	PortUsages             map[string]interface{} `json:"port_usages,omitempty"`
	Proxy                  map[string]interface{} `json:"proxy,omitempty"`
	RadioConfig            map[string]interface{} `json:"radio_config,omitempty"`
	RadiusConfig           map[string]interface{} `json:"radius_config,omitempty"`
	RemoteSyslog           map[string]interface{} `json:"remote_syslog,omitempty"`
	Rogue                  map[string]interface{} `json:"rogue,omitempty"`
	RTSA                   map[string]interface{} `json:"rtsa,omitempty"`
	SimpleAlert            map[string]interface{} `json:"simple_alert,omitempty"`
	SkyATP                 map[string]interface{} `json:"skyatp,omitempty"`
	SLEThresholds          map[string]interface{} `json:"sle_thresholds,omitempty"`
	SNMPConfig             map[string]interface{} `json:"snmp_config,omitempty"`
	SRXApp                 map[string]interface{} `json:"srx_app,omitempty"`
	SSR                    map[string]interface{} `json:"ssr,omitempty"`
	StatusPortal           map[string]interface{} `json:"status_portal,omitempty"`
	Switch                 map[string]interface{} `json:"switch,omitempty"`
	SwitchMatching         map[string]interface{} `json:"switch_matching,omitempty"`
	SwitchMgmt             map[string]interface{} `json:"switch_mgmt,omitempty"`
	SyntheticTest          map[string]interface{} `json:"synthetic_test,omitempty"`
	TuntermMulticastConfig map[string]interface{} `json:"tunterm_multicast_config,omitempty"`
	UplinkPortConfig       map[string]interface{} `json:"uplink_port_config,omitempty"`
	Vars                   map[string]interface{} `json:"vars,omitempty"`
	VNA                    map[string]interface{} `json:"vna,omitempty"`
	VRFConfig              map[string]interface{} `json:"vrf_config,omitempty"`
	VRFInstances           map[string]interface{} `json:"vrf_instances,omitempty"`
	VRRPGroups             map[string]interface{} `json:"vrrp_groups,omitempty"`
	VSInstance             map[string]interface{} `json:"vs_instance,omitempty"`
	WANVNA                 map[string]interface{} `json:"wan_vna,omitempty"`
	WIDS                   map[string]interface{} `json:"wids,omitempty"`
	WiFi                   map[string]interface{} `json:"wifi,omitempty"`
	WiredVNA               map[string]interface{} `json:"wired_vna,omitempty"`
	ZoneOccupancyAlert     map[string]interface{} `json:"zone_occupancy_alert,omitempty"`

	// Additional configuration for any unmapped fields
	AdditionalConfig map[string]interface{} `json:"-"`
}

// GetID returns the site setting ID
func (ss *SiteSetting) GetID() string {
	if ss.ID != nil {
		return *ss.ID
	}
	return ""
}

// GetSiteID returns the site ID
func (ss *SiteSetting) GetSiteID() string {
	if ss.SiteID != nil {
		return *ss.SiteID
	}
	return ""
}

// ToMap converts the site setting to a map representation suitable for API requests
func (ss *SiteSetting) ToMap() map[string]interface{} {
	result := make(map[string]interface{})

	// Add core identification fields
	if ss.ID != nil {
		result["id"] = *ss.ID
	}
	if ss.SiteID != nil {
		result["site_id"] = *ss.SiteID
	}
	if ss.OrgID != nil {
		result["org_id"] = *ss.OrgID
	}
	if ss.CreatedTime != nil {
		result["created_time"] = *ss.CreatedTime
	}
	if ss.ModifiedTime != nil {
		result["modified_time"] = *ss.ModifiedTime
	}
	if ss.ForSite != nil {
		result["for_site"] = *ss.ForSite
	}

	// Add simple configuration fields
	if ss.AdditionalConfigCmds != nil {
		result["additional_config_cmds"] = *ss.AdditionalConfigCmds
	}
	if ss.GatewayAdditionalConfigCmds != nil {
		result["gateway_additional_config_cmds"] = *ss.GatewayAdditionalConfigCmds
	}
	if ss.APUpdownThreshold != nil {
		result["ap_updown_threshold"] = *ss.APUpdownThreshold
	}
	if ss.AutoUpgradeLinecard != nil {
		result["auto_upgrade_linecard"] = *ss.AutoUpgradeLinecard
	}
	if ss.DenylistURL != nil {
		result["blacklist_url"] = *ss.DenylistURL // Mist API field name
	}
	if ss.ConfigAutoRevert != nil {
		result["config_auto_revert"] = *ss.ConfigAutoRevert
	}
	if ss.DefaultPortUsage != nil {
		result["default_port_usage"] = *ss.DefaultPortUsage
	}
	if ss.DeviceUpdownThreshold != nil {
		result["device_updown_threshold"] = *ss.DeviceUpdownThreshold
	}
	if ss.DNSServers != nil {
		result["dns_servers"] = *ss.DNSServers
	}
	if ss.DNSSuffix != nil {
		result["dns_suffix"] = *ss.DNSSuffix
	}
	if ss.EnableUnii4 != nil {
		result["enable_unii_4"] = *ss.EnableUnii4
	}
	if ss.GatewayUpdownThreshold != nil {
		result["gateway_updown_threshold"] = *ss.GatewayUpdownThreshold
	}
	if ss.NTPServers != nil {
		result["ntp_servers"] = *ss.NTPServers
	}
	if ss.PersistConfigOnDevice != nil {
		result["persist_config_on_device"] = *ss.PersistConfigOnDevice
	}
	if ss.RemoveExistingConfigs != nil {
		result["remove_existing_configs"] = *ss.RemoveExistingConfigs
	}
	if ss.ReportGatt != nil {
		result["report_gatt"] = *ss.ReportGatt
	}
	if ss.SSHKeys != nil {
		result["ssh_keys"] = *ss.SSHKeys
	}
	if ss.SwitchUpdownThreshold != nil {
		result["switch_updown_threshold"] = *ss.SwitchUpdownThreshold
	}
	if ss.TrackAnonymousDevices != nil {
		result["track_anonymous_devices"] = *ss.TrackAnonymousDevices
	}
	if ss.TuntermMonitoringDisabled != nil {
		result["tunterm_monitoring_disabled"] = *ss.TuntermMonitoringDisabled
	}
	if ss.WatchedStationURL != nil {
		result["watched_station_url"] = *ss.WatchedStationURL
	}
	if ss.AllowlistURL != nil {
		result["whitelist_url"] = *ss.AllowlistURL // Mist API field name
	}

	// Add array fields
	if ss.ACLPolicies != nil {
		result["acl_policies"] = ss.ACLPolicies
	}
	if ss.DisabledSystemDefinedPortUsages != nil {
		result["disabled_system_defined_port_usages"] = *ss.DisabledSystemDefinedPortUsages
	}
	if ss.TuntermMonitoring != nil {
		result["tunterm_monitoring"] = ss.TuntermMonitoring
	}

	// Add complex nested configuration objects (preserve exact structure)
	complexFields := map[string]map[string]interface{}{
		"acl_tags": ss.ACLTags, "analytic": ss.Analytic, "ap_matching": ss.APMatching,
		"ap_port_config": ss.APPortConfig, "auto_placement": ss.AutoPlacement, "auto_upgrade": ss.AutoUpgrade,
		"ble_config": ss.BLEConfig, "config_push_policy": ss.ConfigPushPolicy, "critical_url_monitoring": ss.CriticalURLMonitoring,
		"dhcp_snooping": ss.DHCPSnooping, "engagement": ss.Engagement, "evpn_options": ss.EVPNOptions,
		"extra_routes": ss.ExtraRoutes, "extra_routes6": ss.ExtraRoutes6, "flags": ss.Flags,
		"gateway": ss.Gateway, "gateway_mgmt": ss.GatewayMgmt, "juniper_srx": ss.JuniperSRX,
		"led": ss.LED, "marvis": ss.Marvis, "mist_nac": ss.MistNAC,
		"mxedge": ss.MXEdge, "mxedge_mgmt": ss.MXEdgeMgmt, "mxtunnels": ss.MXTunnels,
		"networks": ss.Networks, "occupancy": ss.Occupancy, "ospf_areas": ss.OSPFAreas,
		"paloalto_networks": ss.PaloAltoNetworks, "port_mirroring": ss.PortMirroring, "port_usages": ss.PortUsages,
		"proxy": ss.Proxy, "radio_config": ss.RadioConfig, "radius_config": ss.RadiusConfig,
		"remote_syslog": ss.RemoteSyslog, "rogue": ss.Rogue, "rtsa": ss.RTSA,
		"simple_alert": ss.SimpleAlert, "skyatp": ss.SkyATP, "sle_thresholds": ss.SLEThresholds,
		"snmp_config": ss.SNMPConfig, "srx_app": ss.SRXApp, "ssr": ss.SSR,
		"status_portal": ss.StatusPortal, "switch": ss.Switch, "switch_matching": ss.SwitchMatching,
		"switch_mgmt": ss.SwitchMgmt, "synthetic_test": ss.SyntheticTest, "tunterm_multicast_config": ss.TuntermMulticastConfig,
		"uplink_port_config": ss.UplinkPortConfig, "vars": ss.Vars, "vna": ss.VNA,
		"vrf_config": ss.VRFConfig, "vrf_instances": ss.VRFInstances, "vrrp_groups": ss.VRRPGroups,
		"vs_instance": ss.VSInstance, "wan_vna": ss.WANVNA, "wids": ss.WIDS,
		"wifi": ss.WiFi, "wired_vna": ss.WiredVNA, "zone_occupancy_alert": ss.ZoneOccupancyAlert,
	}

	for fieldName, configMap := range complexFields {
		if configMap != nil {
			result[fieldName] = configMap
		}
	}

	// Add additional configuration
	for key, value := range ss.AdditionalConfig {
		result[key] = value
	}

	return result
}

// FromMap populates the site setting from a map representation (e.g., from API response)
func (ss *SiteSetting) FromMap(data map[string]interface{}) error {
	if data == nil {
		return fmt.Errorf("data cannot be nil")
	}

	// Initialize additional config if nil
	if ss.AdditionalConfig == nil {
		ss.AdditionalConfig = make(map[string]interface{})
	}

	// Extract core identification fields
	if id, ok := data["id"].(string); ok {
		ss.ID = &id
	}
	if siteID, ok := data["site_id"].(string); ok {
		ss.SiteID = &siteID
	}
	if orgID, ok := data["org_id"].(string); ok {
		ss.OrgID = &orgID
	}
	if createdTime, ok := data["created_time"].(float64); ok {
		ss.CreatedTime = &createdTime
	}
	if modifiedTime, ok := data["modified_time"].(float64); ok {
		ss.ModifiedTime = &modifiedTime
	}
	if forSite, ok := data["for_site"].(bool); ok {
		ss.ForSite = &forSite
	}

	// Extract simple configuration fields
	if additionalConfigCmds, ok := data["additional_config_cmds"].([]interface{}); ok {
		var cmds []string
		for _, cmd := range additionalConfigCmds {
			if cmdStr, ok := cmd.(string); ok {
				cmds = append(cmds, cmdStr)
			}
		}
		if len(cmds) > 0 {
			ss.AdditionalConfigCmds = &cmds
		}
	}

	if gatewayAdditionalConfigCmds, ok := data["gateway_additional_config_cmds"].([]interface{}); ok {
		var cmds []string
		for _, cmd := range gatewayAdditionalConfigCmds {
			if cmdStr, ok := cmd.(string); ok {
				cmds = append(cmds, cmdStr)
			}
		}
		if len(cmds) > 0 {
			ss.GatewayAdditionalConfigCmds = &cmds
		}
	}

	if apUpdownThreshold, ok := data["ap_updown_threshold"].(float64); ok {
		threshold := int(apUpdownThreshold)
		ss.APUpdownThreshold = &threshold
	}
	if autoUpgradeLinecard, ok := data["auto_upgrade_linecard"].(bool); ok {
		ss.AutoUpgradeLinecard = &autoUpgradeLinecard
	}
	if blacklistURL, ok := data["blacklist_url"].(string); ok { // Mist API field name
		ss.DenylistURL = &blacklistURL
	}
	if configAutoRevert, ok := data["config_auto_revert"].(bool); ok {
		ss.ConfigAutoRevert = &configAutoRevert
	}
	if defaultPortUsage, ok := data["default_port_usage"].(string); ok {
		ss.DefaultPortUsage = &defaultPortUsage
	}
	if deviceUpdownThreshold, ok := data["device_updown_threshold"].(float64); ok {
		threshold := int(deviceUpdownThreshold)
		ss.DeviceUpdownThreshold = &threshold
	}

	if dnsServers, ok := data["dns_servers"].([]interface{}); ok {
		var servers []string
		for _, server := range dnsServers {
			if serverStr, ok := server.(string); ok {
				servers = append(servers, serverStr)
			}
		}
		if len(servers) > 0 {
			ss.DNSServers = &servers
		}
	}

	if dnsSuffix, ok := data["dns_suffix"].([]interface{}); ok {
		var suffixes []string
		for _, suffix := range dnsSuffix {
			if suffixStr, ok := suffix.(string); ok {
				suffixes = append(suffixes, suffixStr)
			}
		}
		if len(suffixes) > 0 {
			ss.DNSSuffix = &suffixes
		}
	}

	if enableUnii4, ok := data["enable_unii_4"].(bool); ok {
		ss.EnableUnii4 = &enableUnii4
	}
	if gatewayUpdownThreshold, ok := data["gateway_updown_threshold"].(float64); ok {
		threshold := int(gatewayUpdownThreshold)
		ss.GatewayUpdownThreshold = &threshold
	}

	if ntpServers, ok := data["ntp_servers"].([]interface{}); ok {
		var servers []string
		for _, server := range ntpServers {
			if serverStr, ok := server.(string); ok {
				servers = append(servers, serverStr)
			}
		}
		if len(servers) > 0 {
			ss.NTPServers = &servers
		}
	}

	if persistConfigOnDevice, ok := data["persist_config_on_device"].(bool); ok {
		ss.PersistConfigOnDevice = &persistConfigOnDevice
	}
	if removeExistingConfigs, ok := data["remove_existing_configs"].(bool); ok {
		ss.RemoveExistingConfigs = &removeExistingConfigs
	}
	if reportGatt, ok := data["report_gatt"].(bool); ok {
		ss.ReportGatt = &reportGatt
	}

	if sshKeys, ok := data["ssh_keys"].([]interface{}); ok {
		var keys []string
		for _, key := range sshKeys {
			if keyStr, ok := key.(string); ok {
				keys = append(keys, keyStr)
			}
		}
		if len(keys) > 0 {
			ss.SSHKeys = &keys
		}
	}

	if switchUpdownThreshold, ok := data["switch_updown_threshold"].(float64); ok {
		threshold := int(switchUpdownThreshold)
		ss.SwitchUpdownThreshold = &threshold
	}
	if trackAnonymousDevices, ok := data["track_anonymous_devices"].(bool); ok {
		ss.TrackAnonymousDevices = &trackAnonymousDevices
	}
	if tuntermMonitoringDisabled, ok := data["tunterm_monitoring_disabled"].(bool); ok {
		ss.TuntermMonitoringDisabled = &tuntermMonitoringDisabled
	}
	if watchedStationURL, ok := data["watched_station_url"].(string); ok {
		ss.WatchedStationURL = &watchedStationURL
	}
	if whitelistURL, ok := data["whitelist_url"].(string); ok { // Mist API field name
		ss.AllowlistURL = &whitelistURL
	}

	// Handle array fields with complex objects
	if aclPolicies, ok := data["acl_policies"].([]interface{}); ok {
		var policies []map[string]interface{}
		for _, policy := range aclPolicies {
			if policyMap, ok := policy.(map[string]interface{}); ok {
				policies = append(policies, policyMap)
			}
		}
		if len(policies) > 0 {
			ss.ACLPolicies = policies
		}
	}

	if disabledPortUsages, ok := data["disabled_system_defined_port_usages"].([]interface{}); ok {
		var usages []string
		for _, usage := range disabledPortUsages {
			if usageStr, ok := usage.(string); ok {
				usages = append(usages, usageStr)
			}
		}
		if len(usages) > 0 {
			ss.DisabledSystemDefinedPortUsages = &usages
		}
	}

	if tuntermMonitoring, ok := data["tunterm_monitoring"].([]interface{}); ok {
		var monitoring []map[string]interface{}
		for _, monitor := range tuntermMonitoring {
			if monitorMap, ok := monitor.(map[string]interface{}); ok {
				monitoring = append(monitoring, monitorMap)
			}
		}
		if len(monitoring) > 0 {
			ss.TuntermMonitoring = monitoring
		}
	}

	// Handle complex nested configuration objects (preserve exact structure like APDevice)
	complexFields := map[string]*map[string]interface{}{
		"acl_tags": &ss.ACLTags, "analytic": &ss.Analytic, "ap_matching": &ss.APMatching,
		"ap_port_config": &ss.APPortConfig, "auto_placement": &ss.AutoPlacement, "auto_upgrade": &ss.AutoUpgrade,
		"ble_config": &ss.BLEConfig, "config_push_policy": &ss.ConfigPushPolicy, "critical_url_monitoring": &ss.CriticalURLMonitoring,
		"dhcp_snooping": &ss.DHCPSnooping, "engagement": &ss.Engagement, "evpn_options": &ss.EVPNOptions,
		"extra_routes": &ss.ExtraRoutes, "extra_routes6": &ss.ExtraRoutes6, "flags": &ss.Flags,
		"gateway": &ss.Gateway, "gateway_mgmt": &ss.GatewayMgmt, "juniper_srx": &ss.JuniperSRX,
		"led": &ss.LED, "marvis": &ss.Marvis, "mist_nac": &ss.MistNAC,
		"mxedge": &ss.MXEdge, "mxedge_mgmt": &ss.MXEdgeMgmt, "mxtunnels": &ss.MXTunnels,
		"networks": &ss.Networks, "occupancy": &ss.Occupancy, "ospf_areas": &ss.OSPFAreas,
		"paloalto_networks": &ss.PaloAltoNetworks, "port_mirroring": &ss.PortMirroring, "port_usages": &ss.PortUsages,
		"proxy": &ss.Proxy, "radio_config": &ss.RadioConfig, "radius_config": &ss.RadiusConfig,
		"remote_syslog": &ss.RemoteSyslog, "rogue": &ss.Rogue, "rtsa": &ss.RTSA,
		"simple_alert": &ss.SimpleAlert, "skyatp": &ss.SkyATP, "sle_thresholds": &ss.SLEThresholds,
		"snmp_config": &ss.SNMPConfig, "srx_app": &ss.SRXApp, "ssr": &ss.SSR,
		"status_portal": &ss.StatusPortal, "switch": &ss.Switch, "switch_matching": &ss.SwitchMatching,
		"switch_mgmt": &ss.SwitchMgmt, "synthetic_test": &ss.SyntheticTest, "tunterm_multicast_config": &ss.TuntermMulticastConfig,
		"uplink_port_config": &ss.UplinkPortConfig, "vars": &ss.Vars, "vna": &ss.VNA,
		"vrf_config": &ss.VRFConfig, "vrf_instances": &ss.VRFInstances, "vrrp_groups": &ss.VRRPGroups,
		"vs_instance": &ss.VSInstance, "wan_vna": &ss.WANVNA, "wids": &ss.WIDS,
		"wifi": &ss.WiFi, "wired_vna": &ss.WiredVNA, "zone_occupancy_alert": &ss.ZoneOccupancyAlert,
	}

	for fieldName, configPtr := range complexFields {
		if configData, ok := data[fieldName].(map[string]interface{}); ok {
			*configPtr = configData
		}
	}

	// Store any unknown fields in AdditionalConfig (following APDevice pattern)
	knownFields := map[string]bool{
		// Core identification fields
		"id": true, "site_id": true, "org_id": true, "created_time": true, "modified_time": true, "for_site": true,
		// Simple configuration fields
		"additional_config_cmds": true, "gateway_additional_config_cmds": true, "ap_updown_threshold": true,
		"auto_upgrade_linecard": true, "blacklist_url": true, "config_auto_revert": true, "default_port_usage": true,
		"device_updown_threshold": true, "dns_servers": true, "dns_suffix": true, "enable_unii_4": true,
		"gateway_updown_threshold": true, "ntp_servers": true, "persist_config_on_device": true, "remove_existing_configs": true,
		"report_gatt": true, "ssh_keys": true, "switch_updown_threshold": true, "track_anonymous_devices": true,
		"tunterm_monitoring_disabled": true, "watched_station_url": true, "whitelist_url": true,
		// Array fields
		"acl_policies": true, "disabled_system_defined_port_usages": true, "tunterm_monitoring": true,
		// Complex configuration objects
		"acl_tags": true, "analytic": true, "ap_matching": true, "ap_port_config": true, "auto_placement": true, "auto_upgrade": true,
		"ble_config": true, "config_push_policy": true, "critical_url_monitoring": true, "dhcp_snooping": true, "engagement": true, "evpn_options": true,
		"extra_routes": true, "extra_routes6": true, "flags": true, "gateway": true, "gateway_mgmt": true, "juniper_srx": true,
		"led": true, "marvis": true, "mist_nac": true, "mxedge": true, "mxedge_mgmt": true, "mxtunnels": true,
		"networks": true, "occupancy": true, "ospf_areas": true, "paloalto_networks": true, "port_mirroring": true, "port_usages": true,
		"proxy": true, "radio_config": true, "radius_config": true, "remote_syslog": true, "rogue": true, "rtsa": true,
		"simple_alert": true, "skyatp": true, "sle_thresholds": true, "snmp_config": true, "srx_app": true, "ssr": true,
		"status_portal": true, "switch": true, "switch_matching": true, "switch_mgmt": true, "synthetic_test": true, "tunterm_multicast_config": true,
		"uplink_port_config": true, "vars": true, "vna": true, "vrf_config": true, "vrf_instances": true, "vrrp_groups": true,
		"vs_instance": true, "wan_vna": true, "wids": true, "wifi": true, "wired_vna": true, "zone_occupancy_alert": true,
	}

	for k, v := range data {
		if !knownFields[k] {
			ss.AdditionalConfig[k] = v
		}
	}

	return nil
}

// NewSiteSettingFromMap creates a new site setting from a map representation
func NewSiteSettingFromMap(data map[string]interface{}) (*SiteSetting, error) {
	siteSetting := &SiteSetting{
		AdditionalConfig: make(map[string]interface{}),
	}

	if err := siteSetting.FromMap(data); err != nil {
		return nil, fmt.Errorf("failed to create site setting from map: %w", err)
	}

	return siteSetting, nil
}

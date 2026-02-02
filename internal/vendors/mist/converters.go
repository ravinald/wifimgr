package mist

import (
	"strings"

	"github.com/ravinald/wifimgr/api"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// convertSiteToSiteInfo converts a Mist MistSite to vendors.SiteInfo.
func convertSiteToSiteInfo(site *api.MistSite) *vendors.SiteInfo {
	if site == nil {
		return nil
	}

	info := &vendors.SiteInfo{
		SourceVendor: "mist",
	}

	if site.ID != nil {
		info.ID = *site.ID
	}
	if site.Name != nil {
		info.Name = *site.Name
	}
	if site.Timezone != nil {
		info.Timezone = *site.Timezone
	}
	if site.Address != nil {
		info.Address = *site.Address
	}
	if site.CountryCode != nil {
		info.CountryCode = *site.CountryCode
	}
	if site.Notes != nil {
		info.Notes = *site.Notes
	}
	if site.Latlng != nil {
		if site.Latlng.Lat != nil {
			info.Latitude = *site.Latlng.Lat
		}
		if site.Latlng.Lng != nil {
			info.Longitude = *site.Latlng.Lng
		}
	}

	return info
}

// convertSiteInfoToSite converts a vendors.SiteInfo to Mist MistSite for create/update operations.
func convertSiteInfoToSite(info *vendors.SiteInfo) *api.MistSite {
	if info == nil {
		return nil
	}

	site := &api.MistSite{}

	if info.ID != "" {
		site.ID = &info.ID
	}
	if info.Name != "" {
		site.Name = &info.Name
	}
	if info.Timezone != "" {
		site.Timezone = &info.Timezone
	}
	if info.Address != "" {
		site.Address = &info.Address
	}
	if info.CountryCode != "" {
		site.CountryCode = &info.CountryCode
	}
	if info.Notes != "" {
		site.Notes = &info.Notes
	}
	if info.Latitude != 0 || info.Longitude != 0 {
		site.Latlng = &api.MistLatLng{
			Lat: &info.Latitude,
			Lng: &info.Longitude,
		}
	}

	return site
}

// convertInventoryItemToVendor converts a Mist MistInventoryItem to vendors.InventoryItem.
func convertInventoryItemToVendor(item *api.MistInventoryItem) *vendors.InventoryItem {
	if item == nil {
		return nil
	}

	inv := &vendors.InventoryItem{
		Claimed:      true, // If in inventory, it's claimed
		SourceVendor: "mist",
	}

	// ID is the Mist device UUID - needed for config API calls
	if item.ID != nil {
		inv.ID = *item.ID
	}
	if item.MAC != nil {
		inv.MAC = normalizeMAC(*item.MAC)
	}
	if item.Serial != nil {
		inv.Serial = *item.Serial
	}
	if item.Model != nil {
		inv.Model = *item.Model
	}
	if item.Name != nil {
		inv.Name = *item.Name
	}
	if item.Type != nil {
		inv.Type = *item.Type
	}
	if item.SiteID != nil {
		inv.SiteID = *item.SiteID
	}

	return inv
}

// convertUnifiedDeviceToDeviceInfo converts a Mist UnifiedDevice to vendors.DeviceInfo.
func convertUnifiedDeviceToDeviceInfo(device *api.UnifiedDevice) *vendors.DeviceInfo {
	if device == nil {
		return nil
	}

	info := &vendors.DeviceInfo{
		SourceVendor: "mist",
	}

	if device.ID != nil {
		info.ID = *device.ID
	}
	if device.MAC != nil {
		info.MAC = normalizeMAC(*device.MAC)
	}
	if device.Serial != nil {
		info.Serial = *device.Serial
	}
	if device.Name != nil {
		info.Name = *device.Name
	}
	if device.Model != nil {
		info.Model = *device.Model
	}
	if device.Type != nil {
		info.Type = *device.Type
	}
	if device.SiteID != nil {
		info.SiteID = *device.SiteID
	}
	if device.Notes != nil {
		info.Notes = *device.Notes
	}
	if device.DeviceProfileID != nil {
		info.DeviceProfileID = *device.DeviceProfileID
	}

	// Map connected status
	if device.Connected != nil {
		if *device.Connected {
			info.Status = "connected"
		} else {
			info.Status = "disconnected"
		}
	}

	// Extract IP and version from device config if available
	if ip, ok := device.DeviceConfig["ip"].(string); ok {
		info.IP = ip
	}
	if version, ok := device.DeviceConfig["version"].(string); ok {
		info.Version = version
	}

	return info
}

// convertDeviceInfoToUnified converts a vendors.DeviceInfo to Mist UnifiedDevice for update operations.
func convertDeviceInfoToUnified(info *vendors.DeviceInfo) *api.UnifiedDevice {
	if info == nil {
		return nil
	}

	device := &api.UnifiedDevice{
		BaseDevice: api.BaseDevice{
			AdditionalConfig: make(map[string]interface{}),
		},
		DeviceConfig: make(map[string]interface{}),
	}

	if info.ID != "" {
		device.ID = &info.ID
	}
	if info.MAC != "" {
		device.MAC = &info.MAC
	}
	if info.Name != "" {
		device.Name = &info.Name
	}
	if info.Type != "" {
		device.Type = &info.Type
		device.DeviceType = info.Type
	}
	if info.SiteID != "" {
		device.SiteID = &info.SiteID
	}
	if info.Notes != "" {
		device.Notes = &info.Notes
	}
	if info.DeviceProfileID != "" {
		device.DeviceProfileID = &info.DeviceProfileID
	}

	return device
}

// convertDeviceProfileToVendor converts a Mist DeviceProfile to vendors.DeviceProfile.
func convertDeviceProfileToVendor(profile *api.DeviceProfile) *vendors.DeviceProfile {
	if profile == nil {
		return nil
	}

	vp := &vendors.DeviceProfile{
		SourceVendor: "mist",
	}

	if profile.ID != nil {
		vp.ID = *profile.ID
	}
	if profile.Name != nil {
		vp.Name = *profile.Name
	}
	if profile.Type != nil {
		vp.Type = *profile.Type
	}
	if profile.OrgID != nil {
		vp.OrgID = *profile.OrgID
	}
	if profile.ForSite != nil {
		vp.ForSite = *profile.ForSite
	}
	if profile.SiteID != nil {
		vp.SiteID = *profile.SiteID
	}

	return vp
}

// convertWiredClientToVendor converts a Mist MistWiredClient to vendors.WiredClient.
func convertWiredClientToVendor(client *api.MistWiredClient) *vendors.WiredClient {
	if client == nil {
		return nil
	}

	wc := &vendors.WiredClient{
		SourceVendor: "mist",
	}

	// Get MAC from either field
	if client.ClientMAC != nil {
		wc.MAC = normalizeMAC(*client.ClientMAC)
	} else if client.MAC != nil {
		wc.MAC = normalizeMAC(*client.MAC)
	}

	if client.SiteID != nil {
		wc.SiteID = *client.SiteID
	}

	// Get last known values
	if client.LastIP != nil {
		wc.IP = *client.LastIP
	} else if len(client.IP) > 0 {
		wc.IP = client.IP[0]
	}

	if client.LastHostname != nil {
		wc.Hostname = *client.LastHostname
	} else if len(client.Hostname) > 0 {
		wc.Hostname = client.Hostname[0]
	}

	if client.LastDeviceMAC != nil {
		wc.SwitchMAC = normalizeMAC(*client.LastDeviceMAC)
	} else if len(client.DeviceMAC) > 0 {
		wc.SwitchMAC = normalizeMAC(client.DeviceMAC[0])
	}

	if client.LastPortID != nil {
		wc.PortID = *client.LastPortID
	} else if len(client.PortID) > 0 {
		wc.PortID = client.PortID[0]
	}

	if client.LastVLAN != nil {
		wc.VLAN = *client.LastVLAN
	} else if len(client.VLAN) > 0 {
		wc.VLAN = client.VLAN[0]
	}

	if client.Manufacture != nil {
		wc.Manufacturer = *client.Manufacture
	}

	return wc
}

// convertWirelessClientToVendor converts a Mist MistWirelessClient to vendors.WirelessClient.
func convertWirelessClientToVendor(client *api.MistWirelessClient) *vendors.WirelessClient {
	if client == nil {
		return nil
	}

	wc := &vendors.WirelessClient{
		SourceVendor: "mist",
	}

	if client.MAC != nil {
		wc.MAC = normalizeMAC(*client.MAC)
	}

	if client.SiteID != nil {
		wc.SiteID = *client.SiteID
	}

	// Get last known values
	if client.LastIP != nil {
		wc.IP = *client.LastIP
	} else if len(client.IP) > 0 {
		wc.IP = client.IP[0]
	}

	if client.LastHostname != nil {
		wc.Hostname = *client.LastHostname
	} else if len(client.Hostname) > 0 {
		wc.Hostname = client.Hostname[0]
	}

	if client.LastAP != nil {
		wc.APMAC = normalizeMAC(*client.LastAP)
	} else if len(client.AP) > 0 {
		wc.APMAC = normalizeMAC(client.AP[0])
	}

	if client.LastSSID != nil {
		wc.SSID = *client.LastSSID
	} else if len(client.SSID) > 0 {
		wc.SSID = client.SSID[0]
	}

	if client.LastVLAN != nil {
		wc.VLAN = *client.LastVLAN
	} else if len(client.VLAN) > 0 {
		wc.VLAN = client.VLAN[0]
	}

	if client.Band != nil {
		wc.Band = *client.Band
	}

	if client.Manufacture != nil {
		wc.Manufacturer = *client.Manufacture
	}

	if client.LastOS != nil {
		wc.OS = *client.LastOS
	} else if len(client.OS) > 0 {
		wc.OS = client.OS[0]
	}

	return wc
}

// convertRFTemplateToVendor converts a Mist MistRFTemplate to vendors.RFTemplate.
func convertRFTemplateToVendor(template *api.MistRFTemplate) *vendors.RFTemplate {
	if template == nil {
		return nil
	}

	vt := &vendors.RFTemplate{
		SourceVendor: "mist",
	}

	if template.ID != nil {
		vt.ID = *template.ID
	}
	if template.Name != nil {
		vt.Name = *template.Name
	}
	if template.OrgID != nil {
		vt.OrgID = *template.OrgID
	}

	return vt
}

// convertGatewayTemplateToVendor converts a Mist MistGatewayTemplate to vendors.GatewayTemplate.
func convertGatewayTemplateToVendor(template *api.MistGatewayTemplate) *vendors.GatewayTemplate {
	if template == nil {
		return nil
	}

	vt := &vendors.GatewayTemplate{
		SourceVendor: "mist",
	}

	if template.ID != nil {
		vt.ID = *template.ID
	}
	if template.Name != nil {
		vt.Name = *template.Name
	}
	if template.OrgID != nil {
		vt.OrgID = *template.OrgID
	}

	return vt
}

// convertWLANTemplateToVendor converts a Mist MistWLANTemplate to vendors.WLANTemplate.
func convertWLANTemplateToVendor(template *api.MistWLANTemplate) *vendors.WLANTemplate {
	if template == nil {
		return nil
	}

	vt := &vendors.WLANTemplate{
		SourceVendor: "mist",
	}

	if template.ID != nil {
		vt.ID = *template.ID
	}
	if template.Name != nil {
		vt.Name = *template.Name
	}
	if template.OrgID != nil {
		vt.OrgID = *template.OrgID
	}

	return vt
}

// convertAPConfigToVendor converts a Mist APConfig to vendors.APConfig.
func convertAPConfigToVendor(config *api.APConfig) *vendors.APConfig {
	if config == nil {
		return nil
	}

	vc := &vendors.APConfig{
		Config:       make(map[string]interface{}),
		SourceVendor: "mist",
	}

	if config.ID != nil {
		vc.ID = *config.ID
	}
	if config.Name != nil {
		vc.Name = *config.Name
	}
	if config.MAC != nil {
		vc.MAC = normalizeMAC(*config.MAC)
	}
	if config.SiteID != nil {
		vc.SiteID = *config.SiteID
	}

	// Copy all config data
	vc.Config = config.ToMap()

	return vc
}

// convertSwitchConfigToVendor converts a Mist SwitchConfig to vendors.SwitchConfig.
func convertSwitchConfigToVendor(config *api.SwitchConfig) *vendors.SwitchConfig {
	if config == nil {
		return nil
	}

	vc := &vendors.SwitchConfig{
		Config:       make(map[string]interface{}),
		SourceVendor: "mist",
	}

	if config.ID != nil {
		vc.ID = *config.ID
	}
	if config.Name != nil {
		vc.Name = *config.Name
	}
	if config.MAC != nil {
		vc.MAC = normalizeMAC(*config.MAC)
	}
	if config.SiteID != nil {
		vc.SiteID = *config.SiteID
	}

	// Copy all config data
	vc.Config = config.ToMap()

	return vc
}

// convertGatewayConfigToVendor converts a Mist GatewayConfig to vendors.GatewayConfig.
func convertGatewayConfigToVendor(config *api.GatewayConfig) *vendors.GatewayConfig {
	if config == nil {
		return nil
	}

	vc := &vendors.GatewayConfig{
		Config:       make(map[string]interface{}),
		SourceVendor: "mist",
	}

	if config.ID != nil {
		vc.ID = *config.ID
	}
	if config.Name != nil {
		vc.Name = *config.Name
	}
	if config.MAC != nil {
		vc.MAC = normalizeMAC(*config.MAC)
	}
	if config.SiteID != nil {
		vc.SiteID = *config.SiteID
	}

	// Copy all config data
	vc.Config = config.ToMap()

	return vc
}

// normalizeMAC normalizes a MAC address to lowercase without separators.
func normalizeMAC(mac string) string {
	// Remove colons and dashes, convert to lowercase
	mac = strings.ReplaceAll(mac, ":", "")
	mac = strings.ReplaceAll(mac, "-", "")
	return strings.ToLower(mac)
}

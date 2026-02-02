package meraki

import (
	"strings"

	meraki "github.com/meraki/dashboard-api-go/v5/sdk"

	"github.com/ravinald/wifimgr/internal/vendors"
)

// convertNetworkToSiteInfo converts a Meraki network to vendors.SiteInfo.
func convertNetworkToSiteInfo(network *meraki.ResponseItemOrganizationsGetOrganizationNetworks) *vendors.SiteInfo {
	if network == nil {
		return nil
	}

	return &vendors.SiteInfo{
		ID:           network.ID,
		Name:         network.Name,
		Timezone:     network.TimeZone,
		Notes:        network.Notes,
		SourceVendor: "meraki",
	}
}

// convertDeviceToInventoryItem converts a Meraki device to vendors.InventoryItem.
func convertDeviceToInventoryItem(device *meraki.ResponseItemOrganizationsGetOrganizationDevices) *vendors.InventoryItem {
	if device == nil {
		return nil
	}

	return &vendors.InventoryItem{
		ID:           device.Serial, // Meraki uses serial as device identifier
		MAC:          normalizeMAC(device.Mac),
		Serial:       device.Serial,
		Model:        device.Model,
		Name:         device.Name,
		Type:         mapProductTypeToDeviceType(device.ProductType),
		SiteID:       device.NetworkID,
		Claimed:      true, // If in org, it's claimed
		SourceVendor: "meraki",
	}
}

// convertDeviceToDeviceInfo converts a Meraki device to vendors.DeviceInfo.
func convertDeviceToDeviceInfo(device *meraki.ResponseItemOrganizationsGetOrganizationDevices) *vendors.DeviceInfo {
	if device == nil {
		return nil
	}

	info := &vendors.DeviceInfo{
		MAC:          normalizeMAC(device.Mac),
		Serial:       device.Serial,
		Model:        device.Model,
		Name:         device.Name,
		Type:         mapProductTypeToDeviceType(device.ProductType),
		SiteID:       device.NetworkID,
		Notes:        device.Notes,
		IP:           device.LanIP,
		Version:      device.Firmware,
		SourceVendor: "meraki",
	}

	if device.Lat != nil {
		info.Latitude = *device.Lat
	}
	if device.Lng != nil {
		info.Longitude = *device.Lng
	}

	return info
}

// mapProductTypeToDeviceType converts Meraki product types to wifimgr types.
func mapProductTypeToDeviceType(productType string) string {
	switch productType {
	case "wireless":
		return "ap"
	case "switch":
		return "switch"
	case "appliance":
		return "gateway"
	case "camera":
		return "camera"
	case "cellularGateway":
		return "cellular"
	case "sensor":
		return "sensor"
	default:
		return productType
	}
}

// mapDeviceTypeToProductType converts wifimgr types to Meraki product types.
func mapDeviceTypeToProductType(deviceType string) string {
	switch deviceType {
	case "ap":
		return "wireless"
	case "switch":
		return "switch"
	case "gateway":
		return "appliance"
	case "camera":
		return "camera"
	case "cellular":
		return "cellularGateway"
	case "sensor":
		return "sensor"
	default:
		return deviceType
	}
}

// normalizeMAC normalizes a MAC address to lowercase without separators.
func normalizeMAC(mac string) string {
	mac = strings.ReplaceAll(mac, ":", "")
	mac = strings.ReplaceAll(mac, "-", "")
	mac = strings.ReplaceAll(mac, ".", "")
	return strings.ToLower(mac)
}

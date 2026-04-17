package ubiquiti

import (
	"strings"

	"github.com/ravinald/wifimgr/internal/vendors"
)

// FlatDevice is a device with host context flattened into it.
type FlatDevice struct {
	Device
	HostID   string
	HostName string
	SiteID   string
}

// flattenDevices flattens host-grouped device responses into a flat list.
// Filters to network devices only and resolves site IDs via hostSiteMap.
func flattenDevices(groups []HostDeviceGroup, hostSiteMap map[string]string) []FlatDevice {
	var result []FlatDevice
	for _, group := range groups {
		siteID := hostSiteMap[group.HostID]
		for _, device := range group.Devices {
			if !IsNetworkDevice(device) {
				continue
			}
			result = append(result, FlatDevice{
				Device:   device,
				HostID:   group.HostID,
				HostName: group.HostName,
				SiteID:   siteID,
			})
		}
	}
	return result
}

// convertSiteToSiteInfo converts a Ubiquiti Site to a vendor-agnostic SiteInfo.
// hostNameMap maps host IDs to host names for enriching "default" site names.
func convertSiteToSiteInfo(site Site, hostNameMap map[string]string) *vendors.SiteInfo {
	return &vendors.SiteInfo{
		ID:           site.GetID(),
		Name:         buildSiteName(site, hostNameMap),
		Timezone:     site.Meta.Timezone,
		Notes:        site.Meta.Desc,
		SourceVendor: "ubiquiti",
	}
}

// buildSiteName creates a friendly site name.
// UniFi controllers often have a single "default" site, so the host name
// (the console name set by the admin) is used as the meaningful identifier.
func buildSiteName(site Site, hostNameMap map[string]string) string {
	hostName := hostNameMap[site.HostID]
	siteName := site.GetName()

	if hostName == "" {
		return siteName
	}

	// If site name is "default" or empty, use just the host name
	if strings.EqualFold(siteName, "default") || siteName == "" {
		return hostName
	}

	// Custom site name: combine "HostName - SiteName"
	return hostName + " - " + siteName
}

// buildHostNameMap builds a map from host ID to host reported name.
func buildHostNameMap(hosts []Host) map[string]string {
	m := make(map[string]string, len(hosts))
	for _, host := range hosts {
		name := host.ReportedState.Name
		if name == "" {
			name = host.ReportedState.Hostname
		}
		if name != "" {
			m[host.ID] = name
		}
	}
	return m
}

// buildHostNameMapFromDevices builds a host name map from device groups.
// The device groups contain the same compound hostId as sites, making this
// a reliable way to resolve host names when the hosts API uses different IDs.
func buildHostNameMapFromDevices(groups []HostDeviceGroup) map[string]string {
	m := make(map[string]string, len(groups))
	for _, g := range groups {
		if g.HostID != "" && g.HostName != "" {
			m[g.HostID] = g.HostName
		}
	}
	return m
}

// convertFlatDeviceToInventoryItem converts a FlatDevice to a vendor-agnostic InventoryItem.
func convertFlatDeviceToInventoryItem(d FlatDevice) *vendors.InventoryItem {
	return &vendors.InventoryItem{
		ID:           d.ID,
		MAC:          normalizeMAC(d.MAC),
		Model:        d.Model,
		Name:         d.Name,
		Type:         classifyDevice(d.Device),
		SiteID:       d.SiteID,
		Claimed:      d.IsManaged,
		SourceVendor: "ubiquiti",
	}
}

// convertFlatDeviceToDeviceInfo converts a FlatDevice to a vendor-agnostic DeviceInfo.
func convertFlatDeviceToDeviceInfo(d FlatDevice) *vendors.DeviceInfo {
	return &vendors.DeviceInfo{
		ID:           d.ID,
		MAC:          normalizeMAC(d.MAC),
		Name:         d.Name,
		Model:        d.Model,
		Type:         classifyDevice(d.Device),
		SiteID:       d.SiteID,
		Status:       normalizeStatus(d.Status),
		IP:           d.IP,
		Version:      d.Version,
		Notes:        d.Note,
		SourceVendor: "ubiquiti",
	}
}

// normalizeStatus maps Ubiquiti status values to vendor-standard values.
func normalizeStatus(status string) string {
	switch strings.ToLower(status) {
	case "online":
		return "connected"
	case "offline":
		return "disconnected"
	default:
		return strings.ToLower(status)
	}
}

// normalizeMAC normalizes a MAC address to lowercase without separators.
func normalizeMAC(mac string) string {
	mac = strings.ReplaceAll(mac, ":", "")
	mac = strings.ReplaceAll(mac, "-", "")
	mac = strings.ReplaceAll(mac, ".", "")
	return strings.ToLower(mac)
}

// buildHostSiteMap builds a map from host ID to site ID.
func buildHostSiteMap(sites []Site) map[string]string {
	m := make(map[string]string, len(sites))
	for _, site := range sites {
		m[site.HostID] = site.GetID()
	}
	return m
}

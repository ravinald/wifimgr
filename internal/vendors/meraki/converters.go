package meraki

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	meraki "github.com/meraki/dashboard-api-go/v5/sdk"

	"github.com/ravinald/wifimgr/internal/vendors"
)

// httpRespBody returns the response body bytes, or nil if the resty response
// is missing. Centralized so callers don't repeat the nil dance.
func httpRespBody(r *resty.Response) []byte {
	if r == nil {
		return nil
	}
	return r.Body()
}

// applyClientTimes copies non-zero timestamps from src into the dst fields,
// preserving any value the SDK already parsed. Used to merge wire-parsed
// timestamps into the SDK-typed client struct.
func applyClientTimes(dstFirst, dstLast *time.Time, src clientTimestamps) {
	if dstFirst != nil && dstFirst.IsZero() && !src.FirstSeen.IsZero() {
		*dstFirst = src.FirstSeen
	}
	if dstLast != nil && dstLast.IsZero() && !src.LastSeen.IsZero() {
		*dstLast = src.LastSeen
	}
}

// epochSecondsToTime converts a Meraki *int Unix-epoch (seconds) to UTC time.
// Used by SDK-typed paths; the wire-level parser below is what salvages
// timestamps when Meraki returns ISO 8601 strings instead.
func epochSecondsToTime(sec *int) time.Time {
	if sec == nil || *sec == 0 {
		return time.Time{}
	}
	return time.Unix(int64(*sec), 0).UTC()
}

// clientTimestamps holds first/last-seen times parsed from a Meraki HTTP
// response body, independent of what the SDK managed to unmarshal.
type clientTimestamps struct {
	FirstSeen time.Time
	LastSeen  time.Time
}

// decodeFlexTime accepts a Meraki firstSeen / lastSeen value in any of the
// three forms the dashboard API has been observed to emit:
//   - integer Unix epoch seconds
//   - RFC 3339 / ISO 8601 string (with or without sub-second precision)
//   - JSON null or absent
//
// The v5 SDK declares both fields as *int, so a string-shaped response
// silently leaves the pointer nil. Calling this on the raw bytes recovers
// the value either way.
func decodeFlexTime(raw json.RawMessage) time.Time {
	if len(raw) == 0 || string(raw) == "null" {
		return time.Time{}
	}
	var n int64
	if err := json.Unmarshal(raw, &n); err == nil {
		if n <= 0 {
			return time.Time{}
		}
		return time.Unix(n, 0).UTC()
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil && s != "" {
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
			if t, err := time.Parse(layout, s); err == nil {
				return t.UTC()
			}
		}
	}
	return time.Time{}
}

// parseNetworkClientTimes pulls firstSeen / lastSeen out of a
// GET /networks/{id}/clients response body, keyed by normalized MAC. Returns
// nil on unmarshal failure so callers can fall back to whatever the SDK
// supplied without a hard error.
func parseNetworkClientTimes(body []byte) map[string]clientTimestamps {
	if len(body) == 0 {
		return nil
	}
	var raws []struct {
		Mac       string          `json:"mac"`
		FirstSeen json.RawMessage `json:"firstSeen"`
		LastSeen  json.RawMessage `json:"lastSeen"`
	}
	if err := json.Unmarshal(body, &raws); err != nil {
		return nil
	}
	out := make(map[string]clientTimestamps, len(raws))
	for _, r := range raws {
		if r.Mac == "" {
			continue
		}
		ts := clientTimestamps{
			FirstSeen: decodeFlexTime(r.FirstSeen),
			LastSeen:  decodeFlexTime(r.LastSeen),
		}
		if ts.FirstSeen.IsZero() && ts.LastSeen.IsZero() {
			continue
		}
		out[vendors.NormalizeMAC(r.Mac)] = ts
	}
	return out
}

// parseOrgClientSearchTimes pulls firstSeen / lastSeen out of a
// GET /organizations/{id}/clients/search response, keyed by the network ID
// of each sighting. The org-search response is a single object with a
// records array; each record carries its own timestamps and network ID, so
// a per-MAC key would collide across networks.
func parseOrgClientSearchTimes(body []byte) map[string]clientTimestamps {
	if len(body) == 0 {
		return nil
	}
	var resp struct {
		Records []struct {
			FirstSeen json.RawMessage `json:"firstSeen"`
			LastSeen  json.RawMessage `json:"lastSeen"`
			Network   struct {
				ID string `json:"id"`
			} `json:"network"`
		} `json:"records"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil
	}
	out := make(map[string]clientTimestamps, len(resp.Records))
	for _, rec := range resp.Records {
		if rec.Network.ID == "" {
			continue
		}
		ts := clientTimestamps{
			FirstSeen: decodeFlexTime(rec.FirstSeen),
			LastSeen:  decodeFlexTime(rec.LastSeen),
		}
		if ts.FirstSeen.IsZero() && ts.LastSeen.IsZero() {
			continue
		}
		out[rec.Network.ID] = ts
	}
	return out
}

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

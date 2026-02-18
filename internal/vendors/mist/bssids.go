package mist

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/ravinald/wifimgr/api"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// bssidsService implements vendors.BSSIDsService for Mist.
// BSSIDs are derived from radio base MACs and cross-referenced with site WLANs.
type bssidsService struct {
	client api.Client
	orgID  string
}

// List retrieves all BSSIDs across all sites by deriving them from AP radio stats and WLANs.
func (s *bssidsService) List(ctx context.Context) ([]*vendors.BSSIDEntry, error) {
	logging.Debugf("[mist] Fetching BSSIDs for org %s", s.orgID)

	// Get all sites
	sites, err := s.client.GetSites(ctx, s.orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sites: %w", err)
	}

	// Get org-level WLANs (these apply to all sites)
	orgWLANs, err := s.client.GetWLANs(ctx, s.orgID)
	if err != nil {
		logging.Warnf("[mist] Failed to get org WLANs: %v", err)
		orgWLANs = nil
	}

	var allEntries []*vendors.BSSIDEntry

	for _, site := range sites {
		siteID := derefStr(site.ID)
		siteName := derefStr(site.Name)
		if siteID == "" {
			continue
		}

		// Get AP stats for this site
		apStats, err := s.client.GetAPStats(ctx, siteID)
		if err != nil {
			logging.Debugf("[mist] Failed to get AP stats for site %s: %v", siteName, err)
			continue
		}

		if len(apStats) == 0 {
			continue
		}

		// Get site-level WLANs
		siteWLANs, err := s.client.GetSiteWLANs(ctx, siteID)
		if err != nil {
			logging.Debugf("[mist] Failed to get site WLANs for %s: %v", siteName, err)
			continue
		}

		// Merge org-level and site-level WLANs, sorted by ID for deterministic ordering
		effectiveWLANs := mergeWLANs(orgWLANs, siteWLANs)

		for _, apStat := range apStats {
			apName, _ := apStat["name"].(string)
			apMAC, _ := apStat["mac"].(string)
			apSerial, _ := apStat["serial"].(string)

			radioStat, ok := apStat["radio_stat"].(map[string]interface{})
			if !ok {
				continue
			}

			// Process each radio band
			for _, bandKey := range []string{"band_24", "band_5", "band_6"} {
				bandData, ok := radioStat[bandKey].(map[string]interface{})
				if !ok {
					continue
				}

				baseMAC, _ := bandData["mac"].(string)
				if baseMAC == "" {
					continue
				}

				channel := intFromMap(bandData, "channel")
				bandwidth := intFromMap(bandData, "bandwidth")
				power := intFromMap(bandData, "power")

				bandLabel := bandKeyToLabel(bandKey)

				// Derive BSSIDs: one per WLAN at incrementing MAC offsets
				for idx, wlan := range effectiveWLANs {
					bssid := incrementMAC(baseMAC, idx)
					if bssid == "" {
						continue
					}

					entry := &vendors.BSSIDEntry{
						BSSID:          vendors.NormalizeMAC(bssid),
						APName:         apName,
						APSerial:       apSerial,
						APMAC:          vendors.NormalizeMAC(apMAC),
						SiteID:         siteID,
						SiteName:       siteName,
						SSIDName:       wlan.ssid,
						SSIDNumber:     idx,
						Band:           bandLabel,
						Channel:        channel,
						ChannelWidth:   bandwidth,
						Power:          power,
						IsBroadcasting: wlan.enabled,
					}
					allEntries = append(allEntries, entry)
				}
			}
		}
	}

	logging.Debugf("[mist] Derived %d BSSIDs", len(allEntries))
	return allEntries, nil
}

// wlanInfo holds the minimal WLAN info needed for BSSID derivation.
type wlanInfo struct {
	id      string
	ssid    string
	enabled bool
}

// mergeWLANs combines org-level and site-level WLANs, sorted by ID for deterministic ordering.
// Site-level WLANs with the same SSID override org-level ones.
func mergeWLANs(orgWLANs []api.MistWLAN, siteWLANs []api.MistWLAN) []wlanInfo {
	seen := make(map[string]wlanInfo)

	for _, w := range orgWLANs {
		ssid := derefStr(w.SSID)
		if ssid == "" {
			continue
		}
		seen[ssid] = wlanInfo{id: derefStr(w.ID), ssid: ssid, enabled: derefBool(w.Enabled)}
	}
	for _, w := range siteWLANs {
		ssid := derefStr(w.SSID)
		if ssid == "" {
			continue
		}
		seen[ssid] = wlanInfo{id: derefStr(w.ID), ssid: ssid, enabled: derefBool(w.Enabled)}
	}

	result := make([]wlanInfo, 0, len(seen))
	for _, w := range seen {
		result = append(result, w)
	}

	// Sort by ID for deterministic BSSID assignment
	sort.Slice(result, func(i, j int) bool {
		return result[i].id < result[j].id
	})

	return result
}

// incrementMAC adds offset to the last byte of a MAC address.
func incrementMAC(baseMAC string, offset int) string {
	normalized := vendors.NormalizeMAC(baseMAC)
	if len(normalized) != 12 {
		return ""
	}

	lastByte := hexToByte(normalized[10:12])
	newLastByte := (int(lastByte) + offset) & 0xFF

	return normalized[:10] + fmt.Sprintf("%02x", newLastByte)
}

// hexToByte converts a 2-char hex string to a byte.
func hexToByte(s string) byte {
	if len(s) != 2 {
		return 0
	}
	var b byte
	for _, c := range s {
		b <<= 4
		switch {
		case c >= '0' && c <= '9':
			b |= byte(c - '0')
		case c >= 'a' && c <= 'f':
			b |= byte(c - 'a' + 10)
		case c >= 'A' && c <= 'F':
			b |= byte(c - 'A' + 10)
		}
	}
	return b
}

// intFromMap extracts an int from a map[string]interface{}, handling float64 JSON conversion.
func intFromMap(m map[string]interface{}, key string) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return 0
}

// bandKeyToLabel converts Mist radio stat band keys to display labels.
func bandKeyToLabel(key string) string {
	switch strings.TrimPrefix(key, "band_") {
	case "24":
		return "2.4"
	case "5":
		return "5"
	case "6":
		return "6"
	default:
		return key
	}
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func derefBool(p *bool) bool {
	if p == nil {
		return false
	}
	return *p
}

var _ vendors.BSSIDsService = (*bssidsService)(nil)

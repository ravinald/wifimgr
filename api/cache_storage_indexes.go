package api

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/spf13/viper"

	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/macaddr"
)

// buildIndexes constructs all lookup indexes from the cache data
func (cm *CacheManager) buildIndexes() error {
	cm.indexes = NewCacheIndexes()

	// Loop through all organizations
	for orgID, orgData := range cm.cache.Orgs {
		// Index organization stats
		if orgData.OrgStats != nil {
			cm.indexes.OrgsByID[orgID] = orgData.OrgStats
			if orgData.OrgStats.Name != nil {
				cm.indexes.OrgsByName[*orgData.OrgStats.Name] = orgData.OrgStats
			}
		}

		// Index sites info for this org
		for i := range orgData.Sites.Info {
			site := &orgData.Sites.Info[i]
			if site.Name != nil {
				cm.indexes.SitesByName[*site.Name] = site
			}
			if site.ID != nil {
				cm.indexes.SitesByID[*site.ID] = site
			}
		}

		// Index site settings for this org
		for i := range orgData.Sites.Settings {
			setting := &orgData.Sites.Settings[i]
			if setting.SiteID != nil {
				cm.indexes.SiteSettingsBySiteID[*setting.SiteID] = setting
			}
			if setting.ID != nil {
				cm.indexes.SiteSettingsByID[*setting.ID] = setting
			}
		}

		// Index RF templates for this org
		for i := range orgData.Templates.RF {
			template := &orgData.Templates.RF[i]
			if template.Name != nil {
				cm.indexes.RFTemplatesByName[*template.Name] = template
			}
			if template.ID != nil {
				cm.indexes.RFTemplatesByID[*template.ID] = template
			}
		}

		// Index Gateway templates for this org
		for i := range orgData.Templates.Gateway {
			template := &orgData.Templates.Gateway[i]
			if template.Name != nil {
				cm.indexes.GWTemplatesByName[*template.Name] = template
			}
			if template.ID != nil {
				cm.indexes.GWTemplatesByID[*template.ID] = template
			}
		}

		// Index WLAN templates for this org
		for i := range orgData.Templates.WLAN {
			template := &orgData.Templates.WLAN[i]
			if template.Name != nil {
				cm.indexes.WLANTemplatesByName[*template.Name] = template
			}
			if template.ID != nil {
				cm.indexes.WLANTemplatesByID[*template.ID] = template
			}
		}

		// Index networks for this org
		for i := range orgData.Networks {
			network := &orgData.Networks[i]
			if network.Name != nil {
				cm.indexes.NetworksByName[*network.Name] = network
			}
			if network.ID != nil {
				cm.indexes.NetworksByID[*network.ID] = network
			}
		}

		// Index org WLANs for this org
		for i := range orgData.WLANs.Org {
			wlan := &orgData.WLANs.Org[i]
			if wlan.SSID != nil {
				cm.indexes.OrgWLANsByName[*wlan.SSID] = wlan
			}
			if wlan.ID != nil {
				cm.indexes.OrgWLANsByID[*wlan.ID] = wlan
			}
		}

		// Index site WLANs for this org
		for siteID, wlans := range orgData.WLANs.Sites {
			cm.indexes.SiteWLANsByName[siteID] = make(map[string]*MistWLAN)
			cm.indexes.SiteWLANsByID[siteID] = make(map[string]*MistWLAN)

			for i := range wlans {
				wlan := &wlans[i]
				if wlan.SSID != nil {
					cm.indexes.SiteWLANsByName[siteID][*wlan.SSID] = wlan
				}
				if wlan.ID != nil {
					cm.indexes.SiteWLANsByID[siteID][*wlan.ID] = wlan
				}
			}
		}

		// Index APs for this org (now a map)
		for mac, ap := range orgData.Inventory.AP {
			apCopy := ap
			apPtr := &apCopy
			if apPtr.Name != nil {
				cm.indexes.APsByName[*apPtr.Name] = apPtr
			}
			// MAC is already the key, just normalize it for the index
			normalizedMAC := macaddr.NormalizeFast(mac)
			cm.indexes.APsByMAC[normalizedMAC] = apPtr
			if apPtr.SiteID != nil {
				cm.indexes.APsBySite[*apPtr.SiteID] = append(cm.indexes.APsBySite[*apPtr.SiteID], apPtr)
			}
		}

		// Index Switches for this org (now a map)
		for mac, sw := range orgData.Inventory.Switch {
			swCopy := sw
			swPtr := &swCopy
			if swPtr.Name != nil {
				cm.indexes.SwitchesByName[*swPtr.Name] = swPtr
			}
			// MAC is already the key, just normalize it for the index
			normalizedMAC := macaddr.NormalizeFast(mac)
			cm.indexes.SwitchesByMAC[normalizedMAC] = swPtr
			if swPtr.SiteID != nil {
				cm.indexes.SwitchesBySite[*swPtr.SiteID] = append(cm.indexes.SwitchesBySite[*swPtr.SiteID], swPtr)
			}
		}

		// Index Gateways for this org (now a map)
		for mac, gw := range orgData.Inventory.Gateway {
			gwCopy := gw
			gwPtr := &gwCopy
			if gwPtr.Name != nil {
				cm.indexes.GatewaysByName[*gwPtr.Name] = gwPtr
			}
			// MAC is already the key, just normalize it for the index
			normalizedMAC := macaddr.NormalizeFast(mac)
			cm.indexes.GatewaysByMAC[normalizedMAC] = gwPtr
			if gwPtr.SiteID != nil {
				cm.indexes.GatewaysBySite[*gwPtr.SiteID] = append(cm.indexes.GatewaysBySite[*gwPtr.SiteID], gwPtr)
			}
		}

		// Index Device Profiles for this org
		for i := range orgData.Profiles.Devices {
			profile := &orgData.Profiles.Devices[i]
			if profile.Name != nil {
				cm.indexes.DeviceProfilesByName[*profile.Name] = profile
			}
			if profile.ID != nil {
				cm.indexes.DeviceProfilesByID[*profile.ID] = profile
			}
		}

		// Index Device Profile Details for this org
		for i := range orgData.Profiles.Details {
			detail := &orgData.Profiles.Details[i]
			if nameInterface, hasName := (*detail)["name"]; hasName {
				if name, ok := nameInterface.(string); ok {
					cm.indexes.DeviceProfileDetailsByName[name] = detail
				}
			}
			if idInterface, hasID := (*detail)["id"]; hasID {
				if id, ok := idInterface.(string); ok {
					cm.indexes.DeviceProfileDetailsByID[id] = detail
				}
			}
		}

		// Index AP Configs for this org (now a map)
		for mac, config := range orgData.Configs.AP {
			configCopy := config
			configPtr := &configCopy
			if configPtr.Name != nil {
				cm.indexes.APConfigsByName[*configPtr.Name] = configPtr
			}
			// MAC is already the key, just normalize it for the index
			normalizedMAC := macaddr.NormalizeFast(mac)
			cm.indexes.APConfigsByMAC[normalizedMAC] = configPtr
		}

		// Index Switch Configs for this org (now a map)
		for mac, config := range orgData.Configs.Switch {
			configCopy := config
			configPtr := &configCopy
			if configPtr.Name != nil {
				cm.indexes.SwitchConfigsByName[*configPtr.Name] = configPtr
			}
			// MAC is already the key, just normalize it for the index
			normalizedMAC := macaddr.NormalizeFast(mac)
			cm.indexes.SwitchConfigsByMAC[normalizedMAC] = configPtr
		}

		// Index Gateway Configs for this org (now a map)
		for mac, config := range orgData.Configs.Gateway {
			configCopy := config
			configPtr := &configCopy
			if configPtr.Name != nil {
				cm.indexes.GatewayConfigsByName[*configPtr.Name] = configPtr
			}
			// MAC is already the key, just normalize it for the index
			normalizedMAC := macaddr.NormalizeFast(mac)
			cm.indexes.GatewayConfigsByMAC[normalizedMAC] = configPtr
		}
	}

	// Also load RF templates from multi-vendor cache files
	cm.loadRFTemplatesFromMultiVendorCache()

	return nil
}

// loadRFTemplatesFromMultiVendorCache loads RF templates from per-API cache files.
// This enables resolution of Meraki rfProfileId fields which are stored in multi-vendor cache.
func (cm *CacheManager) loadRFTemplatesFromMultiVendorCache() {
	// Get the multi-vendor cache directory from config
	cacheDir := viper.GetString("cache.dir")
	if cacheDir == "" {
		cacheDir = filepath.Dir(cm.cachePath)
	}

	apisDir := filepath.Join(cacheDir, "apis")
	entries, err := os.ReadDir(apisDir)
	if err != nil {
		logging.Debugf("No multi-vendor cache directory found at %s", apisDir)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		// Skip hidden files (metadata files start with .)
		if entry.Name()[0] == '.' {
			continue
		}

		cachePath := filepath.Join(apisDir, entry.Name())
		cm.loadRFTemplatesFromAPICacheFile(cachePath)
	}
}

// loadRFTemplatesFromAPICacheFile loads RF templates from a single API cache file.
func (cm *CacheManager) loadRFTemplatesFromAPICacheFile(cachePath string) {
	data, err := os.ReadFile(cachePath)
	if err != nil {
		logging.Debugf("Failed to read API cache file %s: %v", cachePath, err)
		return
	}

	// Parse just the templates section to avoid loading everything
	var apiCache struct {
		Meta struct {
			Vendor string `json:"vendor"`
		} `json:"meta"`
		Templates struct {
			RF []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"rf"`
		} `json:"templates"`
	}

	if err := json.Unmarshal(data, &apiCache); err != nil {
		logging.Debugf("Failed to parse API cache file %s: %v", cachePath, err)
		return
	}

	// Add RF templates to indexes
	for _, rf := range apiCache.Templates.RF {
		if rf.ID == "" {
			continue
		}

		// Skip if already in index (don't overwrite)
		if _, exists := cm.indexes.RFTemplatesByID[rf.ID]; exists {
			continue
		}

		// Create a minimal MistRFTemplate with just ID and Name
		id := rf.ID
		name := rf.Name
		rfTemplate := &MistRFTemplate{
			ID:   &id,
			Name: &name,
		}

		cm.indexes.RFTemplatesByID[id] = rfTemplate
		if name != "" {
			cm.indexes.RFTemplatesByName[name] = rfTemplate
		}

		logging.Debugf("Indexed RF template from %s cache: %s (%s)", apiCache.Meta.Vendor, name, id)
	}
}

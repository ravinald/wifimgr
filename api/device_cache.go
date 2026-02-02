package api

import (
	"sync"
	"time"

	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/macaddr"
)

// DeviceCache structure with proper indexing using new bidirectional data handling
type DeviceCache struct {
	// Primary storage: all devices by MAC address
	Devices map[string]UnifiedDevice // MAC -> UnifiedDevice

	// Indexes for quick lookups
	SiteIndex map[string][]string // SiteID -> []MAC
	TypeIndex map[string][]string // DeviceType -> []MAC
	NameIndex map[string]string   // Name -> MAC

	// Cache management
	LastUpdated time.Time
	mu          sync.RWMutex

	// Performance metrics
	stats struct {
		hits   int64
		misses int64
		mu     sync.Mutex
	}
}

// NewDeviceCache creates a new device cache
func NewDeviceCache() *DeviceCache {
	return &DeviceCache{
		Devices:     make(map[string]UnifiedDevice),
		SiteIndex:   make(map[string][]string),
		TypeIndex:   make(map[string][]string),
		NameIndex:   make(map[string]string),
		LastUpdated: time.Now(),
	}
}

// Clear removes all devices from the cache
func (c *DeviceCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Devices = make(map[string]UnifiedDevice)
	c.SiteIndex = make(map[string][]string)
	c.TypeIndex = make(map[string][]string)
	c.NameIndex = make(map[string]string)

	c.LastUpdated = time.Now()
}

// Count returns the total number of devices in the cache
func (c *DeviceCache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.Devices)
}

// CountByType returns the number of devices of each type
func (c *DeviceCache) CountByType() map[string]int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	counts := make(map[string]int)
	for deviceType, macs := range c.TypeIndex {
		counts[deviceType] = len(macs)
	}

	return counts
}

// CountBySite returns the number of devices for each site
func (c *DeviceCache) CountBySite() map[string]int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	counts := make(map[string]int)
	for siteID, macs := range c.SiteIndex {
		counts[siteID] = len(macs)
	}

	return counts
}

// Helper function to remove a MAC address from a slice in an index
func (c *DeviceCache) removeMACFromSlice(key string, mac string, index map[string][]string) {
	macs, found := index[key]
	if !found {
		return
	}

	// Find and remove the MAC from the slice
	for i, m := range macs {
		if m == mac {
			// Remove by swapping with the last element and truncating
			macs[i] = macs[len(macs)-1]
			macs = macs[:len(macs)-1]
			index[key] = macs

			// If the slice is empty, remove the key from the index
			if len(macs) == 0 {
				delete(index, key)
			}

			return
		}
	}
}

// Helper function to append a string to a slice only if it doesn't already exist
func appendUniqueString(slice []string, str string) []string {
	for _, s := range slice {
		if s == str {
			return slice
		}
	}
	return append(slice, str)
}

// ConvertToAPSlice converts a slice of UnifiedDevice structs to AP structs
func ConvertToAPSlice(devices []UnifiedDevice) []AP {
	aps := make([]AP, 0, len(devices))

	for _, device := range devices {
		if device.Type == nil || *device.Type != "ap" {
			continue
		}

		ap := AP{}

		// Copy common fields
		if device.ID != nil {
			uuid := UUID(*device.ID)
			ap.Id = &uuid
		}

		ap.Mac = device.MAC
		ap.Serial = device.Serial
		ap.Name = device.Name
		ap.Model = device.Model
		ap.Type = device.Type
		ap.Magic = device.Magic
		ap.HwRev = device.HwRev
		ap.SKU = device.SKU

		if device.SiteID != nil {
			uuid := UUID(*device.SiteID)
			ap.SiteId = &uuid
		}

		ap.OrgId = device.OrgID
		ap.CreatedTime = device.CreatedTime
		ap.ModifiedTime = device.ModifiedTime
		ap.DeviceProfileId = device.DeviceProfileID
		ap.Connected = device.Connected
		ap.Notes = device.Notes

		// Copy AP-specific fields from DeviceConfig
		if locationVal, ok := device.DeviceConfig["location"]; ok {
			if location, ok := locationVal.([]float64); ok {
				ap.Location = &location
			}
		}

		if orientationVal, ok := device.DeviceConfig["orientation"]; ok {
			if orientation, ok := orientationVal.(int); ok {
				ap.Orientation = &orientation
			}
		}

		if mapIDVal, ok := device.DeviceConfig["map_id"]; ok {
			if mapID, ok := mapIDVal.(string); ok {
				ap.MapID = &mapID
			}
		}

		if radioConfigVal, ok := device.DeviceConfig["radio_config"]; ok {
			if radioConfig, ok := radioConfigVal.(*RadioConfig); ok {
				ap.RadioConfig = radioConfig
			} else if radioConfigMap, ok := radioConfigVal.(map[string]interface{}); ok {
				// Try to convert from map to RadioConfig
				radioConfig := convertMapToRadioConfig(radioConfigMap)
				ap.RadioConfig = &radioConfig
			}
		}

		if ledVal, ok := device.DeviceConfig["led"]; ok {
			if led, ok := ledVal.(bool); ok {
				ap.Led = &led
			}
		}

		if statusVal, ok := device.DeviceConfig["status"]; ok {
			if status, ok := statusVal.(string); ok {
				ap.Status = &status
			}
		}

		if tagUUIDVal, ok := device.DeviceConfig["tag_uuid"]; ok {
			if tagUUID, ok := tagUUIDVal.(string); ok {
				ap.TagUUID = &tagUUID
			}
		}

		if tagIDVal, ok := device.DeviceConfig["tag_id"]; ok {
			if tagID, ok := tagIDVal.(int); ok {
				ap.TagID = &tagID
			}
		}

		if evpnScopeVal, ok := device.DeviceConfig["evpn_scope"]; ok {
			if evpnScope, ok := evpnScopeVal.(string); ok {
				ap.EvpnScope = &evpnScope
			}
		}

		if evpntopoIDVal, ok := device.DeviceConfig["evpntopo_id"]; ok {
			if evpntopoID, ok := evpntopoIDVal.(string); ok {
				ap.EvpntopoID = &evpntopoID
			}
		}

		if stIPBaseVal, ok := device.DeviceConfig["st_ip_base"]; ok {
			if stIPBase, ok := stIPBaseVal.(string); ok {
				ap.StIPBase = &stIPBase
			}
		}

		if bundledMacVal, ok := device.DeviceConfig["bundled_mac"]; ok {
			if bundledMac, ok := bundledMacVal.(string); ok {
				ap.BundledMac = &bundledMac
			}
		}

		aps = append(aps, ap)
	}

	return aps
}

// AddDevice adds or updates a device in the cache
func (c *DeviceCache) AddDevice(device UnifiedDevice) {
	if device.MAC == nil {
		return // Cannot add a device without a MAC address
	}

	// Normalize MAC address for consistent lookups
	normalizedMAC, err := macaddr.Normalize(*device.MAC)
	if err != nil {
		logging.Warnf("Failed to normalize MAC address %s: %v", *device.MAC, err)
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if the device already exists to update indexes properly
	existing, exists := c.Devices[normalizedMAC]

	// Add to primary storage
	c.Devices[normalizedMAC] = device

	// Update SiteIndex
	if device.SiteID != nil {
		siteID := *device.SiteID

		// If existing, remove from old site index if site changed
		if exists && existing.SiteID != nil && *existing.SiteID != siteID {
			c.removeMACFromSlice(*existing.SiteID, normalizedMAC, c.SiteIndex)
		}

		// Add to new site index
		c.SiteIndex[siteID] = appendUniqueString(c.SiteIndex[siteID], normalizedMAC)
	} else if exists && existing.SiteID != nil {
		// Remove from old site index if site ID is now nil
		c.removeMACFromSlice(*existing.SiteID, normalizedMAC, c.SiteIndex)
	}

	// Update TypeIndex
	deviceType := device.DeviceType
	if deviceType == "" && device.Type != nil {
		deviceType = *device.Type
	}

	if deviceType != "" {
		// If existing, remove from old type index if type changed
		if exists && existing.DeviceType != "" && existing.DeviceType != deviceType {
			c.removeMACFromSlice(existing.DeviceType, normalizedMAC, c.TypeIndex)
		}

		// Add to new type index
		c.TypeIndex[deviceType] = appendUniqueString(c.TypeIndex[deviceType], normalizedMAC)
	} else if exists && existing.DeviceType != "" {
		// Remove from old type index if type is now empty
		c.removeMACFromSlice(existing.DeviceType, normalizedMAC, c.TypeIndex)
	}

	// Update NameIndex
	if device.Name != nil && *device.Name != "" {
		deviceName := *device.Name

		// If an existing device had a different name, remove the old mapping
		if exists && existing.Name != nil && *existing.Name != deviceName {
			delete(c.NameIndex, *existing.Name)
		}

		// Add the new name mapping
		c.NameIndex[deviceName] = normalizedMAC
	} else if exists && existing.Name != nil {
		// If the device no longer has a name, remove the old mapping
		delete(c.NameIndex, *existing.Name)
	}

	c.LastUpdated = time.Now()
}

// GetDeviceByMAC retrieves a device by MAC address
func (c *DeviceCache) GetDeviceByMAC(mac string) (UnifiedDevice, bool) {
	normalizedMAC, err := macaddr.Normalize(mac)
	if err != nil {
		c.recordMiss()
		return UnifiedDevice{}, false
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	device, found := c.Devices[normalizedMAC]
	if found {
		c.recordHit()
	} else {
		c.recordMiss()
	}
	return device, found
}

// GetDeviceByName retrieves a device by name
func (c *DeviceCache) GetDeviceByName(name string) (UnifiedDevice, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	mac, found := c.NameIndex[name]
	if !found {
		return UnifiedDevice{}, false
	}

	device, found := c.Devices[mac]
	return device, found
}

// GetDevicesBySite retrieves all devices for a site
func (c *DeviceCache) GetDevicesBySite(siteID string) []UnifiedDevice {
	c.mu.RLock()
	defer c.mu.RUnlock()

	macs, found := c.SiteIndex[siteID]
	if !found {
		return []UnifiedDevice{}
	}

	devices := make([]UnifiedDevice, 0, len(macs))
	for _, mac := range macs {
		if device, found := c.Devices[mac]; found {
			devices = append(devices, device)
		}
	}

	return devices
}

// GetDevicesByType retrieves all devices of a specific type
func (c *DeviceCache) GetDevicesByType(deviceType string) []UnifiedDevice {
	c.mu.RLock()
	defer c.mu.RUnlock()

	macs, found := c.TypeIndex[deviceType]
	if !found {
		return []UnifiedDevice{}
	}

	devices := make([]UnifiedDevice, 0, len(macs))
	for _, mac := range macs {
		if device, found := c.Devices[mac]; found {
			devices = append(devices, device)
		}
	}

	return devices
}

// GetDevicesBySiteAndType retrieves all devices for a site of a specific type
func (c *DeviceCache) GetDevicesBySiteAndType(siteID, deviceType string) []UnifiedDevice {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Get all devices for the site
	macs, found := c.SiteIndex[siteID]
	if !found {
		return []UnifiedDevice{}
	}

	devices := make([]UnifiedDevice, 0)
	for _, mac := range macs {
		if device, found := c.Devices[mac]; found {
			// Check if the device is of the requested type
			if (device.DeviceType == deviceType) ||
				(device.Type != nil && *device.Type == deviceType) {
				devices = append(devices, device)
			}
		}
	}

	return devices
}

// RemoveDevice removes a device from the cache
func (c *DeviceCache) RemoveDevice(mac string) {
	normalizedMAC, err := macaddr.Normalize(mac)
	if err != nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Get the existing device to update indexes
	existing, exists := c.Devices[normalizedMAC]
	if !exists {
		return
	}

	// Remove from SiteIndex
	if existing.SiteID != nil {
		c.removeMACFromSlice(*existing.SiteID, normalizedMAC, c.SiteIndex)
	}

	// Remove from TypeIndex
	deviceType := existing.DeviceType
	if deviceType == "" && existing.Type != nil {
		deviceType = *existing.Type
	}

	if deviceType != "" {
		c.removeMACFromSlice(deviceType, normalizedMAC, c.TypeIndex)
	}

	// Remove from NameIndex
	if existing.Name != nil && *existing.Name != "" {
		delete(c.NameIndex, *existing.Name)
	}

	// Remove from primary storage
	delete(c.Devices, normalizedMAC)

	c.LastUpdated = time.Now()
}

// GetAllDevices returns all devices in the cache
func (c *DeviceCache) GetAllDevices() []UnifiedDevice {
	c.mu.RLock()
	defer c.mu.RUnlock()

	devices := make([]UnifiedDevice, 0, len(c.Devices))
	for _, device := range c.Devices {
		devices = append(devices, device)
	}

	return devices
}

// MergeDeviceInfo merges new device information with existing data
func (c *DeviceCache) MergeDeviceInfo(device UnifiedDevice) {
	if device.MAC == nil {
		return // Cannot merge a device without a MAC address
	}

	normalizedMAC, err := macaddr.Normalize(*device.MAC)
	if err != nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Get existing device if it exists
	existing, found := c.Devices[normalizedMAC]
	if !found {
		// If device doesn't exist yet, just add it
		c.Devices[normalizedMAC] = device

		// Update indexes
		if device.SiteID != nil {
			siteID := *device.SiteID
			c.SiteIndex[siteID] = appendUniqueString(c.SiteIndex[siteID], normalizedMAC)
		}

		deviceType := device.DeviceType
		if deviceType == "" && device.Type != nil {
			deviceType = *device.Type
		}

		if deviceType != "" {
			c.TypeIndex[deviceType] = appendUniqueString(c.TypeIndex[deviceType], normalizedMAC)
		}

		if device.Name != nil && *device.Name != "" {
			c.NameIndex[*device.Name] = normalizedMAC
		}
	} else {
		// Merge the devices and update
		merged := MergeDeviceData(existing, device)
		c.Devices[normalizedMAC] = merged

		// Update indexes if needed

		// SiteIndex
		if merged.SiteID != nil {
			newSiteID := *merged.SiteID

			// If site changed, update indexes
			if existing.SiteID == nil || *existing.SiteID != newSiteID {
				// Remove from old site index
				if existing.SiteID != nil {
					c.removeMACFromSlice(*existing.SiteID, normalizedMAC, c.SiteIndex)
				}

				// Add to new site index
				c.SiteIndex[newSiteID] = appendUniqueString(c.SiteIndex[newSiteID], normalizedMAC)
			}
		}

		// TypeIndex
		deviceType := merged.DeviceType
		if deviceType == "" && merged.Type != nil {
			deviceType = *merged.Type
		}

		existingType := existing.DeviceType
		if existingType == "" && existing.Type != nil {
			existingType = *existing.Type
		}

		if deviceType != "" && deviceType != existingType {
			// Remove from old type index
			if existingType != "" {
				c.removeMACFromSlice(existingType, normalizedMAC, c.TypeIndex)
			}

			// Add to new type index
			c.TypeIndex[deviceType] = appendUniqueString(c.TypeIndex[deviceType], normalizedMAC)
		}

		// NameIndex
		if merged.Name != nil && *merged.Name != "" {
			newName := *merged.Name

			// If name changed, update index
			if existing.Name == nil || *existing.Name != newName {
				// Remove old name mapping
				if existing.Name != nil && *existing.Name != "" {
					delete(c.NameIndex, *existing.Name)
				}

				// Add new name mapping
				c.NameIndex[newName] = normalizedMAC
			}
		}
	}

	c.LastUpdated = time.Now()
}

// Helper function to convert a map to RadioConfig
func convertMapToRadioConfig(m map[string]interface{}) RadioConfig {
	rc := RadioConfig{}

	// Handle top-level RadioConfig fields
	if allowRRMDisableVal, ok := m["allow_rrm_disable"]; ok {
		if allowRRMDisable, ok := allowRRMDisableVal.(bool); ok {
			rc.AllowRRMDisable = &allowRRMDisable
		}
	}

	if antGain24Val, ok := m["ant_gain_24"]; ok {
		if antGain24, ok := antGain24Val.(float64); ok {
			rc.AntGain24 = &antGain24
		}
	}

	if antGain5Val, ok := m["ant_gain_5"]; ok {
		if antGain5, ok := antGain5Val.(float64); ok {
			rc.AntGain5 = &antGain5
		}
	}

	if antGain6Val, ok := m["ant_gain_6"]; ok {
		if antGain6, ok := antGain6Val.(float64); ok {
			rc.AntGain6 = &antGain6
		}
	}

	if antennaModeVal, ok := m["antenna_mode"]; ok {
		if antennaMode, ok := antennaModeVal.(string); ok {
			rc.AntennaMode = &antennaMode
		}
	}

	if band24UsageVal, ok := m["band_24_usage"]; ok {
		if band24Usage, ok := band24UsageVal.(string); ok {
			rc.Band24Usage = &band24Usage
		}
	}

	if fullAutomaticRRMVal, ok := m["full_automatic_rrm"]; ok {
		if fullAutomaticRRM, ok := fullAutomaticRRMVal.(bool); ok {
			rc.FullAutomaticRRM = &fullAutomaticRRM
		}
	}

	if indoorUseVal, ok := m["indoor_use"]; ok {
		if indoorUse, ok := indoorUseVal.(bool); ok {
			rc.IndoorUse = &indoorUse
		}
	}

	if scanningEnabledVal, ok := m["scanning_enabled"]; ok {
		if scanningEnabled, ok := scanningEnabledVal.(bool); ok {
			rc.ScanningEnabled = &scanningEnabled
		}
	}

	// Handle 2.4GHz band
	if band24Val, ok := m["band_24"]; ok {
		if band24Map, ok := band24Val.(map[string]interface{}); ok {
			band24 := RadioConfigBand24{}

			if disabledVal, ok := band24Map["disabled"]; ok {
				if disabled, ok := disabledVal.(bool); ok {
					band24.Disabled = &disabled
				}
			}

			if allowRRMDisableVal, ok := band24Map["allow_rrm_disable"]; ok {
				if allowRRMDisable, ok := allowRRMDisableVal.(bool); ok {
					band24.AllowRRMDisable = &allowRRMDisable
				}
			}

			if antGainVal, ok := band24Map["ant_gain"]; ok {
				if antGain, ok := antGainVal.(float64); ok {
					band24.AntGain = &antGain
				}
			}

			if antennaModeVal, ok := band24Map["antenna_mode"]; ok {
				if antennaMode, ok := antennaModeVal.(string); ok {
					band24.AntennaMode = &antennaMode
				}
			}

			if bandwidthVal, ok := band24Map["bandwidth"]; ok {
				if bandwidthStr, ok := bandwidthVal.(string); ok {
					band24.Bandwidth = &bandwidthStr
				} else if bandwidthInt, ok := bandwidthVal.(int); ok {
					// Convert int to string for bandwidth
					bandwidthStr := StringFromInt(bandwidthInt)
					band24.Bandwidth = &bandwidthStr
				} else if bandwidthFloat, ok := bandwidthVal.(float64); ok {
					bandwidthStr := StringFromInt(int(bandwidthFloat))
					band24.Bandwidth = &bandwidthStr
				}
			}

			if channelVal, ok := band24Map["channel"]; ok {
				if channel, ok := channelVal.(int); ok {
					band24.Channel = &channel
				} else if channelFloat, ok := channelVal.(float64); ok {
					channel := int(channelFloat)
					band24.Channel = &channel
				}
			}

			if channelsVal, ok := band24Map["channels"]; ok {
				if channelsArr, ok := channelsVal.([]int); ok {
					band24.Channels = &channelsArr
				}
			}

			if powerVal, ok := band24Map["power"]; ok {
				if power, ok := powerVal.(int); ok {
					band24.Power = &power
				} else if powerFloat, ok := powerVal.(float64); ok {
					power := int(powerFloat)
					band24.Power = &power
				}
			}

			if powerMaxVal, ok := band24Map["power_max"]; ok {
				if powerMax, ok := powerMaxVal.(int); ok {
					band24.PowerMax = &powerMax
				} else if powerMaxFloat, ok := powerMaxVal.(float64); ok {
					powerMax := int(powerMaxFloat)
					band24.PowerMax = &powerMax
				}
			}

			if powerMinVal, ok := band24Map["power_min"]; ok {
				if powerMin, ok := powerMinVal.(int); ok {
					band24.PowerMin = &powerMin
				} else if powerMinFloat, ok := powerMinVal.(float64); ok {
					powerMin := int(powerMinFloat)
					band24.PowerMin = &powerMin
				}
			}

			if preambleVal, ok := band24Map["preamble"]; ok {
				if preamble, ok := preambleVal.(string); ok {
					band24.Preamble = &preamble
				}
			}

			rc.Band24 = &band24
		}
	}

	// Handle 5GHz band
	if band5Val, ok := m["band_5"]; ok {
		if band5Map, ok := band5Val.(map[string]interface{}); ok {
			band5 := RadioConfigBand5{}

			if disabledVal, ok := band5Map["disabled"]; ok {
				if disabled, ok := disabledVal.(bool); ok {
					band5.Disabled = &disabled
				}
			}

			if allowRRMDisableVal, ok := band5Map["allow_rrm_disable"]; ok {
				if allowRRMDisable, ok := allowRRMDisableVal.(bool); ok {
					band5.AllowRRMDisable = &allowRRMDisable
				}
			}

			if antGainVal, ok := band5Map["ant_gain"]; ok {
				if antGain, ok := antGainVal.(float64); ok {
					band5.AntGain = &antGain
				}
			}

			if antennaModeVal, ok := band5Map["antenna_mode"]; ok {
				if antennaMode, ok := antennaModeVal.(string); ok {
					band5.AntennaMode = &antennaMode
				}
			}

			if bandwidthVal, ok := band5Map["bandwidth"]; ok {
				if bandwidthStr, ok := bandwidthVal.(string); ok {
					band5.Bandwidth = &bandwidthStr
				} else if bandwidthInt, ok := bandwidthVal.(int); ok {
					// Convert int to string for bandwidth
					bandwidthStr := StringFromInt(bandwidthInt)
					band5.Bandwidth = &bandwidthStr
				} else if bandwidthFloat, ok := bandwidthVal.(float64); ok {
					bandwidthStr := StringFromInt(int(bandwidthFloat))
					band5.Bandwidth = &bandwidthStr
				}
			}

			if channelVal, ok := band5Map["channel"]; ok {
				if channel, ok := channelVal.(int); ok {
					band5.Channel = &channel
				} else if channelFloat, ok := channelVal.(float64); ok {
					channel := int(channelFloat)
					band5.Channel = &channel
				}
			}

			if channelsVal, ok := band5Map["channels"]; ok {
				if channelsArr, ok := channelsVal.([]int); ok {
					band5.Channels = &channelsArr
				}
			}

			if powerVal, ok := band5Map["power"]; ok {
				if power, ok := powerVal.(int); ok {
					band5.Power = &power
				} else if powerFloat, ok := powerVal.(float64); ok {
					power := int(powerFloat)
					band5.Power = &power
				}
			}

			if powerMaxVal, ok := band5Map["power_max"]; ok {
				if powerMax, ok := powerMaxVal.(int); ok {
					band5.PowerMax = &powerMax
				} else if powerMaxFloat, ok := powerMaxVal.(float64); ok {
					powerMax := int(powerMaxFloat)
					band5.PowerMax = &powerMax
				}
			}

			if powerMinVal, ok := band5Map["power_min"]; ok {
				if powerMin, ok := powerMinVal.(int); ok {
					band5.PowerMin = &powerMin
				} else if powerMinFloat, ok := powerMinVal.(float64); ok {
					powerMin := int(powerMinFloat)
					band5.PowerMin = &powerMin
				}
			}

			if preambleVal, ok := band5Map["preamble"]; ok {
				if preamble, ok := preambleVal.(string); ok {
					band5.Preamble = &preamble
				}
			}

			rc.Band5 = &band5
		}
	}

	// Handle 5GHz on 2.4GHz radio
	if band5On24RadioVal, ok := m["band_5_on_24_radio"]; ok {
		if band5On24RadioMap, ok := band5On24RadioVal.(map[string]interface{}); ok {
			band5On24Radio := RadioConfigBand5{}

			// Same conversion pattern as band_5
			if disabledVal, ok := band5On24RadioMap["disabled"]; ok {
				if disabled, ok := disabledVal.(bool); ok {
					band5On24Radio.Disabled = &disabled
				}
			}

			if channelVal, ok := band5On24RadioMap["channel"]; ok {
				if channel, ok := channelVal.(int); ok {
					band5On24Radio.Channel = &channel
				} else if channelFloat, ok := channelVal.(float64); ok {
					channel := int(channelFloat)
					band5On24Radio.Channel = &channel
				}
			}

			// Add other field conversions similar to band_5...

			rc.Band5On24Radio = &band5On24Radio
		}
	}

	// Handle 6GHz band
	if band6Val, ok := m["band_6"]; ok {
		if band6Map, ok := band6Val.(map[string]interface{}); ok {
			band6 := RadioConfigBand6{}

			if disabledVal, ok := band6Map["disabled"]; ok {
				if disabled, ok := disabledVal.(bool); ok {
					band6.Disabled = &disabled
				}
			}

			if allowRRMDisableVal, ok := band6Map["allow_rrm_disable"]; ok {
				if allowRRMDisable, ok := allowRRMDisableVal.(bool); ok {
					band6.AllowRRMDisable = &allowRRMDisable
				}
			}

			if antGainVal, ok := band6Map["ant_gain"]; ok {
				if antGain, ok := antGainVal.(float64); ok {
					band6.AntGain = &antGain
				}
			}

			if antennaModeVal, ok := band6Map["antenna_mode"]; ok {
				if antennaMode, ok := antennaModeVal.(string); ok {
					band6.AntennaMode = &antennaMode
				}
			}

			if bandwidthVal, ok := band6Map["bandwidth"]; ok {
				if bandwidthStr, ok := bandwidthVal.(string); ok {
					band6.Bandwidth = &bandwidthStr
				} else if bandwidthInt, ok := bandwidthVal.(int); ok {
					// Convert int to string for bandwidth
					bandwidthStr := StringFromInt(bandwidthInt)
					band6.Bandwidth = &bandwidthStr
				} else if bandwidthFloat, ok := bandwidthVal.(float64); ok {
					bandwidthStr := StringFromInt(int(bandwidthFloat))
					band6.Bandwidth = &bandwidthStr
				}
			}

			if channelVal, ok := band6Map["channel"]; ok {
				if channel, ok := channelVal.(int); ok {
					band6.Channel = &channel
				} else if channelFloat, ok := channelVal.(float64); ok {
					channel := int(channelFloat)
					band6.Channel = &channel
				}
			}

			if channelsVal, ok := band6Map["channels"]; ok {
				if channelsArr, ok := channelsVal.([]int); ok {
					band6.Channels = &channelsArr
				}
			}

			if powerVal, ok := band6Map["power"]; ok {
				if power, ok := powerVal.(int); ok {
					band6.Power = &power
				} else if powerFloat, ok := powerVal.(float64); ok {
					power := int(powerFloat)
					band6.Power = &power
				}
			}

			if powerMaxVal, ok := band6Map["power_max"]; ok {
				if powerMax, ok := powerMaxVal.(int); ok {
					band6.PowerMax = &powerMax
				} else if powerMaxFloat, ok := powerMaxVal.(float64); ok {
					powerMax := int(powerMaxFloat)
					band6.PowerMax = &powerMax
				}
			}

			if powerMinVal, ok := band6Map["power_min"]; ok {
				if powerMin, ok := powerMinVal.(int); ok {
					band6.PowerMin = &powerMin
				} else if powerMinFloat, ok := powerMinVal.(float64); ok {
					powerMin := int(powerMinFloat)
					band6.PowerMin = &powerMin
				}
			}

			if preambleVal, ok := band6Map["preamble"]; ok {
				if preamble, ok := preambleVal.(string); ok {
					band6.Preamble = &preamble
				}
			}

			if standardPowerVal, ok := band6Map["standard_power"]; ok {
				if standardPower, ok := standardPowerVal.(bool); ok {
					band6.StandardPower = &standardPower
				}
			}

			rc.Band6 = &band6
		}
	}

	return rc
}

// recordHit records a cache hit
func (c *DeviceCache) recordHit() {
	c.stats.mu.Lock()
	c.stats.hits++
	c.stats.mu.Unlock()
}

// recordMiss records a cache miss
func (c *DeviceCache) recordMiss() {
	c.stats.mu.Lock()
	c.stats.misses++
	c.stats.mu.Unlock()
}

// GetCacheStats returns cache performance statistics
func (c *DeviceCache) GetCacheStats() (hits, misses int64, hitRate float64) {
	c.stats.mu.Lock()
	defer c.stats.mu.Unlock()

	hits = c.stats.hits
	misses = c.stats.misses
	total := hits + misses
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}
	return
}

package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/macaddr"
)

// Device-related methods for the mistClient using the unified device model

// Legacy device methods have been removed.
// Use the bidirectional device methods (GetDevices, GetDeviceByMAC, etc.) instead.

// Device-related methods using the new bidirectional device types

// GetDevices retrieves all devices of a specific type for a site using the new bidirectional pattern
func (c *mistClient) GetDevices(ctx context.Context, siteID string, deviceType string) ([]UnifiedDevice, error) {
	// Check in-memory device cache first if it's initialized
	if deviceCache != nil {
		var cachedDevices []UnifiedDevice

		if deviceType == "" || deviceType == "all" {
			// Get all devices for the site
			cachedDevices = deviceCache.GetDevicesBySite(siteID)
		} else {
			// Get devices of specific type for the site
			cachedDevices = deviceCache.GetDevicesBySiteAndType(siteID, deviceType)
		}

		if len(cachedDevices) > 0 {
			c.logDebug("In-memory cache hit for %s devices in site %s: returning %d devices", deviceType, siteID, len(cachedDevices))
			return cachedDevices, nil
		}
		c.logDebug("In-memory cache miss for %s devices in site %s", deviceType, siteID)
	}

	// Check file cache before making API call
	cacheAccessor := c.GetCacheAccessor()
	if cacheAccessor != nil {
		c.logDebug("Checking file cache for %s devices in site %s", deviceType, siteID)

		var fileDevices []UnifiedDevice

		switch deviceType {
		case "ap":
			configs, err := cacheAccessor.GetAPsBySite(siteID)
			if err == nil && len(configs) > 0 {
				for _, config := range configs {
					device := NewUnifiedDeviceFromType("ap")
					device.ID = config.ID
					device.Name = config.Name
					device.MAC = config.MAC
					device.Serial = config.Serial
					device.Model = config.Model
					device.Type = StringPtr("ap")
					device.SiteID = config.SiteID
					device.OrgID = config.OrgID
					device.Notes = config.Notes
					device.Tags = config.Tags
					device.DeviceProfileID = config.DeviceProfileID
					device.BaseDevice = config.BaseDevice
					fileDevices = append(fileDevices, *device)
				}
				c.logDebug("Found %d APs in file cache for site %s", len(fileDevices), siteID)
			}
		case "switch":
			configs, err := cacheAccessor.GetSwitchesBySite(siteID)
			if err == nil && len(configs) > 0 {
				for _, config := range configs {
					device := NewUnifiedDeviceFromType("switch")
					device.ID = config.ID
					device.Name = config.Name
					device.MAC = config.MAC
					device.Serial = config.Serial
					device.Model = config.Model
					device.Type = StringPtr("switch")
					device.SiteID = config.SiteID
					device.OrgID = config.OrgID
					device.Notes = config.Notes
					device.Tags = config.Tags
					device.DeviceProfileID = config.DeviceProfileID
					device.BaseDevice = config.BaseDevice
					fileDevices = append(fileDevices, *device)
				}
				c.logDebug("Found %d switches in file cache for site %s", len(fileDevices), siteID)
			}
		case "gateway":
			configs, err := cacheAccessor.GetGatewaysBySite(siteID)
			if err == nil && len(configs) > 0 {
				for _, config := range configs {
					device := NewUnifiedDeviceFromType("gateway")
					device.ID = config.ID
					device.Name = config.Name
					device.MAC = config.MAC
					device.Serial = config.Serial
					device.Model = config.Model
					device.Type = StringPtr("gateway")
					device.SiteID = config.SiteID
					device.OrgID = config.OrgID
					device.Notes = config.Notes
					device.Tags = config.Tags
					device.DeviceProfileID = config.DeviceProfileID
					device.BaseDevice = config.BaseDevice
					fileDevices = append(fileDevices, *device)
				}
				c.logDebug("Found %d gateways in file cache for site %s", len(fileDevices), siteID)
			}
		case "", "all":
			// Get all device types
			apConfigs, _ := cacheAccessor.GetAPsBySite(siteID)
			for _, config := range apConfigs {
				device := NewUnifiedDeviceFromType("ap")
				device.ID = config.ID
				device.Name = config.Name
				device.MAC = config.MAC
				device.Serial = config.Serial
				device.Model = config.Model
				device.Type = StringPtr("ap")
				device.SiteID = config.SiteID
				device.OrgID = config.OrgID
				device.Notes = config.Notes
				device.Tags = config.Tags
				device.DeviceProfileID = config.DeviceProfileID
				device.BaseDevice = config.BaseDevice
				fileDevices = append(fileDevices, *device)
			}

			switchConfigs, _ := cacheAccessor.GetSwitchesBySite(siteID)
			for _, config := range switchConfigs {
				device := NewUnifiedDeviceFromType("switch")
				device.ID = config.ID
				device.Name = config.Name
				device.MAC = config.MAC
				device.Serial = config.Serial
				device.Model = config.Model
				device.Type = StringPtr("switch")
				device.SiteID = config.SiteID
				device.OrgID = config.OrgID
				device.Notes = config.Notes
				device.Tags = config.Tags
				device.DeviceProfileID = config.DeviceProfileID
				device.BaseDevice = config.BaseDevice
				fileDevices = append(fileDevices, *device)
			}

			gatewayConfigs, _ := cacheAccessor.GetGatewaysBySite(siteID)
			for _, config := range gatewayConfigs {
				device := NewUnifiedDeviceFromType("gateway")
				device.ID = config.ID
				device.Name = config.Name
				device.MAC = config.MAC
				device.Serial = config.Serial
				device.Model = config.Model
				device.Type = StringPtr("gateway")
				device.SiteID = config.SiteID
				device.OrgID = config.OrgID
				device.Notes = config.Notes
				device.Tags = config.Tags
				device.DeviceProfileID = config.DeviceProfileID
				device.BaseDevice = config.BaseDevice
				fileDevices = append(fileDevices, *device)
			}
			c.logDebug("Found %d total devices in file cache for site %s", len(fileDevices), siteID)
		}

		if len(fileDevices) > 0 {
			// Populate in-memory cache with file cache data
			if deviceCache == nil {
				InitializeDeviceCache()
			}
			for _, device := range fileDevices {
				deviceCache.AddDevice(device)
			}
			c.logDebug("File cache hit for %s devices in site %s: returning %d devices", deviceType, siteID, len(fileDevices))
			return fileDevices, nil
		}

		c.logDebug("File cache miss for %s devices in site %s", deviceType, siteID)
	}

	// If deviceType is "all", use empty string to get all device types
	apiDeviceType := deviceType
	if deviceType == "all" {
		apiDeviceType = ""
	}

	c.logDebug("Getting %s devices for site %s from API using new bidirectional pattern", deviceType, siteID)

	// Determine the results limit to use
	limit := 100 // Default value
	if c.config.ResultsLimit > 0 {
		limit = c.config.ResultsLimit
		c.logDebug("Using configured results limit: %d", limit)
	}

	var devices []UnifiedDevice
	page := 1
	hasMore := true

	for hasMore {
		c.logDebug("Fetching %s devices page %d with limit %d for site %s", deviceType, page, limit, siteID)

		// Build query parameters
		query := url.Values{}
		if apiDeviceType != "" {
			query.Set("type", apiDeviceType)
		}
		query.Set("limit", fmt.Sprintf("%d", limit))
		if page > 1 {
			query.Set("page", fmt.Sprintf("%d", page))
		}

		// Build the path with query parameters
		path := fmt.Sprintf("/sites/%s/devices?%s", siteID, query.Encode())

		// Use raw JSON unmarshaling to preserve all API response data
		var rawDevices []map[string]interface{}
		if err := c.do(ctx, http.MethodGet, path, nil, &rawDevices); err != nil {
			return nil, fmt.Errorf("failed to get %s devices: %w", deviceType, err)
		}

		if len(rawDevices) == 0 {
			hasMore = false
			continue
		}

		c.logDebug("API request successful, received %d raw devices", len(rawDevices))

		// Convert raw device data to UnifiedDevice using the bidirectional pattern
		pageDevices := make([]UnifiedDevice, 0, len(rawDevices))
		for _, rawDevice := range rawDevices {
			device, err := NewUnifiedDeviceFromMap(rawDevice)
			if err != nil {
				c.logDebug("Failed to convert raw device to UnifiedDevice: %v", err)
				continue
			}
			pageDevices = append(pageDevices, *device)
		}

		// Add the current page of devices to the result
		devices = append(devices, pageDevices...)

		c.logDebug("Converted %d raw devices to UnifiedDevice structs", len(pageDevices))

		// Check if we've received fewer devices than the limit, indicating the last page
		if len(rawDevices) < limit {
			hasMore = false
		} else {
			page++
		}
	}

	c.logDebug("Total devices retrieved: %d", len(devices))

	// Populate device cache with the fetched devices
	if deviceCache != nil {
		for _, device := range devices {
			deviceCache.AddDevice(device)
		}
		c.logDebug("Added %d devices to device cache", len(devices))
	}

	return devices, nil
}

// GetDeviceByMAC retrieves a device by MAC address using the new bidirectional pattern
func (c *mistClient) GetDeviceByMAC(_ context.Context, mac string) (*UnifiedDevice, error) {
	// Normalize the MAC address
	normalizedMAC, err := macaddr.Normalize(mac)
	if err != nil {
		return nil, fmt.Errorf("invalid MAC address %s: %w", mac, err)
	}
	c.logDebug("Getting device by MAC: %s (normalized: %s)", mac, normalizedMAC)

	// OPTIMIZATION: Check in-memory deviceCache FIRST (most likely to have current data)
	if deviceCache != nil {
		if cachedDevice, found := deviceCache.GetDeviceByMAC(normalizedMAC); found {
			c.logDebug("Found device in memory cache for MAC %s", normalizedMAC)
			return &cachedDevice, nil
		}
	}

	// Fallback to CacheAccessor if not in memory cache (for backward compatibility)
	cacheAccessor := c.GetCacheAccessor()
	if cacheAccessor == nil {
		c.logDebug("CacheAccessor is nil")
	}
	if cacheAccessor != nil {
		// Try AP config first
		apConfig, err := cacheAccessor.GetAPConfigByMAC(normalizedMAC)
		if err != nil {
			c.logDebug("GetAPConfigByMAC error for %s: %v", normalizedMAC, err)
		}
		if err == nil && apConfig != nil {
			c.logDebug("Found AP config in file cache for MAC %s", normalizedMAC)
			// Convert APConfig to UnifiedDevice
			unified := NewUnifiedDeviceFromType("ap")
			unified.MAC = apConfig.MAC
			unified.Name = apConfig.Name
			unified.ID = apConfig.ID
			unified.SiteID = apConfig.SiteID
			unified.Serial = apConfig.Serial
			unified.Model = apConfig.Model
			unified.Type = StringPtr("ap")
			unified.Magic = apConfig.Magic
			unified.Notes = apConfig.Notes
			unified.Tags = apConfig.Tags
			unified.DeviceProfileID = apConfig.DeviceProfileID
			// Copy base device fields
			unified.BaseDevice = apConfig.BaseDevice
			unified.DeviceConfig = make(map[string]interface{})
			return unified, nil
		}

		// Try Switch config
		swConfig, err := cacheAccessor.GetSwitchConfigByMAC(normalizedMAC)
		if err == nil && swConfig != nil {
			c.logDebug("Found Switch config in file cache for MAC %s", normalizedMAC)
			// Convert SwitchConfig to UnifiedDevice
			unified := NewUnifiedDeviceFromType("switch")
			unified.MAC = swConfig.MAC
			unified.Name = swConfig.Name
			unified.ID = swConfig.ID
			unified.SiteID = swConfig.SiteID
			unified.Serial = swConfig.Serial
			unified.Model = swConfig.Model
			unified.Type = StringPtr("switch")
			unified.Magic = swConfig.Magic
			unified.Notes = swConfig.Notes
			unified.Tags = swConfig.Tags
			unified.DeviceProfileID = swConfig.DeviceProfileID
			unified.BaseDevice = swConfig.BaseDevice
			unified.DeviceConfig = make(map[string]interface{})
			return unified, nil
		}

		// Try Gateway config
		gwConfig, err := cacheAccessor.GetGatewayConfigByMAC(normalizedMAC)
		if err == nil && gwConfig != nil {
			c.logDebug("Found Gateway config in file cache for MAC %s", normalizedMAC)
			// Convert GatewayConfig to UnifiedDevice
			unified := NewUnifiedDeviceFromType("gateway")
			unified.MAC = gwConfig.MAC
			unified.Name = gwConfig.Name
			unified.ID = gwConfig.ID
			unified.SiteID = gwConfig.SiteID
			unified.Serial = gwConfig.Serial
			unified.Model = gwConfig.Model
			unified.Type = StringPtr("gateway")
			unified.Magic = gwConfig.Magic
			unified.Notes = gwConfig.Notes
			unified.Tags = gwConfig.Tags
			unified.DeviceProfileID = gwConfig.DeviceProfileID
			unified.BaseDevice = gwConfig.BaseDevice
			unified.DeviceConfig = make(map[string]interface{})
			return unified, nil
		}
	}

	c.logDebug("Device not found in any cache")

	// If not in cache, return not found
	// Note: For a full search implementation, this would need to enumerate organizations and sites
	// For now, this method relies on the cache being populated
	return nil, fmt.Errorf("device with MAC %s not found in cache", normalizedMAC)
}

// UpdateDevice updates a device using the new bidirectional pattern
func (c *mistClient) UpdateDevice(ctx context.Context, siteID string, deviceID string, device *UnifiedDevice) (*UnifiedDevice, error) {
	c.logDebug("Updating device %s in site %s using new bidirectional pattern", deviceID, siteID)

	// Convert device to map for API request
	deviceData := device.ToMap()

	// Remove read-only fields that shouldn't be sent in updates
	readOnlyFields := []string{"id", "mac", "serial", "model", "hw_rev", "sku", "created_time", "modified_time", "connected", "adopted"}
	for _, field := range readOnlyFields {
		delete(deviceData, field)
	}

	c.logDebug("Sending device update with %d fields", len(deviceData))

	// Make the API request
	var rawResponse map[string]interface{}
	path := fmt.Sprintf("/sites/%s/devices/%s", siteID, deviceID)
	err := c.do(ctx, http.MethodPut, path, deviceData, &rawResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to update device: %w", err)
	}

	c.logDebug("Device update successful, received response with %d fields", len(rawResponse))

	// Fetch the complete device configuration from the API to ensure we have the applied config
	// This ensures the cache contains the actual state from the API, not just what we sent
	getPath := fmt.Sprintf("/sites/%s/devices/%s", siteID, deviceID)
	var completeResponse map[string]interface{}
	err = c.do(ctx, http.MethodGet, getPath, nil, &completeResponse)
	if err != nil {
		c.logDebug("Failed to fetch updated device config from API: %v", err)
		// Fall back to the update response if we can't fetch the complete config
		completeResponse = rawResponse
	} else {
		c.logDebug("Fetched complete device config from API with %d fields", len(completeResponse))
	}

	// Convert response back to UnifiedDevice
	updatedDevice, err := NewUnifiedDeviceFromMap(completeResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to convert updated device data: %w", err)
	}

	// Update the in-memory device cache with the complete device config
	if deviceCache != nil && updatedDevice.MAC != nil {
		deviceCache.AddDevice(*updatedDevice)
		c.logDebug("Updated device %s in in-memory cache", *updatedDevice.MAC)
	}

	// Update the file cache with the complete device config
	if err := c.updateDeviceInFileCache(ctx, siteID, updatedDevice); err != nil {
		c.logDebug("Failed to update device in file cache: %v", err)
		// Don't fail the operation if cache update fails
	}

	return updatedDevice, nil
}

// updateDeviceInFileCache updates a specific device in the file cache using the new map structure
func (c *mistClient) updateDeviceInFileCache(_ context.Context, _ string, device *UnifiedDevice) error {
	if device == nil || device.MAC == nil || device.Type == nil {
		return fmt.Errorf("invalid device data for cache update")
	}

	normalizedMAC := macaddr.NormalizeOrEmpty(*device.MAC)
	if normalizedMAC == "" {
		return fmt.Errorf("invalid MAC address")
	}

	cacheAccessor := c.GetCacheAccessor()
	if cacheAccessor == nil {
		return nil // No cache accessor, skip update
	}

	// Get the cache manager to update the file cache
	manager, ok := cacheAccessor.(*CacheAccessorImpl)
	if !ok || manager == nil {
		return nil
	}

	cacheManager := manager.GetManager()
	if cacheManager == nil {
		return nil
	}

	// Get the current cache
	cache, err := cacheManager.GetCache()
	if err != nil || cache == nil {
		return fmt.Errorf("failed to get cache: %w", err)
	}

	// Get or create org data
	orgID := c.config.Organization
	if orgID == "" {
		orgID = c.config.OrgID
	}

	if cache.Orgs == nil {
		cache.Orgs = make(map[string]*OrgData)
	}

	orgData := cache.Orgs[orgID]
	if orgData == nil {
		orgData = &OrgData{}
		cache.Orgs[orgID] = orgData
	}

	// Initialize maps if needed
	if orgData.Configs.AP == nil {
		orgData.Configs.AP = make(map[string]APConfig)
	}
	if orgData.Configs.Switch == nil {
		orgData.Configs.Switch = make(map[string]SwitchConfig)
	}
	if orgData.Configs.Gateway == nil {
		orgData.Configs.Gateway = make(map[string]GatewayConfig)
	}

	// Update the appropriate device config based on type
	switch *device.Type {
	case "ap":
		// Convert UnifiedDevice to APConfig
		apConfig := APConfig{
			BaseDevice: device.BaseDevice,
		}
		// Ensure OrgID is set
		if apConfig.OrgID == nil {
			apConfig.OrgID = &orgID
		}
		// Store the entire config map as AdditionalConfig
		if device.DeviceConfig != nil {
			apConfig.AdditionalConfig = device.DeviceConfig
		}
		// Simply set in map
		orgData.Configs.AP[normalizedMAC] = apConfig

	case "switch":
		// Convert UnifiedDevice to SwitchConfig
		switchConfig := SwitchConfig{
			BaseDevice: device.BaseDevice,
		}
		if switchConfig.OrgID == nil {
			switchConfig.OrgID = &orgID
		}
		if device.DeviceConfig != nil {
			switchConfig.AdditionalConfig = device.DeviceConfig
		}
		// Simply set in map
		orgData.Configs.Switch[normalizedMAC] = switchConfig

	case "gateway":
		// Convert UnifiedDevice to GatewayConfig
		gatewayConfig := GatewayConfig{
			BaseDevice: device.BaseDevice,
		}
		if gatewayConfig.OrgID == nil {
			gatewayConfig.OrgID = &orgID
		}
		if device.DeviceConfig != nil {
			gatewayConfig.AdditionalConfig = device.DeviceConfig
		}
		// Simply set in map
		orgData.Configs.Gateway[normalizedMAC] = gatewayConfig
	}

	// Save the updated cache to disk
	if err := cacheManager.SaveCache(); err != nil {
		return fmt.Errorf("failed to save cache: %w", err)
	}

	c.logDebug("Updated device %s (%s) in file cache", normalizedMAC, *device.Type)
	return nil
}

// AssignDevice assigns a device to a site using the new bidirectional pattern
func (c *mistClient) AssignDevice(ctx context.Context, orgID string, siteID string, mac string) (*UnifiedDevice, error) {
	normalizedMAC, err := macaddr.Normalize(mac)
	if err != nil {
		return nil, fmt.Errorf("invalid MAC address %s: %w", mac, err)
	}
	c.logDebug("Assigning device %s to site %s in org %s using new bidirectional pattern", normalizedMAC, siteID, orgID)

	// Prepare assignment request
	assignmentData := map[string]interface{}{
		"site_id": siteID,
	}

	// Make the API request
	var rawResponse map[string]interface{}
	path := fmt.Sprintf("/orgs/%s/inventory/%s/assign", orgID, normalizedMAC)
	err = c.do(ctx, http.MethodPut, path, assignmentData, &rawResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to assign device: %w", err)
	}

	c.logDebug("Device assignment successful")

	// Get the updated device details
	device, err := c.GetDeviceByMAC(ctx, normalizedMAC)
	if err != nil {
		c.logDebug("Failed to get updated device details after assignment: %v", err)
		// Return a basic device structure if we can't get full details
		basicDevice := NewUnifiedDeviceFromType("unknown")
		basicDevice.MAC = &normalizedMAC
		basicDevice.SiteID = &siteID
		return basicDevice, nil
	}

	return device, nil
}

// GetAPDevice retrieves an AP device with type-specific fields using bidirectional pattern
func (c *mistClient) GetAPDevice(ctx context.Context, siteID string, deviceID string) (*APDevice, error) {
	c.logDebug("Getting AP device %s from site %s using new bidirectional pattern", deviceID, siteID)

	// Get raw device data from API
	var rawDevice map[string]interface{}
	path := fmt.Sprintf("/sites/%s/devices/%s", siteID, deviceID)
	err := c.do(ctx, http.MethodGet, path, nil, &rawDevice)
	if err != nil {
		return nil, fmt.Errorf("failed to get AP device: %w", err)
	}

	c.logDebug("Retrieved raw AP device data with %d fields", len(rawDevice))

	// Convert to APDevice using bidirectional pattern
	device, err := NewAPDeviceFromMap(rawDevice)
	if err != nil {
		return nil, fmt.Errorf("failed to convert AP device data: %w", err)
	}

	return device, nil
}

// UpdateAPDevice updates an AP device using the new bidirectional pattern
func (c *mistClient) UpdateAPDevice(ctx context.Context, siteID string, deviceID string, device *APDevice) (*APDevice, error) {
	c.logDebug("Updating AP device %s in site %s using new bidirectional pattern", deviceID, siteID)

	// Convert device to map for API request
	deviceData := device.ToMap()

	// Remove read-only fields
	readOnlyFields := []string{"id", "mac", "serial", "model", "hw_rev", "sku", "created_time", "modified_time", "connected", "adopted"}
	for _, field := range readOnlyFields {
		delete(deviceData, field)
	}

	c.logDebug("Sending AP device update with %d fields", len(deviceData))

	// Make the API request
	var rawResponse map[string]interface{}
	path := fmt.Sprintf("/sites/%s/devices/%s", siteID, deviceID)
	err := c.do(ctx, http.MethodPut, path, deviceData, &rawResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to update AP device: %w", err)
	}

	c.logDebug("AP device update successful")

	// Convert response back to APDevice
	updatedDevice, err := NewAPDeviceFromMap(rawResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to convert updated AP device data: %w", err)
	}

	return updatedDevice, nil
}

// GetSwitchDevice retrieves a switch device using bidirectional pattern
func (c *mistClient) GetSwitchDevice(ctx context.Context, siteID string, deviceID string) (*MistSwitchDevice, error) {
	c.logDebug("Getting switch device %s from site %s using new bidirectional pattern", deviceID, siteID)

	// Get raw device data from API
	var rawDevice map[string]interface{}
	path := fmt.Sprintf("/sites/%s/devices/%s", siteID, deviceID)
	err := c.do(ctx, http.MethodGet, path, nil, &rawDevice)
	if err != nil {
		return nil, fmt.Errorf("failed to get switch device: %w", err)
	}

	// Convert to MistSwitchDevice using bidirectional pattern
	device, err := NewSwitchDeviceFromMap(rawDevice)
	if err != nil {
		return nil, fmt.Errorf("failed to convert switch device data: %w", err)
	}

	return device, nil
}

// GetGatewayDevice retrieves a gateway device using bidirectional pattern
func (c *mistClient) GetGatewayDevice(ctx context.Context, siteID string, deviceID string) (*MistGatewayDevice, error) {
	c.logDebug("Getting gateway device %s from site %s using new bidirectional pattern", deviceID, siteID)

	// Get raw device data from API
	var rawDevice map[string]interface{}
	path := fmt.Sprintf("/sites/%s/devices/%s", siteID, deviceID)
	err := c.do(ctx, http.MethodGet, path, nil, &rawDevice)
	if err != nil {
		return nil, fmt.Errorf("failed to get gateway device: %w", err)
	}

	// Convert to MistGatewayDevice using bidirectional pattern
	device, err := NewGatewayDeviceFromMap(rawDevice)
	if err != nil {
		return nil, fmt.Errorf("failed to convert gateway device data: %w", err)
	}

	return device, nil
}

// CreateDeviceConfiguration creates a device configuration using the new bidirectional pattern
func (c *mistClient) CreateDeviceConfiguration(_ context.Context, siteID string, device DeviceMarshaler) (map[string]interface{}, error) {
	c.logDebug("Creating device configuration for site %s using new bidirectional pattern", siteID)

	// Convert device to configuration map (filters out status fields)
	var configData map[string]interface{}

	switch d := device.(type) {
	case *UnifiedDevice:
		configData = d.ToConfigMap()
	case *APDevice:
		configData = d.ToConfigMap()
	case *MistSwitchDevice:
		configData = d.ToConfigMap()
	case *MistGatewayDevice:
		configData = d.ToConfigMap()
	default:
		return nil, fmt.Errorf("unsupported device type: %T", device)
	}

	c.logDebug("Generated device configuration with %d fields", len(configData))

	// Return the configuration data for use in config files or further processing
	return configData, nil
}

// ApplyDeviceConfiguration applies a device configuration using the new bidirectional pattern
func (c *mistClient) ApplyDeviceConfiguration(ctx context.Context, siteID string, deviceID string, configData map[string]interface{}) (*UnifiedDevice, error) {
	c.logDebug("Applying device configuration to device %s in site %s using new bidirectional pattern", deviceID, siteID)

	// Make the API request to update the device
	var rawResponse map[string]interface{}
	path := fmt.Sprintf("/sites/%s/devices/%s", siteID, deviceID)
	err := c.do(ctx, http.MethodPut, path, configData, &rawResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to apply device configuration: %w", err)
	}

	c.logDebug("Device configuration applied successfully")

	// Convert response back to UnifiedDevice
	updatedDevice, err := NewUnifiedDeviceFromMap(rawResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to convert updated device data: %w", err)
	}

	return updatedDevice, nil
}

// DeleteDevicesFromSite deletes multiple devices from inventory
func (c *mistClient) DeleteDevicesFromSite(ctx context.Context, macs []string) error {
	if len(macs) == 0 {
		return nil
	}

	// If in dry run mode, log and return simulated success
	if c.dryRun {
		logging.Infof("[DRY RUN] Would delete %d devices from inventory", len(macs))
		return nil
	}

	// Process each device
	for _, mac := range macs {
		// First, we need to find the device ID from the MAC
		device, err := c.GetDeviceByMAC(ctx, mac)
		if err != nil {
			c.logDebug("Could not find device with MAC %s: %v", mac, err)
			continue
		}

		if device.ID == nil {
			c.logDebug("Device with MAC %s found but has no ID", mac)
			continue
		}

		// Delete the device from inventory
		path := fmt.Sprintf("/orgs/%s/inventory/%s", c.config.Organization, *device.ID)

		if err := c.do(ctx, http.MethodDelete, path, nil, nil); err != nil {
			return fmt.Errorf("failed to delete device %s from inventory: %w", mac, err)
		}

		// Legacy cache updates removed - the new cache system handles this automatically
	}

	// Invalidate cache for all device types
	if deviceCache != nil {
		deviceCache.Clear()
	}

	return nil
}

// QueryDeviceExtensive performs an extensive query for a device
func (c *mistClient) QueryDeviceExtensive(_ context.Context, _, _ string) error {
	// This is a placeholder for the extensive device query implementation
	// In a real implementation, this would make additional API calls to gather
	// comprehensive information about the device

	// For example, it might fetch:
	// - Device configuration
	// - Device stats
	// - Connected clients
	// - Error logs
	// - etc.

	// Since the actual implementation depends on specific requirements,
	// we return nil for now
	return nil
}

// GetRawDeviceJSON retrieves the raw JSON for a device
func (c *mistClient) GetRawDeviceJSON(ctx context.Context, siteID, deviceID string) (string, error) {
	path := fmt.Sprintf("/sites/%s/devices/%s", siteID, deviceID)

	apiURL := c.buildURL(path)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Token %s", c.config.APIToken))

	if c.rateLimiter != nil {
		c.rateLimiter.wait()
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("API error: status code %d", resp.StatusCode)
	}

	buf := new(strings.Builder)
	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return buf.String(), nil
}

// Global device cache instance
var deviceCache *DeviceCache

// InitializeDeviceCache initializes the global device cache
func InitializeDeviceCache() {
	if deviceCache == nil {
		deviceCache = NewDeviceCache()
	}
}

// GetDeviceCache returns the global device cache instance
func (c *mistClient) GetDeviceCache() *DeviceCache {
	if deviceCache == nil {
		InitializeDeviceCache()
	}
	return deviceCache
}

// ClearCache clears specific cache types or all caches
// cacheType can be:
//   - "all" - clears everything
//   - "inventory" - clears all inventory caches
//   - "inventory-ap", "inventory-switch", "inventory-gateway" - clears specific inventory type
//   - "configs" - clears all device config caches
//   - "configs-ap", "configs-switch", "configs-gateway" - clears specific device config type
//   - "devices" - clears device cache (alias for configs)
func ClearCache(cacheType string) {
	switch cacheType {
	case "all":
		// Clear all cache types - both inventory and configs
		if deviceCache != nil {
			deviceCache.Clear()
		}
		// Clear inventory caches for all clients
		if client := GetClient(); client != nil {
			if mc, ok := client.(*mistClient); ok {
				if mc.inventoryCache != nil {
					mc.inventoryCache.Clear()
				}
			}
		}

	// Device configs (from /sites/{site_id}/devices API)
	case "configs", "devices", "deviceconfigs":
		// Clear device config cache (all types)
		if deviceCache != nil {
			deviceCache.Clear()
		}
	case "configs-ap":
		// Clear only AP configs from device cache
		if deviceCache != nil {
			// Remove all APs from cache
			for _, device := range deviceCache.GetDevicesByType("ap") {
				if device.MAC != nil {
					deviceCache.RemoveDevice(*device.MAC)
				}
			}
		}
	case "configs-switch":
		// Clear only switch configs from device cache
		if deviceCache != nil {
			for _, device := range deviceCache.GetDevicesByType("switch") {
				if device.MAC != nil {
					deviceCache.RemoveDevice(*device.MAC)
				}
			}
		}
	case "configs-gateway":
		// Clear only gateway configs from device cache
		if deviceCache != nil {
			for _, device := range deviceCache.GetDevicesByType("gateway") {
				if device.MAC != nil {
					deviceCache.RemoveDevice(*device.MAC)
				}
			}
		}

	// Inventory (from /orgs/{org_id}/inventory API)
	case "inventory":
		// Clear all inventory caches
		if client := GetClient(); client != nil {
			if mc, ok := client.(*mistClient); ok {
				if mc.inventoryCache != nil {
					mc.inventoryCache.Clear()
				}
			}
		}
	case "inventory-ap", "inventory-switch", "inventory-gateway":
		// Clear specific inventory type from cache
		if client := GetClient(); client != nil {
			if mc, ok := client.(*mistClient); ok && mc.inventoryCache != nil {
				// Since we don't have a method to remove specific keys, clear all inventory
				// This is acceptable since inventory is relatively small
				mc.inventoryCache.Clear()
			}
		}
	}
}

// ClearCacheForSite clears cache entries for a specific site
func ClearCacheForSite(siteID string, cacheType string) {
	switch cacheType {
	case "all":
		// Clear both configs and inventory for the site
		ClearCacheForSite(siteID, "configs")
		// Inventory is org-level but we clear it for completeness
		ClearCache("inventory")

	case "configs", "devices", "deviceconfigs":
		// Clear all device configs for the site
		if deviceCache != nil {
			devices := deviceCache.GetDevicesBySite(siteID)
			for _, device := range devices {
				if device.MAC != nil {
					deviceCache.RemoveDevice(*device.MAC)
				}
			}
		}

	case "configs-ap":
		// Clear only AP configs for the site
		if deviceCache != nil {
			devices := deviceCache.GetDevicesBySiteAndType(siteID, "ap")
			for _, device := range devices {
				if device.MAC != nil {
					deviceCache.RemoveDevice(*device.MAC)
				}
			}
		}

	case "configs-switch":
		// Clear only switch configs for the site
		if deviceCache != nil {
			devices := deviceCache.GetDevicesBySiteAndType(siteID, "switch")
			for _, device := range devices {
				if device.MAC != nil {
					deviceCache.RemoveDevice(*device.MAC)
				}
			}
		}

	case "configs-gateway":
		// Clear only gateway configs for the site
		if deviceCache != nil {
			devices := deviceCache.GetDevicesBySiteAndType(siteID, "gateway")
			for _, device := range devices {
				if device.MAC != nil {
					deviceCache.RemoveDevice(*device.MAC)
				}
			}
		}

	// Note: Inventory is org-level, not site-level, so we can't clear it per site
	case "inventory", "inventory-ap", "inventory-switch", "inventory-gateway":
		// Inventory is managed at org level, so we clear all inventory
		ClearCache(cacheType)
	}
}

// ForceRebuildCache forces a complete rebuild of the cache
func (c *mistClient) ForceRebuildCache(ctx context.Context) error {
	// Get organization ID
	orgID := c.config.Organization
	if orgID == "" {
		return fmt.Errorf("organization ID is not set in client configuration")
	}

	c.logDebug("Starting cache force rebuild for org %s", orgID)

	// For now, we'll call the individual refresh methods directly
	// This avoids import cycles while still providing the functionality
	dataTypes := []string{"sites", "sitesettings", "inventory-ap", "inventory-switch", "inventory-gateway", "deviceprofiles", "rftemplates", "gatewaytemplates", "wlantemplates", "networks", "wlans", "device-configs"}

	for _, dataType := range dataTypes {
		c.logDebug("Refreshing %s data...", dataType)

		var err error
		switch dataType {
		case "sites":
			_, err = c.GetSites(ctx, orgID)
		case "sitesettings":
			// Site settings require iterating through sites
			sites, siteErr := c.GetSites(ctx, orgID)
			if siteErr != nil {
				err = fmt.Errorf("failed to get sites for settings refresh: %w", siteErr)
			} else {
				for _, site := range sites {
					if site.ID != nil {
						_, settingErr := c.GetSiteSetting(ctx, *site.ID)
						if settingErr != nil {
							c.logDebug("Warning: failed to refresh settings for site %s: %v", *site.ID, settingErr)
						}
					}
				}
			}
		case "inventory-ap":
			_, err = c.GetInventory(ctx, orgID, "ap")
		case "inventory-switch":
			_, err = c.GetInventory(ctx, orgID, "switch")
		case "inventory-gateway":
			_, err = c.GetInventory(ctx, orgID, "gateway")
		case "deviceprofiles":
			_, err = c.GetDeviceProfiles(ctx, orgID, "")
		case "rftemplates":
			_, err = c.GetRFTemplates(ctx, orgID)
		case "gatewaytemplates":
			_, err = c.GetGatewayTemplates(ctx, orgID)
		case "wlantemplates":
			_, err = c.GetWLANTemplates(ctx, orgID)
		case "networks":
			_, err = c.GetNetworks(ctx, orgID)
		case "wlans":
			_, err = c.GetWLANs(ctx, orgID)
		case "device-configs":
			// Refresh device configurations for all sites
			sites, siteErr := c.GetSites(ctx, orgID)
			if siteErr != nil {
				err = fmt.Errorf("failed to get sites for device config refresh: %w", siteErr)
			} else {
				for _, site := range sites {
					if site.ID != nil {
						// Get all device types for each site
						for _, devType := range []string{"ap", "switch", "gateway"} {
							_, devErr := c.GetDevices(ctx, *site.ID, devType)
							if devErr != nil {
								c.logDebug("Warning: failed to refresh %s configs for site %s: %v", devType, *site.ID, devErr)
							}
						}
					}
				}
			}
		}

		if err != nil {
			c.logDebug("Warning: failed to refresh %s: %v", dataType, err)
			// Continue with other data types even if one fails
		}
	}

	c.logDebug("Cache force rebuild completed successfully")
	return nil
}

// UpdateCacheForTypes logs that cache update was requested
// The actual cache update happens automatically on next access if expired
func (c *mistClient) UpdateCacheForTypes(_ context.Context, deviceTypes []string, siteNames []string) error {
	c.logDebug("Cache update requested for device types %v and sites %v", deviceTypes, siteNames)

	// The cache will be automatically refreshed when:
	// 1. It has expired based on TTL
	// 2. A refresh command is explicitly run
	// After apply operations, the cache contains the updated device configurations
	// from the API responses during the apply process

	c.logDebug("Cache will be refreshed on next access if expired")
	return nil
}

// PopulateDeviceCacheForSite populates both the device cache and file cache configs for a specific site and device type
// This updates the configs section of the cache with the latest device configurations from the API
func (c *mistClient) PopulateDeviceCacheForSite(ctx context.Context, siteID string, deviceType string) error {
	c.logDebug("Populating device cache configs for site %s, type %s", siteID, deviceType)

	// Clear the configs cache for this site and device type to force fresh API fetch
	// Note: We do NOT clear inventory - only configs since apply only updates configs
	if deviceType == "" || deviceType == "all" {
		ClearCacheForSite(siteID, "configs")
	} else {
		ClearCacheForSite(siteID, fmt.Sprintf("configs-%s", deviceType))
	}

	// Get device configs from API using GET /sites/<site_id>/devices?type=<device_type>
	// This returns the actual device configurations
	devices, err := c.GetDevices(ctx, siteID, deviceType)
	if err != nil {
		return fmt.Errorf("failed to get device configs: %w", err)
	}

	// Ensure in-memory device cache is initialized
	if deviceCache == nil {
		InitializeDeviceCache()
	}

	// Add devices to in-memory cache
	for _, device := range devices {
		deviceCache.AddDevice(device)
	}

	c.logDebug("Added %d %s device configs to in-memory cache for site %s", len(devices), deviceType, siteID)

	// Update the file cache configs section
	cacheAccessor := c.GetCacheAccessor()
	if cacheAccessor != nil {
		// Get the cache manager to update the file cache
		if manager, ok := cacheAccessor.(*CacheAccessorImpl); ok && manager != nil {
			cacheManager := manager.GetManager()
			if cacheManager != nil {
				// Get the current cache
				cache, err := cacheManager.GetCache()
				if err == nil && cache != nil {
					// Get or create org data
					orgID := c.config.Organization
					if orgID == "" {
						orgID = c.config.OrgID
					}

					if cache.Orgs == nil {
						cache.Orgs = make(map[string]*OrgData)
					}

					orgData := cache.Orgs[orgID]
					if orgData == nil {
						orgData = &OrgData{}
						cache.Orgs[orgID] = orgData
					}

					// Initialize maps if needed
					if orgData.Configs.AP == nil {
						orgData.Configs.AP = make(map[string]APConfig)
					}
					if orgData.Configs.Switch == nil {
						orgData.Configs.Switch = make(map[string]SwitchConfig)
					}
					if orgData.Configs.Gateway == nil {
						orgData.Configs.Gateway = make(map[string]GatewayConfig)
					}

					// Update the appropriate device CONFIGS based on type
					// The API returns full device configurations which should be stored in the configs section
					for _, device := range devices {
						if device.MAC == nil {
							continue
						}
						normalizedMAC := macaddr.NormalizeOrEmpty(*device.MAC)
						if normalizedMAC == "" {
							continue
						}

						switch deviceType {
						case "ap":
							// Convert UnifiedDevice to APConfig
							apConfig := APConfig{
								BaseDevice: device.BaseDevice,
							}
							// Ensure OrgID is set
							if apConfig.OrgID == nil {
								apConfig.OrgID = &orgID
							}

							// Copy all device-specific configuration from the UnifiedDevice
							if device.DeviceConfig != nil {
								// Store the entire config map as AdditionalConfig
								apConfig.AdditionalConfig = device.DeviceConfig
							}

							// Simply set in map - no need to search
							orgData.Configs.AP[normalizedMAC] = apConfig

						case "switch":
							// Convert UnifiedDevice to SwitchConfig
							switchConfig := SwitchConfig{
								BaseDevice: device.BaseDevice,
							}
							// Ensure OrgID is set
							if switchConfig.OrgID == nil {
								switchConfig.OrgID = &orgID
							}

							// Copy all device-specific configuration from the UnifiedDevice
							if device.DeviceConfig != nil {
								// Store the entire config map as AdditionalConfig
								switchConfig.AdditionalConfig = device.DeviceConfig
							}

							// Simply set in map - no need to search
							orgData.Configs.Switch[normalizedMAC] = switchConfig

						case "gateway":
							// Convert UnifiedDevice to GatewayConfig
							gatewayConfig := GatewayConfig{
								BaseDevice: device.BaseDevice,
							}
							// Ensure OrgID is set
							if gatewayConfig.OrgID == nil {
								gatewayConfig.OrgID = &orgID
							}

							// Copy all device-specific configuration from the UnifiedDevice
							if device.DeviceConfig != nil {
								// Store the entire config map as AdditionalConfig
								gatewayConfig.AdditionalConfig = device.DeviceConfig
							}

							// Simply set in map - no need to search
							orgData.Configs.Gateway[normalizedMAC] = gatewayConfig
						}
					}

					// Save the updated cache to disk
					if err := cacheManager.SaveCache(); err != nil {
						c.logDebug("Failed to update file cache: %v", err)
					} else {
						c.logDebug("Updated file cache configs with %d %s devices for site %s", len(devices), deviceType, siteID)
					}
				}
			}
		}
	}

	return nil
}

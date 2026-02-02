package api

import (
	"context"
	"fmt"
)

// AP operations
// ============================================================================

// AddMockAP adds a mock AP for testing
func (m *MockClient) AddMockAP(ap AP) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ap.Id == nil {
		id := UUID(fmt.Sprintf("mock-ap-%d", len(m.apsByMAC)+1))
		ap.Id = &id
	}

	if ap.Mac != nil {
		mac := *ap.Mac
		m.apsByMAC[mac] = &ap
	}

	if ap.Serial != nil {
		serial := *ap.Serial
		m.apsBySerial[serial] = &ap
	}

	if ap.SiteId != nil {
		siteID := string(*ap.SiteId)
		if _, exists := m.aps[siteID]; !exists {
			m.aps[siteID] = make(map[string]*AP)
		}

		if ap.Id != nil {
			apID := string(*ap.Id)
			m.aps[siteID][apID] = &ap
		}
	}
}

// GetAPBySerialOrMAC retrieves an AP by serial number or MAC address
func (m *MockClient) GetAPBySerialOrMAC(_ context.Context, siteID, serial, mac string) (*AP, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Try by serial first if provided
	if serial != "" {
		if ap, found := m.apsBySerial[serial]; found {
			// Check if the AP is in the correct site
			if ap.SiteId != nil && string(*ap.SiteId) == siteID {
				return ap, nil
			}
		}
	}

	// Try by MAC if provided
	if mac != "" {
		normalizedMAC := mockNormalizeMAC(mac)
		if ap, found := m.apsByMAC[normalizedMAC]; found {
			// Check if the AP is in the correct site
			if ap.SiteId != nil && string(*ap.SiteId) == siteID {
				return ap, nil
			}
		}
	}

	return nil, fmt.Errorf("AP not found with serial '%s' or MAC '%s' in site %s", serial, mac, siteID)
}

// New bidirectional device methods
// ============================================================================

// GetDevices retrieves devices using the new bidirectional pattern
func (m *MockClient) GetDevices(ctx context.Context, siteID string, deviceType string) ([]UnifiedDevice, error) {
	m.logRequest("GET", fmt.Sprintf("/sites/%s/devices?type=%s", siteID, deviceType), nil)

	// Mock API call with rate limiting
	if m.rateLimiter != nil {
		if err := m.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if site exists
	if _, found := m.sites[siteID]; !found {
		return nil, fmt.Errorf("site with ID %s not found", siteID)
	}

	var devicesNew []UnifiedDevice

	// If device type is "ap" or empty, include APs
	if deviceType == "" || deviceType == "ap" {
		// Get APs for the site
		siteAPs, found := m.aps[siteID]
		if found {
			for _, ap := range siteAPs {
				// Convert AP to UnifiedDevice
				deviceNew := UnifiedDevice{
					BaseDevice: BaseDevice{
						AdditionalConfig: make(map[string]interface{}),
					},
					DeviceConfig: make(map[string]interface{}),
				}

				// Copy all fields
				if ap.Id != nil {
					idStr := string(*ap.Id)
					deviceNew.ID = &idStr
				}
				deviceNew.MAC = ap.Mac
				deviceNew.Name = ap.Name
				deviceNew.Serial = ap.Serial
				deviceNew.Model = ap.Model
				deviceNew.Magic = ap.Magic
				deviceType := "ap"
				deviceNew.Type = &deviceType
				if ap.SiteId != nil {
					siteIDStr := string(*ap.SiteId)
					deviceNew.SiteID = &siteIDStr
				}

				// Add AP-specific properties to config
				if ap.Location != nil {
					deviceNew.DeviceConfig["location"] = *ap.Location
				}
				if ap.Orientation != nil {
					deviceNew.DeviceConfig["orientation"] = *ap.Orientation
				}
				if ap.Status != nil {
					deviceNew.DeviceConfig["status"] = *ap.Status
				}
				if ap.Led != nil {
					deviceNew.DeviceConfig["led"] = *ap.Led
				}
				if ap.MapID != nil {
					deviceNew.DeviceConfig["map_id"] = *ap.MapID
				}

				devicesNew = append(devicesNew, deviceNew)
			}
		}
	}

	// Add logic for other device types here if needed
	// For now, the mock client primarily supports APs

	return devicesNew, nil
}

// GetDeviceByMAC retrieves a device by MAC using the new bidirectional pattern
func (m *MockClient) GetDeviceByMAC(ctx context.Context, mac string) (*UnifiedDevice, error) {
	m.logRequest("GET", fmt.Sprintf("/devices/search?mac=%s", mac), nil)

	// Mock API call with rate limiting
	if m.rateLimiter != nil {
		if err := m.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}
	}

	// Normalize MAC address
	mac = mockNormalizeMAC(mac)

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if the MAC exists in the AP registry
	if ap, found := m.apsByMAC[mac]; found {
		// Convert AP to UnifiedDevice
		deviceNew := UnifiedDevice{
			BaseDevice: BaseDevice{
				AdditionalConfig: make(map[string]interface{}),
			},
			DeviceConfig: make(map[string]interface{}),
		}

		// Copy all fields
		if ap.Id != nil {
			idStr := string(*ap.Id)
			deviceNew.ID = &idStr
		}
		deviceNew.MAC = ap.Mac
		deviceNew.Name = ap.Name
		deviceNew.Serial = ap.Serial
		deviceNew.Model = ap.Model
		deviceNew.Magic = ap.Magic
		deviceType := "ap"
		deviceNew.Type = &deviceType
		if ap.SiteId != nil {
			siteIDStr := string(*ap.SiteId)
			deviceNew.SiteID = &siteIDStr
		}

		// Add AP-specific properties to config
		if ap.Location != nil {
			deviceNew.DeviceConfig["location"] = *ap.Location
		}
		if ap.Orientation != nil {
			deviceNew.DeviceConfig["orientation"] = *ap.Orientation
		}
		if ap.Status != nil {
			deviceNew.DeviceConfig["status"] = *ap.Status
		}
		if ap.Led != nil {
			deviceNew.DeviceConfig["led"] = *ap.Led
		}
		if ap.MapID != nil {
			deviceNew.DeviceConfig["map_id"] = *ap.MapID
		}

		return &deviceNew, nil
	}

	// Check in inventory
	if item, found := m.inventoryByMAC[mac]; found {
		// Convert inventory item to UnifiedDevice
		deviceNew := UnifiedDevice{
			BaseDevice: BaseDevice{
				AdditionalConfig: make(map[string]interface{}),
			},
			DeviceConfig: make(map[string]interface{}),
		}

		// Copy all fields
		if item.Id != nil {
			id := string(*item.Id)
			deviceNew.ID = &id
		}
		deviceNew.MAC = item.Mac
		deviceNew.Name = item.Name
		deviceNew.Serial = item.Serial
		deviceNew.Model = item.Model
		deviceNew.Magic = item.Magic
		deviceNew.Type = item.Type
		if item.SiteId != nil {
			siteId := string(*item.SiteId)
			deviceNew.SiteID = &siteId
		}
		if item.OrgId != nil {
			orgId := string(*item.OrgId)
			deviceNew.OrgID = &orgId
		}
		deviceNew.CreatedTime = item.CreatedTime
		deviceNew.ModifiedTime = item.ModifiedTime
		deviceNew.DeviceProfileID = item.DeviceProfileId
		deviceNew.Connected = item.Connected
		deviceNew.Adopted = item.Adopted
		deviceNew.Hostname = item.Hostname
		deviceNew.JSI = item.JSI

		return &deviceNew, nil
	}

	return nil, fmt.Errorf("device with MAC %s not found", mac)
}

// GetDeviceByID retrieves a device by ID using the new bidirectional pattern
func (m *MockClient) GetDeviceByID(ctx context.Context, siteID, deviceID string) (*UnifiedDevice, error) {
	m.logRequest("GET", fmt.Sprintf("/sites/%s/devices/%s", siteID, deviceID), nil)

	// Mock API call with rate limiting
	if m.rateLimiter != nil {
		if err := m.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if site exists
	if _, found := m.sites[siteID]; !found {
		return nil, fmt.Errorf("site with ID %s not found", siteID)
	}

	// Check if the site has any APs
	siteAPs, found := m.aps[siteID]
	if !found {
		return nil, fmt.Errorf("no devices found in site %s", siteID)
	}

	// Search for an AP with the given ID
	for _, ap := range siteAPs {
		if ap.Id != nil && string(*ap.Id) == deviceID {
			// Convert AP to UnifiedDevice
			deviceNew := UnifiedDevice{
				BaseDevice: BaseDevice{
					AdditionalConfig: make(map[string]interface{}),
				},
				DeviceConfig: make(map[string]interface{}),
			}

			// Copy all fields
			if ap.Id != nil {
				idStr := string(*ap.Id)
				deviceNew.ID = &idStr
			}
			deviceNew.MAC = ap.Mac
			deviceNew.Name = ap.Name
			deviceNew.Serial = ap.Serial
			deviceNew.Model = ap.Model
			deviceNew.Magic = ap.Magic
			deviceType := "ap"
			deviceNew.Type = &deviceType
			if ap.SiteId != nil {
				siteIDStr := string(*ap.SiteId)
				deviceNew.SiteID = &siteIDStr
			}

			// Add AP-specific properties to config
			if ap.Location != nil {
				deviceNew.DeviceConfig["location"] = *ap.Location
			}
			if ap.Orientation != nil {
				deviceNew.DeviceConfig["orientation"] = *ap.Orientation
			}
			if ap.Status != nil {
				deviceNew.DeviceConfig["status"] = *ap.Status
			}
			if ap.Led != nil {
				deviceNew.DeviceConfig["led"] = *ap.Led
			}
			if ap.MapID != nil {
				deviceNew.DeviceConfig["map_id"] = *ap.MapID
			}

			return &deviceNew, nil
		}
	}

	return nil, fmt.Errorf("device with ID %s not found in site %s", deviceID, siteID)
}

// GetDeviceByName retrieves a device by name using the new bidirectional pattern
func (m *MockClient) GetDeviceByName(ctx context.Context, siteID, name string) (*UnifiedDevice, error) {
	m.logRequest("GET", fmt.Sprintf("/sites/%s/devices?name=%s", siteID, name), nil)

	// Mock API call with rate limiting
	if m.rateLimiter != nil {
		if err := m.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if site exists
	if _, found := m.sites[siteID]; !found {
		return nil, fmt.Errorf("site with ID %s not found", siteID)
	}

	// Check if the site has any APs
	siteAPs, found := m.aps[siteID]
	if !found {
		return nil, fmt.Errorf("no devices found in site %s", siteID)
	}

	// Search for an AP with the given name
	for _, ap := range siteAPs {
		if ap.Name != nil && *ap.Name == name {
			// Convert AP to UnifiedDevice
			deviceNew := UnifiedDevice{
				BaseDevice: BaseDevice{
					AdditionalConfig: make(map[string]interface{}),
				},
				DeviceConfig: make(map[string]interface{}),
			}

			// Copy all fields
			if ap.Id != nil {
				idStr := string(*ap.Id)
				deviceNew.ID = &idStr
			}
			deviceNew.MAC = ap.Mac
			deviceNew.Name = ap.Name
			deviceNew.Serial = ap.Serial
			deviceNew.Model = ap.Model
			deviceNew.Magic = ap.Magic
			deviceType := "ap"
			deviceNew.Type = &deviceType
			if ap.SiteId != nil {
				siteIDStr := string(*ap.SiteId)
				deviceNew.SiteID = &siteIDStr
			}

			// Add AP-specific properties to config
			if ap.Location != nil {
				deviceNew.DeviceConfig["location"] = *ap.Location
			}
			if ap.Orientation != nil {
				deviceNew.DeviceConfig["orientation"] = *ap.Orientation
			}
			if ap.Status != nil {
				deviceNew.DeviceConfig["status"] = *ap.Status
			}
			if ap.Led != nil {
				deviceNew.DeviceConfig["led"] = *ap.Led
			}
			if ap.MapID != nil {
				deviceNew.DeviceConfig["map_id"] = *ap.MapID
			}

			return &deviceNew, nil
		}
	}

	return nil, fmt.Errorf("device with name %s not found in site %s", name, siteID)
}

// GetDevicesByType retrieves devices by type using the new bidirectional pattern
func (m *MockClient) GetDevicesByType(ctx context.Context, siteID string, deviceType string) ([]UnifiedDevice, error) {
	// This is just an alias for GetDevices
	return m.GetDevices(ctx, siteID, deviceType)
}

// UpdateDeviceNew updates a device using the new bidirectional pattern
func (m *MockClient) UpdateDeviceNew(ctx context.Context, siteID string, deviceID string, device *UnifiedDevice) (*UnifiedDevice, error) {
	m.logRequest("PUT", fmt.Sprintf("/sites/%s/devices/%s", siteID, deviceID), device)

	// Mock API call with rate limiting
	if m.rateLimiter != nil {
		if err := m.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}
	}

	// Convert to old UnifiedDevice format for existing mock logic
	oldDevice := UnifiedDevice{
		BaseDevice: BaseDevice{
			ID:     device.ID,
			MAC:    device.MAC,
			Name:   device.Name,
			Serial: device.Serial,
			Model:  device.Model,
			Magic:  device.Magic,
			SiteID: device.SiteID,
			OrgID:  device.OrgID,
		},
		DeviceConfig: device.DeviceConfig,
	}

	if device.Type != nil {
		oldDevice.Type = device.Type
		oldDevice.DeviceType = *device.Type
	}

	// In a mock client, simply return the provided device without changes
	return device, nil
}

// UpdateDevice delegates to UpdateDeviceNew for interface compatibility
func (m *MockClient) UpdateDevice(ctx context.Context, siteID string, deviceID string, device *UnifiedDevice) (*UnifiedDevice, error) {
	return m.UpdateDeviceNew(ctx, siteID, deviceID, device)
}

// AssignDevice assigns a device using the new bidirectional pattern
func (m *MockClient) AssignDevice(ctx context.Context, orgID string, siteID string, mac string) (*UnifiedDevice, error) {
	m.logRequest("PUT", fmt.Sprintf("/orgs/%s/inventory/%s/assign", orgID, mac), map[string]interface{}{
		"site_id": siteID,
	})

	// Mock API call with rate limiting
	if m.rateLimiter != nil {
		if err := m.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}
	}

	// Mock implementation for device assignment
	m.logRequest("POST", fmt.Sprintf("/orgs/%s/inventory/assign", m.config.Organization), map[string]interface{}{
		"site_id": siteID,
		"macs":    []string{mac},
	})

	// Normalize MAC address
	mac = mockNormalizeMAC(mac)

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if the MAC exists in the inventory
	item, found := m.inventoryByMAC[mac]
	if !found {
		// Auto-create a sample inventory item for testing
		id := fmt.Sprintf("inv-%s", mac)
		deviceType := "ap" // Assume AP for mock
		idUUID := UUID(id)
		item = InventoryItem{
			Id:     &idUUID,
			Mac:    &mac,
			Type:   &deviceType,
			Magic:  StringPtr(fmt.Sprintf("magic-%s", mac)),
			Serial: StringPtr(fmt.Sprintf("SN-%s", mac)),
			Model:  StringPtr("Mock-Model"),
		}
		m.inventory = append(m.inventory, item)
		m.inventoryByMAC[mac] = item
	}

	// Update the site ID
	siteUUID := UUID(siteID)
	item.SiteId = &siteUUID
	m.inventoryByMAC[mac] = item

	// Get the assigned device using new method
	assignedDevice, err := m.GetDeviceByMAC(ctx, mac)
	if err != nil {
		return nil, err
	}

	return assignedDevice, nil
}

// UnassignDevice unassigns a device using the new bidirectional pattern
func (m *MockClient) UnassignDevice(ctx context.Context, orgID string, _ string, deviceID string) error {
	m.logRequest("POST", fmt.Sprintf("/orgs/%s/inventory/unassign", orgID), map[string]interface{}{
		"device_id": deviceID,
	})

	// Mock API call with rate limiting
	if m.rateLimiter != nil {
		if err := m.rateLimiter.Wait(ctx); err != nil {
			return err
		}
	}

	// Mock implementation - just return success for device unassignment
	return nil
}

// AssignDevicesToSite assigns multiple devices using the new bidirectional pattern
func (m *MockClient) AssignDevicesToSite(ctx context.Context, orgID string, siteID string, macs []string, noReassign bool) error {
	m.logRequest("POST", fmt.Sprintf("/orgs/%s/inventory/assign", orgID), map[string]interface{}{
		"site_id":     siteID,
		"macs":        macs,
		"no_reassign": noReassign,
	})

	// Mock API call with rate limiting
	if m.rateLimiter != nil {
		if err := m.rateLimiter.Wait(ctx); err != nil {
			return err
		}
	}

	// Mock implementation: just return success
	// In a real implementation, this would assign devices to the specified site
	return nil
}

// UnassignDevicesFromSite unassigns multiple devices using the new bidirectional pattern
func (m *MockClient) UnassignDevicesFromSite(ctx context.Context, orgID string, macs []string) error {
	m.logRequest("POST", fmt.Sprintf("/orgs/%s/inventory/unassign", orgID), map[string]interface{}{
		"macs": macs,
	})

	// Mock API call with rate limiting
	if m.rateLimiter != nil {
		if err := m.rateLimiter.Wait(ctx); err != nil {
			return err
		}
	}

	// Mock implementation: just return success
	// In a real implementation, this would unassign devices from any site
	return nil
}

// QueryDeviceExtensive performs a detailed query on a device
func (m *MockClient) QueryDeviceExtensive(_ context.Context, _, _ string) error {
	// Mock implementation - just return success
	return nil
}

// GetRawDeviceJSON retrieves the raw JSON for a device
func (m *MockClient) GetRawDeviceJSON(_ context.Context, siteID, deviceID string) (string, error) {
	// Mock implementation - return basic JSON
	return fmt.Sprintf(`{"id": "%s", "site_id": "%s", "mock": true}`, deviceID, siteID), nil
}

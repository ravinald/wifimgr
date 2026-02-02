package api

import (
	"context"
	"fmt"

	"github.com/ravinald/wifimgr/internal/macaddr"
)

// Inventory Operations
// ============================================================================

// Helper function for MAC address normalization in the mock client
func mockNormalizeMAC(mac string) string {
	// Use the standard macaddr.Normalize function
	normalized, err := macaddr.Normalize(mac)
	if err != nil {
		// If normalization fails, return original string
		// This is not ideal but maintains backward compatibility
		return mac
	}
	return normalized
}

// AddInventoryItem adds an inventory item to the mock client's inventory
func (m *MockClient) AddInventoryItem(item InventoryItem) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if item.Id == nil {
		id := fmt.Sprintf("mock-inv-%d", len(m.inventory)+1)
		idUUID := UUID(id)
		item.Id = &idUUID
	}

	// Add to inventory list
	m.inventory = append(m.inventory, item)

	// Add to MAC lookup map
	if item.Mac != nil {
		mac := mockNormalizeMAC(*item.Mac)
		m.inventoryByMAC[mac] = item
	}
}

// DeleteDevicesFromSite deletes devices from a site
func (m *MockClient) DeleteDevicesFromSite(_ context.Context, _ []string) error {
	// Mock implementation - just return success
	return nil
}

// New bidirectional inventory methods
// ============================================================================

// GetInventory retrieves inventory items using the new bidirectional pattern
func (m *MockClient) GetInventory(ctx context.Context, orgID string, deviceType string) ([]*MistInventoryItem, error) {
	m.logRequest("GET", fmt.Sprintf("/api/v1/orgs/%s/inventory?type=%s", orgID, deviceType), nil)

	// Mock API call with rate limiting
	if m.rateLimiter != nil {
		if err := m.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Filter inventory based on device type
	var filteredItems []*MistInventoryItem
	for _, item := range m.inventory {
		if deviceType == "all" || deviceType == "" || (item.Type != nil && *item.Type == deviceType) {
			// Convert InventoryItem to MistInventoryItem
			itemNew := &MistInventoryItem{
				AdditionalConfig: make(map[string]interface{}),
			}

			// Copy all fields
			if item.Id != nil {
				id := string(*item.Id)
				itemNew.ID = &id
			}
			itemNew.MAC = item.Mac
			itemNew.Serial = item.Serial
			itemNew.Name = item.Name
			itemNew.Model = item.Model
			itemNew.Type = item.Type
			itemNew.Magic = item.Magic
			itemNew.HwRev = item.HwRev
			itemNew.SKU = item.SKU
			if item.SiteId != nil {
				siteId := string(*item.SiteId)
				itemNew.SiteID = &siteId
			}
			if item.OrgId != nil {
				orgId := string(*item.OrgId)
				itemNew.OrgID = &orgId
			}
			itemNew.CreatedTime = item.CreatedTime
			itemNew.ModifiedTime = item.ModifiedTime
			itemNew.DeviceProfileID = item.DeviceProfileId
			itemNew.Connected = item.Connected
			itemNew.Adopted = item.Adopted
			itemNew.Hostname = item.Hostname
			itemNew.JSI = item.JSI
			itemNew.ChassisMac = item.ChassisMac
			itemNew.ChassisSerial = item.ChassisSerial
			itemNew.VcMac = item.VcMac

			filteredItems = append(filteredItems, itemNew)
		}
	}

	return filteredItems, nil
}

// GetInventoryItem retrieves a specific inventory item by ID
func (m *MockClient) GetInventoryItem(ctx context.Context, orgID string, itemID string) (*MistInventoryItem, error) {
	m.logRequest("GET", fmt.Sprintf("/api/v1/orgs/%s/inventory/%s", orgID, itemID), nil)

	// Mock API call with rate limiting
	if m.rateLimiter != nil {
		if err := m.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Find item by ID
	for _, item := range m.inventory {
		if item.Id != nil && string(*item.Id) == itemID {
			// Convert to MistInventoryItem
			itemNew := &MistInventoryItem{
				AdditionalConfig: make(map[string]interface{}),
			}

			// Copy all fields
			if item.Id != nil {
				id := string(*item.Id)
				itemNew.ID = &id
			}
			itemNew.MAC = item.Mac
			itemNew.Serial = item.Serial
			itemNew.Name = item.Name
			itemNew.Model = item.Model
			itemNew.Type = item.Type
			itemNew.Magic = item.Magic
			itemNew.HwRev = item.HwRev
			itemNew.SKU = item.SKU
			if item.SiteId != nil {
				siteId := string(*item.SiteId)
				itemNew.SiteID = &siteId
			}
			if item.OrgId != nil {
				orgId := string(*item.OrgId)
				itemNew.OrgID = &orgId
			}
			itemNew.CreatedTime = item.CreatedTime
			itemNew.ModifiedTime = item.ModifiedTime
			itemNew.DeviceProfileID = item.DeviceProfileId
			itemNew.Connected = item.Connected
			itemNew.Adopted = item.Adopted
			itemNew.Hostname = item.Hostname
			itemNew.JSI = item.JSI

			return itemNew, nil
		}
	}

	return nil, fmt.Errorf("inventory item with ID %s not found", itemID)
}

// GetInventoryItemByMAC retrieves an inventory item by MAC address
func (m *MockClient) GetInventoryItemByMAC(ctx context.Context, orgID string, macAddress string) (*MistInventoryItem, error) {
	m.logRequest("GET", fmt.Sprintf("/api/v1/orgs/%s/inventory/search?mac=%s", orgID, macAddress), nil)

	// Mock API call with rate limiting
	if m.rateLimiter != nil {
		if err := m.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	normalizedMAC := mockNormalizeMAC(macAddress)
	item, found := m.inventoryByMAC[normalizedMAC]
	if !found {
		return nil, fmt.Errorf("inventory item with MAC '%s' not found", macAddress)
	}

	// Convert to MistInventoryItem
	itemNew := &MistInventoryItem{
		AdditionalConfig: make(map[string]interface{}),
	}

	// Copy all fields
	if item.Id != nil {
		id := string(*item.Id)
		itemNew.ID = &id
	}
	itemNew.MAC = item.Mac
	itemNew.Serial = item.Serial
	itemNew.Name = item.Name
	itemNew.Model = item.Model
	itemNew.Type = item.Type
	itemNew.Magic = item.Magic
	itemNew.HwRev = item.HwRev
	itemNew.SKU = item.SKU
	if item.SiteId != nil {
		siteId := string(*item.SiteId)
		itemNew.SiteID = &siteId
	}
	if item.OrgId != nil {
		orgId := string(*item.OrgId)
		itemNew.OrgID = &orgId
	}
	itemNew.CreatedTime = item.CreatedTime
	itemNew.ModifiedTime = item.ModifiedTime
	itemNew.DeviceProfileID = item.DeviceProfileId
	itemNew.Connected = item.Connected
	itemNew.Adopted = item.Adopted
	itemNew.Hostname = item.Hostname
	itemNew.JSI = item.JSI

	return itemNew, nil
}

// UpdateInventoryItem updates an existing inventory item
func (m *MockClient) UpdateInventoryItem(ctx context.Context, orgID string, itemID string, item *MistInventoryItem) (*MistInventoryItem, error) {
	m.logRequest("PUT", fmt.Sprintf("/api/v1/orgs/%s/inventory/%s", orgID, itemID), item)

	// Mock API call with rate limiting
	if m.rateLimiter != nil {
		if err := m.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Find the existing item by ID
	for i, existingItem := range m.inventory {
		if existingItem.Id != nil && string(*existingItem.Id) == itemID {
			// Update the existing item with new data
			m.inventory[i].Name = item.Name
			m.inventory[i].Model = item.Model
			m.inventory[i].Type = item.Type
			m.inventory[i].Magic = item.Magic
			m.inventory[i].HwRev = item.HwRev
			m.inventory[i].SKU = item.SKU
			if item.SiteID != nil {
				siteUUID := UUID(*item.SiteID)
				m.inventory[i].SiteId = &siteUUID
			}
			m.inventory[i].DeviceProfileId = item.DeviceProfileID
			m.inventory[i].Hostname = item.Hostname

			// Update the MAC index if MAC changed
			if item.MAC != nil {
				normalizedMAC := mockNormalizeMAC(*item.MAC)
				m.inventoryByMAC[normalizedMAC] = m.inventory[i]
			}

			// Return the updated item as MistInventoryItem
			updatedItem := *item // Copy the input
			updatedItem.ID = &itemID
			return &updatedItem, nil
		}
	}

	return nil, fmt.Errorf("inventory item with ID %s not found", itemID)
}

// ClaimInventoryItem claims inventory items using claim codes
func (m *MockClient) ClaimInventoryItem(ctx context.Context, orgID string, claimCodes []string) ([]*MistInventoryItem, error) {
	m.logRequest("POST", fmt.Sprintf("/api/v1/orgs/%s/inventory", orgID), map[string]interface{}{
		"op":   "assign",
		"macs": claimCodes,
	})

	// Mock API call with rate limiting
	if m.rateLimiter != nil {
		if err := m.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}
	}

	// Return mock claimed items
	var claimedItems []*MistInventoryItem
	for i, code := range claimCodes {
		item := &MistInventoryItem{
			Magic:            &code,
			OrgID:            &orgID,
			AdditionalConfig: make(map[string]interface{}),
		}

		// Generate mock data
		mockID := fmt.Sprintf("claimed-item-%d", i+1)
		mockMAC := fmt.Sprintf("aa:bb:cc:dd:ee:%02x", i+1)
		mockSerial := fmt.Sprintf("MOCK-SERIAL-%d", i+1)
		mockModel := "MockDevice"
		mockType := "ap"

		item.ID = &mockID
		item.MAC = &mockMAC
		item.Serial = &mockSerial
		item.Model = &mockModel
		item.Type = &mockType

		claimedItems = append(claimedItems, item)
	}

	return claimedItems, nil
}

// ReleaseInventoryItem releases inventory items from the organization
func (m *MockClient) ReleaseInventoryItem(ctx context.Context, orgID string, itemIDs []string) error {
	m.logRequest("POST", fmt.Sprintf("/api/v1/orgs/%s/inventory", orgID), map[string]interface{}{
		"op":   "unassign",
		"macs": itemIDs,
	})

	// Mock API call with rate limiting
	if m.rateLimiter != nil {
		if err := m.rateLimiter.Wait(ctx); err != nil {
			return err
		}
	}

	// Mock implementation - just return success
	return nil
}

// AssignInventoryItemsToSite assigns inventory items to a site
func (m *MockClient) AssignInventoryItemsToSite(ctx context.Context, orgID string, siteID string, itemMACs []string) error {
	m.logRequest("POST", fmt.Sprintf("/api/v1/orgs/%s/inventory", orgID), map[string]interface{}{
		"op":      "assign",
		"site_id": siteID,
		"macs":    itemMACs,
	})

	// Mock API call with rate limiting
	if m.rateLimiter != nil {
		if err := m.rateLimiter.Wait(ctx); err != nil {
			return err
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Update site assignment for the items
	for _, mac := range itemMACs {
		normalizedMAC := mockNormalizeMAC(mac)
		if item, found := m.inventoryByMAC[normalizedMAC]; found {
			siteUUID := UUID(siteID)
			item.SiteId = &siteUUID
			m.inventoryByMAC[normalizedMAC] = item
		}
	}

	return nil
}

// UnassignInventoryItemsFromMistSite unassigns inventory items from their current site
func (m *MockClient) UnassignInventoryItemsFromMistSite(ctx context.Context, orgID string, itemMACs []string) error {
	m.logRequest("POST", fmt.Sprintf("/api/v1/orgs/%s/inventory", orgID), map[string]interface{}{
		"op":   "unassign",
		"macs": itemMACs,
	})

	// Mock API call with rate limiting
	if m.rateLimiter != nil {
		if err := m.rateLimiter.Wait(ctx); err != nil {
			return err
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Remove site assignment for the items
	for _, mac := range itemMACs {
		normalizedMAC := mockNormalizeMAC(mac)
		if item, found := m.inventoryByMAC[normalizedMAC]; found {
			item.SiteId = nil
			m.inventoryByMAC[normalizedMAC] = item
		}
	}

	return nil
}

// UnassignInventoryItemsFromSite delegates to UnassignInventoryItemsFromMistSite for interface compatibility
func (m *MockClient) UnassignInventoryItemsFromSite(ctx context.Context, orgID string, itemMACs []string) error {
	return m.UnassignInventoryItemsFromMistSite(ctx, orgID, itemMACs)
}

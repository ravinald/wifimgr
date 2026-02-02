package mist

import (
	"context"
	"fmt"

	"github.com/ravinald/wifimgr/api"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// inventoryService implements vendors.InventoryService for Mist.
type inventoryService struct {
	client api.Client
	orgID  string
}

// List returns inventory items, optionally filtered by device type.
func (s *inventoryService) List(ctx context.Context, deviceType string) ([]*vendors.InventoryItem, error) {
	items, err := s.client.GetInventory(ctx, s.orgID, deviceType)
	if err != nil {
		return nil, fmt.Errorf("failed to get inventory: %w", err)
	}

	result := make([]*vendors.InventoryItem, 0, len(items))
	for _, item := range items {
		inv := convertInventoryItemToVendor(item)
		if inv != nil {
			result = append(result, inv)
		}
	}

	return result, nil
}

// ByMAC finds an inventory item by MAC address.
func (s *inventoryService) ByMAC(ctx context.Context, mac string) (*vendors.InventoryItem, error) {
	item, err := s.client.GetInventoryItemByMAC(ctx, s.orgID, mac)
	if err != nil {
		return nil, fmt.Errorf("failed to get inventory item by MAC %q: %w", mac, err)
	}

	return convertInventoryItemToVendor(item), nil
}

// BySerial finds an inventory item by serial number.
func (s *inventoryService) BySerial(ctx context.Context, serial string) (*vendors.InventoryItem, error) {
	// Mist API uses ID for serial lookup - need to search inventory
	// Try all device types
	for _, deviceType := range []string{"ap", "switch", "gateway"} {
		items, err := s.client.GetInventory(ctx, s.orgID, deviceType)
		if err != nil {
			continue
		}

		for _, item := range items {
			if item.Serial != nil && *item.Serial == serial {
				return convertInventoryItemToVendor(item), nil
			}
		}
	}

	return nil, fmt.Errorf("inventory item with serial %q not found", serial)
}

// Claim adds devices to the organization's inventory using claim codes.
func (s *inventoryService) Claim(ctx context.Context, claimCodes []string) ([]*vendors.InventoryItem, error) {
	items, err := s.client.ClaimInventoryItem(ctx, s.orgID, claimCodes)
	if err != nil {
		return nil, fmt.Errorf("failed to claim devices: %w", err)
	}

	result := make([]*vendors.InventoryItem, 0, len(items))
	for _, item := range items {
		inv := convertInventoryItemToVendor(item)
		if inv != nil {
			result = append(result, inv)
		}
	}

	return result, nil
}

// Release removes devices from the organization's inventory.
func (s *inventoryService) Release(ctx context.Context, serials []string) error {
	// Mist API expects item IDs, not serials - need to resolve
	var itemIDs []string
	for _, serial := range serials {
		item, err := s.BySerial(ctx, serial)
		if err != nil {
			return fmt.Errorf("failed to find device with serial %q: %w", serial, err)
		}
		// Get the original inventory item to find the ID
		mistItem, err := s.client.GetInventoryItemByMAC(ctx, s.orgID, item.MAC)
		if err != nil {
			return fmt.Errorf("failed to get inventory item: %w", err)
		}
		if mistItem.ID != nil {
			itemIDs = append(itemIDs, *mistItem.ID)
		}
	}

	if err := s.client.ReleaseInventoryItem(ctx, s.orgID, itemIDs); err != nil {
		return fmt.Errorf("failed to release devices: %w", err)
	}

	return nil
}

// AssignToSite assigns devices to a site.
func (s *inventoryService) AssignToSite(ctx context.Context, siteID string, macs []string) error {
	if err := s.client.AssignInventoryItemsToSite(ctx, s.orgID, siteID, macs); err != nil {
		return fmt.Errorf("failed to assign devices to site: %w", err)
	}
	return nil
}

// UnassignFromSite removes devices from their assigned site.
func (s *inventoryService) UnassignFromSite(ctx context.Context, macs []string) error {
	if err := s.client.UnassignInventoryItemsFromSite(ctx, s.orgID, macs); err != nil {
		return fmt.Errorf("failed to unassign devices from site: %w", err)
	}
	return nil
}

// Ensure inventoryService implements vendors.InventoryService at compile time.
var _ vendors.InventoryService = (*inventoryService)(nil)

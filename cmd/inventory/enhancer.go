package inventory

import (
	"github.com/ravinald/wifimgr/internal/vendors"
)

// EnhanceInventoryItems ensures all required fields for display are populated
// With vendors.InventoryItem using plain strings, this is simpler than the legacy version
func EnhanceInventoryItems(items []*vendors.InventoryItem) []*vendors.InventoryItem {
	return EnhanceInventoryItemsWithClient(items, nil)
}

// EnhanceInventoryItemsWithClient ensures all required fields for display are populated
// This version accepts a client parameter for dependency injection (currently unused)
func EnhanceInventoryItemsWithClient(items []*vendors.InventoryItem, _ interface{}) []*vendors.InventoryItem {
	result := make([]*vendors.InventoryItem, 0, len(items))

	for _, item := range items {
		if item == nil {
			continue
		}
		result = append(result, item)
	}

	return result
}

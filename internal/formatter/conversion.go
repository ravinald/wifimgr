package formatter

import (
	"github.com/ravinald/wifimgr/api"
)

// ConvertToInterfaces converts a slice of any type to a slice of interfaces
func ConvertToInterfaces[T any](items []T) []interface{} {
	result := make([]interface{}, len(items))
	for i, item := range items {
		result[i] = item
	}
	return result
}

// ConvertSitesToInterfaces converts a slice of Site structs to a slice of interfaces
func ConvertSitesToInterfaces(sites []api.Site) []interface{} {
	return ConvertToInterfaces(sites)
}

// ConvertAPsToInterfaces converts a slice of AP structs to a slice of interfaces
func ConvertAPsToInterfaces(aps []api.AP) []interface{} {
	return ConvertToInterfaces(aps)
}

// ConvertInventoryToInterfaces converts a slice of inventory items to a slice of interfaces
func ConvertInventoryToInterfaces(items []api.InventoryItem) []interface{} {
	return ConvertToInterfaces(items)
}

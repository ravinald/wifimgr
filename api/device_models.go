package api

import (
	"strconv"
)

// Helper function to get a string pointer
func StringPtr(s string) *string {
	return &s
}

// Helper function to get a UUID pointer
func UUIDPtr(s string) *UUID {
	uuid := UUID(s)
	return &uuid
}

// Helper function to get a bool pointer
func BoolPtr(b bool) *bool {
	return &b
}

// Helper function to get an int pointer
func IntPtr(i int) *int {
	return &i
}

// Helper function to get a float64 pointer
func Float64Ptr(f float64) *float64 {
	return &f
}

// Helper function to get an int64 pointer
func Int64Ptr(i int64) *int64 {
	return &i
}

// Helper function to convert int to string
func StringFromInt(i int) string {
	return strconv.Itoa(i)
}

// MergeDeviceData merges data from inventory and site-specific API responses
func MergeDeviceData(base UnifiedDevice, update UnifiedDevice) UnifiedDevice {
	result := base

	// Only update fields from the update source if they are not nil
	if update.ID != nil {
		result.ID = update.ID
	}

	if update.MAC != nil {
		result.MAC = update.MAC
	}

	if update.Serial != nil {
		result.Serial = update.Serial
	}

	if update.Name != nil && *update.Name != "" {
		result.Name = update.Name
	}

	if update.Model != nil {
		result.Model = update.Model
	}

	if update.Type != nil {
		result.Type = update.Type
		result.DeviceType = *update.Type
	}

	// Critical: Preserve Magic field from either source
	if result.Magic == nil && update.Magic != nil {
		result.Magic = update.Magic
	}

	if update.HwRev != nil {
		result.HwRev = update.HwRev
	}

	if update.SKU != nil {
		result.SKU = update.SKU
	}

	if update.SiteID != nil {
		result.SiteID = update.SiteID
	}

	if update.OrgID != nil {
		result.OrgID = update.OrgID
	}

	if update.CreatedTime != nil {
		result.CreatedTime = update.CreatedTime
	}

	if update.ModifiedTime != nil {
		result.ModifiedTime = update.ModifiedTime
	}

	if update.DeviceProfileID != nil {
		result.DeviceProfileID = update.DeviceProfileID
	}

	if update.Connected != nil {
		result.Connected = update.Connected
	}

	if update.Adopted != nil {
		result.Adopted = update.Adopted
	}

	if update.Hostname != nil {
		result.Hostname = update.Hostname
	}

	if update.Notes != nil {
		result.Notes = update.Notes
	}

	if update.JSI != nil {
		result.JSI = update.JSI
	}

	if update.Tags != nil {
		result.Tags = update.Tags
	}

	// Merge DeviceConfig maps
	if result.DeviceConfig == nil {
		result.DeviceConfig = make(map[string]interface{})
	}

	for k, v := range update.DeviceConfig {
		result.DeviceConfig[k] = v
	}

	// Merge AdditionalConfig maps
	if result.AdditionalConfig == nil {
		result.AdditionalConfig = make(map[string]interface{})
	}

	for k, v := range update.AdditionalConfig {
		result.AdditionalConfig[k] = v
	}

	return result
}

// ConvertFromRawMap converts a raw map to a UnifiedDevice
func ConvertFromRawMap(rawData map[string]interface{}) (*UnifiedDevice, error) {
	// Use the existing NewUnifiedDeviceFromMap function which already handles this properly
	return NewUnifiedDeviceFromMap(rawData)
}

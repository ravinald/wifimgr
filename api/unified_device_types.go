package api

import (
	"fmt"
)

// DeviceMarshaler defines the common interface for bidirectional device data transformation.
// It provides type-safe access to core device fields and marshaling operations.
type DeviceMarshaler interface {
	GetID() *string
	GetMAC() *string
	GetSerial() *string
	GetName() *string
	GetModel() *string
	GetType() *string
	GetMagic() *string
	GetSiteID() *string
	GetOrgID() *string
	GetDeviceProfileID() *string
	ToMap() map[string]interface{}
	FromMap(data map[string]interface{}) error
	ToConfigMap() map[string]interface{}
	FromConfigMap(data map[string]interface{}) error
}

// BaseDevice represents the core device fields with bidirectional data handling
type BaseDevice struct {
	// Core identification
	ID     *string `json:"id,omitempty"`
	MAC    *string `json:"mac,omitempty"`
	Serial *string `json:"serial,omitempty"`
	Name   *string `json:"name,omitempty"`
	Model  *string `json:"model,omitempty"`
	Type   *string `json:"type,omitempty"`
	Magic  *string `json:"magic,omitempty"`
	HwRev  *string `json:"hw_rev,omitempty"`
	SKU    *string `json:"sku,omitempty"`

	// Metadata
	SiteID          *string `json:"site_id,omitempty"`
	OrgID           *string `json:"org_id,omitempty"`
	CreatedTime     *int64  `json:"created_time,omitempty"`
	ModifiedTime    *int64  `json:"modified_time,omitempty"`
	DeviceProfileID *string `json:"deviceprofile_id,omitempty"`

	// Status
	Connected *bool   `json:"connected,omitempty"`
	Adopted   *bool   `json:"adopted,omitempty"`
	Hostname  *string `json:"hostname,omitempty"`
	Notes     *string `json:"notes,omitempty"`
	JSI       *bool   `json:"jsi,omitempty"`

	// Basic config
	Tags *[]string `json:"tags,omitempty"`

	// Additional configuration for any unmapped fields
	AdditionalConfig map[string]interface{} `json:"-"`
}

// UnifiedDevice is a universal container that can represent any device type with complete data preservation
type UnifiedDevice struct {
	BaseDevice

	// Device-specific configuration stored as maps for flexibility
	// This preserves all device-specific fields while maintaining type safety for core fields
	DeviceConfig map[string]interface{} `json:"device_config,omitempty"`

	// Internal: not serialized, used for type identification and routing
	DeviceType string `json:"-"`
}

// Implement DeviceMarshaler for BaseDevice
func (bd *BaseDevice) GetID() *string              { return bd.ID }
func (bd *BaseDevice) GetMAC() *string             { return bd.MAC }
func (bd *BaseDevice) GetSerial() *string          { return bd.Serial }
func (bd *BaseDevice) GetName() *string            { return bd.Name }
func (bd *BaseDevice) GetModel() *string           { return bd.Model }
func (bd *BaseDevice) GetType() *string            { return bd.Type }
func (bd *BaseDevice) GetMagic() *string           { return bd.Magic }
func (bd *BaseDevice) GetSiteID() *string          { return bd.SiteID }
func (bd *BaseDevice) GetOrgID() *string           { return bd.OrgID }
func (bd *BaseDevice) GetDeviceProfileID() *string { return bd.DeviceProfileID }

// FromMap populates the BaseDevice from API response data
func (bd *BaseDevice) FromMap(data map[string]interface{}) error {
	// Parse core fields with type safety
	if id, ok := data["id"].(string); ok {
		bd.ID = &id
	}
	if mac, ok := data["mac"].(string); ok {
		bd.MAC = &mac
	}
	if serial, ok := data["serial"].(string); ok {
		bd.Serial = &serial
	}
	if name, ok := data["name"].(string); ok {
		bd.Name = &name
	}
	if model, ok := data["model"].(string); ok {
		bd.Model = &model
	}
	if deviceType, ok := data["type"].(string); ok {
		bd.Type = &deviceType
	}
	if magic, ok := data["magic"].(string); ok {
		bd.Magic = &magic
	}
	if hwRev, ok := data["hw_rev"].(string); ok {
		bd.HwRev = &hwRev
	}
	if sku, ok := data["sku"].(string); ok {
		bd.SKU = &sku
	}

	// Parse metadata fields
	if siteID, ok := data["site_id"].(string); ok {
		bd.SiteID = &siteID
	}
	if orgID, ok := data["org_id"].(string); ok {
		bd.OrgID = &orgID
	}
	if createdTime, ok := data["created_time"].(float64); ok {
		ct := int64(createdTime)
		bd.CreatedTime = &ct
	}
	if modifiedTime, ok := data["modified_time"].(float64); ok {
		mt := int64(modifiedTime)
		bd.ModifiedTime = &mt
	}
	if deviceProfileID, ok := data["deviceprofile_id"].(string); ok {
		bd.DeviceProfileID = &deviceProfileID
	}

	// Parse status fields
	if connected, ok := data["connected"].(bool); ok {
		bd.Connected = &connected
	}
	if adopted, ok := data["adopted"].(bool); ok {
		bd.Adopted = &adopted
	}
	if hostname, ok := data["hostname"].(string); ok {
		bd.Hostname = &hostname
	}
	if notes, ok := data["notes"].(string); ok {
		bd.Notes = &notes
	}
	if jsi, ok := data["jsi"].(bool); ok {
		bd.JSI = &jsi
	}

	// Parse tags
	if tags, ok := data["tags"].([]interface{}); ok {
		tagStrings := make([]string, 0, len(tags))
		for _, tag := range tags {
			if tagStr, tagOk := tag.(string); tagOk {
				tagStrings = append(tagStrings, tagStr)
			}
		}
		if len(tagStrings) > 0 {
			bd.Tags = &tagStrings
		}
	}

	// Initialize additional config if nil
	if bd.AdditionalConfig == nil {
		bd.AdditionalConfig = make(map[string]interface{})
	}

	// Store any unknown fields in AdditionalConfig (following APDevice pattern)
	knownFields := map[string]bool{
		// Core identification
		"id": true, "mac": true, "serial": true, "name": true, "model": true,
		"type": true, "magic": true, "hw_rev": true, "sku": true,
		// Metadata
		"site_id": true, "org_id": true, "created_time": true, "modified_time": true,
		"deviceprofile_id": true,
		// Status
		"connected": true, "adopted": true, "hostname": true, "notes": true, "jsi": true,
		// Basic config
		"tags": true,
	}

	for k, v := range data {
		if !knownFields[k] {
			bd.AdditionalConfig[k] = v
		}
	}

	return nil
}

// ToMap converts the BaseDevice to a map for API operations
func (bd *BaseDevice) ToMap() map[string]interface{} {
	result := make(map[string]interface{})

	// Add core fields
	if bd.ID != nil {
		result["id"] = *bd.ID
	}
	if bd.MAC != nil {
		result["mac"] = *bd.MAC
	}
	if bd.Serial != nil {
		result["serial"] = *bd.Serial
	}
	if bd.Name != nil {
		result["name"] = *bd.Name
	}
	if bd.Model != nil {
		result["model"] = *bd.Model
	}
	if bd.Type != nil {
		result["type"] = *bd.Type
	}
	if bd.Magic != nil {
		result["magic"] = *bd.Magic
	}
	if bd.HwRev != nil {
		result["hw_rev"] = *bd.HwRev
	}
	if bd.SKU != nil {
		result["sku"] = *bd.SKU
	}

	// Add metadata fields
	if bd.SiteID != nil {
		result["site_id"] = *bd.SiteID
	}
	if bd.OrgID != nil {
		result["org_id"] = *bd.OrgID
	}
	if bd.CreatedTime != nil {
		result["created_time"] = *bd.CreatedTime
	}
	if bd.ModifiedTime != nil {
		result["modified_time"] = *bd.ModifiedTime
	}
	if bd.DeviceProfileID != nil {
		result["deviceprofile_id"] = *bd.DeviceProfileID
	}

	// Add status fields
	if bd.Connected != nil {
		result["connected"] = *bd.Connected
	}
	if bd.Adopted != nil {
		result["adopted"] = *bd.Adopted
	}
	if bd.Hostname != nil {
		result["hostname"] = *bd.Hostname
	}
	if bd.Notes != nil {
		result["notes"] = *bd.Notes
	}
	if bd.JSI != nil {
		result["jsi"] = *bd.JSI
	}

	// Add tags
	if bd.Tags != nil {
		result["tags"] = *bd.Tags
	}

	// Add additional configuration
	for key, value := range bd.AdditionalConfig {
		result[key] = value
	}

	return result
}

// ToConfigMap converts device to configuration map (for config files)
func (bd *BaseDevice) ToConfigMap() map[string]interface{} {
	result := make(map[string]interface{})

	// Only include configuration-relevant fields, not status fields
	if bd.Name != nil {
		result["name"] = *bd.Name
	}
	if bd.Magic != nil {
		result["magic"] = *bd.Magic
	}
	if bd.DeviceProfileID != nil {
		result["deviceprofile_id"] = *bd.DeviceProfileID
	}
	if bd.Notes != nil {
		result["notes"] = *bd.Notes
	}
	if bd.Tags != nil {
		result["tags"] = *bd.Tags
	}

	// Add additional configuration fields that should be included in config
	statusFields := map[string]bool{
		"id": true, "created_time": true, "modified_time": true,
		"connected": true, "adopted": true, "hostname": true, "jsi": true, "last_seen": true,
		"hw_rev": true,
	}

	for k, v := range bd.AdditionalConfig {
		if !statusFields[k] {
			result[k] = v
		}
	}

	return result
}

// FromConfigMap populates device from configuration map (from config files)
func (bd *BaseDevice) FromConfigMap(data map[string]interface{}) error {
	// Parse configuration fields only
	if name, ok := data["name"].(string); ok {
		bd.Name = &name
	}
	if magic, ok := data["magic"].(string); ok {
		bd.Magic = &magic
	}
	if deviceProfileID, ok := data["deviceprofile_id"].(string); ok {
		bd.DeviceProfileID = &deviceProfileID
	}
	if notes, ok := data["notes"].(string); ok {
		bd.Notes = &notes
	}

	// Parse tags
	if tags, ok := data["tags"].([]interface{}); ok {
		tagStrings := make([]string, 0, len(tags))
		for _, tag := range tags {
			if tagStr, tagOk := tag.(string); tagOk {
				tagStrings = append(tagStrings, tagStr)
			}
		}
		if len(tagStrings) > 0 {
			bd.Tags = &tagStrings
		}
	}

	// Initialize additional config if nil
	if bd.AdditionalConfig == nil {
		bd.AdditionalConfig = make(map[string]interface{})
	}

	// Store any unknown configuration fields in AdditionalConfig
	knownConfigFields := map[string]bool{
		"name": true, "notes": true, "tags": true, "deviceprofile_id": true,
	}

	// Exclude status fields as they shouldn't be in config
	statusFields := map[string]bool{
		"id": true, "created_time": true, "modified_time": true,
		"connected": true, "adopted": true, "hostname": true, "jsi": true,
		"hw_rev": true, "last_seen": true, "ip": true,
	}

	for k, v := range data {
		if !knownConfigFields[k] && !statusFields[k] {
			bd.AdditionalConfig[k] = v
		}
	}

	return nil
}

// Implement DeviceMarshaler for UnifiedDevice
func (ud *UnifiedDevice) GetID() *string     { return ud.BaseDevice.GetID() }
func (ud *UnifiedDevice) GetMAC() *string    { return ud.BaseDevice.GetMAC() }
func (ud *UnifiedDevice) GetSerial() *string { return ud.BaseDevice.GetSerial() }
func (ud *UnifiedDevice) GetName() *string   { return ud.BaseDevice.GetName() }
func (ud *UnifiedDevice) GetModel() *string  { return ud.BaseDevice.GetModel() }
func (ud *UnifiedDevice) GetType() *string   { return ud.BaseDevice.GetType() }
func (ud *UnifiedDevice) GetMagic() *string  { return ud.BaseDevice.GetMagic() }
func (ud *UnifiedDevice) GetSiteID() *string { return ud.BaseDevice.GetSiteID() }
func (ud *UnifiedDevice) GetOrgID() *string  { return ud.BaseDevice.GetOrgID() }
func (ud *UnifiedDevice) GetDeviceProfileID() *string {
	return ud.BaseDevice.GetDeviceProfileID()
}

// FromMap populates the UnifiedDevice from API response data
func (ud *UnifiedDevice) FromMap(data map[string]interface{}) error {
	// First populate base device fields
	if err := ud.BaseDevice.FromMap(data); err != nil {
		return fmt.Errorf("failed to populate base device fields: %w", err)
	}

	// Determine device type
	if deviceType, ok := data["type"].(string); ok {
		ud.DeviceType = deviceType
	}

	// Initialize device config map
	ud.DeviceConfig = make(map[string]interface{})

	// Define which fields belong to base device vs device-specific config
	baseFields := map[string]bool{
		"id": true, "mac": true, "serial": true, "name": true, "model": true, "type": true,
		"magic": true, "hw_rev": true, "sku": true, "site_id": true, "org_id": true,
		"created_time": true, "modified_time": true, "deviceprofile_id": true,
		"connected": true, "adopted": true, "hostname": true, "notes": true, "jsi": true, "tags": true,
	}

	// Store device-specific fields in DeviceConfig
	for k, v := range data {
		if !baseFields[k] {
			ud.DeviceConfig[k] = v
		}
	}

	return nil
}

// ToMap converts the UnifiedDevice to a map for API operations
func (ud *UnifiedDevice) ToMap() map[string]interface{} {
	// Start with base device fields
	result := ud.BaseDevice.ToMap()

	// Add device-specific configuration
	for k, v := range ud.DeviceConfig {
		if _, exists := result[k]; !exists {
			result[k] = v
		}
	}

	return result
}

// ToConfigMap converts unified device to configuration map (for config files)
func (ud *UnifiedDevice) ToConfigMap() map[string]interface{} {
	// Start with base configuration fields
	result := ud.BaseDevice.ToConfigMap()

	// Add device-specific configuration, filtering out status fields
	statusFields := map[string]bool{
		"connected": true, "adopted": true, "last_seen": true, "uptime": true,
		"version": true, "hw_rev": true, "sku": true, "ip": true, "status": true,
	}

	for k, v := range ud.DeviceConfig {
		if !statusFields[k] {
			if _, exists := result[k]; !exists {
				result[k] = v
			}
		}
	}

	return result
}

// FromConfigMap populates unified device from configuration map (from config files)
func (ud *UnifiedDevice) FromConfigMap(data map[string]interface{}) error {
	// First populate base device configuration
	if err := ud.BaseDevice.FromConfigMap(data); err != nil {
		return fmt.Errorf("failed to populate base device config: %w", err)
	}

	// Initialize device config map
	ud.DeviceConfig = make(map[string]interface{})

	// Define which fields belong to base device
	baseFields := map[string]bool{
		"name": true, "magic": true, "deviceprofile_id": true, "notes": true, "tags": true,
	}

	// Store device-specific configuration in DeviceConfig
	for k, v := range data {
		if !baseFields[k] {
			ud.DeviceConfig[k] = v
		}
	}

	return nil
}

// NewUnifiedDeviceFromType creates a new UnifiedDevice with the specified device type
func NewUnifiedDeviceFromType(deviceType string) *UnifiedDevice {
	return &UnifiedDevice{
		BaseDevice: BaseDevice{
			Type:             &deviceType,
			AdditionalConfig: make(map[string]interface{}),
		},
		DeviceConfig: make(map[string]interface{}),
		DeviceType:   deviceType,
	}
}

// NewUnifiedDeviceFromMap creates a new UnifiedDevice from API response data
func NewUnifiedDeviceFromMap(data map[string]interface{}) (*UnifiedDevice, error) {
	device := &UnifiedDevice{}
	if err := device.FromMap(data); err != nil {
		return nil, fmt.Errorf("failed to create unified device from map: %w", err)
	}
	return device, nil
}

// GetDeviceTypeFromMap determines device type from raw API data
func GetDeviceTypeFromMap(data map[string]interface{}) string {
	if deviceType, ok := data["type"].(string); ok {
		return deviceType
	}
	return "unknown"
}

// Verify device types implement DeviceMarshaler at compile time
var (
	_ DeviceMarshaler = (*BaseDevice)(nil)
	_ DeviceMarshaler = (*UnifiedDevice)(nil)
)

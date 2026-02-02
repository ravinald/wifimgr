package api

import (
	"fmt"

	"github.com/ravinald/wifimgr/internal/macaddr"
)

// InventoryMarshaler defines the interface for bidirectional inventory data transformation
// between API representations and structured types.
type InventoryMarshaler interface {
	GetID() string
	GetMAC() string
	GetType() string
	ToMap() map[string]interface{}
	FromMap(data map[string]interface{}) error
	ToConfigMap() map[string]interface{}
	FromConfigMap(data map[string]interface{}) error
	GetRaw() map[string]interface{}
	SetRaw(data map[string]interface{})
}

// MistInventoryItem represents a Mist inventory item with bidirectional data handling
type MistInventoryItem struct {
	// Core identification fields
	ID     *string `json:"id,omitempty"`
	MAC    *string `json:"mac,omitempty"`
	Serial *string `json:"serial,omitempty"`
	Name   *string `json:"name,omitempty"`
	Model  *string `json:"model,omitempty"`
	SKU    *string `json:"sku,omitempty"`
	HwRev  *string `json:"hw_rev,omitempty"`
	Type   *string `json:"type,omitempty"`
	Magic  *string `json:"magic,omitempty"`

	// Network and system fields
	Hostname *string `json:"hostname,omitempty"`
	JSI      *bool   `json:"jsi,omitempty"`

	// Virtual Chassis fields
	ChassisMac    *string `json:"chassis_mac,omitempty"`
	ChassisSerial *string `json:"chassis_serial,omitempty"`
	VcMac         *string `json:"vc_mac,omitempty"`

	// Organization and site assignment
	OrgID           *string `json:"org_id,omitempty"`
	SiteID          *string `json:"site_id,omitempty"`
	DeviceProfileID *string `json:"deviceprofile_id,omitempty"`

	// Status fields (read-only, not included in config)
	CreatedTime  *int64 `json:"created_time,omitempty"`
	ModifiedTime *int64 `json:"modified_time,omitempty"`
	Connected    *bool  `json:"connected,omitempty"`
	Adopted      *bool  `json:"adopted,omitempty"`

	// Additional flexible configuration stored as maps
	AdditionalConfig map[string]interface{} `json:"-"`

	// Raw contains the complete API response data for full preservation
	Raw map[string]interface{} `json:"-"`
}

// GetID returns the inventory item ID as a string
func (i *MistInventoryItem) GetID() string {
	if i.ID != nil {
		return *i.ID
	}
	return ""
}

// GetMAC returns the inventory item MAC address, normalized
func (i *MistInventoryItem) GetMAC() string {
	if i.MAC != nil {
		normalized, err := macaddr.Normalize(*i.MAC)
		if err != nil {
			return *i.MAC // Return original if normalization fails
		}
		return normalized
	}
	return ""
}

// GetType returns the inventory item type
func (i *MistInventoryItem) GetType() string {
	if i.Type != nil {
		return *i.Type
	}
	return ""
}

// ToMap converts the inventory item to a map representation suitable for API requests
func (i *MistInventoryItem) ToMap() map[string]interface{} {
	result := make(map[string]interface{})

	// Add typed fields
	if i.ID != nil {
		result["id"] = *i.ID
	}
	if i.MAC != nil {
		result["mac"] = *i.MAC
	}
	if i.Serial != nil {
		result["serial"] = *i.Serial
	}
	if i.Name != nil {
		result["name"] = *i.Name
	}
	if i.Model != nil {
		result["model"] = *i.Model
	}
	if i.SKU != nil {
		result["sku"] = *i.SKU
	}
	if i.HwRev != nil {
		result["hw_rev"] = *i.HwRev
	}
	if i.Type != nil {
		result["type"] = *i.Type
	}
	if i.Magic != nil {
		result["magic"] = *i.Magic
	}
	if i.Hostname != nil {
		result["hostname"] = *i.Hostname
	}
	if i.JSI != nil {
		result["jsi"] = *i.JSI
	}
	if i.ChassisMac != nil {
		result["chassis_mac"] = *i.ChassisMac
	}
	if i.ChassisSerial != nil {
		result["chassis_serial"] = *i.ChassisSerial
	}
	if i.VcMac != nil {
		result["vc_mac"] = *i.VcMac
	}
	if i.OrgID != nil {
		result["org_id"] = *i.OrgID
	}
	if i.SiteID != nil {
		result["site_id"] = *i.SiteID
	}
	if i.DeviceProfileID != nil {
		result["deviceprofile_id"] = *i.DeviceProfileID
	}

	// Add status fields
	if i.CreatedTime != nil {
		result["created_time"] = *i.CreatedTime
	}
	if i.ModifiedTime != nil {
		result["modified_time"] = *i.ModifiedTime
	}
	if i.Connected != nil {
		result["connected"] = *i.Connected
	}
	if i.Adopted != nil {
		result["adopted"] = *i.Adopted
	}

	// Add additional configuration
	for key, value := range i.AdditionalConfig {
		result[key] = value
	}

	// Add any raw data that wasn't captured above
	for key, value := range i.Raw {
		if _, exists := result[key]; !exists {
			result[key] = value
		}
	}

	return result
}

// FromMap populates the inventory item from a map representation (e.g., from API response)
func (i *MistInventoryItem) FromMap(data map[string]interface{}) error {
	if data == nil {
		return fmt.Errorf("data cannot be nil")
	}

	// Store raw data for complete preservation
	i.Raw = make(map[string]interface{})
	for k, v := range data {
		i.Raw[k] = v
	}

	// Initialize additional config if nil
	if i.AdditionalConfig == nil {
		i.AdditionalConfig = make(map[string]interface{})
	}

	// Extract typed fields
	if id, ok := data["id"].(string); ok {
		i.ID = &id
	}
	if mac, ok := data["mac"].(string); ok {
		i.MAC = &mac
	}
	if serial, ok := data["serial"].(string); ok {
		i.Serial = &serial
	}
	if name, ok := data["name"].(string); ok {
		i.Name = &name
	}
	if model, ok := data["model"].(string); ok {
		i.Model = &model
	}
	if sku, ok := data["sku"].(string); ok {
		i.SKU = &sku
	}
	if hwRev, ok := data["hw_rev"].(string); ok {
		i.HwRev = &hwRev
	}
	if deviceType, ok := data["type"].(string); ok {
		i.Type = &deviceType
	}
	if magic, ok := data["magic"].(string); ok {
		i.Magic = &magic
	}
	if hostname, ok := data["hostname"].(string); ok {
		i.Hostname = &hostname
	}
	if jsi, ok := data["jsi"].(bool); ok {
		i.JSI = &jsi
	}
	if chassisMac, ok := data["chassis_mac"].(string); ok {
		i.ChassisMac = &chassisMac
	}
	if chassisSerial, ok := data["chassis_serial"].(string); ok {
		i.ChassisSerial = &chassisSerial
	}
	if vcMac, ok := data["vc_mac"].(string); ok {
		i.VcMac = &vcMac
	}
	if orgID, ok := data["org_id"].(string); ok {
		i.OrgID = &orgID
	}
	if siteID, ok := data["site_id"].(string); ok {
		i.SiteID = &siteID
	}
	if deviceProfileID, ok := data["deviceprofile_id"].(string); ok {
		i.DeviceProfileID = &deviceProfileID
	}

	// Handle status fields with proper type conversion
	if createdTime, ok := data["created_time"].(float64); ok {
		createdTimeInt := int64(createdTime)
		i.CreatedTime = &createdTimeInt
	} else if createdTime, ok := data["created_time"].(int64); ok {
		i.CreatedTime = &createdTime
	}
	if modifiedTime, ok := data["modified_time"].(float64); ok {
		modifiedTimeInt := int64(modifiedTime)
		i.ModifiedTime = &modifiedTimeInt
	} else if modifiedTime, ok := data["modified_time"].(int64); ok {
		i.ModifiedTime = &modifiedTime
	}
	if connected, ok := data["connected"].(bool); ok {
		i.Connected = &connected
	}
	if adopted, ok := data["adopted"].(bool); ok {
		i.Adopted = &adopted
	}

	// Store any additional fields in AdditionalConfig
	statusFields := map[string]bool{
		"created_time":  true,
		"modified_time": true,
		"connected":     true,
		"adopted":       true,
	}

	knownFields := map[string]bool{
		"id":               true,
		"mac":              true,
		"serial":           true,
		"name":             true,
		"model":            true,
		"sku":              true,
		"hw_rev":           true,
		"type":             true,
		"magic":            true,
		"hostname":         true,
		"jsi":              true,
		"chassis_mac":      true,
		"chassis_serial":   true,
		"vc_mac":           true,
		"org_id":           true,
		"site_id":          true,
		"deviceprofile_id": true,
		"created_time":     true,
		"modified_time":    true,
		"connected":        true,
		"adopted":          true,
	}

	for key, value := range data {
		if !knownFields[key] && !statusFields[key] {
			i.AdditionalConfig[key] = value
		}
	}

	return nil
}

// ToConfigMap converts the inventory item to a map suitable for configuration files (excludes status fields)
func (i *MistInventoryItem) ToConfigMap() map[string]interface{} {
	result := make(map[string]interface{})

	// Add only configuration fields (exclude status fields)
	if i.MAC != nil {
		result["mac"] = *i.MAC
	}
	if i.Serial != nil {
		result["serial"] = *i.Serial
	}
	if i.Name != nil {
		result["name"] = *i.Name
	}
	if i.Model != nil {
		result["model"] = *i.Model
	}
	if i.SKU != nil {
		result["sku"] = *i.SKU
	}
	if i.HwRev != nil {
		result["hw_rev"] = *i.HwRev
	}
	if i.Type != nil {
		result["type"] = *i.Type
	}
	if i.Magic != nil {
		result["magic"] = *i.Magic
	}
	if i.Hostname != nil {
		result["hostname"] = *i.Hostname
	}
	if i.JSI != nil {
		result["jsi"] = *i.JSI
	}
	if i.ChassisMac != nil {
		result["chassis_mac"] = *i.ChassisMac
	}
	if i.ChassisSerial != nil {
		result["chassis_serial"] = *i.ChassisSerial
	}
	if i.VcMac != nil {
		result["vc_mac"] = *i.VcMac
	}
	if i.SiteID != nil {
		result["site_id"] = *i.SiteID
	}
	if i.DeviceProfileID != nil {
		result["deviceprofile_id"] = *i.DeviceProfileID
	}

	// Add additional configuration (excluding status fields)
	statusFields := map[string]bool{
		"id":            true,
		"org_id":        true,
		"created_time":  true,
		"modified_time": true,
		"connected":     true,
		"adopted":       true,
		"hw_rev":        true,
	}

	for key, value := range i.AdditionalConfig {
		if !statusFields[key] {
			result[key] = value
		}
	}

	return result
}

// FromConfigMap populates the inventory item from a configuration map (excludes status fields)
func (i *MistInventoryItem) FromConfigMap(data map[string]interface{}) error {
	if data == nil {
		return fmt.Errorf("data cannot be nil")
	}

	// Initialize additional config if nil
	if i.AdditionalConfig == nil {
		i.AdditionalConfig = make(map[string]interface{})
	}

	// Extract typed fields
	if mac, ok := data["mac"].(string); ok {
		i.MAC = &mac
	}
	if serial, ok := data["serial"].(string); ok {
		i.Serial = &serial
	}
	if name, ok := data["name"].(string); ok {
		i.Name = &name
	}
	if model, ok := data["model"].(string); ok {
		i.Model = &model
	}
	if sku, ok := data["sku"].(string); ok {
		i.SKU = &sku
	}
	if hwRev, ok := data["hw_rev"].(string); ok {
		i.HwRev = &hwRev
	}
	if deviceType, ok := data["type"].(string); ok {
		i.Type = &deviceType
	}
	if magic, ok := data["magic"].(string); ok {
		i.Magic = &magic
	}
	if hostname, ok := data["hostname"].(string); ok {
		i.Hostname = &hostname
	}
	if jsi, ok := data["jsi"].(bool); ok {
		i.JSI = &jsi
	}
	if chassisMac, ok := data["chassis_mac"].(string); ok {
		i.ChassisMac = &chassisMac
	}
	if chassisSerial, ok := data["chassis_serial"].(string); ok {
		i.ChassisSerial = &chassisSerial
	}
	if vcMac, ok := data["vc_mac"].(string); ok {
		i.VcMac = &vcMac
	}
	if siteID, ok := data["site_id"].(string); ok {
		i.SiteID = &siteID
	}
	if deviceProfileID, ok := data["deviceprofile_id"].(string); ok {
		i.DeviceProfileID = &deviceProfileID
	}

	// Store any additional fields in AdditionalConfig (exclude status fields)
	knownConfigFields := map[string]bool{
		"mac":              true,
		"serial":           true,
		"name":             true,
		"model":            true,
		"sku":              true,
		"type":             true,
		"magic":            true,
		"hostname":         true,
		"jsi":              true,
		"chassis_mac":      true,
		"chassis_serial":   true,
		"vc_mac":           true,
		"site_id":          true,
		"deviceprofile_id": true,
	}

	statusFields := map[string]bool{
		"id":            true,
		"org_id":        true,
		"created_time":  true,
		"modified_time": true,
		"connected":     true,
		"adopted":       true,
	}

	for key, value := range data {
		if !knownConfigFields[key] && !statusFields[key] {
			i.AdditionalConfig[key] = value
		}
	}

	return nil
}

// GetRaw returns the raw API data
func (i *MistInventoryItem) GetRaw() map[string]interface{} {
	if i.Raw == nil {
		return make(map[string]interface{})
	}
	// Return a copy to prevent external modifications
	result := make(map[string]interface{})
	for k, v := range i.Raw {
		result[k] = v
	}
	return result
}

// SetRaw sets the raw API data
func (i *MistInventoryItem) SetRaw(data map[string]interface{}) {
	i.Raw = make(map[string]interface{})
	for k, v := range data {
		i.Raw[k] = v
	}
}

// NewInventoryItemFromMap creates a new inventory item from a map representation
func NewInventoryItemFromMap(data map[string]interface{}) (*MistInventoryItem, error) {
	item := &MistInventoryItem{
		AdditionalConfig: make(map[string]interface{}),
	}

	if err := item.FromMap(data); err != nil {
		return nil, fmt.Errorf("failed to create inventory item from map: %w", err)
	}

	return item, nil
}

// NewInventoryItemFromConfigMap creates a new inventory item from a configuration map
func NewInventoryItemFromConfigMap(data map[string]interface{}) (*MistInventoryItem, error) {
	item := &MistInventoryItem{
		AdditionalConfig: make(map[string]interface{}),
	}

	if err := item.FromConfigMap(data); err != nil {
		return nil, fmt.Errorf("failed to create inventory item from config map: %w", err)
	}

	return item, nil
}

// ConvertInventoryItemToNew converts a legacy InventoryItem to MistInventoryItem
func ConvertInventoryItemToNew(oldItem InventoryItem) *MistInventoryItem {
	item := &MistInventoryItem{
		AdditionalConfig: make(map[string]interface{}),
		Raw:              make(map[string]interface{}),
	}

	// Convert typed fields
	if oldItem.Id != nil {
		id := string(*oldItem.Id)
		item.ID = &id
	}
	item.MAC = oldItem.Mac
	item.Serial = oldItem.Serial
	item.Name = oldItem.Name
	item.Model = oldItem.Model
	item.SKU = oldItem.SKU
	item.HwRev = oldItem.HwRev
	item.Type = oldItem.Type
	item.Magic = oldItem.Magic
	item.Hostname = oldItem.Hostname
	item.JSI = oldItem.JSI
	item.ChassisMac = oldItem.ChassisMac
	item.ChassisSerial = oldItem.ChassisSerial
	item.VcMac = oldItem.VcMac
	if oldItem.OrgId != nil {
		orgId := string(*oldItem.OrgId)
		item.OrgID = &orgId
	}
	if oldItem.SiteId != nil {
		siteId := string(*oldItem.SiteId)
		item.SiteID = &siteId
	}
	item.DeviceProfileID = oldItem.DeviceProfileId
	item.CreatedTime = oldItem.CreatedTime
	item.ModifiedTime = oldItem.ModifiedTime
	item.Connected = oldItem.Connected
	item.Adopted = oldItem.Adopted

	// Store in raw data as well for consistency
	if item.ID != nil {
		item.Raw["id"] = *item.ID
	}
	if item.MAC != nil {
		item.Raw["mac"] = *item.MAC
	}
	if item.Serial != nil {
		item.Raw["serial"] = *item.Serial
	}
	if item.Name != nil {
		item.Raw["name"] = *item.Name
	}
	if item.Model != nil {
		item.Raw["model"] = *item.Model
	}
	if item.SKU != nil {
		item.Raw["sku"] = *item.SKU
	}
	if item.HwRev != nil {
		item.Raw["hw_rev"] = *item.HwRev
	}
	if item.Type != nil {
		item.Raw["type"] = *item.Type
	}
	if item.Magic != nil {
		item.Raw["magic"] = *item.Magic
	}
	if item.Hostname != nil {
		item.Raw["hostname"] = *item.Hostname
	}
	if item.JSI != nil {
		item.Raw["jsi"] = *item.JSI
	}
	if item.ChassisMac != nil {
		item.Raw["chassis_mac"] = *item.ChassisMac
	}
	if item.ChassisSerial != nil {
		item.Raw["chassis_serial"] = *item.ChassisSerial
	}
	if item.VcMac != nil {
		item.Raw["vc_mac"] = *item.VcMac
	}
	if item.OrgID != nil {
		item.Raw["org_id"] = *item.OrgID
	}
	if item.SiteID != nil {
		item.Raw["site_id"] = *item.SiteID
	}
	if item.DeviceProfileID != nil {
		item.Raw["deviceprofile_id"] = *item.DeviceProfileID
	}
	if item.CreatedTime != nil {
		item.Raw["created_time"] = *item.CreatedTime
	}
	if item.ModifiedTime != nil {
		item.Raw["modified_time"] = *item.ModifiedTime
	}
	if item.Connected != nil {
		item.Raw["connected"] = *item.Connected
	}
	if item.Adopted != nil {
		item.Raw["adopted"] = *item.Adopted
	}

	return item
}

// ConvertInventoryItemFromNew converts an MistInventoryItem back to legacy InventoryItem
func ConvertInventoryItemFromNew(newItem *MistInventoryItem) InventoryItem {
	item := InventoryItem{}

	// Convert typed fields
	if newItem.ID != nil {
		id := UUID(*newItem.ID)
		item.Id = &id
	}
	item.Mac = newItem.MAC
	item.Serial = newItem.Serial
	item.Name = newItem.Name
	item.Model = newItem.Model
	item.SKU = newItem.SKU
	item.HwRev = newItem.HwRev
	item.Type = newItem.Type
	item.Magic = newItem.Magic
	item.Hostname = newItem.Hostname
	item.JSI = newItem.JSI
	item.ChassisMac = newItem.ChassisMac
	item.ChassisSerial = newItem.ChassisSerial
	item.VcMac = newItem.VcMac
	if newItem.OrgID != nil {
		orgId := UUID(*newItem.OrgID)
		item.OrgId = &orgId
	}
	if newItem.SiteID != nil {
		siteId := UUID(*newItem.SiteID)
		item.SiteId = &siteId
	}
	item.DeviceProfileId = newItem.DeviceProfileID
	item.CreatedTime = newItem.CreatedTime
	item.ModifiedTime = newItem.ModifiedTime
	item.Connected = newItem.Connected
	item.Adopted = newItem.Adopted

	return item
}

// Verify MistInventoryItem implements InventoryMarshaler at compile time
var _ InventoryMarshaler = (*MistInventoryItem)(nil)

package api

import (
	"fmt"
)

// APDevice represents a Mist access point with complete data preservation
type APDevice struct {
	BaseDevice

	// AP-specific typed fields for frequently used configurations
	// Position and orientation
	Location    *[]float64 `json:"location,omitempty"`
	Orientation *int       `json:"orientation,omitempty"`
	MapID       *string    `json:"map_id,omitempty"`
	Height      *float64   `json:"height,omitempty"`
	X           *float64   `json:"x,omitempty"`
	Y           *float64   `json:"y,omitempty"`

	// Site and locking
	ForSite *bool `json:"for_site,omitempty"`
	Locked  *bool `json:"locked,omitempty"`

	// Hardware configuration - common fields with typed access
	Led            *bool `json:"led,omitempty"`
	DisableEth1    *bool `json:"disable_eth1,omitempty"`
	DisableEth2    *bool `json:"disable_eth2,omitempty"`
	DisableEth3    *bool `json:"disable_eth3,omitempty"`
	DisableModule  *bool `json:"disable_module,omitempty"`
	PoEPassthrough *bool `json:"poe_passthrough,omitempty"`

	// Complex configuration stored as maps for flexibility and evolution
	RadioConfig        map[string]interface{} `json:"radio_config,omitempty"`
	BleConfig          map[string]interface{} `json:"ble_config,omitempty"`
	IotConfig          map[string]interface{} `json:"iot_config,omitempty"`
	IPConfig           map[string]interface{} `json:"ip_config,omitempty"`
	MeshConfig         map[string]interface{} `json:"mesh,omitempty"`
	PortConfig         map[string]interface{} `json:"port_config,omitempty"`
	UplinkPortConfig   map[string]interface{} `json:"uplink_port_config,omitempty"`
	LedConfig          map[string]interface{} `json:"led_config,omitempty"`
	AeroscoutConfig    map[string]interface{} `json:"aeroscout,omitempty"`
	CentrakConfig      map[string]interface{} `json:"centrak,omitempty"`
	ClientBridgeConfig map[string]interface{} `json:"client_bridge,omitempty"`
	EslConfig          map[string]interface{} `json:"esl_config,omitempty"`
	PwrConfig          map[string]interface{} `json:"pwr_config,omitempty"`
	UsbConfig          map[string]interface{} `json:"usb_config,omitempty"`

	// Additional config for unknown fields
	AdditionalConfig map[string]interface{} `json:"-"`
}

// APDeviceInterface defines the interface for AP device operations.
// Note: The "Interface" suffix is retained because there is a struct named APDevice.
type APDeviceInterface interface {
	DeviceMarshaler
	GetLocation() *[]float64
	GetOrientation() *int
	GetMapID() *string
	GetHeight() *float64
	GetX() *float64
	GetY() *float64
	GetForSite() *bool
	GetLocked() *bool
	GetRadioConfig() map[string]interface{}
	GetBleConfig() map[string]interface{}
	GetIotConfig() map[string]interface{}
}

// Implement APDeviceInterface
func (ap *APDevice) GetLocation() *[]float64                { return ap.Location }
func (ap *APDevice) GetOrientation() *int                   { return ap.Orientation }
func (ap *APDevice) GetMapID() *string                      { return ap.MapID }
func (ap *APDevice) GetHeight() *float64                    { return ap.Height }
func (ap *APDevice) GetX() *float64                         { return ap.X }
func (ap *APDevice) GetY() *float64                         { return ap.Y }
func (ap *APDevice) GetForSite() *bool                      { return ap.ForSite }
func (ap *APDevice) GetLocked() *bool                       { return ap.Locked }
func (ap *APDevice) GetRadioConfig() map[string]interface{} { return ap.RadioConfig }
func (ap *APDevice) GetBleConfig() map[string]interface{}   { return ap.BleConfig }
func (ap *APDevice) GetIotConfig() map[string]interface{}   { return ap.IotConfig }

// FromMap populates the APDevice from API response data
func (ap *APDevice) FromMap(data map[string]interface{}) error {
	// First populate base device fields
	if err := ap.BaseDevice.FromMap(data); err != nil {
		return fmt.Errorf("failed to populate base device fields: %w", err)
	}

	// Parse AP-specific typed fields
	if location, ok := data["location"].([]interface{}); ok {
		floatLocation := make([]float64, 0, len(location))
		for _, loc := range location {
			if f, fOk := loc.(float64); fOk {
				floatLocation = append(floatLocation, f)
			}
		}
		if len(floatLocation) > 0 {
			ap.Location = &floatLocation
		}
	}

	if orientation, ok := data["orientation"].(float64); ok {
		orientInt := int(orientation)
		ap.Orientation = &orientInt
	}

	if mapID, ok := data["map_id"].(string); ok {
		ap.MapID = &mapID
	}

	if height, ok := data["height"].(float64); ok {
		ap.Height = &height
	}

	if x, ok := data["x"].(float64); ok {
		ap.X = &x
	}

	if y, ok := data["y"].(float64); ok {
		ap.Y = &y
	}

	if forSite, ok := data["for_site"].(bool); ok {
		ap.ForSite = &forSite
	}

	if locked, ok := data["locked"].(bool); ok {
		ap.Locked = &locked
	}

	if led, ok := data["led"].(bool); ok {
		ap.Led = &led
	}

	if disableEth1, ok := data["disable_eth1"].(bool); ok {
		ap.DisableEth1 = &disableEth1
	}

	if disableEth2, ok := data["disable_eth2"].(bool); ok {
		ap.DisableEth2 = &disableEth2
	}

	if disableEth3, ok := data["disable_eth3"].(bool); ok {
		ap.DisableEth3 = &disableEth3
	}

	if disableModule, ok := data["disable_module"].(bool); ok {
		ap.DisableModule = &disableModule
	}

	if poePassthrough, ok := data["poe_passthrough"].(bool); ok {
		ap.PoEPassthrough = &poePassthrough
	}

	// Parse complex configuration objects
	configFields := map[string]*map[string]interface{}{
		"radio_config":       &ap.RadioConfig,
		"ble_config":         &ap.BleConfig,
		"iot_config":         &ap.IotConfig,
		"ip_config":          &ap.IPConfig,
		"mesh":               &ap.MeshConfig,
		"port_config":        &ap.PortConfig,
		"uplink_port_config": &ap.UplinkPortConfig,
		"led_config":         &ap.LedConfig,
		"aeroscout":          &ap.AeroscoutConfig,
		"centrak":            &ap.CentrakConfig,
		"client_bridge":      &ap.ClientBridgeConfig,
		"esl_config":         &ap.EslConfig,
		"pwr_config":         &ap.PwrConfig,
		"usb_config":         &ap.UsbConfig,
	}

	for fieldName, configPtr := range configFields {
		if configData, ok := data[fieldName].(map[string]interface{}); ok {
			*configPtr = configData
		}
	}

	// Store any unknown fields in AdditionalConfig
	ap.AdditionalConfig = make(map[string]interface{})
	knownFields := map[string]bool{
		// Base device fields
		"id": true, "mac": true, "serial": true, "name": true, "model": true, "type": true,
		"magic": true, "hw_rev": true, "sku": true, "site_id": true, "org_id": true,
		"created_time": true, "modified_time": true, "deviceprofile_id": true,
		"connected": true, "adopted": true, "hostname": true, "notes": true, "jsi": true, "tags": true,

		// AP-specific typed fields
		"location": true, "orientation": true, "map_id": true, "height": true, "x": true, "y": true,
		"for_site": true, "locked": true, "led": true, "disable_eth1": true, "disable_eth2": true,
		"disable_eth3": true, "disable_module": true, "poe_passthrough": true,

		// AP complex config fields
		"radio_config": true, "ble_config": true, "iot_config": true, "ip_config": true,
		"mesh": true, "port_config": true, "uplink_port_config": true, "led_config": true,
		"aeroscout": true, "centrak": true, "client_bridge": true, "esl_config": true,
		"pwr_config": true, "usb_config": true,
	}

	for k, v := range data {
		if !knownFields[k] {
			ap.AdditionalConfig[k] = v
		}
	}

	return nil
}

// ToMap converts the APDevice to a map for API operations
func (ap *APDevice) ToMap() map[string]interface{} {
	// Start with base device fields
	result := ap.BaseDevice.ToMap()

	// Add AP-specific typed fields
	if ap.Location != nil {
		result["location"] = *ap.Location
	}
	if ap.Orientation != nil {
		result["orientation"] = *ap.Orientation
	}
	if ap.MapID != nil {
		result["map_id"] = *ap.MapID
	}
	if ap.Height != nil {
		result["height"] = *ap.Height
	}
	if ap.X != nil {
		result["x"] = *ap.X
	}
	if ap.Y != nil {
		result["y"] = *ap.Y
	}
	if ap.ForSite != nil {
		result["for_site"] = *ap.ForSite
	}
	if ap.Locked != nil {
		result["locked"] = *ap.Locked
	}
	if ap.Led != nil {
		result["led"] = *ap.Led
	}
	if ap.DisableEth1 != nil {
		result["disable_eth1"] = *ap.DisableEth1
	}
	if ap.DisableEth2 != nil {
		result["disable_eth2"] = *ap.DisableEth2
	}
	if ap.DisableEth3 != nil {
		result["disable_eth3"] = *ap.DisableEth3
	}
	if ap.DisableModule != nil {
		result["disable_module"] = *ap.DisableModule
	}
	if ap.PoEPassthrough != nil {
		result["poe_passthrough"] = *ap.PoEPassthrough
	}

	// Add complex configuration objects
	if ap.RadioConfig != nil {
		result["radio_config"] = ap.RadioConfig
	}
	if ap.BleConfig != nil {
		result["ble_config"] = ap.BleConfig
	}
	if ap.IotConfig != nil {
		result["iot_config"] = ap.IotConfig
	}
	if ap.IPConfig != nil {
		result["ip_config"] = ap.IPConfig
	}
	if ap.MeshConfig != nil {
		result["mesh"] = ap.MeshConfig
	}
	if ap.PortConfig != nil {
		result["port_config"] = ap.PortConfig
	}
	if ap.UplinkPortConfig != nil {
		result["uplink_port_config"] = ap.UplinkPortConfig
	}
	if ap.LedConfig != nil {
		result["led_config"] = ap.LedConfig
	}
	if ap.AeroscoutConfig != nil {
		result["aeroscout"] = ap.AeroscoutConfig
	}
	if ap.CentrakConfig != nil {
		result["centrak"] = ap.CentrakConfig
	}
	if ap.ClientBridgeConfig != nil {
		result["client_bridge"] = ap.ClientBridgeConfig
	}
	if ap.EslConfig != nil {
		result["esl_config"] = ap.EslConfig
	}
	if ap.PwrConfig != nil {
		result["pwr_config"] = ap.PwrConfig
	}
	if ap.UsbConfig != nil {
		result["usb_config"] = ap.UsbConfig
	}

	// Add additional unknown fields
	for k, v := range ap.AdditionalConfig {
		if _, exists := result[k]; !exists {
			result[k] = v
		}
	}

	return result
}

// ToConfigMap converts AP device to configuration map (for config files)
func (ap *APDevice) ToConfigMap() map[string]interface{} {
	// Start with base configuration fields
	result := ap.BaseDevice.ToConfigMap()

	// Add AP-specific configuration fields (exclude status/runtime fields)
	if ap.Location != nil {
		result["location"] = *ap.Location
	}
	if ap.Orientation != nil {
		result["orientation"] = *ap.Orientation
	}
	if ap.MapID != nil {
		result["map_id"] = *ap.MapID
	}
	if ap.Height != nil {
		result["height"] = *ap.Height
	}
	if ap.X != nil {
		result["x"] = *ap.X
	}
	if ap.Y != nil {
		result["y"] = *ap.Y
	}
	if ap.ForSite != nil {
		result["for_site"] = *ap.ForSite
	}
	if ap.Locked != nil {
		result["locked"] = *ap.Locked
	}
	if ap.Led != nil {
		result["led"] = *ap.Led
	}
	if ap.DisableEth1 != nil {
		result["disable_eth1"] = *ap.DisableEth1
	}
	if ap.DisableEth2 != nil {
		result["disable_eth2"] = *ap.DisableEth2
	}
	if ap.DisableEth3 != nil {
		result["disable_eth3"] = *ap.DisableEth3
	}
	if ap.DisableModule != nil {
		result["disable_module"] = *ap.DisableModule
	}
	if ap.PoEPassthrough != nil {
		result["poe_passthrough"] = *ap.PoEPassthrough
	}

	// Add configuration objects (all are configuration, not status)
	if ap.RadioConfig != nil {
		result["radio_config"] = ap.RadioConfig
	}
	if ap.BleConfig != nil {
		result["ble_config"] = ap.BleConfig
	}
	if ap.IotConfig != nil {
		result["iot_config"] = ap.IotConfig
	}
	if ap.IPConfig != nil {
		result["ip_config"] = ap.IPConfig
	}
	if ap.MeshConfig != nil {
		result["mesh"] = ap.MeshConfig
	}
	if ap.PortConfig != nil {
		result["port_config"] = ap.PortConfig
	}
	if ap.UplinkPortConfig != nil {
		result["uplink_port_config"] = ap.UplinkPortConfig
	}
	if ap.LedConfig != nil {
		result["led_config"] = ap.LedConfig
	}
	if ap.AeroscoutConfig != nil {
		result["aeroscout"] = ap.AeroscoutConfig
	}
	if ap.CentrakConfig != nil {
		result["centrak"] = ap.CentrakConfig
	}
	if ap.ClientBridgeConfig != nil {
		result["client_bridge"] = ap.ClientBridgeConfig
	}
	if ap.EslConfig != nil {
		result["esl_config"] = ap.EslConfig
	}
	if ap.PwrConfig != nil {
		result["pwr_config"] = ap.PwrConfig
	}
	if ap.UsbConfig != nil {
		result["usb_config"] = ap.UsbConfig
	}

	// Add configuration fields from AdditionalConfig, filtering out status fields
	statusFields := map[string]bool{
		"connected": true, "adopted": true, "last_seen": true, "uptime": true,
		"version": true, "ip": true, "status": true, "hw_rev": true, "sku": true,
	}

	for k, v := range ap.AdditionalConfig {
		if !statusFields[k] {
			if _, exists := result[k]; !exists {
				result[k] = v
			}
		}
	}

	return result
}

// FromConfigMap populates AP device from configuration map (from config files)
func (ap *APDevice) FromConfigMap(data map[string]interface{}) error {
	// First populate base device configuration
	if err := ap.BaseDevice.FromConfigMap(data); err != nil {
		return fmt.Errorf("failed to populate base device config: %w", err)
	}

	// Parse AP-specific configuration fields
	if location, ok := data["location"].([]interface{}); ok {
		floatLocation := make([]float64, 0, len(location))
		for _, loc := range location {
			if f, fOk := loc.(float64); fOk {
				floatLocation = append(floatLocation, f)
			}
		}
		if len(floatLocation) > 0 {
			ap.Location = &floatLocation
		}
	}

	if orientation, ok := data["orientation"].(float64); ok {
		orientInt := int(orientation)
		ap.Orientation = &orientInt
	}

	if mapID, ok := data["map_id"].(string); ok {
		ap.MapID = &mapID
	}

	if height, ok := data["height"].(float64); ok {
		ap.Height = &height
	}

	if x, ok := data["x"].(float64); ok {
		ap.X = &x
	}

	if y, ok := data["y"].(float64); ok {
		ap.Y = &y
	}

	if forSite, ok := data["for_site"].(bool); ok {
		ap.ForSite = &forSite
	}

	if locked, ok := data["locked"].(bool); ok {
		ap.Locked = &locked
	}

	if led, ok := data["led"].(bool); ok {
		ap.Led = &led
	}

	if disableEth1, ok := data["disable_eth1"].(bool); ok {
		ap.DisableEth1 = &disableEth1
	}

	if disableEth2, ok := data["disable_eth2"].(bool); ok {
		ap.DisableEth2 = &disableEth2
	}

	if disableEth3, ok := data["disable_eth3"].(bool); ok {
		ap.DisableEth3 = &disableEth3
	}

	if disableModule, ok := data["disable_module"].(bool); ok {
		ap.DisableModule = &disableModule
	}

	if poePassthrough, ok := data["poe_passthrough"].(bool); ok {
		ap.PoEPassthrough = &poePassthrough
	}

	// Parse complex configuration objects
	configFields := map[string]*map[string]interface{}{
		"radio_config":       &ap.RadioConfig,
		"ble_config":         &ap.BleConfig,
		"iot_config":         &ap.IotConfig,
		"ip_config":          &ap.IPConfig,
		"mesh":               &ap.MeshConfig,
		"port_config":        &ap.PortConfig,
		"uplink_port_config": &ap.UplinkPortConfig,
		"led_config":         &ap.LedConfig,
		"aeroscout":          &ap.AeroscoutConfig,
		"centrak":            &ap.CentrakConfig,
		"client_bridge":      &ap.ClientBridgeConfig,
		"esl_config":         &ap.EslConfig,
		"pwr_config":         &ap.PwrConfig,
		"usb_config":         &ap.UsbConfig,
	}

	for fieldName, configPtr := range configFields {
		if configData, ok := data[fieldName].(map[string]interface{}); ok {
			*configPtr = configData
		}
	}

	// Store any unknown configuration fields in AdditionalConfig
	ap.AdditionalConfig = make(map[string]interface{})
	knownFields := map[string]bool{
		// Base device config fields
		"name": true, "magic": true, "deviceprofile_id": true, "notes": true, "tags": true,

		// AP-specific fields
		"location": true, "orientation": true, "map_id": true, "height": true, "x": true, "y": true,
		"for_site": true, "locked": true, "led": true, "disable_eth1": true, "disable_eth2": true,
		"disable_eth3": true, "disable_module": true, "poe_passthrough": true,

		// AP complex config fields
		"radio_config": true, "ble_config": true, "iot_config": true, "ip_config": true,
		"mesh": true, "port_config": true, "uplink_port_config": true, "led_config": true,
		"aeroscout": true, "centrak": true, "client_bridge": true, "esl_config": true,
		"pwr_config": true, "usb_config": true,
	}

	for k, v := range data {
		if !knownFields[k] {
			ap.AdditionalConfig[k] = v
		}
	}

	return nil
}

// NewAPDeviceFromMap creates a new APDevice from API response data
func NewAPDeviceFromMap(data map[string]interface{}) (*APDevice, error) {
	device := &APDevice{}
	if err := device.FromMap(data); err != nil {
		return nil, fmt.Errorf("failed to create AP device from map: %w", err)
	}
	return device, nil
}

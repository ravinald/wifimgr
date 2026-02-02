package api

import "fmt"

// DeviceConfig represents a device configuration retrieved from the API
// It's a wrapper that can hold any device type configuration
type DeviceConfig struct {
	// The actual device configuration data stored as interface{}
	// to support AP, Switch, and Gateway configs
	Data interface{} `json:"data"`

	// Device type for runtime type assertion
	Type string `json:"type"`

	// Common fields for indexing
	ID     string `json:"id"`
	Name   string `json:"name"`
	MAC    string `json:"mac"`
	SiteID string `json:"site_id"`
}

// APConfig represents an AP device configuration
// This uses the same structure as APDevice since configs follow the same schema
type APConfig = APDevice

// SwitchConfig represents a Switch device configuration
// This uses the same structure as MistSwitchDevice since configs follow the same schema
type SwitchConfig = MistSwitchDevice

// GatewayConfig represents a Gateway device configuration
// This uses the same structure as MistGatewayDevice since configs follow the same schema
type GatewayConfig = MistGatewayDevice

// DeviceConfigResponse represents the API response for a device configuration
type DeviceConfigResponse struct {
	// Raw JSON data from the API
	Raw map[string]interface{}
}

// ToDeviceConfig converts the raw response to a typed DeviceConfig
func (r *DeviceConfigResponse) ToDeviceConfig() (*DeviceConfig, error) {
	// Extract device type from raw data
	deviceType, ok := r.Raw["type"].(string)
	if !ok {
		return nil, fmt.Errorf("device type not found in response")
	}

	// Extract common fields for indexing
	config := &DeviceConfig{
		Type: deviceType,
	}

	// Extract ID
	if id, ok := r.Raw["id"].(string); ok {
		config.ID = id
	}

	// Extract Name
	if name, ok := r.Raw["name"].(string); ok {
		config.Name = name
	}

	// Extract MAC
	if mac, ok := r.Raw["mac"].(string); ok {
		config.MAC = mac
	}

	// Extract SiteID
	if siteID, ok := r.Raw["site_id"].(string); ok {
		config.SiteID = siteID
	}

	// Convert raw data to appropriate device type
	switch deviceType {
	case "ap":
		ap := &APConfig{}
		if err := ap.FromMap(r.Raw); err != nil {
			return nil, fmt.Errorf("failed to parse AP config: %w", err)
		}
		config.Data = ap

	case "switch":
		sw := &SwitchConfig{}
		if err := sw.FromMap(r.Raw); err != nil {
			return nil, fmt.Errorf("failed to parse Switch config: %w", err)
		}
		config.Data = sw

	case "gateway":
		gw := &GatewayConfig{}
		if err := gw.FromMap(r.Raw); err != nil {
			return nil, fmt.Errorf("failed to parse Gateway config: %w", err)
		}
		config.Data = gw

	default:
		return nil, fmt.Errorf("unknown device type: %s", deviceType)
	}

	return config, nil
}

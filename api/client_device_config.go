package api

import (
	"context"
	"fmt"
	"net/http"
)

// GetDeviceConfig retrieves the configuration for a specific device
func (c *mistClient) GetDeviceConfig(ctx context.Context, siteID, deviceID string) (*DeviceConfigResponse, error) {
	path := fmt.Sprintf("/sites/%s/devices/%s", siteID, deviceID)

	var rawResponse map[string]interface{}
	if err := c.do(ctx, http.MethodGet, path, nil, &rawResponse); err != nil {
		return nil, fmt.Errorf("failed to get device config: %w", err)
	}

	return &DeviceConfigResponse{Raw: rawResponse}, nil
}

// GetAPConfig retrieves the configuration for a specific AP device
func (c *mistClient) GetAPConfig(ctx context.Context, siteID, deviceID string) (*APConfig, error) {
	resp, err := c.GetDeviceConfig(ctx, siteID, deviceID)
	if err != nil {
		return nil, err
	}

	config, err := resp.ToDeviceConfig()
	if err != nil {
		return nil, err
	}

	if config.Type != "ap" {
		return nil, fmt.Errorf("device is not an AP, got type: %s", config.Type)
	}

	apConfig, ok := config.Data.(*APConfig)
	if !ok {
		return nil, fmt.Errorf("failed to cast to APConfig")
	}

	return apConfig, nil
}

// GetSwitchConfig retrieves the configuration for a specific Switch device
func (c *mistClient) GetSwitchConfig(ctx context.Context, siteID, deviceID string) (*SwitchConfig, error) {
	resp, err := c.GetDeviceConfig(ctx, siteID, deviceID)
	if err != nil {
		return nil, err
	}

	config, err := resp.ToDeviceConfig()
	if err != nil {
		return nil, err
	}

	if config.Type != "switch" {
		return nil, fmt.Errorf("device is not a Switch, got type: %s", config.Type)
	}

	switchConfig, ok := config.Data.(*SwitchConfig)
	if !ok {
		return nil, fmt.Errorf("failed to cast to SwitchConfig")
	}

	return switchConfig, nil
}

// GetGatewayConfig retrieves the configuration for a specific Gateway device
func (c *mistClient) GetGatewayConfig(ctx context.Context, siteID, deviceID string) (*GatewayConfig, error) {
	resp, err := c.GetDeviceConfig(ctx, siteID, deviceID)
	if err != nil {
		return nil, err
	}

	config, err := resp.ToDeviceConfig()
	if err != nil {
		return nil, err
	}

	if config.Type != "gateway" {
		return nil, fmt.Errorf("device is not a Gateway, got type: %s", config.Type)
	}

	gatewayConfig, ok := config.Data.(*GatewayConfig)
	if !ok {
		return nil, fmt.Errorf("failed to cast to GatewayConfig")
	}

	return gatewayConfig, nil
}

// BatchGetDeviceConfigs retrieves configurations for multiple devices
// Returns a map of deviceID -> DeviceConfig and a map of deviceID -> error for failures
func (c *mistClient) BatchGetDeviceConfigs(ctx context.Context, devices []DeviceInfo) (map[string]*DeviceConfig, map[string]error) {
	configs := make(map[string]*DeviceConfig)
	errors := make(map[string]error)

	// Process devices in batches to avoid overwhelming the API
	batchSize := 10
	for i := 0; i < len(devices); i += batchSize {
		end := i + batchSize
		if end > len(devices) {
			end = len(devices)
		}

		batch := devices[i:end]

		// Process batch concurrently
		type result struct {
			deviceID string
			config   *DeviceConfig
			err      error
		}

		results := make(chan result, len(batch))

		for _, device := range batch {
			go func(d DeviceInfo) {
				resp, err := c.GetDeviceConfig(ctx, d.SiteID, d.DeviceID)
				if err != nil {
					results <- result{deviceID: d.DeviceID, err: err}
					return
				}

				config, err := resp.ToDeviceConfig()
				if err != nil {
					results <- result{deviceID: d.DeviceID, err: err}
					return
				}

				results <- result{deviceID: d.DeviceID, config: config}
			}(device)
		}

		// Collect results
		for j := 0; j < len(batch); j++ {
			r := <-results
			if r.err != nil {
				errors[r.deviceID] = r.err
			} else {
				configs[r.deviceID] = r.config
			}
		}
	}

	return configs, errors
}

// DeviceInfo contains the minimal information needed to fetch a device config
type DeviceInfo struct {
	DeviceID string
	SiteID   string
	Name     string
	MAC      string
	Type     string
}

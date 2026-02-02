package mist

import (
	"context"
	"fmt"

	"github.com/ravinald/wifimgr/api"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// configsService implements vendors.ConfigsService for Mist.
type configsService struct {
	client api.Client
	orgID  string
}

// GetAPConfig returns the full configuration for an AP.
func (s *configsService) GetAPConfig(ctx context.Context, siteID, deviceID string) (*vendors.APConfig, error) {
	config, err := s.client.GetAPConfig(ctx, siteID, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get AP config: %w", err)
	}

	return convertAPConfigToVendor(config), nil
}

// GetSwitchConfig returns the full configuration for a switch.
func (s *configsService) GetSwitchConfig(ctx context.Context, siteID, deviceID string) (*vendors.SwitchConfig, error) {
	config, err := s.client.GetSwitchConfig(ctx, siteID, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get switch config: %w", err)
	}

	return convertSwitchConfigToVendor(config), nil
}

// GetGatewayConfig returns the full configuration for a gateway.
func (s *configsService) GetGatewayConfig(ctx context.Context, siteID, deviceID string) (*vendors.GatewayConfig, error) {
	config, err := s.client.GetGatewayConfig(ctx, siteID, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get gateway config: %w", err)
	}

	return convertGatewayConfigToVendor(config), nil
}

// Ensure configsService implements vendors.ConfigsService at compile time.
var _ vendors.ConfigsService = (*configsService)(nil)

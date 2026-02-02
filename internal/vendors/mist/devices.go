package mist

import (
	"context"
	"fmt"

	"github.com/ravinald/wifimgr/api"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// devicesService implements vendors.DevicesService for Mist.
type devicesService struct {
	client api.Client
	orgID  string
}

// List returns devices in a site, optionally filtered by type.
func (s *devicesService) List(ctx context.Context, siteID, deviceType string) ([]*vendors.DeviceInfo, error) {
	devices, err := s.client.GetDevices(ctx, siteID, deviceType)
	if err != nil {
		return nil, fmt.Errorf("failed to get devices: %w", err)
	}

	result := make([]*vendors.DeviceInfo, 0, len(devices))
	for i := range devices {
		info := convertUnifiedDeviceToDeviceInfo(&devices[i])
		if info != nil {
			result = append(result, info)
		}
	}

	return result, nil
}

// Get finds a device by its vendor-specific ID within a site.
func (s *devicesService) Get(ctx context.Context, siteID, deviceID string) (*vendors.DeviceInfo, error) {
	device, err := s.client.GetDeviceByID(ctx, siteID, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get device by ID %q: %w", deviceID, err)
	}

	return convertUnifiedDeviceToDeviceInfo(device), nil
}

// ByMAC finds a device by MAC address.
func (s *devicesService) ByMAC(ctx context.Context, mac string) (*vendors.DeviceInfo, error) {
	device, err := s.client.GetDeviceByMAC(ctx, mac)
	if err != nil {
		return nil, fmt.Errorf("failed to get device by MAC %q: %w", mac, err)
	}

	return convertUnifiedDeviceToDeviceInfo(device), nil
}

// Update modifies a device's configuration.
func (s *devicesService) Update(ctx context.Context, siteID, deviceID string, device *vendors.DeviceInfo) (*vendors.DeviceInfo, error) {
	mistDevice := convertDeviceInfoToUnified(device)
	if mistDevice == nil {
		return nil, fmt.Errorf("invalid device info")
	}

	updated, err := s.client.UpdateDevice(ctx, siteID, deviceID, mistDevice)
	if err != nil {
		return nil, fmt.Errorf("failed to update device: %w", err)
	}

	return convertUnifiedDeviceToDeviceInfo(updated), nil
}

// Rename changes the device name.
func (s *devicesService) Rename(ctx context.Context, siteID, deviceID, newName string) error {
	device := &api.UnifiedDevice{
		BaseDevice: api.BaseDevice{
			AdditionalConfig: make(map[string]interface{}),
		},
		DeviceConfig: make(map[string]interface{}),
	}
	device.Name = &newName

	_, err := s.client.UpdateDevice(ctx, siteID, deviceID, device)
	if err != nil {
		return fmt.Errorf("failed to rename device: %w", err)
	}

	return nil
}

// UpdateConfig applies a raw configuration map to a device.
func (s *devicesService) UpdateConfig(ctx context.Context, siteID, deviceID string, config map[string]interface{}) error {
	device := &api.UnifiedDevice{
		BaseDevice: api.BaseDevice{
			AdditionalConfig: config,
		},
		DeviceConfig: config,
	}

	// Extract name if present
	if name, ok := config["name"].(string); ok {
		device.Name = &name
	}

	// Extract notes if present
	if notes, ok := config["notes"].(string); ok {
		device.Notes = &notes
	}

	_, err := s.client.UpdateDevice(ctx, siteID, deviceID, device)
	if err != nil {
		return fmt.Errorf("failed to update device config: %w", err)
	}

	return nil
}

// Ensure devicesService implements vendors.DevicesService at compile time.
var _ vendors.DevicesService = (*devicesService)(nil)

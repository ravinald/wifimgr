package meraki

import (
	"context"
	"fmt"

	meraki "github.com/meraki/dashboard-api-go/v5/sdk"

	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// devicesService implements vendors.DevicesService for Meraki.
type devicesService struct {
	dashboard      *meraki.Client
	orgID          string
	rateLimiter    *RateLimiter
	retryConfig    *RetryConfig
	suppressOutput bool
}

// List returns devices in a network, optionally filtered by type.
func (s *devicesService) List(ctx context.Context, siteID, deviceType string) ([]*vendors.DeviceInfo, error) {
	logging.Debugf("[meraki] Listing devices for org %s, siteID=%q, deviceType=%q", s.orgID, siteID, deviceType)

	params := &meraki.GetOrganizationDevicesQueryParams{
		PerPage: -1, // Fetch all
	}

	if siteID != "" {
		params.NetworkIDs = []string{siteID}
	}

	if deviceType != "" {
		productType := mapDeviceTypeToProductType(deviceType)
		params.ProductTypes = []string{productType}
	}

	retryState := NewRetryState(s.retryConfig)
	var devices *meraki.ResponseOrganizationsGetOrganizationDevices
	var err error

	for {
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return nil, fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		var resp *meraki.ResponseOrganizationsGetOrganizationDevices
		if s.suppressOutput {
			restore := suppressStdout()
			resp, _, err = s.dashboard.Organizations.GetOrganizationDevices(s.orgID, params)
			restore()
		} else {
			resp, _, err = s.dashboard.Organizations.GetOrganizationDevices(s.orgID, params)
		}
		devices = resp

		if err == nil {
			break
		}

		if !retryState.ShouldRetry(err) {
			logging.Debugf("[meraki] Failed to get devices: %v", err)
			return nil, fmt.Errorf("failed to get devices: %w", err)
		}

		if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
			return nil, fmt.Errorf("retry wait failed: %w", waitErr)
		}
	}

	if devices == nil {
		logging.Debug("[meraki] No devices returned")
		return []*vendors.DeviceInfo{}, nil
	}

	infos := make([]*vendors.DeviceInfo, 0, len(*devices))
	for i := range *devices {
		info := convertDeviceToDeviceInfo(&(*devices)[i])
		if info != nil {
			infos = append(infos, info)
		}
	}

	logging.Debugf("[meraki] Listed %d devices", len(infos))
	return infos, nil
}

// ByMAC finds a device by MAC address.
func (s *devicesService) ByMAC(ctx context.Context, mac string) (*vendors.DeviceInfo, error) {
	params := &meraki.GetOrganizationDevicesQueryParams{
		PerPage: -1,
	}

	retryState := NewRetryState(s.retryConfig)
	var devices *meraki.ResponseOrganizationsGetOrganizationDevices
	var err error

	for {
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return nil, fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		devices, _, err = s.dashboard.Organizations.GetOrganizationDevices(s.orgID, params)
		if err == nil {
			break
		}

		if !retryState.ShouldRetry(err) {
			return nil, fmt.Errorf("failed to get devices: %w", err)
		}

		if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
			return nil, fmt.Errorf("retry wait failed: %w", waitErr)
		}
	}

	if devices == nil {
		return nil, &vendors.DeviceNotFoundError{Identifier: mac}
	}

	normalizedMAC := normalizeMAC(mac)
	for i := range *devices {
		if normalizeMAC((*devices)[i].Mac) == normalizedMAC {
			return convertDeviceToDeviceInfo(&(*devices)[i]), nil
		}
	}

	return nil, &vendors.DeviceNotFoundError{Identifier: mac}
}

// Get finds a device by its serial (Meraki uses serial as device ID).
func (s *devicesService) Get(ctx context.Context, _, deviceID string) (*vendors.DeviceInfo, error) {
	// In Meraki, deviceID is the serial number
	retryState := NewRetryState(s.retryConfig)
	var device *meraki.ResponseDevicesGetDevice
	var err error

	for {
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return nil, fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		device, _, err = s.dashboard.Devices.GetDevice(deviceID)
		if err == nil {
			break
		}

		if !retryState.ShouldRetry(err) {
			return nil, fmt.Errorf("failed to get device %s: %w", deviceID, err)
		}

		if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
			return nil, fmt.Errorf("retry wait failed: %w", waitErr)
		}
	}

	if device == nil {
		return nil, &vendors.DeviceNotFoundError{Identifier: deviceID}
	}

	return &vendors.DeviceInfo{
		ID:           device.Serial,
		MAC:          normalizeMAC(device.Mac),
		Serial:       device.Serial,
		Model:        device.Model,
		Name:         device.Name,
		SiteID:       device.NetworkID,
		Notes:        device.Notes,
		IP:           device.LanIP,
		Version:      device.Firmware,
		SourceVendor: "meraki",
	}, nil
}

// Update modifies a device's configuration.
func (s *devicesService) Update(ctx context.Context, _, deviceID string, device *vendors.DeviceInfo) (*vendors.DeviceInfo, error) {
	if device == nil {
		return nil, fmt.Errorf("device cannot be nil")
	}

	// deviceID in Meraki is the serial number
	request := &meraki.RequestDevicesUpdateDevice{
		Name:  device.Name,
		Notes: device.Notes,
	}

	// Handle optional fields
	if device.Latitude != 0 || device.Longitude != 0 {
		request.Lat = &device.Latitude
		request.Lng = &device.Longitude
	}

	retryState := NewRetryState(s.retryConfig)
	var updatedDevice *meraki.ResponseDevicesUpdateDevice
	var err error

	for {
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return nil, fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		updatedDevice, _, err = s.dashboard.Devices.UpdateDevice(deviceID, request)
		if err == nil {
			break
		}

		if !retryState.ShouldRetry(err) {
			return nil, fmt.Errorf("failed to update device %s: %w", deviceID, err)
		}

		if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
			return nil, fmt.Errorf("retry wait failed: %w", waitErr)
		}
	}

	return &vendors.DeviceInfo{
		ID:           updatedDevice.Serial,
		MAC:          normalizeMAC(updatedDevice.Mac),
		Serial:       updatedDevice.Serial,
		Model:        updatedDevice.Model,
		Name:         updatedDevice.Name,
		SiteID:       updatedDevice.NetworkID,
		Notes:        updatedDevice.Notes,
		IP:           updatedDevice.LanIP,
		Version:      updatedDevice.Firmware,
		SourceVendor: "meraki",
	}, nil
}

// Rename changes the device name.
func (s *devicesService) Rename(ctx context.Context, _, deviceID, newName string) error {
	// deviceID in Meraki is the serial number
	request := &meraki.RequestDevicesUpdateDevice{
		Name: newName,
	}

	retryState := NewRetryState(s.retryConfig)

	for {
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		_, _, err := s.dashboard.Devices.UpdateDevice(deviceID, request)
		if err == nil {
			return nil
		}

		if !retryState.ShouldRetry(err) {
			return fmt.Errorf("failed to rename device %s: %w", deviceID, err)
		}

		if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
			return fmt.Errorf("retry wait failed: %w", waitErr)
		}
	}
}

// UpdateConfig applies a raw configuration map to a device.
func (s *devicesService) UpdateConfig(ctx context.Context, _, deviceID string, config map[string]interface{}) error {
	// deviceID in Meraki is the serial number
	request := &meraki.RequestDevicesUpdateDevice{}

	// Map common fields from config
	if name, ok := config["name"].(string); ok {
		request.Name = name
	}
	if notes, ok := config["notes"].(string); ok {
		request.Notes = notes
	}
	if lat, ok := config["lat"].(float64); ok {
		request.Lat = &lat
	}
	if lng, ok := config["lng"].(float64); ok {
		request.Lng = &lng
	}
	if address, ok := config["address"].(string); ok {
		request.Address = address
	}
	if floorPlanID, ok := config["floor_plan_id"].(string); ok {
		request.FloorPlanID = floorPlanID
	}

	retryState := NewRetryState(s.retryConfig)

	for {
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		_, _, err := s.dashboard.Devices.UpdateDevice(deviceID, request)
		if err == nil {
			return nil
		}

		if !retryState.ShouldRetry(err) {
			return fmt.Errorf("failed to update device config %s: %w", deviceID, err)
		}

		if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
			return fmt.Errorf("retry wait failed: %w", waitErr)
		}
	}
}

// Ensure devicesService implements vendors.DevicesService at compile time.
var _ vendors.DevicesService = (*devicesService)(nil)

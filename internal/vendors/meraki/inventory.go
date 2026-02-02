package meraki

import (
	"context"
	"fmt"

	meraki "github.com/meraki/dashboard-api-go/v5/sdk"

	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// inventoryService implements vendors.InventoryService for Meraki.
type inventoryService struct {
	dashboard      *meraki.Client
	orgID          string
	rateLimiter    *RateLimiter
	retryConfig    *RetryConfig
	suppressOutput bool
}

// List returns inventory items, optionally filtered by device type.
func (s *inventoryService) List(ctx context.Context, deviceType string) ([]*vendors.InventoryItem, error) {
	logging.Debugf("[meraki] Fetching inventory for org %s, deviceType=%q", s.orgID, deviceType)

	params := &meraki.GetOrganizationDevicesQueryParams{
		PerPage: -1, // Fetch all
	}

	// Map wifimgr device types to Meraki product types
	if deviceType != "" {
		productType := mapDeviceTypeToProductType(deviceType)
		params.ProductTypes = []string{productType}
		logging.Debugf("[meraki] Filtering by product type: %s", productType)
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
		return []*vendors.InventoryItem{}, nil
	}

	items := make([]*vendors.InventoryItem, 0, len(*devices))
	for i := range *devices {
		item := convertDeviceToInventoryItem(&(*devices)[i])
		if item != nil {
			items = append(items, item)
		}
	}

	logging.Debugf("[meraki] Fetched %d devices", len(items))
	return items, nil
}

// ByMAC finds an inventory item by MAC address.
func (s *inventoryService) ByMAC(ctx context.Context, mac string) (*vendors.InventoryItem, error) {
	// Meraki API doesn't support direct MAC lookup, search all devices
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
			return convertDeviceToInventoryItem(&(*devices)[i]), nil
		}
	}

	return nil, &vendors.DeviceNotFoundError{Identifier: mac}
}

// BySerial finds an inventory item by serial number.
func (s *inventoryService) BySerial(ctx context.Context, serial string) (*vendors.InventoryItem, error) {
	retryState := NewRetryState(s.retryConfig)
	var device *meraki.ResponseDevicesGetDevice
	var err error

	for {
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return nil, fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		device, _, err = s.dashboard.Devices.GetDevice(serial)
		if err == nil {
			break
		}

		if !retryState.ShouldRetry(err) {
			return nil, fmt.Errorf("failed to get device %s: %w", serial, err)
		}

		if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
			return nil, fmt.Errorf("retry wait failed: %w", waitErr)
		}
	}

	if device == nil {
		return nil, &vendors.DeviceNotFoundError{Identifier: serial}
	}

	return &vendors.InventoryItem{
		MAC:          normalizeMAC(device.Mac),
		Serial:       device.Serial,
		Model:        device.Model,
		Name:         device.Name,
		SiteID:       device.NetworkID,
		Claimed:      true,
		SourceVendor: "meraki",
	}, nil
}

// Claim adds devices to the organization using claim codes.
// Note: Meraki uses serial numbers for claiming, not claim codes like Mist.
func (s *inventoryService) Claim(ctx context.Context, claimCodes []string) ([]*vendors.InventoryItem, error) {
	// Meraki claims by serial number
	request := &meraki.RequestOrganizationsClaimIntoOrganization{
		Serials: claimCodes, // In Meraki, "claim codes" are actually serials
	}

	retryState := NewRetryState(s.retryConfig)

	for {
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return nil, fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		_, _, err := s.dashboard.Organizations.ClaimIntoOrganization(s.orgID, request)
		if err == nil {
			break
		}

		if !retryState.ShouldRetry(err) {
			return nil, fmt.Errorf("failed to claim devices: %w", err)
		}

		if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
			return nil, fmt.Errorf("retry wait failed: %w", waitErr)
		}
	}

	// Fetch the newly claimed devices
	var items []*vendors.InventoryItem
	for _, serial := range claimCodes {
		item, err := s.BySerial(ctx, serial)
		if err == nil {
			items = append(items, item)
		}
	}

	return items, nil
}

// Release removes devices from the organization.
func (s *inventoryService) Release(ctx context.Context, serials []string) error {
	// Meraki removes devices by unclaiming them from their networks
	// Note: This removes devices from their network, doesn't unclaim from org
	for _, serial := range serials {
		// Get device to find its network
		retryState := NewRetryState(s.retryConfig)
		var device *meraki.ResponseDevicesGetDevice
		var err error

		for {
			if s.rateLimiter != nil {
				if err := s.rateLimiter.Acquire(ctx); err != nil {
					return fmt.Errorf("rate limit acquire failed: %w", err)
				}
			}

			device, _, err = s.dashboard.Devices.GetDevice(serial)
			if err == nil {
				break
			}

			if !retryState.ShouldRetry(err) {
				return fmt.Errorf("failed to get device %s: %w", serial, err)
			}

			if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
				return fmt.Errorf("retry wait failed: %w", waitErr)
			}
		}

		if device.NetworkID != "" {
			request := &meraki.RequestNetworksRemoveNetworkDevices{
				Serial: serial,
			}

			retryState = NewRetryState(s.retryConfig)
			for {
				if s.rateLimiter != nil {
					if err := s.rateLimiter.Acquire(ctx); err != nil {
						return fmt.Errorf("rate limit acquire failed: %w", err)
					}
				}

				_, err = s.dashboard.Networks.RemoveNetworkDevices(device.NetworkID, request)
				if err == nil {
					break
				}

				if !retryState.ShouldRetry(err) {
					return fmt.Errorf("failed to release device %s: %w", serial, err)
				}

				if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
					return fmt.Errorf("retry wait failed: %w", waitErr)
				}
			}
		}
	}
	return nil
}

// AssignToSite assigns devices to a network.
func (s *inventoryService) AssignToSite(ctx context.Context, siteID string, macs []string) error {
	// Meraki assigns devices by serial, need to look up serials from MACs
	for _, mac := range macs {
		item, err := s.ByMAC(ctx, mac)
		if err != nil {
			return fmt.Errorf("device %s not found: %w", mac, err)
		}

		request := &meraki.RequestNetworksClaimNetworkDevices{
			Serials: []string{item.Serial},
		}

		retryState := NewRetryState(s.retryConfig)
		for {
			if s.rateLimiter != nil {
				if err := s.rateLimiter.Acquire(ctx); err != nil {
					return fmt.Errorf("rate limit acquire failed: %w", err)
				}
			}

			_, _, err = s.dashboard.Networks.ClaimNetworkDevices(siteID, request, nil)
			if err == nil {
				break
			}

			if !retryState.ShouldRetry(err) {
				return fmt.Errorf("failed to assign device %s to network %s: %w", mac, siteID, err)
			}

			if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
				return fmt.Errorf("retry wait failed: %w", waitErr)
			}
		}
	}

	return nil
}

// UnassignFromSite removes devices from their assigned network.
func (s *inventoryService) UnassignFromSite(ctx context.Context, macs []string) error {
	for _, mac := range macs {
		item, err := s.ByMAC(ctx, mac)
		if err != nil {
			return fmt.Errorf("device %s not found: %w", mac, err)
		}

		if item.SiteID == "" {
			continue // Device not assigned to any network
		}

		request := &meraki.RequestNetworksRemoveNetworkDevices{
			Serial: item.Serial,
		}

		retryState := NewRetryState(s.retryConfig)
		for {
			if s.rateLimiter != nil {
				if err := s.rateLimiter.Acquire(ctx); err != nil {
					return fmt.Errorf("rate limit acquire failed: %w", err)
				}
			}

			_, err = s.dashboard.Networks.RemoveNetworkDevices(item.SiteID, request)
			if err == nil {
				break
			}

			if !retryState.ShouldRetry(err) {
				return fmt.Errorf("failed to unassign device %s: %w", mac, err)
			}

			if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
				return fmt.Errorf("retry wait failed: %w", waitErr)
			}
		}
	}

	return nil
}

// Ensure inventoryService implements vendors.InventoryService at compile time.
var _ vendors.InventoryService = (*inventoryService)(nil)

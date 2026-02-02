package meraki

import (
	"context"
	"fmt"
	"sync"

	meraki "github.com/meraki/dashboard-api-go/v5/sdk"

	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// configsService implements vendors.ConfigsService for Meraki.
type configsService struct {
	dashboard      *meraki.Client
	orgID          string
	rateLimiter    *RateLimiter
	retryConfig    *RetryConfig
	suppressOutput bool

	// Cache for network BLE status to avoid redundant API calls
	networkBLECache map[string]bool // networkID -> BLE enabled in unique mode
	networkBLEMu    sync.RWMutex
}

// GetAPConfig returns the configuration for a wireless AP.
func (s *configsService) GetAPConfig(ctx context.Context, siteID, deviceID string) (*vendors.APConfig, error) {
	logging.Debugf("[meraki] Getting AP config for device %s in network %s", deviceID, siteID)

	// deviceID in Meraki is the serial number
	retryState := NewRetryState(s.retryConfig)
	var device *meraki.ResponseDevicesGetDevice
	var err error

	for {
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return nil, fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		if s.suppressOutput {
			restore := suppressStdout()
			device, _, err = s.dashboard.Devices.GetDevice(deviceID)
			restore()
		} else {
			device, _, err = s.dashboard.Devices.GetDevice(deviceID)
		}
		if err == nil {
			break
		}

		if !retryState.ShouldRetry(err) {
			logging.Debugf("[meraki] Failed to get device %s: %v", deviceID, err)
			return nil, fmt.Errorf("failed to get device %s: %w", deviceID, err)
		}

		if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
			return nil, fmt.Errorf("retry wait failed: %w", waitErr)
		}
	}

	if device == nil {
		return nil, &vendors.DeviceNotFoundError{Identifier: deviceID}
	}

	config := &vendors.APConfig{
		ID:           device.Serial,
		Name:         device.Name,
		MAC:          normalizeMAC(device.Mac),
		SiteID:       device.NetworkID,
		Config:       make(map[string]interface{}),
		SourceVendor: "meraki",
	}

	// Get wireless radio settings if available (optional - may not be supported by device)
	if s.rateLimiter != nil {
		if err := s.rateLimiter.Acquire(ctx); err != nil {
			return nil, fmt.Errorf("rate limit acquire failed: %w", err)
		}
	}
	radioSettings, _, radioErr := s.dashboard.Wireless.GetDeviceWirelessRadioSettings(deviceID)
	if radioErr != nil {
		logging.Debugf("[meraki] Radio settings not available for device %s (expected for some devices): %v", deviceID, radioErr)
	} else if radioSettings != nil {
		config.Config["radio_settings"] = radioSettings
	}

	// Get bluetooth settings if available (optional - requires BLE to be enabled on network in unique mode)
	// Check cached network BLE status to avoid redundant API calls
	if s.isNetworkBLEEnabled(ctx, siteID) {
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return nil, fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}
		btSettings, _, btErr := s.dashboard.Wireless.GetDeviceWirelessBluetoothSettings(deviceID)
		if btErr != nil {
			logging.Debugf("[meraki] Bluetooth settings not available for device %s: %v", deviceID, btErr)
		} else if btSettings != nil {
			config.Config["bluetooth_settings"] = btSettings
		}
	} else {
		logging.Debugf("[meraki] Skipping device BLE settings for %s (network BLE not enabled in unique mode)", deviceID)
	}

	return config, nil
}

// GetSwitchConfig returns the configuration for a switch.
func (s *configsService) GetSwitchConfig(ctx context.Context, siteID, deviceID string) (*vendors.SwitchConfig, error) {
	retryState := NewRetryState(s.retryConfig)
	var device *meraki.ResponseDevicesGetDevice
	var err error

	for {
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return nil, fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		if s.suppressOutput {
			restore := suppressStdout()
			device, _, err = s.dashboard.Devices.GetDevice(deviceID)
			restore()
		} else {
			device, _, err = s.dashboard.Devices.GetDevice(deviceID)
		}
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

	config := &vendors.SwitchConfig{
		ID:           device.Serial,
		Name:         device.Name,
		MAC:          normalizeMAC(device.Mac),
		SiteID:       device.NetworkID,
		Config:       make(map[string]interface{}),
		SourceVendor: "meraki",
	}

	// Get switch ports if available (optional - may not be supported by all switch models)
	if s.rateLimiter != nil {
		if err := s.rateLimiter.Acquire(ctx); err != nil {
			return nil, fmt.Errorf("rate limit acquire failed: %w", err)
		}
	}
	ports, _, portsErr := s.dashboard.Switch.GetDeviceSwitchPorts(deviceID)
	if portsErr != nil {
		logging.Debugf("[meraki] Switch ports not available for device %s: %v", deviceID, portsErr)
	} else if ports != nil {
		config.Config["ports"] = ports
	}

	// Get switch routing interfaces if available (optional - requires L3 routing)
	if s.rateLimiter != nil {
		if err := s.rateLimiter.Acquire(ctx); err != nil {
			return nil, fmt.Errorf("rate limit acquire failed: %w", err)
		}
	}
	interfaces, _, ifaceErr := s.dashboard.Switch.GetDeviceSwitchRoutingInterfaces(deviceID, nil)
	if ifaceErr != nil {
		logging.Debugf("[meraki] Routing interfaces not available for device %s: %v", deviceID, ifaceErr)
	} else if interfaces != nil {
		config.Config["routing_interfaces"] = interfaces
	}

	return config, nil
}

// GetGatewayConfig returns the configuration for a gateway/appliance.
func (s *configsService) GetGatewayConfig(ctx context.Context, siteID, deviceID string) (*vendors.GatewayConfig, error) {
	retryState := NewRetryState(s.retryConfig)
	var device *meraki.ResponseDevicesGetDevice
	var err error

	for {
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return nil, fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		if s.suppressOutput {
			restore := suppressStdout()
			device, _, err = s.dashboard.Devices.GetDevice(deviceID)
			restore()
		} else {
			device, _, err = s.dashboard.Devices.GetDevice(deviceID)
		}
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

	config := &vendors.GatewayConfig{
		ID:           device.Serial,
		Name:         device.Name,
		MAC:          normalizeMAC(device.Mac),
		SiteID:       device.NetworkID,
		Config:       make(map[string]interface{}),
		SourceVendor: "meraki",
	}

	// Get appliance uplinks if available (optional - may not be supported by all appliance models)
	if s.rateLimiter != nil {
		if err := s.rateLimiter.Acquire(ctx); err != nil {
			return nil, fmt.Errorf("rate limit acquire failed: %w", err)
		}
	}
	uplinks, _, uplinksErr := s.dashboard.Appliance.GetDeviceApplianceUplinksSettings(deviceID)
	if uplinksErr != nil {
		logging.Debugf("[meraki] Uplinks settings not available for device %s: %v", deviceID, uplinksErr)
	} else if uplinks != nil {
		config.Config["uplinks"] = uplinks
	}

	return config, nil
}

// isNetworkBLEEnabled checks if BLE advertising is enabled in unique mode for a network.
// Results are cached to avoid redundant API calls when fetching configs for multiple devices.
func (s *configsService) isNetworkBLEEnabled(ctx context.Context, networkID string) bool {
	// Check cache first
	s.networkBLEMu.RLock()
	if s.networkBLECache != nil {
		if enabled, ok := s.networkBLECache[networkID]; ok {
			s.networkBLEMu.RUnlock()
			return enabled
		}
	}
	s.networkBLEMu.RUnlock()

	// Fetch from API
	if s.rateLimiter != nil {
		if err := s.rateLimiter.Acquire(ctx); err != nil {
			logging.Debugf("[meraki] Rate limit acquire failed for network BLE check: %v", err)
			return false
		}
	}

	networkBLE, _, err := s.dashboard.Wireless.GetNetworkWirelessBluetoothSettings(networkID)
	enabled := err == nil && networkBLE != nil &&
		networkBLE.AdvertisingEnabled != nil && *networkBLE.AdvertisingEnabled &&
		networkBLE.MajorMinorAssignmentMode == "Unique"

	// Cache the result
	s.networkBLEMu.Lock()
	if s.networkBLECache == nil {
		s.networkBLECache = make(map[string]bool)
	}
	s.networkBLECache[networkID] = enabled
	s.networkBLEMu.Unlock()

	if !enabled {
		logging.Debugf("[meraki] Network %s does not have BLE advertising enabled in unique mode", networkID)
	}

	return enabled
}

// Ensure configsService implements vendors.ConfigsService at compile time.
var _ vendors.ConfigsService = (*configsService)(nil)

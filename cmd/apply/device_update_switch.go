package apply

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ravinald/jsondiff/pkg/jsondiff"
	"github.com/spf13/viper"

	"github.com/ravinald/wifimgr/api"
	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/macaddr"
	"github.com/ravinald/wifimgr/internal/symbols"
)

// SwitchUpdater implements DeviceUpdater for Switches
type SwitchUpdater struct {
	*BaseDeviceUpdater
	batchLoader *DeviceBatchLoader // Reusable batch loader for device lookups
}

// NewSwitchUpdater creates a new Switch updater
func NewSwitchUpdater() *SwitchUpdater {
	return &SwitchUpdater{
		BaseDeviceUpdater: NewBaseDeviceUpdater("switch"),
	}
}

// GetConfiguredDevices extracts Switch MAC addresses from site configuration
func (s *SwitchUpdater) GetConfiguredDevices(siteConfig SiteConfig) []string {
	configuredSwitches := make([]string, 0, len(siteConfig.Devices.Switches))

	// Switches is a map with MAC as key
	for mac := range siteConfig.Devices.Switches {
		normalizedMAC := macaddr.NormalizeOrEmpty(mac)
		if normalizedMAC != "" {
			configuredSwitches = append(configuredSwitches, normalizedMAC)
		}
	}

	return configuredSwitches
}

// FindDevicesInventoryStatus checks Switch status in inventory and cache
func (s *SwitchUpdater) FindDevicesInventoryStatus(client api.Client, cfg *config.Config, configuredSwitches []string) ([]DeviceInventoryStatus, error) {
	ctx := context.Background()

	// Create inventory checker once for reuse
	inventoryChecker, err := NewInventoryChecker(ctx, client, cfg, s.deviceType)
	if err != nil {
		return nil, fmt.Errorf("failed to create inventory checker: %w", err)
	}

	// Store it for later use in UpdateDeviceConfigurations (using base method)
	s.SetInventoryChecker(inventoryChecker)

	statusList := make([]DeviceInventoryStatus, 0, len(configuredSwitches))

	for _, mac := range configuredSwitches {
		status := DeviceInventoryStatus{
			MAC:             mac,
			InInventory:     inventoryChecker.IsInLocalInventory(mac),
			InCache:         inventoryChecker.IsInAPIInventory(mac),
			CurrentSiteName: "",
			CurrentSiteID:   "",
		}

		// Get current site assignment using O(1) lookup from cached inventory
		if status.InCache {
			siteID, siteName, _ := inventoryChecker.GetSiteAssignment(mac)
			status.CurrentSiteID = siteID
			status.CurrentSiteName = siteName
		}

		statusList = append(statusList, status)
	}

	logging.Debugf("Inventory status check completed for %d switches", len(statusList))
	return statusList, nil
}

// FindDevicesToUpdate identifies Switches that need configuration updates
func (s *SwitchUpdater) FindDevicesToUpdate(ctx context.Context, client api.Client, _ *config.Config, siteConfig SiteConfig, configuredSwitches []string, siteID string, apiLabel string) ([]string, error) {
	// Create batch loader for efficient device lookups and store for reuse in UpdateDeviceConfigurations
	batchLoader, err := NewDeviceBatchLoader(ctx, client, siteID, s.deviceType)
	if err != nil {
		return nil, fmt.Errorf("failed to create batch loader: %w", err)
	}
	s.batchLoader = batchLoader // Store for reuse - eliminates duplicate API call
	logging.Debugf("Batch loader created with %d devices for comparison", batchLoader.GetDeviceCount())

	// Get managed keys for switch devices from the API-specific config
	managedKeys := getManagedKeysForDevice(apiLabel, "switch")

	switchesToUpdate := make([]string, 0)

	for _, mac := range configuredSwitches {
		// Get desired config from site configuration
		desiredConfig, found := s.GetDeviceConfigFromSite(siteConfig, mac)
		if !found {
			logging.Warnf("Switch %s is in the list but not found in site configuration", mac)
			continue
		}

		// Expand template references (device_template)
		expandedConfig, err := expandDeviceConfigWithTemplates(desiredConfig, siteConfig)
		if err != nil {
			logging.Warnf("Error expanding templates for Switch %s: %v - using unexpanded config", mac, err)
		} else {
			desiredConfig = expandedConfig
		}

		// Get current device state
		device, err := batchLoader.GetDeviceByMAC(mac)
		if err != nil {
			logging.Warnf("Error getting device by MAC %s: %v", mac, err)
			fmt.Printf("-> switch %s: not found in API cache for this site (skipping diff)\n", mac)
			continue
		}

		// Get current config
		currentConfig := device.ToConfigMap()

		// Compare configurations using managed keys
		needsUpdate := compareDeviceConfigsWithManagedKeys(currentConfig, desiredConfig, managedKeys)
		logging.Debugf("Switch %s - Current config: %+v", mac, currentConfig)
		logging.Debugf("Switch %s - Desired config: %+v", mac, desiredConfig)
		logging.Debugf("Switch %s - Needs update: %v", mac, needsUpdate)

		if needsUpdate {
			switchesToUpdate = append(switchesToUpdate, mac)
			logging.Debugf("Switch %s needs configuration update", mac)

			// Show JSON diff if in diff mode or debug enabled
			if viper.GetBool("show_diff") || viper.GetString("logging.level") == "debug" {
				// Get site name from siteConfig
				siteName := ""
				if siteNameVal, ok := siteConfig.SiteConfig["name"]; ok {
					if nameStr, ok := siteNameVal.(string); ok {
						siteName = nameStr
					}
				}
				if siteName == "" {
					// Fallback to site ID if name not found
					siteName = siteID
				}
				showSwitchConfigDiffWithManagedKeys(mac, currentConfig, desiredConfig, managedKeys, siteName)
			}
		} else {
			logging.Debugf("Switch %s configuration is up to date", mac)
		}
	}

	return switchesToUpdate, nil
}

// UpdateDeviceConfigurations applies configuration updates to Switches
func (s *SwitchUpdater) UpdateDeviceConfigurations(ctx context.Context, client api.Client, cfg *config.Config, siteConfig SiteConfig, macs []string, siteID string, apiLabel string) error {
	logging.Infof("Updating configuration for %d switches in site %s", len(macs), siteID)

	// Reuse existing batch loader if available (from FindDevicesToUpdate), otherwise create new one
	batchLoader := s.batchLoader
	if batchLoader == nil {
		var err error
		batchLoader, err = NewDeviceBatchLoader(ctx, client, siteID, s.deviceType)
		if err != nil {
			return fmt.Errorf("failed to create batch loader: %w", err)
		}
		logging.Debugf("Batch loader created with %d devices", batchLoader.GetDeviceCount())
	} else {
		logging.Debugf("Reusing batch loader with %d devices (avoiding duplicate API call)", batchLoader.GetDeviceCount())
	}

	// Use existing inventory checker if available, otherwise create new one
	inventoryChecker := s.GetInventoryChecker()
	if inventoryChecker == nil {
		var inventoryErr error
		inventoryChecker, inventoryErr = NewInventoryChecker(ctx, client, cfg, s.deviceType)
		if inventoryErr != nil {
			logging.Warnf("Could not create inventory checker for safety validation: %v", inventoryErr)
		}
	}

	// Additional safety check - verify all devices are in inventory
	if inventoryChecker != nil {
		for _, mac := range macs {
			if !inventoryChecker.IsInInventory(mac) {
				logging.Errorf("SAFETY CHECK FAILED: Device %s is not in inventory - refusing to update", mac)
				return fmt.Errorf("device %s is not in inventory - refusing to update for safety", mac)
			}
		}
	}

	var siteName string
	if nameVal, ok := siteConfig.SiteConfig["name"]; ok {
		if name, ok := nameVal.(string); ok {
			siteName = name
		}
	}
	if siteName == "" {
		siteName = siteID
	}

	var failedDevices []string
	successCount := 0

	// Build profile name-to-ID map ONCE before processing devices
	profileNameToID, err := buildProfileNameToIDMap(ctx, client, cfg.API.Credentials.OrgID)
	if err != nil {
		logging.Warnf("Could not build profile name map: %v - profile translations may fail", err)
		profileNameToID = make(map[string]string)
	}

	for _, mac := range macs {
		switchConfig, found := s.GetDeviceConfigFromSite(siteConfig, mac)
		if !found {
			logging.Warnf("Switch %s is in the list to update but not found in site configuration", mac)
			continue
		}

		// Expand template references (device_template)
		expandedConfig, err := expandDeviceConfigWithTemplates(switchConfig, siteConfig)
		if err != nil {
			logging.Warnf("Error expanding templates for Switch %s: %v - using unexpanded config", mac, err)
		} else {
			switchConfig = expandedConfig
		}

		device, err := batchLoader.GetDeviceByMAC(mac)
		if err != nil {
			logging.Warnf("Error getting device by MAC %s: %v", mac, err)
			continue
		}

		if device.ID == nil {
			logging.Warnf("Device %s has no ID, skipping configuration update", mac)
			continue
		}

		deviceID := *device.ID
		deviceName := "<unnamed>"
		if device.Name != nil {
			deviceName = *device.Name
		}

		logging.Debugf("Updating configuration for Switch %s (ID: %s, Name: %s)", mac, deviceID, deviceName)

		updatedDevice := *device

		// Handle _name suffix translations using cached profile map (O(1) lookup)
		translatedConfig := translateNameFieldsWithCache(switchConfig, profileNameToID)

		// Filter config to only include managed keys if configured
		managedKeys := getManagedKeysForDevice(apiLabel, "switch")
		var filteredConfig map[string]any
		if len(managedKeys) > 0 {
			filteredConfig = filterConfigByManagedKeys(translatedConfig, managedKeys)
			logging.Debugf("Filtered config to %d managed keys for Switch %s", len(filteredConfig), mac)
		} else {
			filteredConfig = translatedConfig
		}

		if err := updatedDevice.FromConfigMap(filteredConfig); err != nil {
			logging.Errorf("Error applying configuration to device %s using FromConfigMap: %v", mac, err)
			failedDevices = append(failedDevices, mac)
			continue
		}

		if updatedDevice.SiteID == nil || *updatedDevice.SiteID != siteID {
			updatedDevice.SiteID = &siteID
			logging.Debugf("Preserved site ID %s for device %s during configuration update", siteID, mac)
		}

		updatedResult, err := client.UpdateDevice(ctx, siteID, deviceID, &updatedDevice)
		if err != nil {
			logging.Errorf("Error updating Switch %s configuration via API: %v", mac, err)
			failedDevices = append(failedDevices, mac)
			continue
		}

		if updatedResult != nil {
			logging.Infof("%s Successfully updated configuration for Switch %s (Name: %s)", symbols.SuccessPrefix(), mac, deviceName)
			successCount++

			configFields := len(filteredConfig)
			logging.Debugf("Applied %d configuration fields to Switch %s", configFields, mac)

			for key := range filteredConfig {
				if key != "magic" {
					logging.Debugf("  - %s: configured", key)
				}
			}
		}
	}

	if len(failedDevices) > 0 {
		logging.Errorf("Configuration failed for %d out of %d devices", len(failedDevices), len(macs))
		for _, failedMAC := range failedDevices {
			logging.Errorf("  - Failed device: %s", failedMAC)
		}
		logging.Errorf("Tip: To restore previous config, use: apply rollback %s", siteName)
		return fmt.Errorf("configuration failed for %d out of %d devices", len(failedDevices), len(macs))
	}

	logging.Infof("%s Completed configuration updates for %d switches in site %s (%d successful, %d failed)",
		symbols.SuccessPrefix(), len(macs), siteID, successCount, len(failedDevices))
	return nil
}

// GetDeviceConfigFromSite extracts Switch-specific config from site configuration
func (s *SwitchUpdater) GetDeviceConfigFromSite(siteConfig SiteConfig, mac string) (map[string]any, bool) {
	normalizedTargetMAC := macaddr.NormalizeOrEmpty(mac)

	// Try direct lookup first (in case keys match exactly)
	if switchConfig, ok := siteConfig.Devices.Switches[normalizedTargetMAC]; ok {
		return switchConfig, true
	}
	if switchConfig, ok := siteConfig.Devices.Switches[mac]; ok {
		return switchConfig, true
	}

	// Iterate and compare normalized versions (handles colon-separated keys)
	for configMac, switchConfig := range siteConfig.Devices.Switches {
		if macaddr.NormalizeOrEmpty(configMac) == normalizedTargetMAC {
			return switchConfig, true
		}
	}

	return nil, false
}

// showSwitchConfigDiffWithManagedKeys displays a colored JSON diff with managed keys highlighted
func showSwitchConfigDiffWithManagedKeys(mac string, currentConfig, desiredConfig map[string]any, managedKeys []string, siteName string) {
	// Filter out status fields that shouldn't be compared
	filteredCurrent := filterStatusFields(currentConfig)
	filteredDesired := filterStatusFields(desiredConfig)

	// Remove MAC from desired config for comparison
	delete(filteredDesired, "mac")

	// Pre-filter configs to only include managed keys BEFORE diffing
	// This ensures only managed fields appear in the diff output
	if len(managedKeys) > 0 {
		filteredCurrent = filterConfigByManagedKeys(filteredCurrent, managedKeys)
		filteredDesired = filterConfigByManagedKeys(filteredDesired, managedKeys)
	}

	// Debug logging
	logging.Debugf("Switch %s - Filtered current fields: %v", mac, getMapKeys(filteredCurrent))
	logging.Debugf("Switch %s - Filtered desired fields: %v", mac, getMapKeys(filteredDesired))

	// Convert to JSON for diff
	currentJSON, err1 := json.MarshalIndent(filteredCurrent, "", "  ")
	desiredJSON, err2 := json.MarshalIndent(filteredDesired, "", "  ")

	if err1 != nil || err2 != nil {
		logging.Warnf("Could not generate JSON diff for device %s", mac)
		return
	}

	// Create diff options - no need for IncludeFields since we pre-filtered
	opts := jsondiff.DiffOptions{
		ContextLines: 3,
		SortJSON:     true,
	}

	diffs, err := jsondiff.Diff(currentJSON, desiredJSON, opts)
	if err != nil {
		logging.Warnf("Error generating diff for device %s: %v", mac, err)
		return
	}

	// Enhance diffs with inline changes
	diffs = jsondiff.EnhanceDiffsWithInlineChanges(diffs)

	// Format and display with custom markers
	formatter := jsondiff.NewFormatter(nil)
	formatter.SetMarkers("API Cache", siteName, "Both")

	var output string
	if viper.GetBool("split_diff") {
		output = formatter.FormatSideBySide(diffs, "API Cache", siteName)
	} else {
		output = formatter.Format(diffs)
	}

	if output != "" {
		fmt.Printf("\nConfiguration differences for switch %s:\n", mac)
		fmt.Println(output)
	}
}

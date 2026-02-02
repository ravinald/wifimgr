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
	"github.com/ravinald/wifimgr/internal/vendors"
	"github.com/ravinald/wifimgr/internal/vendors/meraki"
	"github.com/ravinald/wifimgr/internal/vendors/mist"
)

// APUpdater implements DeviceUpdater for Access Points
type APUpdater struct {
	*BaseDeviceUpdater
	batchLoader *DeviceBatchLoader // Reusable batch loader for device lookups
}

// NewAPUpdater creates a new AP updater
func NewAPUpdater() *APUpdater {
	return &APUpdater{
		BaseDeviceUpdater: NewBaseDeviceUpdater("ap"),
	}
}

// GetConfiguredDevices extracts AP MAC addresses from site configuration
func (a *APUpdater) GetConfiguredDevices(siteConfig SiteConfig) []string {
	configuredAPs := make([]string, 0, len(siteConfig.Devices.APs))

	// Now APs is a map with MAC as key
	for mac := range siteConfig.Devices.APs {
		normalizedMAC := macaddr.NormalizeOrEmpty(mac)
		if normalizedMAC != "" {
			configuredAPs = append(configuredAPs, normalizedMAC)
		}
	}

	return configuredAPs
}

// FindDevicesInventoryStatus checks AP status in inventory and cache
func (a *APUpdater) FindDevicesInventoryStatus(client api.Client, cfg *config.Config, configuredAPs []string) ([]DeviceInventoryStatus, error) {
	ctx := context.Background()

	// Create inventory checker once for reuse
	inventoryChecker, err := NewInventoryChecker(ctx, client, cfg, a.deviceType)
	if err != nil {
		return nil, fmt.Errorf("failed to create inventory checker: %w", err)
	}

	// Store it for later use in UpdateDeviceConfigurations (using base method)
	a.SetInventoryChecker(inventoryChecker)

	statusList := make([]DeviceInventoryStatus, 0, len(configuredAPs))

	for _, mac := range configuredAPs {
		status := DeviceInventoryStatus{
			MAC:             mac,
			InInventory:     inventoryChecker.IsInLocalInventory(mac),
			InCache:         inventoryChecker.IsInAPIInventory(mac),
			CurrentSiteName: "",
			CurrentSiteID:   "",
		}

		// Get current site assignment using O(1) lookup from cached inventory
		// This replaces the previous O(n) loop that called GetInventory for each device
		if status.InCache {
			siteID, siteName, _ := inventoryChecker.GetSiteAssignment(mac)
			status.CurrentSiteID = siteID
			status.CurrentSiteName = siteName
		}

		statusList = append(statusList, status)
	}

	logging.Debugf("Inventory status check completed for %d APs", len(statusList))
	return statusList, nil
}

// FindDevicesToUpdate identifies APs that need configuration updates
func (a *APUpdater) FindDevicesToUpdate(ctx context.Context, client api.Client, _ *config.Config, siteConfig SiteConfig, configuredAPs []string, siteID string, apiLabel string) ([]string, error) {
	// Create batch loader for efficient device lookups and store for reuse in UpdateDeviceConfigurations
	batchLoader, err := NewDeviceBatchLoader(ctx, client, siteID, a.deviceType)
	if err != nil {
		return nil, fmt.Errorf("failed to create batch loader: %w", err)
	}
	a.batchLoader = batchLoader // Store for reuse - eliminates duplicate API call
	logging.Debugf("Batch loader created with %d devices for comparison", batchLoader.GetDeviceCount())

	// Get managed keys for AP devices from the API-specific config
	managedKeys := getManagedKeysForDevice(apiLabel, "ap")

	apsToUpdate := make([]string, 0)

	for _, mac := range configuredAPs {
		// Get desired config from site configuration
		desiredConfig, found := a.GetDeviceConfigFromSite(siteConfig, mac)
		if !found {
			logging.Warnf("AP %s is in the list but not found in site configuration", mac)
			continue
		}

		// Get current device state
		device, err := batchLoader.GetDeviceByMAC(mac)
		if err != nil {
			logging.Warnf("Error getting device by MAC %s: %v", mac, err)
			fmt.Printf("→ ap %s: not found in API cache for this site (skipping diff)\n", mac)
			continue
		}

		// Get current config
		currentConfig := device.ToConfigMap()

		// Compare configurations using managed keys
		needsUpdate := compareDeviceConfigsWithManagedKeys(currentConfig, desiredConfig, managedKeys)
		logging.Debugf("AP %s - Current config: %+v", mac, currentConfig)
		logging.Debugf("AP %s - Desired config: %+v", mac, desiredConfig)
		logging.Debugf("AP %s - Needs update: %v", mac, needsUpdate)

		if needsUpdate {
			apsToUpdate = append(apsToUpdate, mac)
			logging.Debugf("AP %s needs configuration update", mac)

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
				showDeviceConfigDiffWithManagedKeys(mac, currentConfig, desiredConfig, managedKeys, siteName)
			}
		} else {
			logging.Debugf("AP %s configuration is up to date", mac)
		}
	}

	return apsToUpdate, nil
}

// UpdateDeviceConfigurations applies configuration updates to APs
func (a *APUpdater) UpdateDeviceConfigurations(ctx context.Context, client api.Client, cfg *config.Config, siteConfig SiteConfig, macs []string, siteID string, apiLabel string) error {
	logging.Infof("Updating configuration for %d APs in site %s", len(macs), siteID)

	// Reuse existing batch loader if available (from FindDevicesToUpdate), otherwise create new one
	batchLoader := a.batchLoader
	if batchLoader == nil {
		var err error
		batchLoader, err = NewDeviceBatchLoader(ctx, client, siteID, a.deviceType)
		if err != nil {
			return fmt.Errorf("failed to create batch loader: %w", err)
		}
		logging.Debugf("Batch loader created with %d devices", batchLoader.GetDeviceCount())
	} else {
		logging.Debugf("Reusing batch loader with %d devices (avoiding duplicate API call)", batchLoader.GetDeviceCount())
	}

	// Use existing inventory checker if available, otherwise create new one
	inventoryChecker := a.GetInventoryChecker()
	if inventoryChecker == nil {
		var inventoryErr error
		inventoryChecker, inventoryErr = NewInventoryChecker(ctx, client, cfg, a.deviceType)
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

	// Legacy backup creation removed - backups are now handled by apply_generic.go
	// which creates config file backups in the format: <config-filename>.json.<index>

	var failedDevices []string
	successCount := 0

	// Build profile name-to-ID map ONCE before processing devices
	// This replaces N API calls with 1 API call for N devices
	profileNameToID, err := buildProfileNameToIDMap(ctx, client, cfg.API.Credentials.OrgID)
	if err != nil {
		logging.Warnf("Could not build profile name map: %v - profile translations may fail", err)
		profileNameToID = make(map[string]string) // Empty map to avoid nil checks
	}

	for _, mac := range macs {
		apConfig, found := a.GetDeviceConfigFromSite(siteConfig, mac)
		if !found {
			logging.Warnf("AP %s is in the list to update but not found in site configuration", mac)
			continue
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

		logging.Debugf("Updating configuration for AP %s (ID: %s, Name: %s)", mac, deviceID, deviceName)

		// Validate configuration before applying
		// TODO: Derive vendor name from client or API label instead of hardcoding
		// For now, we assume Mist since that's the current implementation
		vendorName := "mist"
		if validationErrors := validateAPConfig(apConfig, mac, vendorName); validationErrors != nil {
			if DisplayConfigValidationErrors(validationErrors, mac, vendorName) {
				logging.Errorf("Configuration validation failed for AP %s, skipping update", mac)
				failedDevices = append(failedDevices, mac)
				continue
			}
		}

		updatedDevice := *device

		// Handle _name suffix translations using cached profile map (O(1) lookup)
		translatedConfig := translateNameFieldsWithCache(apConfig, profileNameToID)

		// Filter config to only include managed keys if configured
		managedKeys := getManagedKeysForDevice(apiLabel, "ap")
		var filteredConfig map[string]any
		if len(managedKeys) > 0 {
			filteredConfig = filterConfigByManagedKeys(translatedConfig, managedKeys)
			logging.Debugf("Filtered config to %d managed keys for AP %s", len(filteredConfig), mac)
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
			logging.Errorf("Error updating AP %s configuration via API: %v", mac, err)
			failedDevices = append(failedDevices, mac)
			continue
		}

		if updatedResult != nil {
			logging.Infof("%s Successfully updated configuration for AP %s (Name: %s)", symbols.SuccessPrefix(), mac, deviceName)
			successCount++

			configFields := len(filteredConfig)
			logging.Debugf("Applied %d configuration fields to AP %s", configFields, mac)

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

	logging.Infof("%s Completed configuration updates for %d APs in site %s (%d successful, %d failed)",
		symbols.SuccessPrefix(), len(macs), siteID, successCount, len(failedDevices))
	return nil
}

// GetDeviceConfigFromSite extracts AP-specific config from site configuration
func (a *APUpdater) GetDeviceConfigFromSite(siteConfig SiteConfig, mac string) (map[string]any, bool) {
	normalizedTargetMAC := macaddr.NormalizeOrEmpty(mac)

	// Try direct lookup first (in case keys match exactly)
	if apConfig, ok := siteConfig.Devices.APs[normalizedTargetMAC]; ok {
		return apConfig, true
	}
	if apConfig, ok := siteConfig.Devices.APs[mac]; ok {
		return apConfig, true
	}

	// Iterate and compare normalized versions (handles colon-separated keys like "5c:5b:35:8e:4c:f9")
	for configMac, apConfig := range siteConfig.Devices.APs {
		if macaddr.NormalizeOrEmpty(configMac) == normalizedTargetMAC {
			return apConfig, true
		}
	}

	return nil, false
}

// translateNameFieldsWithCache translates fields with _name suffix to their ID equivalents
// using a pre-built profileNameToID map for O(1) lookups instead of making API calls.
// This is the optimized version that should be used in loops.
func translateNameFieldsWithCache(config map[string]any, profileNameToID map[string]string) map[string]any {
	translatedConfig := make(map[string]any)

	// Copy all fields
	for k, v := range config {
		translatedConfig[k] = v
	}

	// Handle deviceprofile_name to deviceprofile_id translation using cached map
	if profileName, ok := config["deviceprofile_name"]; ok {
		if profileNameStr, ok := profileName.(string); ok {
			if profileID, found := profileNameToID[profileNameStr]; found {
				translatedConfig["deviceprofile_id"] = profileID
				delete(translatedConfig, "deviceprofile_name")
				logging.Debugf("Translated deviceprofile_name '%s' to deviceprofile_id '%s'", profileNameStr, profileID)
			} else {
				logging.Warnf("Device profile '%s' not found in cache", profileNameStr)
			}
		}
	}

	// Handle other _name suffix translations as needed
	// For now, we'll just remove any unhandled _name fields
	keysToRemove := []string{}
	for k := range translatedConfig {
		if len(k) > 5 && k[len(k)-5:] == "_name" && k != "name" {
			keysToRemove = append(keysToRemove, k)
		}
	}

	for _, k := range keysToRemove {
		if k != "deviceprofile_name" { // Already handled above
			delete(translatedConfig, k)
			logging.Debugf("Removed untranslated field: %s", k)
		}
	}

	return translatedConfig
}

// buildProfileNameToIDMap fetches device profiles once and builds a name-to-ID lookup map.
// This should be called once before processing multiple devices.
func buildProfileNameToIDMap(ctx context.Context, client api.Client, orgID string) (map[string]string, error) {
	profileNameToID := make(map[string]string)

	profiles, err := client.GetDeviceProfiles(ctx, orgID, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get device profiles: %w", err)
	}

	for _, profile := range profiles {
		if profile.Name != nil && profile.ID != nil {
			profileNameToID[*profile.Name] = *profile.ID
		}
	}

	logging.Debugf("Built profile name-to-ID map with %d entries", len(profileNameToID))
	return profileNameToID, nil
}

// showDeviceConfigDiffWithManagedKeys displays a colored JSON diff with managed keys highlighted
func showDeviceConfigDiffWithManagedKeys(mac string, currentConfig, desiredConfig map[string]any, managedKeys []string, siteName string) {
	// Filter out status fields that shouldn't be compared
	filteredCurrent := filterStatusFields(currentConfig)
	filteredDesired := filterStatusFields(desiredConfig)

	// Remove MAC from desired config for comparison since it's an identifier, not a configuration field
	// The MAC is already known (it's what we're using to look up the device)
	delete(filteredDesired, "mac")

	// Debug: Log what fields we have before diff
	logging.Debugf("Device %s - Filtered current fields: %v", mac, getMapKeys(filteredCurrent))
	logging.Debugf("Device %s - Filtered desired fields: %v", mac, getMapKeys(filteredDesired))

	// Convert to JSON for diff
	currentJSON, err1 := json.MarshalIndent(filteredCurrent, "", "  ")
	desiredJSON, err2 := json.MarshalIndent(filteredDesired, "", "  ")

	if err1 != nil || err2 != nil {
		logging.Warnf("Could not generate JSON diff for device %s", mac)
		return
	}

	// Create diff with managed keys as include fields
	opts := jsondiff.DiffOptions{
		ContextLines:  3,
		SortJSON:      true,
		IncludeFields: managedKeys, // Use managed keys as include filter
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
		// Use side-by-side format
		output = formatter.FormatSideBySide(diffs, "API Cache", siteName)
	} else {
		// Use unified format (default)
		output = formatter.Format(diffs)
	}

	if output != "" {
		fmt.Printf("\nConfiguration differences for device %s:\n", mac)
		fmt.Println(output)
	}
}

// filterStatusFields removes fields that shouldn't be compared from the config
func filterStatusFields(config map[string]any) map[string]any {
	filtered := make(map[string]any)

	statusFields := map[string]bool{
		"id":            true,
		"created_time":  true,
		"modified_time": true,
		"site_id":       true,
		"org_id":        true,
		"status":        true,
		"last_seen":     true,
		"uptime":        true,
		"version":       true,
		"serial":        true,
		"model":         true,
		"type":          true,
		"magic":         true,
		"connected":     true,
	}

	for k, v := range config {
		if !statusFields[k] {
			filtered[k] = v
		}
	}

	return filtered
}

// getMapKeys returns the keys of a map as a slice
func getMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// validateAPConfig validates AP configuration using the converter's validation logic.
// Returns nil if validation passes, or a slice of validation errors.
func validateAPConfig(config map[string]any, mac string, vendorName string) []error {
	// Import the appropriate converter based on vendor
	// For now we only support Mist, but this is where multi-vendor would be wired up
	switch vendorName {
	case "mist":
		// Convert map to APDeviceConfig for validation
		// The converters work with typed structs, not maps, so we use FromMistAPConfig
		// to parse and validate the configuration
		apCfg, warnings := convertMapToAPConfig(config, mac, vendorName)
		if apCfg == nil {
			// If we can't parse the config, treat it as a validation error
			return warnings
		}

		// Run vendor-specific validation
		vendorErrors := apCfg.ValidateForVendor(vendorName)
		if len(vendorErrors) > 0 {
			return vendorErrors
		}

		// Display any warnings from the conversion process
		if len(warnings) > 0 {
			DisplayConfigWarnings(warnings, mac)
		}

		return nil

	case "meraki":
		// TODO: Add Meraki validation when multi-vendor support is added
		logging.Debugf("Meraki validation not yet implemented")
		return nil

	default:
		logging.Debugf("Unknown vendor %s, skipping validation", vendorName)
		return nil
	}
}

// convertMapToAPConfig converts a map configuration to APDeviceConfig using the
// appropriate vendor converter. This allows us to use the typed validation logic.
func convertMapToAPConfig(config map[string]any, mac string, vendorName string) (*vendors.APDeviceConfig, []error) {
	switch vendorName {
	case "mist":
		return mist.FromMistAPConfig(config, mac)
	case "meraki":
		return meraki.FromMerakiAPConfig(config, mac)
	default:
		return nil, []error{fmt.Errorf("unknown vendor: %s", vendorName)}
	}
}

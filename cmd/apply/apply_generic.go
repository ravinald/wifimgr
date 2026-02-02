package apply

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"

	"github.com/ravinald/wifimgr/api"
	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/symbols"
	"github.com/ravinald/wifimgr/internal/validation"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// FileHashCache stores file hashes and modification times for change detection
type FileHashCache struct {
	Hashes map[string]FileHashInfo `json:"hashes"`
}

// FileHashInfo stores hash and modification time for a file
type FileHashInfo struct {
	Hash    string    `json:"hash"`
	ModTime time.Time `json:"mod_time"`
}

// getDeviceUpdater returns the appropriate DeviceUpdater for the device type.
func getDeviceUpdater(deviceType string) (DeviceUpdater, error) {
	switch deviceType {
	case "ap":
		return NewAPUpdater(), nil
	case "switch":
		return NewSwitchUpdater(), nil
	case "gateway":
		return NewGatewayUpdater(), nil
	default:
		return nil, fmt.Errorf("unknown device type: %s", deviceType)
	}
}

// getManagedKeysForDevice returns the managed keys for a device type from the API config.
// It reads from api.<apiLabel>.managed_keys.<deviceType> in the viper config.
func getManagedKeysForDevice(apiLabel, deviceType string) []string {
	if apiLabel == "" {
		return nil
	}

	// Read managed_keys from viper: api.<label>.managed_keys.<device_type>
	keyPath := fmt.Sprintf("api.%s.managed_keys.%s", apiLabel, deviceType)
	keys := viper.GetStringSlice(keyPath)
	if len(keys) > 0 {
		return keys
	}

	// Viper GetStringSlice returns empty for any slices, try GetStringMap
	managedKeysPath := fmt.Sprintf("api.%s.managed_keys", apiLabel)
	managedKeysMap := viper.GetStringMap(managedKeysPath)
	if managedKeysMap != nil {
		if deviceKeys, ok := managedKeysMap[deviceType]; ok {
			// Handle []any from JSON
			if keyList, ok := deviceKeys.([]any); ok {
				result := make([]string, 0, len(keyList))
				for _, k := range keyList {
					if keyStr, ok := k.(string); ok {
						result = append(result, keyStr)
					}
				}
				return result
			}
		}
	}

	return nil
}

// isManagedKeysConfigured checks if managed keys are configured for a device type
func isManagedKeysConfigured(apiLabel, deviceType string) bool {
	keys := getManagedKeysForDevice(apiLabel, deviceType)
	return len(keys) > 0
}

// applySiteGeneric applies device configuration to a site using the generic framework
func applySiteGeneric(ctx context.Context, client api.Client, cfg *config.Config, siteName string, deviceType string, apiLabel string, force bool, diffMode bool) error {
	// Get the appropriate device updater
	updater, err := getDeviceUpdater(deviceType)
	if err != nil {
		return err
	}

	logging.Infof("Applying %s configuration to site: %s (API: %s)", deviceType, siteName, apiLabel)

	// Check if managed keys are configured for this device type
	if !isManagedKeysConfigured(apiLabel, deviceType) {
		logging.Warnf("WARNING: api.%s.managed_keys.%s is not configured", apiLabel, deviceType)
		fmt.Printf("\nWARNING: No managed keys configured for %s devices.\n", deviceType)
		fmt.Printf("   Configuration differences will be shown, but NO changes will be applied.\n")
		fmt.Printf("   Please configure api.%s.managed_keys.%s in your wifimgr-config.json file.\n\n", apiLabel, deviceType)

		// Force diff mode to show what would be changed
		diffMode = true
	}

	if diffMode {
		fmt.Println("Diff mode enabled - showing changes without applying them")
		viper.Set("show_diff", true)
	}

	// Step 1: Check if the config files have changed
	configFiles := cfg.Files.SiteConfigs
	if len(configFiles) == 0 {
		logging.Error("No site configuration files defined in config")
		return fmt.Errorf("no site configuration files defined in config")
	}

	hasChanges, err := checkConfigFilesChanged(cfg, configFiles)
	if err != nil {
		logging.Errorf("Error checking config file changes: %v", err)
		return fmt.Errorf("error checking config file changes: %v", err)
	}

	if !hasChanges && !force {
		logging.Info("No changes detected in configuration files. Use --force to apply anyway.")
		fmt.Println("No changes detected in configuration files. Use --force to apply anyway.")
		return nil
	}

	// Step 2: Get site configuration first (needed for site ID)
	siteConfig, err := getSiteConfiguration(cfg, configFiles, siteName)
	if err != nil {
		return err
	}

	// Step 3: Get site ID
	siteID, err := getSiteIDByName(client, siteName)
	if err != nil {
		logging.Errorf("Error getting site ID for %s: %v", siteName, err)
		return fmt.Errorf("error getting site ID for %s: %v", siteName, err)
	}

	// Step 4: Update the local cache (optimized for site-specific operations)
	// Skip if multi-vendor cache is available (already refreshed before this call)
	if vendors.GetGlobalCacheAccessor() == nil {
		// Legacy mode: use the legacy client to populate cache
		logging.Infof("Updating cache for site %s, device type %s...", siteName, deviceType)
		if err := client.PopulateDeviceCacheForSite(ctx, siteID, deviceType); err != nil {
			logging.Errorf("Error updating site-specific cache: %v", err)
			return fmt.Errorf("error updating site-specific cache: %v", err)
		}
	} else {
		logging.Debugf("Skipping legacy cache population - using multi-vendor cache")
	}

	// Step 5: Get configured and assigned devices
	configuredDevices := updater.GetConfiguredDevices(siteConfig)
	assignedDevices, err := updater.GetAssignedDevices(ctx, client, siteID)
	if err != nil {
		logging.Errorf("Error getting assigned %ss from cache: %v", deviceType, err)
		return fmt.Errorf("error getting assigned %ss from cache: %v", deviceType, err)
	}

	// Step 5.5: Create InventoryChecker ONCE for reuse throughout the apply workflow
	// This eliminates multiple redundant GetInventory API calls
	inventoryChecker, err := NewInventoryChecker(ctx, client, cfg, deviceType)
	if err != nil {
		logging.Warnf("Could not create inventory checker: %v - some safety checks may be skipped", err)
	} else {
		// Store on updater for reuse in FindDevicesInventoryStatus and UpdateDeviceConfigurations
		updater.SetInventoryChecker(inventoryChecker)
		logging.Debugf("Created shared InventoryChecker with %d API items", len(inventoryChecker.apiInventory))
	}

	// Step 6: Find devices to unassign (only if they're in inventory)
	devicesToUnassign, err := findDevicesToUnassignWithInventoryCheck(ctx, client, cfg, assignedDevices, configuredDevices, deviceType, inventoryChecker)
	if err != nil {
		logging.Errorf("Error finding %ss to unassign: %v", deviceType, err)
		return fmt.Errorf("error finding %ss to unassign: %v", deviceType, err)
	}
	if len(devicesToUnassign) > 0 {
		logging.Infof("Found %d %ss to unassign from site %s (in inventory but not in config)", len(devicesToUnassign), deviceType, siteName)
		if !diffMode {
			if err := updater.UnassignDevices(ctx, client, cfg, devicesToUnassign); err != nil {
				logging.Errorf("Error unassigning %ss: %v", deviceType, err)
				return fmt.Errorf("error unassigning %ss: %v", deviceType, err)
			}
		} else {
			fmt.Printf("Would unassign the following %ss from site %s:\n", deviceType, siteName)
			for _, device := range devicesToUnassign {
				fmt.Printf("- %s\n", device)
			}
		}
	}

	// Step 7: Check inventory status
	deviceStatus, err := updater.FindDevicesInventoryStatus(client, cfg, configuredDevices)
	if err != nil {
		logging.Errorf("Error checking inventory status for %ss: %v", deviceType, err)
		return fmt.Errorf("error checking inventory status for %ss: %v", deviceType, err)
	}

	// Process device status and show warnings
	hasWarnings := false
	devicesNotInInventory := make([]string, 0)
	for _, status := range deviceStatus {
		// Check if device is missing from cache (API inventory) - primary concern
		if !status.InCache {
			hasWarnings = true
			devicesNotInInventory = append(devicesNotInInventory, status.MAC)
			logging.Warnf("%s %s: not found in API inventory", deviceType, status.MAC)
			fmt.Printf("→ %s %s: not found in API inventory\n", deviceType, status.MAC)
			continue // Skip other checks if not in cache
		}

		// Device is in cache but not in local inventory file - safety check
		if !status.InInventory {
			hasWarnings = true
			devicesNotInInventory = append(devicesNotInInventory, status.MAC)
			logging.Warnf("%s %s: not in local inventory file", deviceType, status.MAC)
			fmt.Printf("→ %s %s: not in local inventory file\n", deviceType, status.MAC)
		}

		// Device is assigned to a different site
		if status.CurrentSiteID != "" && status.CurrentSiteID != siteID {
			hasWarnings = true
			siteInfo := status.CurrentSiteID
			if status.CurrentSiteName != "" {
				siteInfo = fmt.Sprintf("%s (%s)", status.CurrentSiteName, status.CurrentSiteID)
			}
			logging.Warnf("%s %s: assigned to different site %s", deviceType, status.MAC, siteInfo)
			fmt.Printf("→ %s %s: assigned to different site %s\n", deviceType, status.MAC, siteInfo)
		}
	}

	// Filter out devices not in inventory if force is not used
	configuredDevicesFiltered := configuredDevices
	if !force && len(devicesNotInInventory) > 0 {
		notInInventoryMap := make(map[string]bool)
		for _, mac := range devicesNotInInventory {
			notInInventoryMap[mac] = true
		}

		filtered := make([]string, 0, len(configuredDevices))
		for _, mac := range configuredDevices {
			if !notInInventoryMap[mac] {
				filtered = append(filtered, mac)
			}
		}
		configuredDevicesFiltered = filtered

		fmt.Printf("\nSkipping %d %s(s) not in inventory.\n", len(devicesNotInInventory), deviceType)

		// If all devices were filtered out, return early
		if len(configuredDevicesFiltered) == 0 {
			fmt.Printf("No valid %ss to process. Use --force to include devices not in inventory.\n", deviceType)
			return nil
		}

		fmt.Printf("Continuing with %d valid %s(s).\n\n", len(configuredDevicesFiltered), deviceType)
	}

	// Get inventory checker from updater (created in Step 5.5 - no redundant API call)
	// This reuses the same checker throughout the apply workflow
	inventoryCheckerForFilter := updater.GetInventoryChecker()

	// Step 8: Find devices to assign
	devicesToAssign, err := updater.FindDevicesToAssign(client, cfg, configuredDevicesFiltered, siteID)
	if err != nil {
		logging.Errorf("Error finding %ss to assign: %v", deviceType, err)
		return fmt.Errorf("error finding %ss to assign: %v", deviceType, err)
	}

	// Filter to only assign devices that are in inventory
	if inventoryCheckerForFilter != nil {
		devicesToAssign = inventoryCheckerForFilter.FilterByInventory(devicesToAssign)
	}

	if len(devicesToAssign) > 0 {
		logging.Infof("Found %d %ss to assign to site %s", len(devicesToAssign), deviceType, siteName)
		if !diffMode {
			if err := updater.AssignDevices(ctx, client, cfg, devicesToAssign, siteID); err != nil {
				logging.Errorf("Error assigning %ss: %v", deviceType, err)
				return fmt.Errorf("error assigning %ss: %v", deviceType, err)
			}
		} else {
			fmt.Printf("Would assign the following %ss to site %s:\n", deviceType, siteName)
			for _, device := range devicesToAssign {
				fmt.Printf("- %s\n", device)
			}
		}
	}

	// Step 8.5: Run compatibility check before applying changes (unless --skip-compat-check)
	if !viper.GetBool("skip_compat_check") && !diffMode {
		logging.Debugf("Running API compatibility check for site %s", siteName)
		if err := runCompatibilityCheck(siteConfig, siteName, apiLabel); err != nil {
			// Compatibility check failed - warn but don't abort (can be skipped with --skip-compat-check)
			fmt.Printf("\n%s API compatibility check found issues:\n", symbols.WarningPrefix())
			fmt.Printf("   %v\n", err)
			fmt.Printf("   Use --skip-compat-check to bypass this check (not recommended)\n\n")

			// For now, treat as warning only. Can be changed to error if desired.
			logging.Warnf("Compatibility check found issues but continuing: %v", err)
		} else {
			logging.Debugf("Compatibility check passed for site %s", siteName)
		}
	}

	// Step 9: Find devices to update
	devicesToUpdate, err := updater.FindDevicesToUpdate(ctx, client, cfg, siteConfig, configuredDevicesFiltered, siteID, apiLabel)
	if err != nil {
		logging.Errorf("Error finding %ss to update: %v", deviceType, err)
		return fmt.Errorf("error finding %ss to update: %v", deviceType, err)
	}

	// Filter to only update devices that are in inventory
	if inventoryCheckerForFilter != nil {
		devicesToUpdate = inventoryCheckerForFilter.FilterByInventory(devicesToUpdate)
	}

	if len(devicesToUpdate) > 0 {
		logging.Infof("Found %d %ss to update in site %s", len(devicesToUpdate), deviceType, siteName)
		if !diffMode {
			if err := updater.UpdateDeviceConfigurations(ctx, client, cfg, siteConfig, devicesToUpdate, siteID, apiLabel); err != nil {
				logging.Errorf("Error updating %s configurations: %v", deviceType, err)
				return fmt.Errorf("error updating %s configurations: %v", deviceType, err)
			}
		} else {
			fmt.Printf("Would update the following %ss in site %s:\n", deviceType, siteName)
			for _, device := range devicesToUpdate {
				fmt.Printf("- %s\n", device)
			}
		}
	}

	// Step 10: Check if any changes were made
	if len(devicesToAssign) == 0 && len(devicesToUpdate) == 0 && len(devicesToUnassign) == 0 {
		if hasWarnings {
			fmt.Println("No changes applied due to warnings in the configuration.")
		} else {
			fmt.Println("No changes needed - all devices are already configured correctly.")
		}
	} else if diffMode {
		fmt.Println("Diff mode completed - no changes have been applied")
	} else {
		fmt.Printf("Successfully applied %s configuration to site %s\n", deviceType, siteName)

		// Create backup of the applied configuration
		for _, configFile := range configFiles {
			configFilePath := configFile
			if !filepath.IsAbs(configFilePath) {
				configFilePath = filepath.Join(cfg.Files.ConfigDir, configFile)
			}
			// Check if this config file contains the site
			if siteConfigs, err := getSiteConfigsFromFiles([]string{configFile}); err == nil {
				if _, found := siteConfigs[siteName]; found {
					// This config file contains the site, create a backup
					if err := createConfigBackupAfterApply(cfg, siteName, configFilePath); err != nil {
						logging.Warnf("Failed to create configuration backup: %v", err)
					} else {
						logging.Debugf("Created backup for site %s from config file %s", siteName, configFile)
					}
					break // Only backup the file containing this site
				}
			}
		}

		// Update file hashes after successful apply
		if err := updateFileHashes(cfg, configFiles); err != nil {
			logging.Warnf("Failed to update file hashes: %v", err)
		}
	}

	// Log cache performance statistics
	if deviceCache := client.GetDeviceCache(); deviceCache != nil {
		hits, misses, hitRate := deviceCache.GetCacheStats()
		logging.Debugf("Cache performance - Hits: %d, Misses: %d, Hit Rate: %.2f%%", hits, misses, hitRate)
	}

	return nil
}

// checkConfigFilesChanged checks if any config files have changed using SHA256 hashes
func checkConfigFilesChanged(cfg *config.Config, configFiles []string) (bool, error) {
	// Load cached hashes
	cacheFile := filepath.Join(cfg.Files.ConfigDir, ".file_hashes.json")
	cache := &FileHashCache{Hashes: make(map[string]FileHashInfo)}

	if data, err := os.ReadFile(cacheFile); err == nil {
		if err := json.Unmarshal(data, cache); err != nil {
			logging.Warnf("Failed to parse file hash cache: %v", err)
		}
	}

	hasChanges := false

	for _, configFile := range configFiles {
		filePath := configFile
		if !filepath.IsAbs(filePath) {
			filePath = filepath.Join(cfg.Files.ConfigDir, configFile)
		}

		// Get file info
		info, err := os.Stat(filePath)
		if err != nil {
			return false, fmt.Errorf("failed to stat config file %s: %w", filePath, err)
		}

		// Calculate current hash
		currentHash, err := calculateFileHash(filePath)
		if err != nil {
			return false, fmt.Errorf("failed to calculate hash for %s: %w", filePath, err)
		}

		// Check against cached hash
		if cachedInfo, exists := cache.Hashes[configFile]; exists {
			if cachedInfo.Hash != currentHash || !cachedInfo.ModTime.Equal(info.ModTime()) {
				hasChanges = true
				logging.Infof("Changes detected in config file: %s", configFile)
			}
		} else {
			// No cached hash, assume changed
			hasChanges = true
			logging.Infof("New config file detected: %s", configFile)
		}
	}

	return hasChanges, nil
}

// calculateFileHash calculates SHA256 hash of a file
func calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := file.Close(); err != nil {
			logging.Warnf("Error closing hash file %s: %v", filePath, err)
		}
	}()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// updateFileHashes updates the cached file hashes
func updateFileHashes(cfg *config.Config, configFiles []string) error {
	cache := &FileHashCache{Hashes: make(map[string]FileHashInfo)}

	for _, configFile := range configFiles {
		filePath := configFile
		if !filepath.IsAbs(filePath) {
			filePath = filepath.Join(cfg.Files.ConfigDir, configFile)
		}

		info, err := os.Stat(filePath)
		if err != nil {
			continue
		}

		hash, err := calculateFileHash(filePath)
		if err != nil {
			continue
		}

		cache.Hashes[configFile] = FileHashInfo{
			Hash:    hash,
			ModTime: info.ModTime(),
		}
	}

	// Save cache
	cacheFile := filepath.Join(cfg.Files.ConfigDir, ".file_hashes.json")
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cacheFile, data, 0644)
}

// getSiteConfiguration finds and returns the site configuration
func getSiteConfiguration(cfg *config.Config, configFiles []string, siteName string) (SiteConfig, error) {
	// Set CONFIG_DIR environment variable
	if cfg.Files.ConfigDir != "" {
		if err := os.Setenv("CONFIG_DIR", cfg.Files.ConfigDir); err != nil {
			logging.Errorf("Error setting CONFIG_DIR environment variable: %v", err)
			return SiteConfig{}, fmt.Errorf("error setting CONFIG_DIR environment variable: %v", err)
		}
	}

	siteConfigs, err := getSiteConfigsFromFiles(configFiles)
	if err != nil {
		logging.Errorf("Error reading site configurations: %v", err)
		return SiteConfig{}, fmt.Errorf("error reading site configurations: %v", err)
	}

	// Check if the specified site exists in the configs
	siteConfig, found := siteConfigs[siteName]
	if !found {
		// Search through all config files for the site
		found = false
		for _, configFile := range configFiles {
			filePath := configFile
			if !filepath.IsAbs(filePath) {
				filePath = filepath.Join(cfg.Files.ConfigDir, configFile)
			}

			fileData, err := os.ReadFile(filePath)
			if err != nil {
				logging.Warnf("Error reading config file %s for fallback search: %v", configFile, err)
				continue
			}

			var fileConfig ConfigFileStructure
			if err := json.Unmarshal(fileData, &fileConfig); err != nil {
				logging.Warnf("Error parsing config file %s for fallback search: %v", configFile, err)
				continue
			}

			for configID, siteConf := range fileConfig.Config.Sites {
				if nameVal, ok := siteConf.SiteConfig["name"]; ok {
					if name, ok := nameVal.(string); ok && name == siteName {
						siteConfig = siteConf
						found = true
						logging.Infof("Found site %s with config ID %s in file %s", siteName, configID, configFile)
						break
					}
				}
			}

			if found {
				break
			}
		}

		if !found {
			logging.Errorf("Site %s not found in configuration files", siteName)
			return SiteConfig{}, fmt.Errorf("site %s not found in configuration files", siteName)
		}
	}

	return siteConfig, nil
}

// runCompatibilityCheck runs API compatibility check on the site configuration.
// Returns error if there are compatibility issues.
func runCompatibilityCheck(siteConfig SiteConfig, siteName string, apiLabel string) error {
	// Get global cache accessor
	cacheAccessor := vendors.GetGlobalCacheAccessor()

	// Create schema tracker (in production, this would be loaded from disk)
	schemaTracker := vendors.NewSchemaTracker()

	// Create compatibility checker
	checker := validation.NewCompatibilityChecker(schemaTracker, cacheAccessor)

	// Convert apply.SiteConfig to config.SiteConfigObj for validation
	// Note: apply.SiteConfig uses map[string]any for flexibility during apply operations
	// while config.SiteConfigObj uses typed structs
	siteConfigObj := &config.SiteConfigObj{
		API: apiLabel,
		SiteConfig: config.SiteConfig{
			Name: siteName,
		},
		Devices: config.Devices{
			APs:      convertAPDevicesForValidation(siteConfig.Devices.APs),
			Switches: convertSwitchDevicesForValidation(siteConfig.Devices.Switches),
			WanEdge:  convertGatewayDevicesForValidation(siteConfig.Devices.WanEdge),
		},
	}

	// Extract site name from siteConfig if available
	if name, ok := siteConfig.SiteConfig["name"].(string); ok {
		siteConfigObj.SiteConfig.Name = name
	}

	// Perform compatibility check
	result, err := checker.CheckSite(siteName, siteConfigObj)
	if err != nil {
		return fmt.Errorf("compatibility check failed: %w", err)
	}

	// Check for errors
	if result.HasErrors() {
		return fmt.Errorf("found %d compatibility error(s)", len(result.FilterBySeverity("error")))
	}

	// Warnings are OK, just log them
	if result.HasWarnings() {
		logging.Warnf("Compatibility check found %d warning(s)", len(result.FilterBySeverity("warning")))
	}

	return nil
}

// Helper functions to convert map-based device configs to typed configs for validation

func convertAPDevicesForValidation(devices map[string]map[string]any) map[string]config.APConfig {
	result := make(map[string]config.APConfig)
	for mac, deviceMap := range devices {
		apConfig := config.APConfig{
			MAC: mac,
		}
		// Extract basic fields for validation
		if name, ok := deviceMap["name"].(string); ok {
			apConfig.APDeviceConfig = &vendors.APDeviceConfig{
				Name: name,
			}
		}
		result[mac] = apConfig
	}
	return result
}

func convertSwitchDevicesForValidation(devices map[string]map[string]any) map[string]config.SwitchConfig {
	result := make(map[string]config.SwitchConfig)
	for mac, deviceMap := range devices {
		switchConfig := config.SwitchConfig{}
		if name, ok := deviceMap["name"].(string); ok {
			switchConfig.Name = name
		}
		result[mac] = switchConfig
	}
	return result
}

func convertGatewayDevicesForValidation(devices map[string]map[string]any) map[string]config.WanEdgeConfig {
	result := make(map[string]config.WanEdgeConfig)
	for mac, deviceMap := range devices {
		gwConfig := config.WanEdgeConfig{}
		if name, ok := deviceMap["name"].(string); ok {
			gwConfig.Name = name
		}
		result[mac] = gwConfig
	}
	return result
}

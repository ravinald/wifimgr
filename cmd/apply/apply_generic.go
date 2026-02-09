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
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/ravinald/jsondiff/pkg/jsondiff"
	"github.com/spf13/viper"

	"github.com/ravinald/wifimgr/api"
	configPkg "github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/symbols"
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

// getManagedKeysForDevice returns the managed keys for a device type from the API configPkg.
// It reads from api.<apiLabel>.managed_keys.<deviceType> in the viper configPkg.
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

// applySiteGeneric applies device configuration to a site using the generic framework.
// When refreshAPI is true, the cache is refreshed from the API before applying changes.
// When refreshAPI is false (default), the existing cache is used for efficiency.
func applySiteGeneric(ctx context.Context, client api.Client, cfg *configPkg.Config, siteName string, deviceType string, apiLabel string, force bool, diffMode bool, refreshAPI bool) error {
	// Get the appropriate device updater
	updater, err := getDeviceUpdater(deviceType)
	if err != nil {
		return err
	}

	logging.Infof("Applying %s configuration to site: %s (API: %s)", deviceType, siteName, apiLabel)

	// Load templates if configured
	templates, err := loadTemplatesFromConfig(cfg)
	if err != nil {
		logging.Warnf("Failed to load templates: %v - continuing without template expansion", err)
		templates = configPkg.NewTemplateStore()
	} else if !templates.IsEmpty() {
		templateList := templates.ListTemplates()
		logging.Debugf("Loaded templates - radio: %d, wlan: %d, device: %d",
			len(templateList["radio"]), len(templateList["wlan"]), len(templateList["device"]))
	}

	// Store templates for use by device updaters
	setTemplateStore(templates, apiLabel)

	// Check if managed keys are configured for this device type
	if !isManagedKeysConfigured(apiLabel, deviceType) {
		logging.Warnf("WARNING: api.%s.managed_keys.%s is not configured", apiLabel, deviceType)
		fmt.Printf("\nWARNING: No managed keys configured for %s devices.\n", deviceType)
		fmt.Printf("   Configuration differences will be shown, but NO changes will be applied.\n")
		fmt.Printf("   Please configure api.%s.managed_keys.%s in your wifimgr-configPkg.json file.\n\n", apiLabel, deviceType)

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

	// Note: Even if config files haven't changed, we still need to check WLANs
	// against API state since they might not exist yet on the API.
	skipDeviceUpdates := !hasChanges && !force && !diffMode
	if skipDeviceUpdates {
		logging.Debugf("No config file changes - will skip device updates but still check WLANs")
	}
	if !hasChanges && diffMode {
		logging.Debugf("Diff mode - proceeding to compare against API state")
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
	// By default, skip cache refresh and use existing cached data for efficiency.
	// Use "refresh-api" positional argument to force a fresh API refresh when drift detection is needed.
	if refreshAPI {
		if vendors.GetGlobalCacheAccessor() == nil {
			// Legacy mode: use the legacy client to populate cache
			logging.Infof("Refreshing cache from API for site %s, device type %s...", siteName, deviceType)
			if err := client.PopulateDeviceCacheForSite(ctx, siteID, deviceType); err != nil {
				logging.Errorf("Error updating site-specific cache: %v", err)
				return fmt.Errorf("error updating site-specific cache: %v", err)
			}
		} else {
			logging.Infof("Refreshing multi-vendor cache from API for site %s...", siteName)
			// For multi-vendor, the cache accessor handles refresh
			// Note: Actual refresh implementation depends on vendors package capabilities
		}
	} else {
		logging.Debugf("Using cached data (no API refresh) - use 'refresh-api' to force refresh")
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
	var devicesToAssign []string
	if !skipDeviceUpdates {
		devicesToAssign, err = updater.FindDevicesToAssign(client, cfg, configuredDevicesFiltered, siteID)
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
		}
	}

	// Step 8.5: Apply WLANs BEFORE device updates (WLANs must exist for device WLAN assignments)
	// WLANs are site-level resources that devices reference
	wlanChanges := 0
	if deviceType == "ap" {
		wlanChangeCount, err := applyWLANs(ctx, client, cfg, siteConfig, siteID, apiLabel, diffMode, force)
		if err != nil {
			logging.Errorf("Error applying WLANs: %v", err)
			// Don't fail the whole apply, just warn
			fmt.Printf("Warning: Failed to apply WLANs: %v\n", err)
		} else {
			wlanChanges = wlanChangeCount
		}
	}

	// Step 9: Find devices to update
	var devicesToUpdate []string
	if !skipDeviceUpdates {
		devicesToUpdate, err = updater.FindDevicesToUpdate(ctx, client, cfg, siteConfig, configuredDevicesFiltered, siteID, apiLabel)
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
		}
	} else {
		logging.Debugf("Skipping device update detection - no config file changes")
	}

	// Step 9.5: Apply all changes (unassign, assign, update)
	// Note: API state backup is not created by default. The intent config backup (created after apply)
	// is sufficient for most rollback scenarios. Use "refresh-api" positional argument to refresh
	// cache from API before apply if drift detection is needed.
	if diffMode {
		// Show what would be changed
		if len(devicesToUnassign) > 0 {
			fmt.Printf("Would unassign the following %ss from site %s:\n", deviceType, siteName)
			for _, device := range devicesToUnassign {
				fmt.Printf("  - %s\n", device)
			}
		}
		if len(devicesToAssign) > 0 {
			fmt.Printf("Would assign the following %ss to site %s:\n", deviceType, siteName)
			for _, device := range devicesToAssign {
				fmt.Printf("  - %s\n", device)
			}
		}
		if len(devicesToUpdate) > 0 {
			fmt.Printf("Would update the following %ss in site %s:\n", deviceType, siteName)
			for _, device := range devicesToUpdate {
				fmt.Printf("  - %s\n", device)
			}
		}

		// Show device summary
		totalDevices := len(configuredDevicesFiltered)
		upToDate := totalDevices - len(devicesToUpdate) - len(devicesToAssign)
		if totalDevices > 0 {
			if len(devicesToUpdate) == 0 && len(devicesToAssign) == 0 {
				fmt.Printf("Devices: %d %s(s) checked, all up to date\n", totalDevices, deviceType)
			} else {
				fmt.Printf("Devices: %d %s(s) checked, %d need updates, %d up to date\n",
					totalDevices, deviceType, len(devicesToUpdate)+len(devicesToAssign), upToDate)
			}
		}
	} else {
		// Apply changes in order: unassign, assign, update
		if len(devicesToUnassign) > 0 {
			if err := updater.UnassignDevices(ctx, client, cfg, devicesToUnassign); err != nil {
				logging.Errorf("Error unassigning %ss: %v", deviceType, err)
				return fmt.Errorf("error unassigning %ss: %v", deviceType, err)
			}
		}
		if len(devicesToAssign) > 0 {
			if err := updater.AssignDevices(ctx, client, cfg, devicesToAssign, siteID); err != nil {
				logging.Errorf("Error assigning %ss: %v", deviceType, err)
				return fmt.Errorf("error assigning %ss: %v", deviceType, err)
			}
		}
		if len(devicesToUpdate) > 0 {
			if err := updater.UpdateDeviceConfigurations(ctx, client, cfg, siteConfig, devicesToUpdate, siteID, apiLabel); err != nil {
				logging.Errorf("Error updating %s configurations: %v", deviceType, err)
				return fmt.Errorf("error updating %s configurations: %v", deviceType, err)
			}
		}
	}

	// Step 10: Check if any changes were made
	if len(devicesToAssign) == 0 && len(devicesToUpdate) == 0 && len(devicesToUnassign) == 0 && wlanChanges == 0 {
		if hasWarnings {
			fmt.Println("No changes applied due to warnings in the configuration.")
		} else {
			fmt.Println("No changes needed - all devices and WLANs are already configured correctly.")
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
func checkConfigFilesChanged(cfg *configPkg.Config, configFiles []string) (bool, error) {
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
func updateFileHashes(cfg *configPkg.Config, configFiles []string) error {
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
func getSiteConfiguration(cfg *configPkg.Config, configFiles []string, siteName string) (SiteConfig, error) {
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

// applyWLANs applies WLAN configurations to the API.
// WLANs are created/updated at the site level.
// Collects WLANs from both site profiles AND device configs to ensure all referenced WLANs exist.
// For Mist: sets ap_ids and apply_to based on which devices reference the WLAN.
// Returns the number of WLANs created or updated.
func applyWLANs(ctx context.Context, client api.Client, cfg *configPkg.Config, siteConfig SiteConfig, siteID string, apiLabel string, diffMode bool, force bool) (int, error) {
	// Collect ALL WLAN labels from both site profiles and device configs
	wlanLabels := collectAllWLANLabels(siteConfig)

	// Build map of WLAN label -> device MACs that reference it
	wlanToDevices := collectWLANDeviceMapping(siteConfig)
	if len(wlanLabels) == 0 {
		logging.Debugf("No WLANs configured for site or devices")
		return 0, nil
	}

	logging.Infof("Processing %d WLAN(s) for site: %v", len(wlanLabels), wlanLabels)

	// Get templates
	templates, templateAPILabel := getTemplateStore()
	if templates == nil || templates.IsEmpty() {
		logging.Warnf("No templates loaded, cannot expand WLAN labels")
		return 0, fmt.Errorf("no templates loaded for WLAN expansion")
	}

	// Validate WLAN assignments before proceeding
	if err := validateWLANAssignments(siteConfig, templates); err != nil {
		return 0, err
	}

	// Determine vendor from API label
	vendor := configPkg.GetVendorFromAPILabel(apiLabel)
	if vendor == "" {
		vendor = configPkg.GetVendorFromAPILabel(templateAPILabel)
	}

	// Expand WLAN templates
	var desiredWLANs []map[string]any
	for _, label := range wlanLabels {
		template, found := templates.GetWLANTemplate(label)
		if !found {
			logging.Warnf("WLAN template '%s' not found", label)
			continue
		}
		// Expand for vendor (handles mist:/meraki: blocks)
		expanded := configPkg.ExpandForVendor(template, vendor)
		// Add the template label for reference
		expanded["_template_label"] = label
		desiredWLANs = append(desiredWLANs, expanded)
		logging.Debugf("Expanded WLAN template '%s': ssid=%v", label, expanded["ssid"])
	}

	if len(desiredWLANs) == 0 {
		logging.Warnf("No WLAN templates could be expanded")
		return 0, nil
	}

	// Meraki: use vendor-agnostic WLANs service with availability tags
	if vendor == "meraki" {
		return applyWLANsMeraki(ctx, cfg, siteConfig, siteID, apiLabel, wlanLabels, wlanToDevices, desiredWLANs, diffMode, force)
	}

	// Get existing WLANs for this site from API
	existingWLANs, err := client.GetSiteWLANs(ctx, siteID)
	if err != nil {
		logging.Warnf("Failed to get existing WLANs for site %s: %v", siteID, err)
		existingWLANs = nil // Treat as empty
	}

	// Build a map of existing WLANs by SSID for quick lookup
	existingBySSID := make(map[string]*api.MistWLAN)
	for i := range existingWLANs {
		w := &existingWLANs[i]
		if w.SSID != nil {
			existingBySSID[*w.SSID] = w
		}
	}

	logging.Debugf("Found %d existing WLANs in site", len(existingWLANs))

	changeCount := 0

	// Process each desired WLAN
	for _, desired := range desiredWLANs {
		ssid, ok := desired["ssid"].(string)
		if !ok || ssid == "" {
			logging.Warnf("WLAN template has no ssid field, skipping")
			continue
		}

		templateLabel := ""
		if label, ok := desired["_template_label"].(string); ok {
			templateLabel = label
			delete(desired, "_template_label") // Don't send to API
		}

		// For Mist: Set ap_ids and apply_to based on which devices reference this WLAN
		if deviceMACs, hasDevices := wlanToDevices[templateLabel]; hasDevices && len(deviceMACs) > 0 {
			// Get AP IDs from the multi-vendor cache accessor
			vendorsCacheAccessor := vendors.GetGlobalCacheAccessor()
			var apIDs []string
			if vendorsCacheAccessor != nil {
				for _, mac := range deviceMACs {
					ap, err := vendorsCacheAccessor.GetAPByMAC(mac)
					if err == nil && ap != nil && ap.ID != "" {
						apIDs = append(apIDs, ap.ID)
						logging.Debugf("WLAN '%s': resolved MAC %s to AP ID %s", templateLabel, mac, ap.ID)
					} else {
						logging.Warnf("WLAN '%s': could not resolve MAC %s to AP ID: %v", templateLabel, mac, err)
					}
				}
			} else {
				logging.Warnf("WLAN '%s': cache accessor not available, cannot resolve MACs to AP IDs", templateLabel)
			}
			if len(apIDs) > 0 {
				desired["ap_ids"] = apIDs
				desired["apply_to"] = "aps"
				logging.Infof("WLAN '%s' will apply to %d specific AP(s)", templateLabel, len(apIDs))
			}
		} else {
			if slices.Contains(siteConfig.WLAN, templateLabel) {
				// Site-level WLAN: explicit AP association (all applicable APs have overrides)
				desired["apply_to"] = "aps"
				desired["ap_ids"] = []string{}
				logging.Infof("WLAN '%s' (site-level) explicitly set with no applicable APs", templateLabel)
			} else {
				// Profile-only WLAN: apply to entire site
				desired["apply_to"] = "site"
				logging.Debugf("WLAN '%s' (profile-only) will apply to entire site", templateLabel)
			}
		}

		existing, exists := existingBySSID[ssid]

		if exists {
			// WLAN exists - check if update needed (or force)
			needsUpdate := wlanNeedsUpdate(existing, desired)
			if needsUpdate || force {
				if diffMode {
					if force && !needsUpdate {
						fmt.Printf("Would force update WLAN '%s' (template: %s) - no changes detected\n", ssid, templateLabel)
					} else {
						fmt.Printf("Would update WLAN '%s' (template: %s)\n", ssid, templateLabel)
						showWLANDiff(existing, desired)
					}
				} else {
					if force && !needsUpdate {
						logging.Infof("Force updating WLAN '%s' (template: %s) - no changes detected", ssid, templateLabel)
					} else {
						logging.Infof("Updating WLAN '%s' (template: %s)", ssid, templateLabel)
					}
					if err := updateWLAN(ctx, client, siteID, *existing.ID, desired); err != nil {
						logging.Errorf("Failed to update WLAN '%s': %v", ssid, err)
						printWLANError("update", ssid, templateLabel, desired, err)
						continue
					}
					fmt.Printf("%s Updated WLAN '%s'\n", symbols.SuccessPrefix(), ssid)
				}
				changeCount++
			} else {
				logging.Debugf("WLAN '%s' is up to date", ssid)
			}
		} else {
			// WLAN doesn't exist - create it
			if diffMode {
				fmt.Printf("Would create WLAN '%s' (template: %s)\n", ssid, templateLabel)
				showWLANConfig(desired)
			} else {
				logging.Infof("Creating WLAN '%s' (template: %s)", ssid, templateLabel)
				if err := createWLAN(ctx, client, siteID, desired); err != nil {
					logging.Errorf("Failed to create WLAN '%s': %v", ssid, err)
					printWLANError("create", ssid, templateLabel, desired, err)
					continue
				}
				fmt.Printf("%s Created WLAN '%s'\n", symbols.SuccessPrefix(), ssid)
			}
			changeCount++
		}
	}

	if changeCount > 0 && !diffMode {
		logging.Infof("Applied %d WLAN change(s)", changeCount)
	}

	return changeCount, nil
}

// wlanNeedsUpdate checks if a WLAN configuration differs from desired state.
// Compares key fields: enabled, band, bands, vlan_id, auth type, auth pairwise, apply_to, ap_ids.
func wlanNeedsUpdate(existing *api.MistWLAN, desired map[string]any) bool {
	// Check enabled
	if desiredEnabled, ok := desired["enabled"].(bool); ok {
		if existing.Enabled == nil || *existing.Enabled != desiredEnabled {
			return true
		}
	}

	// Check band (legacy single band)
	if desiredBand, ok := desired["band"].(string); ok {
		if existing.Band == nil || *existing.Band != desiredBand {
			return true
		}
	}

	// Check bands array (for 6GHz support)
	if desiredBands, ok := desired["bands"].([]any); ok {
		if existing.Bands == nil {
			return true // Existing has no bands, desired has bands
		}
		// Compare band arrays
		existingBands := *existing.Bands
		if len(existingBands) != len(desiredBands) {
			return true
		}
		for i, b := range desiredBands {
			if bs, ok := b.(string); ok {
				if i >= len(existingBands) || existingBands[i] != bs {
					return true
				}
			}
		}
	}

	// Check vlan_id
	if desiredVLAN, ok := desired["vlan_id"].(float64); ok {
		if existing.VlanID == nil || *existing.VlanID != int(desiredVLAN) {
			return true
		}
	}
	if desiredVLAN, ok := desired["vlan_id"].(int); ok {
		if existing.VlanID == nil || *existing.VlanID != desiredVLAN {
			return true
		}
	}

	// Check auth type and pairwise
	if auth, ok := desired["auth"].(map[string]any); ok {
		if authType, ok := auth["type"].(string); ok {
			if existing.Auth.Type == nil || *existing.Auth.Type != authType {
				return true
			}
		}
		// Check pairwise (WPA2/WPA3 security mode)
		if desiredPairwise, ok := auth["pairwise"].([]any); ok {
			if existing.Auth.Pairwise == nil {
				return true // Existing has no pairwise, desired has pairwise
			}
			existingPairwise := *existing.Auth.Pairwise
			if len(existingPairwise) != len(desiredPairwise) {
				return true
			}
			for i, p := range desiredPairwise {
				if ps, ok := p.(string); ok {
					if i >= len(existingPairwise) || existingPairwise[i] != ps {
						return true
					}
				}
			}
		}
	}

	// Check apply_to (site vs aps)
	if desiredApplyTo, ok := desired["apply_to"].(string); ok {
		existingApplyTo := ""
		if existing.ApplyTo != nil {
			existingApplyTo = *existing.ApplyTo
		}
		if existingApplyTo != desiredApplyTo {
			return true
		}
	}

	// Check ap_ids (which APs the WLAN is assigned to)
	if desiredAPIds, ok := desired["ap_ids"].([]string); ok {
		var existingAPIds []string
		if existing.ApIDs != nil {
			existingAPIds = *existing.ApIDs
		}
		if len(existingAPIds) != len(desiredAPIds) {
			return true
		}
		sort.Strings(existingAPIds)
		sort.Strings(desiredAPIds)
		for i, id := range desiredAPIds {
			if existingAPIds[i] != id {
				return true
			}
		}
	}

	return false
}

// createWLAN creates a new WLAN at site level.
func createWLAN(ctx context.Context, client api.Client, siteID string, config map[string]any) error {
	wlan := buildMistWLANFromConfig(config)
	_, err := client.CreateSiteWLAN(ctx, siteID, wlan)
	return err
}

// updateWLAN updates an existing WLAN.
func updateWLAN(ctx context.Context, client api.Client, siteID string, wlanID string, config map[string]any) error {
	wlan := buildMistWLANFromConfig(config)
	_, err := client.UpdateSiteWLAN(ctx, siteID, wlanID, wlan)
	return err
}

// buildMistWLANFromConfig builds a MistWLAN struct from config map.
func buildMistWLANFromConfig(config map[string]any) *api.MistWLAN {
	wlan := &api.MistWLAN{
		AdditionalConfig: make(map[string]any),
	}

	if ssid, ok := config["ssid"].(string); ok {
		wlan.SSID = &ssid
	}

	if enabled, ok := config["enabled"].(bool); ok {
		wlan.Enabled = &enabled
	}

	// Handle band (legacy single value) vs bands (array for 6GHz support)
	if band, ok := config["band"].(string); ok {
		// Convert single band value to bands array for API
		bands := bandToBandsArray(band)
		wlan.Bands = &bands
		// Also set legacy band field for backwards compatibility
		wlan.Band = &band
	}

	// Handle explicit bands array (takes precedence over band)
	if bands, ok := config["bands"].([]any); ok {
		bandsArr := make([]string, 0, len(bands))
		for _, b := range bands {
			if bs, ok := b.(string); ok {
				bandsArr = append(bandsArr, bs)
			}
		}
		if len(bandsArr) > 0 {
			wlan.Bands = &bandsArr
		}
	}

	if vlanID, ok := config["vlan_id"].(float64); ok {
		v := int(vlanID)
		wlan.VlanID = &v
	} else if vlanID, ok := config["vlan_id"].(int); ok {
		wlan.VlanID = &vlanID
	}

	if auth, ok := config["auth"].(map[string]any); ok {
		if authType, ok := auth["type"].(string); ok {
			// Translate auth types to Mist API format
			// "sae" (WPA3-Personal) -> type: "psk" with pairwise: ["wpa3"]
			// "owe" (Enhanced Open) -> type: "open" with pairwise: ["owe"]
			switch authType {
			case "sae":
				// WPA3-Personal: Mist uses type=psk with pairwise=["wpa3"]
				pskType := "psk"
				wlan.Auth.Type = &pskType
				pairwise := []string{"wpa3"}
				wlan.Auth.Pairwise = &pairwise
			case "owe":
				// Enhanced Open: Mist uses type=open with owe flag
				openType := "open"
				wlan.Auth.Type = &openType
				// OWE is handled differently - add to additional config
				wlan.AdditionalConfig["auth_owe"] = true
			default:
				wlan.Auth.Type = &authType
			}
		}

		// Handle explicit pairwise if specified (overrides auto-set from sae translation)
		if pairwise, ok := auth["pairwise"].([]any); ok {
			pairwiseArr := make([]string, 0, len(pairwise))
			for _, p := range pairwise {
				if ps, ok := p.(string); ok {
					pairwiseArr = append(pairwiseArr, ps)
				}
			}
			if len(pairwiseArr) > 0 {
				wlan.Auth.Pairwise = &pairwiseArr
			}
		}

		if psk, ok := auth["psk"].(string); ok {
			// Decrypt PSK if it has the "enc:" prefix
			decryptedPSK, err := configPkg.DecryptIfNeeded(psk, "wlan.auth.psk")
			if err != nil {
				logging.Warnf("Failed to decrypt WLAN PSK: %v", err)
				// Fall back to original value (may fail at API level)
				wlan.Auth.PSK = &psk
			} else {
				wlan.Auth.PSK = &decryptedPSK
			}
		}
	}

	// Handle apply_to
	if applyTo, ok := config["apply_to"].(string); ok {
		wlan.ApplyTo = &applyTo
	}

	// Handle ap_ids
	if apIDs, ok := config["ap_ids"].([]string); ok {
		wlan.ApIDs = &apIDs
	}

	// Pass through additional config fields that aren't explicitly handled
	knownFields := map[string]bool{
		"ssid": true, "enabled": true, "band": true, "bands": true,
		"vlan_id": true, "auth": true, "hidden": true,
		"apply_to": true, "ap_ids": true,
	}
	for key, value := range config {
		if !knownFields[key] && !strings.HasPrefix(key, "_") {
			wlan.AdditionalConfig[key] = value
		}
	}

	// Handle hidden field
	if hidden, ok := config["hidden"].(bool); ok {
		wlan.Hidden = &hidden
	}

	return wlan
}

// bandToBandsArray converts a legacy band value to the bands array format.
// "dual" or "all" -> ["24", "5"]
// "5" -> ["5"]
// "24" or "2.4" -> ["24"]
// "6" -> ["6"]
// For 6GHz support, user should explicitly use bands array with "6".
func bandToBandsArray(band string) []string {
	switch strings.ToLower(band) {
	case "dual", "all":
		return []string{"24", "5"}
	case "5":
		return []string{"5"}
	case "24", "2.4":
		return []string{"24"}
	case "6":
		return []string{"6"}
	default:
		return []string{band}
	}
}

// showWLANDiff shows the differences between existing and desired WLAN config using jsondiff.
func showWLANDiff(existing *api.MistWLAN, desired map[string]any) {
	// Build existing config map for comparison
	existingMap := make(map[string]any)
	if existing.SSID != nil {
		existingMap["ssid"] = *existing.SSID
	}
	if existing.Enabled != nil {
		existingMap["enabled"] = *existing.Enabled
	}
	if existing.Band != nil {
		existingMap["band"] = *existing.Band
	}
	if existing.Bands != nil {
		existingMap["bands"] = *existing.Bands
	}
	if existing.VlanID != nil {
		existingMap["vlan_id"] = *existing.VlanID
	}
	if existing.Auth.Type != nil {
		auth := map[string]any{"type": *existing.Auth.Type}
		if existing.Auth.PSK != nil {
			auth["psk"] = "********" // Mask existing PSK
		}
		if existing.Auth.Pairwise != nil {
			auth["pairwise"] = *existing.Auth.Pairwise
		}
		existingMap["auth"] = auth
	}

	// Include apply_to and ap_ids from struct fields
	if existing.ApplyTo != nil {
		existingMap["apply_to"] = *existing.ApplyTo
	}
	if existing.ApIDs != nil {
		existingMap["ap_ids"] = *existing.ApIDs
	}

	// Mask PSK in desired config for display
	desiredDisplay := maskPSKInConfig(desired)

	showJSONDiff(existingMap, desiredDisplay, "API", "Config")
}

// showWLANConfig shows the WLAN configuration that would be created using jsondiff.
func showWLANConfig(config map[string]any) {
	// For new WLANs, show diff from empty to desired (all additions)
	emptyConfig := make(map[string]any)
	desiredDisplay := maskPSKInConfig(config)

	showJSONDiff(emptyConfig, desiredDisplay, "API", "Config")
}

// maskPSKInConfig creates a copy of config with PSK values masked.
func maskPSKInConfig(config map[string]any) map[string]any {
	displayConfig := make(map[string]any)
	for k, v := range config {
		if k == "auth" {
			if authMap, ok := v.(map[string]any); ok {
				cleanAuth := make(map[string]any)
				for ak, av := range authMap {
					if ak == "psk" {
						cleanAuth[ak] = "********"
					} else {
						cleanAuth[ak] = av
					}
				}
				displayConfig[k] = cleanAuth
			}
		} else {
			displayConfig[k] = v
		}
	}
	return displayConfig
}

// showJSONDiff displays a colorized JSON diff using jsondiff library.
func showJSONDiff(existing, desired map[string]any, existingLabel, desiredLabel string) {
	existingJSON, err1 := json.MarshalIndent(existing, "", "  ")
	desiredJSON, err2 := json.MarshalIndent(desired, "", "  ")

	if err1 != nil || err2 != nil {
		logging.Warnf("Could not generate JSON diff")
		return
	}

	opts := jsondiff.DiffOptions{
		ContextLines: 3,
		SortJSON:     true,
	}

	diffs, err := jsondiff.Diff(existingJSON, desiredJSON, opts)
	if err != nil {
		logging.Warnf("Error generating diff: %v", err)
		return
	}

	// Enhance diffs with inline changes
	diffs = jsondiff.EnhanceDiffsWithInlineChanges(diffs)

	// Format and display
	formatter := jsondiff.NewFormatter(nil)
	formatter.SetMarkers(existingLabel, desiredLabel, "Both")

	var output string
	if viper.GetBool("split_diff") {
		output = formatter.FormatSideBySide(diffs, existingLabel, desiredLabel)
	} else {
		output = formatter.Format(diffs)
	}

	if output != "" {
		fmt.Println(output)
	}
}

// collectWLANDeviceMapping builds a map of WLAN label -> list of device MACs that reference it.
// This is used to set ap_ids on WLANs for device-specific WLAN assignment.
// Priority: device-level wlan overrides site-level wlan for that device.
func collectWLANDeviceMapping(siteConfig SiteConfig) map[string][]string {
	wlanToDevices := make(map[string][]string)

	// Collect all AP MACs for site-level WLAN assignment
	allAPMACs := make([]string, 0, len(siteConfig.Devices.APs))
	devicesWithExplicitWLAN := make(map[string]bool)

	for mac, deviceMap := range siteConfig.Devices.APs {
		allAPMACs = append(allAPMACs, mac)
		// Track devices that have explicit WLAN config (they override site-level)
		if _, hasWLAN := deviceMap["wlan"]; hasWLAN {
			devicesWithExplicitWLAN[mac] = true
		}
	}

	// 1. Map site-level WLANs to APs that don't have explicit WLAN config
	for _, label := range siteConfig.WLAN {
		for _, mac := range allAPMACs {
			if !devicesWithExplicitWLAN[mac] {
				wlanToDevices[label] = append(wlanToDevices[label], mac)
			}
		}
	}
	if len(siteConfig.WLAN) > 0 {
		logging.Debugf("Site-level WLANs %v apply to %d APs without explicit config",
			siteConfig.WLAN, len(allAPMACs)-len(devicesWithExplicitWLAN))
	}

	// 2. Map device-level WLANs (these override site-level for that device)
	for mac, deviceMap := range siteConfig.Devices.APs {
		if deviceWLANs, ok := deviceMap["wlan"].([]any); ok {
			for _, w := range deviceWLANs {
				if label, ok := w.(string); ok {
					wlanToDevices[label] = append(wlanToDevices[label], mac)
				}
			}
		}
	}

	logging.Debugf("WLAN device mapping: %v", wlanToDevices)
	return wlanToDevices
}

// collectAllWLANLabels collects all unique WLAN template labels from profiles, site-level, and device configs.
// This ensures all referenced WLANs are created at the site level before device updates.
// Sources (in order of processing):
//   - profiles.wlan: WLANs to CREATE at site (make available)
//   - wlan (site-level): WLANs to APPLY to all APs
//   - devices.aps[mac].wlan: WLANs to APPLY to specific APs
func collectAllWLANLabels(siteConfig SiteConfig) []string {
	seen := make(map[string]bool)
	var labels []string

	// 1. Get WLANs to create from profiles.wlan
	for _, label := range siteConfig.Profiles.WLAN {
		if !seen[label] {
			seen[label] = true
			labels = append(labels, label)
		}
	}
	logging.Debugf("profiles.wlan (to create): %v", siteConfig.Profiles.WLAN)

	// 2. Get site-level WLANs to apply to all APs
	for _, label := range siteConfig.WLAN {
		if !seen[label] {
			seen[label] = true
			labels = append(labels, label)
		}
	}
	logging.Debugf("site-level wlan (apply to all APs): %v", siteConfig.WLAN)

	// 3. Scan AP device configs for WLAN references
	for mac, deviceMap := range siteConfig.Devices.APs {
		if deviceWLANs, ok := deviceMap["wlan"].([]any); ok {
			for _, w := range deviceWLANs {
				if label, ok := w.(string); ok && !seen[label] {
					seen[label] = true
					labels = append(labels, label)
					logging.Debugf("Device %s references WLAN: %s", mac, label)
				}
			}
		}
	}

	logging.Infof("Collected %d unique WLAN label(s): %v", len(labels), labels)
	return labels
}

// validateWLANAssignments checks that WLAN references are consistent:
//  1. Every WLAN label used in site-level "wlan" or device-level "wlan" must be
//     declared in "profiles.wlan".
//  2. Every WLAN label declared in "profiles.wlan" must have a corresponding
//     template definition in the template store.
func validateWLANAssignments(siteConfig SiteConfig, templates *configPkg.TemplateStore) error {
	profileSet := make(map[string]bool, len(siteConfig.Profiles.WLAN))
	for _, label := range siteConfig.Profiles.WLAN {
		profileSet[label] = true
	}

	var errors []string

	// Check site-level WLAN assignments
	for _, label := range siteConfig.WLAN {
		if !profileSet[label] {
			errors = append(errors, fmt.Sprintf("site-level wlan references '%s' which is not declared in profiles.wlan", label))
		}
	}

	// Check device-level WLAN assignments
	for mac, deviceMap := range siteConfig.Devices.APs {
		if deviceWLANs, ok := deviceMap["wlan"].([]any); ok {
			for _, w := range deviceWLANs {
				if label, ok := w.(string); ok {
					if !profileSet[label] {
						errors = append(errors, fmt.Sprintf("device %s wlan references '%s' which is not declared in profiles.wlan", mac, label))
					}
				}
			}
		}
	}

	// Check that every profile has a corresponding template
	if templates != nil {
		for _, label := range siteConfig.Profiles.WLAN {
			if _, found := templates.GetWLANTemplate(label); !found {
				errors = append(errors, fmt.Sprintf("profiles.wlan declares '%s' but no WLAN template with that name exists", label))
			}
		}
	}

	if len(errors) > 0 {
		sort.Strings(errors)
		msg := "WLAN configuration errors:\n"
		for _, e := range errors {
			msg += fmt.Sprintf("  - %s\n", e)
		}
		return fmt.Errorf("%s", msg)
	}

	return nil
}

// Template store for the current apply session
var (
	currentTemplateStore *configPkg.TemplateStore
	currentAPILabel      string
)

// loadTemplatesFromConfig loads templates from the configured template files
func loadTemplatesFromConfig(cfg *configPkg.Config) (*configPkg.TemplateStore, error) {
	if len(cfg.Files.Templates) == 0 {
		return configPkg.NewTemplateStore(), nil
	}

	return configPkg.LoadTemplates(cfg.Files.Templates, cfg.Files.ConfigDir)
}

// setTemplateStore sets the current template store for use by device updaters
func setTemplateStore(store *configPkg.TemplateStore, apiLabel string) {
	currentTemplateStore = store
	currentAPILabel = apiLabel
}

// getTemplateStore returns the current template store and API label
func getTemplateStore() (*configPkg.TemplateStore, string) {
	return currentTemplateStore, currentAPILabel
}

// expandDeviceConfigWithTemplates expands template references in a device config
// using the current template store. Returns the original config if templates are empty.
func expandDeviceConfigWithTemplates(deviceConfig map[string]any, siteConfig SiteConfig) (map[string]any, error) {
	templates, apiLabel := getTemplateStore()
	if templates == nil || templates.IsEmpty() {
		return deviceConfig, nil
	}

	// Extract site-level WLAN labels from siteConfig
	siteWLANs := configPkg.GetSiteWLANLabels(siteConfig.SiteConfig)

	return configPkg.ExpandDeviceConfig(deviceConfig, siteWLANs, templates, apiLabel)
}

// printWLANError prints a user-friendly error message for WLAN operations
// with context about the configuration that was attempted.
func printWLANError(operation, ssid, templateLabel string, config map[string]any, err error) {
	fmt.Printf("\n%s Failed to %s WLAN '%s'\n", symbols.ErrorPrefix(), operation, ssid)
	if templateLabel != "" {
		fmt.Printf("   Template: %s\n", templateLabel)
	}

	// Show relevant config values that might help diagnose the issue
	if auth, ok := config["auth"].(map[string]any); ok {
		if authType, ok := auth["type"].(string); ok {
			fmt.Printf("   Auth type: %s\n", authType)
		}
	}
	if band, ok := config["band"].(string); ok {
		fmt.Printf("   Band: %s\n", band)
	}

	// Print the actual error message from the API
	fmt.Printf("   Error: %v\n", err)

	// Provide helpful hints based on common error patterns
	errStr := err.Error()
	if containsIgnoreCase(errStr, "security type") || containsIgnoreCase(errStr, "6 GHz") || containsIgnoreCase(errStr, "Wi-Fi 7") {
		fmt.Printf("\n   Hint: 6GHz and Wi-Fi 7 require WPA3 security.\n")
		fmt.Printf("   Supported auth types for 6GHz: 'sae' (WPA3-Personal), 'eap-192' (WPA3-Enterprise), 'owe'\n")
		fmt.Printf("   Example: \"auth\": { \"type\": \"sae\", \"psk\": \"your-password\" }\n")
	}
	fmt.Println()
}

// containsIgnoreCase checks if a string contains a substring (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// applyWLANsMeraki applies WLAN configurations for Meraki using the vendors.Client interface.
// Uses availability tags for per-AP WLAN assignment instead of Mist's ap_ids/apply_to model.
func applyWLANsMeraki(ctx context.Context, _ *configPkg.Config, siteConfig SiteConfig, siteID, apiLabel string,
	_ []string, wlanToDevices map[string][]string, desiredWLANs []map[string]any, diffMode, force bool) (int, error) {

	// Get vendor client from global registry
	registry := vendors.GetGlobalRegistry()
	if registry == nil {
		return 0, fmt.Errorf("vendor registry not initialized")
	}
	vendorClient, err := registry.GetClient(apiLabel)
	if err != nil {
		return 0, fmt.Errorf("failed to get vendor client for %s: %w", apiLabel, err)
	}
	wlansSvc := vendorClient.WLANs()
	if wlansSvc == nil {
		return 0, fmt.Errorf("vendor %s does not support WLANs", apiLabel)
	}

	// Get existing WLANs for this network
	existingWLANs, err := wlansSvc.ListBySite(ctx, siteID)
	if err != nil {
		logging.Warnf("Failed to get existing WLANs for site %s: %v", siteID, err)
		existingWLANs = nil
	}

	// Build lookup by SSID
	existingBySSID := make(map[string]*vendors.WLAN)
	for _, w := range existingWLANs {
		if w.SSID != "" {
			existingBySSID[w.SSID] = w
		}
	}
	logging.Debugf("Found %d existing Meraki SSIDs in network", len(existingBySSID))

	// Build AP tag mapping for device update phase
	tagMapping := buildAPTagMapping(wlanToDevices)
	setAPTagMapping(tagMapping)

	changeCount := 0

	for _, desired := range desiredWLANs {
		ssid, ok := desired["ssid"].(string)
		if !ok || ssid == "" {
			logging.Warnf("WLAN template has no ssid field, skipping")
			continue
		}

		templateLabel := ""
		if label, ok := desired["_template_label"].(string); ok {
			templateLabel = label
			delete(desired, "_template_label")
		}

		// Build vendor WLAN from expanded template config
		wlan := buildVendorWLANFromConfig(desired, siteID)

		// Set availability tags based on device mapping
		if deviceMACs, hasDevices := wlanToDevices[templateLabel]; hasDevices && len(deviceMACs) > 0 {
			// WLAN is assigned to specific APs
			tag := generateWLANAvailabilityTag(templateLabel)
			wlan.Config["availabilityTags"] = []string{tag}
			wlan.Config["availableOnAllAps"] = false
			logging.Infof("WLAN '%s' restricted to %d AP(s) via tag '%s'", templateLabel, len(deviceMACs), tag)
		} else if slices.Contains(siteConfig.WLAN, templateLabel) {
			// Site-level WLAN but all APs have overrides (no applicable APs)
			tag := generateWLANAvailabilityTag(templateLabel)
			wlan.Config["availabilityTags"] = []string{tag}
			wlan.Config["availableOnAllAps"] = false
			logging.Infof("WLAN '%s' (site-level, no applicable APs) via tag '%s'", templateLabel, tag)
		} else {
			// Profile-only WLAN: available on all APs
			allAPs := true
			wlan.Config["availableOnAllAps"] = allAPs
			logging.Debugf("WLAN '%s' (profile-only) will broadcast on all APs", templateLabel)
		}

		existing, exists := existingBySSID[ssid]

		if exists {
			needsUpdate := merakiWLANNeedsUpdate(existing, wlan)
			if needsUpdate || force {
				if diffMode {
					if force && !needsUpdate {
						fmt.Printf("Would force update WLAN '%s' (template: %s) - no changes detected\n", ssid, templateLabel)
					} else {
						fmt.Printf("Would update WLAN '%s' (template: %s)\n", ssid, templateLabel)
					}
				} else {
					logging.Infof("Updating Meraki SSID '%s' (template: %s)", ssid, templateLabel)
					_, err := wlansSvc.Update(ctx, existing.ID, wlan)
					if err != nil {
						logging.Errorf("Failed to update Meraki SSID '%s': %v", ssid, err)
						fmt.Printf("%s Failed to update WLAN '%s': %v\n", symbols.ErrorPrefix(), ssid, err)
						continue
					}
					fmt.Printf("%s Updated WLAN '%s'\n", symbols.SuccessPrefix(), ssid)
				}
				changeCount++
			} else {
				logging.Debugf("Meraki SSID '%s' is up to date", ssid)
			}
		} else {
			if diffMode {
				fmt.Printf("Would create WLAN '%s' (template: %s)\n", ssid, templateLabel)
			} else {
				logging.Infof("Creating Meraki SSID '%s' (template: %s)", ssid, templateLabel)
				_, err := wlansSvc.Create(ctx, wlan)
				if err != nil {
					logging.Errorf("Failed to create Meraki SSID '%s': %v", ssid, err)
					fmt.Printf("%s Failed to create WLAN '%s': %v\n", symbols.ErrorPrefix(), ssid, err)
					continue
				}
				fmt.Printf("%s Created WLAN '%s'\n", symbols.SuccessPrefix(), ssid)
			}
			changeCount++
		}
	}

	if changeCount > 0 && !diffMode {
		logging.Infof("Applied %d Meraki WLAN change(s)", changeCount)
	}

	return changeCount, nil
}

// buildVendorWLANFromConfig converts an expanded WLAN template config map to a vendors.WLAN.
func buildVendorWLANFromConfig(config map[string]any, siteID string) *vendors.WLAN {
	wlan := &vendors.WLAN{
		SiteID: siteID,
		Config: make(map[string]interface{}),
	}

	if ssid, ok := config["ssid"].(string); ok {
		wlan.SSID = ssid
	}

	if enabled, ok := config["enabled"].(bool); ok {
		wlan.Enabled = enabled
	}

	if hidden, ok := config["hidden"].(bool); ok {
		wlan.Hidden = hidden
	}

	// Band selection
	if band, ok := config["band"].(string); ok {
		wlan.Band = band
	}

	// VLAN
	if vlanID, ok := config["vlan_id"].(float64); ok {
		wlan.VLANID = int(vlanID)
	} else if vlanID, ok := config["vlan_id"].(int); ok {
		wlan.VLANID = vlanID
	}

	// Auth type and encryption
	if auth, ok := config["auth"].(map[string]any); ok {
		if authType, ok := auth["type"].(string); ok {
			wlan.AuthType = authType
		}
		if psk, ok := auth["psk"].(string); ok {
			decrypted, err := configPkg.DecryptIfNeeded(psk, "wlan.auth.psk")
			if err != nil {
				logging.Warnf("Failed to decrypt WLAN PSK: %v", err)
				wlan.PSK = psk
			} else {
				wlan.PSK = decrypted
			}
		}
	}

	if encMode, ok := config["encryption_mode"].(string); ok {
		wlan.EncryptionMode = encMode
	}

	// Pass remaining fields into Config for vendor-specific handling
	knownFields := map[string]bool{
		"ssid": true, "enabled": true, "hidden": true, "band": true,
		"vlan_id": true, "auth": true, "encryption_mode": true,
	}
	for key, value := range config {
		if !knownFields[key] && !strings.HasPrefix(key, "_") {
			wlan.Config[key] = value
		}
	}

	return wlan
}

// merakiWLANNeedsUpdate checks if a Meraki WLAN needs updating by comparing key fields.
func merakiWLANNeedsUpdate(existing *vendors.WLAN, desired *vendors.WLAN) bool {
	if existing.SSID != desired.SSID {
		return true
	}
	if existing.Enabled != desired.Enabled {
		return true
	}
	if existing.Hidden != desired.Hidden {
		return true
	}
	if desired.Band != "" && existing.Band != desired.Band {
		return true
	}
	if desired.VLANID != 0 && existing.VLANID != desired.VLANID {
		return true
	}
	if desired.AuthType != "" && existing.AuthType != desired.AuthType {
		return true
	}
	if desired.EncryptionMode != "" && existing.EncryptionMode != desired.EncryptionMode {
		return true
	}

	// Compare availability tags
	existingTags := extractStringSliceFromConfig(existing.Config, "availabilityTags")
	desiredTags := extractStringSliceFromConfig(desired.Config, "availabilityTags")
	if !stringSlicesEqual(existingTags, desiredTags) {
		return true
	}

	// Compare availableOnAllAps
	existingAllAPs := getBoolFromConfig(existing.Config, "availableOnAllAps")
	desiredAllAPs := getBoolFromConfig(desired.Config, "availableOnAllAps")
	return existingAllAPs != desiredAllAPs
}

func extractStringSliceFromConfig(config map[string]interface{}, key string) []string {
	if config == nil {
		return nil
	}
	v, ok := config[key]
	if !ok {
		return nil
	}
	switch val := v.(type) {
	case []string:
		result := make([]string, len(val))
		copy(result, val)
		sort.Strings(result)
		return result
	case []any:
		result := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		sort.Strings(result)
		return result
	default:
		return nil
	}
}

func getBoolFromConfig(config map[string]interface{}, key string) bool {
	if config == nil {
		return true // default: availableOnAllAps = true
	}
	v, ok := config[key]
	if !ok {
		return true
	}
	switch val := v.(type) {
	case bool:
		return val
	case *bool:
		if val != nil {
			return *val
		}
		return true
	default:
		return true
	}
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

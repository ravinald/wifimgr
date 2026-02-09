package apply

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/ravinald/wifimgr/api"
	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/macaddr"
	"github.com/ravinald/wifimgr/internal/symbols"
	"github.com/ravinald/wifimgr/internal/vendors"
	"github.com/ravinald/wifimgr/internal/xdg"
)

// HandleCommand processes apply-related subcommands
func HandleCommand(ctx context.Context, client api.Client, cfg *config.Config, args []string, apiLabel string, force bool) error {
	if len(args) < 2 {
		logging.Error("Not enough parameters provided for apply command")
		return fmt.Errorf("apply command requires at least 2 parameters: <site_name> <device_type|all>")
	}

	// Extract site name and command/device type
	siteName := args[0]
	command := args[1]

	// Check if diff, split, and refresh-api positional arguments are present
	diffMode := false
	splitDiff := false
	refreshAPI := false
	for _, arg := range args[2:] {
		switch arg {
		case "diff":
			diffMode = true
		case "split":
			splitDiff = true
		case "refresh-api":
			refreshAPI = true
		}
	}
	// Set viper values for use in diff display functions
	if diffMode {
		viper.Set("show_diff", true)
	}
	if splitDiff {
		viper.Set("split_diff", true)
	}

	// Handle backup management commands
	switch command {
	case "rollback":
		return handleRollbackCommand(ctx, client, cfg, siteName, args[2:])
	case "list-backups":
		return handleListBackupsCommand(cfg, siteName)
	case "cleanup-backups":
		return handleCleanupBackupsCommand(cfg, args[2:])
	case "validate-backup":
		return handleValidateBackupCommand(args[2:])
	case "device-profile":
		// Handle device-profile apply command
		deviceFilter := "all"
		if len(args) > 2 {
			deviceFilter = args[2]
		}
		return applyDeviceProfiles(ctx, client, cfg, siteName, deviceFilter, force, diffMode)
	}

	// Standard device type apply command
	deviceType := command
	logging.Infof("Executing apply command for site: %s, device type: %s, API: %s", siteName, deviceType, apiLabel)

	// Handle the "all" case - currently only AP is supported
	if deviceType == "all" {
		// Apply AP configuration (only supported device type)
		if err := applyDeviceToSite(ctx, client, cfg, siteName, "ap", apiLabel, force, diffMode, refreshAPI); err != nil {
			logging.Errorf("Error applying AP configuration to site %s: %v", siteName, err)
			return fmt.Errorf("AP apply error: %w", err)
		}

		// Note: Switch and gateway support planned for future release
		logging.Debugf("Skipping switch and gateway - not yet supported")

		return nil
	}

	// Apply specific device type
	return applyDeviceToSite(ctx, client, cfg, siteName, deviceType, apiLabel, force, diffMode, refreshAPI)
}

// applyDeviceToSite applies a specific device type configuration to a site
func applyDeviceToSite(ctx context.Context, client api.Client, cfg *config.Config, siteName string, deviceType string, apiLabel string, force bool, diffMode bool, refreshAPI bool) error {
	// Use the new generic framework
	return applySiteGeneric(ctx, client, cfg, siteName, deviceType, apiLabel, force, diffMode, refreshAPI)
}

// Helper functions

// SiteConfig represents a site configuration in the config file
type SiteConfig struct {
	SiteConfig map[string]any `json:"site_config"`
	Profiles   struct {
		WLAN   []string `json:"wlan,omitempty"`   // WLAN template labels to CREATE at site
		Radio  []string `json:"radio,omitempty"`  // Radio template labels
		Device []string `json:"device,omitempty"` // Device template labels
	} `json:"profiles,omitempty"`
	WLAN    []string `json:"wlan,omitempty"` // WLANs to APPLY to all APs (site-wide default)
	Devices struct {
		APs      map[string]map[string]any `json:"ap"`      // AP is a map of MAC -> config
		Switches map[string]map[string]any `json:"switch"`  // Switch is a map of MAC -> config
		WanEdge  map[string]map[string]any `json:"gateway"` // Gateway is a map of MAC -> config
	} `json:"devices"`
	LastModified string `json:"last_modified,omitempty"` // UTC timestamp when config was last modified
}

// ConfigFileStructure represents the structure of a site config file
type ConfigFileStructure struct {
	Version      int           `json:"version"`
	Config       ConfigWrapper `json:"config"`                  // New format with sites wrapper
	LastModified string        `json:"last_modified,omitempty"` // UTC timestamp when file was last modified
}

// ConfigWrapper wraps the sites map
type ConfigWrapper struct {
	Sites map[string]SiteConfig `json:"sites"`
}

// getSiteConfigsFromFiles reads and parses site configurations from config files
func getSiteConfigsFromFiles(configFiles []string) (map[string]SiteConfig, error) {
	siteConfigs := make(map[string]SiteConfig)

	// Get the config directory from the environment or XDG default
	configDir := os.Getenv("CONFIG_DIR")
	if configDir == "" {
		configDir = xdg.GetConfigDir()
	}

	for _, fileName := range configFiles {
		// Join with the config directory if the path is not absolute
		filePath := fileName
		if !filepath.IsAbs(filePath) {
			filePath = filepath.Join(configDir, fileName)
		}

		fileData, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("error reading config file %s: %w", filePath, err)
		}

		var fileConfig ConfigFileStructure
		if err := json.Unmarshal(fileData, &fileConfig); err != nil {
			return nil, fmt.Errorf("error parsing config file %s: %w", filePath, err)
		}

		// Get sites from the config.sites wrapper
		sitesMap := fileConfig.Config.Sites

		// Add sites from this file to the combined map using the actual site name
		for configID, siteConfig := range sitesMap {
			// Extract the actual site name from site_config.name
			siteName, ok := getSiteNameFromConfig(siteConfig)
			if !ok {
				logging.Warnf("Site configuration with ID %s does not have a valid name, using config ID as fallback", configID)
				siteName = configID // Fallback to config ID if name is not available
			}

			siteConfigs[siteName] = siteConfig
		}
	}

	return siteConfigs, nil
}

// getSiteNameFromConfig extracts the site name from the site_config section
func getSiteNameFromConfig(siteConfig SiteConfig) (string, bool) {
	if nameVal, ok := siteConfig.SiteConfig["name"]; ok {
		if name, ok := nameVal.(string); ok && name != "" {
			return name, true
		}
	}
	return "", false
}

// getSiteIDByName gets the site ID for a site name.
// First checks the multi-vendor cache, then falls back to API lookup.
func getSiteIDByName(client api.Client, siteName string) (string, error) {
	// Try cache first (multi-vendor cache)
	if accessor := vendors.GetGlobalCacheAccessor(); accessor != nil {
		if site, err := accessor.GetSiteByName(siteName); err == nil && site.ID != "" {
			logging.Debugf("Found site ID for '%s' in cache: %s", siteName, site.ID)
			return site.ID, nil
		}
	}

	// Fall back to API lookup
	ctx := context.Background()
	site, err := client.GetSiteByIdentifier(ctx, siteName)
	if err != nil {
		return "", fmt.Errorf("failed to get site by name '%s': %w", siteName, err)
	}

	if site.ID == nil {
		return "", fmt.Errorf("site '%s' has no ID", siteName)
	}

	return *site.ID, nil
}

// ConfigurationBackup represents a backup of device configurations for rollback
type ConfigurationBackup struct {
	Timestamp      int64                     `json:"timestamp"`
	SiteID         string                    `json:"site_id"`
	SiteName       string                    `json:"site_name"`
	Operation      string                    `json:"operation"`
	DeviceCount    int                       `json:"device_count"`
	Devices        map[string]map[string]any `json:"devices"` // MAC -> config
	BackupFilePath string                    `json:"-"`       // Not serialized
}

// Backups are now file-based using createConfigBackupAfterApply
// which creates config file backups in the format: <config-filename>.json.<index>
// Rollback is also file-based - see rollbackConfigFile()

// listConfigurationBackups lists available configuration backups for a site
func listConfigurationBackups(cfg *config.Config, siteName string) ([]ConfigurationBackup, error) {
	// Use the new rotation-aware listing
	return listBackupsWithRotation(cfg, siteName)
}

// cleanupOldBackups removes backup files based on rotation and age policies
// Note: maxAgeDays is currently unused - cleanup uses config.Files.ConfigBackups count instead
func cleanupOldBackups(cfg *config.Config, _ int) error {
	// Use the new config-aware cleanup that respects rotation limits
	return cleanupBackupsWithConfig(cfg)
}

// Command handlers for backup management CLI interface
// ============================================================================

// handleRollbackCommand handles file-based rollback of intent configuration
// This does NOT send anything to the API - it only manipulates config files.
// The operator can then review, edit, diff, and explicitly apply when ready.
func handleRollbackCommand(_ context.Context, _ api.Client, cfg *config.Config, siteName string, args []string) error {
	// Parse backup index (default 0 = most recent backup)
	backupIndex := 0
	if len(args) > 0 {
		if _, err := fmt.Sscanf(args[0], "%d", &backupIndex); err != nil {
			return fmt.Errorf("invalid backup index: %s (expected a number)", args[0])
		}
	}

	// Find the config file that contains this site
	configFilePath, err := findConfigFileForSite(cfg, siteName)
	if err != nil {
		return err
	}

	// Perform file-based rollback
	return rollbackConfigFile(cfg, siteName, configFilePath, backupIndex)
}

// findConfigFileForSite finds the config file path that contains the specified site
func findConfigFileForSite(cfg *config.Config, siteName string) (string, error) {
	for _, configFile := range cfg.Files.SiteConfigs {
		filePath := configFile
		if !filepath.IsAbs(filePath) {
			filePath = filepath.Join(cfg.Files.ConfigDir, configFile)
		}

		fileData, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		var fileConfig ConfigFileStructure
		if err := json.Unmarshal(fileData, &fileConfig); err != nil {
			continue
		}

		// Check the config.sites section
		for _, siteConfig := range fileConfig.Config.Sites {
			if nameVal, ok := siteConfig.SiteConfig["name"]; ok {
				if name, ok := nameVal.(string); ok && name == siteName {
					return filePath, nil
				}
			}
		}
	}

	return "", fmt.Errorf("site %s not found in any configuration file", siteName)
}

// rollbackConfigFile performs a file-based rollback:
// 1. Rotates existing backups (increment indices)
// 2. Copies current intent config to new .0 backup
// 3. Copies selected backup to become current intent config
func rollbackConfigFile(cfg *config.Config, siteName string, configFilePath string, backupIndex int) error {
	backupDir := filepath.Join(cfg.Files.ConfigDir, "backups")
	baseFileName := filepath.Base(configFilePath)

	// Verify the backup file exists
	backupFilePath := filepath.Join(backupDir, fmt.Sprintf("%s.%d", baseFileName, backupIndex))
	if _, err := os.Stat(backupFilePath); os.IsNotExist(err) {
		return fmt.Errorf("backup file not found: %s.%d", baseFileName, backupIndex)
	}

	// Get max backups from config
	maxBackups := 10
	if cfg.Files.ConfigBackups > 0 {
		maxBackups = cfg.Files.ConfigBackups
	}

	fmt.Printf("Rolling back site %s from backup index %d\n", siteName, backupIndex)
	fmt.Printf("  Config file: %s\n", configFilePath)
	fmt.Printf("  Backup file: %s\n", backupFilePath)

	// Step 1: Rotate existing backups to make room for new .0
	logging.Debugf("Rotating existing backups")
	if err := rotateConfigFileBackups(backupDir, baseFileName, maxBackups); err != nil {
		return fmt.Errorf("failed to rotate backups: %w", err)
	}

	// Step 2: Copy current intent config to new .0 backup
	logging.Debugf("Backing up current config to .0")
	currentData, err := os.ReadFile(configFilePath)
	if err != nil {
		return fmt.Errorf("failed to read current config: %w", err)
	}

	// Add last_modified timestamp to the backup
	var configData map[string]any
	if err := json.Unmarshal(currentData, &configData); err != nil {
		return fmt.Errorf("failed to parse current config: %w", err)
	}
	configData["last_modified"] = time.Now().UTC().Format(time.RFC3339)

	backupData, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal backup: %w", err)
	}

	newBackupPath := filepath.Join(backupDir, fmt.Sprintf("%s.0", baseFileName))
	if err := os.WriteFile(newBackupPath, backupData, 0644); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}
	fmt.Printf("  Created backup: %s.0 (previous config)\n", baseFileName)

	// Step 3: Copy selected backup to current intent config
	// Note: After rotation, the backup index has shifted by 1
	shiftedBackupPath := filepath.Join(backupDir, fmt.Sprintf("%s.%d", baseFileName, backupIndex+1))
	logging.Debugf("Restoring from %s to %s", shiftedBackupPath, configFilePath)

	restoreData, err := os.ReadFile(shiftedBackupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	if err := os.WriteFile(configFilePath, restoreData, 0644); err != nil {
		return fmt.Errorf("failed to restore config: %w", err)
	}

	fmt.Printf("  Restored: %s.%d -> %s\n", baseFileName, backupIndex+1, baseFileName)
	fmt.Printf("\nRollback complete. The configuration has NOT been applied to the API.\n")
	fmt.Printf("To review changes: wifimgr apply site %s ap diff\n", siteName)
	fmt.Printf("To apply changes:  wifimgr apply site %s ap\n", siteName)

	return nil
}

// handleListBackupsCommand lists available backups for a site
func handleListBackupsCommand(cfg *config.Config, siteName string) error {
	backups, err := listConfigurationBackups(cfg, siteName)
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	if len(backups) == 0 {
		fmt.Printf("No backups found for site %s\n", siteName)
		return nil
	}

	fmt.Printf("Configuration backups for site %s:\n\n", siteName)
	fmt.Printf("%-20s %-10s %-15s %s\n", "Timestamp", "Devices", "Operation", "File")
	fmt.Printf("%-20s %-10s %-15s %s\n", "────────────────────", "───────", "──────────────", "────")

	for _, backup := range backups {
		timestamp := time.Unix(backup.Timestamp, 0).Format("2006-01-02 15:04:05")
		fileName := filepath.Base(backup.BackupFilePath)
		fmt.Printf("%-20s %-10d %-15s %s\n", timestamp, backup.DeviceCount, backup.Operation, fileName)
	}

	fmt.Printf("\nUse 'apply rollback %s [index]' to restore from a backup (default: 0 = most recent)\n", siteName)
	return nil
}

// handleCleanupBackupsCommand cleans up old backup files
func handleCleanupBackupsCommand(cfg *config.Config, args []string) error {
	// Default from config, fallback to 30 days if not set
	maxAgeDays := viper.GetInt("backup.retention_days")
	if maxAgeDays == 0 {
		maxAgeDays = 30
	}

	if len(args) > 0 {
		if args[0] == "--days" && len(args) > 1 {
			var err error
			maxAgeDays, err = fmt.Sscanf(args[1], "%d", &maxAgeDays)
			if err != nil {
				return fmt.Errorf("invalid days value: %s", args[1])
			}
		}
	}

	fmt.Printf("Cleaning up backups older than %d days...\n", maxAgeDays)
	return cleanupOldBackups(cfg, maxAgeDays)
}

// handleValidateBackupCommand validates the integrity of a config backup file
func handleValidateBackupCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("backup file path required for validation")
	}

	backupFile := args[0]
	if !filepath.IsAbs(backupFile) {
		// Try current directory first, then assume relative to XDG backups
		if _, err := os.Stat(backupFile); os.IsNotExist(err) {
			backupFile = filepath.Join(xdg.GetBackupsDir(), backupFile)
		}
	}

	fmt.Printf("Validating backup file: %s\n", backupFile)

	// Read backup file
	backupData, err := os.ReadFile(backupFile)
	if err != nil {
		fmt.Printf("%s Validation failed: cannot read file: %v\n", symbols.FailurePrefix(), err)
		return err
	}

	// Parse as config file structure
	var configData ConfigFileStructure
	if err := json.Unmarshal(backupData, &configData); err != nil {
		fmt.Printf("%s Validation failed: invalid JSON: %v\n", symbols.FailurePrefix(), err)
		return err
	}

	// Check for required fields
	if configData.Version == 0 {
		fmt.Printf("Warning: missing version field\n")
	}

	// Count sites and devices
	siteCount := 0
	deviceCount := 0
	var siteNames []string

	// Check the config.sites section
	for _, siteConfig := range configData.Config.Sites {
		siteCount++
		if nameVal, ok := siteConfig.SiteConfig["name"]; ok {
			if name, ok := nameVal.(string); ok {
				siteNames = append(siteNames, name)
			}
		}
		deviceCount += len(siteConfig.Devices.APs)
		deviceCount += len(siteConfig.Devices.Switches)
		deviceCount += len(siteConfig.Devices.WanEdge)
	}

	if siteCount == 0 {
		fmt.Printf("%s Validation failed: no sites found in backup\n", symbols.FailurePrefix())
		return fmt.Errorf("no sites found in backup file")
	}

	fmt.Printf("%s Backup validation passed\n\n", symbols.SuccessPrefix())
	fmt.Printf("Backup Details:\n")
	fmt.Printf("  Version: %d\n", configData.Version)
	fmt.Printf("  Sites: %d (%s)\n", siteCount, strings.Join(siteNames, ", "))
	fmt.Printf("  Total Devices: %d\n", deviceCount)

	// Show last_modified if present
	var rawData map[string]any
	if err := json.Unmarshal(backupData, &rawData); err == nil {
		if lastModified, ok := rawData["last_modified"].(string); ok {
			fmt.Printf("  Last Modified: %s\n", lastModified)
		}
	}

	return nil
}

// applyDeviceProfiles applies device profile configurations to APs in a site
func applyDeviceProfiles(ctx context.Context, client api.Client, cfg *config.Config, siteName string, deviceFilter string, force bool, diffMode bool) error {
	logging.Infof("Applying device profile configuration to site: %s, device filter: %s", siteName, deviceFilter)

	// Step 1: Get the site configuration
	configFiles := cfg.Files.SiteConfigs
	if len(configFiles) == 0 {
		logging.Error("No site configuration files defined in config")
		return fmt.Errorf("no site configuration files defined in config")
	}

	// Get site configurations
	siteConfigs, err := getSiteConfigsFromFiles(configFiles)
	if err != nil {
		logging.Errorf("Error reading site configurations: %v", err)
		return fmt.Errorf("error reading site configurations: %v", err)
	}

	// Find the site configuration
	siteConfig, found := siteConfigs[siteName]
	if !found {
		logging.Errorf("Site %s not found in configuration files", siteName)
		return fmt.Errorf("site %s not found in configuration files", siteName)
	}

	// Step 2: Get site ID
	siteID, err := getSiteIDByName(client, siteName)
	if err != nil {
		logging.Errorf("Error getting site ID for %s: %v", siteName, err)
		return fmt.Errorf("error getting site ID for %s: %v", siteName, err)
	}

	// Step 3: Get configured APs with device profiles and store device names
	apsWithProfiles := make(map[string]string)       // MAC -> deviceprofile_name
	deviceNamesFromConfig := make(map[string]string) // MAC -> device name

	for _, apConfig := range siteConfig.Devices.APs {
		if macAddr, ok := apConfig["mac"]; ok {
			if macStr, ok := macAddr.(string); ok {
				normalizedMAC := macaddr.NormalizeOrEmpty(macStr)
				if normalizedMAC != "" {
					// Store device name if available
					if deviceName, ok := apConfig["name"]; ok {
						if name, ok := deviceName.(string); ok {
							deviceNamesFromConfig[normalizedMAC] = name

							// Filter by device name if specified
							if deviceFilter != "all" && name != deviceFilter {
								continue
							}
						}
					}

					// Check for device profile
					if profileName, ok := apConfig["deviceprofile_name"]; ok {
						if profileStr, ok := profileName.(string); ok && profileStr != "" {
							apsWithProfiles[normalizedMAC] = profileStr
						}
					}
				}
			}
		}
	}

	if len(apsWithProfiles) == 0 && deviceFilter != "all" {
		return fmt.Errorf("no device found with name: %s", deviceFilter)
	}

	// Step 4: Get device profile name to ID mapping
	profileNameToID := make(map[string]string)
	profiles, err := client.GetDeviceProfiles(ctx, cfg.API.Credentials.OrgID, "")
	if err != nil {
		logging.Errorf("Error getting device profiles: %v", err)
		return fmt.Errorf("error getting device profiles: %v", err)
	}

	for _, profile := range profiles {
		if profile.Name != nil && profile.ID != nil {
			profileNameToID[*profile.Name] = *profile.ID
		}
	}

	// Step 5: Get current device profile assignments
	devices, err := client.GetDevicesByType(ctx, siteID, "ap")
	if err != nil {
		logging.Errorf("Error getting devices: %v", err)
		return fmt.Errorf("error getting devices: %v", err)
	}

	currentAssignments := make(map[string]string) // MAC -> deviceprofile_id
	for _, device := range devices {
		if device.MAC != nil && device.DeviceProfileID != nil && *device.DeviceProfileID != "" {
			normalizedMAC := macaddr.NormalizeOrEmpty(*device.MAC)
			if normalizedMAC != "" {
				currentAssignments[normalizedMAC] = *device.DeviceProfileID
			}
		}
	}

	// Step 6: Determine assignments and unassignments
	toAssign := make(map[string][]string)   // profile_id -> []MACs
	toUnassign := make(map[string][]string) // profile_id -> []MACs
	deviceNames := make(map[string]string)  // MAC -> device name for display

	// Check devices that need profile assignment/update
	for mac, profileName := range apsWithProfiles {
		profileID, found := profileNameToID[profileName]
		if !found {
			logging.Errorf("Device profile '%s' not found for AP %s", profileName, mac)
			return fmt.Errorf("device profile '%s' not found", profileName)
		}

		currentProfileID, hasProfile := currentAssignments[mac]
		if !hasProfile || currentProfileID != profileID {
			// Need to assign this profile
			if toAssign[profileID] == nil {
				toAssign[profileID] = make([]string, 0)
			}
			toAssign[profileID] = append(toAssign[profileID], mac)
			// Store device name for display
			if name, ok := deviceNamesFromConfig[mac]; ok {
				deviceNames[mac] = name
			}
		}
	}

	// Check devices that need profile unassignment (only if filter is "all")
	if deviceFilter == "all" {
		for mac, currentProfileID := range currentAssignments {
			if _, shouldHaveProfile := apsWithProfiles[mac]; !shouldHaveProfile && currentProfileID != "" {
				// Group by profile ID for unassignment
				if toUnassign[currentProfileID] == nil {
					toUnassign[currentProfileID] = make([]string, 0)
				}
				toUnassign[currentProfileID] = append(toUnassign[currentProfileID], mac)
				// Get device name from API cache
				for _, device := range devices {
					if device.MAC != nil && macaddr.NormalizeOrEmpty(*device.MAC) == mac {
						if device.Name != nil {
							deviceNames[mac] = *device.Name
						}
						break
					}
				}
			}
		}
	}

	// Step 7: Apply changes
	changesMade := false

	// Unassign profiles
	if len(toUnassign) > 0 {
		changesMade = true
		totalToUnassign := 0
		for _, macs := range toUnassign {
			totalToUnassign += len(macs)
		}

		if diffMode {
			fmt.Printf("Would unassign device profiles from %d APs:\n", totalToUnassign)
			for profileID, macs := range toUnassign {
				// Find profile name for display
				profileName := profileID
				for name, id := range profileNameToID {
					if id == profileID {
						profileName = name
						break
					}
				}
				fmt.Printf("Profile '%s':\n", profileName)
				for _, mac := range macs {
					if deviceName, ok := deviceNames[mac]; ok {
						fmt.Printf("  - %s (%s)\n", deviceName, mac)
					} else {
						fmt.Printf("  - %s\n", mac)
					}
				}
			}
		} else {
			logging.Infof("Unassigning device profiles from %d APs", totalToUnassign)
			for profileID, macs := range toUnassign {
				err := client.UnassignDeviceProfiles(ctx, cfg.API.Credentials.OrgID, profileID, macs)
				if err != nil {
					logging.Errorf("Error unassigning device profile %s: %v", profileID, err)
					return fmt.Errorf("error unassigning device profile: %v", err)
				}
			}
			fmt.Printf("Successfully unassigned device profiles from %d APs\n", totalToUnassign)
		}
	}

	// Assign profiles
	for profileID, macs := range toAssign {
		changesMade = true
		profileName := ""
		for name, id := range profileNameToID {
			if id == profileID {
				profileName = name
				break
			}
		}

		if diffMode {
			fmt.Printf("Would assign device profile '%s' to %d APs:\n", profileName, len(macs))
			for _, mac := range macs {
				if deviceName, ok := deviceNames[mac]; ok {
					fmt.Printf("- %s (%s)\n", deviceName, mac)
				} else {
					fmt.Printf("- %s\n", mac)
				}
			}
		} else {
			logging.Infof("Assigning device profile '%s' to %d APs", profileName, len(macs))
			result, err := client.AssignDeviceProfile(ctx, cfg.API.Credentials.OrgID, profileID, macs)
			if err != nil {
				logging.Errorf("Error assigning device profile: %v", err)
				return fmt.Errorf("error assigning device profile '%s': %v", profileName, err)
			}

			// Check results
			if result.Success != nil && len(result.Success) != len(macs) {
				// Some assignments failed
				failedMACs := make([]string, 0)
				successMap := make(map[string]bool)
				for _, mac := range result.Success {
					successMap[mac] = true
				}
				for _, mac := range macs {
					if !successMap[mac] {
						failedMACs = append(failedMACs, mac)
					}
				}
				if len(failedMACs) > 0 {
					logging.Errorf("Failed to assign profile to %d devices", len(failedMACs))
					for _, mac := range failedMACs {
						// Try to find device name
						deviceName := mac
						for _, apConfig := range siteConfig.Devices.APs {
							if configMAC, ok := apConfig["mac"]; ok {
								if configMACStr, ok := configMAC.(string); ok {
									normalizedConfigMAC := macaddr.NormalizeOrEmpty(configMACStr)
									if normalizedConfigMAC == mac {
										if name, ok := apConfig["name"]; ok {
											if nameStr, ok := name.(string); ok {
												deviceName = nameStr
											}
										}
										break
									}
								}
							}
						}
						fmt.Printf("- Failed to assign profile to %s\n", deviceName)
					}
					return fmt.Errorf("failed to assign profile to %d devices", len(failedMACs))
				}
			}

			fmt.Printf("Successfully assigned device profile '%s' to %d APs\n", profileName, len(macs))
		}
	}

	if !changesMade {
		if diffMode {
			fmt.Println("No device profile changes needed")
		} else if !force {
			fmt.Println("No device profile changes needed - all devices already have correct profiles. Use --force to apply anyway.")
			return nil
		} else {
			fmt.Println("No device profile changes detected, but --force flag is set. Proceeding with apply.")
		}
	} else if diffMode {
		fmt.Println("\nDiff mode completed - no changes have been applied")
	} else {
		// Refresh cache for APs in this site after changes
		logging.Infof("Refreshing cache for site %s APs after device profile changes", siteName)
		err := client.UpdateCacheForTypes(ctx, []string{"ap"}, []string{siteName})
		if err != nil {
			logging.Warnf("Failed to refresh cache after device profile changes: %v", err)
			// Don't fail the operation, just warn
		} else {
			logging.Infof("Cache refreshed successfully for site %s APs", siteName)
		}
	}

	return nil
}

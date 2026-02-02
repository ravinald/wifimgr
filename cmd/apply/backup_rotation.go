package apply

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/xdg"
)

// createRotatedBackup function removed - legacy backup format no longer used
// Backups are now created by createConfigBackupAfterApply in the format: <config-filename>.json.<index>

// rotateConfigFileBackups rotates existing backup files for a config file
func rotateConfigFileBackups(backupDir string, baseFileName string, maxBackups int) error {
	// Get all backup files for this config file
	// Pattern: baseFileName.0, baseFileName.1, etc.
	basePattern := filepath.Join(backupDir, baseFileName)

	// Find existing backups with serial numbers
	var existingBackups []string
	for i := 0; i < maxBackups; i++ {
		backupPath := fmt.Sprintf("%s.%d", basePattern, i)
		if _, err := os.Stat(backupPath); err == nil {
			existingBackups = append(existingBackups, backupPath)
		}
	}

	// Rotate backups by incrementing their serial numbers
	// Start from the highest number and work backwards
	for i := len(existingBackups) - 1; i >= 0; i-- {
		oldPath := existingBackups[i]
		newSerial := i + 1

		if newSerial >= maxBackups {
			// Delete backups beyond the limit
			if err := os.Remove(oldPath); err != nil {
				logging.Warnf("Failed to remove old backup %s: %v", oldPath, err)
			} else {
				logging.Debugf("Removed old backup: %s", filepath.Base(oldPath))
			}
		} else {
			// Rename to increment serial
			newPath := fmt.Sprintf("%s.%d", basePattern, newSerial)
			if err := os.Rename(oldPath, newPath); err != nil {
				logging.Warnf("Failed to rotate backup %s to %s: %v", oldPath, newPath, err)
			} else {
				logging.Debugf("Rotated backup: %s -> %s", filepath.Base(oldPath), filepath.Base(newPath))
			}
		}
	}

	return nil
}

// rotateBackups function removed - legacy backup format no longer used

// listBackupsWithRotation lists backups with rotation index information
func listBackupsWithRotation(_ *config.Config, siteName string) ([]ConfigurationBackup, error) {
	backupDir := xdg.GetBackupsDir()

	// Check if backup directory exists
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return []ConfigurationBackup{}, nil
	}

	// Read backup directory
	files, err := os.ReadDir(backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	var backups []ConfigurationBackup

	// Group backups by base filename and sort by serial
	backupGroups := make(map[string][]string) // base filename -> list of backup paths

	for _, file := range files {
		fileName := file.Name()

		// Check if it's a backup file (format: filename.json.N)
		if strings.Contains(fileName, ".json.") {
			// Extract base filename and serial
			lastDot := strings.LastIndex(fileName, ".")
			if lastDot > 0 {
				baseFile := fileName[:lastDot] // e.g., "us-oak-pina.json"
				backupPath := filepath.Join(backupDir, fileName)

				// Check if this backup contains the specified site
				if siteName != "" {
					// Read the file to check if it contains the site
					data, err := os.ReadFile(backupPath)
					if err != nil {
						continue
					}

					var configData ConfigFileStructure
					if err := json.Unmarshal(data, &configData); err != nil {
						continue
					}

					// Check if this config contains the specified site
					foundSite := false
					for _, siteConfig := range configData.Config.Sites {
						if siteNameVal, ok := siteConfig.SiteConfig["name"]; ok {
							if name, ok := siteNameVal.(string); ok && name == siteName {
								foundSite = true
								break
							}
						}
					}

					if !foundSite {
						continue
					}
				}

				backupGroups[baseFile] = append(backupGroups[baseFile], backupPath)
			}
		}
	}

	// Process each backup group
	for _, paths := range backupGroups {
		// Sort paths by serial number
		sort.Slice(paths, func(i, j int) bool {
			// Extract serial numbers
			serialI := extractSerial(paths[i])
			serialJ := extractSerial(paths[j])
			return serialI < serialJ // Lower serial = more recent
		})

		// Create backup entries
		for _, backupPath := range paths {
			backupData, err := os.ReadFile(backupPath)
			if err != nil {
				logging.Warnf("Failed to read backup file %s: %v", backupPath, err)
				continue
			}

			// Parse to get metadata
			var configData map[string]any
			if err := json.Unmarshal(backupData, &configData); err != nil {
				logging.Warnf("Failed to parse backup file %s: %v", backupPath, err)
				continue
			}

			// Extract timestamp from last_modified field
			var timestamp int64
			if lastModified, ok := configData["last_modified"].(string); ok {
				if t, err := time.Parse(time.RFC3339, lastModified); err == nil {
					timestamp = t.Unix()
				}
			}

			// Create a simplified backup entry
			backup := ConfigurationBackup{
				Timestamp:      timestamp,
				SiteName:       siteName,
				Operation:      "apply_configuration",
				BackupFilePath: backupPath,
			}

			backups = append(backups, backup)
		}
	}

	// Sort all backups by timestamp (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp > backups[j].Timestamp
	})

	return backups, nil
}

// extractSerial extracts the serial number from a backup filename
func extractSerial(path string) int {
	fileName := filepath.Base(path)
	lastDot := strings.LastIndex(fileName, ".")
	if lastDot > 0 {
		serialStr := fileName[lastDot+1:]
		var serial int
		if _, err := fmt.Sscanf(serialStr, "%d", &serial); err == nil {
			return serial
		}
	}
	return 999 // Default high number for files without proper serial
}

// cleanupBackupsWithConfig removes backups based on configured retention policy
func cleanupBackupsWithConfig(cfg *config.Config) error {
	backupDir := xdg.GetBackupsDir()

	// Check if backup directory exists
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return nil // No backups to clean
	}

	// Get the number of backups to maintain from config
	maxBackups := 10 // Default
	if cfg.Files.ConfigBackups > 0 {
		maxBackups = cfg.Files.ConfigBackups
	}

	// Group backups by base config file
	configFileBackups := make(map[string][]string) // base filename -> list of backup paths

	files, err := os.ReadDir(backupDir)
	if err != nil {
		return fmt.Errorf("failed to read backup directory: %w", err)
	}

	// First, handle new format backups (filename.json.N)
	for _, file := range files {
		fileName := file.Name()

		// Check if it's a backup file (format: filename.json.N)
		if strings.Contains(fileName, ".json.") {
			lastDot := strings.LastIndex(fileName, ".")
			if lastDot > 0 {
				baseFile := fileName[:lastDot] // e.g., "us-oak-pina.json"
				backupPath := filepath.Join(backupDir, fileName)
				configFileBackups[baseFile] = append(configFileBackups[baseFile], backupPath)
			}
		}
	}

	// Rotate backups for each config file
	for baseFile := range configFileBackups {
		logging.Debugf("Processing backups for config file %s", baseFile)
		if err := rotateConfigFileBackups(backupDir, baseFile, maxBackups); err != nil {
			logging.Warnf("Failed to rotate backups for %s: %v", baseFile, err)
		}
	}

	// Legacy backup format handling removed - only config file backups are used now

	// Also remove very old backups regardless of rotation
	retentionDays := viper.GetInt("backup.retention_days")
	if retentionDays == 0 {
		retentionDays = 30
	}
	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)
	removedCount := 0

	for _, file := range files {
		if !strings.Contains(file.Name(), ".json") {
			continue
		}

		backupPath := filepath.Join(backupDir, file.Name())

		// Try to parse the backup file to get timestamp
		backupData, err := os.ReadFile(backupPath)
		if err != nil {
			continue
		}

		// Check for last_modified field (new format)
		var configData map[string]any
		if err := json.Unmarshal(backupData, &configData); err == nil {
			if lastModified, ok := configData["last_modified"].(string); ok {
				if t, err := time.Parse(time.RFC3339, lastModified); err == nil {
					if t.Before(cutoffTime) {
						if err := os.Remove(backupPath); err != nil {
							logging.Warnf("Failed to remove old backup %s: %v", backupPath, err)
						} else {
							logging.Debugf("Removed old backup: %s", file.Name())
							removedCount++
						}
					}
				}
			}
		}
	}

	if removedCount > 0 {
		logging.Infof("Cleaned up %d old configuration backups", removedCount)
	}

	return nil
}

// createConfigBackupAfterApply creates a backup of the applied configuration file
// Note: siteName is unused - backups are file-based, not site-specific
func createConfigBackupAfterApply(cfg *config.Config, _ string, configFilePath string) error {
	backupDir := xdg.GetBackupsDir()
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Get the base filename for the backup
	baseFileName := filepath.Base(configFilePath)

	// Get the number of backups to maintain from config
	maxBackups := 10 // Default
	if cfg.Files.ConfigBackups > 0 {
		maxBackups = cfg.Files.ConfigBackups
	}

	// Rotate existing backups first (increment their serial numbers)
	if err := rotateConfigFileBackups(backupDir, baseFileName, maxBackups); err != nil {
		logging.Warnf("Failed to rotate backups: %v", err)
	}

	// Read the original config file
	originalData, err := os.ReadFile(configFilePath)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", configFilePath, err)
	}

	// Parse the config to add last_modified
	var configData map[string]any
	if err := json.Unmarshal(originalData, &configData); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Add last_modified timestamp in UTC at the root level
	configData["last_modified"] = time.Now().UTC().Format(time.RFC3339)

	// For each site in the config, also add last_modified if it has the structure
	if configSection, ok := configData["config"].(map[string]any); ok {
		for _, siteData := range configSection {
			if site, ok := siteData.(map[string]any); ok {
				site["last_modified"] = time.Now().UTC().Format(time.RFC3339)
			}
		}
	}

	// Create the backup file with serial 0 (most recent)
	backupFileName := fmt.Sprintf("%s.0", baseFileName)
	backupPath := filepath.Join(backupDir, backupFileName)

	// Marshal with indentation for readability
	backupData, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal backup data: %w", err)
	}

	if err := os.WriteFile(backupPath, backupData, 0644); err != nil {
		return fmt.Errorf("failed to save backup to %s: %w", backupPath, err)
	}

	logging.Infof("Configuration backup saved: %s", backupFileName)
	return nil
}

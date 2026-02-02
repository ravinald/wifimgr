package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ravinald/wifimgr/api"
	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/formatter"
	"github.com/ravinald/wifimgr/internal/logging"
)

// BackupInfo represents information about a backup file
type BackupInfo struct {
	SiteName     string
	Version      int
	FileName     string
	LastModified string
	Serial       int
	FilePath     string
}

// ConfigFileData represents the structure of a config file for parsing
type ConfigFileData struct {
	Version      int                       `json:"version"`
	LastModified string                    `json:"last_modified,omitempty"`
	Config       map[string]SiteConfigData `json:"config"`
}

// SiteConfigData represents a site configuration
type SiteConfigData struct {
	SiteConfig   map[string]interface{} `json:"site_config"`
	LastModified string                 `json:"last_modified,omitempty"`
}

// NewBackupCommand creates the backup command
func NewBackupCommand(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Manage configuration backups",
		Long:  `List and restore configuration backups`,
	}

	// Add subcommands
	cmd.AddCommand(newListCommand(cfg))
	cmd.AddCommand(newRestoreCommand(cfg))

	return cmd
}

// newListCommand creates the list subcommand
func newListCommand(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "list [all | <site>]",
		Short: "List configuration backups",
		Long:  `List configuration backups for all sites or a specific site`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Determine filter
			filter := "all"
			if len(args) > 0 {
				filter = args[0]
			}

			return listBackups(cfg, filter)
		},
	}
}

// newRestoreCommand creates the restore subcommand
func newRestoreCommand(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "restore <site> <serial>",
		Short: "Restore a configuration backup",
		Long:  `Restore a specific configuration backup to the config directory`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			siteName := args[0]
			serialStr := args[1]

			serial, err := strconv.Atoi(serialStr)
			if err != nil {
				return fmt.Errorf("invalid serial number: %s", serialStr)
			}

			return restoreBackup(cfg, siteName, serial)
		},
	}
}

// listBackups lists configuration backups
func listBackups(cfg *config.Config, filter string) error {
	backupDir := filepath.Join(cfg.Files.ConfigDir, "backups")

	// Check if backup directory exists
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		fmt.Println("No backups found")
		return nil
	}

	// Read backup directory
	files, err := os.ReadDir(backupDir)
	if err != nil {
		return fmt.Errorf("failed to read backup directory: %w", err)
	}

	// Collect backup information
	var backups []BackupInfo

	for _, file := range files {
		fileName := file.Name()

		// Skip non-backup files
		if !strings.Contains(fileName, ".json.") {
			continue
		}

		// Extract base filename and serial
		lastDot := strings.LastIndex(fileName, ".")
		if lastDot <= 0 {
			continue
		}
		serialStr := fileName[lastDot+1:]

		serial, err := strconv.Atoi(serialStr)
		if err != nil {
			continue // Skip files without proper serial
		}

		// Read the backup file to get metadata
		backupPath := filepath.Join(backupDir, fileName)
		backupData, err := os.ReadFile(backupPath)
		if err != nil {
			logging.Warnf("Failed to read backup file %s: %v", fileName, err)
			continue
		}

		// Parse the config to get version and sites
		var configData ConfigFileData
		if err := json.Unmarshal(backupData, &configData); err != nil {
			logging.Warnf("Failed to parse backup file %s: %v", fileName, err)
			continue
		}

		// Extract site names from the config
		for _, siteConfig := range configData.Config {
			siteName := ""
			if nameVal, ok := siteConfig.SiteConfig["name"]; ok {
				if name, ok := nameVal.(string); ok {
					siteName = name
				}
			}

			if siteName == "" {
				continue
			}

			// Apply filter
			if filter != "all" && siteName != filter {
				continue
			}

			// Get last modified time
			lastModified := configData.LastModified
			if lastModified == "" && siteConfig.LastModified != "" {
				lastModified = siteConfig.LastModified
			}

			// Format last modified for display
			if lastModified != "" {
				if t, err := time.Parse(time.RFC3339, lastModified); err == nil {
					lastModified = t.Format("2006-01-02 15:04:05")
				}
			}

			backups = append(backups, BackupInfo{
				SiteName:     siteName,
				Version:      configData.Version,
				FileName:     fileName,
				LastModified: lastModified,
				Serial:       serial,
				FilePath:     backupPath,
			})
		}
	}

	if len(backups) == 0 {
		if filter == "all" {
			fmt.Println("No backups found")
		} else {
			fmt.Printf("No backups found for site: %s\n", filter)
		}
		return nil
	}

	// Sort backups: first by site name, then by serial (lower serial = more recent)
	sort.Slice(backups, func(i, j int) bool {
		if backups[i].SiteName != backups[j].SiteName {
			return backups[i].SiteName < backups[j].SiteName
		}
		return backups[i].Serial < backups[j].Serial
	})

	// Configure table columns
	columns := []formatter.TableColumn{
		{Field: "site", Header: "Site", MaxWidth: 20},
		{Field: "id", Header: "ID", MaxWidth: 4},
		{Field: "filename", Header: "File Name", MaxWidth: 0}, // 0 = auto-size and scale
		{Field: "last_modified", Header: "Last Modified", MaxWidth: 20},
	}

	// Convert backups to table data
	var tableData []formatter.GenericTableData
	for _, backup := range backups {
		data := formatter.GenericTableData{
			"site":          backup.SiteName,
			"id":            fmt.Sprintf("%d", backup.Serial),
			"filename":      backup.FileName,
			"last_modified": backup.LastModified,
		}
		tableData = append(tableData, data)
	}

	// Create table config
	tableConfig := formatter.TableConfig{
		Columns:       columns,
		BoldHeaders:   true,
		ShowSeparator: true,
	}

	// Create and render table
	table := formatter.NewBubbleTable(tableConfig, tableData, false)
	fmt.Print(table.RenderStatic())

	fmt.Printf("\nTotal backups: %d\n", len(backups))
	if filter == "all" {
		// Count unique sites
		siteMap := make(map[string]bool)
		for _, backup := range backups {
			siteMap[backup.SiteName] = true
		}
		fmt.Printf("Sites with backups: %d\n", len(siteMap))
	}

	return nil
}

// restoreBackup restores a specific backup
func restoreBackup(cfg *config.Config, siteName string, serial int) error {
	backupDir := filepath.Join(cfg.Files.ConfigDir, "backups")

	// Find the backup file
	files, err := os.ReadDir(backupDir)
	if err != nil {
		return fmt.Errorf("failed to read backup directory: %w", err)
	}

	var backupFile string
	var baseFileName string

	for _, file := range files {
		fileName := file.Name()

		// Check if it's a backup file with the correct serial
		if !strings.HasSuffix(fileName, fmt.Sprintf(".%d", serial)) {
			continue
		}

		// Read the file to check if it contains the site
		backupPath := filepath.Join(backupDir, fileName)
		backupData, err := os.ReadFile(backupPath)
		if err != nil {
			continue
		}

		// Parse the config to check for the site
		var configData ConfigFileData
		if err := json.Unmarshal(backupData, &configData); err != nil {
			continue
		}

		// Check if this config contains the specified site
		for _, siteConfig := range configData.Config {
			if nameVal, ok := siteConfig.SiteConfig["name"]; ok {
				if name, ok := nameVal.(string); ok && name == siteName {
					backupFile = backupPath
					// Extract base filename (remove serial)
					lastDot := strings.LastIndex(fileName, ".")
					if lastDot > 0 {
						baseFileName = fileName[:lastDot]
					}
					break
				}
			}
		}

		if backupFile != "" {
			break
		}
	}

	if backupFile == "" {
		return fmt.Errorf("backup not found for site %s with serial %d", siteName, serial)
	}

	// Read the backup file
	backupData, err := os.ReadFile(backupFile)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	// Parse the backup to remove last_modified fields before restoring
	var configData map[string]interface{}
	if err := json.Unmarshal(backupData, &configData); err != nil {
		return fmt.Errorf("failed to parse backup file: %w", err)
	}

	// Remove last_modified fields
	delete(configData, "last_modified")
	if configSection, ok := configData["config"].(map[string]interface{}); ok {
		for _, siteData := range configSection {
			if site, ok := siteData.(map[string]interface{}); ok {
				delete(site, "last_modified")
			}
		}
	}

	// Determine the target file path
	targetPath := filepath.Join(cfg.Files.ConfigDir, baseFileName)

	// Check if target file exists and create a backup of current version
	if _, err := os.Stat(targetPath); err == nil {
		// Create a timestamped backup of the current file
		timestamp := time.Now().Format("20060102_150405")
		currentBackupPath := filepath.Join(backupDir, fmt.Sprintf("%s.before_restore_%s", baseFileName, timestamp))

		currentData, err := os.ReadFile(targetPath)
		if err != nil {
			return fmt.Errorf("failed to read current config file: %w", err)
		}

		if err := os.WriteFile(currentBackupPath, currentData, 0644); err != nil {
			return fmt.Errorf("failed to backup current config: %w", err)
		}

		logging.Infof("Current configuration backed up to: %s", filepath.Base(currentBackupPath))
	}

	// Marshal the cleaned config data
	restoredData, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal restored config: %w", err)
	}

	// Write the restored configuration
	if err := os.WriteFile(targetPath, restoredData, 0644); err != nil {
		return fmt.Errorf("failed to write restored config: %w", err)
	}

	fmt.Printf("Successfully restored backup for site %s\n", siteName)
	fmt.Printf("   Source: %s (serial %d)\n", filepath.Base(backupFile), serial)
	fmt.Printf("   Target: %s\n", targetPath)

	return nil
}

// HandleCommand is the entry point for backup commands
func HandleCommand(ctx context.Context, client api.Client, cfg *config.Config, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("backup command requires a subcommand: list or restore")
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "list":
		filter := "all"
		if len(subArgs) > 0 {
			filter = subArgs[0]
		}
		return listBackups(cfg, filter)

	case "restore":
		if len(subArgs) < 2 {
			return fmt.Errorf("restore command requires site name and serial number")
		}
		siteName := subArgs[0]
		serial, err := strconv.Atoi(subArgs[1])
		if err != nil {
			return fmt.Errorf("invalid serial number: %s", subArgs[1])
		}
		return restoreBackup(cfg, siteName, serial)

	default:
		return fmt.Errorf("unknown backup subcommand: %s", subcommand)
	}
}

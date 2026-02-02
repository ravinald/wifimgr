/*
Copyright © 2025 Ravi Pina <ravi@pina.org>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ravinald/wifimgr/internal/cmdutils"
	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/formatter"
	"github.com/ravinald/wifimgr/internal/logging"
)

// intentProfilesCmd represents the "show intent profiles" command
var intentProfilesCmd = &cobra.Command{
	Use:   "profiles [profile-name]",
	Short: "Show device profiles from local intent files",
	Long: `Show device profiles defined in local intent configuration files.

Without arguments, displays all device profiles in a table format.
With a profile name argument, shows the raw JSON of the device profile details.

Arguments:
  profile-name - Optional profile name to show details for

Examples:
  wifimgr show intent profiles              - Show all device profiles in table format
  wifimgr show intent profiles AP-Profile   - Show specific profile details in JSON format`,
	Args: cobra.MaximumNArgs(1),
	RunE: runShowIntentProfiles,
}

func init() {
	intentCmd.AddCommand(intentProfilesCmd)
}

// DeviceProfilesConfig represents the structure of device profiles configuration
type DeviceProfilesConfig struct {
	Version int                         `json:"version"`
	Config  DeviceProfilesConfigContent `json:"config"`
}

type DeviceProfilesConfigContent struct {
	DeviceProfiles DeviceProfilesMap `json:"device_profiles"`
}

type DeviceProfilesMap struct {
	AP      map[string]interface{} `json:"ap"`
	Switch  map[string]interface{} `json:"switch"`
	Gateway map[string]interface{} `json:"gateway"`
}

func runShowIntentProfiles(cmd *cobra.Command, args []string) error {
	// Check for help keyword in positional arguments
	if cmdutils.ContainsHelp(args) {
		return cmd.Help()
	}

	logger := logging.GetLogger()
	logger.Info("Executing show intent profiles command")

	// Get device profile files
	profileFiles := viper.GetStringSlice("files.device_profiles")
	configDir := viper.GetString("files.config_dir")

	// If no profile files specified, check for default device-profiles.json
	if len(profileFiles) == 0 {
		defaultFile := "device-profiles.json"
		if _, err := os.Stat(filepath.Join(configDir, defaultFile)); err == nil {
			profileFiles = []string{defaultFile}
		} else {
			logging.Warn("No device profile files specified or found")
			return fmt.Errorf("no device profile files specified in config")
		}
	}

	// Create duplicate tracker
	duplicateTracker := config.NewDuplicateTracker()

	// Collect all profiles from all files
	allProfiles := make(map[string]map[string]interface{})
	allProfiles["ap"] = make(map[string]interface{})
	allProfiles["switch"] = make(map[string]interface{})
	allProfiles["gateway"] = make(map[string]interface{})

	// Load profiles from each file
	for _, profileFile := range profileFiles {
		fullPath := filepath.Join(configDir, profileFile)

		// Read raw file data for line number estimation
		rawData, err := os.ReadFile(fullPath)
		if err != nil {
			logger.WithError(err).Errorf("Failed to read profile file %s", fullPath)
			continue
		}

		var profileConfig DeviceProfilesConfig
		if err := json.Unmarshal(rawData, &profileConfig); err != nil {
			logger.WithError(err).Errorf("Failed to parse profile file %s", fullPath)
			continue
		}

		// Check for duplicates and merge profiles from this file into allProfiles
		for profileName, profile := range profileConfig.Config.DeviceProfiles.AP {
			// Extract name from profile data if available
			name := config.ExtractNameFromJSON(profile)
			if name == "" {
				name = profileName
			}

			// Estimate line number
			keyPath := []string{"config", "device_profiles", "ap", profileName}
			line := config.EstimateLineNumber(rawData, keyPath)

			duplicateTracker.CheckAndAdd("device_profile", "ap", name, profileFile, line)
			allProfiles["ap"][profileName] = profile
		}
		for profileName, profile := range profileConfig.Config.DeviceProfiles.Switch {
			// Extract name from profile data if available
			name := config.ExtractNameFromJSON(profile)
			if name == "" {
				name = profileName
			}

			// Estimate line number
			keyPath := []string{"config", "device_profiles", "switch", profileName}
			line := config.EstimateLineNumber(rawData, keyPath)

			duplicateTracker.CheckAndAdd("device_profile", "switch", name, profileFile, line)
			allProfiles["switch"][profileName] = profile
		}
		for profileName, profile := range profileConfig.Config.DeviceProfiles.Gateway {
			// Extract name from profile data if available
			name := config.ExtractNameFromJSON(profile)
			if name == "" {
				name = profileName
			}

			// Estimate line number
			keyPath := []string{"config", "device_profiles", "gateway", profileName}
			line := config.EstimateLineNumber(rawData, keyPath)

			duplicateTracker.CheckAndAdd("device_profile", "gateway", name, profileFile, line)
			allProfiles["gateway"][profileName] = profile
		}
	}

	// If profile name is provided, show specific profile details
	if len(args) > 0 {
		profileName := args[0]
		return showIntentProfileDetails(allProfiles, profileName)
	}

	// Otherwise, show all profiles in table format
	return showAllIntentProfiles(allProfiles)
}

func showIntentProfileDetails(allProfiles map[string]map[string]interface{}, profileName string) error {
	logger := logging.GetLogger()

	// Search for the profile in all device types
	var profileDetail interface{}
	var profileType string

	for deviceType, profiles := range allProfiles {
		if profile, ok := profiles[profileName]; ok {
			profileDetail = profile
			profileType = deviceType
			break
		}
	}

	if profileDetail == nil {
		fmt.Printf("Profile not found: %s\n", profileName)
		return nil
	}

	// Add type field to the profile for display
	if profileMap, ok := profileDetail.(map[string]interface{}); ok {
		profileMap["type"] = profileType
	}

	// Marshal and print with color
	jsonData, err := formatter.MarshalJSONWithColorIndent(profileDetail, "", "  ")
	if err != nil {
		logger.WithError(err).Error("Failed to marshal profile detail to JSON")
		return err
	}

	fmt.Println(string(jsonData))
	return nil
}

func showAllIntentProfiles(allProfiles map[string]map[string]interface{}) error {

	// Create table data
	var tableData []formatter.GenericTableData

	// Process each device type
	for deviceType, profiles := range allProfiles {
		for profileName, profile := range profiles {
			row := make(map[string]interface{})

			// Core fields
			row["name"] = profileName
			row["type"] = strings.ToUpper(deviceType)
			row["source"] = "Intent"

			// Try to extract additional fields from the profile
			if profileMap, ok := profile.(map[string]interface{}); ok {
				// Look for common fields that might exist
				if nameField, exists := profileMap["name"]; exists {
					row["name"] = nameField
				}

				// Count certain configuration elements if present
				configCount := 0
				for key := range profileMap {
					// Count non-metadata fields as configuration items
					if key != "name" && key != "type" && key != "id" {
						configCount++
					}
				}
				row["config_items"] = configCount
			}

			tableData = append(tableData, formatter.GenericTableData(row))
		}
	}

	if len(tableData) == 0 {
		fmt.Println("No device profiles found in intent files")
		return nil
	}

	// Sort by name
	sort.Slice(tableData, func(i, j int) bool {
		nameI := fmt.Sprintf("%v", tableData[i]["name"])
		nameJ := fmt.Sprintf("%v", tableData[j]["name"])
		return strings.ToLower(nameI) < strings.ToLower(nameJ)
	})

	// Define columns
	columns := []formatter.TableColumn{
		{Field: "name", Title: "Name"},
		{Field: "type", Title: "Type"},
		{Field: "source", Title: "Source"},
		{Field: "config_items", Title: "Config Items"},
	}

	// Create table config
	tableConfig := formatter.TableConfig{
		Title:       fmt.Sprintf("Device Profiles from Intent Files (%d)", len(tableData)),
		Columns:     columns,
		Format:      "table",
		BoldHeaders: true,
	}

	// Print table
	printer := formatter.NewGenericTablePrinter(tableConfig, tableData)
	fmt.Print(printer.Print())

	return nil
}

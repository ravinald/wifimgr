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

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ravinald/wifimgr/internal/cmdutils"
	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/formatter"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/patterns"
)

// intentApCmd represents the "show intent ap" command
var intentApCmd = &cobra.Command{
	Use:   "ap [name-or-mac] [json] [no-resolve]",
	Short: "Show access point configuration from intent files",
	Long: `Show access point configuration from local intent files.

Arguments:
  name-or-mac  - Optional AP name or MAC address filter
  json         - Output in JSON format instead of table
  no-resolve   - When using JSON format, show raw IDs without resolving to names

Examples:
  wifimgr show intent ap                     - Show all AP configs from intent files
  wifimgr show intent ap AP-Name             - Show specific AP config by name
  wifimgr show intent ap 00:11:22:33:44:55  - Show specific AP config by MAC address
  wifimgr show intent ap AP-Name json        - Show specific AP config in JSON format
  wifimgr show intent ap json                - Show all AP configs in JSON format
  wifimgr show intent ap json no-resolve     - Show all AP configs in JSON without field resolution`,
	Args: cobra.MaximumNArgs(3),
	RunE: runIntentAP,
}

func runIntentAP(cmd *cobra.Command, args []string) error {
	// Check for help keyword in positional arguments
	if cmdutils.ContainsHelp(args) {
		return cmd.Help()
	}

	filter := ""
	format := "table"
	resolve := true

	// Parse arguments
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "json":
			format = "json"
		case "no-resolve":
			resolve = false
		default:
			// If it's not a format option, it must be the filter
			if filter == "" {
				filter = arg
			}
		}
	}

	// Load site configurations
	siteConfigFiles := viper.GetStringSlice("files.site_configs")
	configDir := viper.GetString("files.config_dir")

	var allAPs []formatter.GenericTableData

	// Search through all site config files
	for _, siteConfigFile := range siteConfigFiles {
		siteConfig, err := config.LoadSiteConfig(configDir, siteConfigFile)
		if err != nil {
			logging.Errorf("Failed to load site configuration from %s: %v", siteConfigFile, err)
			continue
		}

		// Process each site in the config
		for siteName, siteObj := range siteConfig.Config.Sites {
			// Process APs for this site
			for _, ap := range siteObj.Devices.APs {
				// Apply filter if provided
				if filter != "" && !patterns.Equals(ap.Name, filter) && !patterns.Equals(ap.Magic, filter) {
					continue
				}

				data := make(formatter.GenericTableData)
				data["name"] = ap.Name
				data["magic"] = ap.Magic
				data["site_name"] = siteName
				if len(ap.Tags) > 0 {
					data["tags"] = fmt.Sprintf("%v", ap.Tags)
				} else {
					data["tags"] = ""
				}
				data["notes"] = ap.Notes
				data["type"] = "ap"
				allAPs = append(allAPs, data)
			}
		}
	}

	// Handle JSON format first (before checking allAPs which is only for table format)
	if format == "json" {
		// For JSON output, load raw JSON data to preserve only defined fields
		var rawAPs []interface{}

		// Re-load config files as raw JSON
		for _, siteConfigFile := range siteConfigFiles {
			configFilePath := filepath.Join(configDir, siteConfigFile)

			// Read the raw JSON file
			fileBytes, err := os.ReadFile(configFilePath)
			if err != nil {
				logging.Errorf("Failed to read config file %s: %v", siteConfigFile, err)
				continue
			}

			// Parse as raw JSON
			var rawConfig map[string]interface{}
			if err := json.Unmarshal(fileBytes, &rawConfig); err != nil {
				logging.Errorf("Failed to parse JSON from %s: %v", siteConfigFile, err)
				continue
			}

			// Check if there's a "config" wrapper (new format) or direct sites (old format)
			var sitesData map[string]interface{}
			if configSection, hasConfig := rawConfig["config"].(map[string]interface{}); hasConfig {
				// New format with "config" wrapper
				sitesData = configSection
			} else {
				// Old format - sites at top level (excluding version)
				sitesData = make(map[string]interface{})
				for k, v := range rawConfig {
					if k != "version" {
						sitesData[k] = v
					}
				}
			}

			// Process each site
			for siteName, value := range sitesData {
				siteData, ok := value.(map[string]interface{})
				if !ok {
					continue
				}

				// Get devices section
				devices, ok := siteData["devices"].(map[string]interface{})
				if !ok {
					continue
				}

				// Get AP array
				aps, ok := devices["ap"].([]interface{})
				if !ok {
					continue
				}

				// Process each AP
				for _, apInterface := range aps {
					ap, ok := apInterface.(map[string]interface{})
					if !ok {
						continue
					}

					// Apply filter if provided
					apName, _ := ap["name"].(string)
					apMagic, _ := ap["magic"].(string)
					apMAC, _ := ap["mac"].(string)
					if filter != "" && !patterns.Equals(apName, filter) && !patterns.Equals(apMagic, filter) && !patterns.Equals(apMAC, filter) {
						continue
					}

					// Add site_id field if we can resolve it
					if cacheAccessor, err := cmdutils.GetCacheAccessor(); err == nil {
						if site, err := cacheAccessor.GetSiteByName(siteName); err == nil && site != nil && site.ID != "" {
							ap["site_id"] = site.ID
						}
					}

					// Apply field resolution if enabled
					if resolve {
						tableData := []formatter.GenericTableData{ap}
						if err := cmdutils.ApplyFieldResolution(tableData, true); err == nil {
							ap = tableData[0]
						}
					}

					// Only append non-nil APs
					if ap != nil {
						rawAPs = append(rawAPs, ap)
					}
				}
			}
		}

		// Output JSON
		var jsonData []byte
		var err error

		if len(rawAPs) == 0 {
			// No APs found
			if filter != "" {
				fmt.Printf("No access points found matching '%s' in intent files\n", filter)
			} else {
				fmt.Println("No access points found in intent files")
			}
			return nil
		} else if len(rawAPs) == 1 {
			// Single AP - output just the object
			jsonData, err = formatter.MarshalJSONWithColorIndent(rawAPs[0], "", "  ")
		} else {
			// Multiple APs - output as array
			jsonData, err = formatter.MarshalJSONWithColorIndent(rawAPs, "", "  ")
		}

		if err != nil {
			return fmt.Errorf("error marshalling JSON: %v", err)
		}

		fmt.Println(string(jsonData))
		return nil
	}

	// Table format (existing code)
	// Check if we have data for table format
	if len(allAPs) == 0 {
		if filter != "" {
			fmt.Printf("No access points found matching '%s' in intent files\n", filter)
		} else {
			fmt.Println("No access points found in intent files")
		}
		return nil
	}

	// Sort by site_name, then by name using natural sorting
	formatter.SortTableData(allAPs)

	// Create table configuration
	tableConfig := formatter.TableConfig{
		Format:        format,
		Title:         "Access Points from Intent Files",
		BoldHeaders:   true,
		ShowSeparator: true,
		Columns: []formatter.TableColumn{
			{Field: "name", Title: "Name", MaxWidth: 30},
			{Field: "site_name", Title: "Site", MaxWidth: 20},
			{Field: "magic", Title: "Magic", MaxWidth: 40},
			{Field: "tags", Title: "Tags", MaxWidth: 20},
			{Field: "notes", Title: "Notes", MaxWidth: 0},
		},
	}

	// Create and render the table
	printer := formatter.NewGenericTablePrinter(tableConfig, allAPs)
	fmt.Print(printer.Print())

	return nil
}

func init() {
	intentCmd.AddCommand(intentApCmd)
}

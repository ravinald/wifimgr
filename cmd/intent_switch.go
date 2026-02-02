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

// intentSwitchCmd represents the "show intent switch" command
var intentSwitchCmd = &cobra.Command{
	Use:   "switch [name-or-mac] [json] [no-resolve]",
	Short: "Show switch configuration from intent files",
	Long: `Show switch configuration from local intent files.

Arguments:
  name-or-mac  - Optional switch name or MAC address filter
  json         - Output in JSON format instead of table
  no-resolve   - When using JSON format, show raw IDs without resolving to names

Examples:
  wifimgr show intent switch                     - Show all switch configs from intent files
  wifimgr show intent switch SW-Name             - Show specific switch config by name
  wifimgr show intent switch 00:11:22:33:44:55  - Show specific switch config by MAC address
  wifimgr show intent switch SW-Name json        - Show specific switch config in JSON format
  wifimgr show intent switch json                - Show all switch configs in JSON format
  wifimgr show intent switch json no-resolve     - Show all switch configs in JSON without field resolution`,
	Args: cobra.MaximumNArgs(3),
	RunE: runIntentSwitch,
}

func runIntentSwitch(cmd *cobra.Command, args []string) error {
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

	var allSwitches []formatter.GenericTableData

	// Search through all site config files
	for _, siteConfigFile := range siteConfigFiles {
		siteConfig, err := config.LoadSiteConfig(configDir, siteConfigFile)
		if err != nil {
			logging.Errorf("Failed to load site configuration from %s: %v", siteConfigFile, err)
			continue
		}

		// Process each site in the config
		for siteName, siteObj := range siteConfig.Config.Sites {
			// Process switches for this site
			for _, sw := range siteObj.Devices.Switches {
				// Apply filter if provided
				if filter != "" && !patterns.Equals(sw.Name, filter) && !patterns.Equals(sw.Magic, filter) {
					continue
				}

				data := make(formatter.GenericTableData)
				data["name"] = sw.Name
				data["magic"] = sw.Magic
				data["site"] = siteName
				data["role"] = sw.Role
				if len(sw.Tags) > 0 {
					data["tags"] = fmt.Sprintf("%v", sw.Tags)
				} else {
					data["tags"] = ""
				}
				data["notes"] = sw.Notes
				data["type"] = "switch"
				allSwitches = append(allSwitches, data)
			}
		}
	}

	// Handle JSON format first (before checking allSwitches which is only for table format)
	if format == "json" {
		// For JSON output, load raw JSON data to preserve only defined fields
		var rawSwitches []interface{}

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

				// Get switch array
				switches, ok := devices["switch"].([]interface{})
				if !ok {
					continue
				}

				// Process each switch
				for _, switchInterface := range switches {
					sw, ok := switchInterface.(map[string]interface{})
					if !ok {
						continue
					}

					// Apply filter if provided
					swName, _ := sw["name"].(string)
					swMagic, _ := sw["magic"].(string)
					if filter != "" && !patterns.Equals(swName, filter) && !patterns.Equals(swMagic, filter) {
						continue
					}

					// Add site_id field if we can resolve it
					if cacheAccessor, err := cmdutils.GetCacheAccessor(); err == nil {
						if site, err := cacheAccessor.GetSiteByName(siteName); err == nil && site != nil && site.ID != "" {
							sw["site_id"] = site.ID
						}
					}

					// Apply field resolution if enabled
					if resolve {
						tableData := []formatter.GenericTableData{sw}
						if err := cmdutils.ApplyFieldResolution(tableData, true); err == nil {
							sw = tableData[0]
						}
					}

					rawSwitches = append(rawSwitches, sw)
				}
			}
		}

		// Output JSON
		var jsonData []byte
		var err error

		if len(rawSwitches) == 0 {
			// No switches found
			if filter != "" {
				fmt.Printf("No switches found matching '%s' in intent files\n", filter)
			} else {
				fmt.Println("No switches found in intent files")
			}
			return nil
		} else if len(rawSwitches) == 1 {
			// Single switch - output just the object
			jsonData, err = formatter.MarshalJSONWithColorIndent(rawSwitches[0], "", "  ")
		} else {
			// Multiple switches - output as array
			jsonData, err = formatter.MarshalJSONWithColorIndent(rawSwitches, "", "  ")
		}

		if err != nil {
			return fmt.Errorf("error marshalling JSON: %v", err)
		}

		fmt.Println(string(jsonData))
		return nil
	}

	// Table format (existing code)
	// Check if we have data for table format
	if len(allSwitches) == 0 {
		if filter != "" {
			fmt.Printf("No switches found matching '%s' in intent files\n", filter)
		} else {
			fmt.Println("No switches found in intent files")
		}
		return nil
	}

	// Create table configuration
	tableConfig := formatter.TableConfig{
		Format:        format,
		Title:         "Switches from Intent Files",
		BoldHeaders:   true,
		ShowSeparator: true,
		Columns: []formatter.TableColumn{
			{Field: "name", Title: "Name", MaxWidth: 30},
			{Field: "site", Title: "Site", MaxWidth: 20},
			{Field: "magic", Title: "Magic", MaxWidth: 40},
			{Field: "role", Title: "Role", MaxWidth: 15},
			{Field: "tags", Title: "Tags", MaxWidth: 20},
			{Field: "notes", Title: "Notes", MaxWidth: 0},
		},
	}

	// Create and render the table
	printer := formatter.NewGenericTablePrinter(tableConfig, allSwitches)
	fmt.Print(printer.Print())

	return nil
}

func init() {
	intentCmd.AddCommand(intentSwitchCmd)
}

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
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ravinald/wifimgr/internal/cmdutils"
	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/formatter"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/patterns"
)

// intentGatewayCmd represents the "show intent gateway" command
var intentGatewayCmd = &cobra.Command{
	Use:   "gateway [name-or-mac]",
	Short: "Show gateway configuration from intent files",
	Long: `Show gateway configuration from local intent files.

Arguments:
  name-or-mac  - Optional gateway name or MAC address filter

Examples:
  wifimgr show intent gateway                     - Show all gateway configs from intent files
  wifimgr show intent gateway GW-Name             - Show specific gateway config by name
  wifimgr show intent gateway 00:11:22:33:44:55  - Show specific gateway config by MAC address`,
	Args: cobra.MaximumNArgs(1),
	RunE: runIntentGateway,
}

func runIntentGateway(cmd *cobra.Command, args []string) error {
	// Check for help keyword in positional arguments
	if cmdutils.ContainsHelp(args) {
		return cmd.Help()
	}

	filter := ""
	if len(args) > 0 {
		filter = args[0]
	}

	// Load site configurations
	siteConfigFiles := viper.GetStringSlice("files.site_configs")
	configDir := viper.GetString("files.config_dir")

	var allGateways []formatter.GenericTableData

	// Search through all site config files
	for _, siteConfigFile := range siteConfigFiles {
		siteConfig, err := config.LoadSiteConfig(configDir, siteConfigFile)
		if err != nil {
			logging.Errorf("Failed to load site configuration from %s: %v", siteConfigFile, err)
			continue
		}

		// Process each site in the config
		for siteName, siteObj := range siteConfig.Config.Sites {
			// Process gateways/wan edge devices for this site
			for _, gw := range siteObj.Devices.WanEdge {
				// Apply filter if provided
				if filter != "" && !patterns.Equals(gw.Name, filter) && !patterns.Equals(gw.Magic, filter) {
					continue
				}

				data := make(formatter.GenericTableData)
				data["name"] = gw.Name
				data["magic"] = gw.Magic
				data["site"] = siteName
				if len(gw.Tags) > 0 {
					data["tags"] = fmt.Sprintf("%v", gw.Tags)
				} else {
					data["tags"] = ""
				}
				data["notes"] = gw.Notes
				data["type"] = "gateway"
				allGateways = append(allGateways, data)
			}
		}
	}

	if len(allGateways) == 0 {
		if filter != "" {
			fmt.Printf("No gateways found matching '%s' in intent files\n", filter)
		} else {
			fmt.Println("No gateways found in intent files")
		}
		return nil
	}

	// Get format
	format := viper.GetString("display.format")
	if format == "" {
		format = "table"
	}

	// Create table configuration
	tableConfig := formatter.TableConfig{
		Format:        format,
		Title:         "Gateways from Intent Files",
		BoldHeaders:   true,
		ShowSeparator: true,
		Columns: []formatter.TableColumn{
			{Field: "name", Title: "Name", MaxWidth: 30},
			{Field: "site", Title: "Site", MaxWidth: 20},
			{Field: "magic", Title: "Magic", MaxWidth: 40},
			{Field: "tags", Title: "Tags", MaxWidth: 20},
			{Field: "notes", Title: "Notes", MaxWidth: 0},
		},
	}

	// Create and render the table
	printer := formatter.NewGenericTablePrinter(tableConfig, allGateways)
	fmt.Print(printer.Print())

	return nil
}

func init() {
	intentCmd.AddCommand(intentGatewayCmd)
}

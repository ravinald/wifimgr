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

// intentSiteCmd represents the "show intent site" command
var intentSiteCmd = &cobra.Command{
	Use:   "site [site-name] [ap|switch|gateway|all]",
	Short: "List sites or show site-specific device configurations from intent files",
	Long: `List all sites defined in local intent configuration files or show device
configurations for a specific site.

When called without arguments, displays all sites with device counts.
When called with a site name and optional device type, displays the devices
for that specific site.

Arguments:
  site-name    - Optional site name filter or specific site to display
  device-type  - Type of devices to show: ap, switch, gateway, or all (default: all)
                 Only used when a specific site is provided

Examples:
  wifimgr show intent site                    - Show all sites from intent files
  wifimgr show intent site SITE-NAME          - Show all devices for a specific site
  wifimgr show intent site SITE-NAME ap       - Show only APs for the site
  wifimgr show intent site SITE-NAME switch   - Show only switches for the site
  wifimgr show intent site SITE-NAME gateway  - Show only gateways for the site
  wifimgr show intent site SITE-NAME all      - Show all devices for the site`,
	Args: cobra.RangeArgs(0, 2),
	RunE: runIntentSite,
}

func runIntentSite(cmd *cobra.Command, args []string) error {
	// Check for help keyword in positional arguments
	if cmdutils.ContainsHelp(args) {
		return cmd.Help()
	}

	// Load site configurations
	siteConfigFiles := viper.GetStringSlice("files.site_configs")
	configDir := viper.GetString("files.config_dir")

	if len(siteConfigFiles) == 0 {
		logging.Warn("No site configuration files specified")
		return fmt.Errorf("no site configuration files specified in config")
	}

	// Check if this is a request for a specific site's devices
	if len(args) > 0 {
		siteName := args[0]
		deviceType := "all"

		if len(args) > 1 {
			deviceType = args[1]
			// Validate device type
			validTypes := []string{"ap", "switch", "gateway", "all"}
			valid := false
			for _, t := range validTypes {
				if deviceType == t {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("invalid device type: %s. Must be one of: ap, switch, gateway, all", deviceType)
			}
		}

		// Try to find and display the specific site
		var targetSiteConfig *config.SiteConfigObj
		var actualSiteName string

		// Search through all site config files for the requested site
		for _, siteConfigFile := range siteConfigFiles {
			siteConfig, err := config.LoadSiteConfig(configDir, siteConfigFile)
			if err != nil {
				logging.Errorf("Failed to load site configuration from %s: %v", siteConfigFile, err)
				continue
			}

			// Check if this file contains the site we're looking for
			for configSiteName, siteObj := range siteConfig.Config.Sites {
				if patterns.Equals(configSiteName, siteName) {
					targetSiteConfig = &siteObj
					actualSiteName = configSiteName
					break
				}
			}
			if targetSiteConfig != nil {
				break
			}
		}

		// If we found a matching site, display its devices
		if targetSiteConfig != nil {
			format := viper.GetString("display.format")
			if format == "" {
				format = "table"
			}

			// Display devices based on type
			switch deviceType {
			case "ap":
				return displaySiteAPs(actualSiteName, targetSiteConfig, format)
			case "switch":
				return displaySiteSwitches(actualSiteName, targetSiteConfig, format)
			case "gateway":
				return displaySiteGateways(actualSiteName, targetSiteConfig, format)
			case "all":
				// Display all device types
				if err := displaySiteAPs(actualSiteName, targetSiteConfig, format); err != nil {
					return err
				}
				fmt.Println() // Add spacing between device types
				if err := displaySiteSwitches(actualSiteName, targetSiteConfig, format); err != nil {
					return err
				}
				fmt.Println() // Add spacing between device types
				if err := displaySiteGateways(actualSiteName, targetSiteConfig, format); err != nil {
					return err
				}
			}
			return nil
		}
		// If no exact match, fall through to show filtered site list
	}

	// Show site list (with optional filter)
	filter := ""
	if len(args) > 0 {
		filter = args[0]
	}

	var allSites []formatter.GenericTableData

	// Load and process all site config files
	for _, siteConfigFile := range siteConfigFiles {
		siteConfig, err := config.LoadSiteConfig(configDir, siteConfigFile)
		if err != nil {
			logging.Errorf("Failed to load site configuration from %s: %v", siteConfigFile, err)
			continue
		}

		// Process each site in the config
		for siteName, siteObj := range siteConfig.Config.Sites {
			// Apply filter if provided
			if filter != "" && !patterns.Equals(siteName, filter) {
				continue
			}

			// Count devices in this site
			apCount := len(siteObj.Devices.APs)
			switchCount := len(siteObj.Devices.Switches)
			gatewayCount := len(siteObj.Devices.WanEdge)
			totalDevices := apCount + switchCount + gatewayCount

			data := make(formatter.GenericTableData)
			data["name"] = siteName
			data["address"] = siteObj.SiteConfig.Address
			data["country_code"] = siteObj.SiteConfig.CountryCode
			data["timezone"] = siteObj.SiteConfig.Timezone
			data["ap_count"] = apCount
			data["switch_count"] = switchCount
			data["gateway_count"] = gatewayCount
			data["total_devices"] = totalDevices
			data["notes"] = siteObj.SiteConfig.Notes

			// Add location if available
			if siteObj.SiteConfig.LatLng != nil {
				data["lat"] = siteObj.SiteConfig.LatLng.Lat
				data["lng"] = siteObj.SiteConfig.LatLng.Lng
			}

			allSites = append(allSites, data)
		}
	}

	if len(allSites) == 0 {
		if filter != "" {
			fmt.Printf("No sites found matching '%s' in intent files\n", filter)
		} else {
			fmt.Println("No sites found in intent files")
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
		Title:         "Sites from Intent Files",
		BoldHeaders:   true,
		ShowSeparator: true,
		Columns: []formatter.TableColumn{
			{Field: "name", Title: "Site Name", MaxWidth: 25},
			{Field: "address", Title: "Address", MaxWidth: 40},
			{Field: "country_code", Title: "Country", MaxWidth: 7},
			{Field: "timezone", Title: "Timezone", MaxWidth: 20},
			{Field: "ap_count", Title: "APs", MaxWidth: 5},
			{Field: "switch_count", Title: "SWs", MaxWidth: 5},
			{Field: "gateway_count", Title: "GWs", MaxWidth: 5},
			{Field: "total_devices", Title: "Total", MaxWidth: 6},
		},
	}

	// Create and render the table
	printer := formatter.NewGenericTablePrinter(tableConfig, allSites)
	fmt.Print(printer.Print())

	return nil
}

func init() {
	intentCmd.AddCommand(intentSiteCmd)
}

// displaySiteAPs displays access points for a site
func displaySiteAPs(siteName string, siteConfig *config.SiteConfigObj, format string) error {
	if len(siteConfig.Devices.APs) == 0 {
		fmt.Printf("No access points configured for site %s\n", siteName)
		return nil
	}

	// Convert APConfig to GenericTableData
	var tableData []formatter.GenericTableData
	for _, ap := range siteConfig.Devices.APs {
		data := make(formatter.GenericTableData)
		data["name"] = ap.Name
		data["magic"] = ap.Magic
		if len(ap.Tags) > 0 {
			data["tags"] = fmt.Sprintf("%v", ap.Tags)
		} else {
			data["tags"] = ""
		}
		data["notes"] = ap.Notes
		data["site"] = siteName
		data["type"] = "ap"
		tableData = append(tableData, data)
	}

	// Create table configuration
	tableConfig := formatter.TableConfig{
		Format:        format,
		Title:         fmt.Sprintf("Access Points for site %s", siteName),
		BoldHeaders:   true,
		ShowSeparator: true,
		Columns: []formatter.TableColumn{
			{Field: "name", Title: "Name", MaxWidth: 30},
			{Field: "magic", Title: "Magic", MaxWidth: 40},
			{Field: "tags", Title: "Tags", MaxWidth: 20},
			{Field: "notes", Title: "Notes", MaxWidth: 0},
		},
	}

	// Create and render the table
	printer := formatter.NewGenericTablePrinter(tableConfig, tableData)
	fmt.Print(printer.Print())

	return nil
}

// displaySiteSwitches displays switches for a site
func displaySiteSwitches(siteName string, siteConfig *config.SiteConfigObj, format string) error {
	if len(siteConfig.Devices.Switches) == 0 {
		fmt.Printf("No switches configured for site %s\n", siteName)
		return nil
	}

	// Convert SwitchConfig to GenericTableData
	var tableData []formatter.GenericTableData
	for _, sw := range siteConfig.Devices.Switches {
		data := make(formatter.GenericTableData)
		data["name"] = sw.Name
		data["magic"] = sw.Magic
		data["role"] = sw.Role
		if len(sw.Tags) > 0 {
			data["tags"] = fmt.Sprintf("%v", sw.Tags)
		} else {
			data["tags"] = ""
		}
		data["notes"] = sw.Notes
		data["site"] = siteName
		data["type"] = "switch"
		tableData = append(tableData, data)
	}

	// Create table configuration
	tableConfig := formatter.TableConfig{
		Format:        format,
		Title:         fmt.Sprintf("Switches for site %s", siteName),
		BoldHeaders:   true,
		ShowSeparator: true,
		Columns: []formatter.TableColumn{
			{Field: "name", Title: "Name", MaxWidth: 30},
			{Field: "magic", Title: "Magic", MaxWidth: 40},
			{Field: "role", Title: "Role", MaxWidth: 15},
			{Field: "tags", Title: "Tags", MaxWidth: 20},
			{Field: "notes", Title: "Notes", MaxWidth: 0},
		},
	}

	// Create and render the table
	printer := formatter.NewGenericTablePrinter(tableConfig, tableData)
	fmt.Print(printer.Print())

	return nil
}

// displaySiteGateways displays gateways/wan edge devices for a site
func displaySiteGateways(siteName string, siteConfig *config.SiteConfigObj, format string) error {
	if len(siteConfig.Devices.WanEdge) == 0 {
		fmt.Printf("No gateways configured for site %s\n", siteName)
		return nil
	}

	// Convert WanEdgeConfig to GenericTableData
	var tableData []formatter.GenericTableData
	for _, gw := range siteConfig.Devices.WanEdge {
		data := make(formatter.GenericTableData)
		data["name"] = gw.Name
		data["magic"] = gw.Magic
		if len(gw.Tags) > 0 {
			data["tags"] = fmt.Sprintf("%v", gw.Tags)
		} else {
			data["tags"] = ""
		}
		data["notes"] = gw.Notes
		data["site"] = siteName
		data["type"] = "gateway"
		tableData = append(tableData, data)
	}

	// Create table configuration
	tableConfig := formatter.TableConfig{
		Format:        format,
		Title:         fmt.Sprintf("Gateways for site %s", siteName),
		BoldHeaders:   true,
		ShowSeparator: true,
		Columns: []formatter.TableColumn{
			{Field: "name", Title: "Name", MaxWidth: 30},
			{Field: "magic", Title: "Magic", MaxWidth: 40},
			{Field: "tags", Title: "Tags", MaxWidth: 20},
			{Field: "notes", Title: "Notes", MaxWidth: 0},
		},
	}

	// Create and render the table
	printer := formatter.NewGenericTablePrinter(tableConfig, tableData)
	fmt.Print(printer.Print())

	return nil
}

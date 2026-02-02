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
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/formatter"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/pdf"
	"github.com/ravinald/wifimgr/internal/vendors"
)

var importPDFCmd = &cobra.Command{
	Use:   "pdf file <file-path> site <site-name> [api <api-name>] [prefix <prefix>] [suffix <suffix>] [create]",
	Short: "Import AP radio configurations from PDF floor plans",
	Long: `Import AP radio configuration data from PDF floor plans and update site configuration.

Extracts AP radio settings (channel, power, bandwidth) from PDF floor plans
and updates the corresponding site configuration file.

Basic Usage:
  wifimgr import pdf file plan.pdf site US-LAB-01
  wifimgr import pdf file plan.pdf site US-LAB-01 create

With API Override (sets device-level api if different from site):
  wifimgr import pdf file plan.pdf site US-LAB-01 api meraki
  wifimgr import pdf file plan.pdf site US-LAB-01 api mist

With AP Name Transformations:
  wifimgr import pdf file plan.pdf site US-LAB-01 prefix PROD-
  wifimgr import pdf file plan.pdf site US-LAB-01 suffix -V2
  wifimgr import pdf file plan.pdf site US-LAB-01 prefix PROD- suffix -V2

Combined Options:
  wifimgr import pdf file plan.pdf site US-LAB-01 api meraki prefix PROD- create

Arguments:
  file           Required. Keyword followed by path to PDF file
  site           Required. Keyword followed by site name
  api            Optional. Keyword followed by API name (must be defined in config)
                 If different from site's api, sets device-level api on imported APs
  prefix         Optional. Keyword followed by prefix to add to AP names for matching
  suffix         Optional. Keyword followed by suffix to add to AP names for matching
  create         Optional. Auto-create AP entries for devices not found in config

What it Does:
  1. Parses PDF for AP configurations (channel, power, width for 2.4G, 5G, 6G bands)
  2. Applies prefix/suffix to AP names for matching against site config
  3. Matches AP names from PDF to existing site configuration
  4. Updates radio configuration for matching APs
  5. If api differs from site api, sets device-level api on each AP
  6. With 'create': Creates new AP entries for unmatched devices (placeholder MACs)
  7. Saves updated configuration file
  8. Displays summary table of parsed configurations

Output Location:
  Updates existing site config at ./config/<site-name>.json

Note: When using 'create', new APs are assigned placeholder MAC addresses
(format: 000000000001, 000000000002, etc.) that should be replaced with
actual MAC addresses when the devices are discovered or provisioned.`,
	Args: func(cmd *cobra.Command, args []string) error {
		// Allow "help" as a special keyword
		for _, arg := range args {
			if strings.ToLower(arg) == "help" {
				return nil // Will be handled in RunE
			}
		}
		// Otherwise require at least 4 args: file <path> site <name>
		if len(args) < 4 {
			return fmt.Errorf("requires at least 4 arg(s), only received %d", len(args))
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check for help keyword in positional arguments
		for _, arg := range args {
			if strings.ToLower(arg) == "help" {
				return cmd.Help()
			}
		}

		// Parse arguments to find file path, site name, api, prefix, suffix, and create mode
		var filePath string
		var siteName string
		var apiName string
		var apPrefix string
		var apSuffix string
		var createMode bool
		fileType := "pdf" // default

		// Parse positional arguments
		i := 0
		for i < len(args) {
			switch strings.ToLower(args[i]) {
			case "type":
				if i+1 < len(args) {
					fileType = args[i+1]
					i += 2
				} else {
					return fmt.Errorf("type requires a value")
				}
			case "file":
				if i+1 < len(args) {
					filePath = args[i+1]
					i += 2
				} else {
					return fmt.Errorf("file requires a path")
				}
			case "site":
				if i+1 < len(args) {
					siteName = args[i+1]
					i += 2
				} else {
					return fmt.Errorf("site requires a name")
				}
			case "api":
				if i+1 < len(args) {
					apiName = args[i+1]
					i += 2
				} else {
					return fmt.Errorf("api requires a name")
				}
			case "prefix":
				if i+1 < len(args) {
					apPrefix = args[i+1]
					i += 2
				} else {
					return fmt.Errorf("prefix requires a value")
				}
			case "suffix":
				if i+1 < len(args) {
					apSuffix = args[i+1]
					i += 2
				} else {
					return fmt.Errorf("suffix requires a value")
				}
			case "create":
				createMode = true
				i++
			default:
				// If we haven't found a file path yet and this isn't a known keyword,
				// treat it as the file path (for shorthand "import file.pdf site NAME")
				if filePath == "" && args[i] != "pdf" {
					filePath = args[i]
				}
				i++
			}
		}

		if filePath == "" {
			return fmt.Errorf("file path is required")
		}

		if siteName == "" {
			return fmt.Errorf("site name is required")
		}

		// If API is provided, validate it exists in config
		if apiName != "" {
			configuredAPIs := viper.GetStringMap("api")
			var validAPINames []string
			for name := range configuredAPIs {
				validAPINames = append(validAPINames, name)
			}
			sort.Strings(validAPINames) // Sort for consistent output

			// Validate API exists in config (case-insensitive)
			apiFound := false
			for name := range configuredAPIs {
				if strings.EqualFold(name, apiName) {
					apiName = name // Use the actual key from config
					apiFound = true
					break
				}
			}
			if !apiFound {
				return fmt.Errorf("api '%s' not found in configuration. Valid options: %s", apiName, strings.Join(validAPINames, ", "))
			}

			logging.Debugf("Using API override: %s", apiName)
		}

		// Only support PDF for now
		if fileType != "pdf" {
			return fmt.Errorf("unsupported file type: %s (only 'pdf' is supported)", fileType)
		}

		logging.Debugf("Importing PDF file: %s for site: %s", filePath, siteName)

		// Check if file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", filePath)
		}

		// Parse the PDF file
		parser := pdf.NewParser()
		apConfigs, err := parser.ParseFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to parse PDF: %w", err)
		}

		if len(apConfigs) == 0 {
			fmt.Println("No AP configurations found in the PDF file")
			return nil
		}

		// Load site configuration using the site index
		siteConfigPath, found := config.GetSiteConfigFullPath(siteName)
		if !found {
			return fmt.Errorf("site '%s' not found in configuration. Check that the site is defined in one of the files listed in .files.site_configs", siteName)
		}

		// Check if site config exists
		if _, err := os.Stat(siteConfigPath); os.IsNotExist(err) {
			return fmt.Errorf("site configuration file not found: %s", siteConfigPath)
		}

		// Get the config directory for LoadSiteConfig
		configDir := viper.GetString("files.config_dir")
		if configDir == "" {
			configDir = "./config"
		}

		// Get the relative path for LoadSiteConfig
		siteConfigFile, _ := config.GetSiteConfigPath(siteName)

		// Load the site configuration
		siteConfig, err := config.LoadSiteConfig(configDir, siteConfigFile)
		if err != nil {
			return fmt.Errorf("failed to load site configuration: %w", err)
		}

		logging.Infof("Loaded site configuration for %s", siteName)

		// Track updates
		updatedAPs := []string{}
		createdAPs := []string{}
		notFoundAPs := []string{}
		placeholderMACCounter := 1

		// Find the matching site key
		var siteKey string
		for key := range siteConfig.Config.Sites {
			if strings.EqualFold(key, siteName) {
				siteKey = key
				break
			}
		}

		if siteKey == "" {
			return fmt.Errorf("site %s not found in configuration", siteName)
		}

		// Get the site data (we need to work with the actual map entry)
		siteData := siteConfig.Config.Sites[siteKey]

		// Determine if we need to set device-level API
		// This happens when api argument is provided and differs from site's API
		siteAPI := siteData.API
		setDeviceAPI := apiName != "" && !strings.EqualFold(apiName, siteAPI)
		if setDeviceAPI {
			logging.Infof("API '%s' differs from site API '%s' - will set device-level API on imported APs", apiName, siteAPI)
		} else if apiName != "" {
			logging.Debugf("API '%s' matches site API - no device-level override needed", apiName)
		}

		// Initialize APs map if nil
		if siteData.Devices.APs == nil {
			siteData.Devices.APs = make(map[string]config.APConfig)
		}

		// Find the highest existing placeholder MAC to avoid collisions
		for mac := range siteData.Devices.APs {
			if strings.HasPrefix(mac, "00000000") {
				// Parse the numeric suffix
				if num, err := strconv.Atoi(mac[8:]); err == nil && num >= placeholderMACCounter {
					placeholderMACCounter = num + 1
				}
			}
		}

		// Load defaults configuration if available
		defaults := loadDefaults()

		// Update matching APs
		for _, pdfAP := range apConfigs {
			apFound := false

			// Apply prefix and suffix to AP name for matching
			matchName := apPrefix + pdfAP.Name + apSuffix

			// Look for matching AP in site config (map structure)
			for macAddr, configAP := range siteData.Devices.APs {
				if configAP.Name == matchName {
					apFound = true
					logging.Debugf("Found matching AP: %s (MAC: %s)", matchName, macAddr)

					// Update radio configurations (need to update the map entry)
					updatedAP := configAP
					updateAPRadioConfig(&updatedAP, pdfAP, defaults)

					// Set device-level API if different from site
					if setDeviceAPI {
						updatedAP.API = apiName
					}

					siteData.Devices.APs[macAddr] = updatedAP
					updatedAPs = append(updatedAPs, matchName)
					break
				}
			}

			if !apFound {
				if createMode {
					// Create a new AP entry with placeholder MAC
					placeholderMAC := fmt.Sprintf("%012d", placeholderMACCounter)
					placeholderMACCounter++

					newAP := config.APConfig{
						MAC:            placeholderMAC,
						APDeviceConfig: &vendors.APDeviceConfig{Name: matchName},
					}
					updateAPRadioConfig(&newAP, pdfAP, defaults)

					// Set device-level API if different from site
					if setDeviceAPI {
						newAP.API = apiName
					}

					siteData.Devices.APs[placeholderMAC] = newAP
					createdAPs = append(createdAPs, matchName)
					logging.Debugf("Created new AP: %s (placeholder MAC: %s)", matchName, placeholderMAC)
				} else {
					notFoundAPs = append(notFoundAPs, matchName)
				}
			}
		}

		// Put the modified site data back into the map
		siteConfig.Config.Sites[siteKey] = siteData

		// Save updated configuration if there were updates or creations
		if len(updatedAPs) > 0 || len(createdAPs) > 0 {
			// Marshal the updated config to JSON
			jsonData, err := json.MarshalIndent(siteConfig, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal updated configuration: %w", err)
			}

			// Write back to file
			if err := os.WriteFile(siteConfigPath, jsonData, 0644); err != nil {
				return fmt.Errorf("failed to save updated configuration: %w", err)
			}

			fmt.Printf("\nSuccessfully updated site configuration: %s\n", siteConfigPath)
			if len(updatedAPs) > 0 {
				fmt.Printf("\nUpdated %d APs:\n", len(updatedAPs))
				for _, ap := range updatedAPs {
					fmt.Printf("  - %s\n", ap)
				}
			}
			if len(createdAPs) > 0 {
				fmt.Printf("\nCreated %d APs (with placeholder MACs):\n", len(createdAPs))
				for _, ap := range createdAPs {
					fmt.Printf("  - %s\n", ap)
				}
				fmt.Println("\nNote: Created APs have placeholder MAC addresses.")
				fmt.Println("   Replace with actual MACs when devices are discovered.")
			}
		} else {
			fmt.Println("\nWarning: No matching APs found to update in site configuration")
			if !createMode && len(notFoundAPs) > 0 {
				fmt.Println("   Use 'create' argument to auto-create missing AP entries.")
			}
		}

		if len(notFoundAPs) > 0 {
			fmt.Printf("\nWarning: APs from PDF not found in site configuration (%d):\n", len(notFoundAPs))
			for _, ap := range notFoundAPs {
				fmt.Printf("  - %s\n", ap)
			}
			if !createMode {
				fmt.Println("\n   Tip: Use 'create' argument to auto-create these AP entries.")
			}
		}

		// Display summary table
		displayAPSummary(apConfigs, apPrefix, apSuffix, defaults)

		return nil
	},
}

// APDefaults represents the defaults configuration for APs
type APDefaults struct {
	Band24 struct {
		Bandwidth int `json:"bandwidth" mapstructure:"bandwidth"`
	} `json:"band_24" mapstructure:"band_24"`
	Band5 struct {
		Bandwidth int `json:"bandwidth" mapstructure:"bandwidth"`
	} `json:"band_5" mapstructure:"band_5"`
	Band6 struct {
		Bandwidth int `json:"bandwidth" mapstructure:"bandwidth"`
	} `json:"band_6" mapstructure:"band_6"`
}

// loadDefaults loads the defaults configuration from the app config
func loadDefaults() *APDefaults {
	defaults := &APDefaults{}

	// Set default values if not configured
	defaults.Band24.Bandwidth = 20
	defaults.Band5.Bandwidth = 20
	defaults.Band6.Bandwidth = 20

	// Try to load from viper config
	if viper.IsSet("defaults.ap.band_24.bandwidth") {
		defaults.Band24.Bandwidth = viper.GetInt("defaults.ap.band_24.bandwidth")
	}
	if viper.IsSet("defaults.ap.band_5.bandwidth") {
		defaults.Band5.Bandwidth = viper.GetInt("defaults.ap.band_5.bandwidth")
	}
	if viper.IsSet("defaults.ap.band_6.bandwidth") {
		defaults.Band6.Bandwidth = viper.GetInt("defaults.ap.band_6.bandwidth")
	}

	logging.Debugf("Loaded defaults - 2.4G: %dMHz, 5G: %dMHz, 6G: %dMHz",
		defaults.Band24.Bandwidth, defaults.Band5.Bandwidth, defaults.Band6.Bandwidth)

	return defaults
}

// updateAPRadioConfig updates the radio configuration of an AP based on PDF data
func updateAPRadioConfig(configAP *config.APConfig, pdfAP *pdf.APConfig, defaults *APDefaults) {
	// Initialize Config if needed
	if configAP.Config == (config.APHWConfig{}) {
		configAP.Config = config.APHWConfig{
			LEDEnabled: true, // Default
		}
	}

	// Update 2.4G band
	if pdfAP.Band24G != nil {
		configAP.Config.Band24 = convertToBandCfg(pdfAP.Band24G, defaults.Band24.Bandwidth)
	} else {
		configAP.Config.Band24.Disabled = true
	}

	// Update 5G band
	if pdfAP.Band5G != nil {
		configAP.Config.Band5 = convertToBandCfg(pdfAP.Band5G, defaults.Band5.Bandwidth)
	} else {
		configAP.Config.Band5.Disabled = true
	}

	// Update 6G band
	if pdfAP.Band6G != nil {
		configAP.Config.Band6 = convertToBandCfg(pdfAP.Band6G, defaults.Band6.Bandwidth)
	} else {
		configAP.Config.Band6.Disabled = true
	}
}

// convertToBandCfg converts PDF band config to site config band format
func convertToBandCfg(pdfBand *pdf.BandConfig, defaultBandwidth int) config.BandCfg {
	cfg := config.BandCfg{
		Disabled: false,
	}

	// Parse channel
	if pdfBand.Channel == "auto" || pdfBand.Channel == "0" {
		cfg.Channel = 0 // 0 means auto in the config
	} else if channel, err := strconv.Atoi(pdfBand.Channel); err == nil {
		cfg.Channel = channel
	}

	// Parse power
	if pdfBand.Power == "auto" {
		cfg.TxPower = 0 // 0 or omitted means auto
	} else if power, err := strconv.Atoi(pdfBand.Power); err == nil {
		cfg.TxPower = power
	}

	// Parse width/bandwidth - use default if empty or not parseable
	if pdfBand.Width != "" {
		if width, err := strconv.Atoi(pdfBand.Width); err == nil {
			cfg.Bandwidth = width
		} else {
			cfg.Bandwidth = defaultBandwidth
		}
	} else {
		cfg.Bandwidth = defaultBandwidth
	}

	return cfg
}

// displayAPSummary displays a summary table of AP configurations from PDF
func displayAPSummary(apConfigs []*pdf.APConfig, prefix, suffix string, defaults *APDefaults) {
	// Convert to table data format
	tableData := make([]map[string]interface{}, 0, len(apConfigs))
	for _, ap := range apConfigs {
		// Format AP name with prefix/suffix shown in parentheses
		displayName := ap.Name
		if prefix != "" || suffix != "" {
			// Show actual name with prefix/suffix in gray parentheses
			fullName := prefix + ap.Name + suffix
			displayName = fmt.Sprintf("%s (%s)", ap.Name, fullName)
		}

		row := map[string]interface{}{
			"ap_name": displayName,
		}

		// Format 2.4G band
		if ap.Band24G != nil {
			row["band_24g"] = formatBandConfigWithDefaults(ap.Band24G, defaults.Band24.Bandwidth)
		} else {
			row["band_24g"] = "disabled"
		}

		// Format 5G band
		if ap.Band5G != nil {
			row["band_5g"] = formatBandConfigWithDefaults(ap.Band5G, defaults.Band5.Bandwidth)
		} else {
			row["band_5g"] = "disabled"
		}

		// Format 6G band
		if ap.Band6G != nil {
			row["band_6g"] = formatBandConfigWithDefaults(ap.Band6G, defaults.Band6.Bandwidth)
		} else {
			row["band_6g"] = "disabled"
		}

		tableData = append(tableData, row)
	}

	// Define columns for the table
	columns := []formatter.SimpleColumn{
		{Header: "AP Name", Field: "ap_name"},
		{Header: "2.4G", Field: "band_24g"},
		{Header: "5G", Field: "band_5g"},
		{Header: "6G", Field: "band_6g"},
	}

	// Configure table options
	options := formatter.SimpleTableOptions{
		Title:         "\nAP Configurations from PDF:",
		BoldHeaders:   true,
		ShowSeparator: true,
	}

	// Render the table
	tableOutput := formatter.RenderSimpleTable(tableData, columns, options)
	fmt.Print(tableOutput)
}

// formatBandConfigWithDefaults formats band configuration and indicates defaults
func formatBandConfigWithDefaults(band *pdf.BandConfig, defaultBandwidth int) string {
	if band == nil {
		return "disabled"
	}

	// Build the display string with indicators for defaults
	channel := band.Channel
	power := band.Power
	width := band.Width

	// Mark auto values with parentheses to indicate they're special/default
	if channel == "auto" {
		channel = "(auto)"
	}
	if power == "auto" {
		power = "(auto)"
	}

	// If width is empty, it means default bandwidth was used
	if width == "" {
		width = fmt.Sprintf("(%d)", defaultBandwidth)
	} else if widthInt, err := strconv.Atoi(width); err == nil && widthInt == defaultBandwidth {
		// If the width matches the default, indicate it
		width = fmt.Sprintf("(%d)", widthInt)
	}

	// Format as channel/power/width with defaults/auto in parentheses
	return fmt.Sprintf("%s/%s/%s", channel, power, width)
}

func init() {
	importCmd.AddCommand(importPDFCmd)
}

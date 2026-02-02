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
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ravinald/wifimgr/internal/cmdutils"
	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/formatter"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/patterns"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// apiWLANsCmd represents the "show api wlans" command
var apiWLANsCmd = &cobra.Command{
	Use:   "wlans [site-name] [ssid-filter] [json|csv] [no-resolve]",
	Short: "Show WLANs/SSIDs from API cache",
	Long: `Show WLANs/SSIDs retrieved from the local API cache.

Without arguments, displays all WLANs in a table format.
With one argument, it tries to match a site name first, then falls back to SSID filter.
With two arguments, the first is treated as site name, the second as SSID filter.

Arguments:
  site-name    - Optional site name to filter by (checked first)
  ssid-filter  - Optional SSID name or WLAN ID to filter by
  json         - Output in JSON format (includes raw vendor config data)
  csv          - Output in CSV format
  no-resolve   - Disable field ID to name resolution

When multiple WLANs match (e.g., same SSID across sites), all are shown in a table.
When exactly one WLAN matches a filter, JSON details are shown automatically.

Examples:
  wifimgr show api wlans                        - Show all WLANs in table format
  wifimgr show api wlans US-LAB-01              - Show WLANs for site US-LAB-01
  wifimgr show api wlans Guest-WiFi             - Show WLANs matching "Guest-WiFi"
  wifimgr show api wlans US-LAB-01 Guest-WiFi   - Show "Guest-WiFi" WLAN in site US-LAB-01
  wifimgr show api wlans L_123456:3             - Show WLAN by Meraki ID
  wifimgr show api wlans json                   - Show all WLANs in JSON format
  wifimgr show api wlans US-LAB-01 json         - Show WLANs for site in JSON format
  wifimgr show api wlans no-resolve             - Show all WLANs without field resolution`,
	Args: func(cmd *cobra.Command, args []string) error {
		// Allow "help" as a special keyword
		for _, arg := range args {
			if strings.ToLower(arg) == "help" {
				return nil
			}
		}
		return nil
	},
	RunE: runShowAPIWLANs,
}

func init() {
	apiCmd.AddCommand(apiWLANsCmd)
}

func runShowAPIWLANs(cmd *cobra.Command, args []string) error {
	// Check for help keyword in positional arguments
	for _, arg := range args {
		if strings.ToLower(arg) == "help" {
			return cmd.Help()
		}
	}

	logger := logging.GetLogger()
	logger.Info("Executing show api wlans command")

	// Get cache accessor
	cacheAccessor, err := cmdutils.GetCacheAccessor()
	if err != nil {
		logger.WithError(err).Error("Failed to get cache accessor")
		return err
	}

	// Parse arguments - collect positional args and keywords separately
	var positionalArgs []string
	format := "table"
	noResolve := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "json":
			format = "json"
		case "csv":
			format = "csv"
		case "no-resolve":
			noResolve = true
		case "site":
			// Legacy support: "site <site-name>" syntax
			if i+1 < len(args) {
				i++
				positionalArgs = append([]string{args[i]}, positionalArgs...)
			}
		default:
			positionalArgs = append(positionalArgs, arg)
		}
	}

	// Determine site and SSID filters based on positional args
	var siteFilter string
	var ssidFilter string

	switch len(positionalArgs) {
	case 0:
		// No filters
	case 1:
		// Single arg: try to match as site first, then as SSID
		arg := positionalArgs[0]
		// Check if this matches a known site
		if site, err := cacheAccessor.GetSiteByName(arg); err == nil && site != nil {
			siteFilter = arg
		} else {
			// Not a known site name, treat as SSID filter
			ssidFilter = arg
		}
	default:
		// Two or more args: first is site, second is SSID
		siteFilter = positionalArgs[0]
		ssidFilter = positionalArgs[1]
	}

	// Get all WLANs
	wlans := cacheAccessor.GetAllWLANs()

	if len(wlans) == 0 {
		fmt.Println("No WLANs found in cache")
		return nil
	}

	// Filter by site if specified
	if siteFilter != "" {
		var filtered []*vendors.WLAN
		for _, w := range wlans {
			// Try to match site by name or ID
			if w.SiteID != "" {
				if patterns.Contains(w.SiteID, siteFilter) {
					filtered = append(filtered, w)
				} else if !noResolve {
					// Try to resolve site name
					if site, err := cacheAccessor.GetSiteByID(w.SiteID); err == nil && site.Name != "" {
						if patterns.Contains(site.Name, siteFilter) {
							filtered = append(filtered, w)
						}
					}
				}
			} else if w.SiteID == "" {
				// Org-level WLANs: include if site filter matches "org" or similar
				if patterns.Contains("org-level", siteFilter) || patterns.Contains("org", siteFilter) {
					filtered = append(filtered, w)
				}
			}
		}
		wlans = filtered
	}

	// Filter by SSID name/ID if specified
	if ssidFilter != "" {
		var filtered []*vendors.WLAN
		for _, w := range wlans {
			if patterns.Contains(w.SSID, ssidFilter) || patterns.Contains(w.ID, ssidFilter) {
				filtered = append(filtered, w)
			}
		}
		wlans = filtered
	}

	if len(wlans) == 0 {
		if ssidFilter != "" && siteFilter != "" {
			fmt.Printf("No WLANs found matching '%s' in site '%s'\n", ssidFilter, siteFilter)
		} else if ssidFilter != "" {
			fmt.Printf("No WLANs found matching '%s'\n", ssidFilter)
		} else if siteFilter != "" {
			fmt.Printf("No WLANs found for site '%s'\n", siteFilter)
		}
		return nil
	}

	// Sort WLANs by site name, then by SSID
	sort.Slice(wlans, func(i, j int) bool {
		// Resolve site names for comparison
		siteNameI := wlans[i].SiteID
		siteNameJ := wlans[j].SiteID
		if site, err := cacheAccessor.GetSiteByID(wlans[i].SiteID); err == nil && site.Name != "" {
			siteNameI = site.Name
		}
		if site, err := cacheAccessor.GetSiteByID(wlans[j].SiteID); err == nil && site.Name != "" {
			siteNameJ = site.Name
		}
		if siteNameI != siteNameJ {
			return siteNameI < siteNameJ
		}
		return wlans[i].SSID < wlans[j].SSID
	})

	// Handle JSON output
	if format == "json" {
		return outputWLANsJSON(wlans)
	}

	// If single WLAN matched with a filter and table format, show JSON details
	// (only when user explicitly filtered, not when showing all)
	if len(wlans) == 1 && (ssidFilter != "" || siteFilter != "") && format == "table" {
		return showWLANDetails(wlans[0])
	}

	// Build table data
	return outputWLANsTable(wlans, cacheAccessor, noResolve, format)
}

func showWLANDetails(wlan *vendors.WLAN) error {
	// Marshal and print with color using MarshalJSONWithColorIndent
	jsonData, err := formatter.MarshalJSONWithColorIndent(wlan, "", "  ")
	if err != nil {
		logging.GetLogger().WithError(err).Error("Failed to marshal WLAN to JSON")
		return err
	}

	fmt.Println(string(jsonData))
	return nil
}

func outputWLANsJSON(wlans []*vendors.WLAN) error {
	var jsonData []byte
	var err error

	if len(wlans) == 1 {
		jsonData, err = formatter.MarshalJSONWithColorIndent(wlans[0], "", "  ")
	} else {
		jsonData, err = formatter.MarshalJSONWithColorIndent(wlans, "", "  ")
	}

	if err != nil {
		return fmt.Errorf("error marshalling JSON: %w", err)
	}

	fmt.Println(string(jsonData))
	return nil
}

func outputWLANsTable(wlans []*vendors.WLAN, cacheAccessor *vendors.CacheAccessor, noResolve bool, format string) error {
	var tableData []formatter.GenericTableData

	for _, wlan := range wlans {
		row := make(map[string]interface{})

		row["ssid"] = wlan.SSID
		row["id"] = wlan.ID

		// Enabled status - store as bool for formatter to handle
		row["enabled"] = wlan.Enabled

		// Hidden status - store as bool for formatter to handle
		row["hidden"] = wlan.Hidden

		// Site/Network information
		if wlan.SiteID != "" {
			if !noResolve {
				if site, err := cacheAccessor.GetSiteByID(wlan.SiteID); err == nil && site.Name != "" {
					row["site"] = site.Name
				} else {
					row["site"] = wlan.SiteID
				}
			} else {
				row["site"] = wlan.SiteID
			}
		} else {
			row["site"] = "(org-level)"
		}

		// Authentication type
		row["auth_type"] = wlan.AuthType

		// Encryption mode
		row["encryption"] = wlan.EncryptionMode

		// Band
		row["band"] = wlan.Band

		// VLAN
		if wlan.VLANID > 0 {
			row["vlan"] = wlan.VLANID
		} else {
			row["vlan"] = ""
		}

		// Vendor
		row["vendor"] = wlan.SourceVendor

		tableData = append(tableData, formatter.GenericTableData(row))
	}

	// Define default columns
	columns := []formatter.TableColumn{
		{Field: "ssid", Title: "SSID"},
		{Field: "enabled", Title: "Enabled"},
		{Field: "hidden", Title: "Hidden"},
		{Field: "site", Title: "Site/Network"},
		{Field: "auth_type", Title: "Auth Type"},
		{Field: "encryption", Title: "Encryption"},
		{Field: "band", Title: "Band"},
		{Field: "vlan", Title: "VLAN"},
		{Field: "vendor", Title: "Vendor"},
		{Field: "id", Title: "ID"},
	}

	// Create command path for config lookup
	commandPath := "show.api.wlans"

	// Check if there's a command-specific format in the config
	displayCommands := viper.GetStringMap("display.commands")
	commandFormatRaw, hasCommandConfig := displayCommands[commandPath]

	var commandFormat config.CommandFormat
	if hasCommandConfig {
		if cmdMap, ok := commandFormatRaw.(map[string]interface{}); ok {
			if formatVal, ok := cmdMap["format"].(string); ok {
				commandFormat.Format = formatVal
			}
			if fieldsVal, ok := cmdMap["fields"].([]interface{}); ok {
				commandFormat.Fields = fieldsVal
			}
			if titleVal, ok := cmdMap["title"].(string); ok {
				commandFormat.Title = titleVal
			}
		}
	}

	// Build title
	title := fmt.Sprintf("WLANs (%d)", len(tableData))
	if hasCommandConfig && commandFormat.Title != "" {
		title = commandFormat.Title
	}

	// Create table config
	tableConfig := formatter.TableConfig{
		Title:       title,
		Columns:     columns,
		Format:      format,
		BoldHeaders: true,
		CommandPath: commandPath,
	}

	// Set format from config if not overridden by argument
	if tableConfig.Format == "" && hasCommandConfig {
		tableConfig.Format = commandFormat.Format
	}
	if tableConfig.Format == "" {
		tableConfig.Format = "table"
	}

	// Print table
	printer := formatter.NewGenericTablePrinter(tableConfig, tableData)

	// Use config-driven columns if available, otherwise use defaults
	if hasCommandConfig && commandFormat.Fields != nil {
		printer.LoadColumnsFromConfig(commandFormat.Fields)
	} else {
		printer.Config.Columns = columns
	}

	fmt.Print(printer.Print())

	return nil
}

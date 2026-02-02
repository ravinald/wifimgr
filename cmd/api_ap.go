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
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ravinald/wifimgr/internal/cmdutils"
	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/formatter"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// apiApCmd represents the "show api ap" command
var apiApCmd = &cobra.Command{
	Use:   "ap [name-or-mac] [site site-name] [target api-label] [format] [all] [no-resolve]",
	Short: "Show access point information from API cache",
	Long: `Show access point information retrieved from the local API cache.

This command displays AP device data from the local cache with connection status indicators.

When multiple APIs are configured:
  - Without target: Aggregates APs from all APIs
  - With target: Shows APs from the specified API only

Arguments:
  name-or-mac  - Optional AP name or MAC address filter
  site         - Keyword followed by site name for filtering
  target       - Keyword followed by API label to target specific API
  format       - Output format: "json" or "csv" (default: table)
  all          - Show all fields (json format only)
  no-resolve   - Disable field ID to name resolution

Examples:
  wifimgr show api ap                          - Show all APs
  wifimgr show api ap site US-LAB-01           - Show APs for specific site
  wifimgr show api ap AP-NAME                  - Show specific AP by name
  wifimgr show api ap 00:11:22:33:44:55        - Show specific AP by MAC address
  wifimgr show api ap json                     - Show all APs in JSON format
  wifimgr show api ap AP-NAME json all         - Show all fields for AP in JSON
  wifimgr show api ap json no-resolve          - Show JSON with raw IDs
  wifimgr show api ap target mist-prod         - Show APs from mist-prod only`,
	Args: cmdutils.ValidateShowAPArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check for help keyword in positional arguments
		if cmdutils.ContainsHelp(args) {
			return cmd.Help()
		}

		// Parse positional arguments using the utility
		parsed, err := cmdutils.ParseShowArgs(args)
		if err != nil {
			return err
		}

		// Set API target from positional argument
		SetAPITarget(parsed.Target)

		return showAPsMultiVendor(globalContext, parsed)
	},
}

// showAPsMultiVendor shows APs from one or more APIs in multi-vendor mode.
// APs that are in the inventory.json file are highlighted in green.
func showAPsMultiVendor(_ context.Context, parsed *cmdutils.ParsedShowArgs) error {
	// Validate --api flag if provided
	if err := ValidateAPIFlag(); err != nil {
		return err
	}

	cacheMgr := GetCacheManager()
	if cacheMgr == nil {
		return fmt.Errorf("cache manager not initialized")
	}

	targetAPIs := GetTargetAPIs()
	if len(targetAPIs) == 0 {
		return fmt.Errorf("no APIs configured")
	}

	// Load managed MACs from inventory file for highlighting
	inventoryPath := viper.GetString("files.inventory")
	var managedMACs map[string]string
	if inventoryPath != "" {
		managedMACs = loadManagedMACs(inventoryPath, []string{"ap"})
		if managedMACs != nil {
			logging.Debugf("Loaded %d managed AP MACs for highlighting", len(managedMACs))
		}
	}

	// Collect APs from all target APIs
	var allAPs []formatter.GenericTableData
	apiCounts := make(map[string]int)

	for _, apiLabel := range targetAPIs {
		cache, err := cacheMgr.GetAPICache(apiLabel)
		if err != nil {
			// Skip APIs with no cache
			continue
		}

		for mac, item := range cache.Inventory.AP {
			normalizedMAC := vendors.NormalizeMAC(mac)

			// Apply site filter if specified
			if parsed.SiteName != "" {
				siteID, ok := cache.SiteIndex.ByName[parsed.SiteName]
				if !ok || item.SiteID != siteID {
					continue
				}
			}

			// Apply name/MAC filter if specified
			if parsed.Filter != "" {
				if cmdutils.IsMAC(parsed.Filter) {
					if normalizedMAC != vendors.NormalizeMAC(parsed.Filter) {
						continue
					}
				} else {
					if item.Name == "" || !containsIgnoreCase(item.Name, parsed.Filter) {
						continue
					}
				}
			}

			// Check if device is in managed inventory for highlighting
			_, isManaged := managedMACs[normalizedMAC]

			// Convert to table data - highlight name if managed
			displayName := item.Name
			if isManaged && displayName != "" {
				displayName = "GREEN_TEXT:" + displayName
			}

			data := formatter.GenericTableData{
				"name":    displayName,
				"mac":     item.MAC,
				"serial":  item.Serial,
				"model":   item.Model,
				"type":    item.Type,
				"site_id": item.SiteID,
				"api":     apiLabel,
			}

			// Look up status from DeviceStatus section
			if status, ok := cache.DeviceStatus[normalizedMAC]; ok {
				data["status"] = status.Status
			} else {
				data["status"] = "offline"
			}

			// Resolve site name from cache (unless no-resolve is set)
			if parsed.NoResolve {
				// Show raw site_id when no-resolve is set
				data["site_name"] = item.SiteID
			} else if siteName, ok := cache.SiteIndex.ByID[item.SiteID]; ok {
				data["site_name"] = siteName
			} else {
				// Fallback to site_id if name not found
				data["site_name"] = item.SiteID
			}

			allAPs = append(allAPs, data)
			apiCounts[apiLabel]++
		}
	}

	// Sort APs by site, name, type, mac
	formatter.SortTableData(allAPs)

	// Apply field resolution (convert field IDs to names)
	if !parsed.NoResolve {
		if err := cmdutils.ApplyFieldResolution(allAPs, true); err != nil {
			logging.Debugf("Field resolution warning: %v", err)
		}
	}

	// Build title
	title := fmt.Sprintf("AP Devices (%d)", len(allAPs))
	if len(targetAPIs) > 1 {
		title = fmt.Sprintf("AP Devices (%d from %d APIs)", len(allAPs), len(apiCounts))
	} else if apiFlag != "" {
		title = fmt.Sprintf("AP Devices from %s (%d)", apiFlag, len(allAPs))
	}

	if len(allAPs) == 0 {
		fmt.Printf("%s:\n", title)
		fmt.Println("No AP devices found")
		return nil
	}

	// Determine columns - add API column when showing multiple APIs
	columns := []formatter.TableColumn{
		{Field: "name", Title: "Name", MaxWidth: 0},
		{Field: "mac", Title: "MAC", MaxWidth: 0},
		{Field: "serial", Title: "Serial", MaxWidth: 0},
		{Field: "model", Title: "Model", MaxWidth: 0},
		{Field: "status", Title: "Status", MaxWidth: 0, IsStatusField: true},
		{Field: "site_name", Title: "Site", MaxWidth: 0},
	}

	// Add API column when showing from multiple APIs
	if len(targetAPIs) > 1 || apiFlag == "" {
		columns = append(columns, formatter.TableColumn{Field: "api", Title: "API", MaxWidth: 0})
	}

	// Create command path for config lookup
	commandPath := "show.api.ap"

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

	// Create cache accessor for cache.* field lookups
	cacheAccessor, err := cmdutils.NewCacheTableAccessor()
	if err != nil {
		logging.Debugf("Cache accessor not available: %v", err)
		cacheAccessor = nil
	}

	// Override title from config if available
	if hasCommandConfig && commandFormat.Title != "" {
		title = commandFormat.Title
	}

	// Create table config
	tableConfig := formatter.TableConfig{
		Title:         title,
		Format:        parsed.Format,
		BoldHeaders:   true,
		ShowSeparator: true,
		CommandPath:   commandPath,
		CacheAccess:   cacheAccessor,
		ShowAllFields: parsed.ShowAll,
		Columns:       columns,
	}

	// Set format from config if not overridden by argument
	if tableConfig.Format == "" && hasCommandConfig {
		tableConfig.Format = commandFormat.Format
	}
	if tableConfig.Format == "" {
		tableConfig.Format = "table"
	}

	// Create table printer
	printer := formatter.NewGenericTablePrinter(tableConfig, allAPs)

	// Use config-driven columns if available, otherwise use defaults
	if hasCommandConfig && commandFormat.Fields != nil {
		printer.LoadColumnsFromConfig(commandFormat.Fields)
	} else {
		printer.Config.Columns = columns
	}

	fmt.Print(printer.Print())

	// Show cache timestamp
	printCacheTimestamp(cacheMgr, targetAPIs, tableConfig.Format)

	return nil
}

// containsIgnoreCase checks if s contains substr (case-insensitive).
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(substr) == 0 ||
		(len(s) > 0 && containsIgnoreCase(s[1:], substr)) ||
		(len(s) >= len(substr) && equalFoldPrefix(s, substr) && containsIgnoreCase(s[len(substr):], "")))
}

func equalFoldPrefix(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		c1, c2 := s[i], prefix[i]
		if c1 >= 'A' && c1 <= 'Z' {
			c1 += 'a' - 'A'
		}
		if c2 >= 'A' && c2 <= 'Z' {
			c2 += 'a' - 'A'
		}
		if c1 != c2 {
			return false
		}
	}
	return true
}

func init() {
	apiCmd.AddCommand(apiApCmd)
}

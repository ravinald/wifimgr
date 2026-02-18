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

	"github.com/ravinald/wifimgr/internal/cmdutils"
	"github.com/ravinald/wifimgr/internal/formatter"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/macaddr"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// apiBSSIDCmd represents the "show api bssid" command
var apiBSSIDCmd = &cobra.Command{
	Use:   "bssid [bssid-or-ap-name] [essid ssid-name] [sort essid|ap] [site site-name] [target api-label] [format json|csv] [all] [no-resolve]",
	Short: "Show BSSID-to-AP mappings from API cache",
	Long: `Show BSSID-to-AP mappings retrieved from the local API cache.

This command displays all BSSIDs with their parent AP, SSID, radio band, and broadcast status.
Use it to identify which AP and SSID a given BSSID belongs to.

Results are always sorted by site name. Use "sort" to add a secondary sort by SSID or AP name.

When multiple APIs are configured:
  - Without target: Aggregates BSSIDs from all APIs
  - With target: Shows BSSIDs from the specified API only

Arguments:
  bssid-or-ap-name  - Optional BSSID MAC address or AP name filter
  essid             - Keyword followed by SSID name to filter by wireless network
  sort              - Keyword followed by secondary sort: "essid" or "ap"
  site              - Keyword followed by site name for filtering
  target            - Keyword followed by API label to target specific API
  format            - Output format: "json" or "csv" (default: table)
  all               - Show all fields (json format only)
  no-resolve        - Disable field ID to name resolution

Examples:
  wifimgr show api bssid                          - Show all BSSIDs
  wifimgr show api bssid site US-LAB-01           - Show BSSIDs for specific site
  wifimgr show api bssid essid Corp-WiFi          - Show BSSIDs broadcasting Corp-WiFi
  wifimgr show api bssid essid "Guest WiFi"       - Filter by SSID with spaces
  wifimgr show api bssid sort essid               - Sort by site, then SSID name
  wifimgr show api bssid sort ap                  - Sort by site, then AP name
  wifimgr show api bssid AP-NAME essid Corp-WiFi  - AP filter + SSID filter combined
  wifimgr show api bssid aa:bb:cc:dd:ee:ff        - Find specific BSSID
  wifimgr show api bssid AP-NAME                  - Show BSSIDs from matching APs
  wifimgr show api bssid json                     - Show all BSSIDs in JSON format
  wifimgr show api bssid json all                 - Show all fields in JSON
  wifimgr show api bssid target mist-prod         - Show BSSIDs from mist-prod only`,
	Args: cmdutils.ValidateShowAPArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if cmdutils.ContainsHelp(args) {
			return cmd.Help()
		}

		parsed, err := cmdutils.ParseShowArgs(args)
		if err != nil {
			return err
		}

		SetAPITarget(parsed.Target)

		return showBSSIDsMultiVendor(globalContext, parsed)
	},
}

// showBSSIDsMultiVendor shows BSSIDs from one or more APIs in multi-vendor mode.
func showBSSIDsMultiVendor(_ context.Context, parsed *cmdutils.ParsedShowArgs) error {
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

	var allBSSIDs []formatter.GenericTableData
	apiCounts := make(map[string]int)

	for _, apiLabel := range targetAPIs {
		cache, err := cacheMgr.GetAPICache(apiLabel)
		if err != nil {
			continue
		}

		for bssid, entry := range cache.BSSIDs {
			normalizedBSSID := vendors.NormalizeMAC(bssid)

			// Apply site filter
			if parsed.SiteName != "" {
				siteID, ok := cache.SiteIndex.ByName[parsed.SiteName]
				if !ok || entry.SiteID != siteID {
					// Also try matching by site name directly (some vendors store name)
					if entry.SiteName != parsed.SiteName {
						continue
					}
				}
			}

			// Apply filter (BSSID MAC or AP name)
			if parsed.Filter != "" {
				if cmdutils.IsMAC(parsed.Filter) {
					if normalizedBSSID != vendors.NormalizeMAC(parsed.Filter) {
						continue
					}
				} else {
					if entry.APName == "" || !containsIgnoreCase(entry.APName, parsed.Filter) {
						continue
					}
				}
			}

			// Apply ESSID filter
			if parsed.ESSIDName != "" {
				if !containsIgnoreCase(entry.SSIDName, parsed.ESSIDName) {
					continue
				}
			}

			// Look up AP status
			status := "offline"
			if entry.APMAC != "" {
				if ds, ok := cache.DeviceStatus[vendors.NormalizeMAC(entry.APMAC)]; ok {
					status = ds.Status
				}
			}

			// Format BSSID with colons for display
			displayBSSID := normalizedBSSID
			if formatted, err := macaddr.Format(normalizedBSSID, macaddr.FormatColon); err == nil {
				displayBSSID = formatted
			}

			broadcasting := "no"
			if entry.IsBroadcasting {
				broadcasting = "yes"
			}

			// Resolve site name
			siteName := entry.SiteName
			if !parsed.NoResolve && siteName == "" {
				if name, ok := cache.SiteIndex.ByID[entry.SiteID]; ok {
					siteName = name
				}
			}
			if siteName == "" {
				siteName = entry.SiteID
			}

			data := formatter.GenericTableData{
				"status":        status,
				"site_name":     siteName,
				"ap_name":       entry.APName,
				"bssid":         displayBSSID,
				"ssid_name":     entry.SSIDName,
				"ssid_number":   entry.SSIDNumber,
				"band":          entry.Band,
				"channel":       entry.Channel,
				"channel_width": entry.ChannelWidth,
				"power":         entry.Power,
				"broadcasting":  broadcasting,
				"ap_serial":     entry.APSerial,
				"ap_mac":        entry.APMAC,
				"site_id":       entry.SiteID,
				"api":           apiLabel,
			}

			allBSSIDs = append(allBSSIDs, data)
			apiCounts[apiLabel]++
		}
	}

	// Primary sort: site_name; secondary/tertiary from "sort" keyword
	sortFields := []string{"site_name"}
	switch parsed.SortField {
	case "essid":
		sortFields = append(sortFields, "ssid_name", "ap_name")
	case "ap":
		sortFields = append(sortFields, "ap_name", "ssid_name")
	}
	sortFields = append(sortFields, "bssid")
	formatter.SortTableDataBy(allBSSIDs, sortFields...)

	// Build title
	title := fmt.Sprintf("BSSIDs (%d)", len(allBSSIDs))
	if len(targetAPIs) > 1 {
		title = fmt.Sprintf("BSSIDs (%d from %d APIs)", len(allBSSIDs), len(apiCounts))
	} else if apiFlag != "" {
		title = fmt.Sprintf("BSSIDs from %s (%d)", apiFlag, len(allBSSIDs))
	}

	if len(allBSSIDs) == 0 {
		fmt.Printf("%s:\n", title)
		fmt.Println("No BSSIDs found")
		return nil
	}

	columns := []formatter.TableColumn{
		{Field: "status", Title: "Status", IsStatusField: true},
		{Field: "site_name", Title: "Site"},
		{Field: "ap_name", Title: "AP Name"},
		{Field: "bssid", Title: "BSSID"},
		{Field: "ssid_name", Title: "SSID"},
		{Field: "band", Title: "Band"},
		{Field: "channel", Title: "Ch"},
		{Field: "broadcasting", Title: "Bcast"},
	}

	if len(targetAPIs) > 1 || apiFlag == "" {
		columns = append(columns, formatter.TableColumn{Field: "api", Title: "API"})
	}

	commandPath := "show.api.bssid"

	// Create cache accessor for cache.* field lookups
	cacheAccessor, err := cmdutils.NewCacheTableAccessor()
	if err != nil {
		logging.Debugf("Cache accessor not available: %v", err)
		cacheAccessor = nil
	}

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

	if tableConfig.Format == "" {
		tableConfig.Format = "table"
	}

	printer := formatter.NewGenericTablePrinter(tableConfig, allBSSIDs)
	printer.Config.Columns = columns

	fmt.Print(printer.Print())

	printCacheTimestamp(cacheMgr, targetAPIs, tableConfig.Format)

	return nil
}

func init() {
	apiCmd.AddCommand(apiBSSIDCmd)
}

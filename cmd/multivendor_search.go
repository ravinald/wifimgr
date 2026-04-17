package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/ravinald/wifimgr/internal/formatter"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// searchWirelessMultiVendor searches for wireless clients across multiple APIs.
// When detail is true, extra columns (Band, State) are rendered and populated
// from live response (Status) and the per-client detail cache (Band).
func searchWirelessMultiVendor(ctx context.Context, searchText, siteID, format string, force, _, detail bool) error {
	// Validate target API if provided
	if err := ValidateAPIFlag(); err != nil {
		return err
	}

	registry := GetAPIRegistry()
	if registry == nil {
		return fmt.Errorf("API registry not initialized")
	}

	targetAPIs := GetTargetAPIs()
	if len(targetAPIs) == 0 {
		return fmt.Errorf("no APIs configured")
	}

	// Phase 1: Estimate costs and confirm if needed
	if !force {
		if err := confirmExpensiveSearchIfNeeded(ctx, registry, targetAPIs, searchText, siteID); err != nil {
			return err
		}
	}

	// Collect results from all target APIs
	var allResults []formatter.GenericTableData
	apiCounts := make(map[string]int)
	apisWithSearch := 0

	// Detail-mode bookkeeping: how many rows got a cached Band record and the
	// freshest FetchedAt across them, for the footer timestamp.
	var (
		detailCacheHits   int
		newestDetailFetch time.Time
	)

	cacheMgr := GetCacheManager()

	for _, apiLabel := range targetAPIs {
		client, err := registry.GetClient(apiLabel)
		if err != nil {
			continue
		}

		// Check if vendor supports search
		searchSvc := client.Search()
		if searchSvc == nil {
			// This vendor doesn't support search
			continue
		}
		apisWithSearch++

		opts := vendors.SearchOptions{SiteID: resolveSearchSiteID(cacheMgr, apiLabel, siteID)}

		results, err := searchSvc.SearchWirelessClients(ctx, searchText, opts)
		if err != nil {
			// Log error but continue with other APIs
			fmt.Printf("WARN  Search failed for %s: %v\n", apiLabel, err)
			continue
		}

		if results == nil || len(results.Results) == 0 {
			continue
		}

		// Pull the per-API cache once per loop iteration; enrichment is
		// best-effort, so a missing cache just leaves fields empty.
		var apiCache *vendors.APICache
		if cacheMgr != nil {
			apiCache, _ = cacheMgr.GetAPICache(apiLabel)
		}

		// Convert results to table data
		for _, client := range results.Results {
			enrichWirelessClientFromCache(client, apiCache)

			// Fill Band from the persistent client-detail cache when the
			// operator asked for detail. Doing it here keeps the default
			// path identical to before — cost unchanged, no cache hits.
			if detail && cacheMgr != nil && client.Band == "" && client.MAC != "" {
				if rec, ok := cacheMgr.LookupClientDetail(apiLabel, client.MAC); ok {
					client.Band = rec.Band
					if rec.FetchedAt.After(newestDetailFetch) {
						newestDetailFetch = rec.FetchedAt
					}
					detailCacheHits++
				}
			}

			vendorName, _ := registry.GetVendor(apiLabel)
			data := formatter.GenericTableData{
				"mac":       client.MAC,
				"ip":        client.IP,
				"hostname":  client.Hostname,
				"ssid":      client.SSID,
				"ap_name":   client.APName,
				"ap_mac":    client.APMAC,
				"band":      client.Band,
				"state":     client.Status,
				"vlan":      client.VLAN,
				"site_id":   client.SiteID,
				"site_name": client.SiteName,
				"api":       apiLabel,
				"vendor":    vendorName,
			}
			allResults = append(allResults, data)
			apiCounts[apiLabel]++
		}
	}

	if apisWithSearch == 0 {
		return fmt.Errorf("no APIs support wireless client search")
	}

	// Build title
	title := fmt.Sprintf("Wireless Clients (%d)", len(allResults))
	if len(apiCounts) > 1 {
		title = fmt.Sprintf("Wireless Clients (%d from %d APIs)", len(allResults), len(apiCounts))
	} else if apiFlag != "" {
		title = fmt.Sprintf("Wireless Clients from %s (%d)", apiFlag, len(allResults))
	}

	if len(allResults) == 0 {
		fmt.Printf("%s:\n", title)
		fmt.Printf("No clients found matching '%s'\n", searchText)
		return nil
	}

	columns := buildWirelessSearchColumns(siteID, len(targetAPIs), detail)

	// Create table config
	tableConfig := formatter.TableConfig{
		Title:         title,
		Format:        format,
		BoldHeaders:   true,
		ShowSeparator: true,
		Columns:       columns,
	}

	if tableConfig.Format == "" {
		tableConfig.Format = "table"
	}

	// Create and print table
	printer := formatter.NewGenericTablePrinter(tableConfig, allResults)
	printer.Config.Columns = columns
	fmt.Print(printer.Print())

	if detail {
		printDetailFooter(detailCacheHits, newestDetailFetch, siteID)
	}

	return nil
}

// searchWiredMultiVendor searches for wired clients across multiple APIs.
func searchWiredMultiVendor(ctx context.Context, searchText, siteID, format string, force, _, _ bool) error {
	// Validate target API if provided
	if err := ValidateAPIFlag(); err != nil {
		return err
	}

	registry := GetAPIRegistry()
	if registry == nil {
		return fmt.Errorf("API registry not initialized")
	}

	targetAPIs := GetTargetAPIs()
	if len(targetAPIs) == 0 {
		return fmt.Errorf("no APIs configured")
	}

	// Phase 1: Estimate costs and confirm if needed
	if !force {
		if err := confirmExpensiveSearchIfNeeded(ctx, registry, targetAPIs, searchText, siteID); err != nil {
			return err
		}
	}

	// Collect results from all target APIs
	var allResults []formatter.GenericTableData
	apiCounts := make(map[string]int)
	apisWithSearch := 0

	cacheMgr := GetCacheManager()

	for _, apiLabel := range targetAPIs {
		client, err := registry.GetClient(apiLabel)
		if err != nil {
			continue
		}

		// Check if vendor supports search
		searchSvc := client.Search()
		if searchSvc == nil {
			// This vendor doesn't support search
			continue
		}
		apisWithSearch++

		opts := vendors.SearchOptions{SiteID: resolveSearchSiteID(cacheMgr, apiLabel, siteID)}

		results, err := searchSvc.SearchWiredClients(ctx, searchText, opts)
		if err != nil {
			// Log error but continue with other APIs
			fmt.Printf("WARN  Search failed for %s: %v\n", apiLabel, err)
			continue
		}

		if results == nil || len(results.Results) == 0 {
			continue
		}

		var apiCache *vendors.APICache
		if cacheMgr != nil {
			apiCache, _ = cacheMgr.GetAPICache(apiLabel)
		}

		// Convert results to table data
		for _, wiredClient := range results.Results {
			enrichWiredClientFromCache(wiredClient, apiCache)
			vendorName, _ := registry.GetVendor(apiLabel)
			data := formatter.GenericTableData{
				"mac":         wiredClient.MAC,
				"ip":          wiredClient.IP,
				"hostname":    wiredClient.Hostname,
				"switch_name": wiredClient.SwitchName,
				"switch_mac":  wiredClient.SwitchMAC,
				"port":        wiredClient.PortID,
				"vlan":        wiredClient.VLAN,
				"site_id":     wiredClient.SiteID,
				"site_name":   wiredClient.SiteName,
				"api":         apiLabel,
				"vendor":      vendorName,
			}
			allResults = append(allResults, data)
			apiCounts[apiLabel]++
		}
	}

	if apisWithSearch == 0 {
		return fmt.Errorf("no APIs support wired client search")
	}

	// Build title
	title := fmt.Sprintf("Wired Clients (%d)", len(allResults))
	if len(apiCounts) > 1 {
		title = fmt.Sprintf("Wired Clients (%d from %d APIs)", len(allResults), len(apiCounts))
	} else if apiFlag != "" {
		title = fmt.Sprintf("Wired Clients from %s (%d)", apiFlag, len(allResults))
	}

	if len(allResults) == 0 {
		fmt.Printf("%s:\n", title)
		fmt.Printf("No clients found matching '%s'\n", searchText)
		return nil
	}

	columns := buildWiredSearchColumns(siteID, len(targetAPIs), false)

	// Create table config
	tableConfig := formatter.TableConfig{
		Title:         title,
		Format:        format,
		BoldHeaders:   true,
		ShowSeparator: true,
		Columns:       columns,
	}

	if tableConfig.Format == "" {
		tableConfig.Format = "table"
	}

	// Create and print table
	printer := formatter.NewGenericTablePrinter(tableConfig, allResults)
	printer.Config.Columns = columns
	fmt.Print(printer.Print())

	return nil
}

// confirmExpensiveSearchIfNeeded estimates search cost and prompts for confirmation if needed.
func confirmExpensiveSearchIfNeeded(ctx context.Context, registry *vendors.APIClientRegistry, targetAPIs []string, searchText, siteID string) error {
	totalCalls := 0
	var expensiveAPIs []string

	for _, apiLabel := range targetAPIs {
		client, err := registry.GetClient(apiLabel)
		if err != nil {
			continue
		}

		searchSvc := client.Search()
		if searchSvc == nil {
			continue
		}

		estimate, err := searchSvc.EstimateSearchCost(ctx, searchText, siteID)
		if err != nil {
			continue
		}

		totalCalls += estimate.APICalls
		if estimate.NeedsConfirmation {
			expensiveAPIs = append(expensiveAPIs,
				fmt.Sprintf("%s: %s", apiLabel, estimate.Description))
		}
	}

	// Prompt if any API requires confirmation and we're in interactive mode
	if len(expensiveAPIs) > 0 && isInteractive() {
		fmt.Printf("\nWARNING: This search is expensive for some APIs:\n")
		for _, desc := range expensiveAPIs {
			fmt.Printf("  - %s\n", desc)
		}
		fmt.Printf("\nTotal API calls: %d\n", totalCalls)
		fmt.Printf("\nContinue? [y/N]: ")

		if !confirmPrompt() {
			return fmt.Errorf("search cancelled by user")
		}
		fmt.Println() // Add newline after confirmation
	}

	return nil
}

// enrichWirelessClientFromCache fills search-result fields that vendor search
// endpoints don't always populate themselves: AP name (via Inventory.AP keyed
// by normalized MAC) and site name (via SiteIndex.ByID). Best-effort — a nil
// cache or missing entries leave the original fields untouched.
func enrichWirelessClientFromCache(c *vendors.WirelessClient, cache *vendors.APICache) {
	if c == nil || cache == nil {
		return
	}
	if c.APName == "" && c.APMAC != "" {
		if ap, ok := cache.Inventory.AP[vendors.NormalizeMAC(c.APMAC)]; ok && ap != nil {
			c.APName = ap.Name
		}
	}
	if c.SiteName == "" && c.SiteID != "" {
		if name, ok := cache.SiteIndex.ByID[c.SiteID]; ok {
			c.SiteName = name
		}
	}
}

// enrichWiredClientFromCache is the WiredClient analogue — fills Switch name
// (via Inventory.Switch by MAC) and site name (via SiteIndex.ByID).
func enrichWiredClientFromCache(c *vendors.WiredClient, cache *vendors.APICache) {
	if c == nil || cache == nil {
		return
	}
	if c.SwitchName == "" && c.SwitchMAC != "" {
		if sw, ok := cache.Inventory.Switch[vendors.NormalizeMAC(c.SwitchMAC)]; ok && sw != nil {
			c.SwitchName = sw.Name
		}
	}
	if c.SiteName == "" && c.SiteID != "" {
		if name, ok := cache.SiteIndex.ByID[c.SiteID]; ok {
			c.SiteName = name
		}
	}
}

// buildWirelessSearchColumns picks the columns for the wireless search table.
// The Site column is dropped when the user explicitly scoped to a single site —
// every row would carry the same value. The API column is added when results
// may span multiple APIs. When detail is true, Band and State columns are
// included with a `[*]` marker so operators know the data is cache-sourced
// (Band) or live-but-detail-gated (State).
func buildWirelessSearchColumns(siteFilter string, targetAPICount int, detail bool) []formatter.TableColumn {
	cols := []formatter.TableColumn{
		{Field: "mac", Title: "MAC", MaxWidth: 0},
		{Field: "ip", Title: "IP", MaxWidth: 0},
		{Field: "hostname", Title: "Hostname", MaxWidth: 0},
		{Field: "ssid", Title: "SSID", MaxWidth: 0},
		{Field: "ap_name", Title: "AP Name", MaxWidth: 0},
	}
	if detail {
		cols = append(cols,
			formatter.TableColumn{Field: "band", Title: "Band [*]", MaxWidth: 0},
			formatter.TableColumn{Field: "state", Title: "State [*]", MaxWidth: 0},
		)
	}
	if siteFilter == "" {
		cols = append(cols, formatter.TableColumn{Field: "site_name", Title: "Site", MaxWidth: 0})
	}
	if targetAPICount > 1 || apiFlag == "" {
		cols = append(cols, formatter.TableColumn{Field: "api", Title: "API", MaxWidth: 0})
	}
	return cols
}

// buildWiredSearchColumns mirrors buildWirelessSearchColumns for wired search.
// Meraki wired clients don't participate in the band cache today, but the
// `detail` flag is carried for symmetry.
func buildWiredSearchColumns(siteFilter string, targetAPICount int, _ bool) []formatter.TableColumn {
	cols := []formatter.TableColumn{
		{Field: "mac", Title: "MAC", MaxWidth: 0},
		{Field: "ip", Title: "IP", MaxWidth: 0},
		{Field: "hostname", Title: "Hostname", MaxWidth: 0},
		{Field: "switch_name", Title: "Switch", MaxWidth: 0},
		{Field: "port", Title: "Port", MaxWidth: 0},
		{Field: "vlan", Title: "VLAN", MaxWidth: 0},
	}
	if siteFilter == "" {
		cols = append(cols, formatter.TableColumn{Field: "site_name", Title: "Site", MaxWidth: 0})
	}
	if targetAPICount > 1 || apiFlag == "" {
		cols = append(cols, formatter.TableColumn{Field: "api", Title: "API", MaxWidth: 0})
	}
	return cols
}

// printDetailFooter emits the provenance footer for `search ... detail`. The
// Band column carries a `[*]` marker in its header; this footer says when the
// cache was last refreshed, or nudges the operator to run `refresh client site`
// when nothing is cached.
func printDetailFooter(cacheHits int, newest time.Time, siteFilter string) {
	fmt.Println()
	if cacheHits == 0 {
		if siteFilter == "" {
			fmt.Println("[*] no client detail cached — run `refresh client site <name>` to populate")
		} else {
			fmt.Printf("[*] no client detail cached for %s — run `refresh client site %q` to populate\n",
				siteFilter, siteFilter)
		}
		return
	}
	if siteFilter == "" {
		fmt.Printf("[*] last refreshed %s — run `refresh client site <name>` to update\n",
			newest.Format(time.RFC3339))
	} else {
		fmt.Printf("[*] last refreshed %s — run `refresh client site %q` to update\n",
			newest.Format(time.RFC3339), siteFilter)
	}
}

// resolveSearchSiteID maps a user-supplied site argument to the vendor site ID
// for the given API. If the value matches a site name in that API's cache it is
// replaced with the corresponding ID; otherwise the value is returned as-is,
// which lets the caller pass raw Mist UUIDs or Meraki L_xxx network IDs.
func resolveSearchSiteID(cacheMgr *vendors.CacheManager, apiLabel, siteArg string) string {
	if siteArg == "" || cacheMgr == nil {
		return siteArg
	}
	if id, err := cacheMgr.GetSiteIDByName(apiLabel, siteArg); err == nil && id != "" {
		return id
	}
	return siteArg
}

// isInteractive returns true if stdin is a terminal.
func isInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) // #nosec G115 -- file descriptors are small non-negative integers
}

// confirmPrompt reads user input and returns true if they confirm.
func confirmPrompt() bool {
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes"
}

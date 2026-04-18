package cmd

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/maruel/natural"
	"golang.org/x/term"

	"github.com/ravinald/wifimgr/internal/formatter"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// searchWirelessMultiVendor searches for wireless clients across multiple APIs.
// When detail or extensive is true, extra columns (Band, State) are rendered
// from the client-detail cache (Band) and live response (Status). The two
// modes differ in which rows appear:
//
//   - detail:    Online clients only. Quick health overview — what's
//                currently connected and on which band.
//   - extensive: Online and offline clients. Useful for historical or
//                troubleshooting views where disconnected devices matter.
//
// Both require the client-detail cache to be populated ahead of time via
// `wifimgr refresh client site <name>`.
func searchWirelessMultiVendor(ctx context.Context, searchText, siteID, format string, force, _, detail, extensive bool) error {
	// detail columns turn on whenever either flag is set; extensive only
	// broadens the row set.
	showDetail := detail || extensive
	includeOffline := extensive
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

	// Bookkeeping for the Band cache: how many rows got a cached Band record
	// and the newest FetchedAt across them, for the "last refreshed" footer.
	var (
		bandCacheHits     int
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
			// detail mode drops offline clients; extensive mode keeps them.
			// Default mode (neither flag) doesn't touch the row set so
			// existing behavior stays intact.
			if showDetail && !includeOffline && !isOnlineStatus(client.Status) {
				continue
			}

			enrichWirelessClientFromCache(client, apiCache)

			// Fill Band from the persistent client-detail cache whenever the
			// row still has a blank Band. Band lives in the default column
			// set now, so this enrichment runs for every search — it's a
			// lightweight in-memory lookup against a cache that's already
			// resident. Mist carries Band natively on the primary response
			// and short-circuits via the `client.Band == ""` guard.
			if cacheMgr != nil && client.Band == "" && client.MAC != "" {
				if rec, ok := cacheMgr.LookupClientDetail(apiLabel, client.MAC); ok {
					client.Band = rec.Band
					if rec.FetchedAt.After(newestDetailFetch) {
						newestDetailFetch = rec.FetchedAt
					}
					bandCacheHits++
				}
			}

			// Re-derive State from Band evidence. Meraki's native Status
			// reports recent visibility (client was seen in the last hour
			// or so), not current association. Once the client-detail
			// cache is populated, a non-empty Band is the authoritative
			// "on-air in the last 24h" signal.
			client.Status = deriveClientState(client, apiCache)

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

	sortWirelessRows(allResults)

	columns := buildWirelessSearchColumns(siteID, len(targetAPIs), showDetail)

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

	// Footer fires whenever the Band column was populated from cache on at
	// least one row. Band is a default column, so this now appears on any
	// plain `search wireless` after a refresh — no need to wait for detail.
	if bandCacheHits > 0 || showDetail {
		printBandCacheFooter(bandCacheHits, newestDetailFetch, siteID)
	}

	return nil
}

// compareMACs returns -1, 0, or 1 comparing two MAC strings in their natural
// byte order via net.ParseMAC. Unparseable input yields a nil byte slice,
// which sorts before any valid address — tolerable for our use case (vendor
// APIs consistently return parseable MACs; a corrupt one bubbling to the top
// is better than a silent panic).
func compareMACs(a, b string) int {
	ai, _ := net.ParseMAC(a)
	bi, _ := net.ParseMAC(b)
	return bytes.Compare(ai, bi)
}

// sortWirelessRows orders the wireless search table by SSID, then AP name,
// then client MAC byte-order. SSID uses natural.Less so SSIDs with trailing
// digits ("guest-1", "guest-10") order sensibly. AP name goes through the
// configurable name extractor from display.sort.ap_name — operators whose
// hostnames encode floor / building / AP number can group by those
// instead of by leading characters. Absent config, AP name falls back to
// natural.Less on the full hostname. Stable sort so rows that tie on all
// three keep their vendor response order.
func sortWirelessRows(rows []formatter.GenericTableData) {
	apExtractor := configuredAPNameExtractor()
	sort.SliceStable(rows, func(i, j int) bool {
		si, _ := rows[i]["ssid"].(string)
		sj, _ := rows[j]["ssid"].(string)
		if si != sj {
			return natural.Less(si, sj)
		}
		ai, _ := rows[i]["ap_name"].(string)
		aj, _ := rows[j]["ap_name"].(string)
		if ai != aj {
			if cmp := compareByConfiguredName(apExtractor, ai, aj); cmp != 0 {
				return cmp < 0
			}
		}
		mi, _ := rows[i]["mac"].(string)
		mj, _ := rows[j]["mac"].(string)
		return compareMACs(mi, mj) < 0
	})
}

// sortWiredRows orders the wired search table by Switch, Port, MAC.
// Switch name goes through the configurable name extractor from
// display.sort.switch_name (absent config, falls back to natural.Less).
// Port uses natural.Less so stacked formats like "1/1/2" sort before
// "1/1/10"; the old lexical sort would have put "10" before "2". MAC
// is byte-order on the parsed form.
func sortWiredRows(rows []formatter.GenericTableData) {
	swExtractor := configuredSwitchNameExtractor()
	sort.SliceStable(rows, func(i, j int) bool {
		si, _ := rows[i]["switch_name"].(string)
		sj, _ := rows[j]["switch_name"].(string)
		if si != sj {
			if cmp := compareByConfiguredName(swExtractor, si, sj); cmp != 0 {
				return cmp < 0
			}
		}
		pi, _ := rows[i]["port"].(string)
		pj, _ := rows[j]["port"].(string)
		if pi != pj {
			return natural.Less(pi, pj)
		}
		mi, _ := rows[i]["mac"].(string)
		mj, _ := rows[j]["mac"].(string)
		return compareMACs(mi, mj) < 0
	})
}

// isOnlineStatus reports whether a vendor-supplied status string represents a
// currently-connected client. Meraki returns "Online" / "Offline"; Mist
// currently returns empty on the search response so callers treating empty as
// "unknown, keep the row" is the safest behavior.
func isOnlineStatus(s string) bool {
	if s == "" {
		return true // unknown state — don't hide the row
	}
	return strings.EqualFold(s, "online")
}

// deriveClientState picks the State value for a wireless search row.
// Meraki's `status` field is the canonical source — it's what the Meraki
// dashboard and the official API examples use. We apply exactly one override:
// when Meraki claims `Online` but the per-API ClientDetail cache has been
// populated (`refresh client` / `refresh all` has run) AND we still have no
// Band evidence for this MAC, the Online claim isn't substantiated by any
// recent on-air activity, so we report Offline instead.
//
// Override conditions (all must hold):
//
//   - Cache has at least one ClientDetail record (so a "no evidence" finding
//     is meaningful — we've actually looked for this MAC, not just never
//     refreshed).
//   - Meraki says `Online`.
//   - Band is empty after all enrichment paths.
//
// In every other case — Offline per Meraki, Band present, or fresh install
// with an empty cache — the vendor's own status flows through untouched.
func deriveClientState(client *vendors.WirelessClient, cache *vendors.APICache) string {
	if client == nil {
		return ""
	}
	if hasClientDetailCache(cache) &&
		strings.EqualFold(client.Status, "online") &&
		client.Band == "" {
		return "Offline"
	}
	return client.Status
}

// hasClientDetailCache returns true if the per-API cache holds at least one
// ClientDetail record — used to gate the band-derived State rule. Safe
// against nil cache.
func hasClientDetailCache(cache *vendors.APICache) bool {
	return cache != nil && len(cache.ClientDetail) > 0
}

// searchWiredMultiVendor searches for wired clients across multiple APIs.
func searchWiredMultiVendor(ctx context.Context, searchText, siteID, format string, force, _, _, _ bool) error {
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

	sortWiredRows(allResults)

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
// Band is now a default column, backed by the local client-detail cache —
// the `[*]` marker signals "cached, may be stale" and the footer under the
// table says when the cache was last refreshed. State only shows under
// detail/extensive; it's live from the API response so no marker.
// Site drops when the user explicitly scoped to a single site (every row
// carries the same value). API is added when results may span multiple APIs.
func buildWirelessSearchColumns(siteFilter string, targetAPICount int, showDetail bool) []formatter.TableColumn {
	cols := []formatter.TableColumn{
		{Field: "mac", Title: "MAC", MaxWidth: 0},
		{Field: "ip", Title: "IP", MaxWidth: 0},
		{Field: "hostname", Title: "Hostname", MaxWidth: 0},
		{Field: "ssid", Title: "SSID", MaxWidth: 0},
		{Field: "ap_name", Title: "AP Name", MaxWidth: 0},
		{Field: "band", Title: "Band [*]", MaxWidth: 0},
	}
	if showDetail {
		cols = append(cols, formatter.TableColumn{Field: "state", Title: "State", MaxWidth: 0})
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

// printBandCacheFooter emits the provenance footer under the wireless search
// table. The Band column carries a `[*]` marker; this footer explains when
// the Band cache was last refreshed or nudges the operator to populate it.
// Fires whenever cacheHits > 0 OR the operator asked for detail/extensive
// (so they see the footer even when nothing matched).
func printBandCacheFooter(cacheHits int, newest time.Time, siteFilter string) {
	fmt.Println()
	if cacheHits == 0 {
		if siteFilter == "" {
			fmt.Println("[*] no Band data cached — run `refresh client site <name>` to populate")
		} else {
			fmt.Printf("[*] no Band data cached for %s — run `refresh client site %q` to populate\n",
				siteFilter, siteFilter)
		}
		return
	}
	if siteFilter == "" {
		fmt.Printf("[*] Band cache last refreshed %s — run `refresh client site <name>` to update\n",
			newest.Format(time.RFC3339))
	} else {
		fmt.Printf("[*] Band cache last refreshed %s — run `refresh client site %q` to update\n",
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

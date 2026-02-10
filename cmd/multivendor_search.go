package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/ravinald/wifimgr/internal/formatter"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// searchWirelessMultiVendor searches for wireless clients across multiple APIs.
func searchWirelessMultiVendor(ctx context.Context, searchText, siteID, format string, force, _ bool) error {
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

	opts := vendors.SearchOptions{SiteID: siteID}

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

		results, err := searchSvc.SearchWirelessClients(ctx, searchText, opts)
		if err != nil {
			// Log error but continue with other APIs
			fmt.Printf("WARN  Search failed for %s: %v\n", apiLabel, err)
			continue
		}

		if results == nil || len(results.Results) == 0 {
			continue
		}

		// Convert results to table data
		for _, client := range results.Results {
			vendorName, _ := registry.GetVendor(apiLabel)
			data := formatter.GenericTableData{
				"mac":       client.MAC,
				"ip":        client.IP,
				"hostname":  client.Hostname,
				"ssid":      client.SSID,
				"ap_name":   client.APName,
				"ap_mac":    client.APMAC,
				"band":      client.Band,
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

	// Define columns
	columns := []formatter.TableColumn{
		{Field: "mac", Title: "MAC", MaxWidth: 0},
		{Field: "ip", Title: "IP", MaxWidth: 0},
		{Field: "hostname", Title: "Hostname", MaxWidth: 0},
		{Field: "ssid", Title: "SSID", MaxWidth: 0},
		{Field: "ap_name", Title: "AP Name", MaxWidth: 0},
		{Field: "band", Title: "Band", MaxWidth: 0},
		{Field: "site_name", Title: "Site", MaxWidth: 0},
	}

	// Add API column when showing from multiple APIs
	if len(targetAPIs) > 1 || apiFlag == "" {
		columns = append(columns, formatter.TableColumn{Field: "api", Title: "API", MaxWidth: 0})
	}

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

// searchWiredMultiVendor searches for wired clients across multiple APIs.
func searchWiredMultiVendor(ctx context.Context, searchText, siteID, format string, force, _ bool) error {
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

	opts := vendors.SearchOptions{SiteID: siteID}

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

		results, err := searchSvc.SearchWiredClients(ctx, searchText, opts)
		if err != nil {
			// Log error but continue with other APIs
			fmt.Printf("WARN  Search failed for %s: %v\n", apiLabel, err)
			continue
		}

		if results == nil || len(results.Results) == 0 {
			continue
		}

		// Convert results to table data
		for _, wiredClient := range results.Results {
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

	// Define columns
	columns := []formatter.TableColumn{
		{Field: "mac", Title: "MAC", MaxWidth: 0},
		{Field: "ip", Title: "IP", MaxWidth: 0},
		{Field: "hostname", Title: "Hostname", MaxWidth: 0},
		{Field: "switch_name", Title: "Switch", MaxWidth: 0},
		{Field: "port", Title: "Port", MaxWidth: 0},
		{Field: "vlan", Title: "VLAN", MaxWidth: 0},
		{Field: "site_name", Title: "Site", MaxWidth: 0},
	}

	// Add API column when showing from multiple APIs
	if len(targetAPIs) > 1 || apiFlag == "" {
		columns = append(columns, formatter.TableColumn{Field: "api", Title: "API", MaxWidth: 0})
	}

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

// isInteractive returns true if stdin is a terminal.
func isInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
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

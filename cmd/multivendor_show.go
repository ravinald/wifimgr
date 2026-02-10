package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/ravinald/wifimgr/internal/cmdutils"
	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/formatter"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// showDevicesMultiVendor shows devices of a specific type from one or more APIs.
// deviceType should be "ap", "switch", or "gateway".
// Devices that are in the inventory.json file are highlighted in green.
func showDevicesMultiVendor(_ context.Context, deviceType string, parsed *cmdutils.ParsedShowArgs) error {
	// Validate target API if provided
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
		managedMACs = loadManagedMACs(inventoryPath, []string{deviceType})
		if managedMACs != nil {
			logging.Debugf("Loaded %d managed MACs for highlighting", len(managedMACs))
		}
	}

	// Collect devices from all target APIs
	var allDevices []formatter.GenericTableData
	apiCounts := make(map[string]int)

	for _, apiLabel := range targetAPIs {
		cache, err := cacheMgr.GetAPICache(apiLabel)
		if err != nil {
			// Skip APIs with no cache
			continue
		}

		// Get the appropriate inventory map based on device type
		var inventory map[string]*vendors.InventoryItem
		switch deviceType {
		case "ap":
			inventory = cache.Inventory.AP
		case "switch":
			inventory = cache.Inventory.Switch
		case "gateway":
			inventory = cache.Inventory.Gateway
		default:
			return fmt.Errorf("unknown device type: %s", deviceType)
		}

		for mac, item := range inventory {
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
					if item.Name == "" || !strings.Contains(strings.ToLower(item.Name), strings.ToLower(parsed.Filter)) {
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
				data["status"] = "offline" // Default if no status found
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

			allDevices = append(allDevices, data)
			apiCounts[apiLabel]++
		}
	}

	// Sort devices by site, name, type, mac
	formatter.SortTableData(allDevices)

	// Apply field resolution (convert field IDs to names)
	if !parsed.NoResolve {
		if err := cmdutils.ApplyFieldResolution(allDevices, true); err != nil {
			logging.Debugf("Field resolution warning: %v", err)
		}
	}

	// Build title based on device type
	typeName := getDeviceTypeName(deviceType)
	title := fmt.Sprintf("%s Devices (%d)", typeName, len(allDevices))
	if len(apiCounts) > 1 {
		title = fmt.Sprintf("%s Devices (%d from %d APIs)", typeName, len(allDevices), len(apiCounts))
	} else if apiFlag != "" {
		title = fmt.Sprintf("%s Devices from %s (%d)", typeName, apiFlag, len(allDevices))
	}

	if len(allDevices) == 0 {
		fmt.Printf("%s:\n", title)
		fmt.Printf("No %s devices found\n", strings.ToLower(typeName))
		return nil
	}

	// Create command path for config lookup
	commandPath := fmt.Sprintf("show.api.%s", deviceType)

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

	// Determine default columns - add API column when showing multiple APIs
	defaultColumns := []formatter.TableColumn{
		{Field: "name", Title: "Name", MaxWidth: 0},
		{Field: "mac", Title: "MAC", MaxWidth: 0},
		{Field: "serial", Title: "Serial", MaxWidth: 0},
		{Field: "model", Title: "Model", MaxWidth: 0},
		{Field: "status", Title: "Status", MaxWidth: 0, IsStatusField: true},
		{Field: "site_name", Title: "Site", MaxWidth: 0},
	}

	// Add API column when showing from multiple APIs
	if len(targetAPIs) > 1 || apiFlag == "" {
		defaultColumns = append(defaultColumns, formatter.TableColumn{Field: "api", Title: "API", MaxWidth: 0})
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
		Columns:       defaultColumns,
	}

	// Set format from config if not overridden by argument
	if tableConfig.Format == "" && hasCommandConfig {
		tableConfig.Format = commandFormat.Format
	}
	if tableConfig.Format == "" {
		tableConfig.Format = "table"
	}

	// Create table printer
	printer := formatter.NewGenericTablePrinter(tableConfig, allDevices)

	// Use config-driven columns if available, otherwise use defaults
	if hasCommandConfig && commandFormat.Fields != nil {
		printer.LoadColumnsFromConfig(commandFormat.Fields)
	} else {
		printer.Config.Columns = defaultColumns
	}

	fmt.Print(printer.Print())

	// Show cache timestamp
	printCacheTimestamp(cacheMgr, targetAPIs, tableConfig.Format)

	return nil
}

// showSitesMultiVendor shows sites from one or more APIs.
func showSitesMultiVendor(_ context.Context, parsed *cmdutils.ParsedShowArgs) error {
	// Validate target API if provided
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

	// If a specific site name is provided (not a partial search), check for exact match
	// and show detailed cross-vendor view
	if parsed.Filter != "" {
		exactMatches := findExactSiteMatches(cacheMgr, targetAPIs, parsed.Filter)
		if len(exactMatches) > 0 {
			return showSiteDetailMultiVendor(exactMatches, parsed)
		}
	}

	// Collect sites from all target APIs (list view)
	var allSites []formatter.GenericTableData
	apiCounts := make(map[string]int)

	for _, apiLabel := range targetAPIs {
		cache, err := cacheMgr.GetAPICache(apiLabel)
		if err != nil {
			// Skip APIs with no cache
			continue
		}

		for _, site := range cache.Sites.Info {
			// Apply name filter if specified (substring match for list view)
			if parsed.Filter != "" {
				if !strings.Contains(strings.ToLower(site.Name), strings.ToLower(parsed.Filter)) {
					continue
				}
			}

			// Count devices for this site
			apCount := 0
			switchCount := 0
			gwCount := 0

			for _, item := range cache.Inventory.AP {
				if item.SiteID == site.ID {
					apCount++
				}
			}
			for _, item := range cache.Inventory.Switch {
				if item.SiteID == site.ID {
					switchCount++
				}
			}
			for _, item := range cache.Inventory.Gateway {
				if item.SiteID == site.ID {
					gwCount++
				}
			}

			// Convert to table data
			data := formatter.GenericTableData{
				"name":         site.Name,
				"id":           site.ID,
				"timezone":     site.Timezone,
				"country_code": site.CountryCode,
				"ap_count":     apCount,
				"switch_count": switchCount,
				"gw_count":     gwCount,
				"total":        apCount + switchCount + gwCount,
				"vendor":       cache.Meta.Vendor,
				"api":          apiLabel,
			}

			allSites = append(allSites, data)
			apiCounts[apiLabel]++
		}
	}

	// Sort sites by name
	formatter.SortTableData(allSites)

	// Apply field resolution (convert field IDs to names)
	if !parsed.NoResolve {
		if err := cmdutils.ApplyFieldResolution(allSites, true); err != nil {
			logging.Debugf("Field resolution warning: %v", err)
		}
	}

	// Build title
	title := fmt.Sprintf("Sites (%d)", len(allSites))
	if len(apiCounts) > 1 {
		title = fmt.Sprintf("Sites (%d from %d APIs)", len(allSites), len(apiCounts))
	} else if apiFlag != "" {
		title = fmt.Sprintf("Sites from %s (%d)", apiFlag, len(allSites))
	}

	if len(allSites) == 0 {
		fmt.Printf("%s:\n", title)
		fmt.Println("No sites found")
		return nil
	}

	// Determine columns - add API column when showing multiple APIs
	columns := []formatter.TableColumn{
		{Field: "name", Title: "Name", MaxWidth: 0},
		{Field: "timezone", Title: "Timezone", MaxWidth: 0},
		{Field: "ap_count", Title: "APs", MaxWidth: 0},
		{Field: "switch_count", Title: "Switches", MaxWidth: 0},
		{Field: "gw_count", Title: "Gateways", MaxWidth: 0},
	}

	// Add vendor and API columns when showing from multiple APIs
	if len(targetAPIs) > 1 || apiFlag == "" {
		columns = append(columns,
			formatter.TableColumn{Field: "vendor", Title: "Vendor", MaxWidth: 0},
			formatter.TableColumn{Field: "api", Title: "API", MaxWidth: 0},
		)
	}

	// Create command path for config lookup
	commandPath := "show.api.sites"

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
	printer := formatter.NewGenericTablePrinter(tableConfig, allSites)

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

// siteMatch represents a site found in an API with its device counts
type siteMatch struct {
	apiLabel    string
	vendor      string
	site        *vendors.SiteInfo
	apCount     int
	switchCount int
	gwCount     int
}

// findExactSiteMatches finds sites with an exact name match across all target APIs
func findExactSiteMatches(cacheMgr *vendors.CacheManager, targetAPIs []string, siteName string) []siteMatch {
	var matches []siteMatch

	for _, apiLabel := range targetAPIs {
		cache, err := cacheMgr.GetAPICache(apiLabel)
		if err != nil {
			continue
		}

		for i := range cache.Sites.Info {
			site := &cache.Sites.Info[i]
			if strings.EqualFold(site.Name, siteName) {
				// Count devices for this site
				apCount := 0
				switchCount := 0
				gwCount := 0

				for _, item := range cache.Inventory.AP {
					if item.SiteID == site.ID {
						apCount++
					}
				}
				for _, item := range cache.Inventory.Switch {
					if item.SiteID == site.ID {
						switchCount++
					}
				}
				for _, item := range cache.Inventory.Gateway {
					if item.SiteID == site.ID {
						gwCount++
					}
				}

				matches = append(matches, siteMatch{
					apiLabel:    apiLabel,
					vendor:      cache.Meta.Vendor,
					site:        site,
					apCount:     apCount,
					switchCount: switchCount,
					gwCount:     gwCount,
				})
			}
		}
	}

	return matches
}

// showSiteDetailMultiVendor shows detailed information for a site across vendors
func showSiteDetailMultiVendor(matches []siteMatch, parsed *cmdutils.ParsedShowArgs) error {
	siteName := matches[0].site.Name

	// Handle JSON/CSV format
	if parsed.Format == "json" || parsed.Format == "csv" {
		var tableData []formatter.GenericTableData
		for _, m := range matches {
			data := formatter.GenericTableData{
				"name":         m.site.Name,
				"id":           m.site.ID,
				"timezone":     m.site.Timezone,
				"country_code": m.site.CountryCode,
				"address":      m.site.Address,
				"latitude":     m.site.Latitude,
				"longitude":    m.site.Longitude,
				"notes":        m.site.Notes,
				"ap_count":     m.apCount,
				"switch_count": m.switchCount,
				"gw_count":     m.gwCount,
				"total":        m.apCount + m.switchCount + m.gwCount,
				"vendor":       m.vendor,
				"api":          m.apiLabel,
			}
			tableData = append(tableData, data)
		}

		columns := []formatter.TableColumn{
			{Field: "name", Title: "Name"},
			{Field: "id", Title: "ID"},
			{Field: "timezone", Title: "Timezone"},
			{Field: "country_code", Title: "Country"},
			{Field: "address", Title: "Address"},
			{Field: "latitude", Title: "Latitude"},
			{Field: "longitude", Title: "Longitude"},
			{Field: "notes", Title: "Notes"},
			{Field: "ap_count", Title: "APs"},
			{Field: "switch_count", Title: "Switches"},
			{Field: "gw_count", Title: "Gateways"},
			{Field: "total", Title: "Total"},
			{Field: "vendor", Title: "Vendor"},
			{Field: "api", Title: "API"},
		}

		tableConfig := formatter.TableConfig{
			Title:   fmt.Sprintf("Site: %s", siteName),
			Format:  parsed.Format,
			Columns: columns,
		}

		printer := formatter.NewGenericTablePrinter(tableConfig, tableData)
		fmt.Print(printer.Print())
		return nil
	}

	// Table format - show detailed view
	if len(matches) > 1 {
		fmt.Printf("Site \"%s\" found in %d APIs:\n\n", siteName, len(matches))
	} else {
		fmt.Printf("Site: %s\n\n", siteName)
	}

	for i, m := range matches {
		if len(matches) > 1 {
			fmt.Printf("─── %s (%s) ───\n", m.apiLabel, m.vendor)
		}

		// Site details
		fmt.Printf("  ID:           %s\n", m.site.ID)
		fmt.Printf("  Vendor:       %s\n", m.vendor)
		fmt.Printf("  API:          %s\n", m.apiLabel)
		if m.site.Timezone != "" {
			fmt.Printf("  Timezone:     %s\n", m.site.Timezone)
		}
		if m.site.CountryCode != "" {
			fmt.Printf("  Country:      %s\n", m.site.CountryCode)
		}
		if m.site.Address != "" {
			fmt.Printf("  Address:      %s\n", m.site.Address)
		}
		if m.site.Latitude != 0 || m.site.Longitude != 0 {
			fmt.Printf("  Location:     %.6f, %.6f\n", m.site.Latitude, m.site.Longitude)
		}
		if m.site.Notes != "" {
			fmt.Printf("  Notes:        %s\n", m.site.Notes)
		}

		// Device counts
		total := m.apCount + m.switchCount + m.gwCount
		fmt.Printf("\n  Devices:      %d total\n", total)
		fmt.Printf("    APs:        %d\n", m.apCount)
		fmt.Printf("    Switches:   %d\n", m.switchCount)
		fmt.Printf("    Gateways:   %d\n", m.gwCount)

		if i < len(matches)-1 {
			fmt.Println()
		}
	}

	// Show hint for duplicate sites
	if len(matches) > 1 {
		fmt.Println()
		fmt.Println("Tip: Use 'target <label>' to target a specific API when performing operations")
	}

	// Show cache timestamp
	cacheMgr := GetCacheManager()
	var apiLabels []string
	for _, m := range matches {
		apiLabels = append(apiLabels, m.apiLabel)
	}
	printCacheTimestamp(cacheMgr, apiLabels, parsed.Format)

	return nil
}

// showInventoryMultiVendor shows inventory items from one or more APIs.
// deviceType can be "ap", "switch", "gateway", or "" for all.
// If an inventory.json file is configured, only devices listed there are shown.
func showInventoryMultiVendor(_ context.Context, deviceType string, parsed *cmdutils.ParsedShowArgs) error {
	// Validate target API if provided
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

	// Determine which device types to show
	deviceTypes := []string{"ap", "switch", "gateway"}
	if deviceType != "" {
		deviceTypes = []string{deviceType}
	}

	// Check if an inventory file is configured - this filters to only managed devices
	inventoryPath := viper.GetString("files.inventory")
	var managedMACs map[string]string // normalized MAC -> device type
	if inventoryPath != "" {
		managedMACs = loadManagedMACs(inventoryPath, deviceTypes)
		if managedMACs == nil {
			// Error loading inventory file - already logged
			return fmt.Errorf("failed to load inventory file: %s", inventoryPath)
		}
		logging.Debugf("Loaded %d managed MACs from inventory file", len(managedMACs))
	}

	// Load device intents from local site configs to enrich inventory display and detect drift
	deviceIntents := loadDeviceIntentsFromSiteConfigs()

	// Collect inventory items from all target APIs
	var allItems []formatter.GenericTableData
	apiCounts := make(map[string]int)
	hasDrift := false // Track if any device has configuration drift

	for _, apiLabel := range targetAPIs {
		cache, err := cacheMgr.GetAPICache(apiLabel)
		if err != nil {
			// Skip APIs with no cache
			continue
		}

		for _, dt := range deviceTypes {
			var inventory map[string]*vendors.InventoryItem
			switch dt {
			case "ap":
				inventory = cache.Inventory.AP
			case "switch":
				inventory = cache.Inventory.Switch
			case "gateway":
				inventory = cache.Inventory.Gateway
			}

			for mac, item := range inventory {
				normalizedMAC := vendors.NormalizeMAC(mac)

				// If inventory file is configured, only show managed devices
				if managedMACs != nil {
					if _, isManaged := managedMACs[normalizedMAC]; !isManaged {
						continue
					}
				}

				// Apply site filter if specified
				if parsed.SiteName != "" {
					siteID, ok := cache.SiteIndex.ByName[parsed.SiteName]
					if !ok || item.SiteID != siteID {
						continue
					}
				}

				// Determine device name - prefer API cache, fallback to site config intent
				displayName := item.Name
				var hasIntent bool
				var intent deviceIntent
				if deviceIntents != nil {
					intent, hasIntent = deviceIntents[normalizedMAC]
					if displayName == "" && hasIntent && intent.Name != "" {
						displayName = intent.Name
					}
				}

				// Check for configuration drift (intent differs from cache)
				driftMarker := ""
				if hasIntent && hasConfigDrift(cache, normalizedMAC, dt, intent) {
					driftMarker = "* "
					hasDrift = true
				}

				// Apply name/MAC filter if specified
				if parsed.Filter != "" {
					if cmdutils.IsMAC(parsed.Filter) {
						if normalizedMAC != vendors.NormalizeMAC(parsed.Filter) {
							continue
						}
					} else {
						// Filter by name - use enriched displayName which includes site config names
						if displayName == "" || !strings.Contains(strings.ToLower(displayName), strings.ToLower(parsed.Filter)) {
							continue
						}
					}
				}

				// Convert to table data - prepend drift marker to name if needed
				data := formatter.GenericTableData{
					"name":    driftMarker + displayName,
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
					data["status"] = "offline" // Default if no status found
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

				allItems = append(allItems, data)
				apiCounts[apiLabel]++
			}
		}
	}

	// Sort inventory items by site, name, type, mac
	formatter.SortTableData(allItems)

	// Apply field resolution (convert field IDs to names)
	if !parsed.NoResolve {
		if err := cmdutils.ApplyFieldResolution(allItems, true); err != nil {
			logging.Debugf("Field resolution warning: %v", err)
		}
	}

	// Build title
	var title string
	if deviceType == "" {
		title = fmt.Sprintf("Inventory (%d)", len(allItems))
		if len(apiCounts) > 1 {
			title = fmt.Sprintf("Inventory (%d from %d APIs)", len(allItems), len(apiCounts))
		} else if apiFlag != "" {
			title = fmt.Sprintf("Inventory from %s (%d)", apiFlag, len(allItems))
		}
	} else {
		typeName := getDeviceTypeName(deviceType)
		title = fmt.Sprintf("%s Inventory (%d)", typeName, len(allItems))
		if len(apiCounts) > 1 {
			title = fmt.Sprintf("%s Inventory (%d from %d APIs)", typeName, len(allItems), len(apiCounts))
		} else if apiFlag != "" {
			title = fmt.Sprintf("%s Inventory from %s (%d)", typeName, apiFlag, len(allItems))
		}
	}

	if len(allItems) == 0 {
		fmt.Printf("%s:\n", title)
		fmt.Println("No inventory items found")
		return nil
	}

	// Determine columns - add Type column when showing all types
	columns := []formatter.TableColumn{
		{Field: "name", Title: "Name", MaxWidth: 0},
		{Field: "mac", Title: "MAC", MaxWidth: 0},
		{Field: "serial", Title: "Serial", MaxWidth: 0},
		{Field: "model", Title: "Model", MaxWidth: 0},
	}

	// Add Type column when showing all device types
	if deviceType == "" {
		columns = append(columns, formatter.TableColumn{Field: "type", Title: "Type", MaxWidth: 0})
	}

	columns = append(columns,
		formatter.TableColumn{Field: "status", Title: "Status", MaxWidth: 0, IsStatusField: true},
		formatter.TableColumn{Field: "site_name", Title: "Site", MaxWidth: 0},
	)

	// Add API column when showing from multiple APIs
	if len(targetAPIs) > 1 || apiFlag == "" {
		columns = append(columns, formatter.TableColumn{Field: "api", Title: "API", MaxWidth: 0})
	}

	// Create command path for config lookup
	commandPath := "show.inventory"
	if deviceType != "" {
		commandPath = fmt.Sprintf("show.inventory.%s", deviceType)
	}

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
	printer := formatter.NewGenericTablePrinter(tableConfig, allItems)

	// Use config-driven columns if available, otherwise use defaults
	if hasCommandConfig && commandFormat.Fields != nil {
		printer.LoadColumnsFromConfig(commandFormat.Fields)
	} else {
		printer.Config.Columns = columns
	}

	fmt.Print(printer.Print())

	// Show drift note if any devices have configuration drift (only for table format)
	if hasDrift && tableConfig.Format == "table" {
		fmt.Println()
		fmt.Println("* Device has configuration drift from intent")
	}

	// Show cache timestamp
	printCacheTimestamp(cacheMgr, targetAPIs, tableConfig.Format)

	return nil
}

// getDeviceTypeName returns a human-readable name for the device type.
func getDeviceTypeName(deviceType string) string {
	switch deviceType {
	case "ap":
		return "AP"
	case "switch":
		return "Switch"
	case "gateway":
		return "Gateway"
	default:
		// Capitalize first letter manually to avoid deprecated strings.Title
		if len(deviceType) == 0 {
			return deviceType
		}
		return strings.ToUpper(deviceType[:1]) + deviceType[1:]
	}
}

// loadManagedMACs loads MAC addresses from the inventory.json file.
// Returns a map of normalized MAC addresses to device types for the requested device types.
// Returns nil if the file cannot be loaded.
func loadManagedMACs(inventoryPath string, deviceTypes []string) map[string]string {
	// Read the inventory file
	data, err := os.ReadFile(inventoryPath)
	if err != nil {
		logging.Errorf("Failed to read inventory file %s: %v", inventoryPath, err)
		return nil
	}

	// Parse the inventory file structure
	var inventory struct {
		Config struct {
			Inventory struct {
				AP      []string `json:"ap"`
				Switch  []string `json:"switch"`
				Gateway []string `json:"gateway"`
			} `json:"inventory"`
		} `json:"config"`
	}

	if err := json.Unmarshal(data, &inventory); err != nil {
		logging.Errorf("Failed to parse inventory file %s: %v", inventoryPath, err)
		return nil
	}

	// Build map of managed MACs
	managedMACs := make(map[string]string)

	// Check which device types we need
	needAP := false
	needSwitch := false
	needGateway := false
	for _, dt := range deviceTypes {
		switch dt {
		case "ap":
			needAP = true
		case "switch":
			needSwitch = true
		case "gateway":
			needGateway = true
		}
	}

	if needAP {
		for _, mac := range inventory.Config.Inventory.AP {
			managedMACs[vendors.NormalizeMAC(mac)] = "ap"
		}
	}
	if needSwitch {
		for _, mac := range inventory.Config.Inventory.Switch {
			managedMACs[vendors.NormalizeMAC(mac)] = "switch"
		}
	}
	if needGateway {
		for _, mac := range inventory.Config.Inventory.Gateway {
			managedMACs[vendors.NormalizeMAC(mac)] = "gateway"
		}
	}

	return managedMACs
}

// printCacheTimestamp prints the cache refresh timestamp for the displayed APIs.
// It shows how long ago the cache was refreshed in a human-readable format.
func printCacheTimestamp(cacheMgr *vendors.CacheManager, targetAPIs []string, format string) {
	// Skip for non-table formats (JSON/CSV are meant for machine consumption)
	if format == "json" || format == "csv" {
		return
	}

	if cacheMgr == nil || len(targetAPIs) == 0 {
		return
	}

	// Collect refresh times from all displayed APIs
	type apiTime struct {
		label string
		time  time.Time
	}
	var times []apiTime

	for _, apiLabel := range targetAPIs {
		cache, err := cacheMgr.GetAPICache(apiLabel)
		if err != nil {
			continue
		}
		if !cache.Meta.LastRefresh.IsZero() {
			times = append(times, apiTime{label: apiLabel, time: cache.Meta.LastRefresh})
		}
	}

	if len(times) == 0 {
		return
	}

	// Format the output
	fmt.Println()
	if len(times) == 1 {
		fmt.Printf("Cache refreshed: %s (%s ago)\n",
			times[0].time.Format("2006-01-02 15:04:05"),
			formatDuration(time.Since(times[0].time)))
	} else {
		fmt.Println("Cache refreshed:")
		for _, t := range times {
			fmt.Printf("  %s: %s (%s ago)\n",
				t.label,
				t.time.Format("2006-01-02 15:04:05"),
				formatDuration(time.Since(t.time)))
		}
	}
}

// formatDuration formats a duration in a human-readable way.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 minute"
		}
		return fmt.Sprintf("%d minutes", mins)
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	}
	days := int(d.Hours() / 24)
	if days == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", days)
}

// deviceIntent holds the intended configuration for a device from local site configs.
type deviceIntent struct {
	Name string
}

// loadDeviceIntentsFromSiteConfigs loads device intents from local site configuration files.
// Returns a map of normalized MAC address to device intent for each device type.
func loadDeviceIntentsFromSiteConfigs() map[string]deviceIntent {
	siteConfigFiles := viper.GetStringSlice("files.site_configs")
	configDir := viper.GetString("files.config_dir")

	if len(siteConfigFiles) == 0 {
		return nil
	}

	intents := make(map[string]deviceIntent)

	for _, siteConfigFile := range siteConfigFiles {
		siteConfig, err := config.LoadSiteConfig(configDir, siteConfigFile)
		if err != nil {
			logging.Debugf("Failed to load site config %s: %v", siteConfigFile, err)
			continue
		}

		// Process each site in the config
		for _, siteObj := range siteConfig.Config.Sites {
			// Process APs
			for mac, ap := range siteObj.Devices.APs {
				normalizedMAC := vendors.NormalizeMAC(mac)
				intents[normalizedMAC] = deviceIntent{Name: ap.Name}
			}
			// Process Switches
			for mac, sw := range siteObj.Devices.Switches {
				normalizedMAC := vendors.NormalizeMAC(mac)
				intents[normalizedMAC] = deviceIntent{Name: sw.Name}
			}
			// Process Gateways
			for mac, gw := range siteObj.Devices.WanEdge {
				normalizedMAC := vendors.NormalizeMAC(mac)
				intents[normalizedMAC] = deviceIntent{Name: gw.Name}
			}
		}
	}

	if len(intents) > 0 {
		logging.Debugf("Loaded %d device intents from site configs", len(intents))
	}

	return intents
}

// hasConfigDrift checks if a device has configuration drift between cache and intent.
// Currently checks the name field; can be extended to check other fields.
func hasConfigDrift(cache *vendors.APICache, normalizedMAC, deviceType string, intent deviceIntent) bool {
	// Get the device config from cache
	var cacheName string
	switch deviceType {
	case "ap":
		if cfg, ok := cache.Configs.AP[normalizedMAC]; ok && cfg != nil {
			cacheName = cfg.Name
		}
	case "switch":
		if cfg, ok := cache.Configs.Switch[normalizedMAC]; ok && cfg != nil {
			cacheName = cfg.Name
		}
	case "gateway":
		if cfg, ok := cache.Configs.Gateway[normalizedMAC]; ok && cfg != nil {
			cacheName = cfg.Name
		}
	}

	// Compare names - drift if they differ and intent has a name
	if intent.Name != "" && cacheName != intent.Name {
		return true
	}

	return false
}

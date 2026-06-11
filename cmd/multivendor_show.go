package cmd

import (
	"context"
	"errors"
	"fmt"
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

	// Managed-default: show only the armed devices unless `all` widens the
	// scope. Drift markers flag devices whose local intent differs from the
	// cached config — folded in from the former `show inventory` view.
	managed, err := loadManagedMACSet([]string{deviceType})
	if err != nil {
		return err
	}
	deviceIntents := loadDeviceIntentsFromSiteConfigs()
	hasDrift := false

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

			// Object scope: managed devices only, unless `all` was given.
			isManaged := managed[normalizedMAC]
			if !parsed.ShowUnmanaged && !isManaged {
				continue
			}

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

			// Drift marker when local intent differs from the cached config.
			displayName := item.Name
			if intent, ok := deviceIntents[normalizedMAC]; ok {
				if displayName == "" && intent.Name != "" {
					displayName = intent.Name
				}
				if hasConfigDrift(cache, normalizedMAC, deviceType, intent) {
					displayName = "* " + displayName
					hasDrift = true
				}
			}
			// In the widened (`all`) view, green-highlight the managed ones so
			// they stand out among the unmanaged. The managed-default view is
			// already all-managed, so highlighting there would be noise.
			if parsed.ShowUnmanaged && isManaged && item.Name != "" {
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

	// Build title based on device type. "Managed" in the default view; the
	// widened `all` view drops the qualifier.
	typeName := getDeviceTypeName(deviceType)
	scopeWord := "Managed "
	if parsed.ShowUnmanaged {
		scopeWord = ""
	}
	title := fmt.Sprintf("%s%s Devices (%d)", scopeWord, typeName, len(allDevices))
	if len(apiCounts) > 1 {
		title = fmt.Sprintf("%s%s Devices (%d from %d APIs)", scopeWord, typeName, len(allDevices), len(apiCounts))
	} else if apiFlag != "" {
		title = fmt.Sprintf("%s%s Devices from %s (%d)", scopeWord, typeName, apiFlag, len(allDevices))
	}

	if len(allDevices) == 0 {
		fmt.Printf("%s:\n", title)
		if !parsed.ShowUnmanaged {
			fmt.Printf("No managed %s devices. Arm devices in inventory.json or add 'all' to see everything.\n", strings.ToLower(typeName))
		} else {
			fmt.Printf("No %s devices found\n", strings.ToLower(typeName))
		}
		return nil
	}

	// Create command path for config lookup
	commandPath := fmt.Sprintf("show.%s", deviceType)

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
		ShowAllFields: parsed.AllFields(),
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

	// Drift note (table only): mirrors the marker prepended to drifted names.
	if hasDrift && tableConfig.Format == "table" {
		fmt.Println()
		fmt.Println("* Device has configuration drift from intent")
	}

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

	// Managed-default: list only armed sites unless `all` widens the scope.
	managedSites, err := loadManagedSiteSet()
	if err != nil {
		return err
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
			// Object scope: armed sites only, unless `all` was given.
			if !parsed.ShowUnmanaged && !managedSites[strings.ToLower(site.Name)] {
				continue
			}

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

	// Build title. "Managed" in the default view; `all` drops the qualifier.
	scopeWord := "Managed "
	if parsed.ShowUnmanaged {
		scopeWord = ""
	}
	title := fmt.Sprintf("%sSites (%d)", scopeWord, len(allSites))
	if len(apiCounts) > 1 {
		title = fmt.Sprintf("%sSites (%d from %d APIs)", scopeWord, len(allSites), len(apiCounts))
	} else if apiFlag != "" {
		title = fmt.Sprintf("%sSites from %s (%d)", scopeWord, apiFlag, len(allSites))
	}

	if len(allSites) == 0 {
		fmt.Printf("%s:\n", title)
		if !parsed.ShowUnmanaged {
			fmt.Println("No managed sites. Arm devices in inventory.json or add 'all' to see everything.")
		} else {
			fmt.Println("No sites found")
		}
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
	commandPath := "show.sites"

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
		ShowAllFields: parsed.AllFields(),
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

// loadManagedMACSet returns the normalized armed MACs for the given device
// types, unioned across every armed site. A legacy-schema inventory is fatal
// (the operator must migrate); a missing/unreadable file yields an empty set —
// managed-first means nothing is armed until the operator says so.
func loadManagedMACSet(deviceTypes []string) (map[string]bool, error) {
	inv, err := config.LoadInventoryFile(config.InventoryPath(nil))
	if err != nil {
		if errors.Is(err, config.ErrLegacyInventorySchema) {
			return nil, err
		}
		logging.Debugf("inventory unavailable: %v", err)
		return map[string]bool{}, nil
	}
	set := make(map[string]bool)
	for _, dt := range deviceTypes {
		for mac := range inv.NormalizedSet(nil, dt) {
			set[mac] = true
		}
	}
	return set, nil
}

// loadManagedSiteSet returns the armed site names (lowercased) from the
// inventory file — the sites that contain at least one armed device. Same
// fatal/empty semantics as loadManagedMACSet.
func loadManagedSiteSet() (map[string]bool, error) {
	inv, err := config.LoadInventoryFile(config.InventoryPath(nil))
	if err != nil {
		if errors.Is(err, config.ErrLegacyInventorySchema) {
			return nil, err
		}
		logging.Debugf("inventory unavailable: %v", err)
		return map[string]bool{}, nil
	}
	set := make(map[string]bool)
	for _, name := range inv.SiteNames() {
		set[strings.ToLower(name)] = true
	}
	return set, nil
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

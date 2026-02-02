package site

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"

	"github.com/ravinald/wifimgr/api"
	"github.com/ravinald/wifimgr/internal/cmdutils"
	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/formatter"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/patterns"
	"github.com/ravinald/wifimgr/internal/utils"
)

// HandleCommand processes site-related subcommands
func HandleCommand(ctx context.Context, client api.Client, args []string, formatOverride string) error {
	if len(args) < 1 {
		logging.Error("No site subcommand specified")
		return fmt.Errorf("no site subcommand specified")
	}

	logging.Infof("Executing site subcommand: %s", args[0])

	switch args[0] {
	case "list":
		// If no additional arguments, list all sites
		if len(args) == 1 {
			return ListSites(ctx, client, "", formatOverride)
		}
		// If an argument is provided, get details for that specific site
		return GetSite(ctx, client, args[1])
	case "intent_list":
		return ListIntentSites(ctx, client, formatOverride)
	case "get":
		if len(args) < 2 {
			logging.Error("Site ID or name required for site get command")
			return fmt.Errorf("site ID or name required")
		}
		return GetSite(ctx, client, args[1])
	case "create":
		return CreateSite(ctx, client)
	case "update":
		if len(args) < 2 {
			logging.Error("Site ID or name required for site update command")
			return fmt.Errorf("site ID or name required")
		}
		return UpdateSite(ctx, client, args[1])
	case "delete":
		if len(args) < 2 {
			logging.Error("Site ID or name required for site delete command")
			return fmt.Errorf("site ID or name required")
		}
		// Check for force flag
		force := false
		if len(args) > 2 && args[2] == "--force" {
			force = true
		}
		return DeleteSite(ctx, client, args[1], force)
	default:
		logging.Errorf("Unknown site subcommand: %s", args[0])
		return fmt.Errorf("unknown site subcommand: %s", args[0])
	}
}

// ListIntentSites lists sites from local configuration files
func ListIntentSites(ctx context.Context, client api.Client, formatOverride string) error {
	logging.Infof("Listing sites from local configuration files")

	// Check if site config files are specified
	siteConfigFiles := viper.GetStringSlice("files.site_configs")
	if len(siteConfigFiles) == 0 {
		logging.Warn("No site configuration files specified, falling back to API")
		return ListSites(ctx, client, "")
	}

	// Load site configurations using paths from the main config
	var siteConfigs []*config.SiteConfigFile
	configDir := viper.GetString("files.config_dir")
	for _, siteConfigFile := range siteConfigFiles {
		siteConfig, err := config.LoadSiteConfig(configDir, siteConfigFile)
		if err != nil {
			logging.Errorf("Failed to load site configuration from %s: %v", siteConfigFile, err)
			return fmt.Errorf("failed to load site configuration from %s: %w", siteConfigFile, err)
		}
		siteConfigs = append(siteConfigs, siteConfig)
	}

	if len(siteConfigs) == 0 {
		logging.Warn("No site configurations found in config files, falling back to API")
		return ListSites(ctx, client, "")
	}

	// Convert site configs to API site objects
	var sites []api.Site
	siteInAPIMap := make(map[string]bool) // Track which sites are in API
	for i, siteCfg := range siteConfigs {
		for siteName, siteObj := range siteCfg.Config.Sites {
			siteConfig := siteObj.SiteConfig
			if siteConfig.Name == "" {
				continue
			}

			// Create a site object from the config
			site := api.Site{}
			name := siteConfig.Name
			site.Name = &name

			// Check if the site exists in the local cache or API
			var siteID api.UUID
			var siteIDFound bool
			var inAPI = false

			// Try to get the site ID from cache
			if cacheAccessor, err := cmdutils.GetCacheAccessor(); err == nil {
				if site, err := cacheAccessor.GetSiteByName(name); err == nil && site != nil && site.ID != "" {
					siteID = api.UUID(site.ID)
					siteIDFound = true
					inAPI = true // Site found in cache means it exists in API
					logging.Debugf("Using site ID from cache for site %s: %s", name, string(siteID))
				} else {
					logging.Debugf("Site %s not found in cache", name)
				}
			} else {
				logging.Debugf("Cache not available: %v", err)
			}

			// If no site ID in the local cache, check the configuration file for a site_id field
			if !siteIDFound {
				siteConfigFiles := viper.GetStringSlice("files.site_configs")
				if i < len(siteConfigFiles) {
					siteConfigFile := siteConfigFiles[i]
					configDir := viper.GetString("files.config_dir")
					fileBytes, err := os.ReadFile(filepath.Join(configDir, siteConfigFile))
					if err == nil {
						// Parse the file to get access to raw JSON
						var rawData map[string]interface{}
						if err := json.Unmarshal(fileBytes, &rawData); err == nil {
							// Check for config wrapper
							if configData, ok := rawData["config"].(map[string]interface{}); ok {
								// Look for the site key
								if siteData, ok := configData[siteName].(map[string]interface{}); ok {
									// Check for site_id field
									if siteIDVal, ok := siteData["site_id"].(string); ok && siteIDVal != "" {
										siteID = api.UUID(siteIDVal)
										siteIDFound = true
										inAPI = true // If we have a site_id, it likely exists in API
										logging.Debugf("Using site_id from config for site %s: %s", name, string(siteID))
									}
								}
							}
						}
					}
				}
			}

			// If no site ID found in local cache or config, try to get it from the API
			if !siteIDFound {
				orgID := viper.GetString("api.credentials.org_id")
				if siteFromAPI, err := client.GetSiteByName(ctx, name, orgID); err == nil && siteFromAPI.ID != nil {
					// Use the real site ID from API
					siteID = api.UUID(*siteFromAPI.ID)
					inAPI = true
					logging.Debugf("Using real site ID from API for site %s: %s", name, string(siteID))
				} else {
					// If not in API, use the placeholder
					siteID = api.UUID("(known after applied)")
					inAPI = false
					logging.Debugf("Site %s not found in API, using placeholder", name)
				}
			}
			site.Id = &siteID

			// Store the inAPI status in the map
			siteInAPIMap[name] = inAPI

			// Set optional fields if provided
			if siteConfig.Address != "" {
				site.Address = &siteConfig.Address
			}
			if siteConfig.CountryCode != "" {
				site.CountryCode = &siteConfig.CountryCode
			}
			if siteConfig.Timezone != "" {
				site.Timezone = &siteConfig.Timezone
			}
			if siteConfig.Notes != "" {
				site.Notes = &siteConfig.Notes
			}
			if siteConfig.LatLng != nil {
				site.Latlng = siteConfig.LatLng
			}

			sites = append(sites, site)
		}
	}

	// Sort sites alphabetically by name
	sortedSites := api.SortSites(sites)
	logging.Debug("Sites sorted successfully")

	// Create title for the table
	title := fmt.Sprintf("Intent Sites (Local Configuration) (%d)", len(sortedSites))

	// Convert sites to map for the table formatter
	tableData := make([]formatter.GenericTableData, 0, len(sortedSites))
	for _, site := range sortedSites {
		siteData := formatter.GenericTableData{}

		// Name - Store the raw value and mark for green text
		if site.Name != nil {
			siteData["name"] = "GREEN_TEXT:" + *site.Name
		} else {
			siteData["name"] = "GREEN_TEXT:<undefined>"
		}

		// ID - Store the raw value and mark for green text
		if site.Id != nil {
			siteData["id"] = "GREEN_TEXT:" + string(*site.Id)
		} else {
			siteData["id"] = "GREEN_TEXT:"
		}

		// Address - Store the raw value and mark for green text
		if site.Address != nil {
			siteData["address"] = "GREEN_TEXT:" + *site.Address
		} else {
			siteData["address"] = "GREEN_TEXT:"
		}

		// Country Code - Store the raw value and mark for green text
		if site.CountryCode != nil {
			siteData["country_code"] = "GREEN_TEXT:" + *site.CountryCode
		} else {
			siteData["country_code"] = "GREEN_TEXT:"
		}

		// Timezone - Store the raw value and mark for green text
		if site.Timezone != nil {
			siteData["timezone"] = "GREEN_TEXT:" + *site.Timezone
		} else {
			siteData["timezone"] = "GREEN_TEXT:"
		}

		// In API status
		siteName := ""
		if site.Name != nil {
			siteName = *site.Name
		}
		if inAPI, exists := siteInAPIMap[siteName]; exists {
			siteData["in_api"] = inAPI
		} else {
			siteData["in_api"] = false
		}

		tableData = append(tableData, siteData)
	}

	// Create command path for config lookup
	commandPath := "show.intent.site"

	// Check if there's a command-specific format in the config
	displayCommands := viper.GetStringMap("display.commands")
	commandFormatRaw, hasCommandConfig := displayCommands[commandPath]

	var commandFormat map[string]interface{}
	if hasCommandConfig {
		if cfgMap, ok := commandFormatRaw.(map[string]interface{}); ok {
			commandFormat = cfgMap
		} else {
			hasCommandConfig = false
		}
	}

	// Create the table configuration
	tableConfig := formatter.TableConfig{
		Title:         title,
		Format:        formatOverride, // Use format override if provided
		BoldHeaders:   true,
		ShowSeparator: true,
		CommandPath:   commandPath,
		SiteLookup:    client, // Pass client for site name lookups
	}

	// Set the format from config if available and no override
	if tableConfig.Format == "" && hasCommandConfig {
		if format, ok := commandFormat["format"].(string); ok {
			tableConfig.Format = format
		}
	}

	// If the format is still empty, default to "table"
	if tableConfig.Format == "" {
		tableConfig.Format = "table"
	}

	// Create the table printer
	printer := formatter.NewGenericTablePrinter(tableConfig, tableData)

	// If we have a command-specific configuration, use it
	if hasCommandConfig {
		if fields, ok := commandFormat["fields"]; ok {
			printer.LoadColumnsFromConfig(fields)
		}
	} else {
		// Use default columns when no config is available
		printer.Config.Columns = []formatter.TableColumn{
			{Field: "name", Title: "Name", MaxWidth: 0},
			{Field: "id", Title: "ID", MaxWidth: 0},
			{Field: "address", Title: "Address", MaxWidth: 0},
			{Field: "country_code", Title: "Country", MaxWidth: 0},
			{Field: "timezone", Title: "Timezone", MaxWidth: 0},
			{Field: "in_api", Title: "In API", MaxWidth: 0, IsBoolField: true},
		}
	}

	// Print the table
	fmt.Print(printer.Print())

	return nil
}

// ListSites lists all sites in the organization
func ListSites(ctx context.Context, client api.Client, filter string, formatOverride ...interface{}) error {
	format := ""
	showAll := false

	// Parse variable arguments - format first, then showAll
	if len(formatOverride) > 0 {
		if f, ok := formatOverride[0].(string); ok {
			format = f
		}
	}
	if len(formatOverride) > 1 {
		if s, ok := formatOverride[1].(bool); ok {
			showAll = s
		}
	}
	orgID := viper.GetString("api.credentials.org_id")
	logging.Infof("Listing sites for organization %s", logging.FormatOrgID(orgID))

	// Use the bidirectional site method
	sites, err := client.GetSites(ctx, orgID)
	if err != nil {
		logging.Errorf("Failed to list sites: %v", err)
		return fmt.Errorf("failed to list sites: %w", err)
	}

	logging.Infof("Retrieved %d sites", len(sites))

	// Sort sites alphabetically by name using bidirectional methods
	sortedSites := api.SortSitesNew(sites)
	logging.Debug("Sites sorted successfully")

	// Apply filter if provided
	if filter != "" {
		filteredSites := make([]*api.MistSite, 0)
		for _, site := range sortedSites {
			siteName := site.GetName()
			if patterns.Contains(siteName, filter) {
				filteredSites = append(filteredSites, site)
			}
		}
		sortedSites = filteredSites
		logging.Debugf("Filtered to %d sites matching '%s'", len(sortedSites), filter)
	}

	// Create title for the table
	title := fmt.Sprintf("Sites (%d)", len(sortedSites))

	// Convert API sites to map for the table formatter
	tableData := make([]formatter.GenericTableData, 0, len(sortedSites))
	for _, site := range sortedSites {
		siteData := formatter.GenericTableData{}

		// Name - using bidirectional method
		siteName := site.GetName()

		// Check if site has configuration once for all fields
		hasConfig := false
		if siteName != "" {
			hasConfig = hasSiteConfiguration(siteName)
		}

		// Apply green text to all fields if site has configuration
		if siteName != "" {
			if hasConfig {
				siteData["name"] = "GREEN_TEXT:" + siteName
			} else {
				siteData["name"] = siteName
			}
		} else {
			// hasConfig will always be false here since siteName is empty
			siteData["name"] = "<undefined>"
		}

		// ID - using bidirectional method
		siteID := site.GetID()
		if siteID != "" {
			if hasConfig {
				siteData["id"] = "GREEN_TEXT:" + siteID
			} else {
				siteData["id"] = siteID
			}
		} else {
			if hasConfig {
				siteData["id"] = "GREEN_TEXT:<undefined>"
			} else {
				siteData["id"] = "<undefined>"
			}
		}

		// Address
		if site.Address != nil {
			if hasConfig {
				siteData["address"] = "GREEN_TEXT:" + *site.Address
			} else {
				siteData["address"] = *site.Address
			}
		} else {
			if hasConfig {
				siteData["address"] = "GREEN_TEXT:"
			} else {
				siteData["address"] = ""
			}
		}

		// Country Code
		if site.CountryCode != nil {
			if hasConfig {
				siteData["country_code"] = "GREEN_TEXT:" + *site.CountryCode
			} else {
				siteData["country_code"] = *site.CountryCode
			}
		} else {
			if hasConfig {
				siteData["country_code"] = "GREEN_TEXT:"
			} else {
				siteData["country_code"] = ""
			}
		}

		// Timezone
		if site.Timezone != nil {
			if hasConfig {
				siteData["timezone"] = "GREEN_TEXT:" + *site.Timezone
			} else {
				siteData["timezone"] = *site.Timezone
			}
		} else {
			if hasConfig {
				siteData["timezone"] = "GREEN_TEXT:"
			} else {
				siteData["timezone"] = ""
			}
		}

		// Store the hasConfig boolean for the Config column
		siteData["has_config"] = hasConfig

		tableData = append(tableData, siteData)
	}

	// Create command path for config lookup
	commandPath := "show.api.sites"

	// Check if there's a command-specific format in the config
	displayCommands := viper.GetStringMap("display.commands")
	commandFormatRaw, hasCommandConfig := displayCommands[commandPath]

	var commandFormat config.CommandFormat
	if hasCommandConfig {
		// Convert the raw interface{} to CommandFormat
		if cmdMap, ok := commandFormatRaw.(map[string]interface{}); ok {
			// Extract the format
			if formatVal, ok := cmdMap["format"].(string); ok {
				commandFormat.Format = formatVal
			}
			// Extract fields (array of field configurations)
			if fieldsVal, ok := cmdMap["fields"].([]interface{}); ok {
				commandFormat.Fields = fieldsVal
			}
			// Extract title
			if titleVal, ok := cmdMap["title"].(string); ok {
				commandFormat.Title = titleVal
			}
		}
	}

	// Create the table configuration
	tableConfig := formatter.TableConfig{
		Title:         title,
		Format:        format, // Use format override if provided
		BoldHeaders:   true,
		ShowSeparator: true,
		CommandPath:   commandPath,
		ShowAllFields: showAll, // Enable showing all cache fields when true
	}

	// Set the format from config if available and no override
	if tableConfig.Format == "" && hasCommandConfig {
		tableConfig.Format = commandFormat.Format
	}

	// If the format is still empty, default to "table"
	if tableConfig.Format == "" {
		tableConfig.Format = "table"
	}

	// Create the table printer
	printer := formatter.NewGenericTablePrinter(tableConfig, tableData)

	// If we have a command-specific configuration, use it
	if hasCommandConfig {
		printer.LoadColumnsFromConfig(commandFormat.Fields)
	} else {
		// Use default columns when no config is available
		printer.Config.Columns = []formatter.TableColumn{
			{Field: "name", Title: "Name", MaxWidth: 0},
			{Field: "id", Title: "ID", MaxWidth: 0},
			{Field: "address", Title: "Address", MaxWidth: 0},
			{Field: "country_code", Title: "Country", MaxWidth: 0},
			{Field: "timezone", Title: "Timezone", MaxWidth: 0},
			{Field: "has_config", Title: "Config", MaxWidth: 0, IsBoolField: true},
		}
	}

	// Print the table
	fmt.Print(printer.Print())

	return nil
}

// GetSite retrieves details for a specific site
func GetSite(ctx context.Context, client api.Client, identifier string) error {
	// Resolve the site ID based on the identifier format
	siteID, err := utils.ResolveSiteIDViper(ctx, client, identifier)
	if err != nil {
		// Return the descriptive error message directly to the user
		return err
	}

	// Get the site details
	site, err := client.GetSite(ctx, siteID)
	if err != nil {
		if errors.Is(err, api.ErrNotFound) {
			return fmt.Errorf("no site found with ID '%s'", siteID)
		}
		return fmt.Errorf("failed to get site: %w", err)
	}

	// Print site details - only the header in blue
	utils.PrintWithWarning("Site Details:")
	utils.PrintDetailWithWarning("  ID: %s", *site.ID)
	utils.PrintDetailWithWarning("  Name: %s", *site.Name)
	if site.Address != nil {
		utils.PrintDetailWithWarning("  Address: %s", *site.Address)
	}
	if site.CountryCode != nil {
		utils.PrintDetailWithWarning("  Country Code: %s", *site.CountryCode)
	}
	if site.Timezone != nil {
		utils.PrintDetailWithWarning("  Timezone: %s", *site.Timezone)
	}

	return nil
}

// CreateSite creates a new site
func CreateSite(_ context.Context, _ api.Client) error {
	return fmt.Errorf("site creation functionality requires site configuration files - use site config files and apply command instead")
}

// UpdateSite updates an existing site
func UpdateSite(_ context.Context, _ api.Client, _ string) error {
	return fmt.Errorf("site update functionality requires site configuration files - use site config files and apply command instead")
}

// hasSiteConfiguration checks if a site has a configuration file
func hasSiteConfiguration(siteName string) bool {
	siteConfigFiles := viper.GetStringSlice("files.site_configs")
	if len(siteConfigFiles) == 0 {
		logging.Debugf("No site config files specified for site: %s", siteName)
		return false
	}

	configDir := viper.GetString("files.config_dir")

	// Check each site config file
	for _, siteConfigFile := range siteConfigFiles {
		logging.Debugf("Checking site config file: %s for site: %s", siteConfigFile, siteName)

		// Load site configuration
		siteConfig, err := config.LoadSiteConfig(configDir, siteConfigFile)
		if err != nil {
			logging.Debugf("Failed to load site configuration from %s: %v", siteConfigFile, err)
			continue
		}

		// Check each site in the configuration
		for siteKey, siteObj := range siteConfig.Config.Sites {
			logging.Debugf("Comparing site config: %s (key: %s) with site: %s", siteObj.SiteConfig.Name, siteKey, siteName)

			// Use case-insensitive comparison for site names
			if strings.EqualFold(siteObj.SiteConfig.Name, siteName) {
				logging.Infof("Found matching site configuration for site: %s", siteName)
				return true
			}
		}
	}
	logging.Debugf("No matching site configuration found for site: %s", siteName)
	return false
}

// DeleteSite deletes a site
func DeleteSite(ctx context.Context, client api.Client, identifier string, force bool) error {
	// Resolve the site ID based on the identifier format
	siteID, err := utils.ResolveSiteIDViper(ctx, client, identifier)
	if err != nil {
		return err
	}

	// Get the site details to show what will be deleted
	site, err := client.GetSite(ctx, siteID)
	if err != nil {
		if errors.Is(err, api.ErrNotFound) {
			return fmt.Errorf("no site found with ID '%s'", siteID)
		}
		return fmt.Errorf("failed to get site details: %w", err)
	}

	// Display site details to be deleted
	utils.PrintWithWarning("\nYou are about to delete the following site:")
	utils.PrintDetailWithWarning("  ID: %s", *site.ID)
	utils.PrintDetailWithWarning("  Name: %s", *site.Name)
	if site.Address != nil {
		utils.PrintDetailWithWarning("  Address: %s", *site.Address)
	}

	// Prompt for confirmation unless force flag is set
	if !force {
		if !utils.PromptForConfirmation("\nAre you sure you want to delete this site? [y/N]: ") {
			fmt.Println("Site deletion cancelled.")
			return nil
		}
	} else {
		fmt.Println("Force flag set. Proceeding with deletion without confirmation.")
	}

	// Delete the site
	err = client.DeleteSite(ctx, siteID)
	if err != nil {
		return fmt.Errorf("failed to delete site: %w", err)
	}

	utils.PrintWithWarning("Site %s deleted successfully!", siteID)

	return nil
}

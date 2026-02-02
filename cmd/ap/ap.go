package ap

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/ravinald/wifimgr/api"
	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/formatter"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/symbols"
	"github.com/ravinald/wifimgr/internal/utils"
)

// HandleCommand processes AP-related subcommands
func HandleCommand(ctx context.Context, client api.Client, args []string, formatOverride string, _ bool) error {
	if len(args) < 1 {
		logging.Error("No AP subcommand specified")
		return fmt.Errorf("no AP subcommand specified")
	}

	logging.Infof("Executing AP subcommand: %s", args[0])

	switch args[0] {
	case "list":
		if len(args) < 2 {
			logging.Error("Site ID required for AP list command")
			return fmt.Errorf("site ID required")
		}
		return ListAPs(ctx, client, args[1], formatOverride)
	case "get":
		if len(args) < 2 {
			logging.Error("AP identifier (name or MAC) required for AP get command")
			return fmt.Errorf("AP identifier (name or MAC) required")
		}
		return fmt.Errorf("AP get functionality requires site configuration - use 'show api ap' command instead")
	case "update":
		if len(args) < 3 {
			logging.Error("Site ID and AP ID required for AP update command")
			return fmt.Errorf("site ID and AP ID required")
		}
		return fmt.Errorf("AP update functionality requires site configuration - use apply command instead")
	case "assign":
		if len(args) < 2 {
			logging.Error("Site ID required for AP assign command")
			return fmt.Errorf("site ID required")
		}
		return fmt.Errorf("AP assign functionality requires site configuration - use apply command instead")
	case "assign-bulk":
		if len(args) < 2 {
			logging.Error("Site ID required for AP assign-bulk command")
			return fmt.Errorf("site ID required")
		}
		return fmt.Errorf("AP assign-bulk functionality requires site configuration - use apply command instead")
	case "assign-bulk-file":
		// New command to handle file-based bulk assignment
		if len(args) < 2 {
			logging.Error("File path required for AP assign-bulk-file command")
			return fmt.Errorf("file path required")
		}

		// Check if a site name is provided
		if len(args) > 2 {
			// Use the site specified
			return fmt.Errorf("AP assign-bulk-file functionality requires site configuration - use apply command instead")
		}

		// No site specified, assume CSV format
		return fmt.Errorf("AP assign-bulk-file functionality requires site configuration - use apply command instead")
	case "unassign":
		if len(args) < 3 {
			logging.Error("Site ID and AP ID required for AP unassign command")
			return fmt.Errorf("site ID and AP ID required")
		}
		return fmt.Errorf("AP unassign functionality requires site configuration - use apply command instead")
	default:
		logging.Errorf("Unknown AP subcommand: %s", args[0])
		return fmt.Errorf("unknown AP subcommand: %s", args[0])
	}
}

// ListAPs lists all APs in a site
func ListAPs(ctx context.Context, client api.Client, identifier string, formatOverride string) error {
	// Resolve the site ID based on the identifier format
	siteID, err := utils.ResolveSiteIDViper(ctx, client, identifier)
	if err != nil {
		// Return the descriptive error message directly to the user
		return err
	}

	devices, err := client.GetDevicesByType(ctx, siteID, "ap")
	if err != nil {
		if errors.Is(err, api.ErrNotFound) {
			return fmt.Errorf("no site found with ID '%s'", siteID)
		}
		return fmt.Errorf("failed to list APs: %w", err)
	}

	// Convert UnifiedDevice to AP for compatibility
	aps := api.ConvertToAPSlice(devices)

	// Sort the APs according to naming format rules
	sortedAPs := api.SortAPs(aps)

	// Create title for the table
	title := fmt.Sprintf("APs for Site %s (%d)", siteID, len(sortedAPs))

	// Convert APs to map for the table formatter
	tableData := make([]formatter.GenericTableData, 0, len(sortedAPs))
	for _, ap := range sortedAPs {
		apData := formatter.GenericTableData{}

		// Name
		if ap.Name != nil && *ap.Name != "" {
			apData["name"] = *ap.Name
		} else {
			apData["name"] = "<undefined>"
		}

		// ID
		if ap.Id != nil {
			apData["id"] = *ap.Id
		} else {
			apData["id"] = ""
		}

		// MAC
		if ap.Mac != nil {
			apData["mac"] = *ap.Mac
		} else {
			apData["mac"] = ""
		}

		// Serial
		if ap.Serial != nil {
			apData["serial"] = *ap.Serial
		} else {
			apData["serial"] = ""
		}

		// Model
		if ap.Model != nil {
			apData["model"] = *ap.Model
		} else {
			apData["model"] = ""
		}

		// Status
		if ap.Status != nil {
			apData["status"] = *ap.Status
		} else {
			apData["status"] = ""
		}

		tableData = append(tableData, apData)
	}

	// Create command path for config lookup
	commandPath := "show.site.ap"

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

	// Create the table configuration
	tableConfig := formatter.TableConfig{
		Title:         title,
		Format:        formatOverride, // Use override if provided
		BoldHeaders:   true,
		ShowSeparator: true,
		CommandPath:   commandPath,
		SiteLookup:    client, // Pass client for site name lookups
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
			{Field: "mac", Title: "MAC", MaxWidth: 0},
			{Field: "serial", Title: "Serial", MaxWidth: 0},
			{Field: "model", Title: "Model", MaxWidth: 0},
			{Field: "status", Title: "Status", MaxWidth: 0},
		}
	}

	// Print the table
	fmt.Print(printer.Print())

	return nil
}

// GetAP retrieves details for a specific AP
func GetAP(ctx context.Context, client api.Client, cfg *config.Config, identifier, apID string) error {
	// Resolve the site ID based on the identifier format
	siteID, err := utils.ResolveSiteID(ctx, client, cfg, identifier)
	if err != nil {
		// Return the descriptive error message directly to the user
		return err
	}

	device, err := client.GetDeviceByName(ctx, siteID, apID)
	if err != nil {
		if errors.Is(err, api.ErrNotFound) {
			return fmt.Errorf("no AP found with ID '%s' in site '%s'", apID, identifier)
		}
		return fmt.Errorf("failed to get AP: %w", err)
	}

	// Convert the UnifiedDevice to AP for compatibility
	apSlice := api.ConvertToAPSlice([]api.UnifiedDevice{*device})
	if len(apSlice) == 0 {
		return fmt.Errorf("no AP found with ID '%s' in site '%s'", apID, identifier)
	}
	ap := apSlice[0]

	// Print AP details
	utils.PrintWithWarning("AP Details:")
	utils.PrintWithWarning("  ID: %s", *ap.Id)
	utils.PrintWithWarning("  Name: %s", *ap.Name)
	if ap.Mac != nil {
		utils.PrintWithWarning("  MAC: %s", *ap.Mac)
	}
	if ap.Serial != nil {
		utils.PrintWithWarning("  Serial: %s", *ap.Serial)
	}

	return nil
}

// UpdateAP updates an existing AP
func UpdateAP(ctx context.Context, client api.Client, cfg *config.Config, identifier, apID string) error {
	// Check if we have site configs
	if len(cfg.Files.SiteConfigs) == 0 {
		return fmt.Errorf("no site configuration files specified")
	}

	// Load all site configs
	_, siteConfigs, err := config.LoadAllConfigs(cfg.Files.SiteConfigs[0])
	if err != nil {
		return fmt.Errorf("failed to load site configurations: %w", err)
	}

	if len(siteConfigs) == 0 {
		return fmt.Errorf("no site configurations found in config files")
	}

	devices := siteConfigs[0].GetDevices()
	if devices == nil || len(devices.APs) == 0 {
		return fmt.Errorf("no AP configurations found in config files")
	}

	// Resolve the site ID based on the identifier format
	siteID, err := utils.ResolveSiteID(ctx, client, cfg, identifier)
	if err != nil {
		// Return the descriptive error message directly to the user
		return err
	}

	// Use the first AP configuration from the map
	var apCfg config.APConfig
	for _, ap := range devices.APs {
		apCfg = ap
		break
	}

	// Get existing device
	device, err := client.GetDeviceByName(ctx, siteID, apID)
	if err != nil {
		if errors.Is(err, api.ErrNotFound) {
			return fmt.Errorf("no AP found with ID '%s' in site '%s'", apID, identifier)
		}
		return fmt.Errorf("failed to get AP: %w", err)
	}

	// Update device object
	if apCfg.Name != "" {
		device.Name = &apCfg.Name
	}
	if apCfg.Notes != "" {
		device.Notes = &apCfg.Notes
	}

	// Update the device using new method
	deviceID := ""
	if device.ID != nil {
		deviceID = *device.ID
	}
	updatedDevice, err := client.UpdateDevice(ctx, siteID, deviceID, device)
	if err != nil {
		if errors.Is(err, api.ErrNotFound) {
			return fmt.Errorf("failed to update AP with ID '%s' in site '%s'", apID, identifier)
		}
		return fmt.Errorf("failed to update AP: %w", err)
	}

	// Convert the updated UnifiedDevice to AP for compatibility
	updatedAPSlice := api.ConvertToAPSlice([]api.UnifiedDevice{*updatedDevice})
	if len(updatedAPSlice) == 0 {
		return fmt.Errorf("failed to convert updated device to AP")
	}
	updatedAP := updatedAPSlice[0]

	utils.PrintWithWarning("AP updated successfully!")
	utils.PrintWithWarning("  ID: %s", *updatedAP.Id)
	utils.PrintWithWarning("  Name: %s", *updatedAP.Name)

	return nil
}

// AssignAP assigns an AP to a site
func AssignAP(ctx context.Context, client api.Client, cfg *config.Config, identifier string) error {
	// Check if we have site configs
	if len(cfg.Files.SiteConfigs) == 0 {
		return fmt.Errorf("no site configuration files specified")
	}

	// Load all site configs
	_, siteConfigs, err := config.LoadAllConfigs(cfg.Files.SiteConfigs[0])
	if err != nil {
		return fmt.Errorf("failed to load site configurations: %w", err)
	}

	if len(siteConfigs) == 0 {
		return fmt.Errorf("no site configurations found in config files")
	}

	devices := siteConfigs[0].GetDevices()
	if devices == nil || len(devices.APs) == 0 {
		return fmt.Errorf("no AP configurations found in config files")
	}

	// Resolve the site ID based on the identifier format
	siteID, err := utils.ResolveSiteID(ctx, client, cfg, identifier)
	if err != nil {
		// Return the descriptive error message directly to the user
		return err
	}

	// Use the first AP configuration from the map
	var apCfg config.APConfig
	for _, ap := range devices.APs {
		apCfg = ap
		break
	}

	// Set required fields
	// name variable removed (not used)
	magic := ""
	if apCfg.Magic != "" {
		magic = apCfg.Magic
	}

	// Try to assign the AP using bidirectional method
	assignedDevice, err := client.AssignDevice(ctx, cfg.API.Credentials.OrgID, siteID, magic)
	if err != nil {
		if errors.Is(err, api.ErrNotFound) {
			return fmt.Errorf("failed to assign AP to site '%s': site not found", identifier)
		}
		return fmt.Errorf("failed to assign AP: %w", err)
	}

	// Convert to AP for compatibility
	assignedAPSlice := api.ConvertToAPSlice([]api.UnifiedDevice{*assignedDevice})
	if len(assignedAPSlice) == 0 {
		return fmt.Errorf("failed to convert assigned device to AP")
	}
	assignedAP := assignedAPSlice[0]

	utils.PrintWithWarning("AP assigned successfully!")
	utils.PrintWithWarning("  ID: %s", *assignedAP.Id)
	utils.PrintWithWarning("  Name: %s", *assignedAP.Name)

	return nil
}

// AssignBulkAPs assigns multiple APs to a site concurrently
func AssignBulkAPs(ctx context.Context, client api.Client, cfg *config.Config, identifier string) error {
	// Check if we have site configs
	if len(cfg.Files.SiteConfigs) == 0 {
		return fmt.Errorf("no site configuration files specified")
	}

	// Load all site configs
	_, siteConfigs, err := config.LoadAllConfigs(cfg.Files.SiteConfigs[0])
	if err != nil {
		return fmt.Errorf("failed to load site configurations: %w", err)
	}

	if len(siteConfigs) == 0 {
		return fmt.Errorf("no site configurations found in config files")
	}

	devices := siteConfigs[0].GetDevices()
	if devices == nil || len(devices.APs) == 0 {
		return fmt.Errorf("no AP configurations found in config files")
	}

	// Resolve the site ID based on the identifier format
	siteID, err := utils.ResolveSiteID(ctx, client, cfg, identifier)
	if err != nil {
		// Return the descriptive error message directly to the user
		return err
	}

	apConfigs := devices.APs
	fmt.Printf("Preparing to assign %d APs to site %s concurrently...\n", len(apConfigs), siteID)

	// Extract MAC addresses from AP configs
	macs := make([]string, 0, len(apConfigs))
	aps := make([]api.AP, 0, len(apConfigs))
	apConfigList := make([]config.APConfig, 0, len(apConfigs))

	for _, apCfg := range apConfigs {
		apConfigList = append(apConfigList, apCfg)
		ap := api.AP{}

		// Set required fields
		name := apCfg.Name
		ap.Name = &name

		// Use Magic field directly for device identification
		if apCfg.Magic != "" {
			magic := apCfg.Magic
			ap.Magic = &magic
			macs = append(macs, magic)
		}

		aps = append(aps, ap)
	}

	// Start time for measuring performance
	startTime := time.Now()

	// Assign devices to site using bidirectional method
	err = client.AssignDevicesToSite(ctx, cfg.API.Credentials.OrgID, siteID, macs, false)

	// Generate assignment errors array with nil or error values
	assignmentErrors := make([]error, len(aps))
	if err != nil {
		// If overall assignment failed, mark all assignments as failed
		for i := range assignmentErrors {
			assignmentErrors[i] = fmt.Errorf("failed to assign AP: %v", err)
		}
	}

	// Count successful assignments
	successCount := 0
	for _, err := range assignmentErrors {
		if err == nil {
			successCount++
		}
	}

	// Calculate elapsed time
	elapsedTime := time.Since(startTime)

	utils.PrintWithWarning("Bulk AP assignment complete!")
	utils.PrintWithWarning("  Successfully assigned: %d of %d APs", successCount, len(aps))
	utils.PrintWithWarning("  Failed assignments: %d", len(aps)-successCount)
	utils.PrintWithWarning("  Operation took: %v", elapsedTime)

	// Print any errors
	if len(aps)-successCount > 0 {
		utils.PrintWithWarning("\nErrors encountered:")
		for i, err := range assignmentErrors {
			if err != nil {
				utils.PrintWithWarning("  AP %d (%s): %v", i+1, apConfigList[i].Name, err)
			}
		}
	}

	return nil
}

// AssignBulkAPsFromFile assigns multiple APs from a file to a specific site
func AssignBulkAPsFromFile(ctx context.Context, client api.Client, cfg *config.Config, filePath string, identifier string) error {
	// Resolve the site ID based on the identifier format
	siteID, err := utils.ResolveSiteID(ctx, client, cfg, identifier)
	if err != nil {
		// Return the descriptive error message directly to the user
		return err
	}

	// Read the file content
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		logging.Errorf("Failed to read file %s: %v", filePath, err)
		return fmt.Errorf("failed to read file %s: %v", filePath, err)
	}

	// Parse the content to extract MAC addresses
	lines := strings.Split(string(fileContent), "\n")
	var macAddresses []string

	for _, line := range lines {
		// Skip empty lines
		if len(strings.TrimSpace(line)) == 0 {
			continue
		}

		// Extract MAC address from each line
		mac := strings.TrimSpace(line)
		macAddresses = append(macAddresses, mac)
	}

	if len(macAddresses) == 0 {
		return fmt.Errorf("no MAC addresses found in file %s", filePath)
	}

	fmt.Printf("Found %d MAC addresses in file %s\n", len(macAddresses), filePath)
	fmt.Printf("Preparing to assign %d APs to site %s...\n", len(macAddresses), siteID)

	// Create AP objects for each MAC address
	aps := make([]api.AP, len(macAddresses))
	for i, mac := range macAddresses {
		ap := api.AP{}

		// Set MAC as magic field for device identification
		magic := mac
		ap.Magic = &magic

		// Set a default name based on MAC
		name := fmt.Sprintf("AP-%s", mac)
		ap.Name = &name

		aps[i] = ap
	}

	// Removed unused concurrency variable

	// Start time for measuring performance
	startTime := time.Now()

	// Assign APs to site using new bidirectional method
	err = client.AssignDevicesToSite(ctx, cfg.API.Credentials.OrgID, siteID, macAddresses, false)

	// Generate assignment errors array with nil or error values
	assignmentErrors := make([]error, len(aps))
	if err != nil {
		// If overall assignment failed, mark all assignments as failed
		for i := range assignmentErrors {
			assignmentErrors[i] = fmt.Errorf("failed to assign AP: %v", err)
		}
	}

	// Count successful assignments
	successCount := 0
	for _, err := range assignmentErrors {
		if err == nil {
			successCount++
		}
	}

	// Calculate elapsed time
	elapsedTime := time.Since(startTime)

	utils.PrintWithWarning("Bulk AP assignment complete!")
	utils.PrintWithWarning("  Successfully assigned: %d of %d APs", successCount, len(aps))
	utils.PrintWithWarning("  Failed assignments: %d", len(aps)-successCount)
	utils.PrintWithWarning("  Operation took: %v", elapsedTime)

	// Print any errors
	if len(aps)-successCount > 0 {
		utils.PrintWithWarning("\nErrors encountered:")
		for i, err := range assignmentErrors {
			if err != nil {
				utils.PrintWithWarning("  AP %d (%s): %v", i+1, macAddresses[i], err)
			}
		}
	}

	return nil
}

// AssignBulkAPsFromCSV assigns multiple APs from a CSV file where each line has a MAC and site
func AssignBulkAPsFromCSV(ctx context.Context, client api.Client, cfg *config.Config, filePath string) error {
	// Read the file content
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		logging.Errorf("Failed to read file %s: %v", filePath, err)
		return fmt.Errorf("failed to read file %s: %v", filePath, err)
	}

	// Parse the CSV content to extract MAC addresses and site names
	lines := strings.Split(string(fileContent), "\n")

	// Map to store APs by site
	apsBySite := make(map[string][]api.AP)

	// Track original MAC addresses for error reporting
	macAddressesBySite := make(map[string][]string)

	for _, line := range lines {
		// Skip empty lines
		if len(strings.TrimSpace(line)) == 0 {
			continue
		}

		// Split the line into MAC and site
		parts := strings.Split(strings.TrimSpace(line), ",")
		if len(parts) < 2 {
			return fmt.Errorf("invalid format in file: expected MAC,SITE format")
		}

		mac := strings.TrimSpace(parts[0])
		siteName := strings.TrimSpace(parts[1])

		// Create AP object
		ap := api.AP{}

		// Set MAC as magic field for device identification
		magic := mac
		ap.Magic = &magic

		// Set a default name based on MAC
		name := fmt.Sprintf("AP-%s", mac)
		ap.Name = &name

		// Add to the site's APs
		apsBySite[siteName] = append(apsBySite[siteName], ap)

		// Track original MAC address for error reporting
		macAddressesBySite[siteName] = append(macAddressesBySite[siteName], mac)
	}

	if len(apsBySite) == 0 {
		return fmt.Errorf("no valid entries found in file %s", filePath)
	}

	fmt.Printf("Found %d different sites with APs to assign in file %s\n", len(apsBySite), filePath)

	// Removed unused concurrency variable

	// Track overall statistics
	totalAPs := 0
	totalSuccessCount := 0
	startTime := time.Now()

	// Process each site
	for siteName, aps := range apsBySite {
		// Resolve the site ID
		siteID, err := utils.ResolveSiteID(ctx, client, cfg, siteName)
		if err != nil {
			utils.PrintWithWarning("Error: Could not resolve site ID for site '%s': %v", siteName, err)
			continue
		}

		totalAPs += len(aps)
		fmt.Printf("Assigning %d APs to site %s...\n", len(aps), siteID)

		// Assign APs concurrently
		/* Extract MAC addresses for this site's APs */
		siteMacs := make([]string, 0, len(aps))
		for _, ap := range aps {
			if ap.Magic != nil {
				siteMacs = append(siteMacs, *ap.Magic)
			}
		}

		/* Assign devices to this site using new bidirectional method */
		siteErr := client.AssignDevicesToSite(ctx, cfg.API.Credentials.OrgID, siteID, siteMacs, false)

		/* Create error array to match original API behavior */
		assignmentErrors := make([]error, len(aps))
		if siteErr != nil {
			for i := range assignmentErrors {
				assignmentErrors[i] = siteErr
			}
		}

		// Count successful assignments
		successCount := 0
		for _, err := range assignmentErrors {
			if err == nil {
				successCount++
				totalSuccessCount++
			}
		}

		utils.PrintWithWarning("Site %s: assigned %d of %d APs successfully", siteName, successCount, len(aps))

		// Print any errors
		if len(aps)-successCount > 0 {
			utils.PrintWithWarning("  Errors for site %s:", siteName)
			for i, err := range assignmentErrors {
				if err != nil {
					utils.PrintWithWarning("    AP %d (%s): %v", i+1, macAddressesBySite[siteName][i], err)
				}
			}
		}
	}

	// Calculate elapsed time
	elapsedTime := time.Since(startTime)

	utils.PrintWithWarning("\nBulk AP assignment complete!")
	utils.PrintWithWarning("  Successfully assigned: %d of %d APs across %d sites",
		totalSuccessCount, totalAPs, len(apsBySite))
	utils.PrintWithWarning("  Failed assignments: %d", totalAPs-totalSuccessCount)
	utils.PrintWithWarning("  Operation took: %v", elapsedTime)

	return nil
}

// GetAPByIdentifier retrieves AP details using just the AP identifier (MAC or name)
func GetAPByIdentifier(ctx context.Context, client api.Client, cfg *config.Config, apIdentifier string) error {
	logging.Infof("Looking up AP by identifier: %s", apIdentifier)

	// Get all sites
	sites, err := client.GetSites(ctx, cfg.API.Credentials.OrgID)
	if err != nil {
		return fmt.Errorf("failed to get sites: %w", err)
	}

	// Check if the identifier is a MAC address - import macaddr package for this
	var foundAP *api.AP
	var foundSiteName string

	// Optimization: If identifier looks like a MAC address, try direct cache lookup first
	// Note: This requires importing the macaddr package
	// For now, we'll use a simple heuristic to detect MAC addresses
	isMACAddress := strings.Contains(apIdentifier, ":") && len(apIdentifier) == 17

	if isMACAddress {
		logging.Infof("MAC address detected: %s, attempting optimized lookup", apIdentifier)

		// Try direct MAC lookup first
		device, err := client.GetDeviceByMAC(ctx, apIdentifier)
		if err == nil && device != nil && device.Type != nil && *device.Type == "ap" {
			logging.Infof("%s Found AP via optimized MAC lookup", symbols.SuccessPrefix())
			apSlice := api.ConvertToAPSlice([]api.UnifiedDevice{*device})
			if len(apSlice) > 0 {
				foundAP = &apSlice[0]

				// Get site name for the found AP
				if device.SiteID != nil {
					siteID := *device.SiteID
					for _, site := range sites {
						if site.ID != nil && *site.ID == siteID {
							if site.Name != nil {
								foundSiteName = *site.Name
							} else {
								foundSiteName = siteID
							}
							break
						}
					}
				}
			}
		} else {
			logging.Infof("%s Direct MAC lookup failed: %v", symbols.FailurePrefix(), err)
		}
	}

	// If not found via optimized lookup, fallback to iterating through sites
	if foundAP == nil {
		logging.Infof("Falling back to site iteration for identifier: %s", apIdentifier)

		// Look through each site
		for _, site := range sites {
			if site.ID == nil {
				continue
			}

			siteID := *site.ID

			// Try to get AP directly by identifier
			/* Look for AP using the device interface */
			var ap *api.AP

			/* Try to get device by name first */
			device, err := client.GetDeviceByName(ctx, siteID, apIdentifier)
			if err == nil && device != nil && device.Type != nil && *device.Type == "ap" {
				/* Convert to AP */
				apSlice := api.ConvertToAPSlice([]api.UnifiedDevice{*device})
				if len(apSlice) > 0 {
					ap = &apSlice[0]
				}
			} else {
				/* Try to get by MAC directly */
				device, err := client.GetDeviceByMAC(ctx, apIdentifier)
				if err == nil && device != nil && device.Type != nil && *device.Type == "ap" {
					/* Convert to AP */
					apSlice := api.ConvertToAPSlice([]api.UnifiedDevice{*device})
					if len(apSlice) > 0 {
						ap = &apSlice[0]
					}
				}
			}
			if err == nil && ap != nil {
				foundAP = ap
				if site.Name != nil {
					foundSiteName = *site.Name
				} else {
					foundSiteName = siteID
				}
				break
			}
		}
	}

	// If AP wasn't found
	if foundAP == nil {
		return fmt.Errorf("no AP found with identifier '%s' in any site", apIdentifier)
	}

	// Print AP details
	utils.PrintWithWarning("AP Details:")
	utils.PrintWithWarning("  ID: %s", *foundAP.Id)
	utils.PrintWithWarning("  Name: %s", *foundAP.Name)
	if foundAP.Mac != nil {
		utils.PrintWithWarning("  MAC: %s", *foundAP.Mac)
	}
	if foundAP.Serial != nil {
		utils.PrintWithWarning("  Serial: %s", *foundAP.Serial)
	}
	utils.PrintWithWarning("  Site: %s", foundSiteName)

	return nil
}

// UnassignAP unassigns an AP from a site
func UnassignAP(ctx context.Context, client api.Client, cfg *config.Config, identifier, apID string, force bool) error {
	// Resolve the site ID based on the identifier format
	siteID, err := utils.ResolveSiteID(ctx, client, cfg, identifier)
	if err != nil {
		// Return the descriptive error message directly to the user
		return err
	}

	// Get the AP details to show what will be unassigned using new bidirectional method
	device, err := client.GetDeviceByName(ctx, siteID, apID)
	if err != nil {
		if errors.Is(err, api.ErrNotFound) {
			return fmt.Errorf("no AP found with ID '%s' in site '%s'", apID, identifier)
		}
		return fmt.Errorf("failed to get AP details: %w", err)
	}

	// Convert to AP for compatibility
	apSlice := api.ConvertToAPSlice([]api.UnifiedDevice{*device})
	if len(apSlice) == 0 {
		return fmt.Errorf("no AP found with ID '%s' in site '%s'", apID, identifier)
	}
	ap := apSlice[0]

	// Display AP details to be unassigned
	utils.PrintWithWarning("\nYou are about to unassign the following AP from site '%s':", siteID)
	utils.PrintWithWarning("  ID: %s", *ap.Id)
	utils.PrintWithWarning("  Name: %s", *ap.Name)
	if ap.Serial != nil {
		utils.PrintWithWarning("  Serial: %s", *ap.Serial)
	}
	if ap.Mac != nil {
		utils.PrintWithWarning("  MAC: %s", *ap.Mac)
	}

	// Prompt for confirmation unless force flag is set
	if !force {
		if !utils.PromptForConfirmation("\nAre you sure you want to unassign this AP? [y/N]: ") {
			fmt.Println("AP unassignment cancelled.")
			return nil
		}
	} else {
		fmt.Println("Force flag set. Proceeding with unassignment without confirmation.")
	}

	// Unassign the AP using the new bidirectional device interface
	deviceID := ""
	if device.ID != nil {
		deviceID = *device.ID
	}
	err = client.UnassignDevice(ctx, cfg.API.Credentials.OrgID, siteID, deviceID)
	if err != nil {
		if errors.Is(err, api.ErrNotFound) {
			return fmt.Errorf("no AP found with ID '%s' in site '%s'", apID, identifier)
		}
		return fmt.Errorf("failed to unassign AP: %w", err)
	}

	utils.PrintWithWarning("AP %s unassigned successfully from site %s!", apID, siteID)

	return nil
}

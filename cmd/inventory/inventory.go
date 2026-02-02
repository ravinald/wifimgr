package inventory

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/viper"

	"github.com/ravinald/wifimgr/api"
	"github.com/ravinald/wifimgr/internal/cmdutils"
	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/formatter"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/utils"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// LocalInventoryFile represents the structure of the inventory.json file
type LocalInventoryFile struct {
	Version  int `json:"version"`
	Metadata struct {
		Description string `json:"description"`
	} `json:"metadata"`
	Config struct {
		Inventory struct {
			AP      []string `json:"ap"`
			Switch  []string `json:"switch"`
			Gateway []string `json:"gateway"`
		} `json:"inventory"`
	} `json:"config"`
}

// HandleCommand processes inventory-related subcommands
func HandleCommand(ctx context.Context, client api.Client, cfg *config.Config, args []string, formatOverride string, showAll ...bool) error {
	showAllFields := false
	if len(showAll) > 0 {
		showAllFields = showAll[0]
	}

	// The args should already be parsed by the cmd layer
	// args[0] = device type (optional)
	// args[1] = "site" (if present)
	// args[2] = site name (if args[1] == "site")

	deviceType := ""
	siteName := ""

	if len(args) > 0 {
		// Check if first arg is "site"
		if args[0] == "site" {
			// Pattern: site <site-name>
			if len(args) < 2 {
				return fmt.Errorf("'site' requires a site name")
			}
			siteName = args[1]
		} else {
			// First arg is device type
			deviceType = args[0]

			// Check for site after device type
			if len(args) > 1 && args[1] == "site" {
				if len(args) < 3 {
					return fmt.Errorf("'site' requires a site name")
				}
				siteName = args[2]
			}
		}
	}

	// Validate device type if provided
	if deviceType != "" {
		if err := cmdutils.ValidateDeviceType(deviceType); err != nil {
			logging.Errorf("Invalid device type: %v", err)
			return err
		}
		// Normalize device type
		deviceType = cmdutils.NormalizeDeviceType(deviceType)
	}

	// Log what we're doing
	if siteName != "" {
		if deviceType == "" {
			logging.Infof("Showing all inventory items for site: %s", siteName)
		} else {
			logging.Infof("Showing %s inventory items for site: %s", deviceType, siteName)
		}
	} else {
		if deviceType == "" {
			logging.Info("Showing all inventory items")
		} else {
			logging.Infof("Showing %s inventory items", deviceType)
		}
	}

	return ListInventory(ctx, client, cfg, deviceType, siteName, formatOverride, showAllFields)
}

// LoadLocalInventory loads the inventory from the local inventory file
func LoadLocalInventory(filePath string) (*LocalInventoryFile, error) {
	logging.Infof("Loading inventory from local file: %s", filePath)

	file, err := os.Open(filePath)
	if err != nil {
		logging.Errorf("Failed to open inventory file: %v", err)
		return nil, fmt.Errorf("failed to open inventory file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			logging.Warnf("Failed to close inventory file: %v", closeErr)
		}
	}()

	var inventory LocalInventoryFile
	if err := json.NewDecoder(file).Decode(&inventory); err != nil {
		logging.Errorf("Failed to parse inventory file: %v", err)
		return nil, fmt.Errorf("failed to parse inventory file: %w", err)
	}

	// Extract counts for each device type
	apCount := len(inventory.Config.Inventory.AP)
	switchCount := len(inventory.Config.Inventory.Switch)
	gatewayCount := len(inventory.Config.Inventory.Gateway)

	logging.Infof("Successfully loaded inventory from file with %d AP, %d switch, and %d gateway items",
		apCount, switchCount, gatewayCount)

	return &inventory, nil
}

// ConvertLocalInventoryToVendors converts local inventory items to vendors inventory items
// by retrieving device data from the cache using O(1) MAC lookups
func ConvertLocalInventoryToVendors(macAddresses []string, deviceType string) []*vendors.InventoryItem {
	var items []*vendors.InventoryItem

	// Get cache accessor for O(1) lookups
	cacheAccessor, err := cmdutils.GetCacheAccessor()
	if err != nil {
		logging.Errorf("Cache not available: %v", err)
		return items
	}

	// Use O(1) lookups for each MAC address
	for _, mac := range macAddresses {
		var item *vendors.InventoryItem

		// Use proper O(1) MAC lookup based on device type
		switch deviceType {
		case "ap":
			if device, err := cacheAccessor.GetAPByMAC(mac); err == nil && device != nil {
				item = device
			}
		case "switch":
			if device, err := cacheAccessor.GetSwitchByMAC(mac); err == nil && device != nil {
				item = device
			}
		case "gateway":
			if device, err := cacheAccessor.GetGatewayByMAC(mac); err == nil && device != nil {
				item = device
			}
		}

		if item == nil {
			// Device not found in cache, create minimal item with MAC
			logging.Warnf("Device with MAC %s not found in cache", mac)
			item = &vendors.InventoryItem{
				MAC:  mac,
				Type: deviceType,
			}
		}

		items = append(items, item)
	}

	return items
}

// ListInventory shows inventory items of a specific type or all items, optionally filtered by site
func ListInventory(ctx context.Context, client api.Client, cfg *config.Config, deviceType string, siteName string, formatOverride string, showAllFields bool) error {
	if siteName != "" {
		logging.Infof("Showing inventory items of type: %s for site: %s", deviceType, siteName)
	} else {
		logging.Infof("Showing inventory items of type: %s", deviceType)
	}

	// Ensure the site cache is populated before getting inventory
	// This is needed so that site_name can be correctly mapped from site_id
	err := api.EnsureSiteCache(ctx, cfg.API.Credentials.OrgID)
	if err != nil {
		// Log the error but continue, as we can still show inventory without site names
		logging.Warnf("Failed to load site information: %v", err)
		errMsg := fmt.Sprintf("Warning: Failed to load site information: %v", err)
		utils.PrintTextWithWarning(errMsg)
	}

	var inventory []*vendors.InventoryItem

	// Get inventory file path from Viper (since cfg.Files.Inventory might not be populated)
	inventoryPath := viper.GetString("files.inventory")
	if inventoryPath == "" {
		inventoryPath = cfg.Files.Inventory // Fallback to config struct
	}

	if inventoryPath != "" {
		// Use the local inventory file which contains MAC addresses
		logging.Infof("Using local inventory file: %s", inventoryPath)
		localInventory, err := LoadLocalInventory(inventoryPath)
		if err != nil {
			logging.Errorf("Failed to load local inventory: %v", err)
			return fmt.Errorf("failed to load local inventory: %w", err)
		}

		// Convert local inventory to vendors inventory format based on device type
		if deviceType == "" {
			// Get all device types
			inventory = append(inventory, ConvertLocalInventoryToVendors(localInventory.Config.Inventory.AP, "ap")...)
			inventory = append(inventory, ConvertLocalInventoryToVendors(localInventory.Config.Inventory.Switch, "switch")...)
			inventory = append(inventory, ConvertLocalInventoryToVendors(localInventory.Config.Inventory.Gateway, "gateway")...)
		} else {
			// Get specific device type
			switch strings.ToLower(deviceType) {
			case "ap":
				inventory = ConvertLocalInventoryToVendors(localInventory.Config.Inventory.AP, "ap")
			case "switch":
				inventory = ConvertLocalInventoryToVendors(localInventory.Config.Inventory.Switch, "switch")
			case "gateway":
				inventory = ConvertLocalInventoryToVendors(localInventory.Config.Inventory.Gateway, "gateway")
			default:
				logging.Errorf("Unknown device type: %s", deviceType)
				return fmt.Errorf("unknown device type: %s", deviceType)
			}
		}

	} else {
		// Fall back to getting inventory from cache
		logging.Infof("No local inventory file specified, fetching from cache")

		cacheAccessor, err := cmdutils.GetCacheAccessor()
		if err != nil {
			logging.Errorf("Failed to access cache: %v", err)
			return fmt.Errorf("failed to access cache: %w", err)
		}

		// Get devices from cache based on device type
		if deviceType == "" || strings.ToLower(deviceType) == "all" {
			// Get all device types from cache
			inventory = append(inventory, getInventoryFromCache(cacheAccessor, "ap")...)
			inventory = append(inventory, getInventoryFromCache(cacheAccessor, "switch")...)
			inventory = append(inventory, getInventoryFromCache(cacheAccessor, "gateway")...)
		} else {
			// Get specific device type from cache
			inventory = getInventoryFromCache(cacheAccessor, strings.ToLower(deviceType))
		}

		logging.Infof("Retrieved %d inventory items from cache", len(inventory))
	}

	// Enhance inventory items to ensure all required fields are set
	inventory = EnhanceInventoryItems(inventory)

	// Filter by site if specified
	if siteName != "" {
		// Get site ID from site name
		siteInfo, err := client.GetSiteByName(ctx, siteName, cfg.API.Credentials.OrgID)
		if err != nil {
			logging.Errorf("Failed to find site: %s", siteName)
			return fmt.Errorf("failed to find site '%s': %w", siteName, err)
		}

		if siteInfo == nil || siteInfo.ID == nil {
			logging.Errorf("Site not found: %s", siteName)
			return fmt.Errorf("site not found: %s", siteName)
		}

		// Filter inventory items by site ID
		var filteredInventory []*vendors.InventoryItem
		for _, item := range inventory {
			if item.SiteID != "" && item.SiteID == *siteInfo.ID {
				filteredInventory = append(filteredInventory, item)
			}
		}
		inventory = filteredInventory
		logging.Infof("Filtered to %d items for site %s", len(inventory), siteName)
	}

	// Sort inventory items by type and name
	sort.Slice(inventory, func(i, j int) bool {
		if inventory[i].Type != inventory[j].Type {
			return inventory[i].Type < inventory[j].Type
		}
		return inventory[i].Name < inventory[j].Name
	})
	sortedInventory := inventory

	// Build the title based on the device type and site, including count
	title := fmt.Sprintf("All Inventory Items (%d)", len(sortedInventory))
	if deviceType != "" && siteName != "" {
		title = fmt.Sprintf("%s Inventory Items for %s (%d)", strings.ToUpper(deviceType), siteName, len(sortedInventory))
	} else if deviceType != "" {
		title = fmt.Sprintf("%s Inventory Items (%d)", strings.ToUpper(deviceType), len(sortedInventory))
	} else if siteName != "" {
		title = fmt.Sprintf("All Inventory Items for %s (%d)", siteName, len(sortedInventory))
	}

	// Convert inventory items to map for the table formatter
	tableData := make([]formatter.GenericTableData, 0, len(sortedInventory))
	for _, item := range sortedInventory {
		itemData := formatter.GenericTableData{}

		// Name
		if item.Name != "" {
			itemData["name"] = item.Name
		} else {
			itemData["name"] = "<undefined>"
		}

		// Type
		itemData["type"] = item.Type

		// MAC
		itemData["mac"] = item.MAC

		// Serial
		itemData["serial"] = item.Serial

		// Model
		itemData["model"] = item.Model

		// Site ID and name
		if item.SiteID != "" {
			itemData["site_id"] = item.SiteID

			// Try to get site name from cache accessor
			if item.SiteName != "" {
				itemData["site_name"] = item.SiteName
			} else if cacheAccessor, err := cmdutils.GetCacheAccessor(); err == nil {
				if site, err := cacheAccessor.GetSiteByID(item.SiteID); err == nil && site.Name != "" {
					itemData["site_name"] = site.Name
				} else {
					itemData["site_name"] = ""
				}
			} else {
				itemData["site_name"] = ""
			}
		} else {
			itemData["site_id"] = ""
			itemData["site_name"] = ""
		}

		// Connected status - vendors.InventoryItem doesn't have this field
		// Default to unknown since we don't have connection status in inventory
		itemData["connected"] = "?"

		tableData = append(tableData, itemData)
	}

	// Apply field resolution to all table data
	if err := cmdutils.ApplyFieldResolution(tableData, true); err != nil {
		logging.Warnf("Failed to apply field resolution: %v", err)
	}

	if len(sortedInventory) == 0 {
		utils.PrintTextWithWarning(fmt.Sprintf("%s:", title))
		utils.PrintTextWithWarning("No inventory items found")
		return nil
	}

	// Create command path for config lookup
	commandPath := fmt.Sprintf("show.inventory.%s", deviceType)
	if deviceType == "" {
		commandPath = "show.inventory.all"
	}

	// Check if there's a command-specific format in the config using Viper
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
		logging.Warnf("Failed to initialize cache accessor: %v", err)
		cacheAccessor = nil
	}

	// Create the table configuration
	tableConfig := formatter.TableConfig{
		Title:         title,
		Format:        formatOverride, // Use override if provided
		BoldHeaders:   true,
		ShowSeparator: true,
		CommandPath:   commandPath,
		SiteLookup:    client,        // Pass client for site name lookups
		CacheAccess:   cacheAccessor, // Pass cache accessor for cache.* field lookups
		ShowAllFields: showAllFields, // Enable showing all cache fields when true
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
			{Field: "type", Title: "Type", MaxWidth: 0},
			{Field: "mac", Title: "MAC", MaxWidth: 0},
			{Field: "serial", Title: "Serial", MaxWidth: 0},
			{Field: "model", Title: "Model", MaxWidth: 0},
			{Field: "connected", Title: "Connected", MaxWidth: 0, IsBoolField: true, IsConnectionField: true},
			{Field: "site_name", Title: "Site", MaxWidth: 0},
		}
	}

	// Print the table
	fmt.Print(printer.Print())

	return nil
}

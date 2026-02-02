package show

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ravinald/wifimgr/internal/cmdutils"
	"github.com/ravinald/wifimgr/internal/formatter"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/patterns"
	"github.com/ravinald/wifimgr/internal/vendors"
)

var apiConfigCmd = &cobra.Command{
	Use:   "config [<device-name>|<mac>] [site <site-name>] [json|csv] [all] [no-resolve]",
	Short: "Show device configurations from API cache",
	Long: `Display device configurations from API cache for AP, Switch, and Gateway devices.

This command shows the full device configurations as returned by the Mist API.
It displays configurations stored in the cache during the refresh process.

Arguments:
  1. Filter (optional) - Device name or MAC address to filter by
  2. site keyword + site-name (optional) - Filter by site using "site" keyword  
  3. format (optional) - Output format: "json", "csv", or default table
  4. all (optional) - Show all fields (JSON format only)
  5. no-resolve (optional) - Disable field ID to name resolution

Examples:
  wifimgr show api config                        # Show all configs
  wifimgr show api config AP-NAME               # Show config for AP
  wifimgr show api config site US-LAB-01       # Show site configs
  wifimgr show api config AP-NAME json         # Show as JSON
  wifimgr show api config site US-LAB-01 csv  # Show as CSV`,
	Args: cobra.ArbitraryArgs,
	Run:  runShowAPIConfig,
}

// GetAPIConfigCmd returns the api config command for registration
func GetAPIConfigCmd() *cobra.Command {
	return apiConfigCmd
}

func runShowAPIConfig(_ *cobra.Command, args []string) {
	logger := logging.GetLogger()
	logger.Info("Executing show api config command")

	// Parse arguments using the centralized parser
	parsedArgs, err := cmdutils.ParseShowArgs(args)
	if err != nil {
		logger.WithError(err).Error("Failed to parse arguments")
		return
	}

	// Get cache accessor
	cacheAccessor, err := cmdutils.GetCacheAccessor()
	if err != nil {
		logger.WithError(err).Error("Failed to get cache accessor")
		return
	}

	// Collect all device configs
	var allConfigs []interface{}

	// Get AP configs
	apConfigs := cacheAccessor.GetAllAPConfigs()
	for _, config := range apConfigs {
		allConfigs = append(allConfigs, config)
	}

	// Get Switch configs
	switchConfigs := cacheAccessor.GetAllSwitchConfigs()
	for _, config := range switchConfigs {
		allConfigs = append(allConfigs, config)
	}

	// Get Gateway configs
	gatewayConfigs := cacheAccessor.GetAllGatewayConfigs()
	for _, config := range gatewayConfigs {
		allConfigs = append(allConfigs, config)
	}

	if len(allConfigs) == 0 {
		fmt.Println("No device configurations found in cache")
		return
	}

	// Apply filters
	filteredConfigs := filterAPIConfigs(allConfigs, parsedArgs, cacheAccessor)

	if len(filteredConfigs) == 0 {
		fmt.Println("No device configurations match the specified criteria")
		return
	}

	// Output based on format
	switch parsedArgs.Format {
	case "json":
		outputAPIConfigsJSON(filteredConfigs, parsedArgs)
	case "csv":
		outputAPIConfigsCSV(filteredConfigs, parsedArgs, cacheAccessor)
	default:
		outputAPIConfigsTable(filteredConfigs, parsedArgs, cacheAccessor)
	}
}

func filterAPIConfigs(configs []interface{}, args *cmdutils.ParsedShowArgs, accessor *vendors.CacheAccessor) []interface{} {
	var filtered []interface{}

	for _, config := range configs {
		// Type switch to handle different config types
		var deviceName, deviceMAC, siteID string

		switch c := config.(type) {
		case *vendors.APConfig:
			deviceName = c.Name
			deviceMAC = c.MAC
			siteID = c.SiteID
		case *vendors.SwitchConfig:
			deviceName = c.Name
			deviceMAC = c.MAC
			siteID = c.SiteID
		case *vendors.GatewayConfig:
			deviceName = c.Name
			deviceMAC = c.MAC
			siteID = c.SiteID
		default:
			continue
		}

		// Apply device filter
		if args.Filter != "" {
			if !patterns.Contains(deviceName, args.Filter) &&
				!patterns.Contains(deviceMAC, args.Filter) {
				continue
			}
		}

		// Apply site filter
		if args.SiteName != "" {
			if siteID == "" {
				continue
			}

			// Get site info to match by name
			site, err := accessor.GetSiteByID(siteID)
			if err != nil || site.Name == "" {
				continue
			}

			if !patterns.Equals(site.Name, args.SiteName) {
				continue
			}
		}

		filtered = append(filtered, config)
	}

	return filtered
}

func outputAPIConfigsJSON(configs []interface{}, _ *cmdutils.ParsedShowArgs) {
	for i, config := range configs {
		if i > 0 {
			fmt.Println() // Add blank line between configs
		}

		// Convert to map for flexible output
		var output map[string]interface{}

		switch c := config.(type) {
		case *vendors.APConfig:
			output = configToMap(c.Name, c.MAC, c.SiteID, "ap", c.Config)
		case *vendors.SwitchConfig:
			output = configToMap(c.Name, c.MAC, c.SiteID, "switch", c.Config)
		case *vendors.GatewayConfig:
			output = configToMap(c.Name, c.MAC, c.SiteID, "gateway", c.Config)
		}

		if output != nil {
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(output); err != nil {
				logging.GetLogger().WithError(err).Error("Failed to encode JSON output")
			}
		}
	}
}

// configToMap creates a unified map representation of a config
func configToMap(name, mac, siteID, deviceType string, config map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	result["name"] = name
	result["mac"] = mac
	result["site_id"] = siteID
	result["type"] = deviceType

	// Include all config fields
	for k, v := range config {
		result[k] = v
	}

	return result
}

func outputAPIConfigsCSV(configs []interface{}, args *cmdutils.ParsedShowArgs, accessor *vendors.CacheAccessor) {
	// Create table data with more detailed fields for API view
	headers := []string{"Type", "Name", "MAC", "Site", "Model", "Serial", "IP Config", "Profile ID", "Created", "Modified"}
	var rows [][]string

	for _, config := range configs {
		var row []string

		switch c := config.(type) {
		case *vendors.APConfig:
			siteName := c.SiteID
			if c.SiteID != "" && !args.NoResolve {
				if site, err := accessor.GetSiteByID(c.SiteID); err == nil && site.Name != "" {
					siteName = site.Name
				}
			}

			row = []string{
				"AP",
				c.Name,
				c.MAC,
				siteName,
				getConfigString(c.Config, "model"),
				getConfigString(c.Config, "serial"),
				formatIPConfigFromMap(c.Config),
				getConfigString(c.Config, "deviceprofile_id"),
				formatTimestampFromConfig(c.Config, "created_time"),
				formatTimestampFromConfig(c.Config, "modified_time"),
			}

		case *vendors.SwitchConfig:
			siteName := c.SiteID
			if c.SiteID != "" && !args.NoResolve {
				if site, err := accessor.GetSiteByID(c.SiteID); err == nil && site.Name != "" {
					siteName = site.Name
				}
			}

			row = []string{
				"Switch",
				c.Name,
				c.MAC,
				siteName,
				getConfigString(c.Config, "model"),
				getConfigString(c.Config, "serial"),
				formatIPConfigFromMap(c.Config),
				getConfigString(c.Config, "deviceprofile_id"),
				formatTimestampFromConfig(c.Config, "created_time"),
				formatTimestampFromConfig(c.Config, "modified_time"),
			}

		case *vendors.GatewayConfig:
			siteName := c.SiteID
			if c.SiteID != "" && !args.NoResolve {
				if site, err := accessor.GetSiteByID(c.SiteID); err == nil && site.Name != "" {
					siteName = site.Name
				}
			}

			row = []string{
				"Gateway",
				c.Name,
				c.MAC,
				siteName,
				getConfigString(c.Config, "model"),
				getConfigString(c.Config, "serial"),
				formatGatewayIPConfigsFromMap(c.Config),
				getConfigString(c.Config, "deviceprofile_id"),
				formatTimestampFromConfig(c.Config, "created_time"),
				formatTimestampFromConfig(c.Config, "modified_time"),
			}
		}

		if row != nil {
			rows = append(rows, row)
		}
	}

	// Use GenericTablePrinter for CSV output
	var tableData []formatter.GenericTableData
	for _, row := range rows {
		data := make(map[string]interface{})
		for i, header := range headers {
			data[header] = row[i]
		}
		tableData = append(tableData, formatter.GenericTableData(data))
	}

	columns := make([]formatter.TableColumn, len(headers))
	for i, header := range headers {
		columns[i] = formatter.TableColumn{
			Field: header,
			Title: header,
		}
	}

	tableConfig := formatter.TableConfig{
		Columns: columns,
		Format:  "csv",
	}

	printer := formatter.NewGenericTablePrinter(tableConfig, tableData)
	fmt.Print(printer.Print())
}

func outputAPIConfigsTable(configs []interface{}, args *cmdutils.ParsedShowArgs, accessor *vendors.CacheAccessor) {
	// Create table data
	var tableData []formatter.GenericTableData

	for _, config := range configs {
		row := make(map[string]interface{})

		switch c := config.(type) {
		case *vendors.APConfig:
			row["type"] = "AP"
			row["name"] = c.Name
			row["mac"] = c.MAC
			row["model"] = getConfigString(c.Config, "model")
			row["serial"] = getConfigString(c.Config, "serial")
			row["ip_config"] = formatIPConfigFromMap(c.Config)
			row["profile"] = getConfigString(c.Config, "deviceprofile_id")
			row["modified"] = formatTimestampFromConfig(c.Config, "modified_time")

			// Resolve site name
			if c.SiteID != "" && !args.NoResolve {
				if site, err := accessor.GetSiteByID(c.SiteID); err == nil && site.Name != "" {
					row["site_name"] = site.Name
				} else {
					row["site_name"] = c.SiteID
				}
			} else {
				row["site_name"] = c.SiteID
			}

		case *vendors.SwitchConfig:
			row["type"] = "Switch"
			row["name"] = c.Name
			row["mac"] = c.MAC
			row["model"] = getConfigString(c.Config, "model")
			row["serial"] = getConfigString(c.Config, "serial")
			row["ip_config"] = formatIPConfigFromMap(c.Config)
			row["profile"] = getConfigString(c.Config, "deviceprofile_id")
			row["modified"] = formatTimestampFromConfig(c.Config, "modified_time")

			// Resolve site name
			if c.SiteID != "" && !args.NoResolve {
				if site, err := accessor.GetSiteByID(c.SiteID); err == nil && site.Name != "" {
					row["site_name"] = site.Name
				} else {
					row["site_name"] = c.SiteID
				}
			} else {
				row["site_name"] = c.SiteID
			}

		case *vendors.GatewayConfig:
			row["type"] = "Gateway"
			row["name"] = c.Name
			row["mac"] = c.MAC
			row["model"] = getConfigString(c.Config, "model")
			row["serial"] = getConfigString(c.Config, "serial")
			row["ip_config"] = formatGatewayIPConfigsFromMap(c.Config)
			row["profile"] = getConfigString(c.Config, "deviceprofile_id")
			row["modified"] = formatTimestampFromConfig(c.Config, "modified_time")

			// Resolve site name
			if c.SiteID != "" && !args.NoResolve {
				if site, err := accessor.GetSiteByID(c.SiteID); err == nil && site.Name != "" {
					row["site_name"] = site.Name
				} else {
					row["site_name"] = c.SiteID
				}
			} else {
				row["site_name"] = c.SiteID
			}
		}

		if len(row) > 0 {
			tableData = append(tableData, formatter.GenericTableData(row))
		}
	}

	// Define columns
	columns := []formatter.TableColumn{
		{Field: "type", Title: "Type"},
		{Field: "name", Title: "Name"},
		{Field: "mac", Title: "MAC"},
		{Field: "site_name", Title: "Site"},
		{Field: "model", Title: "Model"},
		{Field: "serial", Title: "Serial"},
		{Field: "ip_config", Title: "IP Config"},
		{Field: "profile", Title: "Profile"},
		{Field: "modified", Title: "Modified"},
	}

	// Create table config
	tableConfig := formatter.TableConfig{
		Title:       fmt.Sprintf("API Device Configurations (%d)", len(tableData)),
		Columns:     columns,
		Format:      "table",
		BoldHeaders: true,
	}

	// Print table
	printer := formatter.NewGenericTablePrinter(tableConfig, tableData)
	fmt.Print(printer.Print())
}

// Helper functions for formatting complex fields

// getConfigString extracts a string value from a config map
func getConfigString(config map[string]interface{}, key string) string {
	if config == nil {
		return ""
	}
	if val, ok := config[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

// formatIPConfigFromMap extracts and formats IP config from a config map
func formatIPConfigFromMap(config map[string]interface{}) string {
	if config == nil {
		return ""
	}

	// Try to get ip_config directly
	ipConfig, ok := config["ip_config"].(map[string]interface{})
	if !ok {
		return ""
	}

	var parts []string

	if ip, ok := ipConfig["ip"].(string); ok && ip != "" {
		parts = append(parts, ip)
	}

	if configType, ok := ipConfig["type"].(string); ok && configType != "" {
		parts = append(parts, fmt.Sprintf("(%s)", configType))
	}

	if vlan, ok := ipConfig["vlan_id"]; ok {
		parts = append(parts, fmt.Sprintf("VLAN:%v", vlan))
	}

	return strings.Join(parts, " ")
}

// formatGatewayIPConfigsFromMap extracts and formats gateway IP configs from a config map
func formatGatewayIPConfigsFromMap(config map[string]interface{}) string {
	if config == nil {
		return ""
	}

	portConfig, ok := config["port_config"].(map[string]interface{})
	if !ok {
		return ""
	}

	var configs []string

	// For gateways, look for IP in port config
	for portName, portCfg := range portConfig {
		if configMap, ok := portCfg.(map[string]interface{}); ok {
			if ipConfig, ok := configMap["ip_config"].(map[string]interface{}); ok {
				if ip, ok := ipConfig["ip"].(string); ok && ip != "" {
					configs = append(configs, fmt.Sprintf("%s:%s", portName, ip))
				}
			}
		}
	}

	return strings.Join(configs, ", ")
}

// formatTimestampFromConfig extracts and formats a timestamp from a config map
func formatTimestampFromConfig(config map[string]interface{}, key string) string {
	if config == nil {
		return ""
	}

	val, ok := config[key]
	if !ok {
		return ""
	}

	// Handle different numeric types
	var ts int64
	switch v := val.(type) {
	case int64:
		ts = v
	case float64:
		ts = int64(v)
	case int:
		ts = int64(v)
	default:
		return ""
	}

	if ts == 0 {
		return ""
	}

	t := time.Unix(ts, 0)
	return t.Format("2006-01-02 15:04")
}

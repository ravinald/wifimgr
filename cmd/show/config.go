package show

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ravinald/wifimgr/internal/cmdutils"
	"github.com/ravinald/wifimgr/internal/formatter"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/patterns"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// configCmd represents the show config command
var configCmd = &cobra.Command{
	Use:   "config [<device-name>|<mac>] [site <site-name>] [json|csv] [all] [no-resolve]",
	Short: "Show device configurations from cache",
	Long: `Display device configurations from cache for AP, Switch, and Gateway devices.

This command shows the device configurations that have been cached. It requires
that device configs have been fetched during cache refresh.

Arguments:
  1. Filter (optional) - Device name or MAC address to filter by
  2. site keyword + site-name (optional) - Filter by site using "site" keyword  
  3. format (optional) - Output format: "json", "csv", or default table
  4. all (optional) - Show all fields (JSON format only)
  5. no-resolve (optional) - Disable field ID to name resolution

Examples:
  wifimgr show config                        # Show all device configs
  wifimgr show config AP-NAME               # Show config for specific AP
  wifimgr show config site US-LAB-01       # Show configs for site
  wifimgr show config AP-NAME json         # Show in JSON format
  wifimgr show config site US-LAB-01 csv  # Show site configs as CSV`,
	Args: cobra.ArbitraryArgs,
	Run:  runShowConfig,
}

// GetConfigCmd returns the config command for registration
func GetConfigCmd() *cobra.Command {
	return configCmd
}

func runShowConfig(_ *cobra.Command, args []string) {
	logger := logging.GetLogger()
	logger.Info("Executing show config command")

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
	filteredConfigs := filterConfigs(allConfigs, parsedArgs, cacheAccessor)

	if len(filteredConfigs) == 0 {
		fmt.Println("No device configurations match the specified criteria")
		return
	}

	// Output based on format
	switch parsedArgs.Format {
	case "json":
		outputConfigsJSON(filteredConfigs, parsedArgs)
	case "csv":
		outputConfigsCSV(filteredConfigs)
	default:
		outputConfigsTable(filteredConfigs, parsedArgs, cacheAccessor)
	}
}

func filterConfigs(configs []interface{}, args *cmdutils.ParsedShowArgs, accessor *vendors.CacheAccessor) []interface{} {
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

func outputConfigsJSON(configs []interface{}, _ *cmdutils.ParsedShowArgs) {
	// Create output structure
	output := struct {
		Count   int           `json:"count"`
		Configs []interface{} `json:"configs"`
	}{
		Count:   len(configs),
		Configs: configs,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		logging.GetLogger().WithError(err).Error("Failed to encode JSON output")
	}
}

func outputConfigsCSV(configs []interface{}) {
	// Create table data
	headers := []string{"Type", "Name", "MAC", "Site ID", "Model", "Serial", "IP"}
	var rows [][]string

	for _, config := range configs {
		var row []string

		switch c := config.(type) {
		case *vendors.APConfig:
			row = []string{
				"AP",
				c.Name,
				c.MAC,
				c.SiteID,
				getConfigString(c.Config, "model"),
				getConfigString(c.Config, "serial"),
				getIPFromConfigMap(c.Config),
			}
		case *vendors.SwitchConfig:
			row = []string{
				"Switch",
				c.Name,
				c.MAC,
				c.SiteID,
				getConfigString(c.Config, "model"),
				getConfigString(c.Config, "serial"),
				getIPFromConfigMap(c.Config),
			}
		case *vendors.GatewayConfig:
			row = []string{
				"Gateway",
				c.Name,
				c.MAC,
				c.SiteID,
				getConfigString(c.Config, "model"),
				getConfigString(c.Config, "serial"),
				getIPFromPortConfigMap(c.Config),
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

func outputConfigsTable(configs []interface{}, args *cmdutils.ParsedShowArgs, accessor *vendors.CacheAccessor) {
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
			row["ip"] = getIPFromConfigMap(c.Config)
			row["vlan"] = getVLANFromConfigMap(c.Config)

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
			row["ip"] = getIPFromConfigMap(c.Config)
			row["vlan"] = getVLANFromConfigMap(c.Config)

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
			row["ip"] = getIPFromPortConfigMap(c.Config)
			row["vlan"] = ""

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
		{Field: "ip", Title: "IP"},
		{Field: "vlan", Title: "VLAN"},
	}

	// Create table config
	tableConfig := formatter.TableConfig{
		Title:       fmt.Sprintf("Device Configurations (%d)", len(tableData)),
		Columns:     columns,
		Format:      "table",
		BoldHeaders: true,
	}

	// Print table
	printer := formatter.NewGenericTablePrinter(tableConfig, tableData)
	fmt.Print(printer.Print())
}

// Helper functions to extract data from vendors config maps

// getIPFromConfigMap extracts IP from a device config's ip_config field
func getIPFromConfigMap(config map[string]interface{}) string {
	if config == nil {
		return ""
	}

	ipConfig, ok := config["ip_config"].(map[string]interface{})
	if !ok {
		return ""
	}

	if ip, ok := ipConfig["ip"].(string); ok {
		return ip
	}

	return ""
}

// getIPFromPortConfigMap extracts IP from a gateway's port_config field
func getIPFromPortConfigMap(config map[string]interface{}) string {
	if config == nil {
		return ""
	}

	portConfig, ok := config["port_config"].(map[string]interface{})
	if !ok {
		return ""
	}

	// For gateways, look for IP in port config
	for _, portCfg := range portConfig {
		if configMap, ok := portCfg.(map[string]interface{}); ok {
			if ipConfig, ok := configMap["ip_config"].(map[string]interface{}); ok {
				if ip, ok := ipConfig["ip"].(string); ok && ip != "" {
					return ip
				}
			}
		}
	}

	return ""
}

// getVLANFromConfigMap extracts VLAN from a device config's ip_config field
func getVLANFromConfigMap(config map[string]interface{}) string {
	if config == nil {
		return ""
	}

	ipConfig, ok := config["ip_config"].(map[string]interface{})
	if !ok {
		return ""
	}

	if vlan, ok := ipConfig["vlan_id"]; ok {
		return fmt.Sprintf("%v", vlan)
	}

	return ""
}

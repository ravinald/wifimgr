/*
Copyright © 2025 Ravi Pina <ravi@pina.org>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ravinald/jsondiff/pkg/jsondiff"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ravinald/wifimgr/internal/cmdutils"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/vendors"
	"github.com/ravinald/wifimgr/internal/vendors/meraki"
	"github.com/ravinald/wifimgr/internal/vendors/mist"
	"github.com/ravinald/wifimgr/internal/xdg"
)

// ImportScope defines what data to import
type ImportScope string

const (
	ScopeFull     ImportScope = "full"
	ScopeWLANs    ImportScope = "wlans"
	ScopeProfiles ImportScope = "profiles"
	ScopeAP       ImportScope = "ap"
	ScopeSwitch   ImportScope = "switch"
	ScopeGateway  ImportScope = "gateway"
)

// importAPISiteCmd represents the "import api site" command
var importAPISiteCmd = &cobra.Command{
	Use:   "site [api <api-label>] <site-name> [full|type <scope>] [secrets] [compare] [save] [file <filename>]",
	Short: "Import site configuration from API cache",
	Long: `Import site configuration from the API cache.

By default, this command outputs the configuration to STDOUT for preview.
Use the 'save' argument to write to a local config file.

Basic Usage:
  wifimgr import api site US-SFO-LAB
  wifimgr import api site US-SFO-LAB save
  wifimgr import api site US-SFO-LAB type ap
  wifimgr import api site US-SFO-LAB compare

With Explicit API (when site exists in multiple APIs):
  wifimgr import api site api mist-prod US-SFO-LAB
  wifimgr import api site api meraki US-SFO-LAB save

With Custom Output File:
  wifimgr import api site US-SFO-LAB save file custom.json
  wifimgr import api site US-SFO-LAB save file sites/lab.json
  wifimgr import api site US-SFO-LAB save file /tmp/site.json

Combined Options:
  wifimgr import api site api mist-prod US-SFO-LAB save file sites/sfo.json
  wifimgr import api site US-SFO-LAB type wlans secrets save
  wifimgr import api site US-SFO-LAB | jq '.config'

Arguments:
  api            Optional. Keyword followed by API label to target specific API
  site-name      Required. The site name to import
  full           Optional. Import full site configuration (default)
  type <scope>   Optional. Limit import to specific scope:
                   - wlans     WLAN/SSID configurations only
                   - profiles  Site-specific device profiles only
                   - ap        Access point configurations only
                   - switch    Switch configurations only
                   - gateway   Gateway/firewall configurations only
  secrets        Optional. Include sensitive data (PSK, RADIUS secrets) - redacted by default
  compare        Optional. Compare API state with existing config file (using jsondiff)
  save           Optional. Write to config file (default: print to STDOUT)
  file           Optional. Keyword followed by output filename (relative to config_dir or absolute)

What it Does:
  1. Retrieves site configuration from the specified API cache
  2. Optionally filters to specific scope (wlans, profiles, ap, switch, gateway)
  3. Redacts secrets by default (use 'secrets' to include)
  4. Prints to STDOUT or saves to file if 'save' specified
  5. With 'compare': shows diff between API and existing local config

Output Location:
  Without 'save': Prints JSON to STDOUT
  With 'save' (no file): ~/.config/wifimgr/<api-name>/sites/<site-name>.json
  With 'save file': ~/.config/wifimgr/<filename> (relative) or <filename> (absolute)`,
	Args: func(cmd *cobra.Command, args []string) error {
		// Allow "help" as a special keyword
		for _, arg := range args {
			if strings.ToLower(arg) == "help" {
				return nil
			}
		}
		if len(args) < 1 {
			return fmt.Errorf("requires at least 1 arg(s), only received %d", len(args))
		}
		return nil
	},
	RunE: runImportAPISite,
}

func init() {
	importAPICmd.AddCommand(importAPISiteCmd)
}

// importSiteArgs holds parsed arguments for the import api site command
type importSiteArgs struct {
	apiLabel       string
	siteName       string
	scope          ImportScope
	includeSecrets bool
	compareMode    bool
	saveMode       bool
	outputFile     string
}

// parseImportSiteArgs parses positional arguments for import api site command
func parseImportSiteArgs(args []string) (*importSiteArgs, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("site name required")
	}

	result := &importSiteArgs{
		scope: ScopeFull,
	}

	i := 0

	// Check for optional "api <api-label>" prefix
	if i < len(args) && strings.ToLower(args[i]) == "api" {
		if i+1 >= len(args) {
			return nil, fmt.Errorf("'api' requires an API label")
		}
		result.apiLabel = args[i+1]
		i += 2 // Skip "api" and the label
	}

	// Next argument must be the site name
	if i >= len(args) {
		return nil, fmt.Errorf("site name required")
	}
	result.siteName = args[i]
	i++

	// Parse remaining arguments
	for i < len(args) {
		arg := strings.ToLower(args[i])
		switch arg {
		case "full":
			result.scope = ScopeFull
		case "type":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("'type' requires a scope (wlans, profiles, ap, switch, gateway)")
			}
			scopeArg := strings.ToLower(args[i+1])
			switch scopeArg {
			case "wlans":
				result.scope = ScopeWLANs
			case "profiles":
				result.scope = ScopeProfiles
			case "ap":
				result.scope = ScopeAP
			case "switch":
				result.scope = ScopeSwitch
			case "gateway":
				result.scope = ScopeGateway
			default:
				return nil, fmt.Errorf("invalid scope '%s' - must be one of: wlans, profiles, ap, switch, gateway", scopeArg)
			}
			i++ // Skip the scope value
		case "secrets":
			result.includeSecrets = true
		case "compare":
			result.compareMode = true
		case "save":
			result.saveMode = true
		case "file":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("'file' requires a filename")
			}
			result.outputFile = args[i+1]
			i++ // Skip the filename
		case "help":
			// Already handled in RunE
		default:
			return nil, fmt.Errorf("unknown argument: %s", args[i])
		}
		i++
	}

	// Validate combinations
	if result.outputFile != "" && !result.saveMode {
		return nil, fmt.Errorf("'file' requires 'save' to be specified")
	}

	return result, nil
}

func runImportAPISite(cmd *cobra.Command, args []string) error {
	// Check for help keyword in positional arguments
	for _, arg := range args {
		if strings.ToLower(arg) == "help" {
			return cmd.Help()
		}
	}

	logger := logging.GetLogger()
	logger.Info("Executing import api site command")

	// Parse arguments
	parsed, err := parseImportSiteArgs(args)
	if err != nil {
		return err
	}

	siteName := parsed.siteName

	// Get cache accessor
	cacheAccessor, err := cmdutils.GetCacheAccessor()
	if err != nil {
		return fmt.Errorf("failed to get cache accessor: %w", err)
	}

	// Find the site, optionally filtered by API label
	var site *vendors.SiteInfo
	if parsed.apiLabel != "" {
		// Explicit API specified in positional args
		site, err = cacheAccessor.GetSiteByNameAndAPI(siteName, parsed.apiLabel)
		if err != nil {
			return fmt.Errorf("site '%s' not found in API '%s'", siteName, parsed.apiLabel)
		}
	} else {
		// No API specified - find site in any API
		site, err = cacheAccessor.GetSiteByName(siteName)
		if err != nil {
			return fmt.Errorf("site not found: %s", siteName)
		}
	}

	// Determine the API label to use
	apiLabel := site.SourceAPI

	logger.Infof("Importing site '%s' from API '%s' (scope: %s, save: %v)", siteName, apiLabel, parsed.scope, parsed.saveMode)

	// Build the export data
	exportData, err := buildSiteExportData(cacheAccessor, site, parsed.scope, parsed.includeSecrets)
	if err != nil {
		return fmt.Errorf("failed to build export data: %w", err)
	}

	// Determine output path
	var outputPath string
	if parsed.outputFile != "" {
		// Custom output file specified
		if filepath.IsAbs(parsed.outputFile) {
			// Absolute path - use as-is
			outputPath = parsed.outputFile
		} else {
			// Relative path - join with config_dir
			configDir := viper.GetString("files.config_dir")
			outputPath = filepath.Join(configDir, parsed.outputFile)
		}
	} else {
		// Default output path (XDG config directory)
		outputPath = filepath.Join(xdg.GetConfigDir(), apiLabel, "sites", siteName+".json")
	}

	// Check if file exists (needed for compare mode)
	existingData, fileExists := loadExistingConfig(outputPath)

	if parsed.compareMode {
		return compareSiteConfig(exportData, existingData, fileExists, outputPath, siteName)
	}

	// Without save mode, just print to STDOUT
	if !parsed.saveMode {
		jsonData, err := json.MarshalIndent(exportData, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal data: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Save mode: write to file
	// If file exists, ask for confirmation
	if fileExists {
		if !confirmOverwrite(outputPath) {
			fmt.Println("Import cancelled")
			return nil
		}
	}

	// Write the file
	if err := writeSiteConfig(outputPath, exportData); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Printf("Exported site configuration to: %s\n", outputPath)
	return nil
}

// SiteExportConfig represents the exported site configuration structure
type SiteExportConfig struct {
	Version int                  `json:"version"`
	Config  SiteExportConfigData `json:"config"`
}

// SiteExportConfigData wraps the sites map
type SiteExportConfigData struct {
	Sites map[string]*SiteConfigData `json:"sites"`
}

// SiteConfigData represents the configuration data for a single site
type SiteConfigData struct {
	API        string           `json:"api,omitempty"`
	SiteConfig map[string]any   `json:"site_config"`
	Devices    *DevicesConfig   `json:"devices,omitempty"`
	WLANs      []map[string]any `json:"wlans,omitempty"`
	Profiles   []map[string]any `json:"profiles,omitempty"`
}

// DevicesConfig holds device configurations by type
type DevicesConfig struct {
	AP      map[string]map[string]any `json:"ap,omitempty"`
	Switch  map[string]map[string]any `json:"switch,omitempty"`
	Gateway map[string]map[string]any `json:"gateway,omitempty"`
}

func buildSiteExportData(cacheAccessor *vendors.CacheAccessor, site *vendors.SiteInfo, scope ImportScope, includeSecrets bool) (*SiteExportConfig, error) {
	siteData := &SiteConfigData{
		API: site.SourceAPI,
		SiteConfig: map[string]any{
			"name":         site.Name,
			"address":      site.Address,
			"country_code": site.CountryCode,
			"timezone":     site.Timezone,
			"notes":        site.Notes,
		},
	}

	// Add location if available
	if site.Latitude != 0 || site.Longitude != 0 {
		siteData.SiteConfig["latlng"] = map[string]float64{
			"lat": site.Latitude,
			"lng": site.Longitude,
		}
	}

	// Add devices if scope includes them
	if scope == ScopeFull || scope == ScopeAP || scope == ScopeSwitch || scope == ScopeGateway {
		devices, err := buildDevicesExport(cacheAccessor, site.ID, site.SourceVendor, scope)
		if err != nil {
			logging.Warnf("Failed to build devices export: %v", err)
		} else {
			siteData.Devices = devices
		}
	}

	// Add WLANs if scope includes them
	if scope == ScopeFull || scope == ScopeWLANs {
		wlans := buildWLANsExport(cacheAccessor, site.ID, includeSecrets)
		if len(wlans) > 0 {
			siteData.WLANs = wlans
		}
	}

	// Add site-specific profiles if scope includes them
	if scope == ScopeFull || scope == ScopeProfiles {
		profiles := buildProfilesExport(cacheAccessor, site.ID)
		if len(profiles) > 0 {
			siteData.Profiles = profiles
		}
	}

	return &SiteExportConfig{
		Version: 1,
		Config: SiteExportConfigData{
			Sites: map[string]*SiteConfigData{
				site.Name: siteData,
			},
		},
	}, nil
}

func buildDevicesExport(cacheAccessor *vendors.CacheAccessor, siteID string, sourceVendor string, scope ImportScope) (*DevicesConfig, error) {
	devices := &DevicesConfig{
		AP:      make(map[string]map[string]any),
		Switch:  make(map[string]map[string]any),
		Gateway: make(map[string]map[string]any),
	}

	// Get AP configs for this site (if scope includes APs)
	if scope == ScopeFull || scope == ScopeAP {
		for _, cfg := range cacheAccessor.GetAllAPConfigs() {
			if cfg.SiteID != siteID {
				continue
			}
			mac := vendors.NormalizeMAC(cfg.MAC)

			// Use vendor-specific export function to create structured config
			var apConfig *vendors.APDeviceConfig
			if cfg.Config != nil {
				switch sourceVendor {
				case "mist":
					apConfig = mist.ExportAPConfig(cfg.Config)
				case "meraki":
					apConfig = meraki.ExportAPConfig(cfg.Config)
				default:
					// Fallback: use Mist converter as default
					apConfig = mist.ExportAPConfig(cfg.Config)
				}
			}

			// Build device data with structured config
			deviceData := map[string]any{
				"mac": cfg.MAC,
			}

			// Enrich with inventory data (model, serial)
			if inv, err := cacheAccessor.GetDeviceByMAC(mac); err == nil && inv != nil {
				if inv.Model != "" {
					deviceData["model"] = inv.Model
				}
				if inv.Serial != "" {
					deviceData["serial"] = inv.Serial
				}
			}

			// If we have a structured config, serialize it
			if apConfig != nil {
				// Name from structured config
				if apConfig.Name != "" {
					deviceData["name"] = apConfig.Name
				} else {
					deviceData["name"] = cfg.Name
				}

				// Notes
				if apConfig.Notes != "" {
					deviceData["notes"] = apConfig.Notes
				}

				// Tags
				if len(apConfig.Tags) > 0 {
					deviceData["tags"] = apConfig.Tags
				}

				// Location
				if len(apConfig.Location) > 0 {
					deviceData["location"] = apConfig.Location
				}
				if apConfig.MapID != "" {
					deviceData["map_id"] = apConfig.MapID
				}
				if apConfig.X != nil {
					deviceData["x"] = *apConfig.X
				}
				if apConfig.Y != nil {
					deviceData["y"] = *apConfig.Y
				}
				if apConfig.Height != nil {
					deviceData["height"] = *apConfig.Height
				}
				if apConfig.Orientation != nil {
					deviceData["orientation"] = *apConfig.Orientation
				}

				// Radio config
				if apConfig.RadioConfig != nil {
					deviceData["radio_config"] = apConfig.RadioConfig.ToMap()
				}

				// IP config
				if apConfig.IPConfig != nil {
					deviceData["ip_config"] = apConfig.IPConfig.ToMap()
				}

				// BLE config
				if apConfig.BLEConfig != nil {
					deviceData["ble_config"] = apConfig.BLEConfig.ToMap()
				}

				// Mesh config
				if apConfig.MeshConfig != nil {
					deviceData["mesh"] = apConfig.MeshConfig.ToMap()
				}

				// LED config
				if apConfig.LEDConfig != nil {
					deviceData["led"] = apConfig.LEDConfig.ToMap()
				}

				// Power config
				if apConfig.PowerConfig != nil {
					deviceData["pwr_config"] = apConfig.PowerConfig.ToMap()
				}

				// Hardware flags
				if apConfig.DisableEth1 != nil {
					deviceData["disable_eth1"] = *apConfig.DisableEth1
				}
				if apConfig.DisableEth2 != nil {
					deviceData["disable_eth2"] = *apConfig.DisableEth2
				}
				if apConfig.DisableEth3 != nil {
					deviceData["disable_eth3"] = *apConfig.DisableEth3
				}
				if apConfig.DisableModule != nil {
					deviceData["disable_module"] = *apConfig.DisableModule
				}
				if apConfig.PoEPassthrough != nil {
					deviceData["poe_passthrough"] = *apConfig.PoEPassthrough
				}

				// Device profile - resolve ID to name if possible
				if apConfig.DeviceProfileID != "" {
					if dp, err := cacheAccessor.GetDeviceProfileByID(apConfig.DeviceProfileID); err == nil {
						deviceData["deviceprofile_name"] = dp.Name
					} else {
						deviceData["deviceprofile_id"] = apConfig.DeviceProfileID
					}
				}

				// Variables
				if len(apConfig.Vars) > 0 {
					deviceData["vars"] = apConfig.Vars
				}

				// Vendor-specific extension blocks
				if len(apConfig.Mist) > 0 {
					deviceData["mist"] = apConfig.Mist
				}
				if len(apConfig.Meraki) > 0 {
					deviceData["meraki"] = apConfig.Meraki
				}
			} else {
				// Fallback to basic fields if no config
				deviceData["name"] = cfg.Name
			}

			devices.AP[mac] = deviceData
		}
	}

	// Get Switch configs for this site (if scope includes Switches)
	if scope == ScopeFull || scope == ScopeSwitch {
		for _, cfg := range cacheAccessor.GetAllSwitchConfigs() {
			if cfg.SiteID != siteID {
				continue
			}
			mac := vendors.NormalizeMAC(cfg.MAC)
			deviceData := map[string]any{
				"name": cfg.Name,
				"mac":  cfg.MAC,
			}

			// Enrich with inventory data (model, serial)
			if inv, err := cacheAccessor.GetDeviceByMAC(mac); err == nil && inv != nil {
				if inv.Model != "" {
					deviceData["model"] = inv.Model
				}
				if inv.Serial != "" {
					deviceData["serial"] = inv.Serial
				}
			}

			if cfg.Config != nil {
				if notes, ok := cfg.Config["notes"].(string); ok && notes != "" {
					deviceData["notes"] = notes
				}
				if dpID, ok := cfg.Config["deviceprofile_id"].(string); ok && dpID != "" {
					if dp, err := cacheAccessor.GetDeviceProfileByID(dpID); err == nil {
						deviceData["deviceprofile"] = dp.Name
					} else {
						deviceData["deviceprofile_id"] = dpID
					}
				}
				deviceData["config"] = cfg.Config
			}

			devices.Switch[mac] = deviceData
		}
	}

	// Get Gateway configs for this site (if scope includes Gateways)
	if scope == ScopeFull || scope == ScopeGateway {
		for _, cfg := range cacheAccessor.GetAllGatewayConfigs() {
			if cfg.SiteID != siteID {
				continue
			}
			mac := vendors.NormalizeMAC(cfg.MAC)
			deviceData := map[string]any{
				"name": cfg.Name,
				"mac":  cfg.MAC,
			}

			// Enrich with inventory data (model, serial)
			if inv, err := cacheAccessor.GetDeviceByMAC(mac); err == nil && inv != nil {
				if inv.Model != "" {
					deviceData["model"] = inv.Model
				}
				if inv.Serial != "" {
					deviceData["serial"] = inv.Serial
				}
			}

			if cfg.Config != nil {
				if notes, ok := cfg.Config["notes"].(string); ok && notes != "" {
					deviceData["notes"] = notes
				}
				if dpID, ok := cfg.Config["deviceprofile_id"].(string); ok && dpID != "" {
					if dp, err := cacheAccessor.GetDeviceProfileByID(dpID); err == nil {
						deviceData["deviceprofile"] = dp.Name
					} else {
						deviceData["deviceprofile_id"] = dpID
					}
				}
				deviceData["config"] = cfg.Config
			}

			devices.Gateway[mac] = deviceData
		}
	}

	return devices, nil
}

func buildWLANsExport(cacheAccessor *vendors.CacheAccessor, siteID string, includeSecrets bool) []map[string]any {
	var wlans []map[string]any

	for _, wlan := range cacheAccessor.GetWLANsBySite(siteID) {
		wlanData := map[string]any{
			"ssid":       wlan.SSID,
			"id":         wlan.ID,
			"enabled":    wlan.Enabled,
			"hidden":     wlan.Hidden,
			"auth_type":  wlan.AuthType,
			"encryption": wlan.EncryptionMode,
			"band":       wlan.Band,
		}

		if wlan.VLANID > 0 {
			wlanData["vlan_id"] = wlan.VLANID
		}

		// Include raw config if available
		if wlan.Config != nil {
			config := copyMap(wlan.Config)
			if !includeSecrets {
				// Redact sensitive data
				redactSecrets(config)
			}
			wlanData["config"] = config
		}

		wlans = append(wlans, wlanData)
	}

	return wlans
}

func buildProfilesExport(cacheAccessor *vendors.CacheAccessor, siteID string) []map[string]any {
	var profiles []map[string]any

	for _, profile := range cacheAccessor.GetAllDeviceProfiles() {
		// Only include site-specific profiles
		if !profile.ForSite || profile.SiteID != siteID {
			continue
		}

		profileData := map[string]any{
			"id":       profile.ID,
			"name":     profile.Name,
			"type":     profile.Type,
			"for_site": profile.ForSite,
		}

		profiles = append(profiles, profileData)
	}

	return profiles
}

func loadExistingConfig(path string) (*SiteExportConfig, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}

	var config SiteExportConfig
	if err := json.Unmarshal(data, &config); err != nil {
		logging.Warnf("Failed to parse existing config: %v", err)
		return nil, false
	}

	return &config, true
}

func compareSiteConfig(exportData, existingData *SiteExportConfig, fileExists bool, outputPath, siteName string) error {
	if !fileExists {
		// No existing file - just export
		fmt.Printf("No existing config at %s, exporting...\n", outputPath)
		return writeSiteConfig(outputPath, exportData)
	}

	// Convert to JSON for comparison
	exportJSON, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal export data: %w", err)
	}

	existingJSON, err := json.MarshalIndent(existingData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal existing data: %w", err)
	}

	// Create diff
	opts := jsondiff.DiffOptions{
		ContextLines: 3,
		SortJSON:     true,
	}

	diffs, err := jsondiff.Diff(existingJSON, exportJSON, opts)
	if err != nil {
		return fmt.Errorf("failed to generate diff: %w", err)
	}

	if len(diffs) == 0 {
		fmt.Printf("No differences found for site '%s'\n", siteName)
		return nil
	}

	// Enhance and format diffs
	diffs = jsondiff.EnhanceDiffsWithInlineChanges(diffs)
	formatter := jsondiff.NewFormatter(nil)
	formatter.SetMarkers("Local Config", "API Cache", "Both")
	output := formatter.Format(diffs)

	fmt.Printf("\nConfiguration differences for site '%s':\n", siteName)
	fmt.Println(output)

	return nil
}

func writeSiteConfig(path string, data *SiteExportConfig) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal with indentation
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// Write file
	if err := os.WriteFile(path, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func confirmOverwrite(path string) bool {
	fmt.Printf("File already exists: %s\n", path)
	fmt.Print("Overwrite? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

func copyMap(m map[string]any) map[string]any {
	result := make(map[string]any)
	for k, v := range m {
		if nested, ok := v.(map[string]any); ok {
			result[k] = copyMap(nested)
		} else {
			result[k] = v
		}
	}
	return result
}

func redactSecrets(config map[string]any) {
	// Common secret field names to redact
	secretFields := []string{"psk", "passphrase", "secret", "password", "key"}

	for key, value := range config {
		keyLower := strings.ToLower(key)

		// Check if this is a secret field
		for _, secretField := range secretFields {
			if strings.Contains(keyLower, secretField) {
				config[key] = "***REDACTED***"
				break
			}
		}

		// Recursively check nested maps
		if nested, ok := value.(map[string]any); ok {
			redactSecrets(nested)
		}
	}
}

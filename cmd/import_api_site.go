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
	"github.com/ravinald/wifimgr/internal/config"
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

	// Find the site, optionally filtered by API label. Propagate the accessor
	// error directly so "did you mean?" suggestions embedded in it survive.
	var site *vendors.SiteInfo
	if parsed.apiLabel != "" {
		site, err = cacheAccessor.GetSiteByNameAndAPI(siteName, parsed.apiLabel)
	} else {
		site, err = cacheAccessor.GetSiteByName(siteName)
	}
	if err != nil {
		return err
	}

	// Determine the API label to use
	apiLabel := site.SourceAPI

	logger.Infof("Importing site '%s' from API '%s' (scope: %s, save: %v)", siteName, apiLabel, parsed.scope, parsed.saveMode)

	// Build the export data (site config + companion WLAN template)
	exportData, templateData, err := buildSiteExportData(cacheAccessor, site, parsed.scope, parsed.includeSecrets)
	if err != nil {
		return fmt.Errorf("failed to build export data: %w", err)
	}

	// Determine output paths for both files. The site path follows the
	// operator's --output if provided; the template path is derived from it
	// so both files live in a predictable layout.
	configDir := viper.GetString("files.config_dir")
	sitePath := resolveSiteOutputPath(parsed.outputFile, configDir, apiLabel, siteName)
	templatePath := resolveTemplateOutputPath(sitePath, configDir, apiLabel, siteName)

	// Non-save modes (STDOUT preview, compare) only operate on the site
	// document today — templates are only materialized when saving.
	if parsed.compareMode {
		if exportData == nil {
			return fmt.Errorf("compare is not supported with scope %q", parsed.scope)
		}
		existingData, fileExists := loadExistingConfig(sitePath)
		return compareSiteConfig(exportData, existingData, fileExists, sitePath, siteName)
	}

	if !parsed.saveMode {
		switch {
		case exportData != nil:
			jsonData, err := json.MarshalIndent(exportData, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal data: %w", err)
			}
			fmt.Println(string(jsonData))
		case templateData != nil:
			jsonData, err := json.MarshalIndent(templateData, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal template data: %w", err)
			}
			fmt.Println(string(jsonData))
		}
		return nil
	}

	// Save mode: write whichever files the scope produced, confirming any
	// overwrites first.
	if exportData != nil {
		if _, exists := loadExistingConfig(sitePath); exists {
			if !confirmOverwrite(sitePath) {
				fmt.Println("Import cancelled")
				return nil
			}
		}
		if err := writeSiteConfig(sitePath, exportData); err != nil {
			return fmt.Errorf("failed to write site config: %w", err)
		}
	}

	if templateData != nil {
		if _, err := os.Stat(templatePath); err == nil {
			if !confirmOverwrite(templatePath) {
				fmt.Println("Template import cancelled")
				return nil
			}
		}
		if err := writeWLANTemplateFile(templatePath, templateData); err != nil {
			return fmt.Errorf("failed to write WLAN template: %w", err)
		}
	}

	printActivationHint(sitePath, templatePath, exportData != nil, templateData != nil, configDir)
	return nil
}

// resolveSiteOutputPath picks the on-disk location for the site config file.
// Respects an explicit --output, falls back to <configDir>/<api>/sites/<name>.json.
func resolveSiteOutputPath(outputFile, configDir, apiLabel, siteName string) string {
	if outputFile != "" {
		if filepath.IsAbs(outputFile) {
			return outputFile
		}
		return filepath.Join(configDir, outputFile)
	}
	return filepath.Join(xdg.GetConfigDir(), apiLabel, "sites", siteName+".json")
}

// resolveTemplateOutputPath mirrors the site path into the companion template
// directory. When the site path lives under `.../sites/...`, the template goes
// to the same relative path with `/sites/` swapped for `/wlans/`. Otherwise it
// falls back to `<configDir>/<api>/wlans/<site-name>.json`.
func resolveTemplateOutputPath(sitePath, configDir, apiLabel, siteName string) string {
	if strings.Contains(sitePath, string(os.PathSeparator)+"sites"+string(os.PathSeparator)) {
		return strings.Replace(sitePath, string(os.PathSeparator)+"sites"+string(os.PathSeparator), string(os.PathSeparator)+"wlans"+string(os.PathSeparator), 1)
	}
	return filepath.Join(xdg.GetConfigDir(), apiLabel, "wlans", siteName+".json")
}

// printActivationHint tells the operator exactly what to add to the main
// config, using paths relative to config_dir so the message matches what they
// type into `files.site_configs` / `files.templates`.
func printActivationHint(sitePath, templatePath string, wroteSite, wroteTemplate bool, configDir string) {
	relSite := relativeFromConfigDir(sitePath, configDir)
	relTemplate := relativeFromConfigDir(templatePath, configDir)

	switch {
	case wroteSite && wroteTemplate:
		fmt.Printf("Wrote site config:    %s\n", sitePath)
		fmt.Printf("Wrote WLAN template:  %s\n", templatePath)
		fmt.Printf("\nTo activate, add to your wifimgr-config.json:\n")
		fmt.Printf("  \"files\": {\n")
		fmt.Printf("    \"site_configs\": [ ..., %q ],\n", relSite)
		fmt.Printf("    \"templates\":    [ ..., %q ]\n", relTemplate)
		fmt.Printf("  }\n")
	case wroteSite:
		fmt.Printf("Exported site configuration to: %s\n", sitePath)
		fmt.Printf("\nTo activate, add to files.site_configs: %q\n", relSite)
	case wroteTemplate:
		fmt.Printf("Exported WLAN template to: %s\n", templatePath)
		fmt.Printf("\nTo activate, add to files.templates: %q\n", relTemplate)
	}
}

// relativeFromConfigDir returns path rendered relative to configDir when
// possible, preserving the form the operator types into main config.
func relativeFromConfigDir(path, configDir string) string {
	if configDir == "" {
		return path
	}
	if rel, err := filepath.Rel(configDir, path); err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}
	return path
}

// writeWLANTemplateFile serializes a WLANProfileFile to disk, creating the
// parent directory if needed. Mirrors writeSiteConfig so file permissions and
// directory modes stay consistent between the pair.
func writeWLANTemplateFile(path string, file *config.WLANProfileFile) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal template: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil { // #nosec G304 G703 -- path from operator-controlled config
		return fmt.Errorf("failed to write template file: %w", err)
	}
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

// SiteConfigData represents the configuration data for a single site.
// The shape matches config.SiteConfigObj so exported files load back through
// the same path as hand-written site configs.
type SiteConfigData struct {
	API        string                       `json:"api,omitempty"`
	SiteConfig map[string]any               `json:"site_config"`
	Devices    *DevicesConfig               `json:"devices,omitempty"`
	WLAN       []string                     `json:"wlan,omitempty"`     // template labels; definitions live in the companion wlan_profiles file
	Profiles   config.SiteConfigObjProfiles `json:"profiles,omitempty"` // device/radio/wlan template label refs (device-profile export not yet migrated)
}

// DevicesConfig holds device configurations by type
type DevicesConfig struct {
	AP      map[string]map[string]any `json:"ap,omitempty"`
	Switch  map[string]map[string]any `json:"switch,omitempty"`
	Gateway map[string]map[string]any `json:"gateway,omitempty"`
}

// buildSiteExportData builds both halves of an import: the site config file
// and, when WLANs are in scope, the companion WLAN template file. Either can
// be nil depending on scope — `wlans`-only returns a nil *SiteExportConfig,
// device-type scopes return a nil *config.WLANProfileFile.
func buildSiteExportData(cacheAccessor *vendors.CacheAccessor, site *vendors.SiteInfo, scope ImportScope, includeSecrets bool) (*SiteExportConfig, *config.WLANProfileFile, error) {
	siteSlug := slug(site.Name)

	// WLANs always processed first so label references land on the site entry.
	var wlanLabels []string
	var wlanProfiles map[string]*config.WLANProfile
	if scope == ScopeFull || scope == ScopeWLANs {
		wlanLabels, wlanProfiles = buildWLANsExport(cacheAccessor, site.ID, siteSlug, includeSecrets)
	}

	var templateFile *config.WLANProfileFile
	if len(wlanProfiles) > 0 {
		templateFile = &config.WLANProfileFile{
			Version:      1,
			WLANProfiles: wlanProfiles,
		}
	}

	// `wlans` scope emits only the template file.
	if scope == ScopeWLANs {
		return nil, templateFile, nil
	}

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

	if site.Latitude != 0 || site.Longitude != 0 {
		siteData.SiteConfig["latlng"] = map[string]float64{
			"lat": site.Latitude,
			"lng": site.Longitude,
		}
	}

	if scope == ScopeFull || scope == ScopeAP || scope == ScopeSwitch || scope == ScopeGateway {
		devices, err := buildDevicesExport(cacheAccessor, site.ID, site.SourceVendor, scope)
		if err != nil {
			logging.Warnf("Failed to build devices export: %v", err)
		} else {
			siteData.Devices = devices
		}
	}

	if len(wlanLabels) > 0 {
		siteData.WLAN = wlanLabels
	}

	// NOTE: buildProfilesExport still returns the old `[]map[string]any` shape
	// and is a stale code path for Mist device-profile imports. Migrating it to
	// label-and-file form is tracked separately; today it's a no-op for Meraki
	// (which has no device-profile concept) and harmlessly empty for most Mist
	// sites that rely on org-level profiles.
	_ = scope // placeholder for when the profile export path is migrated

	return &SiteExportConfig{
		Version: 1,
		Config: SiteExportConfigData{
			Sites: map[string]*SiteConfigData{
				site.Name: siteData,
			},
		},
	}, templateFile, nil
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

// buildWLANsExport converts vendor-normalized WLANs for a site into the
// Mist-canonical pair of (label list, label→profile map). Thin wrapper around
// the accessor so the transformation logic stays testable.
func buildWLANsExport(cacheAccessor *vendors.CacheAccessor, siteID, siteSlug string, includeSecrets bool) ([]string, map[string]*config.WLANProfile) {
	return synthesizeWLANLabels(cacheAccessor.GetWLANsBySite(siteID), siteSlug, includeSecrets)
}

// synthesizeWLANLabels is the pure half of buildWLANsExport: given a slice of
// vendor-normalized WLANs and a site slug, produce label references and the
// matching WLANProfile map. Separated so unit tests can drive it directly.
func synthesizeWLANLabels(vendorWLANs []*vendors.WLAN, siteSlug string, includeSecrets bool) ([]string, map[string]*config.WLANProfile) {
	if len(vendorWLANs) == 0 {
		return nil, nil
	}

	labels := make([]string, 0, len(vendorWLANs))
	profiles := make(map[string]*config.WLANProfile, len(vendorWLANs))
	used := make(map[string]int) // collision counter keyed on bare label

	for _, w := range vendorWLANs {
		base := fmt.Sprintf("%s--%s", siteSlug, slug(w.SSID))
		label := base
		if n := used[base]; n > 0 {
			label = fmt.Sprintf("%s-%d", base, n+1)
		}
		used[base]++

		profiles[label] = convertVendorWLANToProfile(w, includeSecrets)
		labels = append(labels, label)
	}

	return labels, profiles
}

// convertVendorWLANToProfile maps a vendor-normalized WLAN into the portable
// WLANProfile shape the template system uses. Vendor-specific metadata
// (network IDs, SSID slot numbers, ipAssignmentMode, etc.) is deliberately
// dropped — this is the portable half. PSK and RADIUS secrets honor
// includeSecrets and are otherwise omitted.
func convertVendorWLANToProfile(w *vendors.WLAN, includeSecrets bool) *config.WLANProfile {
	profile := &config.WLANProfile{
		SSID:    w.SSID,
		Enabled: w.Enabled,
		Hidden:  w.Hidden,
		Band:    normalizeBand(w.Band),
		VLANID:  w.VLANID,
		Auth: config.WLANAuth{
			Type: normalizeAuthType(w.AuthType),
		},
	}

	if includeSecrets && w.PSK != "" {
		profile.Auth.PSK = w.PSK
	}

	if pairwise := derivePairwiseFromConfig(w.Config, w.EncryptionMode); len(pairwise) > 0 {
		profile.Auth.Pairwise = pairwise
	}

	if len(w.RadiusServers) > 0 {
		servers := make([]config.RADIUSServer, 0, len(w.RadiusServers))
		for _, rs := range w.RadiusServers {
			cs := config.RADIUSServer{Host: rs.Host, Port: rs.Port}
			if includeSecrets {
				cs.Secret = rs.Secret
			}
			servers = append(servers, cs)
		}
		profile.Auth.RADIUSServers = servers
	}

	// Bandwidth and portal settings live in the raw vendor config map; pull
	// them opportunistically so the round-trip stays close to lossless without
	// leaking vendor-specific field names into the profile schema.
	if w.Config != nil {
		if up, ok := firstInt(w.Config, "per_client_bandwidth_limit_up", "perClientBandwidthLimitUp"); ok {
			profile.ClientLimitUp = up
		}
		if down, ok := firstInt(w.Config, "per_client_bandwidth_limit_down", "perClientBandwidthLimitDown"); ok {
			profile.ClientLimitDown = down
		}
		if portal := extractPortal(w.Config); portal != nil {
			profile.Portal = portal
		}
	}

	return profile
}

// normalizeAuthType maps vendor-specific auth strings into the canonical
// set used by WLANProfile: "open" (incl. OWE and MAC-RADIUS), "psk" (plain
// pre-shared key), "ipsk" (Meraki-style identity PSK, preserved as its own
// class because it implies per-client key provisioning) and "eap" (any
// 802.1X / enterprise variant).
func normalizeAuthType(t string) string {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "", "open":
		return "open"
	case "open-with-radius", "open-with-nac", "open-enhanced":
		return "open"
	case "psk", "wpa", "wpa-psk", "wpa2-psk", "wpa3-psk", "wpa2/wpa3-psk", "sae":
		return "psk"
	case "ipsk", "ipsk-without-radius", "ipsk-with-radius", "ipsk-with-nac":
		return "ipsk"
	case "eap", "enterprise", "wpa2-enterprise", "wpa3-enterprise",
		"wpa-eap", "wpa2-eap",
		"8021x", "8021x-radius", "8021x-meraki", "8021x-google", "8021x-entra", "8021x-localradius":
		return "eap"
	default:
		return t // leave verbatim so operators can see what came through
	}
}

// normalizeBand maps vendor-specific band labels into WLANProfile's canonical
// "2.4" / "5" / "6" / "dual" / "all" set. Meraki's bandSelection uses long
// strings like "Dual band operation with Band Steering" — band steering is a
// mode flag, not a separate band, so both dual-band variants collapse to
// "dual". Unknown values pass through verbatim.
func normalizeBand(b string) string {
	s := strings.ToLower(strings.TrimSpace(b))
	switch s {
	case "":
		return ""
	case "2.4", "2.4ghz", "2.4 ghz", "2.4 ghz band only":
		return "2.4"
	case "5", "5ghz", "5 ghz", "5 ghz band only":
		return "5"
	case "6", "6ghz", "6 ghz", "6 ghz band only":
		return "6"
	case "dual", "dual band operation", "dual band operation with band steering":
		return "dual"
	case "all", "all bands", "tri-band":
		return "all"
	default:
		return b
	}
}

// derivePairwiseFromConfig prefers Meraki's modern `wpaEncryptionMode` field
// (which carries strings like "WPA2 only" or "WPA3 Transition Mode") when
// present in the raw vendor config. Falls back to the legacy EncryptionMode
// via encryptionModeToPairwise when wpaEncryptionMode is absent.
func derivePairwiseFromConfig(cfg map[string]any, legacyMode string) []string {
	if cfg != nil {
		if v, ok := cfg["wpaEncryptionMode"].(string); ok && v != "" {
			if p := merakiWpaModeToPairwise(v); p != nil {
				return p
			}
		}
	}
	return encryptionModeToPairwise(legacyMode)
}

// merakiWpaModeToPairwise maps Meraki's wpaEncryptionMode string (human-
// readable, varies across API versions) to canonical pairwise cipher labels.
// Unknown values return nil so the caller can fall back.
func merakiWpaModeToPairwise(mode string) []string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "wpa1 only":
		return []string{"wpa"}
	case "wpa1 and wpa2", "wpa2 only":
		return []string{"wpa2-ccmp"}
	case "wpa3 personal", "wpa3 only":
		return []string{"wpa3"}
	case "wpa3 transition mode", "wpa2/wpa3", "wpa2 and wpa3":
		return []string{"wpa2-ccmp", "wpa3"}
	default:
		return nil
	}
}

// encryptionModeToPairwise splits a vendor encryption label into WLANProfile's
// Pairwise list. Empty input returns a nil slice so the field stays omitempty.
func encryptionModeToPairwise(mode string) []string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "none":
		return nil
	case "wpa2":
		return []string{"wpa2-ccmp"}
	case "wpa3":
		return []string{"wpa3"}
	case "wpa2/wpa3", "wpa2+wpa3":
		return []string{"wpa2-ccmp", "wpa3"}
	default:
		return []string{mode}
	}
}

// firstInt returns the first key in the map (checked in order) that resolves to
// an int-like value, along with its int value.
func firstInt(m map[string]any, keys ...string) (int, bool) {
	for _, k := range keys {
		v, ok := m[k]
		if !ok {
			continue
		}
		switch n := v.(type) {
		case int:
			return n, true
		case int64:
			return int(n), true
		case float64:
			return int(n), true
		}
	}
	return 0, false
}

// extractPortal derives a PortalConfig from the raw vendor config if captive
// portal / splash settings are present. Returns nil when the vendor didn't
// advertise a portal.
func extractPortal(m map[string]any) *config.PortalConfig {
	// Meraki: `splashPage` is "None" when disabled, otherwise a human-readable
	// auth type like "Password-protected with Meraki RADIUS" or "Click-through".
	if sp, ok := m["splashPage"].(string); ok && sp != "" && !strings.EqualFold(sp, "none") {
		return &config.PortalConfig{Enabled: true, Auth: sp}
	}
	// Mist: `portal.enabled` bool, `portal.auth` string.
	if raw, ok := m["portal"].(map[string]any); ok {
		enabled, _ := raw["enabled"].(bool)
		auth, _ := raw["auth"].(string)
		if enabled || auth != "" {
			return &config.PortalConfig{Enabled: enabled, Auth: auth}
		}
	}
	return nil
}

// slug lowercases a string and collapses runs of non-alphanumeric characters
// into single hyphens. Used to produce stable label and filename tokens.
func slug(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	lastDash := true // treat leading as dash-boundary so we trim
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	out := b.String()
	return strings.TrimRight(out, "-")
}

func loadExistingConfig(path string) (*SiteExportConfig, bool) {
	data, err := os.ReadFile(path) // #nosec G304 -- path from operator-controlled config
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
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal with indentation
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// Write file
	if err := os.WriteFile(path, jsonData, 0600); err != nil {
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

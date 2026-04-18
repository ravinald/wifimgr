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
	"time"

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
  wifimgr import api site US-SFO-LAB save file import/lab.json
  wifimgr import api site US-SFO-LAB save file /tmp/site.json

Combined Options:
  wifimgr import api site api mist-prod US-SFO-LAB save file import/sfo.json
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
  compare        Optional. Compare API state with existing import file (using jsondiff)
  save           Optional. Write to import file (default: print to STDOUT)
  file           Optional. Keyword followed by output filename (relative to config_dir or absolute)

What it Does:
  1. Retrieves site configuration from the specified API cache
  2. Optionally filters to specific scope (wlans, profiles, ap, switch, gateway)
  3. Redacts secrets by default (use 'secrets' to include)
  4. Prints to STDOUT or saves to a single import file if 'save' specified
  5. With 'compare': shows diff between API and existing local import file

Output Location:
  Without 'save': Prints JSON to STDOUT
  With 'save' (no file): <config_dir>/import/<site-slug>_<api>.json
  With 'save file': <config_dir>/<filename> (relative) or <filename> (absolute)`,
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

	apiLabel := site.SourceAPI
	logger.Infof("Importing site '%s' from API '%s' (scope: %s, save: %v)", siteName, apiLabel, parsed.scope, parsed.saveMode)

	// Build the combined ImportFile envelope (site config + site-local WLAN
	// templates live side-by-side so the file self-describes).
	exportData, err := buildSiteExportData(cacheAccessor, site, parsed.scope, parsed.includeSecrets)
	if err != nil {
		return fmt.Errorf("failed to build export data: %w", err)
	}

	configDir := viper.GetString("files.config_dir")
	outputPath := resolveImportOutputPath(parsed.outputFile, configDir, apiLabel, site.Name)

	if parsed.compareMode {
		if exportData == nil {
			return fmt.Errorf("nothing to compare for scope %q", parsed.scope)
		}
		existingData, fileExists := loadExistingImport(outputPath)
		return compareImportFile(exportData, existingData, fileExists, outputPath, siteName)
	}

	if !parsed.saveMode {
		if exportData == nil {
			fmt.Println("{}")
			return nil
		}
		jsonData, err := json.MarshalIndent(exportData, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal data: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Save mode
	if exportData == nil {
		fmt.Println("Nothing to export")
		return nil
	}
	if _, exists := loadExistingImport(outputPath); exists {
		if !confirmOverwrite(outputPath) {
			fmt.Println("Import cancelled")
			return nil
		}
	}
	if err := writeImportFile(outputPath, exportData); err != nil {
		return fmt.Errorf("failed to write import file: %w", err)
	}

	printActivationHint(outputPath, configDir)
	return nil
}

// resolveImportOutputPath picks the on-disk location for the import file.
// Respects an explicit `file` argument; otherwise falls back to the
// convention <configDir>/import/<site-slug>_<api>.json.
func resolveImportOutputPath(outputFile, configDir, apiLabel, siteName string) string {
	if outputFile != "" {
		if filepath.IsAbs(outputFile) {
			return outputFile
		}
		return filepath.Join(configDir, outputFile)
	}
	baseDir := configDir
	if baseDir == "" {
		baseDir = xdg.GetConfigDir()
	}
	return filepath.Join(baseDir, "import", fmt.Sprintf("%s_%s.json", slug(siteName), apiLabel))
}

// printActivationHint tells the operator exactly what to add to the main
// config, using a path relative to config_dir so the message matches what they
// type into `files.imports`.
func printActivationHint(outputPath, configDir string) {
	rel := relativeFromConfigDir(outputPath, configDir)
	fmt.Printf("Wrote import file: %s\n", outputPath)
	fmt.Printf("\nTo activate, add to your wifimgr-config.json:\n")
	fmt.Printf("  \"files\": {\n")
	fmt.Printf("    \"imports\": [ ..., %q ]\n", rel)
	fmt.Printf("  }\n")
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

// importEnvelope mirrors config.ImportFile for serialization. We keep a local
// type because devices are emitted as raw maps (to preserve all vendor-specific
// fields verbatim), but the on-disk JSON is still valid for loading through
// config.LoadImportFile — unknown fields are dropped during unmarshal.
type importEnvelope struct {
	Version   int                 `json:"version"`
	Source    *importSourceExport `json:"source,omitempty"`
	Config    *siteConfigEnvelope `json:"config,omitempty"`
	Templates *templatesEnvelope  `json:"templates,omitempty"`
}

// importSourceExport mirrors config.ImportSource but lives in cmd/ so we don't
// have to export the "zero time means don't emit" trick from the config side.
type importSourceExport struct {
	API        string    `json:"api,omitempty"`
	Site       string    `json:"site,omitempty"`
	SiteID     string    `json:"site_id,omitempty"`
	Kind       string    `json:"kind,omitempty"`
	ImportedAt time.Time `json:"imported_at,omitempty"`
}

type siteConfigEnvelope struct {
	Sites map[string]*siteObjExport `json:"sites"`
}

// siteObjExport mirrors config.SiteConfigObj. We keep SiteConfig and device
// bodies as loose maps here so vendor-specific fields round-trip without
// requiring us to enumerate every field on the typed struct.
type siteObjExport struct {
	API        string                       `json:"api,omitempty"`
	SiteConfig map[string]any               `json:"site_config"`
	Profiles   config.SiteConfigObjProfiles `json:"profiles,omitempty"`
	WLAN       []string                     `json:"wlan,omitempty"`
	Devices    *devicesExport               `json:"devices,omitempty"`
}

type devicesExport struct {
	AP      map[string]map[string]any `json:"ap,omitempty"`
	Switch  map[string]map[string]any `json:"switch,omitempty"`
	Gateway map[string]map[string]any `json:"gateway,omitempty"`
}

type templatesEnvelope struct {
	WLAN   map[string]map[string]any `json:"wlan,omitempty"`
	Radio  map[string]map[string]any `json:"radio,omitempty"`
	Device map[string]map[string]any `json:"device,omitempty"`
}

// buildSiteExportData builds an ImportFile-shaped envelope containing both
// the site config and the companion WLAN templates used by that site. Returns
// nil when the scope produces nothing to emit.
func buildSiteExportData(cacheAccessor *vendors.CacheAccessor, site *vendors.SiteInfo, scope ImportScope, includeSecrets bool) (*importEnvelope, error) {
	siteSlug := slug(site.Name)

	// WLANs always processed first so label references are ready to attach.
	var wlanLabels []string
	var wlanTemplates map[string]map[string]any
	if scope == ScopeFull || scope == ScopeWLANs {
		wlanLabels, wlanTemplates = buildWLANsExport(cacheAccessor, site.ID, siteSlug, includeSecrets)
	}

	env := &importEnvelope{
		Version: 1,
		Source: &importSourceExport{
			API:        site.SourceAPI,
			Site:       site.Name,
			SiteID:     site.ID,
			Kind:       "site",
			ImportedAt: time.Now().UTC(),
		},
	}

	if len(wlanTemplates) > 0 {
		env.Templates = &templatesEnvelope{WLAN: wlanTemplates}
	}

	// `wlans` scope emits only the templates section — no site body.
	if scope == ScopeWLANs {
		if env.Templates == nil {
			return nil, nil
		}
		// Drop the site-oriented source kind since the file only carries
		// templates in this scope.
		env.Source.Kind = "wlans"
		return env, nil
	}

	siteBody := &siteObjExport{
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
		siteBody.SiteConfig["latlng"] = map[string]float64{
			"lat": site.Latitude,
			"lng": site.Longitude,
		}
	}

	if scope == ScopeFull || scope == ScopeAP || scope == ScopeSwitch || scope == ScopeGateway {
		devices, err := buildDevicesExport(cacheAccessor, site.ID, site.SourceVendor, scope)
		if err != nil {
			logging.Warnf("Failed to build devices export: %v", err)
		} else {
			siteBody.Devices = devices
		}
	}

	if len(wlanLabels) > 0 {
		siteBody.WLAN = wlanLabels
	}

	env.Config = &siteConfigEnvelope{
		Sites: map[string]*siteObjExport{site.Name: siteBody},
	}
	return env, nil
}

func buildDevicesExport(cacheAccessor *vendors.CacheAccessor, siteID string, sourceVendor string, scope ImportScope) (*devicesExport, error) {
	devices := &devicesExport{
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
// Mist-canonical pair of (label list, label→template body map). The template
// body is the JSON shape of a WLANProfile rendered back to a raw map so it
// drops straight into the Templates.WLAN section of the ImportFile envelope.
func buildWLANsExport(cacheAccessor *vendors.CacheAccessor, siteID, siteSlug string, includeSecrets bool) ([]string, map[string]map[string]any) {
	labels, profiles := synthesizeWLANLabels(cacheAccessor.GetWLANsBySite(siteID), siteSlug, includeSecrets)
	if len(profiles) == 0 {
		return labels, nil
	}
	templates := make(map[string]map[string]any, len(profiles))
	for label, p := range profiles {
		m, err := profileToMap(p)
		if err != nil {
			logging.Warnf("Failed to serialize WLAN profile %q: %v", label, err)
			continue
		}
		templates[label] = m
	}
	return labels, templates
}

// profileToMap renders a WLANProfile as a loose map so it fits the
// TemplateDefinitions shape (map[string]map[string]any). Done via JSON
// round-trip to preserve omitempty semantics.
func profileToMap(p *config.WLANProfile) (map[string]any, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
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

// loadExistingImport reads an import file already on disk, returning nil when
// absent or unreadable. Used by the compare flow.
func loadExistingImport(path string) (*importEnvelope, bool) {
	data, err := os.ReadFile(path) // #nosec G304 -- path from operator-controlled config
	if err != nil {
		return nil, false
	}

	var env importEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		logging.Warnf("Failed to parse existing import file: %v", err)
		return nil, false
	}

	return &env, true
}

// compareImportFile prints a jsondiff-formatted diff of the existing file
// against the freshly-built export. Writes the file if none exists yet.
func compareImportFile(exportData, existingData *importEnvelope, fileExists bool, outputPath, siteName string) error {
	if !fileExists {
		fmt.Printf("No existing import file at %s, exporting...\n", outputPath)
		return writeImportFile(outputPath, exportData)
	}

	exportJSON, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal export data: %w", err)
	}

	existingJSON, err := json.MarshalIndent(existingData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal existing data: %w", err)
	}

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

	diffs = jsondiff.EnhanceDiffsWithInlineChanges(diffs)
	formatter := jsondiff.NewFormatter(nil)
	formatter.SetMarkers("Local Import", "API Cache", "Both")
	output := formatter.Format(diffs)

	fmt.Printf("\nConfiguration differences for site '%s':\n", siteName)
	fmt.Println(output)

	return nil
}

// writeImportFile writes the envelope to disk, creating parent directories.
func writeImportFile(path string, env *importEnvelope) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	jsonData, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

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

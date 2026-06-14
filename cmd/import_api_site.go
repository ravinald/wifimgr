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
	"sort"
	"strconv"
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
	Use:   "site <site-name> [config|inventory|all] [full|type <scope>] [source <api-label>] [secrets] [compare] [save] [file <filename>]",
	Short: "Import site configuration from API cache",
	Long: `Import site configuration from the API cache.

By default, this command outputs the configuration to STDOUT for preview.
Use the 'save' argument to write to a local config file.

Pick what to import with config (default), inventory, or all:
  config      Translate the vendor config into a wifimgr import envelope
  inventory   Arm the discovered devices into the per-site allowlist (inventory.json)
  all         Both

Devices a vendor manages through a template or profile (a Meraki network bound
to a configuration template, or Mist device profiles) can't be fully owned by
wifimgr's direct-to-device push. Import still runs, prints a WARN, and stamps a
'_note' into the written file and section.

Basic Usage:
  wifimgr import api site US-SFO-LAB
  wifimgr import api site US-SFO-LAB save
  wifimgr import api site US-SFO-LAB inventory
  wifimgr import api site US-SFO-LAB all save
  wifimgr import api site US-SFO-LAB type ap
  wifimgr import api site US-SFO-LAB compare

With Explicit API (when a site name exists in multiple APIs):
  wifimgr import api site US-SFO-LAB source mist-prod
  wifimgr import api site US-SFO-LAB source meraki save

With Custom Output File:
  wifimgr import api site US-SFO-LAB save file custom.json
  wifimgr import api site US-SFO-LAB save file import/lab.json
  wifimgr import api site US-SFO-LAB save file /tmp/site.json

Combined Options:
  wifimgr import api site US-SFO-LAB source mist-prod save file import/sfo.json
  wifimgr import api site US-SFO-LAB type wlans secrets save
  wifimgr import api site US-SFO-LAB | jq '.config'

Arguments:
  site-name      Required. The site name to import
  source <label> Optional. Keyword + API label to import from, when a site name
                 spans multiple APIs (e.g. the same name on Mist and Aruba)
  config         Optional. Import the configuration envelope (default)
  inventory      Optional. Arm discovered devices into inventory.json
  all            Optional. Import config and arm inventory
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

// importKind selects what an import produces. config translates the vendor
// config into a wifimgr import envelope (the long-standing behavior); inventory
// arms the discovered devices into the per-site allowlist; all does both.
type importKind int

const (
	kindConfig importKind = iota
	kindInventory
	kindAll
)

// importSiteArgs holds parsed arguments for the import api site command
type importSiteArgs struct {
	apiLabel    string
	siteName    string
	kind        importKind
	scope       ImportScope
	compareMode bool
	cmdutils.ImportOutputArgs
}

// parseImportSiteArgs parses positional arguments for import api site command
func parseImportSiteArgs(args []string) (*importSiteArgs, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("site name required")
	}

	result := &importSiteArgs{
		scope: ScopeFull,
	}

	// First argument is the site name; the API (when needed) is given by the
	// trailing 'source <api-label>' keyword.
	result.siteName = args[0]
	i := 1

	// Parse remaining arguments
	for i < len(args) {
		if matched, last, err := result.Consume(args, i); err != nil {
			return nil, err
		} else if matched {
			i = last + 1
			continue
		}
		switch strings.ToLower(args[i]) {
		case "config":
			result.kind = kindConfig
		case "inventory":
			result.kind = kindInventory
		case "all":
			result.kind = kindAll
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
		case "compare":
			result.compareMode = true
		case "source":
			// Disambiguate which API to import from when a site name spans APIs
			// (e.g. the same site on Mist and Aruba). Mirrors `target` elsewhere,
			// but reads as "source" since import pulls from it.
			if i+1 >= len(args) {
				return nil, fmt.Errorf("'source' requires an API label")
			}
			result.apiLabel = args[i+1]
			i++ // Skip the API label
		case "help":
			// Already handled in RunE
		default:
			return nil, fmt.Errorf("unknown argument: %s", args[i])
		}
		i++
	}

	if err := result.Validate(); err != nil {
		return nil, err
	}

	return result, nil
}

func runImportAPISite(cmd *cobra.Command, args []string) error {
	if cmdutils.ContainsHelp(args) {
		return cmd.Help()
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

	// Resolve the site (duplicate-safe; honours an explicit api label), then
	// fetch the full record by its unique ID. Errors propagate directly so the
	// "did you mean?" / duplicate-site guidance survives.
	ref, err := cmdutils.ResolveSite(siteName, parsed.apiLabel)
	if err != nil {
		return err
	}
	site, err := cacheAccessor.GetSiteByID(ref.SiteID)
	if err != nil {
		return err
	}

	apiLabel := site.SourceAPI
	logger.Infof("Importing site '%s' from API '%s' (scope: %s, save: %v)", siteName, apiLabel, parsed.scope, parsed.SaveMode)

	// Build the combined ImportFile envelope (site config + site-local WLAN
	// templates live side-by-side so the file self-describes).
	exportData, err := buildSiteExportData(cacheAccessor, site, parsed.scope, parsed.IncludeSecrets)
	if err != nil {
		return fmt.Errorf("failed to build export data: %w", err)
	}

	// Surface and annotate template/profile-managed devices before emitting
	// anything: direct-to-device push can't fully own those, so the warning and
	// the _note ride along into whatever we write (config envelope, inventory).
	tmpl := detectTemplateManagement(site, exportData)
	if tmpl.managed {
		warnTemplateManaged(site.Name, tmpl)
		annotateEnvelope(exportData, site.Name, tmpl)
	}

	// Inventory: arm the discovered devices into the per-site allowlist. This
	// writes inventory.json directly (no staging) — the apply-time dual-inventory
	// check remains the guard against acting on the wrong device.
	if parsed.kind == kindInventory || parsed.kind == kindAll {
		if err := armSiteInventory(site.Name, exportData, tmpl); err != nil {
			return err
		}
		if parsed.kind == kindInventory {
			return nil
		}
	}

	configDir := viper.GetString("files.config_dir")
	outputPath := resolveImportOutputPath(parsed.OutputFile, configDir, apiLabel, site.Name)

	if parsed.compareMode {
		if exportData == nil {
			return fmt.Errorf("nothing to compare for scope %q", parsed.scope)
		}
		existingData, fileExists := loadExistingImport(outputPath)
		return compareImportFile(exportData, existingData, fileExists, outputPath, siteName)
	}

	if !parsed.SaveMode {
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

// templateFinding records which devices in an import are managed by a vendor
// template or profile, so the import can warn and annotate consistently.
type templateFinding struct {
	managed     bool
	merakiBound bool     // site is bound to a Meraki configuration template
	profileMACs []string // normalized MACs managed by a Mist device profile
	note        string   // operator-facing one-liner stamped into written files
}

// detectTemplateManagement inspects a built envelope for the two ways a vendor
// keeps config out of wifimgr's direct-push reach: a Meraki network bound to a
// configuration template (whole site), and Mist devices bound to a device
// profile (per device, surfaced as deviceprofile_* keys during export).
func detectTemplateManagement(site *vendors.SiteInfo, env *importEnvelope) templateFinding {
	var f templateFinding
	if site != nil && site.BoundToConfigTemplate {
		f.merakiBound = true
	}

	devices := envDevices(env, site)
	if devices != nil {
		for _, group := range []map[string]map[string]any{devices.AP, devices.Switch, devices.Gateway} {
			for mac, body := range group {
				if _, ok := body["deviceprofile_name"]; ok {
					f.profileMACs = append(f.profileMACs, mac)
					continue
				}
				if _, ok := body["deviceprofile_id"]; ok {
					f.profileMACs = append(f.profileMACs, mac)
				}
			}
		}
	}

	f.managed = f.merakiBound || len(f.profileMACs) > 0
	switch {
	case f.merakiBound && len(f.profileMACs) > 0:
		f.note = "template-managed: site bound to a Meraki configuration template and some devices use device profiles; direct config push may be overridden"
	case f.merakiBound:
		f.note = "template-managed: site bound to a Meraki configuration template; direct config push may be overridden"
	case len(f.profileMACs) > 0:
		f.note = fmt.Sprintf("template-managed: %d device(s) use a device profile; direct config push may be overridden", len(f.profileMACs))
	}
	return f
}

// envDevices returns the device export for the site, or nil when the envelope
// carries no site body (e.g. wlans scope).
func envDevices(env *importEnvelope, site *vendors.SiteInfo) *devicesExport {
	if env == nil || env.Config == nil || site == nil {
		return nil
	}
	body, ok := env.Config.Sites[site.Name]
	if !ok || body == nil {
		return nil
	}
	return body.Devices
}

// warnTemplateManaged prints the template/profile finding to the operator. The
// import still proceeds — the user asked for it, and the annotation records why
// the result may be incomplete.
func warnTemplateManaged(siteName string, f templateFinding) {
	fmt.Fprintf(os.Stderr, "WARN: %s: %s\n", siteName, f.note)
	if len(f.profileMACs) > 0 {
		fmt.Fprintf(os.Stderr, "WARN:   profile-managed devices: %s\n", strings.Join(f.profileMACs, ", "))
	}
}

// annotateEnvelope stamps the finding into the config envelope: a site-level
// _note plus a per-device _note on each profile-managed device.
func annotateEnvelope(env *importEnvelope, siteName string, f templateFinding) {
	if env == nil || env.Config == nil {
		return
	}
	body, ok := env.Config.Sites[siteName]
	if !ok || body == nil {
		return
	}
	body.Note = f.note

	if body.Devices == nil {
		return
	}
	const perDevice = "template-managed: bound to a device profile; direct config push may be overridden"
	profile := make(map[string]bool, len(f.profileMACs))
	for _, mac := range f.profileMACs {
		profile[mac] = true
	}
	for _, group := range []map[string]map[string]any{body.Devices.AP, body.Devices.Switch, body.Devices.Gateway} {
		for mac, deviceBody := range group {
			if profile[mac] {
				deviceBody["_note"] = perDevice
			}
		}
	}
}

// armSiteInventory writes the discovered devices into the per-site allowlist,
// pulling MACs straight from the built envelope so config and inventory stay
// keyed by the same set. A template finding rides along as the site note.
func armSiteInventory(siteName string, env *importEnvelope, f templateFinding) error {
	devices := envDevices(env, &vendors.SiteInfo{Name: siteName})
	if devices == nil {
		cmdutils.Noticef("%s: no devices to arm", siteName)
		return nil
	}
	aps := macKeys(devices.AP)
	switches := macKeys(devices.Switch)
	gateways := macKeys(devices.Gateway)
	if len(aps)+len(switches)+len(gateways) == 0 {
		cmdutils.Noticef("%s: no devices to arm", siteName)
		return nil
	}

	path := config.InventoryPath(nil)
	if path == "" {
		return fmt.Errorf("inventory: files.inventory is not configured")
	}
	note := ""
	if f.managed {
		note = f.note
	}
	if err := config.ArmSiteDevices(path, siteName, aps, switches, gateways, note); err != nil {
		return err
	}
	cmdutils.Noticef("Armed %d device(s) for %s in %s", len(aps)+len(switches)+len(gateways), siteName, path)
	return nil
}

// macKeys returns the (normalized) MAC keys of a device export group.
func macKeys(group map[string]map[string]any) []string {
	if len(group) == 0 {
		return nil
	}
	out := make([]string, 0, len(group))
	for mac := range group {
		out = append(out, mac)
	}
	return out
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
	// Note flags template/profile-managed devices in this site (see importKind
	// docs). JSON has no comments; loaders ignore unknown keys, so it rides as data.
	Note string `json:"_note,omitempty"`
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
	var wlanAssign map[string][]string
	if scope == ScopeFull || scope == ScopeWLANs {
		wlanLabels, wlanTemplates, wlanAssign = buildWLANsExport(cacheAccessor, site.ID, siteSlug, includeSecrets)
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

	// Declare every imported WLAN in profiles.wlan so the file is apply-ready
	// (apply rejects site/device wlan labels not declared here). Availability is
	// expressed natively: Meraki via the WLAN's vendor block, Mist via per-AP
	// device-level wlan placement below. We deliberately leave site-level `wlan`
	// empty so a re-apply doesn't impose a site-wide assignment.
	if len(wlanLabels) > 0 {
		siteBody.Profiles.WLAN = wlanLabels
	}
	if len(wlanAssign) > 0 {
		attachDeviceWLANs(siteBody, wlanAssign)
	}

	env.Config = &siteConfigEnvelope{
		Sites: map[string]*siteObjExport{site.Name: siteBody},
	}
	return env, nil
}

// attachDeviceWLANs wires Mist per-AP WLAN assignments (label → AP MACs) into the
// site's device bodies as MAC-keyed device-level wlan lists — the form wifimgr
// authors and apply resolves back to ap_ids. Output is deterministic (labels and
// MACs sorted) so re-importing the same site yields a byte-stable file.
func attachDeviceWLANs(siteBody *siteObjExport, assign map[string][]string) {
	if siteBody.Devices == nil {
		siteBody.Devices = &devicesExport{}
	}
	if siteBody.Devices.AP == nil {
		siteBody.Devices.AP = make(map[string]map[string]any)
	}

	labels := make([]string, 0, len(assign))
	for label := range assign {
		labels = append(labels, label)
	}
	sort.Strings(labels)

	for _, label := range labels {
		macs := append([]string(nil), assign[label]...)
		sort.Strings(macs)
		for _, mac := range macs {
			ap := siteBody.Devices.AP[mac]
			if ap == nil {
				ap = make(map[string]any)
				siteBody.Devices.AP[mac] = ap
			}
			ap["wlan"] = append(toStringList(ap["wlan"]), label)
		}
	}
}

// toStringList coerces a raw wlan-list value (nil, []string, or post-JSON []any)
// into []string for accumulation.
func toStringList(v any) []string {
	switch t := v.(type) {
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, e := range t {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
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
// Mist-canonical triple of (label list, label→template body map, label→assigned
// AP MACs). The template body is the JSON shape of a WLANProfile rendered back to
// a raw map so it drops straight into the Templates.WLAN section of the
// ImportFile envelope. The assignment map is non-empty only for Mist WLANs scoped
// to specific APs (apply_to=aps); the caller wires those into device-level wlan
// lists so apply resolves them back to the same ap_ids — a functional no-op.
// Meraki carries its availability inside the WLAN's vendor block instead.
func buildWLANsExport(cacheAccessor *vendors.CacheAccessor, siteID, siteSlug string, includeSecrets bool) ([]string, map[string]map[string]any, map[string][]string) {
	wlans := cacheAccessor.GetWLANsBySite(siteID)
	labels, profiles, vendorBlocks := synthesizeWLANLabels(wlans, siteSlug, includeSecrets)
	if len(profiles) == 0 {
		return labels, nil, nil
	}
	templates := make(map[string]map[string]any, len(profiles))
	for label, p := range profiles {
		m, err := profileToMap(p)
		if err != nil {
			logging.Warnf("Failed to serialize WLAN profile %q: %v", label, err)
			continue
		}
		// Pin the vendor's native identity (Meraki slot, raw band/auth, availability)
		// alongside the portable profile so apply round-trips without drift.
		for k, v := range vendorBlocks[label] {
			m[k] = v
		}
		templates[label] = m
	}

	// labels[i] corresponds to wlans[i] (synthesizeWLANLabels appends one label
	// per WLAN in order), so we can recover each label's source WLAN to resolve
	// Mist per-AP assignment without threading it through the pure synthesizer.
	idToMAC := apIDToMAC(cacheAccessor, siteID)
	assign := make(map[string][]string)
	for i, w := range wlans {
		if macs := mistAssignedMACs(w, idToMAC); len(macs) > 0 {
			assign[labels[i]] = macs
		}
	}
	return labels, templates, assign
}

// apIDToMAC builds a vendor-AP-ID → MAC map for a site, used to translate Mist
// WLAN ap_ids back into the MAC-keyed device-level wlan lists wifimgr authors.
func apIDToMAC(cacheAccessor *vendors.CacheAccessor, siteID string) map[string]string {
	out := make(map[string]string)
	for _, ap := range cacheAccessor.GetAllAPs() {
		if ap.SiteID == siteID && ap.ID != "" {
			out[ap.ID] = vendors.NormalizeMAC(ap.MAC)
		}
	}
	return out
}

// mistAssignedMACs returns the APs a Mist WLAN is restricted to (apply_to=aps),
// resolved from ap_ids to MACs. Returns nil for site-wide WLANs and non-Mist
// vendors (Meraki expresses availability via its vendor block, not placement).
func mistAssignedMACs(w *vendors.WLAN, idToMAC map[string]string) []string {
	if w.SourceVendor != "mist" {
		return nil
	}
	if applyTo, _ := w.Config["apply_to"].(string); applyTo != "aps" {
		return nil
	}
	var macs []string
	for _, id := range configStrings(w.Config, "ap_ids") {
		if mac, ok := idToMAC[id]; ok {
			macs = append(macs, mac)
		} else {
			logging.Warnf("Mist WLAN %q: ap_id %s not found among site APs; assignment dropped", w.SSID, id)
		}
	}
	return macs
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
// vendor-normalized WLANs and a site slug, produce label references, the
// matching WLANProfile map, and any per-label vendor blocks (e.g. the Meraki
// SSID slot) that pin the WLAN to its native identity. Separated so unit tests
// can drive it directly.
func synthesizeWLANLabels(vendorWLANs []*vendors.WLAN, siteSlug string, includeSecrets bool) ([]string, map[string]*config.WLANProfile, map[string]map[string]any) {
	if len(vendorWLANs) == 0 {
		return nil, nil, nil
	}

	labels := make([]string, 0, len(vendorWLANs))
	profiles := make(map[string]*config.WLANProfile, len(vendorWLANs))
	vendorBlocks := make(map[string]map[string]any, len(vendorWLANs))
	used := make(map[string]int) // collision counter keyed on bare label

	for _, w := range vendorWLANs {
		base := fmt.Sprintf("%s--%s", siteSlug, slug(w.SSID))
		label := base
		if n := used[base]; n > 0 {
			label = fmt.Sprintf("%s-%d", base, n+1)
		}
		used[base]++

		profiles[label] = convertVendorWLANToProfile(w, includeSecrets)
		if block := vendorBlockForWLAN(w); block != nil {
			vendorBlocks[label] = block
		}
		labels = append(labels, label)
	}

	return labels, profiles, vendorBlocks
}

// vendorBlockForWLAN captures the vendor-native attributes the portable
// WLANProfile can't represent losslessly, shaped as a template vendor block
// (e.g. "meraki:") that apply merges over the canonical fields. For Meraki this
// makes import → apply a functional no-op:
//   - number: the SSID slot (0-14), so apply rebinds to that exact slot.
//   - band/auth.type: the raw Meraki enum tokens, kept only when canonicalization
//     would otherwise lose them ("Dual band operation with Band Steering" → "dual",
//     "8021x-radius" → "eap"). The portable profile still carries the canonical
//     form for cross-vendor readability.
//   - availabilityTags/availableOnAllAps: the real Meraki availability model,
//     preserved verbatim so apply never invents synthetic tags.
//
// Returns nil for vendors with nothing to pin (e.g. Mist, whose WLANs are
// first-class objects keyed by stable UUID and assigned to APs via ap_ids).
func vendorBlockForWLAN(w *vendors.WLAN) map[string]any {
	if w.SourceVendor != "meraki" {
		return nil
	}

	block := map[string]any{}
	if slot, ok := merakiSlot(w); ok {
		block["number"] = slot
	}
	// Raw band/auth only when the canonical form drops information — otherwise the
	// canonical value already round-trips and the block stays minimal.
	if w.Band != "" && normalizeBand(w.Band) != w.Band {
		block["band"] = w.Band
	}
	if w.AuthType != "" && normalizeAuthType(w.AuthType) != w.AuthType {
		block["auth"] = map[string]any{"type": w.AuthType}
	}
	if tags := configStrings(w.Config, "availabilityTags"); len(tags) > 0 {
		block["availabilityTags"] = tags
	}
	if b, ok := configBool(w.Config, "availableOnAllAps"); ok {
		block["availableOnAllAps"] = b
	}

	if len(block) == 0 {
		return nil
	}
	return map[string]any{"meraki:": block}
}

// configStrings extracts a []string from a raw config value that may be []string
// or (after JSON round-trip) []any.
func configStrings(m map[string]any, key string) []string {
	switch v := m[key].(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

// configBool reads a bool config value (handles bool and *bool).
func configBool(m map[string]any, key string) (bool, bool) {
	switch v := m[key].(type) {
	case bool:
		return v, true
	case *bool:
		if v != nil {
			return *v, true
		}
	}
	return false, false
}

// merakiSlot extracts the SSID slot number from a Meraki WLAN, preferring the
// raw config field and falling back to the composite ID ("networkID:slot").
func merakiSlot(w *vendors.WLAN) (int, bool) {
	if n, ok := firstInt(w.Config, "number"); ok {
		return n, true
	}
	if i := strings.LastIndex(w.ID, ":"); i >= 0 {
		if n, err := strconv.Atoi(w.ID[i+1:]); err == nil {
			return n, true
		}
	}
	return 0, false
}

// convertVendorWLANToProfile maps a vendor-normalized WLAN into the portable
// WLANProfile shape the template system uses. Vendor-specific metadata
// (network IDs, ipAssignmentMode, etc.) is deliberately dropped here — this is
// the portable half. The one binding worth keeping, the Meraki SSID slot, rides
// alongside in a vendor block (see vendorBlockForWLAN), not in the profile. PSK
// and RADIUS secrets honor includeSecrets and are otherwise omitted.
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

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
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ravinald/wifimgr/internal/cmdutils"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/vendors"
	"github.com/ravinald/wifimgr/internal/xdg"
)

// TemplateImportType enumerates the vendor-level template kinds this command
// can materialize. Today only "wlan" is wired up; the rest are reserved so the
// arg surface stays stable while we add coverage.
type TemplateImportType string

const (
	TemplateTypeWLAN    TemplateImportType = "wlan"
	TemplateTypeRF      TemplateImportType = "rf"
	TemplateTypeDevice  TemplateImportType = "device"
	TemplateTypeGateway TemplateImportType = "gateway"
	TemplateTypeAll     TemplateImportType = "all"
)

// importAPITemplatesCmd imports vendor-level (org-scoped) template constructs
// into a standalone ImportFile. Scoped to a specific API because template
// namespaces live at the API/org boundary.
var importAPITemplatesCmd = &cobra.Command{
	Use:   "templates [target <api-label>] [type wlan|rf|device|gateway|all] [save] [file <filename>] [secrets]",
	Short: "Import vendor-level templates from API cache",
	Long: `Import vendor-level (org-scoped) templates from a specific API.

This command targets templates that live above the site boundary — Mist
org-level WLANs today, with room for RF, device, and gateway templates as
vendors expose them.

Basic Usage:
  wifimgr import api templates target mist-prod
  wifimgr import api templates target mist-prod save
  wifimgr import api templates target mist-prod type wlan save

With Custom Output File:
  wifimgr import api templates target mist-prod save file import/org-wlans.json
  wifimgr import api templates target mist-prod save file /tmp/templates.json

Arguments:
  target         Required. Keyword followed by the API label to target.
  type <kind>    Optional. Template kind to export (default: wlan). Currently
                 only "wlan" is supported; "rf", "device", "gateway", "all"
                 are reserved for later coverage.
  secrets        Optional. Include sensitive data (PSK, RADIUS secrets) —
                 redacted by default.
  save           Optional. Write to import file (default: print to STDOUT).
  file           Optional. Keyword followed by output filename (relative to
                 config_dir or absolute).

What it Does:
  1. Pulls the matching templates from the target API's cache
  2. Converts them into the wifimgr template schema (WLANProfile for WLANs)
  3. Emits a single ImportFile with Templates.<kind> populated
  4. Register the file in files.imports to have apply pick them up

Output Location:
  Without 'save': Prints JSON to STDOUT
  With 'save' (no file): <config_dir>/import/<kind>-template_<api>.json
  With 'save file': <config_dir>/<filename> (relative) or <filename> (absolute)

Vendor Notes:
  Meraki does not have org-level WLAN templates — its SSIDs live inside
  networks. This command reports nothing found and exits cleanly for
  Meraki targets.`,
	Args: func(cmd *cobra.Command, args []string) error {
		// Allow "help" as a special keyword
		for _, arg := range args {
			if strings.ToLower(arg) == "help" {
				return nil
			}
		}
		if len(args) < 1 {
			return fmt.Errorf("requires at least 'target <api-label>' — got %d args", len(args))
		}
		return nil
	},
	RunE: runImportAPITemplates,
}

func init() {
	importAPICmd.AddCommand(importAPITemplatesCmd)
}

// importTemplatesArgs holds parsed arguments for the import api templates command.
type importTemplatesArgs struct {
	apiLabel       string
	templateType   TemplateImportType
	includeSecrets bool
	saveMode       bool
	outputFile     string
}

func parseImportTemplatesArgs(args []string) (*importTemplatesArgs, error) {
	result := &importTemplatesArgs{
		templateType: TemplateTypeWLAN,
	}

	for i := 0; i < len(args); i++ {
		arg := strings.ToLower(args[i])
		switch arg {
		case "target":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("'target' requires an API label")
			}
			result.apiLabel = args[i+1]
			i++
		case "type":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("'type' requires a template kind (wlan, rf, device, gateway, all)")
			}
			kind := strings.ToLower(args[i+1])
			switch kind {
			case "wlan":
				result.templateType = TemplateTypeWLAN
			case "rf", "device", "gateway", "all":
				return nil, fmt.Errorf("template type %q is reserved but not yet implemented; only 'wlan' is supported today", kind)
			default:
				return nil, fmt.Errorf("invalid template type %q - must be one of: wlan, rf, device, gateway, all", kind)
			}
			i++
		case "secrets":
			result.includeSecrets = true
		case "save":
			result.saveMode = true
		case "file":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("'file' requires a filename")
			}
			result.outputFile = args[i+1]
			i++
		case "help":
			// Handled in RunE
		default:
			return nil, fmt.Errorf("unknown argument: %s", args[i])
		}
	}

	if result.apiLabel == "" {
		return nil, fmt.Errorf("'target <api-label>' is required")
	}
	if result.outputFile != "" && !result.saveMode {
		return nil, fmt.Errorf("'file' requires 'save' to be specified")
	}

	return result, nil
}

func runImportAPITemplates(cmd *cobra.Command, args []string) error {
	for _, arg := range args {
		if strings.ToLower(arg) == "help" {
			return cmd.Help()
		}
	}

	logger := logging.GetLogger()
	logger.Info("Executing import api templates command")

	parsed, err := parseImportTemplatesArgs(args)
	if err != nil {
		return err
	}

	registry := GetAPIRegistry()
	if registry == nil || !registry.HasAPI(parsed.apiLabel) {
		return FormatAPINotFoundError(parsed.apiLabel)
	}

	cacheAccessor, err := cmdutils.GetCacheAccessor()
	if err != nil {
		return fmt.Errorf("failed to get cache accessor: %w", err)
	}

	// Build the envelope. For now only WLANs are supported.
	env, err := buildTemplatesExportData(cacheAccessor, parsed.apiLabel, parsed.templateType, parsed.includeSecrets)
	if err != nil {
		return fmt.Errorf("failed to build templates export: %w", err)
	}

	if env == nil || env.Templates == nil || isTemplatesEmpty(env.Templates) {
		fmt.Printf("No org-level %s templates found for API %q.\n", parsed.templateType, parsed.apiLabel)
		fmt.Println("  (This is expected for Meraki targets — Meraki has no org-level WLAN templates.)")
		return nil
	}

	configDir := viper.GetString("files.config_dir")
	outputPath := resolveTemplateImportPath(parsed.outputFile, configDir, parsed.apiLabel, parsed.templateType)

	if !parsed.saveMode {
		data, err := json.MarshalIndent(env, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal data: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	if _, exists := loadExistingImport(outputPath); exists {
		if !confirmOverwrite(outputPath) {
			fmt.Println("Import cancelled")
			return nil
		}
	}

	if err := writeImportFile(outputPath, env); err != nil {
		return fmt.Errorf("failed to write template import: %w", err)
	}
	printActivationHint(outputPath, configDir)
	return nil
}

// resolveTemplateImportPath picks the on-disk location for the template
// import file. <configDir>/import/<kind>-template_<api>.json by convention.
func resolveTemplateImportPath(outputFile, configDir, apiLabel string, kind TemplateImportType) string {
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
	return filepath.Join(baseDir, "import", fmt.Sprintf("%s-template_%s.json", kind, apiLabel))
}

// buildTemplatesExportData materializes an ImportFile envelope containing
// vendor-level templates for the given API.
func buildTemplatesExportData(cacheAccessor *vendors.CacheAccessor, apiLabel string, kind TemplateImportType, includeSecrets bool) (*importEnvelope, error) {
	env := &importEnvelope{
		Version: 1,
		Source: &importSourceExport{
			API:        apiLabel,
			Kind:       fmt.Sprintf("%s-templates", kind),
			ImportedAt: time.Now().UTC(),
		},
	}

	switch kind {
	case TemplateTypeWLAN:
		wlans := collectOrgWLANs(cacheAccessor, apiLabel)
		templates := synthesizeOrgWLANTemplates(wlans, includeSecrets)
		if len(templates) == 0 {
			return env, nil
		}
		env.Templates = &templatesEnvelope{WLAN: templates}
		return env, nil
	default:
		return nil, fmt.Errorf("template type %q is not implemented", kind)
	}
}

// collectOrgWLANs returns the org-level WLANs for the target API. "Org-level"
// means SiteID is empty — those are the templates that apply across sites
// (Mist's notion; Meraki has none, so this naturally returns empty).
func collectOrgWLANs(cacheAccessor *vendors.CacheAccessor, apiLabel string) []*vendors.WLAN {
	var result []*vendors.WLAN
	for _, w := range cacheAccessor.GetAllWLANs() {
		if w == nil {
			continue
		}
		if w.SourceAPI != apiLabel {
			continue
		}
		if w.SiteID != "" {
			continue
		}
		result = append(result, w)
	}
	return result
}

// synthesizeOrgWLANTemplates builds the Templates.WLAN map for org-level
// WLANs. Labels are bare slugs of the SSID — no site prefix, because the
// scope IS the whole org. Collisions append a counter suffix.
func synthesizeOrgWLANTemplates(ws []*vendors.WLAN, includeSecrets bool) map[string]map[string]any {
	if len(ws) == 0 {
		return nil
	}
	out := make(map[string]map[string]any, len(ws))
	used := make(map[string]int)
	for _, w := range ws {
		base := slug(w.SSID)
		if base == "" {
			logging.Warnf("[import templates] skipping WLAN with unnamed/untranslatable SSID: %q", w.SSID)
			continue
		}
		label := base
		if n := used[base]; n > 0 {
			label = fmt.Sprintf("%s-%d", base, n+1)
		}
		used[base]++

		profile := convertVendorWLANToProfile(w, includeSecrets)
		m, err := profileToMap(profile)
		if err != nil {
			logging.Warnf("[import templates] failed to serialize WLAN %q: %v", w.SSID, err)
			continue
		}
		out[label] = m
	}
	return out
}

// isTemplatesEmpty reports whether the Templates envelope has any content
// across the supported kinds. Helper for the "nothing found" branch.
func isTemplatesEmpty(t *templatesEnvelope) bool {
	if t == nil {
		return true
	}
	return len(t.WLAN) == 0 && len(t.Radio) == 0 && len(t.Device) == 0
}

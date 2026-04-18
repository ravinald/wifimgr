package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// ImportFile is the shape written by `wifimgr import ...` commands. It
// captures a snapshot of vendor state translated into the wifimgr schema
// and is intended to be directly loadable — register one under
// `files.imports` in the main config and the loader dispatches:
//
//   - Config.Sites → same registry as files.site_configs
//   - Templates    → same global TemplateStore as files.templates
//
// Both sections are optional and independent. A per-site import has
// Config populated and Templates holding only site-local WLANs; a
// vendor-template import (e.g. Mist org-level WLAN templates) has
// Templates populated and Config nil.
type ImportFile struct {
	Version   int                  `json:"version"`
	Source    ImportSource         `json:"source,omitempty"`
	Config    *SiteConfigWrapper   `json:"config,omitempty"`
	Templates *TemplateDefinitions `json:"templates,omitempty"`
}

// ImportSource is human-facing provenance metadata. The loader ignores
// it; it exists so an operator who renames or moves the file can still
// trace it back to its origin.
type ImportSource struct {
	API        string    `json:"api,omitempty"`
	Site       string    `json:"site,omitempty"`
	SiteID     string    `json:"site_id,omitempty"`
	Kind       string    `json:"kind,omitempty"` // "site" or "wlan-templates" etc.
	ImportedAt time.Time `json:"imported_at,omitempty"`
}

// validImportFileKeys are the only keys allowed at the top level of an
// ImportFile. We validate the envelope strictly but leave the payload
// permissive because real imported site configs carry vendor-snapshot
// provenance (serial, model, site_id) in the APConfig / SwitchConfig
// sections that isn't part of the canonical intent shape.
var validImportFileKeys = map[string]struct{}{
	"version":   {},
	"source":    {},
	"config":    {},
	"templates": {},
}

// LoadImportFile reads and parses an ImportFile from disk. The envelope
// is validated against validImportFileKeys so a typo at the top level
// (e.g. "template" instead of "templates") surfaces as a parse error.
// The payload is decoded permissively.
func LoadImportFile(path string) (*ImportFile, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- path from operator-controlled config
	if err != nil {
		return nil, fmt.Errorf("failed to read import file %s: %w", path, err)
	}

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("failed to parse import file %s: %w", path, err)
	}
	for key := range envelope {
		if _, ok := validImportFileKeys[key]; !ok {
			return nil, fmt.Errorf("import file %s: unknown top-level field %q (expected one of: version, source, config, templates)", path, key)
		}
	}

	var file ImportFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("failed to parse import file %s: %w", path, err)
	}
	return &file, nil
}

// MergeImportTemplates merges the `Templates` section of an ImportFile
// into a TemplateStore. Later entries overwrite earlier ones with the
// same label (same policy as hand-authored template files). Templates
// are keyed by bare label; the caller is responsible for any
// vendor-scoping that belongs in the label itself.
func MergeImportTemplates(store *TemplateStore, imp *ImportFile) {
	if store == nil || imp == nil || imp.Templates == nil {
		return
	}
	for name, tmpl := range imp.Templates.Radio {
		store.Radio[name] = tmpl
	}
	for name, tmpl := range imp.Templates.WLAN {
		store.WLAN[name] = tmpl
	}
	for name, tmpl := range imp.Templates.Device {
		store.Device[name] = tmpl
	}
}

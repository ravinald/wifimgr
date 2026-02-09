package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ravinald/wifimgr/internal/logging"
)

// TemplateStore holds all loaded templates organized by type
type TemplateStore struct {
	Radio  map[string]map[string]any // name -> config
	WLAN   map[string]map[string]any // name -> config
	Device map[string]map[string]any // name -> config
}

// TemplateFile represents the structure of a template file
type TemplateFile struct {
	Version   int                 `json:"version"`
	Templates TemplateDefinitions `json:"templates"`
}

// TemplateDefinitions groups templates by type
type TemplateDefinitions struct {
	Radio  map[string]map[string]any `json:"radio,omitempty"`
	WLAN   map[string]map[string]any `json:"wlan,omitempty"`
	Device map[string]map[string]any `json:"device,omitempty"`
}

// NewTemplateStore creates an empty template store
func NewTemplateStore() *TemplateStore {
	return &TemplateStore{
		Radio:  make(map[string]map[string]any),
		WLAN:   make(map[string]map[string]any),
		Device: make(map[string]map[string]any),
	}
}

// LoadTemplates loads templates from the given file paths
// Paths can be relative (resolved against configDir) or absolute
func LoadTemplates(paths []string, configDir string) (*TemplateStore, error) {
	store := NewTemplateStore()

	if len(paths) == 0 {
		logging.Debugf("No template files configured")
		return store, nil
	}

	for _, path := range paths {
		// Resolve relative paths
		filePath := path
		if !filepath.IsAbs(path) && configDir != "" {
			filePath = filepath.Join(configDir, path)
		}

		if err := store.loadFromFile(filePath); err != nil {
			return nil, fmt.Errorf("failed to load template file %s: %w", path, err)
		}
	}

	logging.Debugf("Loaded templates: %d radio, %d wlan, %d device",
		len(store.Radio), len(store.WLAN), len(store.Device))

	return store, nil
}

// loadFromFile loads templates from a single file
func (s *TemplateStore) loadFromFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var templateFile TemplateFile
	if err := json.Unmarshal(data, &templateFile); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Validate version
	if templateFile.Version != 1 {
		logging.Warnf("Template file %s has version %d, expected 1", filePath, templateFile.Version)
	}

	// Merge templates into store
	for name, config := range templateFile.Templates.Radio {
		if _, exists := s.Radio[name]; exists {
			logging.Warnf("Radio template '%s' defined multiple times, later definition wins", name)
		}
		s.Radio[name] = config
		logging.Debugf("Loaded radio template: %s", name)
	}

	for name, config := range templateFile.Templates.WLAN {
		if _, exists := s.WLAN[name]; exists {
			logging.Warnf("WLAN template '%s' defined multiple times, later definition wins", name)
		}
		s.WLAN[name] = config
		logging.Debugf("Loaded WLAN template: %s", name)
	}

	for name, config := range templateFile.Templates.Device {
		if _, exists := s.Device[name]; exists {
			logging.Warnf("Device template '%s' defined multiple times, later definition wins", name)
		}
		s.Device[name] = config
		logging.Debugf("Loaded device template: %s", name)
	}

	return nil
}

// GetRadioTemplate retrieves a radio template by name
func (s *TemplateStore) GetRadioTemplate(name string) (map[string]any, bool) {
	t, ok := s.Radio[name]
	return t, ok
}

// GetWLANTemplate retrieves a WLAN template by name
func (s *TemplateStore) GetWLANTemplate(name string) (map[string]any, bool) {
	t, ok := s.WLAN[name]
	return t, ok
}

// GetDeviceTemplate retrieves a device template by name
func (s *TemplateStore) GetDeviceTemplate(name string) (map[string]any, bool) {
	t, ok := s.Device[name]
	return t, ok
}

// IsEmpty returns true if no templates are loaded
func (s *TemplateStore) IsEmpty() bool {
	return len(s.Radio) == 0 && len(s.WLAN) == 0 && len(s.Device) == 0
}

// ListTemplates returns all template names by type
func (s *TemplateStore) ListTemplates() map[string][]string {
	result := make(map[string][]string)

	radioNames := make([]string, 0, len(s.Radio))
	for name := range s.Radio {
		radioNames = append(radioNames, name)
	}
	result["radio"] = radioNames

	wlanNames := make([]string, 0, len(s.WLAN))
	for name := range s.WLAN {
		wlanNames = append(wlanNames, name)
	}
	result["wlan"] = wlanNames

	deviceNames := make([]string, 0, len(s.Device))
	for name := range s.Device {
		deviceNames = append(deviceNames, name)
	}
	result["device"] = deviceNames

	return result
}

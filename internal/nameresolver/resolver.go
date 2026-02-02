// Package nameresolver provides name-to-ID resolution for configuration references.
// It allows operators to use human-readable names instead of opaque IDs for reference
// fields like device profiles, RF profiles, and maps.
package nameresolver

import (
	"fmt"
	"strings"
	"sync"

	"github.com/ravinald/wifimgr/internal/vendors"
)

// Resolver provides name-to-ID lookups using cached data.
type Resolver struct {
	mu sync.RWMutex

	// Per-API profile maps (api_label -> name -> ID)
	deviceProfiles map[string]map[string]string
	rfProfiles     map[string]map[string]string

	// Site-level maps (site -> name -> ID)
	siteMaps map[string]map[string]string
}

// NewResolver creates a new empty resolver.
// Call LoadFromCache to populate with data.
func NewResolver() *Resolver {
	return &Resolver{
		deviceProfiles: make(map[string]map[string]string),
		rfProfiles:     make(map[string]map[string]string),
		siteMaps:       make(map[string]map[string]string),
	}
}

// LoadDeviceProfiles populates device profile mappings from cache data.
func (r *Resolver) LoadDeviceProfiles(apiLabel string, profiles []*vendors.DeviceProfile) {
	r.mu.Lock()
	defer r.mu.Unlock()

	nameToID := make(map[string]string, len(profiles))
	for _, p := range profiles {
		if p.Name != "" && p.ID != "" {
			nameToID[p.Name] = p.ID
		}
	}
	r.deviceProfiles[apiLabel] = nameToID
}

// LoadRFProfiles populates RF profile mappings (Meraki-specific).
func (r *Resolver) LoadRFProfiles(apiLabel string, profiles map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.rfProfiles[apiLabel] = profiles
}

// LoadMaps populates map/floorplan mappings for a site.
func (r *Resolver) LoadMaps(siteID string, maps map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.siteMaps[siteID] = maps
}

// ResolveDeviceProfile returns the ID for a device profile name.
// Returns an error if the name is not found.
func (r *Resolver) ResolveDeviceProfile(apiLabel, name string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	profiles, ok := r.deviceProfiles[apiLabel]
	if !ok {
		return "", fmt.Errorf("no device profiles loaded for API %q", apiLabel)
	}

	id, ok := profiles[name]
	if !ok {
		return "", &ResolutionError{
			RefType: "device profile",
			Name:    name,
			API:     apiLabel,
		}
	}

	return id, nil
}

// ResolveRFProfile returns the ID for an RF profile name (Meraki).
func (r *Resolver) ResolveRFProfile(apiLabel, name string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	profiles, ok := r.rfProfiles[apiLabel]
	if !ok {
		return "", fmt.Errorf("no RF profiles loaded for API %q", apiLabel)
	}

	id, ok := profiles[name]
	if !ok {
		return "", &ResolutionError{
			RefType: "RF profile",
			Name:    name,
			API:     apiLabel,
		}
	}

	return id, nil
}

// ResolveMap returns the ID for a map/floorplan name.
func (r *Resolver) ResolveMap(siteID, name string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	maps, ok := r.siteMaps[siteID]
	if !ok {
		return "", fmt.Errorf("no maps loaded for site %q", siteID)
	}

	id, ok := maps[name]
	if !ok {
		return "", &ResolutionError{
			RefType: "map",
			Name:    name,
			Site:    siteID,
		}
	}

	return id, nil
}

// ResolveAPConfig resolves all name references in an APDeviceConfig.
// It modifies the config in place, replacing *_name fields with resolved *_id values.
func (r *Resolver) ResolveAPConfig(cfg *vendors.APDeviceConfig, apiLabel, siteID string) error {
	if cfg == nil {
		return nil
	}

	// Resolve device profile
	if cfg.DeviceProfileName != "" {
		id, err := r.ResolveDeviceProfile(apiLabel, cfg.DeviceProfileName)
		if err != nil {
			return err
		}
		cfg.DeviceProfileID = id
		cfg.DeviceProfileName = "" // Clear the name field
	}

	// Resolve map
	if cfg.MapName != "" {
		id, err := r.ResolveMap(siteID, cfg.MapName)
		if err != nil {
			return err
		}
		cfg.MapID = id
		cfg.MapName = "" // Clear the name field
	}

	// Resolve Meraki RF profile if present
	if cfg.RadioConfig != nil && cfg.RadioConfig.Meraki != nil {
		if rfName, ok := cfg.RadioConfig.Meraki["rf_profile_name"].(string); ok && rfName != "" {
			id, err := r.ResolveRFProfile(apiLabel, rfName)
			if err != nil {
				return err
			}
			cfg.RadioConfig.Meraki["rf_profile_id"] = id
			delete(cfg.RadioConfig.Meraki, "rf_profile_name")
		}
	}

	return nil
}

// ListDeviceProfiles returns all known device profile names for an API.
func (r *Resolver) ListDeviceProfiles(apiLabel string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	profiles, ok := r.deviceProfiles[apiLabel]
	if !ok {
		return nil
	}

	names := make([]string, 0, len(profiles))
	for name := range profiles {
		names = append(names, name)
	}
	return names
}

// ListRFProfiles returns all known RF profile names for an API.
func (r *Resolver) ListRFProfiles(apiLabel string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	profiles, ok := r.rfProfiles[apiLabel]
	if !ok {
		return nil
	}

	names := make([]string, 0, len(profiles))
	for name := range profiles {
		names = append(names, name)
	}
	return names
}

// ListMaps returns all known map names for a site.
func (r *Resolver) ListMaps(siteID string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	maps, ok := r.siteMaps[siteID]
	if !ok {
		return nil
	}

	names := make([]string, 0, len(maps))
	for name := range maps {
		names = append(names, name)
	}
	return names
}

// SuggestSimilar returns similar names for a failed lookup.
// Useful for providing helpful error messages.
func (r *Resolver) SuggestSimilar(refType, name, apiOrSite string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var candidates []string

	switch refType {
	case "device profile":
		if profiles, ok := r.deviceProfiles[apiOrSite]; ok {
			for n := range profiles {
				candidates = append(candidates, n)
			}
		}
	case "RF profile":
		if profiles, ok := r.rfProfiles[apiOrSite]; ok {
			for n := range profiles {
				candidates = append(candidates, n)
			}
		}
	case "map":
		if maps, ok := r.siteMaps[apiOrSite]; ok {
			for n := range maps {
				candidates = append(candidates, n)
			}
		}
	}

	return findSimilar(name, candidates, 3)
}

// findSimilar finds the most similar strings to target from candidates.
func findSimilar(target string, candidates []string, maxResults int) []string {
	if len(candidates) == 0 {
		return nil
	}

	type scored struct {
		name  string
		score int
	}

	targetLower := strings.ToLower(target)
	var scored_items []scored

	for _, c := range candidates {
		cLower := strings.ToLower(c)
		score := 0

		// Exact prefix match
		if strings.HasPrefix(cLower, targetLower) {
			score += 100
		}

		// Contains match
		if strings.Contains(cLower, targetLower) {
			score += 50
		}

		// Word match
		targetWords := strings.Fields(targetLower)
		for _, w := range targetWords {
			if strings.Contains(cLower, w) {
				score += 10
			}
		}

		if score > 0 {
			scored_items = append(scored_items, scored{name: c, score: score})
		}
	}

	// Sort by score descending
	for i := 0; i < len(scored_items); i++ {
		for j := i + 1; j < len(scored_items); j++ {
			if scored_items[j].score > scored_items[i].score {
				scored_items[i], scored_items[j] = scored_items[j], scored_items[i]
			}
		}
	}

	result := make([]string, 0, maxResults)
	for i := 0; i < len(scored_items) && i < maxResults; i++ {
		result = append(result, scored_items[i].name)
	}

	return result
}

// ResolutionError represents a failed name resolution.
type ResolutionError struct {
	RefType string // "device profile", "RF profile", "map"
	Name    string
	API     string
	Site    string
}

func (e *ResolutionError) Error() string {
	if e.API != "" {
		return fmt.Sprintf("%s %q not found for API %q", e.RefType, e.Name, e.API)
	}
	if e.Site != "" {
		return fmt.Sprintf("%s %q not found in site %q", e.RefType, e.Name, e.Site)
	}
	return fmt.Sprintf("%s %q not found", e.RefType, e.Name)
}

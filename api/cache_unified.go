package api

import (
	"fmt"
)

// UnifiedCache implements Cacher using the existing CacheManager
type UnifiedCache struct {
	*CacheManager
	orgID string
	dirty bool
}

// Load loads the cache from disk
func (uc *UnifiedCache) Load() error {
	return uc.Initialize()
}

// Save saves the cache to disk
func (uc *UnifiedCache) Save() error {
	if err := uc.SaveCache(); err != nil {
		return err
	}
	uc.dirty = false
	return nil
}

// Clear clears the cache
func (uc *UnifiedCache) Clear() error {
	cache := &Cache{
		Version: 1,
		Orgs:    make(map[string]*OrgData),
	}
	cache.Orgs[uc.orgID] = createEmptyOrgData()
	if err := uc.ReplaceCache(cache); err != nil {
		return fmt.Errorf("failed to replace cache: %w", err)
	}
	uc.dirty = true
	return nil
}

// GetOrgData returns organization data, creating it if necessary
func (uc *UnifiedCache) GetOrgData(orgID string) (*OrgData, error) {
	cache, err := uc.GetCache()
	if err != nil {
		return nil, err
	}

	if orgData, exists := cache.Orgs[orgID]; exists {
		return orgData, nil
	}

	// Auto-create if doesn't exist
	newOrgData := createEmptyOrgData()
	cache.Orgs[orgID] = newOrgData
	uc.dirty = true
	return newOrgData, nil
}

// SetOrgData sets organization data
func (uc *UnifiedCache) SetOrgData(orgID string, data *OrgData) error {
	cache, err := uc.GetCache()
	if err != nil {
		return err
	}

	cache.Orgs[orgID] = data
	uc.dirty = true
	return nil
}

// GetSite retrieves a site by name or ID
func (uc *UnifiedCache) GetSite(identifier string) (*MistSite, error) {
	orgData, err := uc.GetOrgData(uc.orgID)
	if err != nil {
		return nil, err
	}

	for i := range orgData.Sites.Info {
		site := &orgData.Sites.Info[i]
		if site.Name != nil && *site.Name == identifier {
			return site, nil
		}
		if site.ID != nil && *site.ID == identifier {
			return site, nil
		}
	}

	return nil, fmt.Errorf("site not found: %s", identifier)
}

// GetAllSites returns all sites
func (uc *UnifiedCache) GetAllSites() ([]*MistSite, error) {
	orgData, err := uc.GetOrgData(uc.orgID)
	if err != nil {
		return nil, err
	}

	sites := make([]*MistSite, len(orgData.Sites.Info))
	for i := range orgData.Sites.Info {
		sites[i] = &orgData.Sites.Info[i]
	}
	return sites, nil
}

// UpdateSite updates or adds a site
func (uc *UnifiedCache) UpdateSite(site *MistSite) error {
	orgData, err := uc.GetOrgData(uc.orgID)
	if err != nil {
		return err
	}

	// Find and update existing site
	for i := range orgData.Sites.Info {
		existing := &orgData.Sites.Info[i]
		if (site.ID != nil && existing.ID != nil && *site.ID == *existing.ID) ||
			(site.Name != nil && existing.Name != nil && *site.Name == *existing.Name) {
			orgData.Sites.Info[i] = *site
			uc.dirty = true
			return nil
		}
	}

	// Add new site
	orgData.Sites.Info = append(orgData.Sites.Info, *site)
	uc.dirty = true
	return nil
}

// GetMetadata returns cache metadata (deprecated - metadata now in separate file)
func (uc *UnifiedCache) GetMetadata() *CacheMetadata {
	// Return minimal metadata for compatibility
	return &CacheMetadata{
		Version: "1.0",
		OrgID:   uc.orgID,
	}
}

// UpdateMetadata updates cache metadata (deprecated - metadata now in separate file)
func (uc *UnifiedCache) UpdateMetadata(_ *CacheMetadata) {
	// No-op since metadata is no longer stored in cache
	// Mark as dirty to trigger save which will update the separate metadata file
	uc.dirty = true
}

// IsDirty returns whether the cache has unsaved changes
func (uc *UnifiedCache) IsDirty() bool {
	return uc.dirty
}

// MarkDirty marks the cache as having unsaved changes
func (uc *UnifiedCache) MarkDirty() {
	uc.dirty = true
}

// GetPath returns the cache file path
func (uc *UnifiedCache) GetPath() string {
	return uc.cachePath
}

// createEmptyOrgData creates an empty OrgData structure with all fields initialized
func createEmptyOrgData() *OrgData {
	orgData := &OrgData{}
	orgData.Sites.Info = []MistSite{}
	orgData.Sites.Settings = []SiteSetting{}
	orgData.Templates.RF = []MistRFTemplate{}
	orgData.Templates.Gateway = []MistGatewayTemplate{}
	orgData.Templates.WLAN = []MistWLANTemplate{}
	orgData.Networks = []MistNetwork{}
	orgData.WLANs.Org = []MistWLAN{}
	orgData.WLANs.Sites = make(map[string][]MistWLAN)
	orgData.Inventory.AP = make(map[string]APDevice)
	orgData.Inventory.Switch = make(map[string]MistSwitchDevice)
	orgData.Inventory.Gateway = make(map[string]MistGatewayDevice)
	orgData.Profiles.Devices = []DeviceProfile{}
	orgData.Profiles.Details = []map[string]any{}
	orgData.Configs.AP = make(map[string]APConfig)
	orgData.Configs.Switch = make(map[string]SwitchConfig)
	orgData.Configs.Gateway = make(map[string]GatewayConfig)
	return orgData
}

// Verify UnifiedCache implements Cacher at compile time
var _ Cacher = (*UnifiedCache)(nil)

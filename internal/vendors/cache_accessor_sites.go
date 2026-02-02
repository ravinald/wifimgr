package vendors

import "fmt"

// Site lookups

// GetSiteByID returns a site by its ID.
func (ca *CacheAccessor) GetSiteByID(id string) (*SiteInfo, error) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	site, ok := ca.indexes.SitesByID[id]
	if !ok {
		return nil, fmt.Errorf("site not found: %s", id)
	}
	return site, nil
}

// GetSiteByName returns a site by its name.
func (ca *CacheAccessor) GetSiteByName(name string) (*SiteInfo, error) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	site, ok := ca.indexes.SitesByName[name]
	if !ok {
		return nil, fmt.Errorf("site not found: %s", name)
	}
	return site, nil
}

// GetSiteByNameAndAPI returns a site by its name from a specific API.
func (ca *CacheAccessor) GetSiteByNameAndAPI(name, apiLabel string) (*SiteInfo, error) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	// First check if the site exists at all
	site, ok := ca.indexes.SitesByName[name]
	if !ok {
		return nil, fmt.Errorf("site not found: %s", name)
	}

	// Check if it's from the requested API
	if site.SourceAPI != apiLabel {
		return nil, fmt.Errorf("site '%s' not found in API '%s' (found in API '%s')", name, apiLabel, site.SourceAPI)
	}

	return site, nil
}

// GetAllSites returns all sites from all APIs.
func (ca *CacheAccessor) GetAllSites() []*SiteInfo {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	sites := make([]*SiteInfo, 0, len(ca.indexes.SitesByID))
	for _, site := range ca.indexes.SitesByID {
		sites = append(sites, site)
	}
	return sites
}

// GetNetworkByID returns a network by its ID.
// In the multi-vendor model, networks are unified with sites.
// For Meraki, networks ARE sites. For Mist, this delegates to site lookup.
func (ca *CacheAccessor) GetNetworkByID(id string) (*SiteInfo, error) {
	return ca.GetSiteByID(id)
}

// GetNetworkByName returns a network by its name.
func (ca *CacheAccessor) GetNetworkByName(name string) (*SiteInfo, error) {
	return ca.GetSiteByName(name)
}

// Template lookups

// GetRFTemplateByID returns an RF template by its ID.
func (ca *CacheAccessor) GetRFTemplateByID(id string) (*RFTemplate, error) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	tmpl, ok := ca.indexes.RFTemplatesByID[id]
	if !ok {
		return nil, fmt.Errorf("RF template not found: %s", id)
	}
	return tmpl, nil
}

// GetRFTemplateByName returns an RF template by its name.
func (ca *CacheAccessor) GetRFTemplateByName(name string) (*RFTemplate, error) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	tmpl, ok := ca.indexes.RFTemplatesByName[name]
	if !ok {
		return nil, fmt.Errorf("RF template not found: %s", name)
	}
	return tmpl, nil
}

// GetAllRFTemplates returns all RF templates from all APIs.
func (ca *CacheAccessor) GetAllRFTemplates() []*RFTemplate {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	templates := make([]*RFTemplate, 0, len(ca.indexes.RFTemplatesByID))
	for _, tmpl := range ca.indexes.RFTemplatesByID {
		templates = append(templates, tmpl)
	}
	return templates
}

// GetGWTemplateByID returns a Gateway template by its ID.
func (ca *CacheAccessor) GetGWTemplateByID(id string) (*GatewayTemplate, error) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	tmpl, ok := ca.indexes.GWTemplatesByID[id]
	if !ok {
		return nil, fmt.Errorf("gateway template not found: %s", id)
	}
	return tmpl, nil
}

// GetGWTemplateByName returns a Gateway template by its name.
func (ca *CacheAccessor) GetGWTemplateByName(name string) (*GatewayTemplate, error) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	tmpl, ok := ca.indexes.GWTemplatesByName[name]
	if !ok {
		return nil, fmt.Errorf("gateway template not found: %s", name)
	}
	return tmpl, nil
}

// GetWLANTemplateByID returns a WLAN template by its ID.
func (ca *CacheAccessor) GetWLANTemplateByID(id string) (*WLANTemplate, error) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	tmpl, ok := ca.indexes.WLANTemplatesByID[id]
	if !ok {
		return nil, fmt.Errorf("WLAN template not found: %s", id)
	}
	return tmpl, nil
}

// GetWLANTemplateByName returns a WLAN template by its name.
func (ca *CacheAccessor) GetWLANTemplateByName(name string) (*WLANTemplate, error) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	tmpl, ok := ca.indexes.WLANTemplatesByName[name]
	if !ok {
		return nil, fmt.Errorf("WLAN template not found: %s", name)
	}
	return tmpl, nil
}

// Device Profile lookups

// GetDeviceProfileByID returns a device profile by its ID.
func (ca *CacheAccessor) GetDeviceProfileByID(id string) (*DeviceProfile, error) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	profile, ok := ca.indexes.DeviceProfilesByID[id]
	if !ok {
		return nil, fmt.Errorf("device profile not found: %s", id)
	}
	return profile, nil
}

// GetDeviceProfileByName returns a device profile by its name.
func (ca *CacheAccessor) GetDeviceProfileByName(name string) (*DeviceProfile, error) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	profile, ok := ca.indexes.DeviceProfilesByName[name]
	if !ok {
		return nil, fmt.Errorf("device profile not found: %s", name)
	}
	return profile, nil
}

// GetAllDeviceProfiles returns all device profiles from all APIs.
func (ca *CacheAccessor) GetAllDeviceProfiles() []*DeviceProfile {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	profiles := make([]*DeviceProfile, 0, len(ca.indexes.DeviceProfilesByID))
	for _, profile := range ca.indexes.DeviceProfilesByID {
		profiles = append(profiles, profile)
	}
	return profiles
}

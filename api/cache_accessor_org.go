package api

import (
	"fmt"
)

// GetOrgByName retrieves an organization by name with O(1) lookup
func (ca *CacheAccessorImpl) GetOrgByName(name string) (*OrgStats, error) {
	indexes, err := ca.manager.GetIndexes()
	if err != nil {
		return nil, err
	}

	org, exists := indexes.OrgsByName[name]
	if !exists {
		return nil, fmt.Errorf("organization not found: %s", name)
	}

	return org, nil
}

// GetOrgByID retrieves an organization by ID with O(1) lookup
func (ca *CacheAccessorImpl) GetOrgByID(id string) (*OrgStats, error) {
	indexes, err := ca.manager.GetIndexes()
	if err != nil {
		return nil, err
	}

	org, exists := indexes.OrgsByID[id]
	if !exists {
		return nil, fmt.Errorf("organization not found: %s", id)
	}

	return org, nil
}

// GetAllOrgs returns all organizations
func (ca *CacheAccessorImpl) GetAllOrgs() ([]*OrgStats, error) {
	cache, err := ca.manager.GetCache()
	if err != nil {
		return nil, err
	}

	orgs := make([]*OrgStats, 0, len(cache.Orgs))
	for _, orgData := range cache.Orgs {
		if orgData.OrgStats != nil {
			orgs = append(orgs, orgData.OrgStats)
		}
	}

	return orgs, nil
}

// GetSiteByName retrieves a site by name with O(1) lookup
func (ca *CacheAccessorImpl) GetSiteByName(name string) (*MistSite, error) {
	indexes, err := ca.manager.GetIndexes()
	if err != nil {
		return nil, err
	}

	site, exists := indexes.SitesByName[name]
	if !exists {
		return nil, fmt.Errorf("site not found: %s", name)
	}

	return site, nil
}

// GetSiteByID retrieves a site by ID with O(1) lookup
func (ca *CacheAccessorImpl) GetSiteByID(id string) (*MistSite, error) {
	indexes, err := ca.manager.GetIndexes()
	if err != nil {
		return nil, err
	}

	site, exists := indexes.SitesByID[id]
	if !exists {
		return nil, fmt.Errorf("site not found: %s", id)
	}

	return site, nil
}

// GetAllSites returns all sites
func (ca *CacheAccessorImpl) GetAllSites() ([]*MistSite, error) {
	cache, err := ca.manager.GetCache()
	if err != nil {
		return nil, err
	}

	var sites []*MistSite
	for _, orgData := range cache.Orgs {
		for i := range orgData.Sites.Info {
			sites = append(sites, &orgData.Sites.Info[i])
		}
	}

	return sites, nil
}

// GetSiteSettingBySiteID retrieves a site setting by site ID with O(1) lookup
func (ca *CacheAccessorImpl) GetSiteSettingBySiteID(siteID string) (*SiteSetting, error) {
	indexes, err := ca.manager.GetIndexes()
	if err != nil {
		return nil, err
	}

	setting, exists := indexes.SiteSettingsBySiteID[siteID]
	if !exists {
		return nil, fmt.Errorf("site setting not found for site: %s", siteID)
	}

	return setting, nil
}

// GetSiteSettingByID retrieves a site setting by ID with O(1) lookup
func (ca *CacheAccessorImpl) GetSiteSettingByID(id string) (*SiteSetting, error) {
	indexes, err := ca.manager.GetIndexes()
	if err != nil {
		return nil, err
	}

	setting, exists := indexes.SiteSettingsByID[id]
	if !exists {
		return nil, fmt.Errorf("site setting not found: %s", id)
	}

	return setting, nil
}

// GetAllSiteSettings returns all site settings
func (ca *CacheAccessorImpl) GetAllSiteSettings() ([]*SiteSetting, error) {
	cache, err := ca.manager.GetCache()
	if err != nil {
		return nil, err
	}

	var settings []*SiteSetting
	for _, orgData := range cache.Orgs {
		for i := range orgData.Sites.Settings {
			settings = append(settings, &orgData.Sites.Settings[i])
		}
	}

	return settings, nil
}

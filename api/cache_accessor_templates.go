package api

import (
	"fmt"
)

// GetRFTemplateByName retrieves an RF template by name with O(1) lookup
func (ca *CacheAccessorImpl) GetRFTemplateByName(name string) (*MistRFTemplate, error) {
	indexes, err := ca.manager.GetIndexes()
	if err != nil {
		return nil, err
	}

	template, exists := indexes.RFTemplatesByName[name]
	if !exists {
		return nil, fmt.Errorf("RF template not found: %s", name)
	}

	return template, nil
}

// GetRFTemplateByID retrieves an RF template by ID with O(1) lookup
func (ca *CacheAccessorImpl) GetRFTemplateByID(id string) (*MistRFTemplate, error) {
	indexes, err := ca.manager.GetIndexes()
	if err != nil {
		return nil, err
	}

	template, exists := indexes.RFTemplatesByID[id]
	if !exists {
		return nil, fmt.Errorf("RF template not found: %s", id)
	}

	return template, nil
}

// GetAllRFTemplates returns all RF templates
func (ca *CacheAccessorImpl) GetAllRFTemplates() ([]*MistRFTemplate, error) {
	cache, err := ca.manager.GetCache()
	if err != nil {
		return nil, err
	}

	var templates []*MistRFTemplate
	for _, orgData := range cache.Orgs {
		for i := range orgData.Templates.RF {
			templates = append(templates, &orgData.Templates.RF[i])
		}
	}

	return templates, nil
}

// GetGWTemplateByName retrieves a gateway template by name with O(1) lookup
func (ca *CacheAccessorImpl) GetGWTemplateByName(name string) (*MistGatewayTemplate, error) {
	indexes, err := ca.manager.GetIndexes()
	if err != nil {
		return nil, err
	}

	template, exists := indexes.GWTemplatesByName[name]
	if !exists {
		return nil, fmt.Errorf("gateway template not found: %s", name)
	}

	return template, nil
}

// GetGWTemplateByID retrieves a gateway template by ID with O(1) lookup
func (ca *CacheAccessorImpl) GetGWTemplateByID(id string) (*MistGatewayTemplate, error) {
	indexes, err := ca.manager.GetIndexes()
	if err != nil {
		return nil, err
	}

	template, exists := indexes.GWTemplatesByID[id]
	if !exists {
		return nil, fmt.Errorf("gateway template not found: %s", id)
	}

	return template, nil
}

// GetAllGWTemplates returns all gateway templates
func (ca *CacheAccessorImpl) GetAllGWTemplates() ([]*MistGatewayTemplate, error) {
	cache, err := ca.manager.GetCache()
	if err != nil {
		return nil, err
	}

	var templates []*MistGatewayTemplate
	for _, orgData := range cache.Orgs {
		for i := range orgData.Templates.Gateway {
			templates = append(templates, &orgData.Templates.Gateway[i])
		}
	}

	return templates, nil
}

// GetWLANTemplateByName retrieves a WLAN template by name with O(1) lookup
func (ca *CacheAccessorImpl) GetWLANTemplateByName(name string) (*MistWLANTemplate, error) {
	indexes, err := ca.manager.GetIndexes()
	if err != nil {
		return nil, err
	}

	template, exists := indexes.WLANTemplatesByName[name]
	if !exists {
		return nil, fmt.Errorf("WLAN template not found: %s", name)
	}

	return template, nil
}

// GetWLANTemplateByID retrieves a WLAN template by ID with O(1) lookup
func (ca *CacheAccessorImpl) GetWLANTemplateByID(id string) (*MistWLANTemplate, error) {
	indexes, err := ca.manager.GetIndexes()
	if err != nil {
		return nil, err
	}

	template, exists := indexes.WLANTemplatesByID[id]
	if !exists {
		return nil, fmt.Errorf("WLAN template not found: %s", id)
	}

	return template, nil
}

// GetAllWLANTemplates returns all WLAN templates
func (ca *CacheAccessorImpl) GetAllWLANTemplates() ([]*MistWLANTemplate, error) {
	cache, err := ca.manager.GetCache()
	if err != nil {
		return nil, err
	}

	var templates []*MistWLANTemplate
	for _, orgData := range cache.Orgs {
		for i := range orgData.Templates.WLAN {
			templates = append(templates, &orgData.Templates.WLAN[i])
		}
	}

	return templates, nil
}

// GetNetworkByName retrieves a network by name with O(1) lookup
func (ca *CacheAccessorImpl) GetNetworkByName(name string) (*MistNetwork, error) {
	indexes, err := ca.manager.GetIndexes()
	if err != nil {
		return nil, err
	}

	network, exists := indexes.NetworksByName[name]
	if !exists {
		return nil, fmt.Errorf("network not found: %s", name)
	}

	return network, nil
}

// GetNetworkByID retrieves a network by ID with O(1) lookup
func (ca *CacheAccessorImpl) GetNetworkByID(id string) (*MistNetwork, error) {
	indexes, err := ca.manager.GetIndexes()
	if err != nil {
		return nil, err
	}

	network, exists := indexes.NetworksByID[id]
	if !exists {
		return nil, fmt.Errorf("network not found: %s", id)
	}

	return network, nil
}

// GetAllNetworks returns all networks
func (ca *CacheAccessorImpl) GetAllNetworks() ([]*MistNetwork, error) {
	cache, err := ca.manager.GetCache()
	if err != nil {
		return nil, err
	}

	var networks []*MistNetwork
	for _, orgData := range cache.Orgs {
		for i := range orgData.Networks {
			networks = append(networks, &orgData.Networks[i])
		}
	}

	return networks, nil
}

// GetOrgWLANByName retrieves an org-level WLAN by name with O(1) lookup
func (ca *CacheAccessorImpl) GetOrgWLANByName(name string) (*MistWLAN, error) {
	indexes, err := ca.manager.GetIndexes()
	if err != nil {
		return nil, err
	}

	wlan, exists := indexes.OrgWLANsByName[name]
	if !exists {
		return nil, fmt.Errorf("org WLAN not found: %s", name)
	}

	return wlan, nil
}

// GetOrgWLANByID retrieves an org-level WLAN by ID with O(1) lookup
func (ca *CacheAccessorImpl) GetOrgWLANByID(id string) (*MistWLAN, error) {
	indexes, err := ca.manager.GetIndexes()
	if err != nil {
		return nil, err
	}

	wlan, exists := indexes.OrgWLANsByID[id]
	if !exists {
		return nil, fmt.Errorf("org WLAN not found: %s", id)
	}

	return wlan, nil
}

// GetAllOrgWLANs returns all org-level WLANs
func (ca *CacheAccessorImpl) GetAllOrgWLANs() ([]*MistWLAN, error) {
	cache, err := ca.manager.GetCache()
	if err != nil {
		return nil, err
	}

	var wlans []*MistWLAN
	for _, orgData := range cache.Orgs {
		for i := range orgData.WLANs.Org {
			wlans = append(wlans, &orgData.WLANs.Org[i])
		}
	}

	return wlans, nil
}

// GetSiteWLANByName retrieves a site-level WLAN by name with O(1) lookup
func (ca *CacheAccessorImpl) GetSiteWLANByName(siteID, name string) (*MistWLAN, error) {
	indexes, err := ca.manager.GetIndexes()
	if err != nil {
		return nil, err
	}

	siteWLANs, exists := indexes.SiteWLANsByName[siteID]
	if !exists {
		return nil, fmt.Errorf("no WLANs found for site: %s", siteID)
	}

	wlan, exists := siteWLANs[name]
	if !exists {
		return nil, fmt.Errorf("WLAN not found in site %s: %s", siteID, name)
	}

	return wlan, nil
}

// GetSiteWLANByID retrieves a site-level WLAN by ID with O(1) lookup
func (ca *CacheAccessorImpl) GetSiteWLANByID(siteID, id string) (*MistWLAN, error) {
	indexes, err := ca.manager.GetIndexes()
	if err != nil {
		return nil, err
	}

	siteWLANs, exists := indexes.SiteWLANsByID[siteID]
	if !exists {
		return nil, fmt.Errorf("no WLANs found for site: %s", siteID)
	}

	wlan, exists := siteWLANs[id]
	if !exists {
		return nil, fmt.Errorf("WLAN not found in site %s: %s", siteID, id)
	}

	return wlan, nil
}

// GetSiteWLANs returns all WLANs for a specific site
func (ca *CacheAccessorImpl) GetSiteWLANs(siteID string) ([]*MistWLAN, error) {
	cache, err := ca.manager.GetCache()
	if err != nil {
		return nil, err
	}

	var result []*MistWLAN
	for _, orgData := range cache.Orgs {
		wlans, exists := orgData.WLANs.Sites[siteID]
		if exists {
			for i := range wlans {
				result = append(result, &wlans[i])
			}
		}
	}

	return result, nil
}

// GetAllSiteWLANs returns all site-level WLANs grouped by site
func (ca *CacheAccessorImpl) GetAllSiteWLANs() (map[string][]*MistWLAN, error) {
	cache, err := ca.manager.GetCache()
	if err != nil {
		return nil, err
	}

	result := make(map[string][]*MistWLAN)
	for _, orgData := range cache.Orgs {
		for siteID, wlans := range orgData.WLANs.Sites {
			siteWLANs := make([]*MistWLAN, len(wlans))
			for i := range wlans {
				siteWLANs[i] = &wlans[i]
			}
			result[siteID] = siteWLANs
		}
	}

	return result, nil
}

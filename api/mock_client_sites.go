package api

import (
	"context"
	"fmt"
)

// Site operations
// ============================================================================

// AddMockSite adds a mock site for testing
func (m *MockClient) AddMockSite(site Site) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if site.Id == nil {
		id := UUID(fmt.Sprintf("mock-site-%d", len(m.sites)+1))
		site.Id = &id
	}

	if site.Id != nil {
		siteID := string(*site.Id)
		siteNew := ConvertSiteToNew(site)
		m.sites[siteID] = siteNew

		if site.Name != nil {
			name := *site.Name
			m.sitesByName[name] = siteNew
		}
	}
}

// GetSiteName gets a site name by ID
func (m *MockClient) GetSiteName(siteID string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	site, found := m.sites[siteID]
	if !found || site.Name == nil {
		return "", false
	}

	return *site.Name, true
}

// GetOrgName gets an organization name by ID
func (m *MockClient) GetOrgName(_ string) (string, bool) {
	// Mock implementation - just return a fixed name
	return "Mock Organization", true
}

// CreateSite creates a new site
func (m *MockClient) CreateSite(ctx context.Context, site *MistSite) (*MistSite, error) {
	m.logRequest("POST", "/orgs/"+m.config.Organization+"/sites", site)

	// Mock API call with rate limiting
	if m.rateLimiter != nil {
		if err := m.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if site.ID == nil {
		id := fmt.Sprintf("mock-site-%d", len(m.sites)+1)
		site.ID = &id
	}

	siteID := *site.ID
	m.sites[siteID] = site

	if site.Name != nil {
		name := *site.Name
		m.sitesByName[name] = site
	}

	return site, nil
}

// UpdateSite delegates to UpdateMistSite for interface compatibility
func (m *MockClient) UpdateSite(ctx context.Context, siteID string, site *MistSite) (*MistSite, error) {
	return m.UpdateMistSite(ctx, siteID, site)
}

// UpdateSiteByName delegates to UpdateSiteByNameNew for interface compatibility
func (m *MockClient) UpdateSiteByName(ctx context.Context, siteName string, site *MistSite) (*MistSite, error) {
	return m.UpdateSiteByNameNew(ctx, siteName, site)
}

// DeleteSite deletes a site
func (m *MockClient) DeleteSite(ctx context.Context, siteID string) error {
	m.logRequest("DELETE", fmt.Sprintf("/sites/%s", siteID), nil)

	// Mock API call with rate limiting
	if m.rateLimiter != nil {
		if err := m.rateLimiter.Wait(ctx); err != nil {
			return err
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	site, found := m.sites[siteID]
	if !found {
		return fmt.Errorf("site with ID %s not found", siteID)
	}

	if site.Name != nil {
		delete(m.sitesByName, *site.Name)
	}

	delete(m.sites, siteID)

	// Also remove any APs associated with this site
	delete(m.aps, siteID)

	return nil
}

// DeleteSiteByName deletes a site by name
func (m *MockClient) DeleteSiteByName(ctx context.Context, siteName string) error {
	siteNew, err := m.GetSiteByName(ctx, siteName, m.config.Organization)
	if err != nil {
		return fmt.Errorf("site with name %s not found: %w", siteName, err)
	}

	return m.DeleteSite(ctx, *siteNew.ID)
}

// New bidirectional site methods implementation
// ============================================================================

// GetSitesNew retrieves all sites using the new bidirectional format
func (m *MockClient) GetSitesNew(ctx context.Context, orgID string) ([]*MistSite, error) {
	m.logRequest("GET", fmt.Sprintf("/orgs/%s/sites", orgID), nil)

	// Mock API call with rate limiting
	if m.rateLimiter != nil {
		if err := m.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	sites := make([]*MistSite, 0, len(m.sites))
	for _, site := range m.sites {
		sites = append(sites, site)
	}

	return sites, nil
}

// GetSites delegates to GetSitesNew for interface compatibility
func (m *MockClient) GetSites(ctx context.Context, orgID string) ([]*MistSite, error) {
	return m.GetSitesNew(ctx, orgID)
}

// GetMistSite retrieves a site by ID using the new bidirectional format
func (m *MockClient) GetMistSite(ctx context.Context, siteID string) (*MistSite, error) {
	m.logRequest("GET", fmt.Sprintf("/sites/%s", siteID), nil)

	// Mock API call with rate limiting
	if m.rateLimiter != nil {
		if err := m.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	site, found := m.sites[siteID]
	if !found {
		return nil, fmt.Errorf("site with ID %s not found", siteID)
	}

	return site, nil
}

// GetSite delegates to GetMistSite for interface compatibility
func (m *MockClient) GetSite(ctx context.Context, siteID string) (*MistSite, error) {
	return m.GetMistSite(ctx, siteID)
}

// GetSiteByIdentifier delegates to GetSiteByIdentifierNew for interface compatibility
func (m *MockClient) GetSiteByIdentifier(ctx context.Context, siteIdentifier string) (*MistSite, error) {
	return m.GetSiteByIdentifierNew(ctx, siteIdentifier)
}

// GetSiteByName retrieves a site by name using the new bidirectional format
func (m *MockClient) GetSiteByName(ctx context.Context, name, orgID string) (*MistSite, error) {
	m.logRequest("GET", fmt.Sprintf("/orgs/%s/sites?name=%s", orgID, name), nil)

	// Mock API call with rate limiting
	if m.rateLimiter != nil {
		if err := m.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	site, found := m.sitesByName[name]
	if !found {
		return nil, fmt.Errorf("site with name %s not found", name)
	}

	return site, nil
}

// GetSiteByIdentifierNew retrieves a site by ID or name using the new bidirectional format
func (m *MockClient) GetSiteByIdentifierNew(ctx context.Context, siteIdentifier string) (*MistSite, error) {
	// First try by ID
	site, err := m.GetMistSite(ctx, siteIdentifier)
	if err == nil {
		return site, nil
	}

	// Then try by name
	return m.GetSiteByName(ctx, siteIdentifier, m.config.Organization)
}

// CreateMistSite creates a new site using the new bidirectional format
func (m *MockClient) CreateMistSite(ctx context.Context, site *MistSite) (*MistSite, error) {
	m.logRequest("POST", "/orgs/"+m.config.Organization+"/sites", site)

	// Mock API call with rate limiting
	if m.rateLimiter != nil {
		if err := m.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate ID if not provided
	if site.ID == nil {
		id := fmt.Sprintf("mock-site-%d", len(m.sites)+1)
		site.ID = &id
	}

	// Store in mock data
	if site.ID != nil {
		siteID := *site.ID
		m.sites[siteID] = site
		if site.Name != nil {
			m.sitesByName[*site.Name] = site
		}
	}

	return site, nil
}

// UpdateMistSite updates an existing site using the new bidirectional format
func (m *MockClient) UpdateMistSite(ctx context.Context, siteID string, site *MistSite) (*MistSite, error) {
	m.logRequest("PUT", fmt.Sprintf("/sites/%s", siteID), site)

	// Mock API call with rate limiting
	if m.rateLimiter != nil {
		if err := m.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if site exists
	if _, found := m.sites[siteID]; !found {
		return nil, fmt.Errorf("site with ID %s not found", siteID)
	}

	// Set the ID
	site.ID = &siteID

	// Update in mock data
	m.sites[siteID] = site
	if site.Name != nil {
		m.sitesByName[*site.Name] = site
	}

	return site, nil
}

// UpdateSiteByNameNew updates an existing site by name using the new bidirectional format
func (m *MockClient) UpdateSiteByNameNew(ctx context.Context, siteName string, site *MistSite) (*MistSite, error) {
	// Get the site to find its ID
	existingSite, err := m.GetSiteByName(ctx, siteName, m.config.Organization)
	if err != nil {
		return nil, err
	}

	siteID := existingSite.GetID()
	if siteID == "" {
		return nil, fmt.Errorf("site '%s' found but has no ID", siteName)
	}

	// Update the site using its ID
	return m.UpdateMistSite(ctx, siteID, site)
}

// DeleteMistSite deletes a site by ID using the new bidirectional format
func (m *MockClient) DeleteMistSite(ctx context.Context, siteID string) error {
	m.logRequest("DELETE", fmt.Sprintf("/sites/%s", siteID), nil)

	// Mock API call with rate limiting
	if m.rateLimiter != nil {
		if err := m.rateLimiter.Wait(ctx); err != nil {
			return err
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	site, found := m.sites[siteID]
	if !found {
		return fmt.Errorf("site with ID %s not found", siteID)
	}

	// Remove from both maps
	delete(m.sites, siteID)
	if site.Name != nil {
		delete(m.sitesByName, *site.Name)
	}

	return nil
}

// DeleteSiteByNameNew deletes a site by name using the new bidirectional format
func (m *MockClient) DeleteSiteByNameNew(ctx context.Context, siteName string) error {
	// Get the site to find its ID
	site, err := m.GetSiteByName(ctx, siteName, m.config.Organization)
	if err != nil {
		return err
	}

	siteID := site.GetID()
	if siteID == "" {
		return fmt.Errorf("site '%s' found but has no ID", siteName)
	}

	// Delete the site using its ID
	return m.DeleteMistSite(ctx, siteID)
}

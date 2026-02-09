package api

import (
	"context"
	"strings"
)

// fetchAPIData performs a generic GET request and returns raw interface{} data
func (m *MockClient) fetchAPIData(ctx context.Context, path string) (interface{}, error) {
	m.logRequest("GET", path, nil)

	// Mock API call with rate limiting
	if m.rateLimiter != nil {
		if err := m.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}
	}

	// Return mock data based on the endpoint
	switch {
	case strings.Contains(path, "/rftemplates"):
		return []interface{}{
			map[string]interface{}{
				"id":     "rf-template-1",
				"name":   "Mock RF Template",
				"org_id": "mock-org-id",
			},
		}, nil
	case strings.Contains(path, "/gatewaytemplates"):
		return []interface{}{
			map[string]interface{}{
				"id":     "gw-template-1",
				"name":   "Mock Gateway Template",
				"org_id": "mock-org-id",
			},
		}, nil
	case strings.Contains(path, "/templates"):
		return []interface{}{
			map[string]interface{}{
				"id":     "wlan-template-1",
				"name":   "Mock WLAN Template",
				"org_id": "mock-org-id",
			},
		}, nil
	case strings.Contains(path, "/networks"):
		return []interface{}{
			map[string]interface{}{
				"id":     "network-1",
				"name":   "Mock Network",
				"org_id": "mock-org-id",
			},
		}, nil
	case strings.Contains(path, "/wlans"):
		return []interface{}{
			map[string]interface{}{
				"id":     "wlan-1",
				"ssid":   "Mock WLAN",
				"org_id": "mock-org-id",
			},
		}, nil
	default:
		return []interface{}{}, nil
	}
}

// GetRFTemplates returns mock RF templates
func (m *MockClient) GetRFTemplates(_ context.Context, orgID string) ([]MistRFTemplate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return mock RF template data
	return []MistRFTemplate{
		{
			ID:    StringPtr("rf-template-1"),
			Name:  StringPtr("Mock RF Template"),
			OrgID: StringPtr(orgID),
		},
	}, nil
}

// GetGatewayTemplates returns mock gateway templates
func (m *MockClient) GetGatewayTemplates(_ context.Context, orgID string) ([]MistGatewayTemplate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return mock gateway template data
	return []MistGatewayTemplate{
		{
			ID:    StringPtr("gw-template-1"),
			Name:  StringPtr("Mock Gateway Template"),
			OrgID: StringPtr(orgID),
		},
	}, nil
}

// GetWLANTemplates returns mock WLAN templates
func (m *MockClient) GetWLANTemplates(_ context.Context, orgID string) ([]MistWLANTemplate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return mock WLAN template data
	return []MistWLANTemplate{
		{
			ID:    StringPtr("wlan-template-1"),
			Name:  StringPtr("Mock WLAN Template"),
			OrgID: StringPtr(orgID),
		},
	}, nil
}

// GetNetworks returns mock networks
func (m *MockClient) GetNetworks(_ context.Context, orgID string) ([]MistNetwork, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return mock network data
	return []MistNetwork{
		{
			ID:    StringPtr("network-1"),
			Name:  StringPtr("Mock Network"),
			OrgID: StringPtr(orgID),
		},
	}, nil
}

// GetWLANs returns mock WLANs (org-level)
func (m *MockClient) GetWLANs(_ context.Context, orgID string) ([]MistWLAN, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return mock WLAN data
	return []MistWLAN{
		{
			ID:    StringPtr("wlan-1"),
			SSID:  StringPtr("Mock WLAN"),
			OrgID: StringPtr(orgID),
		},
	}, nil
}

// GetSiteWLANs returns mock site-level WLANs
func (m *MockClient) GetSiteWLANs(_ context.Context, siteID string) ([]MistWLAN, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return mock site-level WLAN data
	return []MistWLAN{
		{
			ID:     StringPtr("site-wlan-1"),
			SSID:   StringPtr("Mock Site WLAN"),
			SiteID: StringPtr(siteID),
			OrgID:  StringPtr("mock-org-id"),
		},
	}, nil
}

// GetSiteSetting retrieves site setting by site ID (mock implementation)
func (m *MockClient) GetSiteSetting(_ context.Context, siteID string) (*SiteSetting, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Mock implementation - return a basic site setting
	return &SiteSetting{
		ID:     StringPtr("mock-setting-id"),
		SiteID: &siteID,
		OrgID:  StringPtr("mock-org-id"),
		AdditionalConfig: map[string]interface{}{
			"mock_config": "mock_value",
		},
	}, nil
}

// CreateOrgWLAN creates a new org-level WLAN (mock implementation)
func (m *MockClient) CreateOrgWLAN(_ context.Context, orgID string, wlan *MistWLAN) (*MistWLAN, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create a mock response with a generated ID
	mockID := "mock-wlan-" + orgID
	return &MistWLAN{
		ID:               &mockID,
		SSID:             wlan.SSID,
		OrgID:            &orgID,
		AdditionalConfig: make(map[string]interface{}),
	}, nil
}

// CreateSiteWLAN creates a new site-level WLAN (mock implementation)
func (m *MockClient) CreateSiteWLAN(_ context.Context, siteID string, wlan *MistWLAN) (*MistWLAN, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create a mock response with a generated ID
	mockID := "mock-wlan-" + siteID
	return &MistWLAN{
		ID:               &mockID,
		SSID:             wlan.SSID,
		SiteID:           &siteID,
		AdditionalConfig: make(map[string]interface{}),
	}, nil
}

// UpdateOrgWLAN updates an existing org-level WLAN (mock implementation)
func (m *MockClient) UpdateOrgWLAN(_ context.Context, orgID string, wlanID string, wlan *MistWLAN) (*MistWLAN, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Return the updated WLAN with the provided ID
	wlan.ID = &wlanID
	wlan.OrgID = &orgID
	return wlan, nil
}

// UpdateSiteWLAN updates an existing site-level WLAN (mock implementation)
func (m *MockClient) UpdateSiteWLAN(_ context.Context, siteID string, wlanID string, wlan *MistWLAN) (*MistWLAN, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Return the updated WLAN with the provided ID
	wlan.ID = &wlanID
	wlan.SiteID = &siteID
	return wlan, nil
}

// DeleteOrgWLAN deletes an org-level WLAN (mock implementation)
func (m *MockClient) DeleteOrgWLAN(_ context.Context, orgID string, wlanID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Mock implementation - always succeed
	return nil
}

// DeleteSiteWLAN deletes a site-level WLAN (mock implementation)
func (m *MockClient) DeleteSiteWLAN(_ context.Context, siteID string, wlanID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Mock implementation - always succeed
	return nil
}

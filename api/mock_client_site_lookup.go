package api

import (
	"context"
	"regexp"
	"strings"

	"github.com/ravinald/wifimgr/internal/logging"
)

// LookupSiteByID implements the Client interface for Client
func (m *MockClient) LookupSiteByID(siteID string, options ...SiteLookupOptions) (Site, bool, error) {
	// Set default options if none provided
	opts := DefaultSiteLookupOptions()
	if len(options) > 0 {
		opts = options[0]
	}

	// Ensure we have a context
	if opts.Ctx == nil {
		opts.Ctx = context.Background()
	}

	// First try the mock sites
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Get a copy of the site map, converting from MistSite to Site
	sitesMap := make(map[string]*Site)
	for id, siteNew := range m.sites {
		site := ConvertSiteFromNew(siteNew)
		sitesMap[id] = &site
	}

	// Now search through all sites
	for _, site := range sitesMap {
		if site.Id != nil && string(*site.Id) == siteID {
			logging.Debugf("[MOCK] Found site with ID %s in mock sites", siteID)
			siteCopy := *site
			return siteCopy, true, nil
		}
	}

	// If not found and we're allowing API calls, try via API (which is simulated in the mock)
	if opts.SearchOrder != "cache-only" {
		siteNew, err := m.GetMistSite(opts.Ctx, siteID)
		if err == nil && siteNew != nil {
			logging.Debugf("[MOCK] Found site with ID %s via simulated API call", siteID)
			// Convert MistSite to Site for compatibility
			site := Site{
				Id:          (*UUID)(siteNew.ID),
				Name:        siteNew.Name,
				Address:     siteNew.Address,
				CountryCode: siteNew.CountryCode,
				Timezone:    siteNew.Timezone,
				Notes:       siteNew.Notes,
			}
			if siteNew.Latlng != nil {
				site.Latlng = &LatLng{
					Lat: getMockFloat64Value(siteNew.Latlng.Lat),
					Lng: getMockFloat64Value(siteNew.Latlng.Lng),
				}
			}
			return site, true, nil
		}
	}

	// Not found
	return Site{}, false, nil
}

// LookupSiteByName implements the Client interface for Client
func (m *MockClient) LookupSiteByName(siteName string, options ...SiteLookupOptions) (Site, bool, error) {
	// Set default options if none provided
	opts := DefaultSiteLookupOptions()
	if len(options) > 0 {
		opts = options[0]
	}

	// Ensure we have a context
	if opts.Ctx == nil {
		opts.Ctx = context.Background()
	}

	// First try the mock sites
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Get a copy of the site map, converting from MistSite to Site
	sitesMap := make(map[string]*Site)
	for id, siteNew := range m.sites {
		site := ConvertSiteFromNew(siteNew)
		sitesMap[id] = &site
	}

	// Now search through all sites
	for _, site := range sitesMap {
		if site.Name != nil && *site.Name == siteName {
			logging.Debugf("[MOCK] Found site with name %s in mock sites", siteName)
			siteCopy := *site
			return siteCopy, true, nil
		}
	}

	// If not found and we're allowing API calls, try via API (which is simulated in the mock)
	if opts.SearchOrder != "cache-only" {
		// Use the provided org ID or default to the mock's organization ID
		orgID := opts.OrgID
		if orgID == "" {
			orgID = "mock-org-id" // Default mock org ID
		}

		siteNew, err := m.GetSiteByName(opts.Ctx, siteName, orgID)
		if err == nil && siteNew != nil {
			logging.Debugf("[MOCK] Found site with name %s via simulated API call", siteName)
			// Convert MistSite to Site for compatibility
			site := Site{
				Id:          (*UUID)(siteNew.ID),
				Name:        siteNew.Name,
				Address:     siteNew.Address,
				CountryCode: siteNew.CountryCode,
				Timezone:    siteNew.Timezone,
				Notes:       siteNew.Notes,
			}
			if siteNew.Latlng != nil {
				site.Latlng = &LatLng{
					Lat: getMockFloat64Value(siteNew.Latlng.Lat),
					Lng: getMockFloat64Value(siteNew.Latlng.Lng),
				}
			}
			return site, true, nil
		}
	}

	// Not found
	return Site{}, false, nil
}

// Helper function to safely get float64 value from pointer
func getMockFloat64Value(ptr *float64) float64 {
	if ptr != nil {
		return *ptr
	}
	return 0.0
}

// LookupSiteByIdentifier implements the Client interface for Client
func (m *MockClient) LookupSiteByIdentifier(identifier string, options ...SiteLookupOptions) (Site, bool, error) {
	// Set default options if none provided
	opts := DefaultSiteLookupOptions()
	if len(options) > 0 {
		opts = options[0]
	}

	// Ensure we have a context
	if opts.Ctx == nil {
		opts.Ctx = context.Background()
	}

	// Try to determine the type of identifier

	// Check if it's a UUID
	uuidPattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	if uuidPattern.MatchString(strings.ToLower(identifier)) {
		// It's a UUID, so search by ID
		return m.LookupSiteByID(identifier, opts)
	}

	// For site names in the mock environment, we'll handle a special case for testing
	// This allows testing with patterns like US-SFO-TEST, etc.
	siteNamePattern := regexp.MustCompile(`^[A-Z]{2}-[A-Z]{3}-[A-Z\d]+$`)
	if siteNamePattern.MatchString(identifier) {
		// It matches our expected site name pattern
		return m.LookupSiteByName(identifier, opts)
	}

	// If it doesn't match any specific pattern, try both ID and name
	// Try by ID first
	site, found, _ := m.LookupSiteByID(identifier, opts)
	if found {
		return site, true, nil
	}

	// Then try by name
	return m.LookupSiteByName(identifier, opts)
}

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/patterns"
)

// Site-related methods for the mistClient

// GetSiteName retrieves a site name from the cache by ID
func (c *mistClient) GetSiteName(siteID string) (string, bool) {
	// First try the in-memory sitesCache
	if sites, found := c.sitesCache.Get("sites"); found {
		for _, site := range sites {
			if site.Id != nil && string(*site.Id) == siteID {
				if site.Name != nil {
					logging.Debugf("Found site name %s for ID %s in sitesCache", *site.Name, siteID)
					return *site.Name, true
				}
				return "", false
			}
		}
	}

	// Legacy cache operations disabled
	if false {
		// Would check local cache here, but legacy cache system removed
		logging.Debugf("Found site name %s for ID %s in localCache", "name", siteID)
		// return name, true
	}

	// Not found in any cache
	return "", false
}

// GetOrgName retrieves an organization name from the cache by ID
func (c *mistClient) GetOrgName(orgID string) (string, bool) {
	// Get cache accessor
	cacheAccessor := c.GetCacheAccessor()
	if cacheAccessor == nil {
		return "", false
	}

	// Try to get org from cache
	org, err := cacheAccessor.GetOrgByID(orgID)
	if err != nil {
		return "", false
	}

	if org != nil && org.Name != nil {
		logging.Debugf("Found org name %s for ID %s in cache", *org.Name, orgID)
		return *org.Name, true
	}

	return "", false
}

// Site-related methods using the new bidirectional data handling

// GetSites retrieves all sites for the specified organization using raw JSON unmarshaling
func (c *mistClient) GetSites(ctx context.Context, orgID string) ([]*MistSite, error) {
	// Try to get from cache first
	cacheAccessor := c.GetCacheAccessor()
	if cacheAccessor != nil {
		sites, err := cacheAccessor.GetAllSites()
		if err == nil && len(sites) > 0 {
			c.logDebug("Cache hit for sites from cache accessor")
			// Filter by org ID if needed
			filteredSites := make([]*MistSite, 0, len(sites))
			for _, site := range sites {
				if site.OrgID != nil && *site.OrgID == orgID {
					filteredSites = append(filteredSites, site)
				}
			}
			return filteredSites, nil
		}
	}

	c.logDebug("Cache miss for sites")

	// Determine the results limit to use
	limit := 100 // Default value
	if c.config.ResultsLimit > 0 {
		limit = c.config.ResultsLimit
		c.logDebug("Using configured results limit: %d", limit)
	}

	var allSites []*MistSite
	page := 1
	hasMore := true

	for hasMore {
		c.logDebug("Fetching sites page %d with limit %d", page, limit)

		// Build query parameters
		query := url.Values{}
		query.Set("limit", fmt.Sprintf("%d", limit))
		if page > 1 {
			query.Set("page", fmt.Sprintf("%d", page))
		}

		// Build the path with query parameters
		path := fmt.Sprintf("/orgs/%s/sites?%s", orgID, query.Encode())

		// Use raw JSON unmarshaling to preserve all data
		var rawResponse json.RawMessage
		if err := c.do(ctx, http.MethodGet, path, nil, &rawResponse); err != nil {
			return nil, fmt.Errorf("failed to get sites: %w", err)
		}

		// Parse the raw JSON to map slice
		var rawSites []map[string]interface{}
		if err := json.Unmarshal(rawResponse, &rawSites); err != nil {
			return nil, fmt.Errorf("failed to unmarshal sites response: %w", err)
		}

		if len(rawSites) == 0 {
			hasMore = false
			continue
		}

		// Convert each raw site to MistSite
		for _, rawSite := range rawSites {
			site, err := NewSiteFromMap(rawSite)
			if err != nil {
				c.logDebug("Failed to create site from map: %v", err)
				continue
			}
			allSites = append(allSites, site)
		}

		// Check if we've received fewer sites than the limit, indicating the last page
		if len(rawSites) < limit {
			hasMore = false
		} else {
			page++
		}
	}

	// Legacy cache operations disabled - site caching modernized

	return allSites, nil
}

// GetSite retrieves a specific site by ID using raw JSON unmarshaling
func (c *mistClient) GetSite(ctx context.Context, siteID string) (*MistSite, error) {
	// Use raw JSON unmarshaling to preserve all data
	var rawResponse json.RawMessage
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/sites/%s", siteID), nil, &rawResponse)
	if err != nil {
		return nil, formatError("failed to get site", err)
	}

	// Parse the raw JSON to map
	var rawSite map[string]interface{}
	if err := json.Unmarshal(rawResponse, &rawSite); err != nil {
		return nil, fmt.Errorf("failed to unmarshal site response: %w", err)
	}

	// Convert to MistSite
	site, err := NewSiteFromMap(rawSite)
	if err != nil {
		return nil, fmt.Errorf("failed to create site from API response: %w", err)
	}

	return site, nil
}

// GetSiteByName retrieves a site by its name within an organization using the new implementation
func (c *mistClient) GetSiteByName(ctx context.Context, name, orgID string) (*MistSite, error) {
	sites, err := c.GetSites(ctx, orgID)
	if err != nil {
		return nil, formatError("failed to get sites", err)
	}

	for _, site := range sites {
		if patterns.Equals(site.GetName(), name) {
			return site, nil
		}
	}

	return nil, fmt.Errorf("site with name '%s' not found", name)
}

// GetSiteByIdentifier retrieves a site by either its ID or name using the new implementation
func (c *mistClient) GetSiteByIdentifier(ctx context.Context, siteIdentifier string) (*MistSite, error) {
	// Try to get by ID first
	site, err := c.GetSite(ctx, siteIdentifier)
	if err == nil {
		return site, nil
	}

	// If not found by ID, try by name
	return c.GetSiteByName(ctx, siteIdentifier, c.config.Organization)
}

// CreateSite creates a new site in the specified organization using the new implementation
func (c *mistClient) CreateSite(ctx context.Context, site *MistSite) (*MistSite, error) {
	// If in dry run mode, log and return simulated success
	if c.dryRun {
		c.logDebug("[DRY RUN] Would create site: %+v", site)
		// Create a simulated response with basic info
		simulatedID := "dry-run-site-id"
		simulatedSite := &MistSite{
			ID:               &simulatedID,
			Name:             site.Name,
			AdditionalConfig: make(map[string]interface{}),
		}
		return simulatedSite, nil
	}

	// Convert site to map for API request
	siteData := site.ToMap()

	// Use raw JSON unmarshaling to preserve all data in response
	var rawResponse json.RawMessage
	err := c.do(ctx, http.MethodPost, fmt.Sprintf("/orgs/%s/sites", c.config.Organization), siteData, &rawResponse)
	if err != nil {
		return nil, formatError("failed to create site", err)
	}

	// Parse the raw JSON to map
	var rawSite map[string]interface{}
	if err := json.Unmarshal(rawResponse, &rawSite); err != nil {
		return nil, fmt.Errorf("failed to unmarshal create site response: %w", err)
	}

	// Convert to MistSite
	createdSite, err := NewSiteFromMap(rawSite)
	if err != nil {
		return nil, fmt.Errorf("failed to create site from API response: %w", err)
	}

	// Invalidate sites cache
	c.sitesCache.Delete("sites")
	// Note: Local cache for sites_new will be updated in the normal cache refresh cycle

	return createdSite, nil
}

// UpdateSite updates an existing site by ID using the new implementation
func (c *mistClient) UpdateSite(ctx context.Context, siteID string, site *MistSite) (*MistSite, error) {
	// If in dry run mode, log and return simulated success
	if c.dryRun {
		c.logDebug("[DRY RUN] Would update site %s: %+v", siteID, site)
		// Return the input site as if it were updated
		site.ID = &siteID
		return site, nil
	}

	// Convert site to map for API request
	siteData := site.ToMap()

	// Use raw JSON unmarshaling to preserve all data in response
	var rawResponse json.RawMessage
	err := c.do(ctx, http.MethodPut, fmt.Sprintf("/sites/%s", siteID), siteData, &rawResponse)
	if err != nil {
		return nil, formatError("failed to update site", err)
	}

	// Parse the raw JSON to map
	var rawSite map[string]interface{}
	if err := json.Unmarshal(rawResponse, &rawSite); err != nil {
		return nil, fmt.Errorf("failed to unmarshal update site response: %w", err)
	}

	// Convert to MistSite
	updatedSite, err := NewSiteFromMap(rawSite)
	if err != nil {
		return nil, fmt.Errorf("failed to create site from API response: %w", err)
	}

	// Invalidate sites cache
	c.sitesCache.Delete("sites")
	c.sitesCache.Delete("sites_new")

	return updatedSite, nil
}

// UpdateSiteByName updates an existing site by name using the new implementation
func (c *mistClient) UpdateSiteByName(ctx context.Context, siteName string, site *MistSite) (*MistSite, error) {
	// Get the site to find its ID
	existingSite, err := c.GetSiteByName(ctx, siteName, c.config.Organization)
	if err != nil {
		return nil, formatError("failed to find site by name", err)
	}

	siteID := existingSite.GetID()
	if siteID == "" {
		return nil, fmt.Errorf("site '%s' found but has no ID", siteName)
	}

	// Update the site using its ID
	return c.UpdateSite(ctx, siteID, site)
}

// DeleteSite deletes a site by ID using the new implementation
func (c *mistClient) DeleteSite(ctx context.Context, siteID string) error {
	// If in dry run mode, log and return simulated success
	if c.dryRun {
		c.logDebug("[DRY RUN] Would delete site: %s", siteID)
		return nil
	}

	// Real implementation
	err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/sites/%s", siteID), nil, nil)
	if err != nil {
		return formatError("failed to delete site", err)
	}

	// Invalidate sites cache
	c.sitesCache.Delete("sites")
	c.sitesCache.Delete("sites_new")

	return nil
}

// DeleteSiteByName deletes a site by name using the new implementation
func (c *mistClient) DeleteSiteByName(ctx context.Context, siteName string) error {
	// Get the site to find its ID
	site, err := c.GetSiteByName(ctx, siteName, c.config.Organization)
	if err != nil {
		return formatError("failed to find site by name", err)
	}

	siteID := site.GetID()
	if siteID == "" {
		return fmt.Errorf("site '%s' found but has no ID", siteName)
	}

	// Delete the site using its ID
	return c.DeleteSite(ctx, siteID)
}

// GetSiteSetting retrieves site settings for a specific site using raw JSON unmarshaling
func (c *mistClient) GetSiteSetting(ctx context.Context, siteID string) (*SiteSetting, error) {
	// Use raw JSON unmarshaling to preserve all data
	var rawResponse json.RawMessage
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/sites/%s/setting", siteID), nil, &rawResponse)
	if err != nil {
		return nil, formatError("failed to get site setting", err)
	}

	// Parse the raw JSON to map
	var rawSiteSetting map[string]interface{}
	if err := json.Unmarshal(rawResponse, &rawSiteSetting); err != nil {
		return nil, fmt.Errorf("failed to unmarshal site setting response: %w", err)
	}

	// Convert to SiteSetting
	siteSetting, err := NewSiteSettingFromMap(rawSiteSetting)
	if err != nil {
		return nil, fmt.Errorf("failed to create site setting from API response: %w", err)
	}

	return siteSetting, nil
}

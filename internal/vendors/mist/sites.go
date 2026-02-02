package mist

import (
	"context"
	"fmt"

	"github.com/ravinald/wifimgr/api"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// sitesService implements vendors.SitesService for Mist.
type sitesService struct {
	client api.Client
	orgID  string
}

// List returns all sites in the organization.
func (s *sitesService) List(ctx context.Context) ([]*vendors.SiteInfo, error) {
	sites, err := s.client.GetSites(ctx, s.orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sites: %w", err)
	}

	result := make([]*vendors.SiteInfo, 0, len(sites))
	for _, site := range sites {
		info := convertSiteToSiteInfo(site)
		if info != nil {
			result = append(result, info)
		}
	}

	return result, nil
}

// Get finds a site by its vendor-specific ID.
func (s *sitesService) Get(ctx context.Context, id string) (*vendors.SiteInfo, error) {
	site, err := s.client.GetSite(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get site by ID %q: %w", id, err)
	}

	return convertSiteToSiteInfo(site), nil
}

// ByName finds a site by its human-readable name.
func (s *sitesService) ByName(ctx context.Context, name string) (*vendors.SiteInfo, error) {
	site, err := s.client.GetSiteByName(ctx, name, s.orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get site by name %q: %w", name, err)
	}

	return convertSiteToSiteInfo(site), nil
}

// Create creates a new site.
func (s *sitesService) Create(ctx context.Context, site *vendors.SiteInfo) (*vendors.SiteInfo, error) {
	mistSite := convertSiteInfoToSite(site)
	if mistSite == nil {
		return nil, fmt.Errorf("invalid site info")
	}

	created, err := s.client.CreateSite(ctx, mistSite)
	if err != nil {
		return nil, fmt.Errorf("failed to create site: %w", err)
	}

	return convertSiteToSiteInfo(created), nil
}

// Update modifies an existing site.
func (s *sitesService) Update(ctx context.Context, id string, site *vendors.SiteInfo) (*vendors.SiteInfo, error) {
	mistSite := convertSiteInfoToSite(site)
	if mistSite == nil {
		return nil, fmt.Errorf("invalid site info")
	}

	updated, err := s.client.UpdateSite(ctx, id, mistSite)
	if err != nil {
		return nil, fmt.Errorf("failed to update site: %w", err)
	}

	return convertSiteToSiteInfo(updated), nil
}

// Delete removes a site.
func (s *sitesService) Delete(ctx context.Context, id string) error {
	if err := s.client.DeleteSite(ctx, id); err != nil {
		return fmt.Errorf("failed to delete site: %w", err)
	}
	return nil
}

// Ensure sitesService implements vendors.SitesService at compile time.
var _ vendors.SitesService = (*sitesService)(nil)

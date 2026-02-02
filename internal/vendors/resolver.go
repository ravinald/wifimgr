package vendors

import (
	"context"
	"fmt"
)

// SiteResolver resolves site names to IDs across multiple APIs.
type SiteResolver struct {
	registry *APIClientRegistry
	cache    *CacheManager
}

// NewSiteResolver creates a new site resolver.
func NewSiteResolver(registry *APIClientRegistry, cache *CacheManager) *SiteResolver {
	return &SiteResolver{
		registry: registry,
		cache:    cache,
	}
}

// SiteResolution contains the result of resolving a site name.
type SiteResolution struct {
	SiteName string
	SiteID   string
	APILabel string
	Vendor   string
}

// ResolveSiteID resolves a site name to its ID for a specific API.
// If apiLabel is empty, it uses the cache index to find the API.
func (r *SiteResolver) ResolveSiteID(ctx context.Context, siteName, apiLabel string) (*SiteResolution, error) {
	if apiLabel != "" {
		// Specific API requested
		return r.resolveSiteInAPI(ctx, siteName, apiLabel)
	}

	// Find which APIs have this site
	apis := r.cache.GetSiteAPIs(siteName)
	if len(apis) == 0 {
		return nil, &SiteNotFoundError{SiteName: siteName}
	}

	if len(apis) > 1 {
		return nil, &DuplicateSiteError{
			SiteName:   siteName,
			MatchCount: len(apis),
		}
	}

	return r.resolveSiteInAPI(ctx, siteName, apis[0])
}

// resolveSiteInAPI resolves a site name within a specific API.
func (r *SiteResolver) resolveSiteInAPI(_ context.Context, siteName, apiLabel string) (*SiteResolution, error) {
	siteID, err := r.cache.GetSiteIDByName(apiLabel, siteName)
	if err != nil {
		return nil, err
	}

	vendor, _ := r.registry.GetVendor(apiLabel)

	return &SiteResolution{
		SiteName: siteName,
		SiteID:   siteID,
		APILabel: apiLabel,
		Vendor:   vendor,
	}, nil
}

// ResolveSiteInfo resolves a site name and returns full site info.
func (r *SiteResolver) ResolveSiteInfo(ctx context.Context, siteName, apiLabel string) (*SiteInfo, error) {
	resolution, err := r.ResolveSiteID(ctx, siteName, apiLabel)
	if err != nil {
		return nil, err
	}

	client, err := r.registry.GetClient(resolution.APILabel)
	if err != nil {
		return nil, err
	}

	sitesSvc := client.Sites()
	if sitesSvc == nil {
		return nil, &CapabilityNotSupportedError{
			Capability: "sites",
			APILabel:   resolution.APILabel,
			VendorName: resolution.Vendor,
		}
	}

	site, err := sitesSvc.Get(ctx, resolution.SiteID)
	if err != nil {
		return nil, fmt.Errorf("failed to get site info: %w", err)
	}

	site.SourceAPI = resolution.APILabel
	site.SourceVendor = resolution.Vendor
	return site, nil
}

// FindSiteAPIs returns all APIs that have a site with the given name.
// This is useful for commands that need to show data from multiple vendors.
func (r *SiteResolver) FindSiteAPIs(siteName string) []string {
	return r.cache.GetSiteAPIs(siteName)
}

// ResolveAllSitesWithName resolves a site name across all APIs that have it.
// Returns a resolution for each API, useful for aggregated views.
func (r *SiteResolver) ResolveAllSitesWithName(ctx context.Context, siteName string) ([]*SiteResolution, error) {
	apis := r.cache.GetSiteAPIs(siteName)
	if len(apis) == 0 {
		return nil, &SiteNotFoundError{SiteName: siteName}
	}

	var resolutions []*SiteResolution
	for _, apiLabel := range apis {
		resolution, err := r.resolveSiteInAPI(ctx, siteName, apiLabel)
		if err != nil {
			// Skip APIs with errors but continue
			continue
		}
		resolutions = append(resolutions, resolution)
	}

	if len(resolutions) == 0 {
		return nil, &SiteNotFoundError{SiteName: siteName}
	}

	return resolutions, nil
}

// DeviceResolution contains the result of resolving a device.
type DeviceResolution struct {
	MAC      string
	APILabel string
	Vendor   string
	SiteID   string
	Type     string // "ap", "switch", "gateway"
}

// ResolveDeviceByMAC resolves a device MAC to its owning API.
func (r *SiteResolver) ResolveDeviceByMAC(mac string) (*DeviceResolution, error) {
	item, apiLabel, err := r.cache.FindDeviceByMAC(mac)
	if err != nil {
		return nil, err
	}

	vendor, _ := r.registry.GetVendor(apiLabel)

	return &DeviceResolution{
		MAC:      item.MAC,
		APILabel: apiLabel,
		Vendor:   vendor,
		SiteID:   item.SiteID,
		Type:     item.Type,
	}, nil
}

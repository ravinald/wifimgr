package api

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/ravinald/wifimgr/internal/logging"
)

// SiteLookupOptions contains options for site lookup operations
type SiteLookupOptions struct {
	// Context for API calls
	Ctx context.Context

	// OrgID is the ID of the organization to search in (optional)
	OrgID string

	// SearchOrder determines where to look first (cache, api, or both)
	// Default is both, starting with cache
	SearchOrder string
}

// DefaultSiteLookupOptions returns the default options for site lookup
func DefaultSiteLookupOptions() SiteLookupOptions {
	return SiteLookupOptions{
		Ctx:         context.Background(),
		SearchOrder: "cache-first", // "cache-first", "api-first", "cache-only", "api-only"
	}
}

// LookupSiteByID finds a site by its ID
// It will search in the sites cache first, then fall back to API if needed
// Returns the site, a boolean indicating if it was found, and any error
func (c *mistClient) LookupSiteByID(siteID string, options ...SiteLookupOptions) (Site, bool, error) {
	// Set default options if none provided
	opts := DefaultSiteLookupOptions()
	if len(options) > 0 {
		opts = options[0]
	}

	// Ensure we have a context
	if opts.Ctx == nil {
		opts.Ctx = context.Background()
	}

	var site Site
	var found bool
	var err error

	// Determine search order based on options
	switch opts.SearchOrder {
	case "cache-first", "":
		// Try cache first, then API
		site, found = c.lookupSiteInCache(siteID, "", opts)
		if !found && opts.SearchOrder != "cache-only" {
			site, found, err = c.lookupSiteInAPI(siteID, "", opts)
		}
	case "api-first":
		// Try API first, then cache
		site, found, err = c.lookupSiteInAPI(siteID, "", opts)
		if !found {
			site, found = c.lookupSiteInCache(siteID, "", opts)
		}
	case "cache-only":
		// Only search in cache
		site, found = c.lookupSiteInCache(siteID, "", opts)
	case "api-only":
		// Only search in API
		site, found, err = c.lookupSiteInAPI(siteID, "", opts)
	default:
		return Site{}, false, fmt.Errorf("invalid search order: %s", opts.SearchOrder)
	}

	return site, found, err
}

// LookupSiteByName finds a site by its name
// It will search in the sites cache first, then fall back to API if needed
func (c *mistClient) LookupSiteByName(siteName string, options ...SiteLookupOptions) (Site, bool, error) {
	// Set default options if none provided
	opts := DefaultSiteLookupOptions()
	if len(options) > 0 {
		opts = options[0]
	}

	// Ensure we have a context
	if opts.Ctx == nil {
		opts.Ctx = context.Background()
	}

	var site Site
	var found bool
	var err error

	// Determine search order based on options
	switch opts.SearchOrder {
	case "cache-first", "":
		// Try cache first, then API
		site, found = c.lookupSiteInCache("", siteName, opts)
		if !found && opts.SearchOrder != "cache-only" {
			site, found, err = c.lookupSiteInAPI("", siteName, opts)
		}
	case "api-first":
		// Try API first, then cache
		site, found, err = c.lookupSiteInAPI("", siteName, opts)
		if !found {
			site, found = c.lookupSiteInCache("", siteName, opts)
		}
	case "cache-only":
		// Only search in cache
		site, found = c.lookupSiteInCache("", siteName, opts)
	case "api-only":
		// Only search in API
		site, found, err = c.lookupSiteInAPI("", siteName, opts)
	default:
		return Site{}, false, fmt.Errorf("invalid search order: %s", opts.SearchOrder)
	}

	return site, found, err
}

// LookupSiteByIdentifier finds a site by any type of identifier
// It tries to determine the identifier type (ID or name) and calls the appropriate lookup function
func (c *mistClient) LookupSiteByIdentifier(identifier string, options ...SiteLookupOptions) (Site, bool, error) {
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
		return c.LookupSiteByID(identifier, opts)
	}

	// If not a UUID, try as a site name
	return c.LookupSiteByName(identifier, opts)
}

// lookupSiteInCache searches for a site in the in-memory and local caches
// Either siteID or siteName must be provided (or both)
func (c *mistClient) lookupSiteInCache(siteID, siteName string, _ SiteLookupOptions) (Site, bool) {
	// First try the in-memory sites cache
	if c.sitesCache != nil {
		if sites, found := c.sitesCache.Get("sites"); found {
			// If we have a site ID, search by ID
			if siteID != "" {
				for _, site := range sites {
					if site.Id != nil && string(*site.Id) == siteID {
						logging.Debugf("Found site with ID %s in sites cache", siteID)
						return site, true
					}
				}
			}

			// If we have a site name, search by name
			if siteName != "" {
				for _, site := range sites {
					if site.Name != nil && *site.Name == siteName {
						logging.Debugf("Found site with name %s in sites cache", siteName)
						return site, true
					}
				}
			}
		}
	}

	// Legacy cache operations disabled
	if false {
		// Would search local cache for site by ID/name here, but legacy cache system removed
		if siteID != "" {
			logging.Debugf("Found site with ID %s in local cache", siteID)
			// return site, true
		}
		if siteName != "" {
			logging.Debugf("Found site with name %s in local cache", siteName)
			// return site, true
		}
	}

	// Not found in any cache
	return Site{}, false
}

// lookupSiteInAPI searches for a site using the API
// Either siteID or siteName must be provided (or both)
func (c *mistClient) lookupSiteInAPI(siteID, siteName string, opts SiteLookupOptions) (Site, bool, error) {
	// If we have a site ID, search by ID
	if siteID != "" {
		siteNew, err := c.GetSite(opts.Ctx, siteID)
		if err == nil && siteNew != nil {
			logging.Debugf("Found site with ID %s in API", siteID)
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
					Lat: getFloat64Value(siteNew.Latlng.Lat),
					Lng: getFloat64Value(siteNew.Latlng.Lng),
				}
			}
			return site, true, nil
		}
	}

	// If we have a site name, search by name
	if siteName != "" {
		// Use the provided org ID or default to the client's org ID
		orgID := opts.OrgID
		if orgID == "" {
			orgID = c.config.Organization
		}

		siteNew, err := c.GetSiteByName(opts.Ctx, siteName, orgID)
		if err == nil && siteNew != nil {
			logging.Debugf("Found site with name %s in API", siteName)
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
					Lat: getFloat64Value(siteNew.Latlng.Lat),
					Lng: getFloat64Value(siteNew.Latlng.Lng),
				}
			}
			return site, true, nil
		}
	}

	// Not found via API
	return Site{}, false, nil
}

// Helper function to safely get float64 value from pointer
func getFloat64Value(ptr *float64) float64 {
	if ptr != nil {
		return *ptr
	}
	return 0.0
}

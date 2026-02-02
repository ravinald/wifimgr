package api

import (
	"context"
)

// DeviceLookupOptions contains options for device lookup operations
type DeviceLookupOptions struct {
	// Context for API calls
	Ctx context.Context

	// SiteID is the ID of the site to search in (optional)
	SiteID string

	// OrgID is the ID of the organization to search in (optional)
	OrgID string

	// DeviceType filters the lookup to a specific device type (ap, switch, gateway)
	DeviceType string

	// SearchOrder determines where to look first (cache, api, or both)
	// Default is both, starting with cache
	SearchOrder string
}

// DefaultDeviceLookupOptions returns the default options for device lookup
func DefaultDeviceLookupOptions() DeviceLookupOptions {
	return DeviceLookupOptions{
		Ctx:         context.Background(),
		SearchOrder: "cache-first", // "cache-first", "api-first", "cache-only", "api-only"
	}
}

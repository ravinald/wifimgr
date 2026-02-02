package mist

import (
	"context"
	"fmt"

	"github.com/ravinald/wifimgr/api"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// wlansService implements vendors.WLANsService for Mist.
type wlansService struct {
	client api.Client
	orgID  string
}

// List returns all WLANs in the organization.
// For Mist, this fetches both org-level WLANs and site-level WLANs.
func (s *wlansService) List(ctx context.Context) ([]*vendors.WLAN, error) {
	logging.Debugf("[mist] Fetching WLANs for org %s", s.orgID)

	var result []*vendors.WLAN

	// 1. Fetch org-level WLANs
	orgWLANs, err := s.client.GetWLANs(ctx, s.orgID)
	if err != nil {
		logging.Warnf("[mist] Failed to get org-level WLANs: %v", err)
	} else {
		for _, w := range orgWLANs {
			wlan := convertMistWLAN(&w, s.orgID)
			result = append(result, wlan)
		}
		logging.Debugf("[mist] Fetched %d org-level WLANs", len(orgWLANs))
	}

	// 2. Fetch site-level WLANs for each site
	sites, err := s.client.GetSites(ctx, s.orgID)
	if err != nil {
		logging.Warnf("[mist] Failed to get sites for WLAN fetch: %v", err)
	} else {
		siteWLANCount := 0
		for _, site := range sites {
			if site.ID == nil {
				continue
			}
			siteWLANs, err := s.client.GetSiteWLANs(ctx, *site.ID)
			if err != nil {
				logging.Debugf("[mist] Failed to get WLANs for site %s: %v", *site.ID, err)
				continue
			}
			for _, w := range siteWLANs {
				wlan := convertMistWLAN(&w, s.orgID)
				result = append(result, wlan)
				siteWLANCount++
			}
		}
		logging.Debugf("[mist] Fetched %d site-level WLANs from %d sites", siteWLANCount, len(sites))
	}

	logging.Debugf("[mist] Fetched %d total WLANs", len(result))
	return result, nil
}

// ListBySite returns WLANs for a specific site.
// For Mist, this returns site-level WLANs plus org-level WLANs (which apply to all sites).
func (s *wlansService) ListBySite(ctx context.Context, siteID string) ([]*vendors.WLAN, error) {
	logging.Debugf("[mist] Fetching WLANs for site %s in org %s", siteID, s.orgID)

	var result []*vendors.WLAN

	// 1. Fetch org-level WLANs (these apply to all sites)
	orgWLANs, err := s.client.GetWLANs(ctx, s.orgID)
	if err != nil {
		logging.Warnf("[mist] Failed to get org-level WLANs: %v", err)
	} else {
		for _, w := range orgWLANs {
			wlan := convertMistWLAN(&w, s.orgID)
			result = append(result, wlan)
		}
	}

	// 2. Fetch site-level WLANs for this specific site
	siteWLANs, err := s.client.GetSiteWLANs(ctx, siteID)
	if err != nil {
		logging.Warnf("[mist] Failed to get WLANs for site %s: %v", siteID, err)
	} else {
		for _, w := range siteWLANs {
			wlan := convertMistWLAN(&w, s.orgID)
			result = append(result, wlan)
		}
	}

	logging.Debugf("[mist] Fetched %d WLANs for site %s", len(result), siteID)
	return result, nil
}

// Get returns a specific WLAN by ID.
func (s *wlansService) Get(ctx context.Context, id string) (*vendors.WLAN, error) {
	// Mist doesn't have a direct get-by-ID endpoint for org WLANs,
	// so we fetch all and filter
	wlans, err := s.List(ctx)
	if err != nil {
		return nil, err
	}

	for _, w := range wlans {
		if w.ID == id {
			return w, nil
		}
	}

	return nil, fmt.Errorf("WLAN not found: %s", id)
}

// BySSID finds WLANs by their SSID name.
func (s *wlansService) BySSID(ctx context.Context, ssid string) ([]*vendors.WLAN, error) {
	wlans, err := s.List(ctx)
	if err != nil {
		return nil, err
	}

	var result []*vendors.WLAN
	for _, w := range wlans {
		if w.SSID == ssid {
			result = append(result, w)
		}
	}

	return result, nil
}

// Create creates a new WLAN.
// Note: Not implemented for read-only cache population.
func (s *wlansService) Create(_ context.Context, _ *vendors.WLAN) (*vendors.WLAN, error) {
	return nil, fmt.Errorf("WLAN creation not implemented")
}

// Update modifies an existing WLAN.
// Note: Not implemented for read-only cache population.
func (s *wlansService) Update(_ context.Context, _ string, _ *vendors.WLAN) (*vendors.WLAN, error) {
	return nil, fmt.Errorf("WLAN update not implemented")
}

// Delete removes a WLAN.
// Note: Not implemented for read-only cache population.
func (s *wlansService) Delete(_ context.Context, _ string) error {
	return fmt.Errorf("WLAN deletion not implemented")
}

// convertMistWLAN converts a Mist API WLAN to the vendor-agnostic WLAN type.
func convertMistWLAN(w *api.MistWLAN, orgID string) *vendors.WLAN {
	wlan := &vendors.WLAN{
		OrgID:        orgID,
		SourceVendor: "mist",
	}

	// Core fields
	if w.ID != nil {
		wlan.ID = *w.ID
	}
	if w.SSID != nil {
		wlan.SSID = *w.SSID
	}
	if w.SiteID != nil {
		wlan.SiteID = *w.SiteID
	}

	// Status
	if w.Enabled != nil {
		wlan.Enabled = *w.Enabled
	}
	if w.Hidden != nil {
		wlan.Hidden = *w.Hidden
	}

	// Network settings
	if w.VlanID != nil {
		wlan.VLANID = *w.VlanID
	}
	if w.Band != nil {
		wlan.Band = *w.Band
	}

	// Authentication
	if w.Auth.Type != nil {
		wlan.AuthType = *w.Auth.Type
	}
	// Don't copy PSK to cache for security - will be populated from user config when needed

	// RADIUS servers for enterprise auth
	if w.Auth.Enterprise != nil && w.Auth.Enterprise.Radius != nil {
		server := vendors.RadiusServer{}
		if w.Auth.Enterprise.Radius.Host != nil {
			server.Host = *w.Auth.Enterprise.Radius.Host
		}
		if w.Auth.Enterprise.Radius.Port != nil {
			server.Port = *w.Auth.Enterprise.Radius.Port
		}
		// Don't copy RADIUS secret to cache for security
		wlan.RadiusServers = []vendors.RadiusServer{server}
	}

	// Store full config map for round-trip accuracy
	wlan.Config = w.ToMap()
	// Remove sensitive data from cached config
	if auth, ok := wlan.Config["auth"].(map[string]interface{}); ok {
		delete(auth, "psk")
		if enterprise, ok := auth["enterprise"].(map[string]interface{}); ok {
			if radius, ok := enterprise["radius"].(map[string]interface{}); ok {
				delete(radius, "secret")
			}
		}
	}

	return wlan
}

// Ensure wlansService implements vendors.WLANsService at compile time.
var _ vendors.WLANsService = (*wlansService)(nil)

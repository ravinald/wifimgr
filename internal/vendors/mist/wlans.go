package mist

import (
	"context"
	"encoding/json"
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
// Routes to org-level or site-level endpoint based on SiteID.
func (s *wlansService) Create(ctx context.Context, wlan *vendors.WLAN) (*vendors.WLAN, error) {
	mistWLAN := convertVendorWLANToMist(wlan)

	var created *api.MistWLAN
	var err error

	if wlan.SiteID == "" {
		// Create org-level WLAN
		logging.Debugf("[mist] Creating org-level WLAN: %s", wlan.SSID)
		created, err = s.client.CreateOrgWLAN(ctx, s.orgID, mistWLAN)
	} else {
		// Create site-level WLAN
		logging.Debugf("[mist] Creating site-level WLAN: %s for site %s", wlan.SSID, wlan.SiteID)
		created, err = s.client.CreateSiteWLAN(ctx, wlan.SiteID, mistWLAN)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create WLAN: %w", err)
	}

	return convertMistWLAN(created, s.orgID), nil
}

// Update modifies an existing WLAN.
// Routes to org-level or site-level endpoint based on SiteID.
func (s *wlansService) Update(ctx context.Context, id string, wlan *vendors.WLAN) (*vendors.WLAN, error) {
	mistWLAN := convertVendorWLANToMist(wlan)

	var updated *api.MistWLAN
	var err error

	if wlan.SiteID == "" {
		// Update org-level WLAN
		logging.Debugf("[mist] Updating org-level WLAN: %s", id)
		updated, err = s.client.UpdateOrgWLAN(ctx, s.orgID, id, mistWLAN)
	} else {
		// Update site-level WLAN
		logging.Debugf("[mist] Updating site-level WLAN: %s for site %s", id, wlan.SiteID)
		updated, err = s.client.UpdateSiteWLAN(ctx, wlan.SiteID, id, mistWLAN)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to update WLAN: %w", err)
	}

	return convertMistWLAN(updated, s.orgID), nil
}

// Delete removes a WLAN.
// Must fetch the WLAN first to determine org vs site scope.
func (s *wlansService) Delete(ctx context.Context, id string) error {
	// Fetch the WLAN to determine its scope (org or site level)
	wlan, err := s.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get WLAN for deletion: %w", err)
	}

	if wlan.SiteID == "" {
		// Delete org-level WLAN
		logging.Debugf("[mist] Deleting org-level WLAN: %s", id)
		if err := s.client.DeleteOrgWLAN(ctx, s.orgID, id); err != nil {
			return fmt.Errorf("failed to delete org WLAN: %w", err)
		}
	} else {
		// Delete site-level WLAN
		logging.Debugf("[mist] Deleting site-level WLAN: %s from site %s", id, wlan.SiteID)
		if err := s.client.DeleteSiteWLAN(ctx, wlan.SiteID, id); err != nil {
			return fmt.Errorf("failed to delete site WLAN: %w", err)
		}
	}

	return nil
}

// convertVendorWLANToMist converts a vendor-agnostic WLAN to Mist API WLAN type.
func convertVendorWLANToMist(w *vendors.WLAN) *api.MistWLAN {
	mist := &api.MistWLAN{
		AdditionalConfig: make(map[string]interface{}),
	}

	// Copy explicit struct fields
	if w.SSID != "" {
		mist.SSID = &w.SSID
	}
	mist.Enabled = &w.Enabled
	if w.Hidden {
		mist.Hidden = &w.Hidden
	}
	if w.Band != "" {
		mist.Band = &w.Band
	}
	if w.VLANID != 0 {
		mist.VlanID = &w.VLANID
	}
	if w.AuthType != "" {
		mist.Auth.Type = &w.AuthType
	}
	if w.PSK != "" {
		mist.Auth.PSK = &w.PSK
	}

	// Copy RADIUS servers for enterprise auth
	if len(w.RadiusServers) > 0 {
		rs := w.RadiusServers[0]
		mist.Auth.Enterprise = &struct {
			Radius *struct {
				Host   *string `json:"host,omitempty"`
				Port   *int    `json:"port,omitempty"`
				Secret *string `json:"secret,omitempty"`
			} `json:"radius,omitempty"`
		}{
			Radius: &struct {
				Host   *string `json:"host,omitempty"`
				Port   *int    `json:"port,omitempty"`
				Secret *string `json:"secret,omitempty"`
			}{},
		}
		if rs.Host != "" {
			mist.Auth.Enterprise.Radius.Host = &rs.Host
		}
		if rs.Port != 0 {
			mist.Auth.Enterprise.Radius.Port = &rs.Port
		}
		if rs.Secret != "" {
			mist.Auth.Enterprise.Radius.Secret = &rs.Secret
		}
	}

	// Merge any additional config fields from the original vendor response
	// This preserves fields not explicitly mapped in the struct
	if w.Config != nil {
		for key, value := range w.Config {
			// Skip explicitly mapped keys
			switch key {
			case "id", "ssid", "org_id", "site_id", "enabled", "hidden",
				"band", "vlan_id", "auth", "created_time", "modified_time":
				continue
			default:
				mist.AdditionalConfig[key] = value
			}
		}
	}

	return mist
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

	// Store full config map for round-trip accuracy using JSON marshaling
	wlan.Config = mistWLANToMap(w)
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

// mistWLANToMap converts a MistWLAN to a map using JSON marshaling.
func mistWLANToMap(w *api.MistWLAN) map[string]interface{} {
	data, err := json.Marshal(w)
	if err != nil {
		return nil
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil
	}
	return result
}

// Ensure wlansService implements vendors.WLANsService at compile time.
var _ vendors.WLANsService = (*wlansService)(nil)

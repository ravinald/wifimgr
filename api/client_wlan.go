package api

import (
	"context"
	"fmt"
	"net/http"
)

// WLAN API methods for the mistClient

// GetWLANs retrieves all WLANs for an organization (org-level WLANs)
func (c *mistClient) GetWLANs(ctx context.Context, orgID string) ([]MistWLAN, error) {
	var rawWLANs []map[string]interface{}
	path := fmt.Sprintf("/orgs/%s/wlans", orgID)

	if err := c.do(ctx, http.MethodGet, path, nil, &rawWLANs); err != nil {
		return nil, fmt.Errorf("failed to get WLANs: %w", err)
	}

	// Convert raw WLANs to MistWLAN structs using FromMap
	wlans := make([]MistWLAN, 0, len(rawWLANs))
	for _, rawWLAN := range rawWLANs {
		var wlan MistWLAN
		if err := wlan.FromMap(rawWLAN); err != nil {
			c.logDebug("Failed to convert WLAN: %v", err)
			continue
		}
		wlans = append(wlans, wlan)
	}

	c.logDebug("Retrieved %d org-level WLANs", len(wlans))
	return wlans, nil
}

// GetSiteWLANs retrieves all WLANs for a specific site (site-level WLANs)
func (c *mistClient) GetSiteWLANs(ctx context.Context, siteID string) ([]MistWLAN, error) {
	var rawWLANs []map[string]interface{}
	path := fmt.Sprintf("/sites/%s/wlans", siteID)

	if err := c.do(ctx, http.MethodGet, path, nil, &rawWLANs); err != nil {
		return nil, fmt.Errorf("failed to get site WLANs: %w", err)
	}

	// Convert raw WLANs to MistWLAN structs using FromMap
	wlans := make([]MistWLAN, 0, len(rawWLANs))
	for _, rawWLAN := range rawWLANs {
		var wlan MistWLAN
		if err := wlan.FromMap(rawWLAN); err != nil {
			c.logDebug("Failed to convert site WLAN: %v", err)
			continue
		}
		// Set the site ID on the WLAN if not already set
		if wlan.SiteID == nil {
			wlan.SiteID = &siteID
		}
		wlans = append(wlans, wlan)
	}

	c.logDebug("Retrieved %d site-level WLANs for site %s", len(wlans), siteID)
	return wlans, nil
}

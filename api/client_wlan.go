package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// WLAN API methods for the mistClient

// GetWLANs retrieves all WLANs for an organization (org-level WLANs)
func (c *mistClient) GetWLANs(ctx context.Context, orgID string) ([]MistWLAN, error) {
	var wlans []MistWLAN
	path := fmt.Sprintf("/orgs/%s/wlans", orgID)

	if err := c.do(ctx, http.MethodGet, path, nil, &wlans); err != nil {
		return nil, fmt.Errorf("failed to get WLANs: %w", err)
	}

	c.logDebug("Retrieved %d org-level WLANs", len(wlans))
	return wlans, nil
}

// GetSiteWLANs retrieves all WLANs for a specific site (site-level WLANs)
func (c *mistClient) GetSiteWLANs(ctx context.Context, siteID string) ([]MistWLAN, error) {
	var wlans []MistWLAN
	path := fmt.Sprintf("/sites/%s/wlans", siteID)

	if err := c.do(ctx, http.MethodGet, path, nil, &wlans); err != nil {
		return nil, fmt.Errorf("failed to get site WLANs: %w", err)
	}

	// Set the site ID on WLANs if not already set
	for i := range wlans {
		if wlans[i].SiteID == nil {
			wlans[i].SiteID = &siteID
		}
	}

	c.logDebug("Retrieved %d site-level WLANs for site %s", len(wlans), siteID)
	return wlans, nil
}

// CreateOrgWLAN creates a new org-level WLAN
func (c *mistClient) CreateOrgWLAN(ctx context.Context, orgID string, wlan *MistWLAN) (*MistWLAN, error) {
	if c.dryRun {
		c.logDebug("[DRY RUN] Would create org WLAN: %+v", wlan)
		simulatedID := "dry-run-wlan-id"
		return &MistWLAN{
			ID:    &simulatedID,
			SSID:  wlan.SSID,
			OrgID: &orgID,
		}, nil
	}

	// Marshal WLAN to JSON, then unmarshal to map to remove nil fields
	wlanJSON, err := json.Marshal(wlan)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal WLAN: %w", err)
	}
	var wlanData map[string]any
	if err := json.Unmarshal(wlanJSON, &wlanData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal WLAN: %w", err)
	}

	// Merge AdditionalConfig fields that json:"-" excludes
	for k, v := range wlan.AdditionalConfig {
		wlanData[k] = v
	}

	// Remove read-only fields
	delete(wlanData, "id")
	delete(wlanData, "org_id")
	delete(wlanData, "site_id")
	delete(wlanData, "created_time")
	delete(wlanData, "modified_time")

	var createdWLAN MistWLAN
	path := fmt.Sprintf("/orgs/%s/wlans", orgID)
	if err := c.do(ctx, http.MethodPost, path, wlanData, &createdWLAN); err != nil {
		return nil, fmt.Errorf("failed to create org WLAN: %w", err)
	}

	return &createdWLAN, nil
}

// CreateSiteWLAN creates a new site-level WLAN
func (c *mistClient) CreateSiteWLAN(ctx context.Context, siteID string, wlan *MistWLAN) (*MistWLAN, error) {
	if c.dryRun {
		c.logDebug("[DRY RUN] Would create site WLAN for site %s: %+v", siteID, wlan)
		simulatedID := "dry-run-wlan-id"
		return &MistWLAN{
			ID:     &simulatedID,
			SSID:   wlan.SSID,
			SiteID: &siteID,
		}, nil
	}

	// Marshal WLAN to JSON, then unmarshal to map to remove nil fields
	wlanJSON, err := json.Marshal(wlan)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal WLAN: %w", err)
	}
	var wlanData map[string]any
	if err := json.Unmarshal(wlanJSON, &wlanData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal WLAN: %w", err)
	}

	// Merge AdditionalConfig fields that json:"-" excludes
	for k, v := range wlan.AdditionalConfig {
		wlanData[k] = v
	}

	// Remove read-only fields
	delete(wlanData, "id")
	delete(wlanData, "org_id")
	delete(wlanData, "site_id")
	delete(wlanData, "created_time")
	delete(wlanData, "modified_time")

	var createdWLAN MistWLAN
	path := fmt.Sprintf("/sites/%s/wlans", siteID)
	if err := c.do(ctx, http.MethodPost, path, wlanData, &createdWLAN); err != nil {
		return nil, fmt.Errorf("failed to create site WLAN: %w", err)
	}

	return &createdWLAN, nil
}

// UpdateOrgWLAN updates an existing org-level WLAN
func (c *mistClient) UpdateOrgWLAN(ctx context.Context, orgID string, wlanID string, wlan *MistWLAN) (*MistWLAN, error) {
	if c.dryRun {
		c.logDebug("[DRY RUN] Would update org WLAN %s: %+v", wlanID, wlan)
		wlan.ID = &wlanID
		wlan.OrgID = &orgID
		return wlan, nil
	}

	// Marshal WLAN to JSON, then unmarshal to map to remove nil fields
	wlanJSON, err := json.Marshal(wlan)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal WLAN: %w", err)
	}
	var wlanData map[string]any
	if err := json.Unmarshal(wlanJSON, &wlanData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal WLAN: %w", err)
	}

	// Merge AdditionalConfig fields that json:"-" excludes
	for k, v := range wlan.AdditionalConfig {
		wlanData[k] = v
	}

	// Remove read-only fields
	delete(wlanData, "id")
	delete(wlanData, "org_id")
	delete(wlanData, "site_id")
	delete(wlanData, "created_time")
	delete(wlanData, "modified_time")

	var updatedWLAN MistWLAN
	path := fmt.Sprintf("/orgs/%s/wlans/%s", orgID, wlanID)
	if err := c.do(ctx, http.MethodPut, path, wlanData, &updatedWLAN); err != nil {
		return nil, fmt.Errorf("failed to update org WLAN: %w", err)
	}

	return &updatedWLAN, nil
}

// UpdateSiteWLAN updates an existing site-level WLAN
func (c *mistClient) UpdateSiteWLAN(ctx context.Context, siteID string, wlanID string, wlan *MistWLAN) (*MistWLAN, error) {
	if c.dryRun {
		c.logDebug("[DRY RUN] Would update site WLAN %s for site %s: %+v", wlanID, siteID, wlan)
		wlan.ID = &wlanID
		wlan.SiteID = &siteID
		return wlan, nil
	}

	// Marshal WLAN to JSON, then unmarshal to map to remove nil fields
	wlanJSON, err := json.Marshal(wlan)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal WLAN: %w", err)
	}
	var wlanData map[string]any
	if err := json.Unmarshal(wlanJSON, &wlanData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal WLAN: %w", err)
	}

	// Merge AdditionalConfig fields that json:"-" excludes
	for k, v := range wlan.AdditionalConfig {
		wlanData[k] = v
	}

	// Remove read-only fields
	delete(wlanData, "id")
	delete(wlanData, "org_id")
	delete(wlanData, "site_id")
	delete(wlanData, "created_time")
	delete(wlanData, "modified_time")

	var updatedWLAN MistWLAN
	path := fmt.Sprintf("/sites/%s/wlans/%s", siteID, wlanID)
	if err := c.do(ctx, http.MethodPut, path, wlanData, &updatedWLAN); err != nil {
		return nil, fmt.Errorf("failed to update site WLAN: %w", err)
	}

	return &updatedWLAN, nil
}

// DeleteOrgWLAN deletes an org-level WLAN
func (c *mistClient) DeleteOrgWLAN(ctx context.Context, orgID string, wlanID string) error {
	if c.dryRun {
		c.logDebug("[DRY RUN] Would delete org WLAN: %s", wlanID)
		return nil
	}

	path := fmt.Sprintf("/orgs/%s/wlans/%s", orgID, wlanID)
	if err := c.do(ctx, http.MethodDelete, path, nil, nil); err != nil {
		return fmt.Errorf("failed to delete org WLAN: %w", err)
	}

	return nil
}

// DeleteSiteWLAN deletes a site-level WLAN
func (c *mistClient) DeleteSiteWLAN(ctx context.Context, siteID string, wlanID string) error {
	if c.dryRun {
		c.logDebug("[DRY RUN] Would delete site WLAN: %s", wlanID)
		return nil
	}

	path := fmt.Sprintf("/sites/%s/wlans/%s", siteID, wlanID)
	if err := c.do(ctx, http.MethodDelete, path, nil, nil); err != nil {
		return fmt.Errorf("failed to delete site WLAN: %w", err)
	}

	return nil
}

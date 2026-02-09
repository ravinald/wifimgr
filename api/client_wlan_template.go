package api

import (
	"context"
	"fmt"
	"net/http"
)

// WLAN Template API methods for the mistClient

// GetWLANTemplates retrieves all WLAN templates for an organization
func (c *mistClient) GetWLANTemplates(ctx context.Context, orgID string) ([]MistWLANTemplate, error) {
	var templates []MistWLANTemplate
	path := fmt.Sprintf("/orgs/%s/templates", orgID)

	if err := c.do(ctx, http.MethodGet, path, nil, &templates); err != nil {
		return nil, fmt.Errorf("failed to get WLAN templates: %w", err)
	}

	c.logDebug("Retrieved %d WLAN templates", len(templates))
	return templates, nil
}

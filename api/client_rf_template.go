package api

import (
	"context"
	"fmt"
	"net/http"
)

// RF Template API methods for the mistClient

// GetRFTemplates retrieves all RF templates for an organization
func (c *mistClient) GetRFTemplates(ctx context.Context, orgID string) ([]MistRFTemplate, error) {
	var templates []MistRFTemplate
	path := fmt.Sprintf("/orgs/%s/rftemplates", orgID)

	if err := c.do(ctx, http.MethodGet, path, nil, &templates); err != nil {
		return nil, fmt.Errorf("failed to get RF templates: %w", err)
	}

	c.logDebug("Retrieved %d RF templates", len(templates))
	return templates, nil
}

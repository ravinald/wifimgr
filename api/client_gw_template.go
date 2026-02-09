package api

import (
	"context"
	"fmt"
	"net/http"
)

// Gateway Template API methods for the mistClient

// GetGatewayTemplates retrieves all gateway templates for an organization
func (c *mistClient) GetGatewayTemplates(ctx context.Context, orgID string) ([]MistGatewayTemplate, error) {
	var templates []MistGatewayTemplate
	path := fmt.Sprintf("/orgs/%s/gatewaytemplates", orgID)

	if err := c.do(ctx, http.MethodGet, path, nil, &templates); err != nil {
		return nil, fmt.Errorf("failed to get gateway templates: %w", err)
	}

	c.logDebug("Retrieved %d gateway templates", len(templates))
	return templates, nil
}

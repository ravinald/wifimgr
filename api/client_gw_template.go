package api

import (
	"context"
	"fmt"
	"net/http"
)

// Gateway Template API methods for the mistClient

// GetGatewayTemplates retrieves all gateway templates for an organization
func (c *mistClient) GetGatewayTemplates(ctx context.Context, orgID string) ([]MistGatewayTemplate, error) {
	var rawTemplates []map[string]interface{}
	path := fmt.Sprintf("/orgs/%s/gatewaytemplates", orgID)

	if err := c.do(ctx, http.MethodGet, path, nil, &rawTemplates); err != nil {
		return nil, fmt.Errorf("failed to get gateway templates: %w", err)
	}

	// Convert raw templates to MistGatewayTemplate structs using FromMap
	templates := make([]MistGatewayTemplate, 0, len(rawTemplates))
	for _, rawTemplate := range rawTemplates {
		var template MistGatewayTemplate
		if err := template.FromMap(rawTemplate); err != nil {
			c.logDebug("Failed to convert gateway template: %v", err)
			continue
		}
		templates = append(templates, template)
	}

	c.logDebug("Retrieved %d gateway templates", len(templates))
	return templates, nil
}

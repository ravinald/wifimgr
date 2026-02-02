package api

import (
	"context"
	"fmt"
	"net/http"
)

// RF Template API methods for the mistClient

// GetRFTemplates retrieves all RF templates for an organization
func (c *mistClient) GetRFTemplates(ctx context.Context, orgID string) ([]MistRFTemplate, error) {
	var rawTemplates []map[string]interface{}
	path := fmt.Sprintf("/orgs/%s/rftemplates", orgID)

	if err := c.do(ctx, http.MethodGet, path, nil, &rawTemplates); err != nil {
		return nil, fmt.Errorf("failed to get RF templates: %w", err)
	}

	// Convert raw templates to MistRFTemplate structs using FromMap
	templates := make([]MistRFTemplate, 0, len(rawTemplates))
	for _, rawTemplate := range rawTemplates {
		var template MistRFTemplate
		if err := template.FromMap(rawTemplate); err != nil {
			c.logDebug("Failed to convert RF template: %v", err)
			continue
		}
		templates = append(templates, template)
	}

	c.logDebug("Retrieved %d RF templates", len(templates))
	return templates, nil
}

package api

import (
	"context"
	"fmt"
	"net/http"
)

// WLAN Template API methods for the mistClient

// GetWLANTemplates retrieves all WLAN templates for an organization
func (c *mistClient) GetWLANTemplates(ctx context.Context, orgID string) ([]MistWLANTemplate, error) {
	var rawTemplates []map[string]interface{}
	path := fmt.Sprintf("/orgs/%s/templates", orgID)

	if err := c.do(ctx, http.MethodGet, path, nil, &rawTemplates); err != nil {
		return nil, fmt.Errorf("failed to get WLAN templates: %w", err)
	}

	// Convert raw templates to MistWLANTemplate structs using FromMap
	templates := make([]MistWLANTemplate, 0, len(rawTemplates))
	for _, rawTemplate := range rawTemplates {
		var template MistWLANTemplate
		if err := template.FromMap(rawTemplate); err != nil {
			c.logDebug("Failed to convert WLAN template: %v", err)
			continue
		}
		templates = append(templates, template)
	}

	c.logDebug("Retrieved %d WLAN templates", len(templates))
	return templates, nil
}

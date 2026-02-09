package api

import (
	"context"
	"fmt"
	"net/http"
)

// Network API methods for the mistClient

// GetNetworks retrieves all networks for an organization
func (c *mistClient) GetNetworks(ctx context.Context, orgID string) ([]MistNetwork, error) {
	var networks []MistNetwork
	path := fmt.Sprintf("/orgs/%s/networks", orgID)

	if err := c.do(ctx, http.MethodGet, path, nil, &networks); err != nil {
		return nil, fmt.Errorf("failed to get networks: %w", err)
	}

	c.logDebug("Retrieved %d networks", len(networks))
	return networks, nil
}

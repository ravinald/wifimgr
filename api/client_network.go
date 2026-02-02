package api

import (
	"context"
	"fmt"
	"net/http"
)

// Network API methods for the mistClient

// GetNetworks retrieves all networks for an organization
func (c *mistClient) GetNetworks(ctx context.Context, orgID string) ([]MistNetwork, error) {
	var rawNetworks []map[string]interface{}
	path := fmt.Sprintf("/orgs/%s/networks", orgID)

	if err := c.do(ctx, http.MethodGet, path, nil, &rawNetworks); err != nil {
		return nil, fmt.Errorf("failed to get networks: %w", err)
	}

	// Convert raw networks to MistNetwork structs using FromMap
	networks := make([]MistNetwork, 0, len(rawNetworks))
	for _, rawNetwork := range rawNetworks {
		var network MistNetwork
		if err := network.FromMap(rawNetwork); err != nil {
			c.logDebug("Failed to convert network: %v", err)
			continue
		}
		networks = append(networks, network)
	}

	c.logDebug("Retrieved %d networks", len(networks))
	return networks, nil
}

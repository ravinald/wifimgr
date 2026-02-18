package api

import (
	"context"
	"fmt"
	"net/http"
)

// GetAPStats retrieves AP statistics including radio details for a site.
// Returns raw JSON maps since the radio_stat structure is complex and only partially needed.
func (c *mistClient) GetAPStats(ctx context.Context, siteID string) ([]map[string]interface{}, error) {
	path := fmt.Sprintf("/sites/%s/stats/devices?type=ap", siteID)
	var result []map[string]interface{}
	if err := c.do(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, fmt.Errorf("failed to get AP stats: %w", err)
	}
	return result, nil
}

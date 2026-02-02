package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ravinald/wifimgr/internal/logging"
)

// GetOrgStats retrieves organization statistics from the API
func (c *mistClient) GetOrgStats(ctx context.Context, orgID string) (*OrgStats, error) {
	logger := logging.GetLogger()

	path := fmt.Sprintf("/orgs/%s/stats", orgID)
	logger.WithField("path", path).Debug("Fetching organization stats")

	var orgStats OrgStats
	if err := c.do(ctx, http.MethodGet, path, nil, &orgStats); err != nil {
		logger.WithError(err).Error("Failed to fetch organization stats")
		return nil, fmt.Errorf("failed to fetch organization stats: %w", err)
	}

	logger.WithFields(map[string]interface{}{
		"org_id":   orgID,
		"org_name": orgStats.Name,
	}).Debug("Successfully fetched organization stats")

	return &orgStats, nil
}

// GetOrganizations retrieves all organizations the user has access to
// This is useful for multi-org scenarios but not implemented yet
func (c *mistClient) GetOrganizations(_ context.Context) ([]*OrgStats, error) {
	// This would require a different API endpoint that lists all orgs
	// For now, we only support single org scenario
	return nil, fmt.Errorf("multi-org support not implemented")
}

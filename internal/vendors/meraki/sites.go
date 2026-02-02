package meraki

import (
	"context"
	"fmt"

	meraki "github.com/meraki/dashboard-api-go/v5/sdk"

	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// sitesService implements vendors.SitesService for Meraki.
// In Meraki, Sites map to Networks.
type sitesService struct {
	dashboard      *meraki.Client
	orgID          string
	rateLimiter    *RateLimiter
	retryConfig    *RetryConfig
	suppressOutput bool
}

// List returns all networks in the organization.
func (s *sitesService) List(ctx context.Context) ([]*vendors.SiteInfo, error) {
	logging.Debugf("[meraki] Fetching networks for org %s", s.orgID)

	params := &meraki.GetOrganizationNetworksQueryParams{
		PerPage: -1, // Fetch all
	}

	retryState := NewRetryState(s.retryConfig)
	var networks *meraki.ResponseOrganizationsGetOrganizationNetworks
	var err error

	for {
		// Acquire rate limit token
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return nil, fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		var resp *meraki.ResponseOrganizationsGetOrganizationNetworks
		if s.suppressOutput {
			restore := suppressStdout()
			resp, _, err = s.dashboard.Organizations.GetOrganizationNetworks(s.orgID, params)
			restore()
		} else {
			resp, _, err = s.dashboard.Organizations.GetOrganizationNetworks(s.orgID, params)
		}
		networks = resp

		if err == nil {
			break
		}

		// Check if we should retry on 429
		if !retryState.ShouldRetry(err) {
			logging.Debugf("[meraki] Failed to get networks: %v", err)
			return nil, fmt.Errorf("failed to get networks: %w", err)
		}

		// Wait before retry (handles Retry-After header)
		if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
			return nil, fmt.Errorf("retry wait failed: %w", waitErr)
		}
	}

	if networks == nil {
		logging.Debug("[meraki] No networks returned")
		return []*vendors.SiteInfo{}, nil
	}

	sites := make([]*vendors.SiteInfo, 0, len(*networks))
	for i := range *networks {
		site := convertNetworkToSiteInfo(&(*networks)[i])
		if site != nil {
			sites = append(sites, site)
		}
	}

	logging.Debugf("[meraki] Fetched %d networks", len(sites))
	return sites, nil
}

// Get finds a network by its ID.
func (s *sitesService) Get(ctx context.Context, id string) (*vendors.SiteInfo, error) {
	retryState := NewRetryState(s.retryConfig)
	var network *meraki.ResponseNetworksGetNetwork
	var err error

	for {
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return nil, fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		network, _, err = s.dashboard.Networks.GetNetwork(id)
		if err == nil {
			break
		}

		if !retryState.ShouldRetry(err) {
			return nil, fmt.Errorf("failed to get network %s: %w", id, err)
		}

		if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
			return nil, fmt.Errorf("retry wait failed: %w", waitErr)
		}
	}

	if network == nil {
		return nil, &vendors.SiteNotFoundError{SiteName: id, APILabel: "meraki"}
	}

	return &vendors.SiteInfo{
		ID:           network.ID,
		Name:         network.Name,
		Timezone:     network.TimeZone,
		Notes:        network.Notes,
		SourceVendor: "meraki",
	}, nil
}

// ByName finds a network by its name.
func (s *sitesService) ByName(ctx context.Context, name string) (*vendors.SiteInfo, error) {
	sites, err := s.List(ctx)
	if err != nil {
		return nil, err
	}

	for _, site := range sites {
		if site.Name == name {
			return site, nil
		}
	}

	return nil, &vendors.SiteNotFoundError{SiteName: name, APILabel: "meraki"}
}

// Create creates a new network.
func (s *sitesService) Create(ctx context.Context, site *vendors.SiteInfo) (*vendors.SiteInfo, error) {
	if site == nil {
		return nil, fmt.Errorf("site cannot be nil")
	}

	request := &meraki.RequestOrganizationsCreateOrganizationNetwork{
		Name:     site.Name,
		TimeZone: site.Timezone,
		Notes:    site.Notes,
		// ProductTypes is required - default to combined (all types)
		ProductTypes: []string{"appliance", "switch", "wireless"},
	}

	retryState := NewRetryState(s.retryConfig)
	var network *meraki.ResponseOrganizationsCreateOrganizationNetwork
	var err error

	for {
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return nil, fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		network, _, err = s.dashboard.Organizations.CreateOrganizationNetwork(s.orgID, request)
		if err == nil {
			break
		}

		if !retryState.ShouldRetry(err) {
			return nil, fmt.Errorf("failed to create network: %w", err)
		}

		if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
			return nil, fmt.Errorf("retry wait failed: %w", waitErr)
		}
	}

	return &vendors.SiteInfo{
		ID:           network.ID,
		Name:         network.Name,
		Timezone:     network.TimeZone,
		Notes:        network.Notes,
		SourceVendor: "meraki",
	}, nil
}

// Update modifies an existing network.
func (s *sitesService) Update(ctx context.Context, id string, site *vendors.SiteInfo) (*vendors.SiteInfo, error) {
	if site == nil {
		return nil, fmt.Errorf("site cannot be nil")
	}

	request := &meraki.RequestNetworksUpdateNetwork{
		Name:     site.Name,
		TimeZone: site.Timezone,
		Notes:    site.Notes,
	}

	retryState := NewRetryState(s.retryConfig)
	var network *meraki.ResponseNetworksUpdateNetwork
	var err error

	for {
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return nil, fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		network, _, err = s.dashboard.Networks.UpdateNetwork(id, request)
		if err == nil {
			break
		}

		if !retryState.ShouldRetry(err) {
			return nil, fmt.Errorf("failed to update network %s: %w", id, err)
		}

		if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
			return nil, fmt.Errorf("retry wait failed: %w", waitErr)
		}
	}

	return &vendors.SiteInfo{
		ID:           network.ID,
		Name:         network.Name,
		Timezone:     network.TimeZone,
		Notes:        network.Notes,
		SourceVendor: "meraki",
	}, nil
}

// Delete removes a network.
func (s *sitesService) Delete(ctx context.Context, id string) error {
	retryState := NewRetryState(s.retryConfig)

	for {
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		_, err := s.dashboard.Networks.DeleteNetwork(id)
		if err == nil {
			return nil
		}

		if !retryState.ShouldRetry(err) {
			return fmt.Errorf("failed to delete network %s: %w", id, err)
		}

		if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
			return fmt.Errorf("retry wait failed: %w", waitErr)
		}
	}
}

// Ensure sitesService implements vendors.SitesService at compile time.
var _ vendors.SitesService = (*sitesService)(nil)

package mist

import (
	"context"
	"fmt"
	"time"

	"github.com/ravinald/wifimgr/api"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// searchService implements vendors.SearchService for Mist.
type searchService struct {
	client api.Client
	orgID  string
}

// SearchWiredClients searches for wired clients by text (hostname, MAC, IP, etc.).
// The opts.SiteID can be used to scope the search to a specific site.
func (s *searchService) SearchWiredClients(ctx context.Context, text string, opts vendors.SearchOptions) (*vendors.WiredSearchResults, error) {
	// Mist API supports flexible text search with a single API call.
	// The siteID is passed to the API for site-scoped searches.
	response, err := s.client.SearchWiredClients(ctx, s.orgID, text)
	if err != nil {
		return nil, fmt.Errorf("failed to search wired clients: %w", err)
	}

	if response == nil {
		return &vendors.WiredSearchResults{
			Results: []*vendors.WiredClient{},
			Total:   0,
		}, nil
	}

	results := make([]*vendors.WiredClient, 0, len(response.Results))
	for _, client := range response.Results {
		// If site filter is specified, filter results by site
		if opts.SiteID != "" && client.GetSiteID() != opts.SiteID {
			continue
		}
		wc := convertWiredClientToVendor(client)
		if wc != nil {
			results = append(results, wc)
		}
	}

	total := len(results)

	return &vendors.WiredSearchResults{
		Results: results,
		Total:   total,
	}, nil
}

// SearchWirelessClients searches for wireless clients by text.
// The opts.SiteID can be used to scope the search to a specific site.
func (s *searchService) SearchWirelessClients(ctx context.Context, text string, opts vendors.SearchOptions) (*vendors.WirelessSearchResults, error) {
	// Mist API supports flexible text search with a single API call.
	response, err := s.client.SearchWirelessClients(ctx, s.orgID, text)
	if err != nil {
		return nil, fmt.Errorf("failed to search wireless clients: %w", err)
	}

	if response == nil {
		return &vendors.WirelessSearchResults{
			Results: []*vendors.WirelessClient{},
			Total:   0,
		}, nil
	}

	results := make([]*vendors.WirelessClient, 0, len(response.Results))
	for _, client := range response.Results {
		// If site filter is specified, filter results by site
		if opts.SiteID != "" && client.GetSiteID() != opts.SiteID {
			continue
		}
		wc := convertWirelessClientToVendor(client)
		if wc != nil {
			results = append(results, wc)
		}
	}

	total := len(results)

	return &vendors.WirelessSearchResults{
		Results: results,
		Total:   total,
	}, nil
}

// EstimateSearchCost returns the estimated cost of a search operation.
// Mist API always uses a single API call for searches, so cost is always low.
func (s *searchService) EstimateSearchCost(_ context.Context, _ string, _ string) (*vendors.SearchCostEstimate, error) {
	return &vendors.SearchCostEstimate{
		APICalls:          1,
		EstimatedDuration: 2 * time.Second,
		NeedsConfirmation: false,
		Description:       "Single API call",
	}, nil
}

// Ensure searchService implements vendors.SearchService at compile time.
var _ vendors.SearchService = (*searchService)(nil)

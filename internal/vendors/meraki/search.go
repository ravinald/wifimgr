package meraki

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	meraki "github.com/meraki/dashboard-api-go/v5/sdk"

	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/macaddr"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// searchService implements vendors.SearchService for Meraki.
type searchService struct {
	dashboard      *meraki.Client
	orgID          string
	rateLimiter    *RateLimiter
	retryConfig    *RetryConfig
	suppressOutput bool
}

// SearchWirelessClients searches for wireless clients by text.
// Meraki has different search strategies based on input type:
// - MAC address: Uses org-wide client search (1 API call)
// - IP address: Queries networks with IP filter
// - Other text: Must query all networks and filter locally (expensive)
func (s *searchService) SearchWirelessClients(ctx context.Context, text string, opts vendors.SearchOptions) (*vendors.WirelessSearchResults, error) {
	if opts.SiteID != "" {
		return s.searchNetworkWirelessClients(ctx, opts.SiteID, text)
	}

	// MAC address search uses org-wide endpoint
	if macaddr.IsValid(text) {
		return s.searchOrgWirelessByMAC(ctx, text)
	}

	// IP or text search requires querying all networks
	return s.searchAllNetworksWireless(ctx, text)
}

// SearchWiredClients searches for wired clients by text.
func (s *searchService) SearchWiredClients(ctx context.Context, text string, opts vendors.SearchOptions) (*vendors.WiredSearchResults, error) {
	if opts.SiteID != "" {
		return s.searchNetworkWiredClients(ctx, opts.SiteID, text)
	}

	// MAC address search uses org-wide endpoint
	if macaddr.IsValid(text) {
		return s.searchOrgWiredByMAC(ctx, text)
	}

	// IP or text search requires querying all networks
	return s.searchAllNetworksWired(ctx, text)
}

// EstimateSearchCost returns the estimated cost of a search operation.
func (s *searchService) EstimateSearchCost(ctx context.Context, text string, siteID string) (*vendors.SearchCostEstimate, error) {
	// Site-scoped search is always 1 API call
	if siteID != "" {
		return &vendors.SearchCostEstimate{
			APICalls:          1,
			EstimatedDuration: 2 * time.Second,
			NeedsConfirmation: false,
			Description:       "Single network query",
		}, nil
	}

	// MAC search uses org-wide endpoint (1 API call)
	if macaddr.IsValid(text) {
		return &vendors.SearchCostEstimate{
			APICalls:          1,
			EstimatedDuration: 2 * time.Second,
			NeedsConfirmation: false,
			Description:       "Organization-wide MAC lookup",
		}, nil
	}

	// IP or text search requires querying all networks
	networkCount, err := s.getNetworkCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to estimate search cost: %w", err)
	}

	// Threshold for confirmation (configurable in future)
	const confirmationThreshold = 5

	return &vendors.SearchCostEstimate{
		APICalls:          networkCount,
		EstimatedDuration: time.Duration(networkCount) * 500 * time.Millisecond,
		NeedsConfirmation: networkCount > confirmationThreshold,
		Description:       fmt.Sprintf("Requires querying %d networks", networkCount),
	}, nil
}

// getNetworkCount returns the number of networks in the organization.
func (s *searchService) getNetworkCount(ctx context.Context) (int, error) {
	params := &meraki.GetOrganizationNetworksQueryParams{
		PerPage: -1,
	}

	retryState := NewRetryState(s.retryConfig)
	var networks *meraki.ResponseOrganizationsGetOrganizationNetworks

	for {
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return 0, fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		var err error
		if s.suppressOutput {
			restore := suppressStdout()
			networks, _, err = s.dashboard.Organizations.GetOrganizationNetworks(s.orgID, params)
			restore()
		} else {
			networks, _, err = s.dashboard.Organizations.GetOrganizationNetworks(s.orgID, params)
		}

		if err == nil {
			break
		}

		if !retryState.ShouldRetry(err) {
			return 0, fmt.Errorf("failed to get network count: %w", err)
		}

		if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
			return 0, fmt.Errorf("retry wait failed: %w", waitErr)
		}
	}

	if networks == nil {
		return 0, nil
	}

	return len(*networks), nil
}

// searchOrgWirelessByMAC searches for a wireless client by MAC using the org-wide endpoint.
func (s *searchService) searchOrgWirelessByMAC(ctx context.Context, mac string) (*vendors.WirelessSearchResults, error) {
	logging.Debugf("[meraki] Searching org for wireless client by MAC: %s", mac)

	// Normalize MAC for the API (Meraki expects format like "aa:bb:cc:dd:ee:ff")
	formattedMAC, _ := macaddr.Format(mac, macaddr.FormatColon)

	params := &meraki.GetOrganizationClientsSearchQueryParams{
		Mac: formattedMAC,
	}

	retryState := NewRetryState(s.retryConfig)
	var response *meraki.ResponseOrganizationsGetOrganizationClientsSearch

	for {
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return nil, fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		var err error
		if s.suppressOutput {
			restore := suppressStdout()
			response, _, err = s.dashboard.Organizations.GetOrganizationClientsSearch(s.orgID, params)
			restore()
		} else {
			response, _, err = s.dashboard.Organizations.GetOrganizationClientsSearch(s.orgID, params)
		}

		if err == nil {
			break
		}

		if !retryState.ShouldRetry(err) {
			return nil, fmt.Errorf("failed to search clients: %w", err)
		}

		if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
			return nil, fmt.Errorf("retry wait failed: %w", waitErr)
		}
	}

	results := &vendors.WirelessSearchResults{
		Results: []*vendors.WirelessClient{},
		Total:   0,
	}

	if response == nil || response.Records == nil {
		return results, nil
	}

	// Convert response to vendor-agnostic format
	// The top-level response has Mac/Manufacturer, Records have the sightings
	for i := range *response.Records {
		record := &(*response.Records)[i]
		// For org-wide search, determine wireless by checking if SSID is set
		if record.SSID != "" {
			client := convertOrgClientSearchToWirelessClient(response, record)
			if client != nil {
				results.Results = append(results.Results, client)
			}
		}
	}

	results.Total = len(results.Results)
	return results, nil
}

// searchOrgWiredByMAC searches for a wired client by MAC using the org-wide endpoint.
func (s *searchService) searchOrgWiredByMAC(ctx context.Context, mac string) (*vendors.WiredSearchResults, error) {
	logging.Debugf("[meraki] Searching org for wired client by MAC: %s", mac)

	formattedMAC, _ := macaddr.Format(mac, macaddr.FormatColon)

	params := &meraki.GetOrganizationClientsSearchQueryParams{
		Mac: formattedMAC,
	}

	retryState := NewRetryState(s.retryConfig)
	var response *meraki.ResponseOrganizationsGetOrganizationClientsSearch

	for {
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return nil, fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		var err error
		if s.suppressOutput {
			restore := suppressStdout()
			response, _, err = s.dashboard.Organizations.GetOrganizationClientsSearch(s.orgID, params)
			restore()
		} else {
			response, _, err = s.dashboard.Organizations.GetOrganizationClientsSearch(s.orgID, params)
		}

		if err == nil {
			break
		}

		if !retryState.ShouldRetry(err) {
			return nil, fmt.Errorf("failed to search clients: %w", err)
		}

		if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
			return nil, fmt.Errorf("retry wait failed: %w", waitErr)
		}
	}

	results := &vendors.WiredSearchResults{
		Results: []*vendors.WiredClient{},
		Total:   0,
	}

	if response == nil || response.Records == nil {
		return results, nil
	}

	// Convert response to vendor-agnostic format
	// The top-level response has Mac/Manufacturer, Records have the sightings
	for i := range *response.Records {
		record := &(*response.Records)[i]
		// For org-wide search, determine wired by checking if Switchport is set or SSID is empty
		if record.Switchport != "" || record.SSID == "" {
			client := convertOrgClientSearchToWiredClient(response, record)
			if client != nil {
				results.Results = append(results.Results, client)
			}
		}
	}

	results.Total = len(results.Results)
	return results, nil
}

// searchNetworkWirelessClients searches for wireless clients in a specific network.
func (s *searchService) searchNetworkWirelessClients(ctx context.Context, networkID, text string) (*vendors.WirelessSearchResults, error) {
	logging.Debugf("[meraki] Searching network %s for wireless clients: %s", networkID, text)

	params := &meraki.GetNetworkClientsQueryParams{
		PerPage:                 -1,
		RecentDeviceConnections: []string{"Wireless"},
	}

	// Add filter based on input type
	if macaddr.IsValid(text) {
		params.Mac = text
	} else if isIPAddress(text) {
		params.IP = text
	}

	retryState := NewRetryState(s.retryConfig)
	var clients *meraki.ResponseNetworksGetNetworkClients

	for {
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return nil, fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		var err error
		if s.suppressOutput {
			restore := suppressStdout()
			clients, _, err = s.dashboard.Networks.GetNetworkClients(networkID, params)
			restore()
		} else {
			clients, _, err = s.dashboard.Networks.GetNetworkClients(networkID, params)
		}

		if err == nil {
			break
		}

		if !retryState.ShouldRetry(err) {
			return nil, fmt.Errorf("failed to get network clients: %w", err)
		}

		if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
			return nil, fmt.Errorf("retry wait failed: %w", waitErr)
		}
	}

	results := &vendors.WirelessSearchResults{
		Results: []*vendors.WirelessClient{},
		Total:   0,
	}

	if clients == nil {
		return results, nil
	}

	// Convert and filter results
	for i := range *clients {
		client := &(*clients)[i]
		// Local text filter if not MAC or IP (description field)
		if !macaddr.IsValid(text) && !isIPAddress(text) {
			if !matchesText(client, text) {
				continue
			}
		}
		wc := convertNetworkClientToWirelessClient(client, networkID)
		if wc != nil {
			results.Results = append(results.Results, wc)
		}
	}

	results.Total = len(results.Results)
	return results, nil
}

// searchNetworkWiredClients searches for wired clients in a specific network.
func (s *searchService) searchNetworkWiredClients(ctx context.Context, networkID, text string) (*vendors.WiredSearchResults, error) {
	logging.Debugf("[meraki] Searching network %s for wired clients: %s", networkID, text)

	params := &meraki.GetNetworkClientsQueryParams{
		PerPage:                 -1,
		RecentDeviceConnections: []string{"Wired"},
	}

	// Add filter based on input type
	if macaddr.IsValid(text) {
		params.Mac = text
	} else if isIPAddress(text) {
		params.IP = text
	}

	retryState := NewRetryState(s.retryConfig)
	var clients *meraki.ResponseNetworksGetNetworkClients

	for {
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return nil, fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		var err error
		if s.suppressOutput {
			restore := suppressStdout()
			clients, _, err = s.dashboard.Networks.GetNetworkClients(networkID, params)
			restore()
		} else {
			clients, _, err = s.dashboard.Networks.GetNetworkClients(networkID, params)
		}

		if err == nil {
			break
		}

		if !retryState.ShouldRetry(err) {
			return nil, fmt.Errorf("failed to get network clients: %w", err)
		}

		if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
			return nil, fmt.Errorf("retry wait failed: %w", waitErr)
		}
	}

	results := &vendors.WiredSearchResults{
		Results: []*vendors.WiredClient{},
		Total:   0,
	}

	if clients == nil {
		return results, nil
	}

	// Convert and filter results
	for i := range *clients {
		client := &(*clients)[i]
		// Local text filter if not MAC or IP
		if !macaddr.IsValid(text) && !isIPAddress(text) {
			if !matchesText(client, text) {
				continue
			}
		}
		wc := convertNetworkClientToWiredClient(client, networkID)
		if wc != nil {
			results.Results = append(results.Results, wc)
		}
	}

	results.Total = len(results.Results)
	return results, nil
}

// searchAllNetworksWireless searches all networks for wireless clients (expensive operation).
func (s *searchService) searchAllNetworksWireless(ctx context.Context, text string) (*vendors.WirelessSearchResults, error) {
	logging.Debugf("[meraki] Searching all networks for wireless clients: %s", text)

	// Get all networks
	networks, err := s.getNetworks(ctx)
	if err != nil {
		return nil, err
	}

	results := &vendors.WirelessSearchResults{
		Results: []*vendors.WirelessClient{},
		Total:   0,
	}

	// Search each network
	for _, network := range networks {
		networkResults, err := s.searchNetworkWirelessClients(ctx, network.ID, text)
		if err != nil {
			logging.Debugf("[meraki] Error searching network %s: %v", network.Name, err)
			continue
		}
		results.Results = append(results.Results, networkResults.Results...)
	}

	results.Total = len(results.Results)
	return results, nil
}

// searchAllNetworksWired searches all networks for wired clients (expensive operation).
func (s *searchService) searchAllNetworksWired(ctx context.Context, text string) (*vendors.WiredSearchResults, error) {
	logging.Debugf("[meraki] Searching all networks for wired clients: %s", text)

	// Get all networks
	networks, err := s.getNetworks(ctx)
	if err != nil {
		return nil, err
	}

	results := &vendors.WiredSearchResults{
		Results: []*vendors.WiredClient{},
		Total:   0,
	}

	// Search each network
	for _, network := range networks {
		networkResults, err := s.searchNetworkWiredClients(ctx, network.ID, text)
		if err != nil {
			logging.Debugf("[meraki] Error searching network %s: %v", network.Name, err)
			continue
		}
		results.Results = append(results.Results, networkResults.Results...)
	}

	results.Total = len(results.Results)
	return results, nil
}

// getNetworks returns all networks in the organization.
func (s *searchService) getNetworks(ctx context.Context) ([]meraki.ResponseItemOrganizationsGetOrganizationNetworks, error) {
	params := &meraki.GetOrganizationNetworksQueryParams{
		PerPage: -1,
	}

	retryState := NewRetryState(s.retryConfig)
	var networks *meraki.ResponseOrganizationsGetOrganizationNetworks

	for {
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return nil, fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		var err error
		if s.suppressOutput {
			restore := suppressStdout()
			networks, _, err = s.dashboard.Organizations.GetOrganizationNetworks(s.orgID, params)
			restore()
		} else {
			networks, _, err = s.dashboard.Organizations.GetOrganizationNetworks(s.orgID, params)
		}

		if err == nil {
			break
		}

		if !retryState.ShouldRetry(err) {
			return nil, fmt.Errorf("failed to get networks: %w", err)
		}

		if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
			return nil, fmt.Errorf("retry wait failed: %w", waitErr)
		}
	}

	if networks == nil {
		return []meraki.ResponseItemOrganizationsGetOrganizationNetworks{}, nil
	}

	return *networks, nil
}

// isIPAddress checks if the text looks like an IP address.
func isIPAddress(text string) bool {
	return net.ParseIP(text) != nil
}

// matchesText checks if a client matches the search text (case-insensitive).
// This is used for local filtering when the API doesn't support text search.
func matchesText(client *meraki.ResponseItemNetworksGetNetworkClients, text string) bool {
	text = strings.ToLower(text)

	// Check description (Meraki's equivalent of hostname)
	if client.Description != "" && strings.Contains(strings.ToLower(client.Description), text) {
		return true
	}

	// Check user field
	if client.User != "" && strings.Contains(strings.ToLower(client.User), text) {
		return true
	}

	// Check manufacturer
	if client.Manufacturer != "" && strings.Contains(strings.ToLower(client.Manufacturer), text) {
		return true
	}

	// Check OS
	if client.Os != "" && strings.Contains(strings.ToLower(client.Os), text) {
		return true
	}

	return false
}

// convertOrgClientSearchToWirelessClient converts org client search result to WirelessClient.
// Takes both the response (for Mac/Manufacturer) and the record (for sighting details).
func convertOrgClientSearchToWirelessClient(
	response *meraki.ResponseOrganizationsGetOrganizationClientsSearch,
	record *meraki.ResponseOrganizationsGetOrganizationClientsSearchRecords,
) *vendors.WirelessClient {
	if response == nil || record == nil {
		return nil
	}

	client := &vendors.WirelessClient{
		SourceVendor: "meraki",
		MAC:          response.Mac,
		Hostname:     record.Description,
		IP:           record.IP,
		SSID:         record.SSID,
		Manufacturer: response.Manufacturer,
		OS:           record.Os,
		APMAC:        record.RecentDeviceMac,
	}

	// VLAN is a string in Meraki, store as 0
	client.VLAN = 0

	if record.Network != nil {
		client.SiteID = record.Network.ID
		client.SiteName = record.Network.Name
	}

	return client
}

// convertOrgClientSearchToWiredClient converts org client search result to WiredClient.
// Takes both the response (for Mac/Manufacturer) and the record (for sighting details).
func convertOrgClientSearchToWiredClient(
	response *meraki.ResponseOrganizationsGetOrganizationClientsSearch,
	record *meraki.ResponseOrganizationsGetOrganizationClientsSearchRecords,
) *vendors.WiredClient {
	if response == nil || record == nil {
		return nil
	}

	client := &vendors.WiredClient{
		SourceVendor: "meraki",
		MAC:          response.Mac,
		Hostname:     record.Description,
		IP:           record.IP,
		Manufacturer: response.Manufacturer,
		SwitchMAC:    record.RecentDeviceMac,
		PortID:       record.Switchport,
	}

	// VLAN is a string in Meraki, store as 0
	client.VLAN = 0

	if record.Network != nil {
		client.SiteID = record.Network.ID
		client.SiteName = record.Network.Name
	}

	return client
}

// convertNetworkClientToWirelessClient converts network client to WirelessClient.
func convertNetworkClientToWirelessClient(client *meraki.ResponseItemNetworksGetNetworkClients, networkID string) *vendors.WirelessClient {
	if client == nil {
		return nil
	}

	wc := &vendors.WirelessClient{
		SiteID:       networkID,
		SourceVendor: "meraki",
		MAC:          client.Mac,
		Hostname:     client.Description,
		IP:           client.IP,
		SSID:         client.SSID,
		Manufacturer: client.Manufacturer,
		OS:           client.Os,
		APMAC:        client.RecentDeviceMac,
	}

	// VLAN is a string in Meraki, store as 0
	wc.VLAN = 0

	return wc
}

// convertNetworkClientToWiredClient converts network client to WiredClient.
func convertNetworkClientToWiredClient(client *meraki.ResponseItemNetworksGetNetworkClients, networkID string) *vendors.WiredClient {
	if client == nil {
		return nil
	}

	wc := &vendors.WiredClient{
		SiteID:       networkID,
		SourceVendor: "meraki",
		MAC:          client.Mac,
		Hostname:     client.Description,
		IP:           client.IP,
		Manufacturer: client.Manufacturer,
		SwitchMAC:    client.RecentDeviceMac,
		PortID:       client.Switchport,
	}

	// VLAN is a string in Meraki, store as 0
	wc.VLAN = 0

	return wc
}

// Ensure searchService implements vendors.SearchService at compile time.
var _ vendors.SearchService = (*searchService)(nil)

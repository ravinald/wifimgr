package meraki

import (
	"context"
	"encoding/json"
	"fmt"

	meraki "github.com/meraki/dashboard-api-go/v5/sdk"

	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// templatesService implements vendors.TemplatesService for Meraki.
// Meraki RF Profiles are per-network (site), unlike Mist which has org-level templates.
type templatesService struct {
	dashboard      *meraki.Client
	orgID          string
	rateLimiter    *RateLimiter
	retryConfig    *RetryConfig
	suppressOutput bool
}

// ListRF returns RF profiles from all networks in the organization.
// Meraki RF profiles are per-network, so we iterate through all networks
// that have wireless capability and collect their RF profiles.
func (t *templatesService) ListRF(ctx context.Context) ([]*vendors.RFTemplate, error) {
	logging.Debugf("[meraki] Fetching RF profiles for org %s", t.orgID)

	// First, get all networks to find which ones have wireless capability
	networks, err := t.getNetworksWithWireless(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get networks: %w", err)
	}

	var allProfiles []*vendors.RFTemplate
	profilesSeen := make(map[string]bool) // Deduplicate by ID

	for _, network := range networks {
		profiles, err := t.getRFProfilesForNetwork(ctx, network.ID)
		if err != nil {
			// Log warning but continue with other networks
			logging.Warnf("[meraki] Failed to get RF profiles for network %s: %v", network.ID, err)
			continue
		}

		for _, profile := range profiles {
			// Deduplicate - some profiles might be shared across networks
			if !profilesSeen[profile.ID] {
				profilesSeen[profile.ID] = true
				allProfiles = append(allProfiles, profile)
			}
		}
	}

	logging.Debugf("[meraki] Fetched %d RF profiles from %d networks", len(allProfiles), len(networks))
	return allProfiles, nil
}

// getNetworksWithWireless returns networks that have wireless product type.
func (t *templatesService) getNetworksWithWireless(ctx context.Context) ([]networkInfo, error) {
	params := &meraki.GetOrganizationNetworksQueryParams{
		PerPage: -1, // Fetch all
	}

	retryState := NewRetryState(t.retryConfig)
	var networks *meraki.ResponseOrganizationsGetOrganizationNetworks
	var err error

	for {
		if t.rateLimiter != nil {
			if err := t.rateLimiter.Acquire(ctx); err != nil {
				return nil, fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		if t.suppressOutput {
			restore := suppressStdout()
			networks, _, err = t.dashboard.Organizations.GetOrganizationNetworks(t.orgID, params)
			restore()
		} else {
			networks, _, err = t.dashboard.Organizations.GetOrganizationNetworks(t.orgID, params)
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
		return []networkInfo{}, nil
	}

	// Filter to networks with wireless capability
	var wirelessNetworks []networkInfo
	for _, n := range *networks {
		if hasWireless(n.ProductTypes) {
			wirelessNetworks = append(wirelessNetworks, networkInfo{
				ID:   n.ID,
				Name: n.Name,
			})
		}
	}

	logging.Debugf("[meraki] Found %d networks with wireless capability", len(wirelessNetworks))
	return wirelessNetworks, nil
}

// networkInfo holds minimal network info for RF profile fetching.
type networkInfo struct {
	ID   string
	Name string
}

// hasWireless checks if product types include wireless.
func hasWireless(productTypes []string) bool {
	for _, pt := range productTypes {
		if pt == "wireless" {
			return true
		}
	}
	return false
}

// getRFProfilesForNetwork fetches RF profiles for a specific network.
func (t *templatesService) getRFProfilesForNetwork(ctx context.Context, networkID string) ([]*vendors.RFTemplate, error) {
	retryState := NewRetryState(t.retryConfig)
	var profiles *meraki.ResponseWirelessGetNetworkWirelessRfProfiles
	var err error

	for {
		if t.rateLimiter != nil {
			if err := t.rateLimiter.Acquire(ctx); err != nil {
				return nil, fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		if t.suppressOutput {
			restore := suppressStdout()
			profiles, _, err = t.dashboard.Wireless.GetNetworkWirelessRfProfiles(networkID, nil)
			restore()
		} else {
			profiles, _, err = t.dashboard.Wireless.GetNetworkWirelessRfProfiles(networkID, nil)
		}

		if err == nil {
			break
		}

		if !retryState.ShouldRetry(err) {
			return nil, fmt.Errorf("failed to get RF profiles for network %s: %w", networkID, err)
		}

		if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
			return nil, fmt.Errorf("retry wait failed: %w", waitErr)
		}
	}

	if profiles == nil {
		logging.Debugf("[meraki] No RF profiles returned for network %s (nil response)", networkID)
		return []*vendors.RFTemplate{}, nil
	}

	if len(*profiles) == 0 {
		logging.Debugf("[meraki] No RF profiles defined for network %s", networkID)
		return []*vendors.RFTemplate{}, nil
	}

	result := make([]*vendors.RFTemplate, 0, len(*profiles))
	for _, p := range *profiles {
		logging.Debugf("[meraki] Found RF profile: ID=%s Name=%s NetworkID=%s", p.ID, p.Name, networkID)

		// Convert the full profile to a config map
		config := convertRFProfileToConfig(p)

		result = append(result, &vendors.RFTemplate{
			ID:           p.ID,
			Name:         p.Name,
			OrgID:        t.orgID,
			SiteID:       networkID, // Store the network ID for per-site lookups
			Config:       config,
			SourceVendor: "meraki",
		})
	}

	logging.Debugf("[meraki] Found %d RF profiles for network %s", len(result), networkID)
	return result, nil
}

// convertRFProfileToConfig converts a Meraki RF profile to a config map.
func convertRFProfileToConfig(p meraki.ResponseItemWirelessGetNetworkWirelessRfProfiles) map[string]interface{} {
	// Marshal to JSON and unmarshal to map to get all fields
	data, err := json.Marshal(p)
	if err != nil {
		logging.Warnf("[meraki] Failed to marshal RF profile: %v", err)
		return nil
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		logging.Warnf("[meraki] Failed to unmarshal RF profile to map: %v", err)
		return nil
	}

	// Remove fields that are already in the RFTemplate struct
	delete(config, "id")
	delete(config, "name")
	delete(config, "networkId") // We store this as site_id

	return config
}

// ListGateway returns gateway templates.
// Meraki doesn't have gateway templates in the same way as Mist.
func (t *templatesService) ListGateway(_ context.Context) ([]*vendors.GatewayTemplate, error) {
	// Meraki doesn't have gateway templates - appliance configs are per-device
	return []*vendors.GatewayTemplate{}, nil
}

// ListWLAN returns WLAN templates.
// Meraki doesn't have WLAN templates - SSIDs are configured per-network.
func (t *templatesService) ListWLAN(_ context.Context) ([]*vendors.WLANTemplate, error) {
	// Meraki doesn't have WLAN templates - SSIDs are per-network configurations
	return []*vendors.WLANTemplate{}, nil
}

// Ensure templatesService implements vendors.TemplatesService at compile time.
var _ vendors.TemplatesService = (*templatesService)(nil)

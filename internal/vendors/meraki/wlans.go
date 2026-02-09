package meraki

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	meraki "github.com/meraki/dashboard-api-go/v5/sdk"

	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// wlansService implements vendors.WLANsService for Meraki.
// Meraki SSIDs are per-network (numbered 0-14).
type wlansService struct {
	dashboard      *meraki.Client
	orgID          string
	rateLimiter    *RateLimiter
	retryConfig    *RetryConfig
	suppressOutput bool
}

// List returns all SSIDs from all networks in the organization.
func (s *wlansService) List(ctx context.Context) ([]*vendors.WLAN, error) {
	logging.Debugf("[meraki] Fetching SSIDs for org %s", s.orgID)

	// Get all networks with wireless capability
	networks, err := s.getNetworksWithWireless(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get networks: %w", err)
	}

	var allWLANs []*vendors.WLAN
	for _, network := range networks {
		wlans, err := s.getSSIDsForNetwork(ctx, network.ID, network.Name)
		if err != nil {
			logging.Warnf("[meraki] Failed to get SSIDs for network %s: %v", network.ID, err)
			continue
		}
		allWLANs = append(allWLANs, wlans...)
	}

	logging.Debugf("[meraki] Fetched %d SSIDs from %d networks", len(allWLANs), len(networks))
	return allWLANs, nil
}

// ListBySite returns SSIDs for a specific network.
func (s *wlansService) ListBySite(ctx context.Context, siteID string) ([]*vendors.WLAN, error) {
	logging.Debugf("[meraki] Fetching SSIDs for network %s", siteID)
	return s.getSSIDsForNetwork(ctx, siteID, "")
}

// Get returns a specific SSID by ID.
// The ID format is "networkId:ssidNumber".
func (s *wlansService) Get(ctx context.Context, id string) (*vendors.WLAN, error) {
	// Parse the composite ID
	networkID, number, err := parseSSIDID(id)
	if err != nil {
		return nil, err
	}

	retryState := NewRetryState(s.retryConfig)
	var ssid *meraki.ResponseWirelessGetNetworkWirelessSSID

	for {
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return nil, fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		var fetchErr error
		if s.suppressOutput {
			restore := suppressStdout()
			ssid, _, fetchErr = s.dashboard.Wireless.GetNetworkWirelessSSID(networkID, number)
			restore()
		} else {
			ssid, _, fetchErr = s.dashboard.Wireless.GetNetworkWirelessSSID(networkID, number)
		}

		if fetchErr == nil {
			break
		}

		if !retryState.ShouldRetry(fetchErr) {
			return nil, fmt.Errorf("failed to get SSID %s: %w", id, fetchErr)
		}

		if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
			return nil, fmt.Errorf("retry wait failed: %w", waitErr)
		}
	}

	if ssid == nil {
		return nil, fmt.Errorf("SSID not found: %s", id)
	}

	return convertMerakiSSID(ssid, networkID, s.orgID), nil
}

// BySSID finds SSIDs by their SSID name across all networks.
func (s *wlansService) BySSID(ctx context.Context, ssidName string) ([]*vendors.WLAN, error) {
	wlans, err := s.List(ctx)
	if err != nil {
		return nil, err
	}

	var result []*vendors.WLAN
	for _, w := range wlans {
		if w.SSID == ssidName {
			result = append(result, w)
		}
	}
	return result, nil
}

// Create creates/enables a new SSID.
// Meraki has no create endpoint - we find an available slot (0-14) and configure it via Update.
func (s *wlansService) Create(ctx context.Context, wlan *vendors.WLAN) (*vendors.WLAN, error) {
	if wlan.SiteID == "" {
		return nil, fmt.Errorf("SiteID (network ID) is required for Meraki SSID creation")
	}

	// Find an available SSID slot
	slotNumber, err := s.findAvailableSSIDSlot(ctx, wlan.SiteID)
	if err != nil {
		return nil, fmt.Errorf("failed to find available SSID slot: %w", err)
	}

	logging.Debugf("[meraki] Creating SSID in network %s, slot %s: %s", wlan.SiteID, slotNumber, wlan.SSID)

	// Build the update request
	request := convertVendorWLANToMerakiRequest(wlan)

	// Use the SDK to update the SSID slot
	retryState := NewRetryState(s.retryConfig)
	var updated *meraki.ResponseWirelessUpdateNetworkWirelessSSID

	for {
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return nil, fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		var updateErr error
		if s.suppressOutput {
			restore := suppressStdout()
			updated, _, updateErr = s.dashboard.Wireless.UpdateNetworkWirelessSSID(wlan.SiteID, slotNumber, request)
			restore()
		} else {
			updated, _, updateErr = s.dashboard.Wireless.UpdateNetworkWirelessSSID(wlan.SiteID, slotNumber, request)
		}

		if updateErr == nil {
			break
		}

		if !retryState.ShouldRetry(updateErr) {
			return nil, fmt.Errorf("failed to create SSID: %w", updateErr)
		}

		if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
			return nil, fmt.Errorf("retry wait failed: %w", waitErr)
		}
	}

	return convertMerakiUpdateResponseToWLAN(updated, wlan.SiteID, slotNumber, s.orgID), nil
}

// Update modifies an existing SSID.
func (s *wlansService) Update(ctx context.Context, id string, wlan *vendors.WLAN) (*vendors.WLAN, error) {
	// Parse the composite ID to get network ID and slot number
	networkID, slotNumber, err := parseSSIDID(id)
	if err != nil {
		return nil, err
	}

	logging.Debugf("[meraki] Updating SSID %s in network %s", slotNumber, networkID)

	// Build the update request
	request := convertVendorWLANToMerakiRequest(wlan)

	// Use the SDK to update the SSID
	retryState := NewRetryState(s.retryConfig)
	var updated *meraki.ResponseWirelessUpdateNetworkWirelessSSID

	for {
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return nil, fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		var updateErr error
		if s.suppressOutput {
			restore := suppressStdout()
			updated, _, updateErr = s.dashboard.Wireless.UpdateNetworkWirelessSSID(networkID, slotNumber, request)
			restore()
		} else {
			updated, _, updateErr = s.dashboard.Wireless.UpdateNetworkWirelessSSID(networkID, slotNumber, request)
		}

		if updateErr == nil {
			break
		}

		if !retryState.ShouldRetry(updateErr) {
			return nil, fmt.Errorf("failed to update SSID: %w", updateErr)
		}

		if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
			return nil, fmt.Errorf("retry wait failed: %w", waitErr)
		}
	}

	return convertMerakiUpdateResponseToWLAN(updated, networkID, slotNumber, s.orgID), nil
}

// Delete disables/removes an SSID.
// Meraki has no delete endpoint - we reset the slot to factory defaults.
func (s *wlansService) Delete(ctx context.Context, id string) error {
	// Parse the composite ID to get network ID and slot number
	networkID, slotNumber, err := parseSSIDID(id)
	if err != nil {
		return err
	}

	logging.Debugf("[meraki] Deleting (resetting) SSID %s in network %s", slotNumber, networkID)

	// Build a reset request (disabled, default name, open auth)
	slotNum, _ := strconv.Atoi(slotNumber)
	defaultName := fmt.Sprintf("Unconfigured SSID %d", slotNum+1)
	enabled := false
	authMode := "open"

	request := &meraki.RequestWirelessUpdateNetworkWirelessSSID{
		Name:     defaultName,
		Enabled:  &enabled,
		AuthMode: authMode,
	}

	// Use the SDK to reset the SSID
	retryState := NewRetryState(s.retryConfig)

	for {
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		var updateErr error
		if s.suppressOutput {
			restore := suppressStdout()
			_, _, updateErr = s.dashboard.Wireless.UpdateNetworkWirelessSSID(networkID, slotNumber, request)
			restore()
		} else {
			_, _, updateErr = s.dashboard.Wireless.UpdateNetworkWirelessSSID(networkID, slotNumber, request)
		}

		if updateErr == nil {
			break
		}

		if !retryState.ShouldRetry(updateErr) {
			return fmt.Errorf("failed to delete SSID: %w", updateErr)
		}

		if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
			return fmt.Errorf("retry wait failed: %w", waitErr)
		}
	}

	return nil
}

// findAvailableSSIDSlot finds an unused SSID slot (0-14) in a network.
func (s *wlansService) findAvailableSSIDSlot(ctx context.Context, networkID string) (string, error) {
	retryState := NewRetryState(s.retryConfig)
	var ssids *meraki.ResponseWirelessGetNetworkWirelessSSIDs

	for {
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return "", fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		var fetchErr error
		if s.suppressOutput {
			restore := suppressStdout()
			ssids, _, fetchErr = s.dashboard.Wireless.GetNetworkWirelessSSIDs(networkID)
			restore()
		} else {
			ssids, _, fetchErr = s.dashboard.Wireless.GetNetworkWirelessSSIDs(networkID)
		}

		if fetchErr == nil {
			break
		}

		if !retryState.ShouldRetry(fetchErr) {
			return "", fmt.Errorf("failed to get SSIDs: %w", fetchErr)
		}

		if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
			return "", fmt.Errorf("retry wait failed: %w", waitErr)
		}
	}

	if ssids == nil {
		return "0", nil // If no SSIDs, slot 0 is available
	}

	// Find an available slot (disabled and has default/empty name)
	for _, ssid := range *ssids {
		if ssid.Number == nil {
			continue
		}
		// Check if slot is available (disabled and has default name pattern)
		if ssid.Enabled != nil && !*ssid.Enabled {
			defaultName := fmt.Sprintf("Unconfigured SSID %d", *ssid.Number+1)
			if ssid.Name == "" || ssid.Name == defaultName {
				return strconv.Itoa(*ssid.Number), nil
			}
		}
	}

	return "", fmt.Errorf("no available SSID slots in network %s (all 15 slots are in use)", networkID)
}

// convertVendorWLANToMerakiRequest converts a vendor-agnostic WLAN to Meraki update request.
func convertVendorWLANToMerakiRequest(w *vendors.WLAN) *meraki.RequestWirelessUpdateNetworkWirelessSSID {
	request := &meraki.RequestWirelessUpdateNetworkWirelessSSID{}

	// Basic settings
	if w.SSID != "" {
		request.Name = w.SSID
	}
	request.Enabled = &w.Enabled

	// Visible is the inverse of Hidden
	visible := !w.Hidden
	request.Visible = &visible

	// Auth mode
	if w.AuthType != "" {
		request.AuthMode = w.AuthType
	}

	// PSK for psk auth mode
	if w.PSK != "" {
		request.Psk = w.PSK
	}

	// Encryption mode
	if w.EncryptionMode != "" {
		request.EncryptionMode = w.EncryptionMode
	}

	// Band selection
	if w.Band != "" {
		request.BandSelection = w.Band
	}

	// Default VLAN ID
	if w.VLANID != 0 {
		request.DefaultVLANID = &w.VLANID
	}

	// RADIUS servers for 802.1X
	if len(w.RadiusServers) > 0 {
		var radiusServers []meraki.RequestWirelessUpdateNetworkWirelessSSIDRadiusServers
		for _, rs := range w.RadiusServers {
			server := meraki.RequestWirelessUpdateNetworkWirelessSSIDRadiusServers{
				Host: rs.Host,
			}
			if rs.Port != 0 {
				server.Port = &rs.Port
			}
			if rs.Secret != "" {
				server.Secret = rs.Secret
			}
			radiusServers = append(radiusServers, server)
		}
		request.RadiusServers = &radiusServers
	}

	// Availability tags for per-AP WLAN assignment
	if w.Config != nil {
		if tags := extractStringSlice(w.Config["availabilityTags"]); len(tags) > 0 {
			request.AvailabilityTags = tags
			allAPs := false
			request.AvailableOnAllAps = &allAPs
		}
		if allAPs, ok := w.Config["availableOnAllAps"].(*bool); ok && allAPs != nil {
			request.AvailableOnAllAps = allAPs
		} else if allAPs, ok := w.Config["availableOnAllAps"].(bool); ok {
			request.AvailableOnAllAps = &allAPs
		}
	}

	return request
}

// convertMerakiUpdateResponseToWLAN converts a Meraki update response to vendor-agnostic WLAN.
func convertMerakiUpdateResponseToWLAN(resp *meraki.ResponseWirelessUpdateNetworkWirelessSSID, networkID, slotNumber, orgID string) *vendors.WLAN {
	slotNum, _ := strconv.Atoi(slotNumber)

	wlan := &vendors.WLAN{
		ID:           fmt.Sprintf("%s:%s", networkID, slotNumber),
		OrgID:        orgID,
		SiteID:       networkID,
		SourceVendor: "meraki",
	}

	if resp == nil {
		return wlan
	}

	wlan.SSID = resp.Name

	if resp.Enabled != nil {
		wlan.Enabled = *resp.Enabled
	}

	if resp.Visible != nil {
		wlan.Hidden = !*resp.Visible
	}

	wlan.AuthType = resp.AuthMode
	wlan.EncryptionMode = resp.EncryptionMode
	wlan.Band = resp.BandSelection

	// Convert RADIUS servers
	if resp.RadiusServers != nil {
		for _, rs := range *resp.RadiusServers {
			server := vendors.RadiusServer{
				Host: rs.Host,
			}
			if rs.Port != nil {
				server.Port = *rs.Port
			}
			wlan.RadiusServers = append(wlan.RadiusServers, server)
		}
	}

	// Store config map
	wlan.Config = map[string]interface{}{
		"number":            slotNum,
		"name":              resp.Name,
		"enabled":           resp.Enabled,
		"visible":           resp.Visible,
		"authMode":          resp.AuthMode,
		"encryptionMode":    resp.EncryptionMode,
		"bandSelection":     resp.BandSelection,
		"availabilityTags":  resp.AvailabilityTags,
		"availableOnAllAps": resp.AvailableOnAllAps,
	}

	return wlan
}

// getNetworksWithWireless returns networks that have wireless product type.
func (s *wlansService) getNetworksWithWireless(ctx context.Context) ([]networkInfo, error) {
	params := &meraki.GetOrganizationNetworksQueryParams{
		PerPage: -1,
	}

	retryState := NewRetryState(s.retryConfig)
	var networks *meraki.ResponseOrganizationsGetOrganizationNetworks
	var err error

	for {
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return nil, fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

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
		return []networkInfo{}, nil
	}

	var wirelessNetworks []networkInfo
	for _, n := range *networks {
		if hasWireless(n.ProductTypes) {
			wirelessNetworks = append(wirelessNetworks, networkInfo{
				ID:   n.ID,
				Name: n.Name,
			})
		}
	}

	return wirelessNetworks, nil
}

// getSSIDsForNetwork fetches all SSIDs for a specific network.
func (s *wlansService) getSSIDsForNetwork(ctx context.Context, networkID, networkName string) ([]*vendors.WLAN, error) {
	retryState := NewRetryState(s.retryConfig)
	var ssids *meraki.ResponseWirelessGetNetworkWirelessSSIDs
	var err error

	for {
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return nil, fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		if s.suppressOutput {
			restore := suppressStdout()
			ssids, _, err = s.dashboard.Wireless.GetNetworkWirelessSSIDs(networkID)
			restore()
		} else {
			ssids, _, err = s.dashboard.Wireless.GetNetworkWirelessSSIDs(networkID)
		}

		if err == nil {
			break
		}

		if !retryState.ShouldRetry(err) {
			return nil, fmt.Errorf("failed to get SSIDs for network %s: %w", networkID, err)
		}

		if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
			return nil, fmt.Errorf("retry wait failed: %w", waitErr)
		}
	}

	if ssids == nil {
		return []*vendors.WLAN{}, nil
	}

	result := make([]*vendors.WLAN, 0, len(*ssids))
	for _, ssid := range *ssids {
		// Only include enabled SSIDs with a configured name
		// (Meraki has 15 SSID slots, most are unconfigured)
		if ssid.Enabled != nil && *ssid.Enabled && ssid.Name != "" {
			wlan := convertMerakiSSIDItem(&ssid, networkID, networkName, s.orgID)
			result = append(result, wlan)
		}
	}

	return result, nil
}

// convertMerakiSSIDItem converts a Meraki SSID list item to vendor-agnostic WLAN type.
func convertMerakiSSIDItem(ssid *meraki.ResponseItemWirelessGetNetworkWirelessSSIDs, networkID, networkName, orgID string) *vendors.WLAN {
	wlan := &vendors.WLAN{
		OrgID:        orgID,
		SiteID:       networkID,
		SourceVendor: "meraki",
	}

	// ID is composite: networkId:ssidNumber
	if ssid.Number != nil {
		wlan.ID = fmt.Sprintf("%s:%d", networkID, *ssid.Number)
	}

	wlan.SSID = ssid.Name

	if ssid.Enabled != nil {
		wlan.Enabled = *ssid.Enabled
	}

	// Visible is the inverse of Hidden
	if ssid.Visible != nil {
		wlan.Hidden = !*ssid.Visible
	}

	wlan.AuthType = ssid.AuthMode
	wlan.EncryptionMode = ssid.EncryptionMode
	wlan.Band = ssid.BandSelection

	// Convert RADIUS servers
	if ssid.RadiusServers != nil {
		for _, rs := range *ssid.RadiusServers {
			server := vendors.RadiusServer{
				Host: rs.Host,
			}
			if rs.Port != nil {
				server.Port = *rs.Port
			}
			// Don't copy RADIUS secret for security
			wlan.RadiusServers = append(wlan.RadiusServers, server)
		}
	}

	// Store full config for round-trip accuracy
	wlan.Config = ssidItemToMap(ssid, networkName)

	return wlan
}

// convertMerakiSSID converts a single Meraki SSID to vendor-agnostic WLAN type.
func convertMerakiSSID(ssid *meraki.ResponseWirelessGetNetworkWirelessSSID, networkID, orgID string) *vendors.WLAN {
	wlan := &vendors.WLAN{
		OrgID:        orgID,
		SiteID:       networkID,
		SourceVendor: "meraki",
	}

	if ssid.Number != nil {
		wlan.ID = fmt.Sprintf("%s:%d", networkID, *ssid.Number)
	}

	wlan.SSID = ssid.Name

	if ssid.Enabled != nil {
		wlan.Enabled = *ssid.Enabled
	}

	if ssid.Visible != nil {
		wlan.Hidden = !*ssid.Visible
	}

	wlan.AuthType = ssid.AuthMode
	wlan.EncryptionMode = ssid.EncryptionMode
	wlan.Band = ssid.BandSelection

	// Convert RADIUS servers
	if ssid.RadiusServers != nil {
		for _, rs := range *ssid.RadiusServers {
			server := vendors.RadiusServer{
				Host: rs.Host,
			}
			if rs.Port != nil {
				server.Port = *rs.Port
			}
			wlan.RadiusServers = append(wlan.RadiusServers, server)
		}
	}

	// Store full config
	wlan.Config = ssidToMap(ssid)

	return wlan
}

// ssidItemToMap converts an SSID list item to a config map.
func ssidItemToMap(ssid *meraki.ResponseItemWirelessGetNetworkWirelessSSIDs, networkName string) map[string]interface{} {
	data, err := json.Marshal(ssid)
	if err != nil {
		logging.Warnf("[meraki] Failed to marshal SSID: %v", err)
		return nil
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		logging.Warnf("[meraki] Failed to unmarshal SSID to map: %v", err)
		return nil
	}

	// Add network name for context
	if networkName != "" {
		config["networkName"] = networkName
	}

	return config
}

// ssidToMap converts a single SSID response to a config map.
func ssidToMap(ssid *meraki.ResponseWirelessGetNetworkWirelessSSID) map[string]interface{} {
	data, err := json.Marshal(ssid)
	if err != nil {
		logging.Warnf("[meraki] Failed to marshal SSID: %v", err)
		return nil
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		logging.Warnf("[meraki] Failed to unmarshal SSID to map: %v", err)
		return nil
	}

	return config
}

// parseSSIDID parses a composite SSID ID (networkId:ssidNumber).
func parseSSIDID(id string) (networkID, number string, err error) {
	for i := len(id) - 1; i >= 0; i-- {
		if id[i] == ':' {
			networkID = id[:i]
			number = id[i+1:]
			// Validate number is actually a number
			if _, err := strconv.Atoi(number); err != nil {
				return "", "", fmt.Errorf("invalid SSID ID format: %s (number part '%s' is not numeric)", id, number)
			}
			return networkID, number, nil
		}
	}
	return "", "", fmt.Errorf("invalid SSID ID format: %s (expected networkId:ssidNumber)", id)
}

// extractStringSlice converts an interface{} to []string, handling both
// []string (from Go code) and []any (from JSON unmarshal).
func extractStringSlice(v interface{}) []string {
	switch val := v.(type) {
	case []string:
		return val
	case []any:
		result := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	default:
		return nil
	}
}

// Ensure wlansService implements vendors.WLANsService at compile time.
var _ vendors.WLANsService = (*wlansService)(nil)

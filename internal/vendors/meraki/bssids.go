package meraki

import (
	"context"
	"fmt"

	meraki "github.com/meraki/dashboard-api-go/v5/sdk"

	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// bssidsService implements vendors.BSSIDsService for Meraki.
type bssidsService struct {
	dashboard      *meraki.Client
	orgID          string
	rateLimiter    *RateLimiter
	retryConfig    *RetryConfig
	suppressOutput bool
}

// List retrieves all BSSIDs from the organization using the SSIDs statuses by device endpoint.
func (s *bssidsService) List(ctx context.Context) ([]*vendors.BSSIDEntry, error) {
	logging.Debugf("[meraki] Fetching BSSIDs for org %s", s.orgID)

	params := &meraki.GetOrganizationWirelessSSIDsStatusesByDeviceQueryParams{
		PerPage: 500,
	}

	retryState := NewRetryState(s.retryConfig)
	var resp *meraki.ResponseWirelessGetOrganizationWirelessSSIDsStatusesByDevice
	var err error

	for {
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return nil, fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		if s.suppressOutput {
			restore := suppressStdout()
			resp, _, err = s.dashboard.Wireless.GetOrganizationWirelessSSIDsStatusesByDevice(s.orgID, params)
			restore()
		} else {
			resp, _, err = s.dashboard.Wireless.GetOrganizationWirelessSSIDsStatusesByDevice(s.orgID, params)
		}
		if err == nil {
			break
		}

		if !retryState.ShouldRetry(err) {
			logging.Debugf("[meraki] Failed to get BSSIDs: %v", err)
			return nil, fmt.Errorf("failed to get BSSIDs: %w", err)
		}

		if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
			return nil, fmt.Errorf("retry wait failed: %w", waitErr)
		}
	}

	if resp == nil || resp.Items == nil {
		logging.Debug("[meraki] No BSSID data returned")
		return nil, nil
	}

	var entries []*vendors.BSSIDEntry
	for _, item := range *resp.Items {
		if item.BasicServiceSets == nil {
			continue
		}

		// Extract network/site info
		var networkID, networkName string
		if item.Network != nil {
			networkID = item.Network.ID
			networkName = item.Network.Name
		}

		for _, bss := range *item.BasicServiceSets {
			entry := &vendors.BSSIDEntry{
				BSSID:    normalizeMAC(bss.Bssid),
				APName:   item.Name,
				APSerial: item.Serial,
				SiteID:   networkID,
				SiteName: networkName,
			}

			if bss.SSID != nil {
				entry.SSIDName = bss.SSID.Name
				if bss.SSID.Number != nil {
					entry.SSIDNumber = *bss.SSID.Number
				}
				if bss.SSID.Advertised != nil {
					entry.IsBroadcasting = *bss.SSID.Advertised
				}
			}

			if bss.Radio != nil {
				entry.Band = bss.Radio.Band
				if bss.Radio.Channel != nil {
					entry.Channel = *bss.Radio.Channel
				}
				if bss.Radio.ChannelWidth != nil {
					entry.ChannelWidth = *bss.Radio.ChannelWidth
				}
				if bss.Radio.Power != nil {
					entry.Power = *bss.Radio.Power
				}
				if bss.Radio.IsBroadcasting != nil {
					entry.IsBroadcasting = *bss.Radio.IsBroadcasting
				}
			}

			entries = append(entries, entry)
		}
	}

	logging.Debugf("[meraki] Fetched %d BSSIDs", len(entries))
	return entries, nil
}

var _ vendors.BSSIDsService = (*bssidsService)(nil)

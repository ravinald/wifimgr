package meraki

import (
	"context"
	"fmt"
	"strings"
	"time"

	meraki "github.com/meraki/dashboard-api-go/v5/sdk"

	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// clientDetailService fetches per-client band for a Meraki network by
// calling GetNetworkWirelessClientsConnectionStats once per band (2.4/5/6)
// with a short lookback window. The response for each band filter lists
// the MACs that had connection activity on that band during the window, so
// the union across three calls gives a MAC→band map.
type clientDetailService struct {
	dashboard      *meraki.Client
	orgID          string
	rateLimiter    *RateLimiter
	retryConfig    *RetryConfig
	suppressOutput bool
}

// FetchSiteClientDetail gathers per-client band for every client with
// wireless activity on the site during the last hour. Returns a slice of
// ClientDetail records, one per unique MAC.
func (s *clientDetailService) FetchSiteClientDetail(ctx context.Context, siteID string) ([]*vendors.ClientDetail, error) {
	logging.Debugf("[meraki] fetching per-client band for network %s", siteID)
	fetchedAt := time.Now().UTC()

	// Call order matters: later writes overwrite, so write 2.4 first and
	// higher bands last. A client that roamed mid-window (appearing in
	// multiple bands) ends up labeled with the highest observed band,
	// which is the typical "steered upward" trajectory.
	bandByMAC := make(map[string]string)
	for _, band := range []string{"2.4", "5", "6"} {
		macs, err := s.macsOnBand(ctx, siteID, band)
		if err != nil {
			// Networks without a wireless product (MX-only, switch-only,
			// camera-only, etc.) reject the endpoint. Treat that as "no
			// wireless clients here" instead of a hard failure so bulk
			// iteration across a mixed org doesn't abort on the first
			// appliance-only site.
			if isNonWirelessNetworkError(err) {
				logging.Debugf("[meraki] network %s has no wireless product; skipping", siteID)
				return nil, nil
			}
			return nil, fmt.Errorf("band %s: %w", band, err)
		}
		for _, mac := range macs {
			bandByMAC[vendors.NormalizeMAC(mac)] = band
		}
	}

	records := make([]*vendors.ClientDetail, 0, len(bandByMAC))
	for mac, band := range bandByMAC {
		records = append(records, &vendors.ClientDetail{
			MAC:       mac,
			SiteID:    siteID,
			Band:      band,
			FetchedAt: fetchedAt,
		})
	}
	return records, nil
}

// isNonWirelessNetworkError reports whether err came from Meraki refusing a
// wireless endpoint on a network that has no wireless product (MX-only,
// switch-only, camera-only, etc.). The SDK doesn't expose a typed error for
// this, so we match the error body text. Case-insensitive to survive minor
// wording changes.
func isNonWirelessNetworkError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "this endpoint only supports wireless networks")
}

// macsOnBand calls GetNetworkWirelessClientsConnectionStats with a band
// filter and returns the MACs that had activity on that band during the
// last hour. Short timespan keeps the list focused on current associations
// and keeps the response small.
func (s *clientDetailService) macsOnBand(ctx context.Context, networkID, band string) ([]string, error) {
	params := &meraki.GetNetworkWirelessClientsConnectionStatsQueryParams{
		Timespan: 3600, // one hour of activity
		Band:     band,
	}

	retryState := NewRetryState(s.retryConfig)
	var response *meraki.ResponseWirelessGetNetworkWirelessClientsConnectionStats

	for {
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return nil, fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		var err error
		if s.suppressOutput {
			restore := suppressStdout()
			response, _, err = s.dashboard.Wireless.GetNetworkWirelessClientsConnectionStats(networkID, params)
			restore()
		} else {
			response, _, err = s.dashboard.Wireless.GetNetworkWirelessClientsConnectionStats(networkID, params)
		}
		if err == nil {
			break
		}
		if !retryState.ShouldRetry(err) {
			return nil, fmt.Errorf("connection stats fetch failed: %w", err)
		}
		if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
			return nil, fmt.Errorf("retry wait failed: %w", waitErr)
		}
	}

	if response == nil {
		return nil, nil
	}

	macs := make([]string, 0, len(*response))
	for i := range *response {
		item := &(*response)[i]
		if item.Mac != "" {
			macs = append(macs, item.Mac)
		}
	}
	return macs, nil
}

// Compile-time check that the service satisfies the interface.
var _ vendors.ClientDetailService = (*clientDetailService)(nil)

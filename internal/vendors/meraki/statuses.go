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

// statusesService implements vendors.StatusesService for Meraki.
type statusesService struct {
	dashboard      *meraki.Client
	orgID          string
	rateLimiter    *RateLimiter
	retryConfig    *RetryConfig
	suppressOutput bool
}

// GetAll retrieves the status of all devices in the organization.
// Returns a map of normalized MAC address to DeviceStatus.
func (s *statusesService) GetAll(ctx context.Context) (map[string]*vendors.DeviceStatus, error) {
	logging.Debugf("[meraki] Fetching device statuses for org %s", s.orgID)

	params := &meraki.GetOrganizationDevicesStatusesQueryParams{
		PerPage: -1, // Fetch all
	}

	retryState := NewRetryState(s.retryConfig)
	var statuses *meraki.ResponseOrganizationsGetOrganizationDevicesStatuses
	var err error

	for {
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Acquire(ctx); err != nil {
				return nil, fmt.Errorf("rate limit acquire failed: %w", err)
			}
		}

		if s.suppressOutput {
			restore := suppressStdout()
			statuses, _, err = s.dashboard.Organizations.GetOrganizationDevicesStatuses(s.orgID, params)
			restore()
		} else {
			statuses, _, err = s.dashboard.Organizations.GetOrganizationDevicesStatuses(s.orgID, params)
		}
		if err == nil {
			break
		}

		if !retryState.ShouldRetry(err) {
			logging.Debugf("[meraki] Failed to get device statuses: %v", err)
			return nil, fmt.Errorf("failed to get device statuses: %w", err)
		}

		if waitErr := retryState.WaitBeforeRetry(ctx, nil); waitErr != nil {
			return nil, fmt.Errorf("retry wait failed: %w", waitErr)
		}
	}

	if statuses == nil {
		logging.Debug("[meraki] No device statuses returned")
		return make(map[string]*vendors.DeviceStatus), nil
	}

	result := make(map[string]*vendors.DeviceStatus, len(*statuses))
	for i := range *statuses {
		status := &(*statuses)[i]
		if status.Mac == "" {
			continue
		}

		// Normalize MAC for use as key
		normalizedMAC := normalizeMAC(status.Mac)

		// Parse lastReportedAt timestamp
		var lastReported time.Time
		if status.LastReportedAt != "" {
			// Meraki uses ISO 8601 format
			parsed, parseErr := time.Parse(time.RFC3339, status.LastReportedAt)
			if parseErr != nil {
				logging.Debugf("[meraki] Failed to parse lastReportedAt for %s: %v", normalizedMAC, parseErr)
			} else {
				lastReported = parsed
			}
		}

		result[normalizedMAC] = &vendors.DeviceStatus{
			Status:         normalizeStatus(status.Status),
			LastReportedAt: lastReported,
			IP:             status.LanIP,
			PublicIP:       status.PublicIP,
		}
	}

	logging.Debugf("[meraki] Fetched status for %d devices", len(result))
	return result, nil
}

// normalizeStatus converts Meraki status values to normalized values.
// Meraki uses: online, offline, alerting, dormant
// We keep these as-is since they are already good normalized values.
func normalizeStatus(status string) string {
	s := strings.ToLower(status)
	switch s {
	case "online", "offline", "alerting", "dormant":
		return s
	default:
		// Unknown status, return as-is but lowercase
		return s
	}
}

// Ensure statusesService implements vendors.StatusesService at compile time.
var _ vendors.StatusesService = (*statusesService)(nil)

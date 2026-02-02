package mist

import (
	"context"
	"fmt"

	"github.com/ravinald/wifimgr/api"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// statusesService implements vendors.StatusesService for Mist.
type statusesService struct {
	client api.Client
	orgID  string
}

// GetAll retrieves the status of all devices in the organization.
// Returns a map of normalized MAC address to DeviceStatus.
// For Mist, status is derived from the Connected field of inventory items.
func (s *statusesService) GetAll(ctx context.Context) (map[string]*vendors.DeviceStatus, error) {
	result := make(map[string]*vendors.DeviceStatus)

	// Fetch all device types
	for _, deviceType := range []string{"ap", "switch", "gateway"} {
		items, err := s.client.GetInventory(ctx, s.orgID, deviceType)
		if err != nil {
			return nil, fmt.Errorf("failed to get %s inventory for status: %w", deviceType, err)
		}

		for _, item := range items {
			if item.MAC == nil || *item.MAC == "" {
				continue
			}

			normalizedMAC := normalizeMAC(*item.MAC)

			// Map Connected bool to status string
			status := "offline"
			if item.Connected != nil && *item.Connected {
				status = "online"
			}

			result[normalizedMAC] = &vendors.DeviceStatus{
				Status: status,
				// LastReportedAt not available from inventory endpoint
				// IP not available from inventory endpoint
			}
		}
	}

	return result, nil
}

// Ensure statusesService implements vendors.StatusesService at compile time.
var _ vendors.StatusesService = (*statusesService)(nil)

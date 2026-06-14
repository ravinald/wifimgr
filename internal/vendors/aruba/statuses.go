package aruba

import (
	"context"

	"github.com/ravinald/wifimgr/internal/vendors"
)

type statusesService struct {
	client *Client
}

// GetAll returns per-AP status from `show aps`, keyed by normalized MAC (or
// serial when the table omits a MAC column). The IP a member reports is taken
// as evidence it is up.
func (s *statusesService) GetAll(ctx context.Context) (map[string]*vendors.DeviceStatus, error) {
	aps, err := collectAPs(ctx, s.client)
	if err != nil {
		return nil, err
	}

	statuses := make(map[string]*vendors.DeviceStatus)
	for _, ap := range aps {
		key := ap.MAC
		if key == "" {
			key = ap.Serial
		}
		if key == "" {
			continue
		}
		statuses[key] = &vendors.DeviceStatus{
			Status: deviceStatusVocab(ap.Status),
			IP:     ap.IP,
		}
	}
	return statuses, nil
}

// deviceStatusVocab maps the connected/disconnected vocabulary used on
// DeviceInfo to the online/offline vocabulary DeviceStatus carries, which is
// what the table formatter's status symbols (online→C, offline→D) expect.
func deviceStatusVocab(status string) string {
	switch status {
	case "connected":
		return "online"
	case "disconnected":
		return "offline"
	default:
		return status
	}
}

var _ vendors.StatusesService = (*statusesService)(nil)

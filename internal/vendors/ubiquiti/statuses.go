package ubiquiti

import (
	"context"

	"github.com/ravinald/wifimgr/internal/vendors"
)

type statusesService struct {
	client *Client
}

func (s *statusesService) GetAll(ctx context.Context) (map[string]*vendors.DeviceStatus, error) {
	groups, err := s.client.GetDevices(ctx)
	if err != nil {
		return nil, err
	}

	statuses := make(map[string]*vendors.DeviceStatus)
	for _, group := range groups {
		for _, device := range group.Devices {
			if !IsNetworkDevice(device) {
				continue
			}
			mac := normalizeMAC(device.MAC)
			statuses[mac] = &vendors.DeviceStatus{
				Status: normalizeStatus(device.Status),
				IP:     device.IP,
			}
		}
	}
	return statuses, nil
}

var _ vendors.StatusesService = (*statusesService)(nil)

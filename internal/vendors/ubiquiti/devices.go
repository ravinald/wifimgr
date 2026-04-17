package ubiquiti

import (
	"context"

	"github.com/ravinald/wifimgr/internal/vendors"
)

type devicesService struct {
	client *Client
}

func (s *devicesService) List(ctx context.Context, siteID, deviceType string) ([]*vendors.DeviceInfo, error) {
	groups, err := s.client.GetDevices(ctx)
	if err != nil {
		return nil, err
	}

	sites, err := s.client.GetSites(ctx)
	if err != nil {
		return nil, err
	}

	hostSiteMap := buildHostSiteMap(sites)
	flatDevices := flattenDevices(groups, hostSiteMap)

	var result []*vendors.DeviceInfo
	for _, d := range flatDevices {
		info := convertFlatDeviceToDeviceInfo(d)
		if siteID != "" && info.SiteID != siteID {
			continue
		}
		if deviceType != "" && info.Type != deviceType {
			continue
		}
		result = append(result, info)
	}
	return result, nil
}

func (s *devicesService) Get(ctx context.Context, siteID, deviceID string) (*vendors.DeviceInfo, error) {
	devices, err := s.List(ctx, siteID, "")
	if err != nil {
		return nil, err
	}
	for _, d := range devices {
		if d.ID == deviceID {
			return d, nil
		}
	}
	return nil, &vendors.DeviceNotFoundError{Identifier: deviceID}
}

func (s *devicesService) ByMAC(ctx context.Context, mac string) (*vendors.DeviceInfo, error) {
	devices, err := s.List(ctx, "", "")
	if err != nil {
		return nil, err
	}
	normalized := normalizeMAC(mac)
	for _, d := range devices {
		if d.MAC == normalized {
			return d, nil
		}
	}
	return nil, &vendors.DeviceNotFoundError{Identifier: mac}
}

func (s *devicesService) Update(_ context.Context, _, _ string, _ *vendors.DeviceInfo) (*vendors.DeviceInfo, error) {
	return nil, &vendors.CapabilityNotSupportedError{
		Capability: "device update",
		VendorName: "ubiquiti",
	}
}

func (s *devicesService) Rename(_ context.Context, _, _, _ string) error {
	return &vendors.CapabilityNotSupportedError{
		Capability: "device rename",
		VendorName: "ubiquiti",
	}
}

func (s *devicesService) UpdateConfig(_ context.Context, _, _ string, _ map[string]any) error {
	return &vendors.CapabilityNotSupportedError{
		Capability: "device config update",
		VendorName: "ubiquiti",
	}
}

var _ vendors.DevicesService = (*devicesService)(nil)

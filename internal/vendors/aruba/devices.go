package aruba

import (
	"context"

	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/vendors"
)

type devicesService struct {
	client *Client
	siteID string
}

func (s *devicesService) List(ctx context.Context, siteID, deviceType string) ([]*vendors.DeviceInfo, error) {
	if deviceType != "" && deviceType != "ap" {
		return nil, nil
	}
	aps, err := collectAPs(ctx, s.client)
	if err != nil {
		return nil, err
	}
	var devices []*vendors.DeviceInfo
	for _, ap := range aps {
		d := deviceInfoFromAP(ap, s.siteID, "")
		if siteID != "" && d.SiteID != siteID {
			continue
		}
		devices = append(devices, d)
	}
	return devices, nil
}

func (s *devicesService) Get(ctx context.Context, siteID, deviceID string) (*vendors.DeviceInfo, error) {
	devices, err := s.List(ctx, siteID, "ap")
	if err != nil {
		return nil, err
	}
	for _, d := range devices {
		if d.ID == deviceID {
			return d, nil
		}
	}
	return nil, &vendors.DeviceNotFoundError{Identifier: deviceID, APILabel: vendorName}
}

func (s *devicesService) ByMAC(ctx context.Context, mac string) (*vendors.DeviceInfo, error) {
	devices, err := s.List(ctx, "", "ap")
	if err != nil {
		return nil, err
	}
	want := normalizeMAC(mac)
	for _, d := range devices {
		if d.MAC == want {
			return d, nil
		}
	}
	return nil, &vendors.DeviceNotFoundError{Identifier: mac, APILabel: vendorName}
}

func (s *devicesService) Update(_ context.Context, _, _ string, _ *vendors.DeviceInfo) (*vendors.DeviceInfo, error) {
	return nil, &vendors.CapabilityNotSupportedError{Capability: "device update", APILabel: vendorName, VendorName: vendorName}
}

// Rename sets the AP hostname via the Action API.
func (s *devicesService) Rename(ctx context.Context, _, _, newName string) error {
	return s.client.PostObject(ctx, "hostname", hostnamePayload(s.client.host, newName))
}

// UpdateConfig translates the managed-key config map into Action/Configuration
// API calls. The first cut handles the AP name; other keys are logged and
// skipped so a partial intent doesn't fail the whole apply. Radio/RF push lands
// once the live `show running-config` field names are confirmed.
func (s *devicesService) UpdateConfig(ctx context.Context, _, _ string, config map[string]any) error {
	for key, val := range config {
		switch key {
		case "name", "hostname":
			if name, ok := val.(string); ok && name != "" {
				if err := s.client.PostObject(ctx, "hostname", hostnamePayload(s.client.host, name)); err != nil {
					return err
				}
			}
		default:
			logging.Debugf("[aruba] UpdateConfig: skipping unmapped key %q", key)
		}
	}
	return nil
}

// Reboot restarts the AP via the Action API.
func (s *devicesService) Reboot(ctx context.Context, _, _ string) error {
	return s.client.PostObject(ctx, "reboot", rebootPayload(s.client.host))
}

var _ vendors.DevicesService = (*devicesService)(nil)

package ubiquiti

import (
	"context"

	"github.com/ravinald/wifimgr/internal/vendors"
)

type inventoryService struct {
	client *Client
}

func (s *inventoryService) List(ctx context.Context, deviceType string) ([]*vendors.InventoryItem, error) {
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

	var result []*vendors.InventoryItem
	for _, d := range flatDevices {
		item := convertFlatDeviceToInventoryItem(d)
		if deviceType != "" && item.Type != deviceType {
			continue
		}
		result = append(result, item)
	}
	return result, nil
}

func (s *inventoryService) ByMAC(ctx context.Context, mac string) (*vendors.InventoryItem, error) {
	items, err := s.List(ctx, "")
	if err != nil {
		return nil, err
	}
	normalized := normalizeMAC(mac)
	for _, item := range items {
		if item.MAC == normalized {
			return item, nil
		}
	}
	return nil, &vendors.DeviceNotFoundError{Identifier: mac}
}

func (s *inventoryService) BySerial(ctx context.Context, serial string) (*vendors.InventoryItem, error) {
	items, err := s.List(ctx, "")
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item.Serial == serial {
			return item, nil
		}
	}
	return nil, &vendors.DeviceNotFoundError{Identifier: serial}
}

func (s *inventoryService) Claim(_ context.Context, _ []string) ([]*vendors.InventoryItem, error) {
	return nil, &vendors.CapabilityNotSupportedError{
		Capability: "device claiming",
		VendorName: "ubiquiti",
	}
}

func (s *inventoryService) Release(_ context.Context, _ []string) error {
	return &vendors.CapabilityNotSupportedError{
		Capability: "device release",
		VendorName: "ubiquiti",
	}
}

func (s *inventoryService) AssignToSite(_ context.Context, _ string, _ []string) error {
	return &vendors.CapabilityNotSupportedError{
		Capability: "device site assignment",
		VendorName: "ubiquiti",
	}
}

func (s *inventoryService) UnassignFromSite(_ context.Context, _ []string) error {
	return &vendors.CapabilityNotSupportedError{
		Capability: "device site unassignment",
		VendorName: "ubiquiti",
	}
}

var _ vendors.InventoryService = (*inventoryService)(nil)

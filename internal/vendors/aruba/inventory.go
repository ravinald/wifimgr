package aruba

import (
	"context"

	"github.com/ravinald/wifimgr/internal/vendors"
)

type inventoryService struct {
	client *Client
	siteID string
}

// List returns the APs in the swarm from `show aps`. Instant has no switches or
// gateways, so any non-AP device type yields an empty list.
func (s *inventoryService) List(ctx context.Context, deviceType string) ([]*vendors.InventoryItem, error) {
	if deviceType != "" && deviceType != "ap" {
		return nil, nil
	}
	aps, err := collectAPs(ctx, s.client)
	if err != nil {
		return nil, err
	}
	var items []*vendors.InventoryItem
	for _, ap := range aps {
		items = append(items, inventoryItemFromAP(ap, s.siteID, ""))
	}
	return items, nil
}

func (s *inventoryService) ByMAC(ctx context.Context, mac string) (*vendors.InventoryItem, error) {
	items, err := s.List(ctx, "ap")
	if err != nil {
		return nil, err
	}
	want := normalizeMAC(mac)
	for _, it := range items {
		if it.MAC == want {
			return it, nil
		}
	}
	return nil, &vendors.DeviceNotFoundError{Identifier: mac, APILabel: vendorName}
}

func (s *inventoryService) BySerial(ctx context.Context, serial string) (*vendors.InventoryItem, error) {
	items, err := s.List(ctx, "ap")
	if err != nil {
		return nil, err
	}
	for _, it := range items {
		if it.Serial == serial {
			return it, nil
		}
	}
	return nil, &vendors.DeviceNotFoundError{Identifier: serial, APILabel: vendorName}
}

// Instant has no cloud inventory to claim against; membership is physical.
func (s *inventoryService) Claim(_ context.Context, _ []string) ([]*vendors.InventoryItem, error) {
	return nil, &vendors.CapabilityNotSupportedError{Capability: "inventory claim", APILabel: vendorName, VendorName: vendorName}
}

func (s *inventoryService) Release(_ context.Context, _ []string) error {
	return &vendors.CapabilityNotSupportedError{Capability: "inventory release", APILabel: vendorName, VendorName: vendorName}
}

func (s *inventoryService) AssignToSite(_ context.Context, _ string, _ []string) error {
	return &vendors.CapabilityNotSupportedError{Capability: "site assignment", APILabel: vendorName, VendorName: vendorName}
}

func (s *inventoryService) UnassignFromSite(_ context.Context, _ []string) error {
	return &vendors.CapabilityNotSupportedError{Capability: "site unassignment", APILabel: vendorName, VendorName: vendorName}
}

var _ vendors.InventoryService = (*inventoryService)(nil)

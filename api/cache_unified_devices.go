package api

import (
	"fmt"
	"strings"
)

// GetDevice retrieves a device by MAC address
func (uc *UnifiedCache) GetDevice(mac string) (*UnifiedDevice, error) {
	orgData, err := uc.GetOrgData(uc.orgID)
	if err != nil {
		return nil, err
	}

	// Normalize MAC address
	mac = strings.ToLower(strings.ReplaceAll(mac, ":", ""))

	// Check APs
	for _, apVal := range orgData.Inventory.AP {
		ap := &apVal
		if ap.MAC != nil && strings.ToLower(strings.ReplaceAll(*ap.MAC, ":", "")) == mac {
			unified := &UnifiedDevice{
				BaseDevice: BaseDevice{
					ID:     ap.ID,
					MAC:    ap.MAC,
					Serial: ap.Serial,
					Name:   ap.Name,
					Model:  ap.Model,
					Type:   ap.Type,
					SiteID: ap.SiteID,
					OrgID:  ap.OrgID,
				},
				DeviceType:   "ap",
				DeviceConfig: ap.ToMap(),
			}
			return unified, nil
		}
	}

	// Check Switches
	for _, swVal := range orgData.Inventory.Switch {
		sw := &swVal
		if sw.MAC != nil && strings.ToLower(strings.ReplaceAll(*sw.MAC, ":", "")) == mac {
			unified := &UnifiedDevice{
				BaseDevice: BaseDevice{
					ID:     sw.ID,
					MAC:    sw.MAC,
					Serial: sw.Serial,
					Name:   sw.Name,
					Model:  sw.Model,
					Type:   sw.Type,
					SiteID: sw.SiteID,
					OrgID:  sw.OrgID,
				},
				DeviceType:   "switch",
				DeviceConfig: sw.ToMap(),
			}
			return unified, nil
		}
	}

	// Check Gateways
	for _, gwVal := range orgData.Inventory.Gateway {
		gw := &gwVal
		if gw.MAC != nil && strings.ToLower(strings.ReplaceAll(*gw.MAC, ":", "")) == mac {
			unified := &UnifiedDevice{
				BaseDevice: BaseDevice{
					ID:     gw.ID,
					MAC:    gw.MAC,
					Serial: gw.Serial,
					Name:   gw.Name,
					Model:  gw.Model,
					Type:   gw.Type,
					SiteID: gw.SiteID,
					OrgID:  gw.OrgID,
				},
				DeviceType:   "gateway",
				DeviceConfig: gw.ToMap(),
			}
			return unified, nil
		}
	}

	return nil, fmt.Errorf("device not found: %s", mac)
}

// GetDevicesByType retrieves devices by site and type
func (uc *UnifiedCache) GetDevicesByType(siteID, deviceType string) ([]UnifiedDevice, error) {
	orgData, err := uc.GetOrgData(uc.orgID)
	if err != nil {
		return nil, err
	}

	var devices []UnifiedDevice

	switch deviceType {
	case "ap":
		for _, apVal := range orgData.Inventory.AP {
			ap := &apVal
			if siteID == "" || (ap.SiteID != nil && *ap.SiteID == siteID) {
				unified := UnifiedDevice{
					BaseDevice: BaseDevice{
						ID:     ap.ID,
						MAC:    ap.MAC,
						Serial: ap.Serial,
						Name:   ap.Name,
						Model:  ap.Model,
						Type:   ap.Type,
						SiteID: ap.SiteID,
						OrgID:  ap.OrgID,
					},
					DeviceType:   "ap",
					DeviceConfig: ap.ToMap(),
				}
				devices = append(devices, unified)
			}
		}
	case "switch":
		for _, swVal := range orgData.Inventory.Switch {
			sw := &swVal
			if siteID == "" || (sw.SiteID != nil && *sw.SiteID == siteID) {
				unified := UnifiedDevice{
					BaseDevice: BaseDevice{
						ID:     sw.ID,
						MAC:    sw.MAC,
						Serial: sw.Serial,
						Name:   sw.Name,
						Model:  sw.Model,
						Type:   sw.Type,
						SiteID: sw.SiteID,
						OrgID:  sw.OrgID,
					},
					DeviceType:   "switch",
					DeviceConfig: sw.ToMap(),
				}
				devices = append(devices, unified)
			}
		}
	case "gateway":
		for _, gwVal := range orgData.Inventory.Gateway {
			gw := &gwVal
			if siteID == "" || (gw.SiteID != nil && *gw.SiteID == siteID) {
				unified := UnifiedDevice{
					BaseDevice: BaseDevice{
						ID:     gw.ID,
						MAC:    gw.MAC,
						Serial: gw.Serial,
						Name:   gw.Name,
						Model:  gw.Model,
						Type:   gw.Type,
						SiteID: gw.SiteID,
						OrgID:  gw.OrgID,
					},
					DeviceType:   "gateway",
					DeviceConfig: gw.ToMap(),
				}
				devices = append(devices, unified)
			}
		}
	default:
		return nil, fmt.Errorf("unknown device type: %s", deviceType)
	}

	return devices, nil
}

// UpdateDevice updates or adds a device
func (uc *UnifiedCache) UpdateDevice(device *UnifiedDevice) error {
	orgData, err := uc.GetOrgData(uc.orgID)
	if err != nil {
		return err
	}

	switch device.DeviceType {
	case "ap":
		ap := &APDevice{}
		if err := ap.FromMap(device.DeviceConfig); err != nil {
			return fmt.Errorf("failed to convert to AP device: %w", err)
		}
		// Ensure base fields are set
		ap.ID = device.ID
		ap.MAC = device.MAC
		ap.Serial = device.Serial
		ap.Name = device.Name
		ap.Model = device.Model
		ap.Type = device.Type
		ap.SiteID = device.SiteID
		ap.OrgID = device.OrgID

		// Initialize map if needed
		if orgData.Inventory.AP == nil {
			orgData.Inventory.AP = make(map[string]APDevice)
		}
		// Update or add AP using MAC as key
		if ap.MAC != nil {
			normalizedMAC := strings.ToLower(strings.ReplaceAll(*ap.MAC, ":", ""))
			orgData.Inventory.AP[normalizedMAC] = *ap
			uc.dirty = true
		}

	case "switch":
		sw := &MistSwitchDevice{}
		if err := sw.FromMap(device.DeviceConfig); err != nil {
			return fmt.Errorf("failed to convert to switch device: %w", err)
		}
		// Ensure base fields are set
		sw.ID = device.ID
		sw.MAC = device.MAC
		sw.Serial = device.Serial
		sw.Name = device.Name
		sw.Model = device.Model
		sw.Type = device.Type
		sw.SiteID = device.SiteID
		sw.OrgID = device.OrgID

		// Initialize map if needed
		if orgData.Inventory.Switch == nil {
			orgData.Inventory.Switch = make(map[string]MistSwitchDevice)
		}
		// Update or add switch using MAC as key
		if sw.MAC != nil {
			normalizedMAC := strings.ToLower(strings.ReplaceAll(*sw.MAC, ":", ""))
			orgData.Inventory.Switch[normalizedMAC] = *sw
			uc.dirty = true
		}

	case "gateway":
		gw := &MistGatewayDevice{}
		if err := gw.FromMap(device.DeviceConfig); err != nil {
			return fmt.Errorf("failed to convert to gateway device: %w", err)
		}
		// Ensure base fields are set
		gw.ID = device.ID
		gw.MAC = device.MAC
		gw.Serial = device.Serial
		gw.Name = device.Name
		gw.Model = device.Model
		gw.Type = device.Type
		gw.SiteID = device.SiteID
		gw.OrgID = device.OrgID

		// Initialize map if needed
		if orgData.Inventory.Gateway == nil {
			orgData.Inventory.Gateway = make(map[string]MistGatewayDevice)
		}
		// Update or add gateway using MAC as key
		if gw.MAC != nil {
			normalizedMAC := strings.ToLower(strings.ReplaceAll(*gw.MAC, ":", ""))
			orgData.Inventory.Gateway[normalizedMAC] = *gw
			uc.dirty = true
		}

	default:
		return fmt.Errorf("unknown device type: %s", device.DeviceType)
	}

	return nil
}

// GetInventory retrieves inventory items by type
func (uc *UnifiedCache) GetInventory(deviceType string) ([]any, error) {
	orgData, err := uc.GetOrgData(uc.orgID)
	if err != nil {
		return nil, err
	}

	var items []any

	switch deviceType {
	case "ap":
		for _, ap := range orgData.Inventory.AP {
			apCopy := ap
			items = append(items, &apCopy)
		}
	case "switch":
		for _, sw := range orgData.Inventory.Switch {
			swCopy := sw
			items = append(items, &swCopy)
		}
	case "gateway":
		for _, gw := range orgData.Inventory.Gateway {
			gwCopy := gw
			items = append(items, &gwCopy)
		}
	default:
		return nil, fmt.Errorf("unknown device type: %s", deviceType)
	}

	return items, nil
}

// UpdateInventory updates inventory items
func (uc *UnifiedCache) UpdateInventory(deviceType string, items []any) error {
	orgData, err := uc.GetOrgData(uc.orgID)
	if err != nil {
		return err
	}

	switch deviceType {
	case "ap":
		orgData.Inventory.AP = make(map[string]APDevice)
		for _, item := range items {
			if ap, ok := item.(*APDevice); ok {
				if ap.MAC != nil {
					normalizedMAC := strings.ToLower(strings.ReplaceAll(*ap.MAC, ":", ""))
					orgData.Inventory.AP[normalizedMAC] = *ap
				}
			}
		}
	case "switch":
		orgData.Inventory.Switch = make(map[string]MistSwitchDevice)
		for _, item := range items {
			if sw, ok := item.(*MistSwitchDevice); ok {
				if sw.MAC != nil {
					normalizedMAC := strings.ToLower(strings.ReplaceAll(*sw.MAC, ":", ""))
					orgData.Inventory.Switch[normalizedMAC] = *sw
				}
			}
		}
	case "gateway":
		orgData.Inventory.Gateway = make(map[string]MistGatewayDevice)
		for _, item := range items {
			if gw, ok := item.(*MistGatewayDevice); ok {
				if gw.MAC != nil {
					normalizedMAC := strings.ToLower(strings.ReplaceAll(*gw.MAC, ":", ""))
					orgData.Inventory.Gateway[normalizedMAC] = *gw
				}
			}
		}
	default:
		return fmt.Errorf("unknown device type: %s", deviceType)
	}

	uc.dirty = true
	return nil
}

package api

import (
	"context"
	"fmt"

	"github.com/ravinald/wifimgr/internal/macaddr"
)

// Additional new bidirectional device methods

// GetDeviceByID retrieves a device by ID using the new bidirectional pattern
func (c *mistClient) GetDeviceByID(ctx context.Context, siteID, deviceID string) (*UnifiedDevice, error) {
	c.logDebug("Getting device %s from site %s using new bidirectional pattern", deviceID, siteID)

	// Get raw device data from API
	var rawDevice map[string]interface{}
	path := fmt.Sprintf("/sites/%s/devices/%s", siteID, deviceID)
	err := c.do(ctx, "GET", path, nil, &rawDevice)
	if err != nil {
		return nil, fmt.Errorf("failed to get device by ID: %w", err)
	}

	c.logDebug("Retrieved raw device data with %d fields", len(rawDevice))

	// Convert to UnifiedDevice using bidirectional pattern
	device, err := NewUnifiedDeviceFromMap(rawDevice)
	if err != nil {
		return nil, fmt.Errorf("failed to convert device data: %w", err)
	}

	return device, nil
}

// GetDeviceByName retrieves a device by name using the new bidirectional pattern
func (c *mistClient) GetDeviceByName(ctx context.Context, siteID, name string) (*UnifiedDevice, error) {
	c.logDebug("Getting device with name '%s' from site %s using new bidirectional pattern", name, siteID)

	// Get all devices for the site
	devices, err := c.GetDevices(ctx, siteID, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get devices for site: %w", err)
	}

	// Find device by name
	for _, device := range devices {
		if device.Name != nil && *device.Name == name {
			c.logDebug("Found device with name '%s'", name)
			return &device, nil
		}
	}

	return nil, fmt.Errorf("device with name '%s' not found in site %s", name, siteID)
}

// GetDevicesByType retrieves devices of a specific type for a site using the new bidirectional pattern
func (c *mistClient) GetDevicesByType(ctx context.Context, siteID string, deviceType string) ([]UnifiedDevice, error) {
	// This is just an alias for GetDevices with better naming
	return c.GetDevices(ctx, siteID, deviceType)
}

// UnassignDevice unassigns a device from a site using the new bidirectional pattern
func (c *mistClient) UnassignDevice(ctx context.Context, orgID string, siteID string, deviceID string) error {
	c.logDebug("Unassigning device %s from site %s in org %s using new bidirectional pattern", deviceID, siteID, orgID)

	// First get the device to get its MAC address
	device, err := c.GetDeviceByID(ctx, siteID, deviceID)
	if err != nil {
		return fmt.Errorf("failed to get device for unassignment: %w", err)
	}

	if device.MAC == nil {
		return fmt.Errorf("device %s has no MAC address for unassignment", deviceID)
	}

	normalizedMAC, err := macaddr.Normalize(*device.MAC)
	if err != nil {
		return fmt.Errorf("invalid MAC address %s: %w", *device.MAC, err)
	}

	// Make the API request to unassign
	path := fmt.Sprintf("/orgs/%s/inventory/%s/unassign", orgID, normalizedMAC)
	err = c.do(ctx, "POST", path, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to unassign device: %w", err)
	}

	c.logDebug("Device unassignment successful")
	return nil
}

// AssignDevicesToSite assigns multiple devices to a site using the new bidirectional pattern
func (c *mistClient) AssignDevicesToSite(ctx context.Context, orgID string, siteID string, macs []string, noReassign bool) error {
	c.logDebug("Assigning %d devices to site %s in org %s using new bidirectional pattern", len(macs), siteID, orgID)

	// Normalize all MAC addresses
	normalizedMACs := make([]string, len(macs))
	for i, mac := range macs {
		normalizedMAC, err := macaddr.Normalize(mac)
		if err != nil {
			return fmt.Errorf("invalid MAC address %s: %w", mac, err)
		}
		normalizedMACs[i] = normalizedMAC
	}

	// Prepare assignment request
	assignmentData := map[string]interface{}{
		"site_id":     siteID,
		"macs":        normalizedMACs,
		"no_reassign": noReassign,
	}

	// Make the API request
	path := fmt.Sprintf("/orgs/%s/inventory/assign", orgID)
	err := c.do(ctx, "POST", path, assignmentData, nil)
	if err != nil {
		return fmt.Errorf("failed to assign devices to site: %w", err)
	}

	c.logDebug("Bulk device assignment successful")
	return nil
}

// UnassignDevicesFromSite unassigns multiple devices from their sites using the new bidirectional pattern
func (c *mistClient) UnassignDevicesFromSite(ctx context.Context, orgID string, macs []string) error {
	c.logDebug("Unassigning %d devices from their sites in org %s using new bidirectional pattern", len(macs), orgID)

	// Normalize all MAC addresses
	normalizedMACs := make([]string, len(macs))
	for i, mac := range macs {
		normalizedMAC, err := macaddr.Normalize(mac)
		if err != nil {
			return fmt.Errorf("invalid MAC address %s: %w", mac, err)
		}
		normalizedMACs[i] = normalizedMAC
	}

	// Prepare unassignment request
	unassignmentData := map[string]interface{}{
		"macs": normalizedMACs,
	}

	// Make the API request
	path := fmt.Sprintf("/orgs/%s/inventory/unassign", orgID)
	err := c.do(ctx, "POST", path, unassignmentData, nil)
	if err != nil {
		return fmt.Errorf("failed to unassign devices from sites: %w", err)
	}

	c.logDebug("Bulk device unassignment successful")
	return nil
}

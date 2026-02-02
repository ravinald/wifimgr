package netbox

import (
	"context"
	"fmt"
	"strings"

	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/macaddr"
)

// DeviceMetadata contains basic device information from NetBox
type DeviceMetadata struct {
	MAC      string
	Name     string
	SiteID   string
	SiteName string
	Model    string
	Serial   string
}

// Syncer handles reverse synchronization from NetBox to wifimgr runtime cache
type Syncer struct {
	client *Client
	config *Config
}

// NewSyncer creates a new Syncer instance
func NewSyncer(config *Config) (*Syncer, error) {
	client, err := NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create NetBox client: %w", err)
	}

	return &Syncer{
		client: client,
		config: config,
	}, nil
}

// SyncFromNetBox queries NetBox for AP devices and returns device metadata.
// The returned map uses normalized MAC addresses as keys.
//
// Parameters:
//   - ctx: Context for the operation
//   - siteName: Site name to filter by (empty string for all sites)
//
// Returns:
//   - Map of MAC address -> DeviceMetadata
//   - Error if the sync operation fails
func (s *Syncer) SyncFromNetBox(ctx context.Context, siteName string) (map[string]*DeviceMetadata, error) {
	// Get site slug if site name provided
	siteSlug := ""
	if siteName != "" {
		siteSlug = s.config.GetSiteSlug(siteName)
		logging.Debugf("Syncing devices from NetBox for site '%s' (slug: %s)", siteName, siteSlug)
	} else {
		logging.Debug("Syncing all AP devices from NetBox")
	}

	// Get role slug for APs
	roleSlug := s.config.GetDeviceRoleSlug("ap")

	// Query NetBox for devices
	devices, err := s.client.GetDevicesBySiteAndRole(ctx, siteSlug, roleSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to query NetBox devices: %w", err)
	}

	logging.Infof("Retrieved %d AP devices from NetBox", len(devices))

	// Build metadata map
	metadata := make(map[string]*DeviceMetadata)

	for _, device := range devices {
		// Get primary MAC address from device interfaces
		mac, err := s.getDeviceMACAddress(ctx, device.ID)
		if err != nil {
			logging.Warnf("Failed to get MAC address for device %s: %v", device.Name, err)
			continue
		}
		if mac == "" {
			logging.Warnf("Device %s has no MAC address, skipping", device.Name)
			continue
		}

		// Normalize MAC address
		normalizedMAC := macaddr.NormalizeOrEmpty(mac)
		if normalizedMAC == "" {
			logging.Warnf("Invalid MAC address '%s' for device %s, skipping", mac, device.Name)
			continue
		}

		// Extract site information
		siteName := ""
		siteID := ""
		if device.Site != nil {
			siteName = device.Site.Name
			siteID = fmt.Sprintf("%d", device.Site.ID)
		}

		// Extract device type information
		model := ""
		if device.DeviceType != nil {
			model = device.DeviceType.Model
		}

		metadata[normalizedMAC] = &DeviceMetadata{
			MAC:      normalizedMAC,
			Name:     device.Name,
			SiteID:   siteID,
			SiteName: siteName,
			Model:    model,
			Serial:   device.Serial,
		}

		logging.Debugf("Synced device: %s (MAC: %s, Site: %s)", device.Name, normalizedMAC, siteName)
	}

	return metadata, nil
}

// getDeviceMACAddress retrieves the primary MAC address for a device by querying its interfaces
func (s *Syncer) getDeviceMACAddress(ctx context.Context, deviceID int64) (string, error) {
	interfaces, err := s.client.GetInterfacesByDevice(ctx, deviceID)
	if err != nil {
		return "", fmt.Errorf("failed to get interfaces: %w", err)
	}

	// Look for eth0 or the first interface with a MAC address
	var primaryMAC string
	for _, iface := range interfaces {
		if iface.MACAddr != "" {
			// Prefer eth0/mgmt interface
			ifaceNameLower := strings.ToLower(iface.Name)
			if ifaceNameLower == "eth0" || ifaceNameLower == "mgmt" || ifaceNameLower == "management" {
				return iface.MACAddr, nil
			}
			// Store first MAC found as fallback
			if primaryMAC == "" {
				primaryMAC = iface.MACAddr
			}
		}
	}

	return primaryMAC, nil
}

// GetDeviceMetadata retrieves metadata for a specific device by MAC address from NetBox
func (s *Syncer) GetDeviceMetadata(ctx context.Context, mac string) (*DeviceMetadata, error) {
	normalizedMAC := macaddr.NormalizeOrEmpty(mac)
	if normalizedMAC == "" {
		return nil, fmt.Errorf("invalid MAC address: %s", mac)
	}

	device, err := s.client.GetDeviceByMAC(ctx, normalizedMAC)
	if err != nil {
		return nil, fmt.Errorf("failed to query NetBox: %w", err)
	}
	if device == nil {
		return nil, nil // Not found
	}

	// Extract site information
	siteName := ""
	siteID := ""
	if device.Site != nil {
		siteName = device.Site.Name
		siteID = fmt.Sprintf("%d", device.Site.ID)
	}

	// Extract device type information
	model := ""
	if device.DeviceType != nil {
		model = device.DeviceType.Model
	}

	return &DeviceMetadata{
		MAC:      normalizedMAC,
		Name:     device.Name,
		SiteID:   siteID,
		SiteName: siteName,
		Model:    model,
		Serial:   device.Serial,
	}, nil
}

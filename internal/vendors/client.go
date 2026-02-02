// Package vendors provides vendor-agnostic interfaces for multi-vendor network management.
//
// This package defines the Client interface and service interfaces that abstract
// vendor-specific APIs (Mist, Meraki, etc.) behind a common interface.
package vendors

import "context"

// Client is the vendor-agnostic interface for multi-vendor operations.
// Services return nil if the vendor does not support that capability.
type Client interface {
	// Core services (all vendors must implement)
	Sites() SitesService
	Inventory() InventoryService
	Devices() DevicesService

	// Optional services (return nil if unsupported)
	Search() SearchService
	Profiles() ProfilesService
	Templates() TemplatesService
	Configs() ConfigsService
	Statuses() StatusesService
	WLANs() WLANsService

	// Metadata
	VendorName() string
	OrgID() string
}

// SitesService provides site/network operations.
// In Mist this maps to Sites, in Meraki this maps to Networks.
type SitesService interface {
	List(ctx context.Context) ([]*SiteInfo, error)
	Get(ctx context.Context, id string) (*SiteInfo, error)
	ByName(ctx context.Context, name string) (*SiteInfo, error)
	Create(ctx context.Context, site *SiteInfo) (*SiteInfo, error)
	Update(ctx context.Context, id string, site *SiteInfo) (*SiteInfo, error)
	Delete(ctx context.Context, id string) error
}

// InventoryService provides inventory operations.
// Inventory represents devices claimed to the organization.
type InventoryService interface {
	List(ctx context.Context, deviceType string) ([]*InventoryItem, error)
	ByMAC(ctx context.Context, mac string) (*InventoryItem, error)
	BySerial(ctx context.Context, serial string) (*InventoryItem, error)
	Claim(ctx context.Context, claimCodes []string) ([]*InventoryItem, error)
	Release(ctx context.Context, serials []string) error
	AssignToSite(ctx context.Context, siteID string, macs []string) error
	UnassignFromSite(ctx context.Context, macs []string) error
}

// DevicesService provides device operations.
// Devices are inventory items that have been assigned to a site.
type DevicesService interface {
	List(ctx context.Context, siteID, deviceType string) ([]*DeviceInfo, error)
	Get(ctx context.Context, siteID, deviceID string) (*DeviceInfo, error)
	ByMAC(ctx context.Context, mac string) (*DeviceInfo, error)
	Update(ctx context.Context, siteID, deviceID string, device *DeviceInfo) (*DeviceInfo, error)
	Rename(ctx context.Context, siteID, deviceID, newName string) error
	UpdateConfig(ctx context.Context, siteID, deviceID string, config map[string]interface{}) error
}

// SearchService provides client search operations.
// This searches for end-user devices connected to the network infrastructure.
type SearchService interface {
	// SearchWiredClients searches for wired clients by text (hostname, MAC, IP, etc.)
	// Use opts.SiteID to scope the search to a specific site/network.
	SearchWiredClients(ctx context.Context, text string, opts SearchOptions) (*WiredSearchResults, error)

	// SearchWirelessClients searches for wireless clients by text
	// Use opts.SiteID to scope the search to a specific site/network.
	SearchWirelessClients(ctx context.Context, text string, opts SearchOptions) (*WirelessSearchResults, error)

	// EstimateSearchCost returns the estimated cost of a search operation.
	// This allows the command layer to warn users before expensive operations.
	// siteID can be empty for org-wide search.
	EstimateSearchCost(ctx context.Context, text string, siteID string) (*SearchCostEstimate, error)
}

// ProfilesService provides device profile operations.
// Device profiles are templates that can be assigned to devices.
// This is primarily a Mist concept - Meraki returns nil for this service.
type ProfilesService interface {
	List(ctx context.Context, profileType string) ([]*DeviceProfile, error)
	Get(ctx context.Context, profileID string) (*DeviceProfile, error)
	ByName(ctx context.Context, name, profileType string) (*DeviceProfile, error)
	Assign(ctx context.Context, profileID string, macs []string) error
	Unassign(ctx context.Context, profileID string, macs []string) error
}

// TemplatesService provides template operations.
// Templates are org-level configurations that can be applied to sites.
// This is primarily a Mist concept - Meraki returns nil for this service.
type TemplatesService interface {
	// ListRF returns RF templates
	ListRF(ctx context.Context) ([]*RFTemplate, error)

	// ListGateway returns gateway templates
	ListGateway(ctx context.Context) ([]*GatewayTemplate, error)

	// ListWLAN returns WLAN templates
	ListWLAN(ctx context.Context) ([]*WLANTemplate, error)
}

// ConfigsService provides device configuration operations.
// This retrieves the full configuration of individual devices.
type ConfigsService interface {
	// GetAPConfig returns the full configuration for an AP
	GetAPConfig(ctx context.Context, siteID, deviceID string) (*APConfig, error)

	// GetSwitchConfig returns the full configuration for a switch
	GetSwitchConfig(ctx context.Context, siteID, deviceID string) (*SwitchConfig, error)

	// GetGatewayConfig returns the full configuration for a gateway
	GetGatewayConfig(ctx context.Context, siteID, deviceID string) (*GatewayConfig, error)
}

// StatusesService provides device status operations.
// This retrieves the current status of devices in the organization.
type StatusesService interface {
	// GetAll returns the status of all devices in the organization.
	// Returns a map of normalized MAC address to DeviceStatus.
	GetAll(ctx context.Context) (map[string]*DeviceStatus, error)
}

// WLANsService provides WLAN/SSID configuration operations.
// For Mist: Manages org-level and site-level WLANs.
// For Meraki: Manages per-network SSIDs (numbered 0-14).
type WLANsService interface {
	List(ctx context.Context) ([]*WLAN, error)
	ListBySite(ctx context.Context, siteID string) ([]*WLAN, error)
	Get(ctx context.Context, id string) (*WLAN, error)
	BySSID(ctx context.Context, ssid string) ([]*WLAN, error)
	Create(ctx context.Context, wlan *WLAN) (*WLAN, error)
	Update(ctx context.Context, id string, wlan *WLAN) (*WLAN, error)
	Delete(ctx context.Context, id string) error
}

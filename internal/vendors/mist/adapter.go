package mist

import (
	"github.com/ravinald/wifimgr/api"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// Adapter wraps the legacy api.Client to implement vendors.Client.
// This provides a vendor-agnostic interface for Mist API operations.
type Adapter struct {
	legacy api.Client
	orgID  string
}

// NewAdapter creates a Mist adapter from the legacy client.
func NewAdapter(legacyClient api.Client, orgID string) vendors.Client {
	return &Adapter{
		legacy: legacyClient,
		orgID:  orgID,
	}
}

// VendorName returns the vendor identifier.
func (a *Adapter) VendorName() string {
	return "mist"
}

// OrgID returns the organization ID.
func (a *Adapter) OrgID() string {
	return a.orgID
}

// Sites returns the SitesService for site operations.
func (a *Adapter) Sites() vendors.SitesService {
	return &sitesService{client: a.legacy, orgID: a.orgID}
}

// Inventory returns the InventoryService for inventory operations.
func (a *Adapter) Inventory() vendors.InventoryService {
	return &inventoryService{client: a.legacy, orgID: a.orgID}
}

// Devices returns the DevicesService for device operations.
func (a *Adapter) Devices() vendors.DevicesService {
	return &devicesService{client: a.legacy, orgID: a.orgID}
}

// Search returns the SearchService for client search operations.
// Mist supports both wired and wireless client search.
func (a *Adapter) Search() vendors.SearchService {
	return &searchService{client: a.legacy, orgID: a.orgID}
}

// Profiles returns the ProfilesService for device profile operations.
// Mist supports device profiles for APs, switches, and gateways.
func (a *Adapter) Profiles() vendors.ProfilesService {
	return &profilesService{client: a.legacy, orgID: a.orgID}
}

// Templates returns the TemplatesService for template operations.
// Mist supports RF, gateway, and WLAN templates.
func (a *Adapter) Templates() vendors.TemplatesService {
	return &templatesService{client: a.legacy, orgID: a.orgID}
}

// Configs returns the ConfigsService for device configuration operations.
func (a *Adapter) Configs() vendors.ConfigsService {
	return &configsService{client: a.legacy, orgID: a.orgID}
}

// Statuses returns the StatusesService for device status information.
func (a *Adapter) Statuses() vendors.StatusesService {
	return &statusesService{client: a.legacy, orgID: a.orgID}
}

// WLANs returns the WLANsService for WLAN/SSID operations.
// Mist supports org-level and site-level WLANs.
func (a *Adapter) WLANs() vendors.WLANsService {
	return &wlansService{client: a.legacy, orgID: a.orgID}
}

// LegacyClient returns the underlying api.Client for advanced operations.
// This should only be used when vendor-specific functionality is required.
// Implements vendors.LegacyClientAccessor.
func (a *Adapter) LegacyClient() any {
	return a.legacy
}

// Ensure Adapter implements vendors.Client at compile time.
var _ vendors.Client = (*Adapter)(nil)

// Ensure Adapter implements vendors.LegacyClientAccessor at compile time.
var _ vendors.LegacyClientAccessor = (*Adapter)(nil)

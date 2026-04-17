// Package ubiquiti provides a Ubiquiti implementation of the vendors.Client interface.
// Phase 1 supports read-only operations via the Site Manager API v1.0.0.
package ubiquiti

import (
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// Adapter implements vendors.Client for the Ubiquiti Site Manager API.
type Adapter struct {
	client *Client
}

// NewAdapter creates a new Ubiquiti adapter.
func NewAdapter(apiKey, baseURL string, opts ...ClientOption) (vendors.Client, error) {
	if baseURL == "" {
		baseURL = "https://api.ui.com"
	}

	logging.Debugf("[ubiquiti] Creating Ubiquiti client with base URL %s", baseURL)

	client := NewClient(apiKey, baseURL, opts...)

	logging.Debugf("[ubiquiti] Successfully created Ubiquiti client with rate limiting")
	return &Adapter{client: client}, nil
}

func (a *Adapter) VendorName() string { return "ubiquiti" }
func (a *Adapter) OrgID() string      { return "" }

func (a *Adapter) Sites() vendors.SitesService {
	return &sitesService{client: a.client}
}

func (a *Adapter) Inventory() vendors.InventoryService {
	return &inventoryService{client: a.client}
}

func (a *Adapter) Devices() vendors.DevicesService {
	return &devicesService{client: a.client}
}

func (a *Adapter) Statuses() vendors.StatusesService {
	return &statusesService{client: a.client}
}

// Phase 1: unsupported services return nil.
func (a *Adapter) Search() vendors.SearchService             { return nil }
func (a *Adapter) Profiles() vendors.ProfilesService         { return nil }
func (a *Adapter) Templates() vendors.TemplatesService       { return nil }
func (a *Adapter) Configs() vendors.ConfigsService           { return nil }
func (a *Adapter) WLANs() vendors.WLANsService               { return nil }
func (a *Adapter) BSSIDs() vendors.BSSIDsService             { return nil }
func (a *Adapter) ClientDetail() vendors.ClientDetailService { return nil }

var _ vendors.Client = (*Adapter)(nil)

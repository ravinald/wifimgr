package aruba

import (
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// Adapter implements vendors.Client for a standalone Instant AP. One adapter
// targets one Virtual Controller, which wifimgr models as a single site.
type Adapter struct {
	client *Client
	siteID string // the VC host; stable site identifier for this swarm
}

// NewAdapter creates an Aruba Instant client for the VC at baseURL.
func NewAdapter(user, passwd, baseURL string, opts ...ClientOption) (vendors.Client, error) {
	client := NewClient(user, passwd, baseURL, opts...)
	logging.Debugf("[aruba] created Instant client for %s", client.host)
	return &Adapter{client: client, siteID: client.host}, nil
}

func (a *Adapter) VendorName() string { return vendorName }

// OrgID is empty: a standalone Instant deployment has no org concept.
func (a *Adapter) OrgID() string { return "" }

func (a *Adapter) Sites() vendors.SitesService {
	return &sitesService{client: a.client, siteID: a.siteID}
}

func (a *Adapter) Inventory() vendors.InventoryService {
	return &inventoryService{client: a.client, siteID: a.siteID}
}

func (a *Adapter) Devices() vendors.DevicesService {
	return &devicesService{client: a.client, siteID: a.siteID}
}

func (a *Adapter) WLANs() vendors.WLANsService {
	return &wlansService{client: a.client, siteID: a.siteID}
}

func (a *Adapter) Configs() vendors.ConfigsService {
	return &configsService{client: a.client, siteID: a.siteID}
}

func (a *Adapter) Statuses() vendors.StatusesService {
	return &statusesService{client: a.client}
}

// Unsupported services. Instant's device-local API exposes no org inventory
// claim, client search, device profiles, org templates, BSSID listing, or the
// per-client band supplement Meraki needs.
func (a *Adapter) Search() vendors.SearchService             { return nil }
func (a *Adapter) Profiles() vendors.ProfilesService         { return nil }
func (a *Adapter) Templates() vendors.TemplatesService       { return nil }
func (a *Adapter) BSSIDs() vendors.BSSIDsService             { return nil }
func (a *Adapter) ClientDetail() vendors.ClientDetailService { return nil }

var _ vendors.Client = (*Adapter)(nil)

package vendors

import (
	"context"
	"fmt"
)

// MockClient is a mock implementation of the vendors.Client interface for testing.
type MockClient struct {
	vendor string
	orgID  string

	// Services - set to nil to simulate unsupported capabilities
	sitesService     SitesService
	inventoryService InventoryService
	devicesService   DevicesService
	searchService    SearchService
	profilesService  ProfilesService
	templatesService TemplatesService
	configsService   ConfigsService
	statusesService  StatusesService
	wlansService     WLANsService

	// Error injection for testing error handling
	SitesError     error
	InventoryError error
	DevicesError   error
	SearchError    error
}

// NewMockClient creates a new mock client with default services.
func NewMockClient(vendor, orgID string) *MockClient {
	return &MockClient{
		vendor:           vendor,
		orgID:            orgID,
		sitesService:     NewMockSitesService(),
		inventoryService: NewMockInventoryService(),
		devicesService:   NewMockDevicesService(),
	}
}

// NewMockClientWithAllServices creates a mock client with all services enabled.
func NewMockClientWithAllServices(vendor, orgID string) *MockClient {
	return &MockClient{
		vendor:           vendor,
		orgID:            orgID,
		sitesService:     NewMockSitesService(),
		inventoryService: NewMockInventoryService(),
		devicesService:   NewMockDevicesService(),
		searchService:    NewMockSearchService(),
		profilesService:  NewMockProfilesService(),
		templatesService: NewMockTemplatesService(),
		configsService:   NewMockConfigsService(),
		statusesService:  NewMockStatusesService(),
		wlansService:     NewMockWLANsService(),
	}
}

func (m *MockClient) Sites() SitesService         { return m.sitesService }
func (m *MockClient) Inventory() InventoryService { return m.inventoryService }
func (m *MockClient) Devices() DevicesService     { return m.devicesService }
func (m *MockClient) Search() SearchService       { return m.searchService }
func (m *MockClient) Profiles() ProfilesService   { return m.profilesService }
func (m *MockClient) Templates() TemplatesService { return m.templatesService }
func (m *MockClient) Configs() ConfigsService     { return m.configsService }
func (m *MockClient) Statuses() StatusesService   { return m.statusesService }
func (m *MockClient) WLANs() WLANsService         { return m.wlansService }
func (m *MockClient) VendorName() string          { return m.vendor }
func (m *MockClient) OrgID() string               { return m.orgID }

// SetSitesService sets a custom sites service for testing.
func (m *MockClient) SetSitesService(svc SitesService) { m.sitesService = svc }

// SetInventoryService sets a custom inventory service for testing.
func (m *MockClient) SetInventoryService(svc InventoryService) { m.inventoryService = svc }

// SetSearchService sets a custom search service for testing.
func (m *MockClient) SetSearchService(svc SearchService) { m.searchService = svc }

// MockSitesService is a mock implementation of SitesService.
type MockSitesService struct {
	Sites     []*SiteInfo
	SitesByID map[string]*SiteInfo
	Error     error
}

// NewMockSitesService creates a new mock sites service with sample data.
func NewMockSitesService() *MockSitesService {
	sites := []*SiteInfo{
		{ID: "site-001", Name: "US-SFO-LAB", Timezone: "America/Los_Angeles", CountryCode: "US"},
		{ID: "site-002", Name: "US-NYC-OFFICE", Timezone: "America/New_York", CountryCode: "US"},
		{ID: "site-003", Name: "EU-LON-DC", Timezone: "Europe/London", CountryCode: "GB"},
	}
	byID := make(map[string]*SiteInfo)
	for _, s := range sites {
		byID[s.ID] = s
	}
	return &MockSitesService{Sites: sites, SitesByID: byID}
}

func (m *MockSitesService) List(_ context.Context) ([]*SiteInfo, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	return m.Sites, nil
}

func (m *MockSitesService) Get(_ context.Context, id string) (*SiteInfo, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	if s, ok := m.SitesByID[id]; ok {
		return s, nil
	}
	return nil, &SiteNotFoundError{SiteName: id}
}

func (m *MockSitesService) ByName(_ context.Context, name string) (*SiteInfo, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	for _, s := range m.Sites {
		if s.Name == name {
			return s, nil
		}
	}
	return nil, &SiteNotFoundError{SiteName: name}
}

func (m *MockSitesService) Create(_ context.Context, site *SiteInfo) (*SiteInfo, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	site.ID = fmt.Sprintf("site-%03d", len(m.Sites)+1)
	m.Sites = append(m.Sites, site)
	m.SitesByID[site.ID] = site
	return site, nil
}

func (m *MockSitesService) Update(_ context.Context, id string, site *SiteInfo) (*SiteInfo, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	if _, ok := m.SitesByID[id]; !ok {
		return nil, &SiteNotFoundError{SiteName: id}
	}
	site.ID = id
	m.SitesByID[id] = site
	return site, nil
}

func (m *MockSitesService) Delete(_ context.Context, id string) error {
	if m.Error != nil {
		return m.Error
	}
	delete(m.SitesByID, id)
	return nil
}

// MockInventoryService is a mock implementation of InventoryService.
type MockInventoryService struct {
	Items      []*InventoryItem
	itemsByMAC map[string]*InventoryItem
	bySerial   map[string]*InventoryItem
	Error      error
}

// NewMockInventoryService creates a new mock inventory service with sample data.
func NewMockInventoryService() *MockInventoryService {
	items := []*InventoryItem{
		{MAC: "aabbccddeef0", Serial: "AP001", Model: "AP43", Name: "AP-Floor1-01", Type: "ap", SiteID: "site-001"},
		{MAC: "aabbccddeef1", Serial: "AP002", Model: "AP43", Name: "AP-Floor1-02", Type: "ap", SiteID: "site-001"},
		{MAC: "aabbccddeef2", Serial: "SW001", Model: "EX4300", Name: "SW-Floor1-01", Type: "switch", SiteID: "site-001"},
		{MAC: "aabbccddeef3", Serial: "GW001", Model: "SRX300", Name: "GW-Main", Type: "gateway", SiteID: "site-001"},
	}
	byMAC := make(map[string]*InventoryItem)
	bySerial := make(map[string]*InventoryItem)
	for _, item := range items {
		byMAC[item.MAC] = item
		bySerial[item.Serial] = item
	}
	return &MockInventoryService{Items: items, itemsByMAC: byMAC, bySerial: bySerial}
}

func (m *MockInventoryService) List(_ context.Context, deviceType string) ([]*InventoryItem, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	if deviceType == "" {
		return m.Items, nil
	}
	var filtered []*InventoryItem
	for _, item := range m.Items {
		if item.Type == deviceType {
			filtered = append(filtered, item)
		}
	}
	return filtered, nil
}

func (m *MockInventoryService) ByMAC(_ context.Context, mac string) (*InventoryItem, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	if item, ok := m.itemsByMAC[NormalizeMAC(mac)]; ok {
		return item, nil
	}
	return nil, &DeviceNotFoundError{Identifier: mac}
}

func (m *MockInventoryService) BySerial(_ context.Context, serial string) (*InventoryItem, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	if item, ok := m.bySerial[serial]; ok {
		return item, nil
	}
	return nil, &DeviceNotFoundError{Identifier: serial}
}

func (m *MockInventoryService) Claim(_ context.Context, _ []string) ([]*InventoryItem, error) {
	return nil, m.Error
}

func (m *MockInventoryService) Release(_ context.Context, _ []string) error {
	return m.Error
}

func (m *MockInventoryService) AssignToSite(_ context.Context, siteID string, macs []string) error {
	if m.Error != nil {
		return m.Error
	}
	for _, mac := range macs {
		if item, ok := m.itemsByMAC[NormalizeMAC(mac)]; ok {
			item.SiteID = siteID
		}
	}
	return nil
}

func (m *MockInventoryService) UnassignFromSite(_ context.Context, macs []string) error {
	if m.Error != nil {
		return m.Error
	}
	for _, mac := range macs {
		if item, ok := m.itemsByMAC[NormalizeMAC(mac)]; ok {
			item.SiteID = ""
		}
	}
	return nil
}

// MockDevicesService is a mock implementation of DevicesService.
type MockDevicesService struct {
	Devices      []*DeviceInfo
	devicesByMAC map[string]*DeviceInfo
	Error        error
}

// NewMockDevicesService creates a new mock devices service.
func NewMockDevicesService() *MockDevicesService {
	devices := []*DeviceInfo{
		{ID: "dev-001", MAC: "aabbccddeef0", Name: "AP-Floor1-01", Model: "AP43", Type: "ap", SiteID: "site-001", Status: "connected"},
		{ID: "dev-002", MAC: "aabbccddeef1", Name: "AP-Floor1-02", Model: "AP43", Type: "ap", SiteID: "site-001", Status: "connected"},
	}
	byMAC := make(map[string]*DeviceInfo)
	for _, d := range devices {
		byMAC[d.MAC] = d
	}
	return &MockDevicesService{Devices: devices, devicesByMAC: byMAC}
}

func (m *MockDevicesService) List(_ context.Context, siteID, deviceType string) ([]*DeviceInfo, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	var filtered []*DeviceInfo
	for _, d := range m.Devices {
		if siteID != "" && d.SiteID != siteID {
			continue
		}
		if deviceType != "" && d.Type != deviceType {
			continue
		}
		filtered = append(filtered, d)
	}
	return filtered, nil
}

func (m *MockDevicesService) Get(_ context.Context, _, deviceID string) (*DeviceInfo, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	for _, d := range m.Devices {
		if d.ID == deviceID {
			return d, nil
		}
	}
	return nil, &DeviceNotFoundError{Identifier: deviceID}
}

func (m *MockDevicesService) ByMAC(_ context.Context, mac string) (*DeviceInfo, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	if d, ok := m.devicesByMAC[NormalizeMAC(mac)]; ok {
		return d, nil
	}
	return nil, &DeviceNotFoundError{Identifier: mac}
}

func (m *MockDevicesService) Update(_ context.Context, _, _ string, device *DeviceInfo) (*DeviceInfo, error) {
	return device, m.Error
}

func (m *MockDevicesService) Rename(_ context.Context, _, deviceID, newName string) error {
	if m.Error != nil {
		return m.Error
	}
	for _, d := range m.Devices {
		if d.ID == deviceID {
			d.Name = newName
			return nil
		}
	}
	return &DeviceNotFoundError{Identifier: deviceID}
}

func (m *MockDevicesService) UpdateConfig(_ context.Context, _, _ string, _ map[string]interface{}) error {
	return m.Error
}

// MockSearchService is a mock implementation of SearchService.
type MockSearchService struct {
	WirelessResults *WirelessSearchResults
	WiredResults    *WiredSearchResults
	Error           error
}

// NewMockSearchService creates a new mock search service.
func NewMockSearchService() *MockSearchService {
	return &MockSearchService{
		WirelessResults: &WirelessSearchResults{
			Results: []*WirelessClient{
				{MAC: "client001", Hostname: "laptop-john", IP: "10.0.1.100", SSID: "Corp-WiFi", APMAC: "aabbccddeef0"},
			},
			Total: 1,
		},
		WiredResults: &WiredSearchResults{
			Results: []*WiredClient{
				{MAC: "client002", Hostname: "desktop-jane", IP: "10.0.2.100", SwitchMAC: "aabbccddeef2", PortID: "ge-0/0/1"},
			},
			Total: 1,
		},
	}
}

func (m *MockSearchService) SearchWirelessClients(_ context.Context, _ string, _ SearchOptions) (*WirelessSearchResults, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	return m.WirelessResults, nil
}

func (m *MockSearchService) SearchWiredClients(_ context.Context, _ string, _ SearchOptions) (*WiredSearchResults, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	return m.WiredResults, nil
}

func (m *MockSearchService) EstimateSearchCost(_ context.Context, _ string, _ string) (*SearchCostEstimate, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	// Mock always returns low cost (Mist-like behavior)
	return &SearchCostEstimate{
		APICalls:          1,
		NeedsConfirmation: false,
		Description:       "Single API call",
	}, nil
}

// MockProfilesService is a mock implementation of ProfilesService.
type MockProfilesService struct {
	Profiles []*DeviceProfile
	Error    error
}

// NewMockProfilesService creates a new mock profiles service.
func NewMockProfilesService() *MockProfilesService {
	return &MockProfilesService{
		Profiles: []*DeviceProfile{
			{ID: "profile-001", Name: "Default-AP-Profile", Type: "ap"},
			{ID: "profile-002", Name: "High-Density-AP", Type: "ap"},
		},
	}
}

func (m *MockProfilesService) List(_ context.Context, _ string) ([]*DeviceProfile, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	return m.Profiles, nil
}

func (m *MockProfilesService) Get(_ context.Context, profileID string) (*DeviceProfile, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	for _, p := range m.Profiles {
		if p.ID == profileID {
			return p, nil
		}
	}
	return nil, fmt.Errorf("profile not found: %s", profileID)
}

func (m *MockProfilesService) ByName(_ context.Context, name, _ string) (*DeviceProfile, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	for _, p := range m.Profiles {
		if p.Name == name {
			return p, nil
		}
	}
	return nil, fmt.Errorf("profile not found: %s", name)
}

func (m *MockProfilesService) Assign(_ context.Context, _ string, _ []string) error {
	return m.Error
}

func (m *MockProfilesService) Unassign(_ context.Context, _ string, _ []string) error {
	return m.Error
}

// MockTemplatesService is a mock implementation of TemplatesService.
type MockTemplatesService struct {
	Error error
}

// NewMockTemplatesService creates a new mock templates service.
func NewMockTemplatesService() *MockTemplatesService {
	return &MockTemplatesService{}
}

func (m *MockTemplatesService) ListRF(_ context.Context) ([]*RFTemplate, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	return []*RFTemplate{{ID: "rf-001", Name: "Default-RF"}}, nil
}

func (m *MockTemplatesService) ListGateway(_ context.Context) ([]*GatewayTemplate, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	return []*GatewayTemplate{{ID: "gw-001", Name: "Default-Gateway"}}, nil
}

func (m *MockTemplatesService) ListWLAN(_ context.Context) ([]*WLANTemplate, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	return []*WLANTemplate{{ID: "wlan-001", Name: "Corp-WiFi"}}, nil
}

// MockConfigsService is a mock implementation of ConfigsService.
type MockConfigsService struct {
	Error error
}

// NewMockConfigsService creates a new mock configs service.
func NewMockConfigsService() *MockConfigsService {
	return &MockConfigsService{}
}

func (m *MockConfigsService) GetAPConfig(_ context.Context, siteID, deviceID string) (*APConfig, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	return &APConfig{ID: deviceID, SiteID: siteID, Config: map[string]interface{}{}}, nil
}

func (m *MockConfigsService) GetSwitchConfig(_ context.Context, siteID, deviceID string) (*SwitchConfig, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	return &SwitchConfig{ID: deviceID, SiteID: siteID, Config: map[string]interface{}{}}, nil
}

func (m *MockConfigsService) GetGatewayConfig(_ context.Context, siteID, deviceID string) (*GatewayConfig, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	return &GatewayConfig{ID: deviceID, SiteID: siteID, Config: map[string]interface{}{}}, nil
}

// MockStatusesService is a mock implementation of StatusesService.
type MockStatusesService struct {
	Statuses map[string]*DeviceStatus
	Error    error
}

// NewMockStatusesService creates a new mock statuses service with sample data.
func NewMockStatusesService() *MockStatusesService {
	return &MockStatusesService{
		Statuses: map[string]*DeviceStatus{
			"aabbccddeef0": {Status: "online"},
			"aabbccddeef1": {Status: "online"},
			"aabbccddeef2": {Status: "online"},
			"aabbccddeef3": {Status: "offline"},
		},
	}
}

func (m *MockStatusesService) GetAll(_ context.Context) (map[string]*DeviceStatus, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	return m.Statuses, nil
}

// MockWLANsService is a mock implementation of WLANsService.
type MockWLANsService struct {
	WLANs     []*WLAN
	wlansById map[string]*WLAN
	bySite    map[string][]*WLAN
	Error     error
}

// NewMockWLANsService creates a new mock WLANs service with sample data.
func NewMockWLANsService() *MockWLANsService {
	wlans := []*WLAN{
		{ID: "wlan-001", SSID: "Corp-WiFi", OrgID: "org-001", SiteID: "", Enabled: true, AuthType: "psk", VLANID: 100},
		{ID: "wlan-002", SSID: "Guest-WiFi", OrgID: "org-001", SiteID: "", Enabled: true, AuthType: "open", VLANID: 200},
		{ID: "wlan-003", SSID: "Lab-WiFi", OrgID: "org-001", SiteID: "site-001", Enabled: true, AuthType: "psk", VLANID: 300},
	}
	byID := make(map[string]*WLAN)
	bySite := make(map[string][]*WLAN)
	for _, w := range wlans {
		byID[w.ID] = w
		if w.SiteID != "" {
			bySite[w.SiteID] = append(bySite[w.SiteID], w)
		}
	}
	return &MockWLANsService{WLANs: wlans, wlansById: byID, bySite: bySite}
}

func (m *MockWLANsService) List(_ context.Context) ([]*WLAN, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	return m.WLANs, nil
}

func (m *MockWLANsService) ListBySite(_ context.Context, siteID string) ([]*WLAN, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	// Return org-level WLANs (no siteID) plus site-specific WLANs
	var result []*WLAN
	for _, w := range m.WLANs {
		if w.SiteID == "" || w.SiteID == siteID {
			result = append(result, w)
		}
	}
	return result, nil
}

func (m *MockWLANsService) Get(_ context.Context, id string) (*WLAN, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	if w, ok := m.wlansById[id]; ok {
		return w, nil
	}
	return nil, fmt.Errorf("WLAN not found: %s", id)
}

func (m *MockWLANsService) BySSID(_ context.Context, ssid string) ([]*WLAN, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	var result []*WLAN
	for _, w := range m.WLANs {
		if w.SSID == ssid {
			result = append(result, w)
		}
	}
	return result, nil
}

func (m *MockWLANsService) Create(_ context.Context, wlan *WLAN) (*WLAN, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	wlan.ID = fmt.Sprintf("wlan-%03d", len(m.WLANs)+1)
	m.WLANs = append(m.WLANs, wlan)
	m.wlansById[wlan.ID] = wlan
	return wlan, nil
}

func (m *MockWLANsService) Update(_ context.Context, id string, wlan *WLAN) (*WLAN, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	if _, ok := m.wlansById[id]; !ok {
		return nil, fmt.Errorf("WLAN not found: %s", id)
	}
	wlan.ID = id
	m.wlansById[id] = wlan
	return wlan, nil
}

func (m *MockWLANsService) Delete(_ context.Context, id string) error {
	if m.Error != nil {
		return m.Error
	}
	delete(m.wlansById, id)
	return nil
}

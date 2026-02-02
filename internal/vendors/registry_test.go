package vendors

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

func TestNewAPIClientRegistry(t *testing.T) {
	r := NewAPIClientRegistry()
	if r == nil {
		t.Fatal("NewAPIClientRegistry returned nil")
	}
	if r.Count() != 0 {
		t.Errorf("expected 0 clients, got %d", r.Count())
	}
}

func TestRegisterFactory(t *testing.T) {
	r := NewAPIClientRegistry()

	factory := func(config *APIConfig) (Client, error) {
		return NewMockClient(config.Vendor, config.Credentials["org_id"]), nil
	}

	r.RegisterFactory("mist", factory)
	r.RegisterFactory("meraki", factory)

	// Factory registration is internal, so we test it indirectly via InitializeClients
	configs := map[string]*APIConfig{
		"lab": {
			Label:       "lab",
			Vendor:      "mist",
			Credentials: map[string]string{"org_id": "org-123"},
		},
	}

	errs := r.InitializeClients(configs)
	if len(errs) > 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	if r.Count() != 1 {
		t.Errorf("expected 1 client, got %d", r.Count())
	}
}

func TestInitializeClients_UnsupportedVendor(t *testing.T) {
	r := NewAPIClientRegistry()

	configs := map[string]*APIConfig{
		"unknown": {
			Label:  "unknown",
			Vendor: "unsupported_vendor",
		},
	}

	errs := r.InitializeClients(configs)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if r.Count() != 0 {
		t.Errorf("expected 0 clients after failed init, got %d", r.Count())
	}
}

func TestInitializeClients_FactoryError(t *testing.T) {
	r := NewAPIClientRegistry()

	factoryError := errors.New("connection failed")
	r.RegisterFactory("failing", func(config *APIConfig) (Client, error) {
		return nil, factoryError
	})

	configs := map[string]*APIConfig{
		"fail": {
			Label:  "fail",
			Vendor: "failing",
		},
	}

	errs := r.InitializeClients(configs)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if r.Count() != 0 {
		t.Errorf("expected 0 clients after failed init, got %d", r.Count())
	}
}

func TestInitializeClients_PartialSuccess(t *testing.T) {
	r := NewAPIClientRegistry()

	// Register good factory
	r.RegisterFactory("mock", func(config *APIConfig) (Client, error) {
		return NewMockClient(config.Vendor, config.Credentials["org_id"]), nil
	})

	configs := map[string]*APIConfig{
		"good": {
			Label:       "good",
			Vendor:      "mock",
			Credentials: map[string]string{"org_id": "org-good"},
		},
		"bad": {
			Label:  "bad",
			Vendor: "unsupported",
		},
	}

	errs := r.InitializeClients(configs)
	if len(errs) != 1 {
		t.Errorf("expected 1 error for unsupported vendor, got %d", len(errs))
	}
	if r.Count() != 1 {
		t.Errorf("expected 1 successful client, got %d", r.Count())
	}
	if !r.HasAPI("good") {
		t.Error("expected 'good' API to be registered")
	}
	if r.HasAPI("bad") {
		t.Error("'bad' API should not be registered")
	}
}

func TestGetClient(t *testing.T) {
	r := setupRegistryWithMockClient(t)

	client, err := r.GetClient("test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("GetClient returned nil client")
	}
	if client.VendorName() != "mock" {
		t.Errorf("expected vendor 'mock', got %q", client.VendorName())
	}
}

func TestGetClient_NotFound(t *testing.T) {
	r := NewAPIClientRegistry()

	_, err := r.GetClient("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent API")
	}

	var apiErr *APINotFoundError
	if !errors.As(err, &apiErr) {
		t.Errorf("expected APINotFoundError, got %T", err)
	}
}

func TestMustGetClient_Panic(t *testing.T) {
	r := NewAPIClientRegistry()

	defer func() {
		if recover() == nil {
			t.Error("expected panic for nonexistent API")
		}
	}()

	r.MustGetClient("nonexistent")
}

func TestGetAllLabels(t *testing.T) {
	r := NewAPIClientRegistry()
	r.RegisterFactory("mock", func(config *APIConfig) (Client, error) {
		return NewMockClient(config.Vendor, config.Credentials["org_id"]), nil
	})

	configs := map[string]*APIConfig{
		"zebra": {Label: "zebra", Vendor: "mock", Credentials: map[string]string{"org_id": "1"}},
		"alpha": {Label: "alpha", Vendor: "mock", Credentials: map[string]string{"org_id": "2"}},
		"beta":  {Label: "beta", Vendor: "mock", Credentials: map[string]string{"org_id": "3"}},
	}

	r.InitializeClients(configs)

	labels := r.GetAllLabels()
	if len(labels) != 3 {
		t.Fatalf("expected 3 labels, got %d", len(labels))
	}

	// Should be sorted
	expected := []string{"alpha", "beta", "zebra"}
	for i, label := range labels {
		if label != expected[i] {
			t.Errorf("label[%d] = %q, want %q", i, label, expected[i])
		}
	}
}

func TestHasAPI(t *testing.T) {
	r := setupRegistryWithMockClient(t)

	if !r.HasAPI("test") {
		t.Error("expected HasAPI('test') to return true")
	}
	if r.HasAPI("nonexistent") {
		t.Error("expected HasAPI('nonexistent') to return false")
	}
}

func TestGetVendor(t *testing.T) {
	r := setupRegistryWithMockClient(t)

	vendor, err := r.GetVendor("test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vendor != "mock" {
		t.Errorf("expected vendor 'mock', got %q", vendor)
	}
}

func TestGetVendor_NotFound(t *testing.T) {
	r := NewAPIClientRegistry()

	_, err := r.GetVendor("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent API")
	}
}

func TestGetOrgID(t *testing.T) {
	r := setupRegistryWithMockClient(t)

	orgID, err := r.GetOrgID("test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orgID != "org-test" {
		t.Errorf("expected org_id 'org-test', got %q", orgID)
	}
}

func TestGetConfig(t *testing.T) {
	r := setupRegistryWithMockClient(t)

	config, err := r.GetConfig("test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if config.Label != "test" {
		t.Errorf("expected label 'test', got %q", config.Label)
	}
	if config.Vendor != "mock" {
		t.Errorf("expected vendor 'mock', got %q", config.Vendor)
	}
}

func TestForEachAPI(t *testing.T) {
	r := NewAPIClientRegistry()
	r.RegisterFactory("mock", func(config *APIConfig) (Client, error) {
		return NewMockClient(config.Vendor, config.Credentials["org_id"]), nil
	})

	configs := map[string]*APIConfig{
		"api1": {Label: "api1", Vendor: "mock", Credentials: map[string]string{"org_id": "1"}},
		"api2": {Label: "api2", Vendor: "mock", Credentials: map[string]string{"org_id": "2"}},
	}
	r.InitializeClients(configs)

	var visited []string
	err := r.ForEachAPI(func(label string, client Client) error {
		visited = append(visited, label)
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(visited) != 2 {
		t.Errorf("expected 2 visited, got %d", len(visited))
	}
	// Should be sorted
	if visited[0] != "api1" || visited[1] != "api2" {
		t.Errorf("expected sorted iteration, got %v", visited)
	}
}

func TestForEachAPI_StopsOnError(t *testing.T) {
	r := NewAPIClientRegistry()
	r.RegisterFactory("mock", func(config *APIConfig) (Client, error) {
		return NewMockClient(config.Vendor, config.Credentials["org_id"]), nil
	})

	configs := map[string]*APIConfig{
		"api1": {Label: "api1", Vendor: "mock", Credentials: map[string]string{"org_id": "1"}},
		"api2": {Label: "api2", Vendor: "mock", Credentials: map[string]string{"org_id": "2"}},
	}
	r.InitializeClients(configs)

	testError := errors.New("test error")
	count := 0
	err := r.ForEachAPI(func(label string, client Client) error {
		count++
		return testError
	})

	if err == nil {
		t.Fatal("expected error")
	}
	if count != 1 {
		t.Errorf("expected iteration to stop after first error, count=%d", count)
	}
}

func TestForEachAPIParallel(t *testing.T) {
	r := NewAPIClientRegistry()
	r.RegisterFactory("mock", func(config *APIConfig) (Client, error) {
		return NewMockClient(config.Vendor, config.Credentials["org_id"]), nil
	})

	configs := map[string]*APIConfig{
		"api1": {Label: "api1", Vendor: "mock", Credentials: map[string]string{"org_id": "1"}},
		"api2": {Label: "api2", Vendor: "mock", Credentials: map[string]string{"org_id": "2"}},
		"api3": {Label: "api3", Vendor: "mock", Credentials: map[string]string{"org_id": "3"}},
	}
	r.InitializeClients(configs)

	var count int32
	errs := r.ForEachAPIParallel(func(label string, client Client) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	if count != 3 {
		t.Errorf("expected 3 executions, got %d", count)
	}
}

func TestForEachAPIParallel_CollectsErrors(t *testing.T) {
	r := NewAPIClientRegistry()
	r.RegisterFactory("mock", func(config *APIConfig) (Client, error) {
		return NewMockClient(config.Vendor, config.Credentials["org_id"]), nil
	})

	configs := map[string]*APIConfig{
		"api1": {Label: "api1", Vendor: "mock", Credentials: map[string]string{"org_id": "1"}},
		"api2": {Label: "api2", Vendor: "mock", Credentials: map[string]string{"org_id": "2"}},
	}
	r.InitializeClients(configs)

	errs := r.ForEachAPIParallel(func(label string, client Client) error {
		if label == "api1" {
			return errors.New("api1 failed")
		}
		return nil
	})

	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if _, ok := errs["api1"]; !ok {
		t.Error("expected error for api1")
	}
}

func TestGetStatus(t *testing.T) {
	r := setupRegistryWithMockClient(t)

	statuses := r.GetStatus()
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(statuses))
	}

	status := statuses[0]
	if status.Label != "test" {
		t.Errorf("expected label 'test', got %q", status.Label)
	}
	if status.Vendor != "mock" {
		t.Errorf("expected vendor 'mock', got %q", status.Vendor)
	}
	if !status.Healthy {
		t.Error("expected healthy status")
	}

	// Core capabilities
	hasCore := map[string]bool{"sites": false, "inventory": false, "devices": false}
	for _, capability := range status.Capabilities {
		hasCore[capability] = true
	}
	for capability, found := range hasCore {
		if !found {
			t.Errorf("missing core capability: %s", capability)
		}
	}
}

func TestListCapabilities(t *testing.T) {
	// Test with all services enabled
	fullClient := NewMockClientWithAllServices("mock", "org-1")
	caps := listCapabilities(fullClient)

	expected := []string{"sites", "inventory", "devices", "search", "profiles", "templates", "configs"}
	if len(caps) != len(expected) {
		t.Errorf("expected %d capabilities, got %d: %v", len(expected), len(caps), caps)
	}

	// Test with minimal services
	minimalClient := NewMockClient("mock", "org-1")
	minimalClient.SetSearchService(nil)
	minimalCaps := listCapabilities(minimalClient)

	// Should have core capabilities but not search (since mock has no search by default)
	// Note: NewMockClient doesn't set search, profiles, templates, configs by default
	if len(minimalCaps) < 3 {
		t.Errorf("expected at least 3 capabilities for minimal client, got %d", len(minimalCaps))
	}
}

// Helper to create a registry with a mock client
func setupRegistryWithMockClient(t *testing.T) *APIClientRegistry {
	t.Helper()
	r := NewAPIClientRegistry()
	r.RegisterFactory("mock", func(config *APIConfig) (Client, error) {
		return NewMockClient(config.Vendor, config.Credentials["org_id"]), nil
	})

	configs := map[string]*APIConfig{
		"test": {
			Label:       "test",
			Vendor:      "mock",
			Credentials: map[string]string{"org_id": "org-test"},
		},
	}
	r.InitializeClients(configs)
	return r
}

// Test concurrent access to registry
func TestRegistry_ConcurrentAccess(t *testing.T) {
	r := setupRegistryWithMockClient(t)

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_, _ = r.GetClient("test")
				r.HasAPI("test")
				r.GetAllLabels()
				_, _ = r.GetVendor("test")
				r.Count()
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

// Test that mock client implements the interface correctly
func TestMockClient_ImplementsInterface(t *testing.T) {
	var _ Client = (*MockClient)(nil) // Compile-time check

	client := NewMockClient("test-vendor", "org-123")

	if client.VendorName() != "test-vendor" {
		t.Errorf("expected VendorName='test-vendor', got %q", client.VendorName())
	}
	if client.OrgID() != "org-123" {
		t.Errorf("expected OrgID='org-123', got %q", client.OrgID())
	}

	// Test service accessors
	if client.Sites() == nil {
		t.Error("Sites() should not be nil for default mock")
	}
	if client.Inventory() == nil {
		t.Error("Inventory() should not be nil for default mock")
	}
	if client.Devices() == nil {
		t.Error("Devices() should not be nil for default mock")
	}
	// Search, Profiles, Templates, Configs are nil by default
	if client.Search() != nil {
		t.Error("Search() should be nil for default mock")
	}
}

func TestMockSitesService(t *testing.T) {
	ctx := context.Background()
	svc := NewMockSitesService()

	// Test List
	sites, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(sites) != 3 {
		t.Errorf("expected 3 sites, got %d", len(sites))
	}

	// Test ByName
	site, err := svc.ByName(ctx, "US-SFO-LAB")
	if err != nil {
		t.Fatalf("ByName error: %v", err)
	}
	if site.ID != "site-001" {
		t.Errorf("expected ID='site-001', got %q", site.ID)
	}

	// Test Get
	site, err = svc.Get(ctx, "site-002")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if site.Name != "US-NYC-OFFICE" {
		t.Errorf("expected Name='US-NYC-OFFICE', got %q", site.Name)
	}

	// Test not found
	_, err = svc.ByName(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent site")
	}
	var siteErr *SiteNotFoundError
	if !errors.As(err, &siteErr) {
		t.Errorf("expected SiteNotFoundError, got %T", err)
	}

	// Test Create
	newSite := &SiteInfo{Name: "NEW-SITE", Timezone: "UTC", CountryCode: "US"}
	created, err := svc.Create(ctx, newSite)
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if created.ID == "" {
		t.Error("expected ID to be assigned")
	}

	// Test error injection
	svc.Error = errors.New("test error")
	_, err = svc.List(ctx)
	if err == nil {
		t.Error("expected error when Error is set")
	}
}

func TestMockInventoryService(t *testing.T) {
	ctx := context.Background()
	svc := NewMockInventoryService()

	// Test List all
	items, err := svc.List(ctx, "")
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(items) != 4 {
		t.Errorf("expected 4 items, got %d", len(items))
	}

	// Test List by type
	items, err = svc.List(ctx, "ap")
	if err != nil {
		t.Fatalf("List(ap) error: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 APs, got %d", len(items))
	}

	// Test ByMAC
	item, err := svc.ByMAC(ctx, "aabbccddeef0")
	if err != nil {
		t.Fatalf("ByMAC error: %v", err)
	}
	if item.Serial != "AP001" {
		t.Errorf("expected Serial='AP001', got %q", item.Serial)
	}

	// Test BySerial
	item, err = svc.BySerial(ctx, "SW001")
	if err != nil {
		t.Fatalf("BySerial error: %v", err)
	}
	if item.Type != "switch" {
		t.Errorf("expected Type='switch', got %q", item.Type)
	}

	// Test AssignToSite
	err = svc.AssignToSite(ctx, "site-new", []string{"aabbccddeef0"})
	if err != nil {
		t.Fatalf("AssignToSite error: %v", err)
	}
	item, _ = svc.ByMAC(ctx, "aabbccddeef0")
	if item.SiteID != "site-new" {
		t.Errorf("expected SiteID='site-new', got %q", item.SiteID)
	}
}

func TestMockDevicesService(t *testing.T) {
	ctx := context.Background()
	svc := NewMockDevicesService()

	// Test List
	devices, err := svc.List(ctx, "", "")
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(devices) != 2 {
		t.Errorf("expected 2 devices, got %d", len(devices))
	}

	// Test List with site filter
	devices, err = svc.List(ctx, "site-001", "")
	if err != nil {
		t.Fatalf("List(site-001) error: %v", err)
	}
	if len(devices) != 2 {
		t.Errorf("expected 2 devices in site-001, got %d", len(devices))
	}

	// Test ByMAC
	device, err := svc.ByMAC(ctx, "aabbccddeef0")
	if err != nil {
		t.Fatalf("ByMAC error: %v", err)
	}
	if device.Name != "AP-Floor1-01" {
		t.Errorf("expected Name='AP-Floor1-01', got %q", device.Name)
	}
}

func TestMockSearchService(t *testing.T) {
	ctx := context.Background()
	svc := NewMockSearchService()
	opts := SearchOptions{}

	// Test wireless search
	results, err := svc.SearchWirelessClients(ctx, "laptop", opts)
	if err != nil {
		t.Fatalf("SearchWirelessClients error: %v", err)
	}
	if results.Total != 1 {
		t.Errorf("expected Total=1, got %d", results.Total)
	}

	// Test wired search
	wiredResults, err := svc.SearchWiredClients(ctx, "desktop", opts)
	if err != nil {
		t.Fatalf("SearchWiredClients error: %v", err)
	}
	if wiredResults.Total != 1 {
		t.Errorf("expected Total=1, got %d", wiredResults.Total)
	}

	// Test cost estimation
	cost, err := svc.EstimateSearchCost(ctx, "laptop", "")
	if err != nil {
		t.Fatalf("EstimateSearchCost error: %v", err)
	}
	if cost.APICalls != 1 {
		t.Errorf("expected APICalls=1, got %d", cost.APICalls)
	}
	if cost.NeedsConfirmation {
		t.Error("expected NeedsConfirmation=false for mock")
	}
}

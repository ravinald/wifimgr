package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// MockClient is a mock implementation of the Client interface
// for testing purposes. It simulates API responses without making
// actual HTTP requests to the Mist API.
type MockClient struct {
	config             Config
	httpClient         *http.Client
	rateLimiter        *mockRateLimiter
	sitesCache         *mockCache[[]Site]
	apsCache           *mockCache[[]AP]
	inventoryCache     *mockCache[[]InventoryItem]
	deviceProfileCache *mockCache[[]DeviceProfile]
	deviceCache        *DeviceCache
	debug              bool

	// Mock data storage
	sites          map[string]*MistSite        // siteID -> MistSite
	sitesByName    map[string]*MistSite        // name -> MistSite
	aps            map[string]map[string]*AP   // siteID -> apID -> AP
	apsBySerial    map[string]*AP              // serial -> AP
	apsByMAC       map[string]*AP              // MAC -> AP
	inventory      []InventoryItem             // inventory items list
	inventoryByMAC map[string]InventoryItem    // MAC -> InventoryItem
	deviceProfiles []DeviceProfile             // device profiles list
	profilesByName map[string]*DeviceProfile   // name -> profile
	profilesByID   map[string]*DeviceProfile   // id -> profile
	profilesByType map[string][]*DeviceProfile // type -> profiles

	mu sync.RWMutex
}

// Ensure MockClient implements the Client interface
var _ Client = (*MockClient)(nil)

// NewMockClient creates a new mock implementation of the Mist client
func NewMockClient(config Config) Client {
	// Initialize rate limiter if rate limiting is enabled
	var limiter *mockRateLimiter
	if config.RateLimit > 0 {
		rateDuration := config.RateDuration
		if rateDuration == 0 {
			rateDuration = time.Minute
		}
		limiter = newMockRateLimiter(config.RateLimit, rateDuration)
	}

	return &MockClient{
		config:             config,
		httpClient:         &http.Client{Timeout: config.Timeout},
		rateLimiter:        limiter,
		sitesCache:         newMockCache[[]Site](5 * time.Minute),
		apsCache:           newMockCache[[]AP](5 * time.Minute),
		inventoryCache:     newMockCache[[]InventoryItem](5 * time.Minute),
		deviceProfileCache: newMockCache[[]DeviceProfile](5 * time.Minute),
		deviceCache:        nil, // Mock client doesn't need real device cache
		debug:              config.Debug,

		// Initialize mock data stores
		sites:          make(map[string]*MistSite),
		sitesByName:    make(map[string]*MistSite),
		aps:            make(map[string]map[string]*AP),
		apsBySerial:    make(map[string]*AP),
		apsByMAC:       make(map[string]*AP),
		inventory:      []InventoryItem{},
		inventoryByMAC: make(map[string]InventoryItem),
		deviceProfiles: []DeviceProfile{},
		profilesByName: make(map[string]*DeviceProfile),
		profilesByID:   make(map[string]*DeviceProfile),
		profilesByType: make(map[string][]*DeviceProfile),
	}
}

// Logging and utility methods
// ============================================================================

// logRequest logs API request details in debug mode
func (m *MockClient) logRequest(method, path string, body interface{}) {
	if m.debug {
		bodyStr := ""
		if body != nil {
			bodyJSON, _ := json.Marshal(body)
			bodyStr = string(bodyJSON)
		}

		if bodyStr != "" {
			fmt.Printf("[DEBUG-MOCK] API Request: %s %s with body %s\n", method, path, bodyStr)
		} else {
			fmt.Printf("[DEBUG-MOCK] API Request: %s %s\n", method, path)
		}
	}
}

// Client configuration methods
// ============================================================================

// SetRateLimit sets the rate limit for the client
func (m *MockClient) SetRateLimit(limit int, duration time.Duration) {
	if limit > 0 {
		m.rateLimiter = newMockRateLimiter(limit, duration)
		m.config.RateLimit = limit
		m.config.RateDuration = duration
	} else {
		m.rateLimiter = nil
	}
}

// SetResultsLimit sets the results limit for pagination
func (m *MockClient) SetResultsLimit(limit int) {
	m.config.ResultsLimit = limit
}

// SetDebug enables or disables debug mode
func (m *MockClient) SetDebug(debug bool) {
	m.debug = debug
	m.config.Debug = debug
}

// Close releases resources used by the mock client
func (m *MockClient) Close() error {
	return nil
}

// GetDeviceConfig retrieves the configuration for a specific device (mock implementation)
func (m *MockClient) GetDeviceConfig(ctx context.Context, siteID, deviceID string) (*DeviceConfigResponse, error) {
	// Mock implementation - return empty config
	return &DeviceConfigResponse{Raw: make(map[string]interface{})}, nil
}

// GetAPConfig retrieves the configuration for a specific AP device (mock implementation)
func (m *MockClient) GetAPConfig(ctx context.Context, siteID, deviceID string) (*APConfig, error) {
	// Mock implementation - return empty config
	return &APConfig{}, nil
}

// GetSwitchConfig retrieves the configuration for a specific Switch device (mock implementation)
func (m *MockClient) GetSwitchConfig(ctx context.Context, siteID, deviceID string) (*SwitchConfig, error) {
	// Mock implementation - return empty config
	return &SwitchConfig{}, nil
}

// GetGatewayConfig retrieves the configuration for a specific Gateway device (mock implementation)
func (m *MockClient) GetGatewayConfig(ctx context.Context, siteID, deviceID string) (*GatewayConfig, error) {
	// Mock implementation - return empty config
	return &GatewayConfig{}, nil
}

// BatchGetDeviceConfigs retrieves configurations for multiple devices (mock implementation)
func (m *MockClient) BatchGetDeviceConfigs(ctx context.Context, devices []DeviceInfo) (map[string]*DeviceConfig, map[string]error) {
	// Mock implementation - return empty configs
	configs := make(map[string]*DeviceConfig)
	errors := make(map[string]error)
	return configs, errors
}

package api

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/ravinald/wifimgr/internal/logging"
)

// Client is the interface for Mist API operations
type Client interface {
	// Organizations
	GetOrgStats(ctx context.Context, orgID string) (*OrgStats, error)

	// Sites
	GetSiteName(siteID string) (string, bool)
	GetOrgName(orgID string) (string, bool)
	GetSites(ctx context.Context, orgID string) ([]*MistSite, error)
	GetSite(ctx context.Context, siteID string) (*MistSite, error)
	GetSiteByName(ctx context.Context, name, orgID string) (*MistSite, error)
	GetSiteByIdentifier(ctx context.Context, siteIdentifier string) (*MistSite, error)
	CreateSite(ctx context.Context, site *MistSite) (*MistSite, error)
	UpdateSite(ctx context.Context, siteID string, site *MistSite) (*MistSite, error)
	UpdateSiteByName(ctx context.Context, siteName string, site *MistSite) (*MistSite, error)
	DeleteSite(ctx context.Context, siteID string) error
	DeleteSiteByName(ctx context.Context, siteName string) error

	// Site Settings
	GetSiteSetting(ctx context.Context, siteID string) (*SiteSetting, error)

	// Devices API
	GetDevices(ctx context.Context, siteID string, deviceType string) ([]UnifiedDevice, error)
	GetDeviceByMAC(ctx context.Context, mac string) (*UnifiedDevice, error)
	GetDeviceByID(ctx context.Context, siteID, deviceID string) (*UnifiedDevice, error)
	GetDeviceByName(ctx context.Context, siteID, name string) (*UnifiedDevice, error)
	GetDevicesByType(ctx context.Context, siteID string, deviceType string) ([]UnifiedDevice, error)
	UpdateDevice(ctx context.Context, siteID string, deviceID string, device *UnifiedDevice) (*UnifiedDevice, error)
	AssignDevice(ctx context.Context, orgID string, siteID string, mac string) (*UnifiedDevice, error)
	UnassignDevice(ctx context.Context, orgID string, siteID string, deviceID string) error
	AssignDevicesToSite(ctx context.Context, orgID string, siteID string, macs []string, noReassign bool) error
	UnassignDevicesFromSite(ctx context.Context, orgID string, macs []string) error

	// Inventory API
	DeleteDevicesFromSite(ctx context.Context, macs []string) error
	GetInventoryConfig(inventoryPath string) (*InventoryConfig, error)
	GetInventory(ctx context.Context, orgID string, deviceType string) ([]*MistInventoryItem, error)
	GetInventoryItem(ctx context.Context, orgID string, itemID string) (*MistInventoryItem, error)
	GetInventoryItemByMAC(ctx context.Context, orgID string, macAddress string) (*MistInventoryItem, error)
	UpdateInventoryItem(ctx context.Context, orgID string, itemID string, item *MistInventoryItem) (*MistInventoryItem, error)
	ClaimInventoryItem(ctx context.Context, orgID string, claimCodes []string) ([]*MistInventoryItem, error)
	ReleaseInventoryItem(ctx context.Context, orgID string, itemIDs []string) error
	AssignInventoryItemsToSite(ctx context.Context, orgID string, siteID string, itemMACs []string) error
	UnassignInventoryItemsFromSite(ctx context.Context, orgID string, itemMACs []string) error

	// Cache Management
	ForceRebuildCache(ctx context.Context) error
	UpdateCacheForTypes(ctx context.Context, deviceTypes []string, siteNames []string) error
	PopulateDeviceCacheForSite(ctx context.Context, siteID string, deviceType string) error

	// Device Profiles
	GetDeviceProfiles(ctx context.Context, orgID string, profileType string) ([]DeviceProfile, error)
	GetDeviceProfile(ctx context.Context, orgID string, profileID string) (*DeviceProfile, error)
	GetDeviceProfileByName(ctx context.Context, orgID string, name string, profileType string) (*DeviceProfile, error)
	AssignDeviceProfile(ctx context.Context, orgID string, profileID string, macs []string) (*DeviceProfileAssignResult, error)
	UnassignDeviceProfiles(ctx context.Context, orgID string, profileID string, macs []string) error

	// Templates and Networks
	GetRFTemplates(ctx context.Context, orgID string) ([]MistRFTemplate, error)
	GetGatewayTemplates(ctx context.Context, orgID string) ([]MistGatewayTemplate, error)
	GetWLANTemplates(ctx context.Context, orgID string) ([]MistWLANTemplate, error)
	GetNetworks(ctx context.Context, orgID string) ([]MistNetwork, error)
	GetWLANs(ctx context.Context, orgID string) ([]MistWLAN, error)
	GetSiteWLANs(ctx context.Context, siteID string) ([]MistWLAN, error)

	// Configuration
	SetRateLimit(limit int, duration time.Duration)
	SetResultsLimit(limit int)
	SetDebug(debug bool)

	// Lifecycle
	Close() error // Release resources; call when client is no longer needed

	// Authentication & User Info
	ValidateAPIToken(ctx context.Context) (*SelfResponse, error)
	GetAPIUserInfo(ctx context.Context) (*SelfResponse, error)

	// Returns the configuration directory where settings files are stored
	GetConfigDirectory() string
	// Returns the schema directory where JSON schema files are stored
	GetSchemaDirectory() string

	// Cache operations
	GetCacheAccessor() CacheAccessor
	GetDeviceCache() *DeviceCache

	// Raw data operations for detail view
	GetRawDeviceJSON(ctx context.Context, siteID, deviceID string) (string, error)

	// Device extensive information query
	QueryDeviceExtensive(ctx context.Context, siteID, deviceID string) error

	// AP-specific methods for backward compatibility
	GetAPBySerialOrMAC(ctx context.Context, siteID, serial, mac string) (*AP, error)

	// Search API
	SearchWiredClients(ctx context.Context, orgID string, text string) (*MistWiredClientResponse, error)
	SearchWirelessClients(ctx context.Context, orgID string, text string) (*MistWirelessClientResponse, error)

	// Device Configuration API
	GetDeviceConfig(ctx context.Context, siteID, deviceID string) (*DeviceConfigResponse, error)
	GetAPConfig(ctx context.Context, siteID, deviceID string) (*APConfig, error)
	GetSwitchConfig(ctx context.Context, siteID, deviceID string) (*SwitchConfig, error)
	GetGatewayConfig(ctx context.Context, siteID, deviceID string) (*GatewayConfig, error)
	BatchGetDeviceConfigs(ctx context.Context, devices []DeviceInfo) (map[string]*DeviceConfig, map[string]error)

	// Generic API data fetching
	fetchAPIData(ctx context.Context, path string) (interface{}, error)
}

// Config holds API credentials and options
type Config struct {
	BaseURL      string
	APIToken     string
	Organization string
	Timeout      time.Duration
	Debug        bool
	RateLimit    int           // Requests per minute (0 = no limit)
	RateDuration time.Duration // Duration for rate limiting
	CacheTTL     time.Duration // Cache TTL (0 = default 5 minutes)
	ResultsLimit int           // Maximum results per API call
	HTTPClient   *http.Client  // Custom HTTP client
	MaxRetries   int           // Maximum number of retry attempts (0 = no retries)
	RetryBackoff time.Duration // Initial backoff duration for retries (0 = default 250ms)
	LocalCache   string        // Path to local cache file for ID mappings
	OrgID        string        // Organization ID for cache operations
	Inventory    string        // Path to inventory file
	DryRun       bool          // When true, don't make actual changes via API
}

// mistClient implements the Client interface for the Mist API
type mistClient struct {
	config             Config
	httpClient         *http.Client
	rateLimiter        *rateLimiter
	sitesCache         *cache[[]Site]
	apsCache           *cache[[]AP]
	deviceCache        *cache[[]Device] // Universal device cache for all device types
	inventoryCache     *cache[[]InventoryItem]
	deviceProfileCache *cache[[]DeviceProfile]
	cacheDirectory     string
	debug              bool
	dryRun             bool
	maxRetries         int
	retryBackoff       time.Duration
	mu                 sync.RWMutex
}

// Ensure mistClient implements the Client interface
var _ Client = (*mistClient)(nil)

// ClientOption defines functional options for the client
type ClientOption func(*mistClient)

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *mistClient) {
		c.httpClient = httpClient

		// Setup debug transport if needed
		setupDebugTransport(c)
	}
}

// WithDebug enables debug mode
func WithDebug(debug bool) ClientOption {
	return func(c *mistClient) {
		c.debug = debug
		c.config.Debug = debug

		// Setup debug transport if debug is enabled
		setupDebugTransport(c)
	}
}

// WithDryRun enables dry run mode
func WithDryRun(dryRun bool) ClientOption {
	return func(c *mistClient) {
		c.dryRun = dryRun
		c.config.DryRun = dryRun

		if dryRun {
			c.logDebug("Dry run mode enabled - no actual API changes will be made")
		}
	}
}

// WithRateLimit sets rate limiting
func WithRateLimit(limit int, duration time.Duration) ClientOption {
	return func(c *mistClient) {
		if limit > 0 {
			c.rateLimiter = newRateLimiter(limit, duration)
			c.config.RateLimit = limit
			c.config.RateDuration = duration
		} else {
			c.rateLimiter = nil
		}
	}
}

// WithCacheTTL sets the cache TTL
func WithCacheTTL(ttl time.Duration) ClientOption {
	return func(c *mistClient) {
		if ttl > 0 {
			c.sitesCache = newCache[[]Site](ttl)
			c.apsCache = newCache[[]AP](ttl)
			c.deviceCache = newCache[[]Device](ttl)
			c.inventoryCache = newCache[[]InventoryItem](ttl)
			c.deviceProfileCache = newCache[[]DeviceProfile](ttl)
		} else {
			// Default 5 minute TTL if none specified
			c.sitesCache = newCache[[]Site](5 * time.Minute)
			c.apsCache = newCache[[]AP](5 * time.Minute)
			c.deviceCache = newCache[[]Device](5 * time.Minute)
			c.inventoryCache = newCache[[]InventoryItem](5 * time.Minute)
			c.deviceProfileCache = newCache[[]DeviceProfile](5 * time.Minute)
		}
	}
}

// WithResultsLimit sets the results limit for pagination
func WithResultsLimit(limit int) ClientOption {
	return func(c *mistClient) {
		c.config.ResultsLimit = limit
	}
}

// WithCacheDirectory configures the cache directory path
func WithCacheDirectory(cacheDir string) ClientOption {
	return func(c *mistClient) {
		if cacheDir != "" {
			c.cacheDirectory = cacheDir
		}
	}
}

// WithInventory configures an inventory file path
func WithInventory(inventoryPath string) ClientOption {
	return func(c *mistClient) {
		if inventoryPath != "" {
			c.config.Inventory = inventoryPath
		}
	}
}

// NewClient creates a new client with the provided configuration
func NewClient(config Config) Client {
	// Initialize client instance with basic configs
	client := &mistClient{
		config:       config,
		debug:        config.Debug,
		dryRun:       config.DryRun,
		maxRetries:   config.MaxRetries,
		retryBackoff: config.RetryBackoff,
	}

	// Configure HTTP client
	if config.HTTPClient != nil {
		client.httpClient = config.HTTPClient
	} else {
		// Create a default HTTP client with reasonable timeout
		timeout := config.Timeout
		if timeout <= 0 {
			timeout = 30 * time.Second
		}
		client.httpClient = &http.Client{
			Timeout: timeout,
		}
	}

	// Configure debug transport if debug mode is enabled
	if config.Debug {
		setupDebugTransport(client)
	}

	// Configure rate limiter if specified
	if config.RateLimit > 0 {
		client.rateLimiter = newRateLimiter(config.RateLimit, config.RateDuration)
	}

	// Configure cache with TTL
	cacheTTL := config.CacheTTL
	if cacheTTL <= 0 {
		cacheTTL = 5 * time.Minute
	}
	client.sitesCache = newCache[[]Site](cacheTTL)
	client.apsCache = newCache[[]AP](cacheTTL)
	client.deviceCache = newCache[[]Device](cacheTTL)
	client.inventoryCache = newCache[[]InventoryItem](cacheTTL)
	client.deviceProfileCache = newCache[[]DeviceProfile](cacheTTL)

	// Legacy cache operations disabled - cache system modernized

	return client
}

// NewClientWithOptions creates a new client with the provided options
func NewClientWithOptions(apiToken, baseURL, orgID string, options ...ClientOption) Client {
	// Create client with basic configuration
	client := &mistClient{
		config: Config{
			BaseURL:      baseURL,
			APIToken:     apiToken,
			Organization: orgID,
			Timeout:      30 * time.Second,
		},
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		sitesCache:         newCache[[]Site](5 * time.Minute),
		apsCache:           newCache[[]AP](5 * time.Minute),
		deviceCache:        newCache[[]Device](5 * time.Minute),
		inventoryCache:     newCache[[]InventoryItem](5 * time.Minute),
		deviceProfileCache: newCache[[]DeviceProfile](5 * time.Minute),
		debug:              false,
		dryRun:             false,
		maxRetries:         3,
		retryBackoff:       250 * time.Millisecond,
	}

	// Apply options
	for _, option := range options {
		option(client)
	}

	return client
}

// These types and functions are defined in cache.go

// logDebug outputs debug information - respects the logging level configuration
func (c *mistClient) logDebug(format string, args ...interface{}) {
	// Always use logging.Debugf which will respect the configured logging level
	logging.Debugf(format, args...)
}

// Close releases resources used by the client, including stopping the rate limiter goroutine.
// This should be called when the client is no longer needed to prevent goroutine leaks.
func (c *mistClient) Close() error {
	if c.rateLimiter != nil {
		c.rateLimiter.Close()
	}
	return nil
}

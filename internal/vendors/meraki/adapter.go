// Package meraki provides a Meraki implementation of the vendors.Client interface.
package meraki

import (
	"fmt"
	"os"
	"time"

	meraki "github.com/meraki/dashboard-api-go/v5/sdk"

	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// noopLogger is a logger that discards all output.
// It implements resty.Logger interface.
type noopLogger struct{}

func (n *noopLogger) Errorf(_ string, _ ...interface{}) {}
func (n *noopLogger) Warnf(_ string, _ ...interface{})  {}
func (n *noopLogger) Debugf(_ string, _ ...interface{}) {}

// AdapterOption is a functional option for configuring the Adapter.
type AdapterOption func(*adapterOptions)

type adapterOptions struct {
	suppressOutput bool
}

// WithSuppressOutput enables stdout suppression during API calls.
// This is a workaround for debug print statements in the Meraki SDK.
// See: https://github.com/meraki/dashboard-api-go/issues/72
// See: https://github.com/meraki/dashboard-api-go/pull/75
func WithSuppressOutput(suppress bool) AdapterOption {
	return func(o *adapterOptions) {
		o.suppressOutput = suppress
	}
}

// Adapter implements vendors.Client for Meraki Dashboard API.
type Adapter struct {
	dashboard      *meraki.Client
	orgID          string
	rateLimiter    *RateLimiter
	retryConfig    *RetryConfig
	suppressOutput bool
}

// NewAdapter creates a new Meraki adapter.
// Note: baseURL should be the base URL without /api/v1 - the SDK adds that.
func NewAdapter(apiKey, baseURL, orgID string, opts ...AdapterOption) (vendors.Client, error) {
	// Apply options
	options := &adapterOptions{}
	for _, opt := range opts {
		opt(options)
	}

	if baseURL == "" {
		baseURL = "https://api.meraki.com"
	}

	logging.Debugf("[meraki] Creating Meraki client for org %s with base URL %s", orgID, baseURL)

	// Create Meraki SDK client
	dashboard, err := meraki.NewClientWithOptions(
		baseURL,
		apiKey,
		"false",   // debug
		"wifimgr", // user agent
	)
	if err != nil {
		logging.Debugf("[meraki] Failed to create Meraki client: %v", err)
		return nil, fmt.Errorf("failed to create Meraki client: %w", err)
	}

	// Disable SDK's internal retry mechanism - we have our own rate limiting and retry logic
	// This prevents the SDK from printing "MAX_RETRIES" debug output
	noRetries := 0
	noDelay := time.Duration(0)
	noJitter := time.Duration(0)
	useRetryHeader := false
	if err := dashboard.SetBackoff(&noRetries, &noDelay, &noJitter, &useRetryHeader); err != nil {
		logging.Debugf("[meraki] Failed to configure SDK backoff: %v", err)
	}

	// Suppress SDK's internal error logging - we handle errors ourselves with proper logging
	restyClient := dashboard.RestyClient()
	if restyClient != nil {
		restyClient.SetLogger(&noopLogger{})
	}

	// Create rate limiter: 10 req/sec with 10 burst capacity
	rateLimiter := NewRateLimiter(10, 10)
	retryConfig := DefaultRetryConfig()
	retryConfig.RateLimiter = rateLimiter

	if options.suppressOutput {
		logging.Debugf("[meraki] Stdout suppression enabled for SDK debug output")
	}

	logging.Debugf("[meraki] Successfully created Meraki client for org %s with rate limiting", orgID)
	return &Adapter{
		dashboard:      dashboard,
		orgID:          orgID,
		rateLimiter:    rateLimiter,
		retryConfig:    retryConfig,
		suppressOutput: options.suppressOutput,
	}, nil
}

// suppressStdout temporarily redirects stdout to /dev/null.
// Returns a function to restore stdout. This is a workaround for
// debug print statements in the Meraki SDK (issues #72 and #75).
func suppressStdout() func() {
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		return func() {} // Can't suppress, return no-op
	}
	oldStdout := os.Stdout
	os.Stdout = devNull
	return func() {
		os.Stdout = oldStdout
		_ = devNull.Close() // Ignore error - closing /dev/null
	}
}

// VendorName returns "meraki".
func (a *Adapter) VendorName() string {
	return "meraki"
}

// OrgID returns the organization ID.
func (a *Adapter) OrgID() string {
	return a.orgID
}

// Sites returns the SitesService for network operations.
func (a *Adapter) Sites() vendors.SitesService {
	return &sitesService{
		dashboard:      a.dashboard,
		orgID:          a.orgID,
		rateLimiter:    a.rateLimiter,
		retryConfig:    a.retryConfig,
		suppressOutput: a.suppressOutput,
	}
}

// Inventory returns the InventoryService for device inventory.
func (a *Adapter) Inventory() vendors.InventoryService {
	return &inventoryService{
		dashboard:      a.dashboard,
		orgID:          a.orgID,
		rateLimiter:    a.rateLimiter,
		retryConfig:    a.retryConfig,
		suppressOutput: a.suppressOutput,
	}
}

// Devices returns the DevicesService for device operations.
func (a *Adapter) Devices() vendors.DevicesService {
	return &devicesService{
		dashboard:      a.dashboard,
		orgID:          a.orgID,
		rateLimiter:    a.rateLimiter,
		retryConfig:    a.retryConfig,
		suppressOutput: a.suppressOutput,
	}
}

// Search returns the SearchService for client search operations.
func (a *Adapter) Search() vendors.SearchService {
	return &searchService{
		dashboard:      a.dashboard,
		orgID:          a.orgID,
		rateLimiter:    a.rateLimiter,
		retryConfig:    a.retryConfig,
		suppressOutput: a.suppressOutput,
	}
}

// Profiles returns nil - Meraki doesn't have device profiles like Mist.
func (a *Adapter) Profiles() vendors.ProfilesService {
	return nil
}

// Templates returns the TemplatesService for RF profiles.
// Meraki RF profiles are per-network, fetched via ListRF().
func (a *Adapter) Templates() vendors.TemplatesService {
	return &templatesService{
		dashboard:      a.dashboard,
		orgID:          a.orgID,
		rateLimiter:    a.rateLimiter,
		retryConfig:    a.retryConfig,
		suppressOutput: a.suppressOutput,
	}
}

// Configs returns the ConfigsService for device configuration.
func (a *Adapter) Configs() vendors.ConfigsService {
	return &configsService{
		dashboard:      a.dashboard,
		orgID:          a.orgID,
		rateLimiter:    a.rateLimiter,
		retryConfig:    a.retryConfig,
		suppressOutput: a.suppressOutput,
	}
}

// Statuses returns the StatusesService for device status information.
func (a *Adapter) Statuses() vendors.StatusesService {
	return &statusesService{
		dashboard:      a.dashboard,
		orgID:          a.orgID,
		rateLimiter:    a.rateLimiter,
		retryConfig:    a.retryConfig,
		suppressOutput: a.suppressOutput,
	}
}

// WLANs returns the WLANsService for SSID operations.
// Meraki SSIDs are per-network (numbered 0-14).
func (a *Adapter) WLANs() vendors.WLANsService {
	return &wlansService{
		dashboard:      a.dashboard,
		orgID:          a.orgID,
		rateLimiter:    a.rateLimiter,
		retryConfig:    a.retryConfig,
		suppressOutput: a.suppressOutput,
	}
}

// Ensure Adapter implements vendors.Client at compile time.
var _ vendors.Client = (*Adapter)(nil)

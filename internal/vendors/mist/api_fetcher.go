package mist

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/ravinald/wifimgr/api"
)

// MistAPIFetcher provides a reproducible pattern for fetching data from Mist API endpoints
type MistAPIFetcher struct {
	client api.Client
	orgID  string
}

// NewMistAPIFetcher creates a new Mist API fetcher
func NewMistAPIFetcher(client api.Client, orgID string) *MistAPIFetcher {
	return &MistAPIFetcher{
		client: client,
		orgID:  orgID,
	}
}

// EndpointConfig defines the configuration for each API endpoint
type EndpointConfig struct {
	Path         string
	Method       string
	RequiresSite bool
}

// FetchResult contains the fetched data and metadata
type FetchResult struct {
	Data     interface{}
	Count    int
	Endpoint string
}

// GetEndpointConfigs returns the configuration for all supported Mist API endpoints
func (m *MistAPIFetcher) GetEndpointConfigs() map[string]EndpointConfig {
	return map[string]EndpointConfig{
		"orgstats": {
			Path:         fmt.Sprintf("/orgs/%s/stats", m.orgID),
			Method:       http.MethodGet,
			RequiresSite: false,
		},
		"sites": {
			Path:         fmt.Sprintf("/orgs/%s/sites", m.orgID),
			Method:       http.MethodGet,
			RequiresSite: false,
		},
		"rftemplates": {
			Path:         fmt.Sprintf("/orgs/%s/rftemplates", m.orgID),
			Method:       http.MethodGet,
			RequiresSite: false,
		},
		"gatewaytemplates": {
			Path:         fmt.Sprintf("/orgs/%s/gatewaytemplates", m.orgID),
			Method:       http.MethodGet,
			RequiresSite: false,
		},
		"networks": {
			Path:         fmt.Sprintf("/orgs/%s/networks", m.orgID),
			Method:       http.MethodGet,
			RequiresSite: false,
		},
		"wlans": {
			Path:         fmt.Sprintf("/orgs/%s/wlans", m.orgID),
			Method:       http.MethodGet,
			RequiresSite: false,
		},
		"templates": {
			Path:         fmt.Sprintf("/orgs/%s/templates", m.orgID),
			Method:       http.MethodGet,
			RequiresSite: false,
		},
		"deviceprofiles": {
			Path:         fmt.Sprintf("/orgs/%s/deviceprofiles", m.orgID),
			Method:       http.MethodGet,
			RequiresSite: false,
		},
		"inventory-ap": {
			Path:         fmt.Sprintf("/orgs/%s/inventory?type=ap", m.orgID),
			Method:       http.MethodGet,
			RequiresSite: false,
		},
		"inventory-switch": {
			Path:         fmt.Sprintf("/orgs/%s/inventory?type=switch", m.orgID),
			Method:       http.MethodGet,
			RequiresSite: false,
		},
		"inventory-gateway": {
			Path:         fmt.Sprintf("/orgs/%s/inventory?type=gateway", m.orgID),
			Method:       http.MethodGet,
			RequiresSite: false,
		},
	}
}

// GetSiteEndpointConfigs returns site-specific endpoint configurations
func (m *MistAPIFetcher) GetSiteEndpointConfigs(siteID string) map[string]EndpointConfig {
	return map[string]EndpointConfig{
		"site-wlans": {
			Path:         fmt.Sprintf("/sites/%s/wlans", siteID),
			Method:       http.MethodGet,
			RequiresSite: true,
		},
	}
}

// FetchEndpointData fetches data from a Mist API endpoint using a reproducible pattern
func (m *MistAPIFetcher) FetchEndpointData(ctx context.Context, endpointName string, siteID string) (*FetchResult, error) {
	var config EndpointConfig
	var exists bool

	// First check org-level endpoints
	configs := m.GetEndpointConfigs()
	if config, exists = configs[endpointName]; !exists {
		// Then check site-specific endpoints if siteID provided
		if siteID != "" {
			siteConfigs := m.GetSiteEndpointConfigs(siteID)
			if config, exists = siteConfigs[endpointName]; !exists {
				return nil, fmt.Errorf("unknown endpoint: %s", endpointName)
			}
		} else {
			return nil, fmt.Errorf("unknown endpoint: %s", endpointName)
		}
	}

	// Fetch the data using the client's HTTP interface
	var result interface{}
	if err := m.fetchRawData(ctx, config.Method, config.Path, &result); err != nil {
		return nil, fmt.Errorf("failed to fetch data from %s: %w", config.Path, err)
	}

	// Count items if result is a slice
	count := 0
	if slice, ok := result.([]interface{}); ok {
		count = len(slice)
	}

	return &FetchResult{
		Data:     result,
		Count:    count,
		Endpoint: config.Path,
	}, nil
}

// fetchRawData performs the actual HTTP request to fetch data
func (m *MistAPIFetcher) fetchRawData(ctx context.Context, method, path string, result interface{}) error {
	// Use the client's HTTP capabilities to make the request
	// This method assumes the client has a generic HTTP method available
	// If not, we'll need to adapt based on the actual client interface

	// For now, we'll use reflection or type assertion to access the client's HTTP method
	if httpClient, ok := m.client.(interface {
		Do(ctx context.Context, method, path string, body interface{}, result interface{}) error
	}); ok {
		return httpClient.Do(ctx, method, path, nil, result)
	}

	return fmt.Errorf("client does not support direct HTTP requests")
}

// RefreshDataType refreshes a specific data type using existing client methods
func (m *MistAPIFetcher) RefreshDataType(ctx context.Context, dataType string) (*FetchResult, error) {
	switch dataType {
	case "orgstats":
		orgStats, err := m.client.GetOrgStats(ctx, m.orgID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch org stats: %w", err)
		}
		return &FetchResult{
			Data:     orgStats,
			Count:    1,
			Endpoint: fmt.Sprintf("/orgs/%s/stats", m.orgID),
		}, nil

	case "sites":
		sites, err := m.client.GetSites(ctx, m.orgID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch sites: %w", err)
		}
		return &FetchResult{
			Data:     sites,
			Count:    len(sites),
			Endpoint: fmt.Sprintf("/orgs/%s/sites", m.orgID),
		}, nil

	case "inventory-ap":
		// Clear inventory cache to force fresh API fetch
		api.ClearCache("inventory-ap")
		inventory, err := m.client.GetInventory(ctx, m.orgID, "ap")
		if err != nil {
			return nil, fmt.Errorf("failed to fetch AP inventory: %w", err)
		}
		return &FetchResult{
			Data:     inventory,
			Count:    len(inventory),
			Endpoint: fmt.Sprintf("/orgs/%s/inventory?type=ap", m.orgID),
		}, nil

	case "inventory-switch":
		// Clear inventory cache to force fresh API fetch
		api.ClearCache("inventory-switch")
		inventory, err := m.client.GetInventory(ctx, m.orgID, "switch")
		if err != nil {
			return nil, fmt.Errorf("failed to fetch switch inventory: %w", err)
		}
		return &FetchResult{
			Data:     inventory,
			Count:    len(inventory),
			Endpoint: fmt.Sprintf("/orgs/%s/inventory?type=switch", m.orgID),
		}, nil

	case "inventory-gateway":
		// Clear inventory cache to force fresh API fetch
		api.ClearCache("inventory-gateway")
		inventory, err := m.client.GetInventory(ctx, m.orgID, "gateway")
		if err != nil {
			return nil, fmt.Errorf("failed to fetch gateway inventory: %w", err)
		}
		return &FetchResult{
			Data:     inventory,
			Count:    len(inventory),
			Endpoint: fmt.Sprintf("/orgs/%s/inventory?type=gateway", m.orgID),
		}, nil

	case "deviceprofiles":
		profiles, err := m.client.GetDeviceProfiles(ctx, m.orgID, "")
		if err != nil {
			return nil, fmt.Errorf("failed to fetch device profiles: %w", err)
		}
		return &FetchResult{
			Data:     profiles,
			Count:    len(profiles),
			Endpoint: fmt.Sprintf("/orgs/%s/deviceprofiles", m.orgID),
		}, nil

	case "rftemplates":
		templates, err := m.client.GetRFTemplates(ctx, m.orgID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch RF templates: %w", err)
		}
		return &FetchResult{
			Data:     templates,
			Count:    len(templates),
			Endpoint: fmt.Sprintf("/orgs/%s/rftemplates", m.orgID),
		}, nil

	case "gatewaytemplates":
		templates, err := m.client.GetGatewayTemplates(ctx, m.orgID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch gateway templates: %w", err)
		}
		return &FetchResult{
			Data:     templates,
			Count:    len(templates),
			Endpoint: fmt.Sprintf("/orgs/%s/gatewaytemplates", m.orgID),
		}, nil

	case "wlantemplates":
		templates, err := m.client.GetWLANTemplates(ctx, m.orgID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch WLAN templates: %w", err)
		}
		return &FetchResult{
			Data:     templates,
			Count:    len(templates),
			Endpoint: fmt.Sprintf("/orgs/%s/templates", m.orgID),
		}, nil

	case "networks":
		networks, err := m.client.GetNetworks(ctx, m.orgID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch networks: %w", err)
		}
		return &FetchResult{
			Data:     networks,
			Count:    len(networks),
			Endpoint: fmt.Sprintf("/orgs/%s/networks", m.orgID),
		}, nil

	case "wlans":
		wlans, err := m.client.GetWLANs(ctx, m.orgID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch WLANs: %w", err)
		}
		return &FetchResult{
			Data:     wlans,
			Count:    len(wlans),
			Endpoint: fmt.Sprintf("/orgs/%s/wlans", m.orgID),
		}, nil

	case "sitesettings":
		// For site settings, we need to iterate through all sites and get their settings
		sites, err := m.client.GetSites(ctx, m.orgID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch sites for settings refresh: %w", err)
		}

		var allSettings []*api.SiteSetting
		for _, site := range sites {
			if site.ID != nil {
				setting, err := m.client.GetSiteSetting(ctx, *site.ID)
				if err != nil {
					// Log warning but continue with other sites
					fmt.Printf("Warning: failed to fetch settings for site %s: %v\n", *site.ID, err)
					continue
				}
				if setting != nil {
					allSettings = append(allSettings, setting)
				}
			}
		}

		return &FetchResult{
			Data:     allSettings,
			Count:    len(allSettings),
			Endpoint: "multiple: /sites/{id}/setting",
		}, nil

	case "deviceprofiledetails":
		// Fetch detailed device profile information for each profile
		return m.fetchDeviceProfileDetails(ctx)

	case "deviceconfigs":
		// Fetch device configurations for all devices with non-null site_id
		return m.fetchDeviceConfigs(ctx)

	default:
		return nil, fmt.Errorf("unsupported data type: %s", dataType)
	}
}

// GetSupportedDataTypes returns the list of data types that can be refreshed
func (m *MistAPIFetcher) GetSupportedDataTypes() []string {
	return []string{
		"orgstats",
		"sites",
		"sitesettings",
		"inventory-ap",
		"inventory-switch",
		"inventory-gateway",
		"deviceprofiles",
		"deviceprofiledetails",
		"rftemplates",
		"gatewaytemplates",
		"wlantemplates",
		"networks",
		"wlans",
		"deviceconfigs",
	}
}

// fetchDeviceConfigs fetches configurations for all devices with non-null site_id
func (m *MistAPIFetcher) fetchDeviceConfigs(ctx context.Context) (*FetchResult, error) {
	// Clear the device configs cache to force fresh API fetch (not inventory)
	api.ClearCache("configs")

	// Group devices by site to use bulk fetch per site
	siteDevices := make(map[string]bool) // Track unique sites with devices

	// Check AP inventory for sites with devices
	apInventory, err := m.client.GetInventory(ctx, m.orgID, "ap")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch AP inventory for configs: %w", err)
	}
	for _, item := range apInventory {
		if item.SiteID != nil && *item.SiteID != "" {
			siteDevices[*item.SiteID] = true
		}
	}

	// Check Switch inventory for sites with devices
	switchInventory, err := m.client.GetInventory(ctx, m.orgID, "switch")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch switch inventory for configs: %w", err)
	}
	for _, item := range switchInventory {
		if item.SiteID != nil && *item.SiteID != "" {
			siteDevices[*item.SiteID] = true
		}
	}

	// Check Gateway inventory for sites with devices
	gatewayInventory, err := m.client.GetInventory(ctx, m.orgID, "gateway")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch gateway inventory for configs: %w", err)
	}
	for _, item := range gatewayInventory {
		if item.SiteID != nil && *item.SiteID != "" {
			siteDevices[*item.SiteID] = true
		}
	}

	// Now fetch devices by type for each site to properly capture all fields
	var allConfigs []interface{}
	deviceTypes := []string{"ap", "switch", "gateway"}

	for siteID := range siteDevices {
		// Clear config cache for this site before fetching to ensure fresh data
		api.ClearCacheForSite(siteID, "configs")

		// Fetch each device type separately to properly capture device-specific fields
		for _, deviceType := range deviceTypes {
			// Use GetDevices with specific device type - this will call /sites/{site_id}/devices?type={deviceType}
			devices, err := m.client.GetDevices(ctx, siteID, deviceType)
			if err != nil {
				// Only log a warning if there's an actual error, not if there are simply no devices
				if !strings.Contains(err.Error(), "not found") {
					fmt.Printf("Warning: failed to fetch %s devices for site %s: %v\n", deviceType, siteID, err)
				}
				continue
			}

			// Convert UnifiedDevice objects to proper config types with all fields
			for _, device := range devices {
				var config *api.DeviceConfig

				// Create the appropriate config type with ALL fields from the UnifiedDevice
				switch deviceType {
				case "ap":
					apConfig := &api.APConfig{
						BaseDevice: device.BaseDevice,
					}
					// The UnifiedDevice's DeviceConfig map contains all additional fields
					if device.DeviceConfig != nil {
						apConfig.AdditionalConfig = device.DeviceConfig
					}
					config = &api.DeviceConfig{
						Type: "ap",
						Data: apConfig,
					}
				case "switch":
					switchConfig := &api.SwitchConfig{
						BaseDevice: device.BaseDevice,
					}
					// The UnifiedDevice's DeviceConfig map contains all additional fields
					if device.DeviceConfig != nil {
						switchConfig.AdditionalConfig = device.DeviceConfig
					}
					config = &api.DeviceConfig{
						Type: "switch",
						Data: switchConfig,
					}
				case "gateway":
					gatewayConfig := &api.GatewayConfig{
						BaseDevice: device.BaseDevice,
					}
					// The UnifiedDevice's DeviceConfig map contains all additional fields
					if device.DeviceConfig != nil {
						gatewayConfig.AdditionalConfig = device.DeviceConfig
					}
					config = &api.DeviceConfig{
						Type: "gateway",
						Data: gatewayConfig,
					}
				}

				if config != nil {
					allConfigs = append(allConfigs, config)
				}
			}
		}
	}

	fmt.Printf("Fetched device configs for %d sites (using /sites/<site_id>/devices?type=<device_type> for each type)\n", len(siteDevices))

	return &FetchResult{
		Data:     allConfigs,
		Count:    len(allConfigs),
		Endpoint: "multiple: /sites/{site_id}/devices?type={ap|switch|gateway}",
	}, nil
}

// fetchDeviceProfileDetails fetches detailed information for each device profile
func (m *MistAPIFetcher) fetchDeviceProfileDetails(ctx context.Context) (*FetchResult, error) {
	// First fetch the list of device profiles
	profiles, err := m.client.GetDeviceProfiles(ctx, m.orgID, "")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch device profiles list: %w", err)
	}

	var allProfileDetails []map[string]interface{}
	failedProfiles := make(map[string]error)

	// For each profile, fetch its details
	for _, profile := range profiles {
		if profile.ID == nil {
			continue
		}

		// Fetch the detailed profile
		detailProfile, err := m.client.GetDeviceProfile(ctx, m.orgID, *profile.ID)
		if err != nil {
			profileName := "unknown"
			if profile.Name != nil {
				profileName = *profile.Name
			}
			failedProfiles[profileName] = err
			fmt.Printf("Warning: failed to fetch details for profile %s: %v\n", profileName, err)
			continue
		}

		// Convert the detailed profile to a map
		var profileMap map[string]interface{}

		// Check the type and convert accordingly
		if detailProfile.Type != nil {
			switch *detailProfile.Type {
			case "ap":
				// Create AP profile and convert
				apProfile := &api.DeviceProfileAP{}
				if err := apProfile.FromMap(detailProfile.ToMap()); err == nil {
					profileMap = apProfile.ToMap()
				}
			case "switch":
				// Create Switch profile and convert
				switchProfile := &api.DeviceProfileSwitch{}
				if err := switchProfile.FromMap(detailProfile.ToMap()); err == nil {
					profileMap = switchProfile.ToMap()
				}
			case "gateway":
				// Create Gateway profile and convert
				gatewayProfile := &api.DeviceProfileGateway{}
				if err := gatewayProfile.FromMap(detailProfile.ToMap()); err == nil {
					profileMap = gatewayProfile.ToMap()
				}
			default:
				profileMap = detailProfile.ToMap()
			}
		} else {
			profileMap = detailProfile.ToMap()
		}

		allProfileDetails = append(allProfileDetails, profileMap)
	}

	// Log summary
	fmt.Printf("Fetched details for %d device profiles", len(allProfileDetails))
	if len(failedProfiles) > 0 {
		fmt.Printf(" (%d failed)", len(failedProfiles))
	}
	fmt.Println()

	return &FetchResult{
		Data:     allProfileDetails,
		Count:    len(allProfileDetails),
		Endpoint: fmt.Sprintf("multiple: /orgs/%s/deviceprofiles/{id}", m.orgID),
	}, nil
}

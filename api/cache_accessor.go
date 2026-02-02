package api

import (
	"sync"

	"github.com/spf13/viper"

	"github.com/ravinald/wifimgr/internal/xdg"
)

var (
	globalCacheAccessor     CacheAccessor
	globalCacheAccessorOnce sync.Once
)

// GetGlobalCacheAccessor returns the global singleton cache accessor
func GetGlobalCacheAccessor() CacheAccessor {
	globalCacheAccessorOnce.Do(func() {
		cachePath := viper.GetString("files.cache")
		if cachePath == "" {
			// Derive from cache_dir if files.cache is not explicitly set
			cacheDir := viper.GetString("files.cache_dir")
			if cacheDir == "" {
				cacheDir = xdg.GetCacheDir()
			}
			cachePath = cacheDir + "/cache.json"
		}
		globalCacheAccessor = NewCacheAccessor(cachePath)

		// Cache initialized successfully
	})
	return globalCacheAccessor
}

// CacheAccessor provides O(1) lookup methods for all cached data
type CacheAccessor interface {
	// Organizations
	GetOrgByName(name string) (*OrgStats, error)
	GetOrgByID(id string) (*OrgStats, error)
	GetAllOrgs() ([]*OrgStats, error)

	// Sites
	GetSiteByName(name string) (*MistSite, error)
	GetSiteByID(id string) (*MistSite, error)
	GetAllSites() ([]*MistSite, error)

	// Site Settings
	GetSiteSettingBySiteID(siteID string) (*SiteSetting, error)
	GetSiteSettingByID(id string) (*SiteSetting, error)
	GetAllSiteSettings() ([]*SiteSetting, error)

	// Templates
	GetRFTemplateByName(name string) (*MistRFTemplate, error)
	GetRFTemplateByID(id string) (*MistRFTemplate, error)
	GetAllRFTemplates() ([]*MistRFTemplate, error)

	GetGWTemplateByName(name string) (*MistGatewayTemplate, error)
	GetGWTemplateByID(id string) (*MistGatewayTemplate, error)
	GetAllGWTemplates() ([]*MistGatewayTemplate, error)

	GetWLANTemplateByName(name string) (*MistWLANTemplate, error)
	GetWLANTemplateByID(id string) (*MistWLANTemplate, error)
	GetAllWLANTemplates() ([]*MistWLANTemplate, error)

	// Networks
	GetNetworkByName(name string) (*MistNetwork, error)
	GetNetworkByID(id string) (*MistNetwork, error)
	GetAllNetworks() ([]*MistNetwork, error)

	// WLANs
	GetOrgWLANByName(name string) (*MistWLAN, error)
	GetOrgWLANByID(id string) (*MistWLAN, error)
	GetAllOrgWLANs() ([]*MistWLAN, error)

	GetSiteWLANByName(siteID, name string) (*MistWLAN, error)
	GetSiteWLANByID(siteID, id string) (*MistWLAN, error)
	GetSiteWLANs(siteID string) ([]*MistWLAN, error)
	GetAllSiteWLANs() (map[string][]*MistWLAN, error)

	// Devices
	GetAPByMAC(mac string) (*APDevice, error)
	GetAPByName(name string) (*APDevice, error)
	GetAPsBySite(siteID string) ([]*APDevice, error)
	GetAllAPs() ([]*APDevice, error)

	GetSwitchByMAC(mac string) (*MistSwitchDevice, error)
	GetSwitchByName(name string) (*MistSwitchDevice, error)
	GetSwitchesBySite(siteID string) ([]*MistSwitchDevice, error)
	GetAllSwitches() ([]*MistSwitchDevice, error)

	GetGatewayByMAC(mac string) (*MistGatewayDevice, error)
	GetGatewayByName(name string) (*MistGatewayDevice, error)
	GetGatewaysBySite(siteID string) ([]*MistGatewayDevice, error)
	GetAllGateways() ([]*MistGatewayDevice, error)

	// Device Profiles
	GetDeviceProfileByName(name string) (*DeviceProfile, error)
	GetDeviceProfileByID(id string) (*DeviceProfile, error)
	GetDeviceProfilesByType(deviceType string) ([]*DeviceProfile, error)
	GetAllDeviceProfiles() ([]*DeviceProfile, error)

	// Device Profile Details (full details from individual API calls)
	GetDeviceProfileDetailByName(name string) (*map[string]any, error)
	GetDeviceProfileDetailByID(id string) (*map[string]any, error)
	GetAllDeviceProfileDetails() ([]*map[string]any, error)

	// Device Configurations
	GetAPConfigByMAC(mac string) (*APConfig, error)
	GetAPConfigByName(name string) (*APConfig, error)
	GetAllAPConfigs() ([]*APConfig, error)

	GetSwitchConfigByMAC(mac string) (*SwitchConfig, error)
	GetSwitchConfigByName(name string) (*SwitchConfig, error)
	GetAllSwitchConfigs() ([]*SwitchConfig, error)

	GetGatewayConfigByMAC(mac string) (*GatewayConfig, error)
	GetGatewayConfigByName(name string) (*GatewayConfig, error)
	GetAllGatewayConfigs() ([]*GatewayConfig, error)

	// Utility methods
	GetCacheStats() (map[string]any, error)
	IsInitialized() bool
	NeedsRebuild() bool
	GetManager() *CacheManager
}

// CacheAccessorImpl implements the CacheAccessor interface
type CacheAccessorImpl struct {
	manager *CacheManager
}

// NewCacheAccessor creates a new cache accessor with the given cache path
func NewCacheAccessor(cachePath string) CacheAccessor {
	manager := NewCacheManager(cachePath)
	// Initialize the cache manager to load cache data and build indexes
	// Note: We intentionally ignore the error here to maintain backward compatibility.
	// Accessor methods will handle the uninitialized state and return appropriate errors.
	_ = manager.Initialize()
	return &CacheAccessorImpl{
		manager: manager,
	}
}

// IsInitialized checks if the cache is initialized
func (ca *CacheAccessorImpl) IsInitialized() bool {
	return ca.manager.initialized
}

// NeedsRebuild checks if the cache needs to be rebuilt
func (ca *CacheAccessorImpl) NeedsRebuild() bool {
	// Since metadata has been removed from cache, always return false
	// The cache expiry is handled by the separate metadata file
	return false
}

// GetManager returns the underlying cache manager
func (ca *CacheAccessorImpl) GetManager() *CacheManager {
	return ca.manager
}

// GetCacheStats returns cache statistics
func (ca *CacheAccessorImpl) GetCacheStats() (map[string]any, error) {
	return ca.manager.GetCacheStats()
}

package api

import (
	"fmt"
	"strings"

	"github.com/ravinald/wifimgr/internal/macaddr"
)

// GetDeviceProfileByName retrieves a device profile by name with O(1) lookup
func (ca *CacheAccessorImpl) GetDeviceProfileByName(name string) (*DeviceProfile, error) {
	indexes, err := ca.manager.GetIndexes()
	if err != nil {
		return nil, err
	}

	profile, exists := indexes.DeviceProfilesByName[name]
	if !exists {
		return nil, fmt.Errorf("device profile not found: %s", name)
	}

	return profile, nil
}

// GetDeviceProfileByID retrieves a device profile by ID with O(1) lookup
func (ca *CacheAccessorImpl) GetDeviceProfileByID(id string) (*DeviceProfile, error) {
	indexes, err := ca.manager.GetIndexes()
	if err != nil {
		return nil, err
	}

	profile, exists := indexes.DeviceProfilesByID[id]
	if !exists {
		return nil, fmt.Errorf("device profile not found: %s", id)
	}

	return profile, nil
}

// GetDeviceProfilesByType retrieves device profiles filtered by type
func (ca *CacheAccessorImpl) GetDeviceProfilesByType(deviceType string) ([]*DeviceProfile, error) {
	cache, err := ca.manager.GetCache()
	if err != nil {
		return nil, err
	}

	var profiles []*DeviceProfile
	for _, orgData := range cache.Orgs {
		for i := range orgData.Profiles.Devices {
			profile := &orgData.Profiles.Devices[i]
			if profile.Type != nil && strings.EqualFold(*profile.Type, deviceType) {
				profiles = append(profiles, profile)
			}
		}
	}

	return profiles, nil
}

// GetAllDeviceProfiles returns all device profiles
func (ca *CacheAccessorImpl) GetAllDeviceProfiles() ([]*DeviceProfile, error) {
	cache, err := ca.manager.GetCache()
	if err != nil {
		return nil, err
	}

	var profiles []*DeviceProfile
	for _, orgData := range cache.Orgs {
		for i := range orgData.Profiles.Devices {
			profiles = append(profiles, &orgData.Profiles.Devices[i])
		}
	}

	return profiles, nil
}

// GetDeviceProfileDetailByName retrieves device profile details by name with O(1) lookup
func (ca *CacheAccessorImpl) GetDeviceProfileDetailByName(name string) (*map[string]any, error) {
	indexes, err := ca.manager.GetIndexes()
	if err != nil {
		return nil, err
	}

	detail, exists := indexes.DeviceProfileDetailsByName[name]
	if !exists {
		return nil, fmt.Errorf("device profile detail not found: %s", name)
	}

	return detail, nil
}

// GetDeviceProfileDetailByID retrieves device profile details by ID with O(1) lookup
func (ca *CacheAccessorImpl) GetDeviceProfileDetailByID(id string) (*map[string]any, error) {
	indexes, err := ca.manager.GetIndexes()
	if err != nil {
		return nil, err
	}

	detail, exists := indexes.DeviceProfileDetailsByID[id]
	if !exists {
		return nil, fmt.Errorf("device profile detail not found: %s", id)
	}

	return detail, nil
}

// GetAllDeviceProfileDetails returns all device profile details
func (ca *CacheAccessorImpl) GetAllDeviceProfileDetails() ([]*map[string]any, error) {
	cache, err := ca.manager.GetCache()
	if err != nil {
		return nil, err
	}

	var details []*map[string]any
	for _, orgData := range cache.Orgs {
		for i := range orgData.Profiles.Details {
			details = append(details, &orgData.Profiles.Details[i])
		}
	}

	return details, nil
}

// GetAPConfigByMAC retrieves an AP config by MAC address with O(1) lookup
func (ca *CacheAccessorImpl) GetAPConfigByMAC(mac string) (*APConfig, error) {
	indexes, err := ca.manager.GetIndexes()
	if err != nil {
		return nil, err
	}

	normalizedMAC := macaddr.NormalizeFast(mac)
	config, exists := indexes.APConfigsByMAC[normalizedMAC]
	if !exists {
		return nil, fmt.Errorf("AP config not found: %s", mac)
	}

	return config, nil
}

// GetAPConfigByName retrieves an AP config by name with O(1) lookup
func (ca *CacheAccessorImpl) GetAPConfigByName(name string) (*APConfig, error) {
	indexes, err := ca.manager.GetIndexes()
	if err != nil {
		return nil, err
	}

	config, exists := indexes.APConfigsByName[name]
	if !exists {
		return nil, fmt.Errorf("AP config not found: %s", name)
	}

	return config, nil
}

// GetAllAPConfigs returns all AP configurations
func (ca *CacheAccessorImpl) GetAllAPConfigs() ([]*APConfig, error) {
	cache, err := ca.manager.GetCache()
	if err != nil {
		return nil, err
	}

	var configs []*APConfig
	for _, orgData := range cache.Orgs {
		for _, config := range orgData.Configs.AP {
			configCopy := config
			configs = append(configs, &configCopy)
		}
	}

	return configs, nil
}

// GetSwitchConfigByMAC retrieves a switch config by MAC address with O(1) lookup
func (ca *CacheAccessorImpl) GetSwitchConfigByMAC(mac string) (*SwitchConfig, error) {
	indexes, err := ca.manager.GetIndexes()
	if err != nil {
		return nil, err
	}

	normalizedMAC := macaddr.NormalizeFast(mac)
	config, exists := indexes.SwitchConfigsByMAC[normalizedMAC]
	if !exists {
		return nil, fmt.Errorf("switch config not found: %s", mac)
	}

	return config, nil
}

// GetSwitchConfigByName retrieves a switch config by name with O(1) lookup
func (ca *CacheAccessorImpl) GetSwitchConfigByName(name string) (*SwitchConfig, error) {
	indexes, err := ca.manager.GetIndexes()
	if err != nil {
		return nil, err
	}

	config, exists := indexes.SwitchConfigsByName[name]
	if !exists {
		return nil, fmt.Errorf("switch config not found: %s", name)
	}

	return config, nil
}

// GetAllSwitchConfigs returns all switch configurations
func (ca *CacheAccessorImpl) GetAllSwitchConfigs() ([]*SwitchConfig, error) {
	cache, err := ca.manager.GetCache()
	if err != nil {
		return nil, err
	}

	var configs []*SwitchConfig
	for _, orgData := range cache.Orgs {
		for _, config := range orgData.Configs.Switch {
			configCopy := config
			configs = append(configs, &configCopy)
		}
	}

	return configs, nil
}

// GetGatewayConfigByMAC retrieves a gateway config by MAC address with O(1) lookup
func (ca *CacheAccessorImpl) GetGatewayConfigByMAC(mac string) (*GatewayConfig, error) {
	indexes, err := ca.manager.GetIndexes()
	if err != nil {
		return nil, err
	}

	normalizedMAC := macaddr.NormalizeFast(mac)
	config, exists := indexes.GatewayConfigsByMAC[normalizedMAC]
	if !exists {
		return nil, fmt.Errorf("gateway config not found: %s", mac)
	}

	return config, nil
}

// GetGatewayConfigByName retrieves a gateway config by name with O(1) lookup
func (ca *CacheAccessorImpl) GetGatewayConfigByName(name string) (*GatewayConfig, error) {
	indexes, err := ca.manager.GetIndexes()
	if err != nil {
		return nil, err
	}

	config, exists := indexes.GatewayConfigsByName[name]
	if !exists {
		return nil, fmt.Errorf("gateway config not found: %s", name)
	}

	return config, nil
}

// GetAllGatewayConfigs returns all gateway configurations
func (ca *CacheAccessorImpl) GetAllGatewayConfigs() ([]*GatewayConfig, error) {
	cache, err := ca.manager.GetCache()
	if err != nil {
		return nil, err
	}

	var configs []*GatewayConfig
	for _, orgData := range cache.Orgs {
		for _, config := range orgData.Configs.Gateway {
			configCopy := config
			configs = append(configs, &configCopy)
		}
	}

	return configs, nil
}

package api

import (
	"fmt"
	"strings"

	"github.com/ravinald/wifimgr/internal/logging"
)

// GetConfigs retrieves configurations by type
func (uc *UnifiedCache) GetConfigs(orgID, deviceType string) ([]any, error) {
	orgData, err := uc.GetOrgData(orgID)
	if err != nil {
		return nil, err
	}

	var configs []any

	switch deviceType {
	case "ap":
		for _, config := range orgData.Configs.AP {
			configCopy := config
			configs = append(configs, &configCopy)
		}
	case "switch":
		for _, config := range orgData.Configs.Switch {
			configCopy := config
			configs = append(configs, &configCopy)
		}
	case "gateway":
		for _, config := range orgData.Configs.Gateway {
			configCopy := config
			configs = append(configs, &configCopy)
		}
	case "all":
		for _, config := range orgData.Configs.AP {
			configCopy := config
			configs = append(configs, &configCopy)
		}
		for _, config := range orgData.Configs.Switch {
			configCopy := config
			configs = append(configs, &configCopy)
		}
		for _, config := range orgData.Configs.Gateway {
			configCopy := config
			configs = append(configs, &configCopy)
		}
	default:
		return nil, fmt.Errorf("unknown device type: %s", deviceType)
	}

	return configs, nil
}

// UpdateConfigs replaces configurations
func (uc *UnifiedCache) UpdateConfigs(orgID, deviceType string, configs []any) error {
	orgData, err := uc.GetOrgData(orgID)
	if err != nil {
		return err
	}

	switch deviceType {
	case "ap":
		orgData.Configs.AP = make(map[string]APConfig)
		for _, cfg := range configs {
			if apConfig, ok := cfg.(*APConfig); ok {
				if apConfig.MAC != nil {
					normalizedMAC := strings.ToLower(strings.ReplaceAll(*apConfig.MAC, ":", ""))
					orgData.Configs.AP[normalizedMAC] = *apConfig
				}
			}
		}
	case "switch":
		orgData.Configs.Switch = make(map[string]SwitchConfig)
		for _, cfg := range configs {
			if swConfig, ok := cfg.(*SwitchConfig); ok {
				if swConfig.MAC != nil {
					normalizedMAC := strings.ToLower(strings.ReplaceAll(*swConfig.MAC, ":", ""))
					orgData.Configs.Switch[normalizedMAC] = *swConfig
				}
			}
		}
	case "gateway":
		orgData.Configs.Gateway = make(map[string]GatewayConfig)
		for _, cfg := range configs {
			if gwConfig, ok := cfg.(*GatewayConfig); ok {
				if gwConfig.MAC != nil {
					normalizedMAC := strings.ToLower(strings.ReplaceAll(*gwConfig.MAC, ":", ""))
					orgData.Configs.Gateway[normalizedMAC] = *gwConfig
				}
			}
		}
	default:
		return fmt.Errorf("unknown device type: %s", deviceType)
	}

	uc.dirty = true
	return nil
}

// MergeConfigs merges new configurations with existing ones
func (uc *UnifiedCache) MergeConfigs(orgID, deviceType string, configs []any) error {
	existing, err := uc.GetConfigs(orgID, deviceType)
	if err != nil {
		logging.Debugf("No existing configs to merge: %v", err)
		existing = []any{}
	}

	// Create a map of existing configs by MAC address
	existingMap := make(map[string]any)
	for _, cfg := range existing {
		mac := getConfigMAC(cfg)
		if mac != "" {
			existingMap[mac] = cfg
		}
	}

	// Merge new configs
	for _, cfg := range configs {
		mac := getConfigMAC(cfg)
		if mac != "" {
			existingMap[mac] = cfg
		}
	}

	// Convert map back to slice
	merged := make([]any, 0, len(existingMap))
	for _, cfg := range existingMap {
		merged = append(merged, cfg)
	}

	return uc.UpdateConfigs(orgID, deviceType, merged)
}

// GetProfiles retrieves profiles by type
func (uc *UnifiedCache) GetProfiles(profileType string) ([]any, error) {
	orgData, err := uc.GetOrgData(uc.orgID)
	if err != nil {
		return nil, err
	}

	var profiles []any

	switch profileType {
	case "devices":
		for i := range orgData.Profiles.Devices {
			profiles = append(profiles, &orgData.Profiles.Devices[i])
		}
	case "details":
		for i := range orgData.Profiles.Details {
			profiles = append(profiles, orgData.Profiles.Details[i])
		}
	default:
		return nil, fmt.Errorf("unknown profile type: %s", profileType)
	}

	return profiles, nil
}

// UpdateProfiles updates profiles
func (uc *UnifiedCache) UpdateProfiles(profileType string, profiles []any) error {
	orgData, err := uc.GetOrgData(uc.orgID)
	if err != nil {
		return err
	}

	switch profileType {
	case "devices":
		orgData.Profiles.Devices = make([]DeviceProfile, 0, len(profiles))
		for _, p := range profiles {
			if profile, ok := p.(*DeviceProfile); ok {
				orgData.Profiles.Devices = append(orgData.Profiles.Devices, *profile)
			}
		}
	case "details":
		orgData.Profiles.Details = make([]map[string]any, 0, len(profiles))
		for _, p := range profiles {
			if detail, ok := p.(map[string]any); ok {
				orgData.Profiles.Details = append(orgData.Profiles.Details, detail)
			}
		}
	default:
		return fmt.Errorf("unknown profile type: %s", profileType)
	}

	uc.dirty = true
	return nil
}

// getConfigMAC extracts the MAC address from a config interface
func getConfigMAC(cfg any) string {
	switch c := cfg.(type) {
	case *APConfig:
		if c.MAC != nil {
			return strings.ToLower(strings.ReplaceAll(*c.MAC, ":", ""))
		}
	case *SwitchConfig:
		if c.MAC != nil {
			return strings.ToLower(strings.ReplaceAll(*c.MAC, ":", ""))
		}
	case *GatewayConfig:
		if c.MAC != nil {
			return strings.ToLower(strings.ReplaceAll(*c.MAC, ":", ""))
		}
	}
	return ""
}

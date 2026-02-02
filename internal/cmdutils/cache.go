package cmdutils

import (
	"fmt"
	"strings"

	"github.com/ravinald/wifimgr/internal/formatter"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// GetCacheAccessor returns the cache accessor for commands (delegates to vendors package singleton)
func GetCacheAccessor() (*vendors.CacheAccessor, error) {
	accessor := vendors.GetGlobalCacheAccessor()

	if accessor == nil {
		return nil, fmt.Errorf("cache accessor not initialized")
	}

	// Check if initialization succeeded
	if !accessor.IsInitialized() {
		return nil, fmt.Errorf("cache failed to initialize")
	}

	return accessor, nil
}

// CacheTableAccessor implements formatter.CacheAccessor for table formatting
// It provides access to full device config data and supports nested field lookups.
type CacheTableAccessor struct {
	cacheAccessor *vendors.CacheAccessor
}

// NewCacheTableAccessor creates a table-compatible cache accessor
func NewCacheTableAccessor() (formatter.CacheAccessor, error) {
	accessor, err := GetCacheAccessor()
	if err != nil {
		return nil, err
	}

	return &CacheTableAccessor{
		cacheAccessor: accessor,
	}, nil
}

// GetCachedData retrieves full cached config data by MAC address.
// This returns the complete device configuration including all vendor-specific fields.
func (c *CacheTableAccessor) GetCachedData(index string) (map[string]interface{}, bool) {
	normalizedMAC := vendors.NormalizeMAC(index)

	// Try AP config first (most common)
	if apCfg, err := c.cacheAccessor.GetAPConfigByMAC(normalizedMAC); err == nil && apCfg != nil {
		return c.apConfigToFullMap(apCfg), true
	}

	// Try Switch config
	if swCfg, err := c.cacheAccessor.GetSwitchConfigByMAC(normalizedMAC); err == nil && swCfg != nil {
		return c.switchConfigToFullMap(swCfg), true
	}

	// Try Gateway config
	if gwCfg, err := c.cacheAccessor.GetGatewayConfigByMAC(normalizedMAC); err == nil && gwCfg != nil {
		return c.gatewayConfigToFullMap(gwCfg), true
	}

	// Fallback to inventory item (limited fields)
	device, err := c.cacheAccessor.GetDeviceByMAC(index)
	if err != nil || device == nil {
		return nil, false
	}
	return c.inventoryItemToMap(device), true
}

// GetSiteName provides site name lookup capability
func (c *CacheTableAccessor) GetSiteName(siteID string) (string, bool) {
	site, err := c.cacheAccessor.GetSiteByID(siteID)
	if err != nil || site == nil {
		return "", false
	}
	if site.Name != "" {
		return site.Name, true
	}
	return "", false
}

// GetFieldByPath retrieves a nested field value by dot-separated path.
// This supports paths like "radio_config.band_5.channel" or "ip_config.vlan_id".
func (c *CacheTableAccessor) GetFieldByPath(data map[string]interface{}, path string) (interface{}, bool) {
	return GetNestedValue(data, path)
}

// ResolveID resolves an ID to a human-readable name based on the field type.
// Supports: deviceprofile_id, site_id, map_id, rf_template_id, etc.
func (c *CacheTableAccessor) ResolveID(fieldName string, id string) (string, bool) {
	if id == "" {
		return "", false
	}

	switch fieldName {
	case "deviceprofile_id":
		if profile, err := c.cacheAccessor.GetDeviceProfileByID(id); err == nil {
			return profile.Name, true
		}
	case "site_id":
		if site, err := c.cacheAccessor.GetSiteByID(id); err == nil {
			return site.Name, true
		}
	case "rf_template_id":
		if template, err := c.cacheAccessor.GetRFTemplateByID(id); err == nil {
			return template.Name, true
		}
	}

	return "", false
}

// apConfigToFullMap converts an APConfig to a complete map including all config fields.
func (c *CacheTableAccessor) apConfigToFullMap(cfg *vendors.APConfig) map[string]interface{} {
	result := make(map[string]interface{})

	// Base fields
	result["id"] = cfg.ID
	result["name"] = cfg.Name
	result["mac"] = cfg.MAC
	result["site_id"] = cfg.SiteID
	result["source_api"] = cfg.SourceAPI
	result["source_vendor"] = cfg.SourceVendor
	result["device_type"] = "ap"

	// Resolve site name
	if siteName, ok := c.GetSiteName(cfg.SiteID); ok {
		result["site_name"] = siteName
	}

	// Include all fields from the full config map
	if cfg.Config != nil {
		for k, v := range cfg.Config {
			// Don't overwrite base fields
			if _, exists := result[k]; !exists {
				result[k] = v
			}
		}

		// Also store raw config for nested lookups
		result["config"] = cfg.Config
	}

	// Resolve deviceprofile_id to name if present
	if dpID, ok := cfg.Config["deviceprofile_id"].(string); ok && dpID != "" {
		if dpName, resolved := c.ResolveID("deviceprofile_id", dpID); resolved {
			result["deviceprofile_name"] = dpName
		}
	}

	return result
}

// switchConfigToFullMap converts a SwitchConfig to a complete map.
func (c *CacheTableAccessor) switchConfigToFullMap(cfg *vendors.SwitchConfig) map[string]interface{} {
	result := make(map[string]interface{})

	result["id"] = cfg.ID
	result["name"] = cfg.Name
	result["mac"] = cfg.MAC
	result["site_id"] = cfg.SiteID
	result["source_api"] = cfg.SourceAPI
	result["source_vendor"] = cfg.SourceVendor
	result["device_type"] = "switch"

	if siteName, ok := c.GetSiteName(cfg.SiteID); ok {
		result["site_name"] = siteName
	}

	if cfg.Config != nil {
		for k, v := range cfg.Config {
			if _, exists := result[k]; !exists {
				result[k] = v
			}
		}
		result["config"] = cfg.Config
	}

	if dpID, ok := cfg.Config["deviceprofile_id"].(string); ok && dpID != "" {
		if dpName, resolved := c.ResolveID("deviceprofile_id", dpID); resolved {
			result["deviceprofile_name"] = dpName
		}
	}

	return result
}

// gatewayConfigToFullMap converts a GatewayConfig to a complete map.
func (c *CacheTableAccessor) gatewayConfigToFullMap(cfg *vendors.GatewayConfig) map[string]interface{} {
	result := make(map[string]interface{})

	result["id"] = cfg.ID
	result["name"] = cfg.Name
	result["mac"] = cfg.MAC
	result["site_id"] = cfg.SiteID
	result["source_api"] = cfg.SourceAPI
	result["source_vendor"] = cfg.SourceVendor
	result["device_type"] = "gateway"

	if siteName, ok := c.GetSiteName(cfg.SiteID); ok {
		result["site_name"] = siteName
	}

	if cfg.Config != nil {
		for k, v := range cfg.Config {
			if _, exists := result[k]; !exists {
				result[k] = v
			}
		}
		result["config"] = cfg.Config
	}

	if dpID, ok := cfg.Config["deviceprofile_id"].(string); ok && dpID != "" {
		if dpName, resolved := c.ResolveID("deviceprofile_id", dpID); resolved {
			result["deviceprofile_name"] = dpName
		}
	}

	return result
}

// inventoryItemToMap converts a vendors.InventoryItem to a map (fallback with limited fields).
func (c *CacheTableAccessor) inventoryItemToMap(device *vendors.InventoryItem) map[string]interface{} {
	result := make(map[string]interface{})

	if device.ID != "" {
		result["id"] = device.ID
	}
	if device.Name != "" {
		result["name"] = device.Name
	}
	if device.MAC != "" {
		result["mac"] = device.MAC
	}
	if device.Serial != "" {
		result["serial"] = device.Serial
	}
	if device.Model != "" {
		result["model"] = device.Model
	}
	if device.Type != "" {
		result["device_type"] = device.Type
	}
	if device.SiteID != "" {
		result["site_id"] = device.SiteID
		if siteName, ok := c.GetSiteName(device.SiteID); ok {
			result["site_name"] = siteName
		}
	}
	if device.SourceAPI != "" {
		result["source_api"] = device.SourceAPI
	}
	if device.SourceVendor != "" {
		result["source_vendor"] = device.SourceVendor
	}

	return result
}

// GetNestedValue retrieves a value from a nested map using dot-separated path.
// Supports paths like "radio_config.band_5.channel" or "ip_config.vlan_id".
func GetNestedValue(data map[string]interface{}, path string) (interface{}, bool) {
	if data == nil || path == "" {
		return nil, false
	}

	parts := strings.Split(path, ".")
	current := interface{}(data)

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			val, ok := v[part]
			if !ok {
				return nil, false
			}
			current = val
		default:
			return nil, false
		}
	}

	return current, true
}

// FormatNestedValue formats a nested value for display.
// Handles special cases like arrays, maps, and nil values.
func FormatNestedValue(value interface{}) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case float64:
		// Check if it's a whole number
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%.2f", v)
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case []interface{}:
		// Format arrays as comma-separated values
		parts := make([]string, len(v))
		for i, item := range v {
			parts[i] = FormatNestedValue(item)
		}
		return strings.Join(parts, ", ")
	case map[string]interface{}:
		// For nested objects, return a summary
		return fmt.Sprintf("{%d fields}", len(v))
	default:
		return fmt.Sprintf("%v", v)
	}
}

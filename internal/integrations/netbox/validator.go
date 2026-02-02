package netbox

import (
	"context"
	"fmt"
	"strings"

	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/macaddr"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// Validator checks that NetBox dependencies exist before export operations
type Validator struct {
	client *Client
	config *Config
	cache  *LookupCache
}

// LookupCache holds pre-fetched NetBox object IDs for fast lookup
type LookupCache struct {
	SitesByName                    map[string]int64               // site name (lowercase) -> netbox site ID
	SitesBySlug                    map[string]int64               // site slug -> netbox site ID
	DeviceTypesBySlug              map[string]int64               // device type slug -> netbox device type ID
	DeviceRolesBySlug              map[string]int64               // device role slug -> netbox device role ID
	DevicesByMAC                   map[string]int64               // normalized MAC -> netbox device ID
	WirelessLANsBySSID             map[string]int64               // SSID -> WirelessLAN ID
	InterfaceTemplatesByDeviceType map[int64][]*InterfaceTemplate // device type ID -> interface templates
}

// NewValidator creates a new validator
func NewValidator(client *Client, config *Config) *Validator {
	return &Validator{
		client: client,
		config: config,
		cache:  nil,
	}
}

// Initialize fetches all NetBox lookups into cache
func (v *Validator) Initialize(ctx context.Context) error {
	logging.Info("Initializing NetBox validator - fetching lookup data...")

	v.cache = &LookupCache{
		SitesByName:                    make(map[string]int64),
		SitesBySlug:                    make(map[string]int64),
		DeviceTypesBySlug:              make(map[string]int64),
		DeviceRolesBySlug:              make(map[string]int64),
		DevicesByMAC:                   make(map[string]int64),
		WirelessLANsBySSID:             make(map[string]int64),
		InterfaceTemplatesByDeviceType: make(map[int64][]*InterfaceTemplate),
	}

	// Fetch all sites
	sites, err := v.client.GetAllSites(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch sites: %w", err)
	}
	for _, site := range sites {
		v.cache.SitesByName[strings.ToLower(site.Name)] = site.ID
		v.cache.SitesBySlug[site.Slug] = site.ID
	}
	logging.Debugf("Loaded %d sites from NetBox", len(sites))

	// Fetch all device types
	deviceTypes, err := v.client.GetAllDeviceTypes(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch device types: %w", err)
	}
	for _, dt := range deviceTypes {
		v.cache.DeviceTypesBySlug[dt.Slug] = dt.ID
	}
	logging.Debugf("Loaded %d device types from NetBox", len(deviceTypes))

	// Fetch all device roles
	roles, err := v.client.GetAllDeviceRoles(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch device roles: %w", err)
	}
	for _, role := range roles {
		v.cache.DeviceRolesBySlug[role.Slug] = role.ID
	}
	logging.Debugf("Loaded %d device roles from NetBox", len(roles))

	// Fetch all WirelessLANs (for radio interface linking)
	wirelessLANs, err := v.client.GetAllWirelessLANs(ctx)
	if err != nil {
		// WirelessLAN module may not be enabled, log warning but continue
		logging.Warnf("Failed to fetch wireless LANs (module may not be enabled): %v", err)
	} else {
		for _, wlan := range wirelessLANs {
			v.cache.WirelessLANsBySSID[wlan.SSID] = wlan.ID
		}
		logging.Debugf("Loaded %d wireless LANs from NetBox", len(wirelessLANs))
	}

	logging.Info("NetBox validator initialized successfully")
	return nil
}

// ValidateSite checks if a site exists in NetBox and returns its ID
func (v *Validator) ValidateSite(siteName string) (int64, error) {
	if v.cache == nil {
		return 0, fmt.Errorf("validator not initialized - call Initialize() first")
	}

	// Try exact name match first (case-insensitive)
	if id, ok := v.cache.SitesByName[strings.ToLower(siteName)]; ok {
		return id, nil
	}

	// Try slug match
	slug := v.config.GetSiteSlug(siteName)
	if id, ok := v.cache.SitesBySlug[slug]; ok {
		return id, nil
	}

	return 0, fmt.Errorf("site '%s' not found in NetBox (tried name and slug '%s')", siteName, slug)
}

// ValidateDeviceType checks if a device type exists in NetBox and returns its ID
func (v *Validator) ValidateDeviceType(model string) (int64, error) {
	if v.cache == nil {
		return 0, fmt.Errorf("validator not initialized")
	}

	// Get the slug for this model from config mappings
	slug := v.config.GetDeviceTypeSlug(model)

	if id, ok := v.cache.DeviceTypesBySlug[slug]; ok {
		return id, nil
	}

	return 0, fmt.Errorf("device type '%s' (slug '%s') not found in NetBox", model, slug)
}

// ValidateDeviceRole checks if a device role exists in NetBox and returns its ID.
// It uses the model parameter to check for model-specific role overrides.
// The deviceNetBox parameter can be nil or a *NetBoxDeviceExtension for per-device role overrides.
func (v *Validator) ValidateDeviceRole(deviceType string, model string, deviceNetBox any) (int64, error) {
	if v.cache == nil {
		return 0, fmt.Errorf("validator not initialized")
	}

	// Get the role slug for this device type from config mappings
	// This now supports per-device and model-specific role overrides
	slug := v.config.GetDeviceRoleSlugForModel(deviceType, model, deviceNetBox)

	if id, ok := v.cache.DeviceRolesBySlug[slug]; ok {
		return id, nil
	}

	return 0, fmt.Errorf("device role '%s' (for device type '%s', model '%s') not found in NetBox", slug, deviceType, model)
}

// DeviceValidationResult contains the validation result for a single device
type DeviceValidationResult struct {
	Valid        bool
	SiteID       int64
	DeviceTypeID int64
	DeviceRoleID int64
	Errors       []string
}

// ValidateDevice validates all dependencies for a device
func (v *Validator) ValidateDevice(item *vendors.InventoryItem) *DeviceValidationResult {
	result := &DeviceValidationResult{
		Valid:  true,
		Errors: make([]string, 0),
	}

	// Validate site
	if item.SiteName == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "device has no site assignment")
	} else {
		siteID, err := v.ValidateSite(item.SiteName)
		if err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, err.Error())
		} else {
			result.SiteID = siteID
		}
	}

	// Validate device type (by model)
	if item.Model == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "device has no model")
	} else {
		deviceTypeID, err := v.ValidateDeviceType(item.Model)
		if err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, err.Error())
		} else {
			result.DeviceTypeID = deviceTypeID
		}
	}

	// Validate device role (by device type: ap, switch, gateway)
	// Pass model and NetBox extension to support model-specific and per-device role overrides
	if item.Type == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "device has no type")
	} else {
		roleID, err := v.ValidateDeviceRole(item.Type, item.Model, item.NetBox)
		if err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, err.Error())
		} else {
			result.DeviceRoleID = roleID
		}
	}

	return result
}

// CheckDeviceExists checks if a device already exists in NetBox by MAC address
func (v *Validator) CheckDeviceExists(ctx context.Context, mac string) (int64, bool, error) {
	normalizedMAC := macaddr.NormalizeOrEmpty(mac)
	if normalizedMAC == "" {
		return 0, false, fmt.Errorf("invalid MAC address: %s", mac)
	}

	// Check cache first
	if v.cache != nil {
		if id, ok := v.cache.DevicesByMAC[normalizedMAC]; ok {
			return id, true, nil
		}
	}

	// Query NetBox
	device, err := v.client.GetDeviceByMAC(ctx, mac)
	if err != nil {
		return 0, false, err
	}

	if device == nil {
		return 0, false, nil
	}

	// Cache the result
	if v.cache != nil {
		v.cache.DevicesByMAC[normalizedMAC] = device.ID
	}

	return device.ID, true, nil
}

// GetCacheStats returns statistics about the lookup cache
func (v *Validator) GetCacheStats() map[string]int {
	if v.cache == nil {
		return map[string]int{}
	}

	return map[string]int{
		"sites":        len(v.cache.SitesByName),
		"device_types": len(v.cache.DeviceTypesBySlug),
		"device_roles": len(v.cache.DeviceRolesBySlug),
		"devices":      len(v.cache.DevicesByMAC),
	}
}

// GetSiteID returns the cached site ID for a site name (without validation error)
func (v *Validator) GetSiteID(siteName string) (int64, bool) {
	if v.cache == nil {
		return 0, false
	}

	// Try exact name match first
	if id, ok := v.cache.SitesByName[strings.ToLower(siteName)]; ok {
		return id, true
	}

	// Try slug match
	slug := v.config.GetSiteSlug(siteName)
	if id, ok := v.cache.SitesBySlug[slug]; ok {
		return id, true
	}

	return 0, false
}

// GetDeviceTypeID returns the cached device type ID for a model
func (v *Validator) GetDeviceTypeID(model string) (int64, bool) {
	if v.cache == nil {
		return 0, false
	}

	slug := v.config.GetDeviceTypeSlug(model)
	if id, ok := v.cache.DeviceTypesBySlug[slug]; ok {
		return id, true
	}

	return 0, false
}

// GetDeviceRoleID returns the cached device role ID for a device type.
// Use GetDeviceRoleIDForModel if you need model-specific or per-device role overrides.
func (v *Validator) GetDeviceRoleID(deviceType string) (int64, bool) {
	return v.GetDeviceRoleIDForModel(deviceType, "", nil)
}

// GetDeviceRoleIDForModel returns the cached device role ID with model-specific override support.
// The deviceNetBox parameter can be nil or a *NetBoxDeviceExtension for per-device role overrides.
func (v *Validator) GetDeviceRoleIDForModel(deviceType string, model string, deviceNetBox any) (int64, bool) {
	if v.cache == nil {
		return 0, false
	}

	slug := v.config.GetDeviceRoleSlugForModel(deviceType, model, deviceNetBox)
	if id, ok := v.cache.DeviceRolesBySlug[slug]; ok {
		return id, true
	}

	return 0, false
}

// GetWirelessLANID returns the cached WirelessLAN ID for an SSID
func (v *Validator) GetWirelessLANID(ssid string) (int64, bool) {
	if v.cache == nil {
		return 0, false
	}

	if id, ok := v.cache.WirelessLANsBySSID[ssid]; ok {
		return id, true
	}

	return 0, false
}

// GetInterfaceTemplates returns cached interface templates for a device type.
// If templates are not cached, it fetches them from NetBox and caches the result.
func (v *Validator) GetInterfaceTemplates(ctx context.Context, deviceTypeID int64) ([]*InterfaceTemplate, error) {
	if v.cache == nil {
		return nil, fmt.Errorf("validator not initialized")
	}

	// Check cache first
	if templates, ok := v.cache.InterfaceTemplatesByDeviceType[deviceTypeID]; ok {
		return templates, nil
	}

	// Fetch from NetBox
	templates, err := v.client.GetInterfaceTemplates(ctx, deviceTypeID)
	if err != nil {
		return nil, err
	}

	// Cache the result
	v.cache.InterfaceTemplatesByDeviceType[deviceTypeID] = templates
	logging.Debugf("Cached %d interface templates for device type ID %d", len(templates), deviceTypeID)

	return templates, nil
}

// GetInterfaceTemplate returns a specific interface template by name for a device type
func (v *Validator) GetInterfaceTemplate(ctx context.Context, deviceTypeID int64, interfaceName string) (*InterfaceTemplate, error) {
	templates, err := v.GetInterfaceTemplates(ctx, deviceTypeID)
	if err != nil {
		return nil, err
	}

	for _, tmpl := range templates {
		if tmpl.Name == interfaceName {
			return tmpl, nil
		}
	}

	return nil, nil // Not found
}

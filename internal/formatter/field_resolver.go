package formatter

import (
	"fmt"
	"strings"

	"github.com/ravinald/wifimgr/internal/vendors"
)

// FieldResolver provides field ID to name resolution functionality
type FieldResolver interface {
	// ResolveField resolves a field value (ID) to its human-readable equivalent (name)
	ResolveField(fieldPath string, value interface{}) (interface{}, error)

	// IsResolvableField checks if a field can be resolved
	IsResolvableField(fieldPath string) bool

	// SetResolveMode enables or disables field resolution
	SetResolveMode(resolve bool)

	// GetResolveMode returns current resolution mode
	GetResolveMode() bool
}

// CacheFieldResolver implements FieldResolver using cache indexes
type CacheFieldResolver struct {
	cacheAccessor *vendors.CacheAccessor
	resolveMode   bool
	patterns      *ResolverPatterns
}

// NewCacheFieldResolver creates a new cache-based field resolver
func NewCacheFieldResolver(cacheAccessor *vendors.CacheAccessor) FieldResolver {
	return &CacheFieldResolver{
		cacheAccessor: cacheAccessor,
		resolveMode:   true, // Default to resolving fields
		patterns:      NewResolverPatterns(),
	}
}

// SetResolveMode enables or disables field resolution
func (r *CacheFieldResolver) SetResolveMode(resolve bool) {
	r.resolveMode = resolve
}

// GetResolveMode returns current resolution mode
func (r *CacheFieldResolver) GetResolveMode() bool {
	return r.resolveMode
}

// IsResolvableField checks if a field can be resolved based on patterns
func (r *CacheFieldResolver) IsResolvableField(fieldPath string) bool {
	if !r.resolveMode || r.cacheAccessor == nil {
		return false
	}

	return r.patterns.IsResolvable(fieldPath)
}

// ResolveField resolves a field value using cache indexes
func (r *CacheFieldResolver) ResolveField(fieldPath string, value interface{}) (interface{}, error) {
	if !r.resolveMode || !r.IsResolvableField(fieldPath) {
		return value, nil
	}

	// Convert value to string ID
	idStr, ok := value.(string)
	if !ok || idStr == "" {
		return value, nil
	}

	// Resolve based on field pattern
	resolved, err := r.resolveByPattern(fieldPath, idStr)
	if err != nil {
		// On resolution error, return original ID with undefined indicator
		return fmt.Sprintf("%s (undefined name)", idStr), nil
	}

	if resolved != "" {
		return resolved, nil
	}

	// Return original ID with undefined indicator if no resolution found
	return fmt.Sprintf("%s (undefined name)", idStr), nil
}

// resolveByPattern resolves field based on its pattern
func (r *CacheFieldResolver) resolveByPattern(fieldPath, id string) (string, error) {
	fieldLower := strings.ToLower(fieldPath)

	switch {
	case strings.HasSuffix(fieldLower, "site_id") || fieldLower == "site_id":
		return r.resolveSiteID(id)

	case strings.HasSuffix(fieldLower, "rf_template_id"):
		return r.resolveRFTemplateID(id)

	case strings.HasSuffix(fieldLower, "rfprofileid"):
		// Meraki uses camelCase rfProfileId - resolve using same RF template lookup
		return r.resolveRFTemplateID(id)

	case strings.HasSuffix(fieldLower, "gateway_template_id"):
		return r.resolveGatewayTemplateID(id)

	case strings.HasSuffix(fieldLower, "network_template_id"):
		return r.resolveNetworkID(id)

	case strings.HasSuffix(fieldLower, "ap_template_id"):
		return r.resolveRFTemplateID(id) // AP templates use RF templates

	case strings.HasSuffix(fieldLower, "device_profile_id"):
		return r.resolveDeviceProfileID(id)

	case strings.HasSuffix(fieldLower, "wlan_template_id"):
		return r.resolveWLANTemplateID(id)

	default:
		return "", fmt.Errorf("no resolver for field: %s", fieldPath)
	}
}

// Site ID resolution
func (r *CacheFieldResolver) resolveSiteID(id string) (string, error) {
	site, err := r.cacheAccessor.GetSiteByID(id)
	if err != nil {
		return "", err
	}

	if site.Name != "" {
		return site.Name, nil
	}

	// Name is empty - will trigger undefined name indicator
	return "", fmt.Errorf("site name undefined for ID: %s", id)
}

// RF Template ID resolution
func (r *CacheFieldResolver) resolveRFTemplateID(id string) (string, error) {
	template, err := r.cacheAccessor.GetRFTemplateByID(id)
	if err != nil {
		return "", err
	}

	if template.Name != "" {
		return template.Name, nil
	}

	// Name is empty - will trigger undefined name indicator
	return "", fmt.Errorf("RF template name undefined for ID: %s", id)
}

// Gateway Template ID resolution
func (r *CacheFieldResolver) resolveGatewayTemplateID(id string) (string, error) {
	template, err := r.cacheAccessor.GetGWTemplateByID(id)
	if err != nil {
		return "", err
	}

	if template.Name != "" {
		return template.Name, nil
	}

	// Name is empty - will trigger undefined name indicator
	return "", fmt.Errorf("gateway template name undefined for ID: %s", id)
}

// Network ID resolution
func (r *CacheFieldResolver) resolveNetworkID(id string) (string, error) {
	network, err := r.cacheAccessor.GetNetworkByID(id)
	if err != nil {
		return "", err
	}

	if network.Name != "" {
		return network.Name, nil
	}

	// Name is empty - will trigger undefined name indicator
	return "", fmt.Errorf("network name undefined for ID: %s", id)
}

// Device Profile ID resolution
func (r *CacheFieldResolver) resolveDeviceProfileID(id string) (string, error) {
	profile, err := r.cacheAccessor.GetDeviceProfileByID(id)
	if err != nil {
		return "", err
	}

	if profile.Name != "" {
		return profile.Name, nil
	}

	// Name is empty - will trigger undefined name indicator
	return "", fmt.Errorf("device profile name undefined for ID: %s", id)
}

// WLAN Template ID resolution
func (r *CacheFieldResolver) resolveWLANTemplateID(id string) (string, error) {
	template, err := r.cacheAccessor.GetWLANTemplateByID(id)
	if err != nil {
		return "", err
	}

	if template.Name != "" {
		return template.Name, nil
	}

	// Name is empty - will trigger undefined name indicator
	return "", fmt.Errorf("WLAN template name undefined for ID: %s", id)
}

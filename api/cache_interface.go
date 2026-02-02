package api

// Cacher defines the unified interface for all cache operations.
// It abstracts cache storage, retrieval, and management across different implementations.
type Cacher interface {
	// Core operations
	Load() error
	Save() error
	Clear() error

	// Data access (unified methods)
	GetOrgData(orgID string) (*OrgData, error)
	SetOrgData(orgID string, data *OrgData) error

	// Site operations
	GetSite(identifier string) (*MistSite, error)
	GetAllSites() ([]*MistSite, error)
	UpdateSite(site *MistSite) error

	// Device operations
	GetDevice(mac string) (*UnifiedDevice, error)
	GetDevicesByType(siteID, deviceType string) ([]UnifiedDevice, error)
	UpdateDevice(device *UnifiedDevice) error

	// Config operations
	GetConfigs(orgID, deviceType string) ([]interface{}, error)
	UpdateConfigs(orgID, deviceType string, configs []interface{}) error
	MergeConfigs(orgID, deviceType string, configs []interface{}) error

	// Inventory operations
	GetInventory(deviceType string) ([]interface{}, error)
	UpdateInventory(deviceType string, items []interface{}) error

	// Profile operations
	GetProfiles(profileType string) ([]interface{}, error)
	UpdateProfiles(profileType string, profiles []interface{}) error

	// Metadata
	GetMetadata() *CacheMetadata
	UpdateMetadata(metadata *CacheMetadata)

	// State management
	IsDirty() bool
	MarkDirty()
	GetPath() string
}

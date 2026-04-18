package ubiquiti

// API response types for the Ubiquiti Site Manager API v1.0.0.
// See: https://developer.ui.com/site-manager-api/

// apiResponse is the standard envelope for all Site Manager API responses.
type apiResponse struct {
	Data           any    `json:"data"`
	HTTPStatusCode int    `json:"httpStatusCode"`
	TraceID        string `json:"traceId"`
	NextToken      string `json:"nextToken,omitempty"`
}

// Host represents a controller/console from GET /v1/hosts.
type Host struct {
	ID            string        `json:"id"`
	HardwareID    string        `json:"hardwareId"`
	Type          string        `json:"type"`
	IPAddress     string        `json:"ipAddress"`
	ReportedState ReportedState `json:"reportedState"`
}

// ReportedState contains the host's reported configuration.
type ReportedState struct {
	Name     string `json:"name"`
	Hostname string `json:"hostname"`
	Version  string `json:"version"`
	IP       string `json:"ip"`
}

// Site represents a site from GET /v1/sites.
type Site struct {
	SiteID     string        `json:"siteId"`
	HostID     string        `json:"hostId"`
	Meta       SiteMeta      `json:"meta"`
	Statistics SiteStatistics `json:"statistics"`
	Permission string        `json:"permission"`
	IsOwner    bool          `json:"isOwner"`
}

// SiteMeta contains the site's metadata.
type SiteMeta struct {
	Name       string `json:"name"`
	Desc       string `json:"desc"`
	Timezone   string `json:"timezone"`
	GatewayMAC string `json:"gatewayMac"`
}

// SiteStatistics contains site-level statistics.
type SiteStatistics struct {
	Counts SiteCounts `json:"counts"`
}

// SiteCounts contains device and client counts per site.
type SiteCounts struct {
	TotalDevice       int `json:"totalDevice"`
	CriticalDevice    int `json:"criticalDevice"`
	OfflineDevice     int `json:"offlineDevice"`
	TotalClientDevice int `json:"totalClientDevice"`
}

// GetID returns the site ID.
func (s Site) GetID() string {
	return s.SiteID
}

// GetName returns the site name.
func (s Site) GetName() string {
	return s.Meta.Name
}

// HostDeviceGroup represents a device group from GET /v1/devices.
// Devices are grouped by host in the API response.
type HostDeviceGroup struct {
	HostID   string   `json:"hostId"`
	HostName string   `json:"hostName"`
	Devices  []Device `json:"devices"`
}

// Device represents an individual device within a host device group.
type Device struct {
	ID             string `json:"_id"`
	MAC            string `json:"mac"`
	Name           string `json:"name"`
	Model          string `json:"model"`
	Shortname      string `json:"shortname"`
	IP             string `json:"ip"`
	ProductLine    string `json:"productLine"`
	Status         string `json:"status"` // "online", "offline"
	Version        string `json:"version"`
	FirmwareStatus string `json:"firmwareStatus"`
	IsConsole      bool   `json:"isConsole"`
	IsManaged      bool   `json:"isManaged"`
	StartupTime    any    `json:"startupTime"`
	AdoptionTime   any    `json:"adoptionTime"`
	Note           string `json:"note"`
}

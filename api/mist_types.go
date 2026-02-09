package api

// Mist API Types
// These are Mist-specific types used by api.Client methods.
// For vendor-agnostic types, use the vendors package (internal/vendors/types.go).

// MistNetwork represents an organization network
type MistNetwork struct {
	ID    *string `json:"id,omitempty"`
	Name  *string `json:"name,omitempty"`
	OrgID *string `json:"org_id,omitempty"`

	VlanID   *int    `json:"vlan_id,omitempty"`
	Subnet   *string `json:"subnet,omitempty"`
	Gateway  *string `json:"gateway,omitempty"`
	Subnet6  *string `json:"subnet6,omitempty"`
	Gateway6 *string `json:"gateway6,omitempty"`

	InternalAccess *bool `json:"internal_access,omitempty"`
	InternetAccess *bool `json:"internet_access,omitempty"`
	Isolation      *bool `json:"isolation,omitempty"`

	Multicast *bool     `json:"multicast,omitempty"`
	Tenants   *[]string `json:"tenants,omitempty"`

	CreatedTime  *int64 `json:"created_time,omitempty"`
	ModifiedTime *int64 `json:"modified_time,omitempty"`

	AdditionalConfig map[string]any `json:"-"`
}

// MistWLAN represents a WLAN configuration
type MistWLAN struct {
	ID     *string `json:"id,omitempty"`
	SSID   *string `json:"ssid,omitempty"`
	OrgID  *string `json:"org_id,omitempty"`
	SiteID *string `json:"site_id,omitempty"`

	VlanID    *int    `json:"vlan_id,omitempty"`
	Interface *string `json:"interface,omitempty"`
	Isolation *bool   `json:"isolation,omitempty"`

	Auth struct {
		Type       *string   `json:"type,omitempty"`
		PSK        *string   `json:"psk,omitempty"`
		KeyIdx     *int      `json:"key_idx,omitempty"`
		Keys       *[]string `json:"keys,omitempty"`
		Pairwise   *[]string `json:"pairwise,omitempty"`
		Enterprise *struct {
			Radius *struct {
				Host   *string `json:"host,omitempty"`
				Port   *int    `json:"port,omitempty"`
				Secret *string `json:"secret,omitempty"`
			} `json:"radius,omitempty"`
		} `json:"enterprise,omitempty"`
	} `json:"auth,omitempty"`

	QoS struct {
		Class *string `json:"class,omitempty"`
	} `json:"qos,omitempty"`

	Band    *string   `json:"band,omitempty"`
	Bands   *[]string `json:"bands,omitempty"`
	Enabled *bool     `json:"enabled,omitempty"`
	Hidden  *bool     `json:"hidden,omitempty"`
	ApplyTo *string   `json:"apply_to,omitempty"` // "site" or "aps"
	ApIDs   *[]string `json:"ap_ids,omitempty"`

	CreatedTime  *int64 `json:"created_time,omitempty"`
	ModifiedTime *int64 `json:"modified_time,omitempty"`

	AdditionalConfig map[string]any `json:"-"`
}

// MistRFTemplate represents an RF template
type MistRFTemplate struct {
	ID    *string `json:"id,omitempty"`
	Name  *string `json:"name,omitempty"`
	OrgID *string `json:"org_id,omitempty"`

	AntGain24 *int `json:"ant_gain_24,omitempty"`
	AntGain5  *int `json:"ant_gain_5,omitempty"`
	AntGain6  *int `json:"ant_gain_6,omitempty"`

	Band24         map[string]any `json:"band_24,omitempty"`
	Band24Usage    *string        `json:"band_24_usage,omitempty"`
	Band5          map[string]any `json:"band_5,omitempty"`
	Band5On24Radio map[string]any `json:"band_5_on_24_radio,omitempty"`
	Band6          map[string]any `json:"band_6,omitempty"`

	CountryCode     *string `json:"country_code,omitempty"`
	ForSite         *bool   `json:"for_site,omitempty"`
	ScanningEnabled *bool   `json:"scanning_enabled,omitempty"`

	ModelSpecific map[string]any `json:"model_specific,omitempty"`

	CreatedTime  *int64 `json:"created_time,omitempty"`
	ModifiedTime *int64 `json:"modified_time,omitempty"`
}

// MistGatewayTemplate represents a gateway template
type MistGatewayTemplate struct {
	ID    *string `json:"id,omitempty"`
	Name  *string `json:"name,omitempty"`
	OrgID *string `json:"org_id,omitempty"`
	Type  *string `json:"type,omitempty"`

	Networks   *map[string]any `json:"networks,omitempty"`
	PortConfig *map[string]any `json:"port_config,omitempty"`

	BGPConfig     *map[string]any `json:"bgp_config,omitempty"`
	VRFConfig     *map[string]any `json:"vrf_config,omitempty"`
	RoutingConfig *map[string]any `json:"routing_config,omitempty"`

	CreatedTime  *int64 `json:"created_time,omitempty"`
	ModifiedTime *int64 `json:"modified_time,omitempty"`

	AdditionalConfig map[string]any `json:"-"`
}

// MistWLANTemplate represents a WLAN template
type MistWLANTemplate struct {
	ID    *string `json:"id,omitempty"`
	Name  *string `json:"name,omitempty"`
	OrgID *string `json:"org_id,omitempty"`

	SSID      *string `json:"ssid,omitempty"`
	VlanID    *int    `json:"vlan_id,omitempty"`
	Interface *string `json:"interface,omitempty"`

	Auth *map[string]any `json:"auth,omitempty"`
	QoS  *map[string]any `json:"qos,omitempty"`

	Band    *string `json:"band,omitempty"`
	Enabled *bool   `json:"enabled,omitempty"`
	Hidden  *bool   `json:"hidden,omitempty"`

	CreatedTime  *int64 `json:"created_time,omitempty"`
	ModifiedTime *int64 `json:"modified_time,omitempty"`

	AdditionalConfig map[string]any `json:"-"`
}

package config

import (
	"github.com/ravinald/wifimgr/api"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// Debug levels
const (
	DebugNone  = iota // No debug output
	DebugInfo         // -d: Info level logging
	DebugDebug        // -dd: Debug level logging
	DebugTrace        // -ddd: Trace level logging (most verbose)

	// DebugAll is an alias for DebugDebug for backward compatibility
	DebugAll = DebugDebug
)

// Config represents the main CLI configuration
type Config struct {
	Version int     `json:"version"`
	Files   Files   `json:"files"`
	API     API     `json:"api"`
	Display Display `json:"display"`
	Logging Logging `json:"logging"`
}

// Files represents configuration for file paths
type Files struct {
	ConfigDir     string   `json:"config_dir"`
	SiteConfigs   []string `json:"site_configs"`
	Templates     []string `json:"templates,omitempty"` // Template files (radio, wlan, device)
	Cache         string   `json:"cache"`
	Inventory     string   `json:"inventory"`
	LogFile       string   `json:"log_file"`
	Schemas       string   `json:"schemas"`
	ConfigBackups int      `json:"config_backups"` // Number of backups to keep per site
}

// Logging represents logging configuration settings
type Logging struct {
	Enable bool   `json:"enable"`
	Level  string `json:"level"`
	Format string `json:"format"`
	Stdout bool   `json:"stdout"`
}

// Credentials represents API credentials
type Credentials struct {
	APIID    string `json:"api_id"`
	APIToken string `json:"api_token"`
	OrgID    string `json:"org_id"`

	// Encryption related fields
	KeyEncryptionSalt string `json:"key_encryption_salt,omitempty"`
}

// ManagedKeys represents which configuration keys are managed for each device type
type ManagedKeys struct {
	AP      []string `json:"ap,omitempty"`
	Switch  []string `json:"switch,omitempty"`
	Gateway []string `json:"gateway,omitempty"`
}

// API represents API configuration settings
type API struct {
	Credentials  Credentials  `json:"credentials"`
	URL          string       `json:"url"`
	RateLimit    int          `json:"rate_limit"`
	ResultsLimit int          `json:"results_limit"`
	ManagedKeys  *ManagedKeys `json:"managed_keys,omitempty"`

	currentToken string // Runtime-only: decrypted token for current session
}

// WLANProfile represents a reusable WLAN template
type WLANProfile struct {
	SSID            string        `json:"ssid"`
	Enabled         bool          `json:"enabled"`
	Hidden          bool          `json:"hidden,omitempty"`
	Band            string        `json:"band,omitempty"` // "2.4", "5", "6", "dual", "all"
	VLANID          int           `json:"vlan_id,omitempty"`
	Auth            WLANAuth      `json:"auth"`
	RoamMode        string        `json:"roam_mode,omitempty"`       // "none", "11r", "OKC"
	ClientLimitUp   int           `json:"client_limit_up,omitempty"` // Per-client upload limit (Kbps)
	ClientLimitDown int           `json:"client_limit_down,omitempty"`
	Portal          *PortalConfig `json:"portal,omitempty"`
}

// WLANAuth represents WLAN authentication configuration
type WLANAuth struct {
	Type          string         `json:"type"`                     // "open", "psk", "eap"
	PSK           string         `json:"psk,omitempty"`            // Pre-shared key (may be encrypted)
	Pairwise      []string       `json:"pairwise,omitempty"`       // ["wpa2-ccmp", "wpa3"]
	RADIUSServers []RADIUSServer `json:"radius_servers,omitempty"` // RADIUS servers for 802.1X
}

// PortalConfig represents captive portal settings
type PortalConfig struct {
	Enabled bool   `json:"enabled"`
	Auth    string `json:"auth,omitempty"` // "sponsor", "passphrase", "sso", etc.
}

// RadioProfile represents reusable radio settings
type RadioProfile struct {
	Name      string           `json:"name"`
	Band24    *RadioBandConfig `json:"band_24,omitempty"`
	Band5     *RadioBandConfig `json:"band_5,omitempty"`
	Band6     *RadioBandConfig `json:"band_6,omitempty"`
	AntGain24 int              `json:"ant_gain_24,omitempty"` // External antenna gain (dBi)
	AntGain5  int              `json:"ant_gain_5,omitempty"`
	AntGain6  int              `json:"ant_gain_6,omitempty"`
}

// RadioBandConfig represents radio band settings for a profile
type RadioBandConfig struct {
	Disabled  bool  `json:"disabled,omitempty"`
	Power     int   `json:"power,omitempty"`     // Fixed TX power (dBm)
	PowerMin  int   `json:"power_min,omitempty"` // Min power for auto (dBm)
	PowerMax  int   `json:"power_max,omitempty"` // Max power for auto (dBm)
	Bandwidth int   `json:"bandwidth,omitempty"` // Channel width (20/40/80/160)
	Channels  []int `json:"channels,omitempty"`  // Allowed channel list
}

// WLANProfileFile represents a WLAN profile file schema
type WLANProfileFile struct {
	Version      int                     `json:"version"`
	WLANProfiles map[string]*WLANProfile `json:"wlan_profiles"`
}

// RadioProfileFile represents a radio profile file schema
type RadioProfileFile struct {
	Version       int                      `json:"version"`
	RadioProfiles map[string]*RadioProfile `json:"radio_profiles"`
}

// SiteConfigObjProfiles declares which templates a site uses
type SiteConfigObjProfiles struct {
	WLAN   []string `json:"wlan,omitempty"`   // WLAN template labels to create at site
	Radio  []string `json:"radio,omitempty"`  // Radio template labels
	Device []string `json:"device,omitempty"` // Device template labels
}

// SiteConfigObj represents a site configuration object
type SiteConfigObj struct {
	API        string                `json:"api,omitempty"` // API label for multi-vendor support
	SiteConfig SiteConfig            `json:"site_config"`
	Profiles   SiteConfigObjProfiles `json:"profiles,omitempty"`
	WLAN       []string              `json:"wlan,omitempty"` // WLANs to apply to all APs (site-wide default)
	Devices    Devices               `json:"devices"`
}

// SiteConfigFile represents a site configuration file
// It follows the format: { "version": 1, "config": { "sites": { "SITE_NAME": { "site_config": {...}, "devices": {...} } } } }
type SiteConfigFile struct {
	Version int               `json:"version"`
	Config  SiteConfigWrapper `json:"config"`
}

// SiteConfigWrapper wraps the sites map in the config structure
type SiteConfigWrapper struct {
	Sites map[string]SiteConfigObj `json:"sites"`
}

// Devices represents the device configurations for a site
type Devices struct {
	APs      map[string]APConfig      `json:"ap"`      // Keyed by MAC address
	Switches map[string]SwitchConfig  `json:"switch"`  // Keyed by MAC address
	WanEdge  map[string]WanEdgeConfig `json:"gateway"` // Keyed by MAC address
}

// Display represents display formatting settings
type Display struct {
	Sites     DisplayFormat            `json:"sites"`
	APs       DisplayFormat            `json:"aps"`
	Inventory DisplayFormat            `json:"inventory"`
	Commands  map[string]CommandFormat `json:"commands"`
}

// DisplayFormat represents format settings for a specific category (legacy format)
type DisplayFormat struct {
	Format          string   `json:"format"`                     // "table" or "csv"
	Fields          []string `json:"fields"`                     // API field names to display
	AvailableFields []string `json:"available_fields,omitempty"` // Optional list of all available fields for this category
}

// CommandFormat represents the new format for command-specific display settings
type CommandFormat struct {
	Format          string        `json:"format"`                     // "table" or "csv"
	Fields          []interface{} `json:"fields"`                     // Array of field configuration objects: [{"field": "name", "title": "Name", "width": 32}, ...]
	Title           string        `json:"title"`                      // Optional title for the display
	AvailableFields []string      `json:"available_fields,omitempty"` // Optional list of all available fields for this command
}

// SiteConfig represents a site configuration
type SiteConfig struct {
	Name        string      `json:"name"`
	Address     string      `json:"address"`
	CountryCode string      `json:"country_code"`
	Timezone    string      `json:"timezone"`
	Notes       string      `json:"notes"`
	LatLng      *api.LatLng `json:"latlng"`
	API         string      `json:"api,omitempty"` // API label for multi-vendor support
}

// APConfig represents an AP configuration.
// It combines device identification fields with the vendor-agnostic configuration.
type APConfig struct {
	// Device identification (these are identifiers, not configuration)
	MAC   string `json:"mac"`             // MAC address (key in map)
	Magic string `json:"magic,omitempty"` // Device identification field
	API   string `json:"api,omitempty"`   // API label override (inherits from site if empty)

	// Profile references
	RadioProfile string   `json:"radio_profile,omitempty"` // Reference to RadioProfile name
	WLANs        []string `json:"wlan,omitempty"`          // List of WLAN labels (profile or site wlan)

	// Embed the vendor-agnostic AP device configuration
	// All configuration fields are defined in APDeviceConfig
	*vendors.APDeviceConfig

	// Legacy fields kept for backward compatibility
	// These will be migrated to the new structure in APDeviceConfig
	Config     APHWConfig `json:"config,omitempty"`      // Deprecated: use RadioConfig in APDeviceConfig
	Locked     bool       `json:"locked,omitempty"`      // Map lock status
	VlanID     int        `json:"vlan_id,omitempty"`     // Deprecated: use IPConfig.VlanID
	NTPServers []string   `json:"ntp_servers,omitempty"` // NTP servers list
}

// SwitchConfig represents a switch configuration
type SwitchConfig struct {
	Name                 string                 `json:"name"`
	Tags                 []string               `json:"tags,omitempty"`
	Notes                string                 `json:"notes,omitempty"`
	Magic                string                 `json:"magic,omitempty"` // Device identification field
	Role                 string                 `json:"role,omitempty"`
	IPConfig             IPConfig               `json:"ip_config,omitempty"`
	OobIPConfig          IPConfig               `json:"oob_ip_config,omitempty"`
	PortConfig           map[string]PortConfig  `json:"port_config,omitempty"`
	PortConfigOverwrite  map[string]PortConfig  `json:"port_config_overwrite,omitempty"`
	Networks             []Network              `json:"networks,omitempty"`
	OtherIPConfigs       []OtherIPConfig        `json:"other_ip_configs,omitempty"`
	RouterID             string                 `json:"router_id,omitempty"`
	ExtraRoutes          []Route                `json:"extra_routes,omitempty"`
	AggregateRoutes      []Route                `json:"aggregate_routes,omitempty"`
	OSPFConfig           OSPFConfig             `json:"ospf_config,omitempty"`
	VRRPConfig           []VRRPConfig           `json:"vrrp_config,omitempty"`
	VRFConfig            []VRFConfig            `json:"vrf_config,omitempty"`
	STPConfig            STPConfig              `json:"stp_config,omitempty"`
	DHCPDConfig          DHCPDConfig            `json:"dhcpd_config,omitempty"`
	DHCPSnooping         DHCPSnoopingConfig     `json:"dhcp_snooping,omitempty"`
	DNSServers           []string               `json:"dns_servers,omitempty"`
	DNSSuffix            string                 `json:"dns_suffix,omitempty"`
	NTPServers           []string               `json:"ntp_servers,omitempty"`
	RADIUSConfig         RADIUSConfig           `json:"radius_config,omitempty"`
	ACLTags              []string               `json:"acl_tags,omitempty"`
	ACLPolicies          []ACLPolicy            `json:"acl_policies,omitempty"`
	Managed              bool                   `json:"managed,omitempty"`
	DisableAutoConfig    bool                   `json:"disable_auto_config,omitempty"`
	PortMirroring        []PortMirroringConfig  `json:"port_mirroring,omitempty"`
	IoTConfig            IoTConfig              `json:"iot_config,omitempty"`
	DeviceProfileID      string                 `json:"deviceprofile_id,omitempty"`
	AdditionalConfigCmds []string               `json:"additional_config_cmds,omitempty"`
	Vars                 map[string]interface{} `json:"vars,omitempty"`
}

// WanEdgeConfig represents a WAN edge device configuration
type WanEdgeConfig struct {
	Name  string   `json:"name"`
	Tags  []string `json:"tags,omitempty"`
	Notes string   `json:"notes,omitempty"`
	Magic string   `json:"magic,omitempty"` // Device identification field
}

// APHWConfig represents AP hardware configuration
type APHWConfig struct {
	LEDEnabled      bool    `json:"led_enabled"`
	Band24          BandCfg `json:"band_24"`
	Band5           BandCfg `json:"band_5"`
	Band6           BandCfg `json:"band_6,omitempty"`
	Band24Usage     string  `json:"band_24_usage,omitempty"`
	ScanningEnabled bool    `json:"scanning_enabled,omitempty"`
	IndoorUse       bool    `json:"indoor_use,omitempty"`
}

// BandCfg represents configuration for a wireless band
type BandCfg struct {
	Disabled    bool   `json:"disabled"`
	TxPower     int    `json:"tx_power"`
	Channel     int    `json:"channel"`
	Bandwidth   int    `json:"bandwidth,omitempty"`
	AntennaMode string `json:"antenna_mode,omitempty"`
	PowerMin    int    `json:"power_min,omitempty"`
	PowerMax    int    `json:"power_max,omitempty"`
}

// IPConfig represents IP configuration for a device
type IPConfig struct {
	Type    string   `json:"type,omitempty"`
	IP      string   `json:"ip,omitempty"`
	Netmask string   `json:"netmask,omitempty"`
	Gateway string   `json:"gateway,omitempty"`
	DNS     []string `json:"dns,omitempty"`
}

// MeshConfig represents mesh settings for an AP
type MeshConfig struct {
	Enabled bool   `json:"enabled"`
	Role    string `json:"role,omitempty"`
	Group   string `json:"group,omitempty"`
}

// BLEConfig represents Bluetooth Low Energy settings for an AP
type BLEConfig struct {
	Power     int             `json:"power,omitempty"`
	Mode      string          `json:"mode,omitempty"`
	IBeacon   IBeaconConfig   `json:"ibeacon,omitempty"`
	Eddystone EddystoneConfig `json:"eddystone,omitempty"`
}

// IBeaconConfig represents iBeacon settings
type IBeaconConfig struct {
	UUID       string `json:"uuid,omitempty"`
	Major      int    `json:"major,omitempty"`
	Minor      int    `json:"minor,omitempty"`
	PowerLevel int    `json:"power_level,omitempty"`
}

// EddystoneConfig represents Eddystone beacon settings
type EddystoneConfig struct {
	NamespaceID string `json:"namespace_id,omitempty"`
	InstanceID  string `json:"instance_id,omitempty"`
	PowerLevel  int    `json:"power_level,omitempty"`
}

// PortConfig represents Ethernet port configuration for a device
type PortConfig struct {
	Mode        string   `json:"mode,omitempty"`
	Usage       string   `json:"usage,omitempty"`
	VLANs       []int    `json:"vlans,omitempty"`
	VoiceVLAN   int      `json:"voice_vlan,omitempty"`
	EnablePOE   bool     `json:"enable_poe,omitempty"`
	SpeedDuplex string   `json:"speed_duplex,omitempty"`
	PortAuth    PortAuth `json:"port_auth,omitempty"`
	Disabled    bool     `json:"disabled,omitempty"`
	Description string   `json:"description,omitempty"`
}

// PortAuth represents 802.1x port authentication settings
type PortAuth struct {
	Enabled bool   `json:"enabled"`
	Mode    string `json:"mode,omitempty"`
}

// LACPConfig represents Link Aggregation Control Protocol settings
type LACPConfig struct {
	Enabled     bool     `json:"enabled"`
	PortMembers []string `json:"port_members,omitempty"`
}

// UplinkConfig represents uplink port settings for AP
type UplinkConfig struct {
	Auth8021x    bool   `json:"8021x,omitempty"`
	AuthIdentity string `json:"identity,omitempty"`
	AuthPassword string `json:"password,omitempty"`
}

// IoTConfig represents IoT port configuration
type IoTConfig struct {
	Enabled bool   `json:"enabled"`
	Type    string `json:"type,omitempty"`
}

// PowerConfig represents power management settings
type PowerConfig struct {
	Mode       string `json:"mode,omitempty"`
	BaseValue  int    `json:"base_value,omitempty"`
	OverridePS bool   `json:"override_ps,omitempty"`
}

// Network represents a VLAN network configuration
type Network struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	VlanID      int      `json:"vlan_id"`
	Subnet      string   `json:"subnet,omitempty"`
	Gateway     string   `json:"gateway,omitempty"`
	DHCPEnabled bool     `json:"dhcp_enabled,omitempty"`
	DHCPRelay   []string `json:"dhcp_relay,omitempty"`
}

// OtherIPConfig represents L3 interface configuration
type OtherIPConfig struct {
	NetworkID int    `json:"network_id"`
	IP        string `json:"ip"`
	Netmask   string `json:"netmask"`
	VlanID    int    `json:"vlan_id,omitempty"`
	Subnet    string `json:"subnet,omitempty"`
}

// Route represents a static or aggregate route
type Route struct {
	Network         string `json:"network"`
	NextHop         string `json:"next_hop,omitempty"`
	PreferredSource string `json:"preferred_source,omitempty"`
	Description     string `json:"description,omitempty"`
}

// OSPFConfig represents OSPF routing protocol configuration
type OSPFConfig struct {
	Enabled    bool            `json:"enabled"`
	RouterID   string          `json:"router_id,omitempty"`
	Areas      []OSPFArea      `json:"areas,omitempty"`
	Interfaces []OSPFInterface `json:"interfaces,omitempty"`
}

// OSPFArea represents an OSPF area configuration
type OSPFArea struct {
	ID             string `json:"id"`
	Type           string `json:"type,omitempty"`
	Authentication string `json:"authentication,omitempty"`
}

// OSPFInterface represents OSPF interface configuration
type OSPFInterface struct {
	InterfaceName string `json:"interface_name"`
	AreaID        string `json:"area_id"`
	Priority      int    `json:"priority,omitempty"`
	HelloInterval int    `json:"hello_interval,omitempty"`
	DeadInterval  int    `json:"dead_interval,omitempty"`
	Cost          int    `json:"cost,omitempty"`
}

// VRRPConfig represents Virtual Router Redundancy Protocol settings
type VRRPConfig struct {
	VirtualRouterID int    `json:"virtual_router_id"`
	Priority        int    `json:"priority"`
	VirtualIP       string `json:"virtual_ip"`
	InterfaceName   string `json:"interface_name"`
	Advertisement   int    `json:"advertisement,omitempty"`
	Preempt         bool   `json:"preempt,omitempty"`
}

// VRFConfig represents Virtual Routing and Forwarding settings
type VRFConfig struct {
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	RouteTargets []string `json:"route_targets,omitempty"`
}

// STPConfig represents Spanning Tree Protocol configuration
type STPConfig struct {
	Enabled  bool   `json:"enabled"`
	Mode     string `json:"mode,omitempty"`
	Priority int    `json:"priority,omitempty"`
}

// DHCPDConfig represents DHCP server configuration
type DHCPDConfig struct {
	Enabled bool       `json:"enabled"`
	Pools   []DHCPPool `json:"pools,omitempty"`
}

// DHCPPool represents a DHCP address pool
type DHCPPool struct {
	NetworkID    int          `json:"network_id"`
	StartIP      string       `json:"start_ip"`
	EndIP        string       `json:"end_ip"`
	DefaultLease int          `json:"default_lease,omitempty"`
	MaxLease     int          `json:"max_lease,omitempty"`
	DNS          []string     `json:"dns,omitempty"`
	Options      []DHCPOption `json:"options,omitempty"`
}

// DHCPOption represents a DHCP option
type DHCPOption struct {
	Code  int    `json:"code"`
	Value string `json:"value"`
}

// DHCPSnoopingConfig represents DHCP snooping configuration
type DHCPSnoopingConfig struct {
	Enabled      bool     `json:"enabled"`
	TrustedPorts []string `json:"trusted_ports,omitempty"`
}

// RADIUSConfig represents RADIUS server configuration
type RADIUSConfig struct {
	Servers   []RADIUSServer `json:"servers"`
	SecretKey string         `json:"secret_key,omitempty"`
}

// RADIUSServer represents a RADIUS server
type RADIUSServer struct {
	Host   string `json:"host"`
	Port   int    `json:"port,omitempty"`
	Secret string `json:"secret,omitempty"`
}

// ACLPolicy represents an access control list policy
type ACLPolicy struct {
	Name  string    `json:"name"`
	Rules []ACLRule `json:"rules"`
}

// ACLRule represents a rule within an ACL policy
type ACLRule struct {
	Action     string `json:"action"`
	Protocol   string `json:"protocol,omitempty"`
	SrcNetwork string `json:"src_network,omitempty"`
	DstNetwork string `json:"dst_network,omitempty"`
	SrcPort    string `json:"src_port,omitempty"`
	DstPort    string `json:"dst_port,omitempty"`
}

// PortMirroringConfig represents port mirroring settings
type PortMirroringConfig struct {
	Name            string   `json:"name"`
	SourcePorts     []string `json:"source_ports"`
	DestinationPort string   `json:"destination_port"`
}

// MetadataConfig represents metadata for the configuration
type MetadataConfig struct {
	ExportDate  string `json:"export_date"`
	ExportedBy  string `json:"exported_by"`
	Description string `json:"description"`
}

// CLIOptions holds command-line parameters
type CLIOptions struct {
	ConfigFile    string
	Debug         bool
	DebugLevel    string
	DebugLevelInt int    // exported for use in other packages
	Format        string // Override display format: "table" or "csv"
	Force         bool   // Skip confirmation prompts for destructive operations
	DryRun        bool   // Enable dry-run mode (don't make actual API changes)
	RebuildCache  bool   // Force rebuild of the local cache
	Limit         int    // Override results limit for API pagination
	UseEnvFile    bool   // Read the API token from .env.wifimgr file
}

// SiteIndex provides O(1) lookup of site name to config file path
type SiteIndex struct {
	// SiteToFile maps site name (case-insensitive key) to relative config file path
	SiteToFile map[string]string
	// SiteToKey maps site name (case-insensitive key) to the actual key used in the config file
	SiteToKey map[string]string
}

package api

import (
	"fmt"

	"github.com/ravinald/wifimgr/internal/macaddr"
)

// SearchClientMarshaler defines the interface for bidirectional search client data transformation
// between API representations and structured types.
type SearchClientMarshaler interface {
	GetMAC() string
	GetOrgID() string
	GetSiteID() string
	ToMap() map[string]interface{}
	FromMap(data map[string]interface{}) error
	GetRaw() map[string]interface{}
	SetRaw(data map[string]interface{})
}

// MistDeviceMacPort represents a port connection in the wired client API response with bidirectional handling
type MistDeviceMacPort struct {
	DeviceMAC  *string `json:"device_mac,omitempty"`
	PortID     *string `json:"port_id,omitempty"`
	Start      *string `json:"start,omitempty"`
	When       *string `json:"when,omitempty"`
	IP         *string `json:"ip,omitempty"`
	IP6        *string `json:"ip6,omitempty"`
	VLAN       *int    `json:"vlan,omitempty"`
	PortParent *string `json:"port_parent,omitempty"`
	Node       *string `json:"node,omitempty"`

	// Additional flexible configuration stored as maps
	AdditionalConfig map[string]interface{} `json:"-"`

	// Raw contains the complete API response data for full preservation
	Raw map[string]interface{} `json:"-"`
}

// MistWiredClient represents a Mist wired client with bidirectional data handling
type MistWiredClient struct {
	// Core identification fields
	OrgID     *string `json:"org_id,omitempty"`
	SiteID    *string `json:"site_id,omitempty"`
	ClientMAC *string `json:"client_mac,omitempty"`
	MAC       *string `json:"mac,omitempty"`

	// Timestamp and session information
	Timestamp *float64 `json:"timestamp,omitempty"`

	// Device and port information
	DeviceMacPort []*MistDeviceMacPort `json:"device_mac_port,omitempty"`
	DeviceMAC     []string             `json:"device_mac,omitempty"`
	PortID        []string             `json:"port_id,omitempty"`

	// Network information
	IP       []string `json:"ip,omitempty"`
	IP6      []string `json:"ip6,omitempty"`
	VLAN     []int    `json:"vlan,omitempty"`
	Hostname []string `json:"hostname,omitempty"`
	Username []string `json:"username,omitempty"`

	// Authentication and security
	AuthState  *string `json:"auth_state,omitempty"`
	AuthMethod *string `json:"auth_method,omitempty"`
	RandomMAC  *bool   `json:"random_mac,omitempty"`

	// Device characteristics
	Manufacture *string `json:"manufacture,omitempty"`

	// Last known information
	LastVLANName  *string `json:"last_vlan_name,omitempty"`
	LastVLAN      *int    `json:"last_vlan,omitempty"`
	LastPortID    *string `json:"last_port_id,omitempty"`
	LastHostname  *string `json:"last_hostname,omitempty"`
	LastIP        *string `json:"last_ip,omitempty"`
	LastIP6       *string `json:"last_ip6,omitempty"`
	LastDeviceMAC *string `json:"last_device_mac,omitempty"`

	// Additional flexible configuration stored as maps
	AdditionalConfig map[string]interface{} `json:"-"`

	// Raw contains the complete API response data for full preservation
	Raw map[string]interface{} `json:"-"`
}

// MistWirelessClient represents a Mist wireless client with bidirectional data handling
type MistWirelessClient struct {
	// Core identification fields
	OrgID   *string  `json:"org_id,omitempty"`
	SiteID  *string  `json:"site_id,omitempty"`
	SiteIDs []string `json:"site_ids,omitempty"`
	MAC     *string  `json:"mac,omitempty"`

	// Timestamp and session information
	Timestamp *float64 `json:"timestamp,omitempty"`

	// Access Point and network information
	AP       []string `json:"ap,omitempty"`
	IP       []string `json:"ip,omitempty"`
	Hostname []string `json:"hostname,omitempty"`
	WLANID   []string `json:"wlan_id,omitempty"`
	SSID     []string `json:"ssid,omitempty"`
	VLAN     []int    `json:"vlan,omitempty"`

	// Device information
	Model      []string `json:"model,omitempty"`
	Device     []string `json:"device,omitempty"`
	OS         []string `json:"os,omitempty"`
	OSVersion  []string `json:"os_version,omitempty"`
	Username   []string `json:"username,omitempty"`
	Firmware   []string `json:"firmware,omitempty"`
	SDKVersion []string `json:"sdk_version,omitempty"`
	AppVersion []string `json:"app_version,omitempty"`

	// Device characteristics
	Manufacture *string `json:"mfg,omitempty"`
	Hardware    *string `json:"hardware,omitempty"`

	// Connection characteristics
	RandomMAC *bool   `json:"random_mac,omitempty"`
	Ftc       *bool   `json:"ftc,omitempty"`
	Band      *string `json:"band,omitempty"`
	Protocol  *string `json:"protocol,omitempty"`

	// PSK information
	PskID   []string `json:"psk_id,omitempty"`
	PskName []string `json:"psk_name,omitempty"`

	// Last known information
	LastAP        *string `json:"last_ap,omitempty"`
	LastIP        *string `json:"last_ip,omitempty"`
	LastHostname  *string `json:"last_hostname,omitempty"`
	LastWLANID    *string `json:"last_wlan_id,omitempty"`
	LastSSID      *string `json:"last_ssid,omitempty"`
	LastModel     *string `json:"last_model,omitempty"`
	LastDevice    *string `json:"last_device,omitempty"`
	LastOS        *string `json:"last_os,omitempty"`
	LastOSVersion *string `json:"last_os_version,omitempty"`
	LastFirmware  *string `json:"last_firmware,omitempty"`
	LastVLAN      *int    `json:"last_vlan,omitempty"`

	// Additional flexible configuration stored as maps
	AdditionalConfig map[string]interface{} `json:"-"`

	// Raw contains the complete API response data for full preservation
	Raw map[string]interface{} `json:"-"`
}

// MistWiredClientResponse represents the full response from the wired client search API with bidirectional handling
type MistWiredClientResponse struct {
	Results []*MistWiredClient `json:"results,omitempty"`
	Limit   *int               `json:"limit,omitempty"`
	Start   *int64             `json:"start,omitempty"`
	End     *int64             `json:"end,omitempty"`
	Total   *int               `json:"total,omitempty"`

	// Raw contains the complete API response data for full preservation
	Raw map[string]interface{} `json:"-"`
}

// MistWirelessClientResponse represents the full response from the wireless client search API with bidirectional handling
type MistWirelessClientResponse struct {
	Results []*MistWirelessClient `json:"results,omitempty"`
	Limit   *int                  `json:"limit,omitempty"`
	Start   *int64                `json:"start,omitempty"`
	End     *int64                `json:"end,omitempty"`
	Total   *int                  `json:"total,omitempty"`

	// Raw contains the complete API response data for full preservation
	Raw map[string]interface{} `json:"-"`
}

// MistDeviceMacPort Methods

// ToMap converts the device MAC port to a map representation
func (d *MistDeviceMacPort) ToMap() map[string]interface{} {
	result := make(map[string]interface{})

	// Add typed fields
	if d.DeviceMAC != nil {
		result["device_mac"] = *d.DeviceMAC
	}
	if d.PortID != nil {
		result["port_id"] = *d.PortID
	}
	if d.Start != nil {
		result["start"] = *d.Start
	}
	if d.When != nil {
		result["when"] = *d.When
	}
	if d.IP != nil {
		result["ip"] = *d.IP
	}
	if d.IP6 != nil {
		result["ip6"] = *d.IP6
	}
	if d.VLAN != nil {
		result["vlan"] = *d.VLAN
	}
	if d.PortParent != nil {
		result["port_parent"] = *d.PortParent
	}
	if d.Node != nil {
		result["node"] = *d.Node
	}

	// Add additional configuration
	for key, value := range d.AdditionalConfig {
		result[key] = value
	}

	// Add any raw data that wasn't captured above
	for key, value := range d.Raw {
		if _, exists := result[key]; !exists {
			result[key] = value
		}
	}

	return result
}

// FromMap populates the device MAC port from a map representation
func (d *MistDeviceMacPort) FromMap(data map[string]interface{}) error {
	if data == nil {
		return fmt.Errorf("data cannot be nil")
	}

	// Store raw data for complete preservation
	d.Raw = make(map[string]interface{})
	for k, v := range data {
		d.Raw[k] = v
	}

	// Initialize additional config if nil
	if d.AdditionalConfig == nil {
		d.AdditionalConfig = make(map[string]interface{})
	}

	// Extract typed fields
	if deviceMAC, ok := data["device_mac"].(string); ok {
		d.DeviceMAC = &deviceMAC
	}
	if portID, ok := data["port_id"].(string); ok {
		d.PortID = &portID
	}
	if start, ok := data["start"].(string); ok {
		d.Start = &start
	}
	if when, ok := data["when"].(string); ok {
		d.When = &when
	}
	if ip, ok := data["ip"].(string); ok {
		d.IP = &ip
	}
	if ip6, ok := data["ip6"].(string); ok {
		d.IP6 = &ip6
	}
	if vlan, ok := data["vlan"].(float64); ok {
		vlanInt := int(vlan)
		d.VLAN = &vlanInt
	} else if vlan, ok := data["vlan"].(int); ok {
		d.VLAN = &vlan
	}
	if portParent, ok := data["port_parent"].(string); ok {
		d.PortParent = &portParent
	}
	if node, ok := data["node"].(string); ok {
		d.Node = &node
	}

	// Store any additional fields in AdditionalConfig
	knownFields := map[string]bool{
		"device_mac":  true,
		"port_id":     true,
		"start":       true,
		"when":        true,
		"ip":          true,
		"ip6":         true,
		"vlan":        true,
		"port_parent": true,
		"node":        true,
	}

	for key, value := range data {
		if !knownFields[key] {
			d.AdditionalConfig[key] = value
		}
	}

	return nil
}

// MistWiredClient Methods

// GetMAC returns the wired client MAC address, normalized
func (w *MistWiredClient) GetMAC() string {
	if w.ClientMAC != nil {
		normalized, err := macaddr.Normalize(*w.ClientMAC)
		if err != nil {
			return *w.ClientMAC // Return original if normalization fails
		}
		return normalized
	}
	if w.MAC != nil {
		normalized, err := macaddr.Normalize(*w.MAC)
		if err != nil {
			return *w.MAC // Return original if normalization fails
		}
		return normalized
	}
	return ""
}

// GetOrgID returns the wired client organization ID
func (w *MistWiredClient) GetOrgID() string {
	if w.OrgID != nil {
		return *w.OrgID
	}
	return ""
}

// GetSiteID returns the wired client site ID
func (w *MistWiredClient) GetSiteID() string {
	if w.SiteID != nil {
		return *w.SiteID
	}
	return ""
}

// ToMap converts the wired client to a map representation
func (w *MistWiredClient) ToMap() map[string]interface{} {
	result := make(map[string]interface{})

	// Add typed fields
	if w.OrgID != nil {
		result["org_id"] = *w.OrgID
	}
	if w.SiteID != nil {
		result["site_id"] = *w.SiteID
	}
	if w.ClientMAC != nil {
		result["client_mac"] = *w.ClientMAC
	}
	if w.MAC != nil {
		result["mac"] = *w.MAC
	}
	if w.Timestamp != nil {
		result["timestamp"] = *w.Timestamp
	}

	// Add device MAC port information
	if len(w.DeviceMacPort) > 0 {
		var deviceMacPortList []map[string]interface{}
		for _, dmp := range w.DeviceMacPort {
			deviceMacPortList = append(deviceMacPortList, dmp.ToMap())
		}
		result["device_mac_port"] = deviceMacPortList
	}

	// Add string arrays
	if len(w.DeviceMAC) > 0 {
		result["device_mac"] = w.DeviceMAC
	}
	if len(w.PortID) > 0 {
		result["port_id"] = w.PortID
	}
	if len(w.IP) > 0 {
		result["ip"] = w.IP
	}
	if len(w.IP6) > 0 {
		result["ip6"] = w.IP6
	}
	if len(w.Hostname) > 0 {
		result["hostname"] = w.Hostname
	}
	if len(w.Username) > 0 {
		result["username"] = w.Username
	}

	// Add int arrays
	if len(w.VLAN) > 0 {
		result["vlan"] = w.VLAN
	}

	// Add optional fields
	if w.AuthState != nil {
		result["auth_state"] = *w.AuthState
	}
	if w.AuthMethod != nil {
		result["auth_method"] = *w.AuthMethod
	}
	if w.RandomMAC != nil {
		result["random_mac"] = *w.RandomMAC
	}
	if w.Manufacture != nil {
		result["manufacture"] = *w.Manufacture
	}

	// Add last known information
	if w.LastVLANName != nil {
		result["last_vlan_name"] = *w.LastVLANName
	}
	if w.LastVLAN != nil {
		result["last_vlan"] = *w.LastVLAN
	}
	if w.LastPortID != nil {
		result["last_port_id"] = *w.LastPortID
	}
	if w.LastHostname != nil {
		result["last_hostname"] = *w.LastHostname
	}
	if w.LastIP != nil {
		result["last_ip"] = *w.LastIP
	}
	if w.LastIP6 != nil {
		result["last_ip6"] = *w.LastIP6
	}
	if w.LastDeviceMAC != nil {
		result["last_device_mac"] = *w.LastDeviceMAC
	}

	// Add additional configuration
	for key, value := range w.AdditionalConfig {
		result[key] = value
	}

	// Add any raw data that wasn't captured above
	for key, value := range w.Raw {
		if _, exists := result[key]; !exists {
			result[key] = value
		}
	}

	return result
}

// FromMap populates the wired client from a map representation
func (w *MistWiredClient) FromMap(data map[string]interface{}) error {
	if data == nil {
		return fmt.Errorf("data cannot be nil")
	}

	// Store raw data for complete preservation
	w.Raw = make(map[string]interface{})
	for k, v := range data {
		w.Raw[k] = v
	}

	// Initialize additional config if nil
	if w.AdditionalConfig == nil {
		w.AdditionalConfig = make(map[string]interface{})
	}

	// Extract typed fields
	if orgID, ok := data["org_id"].(string); ok {
		w.OrgID = &orgID
	}
	if siteID, ok := data["site_id"].(string); ok {
		w.SiteID = &siteID
	}
	if clientMAC, ok := data["client_mac"].(string); ok {
		w.ClientMAC = &clientMAC
	}
	if mac, ok := data["mac"].(string); ok {
		w.MAC = &mac
	}
	if timestamp, ok := data["timestamp"].(float64); ok {
		w.Timestamp = &timestamp
	}

	// Handle device MAC port array
	if deviceMacPortData, ok := data["device_mac_port"].([]interface{}); ok {
		for _, item := range deviceMacPortData {
			if itemMap, ok := item.(map[string]interface{}); ok {
				dmp := &MistDeviceMacPort{
					AdditionalConfig: make(map[string]interface{}),
				}
				if err := dmp.FromMap(itemMap); err == nil {
					w.DeviceMacPort = append(w.DeviceMacPort, dmp)
				}
			}
		}
	}

	// Handle string arrays
	if deviceMAC, ok := data["device_mac"].([]interface{}); ok {
		for _, item := range deviceMAC {
			if str, ok := item.(string); ok {
				w.DeviceMAC = append(w.DeviceMAC, str)
			}
		}
	}
	if portID, ok := data["port_id"].([]interface{}); ok {
		for _, item := range portID {
			if str, ok := item.(string); ok {
				w.PortID = append(w.PortID, str)
			}
		}
	}
	if ip, ok := data["ip"].([]interface{}); ok {
		for _, item := range ip {
			if str, ok := item.(string); ok {
				w.IP = append(w.IP, str)
			}
		}
	}
	if ip6, ok := data["ip6"].([]interface{}); ok {
		for _, item := range ip6 {
			if str, ok := item.(string); ok {
				w.IP6 = append(w.IP6, str)
			}
		}
	}
	if hostname, ok := data["hostname"].([]interface{}); ok {
		for _, item := range hostname {
			if str, ok := item.(string); ok {
				w.Hostname = append(w.Hostname, str)
			}
		}
	}
	if username, ok := data["username"].([]interface{}); ok {
		for _, item := range username {
			if str, ok := item.(string); ok {
				w.Username = append(w.Username, str)
			}
		}
	}

	// Handle int arrays
	if vlan, ok := data["vlan"].([]interface{}); ok {
		for _, item := range vlan {
			if val, ok := item.(float64); ok {
				w.VLAN = append(w.VLAN, int(val))
			} else if val, ok := item.(int); ok {
				w.VLAN = append(w.VLAN, val)
			}
		}
	}

	// Handle optional fields
	if authState, ok := data["auth_state"].(string); ok {
		w.AuthState = &authState
	}
	if authMethod, ok := data["auth_method"].(string); ok {
		w.AuthMethod = &authMethod
	}
	if randomMAC, ok := data["random_mac"].(bool); ok {
		w.RandomMAC = &randomMAC
	}
	if manufacture, ok := data["manufacture"].(string); ok {
		w.Manufacture = &manufacture
	}

	// Handle last known information
	if lastVLANName, ok := data["last_vlan_name"].(string); ok {
		w.LastVLANName = &lastVLANName
	}
	if lastVLAN, ok := data["last_vlan"].(float64); ok {
		lastVLANInt := int(lastVLAN)
		w.LastVLAN = &lastVLANInt
	} else if lastVLAN, ok := data["last_vlan"].(int); ok {
		w.LastVLAN = &lastVLAN
	}
	if lastPortID, ok := data["last_port_id"].(string); ok {
		w.LastPortID = &lastPortID
	}
	if lastHostname, ok := data["last_hostname"].(string); ok {
		w.LastHostname = &lastHostname
	}
	if lastIP, ok := data["last_ip"].(string); ok {
		w.LastIP = &lastIP
	}
	if lastIP6, ok := data["last_ip6"].(string); ok {
		w.LastIP6 = &lastIP6
	}
	if lastDeviceMAC, ok := data["last_device_mac"].(string); ok {
		w.LastDeviceMAC = &lastDeviceMAC
	}

	// Store any additional fields in AdditionalConfig
	knownFields := map[string]bool{
		"org_id":          true,
		"site_id":         true,
		"client_mac":      true,
		"mac":             true,
		"timestamp":       true,
		"device_mac_port": true,
		"device_mac":      true,
		"port_id":         true,
		"ip":              true,
		"ip6":             true,
		"hostname":        true,
		"username":        true,
		"vlan":            true,
		"auth_state":      true,
		"auth_method":     true,
		"random_mac":      true,
		"manufacture":     true,
		"last_vlan_name":  true,
		"last_vlan":       true,
		"last_port_id":    true,
		"last_hostname":   true,
		"last_ip":         true,
		"last_ip6":        true,
		"last_device_mac": true,
	}

	for key, value := range data {
		if !knownFields[key] {
			w.AdditionalConfig[key] = value
		}
	}

	return nil
}

// GetRaw returns the raw API data
func (w *MistWiredClient) GetRaw() map[string]interface{} {
	if w.Raw == nil {
		return make(map[string]interface{})
	}
	// Return a copy to prevent external modifications
	result := make(map[string]interface{})
	for k, v := range w.Raw {
		result[k] = v
	}
	return result
}

// SetRaw sets the raw API data
func (w *MistWiredClient) SetRaw(data map[string]interface{}) {
	w.Raw = make(map[string]interface{})
	for k, v := range data {
		w.Raw[k] = v
	}
}

// MistWirelessClient Methods

// GetMAC returns the wireless client MAC address, normalized
func (w *MistWirelessClient) GetMAC() string {
	if w.MAC != nil {
		normalized, err := macaddr.Normalize(*w.MAC)
		if err != nil {
			return *w.MAC // Return original if normalization fails
		}
		return normalized
	}
	return ""
}

// GetOrgID returns the wireless client organization ID
func (w *MistWirelessClient) GetOrgID() string {
	if w.OrgID != nil {
		return *w.OrgID
	}
	return ""
}

// GetSiteID returns the wireless client site ID
func (w *MistWirelessClient) GetSiteID() string {
	if w.SiteID != nil {
		return *w.SiteID
	}
	return ""
}

// ToMap converts the wireless client to a map representation
func (w *MistWirelessClient) ToMap() map[string]interface{} {
	result := make(map[string]interface{})

	// Add typed fields
	if w.OrgID != nil {
		result["org_id"] = *w.OrgID
	}
	if w.SiteID != nil {
		result["site_id"] = *w.SiteID
	}
	if len(w.SiteIDs) > 0 {
		result["site_ids"] = w.SiteIDs
	}
	if w.MAC != nil {
		result["mac"] = *w.MAC
	}
	if w.Timestamp != nil {
		result["timestamp"] = *w.Timestamp
	}

	// Add string arrays
	if len(w.AP) > 0 {
		result["ap"] = w.AP
	}
	if len(w.IP) > 0 {
		result["ip"] = w.IP
	}
	if len(w.Hostname) > 0 {
		result["hostname"] = w.Hostname
	}
	if len(w.WLANID) > 0 {
		result["wlan_id"] = w.WLANID
	}
	if len(w.SSID) > 0 {
		result["ssid"] = w.SSID
	}
	if len(w.Model) > 0 {
		result["model"] = w.Model
	}
	if len(w.Device) > 0 {
		result["device"] = w.Device
	}
	if len(w.OS) > 0 {
		result["os"] = w.OS
	}
	if len(w.OSVersion) > 0 {
		result["os_version"] = w.OSVersion
	}
	if len(w.Username) > 0 {
		result["username"] = w.Username
	}
	if len(w.Firmware) > 0 {
		result["firmware"] = w.Firmware
	}
	if len(w.SDKVersion) > 0 {
		result["sdk_version"] = w.SDKVersion
	}
	if len(w.AppVersion) > 0 {
		result["app_version"] = w.AppVersion
	}
	if len(w.PskID) > 0 {
		result["psk_id"] = w.PskID
	}
	if len(w.PskName) > 0 {
		result["psk_name"] = w.PskName
	}

	// Add int arrays
	if len(w.VLAN) > 0 {
		result["vlan"] = w.VLAN
	}

	// Add optional fields
	if w.Manufacture != nil {
		result["mfg"] = *w.Manufacture
	}
	if w.Hardware != nil {
		result["hardware"] = *w.Hardware
	}
	if w.RandomMAC != nil {
		result["random_mac"] = *w.RandomMAC
	}
	if w.Ftc != nil {
		result["ftc"] = *w.Ftc
	}
	if w.Band != nil {
		result["band"] = *w.Band
	}
	if w.Protocol != nil {
		result["protocol"] = *w.Protocol
	}

	// Add last known information
	if w.LastAP != nil {
		result["last_ap"] = *w.LastAP
	}
	if w.LastIP != nil {
		result["last_ip"] = *w.LastIP
	}
	if w.LastHostname != nil {
		result["last_hostname"] = *w.LastHostname
	}
	if w.LastWLANID != nil {
		result["last_wlan_id"] = *w.LastWLANID
	}
	if w.LastSSID != nil {
		result["last_ssid"] = *w.LastSSID
	}
	if w.LastModel != nil {
		result["last_model"] = *w.LastModel
	}
	if w.LastDevice != nil {
		result["last_device"] = *w.LastDevice
	}
	if w.LastOS != nil {
		result["last_os"] = *w.LastOS
	}
	if w.LastOSVersion != nil {
		result["last_os_version"] = *w.LastOSVersion
	}
	if w.LastFirmware != nil {
		result["last_firmware"] = *w.LastFirmware
	}
	if w.LastVLAN != nil {
		result["last_vlan"] = *w.LastVLAN
	}

	// Add additional configuration
	for key, value := range w.AdditionalConfig {
		result[key] = value
	}

	// Add any raw data that wasn't captured above
	for key, value := range w.Raw {
		if _, exists := result[key]; !exists {
			result[key] = value
		}
	}

	return result
}

// FromMap populates the wireless client from a map representation
func (w *MistWirelessClient) FromMap(data map[string]interface{}) error {
	if data == nil {
		return fmt.Errorf("data cannot be nil")
	}

	// Store raw data for complete preservation
	w.Raw = make(map[string]interface{})
	for k, v := range data {
		w.Raw[k] = v
	}

	// Initialize additional config if nil
	if w.AdditionalConfig == nil {
		w.AdditionalConfig = make(map[string]interface{})
	}

	// Extract typed fields
	if orgID, ok := data["org_id"].(string); ok {
		w.OrgID = &orgID
	}
	if siteID, ok := data["site_id"].(string); ok {
		w.SiteID = &siteID
	}
	if siteIDs, ok := data["site_ids"].([]interface{}); ok {
		for _, item := range siteIDs {
			if str, ok := item.(string); ok {
				w.SiteIDs = append(w.SiteIDs, str)
			}
		}
	}
	if mac, ok := data["mac"].(string); ok {
		w.MAC = &mac
	}
	if timestamp, ok := data["timestamp"].(float64); ok {
		w.Timestamp = &timestamp
	}

	// Handle string arrays
	stringArrayFields := map[string]*[]string{
		"ap":          &w.AP,
		"ip":          &w.IP,
		"hostname":    &w.Hostname,
		"wlan_id":     &w.WLANID,
		"ssid":        &w.SSID,
		"model":       &w.Model,
		"device":      &w.Device,
		"os":          &w.OS,
		"os_version":  &w.OSVersion,
		"username":    &w.Username,
		"firmware":    &w.Firmware,
		"sdk_version": &w.SDKVersion,
		"app_version": &w.AppVersion,
		"psk_id":      &w.PskID,
		"psk_name":    &w.PskName,
	}

	for fieldName, fieldPtr := range stringArrayFields {
		if arrayData, ok := data[fieldName].([]interface{}); ok {
			var stringArray []string
			for _, item := range arrayData {
				if str, ok := item.(string); ok {
					stringArray = append(stringArray, str)
				}
			}
			*fieldPtr = stringArray
		}
	}

	// Handle int arrays
	if vlan, ok := data["vlan"].([]interface{}); ok {
		for _, item := range vlan {
			if val, ok := item.(float64); ok {
				w.VLAN = append(w.VLAN, int(val))
			} else if val, ok := item.(int); ok {
				w.VLAN = append(w.VLAN, val)
			}
		}
	}

	// Handle optional fields
	if manufacture, ok := data["mfg"].(string); ok {
		w.Manufacture = &manufacture
	}
	if hardware, ok := data["hardware"].(string); ok {
		w.Hardware = &hardware
	}
	if randomMAC, ok := data["random_mac"].(bool); ok {
		w.RandomMAC = &randomMAC
	}
	if ftc, ok := data["ftc"].(bool); ok {
		w.Ftc = &ftc
	}
	if band, ok := data["band"].(string); ok {
		w.Band = &band
	}
	if protocol, ok := data["protocol"].(string); ok {
		w.Protocol = &protocol
	}

	// Handle last known information
	if lastAP, ok := data["last_ap"].(string); ok {
		w.LastAP = &lastAP
	}
	if lastIP, ok := data["last_ip"].(string); ok {
		w.LastIP = &lastIP
	}
	if lastHostname, ok := data["last_hostname"].(string); ok {
		w.LastHostname = &lastHostname
	}
	if lastWLANID, ok := data["last_wlan_id"].(string); ok {
		w.LastWLANID = &lastWLANID
	}
	if lastSSID, ok := data["last_ssid"].(string); ok {
		w.LastSSID = &lastSSID
	}
	if lastModel, ok := data["last_model"].(string); ok {
		w.LastModel = &lastModel
	}
	if lastDevice, ok := data["last_device"].(string); ok {
		w.LastDevice = &lastDevice
	}
	if lastOS, ok := data["last_os"].(string); ok {
		w.LastOS = &lastOS
	}
	if lastOSVersion, ok := data["last_os_version"].(string); ok {
		w.LastOSVersion = &lastOSVersion
	}
	if lastFirmware, ok := data["last_firmware"].(string); ok {
		w.LastFirmware = &lastFirmware
	}
	if lastVLAN, ok := data["last_vlan"].(float64); ok {
		lastVLANInt := int(lastVLAN)
		w.LastVLAN = &lastVLANInt
	} else if lastVLAN, ok := data["last_vlan"].(int); ok {
		w.LastVLAN = &lastVLAN
	}

	// Store any additional fields in AdditionalConfig
	knownFields := map[string]bool{
		"org_id":          true,
		"site_id":         true,
		"site_ids":        true,
		"mac":             true,
		"timestamp":       true,
		"ap":              true,
		"ip":              true,
		"hostname":        true,
		"wlan_id":         true,
		"ssid":            true,
		"model":           true,
		"device":          true,
		"os":              true,
		"os_version":      true,
		"username":        true,
		"firmware":        true,
		"sdk_version":     true,
		"app_version":     true,
		"psk_id":          true,
		"psk_name":        true,
		"vlan":            true,
		"mfg":             true,
		"hardware":        true,
		"random_mac":      true,
		"ftc":             true,
		"band":            true,
		"protocol":        true,
		"last_ap":         true,
		"last_ip":         true,
		"last_hostname":   true,
		"last_wlan_id":    true,
		"last_ssid":       true,
		"last_model":      true,
		"last_device":     true,
		"last_os":         true,
		"last_os_version": true,
		"last_firmware":   true,
		"last_vlan":       true,
	}

	for key, value := range data {
		if !knownFields[key] {
			w.AdditionalConfig[key] = value
		}
	}

	return nil
}

// GetRaw returns the raw API data
func (w *MistWirelessClient) GetRaw() map[string]interface{} {
	if w.Raw == nil {
		return make(map[string]interface{})
	}
	// Return a copy to prevent external modifications
	result := make(map[string]interface{})
	for k, v := range w.Raw {
		result[k] = v
	}
	return result
}

// SetRaw sets the raw API data
func (w *MistWirelessClient) SetRaw(data map[string]interface{}) {
	w.Raw = make(map[string]interface{})
	for k, v := range data {
		w.Raw[k] = v
	}
}

// Factory Methods

// NewWiredClientFromMap creates a new wired client from a map representation
func NewWiredClientFromMap(data map[string]interface{}) (*MistWiredClient, error) {
	client := &MistWiredClient{
		AdditionalConfig: make(map[string]interface{}),
	}

	if err := client.FromMap(data); err != nil {
		return nil, fmt.Errorf("failed to create wired client from map: %w", err)
	}

	return client, nil
}

// NewWirelessClientFromMap creates a new wireless client from a map representation
func NewWirelessClientFromMap(data map[string]interface{}) (*MistWirelessClient, error) {
	client := &MistWirelessClient{
		AdditionalConfig: make(map[string]interface{}),
	}

	if err := client.FromMap(data); err != nil {
		return nil, fmt.Errorf("failed to create wireless client from map: %w", err)
	}

	return client, nil
}

// Conversion Methods

// Verify types implement SearchClientMarshaler at compile time
var _ SearchClientMarshaler = (*MistWiredClient)(nil)
var _ SearchClientMarshaler = (*MistWirelessClient)(nil)

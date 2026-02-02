package netbox

import (
	"fmt"
	"strings"

	"github.com/ravinald/wifimgr/internal/macaddr"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// Mapper converts wifimgr types to NetBox request types
type Mapper struct {
	config    *Config
	validator *Validator
}

// NewMapper creates a new mapper
func NewMapper(config *Config, validator *Validator) *Mapper {
	return &Mapper{
		config:    config,
		validator: validator,
	}
}

// ToDeviceRequest converts an InventoryItem to a NetBox DeviceRequest.
// The validation result must be provided with valid IDs for site, device type, and role.
func (m *Mapper) ToDeviceRequest(item *vendors.InventoryItem, validation *DeviceValidationResult) (*DeviceRequest, error) {
	if !validation.Valid {
		return nil, fmt.Errorf("cannot map invalid device: %v", validation.Errors)
	}

	// Generate device name if not set
	name := item.Name
	if name == "" {
		name = m.generateDeviceName(item)
	}

	// Map status
	status := "active" // default status for new devices

	req := &DeviceRequest{
		Name:       name,
		DeviceType: validation.DeviceTypeID,
		Role:       validation.DeviceRoleID,
		Site:       validation.SiteID,
		Serial:     item.Serial,
		Status:     status,
	}

	// Add tag if configured
	if m.config.Mappings.Tag != "" {
		req.Tags = []Tag{{Name: m.config.Mappings.Tag}}
	}

	// Add custom fields with wifimgr metadata
	req.CustomFields = m.buildCustomFields(item)

	return req, nil
}

// ToInterfaceRequest creates an interface request for the device's primary interface.
// Returns an error if the interface type is invalid.
func (m *Mapper) ToInterfaceRequest(deviceID int64, item *vendors.InventoryItem) (*InterfaceRequest, error) {
	// Get effective interface mapping with priority:
	// 1. Device-level override (item.NetBox.Interfaces)
	// 2. Global config (m.config.GetInterfaceMapping)
	// 3. Defaults
	ifaceName, ifaceType := m.getEffectiveInterfaceMapping("eth0", item)

	// Apply device-type-specific defaults if values are still empty
	if ifaceName == "" {
		ifaceName = m.getPrimaryInterfaceName(item.Type)
	}
	if ifaceType == "" {
		ifaceType = m.getInterfaceType(item.Type)
	}

	// Validate interface type
	if err := ValidateInterfaceType(ifaceType); err != nil {
		if typeErr, ok := err.(*InterfaceTypeError); ok {
			typeErr.DeviceName = item.Name
		}
		return nil, err
	}

	// Format MAC address with colons (uppercase for NetBox)
	formattedMAC, _ := macaddr.Format(item.MAC, macaddr.FormatColon)
	formattedMAC = strings.ToUpper(formattedMAC)

	req := &InterfaceRequest{
		Device:  deviceID,
		Name:    ifaceName,
		Type:    ifaceType,
		MACAddr: formattedMAC,
		Enabled: true,
	}

	// Add tag if configured
	if m.config.Mappings.Tag != "" {
		req.Tags = []Tag{{Name: m.config.Mappings.Tag}}
	}

	return req, nil
}

// getEffectiveInterfaceMapping returns the interface name and type for an internal ID,
// checking device-level overrides first, then global config, then defaults.
func (m *Mapper) getEffectiveInterfaceMapping(internalID string, item *vendors.InventoryItem) (name, ifaceType string) {
	// Priority 1: Device-level override
	if item != nil && item.NetBox != nil && item.NetBox.Interfaces != nil {
		if deviceMapping, ok := item.NetBox.Interfaces[internalID]; ok && deviceMapping != nil {
			name = deviceMapping.Name
			ifaceType = deviceMapping.Type
			// If device-level provides both, return immediately
			if name != "" && ifaceType != "" {
				return name, ifaceType
			}
		}
	}

	// Priority 2: Global config
	globalMapping := m.config.GetInterfaceMapping(internalID)
	if globalMapping != nil {
		if name == "" {
			name = globalMapping.Name
		}
		if ifaceType == "" {
			ifaceType = globalMapping.Type
		}
	}

	return name, ifaceType
}

// ToIPAddressRequest creates an IP address request for the device
func (m *Mapper) ToIPAddressRequest(interfaceID int64, ipAddress string) *IPAddressRequest {
	// Ensure IP has CIDR notation
	if !strings.Contains(ipAddress, "/") {
		ipAddress = ipAddress + "/32"
	}

	return &IPAddressRequest{
		Address:            ipAddress,
		Status:             "active",
		AssignedObjectType: "dcim.interface",
		AssignedObjectID:   interfaceID,
	}
}

// generateDeviceName creates a device name from available fields
func (m *Mapper) generateDeviceName(item *vendors.InventoryItem) string {
	// Use MAC address as fallback name (last 6 chars of normalized MAC)
	normalizedMAC := macaddr.NormalizeOrEmpty(item.MAC)
	if normalizedMAC != "" && len(normalizedMAC) >= 6 {
		return fmt.Sprintf("%s-%s", strings.ToUpper(item.Type), strings.ToUpper(normalizedMAC[6:]))
	}

	// Last resort: use serial
	if item.Serial != "" {
		return fmt.Sprintf("%s-%s", strings.ToUpper(item.Type), item.Serial)
	}

	return fmt.Sprintf("%s-UNKNOWN", strings.ToUpper(item.Type))
}

// getInterfaceType returns the NetBox interface type for a device type.
// For APs, this returns the type for the eth0 management interface (Ethernet),
// not the wireless radio interfaces (wifi0/1/2).
func (m *Mapper) getInterfaceType(deviceType string) string {
	switch deviceType {
	case "ap":
		return "1000base-t" // Gigabit Ethernet management interface (eth0)
	case "switch":
		return "1000base-t" // Gigabit Ethernet management interface
	case "gateway":
		return "1000base-t" // Gigabit Ethernet
	default:
		return "other"
	}
}

// getPrimaryInterfaceName returns the primary interface name for a device type
func (m *Mapper) getPrimaryInterfaceName(deviceType string) string {
	switch deviceType {
	case "ap":
		return "eth0" // typical AP management interface
	case "switch":
		return "mgmt0" // typical switch management interface
	case "gateway":
		return "ge-0/0/0" // typical gateway interface
	default:
		return "eth0"
	}
}

// buildCustomFields creates custom fields with wifimgr metadata
func (m *Mapper) buildCustomFields(item *vendors.InventoryItem) map[string]any {
	fields := make(map[string]any)

	// Settings source indicates where device configuration comes from
	// "internal" = vendor API (Mist/Meraki), "netbox" = NetBox (future reverse sync)
	fields["settings_source"] = "internal"

	// Add source API information
	if item.SourceAPI != "" {
		fields["wifimgr_source_api"] = item.SourceAPI
	}
	if item.SourceVendor != "" {
		fields["wifimgr_source_vendor"] = item.SourceVendor
	}

	// Add original vendor ID for reference
	if item.ID != "" {
		fields["wifimgr_vendor_id"] = item.ID
	}

	return fields
}

// MapDeviceForUpdate creates an update request for an existing device
func (m *Mapper) MapDeviceForUpdate(item *vendors.InventoryItem, existingID int64, validation *DeviceValidationResult) (*DeviceRequest, error) {
	req, err := m.ToDeviceRequest(item, validation)
	if err != nil {
		return nil, err
	}

	// Add a comment noting the update source
	req.Comments = fmt.Sprintf("Updated from wifimgr (%s)", item.SourceVendor)

	return req, nil
}

// GetDeviceRoleSlug returns the mapped device role slug for a device type.
// Use GetDeviceRoleSlugForModel if you need model-specific or per-device role overrides.
func (m *Mapper) GetDeviceRoleSlug(deviceType string) string {
	return m.config.GetDeviceRoleSlug(deviceType)
}

// GetDeviceRoleSlugForModel returns the mapped device role slug with model-specific override support.
// The deviceNetBox parameter can be nil or a *NetBoxDeviceExtension for per-device role overrides.
func (m *Mapper) GetDeviceRoleSlugForModel(deviceType string, model string, deviceNetBox any) string {
	return m.config.GetDeviceRoleSlugForModel(deviceType, model, deviceNetBox)
}

// GetDeviceTypeSlug returns the mapped device type slug for a model
func (m *Mapper) GetDeviceTypeSlug(model string) string {
	return m.config.GetDeviceTypeSlug(model)
}

// GetSiteSlug returns the mapped site slug for a site name
func (m *Mapper) GetSiteSlug(siteName string) string {
	return m.config.GetSiteSlug(siteName)
}

// ToRadioInterfaceRequests creates requests for physical radio interfaces (wifi0, wifi1, wifi2)
// based on the AP's radio configuration from the vendor cache.
// Uses configured interface mappings for names and types.
// The item parameter is optional; if provided, device-level interface overrides are checked.
func (m *Mapper) ToRadioInterfaceRequests(deviceID int64, radioConfig *vendors.RadioConfig, item *vendors.InventoryItem) ([]*InterfaceRequest, error) {
	var requests []*InterfaceRequest
	tag := m.getTag()

	// Radio mapping: internal ID -> band config, default name, default type, description
	radioMappings := []struct {
		internalID  string
		bandConfig  *vendors.RadioBandConfig
		defaultName string
		defaultType string
		description string
	}{
		{"radio0", radioConfig.Band24, "wifi0", "ieee802.11n", "2.4 GHz radio"},
		{"radio1", radioConfig.Band5, "wifi1", "ieee802.11ac", "5 GHz radio"},
		{"radio2", radioConfig.Band6, "wifi2", "ieee802.11ax", "6 GHz radio"},
	}

	for _, rm := range radioMappings {
		if rm.bandConfig == nil {
			continue
		}

		// Get effective mapping with priority: device-level -> global config -> defaults
		ifaceName, ifaceType := m.getEffectiveInterfaceMapping(rm.internalID, item)
		if ifaceName == "" {
			ifaceName = rm.defaultName
		}
		if ifaceType == "" {
			ifaceType = rm.defaultType
		}

		// Validate interface type
		if err := ValidateInterfaceType(ifaceType); err != nil {
			return nil, err
		}

		enabled := rm.bandConfig.Disabled == nil || !*rm.bandConfig.Disabled
		requests = append(requests, &InterfaceRequest{
			Device:      deviceID,
			Name:        ifaceName,
			Type:        ifaceType,
			Enabled:     enabled,
			RFRole:      "ap",
			Tags:        tag,
			Description: rm.description,
		})
	}

	return requests, nil
}

// ToVirtualWLANInterfaceRequests creates virtual interface requests for WLANs
// linked to the appropriate parent radio interfaces.
func (m *Mapper) ToVirtualWLANInterfaceRequests(
	deviceID int64,
	radioInterfaces map[string]int64, // "wifi0" -> interface ID
	wlans []*vendors.WLAN,
	wirelessLANIDs map[string]int64, // SSID -> WirelessLAN ID
) []*InterfaceRequest {
	var requests []*InterfaceRequest
	tag := m.getTag()

	for i, wlan := range wlans {
		radios := m.getRadiosForWLAN(wlan.Band)

		for _, radioName := range radios {
			parentID, ok := radioInterfaces[radioName]
			if !ok {
				continue // Radio doesn't exist on this AP
			}

			// Generate virtual interface name: wifi0.0, wifi0.1, wifi1.0, etc.
			radioNum := strings.TrimPrefix(radioName, "wifi")
			ifaceName := fmt.Sprintf("wifi%s.%d", radioNum, i)

			req := &InterfaceRequest{
				Device:      deviceID,
				Name:        ifaceName,
				Type:        "virtual",
				Enabled:     wlan.Enabled,
				Parent:      &parentID,
				Tags:        tag,
				Description: fmt.Sprintf("WLAN: %s", wlan.SSID),
			}

			// Link to NetBox WirelessLAN object if it exists
			if wlanID, ok := wirelessLANIDs[wlan.SSID]; ok {
				req.WirelessLANs = []int64{wlanID}
			}

			requests = append(requests, req)
		}
	}

	return requests
}

// ToWirelessLANRequest creates a NetBox WirelessLAN request from a vendor WLAN
func (m *Mapper) ToWirelessLANRequest(wlan *vendors.WLAN) *WirelessLANRequest {
	req := &WirelessLANRequest{
		SSID:   wlan.SSID,
		Status: "active",
	}

	// Map authentication type
	switch strings.ToLower(wlan.AuthType) {
	case "open":
		req.AuthType = "open"
	case "psk", "wpa2-personal", "wpa3-personal", "wpa2/wpa3-personal":
		req.AuthType = "wpa-personal"
	case "wpa2-enterprise", "wpa3-enterprise", "802.1x":
		req.AuthType = "wpa-enterprise"
	default:
		if wlan.AuthType != "" {
			req.AuthType = "wpa-personal" // Default to PSK if unknown
		}
	}

	// Map encryption cipher
	switch strings.ToLower(wlan.EncryptionMode) {
	case "wpa2":
		req.AuthCipher = "aes"
	case "wpa3", "wpa2/wpa3":
		req.AuthCipher = "aes"
	default:
		req.AuthCipher = "auto"
	}

	// Add tag if configured
	if m.config.Mappings.Tag != "" {
		req.Tags = []Tag{{Name: m.config.Mappings.Tag}}
	}

	return req
}

// getRadiosForWLAN returns which radio interfaces a WLAN should be associated with.
// Uses configured interface names from mappings.
func (m *Mapper) getRadiosForWLAN(band string) []string {
	// Get configured radio names
	radio0Name := "wifi0"
	radio1Name := "wifi1"
	radio2Name := "wifi2"

	if mapping := m.config.GetInterfaceMapping("radio0"); mapping != nil && mapping.Name != "" {
		radio0Name = mapping.Name
	}
	if mapping := m.config.GetInterfaceMapping("radio1"); mapping != nil && mapping.Name != "" {
		radio1Name = mapping.Name
	}
	if mapping := m.config.GetInterfaceMapping("radio2"); mapping != nil && mapping.Name != "" {
		radio2Name = mapping.Name
	}

	switch strings.ToLower(band) {
	case "2.4":
		return []string{radio0Name}
	case "5":
		return []string{radio1Name}
	case "6":
		return []string{radio2Name}
	case "dual":
		return []string{radio0Name, radio1Name} // 2.4 and 5 GHz
	case "all", "":
		return []string{radio0Name, radio1Name} // Default to dual-band
	default:
		return []string{radio0Name, radio1Name} // Default to dual-band
	}
}

// getTag returns the configured tag slice
func (m *Mapper) getTag() []Tag {
	if m.config.Mappings.Tag != "" {
		return []Tag{{Name: m.config.Mappings.Tag}}
	}
	return nil
}

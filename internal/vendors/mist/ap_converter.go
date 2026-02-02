// Package mist provides Mist-specific API conversions.
package mist

import (
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// ToMistAPConfig converts a vendor-agnostic APDeviceConfig to Mist API format.
// It applies Mist-specific field mappings and merges any Mist extension block.
func ToMistAPConfig(cfg *vendors.APDeviceConfig) map[string]any {
	if cfg == nil {
		return nil
	}

	result := cfg.ToMap()

	// Mist-specific transformations
	// Most fields map 1:1 since we use Mist nomenclature as standard

	// Handle radio_config -> radio field naming for Mist API
	if radioConfig, ok := result["radio_config"].(map[string]any); ok {
		// Mist uses "radio" at the top level for radio configuration
		// Keep the structure as-is since it follows Mist naming
		result["radio_config"] = radioConfig
	}

	// Handle LED config - Mist uses "led" as a nested object
	if ledConfig, ok := result["led"].(map[string]any); ok {
		if enabled, ok := ledConfig["enabled"]; ok {
			// Mist API accepts led.enabled
			result["led"] = map[string]any{"enabled": enabled}
		}
	}

	return result
}

// FromMistAPConfig converts Mist API response to vendor-agnostic APDeviceConfig.
// Returns the configuration and a slice of warnings (type assertion failures, unexpected fields, etc.).
func FromMistAPConfig(data map[string]any, mac string) (*vendors.APDeviceConfig, []error) {
	if data == nil {
		return nil, nil
	}

	cfg := &vendors.APDeviceConfig{}
	var warnings []error
	logger := logging.GetLogger()

	// Known fields for unexpected field detection
	knownFields := map[string]bool{
		"name": true, "tags": true, "notes": true,
		"location": true, "orientation": true, "map_id": true,
		"x": true, "y": true, "height": true,
		"deviceprofile_id": true, "vars": true,
		"radio_config": true, "ip_config": true, "ble_config": true,
		"mesh": true, "port_config": true, "led": true, "pwr_config": true,
		"disable_eth1": true, "disable_eth2": true, "disable_eth3": true,
		"poe_passthrough": true,
		// Status fields that appear in API responses but aren't configuration
		"id": true, "site_id": true, "org_id": true, "serial": true,
		"model": true, "type": true, "mac": true, "created_time": true,
		"modified_time": true, "status": true, "last_seen": true,
		"uptime": true, "version": true, "connected": true, "magic": true,
	}

	// Identity - using safe type converters
	if name, err := vendors.SafeString(data, "name", logger); err != nil {
		if fme, ok := err.(*vendors.FieldMappingError); ok {
			fme.Vendor = "mist"
			fme.DeviceMAC = mac
		}
		warnings = append(warnings, err)
	} else {
		cfg.Name = name
	}

	if tags, err := vendors.SafeStringSlice(data, "tags", logger); err != nil {
		if fme, ok := err.(*vendors.FieldMappingError); ok {
			fme.Vendor = "mist"
			fme.DeviceMAC = mac
		}
		warnings = append(warnings, err)
	} else {
		cfg.Tags = tags
	}

	if notes, err := vendors.SafeString(data, "notes", logger); err != nil {
		if fme, ok := err.(*vendors.FieldMappingError); ok {
			fme.Vendor = "mist"
			fme.DeviceMAC = mac
		}
		warnings = append(warnings, err)
	} else {
		cfg.Notes = notes
	}

	// Location - special handling for array
	if loc, ok := data["location"].([]any); ok {
		cfg.Location = interfaceSliceToFloat64Slice(loc)
	}

	if orientation, err := vendors.SafeInt(data, "orientation", logger); err != nil {
		if fme, ok := err.(*vendors.FieldMappingError); ok {
			fme.Vendor = "mist"
			fme.DeviceMAC = mac
		}
		warnings = append(warnings, err)
	} else {
		cfg.Orientation = orientation
	}

	if mapID, err := vendors.SafeString(data, "map_id", logger); err != nil {
		if fme, ok := err.(*vendors.FieldMappingError); ok {
			fme.Vendor = "mist"
			fme.DeviceMAC = mac
		}
		warnings = append(warnings, err)
	} else {
		cfg.MapID = mapID
	}

	if x, err := vendors.SafeFloat64(data, "x", logger); err != nil {
		if fme, ok := err.(*vendors.FieldMappingError); ok {
			fme.Vendor = "mist"
			fme.DeviceMAC = mac
		}
		warnings = append(warnings, err)
	} else {
		cfg.X = x
	}

	if y, err := vendors.SafeFloat64(data, "y", logger); err != nil {
		if fme, ok := err.(*vendors.FieldMappingError); ok {
			fme.Vendor = "mist"
			fme.DeviceMAC = mac
		}
		warnings = append(warnings, err)
	} else {
		cfg.Y = y
	}

	if height, err := vendors.SafeFloat64(data, "height", logger); err != nil {
		if fme, ok := err.(*vendors.FieldMappingError); ok {
			fme.Vendor = "mist"
			fme.DeviceMAC = mac
		}
		warnings = append(warnings, err)
	} else {
		cfg.Height = height
	}

	// Device profile
	if dpID, err := vendors.SafeString(data, "deviceprofile_id", logger); err != nil {
		if fme, ok := err.(*vendors.FieldMappingError); ok {
			fme.Vendor = "mist"
			fme.DeviceMAC = mac
		}
		warnings = append(warnings, err)
	} else {
		cfg.DeviceProfileID = dpID
	}

	// Variables
	if vars, err := vendors.SafeMap(data, "vars", logger); err != nil {
		if fme, ok := err.(*vendors.FieldMappingError); ok {
			fme.Vendor = "mist"
			fme.DeviceMAC = mac
		}
		warnings = append(warnings, err)
	} else {
		cfg.Vars = vars
	}

	// Radio config
	if radioConfig, err := vendors.SafeMap(data, "radio_config", logger); err != nil {
		if fme, ok := err.(*vendors.FieldMappingError); ok {
			fme.Vendor = "mist"
			fme.DeviceMAC = mac
		}
		warnings = append(warnings, err)
	} else if radioConfig != nil {
		cfg.RadioConfig = parseRadioConfig(radioConfig)
	}

	// IP config
	if ipConfig, err := vendors.SafeMap(data, "ip_config", logger); err != nil {
		if fme, ok := err.(*vendors.FieldMappingError); ok {
			fme.Vendor = "mist"
			fme.DeviceMAC = mac
		}
		warnings = append(warnings, err)
	} else if ipConfig != nil {
		cfg.IPConfig = parseIPConfig(ipConfig)
	}

	// BLE config
	if bleConfig, err := vendors.SafeMap(data, "ble_config", logger); err != nil {
		if fme, ok := err.(*vendors.FieldMappingError); ok {
			fme.Vendor = "mist"
			fme.DeviceMAC = mac
		}
		warnings = append(warnings, err)
	} else if bleConfig != nil {
		cfg.BLEConfig = parseBLEConfig(bleConfig)
	}

	// Mesh config
	if mesh, err := vendors.SafeMap(data, "mesh", logger); err != nil {
		if fme, ok := err.(*vendors.FieldMappingError); ok {
			fme.Vendor = "mist"
			fme.DeviceMAC = mac
		}
		warnings = append(warnings, err)
	} else if mesh != nil {
		cfg.MeshConfig = parseMeshConfig(mesh)
	}

	// Port config - special handling for array
	if portConfig, ok := data["port_config"].([]any); ok {
		cfg.PortConfig = parsePortConfigList(portConfig)
	}

	// LED config
	if led, err := vendors.SafeMap(data, "led", logger); err != nil {
		if fme, ok := err.(*vendors.FieldMappingError); ok {
			fme.Vendor = "mist"
			fme.DeviceMAC = mac
		}
		warnings = append(warnings, err)
	} else if led != nil {
		cfg.LEDConfig = parseLEDConfig(led)
	}

	// Power config
	if pwrConfig, err := vendors.SafeMap(data, "pwr_config", logger); err != nil {
		if fme, ok := err.(*vendors.FieldMappingError); ok {
			fme.Vendor = "mist"
			fme.DeviceMAC = mac
		}
		warnings = append(warnings, err)
	} else if pwrConfig != nil {
		cfg.PowerConfig = parsePowerConfig(pwrConfig)
	}

	// Hardware flags
	if disableEth1, err := vendors.SafeBool(data, "disable_eth1", logger); err != nil {
		if fme, ok := err.(*vendors.FieldMappingError); ok {
			fme.Vendor = "mist"
			fme.DeviceMAC = mac
		}
		warnings = append(warnings, err)
	} else {
		cfg.DisableEth1 = disableEth1
	}

	if disableEth2, err := vendors.SafeBool(data, "disable_eth2", logger); err != nil {
		if fme, ok := err.(*vendors.FieldMappingError); ok {
			fme.Vendor = "mist"
			fme.DeviceMAC = mac
		}
		warnings = append(warnings, err)
	} else {
		cfg.DisableEth2 = disableEth2
	}

	if disableEth3, err := vendors.SafeBool(data, "disable_eth3", logger); err != nil {
		if fme, ok := err.(*vendors.FieldMappingError); ok {
			fme.Vendor = "mist"
			fme.DeviceMAC = mac
		}
		warnings = append(warnings, err)
	} else {
		cfg.DisableEth3 = disableEth3
	}

	if poePassthrough, err := vendors.SafeBool(data, "poe_passthrough", logger); err != nil {
		if fme, ok := err.(*vendors.FieldMappingError); ok {
			fme.Vendor = "mist"
			fme.DeviceMAC = mac
		}
		warnings = append(warnings, err)
	} else {
		cfg.PoEPassthrough = poePassthrough
	}

	// Detect unexpected fields
	for field, value := range data {
		if !knownFields[field] {
			warnings = append(warnings, &vendors.UnexpectedFieldWarning{
				Vendor:    "mist",
				DeviceMAC: mac,
				Field:     field,
				Value:     value,
			})
		}
	}

	return cfg, warnings
}

// parseRadioConfig parses radio configuration from Mist API format
func parseRadioConfig(data map[string]any) *vendors.RadioConfig {
	if data == nil {
		return nil
	}

	cfg := &vendors.RadioConfig{}

	if allowRRM, ok := data["allow_rrm_disable"].(bool); ok {
		cfg.AllowRRMDisable = &allowRRM
	}
	if scanning, ok := data["scanning_enabled"].(bool); ok {
		cfg.ScanningEnabled = &scanning
	}
	if indoor, ok := data["indoor_use"].(bool); ok {
		cfg.IndoorUse = &indoor
	}
	if antGain24, ok := data["ant_gain_24"].(float64); ok {
		cfg.AntGain24 = &antGain24
	}
	if antGain5, ok := data["ant_gain_5"].(float64); ok {
		cfg.AntGain5 = &antGain5
	}
	if antGain6, ok := data["ant_gain_6"].(float64); ok {
		cfg.AntGain6 = &antGain6
	}
	if antennaMode, ok := data["antenna_mode"].(string); ok {
		cfg.AntennaMode = &antennaMode
	}
	if band24Usage, ok := data["band_24_usage"].(string); ok {
		cfg.Band24Usage = &band24Usage
	}

	if band24, ok := data["band_24"].(map[string]any); ok {
		cfg.Band24 = parseRadioBandConfig(band24)
	}
	if band5, ok := data["band_5"].(map[string]any); ok {
		cfg.Band5 = parseRadioBandConfig(band5)
	}
	if band5On24, ok := data["band_5_on_24_radio"].(map[string]any); ok {
		cfg.Band5On24Radio = parseRadioBandConfig(band5On24)
	}
	if band6, ok := data["band_6"].(map[string]any); ok {
		cfg.Band6 = parseRadioBandConfig(band6)
	}

	return cfg
}

// parseRadioBandConfig parses per-band radio configuration
func parseRadioBandConfig(data map[string]any) *vendors.RadioBandConfig {
	if data == nil {
		return nil
	}

	cfg := &vendors.RadioBandConfig{}

	if disabled, ok := data["disabled"].(bool); ok {
		cfg.Disabled = &disabled
	}
	if channel, ok := data["channel"].(float64); ok {
		c := int(channel)
		cfg.Channel = &c
	}
	if channels, ok := data["channels"].([]any); ok {
		cfg.Channels = interfaceSliceToIntSlice(channels)
	}
	if power, ok := data["power"].(float64); ok {
		p := int(power)
		cfg.Power = &p
	}
	if powerMin, ok := data["power_min"].(float64); ok {
		p := int(powerMin)
		cfg.PowerMin = &p
	}
	if powerMax, ok := data["power_max"].(float64); ok {
		p := int(powerMax)
		cfg.PowerMax = &p
	}
	if bandwidth, ok := data["bandwidth"].(float64); ok {
		b := int(bandwidth)
		cfg.Bandwidth = &b
	}
	if antennaMode, ok := data["antenna_mode"].(string); ok {
		cfg.AntennaMode = &antennaMode
	}
	if antGain, ok := data["ant_gain"].(float64); ok {
		cfg.AntGain = &antGain
	}
	if preamble, ok := data["preamble"].(string); ok {
		cfg.Preamble = &preamble
	}

	return cfg
}

// parseIPConfig parses IP configuration
func parseIPConfig(data map[string]any) *vendors.IPConfig {
	if data == nil {
		return nil
	}

	cfg := &vendors.IPConfig{}

	if typ, ok := data["type"].(string); ok {
		cfg.Type = &typ
	}
	if ip, ok := data["ip"].(string); ok {
		cfg.IP = &ip
	}
	if netmask, ok := data["netmask"].(string); ok {
		cfg.Netmask = &netmask
	}
	if gateway, ok := data["gateway"].(string); ok {
		cfg.Gateway = &gateway
	}
	if dns, ok := data["dns"].([]any); ok {
		cfg.DNS = interfaceSliceToStringSlice(dns)
	}
	if vlan, ok := data["vlan_id"].(float64); ok {
		v := int(vlan)
		cfg.VlanID = &v
	}

	return cfg
}

// parseBLEConfig parses BLE configuration
func parseBLEConfig(data map[string]any) *vendors.BLEConfig {
	if data == nil {
		return nil
	}

	cfg := &vendors.BLEConfig{}

	if enabled, ok := data["enabled"].(bool); ok {
		cfg.Enabled = &enabled
	}
	if power, ok := data["power"].(float64); ok {
		p := int(power)
		cfg.Power = &p
	}
	if mode, ok := data["mode"].(string); ok {
		cfg.Mode = &mode
	}

	if ibeacon, ok := data["ibeacon"].(map[string]any); ok {
		cfg.IBeacon = parseIBeaconConfig(ibeacon)
	}
	if eddystone, ok := data["eddystone"].(map[string]any); ok {
		cfg.Eddystone = parseEddystoneConfig(eddystone)
	}

	return cfg
}

// parseIBeaconConfig parses iBeacon configuration
func parseIBeaconConfig(data map[string]any) *vendors.IBeaconConfig {
	if data == nil {
		return nil
	}

	cfg := &vendors.IBeaconConfig{}

	if enabled, ok := data["enabled"].(bool); ok {
		cfg.Enabled = &enabled
	}
	if uuid, ok := data["uuid"].(string); ok {
		cfg.UUID = &uuid
	}
	if major, ok := data["major"].(float64); ok {
		m := int(major)
		cfg.Major = &m
	}
	if minor, ok := data["minor"].(float64); ok {
		m := int(minor)
		cfg.Minor = &m
	}
	if power, ok := data["power"].(float64); ok {
		p := int(power)
		cfg.Power = &p
	}

	return cfg
}

// parseEddystoneConfig parses Eddystone configuration
func parseEddystoneConfig(data map[string]any) *vendors.EddystoneConfig {
	if data == nil {
		return nil
	}

	cfg := &vendors.EddystoneConfig{}

	if enabled, ok := data["enabled"].(bool); ok {
		cfg.Enabled = &enabled
	}
	if nsID, ok := data["namespace_id"].(string); ok {
		cfg.NamespaceID = &nsID
	}
	if instID, ok := data["instance_id"].(string); ok {
		cfg.InstanceID = &instID
	}
	if url, ok := data["url"].(string); ok {
		cfg.URL = &url
	}

	return cfg
}

// parseMeshConfig parses mesh configuration
func parseMeshConfig(data map[string]any) *vendors.MeshConfig {
	if data == nil {
		return nil
	}

	cfg := &vendors.MeshConfig{}

	if enabled, ok := data["enabled"].(bool); ok {
		cfg.Enabled = &enabled
	}
	if role, ok := data["role"].(string); ok {
		cfg.Role = &role
	}
	if group, ok := data["group"].(string); ok {
		cfg.Group = &group
	}

	return cfg
}

// parsePortConfigList parses port configuration array
func parsePortConfigList(data []any) []vendors.PortConfig {
	if data == nil {
		return nil
	}

	result := make([]vendors.PortConfig, 0, len(data))
	for _, item := range data {
		if portData, ok := item.(map[string]any); ok {
			port := parsePortConfig(portData)
			result = append(result, port)
		}
	}
	return result
}

// parsePortConfig parses a single port configuration
func parsePortConfig(data map[string]any) vendors.PortConfig {
	cfg := vendors.PortConfig{}

	if portID, ok := data["port_id"].(string); ok {
		cfg.PortID = &portID
	}
	if enabled, ok := data["enabled"].(bool); ok {
		cfg.Enabled = &enabled
	}
	if mode, ok := data["mode"].(string); ok {
		cfg.Mode = &mode
	}
	if vlan, ok := data["vlan_id"].(float64); ok {
		v := int(vlan)
		cfg.VlanID = &v
	}
	if vlanIDs, ok := data["vlan_ids"].([]any); ok {
		cfg.VlanIDs = interfaceSliceToIntSlice(vlanIDs)
	}
	if poe, ok := data["poe_enabled"].(bool); ok {
		cfg.PoEEnabled = &poe
	}
	if speed, ok := data["speed_duplex"].(string); ok {
		cfg.SpeedDuplex = &speed
	}
	if desc, ok := data["description"].(string); ok {
		cfg.Description = &desc
	}

	return cfg
}

// parseLEDConfig parses LED configuration
func parseLEDConfig(data map[string]any) *vendors.LEDConfig {
	if data == nil {
		return nil
	}

	cfg := &vendors.LEDConfig{}

	if enabled, ok := data["enabled"].(bool); ok {
		cfg.Enabled = &enabled
	}
	if brightness, ok := data["brightness"].(float64); ok {
		b := int(brightness)
		cfg.Brightness = &b
	}

	return cfg
}

// parsePowerConfig parses power configuration
func parsePowerConfig(data map[string]any) *vendors.PowerConfig {
	if data == nil {
		return nil
	}

	cfg := &vendors.PowerConfig{}

	if mode, ok := data["mode"].(string); ok {
		cfg.Mode = &mode
	}
	if baseVal, ok := data["base_value"].(float64); ok {
		b := int(baseVal)
		cfg.BaseValue = &b
	}

	return cfg
}

// Helper functions

func interfaceSliceToStringSlice(in []any) []string {
	result := make([]string, 0, len(in))
	for _, v := range in {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func interfaceSliceToFloat64Slice(in []any) []float64 {
	result := make([]float64, 0, len(in))
	for _, v := range in {
		if f, ok := v.(float64); ok {
			result = append(result, f)
		}
	}
	return result
}

func interfaceSliceToIntSlice(in []any) []int {
	result := make([]int, 0, len(in))
	for _, v := range in {
		if f, ok := v.(float64); ok {
			result = append(result, int(f))
		}
	}
	return result
}

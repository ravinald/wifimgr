package mist

import (
	"github.com/ravinald/wifimgr/internal/vendors"
)

// MistSpecificAPFields lists fields that are Mist-specific and should go into the mist: extension block.
// These fields are not part of the common vendor-agnostic schema.
var MistSpecificAPFields = map[string]bool{
	// Mist-specific location features
	"aeroscout":            true,
	"centrak":              true,
	"use_auto_orientation": true,
	"use_auto_placement":   true,

	// Mist-specific networking features
	"client_bridge":   true,
	"esl_config":      true,
	"usb_config":      true,
	"switch_config":   true,
	"port_vlan_id":    true,
	"additional_vlan": true,

	// Mist location/asset tracking
	"site_survey":    true,
	"rtls":           true,
	"zone_occupancy": true,

	// Mist administrative
	"org_id":      true,
	"site_id":     true,
	"id":          true,
	"mac":         true, // MAC is handled separately as device identifier
	"serial":      true,
	"type":        true,
	"model":       true,
	"created_at":  true,
	"modified_at": true,

	// Mist-specific radio features
	"dual_band_5g_3rd_radio": true,
}

// MistSpecificRadioFields lists radio_config fields specific to Mist.
var MistSpecificRadioFields = map[string]bool{
	"dual_band_5g_3rd_radio": true,
}

// ExportAPConfig converts a raw Mist AP config map into a structured APDeviceConfig
// with common fields in standard locations and Mist-specific fields in the mist: extension block.
func ExportAPConfig(rawConfig map[string]any) *vendors.APDeviceConfig {
	if rawConfig == nil {
		return nil
	}

	config := &vendors.APDeviceConfig{
		Mist: make(map[string]any),
	}

	// Extract identity fields
	if name, ok := rawConfig["name"].(string); ok {
		config.Name = name
	}
	if notes, ok := rawConfig["notes"].(string); ok {
		config.Notes = notes
	}
	if tags, ok := rawConfig["tags"].([]any); ok {
		config.Tags = toStringSlice(tags)
	}

	// Extract location fields
	if latlng, ok := rawConfig["latlng"].(map[string]any); ok {
		lat, latOk := latlng["lat"].(float64)
		lng, lngOk := latlng["lng"].(float64)
		if latOk && lngOk {
			config.Location = []float64{lat, lng}
		}
	}
	if mapID, ok := rawConfig["map_id"].(string); ok {
		config.MapID = mapID
	}
	if x, ok := rawConfig["x"].(float64); ok {
		config.X = &x
	}
	if y, ok := rawConfig["y"].(float64); ok {
		config.Y = &y
	}
	if height, ok := rawConfig["height"].(float64); ok {
		config.Height = &height
	}
	if orientation, ok := rawConfig["orientation"].(float64); ok {
		o := int(orientation)
		config.Orientation = &o
	}

	// Extract device profile reference
	if dpID, ok := rawConfig["deviceprofile_id"].(string); ok {
		config.DeviceProfileID = dpID
	}

	// Extract radio configuration
	if radioConfig, ok := rawConfig["radio_config"].(map[string]any); ok {
		config.RadioConfig = exportRadioConfig(radioConfig)
	}

	// Extract IP configuration
	if ipConfig, ok := rawConfig["ip_config"].(map[string]any); ok {
		config.IPConfig = exportIPConfig(ipConfig)
	}

	// Extract BLE configuration
	if bleConfig, ok := rawConfig["ble_config"].(map[string]any); ok {
		config.BLEConfig = exportBLEConfig(bleConfig)
	}

	// Extract mesh configuration
	if meshConfig, ok := rawConfig["mesh"].(map[string]any); ok {
		config.MeshConfig = exportMeshConfig(meshConfig)
	}

	// Extract LED configuration
	if led, ok := rawConfig["led"].(map[string]any); ok {
		config.LEDConfig = exportLEDConfig(led)
	}

	// Extract power configuration
	if pwrConfig, ok := rawConfig["pwr_config"].(map[string]any); ok {
		config.PowerConfig = exportPowerConfig(pwrConfig)
	}

	// Extract hardware flags
	if v, ok := rawConfig["disable_eth1"].(bool); ok {
		config.DisableEth1 = &v
	}
	if v, ok := rawConfig["disable_eth2"].(bool); ok {
		config.DisableEth2 = &v
	}
	if v, ok := rawConfig["disable_eth3"].(bool); ok {
		config.DisableEth3 = &v
	}
	if v, ok := rawConfig["disable_module"].(bool); ok {
		config.DisableModule = &v
	}
	if v, ok := rawConfig["poe_passthrough"].(bool); ok {
		config.PoEPassthrough = &v
	}

	// Extract vars
	if vars, ok := rawConfig["vars"].(map[string]any); ok {
		config.Vars = vars
	}

	// Extract Mist-specific fields into the mist: extension block
	for key, value := range rawConfig {
		if MistSpecificAPFields[key] {
			config.Mist[key] = value
		}
	}

	// Clean up empty mist block
	if len(config.Mist) == 0 {
		config.Mist = nil
	}

	return config
}

// exportRadioConfig converts raw radio config to structured RadioConfig.
func exportRadioConfig(raw map[string]any) *vendors.RadioConfig {
	if raw == nil {
		return nil
	}

	rc := &vendors.RadioConfig{
		Mist: make(map[string]any),
	}

	// Global settings
	if v, ok := raw["allow_rrm_disable"].(bool); ok {
		rc.AllowRRMDisable = &v
	}
	if v, ok := raw["full_automatic_rrm"].(bool); ok {
		rc.FullAutomaticRRM = &v
	}
	if v, ok := raw["indoor_use"].(bool); ok {
		rc.IndoorUse = &v
	}
	if v, ok := raw["scanning_enabled"].(bool); ok {
		rc.ScanningEnabled = &v
	}
	if v, ok := raw["ant_gain_24"].(float64); ok {
		rc.AntGain24 = &v
	}
	if v, ok := raw["ant_gain_5"].(float64); ok {
		rc.AntGain5 = &v
	}
	if v, ok := raw["ant_gain_6"].(float64); ok {
		rc.AntGain6 = &v
	}
	if v, ok := raw["antenna_mode"].(string); ok {
		rc.AntennaMode = &v
	}
	if v, ok := raw["band_24_usage"].(string); ok {
		rc.Band24Usage = &v
	}

	// Per-band configuration
	if band24, ok := raw["band_24"].(map[string]any); ok {
		rc.Band24 = exportRadioBandConfig(band24)
	}
	if band5, ok := raw["band_5"].(map[string]any); ok {
		rc.Band5 = exportRadioBandConfig(band5)
	}
	if band5on24, ok := raw["band_5_on_24_radio"].(map[string]any); ok {
		rc.Band5On24Radio = exportRadioBandConfig(band5on24)
	}
	if band6, ok := raw["band_6"].(map[string]any); ok {
		rc.Band6 = exportRadioBandConfig(band6)
	}

	// Extract Mist-specific radio fields
	for key, value := range raw {
		if MistSpecificRadioFields[key] {
			rc.Mist[key] = value
		}
	}

	if len(rc.Mist) == 0 {
		rc.Mist = nil
	}

	return rc
}

// exportRadioBandConfig converts raw band config to structured RadioBandConfig.
func exportRadioBandConfig(raw map[string]any) *vendors.RadioBandConfig {
	if raw == nil {
		return nil
	}

	bc := &vendors.RadioBandConfig{}

	if v, ok := raw["disabled"].(bool); ok {
		bc.Disabled = &v
	}
	if v, ok := raw["channel"].(float64); ok {
		ch := int(v)
		bc.Channel = &ch
	}
	if channels, ok := raw["channels"].([]any); ok {
		bc.Channels = toIntSlice(channels)
	}
	if v, ok := raw["power"].(float64); ok {
		p := int(v)
		bc.Power = &p
	}
	if v, ok := raw["power_min"].(float64); ok {
		p := int(v)
		bc.PowerMin = &p
	}
	if v, ok := raw["power_max"].(float64); ok {
		p := int(v)
		bc.PowerMax = &p
	}
	if v, ok := raw["bandwidth"].(float64); ok {
		bw := int(v)
		bc.Bandwidth = &bw
	}
	if v, ok := raw["antenna_mode"].(string); ok {
		bc.AntennaMode = &v
	}
	if v, ok := raw["ant_gain"].(float64); ok {
		bc.AntGain = &v
	}
	if v, ok := raw["allow_rrm_disable"].(bool); ok {
		bc.AllowRRMDisable = &v
	}
	if v, ok := raw["preamble"].(string); ok {
		bc.Preamble = &v
	}

	return bc
}

// exportIPConfig converts raw IP config to structured IPConfig.
func exportIPConfig(raw map[string]any) *vendors.IPConfig {
	if raw == nil {
		return nil
	}

	ic := &vendors.IPConfig{}

	if v, ok := raw["type"].(string); ok {
		ic.Type = &v
	}
	if v, ok := raw["ip"].(string); ok {
		ic.IP = &v
	}
	if v, ok := raw["netmask"].(string); ok {
		ic.Netmask = &v
	}
	if v, ok := raw["gateway"].(string); ok {
		ic.Gateway = &v
	}
	if dns, ok := raw["dns"].([]any); ok {
		ic.DNS = toStringSlice(dns)
	}
	if dnsSuffix, ok := raw["dns_suffix"].([]any); ok {
		ic.DNSSuffix = toStringSlice(dnsSuffix)
	}
	if v, ok := raw["vlan_id"].(float64); ok {
		vlan := int(v)
		ic.VlanID = &vlan
	}
	if v, ok := raw["mtu"].(float64); ok {
		mtu := int(v)
		ic.Mtu = &mtu
	}

	return ic
}

// exportBLEConfig converts raw BLE config to structured BLEConfig.
func exportBLEConfig(raw map[string]any) *vendors.BLEConfig {
	if raw == nil {
		return nil
	}

	bc := &vendors.BLEConfig{}

	if v, ok := raw["enabled"].(bool); ok {
		bc.Enabled = &v
	}
	if v, ok := raw["power"].(float64); ok {
		p := int(v)
		bc.Power = &p
	}
	if v, ok := raw["mode"].(string); ok {
		bc.Mode = &v
	}

	if ibeacon, ok := raw["ibeacon"].(map[string]any); ok {
		bc.IBeacon = &vendors.IBeaconConfig{}
		if v, ok := ibeacon["enabled"].(bool); ok {
			bc.IBeacon.Enabled = &v
		}
		if v, ok := ibeacon["uuid"].(string); ok {
			bc.IBeacon.UUID = &v
		}
		if v, ok := ibeacon["major"].(float64); ok {
			m := int(v)
			bc.IBeacon.Major = &m
		}
		if v, ok := ibeacon["minor"].(float64); ok {
			m := int(v)
			bc.IBeacon.Minor = &m
		}
		if v, ok := ibeacon["power"].(float64); ok {
			p := int(v)
			bc.IBeacon.Power = &p
		}
	}

	if eddystone, ok := raw["eddystone"].(map[string]any); ok {
		bc.Eddystone = &vendors.EddystoneConfig{}
		if v, ok := eddystone["enabled"].(bool); ok {
			bc.Eddystone.Enabled = &v
		}
		if v, ok := eddystone["namespace_id"].(string); ok {
			bc.Eddystone.NamespaceID = &v
		}
		if v, ok := eddystone["instance_id"].(string); ok {
			bc.Eddystone.InstanceID = &v
		}
		if v, ok := eddystone["url"].(string); ok {
			bc.Eddystone.URL = &v
		}
	}

	return bc
}

// exportMeshConfig converts raw mesh config to structured MeshConfig.
func exportMeshConfig(raw map[string]any) *vendors.MeshConfig {
	if raw == nil {
		return nil
	}

	mc := &vendors.MeshConfig{}

	if v, ok := raw["enabled"].(bool); ok {
		mc.Enabled = &v
	}
	if v, ok := raw["role"].(string); ok {
		mc.Role = &v
	}
	if v, ok := raw["group"].(string); ok {
		mc.Group = &v
	}

	return mc
}

// exportLEDConfig converts raw LED config to structured LEDConfig.
func exportLEDConfig(raw map[string]any) *vendors.LEDConfig {
	if raw == nil {
		return nil
	}

	lc := &vendors.LEDConfig{}

	if v, ok := raw["enabled"].(bool); ok {
		lc.Enabled = &v
	}
	if v, ok := raw["brightness"].(float64); ok {
		b := int(v)
		lc.Brightness = &b
	}

	return lc
}

// exportPowerConfig converts raw power config to structured PowerConfig.
func exportPowerConfig(raw map[string]any) *vendors.PowerConfig {
	if raw == nil {
		return nil
	}

	pc := &vendors.PowerConfig{}

	if v, ok := raw["mode"].(string); ok {
		pc.Mode = &v
	}
	if v, ok := raw["base_value"].(float64); ok {
		bv := int(v)
		pc.BaseValue = &bv
	}

	return pc
}

// Helper functions

func toStringSlice(arr []any) []string {
	result := make([]string, 0, len(arr))
	for _, v := range arr {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func toIntSlice(arr []any) []int {
	result := make([]int, 0, len(arr))
	for _, v := range arr {
		if f, ok := v.(float64); ok {
			result = append(result, int(f))
		}
	}
	return result
}

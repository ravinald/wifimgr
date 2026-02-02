package vendors

import (
	"testing"
)

func TestAPDeviceConfigValidate(t *testing.T) {
	t.Run("nil config is valid", func(t *testing.T) {
		var cfg *APDeviceConfig
		if err := cfg.Validate(); err != nil {
			t.Errorf("nil config should be valid, got error: %v", err)
		}
	})

	t.Run("empty config is valid", func(t *testing.T) {
		cfg := &APDeviceConfig{}
		if err := cfg.Validate(); err != nil {
			t.Errorf("empty config should be valid, got error: %v", err)
		}
	})

	t.Run("deviceprofile_id only is valid", func(t *testing.T) {
		cfg := &APDeviceConfig{
			DeviceProfileID: "abc-123",
		}
		if err := cfg.Validate(); err != nil {
			t.Errorf("expected valid, got error: %v", err)
		}
	})

	t.Run("deviceprofile_name only is valid", func(t *testing.T) {
		cfg := &APDeviceConfig{
			DeviceProfileName: "My Profile",
		}
		if err := cfg.Validate(); err != nil {
			t.Errorf("expected valid, got error: %v", err)
		}
	})

	t.Run("both deviceprofile_id and deviceprofile_name is invalid", func(t *testing.T) {
		cfg := &APDeviceConfig{
			DeviceProfileID:   "abc-123",
			DeviceProfileName: "My Profile",
		}
		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for mutually exclusive fields")
		}
		if _, ok := err.(*ConfigValidationError); !ok {
			t.Errorf("expected ConfigValidationError, got %T", err)
		}
	})

	t.Run("both map_id and map_name is invalid", func(t *testing.T) {
		cfg := &APDeviceConfig{
			MapID:   "map-123",
			MapName: "Floor 1",
		}
		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for mutually exclusive fields")
		}
	})

	t.Run("radio_config with rf_profile mutual exclusion", func(t *testing.T) {
		cfg := &APDeviceConfig{
			RadioConfig: &RadioConfig{
				Meraki: map[string]interface{}{
					"rf_profile_id":   "prof-123",
					"rf_profile_name": "High Density",
				},
			},
		}
		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for mutually exclusive rf_profile fields")
		}
	})
}

func TestAPDeviceConfigToMap(t *testing.T) {
	t.Run("nil config returns nil", func(t *testing.T) {
		var cfg *APDeviceConfig
		if m := cfg.ToMap(); m != nil {
			t.Errorf("nil config should return nil map, got %v", m)
		}
	})

	t.Run("basic fields", func(t *testing.T) {
		cfg := &APDeviceConfig{
			Name:  "AP-LOBBY-01",
			Tags:  []string{"lobby", "ground-floor"},
			Notes: "Main entrance",
		}
		m := cfg.ToMap()

		if m["name"] != "AP-LOBBY-01" {
			t.Errorf("name = %v, want AP-LOBBY-01", m["name"])
		}
		tags, ok := m["tags"].([]string)
		if !ok || len(tags) != 2 {
			t.Errorf("tags = %v, want [lobby, ground-floor]", m["tags"])
		}
		if m["notes"] != "Main entrance" {
			t.Errorf("notes = %v, want Main entrance", m["notes"])
		}
	})

	t.Run("location fields", func(t *testing.T) {
		orientation := 90
		x := 10.5
		y := 20.5
		height := 3.0
		cfg := &APDeviceConfig{
			Location:    []float64{37.7749, -122.4194},
			Orientation: &orientation,
			MapID:       "map-123",
			X:           &x,
			Y:           &y,
			Height:      &height,
		}
		m := cfg.ToMap()

		if m["map_id"] != "map-123" {
			t.Errorf("map_id = %v, want map-123", m["map_id"])
		}
		if m["orientation"] != 90 {
			t.Errorf("orientation = %v, want 90", m["orientation"])
		}
	})

	t.Run("hardware flags", func(t *testing.T) {
		disableEth1 := true
		poePassthrough := false
		cfg := &APDeviceConfig{
			DisableEth1:    &disableEth1,
			PoEPassthrough: &poePassthrough,
		}
		m := cfg.ToMap()

		if m["disable_eth1"] != true {
			t.Errorf("disable_eth1 = %v, want true", m["disable_eth1"])
		}
		if m["poe_passthrough"] != false {
			t.Errorf("poe_passthrough = %v, want false", m["poe_passthrough"])
		}
	})

	t.Run("nested radio config", func(t *testing.T) {
		channel := 6
		power := 12
		cfg := &APDeviceConfig{
			RadioConfig: &RadioConfig{
				Band24: &RadioBandConfig{
					Channel: &channel,
					Power:   &power,
				},
			},
		}
		m := cfg.ToMap()

		rc, ok := m["radio_config"].(map[string]interface{})
		if !ok {
			t.Fatal("radio_config should be a map")
		}
		b24, ok := rc["band_24"].(map[string]interface{})
		if !ok {
			t.Fatal("band_24 should be a map")
		}
		if b24["channel"] != 6 {
			t.Errorf("channel = %v, want 6", b24["channel"])
		}
		if b24["power"] != 12 {
			t.Errorf("power = %v, want 12", b24["power"])
		}
	})

	t.Run("mist extensions merge at top level", func(t *testing.T) {
		cfg := &APDeviceConfig{
			Name: "AP-01",
			Mist: map[string]interface{}{
				"aeroscout": map[string]interface{}{"enabled": false},
			},
		}
		m := cfg.ToMap()

		if m["name"] != "AP-01" {
			t.Errorf("name = %v, want AP-01", m["name"])
		}
		aeroscout, ok := m["aeroscout"].(map[string]interface{})
		if !ok {
			t.Fatal("aeroscout should be merged at top level")
		}
		if aeroscout["enabled"] != false {
			t.Errorf("aeroscout.enabled = %v, want false", aeroscout["enabled"])
		}
	})
}

func TestRadioConfigToMap(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		var cfg *RadioConfig
		if m := cfg.ToMap(); m != nil {
			t.Errorf("nil should return nil, got %v", m)
		}
	})

	t.Run("global settings", func(t *testing.T) {
		enabled := true
		antGain := 2.5
		cfg := &RadioConfig{
			ScanningEnabled: &enabled,
			AntGain24:       &antGain,
		}
		m := cfg.ToMap()

		if m["scanning_enabled"] != true {
			t.Errorf("scanning_enabled = %v, want true", m["scanning_enabled"])
		}
		if m["ant_gain_24"] != 2.5 {
			t.Errorf("ant_gain_24 = %v, want 2.5", m["ant_gain_24"])
		}
	})

	t.Run("per-band config", func(t *testing.T) {
		channel := 36
		bandwidth := 80
		cfg := &RadioConfig{
			Band5: &RadioBandConfig{
				Channel:   &channel,
				Bandwidth: &bandwidth,
			},
		}
		m := cfg.ToMap()

		b5, ok := m["band_5"].(map[string]interface{})
		if !ok {
			t.Fatal("band_5 should be a map")
		}
		if b5["channel"] != 36 {
			t.Errorf("channel = %v, want 36", b5["channel"])
		}
		if b5["bandwidth"] != 80 {
			t.Errorf("bandwidth = %v, want 80", b5["bandwidth"])
		}
	})
}

func TestIPConfigToMap(t *testing.T) {
	t.Run("dhcp config", func(t *testing.T) {
		typ := "dhcp"
		vlan := 100
		cfg := &IPConfig{
			Type:   &typ,
			VlanID: &vlan,
		}
		m := cfg.ToMap()

		if m["type"] != "dhcp" {
			t.Errorf("type = %v, want dhcp", m["type"])
		}
		if m["vlan_id"] != 100 {
			t.Errorf("vlan_id = %v, want 100", m["vlan_id"])
		}
	})

	t.Run("static config", func(t *testing.T) {
		typ := "static"
		ip := "10.0.1.100"
		netmask := "255.255.255.0"
		gateway := "10.0.1.1"
		cfg := &IPConfig{
			Type:    &typ,
			IP:      &ip,
			Netmask: &netmask,
			Gateway: &gateway,
			DNS:     []string{"8.8.8.8", "8.8.4.4"},
		}
		m := cfg.ToMap()

		if m["ip"] != "10.0.1.100" {
			t.Errorf("ip = %v, want 10.0.1.100", m["ip"])
		}
		if m["gateway"] != "10.0.1.1" {
			t.Errorf("gateway = %v, want 10.0.1.1", m["gateway"])
		}
		dns, ok := m["dns"].([]string)
		if !ok || len(dns) != 2 {
			t.Errorf("dns = %v, want [8.8.8.8, 8.8.4.4]", m["dns"])
		}
	})
}

func TestBLEConfigToMap(t *testing.T) {
	t.Run("basic ble config", func(t *testing.T) {
		enabled := true
		power := 3
		cfg := &BLEConfig{
			Enabled: &enabled,
			Power:   &power,
		}
		m := cfg.ToMap()

		if m["enabled"] != true {
			t.Errorf("enabled = %v, want true", m["enabled"])
		}
		if m["power"] != 3 {
			t.Errorf("power = %v, want 3", m["power"])
		}
	})

	t.Run("ibeacon config", func(t *testing.T) {
		uuid := "f7826da6-4fa2-4e98-8024-bc5b71e0893e"
		major := 100
		minor := 1
		cfg := &BLEConfig{
			IBeacon: &IBeaconConfig{
				UUID:  &uuid,
				Major: &major,
				Minor: &minor,
			},
		}
		m := cfg.ToMap()

		ibeacon, ok := m["ibeacon"].(map[string]interface{})
		if !ok {
			t.Fatal("ibeacon should be a map")
		}
		if ibeacon["uuid"] != uuid {
			t.Errorf("uuid = %v, want %v", ibeacon["uuid"], uuid)
		}
		if ibeacon["major"] != 100 {
			t.Errorf("major = %v, want 100", ibeacon["major"])
		}
	})
}

func TestPortConfigToMap(t *testing.T) {
	t.Run("access port", func(t *testing.T) {
		portID := "eth0"
		mode := "access"
		vlan := 100
		poe := true
		cfg := &PortConfig{
			PortID:     &portID,
			Mode:       &mode,
			VlanID:     &vlan,
			PoEEnabled: &poe,
		}
		m := cfg.ToMap()

		if m["port_id"] != "eth0" {
			t.Errorf("port_id = %v, want eth0", m["port_id"])
		}
		if m["mode"] != "access" {
			t.Errorf("mode = %v, want access", m["mode"])
		}
		if m["vlan_id"] != 100 {
			t.Errorf("vlan_id = %v, want 100", m["vlan_id"])
		}
	})

	t.Run("trunk port", func(t *testing.T) {
		mode := "trunk"
		nativeVlan := 1
		cfg := &PortConfig{
			Mode:       &mode,
			VlanIDs:    []int{100, 200, 300},
			NativeVlan: &nativeVlan,
		}
		m := cfg.ToMap()

		if m["mode"] != "trunk" {
			t.Errorf("mode = %v, want trunk", m["mode"])
		}
		vlans, ok := m["vlan_ids"].([]int)
		if !ok || len(vlans) != 3 {
			t.Errorf("vlan_ids = %v, want [100, 200, 300]", m["vlan_ids"])
		}
	})

	t.Run("port auth config", func(t *testing.T) {
		enabled := true
		mode := "single"
		guestVlan := 999
		cfg := &PortConfig{
			PortAuth: &PortAuthConfig{
				Enabled:   &enabled,
				Mode:      &mode,
				GuestVlan: &guestVlan,
			},
		}
		m := cfg.ToMap()

		portAuth, ok := m["port_auth"].(map[string]interface{})
		if !ok {
			t.Fatal("port_auth should be a map")
		}
		if portAuth["enabled"] != true {
			t.Errorf("enabled = %v, want true", portAuth["enabled"])
		}
		if portAuth["mode"] != "single" {
			t.Errorf("mode = %v, want single", portAuth["mode"])
		}
	})
}

func TestLEDConfigToMap(t *testing.T) {
	enabled := true
	brightness := 50
	cfg := &LEDConfig{
		Enabled:    &enabled,
		Brightness: &brightness,
	}
	m := cfg.ToMap()

	if m["enabled"] != true {
		t.Errorf("enabled = %v, want true", m["enabled"])
	}
	if m["brightness"] != 50 {
		t.Errorf("brightness = %v, want 50", m["brightness"])
	}
}

func TestConfigValidationError(t *testing.T) {
	err := &ConfigValidationError{
		Field:   "test_field",
		Message: "test message",
	}
	expected := "test_field: test message"
	if err.Error() != expected {
		t.Errorf("Error() = %v, want %v", err.Error(), expected)
	}
}

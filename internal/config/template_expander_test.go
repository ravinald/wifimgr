package config

import (
	"reflect"
	"testing"
)

func TestGetVendorFromAPILabel(t *testing.T) {
	tests := []struct {
		apiLabel string
		expected string
	}{
		{"mist-prod", "mist"},
		{"mist-lab", "mist"},
		{"meraki-corp", "meraki"},
		{"meraki", "meraki"},
		{"mist", "mist"},
		{"", ""},
		{"unknown-vendor", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.apiLabel, func(t *testing.T) {
			result := GetVendorFromAPILabel(tt.apiLabel)
			if result != tt.expected {
				t.Errorf("GetVendorFromAPILabel(%q) = %q, want %q", tt.apiLabel, result, tt.expected)
			}
		})
	}
}

func TestExpandForVendor_CommonOnly(t *testing.T) {
	template := map[string]any{
		"power":     15,
		"bandwidth": 40,
	}

	result := ExpandForVendor(template, "mist")

	if result["power"].(int) != 15 {
		t.Errorf("Expected power=15, got %v", result["power"])
	}
	if result["bandwidth"].(int) != 40 {
		t.Errorf("Expected bandwidth=40, got %v", result["bandwidth"])
	}
}

func TestExpandForVendor_WithVendorBlock(t *testing.T) {
	template := map[string]any{
		"power": 15,
		"mist:": map[string]any{
			"scanning_enabled": true,
		},
		"meraki:": map[string]any{
			"rf_profile_id": "meraki-profile-123",
		},
	}

	// Test Mist expansion
	mistResult := ExpandForVendor(template, "mist")
	if mistResult["power"].(int) != 15 {
		t.Errorf("Expected power=15, got %v", mistResult["power"])
	}
	if mistResult["scanning_enabled"] != true {
		t.Errorf("Expected scanning_enabled=true for mist, got %v", mistResult["scanning_enabled"])
	}
	if _, exists := mistResult["rf_profile_id"]; exists {
		t.Error("Did not expect rf_profile_id in mist expansion")
	}
	if _, exists := mistResult["mist:"]; exists {
		t.Error("Vendor block should be removed from result")
	}

	// Test Meraki expansion
	merakiResult := ExpandForVendor(template, "meraki")
	if merakiResult["power"].(int) != 15 {
		t.Errorf("Expected power=15, got %v", merakiResult["power"])
	}
	if merakiResult["rf_profile_id"] != "meraki-profile-123" {
		t.Errorf("Expected rf_profile_id for meraki, got %v", merakiResult["rf_profile_id"])
	}
	if _, exists := merakiResult["scanning_enabled"]; exists {
		t.Error("Did not expect scanning_enabled in meraki expansion")
	}
}

func TestExpandForVendor_DeepMerge(t *testing.T) {
	template := map[string]any{
		"radio_config": map[string]any{
			"band_5": map[string]any{
				"power":     15,
				"bandwidth": 40,
			},
		},
		"mist:": map[string]any{
			"radio_config": map[string]any{
				"band_5": map[string]any{
					"scanning_enabled": true,
				},
			},
		},
	}

	result := ExpandForVendor(template, "mist")

	radioConfig, ok := result["radio_config"].(map[string]any)
	if !ok {
		t.Fatal("Expected radio_config to be a map")
	}

	band5, ok := radioConfig["band_5"].(map[string]any)
	if !ok {
		t.Fatal("Expected band_5 to be a map")
	}

	if band5["power"].(int) != 15 {
		t.Errorf("Expected power=15, got %v", band5["power"])
	}
	if band5["scanning_enabled"] != true {
		t.Errorf("Expected scanning_enabled=true, got %v", band5["scanning_enabled"])
	}
}

func TestMergeConfigs(t *testing.T) {
	dest := map[string]any{
		"a": 1,
		"b": map[string]any{
			"x": 10,
			"y": 20,
		},
	}

	source := map[string]any{
		"a": 2, // Override
		"c": 3, // New key
		"b": map[string]any{
			"y": 25, // Override nested
			"z": 30, // New nested key
		},
	}

	result := mergeConfigs(dest, source)

	if result["a"].(int) != 2 {
		t.Errorf("Expected a=2 (source wins), got %v", result["a"])
	}
	if result["c"].(int) != 3 {
		t.Errorf("Expected c=3, got %v", result["c"])
	}

	b, ok := result["b"].(map[string]any)
	if !ok {
		t.Fatal("Expected b to be a map")
	}
	if b["x"].(int) != 10 {
		t.Errorf("Expected b.x=10, got %v", b["x"])
	}
	if b["y"].(int) != 25 {
		t.Errorf("Expected b.y=25 (source wins), got %v", b["y"])
	}
	if b["z"].(int) != 30 {
		t.Errorf("Expected b.z=30, got %v", b["z"])
	}
}

func TestExpandDeviceConfig_NoTemplates(t *testing.T) {
	deviceConfig := map[string]any{
		"name": "test-device",
		"led":  true,
	}

	result, err := ExpandDeviceConfig(deviceConfig, nil, nil, "mist-prod")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result["name"] != "test-device" {
		t.Errorf("Expected name=test-device, got %v", result["name"])
	}
}

func TestExpandDeviceConfig_RadioProfile(t *testing.T) {
	store := NewTemplateStore()
	// Radio templates use flat structure (no radio_config wrapper)
	// The expander wraps them into radio_config
	store.Radio["high-density"] = map[string]any{
		"band_5": map[string]any{
			"power":     15,
			"bandwidth": 40,
		},
	}

	deviceConfig := map[string]any{
		"name":          "test-ap",
		"radio_profile": "high-density",
	}

	result, err := ExpandDeviceConfig(deviceConfig, nil, store, "mist-prod")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check template was expanded and wrapped into radio_config
	radioConfig, ok := result["radio_config"].(map[string]any)
	if !ok {
		t.Fatal("Expected radio_config from template expansion")
	}

	band5, ok := radioConfig["band_5"].(map[string]any)
	if !ok {
		t.Fatal("Expected band_5 in radio_config")
	}

	if band5["power"].(int) != 15 {
		t.Errorf("Expected power=15 from template, got %v", band5["power"])
	}

	// Check template reference was removed
	if _, exists := result["radio_profile"]; exists {
		t.Error("Expected radio_profile to be removed from result")
	}

	// Check device name preserved
	if result["name"] != "test-ap" {
		t.Errorf("Expected name=test-ap, got %v", result["name"])
	}
}

func TestExpandDeviceConfig_DeviceOverridesTemplate(t *testing.T) {
	store := NewTemplateStore()
	// Radio templates use flat structure (no radio_config wrapper)
	store.Radio["high-density"] = map[string]any{
		"band_5": map[string]any{
			"power":     15,
			"bandwidth": 40,
		},
	}

	deviceConfig := map[string]any{
		"name":          "test-ap",
		"radio_profile": "high-density",
		"radio_config": map[string]any{
			"band_5": map[string]any{
				"power": 20, // Device overrides template
			},
		},
	}

	result, err := ExpandDeviceConfig(deviceConfig, nil, store, "mist-prod")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	radioConfig := result["radio_config"].(map[string]any)
	band5 := radioConfig["band_5"].(map[string]any)

	// Device value should win
	if band5["power"].(int) != 20 {
		t.Errorf("Expected power=20 (device override), got %v", band5["power"])
	}

	// Template value should be preserved if not overridden
	if band5["bandwidth"].(int) != 40 {
		t.Errorf("Expected bandwidth=40 from template, got %v", band5["bandwidth"])
	}
}

func TestExpandDeviceConfig_DeviceTemplate(t *testing.T) {
	store := NewTemplateStore()
	store.Device["standard-ap"] = map[string]any{
		"led": map[string]any{
			"enabled": true,
		},
		"poe_passthrough": false,
	}

	deviceConfig := map[string]any{
		"name":            "test-ap",
		"device_template": "standard-ap",
	}

	result, err := ExpandDeviceConfig(deviceConfig, nil, store, "mist-prod")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	led, ok := result["led"].(map[string]any)
	if !ok {
		t.Fatal("Expected led config from device template")
	}
	if led["enabled"] != true {
		t.Errorf("Expected led.enabled=true, got %v", led["enabled"])
	}
	if result["poe_passthrough"] != false {
		t.Errorf("Expected poe_passthrough=false, got %v", result["poe_passthrough"])
	}
}

func TestExpandDeviceConfig_WLANs_FromDevice(t *testing.T) {
	store := NewTemplateStore()
	store.WLAN["corp-net"] = map[string]any{
		"ssid": "CorpNet",
		"auth": map[string]any{"type": "wpa2-enterprise"},
	}
	store.WLAN["guest-net"] = map[string]any{
		"ssid": "GuestNet",
		"auth": map[string]any{"type": "open"},
	}

	deviceConfig := map[string]any{
		"name": "test-ap",
		"wlan": []any{"corp-net"}, // Device specifies only corp-net
	}

	siteWLANs := []string{"corp-net", "guest-net"} // Site has both

	result, err := ExpandDeviceConfig(deviceConfig, siteWLANs, store, "mist-prod")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	wlan, ok := result["wlan"].([]map[string]any)
	if !ok {
		t.Fatal("Expected wlan to be []map[string]any")
	}

	// Device WLANs should win (only corp-net)
	if len(wlan) != 1 {
		t.Errorf("Expected 1 WLAN (device override), got %d", len(wlan))
	}
	if wlan[0]["ssid"] != "CorpNet" {
		t.Errorf("Expected ssid=CorpNet, got %v", wlan[0]["ssid"])
	}
}

func TestExpandDeviceConfig_WLANs_FromSite(t *testing.T) {
	store := NewTemplateStore()
	store.WLAN["corp-net"] = map[string]any{
		"ssid": "CorpNet",
	}
	store.WLAN["guest-net"] = map[string]any{
		"ssid": "GuestNet",
	}

	deviceConfig := map[string]any{
		"name": "test-ap",
		// No wlan specified - should fall back to site
	}

	siteWLANs := []string{"corp-net", "guest-net"}

	result, err := ExpandDeviceConfig(deviceConfig, siteWLANs, store, "mist-prod")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	wlan, ok := result["wlan"].([]map[string]any)
	if !ok {
		t.Fatal("Expected wlan to be []map[string]any")
	}

	// Should have both site WLANs
	if len(wlan) != 2 {
		t.Errorf("Expected 2 WLANs from site, got %d", len(wlan))
	}
}

func TestExpandDeviceConfig_VendorSpecific(t *testing.T) {
	store := NewTemplateStore()
	// Radio templates use flat structure with vendor-specific blocks
	store.Radio["high-density"] = map[string]any{
		"band_5": map[string]any{
			"power": 15,
		},
		"mist:": map[string]any{
			"scanning_enabled": true,
		},
		"meraki:": map[string]any{
			"rf_profile_id": "meraki-123",
		},
	}

	deviceConfig := map[string]any{
		"name":          "test-ap",
		"radio_profile": "high-density",
	}

	// Test Mist expansion
	mistResult, err := ExpandDeviceConfig(deviceConfig, nil, store, "mist-prod")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Radio template gets wrapped into radio_config
	mistRadio, ok := mistResult["radio_config"].(map[string]any)
	if !ok {
		t.Fatal("Expected radio_config in result")
	}
	if mistRadio["scanning_enabled"] != true {
		t.Errorf("Expected scanning_enabled=true for mist, got %v", mistRadio["scanning_enabled"])
	}
	if _, exists := mistRadio["rf_profile_id"]; exists {
		t.Error("Did not expect rf_profile_id in mist expansion")
	}

	// Test Meraki expansion
	merakiResult, err := ExpandDeviceConfig(deviceConfig, nil, store, "meraki-corp")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	merakiRadio, ok := merakiResult["radio_config"].(map[string]any)
	if !ok {
		t.Fatal("Expected radio_config in result")
	}
	if merakiRadio["rf_profile_id"] != "meraki-123" {
		t.Errorf("Expected rf_profile_id=meraki-123 for meraki, got %v", merakiRadio["rf_profile_id"])
	}
	if _, exists := merakiRadio["scanning_enabled"]; exists {
		t.Error("Did not expect scanning_enabled in meraki expansion")
	}
}

func TestDeepCopy(t *testing.T) {
	original := map[string]any{
		"a": 1,
		"b": map[string]any{
			"c": 2,
		},
		"d": []any{1, 2, 3},
	}

	copied := deepCopy(original).(map[string]any)

	// Modify copied
	copied["a"] = 999
	copied["b"].(map[string]any)["c"] = 888
	copied["d"].([]any)[0] = 777

	// Original should be unchanged
	if original["a"].(int) != 1 {
		t.Errorf("Original a was modified, expected 1, got %v", original["a"])
	}
	if original["b"].(map[string]any)["c"].(int) != 2 {
		t.Errorf("Original b.c was modified, expected 2, got %v", original["b"].(map[string]any)["c"])
	}
	if original["d"].([]any)[0].(int) != 1 {
		t.Errorf("Original d[0] was modified, expected 1, got %v", original["d"].([]any)[0])
	}
}

func TestToStringSlice(t *testing.T) {
	input := []any{"a", "b", "c", 123, "d"}
	result := toStringSlice(input)

	expected := []string{"a", "b", "c", "d"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestGetSiteWLANLabels(t *testing.T) {
	// Test profiles.wlan path
	siteConfig1 := map[string]any{
		"profiles": map[string]any{
			"wlan": []any{"corp-net", "guest-net"},
		},
	}
	result1 := GetSiteWLANLabels(siteConfig1)
	if len(result1) != 2 || result1[0] != "corp-net" {
		t.Errorf("Expected [corp-net, guest-net], got %v", result1)
	}

	// Test direct wlan path
	siteConfig2 := map[string]any{
		"wlan": []any{"wlan-a", "wlan-b"},
	}
	result2 := GetSiteWLANLabels(siteConfig2)
	if len(result2) != 2 || result2[0] != "wlan-a" {
		t.Errorf("Expected [wlan-a, wlan-b], got %v", result2)
	}

	// Test empty
	siteConfig3 := map[string]any{}
	result3 := GetSiteWLANLabels(siteConfig3)
	if len(result3) != 0 {
		t.Errorf("Expected empty slice, got %v", result3)
	}
}

func TestIsVendorBlock(t *testing.T) {
	tests := []struct {
		key      string
		expected bool
	}{
		{"mist:", true},
		{"meraki:", true},
		{"unknown:", true},
		{"mist", false},
		{"power", false},
		{"radio_config", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := isVendorBlock(tt.key)
			if result != tt.expected {
				t.Errorf("isVendorBlock(%q) = %v, want %v", tt.key, result, tt.expected)
			}
		})
	}
}

func TestIsTemplateReferenceField(t *testing.T) {
	tests := []struct {
		key      string
		expected bool
	}{
		{"radio_profile", true},
		{"device_template", true},
		{"wlan", true}, // wlan contains template labels, gets expanded
		{"name", false},
		{"radio_config", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := isTemplateReferenceField(tt.key)
			if result != tt.expected {
				t.Errorf("isTemplateReferenceField(%q) = %v, want %v", tt.key, result, tt.expected)
			}
		})
	}
}

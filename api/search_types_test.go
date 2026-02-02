package api

import (
	"testing"
)

// Compile-time interface compliance checks - no runtime test needed
var _ SearchClientMarshaler = (*MistWiredClient)(nil)
var _ SearchClientMarshaler = (*MistWirelessClient)(nil)

func TestMistWiredClient_GetMAC_Precedence(t *testing.T) {
	// Tests that ClientMAC takes precedence over MAC
	tests := []struct {
		name      string
		clientMAC *string
		mac       *string
		expected  string
	}{
		{"ClientMAC only", StringPtr("00:11:22:33:44:55"), nil, "001122334455"},
		{"MAC only", nil, StringPtr("00:11:22:33:44:66"), "001122334466"},
		{"both - ClientMAC wins", StringPtr("00:11:22:33:44:55"), StringPtr("00:11:22:33:44:66"), "001122334455"},
		{"neither", nil, nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &MistWiredClient{
				ClientMAC: tt.clientMAC,
				MAC:       tt.mac,
			}
			if got := client.GetMAC(); got != tt.expected {
				t.Errorf("GetMAC() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestMistWiredClient_ToMap_NilFieldsExcluded(t *testing.T) {
	client := &MistWiredClient{
		OrgID:     StringPtr("org-123"),
		SiteID:    StringPtr("site-456"),
		ClientMAC: StringPtr("001122334455"),
	}

	result := client.ToMap()

	// Non-nil fields should be present
	if result["org_id"] != "org-123" {
		t.Error("expected org_id in map")
	}

	// Nil arrays should not be present (or be empty)
	if _, exists := result["device_mac"]; exists {
		if devMAC, ok := result["device_mac"].([]string); ok && len(devMAC) > 0 {
			t.Error("nil/empty device_mac should not be in map")
		}
	}
}

func TestMistWiredClient_ToMap_ArraysPreserved(t *testing.T) {
	client := &MistWiredClient{
		OrgID:     StringPtr("org-123"),
		DeviceMAC: []string{"aabbccddeeff", "112233445566"},
		VLAN:      []int{100, 200},
		Hostname:  []string{"host1", "host2"},
	}

	result := client.ToMap()

	if deviceMAC, ok := result["device_mac"].([]string); ok {
		if len(deviceMAC) != 2 || deviceMAC[0] != "aabbccddeeff" {
			t.Error("device_mac array not preserved")
		}
	} else {
		t.Error("device_mac should be string array")
	}

	if vlan, ok := result["vlan"].([]int); ok {
		if len(vlan) != 2 || vlan[0] != 100 {
			t.Error("vlan array not preserved")
		}
	} else {
		t.Error("vlan should be int array")
	}
}

func TestMistWiredClient_ToMap_DeviceMacPortPreserved(t *testing.T) {
	client := &MistWiredClient{
		OrgID: StringPtr("org-123"),
		DeviceMacPort: []*MistDeviceMacPort{
			{
				DeviceMAC: StringPtr("aabbccddeeff"),
				PortID:    StringPtr("ge-0/0/1"),
				VLAN:      IntPtr(100),
			},
		},
	}

	result := client.ToMap()

	if dmpData, ok := result["device_mac_port"].([]map[string]interface{}); ok {
		if len(dmpData) != 1 {
			t.Errorf("expected 1 device_mac_port entry, got %d", len(dmpData))
		}
		if dmpData[0]["device_mac"] != "aabbccddeeff" {
			t.Error("device_mac not preserved in device_mac_port")
		}
	} else {
		t.Error("device_mac_port should be array of maps")
	}
}

func TestMistWiredClient_FromMap_ParsesArrays(t *testing.T) {
	data := map[string]interface{}{
		"org_id":     "org-123",
		"device_mac": []interface{}{"aabbccddeeff", "112233445566"},
		"vlan":       []interface{}{float64(100), float64(200)},
	}

	client := &MistWiredClient{
		AdditionalConfig: make(map[string]interface{}),
	}

	if err := client.FromMap(data); err != nil {
		t.Fatalf("FromMap() error = %v", err)
	}

	if len(client.DeviceMAC) != 2 || client.DeviceMAC[0] != "aabbccddeeff" {
		t.Error("device_mac array not parsed")
	}
	if len(client.VLAN) != 2 || client.VLAN[0] != 100 {
		t.Error("vlan array not parsed")
	}
}

func TestMistWiredClient_FromMap_ParsesDeviceMacPort(t *testing.T) {
	data := map[string]interface{}{
		"org_id": "org-123",
		"device_mac_port": []interface{}{
			map[string]interface{}{
				"device_mac": "aabbccddeeff",
				"port_id":    "ge-0/0/1",
				"vlan":       float64(100),
			},
		},
	}

	client := &MistWiredClient{
		AdditionalConfig: make(map[string]interface{}),
	}

	if err := client.FromMap(data); err != nil {
		t.Fatalf("FromMap() error = %v", err)
	}

	if len(client.DeviceMacPort) != 1 {
		t.Fatalf("expected 1 DeviceMacPort entry, got %d", len(client.DeviceMacPort))
	}
	if *client.DeviceMacPort[0].DeviceMAC != "aabbccddeeff" {
		t.Error("DeviceMAC not parsed in DeviceMacPort")
	}
	if *client.DeviceMacPort[0].VLAN != 100 {
		t.Error("VLAN not parsed in DeviceMacPort")
	}
}

func TestMistWirelessClient_GetMAC_NormalizesFormat(t *testing.T) {
	tests := []struct {
		name     string
		mac      *string
		expected string
	}{
		{"with colons", StringPtr("00:11:22:33:44:77"), "001122334477"},
		{"normalized", StringPtr("001122334477"), "001122334477"},
		{"nil", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &MistWirelessClient{MAC: tt.mac}
			if got := client.GetMAC(); got != tt.expected {
				t.Errorf("GetMAC() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestMistWirelessClient_ToMap_ArraysPreserved(t *testing.T) {
	client := &MistWirelessClient{
		OrgID:   StringPtr("org-123"),
		SiteIDs: []string{"site-456", "site-789"},
		AP:      []string{"ap1", "ap2"},
		SSID:    []string{"corp-wifi", "guest-wifi"},
		VLAN:    []int{100, 200},
	}

	result := client.ToMap()

	if siteIDs, ok := result["site_ids"].([]string); ok {
		if len(siteIDs) != 2 || siteIDs[0] != "site-456" {
			t.Error("site_ids array not preserved")
		}
	} else {
		t.Error("site_ids should be string array")
	}

	if ap, ok := result["ap"].([]string); ok {
		if len(ap) != 2 || ap[0] != "ap1" {
			t.Error("ap array not preserved")
		}
	} else {
		t.Error("ap should be string array")
	}
}

func TestMistWirelessClient_FromMap_ParsesArrays(t *testing.T) {
	data := map[string]interface{}{
		"org_id":   "org-123",
		"site_ids": []interface{}{"site-456", "site-789"},
		"ap":       []interface{}{"ap1", "ap2"},
		"vlan":     []interface{}{float64(100), float64(200)},
	}

	client := &MistWirelessClient{
		AdditionalConfig: make(map[string]interface{}),
	}

	if err := client.FromMap(data); err != nil {
		t.Fatalf("FromMap() error = %v", err)
	}

	if len(client.SiteIDs) != 2 || client.SiteIDs[0] != "site-456" {
		t.Error("site_ids array not parsed")
	}
	if len(client.AP) != 2 || client.AP[0] != "ap1" {
		t.Error("ap array not parsed")
	}
	if len(client.VLAN) != 2 || client.VLAN[0] != 100 {
		t.Error("vlan array not parsed")
	}
}

func TestSearchClient_RoundTrip(t *testing.T) {
	// Test wired client round-trip
	wiredData := map[string]interface{}{
		"org_id":       "org-123",
		"site_id":      "site-456",
		"client_mac":   "001122334455",
		"device_mac":   []interface{}{"aabbccddeeff"},
		"vlan":         []interface{}{float64(100)},
		"custom_field": "custom_value",
	}

	wiredClient, err := NewWiredClientFromMap(wiredData)
	if err != nil {
		t.Fatalf("NewWiredClientFromMap() error = %v", err)
	}

	wiredResult := wiredClient.ToMap()

	if wiredResult["org_id"] != "org-123" {
		t.Error("wired client org_id not preserved")
	}
	if wiredResult["custom_field"] != "custom_value" {
		t.Error("wired client custom_field not preserved")
	}

	// Test wireless client round-trip
	wirelessData := map[string]interface{}{
		"org_id":       "org-123",
		"mac":          "001122334477",
		"ap":           []interface{}{"ap1"},
		"custom_field": "custom_value",
	}

	wirelessClient, err := NewWirelessClientFromMap(wirelessData)
	if err != nil {
		t.Fatalf("NewWirelessClientFromMap() error = %v", err)
	}

	wirelessResult := wirelessClient.ToMap()

	if wirelessResult["org_id"] != "org-123" {
		t.Error("wireless client org_id not preserved")
	}
	if wirelessResult["custom_field"] != "custom_value" {
		t.Error("wireless client custom_field not preserved")
	}
}

func TestMistDeviceMacPort_RoundTrip(t *testing.T) {
	originalData := map[string]interface{}{
		"device_mac":   "aabbccddeeff",
		"port_id":      "ge-0/0/1",
		"vlan":         float64(100),
		"ip":           "192.168.1.100",
		"custom_field": "custom_value",
	}

	dmp := &MistDeviceMacPort{
		AdditionalConfig: make(map[string]interface{}),
	}
	if err := dmp.FromMap(originalData); err != nil {
		t.Fatalf("FromMap() error = %v", err)
	}

	if dmp.DeviceMAC == nil || *dmp.DeviceMAC != "aabbccddeeff" {
		t.Error("DeviceMAC not parsed")
	}
	if dmp.VLAN == nil || *dmp.VLAN != 100 {
		t.Error("VLAN not parsed")
	}

	result := dmp.ToMap()
	if result["device_mac"] != "aabbccddeeff" {
		t.Error("device_mac not preserved in round-trip")
	}
	if result["custom_field"] != "custom_value" {
		t.Error("custom_field not preserved in round-trip")
	}
}

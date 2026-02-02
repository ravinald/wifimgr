package api

import (
	"context"
	"strings"
)

// SearchWiredClients implements the search for wired clients with mock data using new bidirectional types
func (m *MockClient) SearchWiredClients(ctx context.Context, orgID string, text string) (*MistWiredClientResponse, error) {
	// Helper function to create string pointers
	strPtr := func(s string) *string { return &s }
	intPtr := func(i int) *int { return &i }
	float64Ptr := func(f float64) *float64 { return &f }
	boolPtr := func(b bool) *bool { return &b }

	// Create some mock wired clients for testing using the new types
	mockClient := &MistWiredClient{
		OrgID:     strPtr("623c5d95-b7ba-4dc1-8e4d-97e7740f46f3"),
		SiteID:    strPtr("3fc11aba-584d-4834-b643-011268003851"),
		Timestamp: float64Ptr(1747872303.084),
		DeviceMacPort: []*MistDeviceMacPort{
			{
				DeviceMAC:  strPtr("74e7984afd02"),
				PortID:     strPtr("ge-0/0/15"),
				Start:      strPtr("2025-03-04T07:11:48.878+0000"),
				When:       strPtr("2025-05-22T00:05:03.084+0000"),
				IP:         strPtr("10.67.5.158"),
				IP6:        strPtr("fe80:0:0:0:e630:22ff:fe62:1cb6"),
				VLAN:       intPtr(104),
				PortParent: strPtr(""),
				Node:       strPtr(""),
			},
		},
		DeviceMAC:     []string{"74e7984afd02"},
		Username:      []string{},
		VLAN:          []int{104},
		PortID:        []string{"ge-0/0/15"},
		Hostname:      []string{"HTW_e43022621cb6"},
		IP:            []string{"10.67.5.158"},
		IP6:           []string{"fe80:0:0:0:e630:22ff:fe62:1cb6"},
		Manufacture:   strPtr("Hanwha VisionVietNam"),
		ClientMAC:     strPtr("e43022621cb6"),
		RandomMAC:     boolPtr(false),
		AuthState:     strPtr(""),
		AuthMethod:    strPtr(""),
		LastVLANName:  strPtr("Security"),
		LastVLAN:      intPtr(104),
		LastPortID:    strPtr("ge-0/0/15"),
		LastHostname:  strPtr("HTW_e43022621cb6"),
		LastIP:        strPtr("10.67.5.158"),
		LastIP6:       strPtr("fe80:0:0:0:e630:22ff:fe62:1cb6"),
		LastDeviceMAC: strPtr("74e7984afd02"),
		MAC:           strPtr("e43022621cb6"),
	}

	// Set raw data to preserve all fields
	mockClient.SetRaw(map[string]interface{}{
		"org_id":          mockClient.OrgID,
		"site_id":         mockClient.SiteID,
		"timestamp":       mockClient.Timestamp,
		"device_mac_port": mockClient.DeviceMacPort,
		"device_mac":      mockClient.DeviceMAC,
		"username":        mockClient.Username,
		"vlan":            mockClient.VLAN,
		"port_id":         mockClient.PortID,
		"hostname":        mockClient.Hostname,
		"ip":              mockClient.IP,
		"ip6":             mockClient.IP6,
		"manufacture":     mockClient.Manufacture,
		"client_mac":      mockClient.ClientMAC,
		"random_mac":      mockClient.RandomMAC,
		"auth_state":      mockClient.AuthState,
		"auth_method":     mockClient.AuthMethod,
		"last_vlan_name":  mockClient.LastVLANName,
		"last_vlan":       mockClient.LastVLAN,
		"last_port_id":    mockClient.LastPortID,
		"last_hostname":   mockClient.LastHostname,
		"last_ip":         mockClient.LastIP,
		"last_ip6":        mockClient.LastIP6,
		"last_device_mac": mockClient.LastDeviceMAC,
		"mac":             mockClient.MAC,
	})

	// Filter based on search text (case-insensitive)
	var results []*MistWiredClient
	searchTextLower := strings.ToLower(text)

	// Check if any of the fields match the search text
	if strings.Contains(strings.ToLower(mockClient.GetMAC()), searchTextLower) ||
		(len(mockClient.IP) > 0 && strings.Contains(strings.ToLower(mockClient.IP[0]), searchTextLower)) ||
		(len(mockClient.Hostname) > 0 && strings.Contains(strings.ToLower(mockClient.Hostname[0]), searchTextLower)) ||
		(len(mockClient.DeviceMAC) > 0 && strings.Contains(strings.ToLower(mockClient.DeviceMAC[0]), searchTextLower)) ||
		(len(mockClient.PortID) > 0 && strings.Contains(strings.ToLower(mockClient.PortID[0]), searchTextLower)) {
		results = append(results, mockClient)
	}

	// Create the response
	int64Ptr := func(i int64) *int64 { return &i }
	response := &MistWiredClientResponse{
		Results: results,
		Limit:   intPtr(1000),
		Start:   int64Ptr(1747786063),
		End:     int64Ptr(1747872463),
		Total:   intPtr(len(results)),
	}

	return response, nil
}

// SearchWirelessClients implements the search for wireless clients with mock data using new bidirectional types
func (m *MockClient) SearchWirelessClients(ctx context.Context, orgID string, text string) (*MistWirelessClientResponse, error) {
	// Helper function to create string pointers
	strPtr := func(s string) *string { return &s }
	intPtr := func(i int) *int { return &i }
	float64Ptr := func(f float64) *float64 { return &f }
	boolPtr := func(b bool) *bool { return &b }

	// Create some mock wireless clients for testing using the new types
	mockClient := &MistWirelessClient{
		SiteID:        strPtr("3fc11aba-584d-4834-b643-011268003851"),
		SiteIDs:       []string{"3fc11aba-584d-4834-b643-011268003851"},
		OrgID:         strPtr("623c5d95-b7ba-4dc1-8e4d-97e7740f46f3"),
		AP:            []string{"003e7312591c", "003e7311b0c0", "003e73125197"},
		IP:            []string{"10.67.22.210", "10.67.22.192", "10.67.21.90"},
		Hostname:      []string{"99N231001849\u0000"},
		WLANID:        []string{"171f929b-2fb2-4fea-85b7-7334b82ce042"},
		SSID:          []string{"FlexHouse"},
		Timestamp:     float64Ptr(1747873154.399),
		Model:         []string{},
		Device:        []string{},
		OS:            []string{},
		OSVersion:     []string{},
		Username:      []string{},
		Manufacture:   strPtr("TAIYO YUDEN CO."),
		Hardware:      strPtr(""),
		Firmware:      []string{},
		SDKVersion:    []string{},
		AppVersion:    []string{},
		VLAN:          []int{120},
		RandomMAC:     boolPtr(false),
		Ftc:           boolPtr(false),
		Band:          strPtr("5"),
		Protocol:      strPtr("ac"),
		PskID:         []string{},
		PskName:       []string{},
		LastAP:        strPtr("003e7312591c"),
		LastIP:        strPtr("10.67.22.210"),
		LastHostname:  strPtr("99N231001849\u0000"),
		LastWLANID:    strPtr("171f929b-2fb2-4fea-85b7-7334b82ce042"),
		LastSSID:      strPtr("FlexHouse"),
		LastModel:     strPtr(""),
		LastDevice:    strPtr(""),
		LastOS:        strPtr(""),
		LastOSVersion: strPtr(""),
		LastFirmware:  strPtr(""),
		LastVLAN:      intPtr(120),
		MAC:           strPtr("48a493d024fa"),
	}

	// Set raw data to preserve all fields
	mockClient.SetRaw(map[string]interface{}{
		"site_id":         mockClient.SiteID,
		"site_ids":        mockClient.SiteIDs,
		"org_id":          mockClient.OrgID,
		"ap":              mockClient.AP,
		"ip":              mockClient.IP,
		"hostname":        mockClient.Hostname,
		"wlan_id":         mockClient.WLANID,
		"ssid":            mockClient.SSID,
		"timestamp":       mockClient.Timestamp,
		"model":           mockClient.Model,
		"device":          mockClient.Device,
		"os":              mockClient.OS,
		"os_version":      mockClient.OSVersion,
		"username":        mockClient.Username,
		"mfg":             mockClient.Manufacture,
		"hardware":        mockClient.Hardware,
		"firmware":        mockClient.Firmware,
		"sdk_version":     mockClient.SDKVersion,
		"app_version":     mockClient.AppVersion,
		"vlan":            mockClient.VLAN,
		"random_mac":      mockClient.RandomMAC,
		"ftc":             mockClient.Ftc,
		"band":            mockClient.Band,
		"protocol":        mockClient.Protocol,
		"psk_id":          mockClient.PskID,
		"psk_name":        mockClient.PskName,
		"last_ap":         mockClient.LastAP,
		"last_ip":         mockClient.LastIP,
		"last_hostname":   mockClient.LastHostname,
		"last_wlan_id":    mockClient.LastWLANID,
		"last_ssid":       mockClient.LastSSID,
		"last_model":      mockClient.LastModel,
		"last_device":     mockClient.LastDevice,
		"last_os":         mockClient.LastOS,
		"last_os_version": mockClient.LastOSVersion,
		"last_firmware":   mockClient.LastFirmware,
		"last_vlan":       mockClient.LastVLAN,
		"mac":             mockClient.MAC,
	})

	// Filter based on search text (case-insensitive)
	var results []*MistWirelessClient
	searchTextLower := strings.ToLower(text)

	// Check if any of the fields match the search text
	if strings.Contains(strings.ToLower(mockClient.GetMAC()), searchTextLower) ||
		(len(mockClient.IP) > 0 && strings.Contains(strings.ToLower(mockClient.IP[0]), searchTextLower)) ||
		(len(mockClient.Hostname) > 0 && strings.Contains(strings.ToLower(mockClient.Hostname[0]), searchTextLower)) ||
		(len(mockClient.SSID) > 0 && strings.Contains(strings.ToLower(mockClient.SSID[0]), searchTextLower)) ||
		(len(mockClient.AP) > 0 && strings.Contains(strings.ToLower(mockClient.AP[0]), searchTextLower)) {
		results = append(results, mockClient)
	}

	// Create the response
	int64Ptr := func(i int64) *int64 { return &i }
	response := &MistWirelessClientResponse{
		Results: results,
		Limit:   intPtr(10),
		Start:   int64Ptr(1746663804),
		End:     int64Ptr(1747873404),
		Total:   intPtr(len(results)),
	}

	return response, nil
}

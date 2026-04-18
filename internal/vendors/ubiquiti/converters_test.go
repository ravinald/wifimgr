package ubiquiti

import (
	"testing"
)

func TestNormalizeMAC(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"AA:BB:CC:DD:EE:FF", "aabbccddeeff"},
		{"aa-bb-cc-dd-ee-ff", "aabbccddeeff"},
		{"aabb.ccdd.eeff", "aabbccddeeff"},
		{"aabbccddeeff", "aabbccddeeff"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeMAC(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeMAC(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestNormalizeStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"online", "connected"},
		{"Online", "connected"},
		{"ONLINE", "connected"},
		{"offline", "disconnected"},
		{"Offline", "disconnected"},
		{"unknown", "unknown"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeStatus(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeStatus(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestConvertSiteToSiteInfo(t *testing.T) {
	hostNameMap := map[string]string{
		"host-1": "Office-UDM",
	}

	tests := []struct {
		name      string
		site      Site
		wantName  string
		wantNotes string
	}{
		{
			name: "default site uses host name",
			site: Site{
				SiteID: "site-1",
				HostID: "host-1",
				Meta:   SiteMeta{Name: "default", Timezone: "America/New_York"},
			},
			wantName:  "Office-UDM",
			wantNotes: "",
		},
		{
			name: "custom site name combines host and site",
			site: Site{
				SiteID: "site-2",
				HostID: "host-1",
				Meta:   SiteMeta{Name: "Floor2", Timezone: "America/New_York", Desc: "Second floor"},
			},
			wantName:  "Office-UDM - Floor2",
			wantNotes: "Second floor",
		},
		{
			name: "no host name falls back to site name",
			site: Site{
				SiteID: "site-5",
				HostID: "host-unknown",
				Meta:   SiteMeta{Name: "Standalone Site"},
			},
			wantName:  "Standalone Site",
			wantNotes: "",
		},
		{
			name: "site with description populates notes",
			site: Site{
				SiteID: "site-6",
				HostID: "host-1",
				Meta:   SiteMeta{Name: "Lab", Desc: "Test lab"},
			},
			wantName:  "Office-UDM - Lab",
			wantNotes: "Test lab",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := convertSiteToSiteInfo(tt.site, hostNameMap)
			if info.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", info.Name, tt.wantName)
			}
			if info.Notes != tt.wantNotes {
				t.Errorf("Notes = %q, want %q", info.Notes, tt.wantNotes)
			}
			if info.SourceVendor != "ubiquiti" {
				t.Errorf("SourceVendor = %q, want %q", info.SourceVendor, "ubiquiti")
			}
		})
	}
}

func TestSiteGetID(t *testing.T) {
	tests := []struct {
		name   string
		site   Site
		wantID string
	}{
		{"returns siteId", Site{SiteID: "primary"}, "primary"},
		{"empty", Site{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.site.GetID(); got != tt.wantID {
				t.Errorf("GetID() = %q, want %q", got, tt.wantID)
			}
		})
	}
}

func TestSiteGetName(t *testing.T) {
	tests := []struct {
		name     string
		site     Site
		wantName string
	}{
		{"returns meta.name", Site{Meta: SiteMeta{Name: "Meta"}}, "Meta"},
		{"empty", Site{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.site.GetName(); got != tt.wantName {
				t.Errorf("GetName() = %q, want %q", got, tt.wantName)
			}
		})
	}
}

func TestBuildHostNameMap(t *testing.T) {
	hosts := []Host{
		{ID: "h1", ReportedState: ReportedState{Name: "Office-UDM", Hostname: "udm-office"}},
		{ID: "h2", ReportedState: ReportedState{Name: "", Hostname: "warehouse-ucg"}},
		{ID: "h3", ReportedState: ReportedState{Name: "", Hostname: ""}},
	}

	m := buildHostNameMap(hosts)

	if m["h1"] != "Office-UDM" {
		t.Errorf("m[h1] = %q, want %q", m["h1"], "Office-UDM")
	}
	if m["h2"] != "warehouse-ucg" {
		t.Errorf("m[h2] = %q, want %q", m["h2"], "warehouse-ucg")
	}
	if _, ok := m["h3"]; ok {
		t.Errorf("m[h3] should not exist, got %q", m["h3"])
	}
}

func TestConvertFlatDeviceToInventoryItem(t *testing.T) {
	d := FlatDevice{
		Device: Device{
			ID:          "dev-1",
			MAC:         "AA:BB:CC:DD:EE:FF",
			Name:        "AP-Lobby",
			Model:       "U6-Pro",
			ProductLine: "network",
			IsManaged:   true,
		},
		HostID:   "host-1",
		HostName: "UDM-Pro",
		SiteID:   "site-1",
	}

	item := convertFlatDeviceToInventoryItem(d)

	if item.ID != "dev-1" {
		t.Errorf("ID = %q, want %q", item.ID, "dev-1")
	}
	if item.MAC != "aabbccddeeff" {
		t.Errorf("MAC = %q, want %q", item.MAC, "aabbccddeeff")
	}
	if item.Type != "ap" {
		t.Errorf("Type = %q, want %q", item.Type, "ap")
	}
	if item.SiteID != "site-1" {
		t.Errorf("SiteID = %q, want %q", item.SiteID, "site-1")
	}
	if !item.Claimed {
		t.Error("Claimed = false, want true")
	}
}

func TestConvertFlatDeviceToDeviceInfo(t *testing.T) {
	d := FlatDevice{
		Device: Device{
			ID:      "dev-1",
			MAC:     "aa:bb:cc:dd:ee:ff",
			Name:    "SW-Rack1",
			Model:   "USW-Pro-24-PoE",
			IP:      "192.168.1.10",
			Status:  "online",
			Version: "7.0.83",
			Note:    "rack switch",
		},
		SiteID: "site-2",
	}

	info := convertFlatDeviceToDeviceInfo(d)

	if info.Type != "switch" {
		t.Errorf("Type = %q, want %q", info.Type, "switch")
	}
	if info.Status != "connected" {
		t.Errorf("Status = %q, want %q", info.Status, "connected")
	}
	if info.IP != "192.168.1.10" {
		t.Errorf("IP = %q, want %q", info.IP, "192.168.1.10")
	}
}

func TestFlattenDevices(t *testing.T) {
	groups := []HostDeviceGroup{
		{
			HostID:   "host-1",
			HostName: "UDM-Pro",
			Devices: []Device{
				{ID: "1", MAC: "aa:bb:cc:dd:ee:01", Model: "U6-Pro", ProductLine: "network"},
				{ID: "2", MAC: "aa:bb:cc:dd:ee:02", Model: "UNVR-Pro", ProductLine: "protect"},
				{ID: "3", MAC: "aa:bb:cc:dd:ee:03", Model: "USW-Pro-24-PoE", ProductLine: "network"},
			},
		},
		{
			HostID:   "host-2",
			HostName: "UDM-SE",
			Devices: []Device{
				{ID: "4", MAC: "aa:bb:cc:dd:ee:04", Model: "U7-Pro", ProductLine: "network"},
			},
		},
	}

	hostSiteMap := map[string]string{
		"host-1": "site-1",
		"host-2": "site-2",
	}

	result := flattenDevices(groups, hostSiteMap)

	if len(result) != 3 {
		t.Fatalf("flattenDevices returned %d devices, want 3", len(result))
	}
	if result[0].SiteID != "site-1" {
		t.Errorf("result[0].SiteID = %q, want %q", result[0].SiteID, "site-1")
	}
	if result[2].SiteID != "site-2" {
		t.Errorf("result[2].SiteID = %q, want %q", result[2].SiteID, "site-2")
	}
}

func TestBuildHostNameMapFromDevices(t *testing.T) {
	groups := []HostDeviceGroup{
		{HostID: "host-1", HostName: "Office-UDM", Devices: []Device{{ID: "d1"}}},
		{HostID: "host-2", HostName: "Warehouse", Devices: []Device{{ID: "d2"}}},
		{HostID: "", HostName: "NoID", Devices: nil},
	}

	m := buildHostNameMapFromDevices(groups)

	if m["host-1"] != "Office-UDM" {
		t.Errorf("m[host-1] = %q, want %q", m["host-1"], "Office-UDM")
	}
	if m["host-2"] != "Warehouse" {
		t.Errorf("m[host-2] = %q, want %q", m["host-2"], "Warehouse")
	}
	if _, ok := m[""]; ok {
		t.Error("empty hostID should not be in map")
	}
}

func TestBuildHostSiteMap(t *testing.T) {
	sites := []Site{
		{SiteID: "site-1", HostID: "host-1"},
		{SiteID: "site-2", HostID: "host-2"},
	}

	m := buildHostSiteMap(sites)

	if m["host-1"] != "site-1" {
		t.Errorf("m[host-1] = %q, want %q", m["host-1"], "site-1")
	}
	if m["host-2"] != "site-2" {
		t.Errorf("m[host-2] = %q, want %q", m["host-2"], "site-2")
	}
}

package cmd

import (
	"testing"

	"github.com/ravinald/wifimgr/internal/formatter"
	"github.com/ravinald/wifimgr/internal/vendors"
)

func columnFields(cols []formatter.TableColumn) []string {
	out := make([]string, 0, len(cols))
	for _, c := range cols {
		out = append(out, c.Field)
	}
	return out
}

func containsString(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

func TestEnrichWirelessClientFromCache(t *testing.T) {
	cache := &vendors.APICache{}
	cache.Inventory.AP = map[string]*vendors.InventoryItem{
		"aabbccddeeff": {Name: "ap2-15"},
	}
	cache.SiteIndex.ByID = map[string]string{
		"L_123": "MX-MEX-904EN",
	}

	tests := []struct {
		name         string
		in           *vendors.WirelessClient
		wantAPName   string
		wantSiteName string
	}{
		{
			name:         "empty APName filled from inventory",
			in:           &vendors.WirelessClient{APMAC: "aa:bb:cc:dd:ee:ff", SiteID: "L_123"},
			wantAPName:   "ap2-15",
			wantSiteName: "MX-MEX-904EN",
		},
		{
			name:         "existing APName not overwritten",
			in:           &vendors.WirelessClient{APName: "already-set", APMAC: "aa:bb:cc:dd:ee:ff", SiteID: "L_123"},
			wantAPName:   "already-set",
			wantSiteName: "MX-MEX-904EN",
		},
		{
			name:         "missing inventory entry leaves fields empty",
			in:           &vendors.WirelessClient{APMAC: "99:99:99:99:99:99", SiteID: "L_unknown"},
			wantAPName:   "",
			wantSiteName: "",
		},
		{
			name:         "no APMAC means no AP lookup",
			in:           &vendors.WirelessClient{SiteID: "L_123"},
			wantAPName:   "",
			wantSiteName: "MX-MEX-904EN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enrichWirelessClientFromCache(tt.in, cache)
			if tt.in.APName != tt.wantAPName {
				t.Errorf("APName = %q, want %q", tt.in.APName, tt.wantAPName)
			}
			if tt.in.SiteName != tt.wantSiteName {
				t.Errorf("SiteName = %q, want %q", tt.in.SiteName, tt.wantSiteName)
			}
		})
	}
}

func TestEnrichWirelessClientFromCache_NilSafety(t *testing.T) {
	// A nil cache is the best-effort path — the function must not panic and
	// must not modify the client.
	c := &vendors.WirelessClient{APMAC: "aa:bb:cc:dd:ee:ff", SiteID: "L_123"}
	enrichWirelessClientFromCache(c, nil)
	if c.APName != "" || c.SiteName != "" {
		t.Errorf("nil cache should leave client untouched; got %+v", c)
	}

	// Nil client must also be safe.
	enrichWirelessClientFromCache(nil, &vendors.APICache{})
}

func TestEnrichWiredClientFromCache(t *testing.T) {
	cache := &vendors.APICache{}
	cache.Inventory.Switch = map[string]*vendors.InventoryItem{
		"aabbccddeeff": {Name: "sw-core-1"},
	}
	cache.SiteIndex.ByID = map[string]string{
		"L_456": "US-LAB-01",
	}

	c := &vendors.WiredClient{SwitchMAC: "aa:bb:cc:dd:ee:ff", SiteID: "L_456"}
	enrichWiredClientFromCache(c, cache)
	if c.SwitchName != "sw-core-1" {
		t.Errorf("SwitchName = %q, want %q", c.SwitchName, "sw-core-1")
	}
	if c.SiteName != "US-LAB-01" {
		t.Errorf("SiteName = %q, want %q", c.SiteName, "US-LAB-01")
	}
}

func TestBuildWirelessSearchColumns_SiteFilter(t *testing.T) {
	// Save and restore the package-level apiFlag so test ordering doesn't
	// leak state into other tests.
	orig := apiFlag
	t.Cleanup(func() { apiFlag = orig })

	tests := []struct {
		name           string
		siteFilter     string
		apiFlagValue   string
		targetAPICount int
		wantContains   []string
		wantOmits      []string
	}{
		{
			name:           "no filter, multi-API shows Site and API",
			siteFilter:     "",
			apiFlagValue:   "",
			targetAPICount: 2,
			wantContains:   []string{"site_name", "api"},
		},
		{
			name:           "site filter hides Site column",
			siteFilter:     "US-LAB-01",
			apiFlagValue:   "",
			targetAPICount: 2,
			wantContains:   []string{"api"},
			wantOmits:      []string{"site_name"},
		},
		{
			name:           "single-API target omits API column",
			siteFilter:     "",
			apiFlagValue:   "mist-prod",
			targetAPICount: 1,
			wantContains:   []string{"site_name"},
			wantOmits:      []string{"api"},
		},
		{
			name:           "site filter + single API strips both",
			siteFilter:     "US-LAB-01",
			apiFlagValue:   "mist-prod",
			targetAPICount: 1,
			wantOmits:      []string{"site_name", "api"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiFlag = tt.apiFlagValue
			cols := buildWirelessSearchColumns(tt.siteFilter, tt.targetAPICount, false)
			fields := columnFields(cols)
			for _, want := range tt.wantContains {
				if !containsString(fields, want) {
					t.Errorf("expected field %q in %v", want, fields)
				}
			}
			for _, omit := range tt.wantOmits {
				if containsString(fields, omit) {
					t.Errorf("did not expect field %q in %v", omit, fields)
				}
			}
		})
	}
}

func TestIsOnlineStatus(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"Online", true},
		{"online", true},
		{"ONLINE", true},
		{"Offline", false},
		{"offline", false},
		{"", true}, // unknown state defaults to "keep the row"
	}
	for _, tt := range tests {
		if got := isOnlineStatus(tt.in); got != tt.want {
			t.Errorf("isOnlineStatus(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestBuildWirelessSearchColumns_DetailShowsBandAndState(t *testing.T) {
	orig := apiFlag
	t.Cleanup(func() { apiFlag = orig })
	apiFlag = ""

	cols := buildWirelessSearchColumns("site", 1, true)
	fields := columnFields(cols)
	titles := columnTitles(cols)

	if !containsString(fields, "band") || !containsString(fields, "state") {
		t.Errorf("detail=true: expected band and state columns, got %v", fields)
	}
	// Header markers should surface on band/state so the footer timestamp
	// maps unambiguously.
	if !containsString(titles, "Band [*]") {
		t.Errorf("expected 'Band [*]' header, got %v", titles)
	}
	if !containsString(titles, "State [*]") {
		t.Errorf("expected 'State [*]' header, got %v", titles)
	}
}

func TestBuildWirelessSearchColumns_DefaultHidesBandAndState(t *testing.T) {
	orig := apiFlag
	t.Cleanup(func() { apiFlag = orig })
	apiFlag = ""

	cols := buildWirelessSearchColumns("", 2, false)
	fields := columnFields(cols)
	if containsString(fields, "band") || containsString(fields, "state") {
		t.Errorf("detail=false: band/state should be hidden, got %v", fields)
	}
}

func columnTitles(cols []formatter.TableColumn) []string {
	out := make([]string, 0, len(cols))
	for _, c := range cols {
		out = append(out, c.Title)
	}
	return out
}

func TestBuildWiredSearchColumns_SiteFilter(t *testing.T) {
	orig := apiFlag
	t.Cleanup(func() { apiFlag = orig })
	apiFlag = ""

	cols := buildWiredSearchColumns("", 2, false)
	if !containsString(columnFields(cols), "site_name") {
		t.Errorf("no site filter: expected site_name column")
	}

	cols = buildWiredSearchColumns("US-LAB-01", 2, false)
	if containsString(columnFields(cols), "site_name") {
		t.Errorf("site filter: expected site_name to be omitted")
	}
}


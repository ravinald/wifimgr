package aruba

import (
	"strings"
	"testing"
)

// buildTable renders a fixed-width table (header, dash separator, rows) the way
// an Instant `show` command would, so the parser is tested against real column
// alignment rather than hand-counted spacing.
func buildTable(headers []string, widths []int, rows [][]string) string {
	pad := func(cells []string, fill string) string {
		parts := make([]string, len(cells))
		for i, c := range cells {
			w := widths[i]
			if fill == "-" {
				c = strings.Repeat("-", w)
			}
			if len(c) < w {
				c += strings.Repeat(" ", w-len(c))
			}
			parts[i] = c
		}
		return strings.Join(parts, "  ")
	}

	var b strings.Builder
	b.WriteString(pad(headers, "") + "\n")
	b.WriteString(pad(headers, "-") + "\n")
	for _, r := range rows {
		b.WriteString(pad(r, "") + "\n")
	}
	return b.String()
}

func TestParseShowAPs_WithMAC(t *testing.T) {
	headers := []string{"Name", "IP Address", "Type", "Serial#", "MAC Address", "Status"}
	widths := []int{8, 16, 13, 14, 18, 8}
	rows := [][]string{
		{"ap-01", "172.68.104.253", "315(indoor)", "CN0001ABCDE", "00:11:22:33:44:55", "up"},
		{"ap-02", "172.68.104.252", "315(indoor)", "CN0002FGHIJ", "00:11:22:33:44:66", "down"},
	}
	text := "2 Access Points\n" + buildTable(headers, widths, rows)

	aps := parseShowAPs(text)
	if len(aps) != 2 {
		t.Fatalf("got %d APs, want 2", len(aps))
	}

	a := aps[0]
	if a.Name != "ap-01" {
		t.Errorf("name = %q", a.Name)
	}
	if a.IP != "172.68.104.253" {
		t.Errorf("ip = %q", a.IP)
	}
	if a.Model != "315" {
		t.Errorf("model = %q, want 315", a.Model)
	}
	if a.Serial != "CN0001ABCDE" {
		t.Errorf("serial = %q", a.Serial)
	}
	if a.MAC != "001122334455" {
		t.Errorf("mac = %q, want 001122334455", a.MAC)
	}
	if a.Status != "connected" {
		t.Errorf("status = %q, want connected", a.Status)
	}

	if aps[1].Status != "disconnected" {
		t.Errorf("ap-02 status = %q, want disconnected", aps[1].Status)
	}
}

func TestParseShowAPs_NoMACColumn(t *testing.T) {
	headers := []string{"Name", "IP Address", "Type", "Serial#"}
	widths := []int{8, 16, 13, 14}
	rows := [][]string{
		{"ap-01", "10.0.0.10", "535(indoor)", "CN9999ZZZZZ"},
	}
	text := "1 Access Points\n" + buildTable(headers, widths, rows)

	aps := parseShowAPs(text)
	if len(aps) != 1 {
		t.Fatalf("got %d APs, want 1", len(aps))
	}
	if aps[0].MAC != "" {
		t.Errorf("mac = %q, want empty", aps[0].MAC)
	}

	// ID falls back to serial when MAC is absent.
	item := inventoryItemFromAP(aps[0], "site-1", "")
	if item.ID != "CN9999ZZZZZ" {
		t.Errorf("inventory ID = %q, want serial fallback", item.ID)
	}
	if item.Type != "ap" {
		t.Errorf("type = %q, want ap", item.Type)
	}
}

func TestParseFixedWidthTable_NoTable(t *testing.T) {
	if rows := parseFixedWidthTable("just some text\nwith no table"); rows != nil {
		t.Errorf("expected nil for non-table input, got %d rows", len(rows))
	}
}

func TestDeviceInfoFromAP(t *testing.T) {
	ap := apRow{Name: "ap-9", IP: "10.0.0.9", Model: "375", Serial: "S1", MAC: "aabbccddeeff", Status: "connected"}
	d := deviceInfoFromAP(ap, "site-x", "Site X")
	if d.MAC != "aabbccddeeff" || d.ID != "aabbccddeeff" {
		t.Errorf("mac/id = %q/%q", d.MAC, d.ID)
	}
	if d.SiteID != "site-x" || d.SiteName != "Site X" {
		t.Errorf("site = %q/%q", d.SiteID, d.SiteName)
	}
	if d.SourceVendor != "aruba" {
		t.Errorf("vendor = %q", d.SourceVendor)
	}
}

func TestDeviceStatusVocab(t *testing.T) {
	cases := map[string]string{
		"connected":    "online",
		"disconnected": "offline",
		"online":       "online",
		"":             "",
	}
	for in, want := range cases {
		if got := deviceStatusVocab(in); got != want {
			t.Errorf("deviceStatusVocab(%q) = %q, want %q", in, got, want)
		}
	}
}

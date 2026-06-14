package aruba

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// These tests run the parsers against output captured from a live Instant
// cluster (testdata/), so the column-range and quoted-token handling is pinned
// to real device formatting rather than reconstructed samples.

func readFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return string(data)
}

func TestGolden_ShowAPs(t *testing.T) {
	aps := parseShowAPs(stripCLIPrefix(readFixture(t, "show_aps.txt")))
	if len(aps) != 4 {
		t.Fatalf("got %d APs, want 4", len(aps))
	}

	bySerial := map[string]apRow{}
	for _, ap := range aps {
		bySerial[ap.Serial] = ap
	}

	master, ok := bySerial["VNNGK9W3RY"]
	if !ok {
		t.Fatalf("missing AP VNNGK9W3RY; got %+v", aps)
	}
	if master.Name != "us-oak-1506-2-1" {
		t.Errorf("name = %q", master.Name)
	}
	if master.IP != "172.30.8.21" { // trailing '*' (conductor marker) stripped
		t.Errorf("ip = %q, want 172.30.8.21 (no '*')", master.IP)
	}
	if master.Model != "535" {
		t.Errorf("model = %q, want 535", master.Model)
	}
	if master.Status != "connected" {
		t.Errorf("status = %q", master.Status)
	}
	if master.MAC != "" {
		t.Errorf("mac = %q, want empty (show aps has no MAC column)", master.MAC)
	}
}

func TestGolden_SummaryAPMACs(t *testing.T) {
	got := parseSummaryAPMACs(stripCLIPrefix(readFixture(t, "show_summary_aptable.txt")))
	want := map[string]string{
		"172.30.8.21": "d04dc6c8c8a4",
		"172.30.8.22": "d04dc6c8ca06",
		"172.30.8.24": "d04dc6c8ca9e",
		"172.30.8.23": "d04dc6c8cb3a",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d IP->MAC pairs, want %d: %v", len(got), len(want), got)
	}
	for ip, mac := range want {
		if got[ip] != mac {
			t.Errorf("ip %s -> %q, want %q", ip, got[ip], mac)
		}
	}
}

// TestCollectAPs_MergesMAC drives the show-aps + show-summary correlation through
// a stub VC, proving the MAC-less `show aps` rows get their ethernet MAC from the
// summary table by management IP.
func TestCollectAPs_MergesMAC(t *testing.T) {
	apsOut := readFixture(t, "show_aps.txt")
	summaryOut := readFixture(t, "show_summary_aptable.txt")

	writeEnvelope := func(w http.ResponseWriter, body string) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"Status": "Success", "Status-code": 0, "Command output": body,
		})
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/login":
			_, _ = w.Write([]byte(`{"Status":"Success","sid":"s"}`))
		case "/rest/show-cmd":
			cmd := r.URL.Query().Get("cmd") // Go decodes %20 to space
			switch {
			case strings.Contains(cmd, "summary"):
				writeEnvelope(w, summaryOut)
			case strings.Contains(cmd, "aps"):
				writeEnvelope(w, apsOut)
			default:
				http.Error(w, "unknown cmd", http.StatusBadRequest)
			}
		default:
			http.Error(w, "nf", http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c := NewClient("admin", "admin", srv.URL, WithHTTPClient(srv.Client()))
	c.minInterval = 0

	aps, err := collectAPs(context.Background(), c)
	if err != nil {
		t.Fatalf("collectAPs: %v", err)
	}
	if len(aps) != 4 {
		t.Fatalf("got %d APs, want 4", len(aps))
	}

	wantByIP := map[string]string{
		"172.30.8.21": "d04dc6c8c8a4",
		"172.30.8.22": "d04dc6c8ca06",
		"172.30.8.23": "d04dc6c8cb3a",
		"172.30.8.24": "d04dc6c8ca9e",
	}
	for _, ap := range aps {
		if ap.MAC != wantByIP[ap.IP] {
			t.Errorf("AP %s (%s) MAC = %q, want %q", ap.Name, ap.IP, ap.MAC, wantByIP[ap.IP])
		}
		// With a MAC present, inventory keys on it rather than the serial.
		if item := inventoryItemFromAP(ap, "site-1", ""); item.ID != ap.MAC {
			t.Errorf("inventory ID = %q, want MAC %q", item.ID, ap.MAC)
		}
	}
}

func TestGolden_RunningConfigWLANs(t *testing.T) {
	wlans := extractWLANs(parseRunningConfig(stripCLIPrefix(readFixture(t, "running_config.txt"))), "site-1")

	got := map[string]string{} // id -> "band/vlan"
	want := map[string]string{
		"moo":         "all/1024",
		"eye oh tea":  "all/1028",
		"moo two gee": "2.4/1024",
		"moo 5G":      "5/1024",
	}

	for _, w := range wlans {
		got[w.ID] = w.Band + "/" + strconv.Itoa(w.VLANID)
		if w.SSID != w.ID {
			t.Errorf("WLAN %q essid = %q, want match", w.ID, w.SSID)
		}
		if !w.Enabled {
			t.Errorf("WLAN %q should be enabled", w.ID)
		}
		if w.AuthType != "psk" || w.EncryptionMode != "wpa2" {
			t.Errorf("WLAN %q auth/enc = %q/%q, want psk/wpa2", w.ID, w.AuthType, w.EncryptionMode)
		}
	}

	for id, exp := range want {
		if got[id] != exp {
			t.Errorf("WLAN %q = %q, want %q (parsed: %v)", id, got[id], exp, got)
		}
	}
}

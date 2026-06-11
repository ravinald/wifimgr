package formatter

import (
	"strings"
	"testing"
)

func TestIsMACField(t *testing.T) {
	cases := map[string]bool{
		"mac":        true,
		"ap_mac":     true,
		"switch_mac": true,
		"bssid":      true,
		"base_mac":   true,
		"MAC":        true,
		"serial":     false,
		"name":       false,
		"ip":         false,
		"model":      false,
	}
	for field, want := range cases {
		if got := isMACField(field); got != want {
			t.Errorf("isMACField(%q) = %v, want %v", field, got, want)
		}
	}
}

func TestFormatMACDisplay(t *testing.T) {
	cases := map[string]string{
		"683a1e54490f":      "68:3a:1e:54:49:0f",
		"68:3A:1E:54:49:0F": "68:3a:1e:54:49:0f", // uppercase colon -> lowercase colon
		"68-3a-1e-54-49-0f": "68:3a:1e:54:49:0f",
		"683a.1e54.490f":    "68:3a:1e:54:49:0f",
		"":                  "",            // empty passes through
		"laptop-john":       "laptop-john", // hostname in mac column passes through
		"683a1e":            "683a1e",      // partial passes through
	}
	for in, want := range cases {
		if got := formatMACDisplay(in); got != want {
			t.Errorf("formatMACDisplay(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestColonizeMACValues_SerialUntouched(t *testing.T) {
	in := map[string]interface{}{
		"mac":    "683a1e54490f",
		"ap_mac": "001122334455",
		"serial": "001122334455", // same 12-hex shape as a MAC, must stay bare
		"name":   "ap74-1",
		"radios": []interface{}{
			map[string]interface{}{"bssid": "aabbccddeeff"},
		},
	}
	out := colonizeMACValues(in).(map[string]interface{})

	if out["mac"] != "68:3a:1e:54:49:0f" {
		t.Errorf("mac = %v, want colon-hex", out["mac"])
	}
	if out["ap_mac"] != "00:11:22:33:44:55" {
		t.Errorf("ap_mac = %v, want colon-hex", out["ap_mac"])
	}
	if out["serial"] != "001122334455" {
		t.Errorf("serial = %v, want untouched bare hex", out["serial"])
	}
	radios := out["radios"].([]interface{})
	if got := radios[0].(map[string]interface{})["bssid"]; got != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("nested bssid = %v, want colon-hex", got)
	}
}

func macPrinter(format string) *GenericTablePrinter {
	return &GenericTablePrinter{
		Config: TableConfig{
			Format: format,
			Columns: []TableColumn{
				{Field: "name", Title: "Name"},
				{Field: "mac", Title: "MAC"},
				{Field: "serial", Title: "Serial"},
			},
		},
		Data: []GenericTableData{
			{"name": "ap74-1", "mac": "683A1E54490F", "serial": "001122334455"},
		},
	}
}

func TestFormatAsCSV_MAC(t *testing.T) {
	out := macPrinter("csv").Print()
	if !strings.Contains(out, "68:3a:1e:54:49:0f") {
		t.Errorf("CSV missing colon-hex MAC:\n%s", out)
	}
	if strings.Contains(out, "683a1e54490f") || strings.Contains(out, "683A1E54490F") {
		t.Errorf("CSV still contains bare-hex MAC:\n%s", out)
	}
	if !strings.Contains(out, "001122334455") {
		t.Errorf("CSV serial should stay bare hex:\n%s", out)
	}
}

func TestFormatAsJSON_MAC(t *testing.T) {
	out := macPrinter("json").Print()
	if !strings.Contains(out, "68:3a:1e:54:49:0f") {
		t.Errorf("JSON missing colon-hex MAC:\n%s", out)
	}
	if strings.Contains(out, "683a1e54490f") || strings.Contains(out, "683A1E54490F") {
		t.Errorf("JSON still contains bare-hex MAC:\n%s", out)
	}
	if !strings.Contains(out, "001122334455") {
		t.Errorf("JSON serial should stay bare hex:\n%s", out)
	}
}

// Structured output is machine-facing: never any ANSI escape, regardless of
// terminal, so `| jq` and redirects stay clean.
func TestStructuredOutput_NoEscapes(t *testing.T) {
	for _, format := range []string{"json", "csv"} {
		out := macPrinter(format).Print()
		if strings.ContainsRune(out, '\x1b') {
			t.Errorf("%s output contains an ANSI escape:\n%q", format, out)
		}
	}
}

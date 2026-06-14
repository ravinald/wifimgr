package aruba

import (
	"strconv"
	"strings"

	"github.com/ravinald/wifimgr/internal/vendors"
)

// Write payloads for the Configuration and Action APIs. The SSID schema is
// fully documented in the Instant REST API Guide; the Action shapes (hostname,
// reboot) are modeled on the guide's examples and should be confirmed against a
// live VC. Deletes are expressed as action:"delete" in a POST body — Instant
// exposes no HTTP DELETE.

// ssidProfilePayload builds a /rest/ssid create-or-modify body from a WLAN.
// Instant upserts by profile name, so create doubles as update.
func ssidProfilePayload(w *vendors.WLAN) map[string]any {
	name := w.ID
	if name == "" {
		name = w.SSID
	}

	profile := map[string]any{
		"action":       "create",
		"ssid-profile": name,
		"essid": map[string]any{
			"action": "create",
			"value":  w.SSID,
		},
	}

	if opmode := opmodeFromAuth(w.AuthType, w.EncryptionMode); opmode != "" {
		profile["opmode"] = opmode
	}
	if w.PSK != "" {
		profile["wpa-passphrase"] = w.PSK
	}
	if band := bandToRFBand(w.Band); band != "" {
		profile["rf-band"] = band
	}
	if w.VLANID > 0 {
		profile["vlan"] = map[string]any{
			"action": "create",
			"value":  strconv.Itoa(w.VLANID),
		}
	}
	if w.Hidden {
		profile["hide-ssid"] = "enable"
	}
	if w.Enabled {
		profile["enable"] = "yes"
	} else {
		profile["disable"] = "yes"
	}

	return map[string]any{"ssid-profile": profile}
}

// ssidDeletePayload builds a /rest/ssid delete body for a profile name.
func ssidDeletePayload(name string) map[string]any {
	return map[string]any{
		"ssid-profile": map[string]any{
			"action":       "delete",
			"ssid-profile": name,
		},
	}
}

// hostnamePayload builds a /rest/hostname action body for one AP.
func hostnamePayload(iapIP, hostname string) map[string]any {
	return map[string]any{
		"iap_ip_addr":   iapIP,
		"hostname_info": map[string]any{"hostname": hostname},
	}
}

// rebootPayload builds a /rest/reboot action body for one AP.
func rebootPayload(iapIP string) map[string]any {
	return map[string]any{"iap_ip_addr": iapIP}
}

// bandToRFBand maps wifimgr band vocabulary to Instant rf-band values.
func bandToRFBand(band string) string {
	switch strings.ToLower(strings.TrimSpace(band)) {
	case "2.4":
		return "2.4"
	case "5", "dual":
		return "5.0"
	case "6":
		return "6"
	case "all":
		return "all"
	default:
		return ""
	}
}

// opmodeFromAuth maps (auth_type, encryption_mode) back to an Instant opmode.
// The inverse of authFromOpmode for the common cases wifimgr manages.
func opmodeFromAuth(auth, enc string) string {
	auth = strings.ToLower(strings.TrimSpace(auth))
	enc = strings.ToLower(strings.TrimSpace(enc))
	switch auth {
	case "", "open":
		if enc == "wpa3" {
			return "wpa3-open"
		}
		return "opensystem"
	case "psk":
		switch {
		case strings.HasPrefix(enc, "wpa2"):
			return "wpa2-psk-aes"
		case enc == "wpa", strings.Contains(enc, "wpa/wpa2"):
			return "wpa-psk-aes"
		default:
			return "wpa2-psk-aes"
		}
	case "sae", "wpa3", "wpa3-sae":
		return "wpa3-sae-aes"
	case "wpa2-enterprise", "wpa2-eap", "eap":
		return "wpa2-aes"
	case "mpsk":
		return "mpsk-aes"
	default:
		return ""
	}
}

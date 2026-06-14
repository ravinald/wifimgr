package aruba

import (
	"strconv"
	"strings"

	"github.com/ravinald/wifimgr/internal/vendors"
)

// Instant `show running-config` is a flat CLI dump: a profile is a non-indented
// header line followed by indented child lines, terminated by `!` or the next
// header. This file turns that text into structured profiles. It is the one
// brittle read surface in the adapter, so the grammar is kept deliberately
// simple (indentation + headers) and covered heavily by golden-file tests.

// configBlock is one CLI profile (header plus its indented body).
type configBlock struct {
	header string // full header line, e.g. "wlan ssid-profile Guest-Net"
	tokens []string
	lines  []configLine
}

// configLine is one indented child line within a block.
type configLine struct {
	key  string
	args []string
}

// parseRunningConfig splits a running-config dump into blocks. Lines outside any
// block (global single-line statements) become zero-child blocks so callers can
// still inspect them.
func parseRunningConfig(text string) []configBlock {
	var blocks []configBlock
	var cur *configBlock

	flush := func() {
		if cur != nil {
			blocks = append(blocks, *cur)
			cur = nil
		}
	}

	for _, rawLine := range strings.Split(text, "\n") {
		line := strings.TrimRight(rawLine, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		if strings.TrimSpace(line) == "!" {
			flush()
			continue
		}

		indented := line[0] == ' ' || line[0] == '\t'
		trimmed := strings.TrimSpace(line)

		if indented && cur != nil {
			fields := splitTokens(trimmed)
			cur.lines = append(cur.lines, configLine{key: fields[0], args: fields[1:]})
			continue
		}

		// Non-indented line starts a new block.
		flush()
		cur = &configBlock{header: trimmed, tokens: splitTokens(trimmed)}
	}
	flush()

	return blocks
}

// splitTokens splits a CLI line on whitespace, but keeps a double-quoted span
// as one token with the quotes removed. Instant names SSID and ESSID profiles
// with embedded spaces this way (e.g. `essid "eye oh tea"`).
func splitTokens(s string) []string {
	var tokens []string
	var b strings.Builder
	inQuote := false
	started := false // whether b holds a token (covers the empty quoted string)

	for _, r := range s {
		switch {
		case r == '"':
			inQuote = !inQuote
			started = true
		case (r == ' ' || r == '\t') && !inQuote:
			if started {
				tokens = append(tokens, b.String())
				b.Reset()
				started = false
			}
		default:
			b.WriteRune(r)
			started = true
		}
	}
	if started {
		tokens = append(tokens, b.String())
	}
	return tokens
}

// globalValue returns the arguments of the first top-level (zero-child)
// statement whose first token matches key, joined by spaces. Used for global
// settings like `organization <name>` or `virtual-controller-country <cc>`.
func globalValue(blocks []configBlock, key string) string {
	for _, b := range blocks {
		if len(b.lines) == 0 && len(b.tokens) >= 2 && b.tokens[0] == key {
			return strings.Join(b.tokens[1:], " ")
		}
	}
	return ""
}

// ssidProfiles returns the `wlan ssid-profile` blocks.
func ssidProfiles(blocks []configBlock) []configBlock {
	var out []configBlock
	for _, b := range blocks {
		if len(b.tokens) >= 3 && b.tokens[0] == "wlan" && b.tokens[1] == "ssid-profile" {
			out = append(out, b)
		}
	}
	return out
}

// extractWLANs converts ssid-profile blocks into vendor-neutral WLANs.
func extractWLANs(blocks []configBlock, siteID string) []*vendors.WLAN {
	var wlans []*vendors.WLAN
	for _, b := range ssidProfiles(blocks) {
		wlans = append(wlans, wlanFromBlock(b, siteID))
	}
	return wlans
}

func wlanFromBlock(b configBlock, siteID string) *vendors.WLAN {
	name := strings.Join(b.tokens[2:], " ") // profile name (the stable IAP identifier)

	cfg := map[string]any{}
	var (
		essid      string
		opmode     string
		band       string
		hasEnable  bool
		hasDisable bool
		hidden     bool
		vlanID     int
	)

	for _, ln := range b.lines {
		val := strings.Join(ln.args, " ")
		cfg[ln.key] = val
		switch ln.key {
		case "essid":
			essid = val
		case "opmode":
			opmode = val
		case "rf-band":
			band = normalizeBand(val)
		case "enable":
			hasEnable = true
		case "disable":
			hasDisable = true
		case "hide-ssid":
			hidden = true
		case "vlan":
			if len(ln.args) > 0 {
				if n, err := strconv.Atoi(ln.args[0]); err == nil {
					vlanID = n
				}
			}
		}
	}

	if essid == "" {
		essid = name
	}
	auth, enc := authFromOpmode(opmode)

	return &vendors.WLAN{
		ID:             name,
		SSID:           essid,
		SiteID:         siteID,
		Enabled:        hasEnable || !hasDisable,
		Hidden:         hidden,
		Band:           band,
		VLANID:         vlanID,
		AuthType:       auth,
		EncryptionMode: enc,
		Config:         cfg,
		SourceVendor:   vendorName,
	}
}

// normalizeBand maps Instant rf-band values to wifimgr's band vocabulary.
func normalizeBand(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "2.4", "2.4ghz":
		return "2.4"
	case "5", "5.0", "5ghz":
		return "5"
	case "6", "6ghz":
		return "6"
	case "all":
		return "all"
	default:
		return strings.ToLower(v)
	}
}

// authFromOpmode maps an Instant opmode to (auth_type, encryption_mode).
func authFromOpmode(opmode string) (string, string) {
	switch strings.ToLower(strings.TrimSpace(opmode)) {
	case "", "opensystem", "open":
		return "open", ""
	case "wpa3-open":
		return "open", "wpa3"
	case "wpa2-psk-aes", "mpsk-aes":
		return "psk", "wpa2"
	case "wpa3-sae-aes":
		return "sae", "wpa3"
	case "wpa-psk-tkip", "wpa-psk-aes":
		return "psk", "wpa"
	case "wpa-tkip", "wpa-tkipwpa2-aes", "wpa-psktkip":
		return "psk", "wpa/wpa2"
	case "wpa2-aes":
		return "wpa2-enterprise", "wpa2"
	case "static-wep", "dynamicwep":
		return "wep", "wep"
	default:
		return strings.ToLower(opmode), ""
	}
}

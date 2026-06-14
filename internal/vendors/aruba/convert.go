package aruba

import (
	"context"
	"regexp"
	"strings"

	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/vendors"
)

const vendorName = "aruba"

// summaryAPMarker anchors the "N Access Points" table inside `show summary`,
// which is the only place the swarm exposes AP name/IP alongside the ethernet
// MAC that wifimgr keys devices on (`show aps` omits the MAC).
var (
	summaryAPMarker = regexp.MustCompile(`(?m)^\s*\d+\s+Access Points\s*$`)
	ipv4Re          = regexp.MustCompile(`^\d{1,3}(\.\d{1,3}){3}$`)
)

// Instant `show` tables are fixed-width: a header row, a separator row of dash
// runs that marks exact column boundaries, then data rows aligned to those
// columns. Parsing off the dash runs (rather than guessing at whitespace) is
// the reliable way to read these tables, since values themselves contain
// spaces and parentheses.

// column is a half-open character range [start,end) within a fixed-width row.
type column struct {
	header     string
	start, end int
}

// parseFixedWidthTable reads the first dash-separated table in text and returns
// one map per data row, keyed by normalized header. Returns nil if no table is
// found. The final column absorbs any overflow past the separator width.
func parseFixedWidthTable(text string) []map[string]string {
	lines := strings.Split(text, "\n")
	sepIdx := -1
	for i, ln := range lines {
		if isSeparatorLine(strings.TrimRight(ln, "\r")) {
			sepIdx = i
			break
		}
	}
	if sepIdx <= 0 {
		return nil
	}

	cols := columnsFromSeparator(strings.TrimRight(lines[sepIdx], "\r"))
	if len(cols) == 0 {
		return nil
	}
	headerLine := strings.TrimRight(lines[sepIdx-1], "\r")
	for i := range cols {
		cols[i].header = normalizeHeader(sliceRange(headerLine, cols[i].start, cols[i].end))
	}

	var rows []map[string]string
	for _, ln := range lines[sepIdx+1:] {
		ln = strings.TrimRight(ln, "\r")
		if strings.TrimSpace(ln) == "" {
			break // tables end at the first blank line
		}
		if isSeparatorLine(ln) {
			continue
		}
		row := make(map[string]string, len(cols))
		for _, col := range cols {
			row[col.header] = strings.TrimSpace(sliceRange(ln, col.start, col.end))
		}
		rows = append(rows, row)
	}
	return rows
}

// isSeparatorLine reports whether a line is a column separator: only dashes and
// spaces, with at least two dash runs.
func isSeparatorLine(line string) bool {
	if strings.TrimSpace(line) == "" {
		return false
	}
	runs := 0
	inRun := false
	for _, r := range line {
		switch r {
		case '-':
			if !inRun {
				runs++
				inRun = true
			}
		case ' ':
			inRun = false
		default:
			return false
		}
	}
	return runs >= 2
}

// columnsFromSeparator derives column ranges from the dash runs in a separator.
// Each column spans from its dash-run start to the next column's start, not to
// the end of its own dash run: Instant sizes the dash run to the header label,
// which is often narrower than the data beneath it (e.g. a 4-dash "Type" run
// over "535(indoor)"). Column starts are reliable; dash-run ends are not. The
// final column runs to end of line.
func columnsFromSeparator(sep string) []column {
	var starts []int
	inRun := false
	for i, r := range sep {
		if r == '-' {
			if !inRun {
				starts = append(starts, i)
				inRun = true
			}
		} else {
			inRun = false
		}
	}

	cols := make([]column, len(starts))
	for i, start := range starts {
		end := 1 << 30 // last column absorbs the remainder; sliceRange clamps
		if i+1 < len(starts) {
			end = starts[i+1]
		}
		cols[i] = column{start: start, end: end}
	}
	return cols
}

// sliceRange returns line[start:end] clamped to the string bounds.
func sliceRange(line string, start, end int) string {
	if start < 0 {
		start = 0
	}
	if start >= len(line) {
		return ""
	}
	if end > len(line) {
		end = len(line)
	}
	return line[start:end]
}

// normalizeHeader lowercases a column header and strips spaces and '#' so
// "IP Address" and "Serial#" become "ipaddress" and "serial".
func normalizeHeader(h string) string {
	h = strings.ToLower(strings.TrimSpace(h))
	h = strings.ReplaceAll(h, " ", "")
	h = strings.ReplaceAll(h, "#", "")
	return h
}

// apRow is the identity extracted from one `show aps` row.
type apRow struct {
	Name   string
	IP     string
	Model  string
	Serial string
	MAC    string
	Status string
}

// collectAPs reads the swarm's APs, merging the ethernet MAC from
// `show summary` into the `show aps` rows by management IP. If the summary
// fetch fails the rows degrade to MAC-less (still usable for display, but the
// device/inventory keying falls back to serial).
func collectAPs(ctx context.Context, c *Client) ([]apRow, error) {
	out, err := c.ShowCommand(ctx, "show aps")
	if err != nil {
		return nil, err
	}
	aps := parseShowAPs(out)

	summary, err := c.ShowCommand(ctx, "show summary")
	if err != nil {
		logging.Debugf("[aruba] show summary failed; AP MACs unavailable: %v", err)
		return aps, nil
	}
	ipToMAC := parseSummaryAPMACs(summary)
	for i := range aps {
		if aps[i].MAC == "" {
			aps[i].MAC = ipToMAC[aps[i].IP]
		}
	}
	return aps, nil
}

// summaryAP is one row of the "N Access Points" table in `show summary`, which
// carries the ethernet MAC, management IP, and per-AP name together.
type summaryAP struct {
	MAC  string
	IP   string
	Name string
}

// parseSummaryAPs reads the "N Access Points" table from `show summary`. Scoping
// to the marker skips the earlier client table; per-row MAC/IP validation
// tolerates the next section bleeding in when no blank line separates them.
func parseSummaryAPs(text string) []summaryAP {
	loc := summaryAPMarker.FindStringIndex(text)
	if loc == nil {
		return nil
	}
	var out []summaryAP
	for _, row := range parseFixedWidthTable(text[loc[0]:]) {
		mac := normalizeMAC(pick(row, "mac"))
		ip := strings.TrimRight(firstField(pick(row, "ipaddress", "ip")), "*")
		if !isHexMAC(mac) || !ipv4Re.MatchString(ip) {
			continue
		}
		out = append(out, summaryAP{MAC: mac, IP: ip, Name: pick(row, "name")})
	}
	return out
}

// parseSummaryAPMACs maps AP management IP to ethernet MAC.
func parseSummaryAPMACs(text string) map[string]string {
	out := map[string]string{}
	for _, ap := range parseSummaryAPs(text) {
		out[ap.IP] = ap.MAC
	}
	return out
}

// summaryAPNames maps ethernet MAC to the per-AP name.
func summaryAPNames(text string) map[string]string {
	out := map[string]string{}
	for _, ap := range parseSummaryAPs(text) {
		if ap.Name != "" {
			out[ap.MAC] = ap.Name
		}
	}
	return out
}

// isHexMAC reports whether s is a 12-character hex MAC (no separators).
func isHexMAC(s string) bool {
	if len(s) != 12 {
		return false
	}
	for _, r := range s {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

// parseShowAPs reads `show aps` output into per-AP identity rows.
func parseShowAPs(text string) []apRow {
	var aps []apRow
	for _, row := range parseFixedWidthTable(text) {
		name := firstField(pick(row, "name"))
		if name == "" {
			continue
		}
		// Instant flags the conductor's row with a trailing '*' on its IP.
		ip := strings.TrimRight(firstField(pick(row, "ipaddress", "ip")), "*")
		ap := apRow{
			Name:   name,
			IP:     ip,
			Model:  stripParens(pick(row, "type", "model")),
			Serial: pick(row, "serial", "serialnumber"),
			MAC:    normalizeMAC(pick(row, "macaddress", "mac", "wiredmac")),
			Status: apStatus(ip, row),
		}
		aps = append(aps, ap)
	}
	return aps
}

// apStatus infers reachability: an AP reporting an IP is treated as connected,
// matching how Instant lists active members.
func apStatus(ip string, row map[string]string) string {
	if s := pick(row, "status", "state"); s != "" {
		return normalizeStatus(s)
	}
	if ip != "" && ip != "--" {
		return "connected"
	}
	return "disconnected"
}

func inventoryItemFromAP(ap apRow, siteID, siteName string) *vendors.InventoryItem {
	id := ap.MAC
	if id == "" {
		id = ap.Serial
	}
	return &vendors.InventoryItem{
		ID:           id,
		MAC:          ap.MAC,
		Serial:       ap.Serial,
		Model:        ap.Model,
		Name:         ap.Name,
		Type:         "ap",
		SiteID:       siteID,
		SiteName:     siteName,
		Claimed:      true,
		SourceVendor: vendorName,
	}
}

func deviceInfoFromAP(ap apRow, siteID, siteName string) *vendors.DeviceInfo {
	id := ap.MAC
	if id == "" {
		id = ap.Serial
	}
	return &vendors.DeviceInfo{
		ID:           id,
		MAC:          ap.MAC,
		Serial:       ap.Serial,
		Name:         ap.Name,
		Model:        ap.Model,
		Type:         "ap",
		SiteID:       siteID,
		SiteName:     siteName,
		Status:       ap.Status,
		IP:           ap.IP,
		SourceVendor: vendorName,
	}
}

// pick returns the first non-empty value among the given normalized keys.
func pick(row map[string]string, keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(row[k]); v != "" {
			return v
		}
	}
	return ""
}

// firstField returns the first whitespace-delimited token of s.
func firstField(s string) string {
	if f := strings.Fields(s); len(f) > 0 {
		return f[0]
	}
	return ""
}

// stripParens removes a trailing parenthetical, e.g. "225(indoor)" -> "225".
func stripParens(s string) string {
	if i := strings.IndexByte(s, '('); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}

// normalizeStatus maps Instant status text to wifimgr's vocabulary.
func normalizeStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "up", "online":
		return "connected"
	case "down", "offline":
		return "disconnected"
	default:
		return strings.ToLower(strings.TrimSpace(status))
	}
}

// normalizeMAC lowercases a MAC and strips separators.
func normalizeMAC(mac string) string {
	mac = strings.ReplaceAll(mac, ":", "")
	mac = strings.ReplaceAll(mac, "-", "")
	mac = strings.ReplaceAll(mac, ".", "")
	return strings.ToLower(strings.TrimSpace(mac))
}

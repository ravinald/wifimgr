package cmdutils

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// ContainsHelp checks if "help" is present in the arguments
func ContainsHelp(args []string) bool {
	for _, arg := range args {
		if strings.ToLower(arg) == "help" {
			return true
		}
	}
	return false
}

// Show verbosity levels (Junos-style). Default is the configured column set;
// detail and extensive widen the fields shown. extensive currently maps onto
// "all cache fields"; detail is plumbed but reserved for a future column tier.
const (
	VerbosityDetail    = "detail"
	VerbosityExtensive = "extensive"
)

// ParsedShowArgs represents parsed arguments for show commands
type ParsedShowArgs struct {
	Filter        string
	SiteName      string
	Target        string // API target label (e.g., "mist-prod", "meraki")
	ESSIDName     string // SSID name filter (from "essid" keyword)
	SortField     string // Secondary sort field (from "sort" keyword)
	Format        string
	ShowUnmanaged bool   // "all": widen object scope to everything the API has, not just managed
	Verbosity     string // "", "detail", or "extensive" (field verbosity)
	NoResolve     bool
	DeviceType    string
}

// AllFields reports whether every cache field should be shown (the "extensive"
// verbosity). Replaces the former json-only "all" keyword.
func (p *ParsedShowArgs) AllFields() bool {
	return p.Verbosity == VerbosityExtensive
}

// ParseShowArgs parses positional arguments for show commands
// Supports patterns like: [name-or-mac] [site site-name] [target api-label] [json|csv] [all] [no-resolve]
func ParseShowArgs(args []string) (*ParsedShowArgs, error) {
	result := &ParsedShowArgs{
		Format: "table", // default format
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch arg {
		case "site":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("'site' requires a site name")
			}
			if result.SiteName != "" {
				return nil, fmt.Errorf("site specified multiple times")
			}
			result.SiteName = StripQuotes(args[i+1])
			i++ // Skip the site name

		case "target":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("'target' requires an API label")
			}
			if result.Target != "" {
				return nil, fmt.Errorf("target specified multiple times")
			}
			result.Target = StripQuotes(args[i+1])
			i++ // Skip the API label

		case "essid":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("'essid' requires an SSID name")
			}
			if result.ESSIDName != "" {
				return nil, fmt.Errorf("essid specified multiple times")
			}
			result.ESSIDName = StripQuotes(args[i+1])
			i++ // Skip the SSID name

		case "sort":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("'sort' requires a field name (essid, ap)")
			}
			if result.SortField != "" {
				return nil, fmt.Errorf("sort specified multiple times")
			}
			sortVal := strings.ToLower(args[i+1])
			switch sortVal {
			case "essid", "ap":
				result.SortField = sortVal
			default:
				return nil, fmt.Errorf("invalid sort field %q: must be 'essid' or 'ap'", args[i+1])
			}
			i++ // Skip the sort field

		case "format":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("'format' requires a format type (json, csv)")
			}
			if result.Format != "table" {
				return nil, fmt.Errorf("format specified multiple times")
			}
			fmtVal := strings.ToLower(args[i+1])
			switch fmtVal {
			case "json", "csv", "alias":
				result.Format = fmtVal
			default:
				return nil, fmt.Errorf("invalid format %q: must be 'json', 'csv', or 'alias'", args[i+1])
			}
			i++ // Skip the format value

		case "json", "csv", "table", "alias":
			// Bare format tokens are no longer accepted; require the "format" keyword.
			return nil, fmt.Errorf("use 'format %s' instead of bare '%s'", arg, arg)

		case "all":
			// Object scope: show everything the API has, not just managed devices.
			result.ShowUnmanaged = true

		case VerbosityDetail, VerbosityExtensive:
			if result.Verbosity != "" {
				return nil, fmt.Errorf("verbosity specified multiple times (have %q)", result.Verbosity)
			}
			result.Verbosity = arg

		case "no-resolve":
			result.NoResolve = true

		case "ap", "aps", "switch", "switches", "sw", "gateway", "gateways", "gw":
			// Device type for inventory commands
			if result.DeviceType != "" {
				return nil, fmt.Errorf("device type specified multiple times")
			}
			result.DeviceType = NormalizeDeviceType(arg)

		default:
			// Must be a filter (name or MAC address)
			if result.Filter == "" {
				result.Filter = arg
			} else {
				return nil, fmt.Errorf("unexpected argument: %s", arg)
			}
		}
	}

	return result, nil
}

// ValidateShowAPArgs validates arguments for the show ap command
func ValidateShowAPArgs(_ *cobra.Command, args []string) error {
	// Allow "help" as a special keyword (handled in RunE)
	if ContainsHelp(args) {
		return nil
	}
	parsed, err := ParseShowArgs(args)
	if err != nil {
		return err
	}
	if parsed.Format == "alias" {
		return fmt.Errorf("alias format is only supported for 'show api bssid'")
	}
	return nil
}

// ValidateShowBSSIDArgs validates arguments for the show api bssid command.
// Identical to ValidateShowAPArgs but permits the bssid-only "alias" format.
func ValidateShowBSSIDArgs(_ *cobra.Command, args []string) error {
	if ContainsHelp(args) {
		return nil
	}
	_, err := ParseShowArgs(args)
	return err
}

// ParsedApplyArgs represents parsed arguments for apply commands
type ParsedApplyArgs struct {
	SiteName   string
	Operation  string
	DeviceType string
	ExtraArgs  []string
}

// ParseApplyArgs parses positional arguments for apply commands
// Supports pattern: <site-name> <operation> [extra-args...]
func ParseApplyArgs(args []string) (*ParsedApplyArgs, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("apply requires at least 2 arguments: <site-name> <operation>")
	}

	result := &ParsedApplyArgs{
		SiteName:  args[0],
		Operation: args[1],
	}

	// Handle device type operations
	switch result.Operation {
	case "ap", "aps", "switch", "switches", "sw", "gateway", "gateways", "gw", "all":
		result.DeviceType = NormalizeDeviceType(result.Operation)
		if result.DeviceType == "" {
			result.DeviceType = result.Operation // "all" case
		}
	}

	// Store any extra arguments
	if len(args) > 2 {
		result.ExtraArgs = args[2:]
	}

	return result, nil
}

// ImportOutputArgs carries the emit-control keywords every `import api …`
// subcommand shares: decrypt (reveal plaintext secrets vs the encrypted value),
// save (write to disk vs print), and file <name> (output path). Each importer
// embeds this and runs Consume before its own grammar switch so the shared
// keywords behave identically across commands; the command-specific keywords
// stay local.
type ImportOutputArgs struct {
	Decrypt    bool // emit decrypted plaintext secrets instead of the stored enc: value
	SaveMode   bool
	OutputFile string
}

// Consume handles the shared import keyword at args[i]. It reports whether the
// token matched, the index of the last token it consumed (callers advance past
// it), and any error. A non-match returns (false, i, nil) so the caller can fall
// through to its own keywords.
func (o *ImportOutputArgs) Consume(args []string, i int) (matched bool, last int, err error) {
	switch strings.ToLower(args[i]) {
	case "decrypt":
		o.Decrypt = true
	case "save":
		o.SaveMode = true
	case "file":
		if i+1 >= len(args) {
			return true, i, fmt.Errorf("'file' requires a filename")
		}
		o.OutputFile = StripQuotes(args[i+1])
		return true, i + 1, nil
	default:
		return false, i, nil
	}
	return true, i, nil
}

// Validate enforces the cross-keyword rule shared by import commands: a custom
// output file only makes sense when actually writing one.
func (o *ImportOutputArgs) Validate() error {
	if o.OutputFile != "" && !o.SaveMode {
		return fmt.Errorf("'file' requires 'save' to be specified")
	}
	return nil
}

// IsDeviceType reports whether s names a device type wifimgr models under
// "devices", accepting the same aliases as NormalizeDeviceType (aps, sw, gw…).
func IsDeviceType(s string) bool {
	switch NormalizeDeviceType(s) {
	case "ap", "switch", "gateway":
		return true
	default:
		return false
	}
}

// SetSiteAction discriminates what a parsed `set site` invocation does: write a
// config field into the site file, or toggle the inventory.json allowlist.
type SetSiteAction int

const (
	SetActionConfigField SetSiteAction = iota // <device-type> <name> <key-path> <value>
	SetActionArm                              // … managed
	SetActionDisarm                           // … unmanaged
)

// SetScope is the breadth of an arm/disarm: one named device, every device of a
// type at the site, or every device at the site.
type SetScope int

const (
	ScopeSingle    SetScope = iota // <device-type> <name>
	ScopeAllOfType                 // <device-type> all
	ScopeAllTypes                  // all
)

// ParsedSetSiteArgs is the result of parsing the tokens after the site name in a
// `set site <site> …` invocation. DeviceType is canonical (ap/switch/gateway)
// and empty only for ScopeAllTypes; Name is set only for ScopeSingle; KeyPath
// and RawValue are set only for SetActionConfigField.
type ParsedSetSiteArgs struct {
	Action     SetSiteAction
	Scope      SetScope
	DeviceType string
	Name       string
	KeyPath    string
	RawValue   string
}

// armAction maps the trailing managed/unmanaged keyword to its action; the bool
// reports whether the token was one of those keywords at all.
func armAction(token string) (SetSiteAction, bool) {
	switch token {
	case "managed":
		return SetActionArm, true
	case "unmanaged":
		return SetActionDisarm, true
	default:
		return SetActionConfigField, false
	}
}

// ParseSetSiteArgs parses the tokens that follow the site name in `set site`.
// The grammar is irregular, so disambiguation is positional:
//
//	all managed|unmanaged                       -> every device at the site
//	<device-type> all managed|unmanaged         -> every device of a type
//	<device-type> <name> managed|unmanaged      -> one device
//	<device-type> <name> <key-path> <value>     -> one config field
//
// A trailing managed/unmanaged keyword always means arming; four tokens always
// means a config field. Bulk config-field writes are unsupported, so "all" in
// the name slot is only valid with an arm keyword.
func ParseSetSiteArgs(args []string) (*ParsedSetSiteArgs, error) {
	const grammar = "set site <site> <ap|switch|gateway> <name> {<key-path> <value> | managed|unmanaged}" +
		" | <ap|switch|gateway> all managed|unmanaged | all managed|unmanaged"

	if len(args) == 0 {
		return nil, fmt.Errorf("set site requires a device type or 'all'\n  %s", grammar)
	}

	// all managed|unmanaged — every device at the site.
	if args[0] == "all" {
		if len(args) != 2 {
			return nil, fmt.Errorf("'all' takes managed or unmanaged\n  %s", grammar)
		}
		action, ok := armAction(args[1])
		if !ok {
			return nil, fmt.Errorf("expected managed or unmanaged after 'all', got %q", args[1])
		}
		return &ParsedSetSiteArgs{Action: action, Scope: ScopeAllTypes}, nil
	}

	if !IsDeviceType(args[0]) {
		return nil, fmt.Errorf("unknown device type %q (expected ap, switch, gateway, or 'all')", args[0])
	}
	deviceType := NormalizeDeviceType(args[0])
	rest := args[1:]

	if len(rest) == 0 {
		return nil, fmt.Errorf("set site %s requires a device name or 'all'\n  %s", args[0], grammar)
	}

	// <device-type> all managed|unmanaged — every device of this type.
	if rest[0] == "all" {
		if len(rest) != 2 {
			return nil, fmt.Errorf("'all' takes managed or unmanaged (bulk config writes are not supported)\n  %s", grammar)
		}
		action, ok := armAction(rest[1])
		if !ok {
			return nil, fmt.Errorf("expected managed or unmanaged after 'all', got %q", rest[1])
		}
		return &ParsedSetSiteArgs{Action: action, Scope: ScopeAllOfType, DeviceType: deviceType}, nil
	}

	name := StripQuotes(rest[0])
	tail := rest[1:]

	switch len(tail) {
	case 1:
		action, ok := armAction(tail[0])
		if !ok {
			return nil, fmt.Errorf("expected managed, unmanaged, or a key-path and value after %q\n  %s", name, grammar)
		}
		return &ParsedSetSiteArgs{Action: action, Scope: ScopeSingle, DeviceType: deviceType, Name: name}, nil
	case 2:
		return &ParsedSetSiteArgs{
			Action:     SetActionConfigField,
			Scope:      ScopeSingle,
			DeviceType: deviceType,
			Name:       name,
			KeyPath:    tail[0],
			RawValue:   tail[1],
		}, nil
	default:
		return nil, fmt.Errorf("expected managed|unmanaged or <key-path> <value> after %q, got %d extra token(s)\n  %s",
			name, len(tail), grammar)
	}
}

// StripQuotes removes surrounding double quotes from a value.
// The shell normally handles quote stripping, but this provides a defensive
// fallback for values forwarded from scripts that pass quotes through verbatim.
func StripQuotes(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

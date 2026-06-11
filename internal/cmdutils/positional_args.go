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

// ValidateShowAPArgs validates arguments for the show api ap command
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

// StripQuotes removes surrounding double quotes from a value.
// The shell normally handles quote stripping, but this provides a defensive
// fallback for values forwarded from scripts that pass quotes through verbatim.
func StripQuotes(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

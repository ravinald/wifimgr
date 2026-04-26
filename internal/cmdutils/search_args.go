package cmdutils

import (
	"fmt"
	"strings"
)

// SearchArgs holds parsed positional arguments for `wifimgr search` subcommands.
type SearchArgs struct {
	SearchText string
	SiteID     string // site name or site ID; resolved later per-API
	Force      bool
	Format     string
	NoResolve  bool
	Detail     bool // render per-client detail columns; filter out offline clients
	Extensive  bool // like Detail, but include offline clients
}

// ParseSearchArgs parses positional arguments for search commands.
// Expected shape: [<search-text>] [site <site-name-or-id>] [force] [detail|extensive] [json|csv] [no-resolve]
// When the first argument is the "site" keyword, no search text is required —
// the caller should list all clients scoped to the given site.
func ParseSearchArgs(args []string) SearchArgs {
	result := SearchArgs{Format: "table"}

	for i := 0; i < len(args); i++ {
		arg := strings.ToLower(args[i])
		switch arg {
		case "site":
			if i+1 < len(args) {
				result.SiteID = StripQuotes(args[i+1])
				i++
			}
		case "force":
			result.Force = true
		case "json", "csv":
			result.Format = arg
		case "no-resolve":
			result.NoResolve = true
		case "detail":
			result.Detail = true
		case "extensive":
			result.Extensive = true
		default:
			if result.SearchText == "" {
				result.SearchText = args[i]
			}
		}
	}

	return result
}

// ValidateSearchArgs ensures the parsed args have enough information to run a
// meaningful search. At minimum either a search term or a site must be given.
func ValidateSearchArgs(parsed SearchArgs) error {
	if parsed.SearchText == "" && parsed.SiteID == "" {
		return fmt.Errorf("specify a search term or a site to list all clients (e.g. `search wireless laptop` or `search wireless site \"US-LAB-01\"`)")
	}
	return nil
}

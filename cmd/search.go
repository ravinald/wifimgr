/*
Copyright © 2025 Ravi Pina <ravi@pina.org>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// searchCmd represents the search command
var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search for connected devices on the network",
	Long: `Search for devices connected to the network infrastructure.

This command replicates the web GUI's "find connected devices" feature,
allowing you to locate client devices by hostname or MAC address.

Use 'wifimgr search <subcommand> --help' for detailed information about each search type.`,
	Example: `  # Search for wireless devices
  wifimgr search wireless laptop-john

  # Search for wired devices
  wifimgr search wired aa:bb:cc:dd:ee:ff

  # List every client on a site
  wifimgr search wireless site "MX - Av. Ejercito Nacional Mexicano 904"

  # Search within a specific site
  wifimgr search wireless device-name site US-LAB-01`,
}

// searchArgs holds parsed search command arguments.
type searchArgs struct {
	searchText string
	siteID     string // site name or site ID; resolved later per-API
	force      bool
	format     string
	noResolve  bool
	detail     bool // render per-client detail columns; filter out offline clients
	extensive  bool // like detail, but include offline clients
}

// parseSearchArgs parses positional arguments for search commands.
// Expected format: [<search-text>] [site <site-name-or-id>] [force] [detail|extensive] [json|csv] [no-resolve]
// When the first argument is the "site" keyword, no search text is required —
// the caller should list all clients scoped to the given site.
func parseSearchArgs(args []string) searchArgs {
	result := searchArgs{
		format: "table",
	}

	for i := 0; i < len(args); i++ {
		arg := strings.ToLower(args[i])
		switch arg {
		case "site":
			if i+1 < len(args) {
				result.siteID = searchStripQuotes(args[i+1])
				i++
			}
		case "force":
			result.force = true
		case "json", "csv":
			result.format = arg
		case "no-resolve":
			result.noResolve = true
		case "detail":
			result.detail = true
		case "extensive":
			result.extensive = true
		default:
			if result.searchText == "" {
				result.searchText = args[i]
			}
		}
	}

	return result
}

// validateSearchArgs ensures the parsed args have enough information to run a
// meaningful search. At minimum either a search term or a site must be given.
func validateSearchArgs(parsed searchArgs) error {
	if parsed.searchText == "" && parsed.siteID == "" {
		return fmt.Errorf("specify a search term or a site to list all clients (e.g. `search wireless laptop` or `search wireless site \"US-LAB-01\"`)")
	}
	return nil
}

// searchStripQuotes removes surrounding double quotes as a defensive fallback;
// the shell normally handles quoting, but multi-word site names occasionally
// arrive with stray quotes when forwarded from scripts.
func searchStripQuotes(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

func init() {
	rootCmd.AddCommand(searchCmd)
}

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

// wirelessCmd represents the wireless command
var wirelessCmd = &cobra.Command{
	Use:   "wireless [<search-text>] [site <site-name-or-id>] [force] [detail|extensive] [json|csv] [no-resolve]",
	Short: "Search wireless devices",
	Long: `Search for wireless devices by name, MAC address, or other criteria.

When multiple APIs are configured:
  - Without target: Searches across all APIs that support wireless search
  - With target: Searches only the specified API

Omit the search text and pass a site to list every wireless client on that site.
The site value may be either a site name or the vendor's site/network ID.

Arguments:
  search-text   Optional. Text to search for; omit when using "site" alone to list all clients.
  site          Optional. Keyword followed by site name or ID to scope the search.
  force         Optional. Bypass confirmation prompts for expensive searches
  detail        Optional. Render Band and State columns for currently-connected clients only.
                Sourced from the local client-detail cache (Band) and live from the API (State).
                Populate the cache with 'wifimgr refresh client site <name>'.
  extensive     Optional. Like detail, but also includes offline / disconnected clients. Useful
                for historical or troubleshooting views.
  json|csv      Optional. Output format (default: table)
  no-resolve    Optional. Disable field ID to name resolution

Examples:
  wifimgr search wireless laptop                                              # Search for "laptop" in all sites
  wifimgr search wireless laptop site US-LAB-01                               # Search in a specific site
  wifimgr search wireless site "MX - Av. Ejercito Nacional Mexicano 904"     # List every client on that site
  wifimgr search wireless site US-LAB-01 detail                               # Online clients with Band + State columns
  wifimgr search wireless site US-LAB-01 extensive                            # Online + offline with Band + State columns
  wifimgr search wireless site L_3732358191183298569                          # Same, by vendor site ID
  wifimgr search wireless laptop force                                        # Skip confirmation for expensive search
  wifimgr search wireless aa:bb:cc:dd:ee:ff                                   # Search by MAC address
  wifimgr search wireless john json                                           # Search and show as JSON
  wifimgr search wireless laptop target mist-prod                             # Search only in mist-prod`,
	Args: func(cmd *cobra.Command, args []string) error {
		// Allow "help" as a special keyword
		for _, arg := range args {
			if strings.ToLower(arg) == "help" {
				return nil
			}
		}
		if len(args) > 7 {
			return fmt.Errorf("accepts up to 7 arg(s), received %d", len(args))
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check for help keyword in positional arguments
		for _, arg := range args {
			if strings.ToLower(arg) == "help" {
				return cmd.Help()
			}
		}
		parsed := parseSearchArgs(args)
		if err := validateSearchArgs(parsed); err != nil {
			return err
		}
		return searchWirelessMultiVendor(globalContext, parsed.searchText, parsed.siteID, parsed.format, parsed.force, parsed.noResolve, parsed.detail, parsed.extensive)
	},
}

func init() {
	searchCmd.AddCommand(wirelessCmd)
}

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

	"github.com/spf13/cobra"

	"github.com/ravinald/wifimgr/internal/cmdutils"
)

// wiredCmd represents the wired command
var wiredCmd = &cobra.Command{
	Use:   "wired [<search-text>] [site <site-name-or-id>] [force] [detail|extensive] [json|csv] [no-resolve]",
	Short: "Search wired devices",
	Long: `Search for wired devices by name, MAC address, or other criteria.

When multiple APIs are configured:
  - Without target: Searches across all APIs that support wired search
  - With target: Searches only the specified API

Omit the search text and pass a site to list every wired client on that site.
The site value may be either a site name or the vendor's site/network ID.

Arguments:
  search-text   Optional. Text to search for; omit when using "site" alone to list all clients.
  site          Optional. Keyword followed by site name or ID to scope the search.
  force         Optional. Bypass confirmation prompts for expensive searches
  json|csv      Optional. Output format (default: table)
  no-resolve    Optional. Disable field ID to name resolution

Examples:
  wifimgr search wired laptop                                              # Search for "laptop" in all sites
  wifimgr search wired laptop site US-LAB-01                               # Search in a specific site
  wifimgr search wired site "MX - Av. Ejercito Nacional Mexicano 904"     # List every client on that site
  wifimgr search wired site L_3732358191183298569                          # Same, by vendor site ID
  wifimgr search wired laptop force                                        # Skip confirmation for expensive search
  wifimgr search wired aa:bb:cc:dd:ee:ff                                   # Search by MAC address
  wifimgr search wired john json                                           # Search and show as JSON
  wifimgr search wired laptop target mist-prod                             # Search only in mist-prod`,
	Args: func(cmd *cobra.Command, args []string) error {
		if cmdutils.ContainsHelp(args) {
			return nil
		}
		if len(args) > 7 {
			return fmt.Errorf("accepts up to 7 arg(s), received %d", len(args))
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if cmdutils.ContainsHelp(args) {
			return cmd.Help()
		}
		parsed := cmdutils.ParseSearchArgs(args)
		if err := cmdutils.ValidateSearchArgs(parsed); err != nil {
			return err
		}
		return searchWiredMultiVendor(globalContext, parsed.SearchText, parsed.SiteID, parsed.Format, parsed.Force, parsed.NoResolve, parsed.Detail, parsed.Extensive)
	},
}

func init() {
	searchCmd.AddCommand(wiredCmd)
}

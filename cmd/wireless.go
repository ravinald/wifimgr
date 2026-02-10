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
	Use:   "wireless <search-text> [site <site-name>] [force] [json|csv] [no-resolve]",
	Short: "Search wireless devices",
	Long: `Search for wireless devices by name, MAC address, or other criteria.

When multiple APIs are configured:
  - Without target: Searches across all APIs that support wireless search
  - With target: Searches only the specified API

Arguments:
  search-text   Required. Text to search for
  site          Optional. Keyword followed by site name to scope search
  force         Optional. Bypass confirmation prompts for expensive searches
  json|csv      Optional. Output format (default: table)
  no-resolve    Optional. Disable field ID to name resolution

Examples:
  wifimgr search wireless laptop                    # Search for "laptop" in all sites
  wifimgr search wireless laptop site US-LAB-01    # Search in specific site
  wifimgr search wireless laptop force             # Skip confirmation for expensive search
  wifimgr search wireless aa:bb:cc:dd:ee:ff        # Search by MAC address
  wifimgr search wireless john json                # Search and show as JSON
  wifimgr search wireless laptop target mist-prod   # Search only in mist-prod`,
	Args: func(cmd *cobra.Command, args []string) error {
		// Allow "help" as a special keyword
		for _, arg := range args {
			if strings.ToLower(arg) == "help" {
				return nil
			}
		}
		// Otherwise require 1-6 args
		if len(args) < 1 || len(args) > 6 {
			return fmt.Errorf("accepts between 1 and 6 arg(s), received %d", len(args))
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
		return searchWirelessMultiVendor(globalContext, parsed.searchText, parsed.siteID, parsed.format, parsed.force, parsed.noResolve)
	},
}

func init() {
	searchCmd.AddCommand(wirelessCmd)
}

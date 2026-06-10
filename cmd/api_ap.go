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
	"github.com/spf13/cobra"

	"github.com/ravinald/wifimgr/internal/cmdutils"
)

// apiApCmd represents the "show ap" command.
var apiApCmd = &cobra.Command{
	Use:   "ap [name-or-mac] [site site-name] [target api-label] [all] [detail|extensive] [format json|csv] [no-resolve]",
	Short: "Show access points wifimgr manages (add 'all' for every AP the API knows)",
	Long: `Show access point data from the local API cache.

By default this lists only the APs armed in your per-site inventory — the ones
wifimgr manages — with a '*' marker when local intent has drifted from the
cached config. Add 'all' to widen the view to every AP the API knows about
(managed ones are highlighted).

Arguments:
  name-or-mac  - Optional AP name or MAC address filter
  site         - Keyword followed by site name for filtering
  target       - Keyword followed by API label to target specific API
  all          - Show every AP the API has, not just managed
  detail       - Reserved verbosity level (field set unchanged for now)
  extensive    - Show all cache fields
  format       - Output format: "json" or "csv" (default: table)
  no-resolve   - Disable field ID to name resolution

Examples:
  wifimgr show ap                          - Managed APs
  wifimgr show ap all                      - Every AP the API knows
  wifimgr show ap site US-LAB-01           - Managed APs in a site
  wifimgr show ap AP-NAME                  - A managed AP by name
  wifimgr show ap format json extensive    - Managed APs, all fields, JSON
  wifimgr show ap target mist-prod         - Managed APs from mist-prod only`,
	Args: cmdutils.ValidateShowAPArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if cmdutils.ContainsHelp(args) {
			return cmd.Help()
		}

		parsed, err := cmdutils.ParseShowArgs(args)
		if err != nil {
			return err
		}

		SetAPITarget(parsed.Target)

		return showDevicesMultiVendor(globalContext, "ap", parsed)
	},
}

// containsIgnoreCase checks if s contains substr (case-insensitive).
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(substr) == 0 ||
		(len(s) > 0 && containsIgnoreCase(s[1:], substr)) ||
		(len(s) >= len(substr) && equalFoldPrefix(s, substr) && containsIgnoreCase(s[len(substr):], "")))
}

func equalFoldPrefix(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		c1, c2 := s[i], prefix[i]
		if c1 >= 'A' && c1 <= 'Z' {
			c1 += 'a' - 'A'
		}
		if c2 >= 'A' && c2 <= 'Z' {
			c2 += 'a' - 'A'
		}
		if c1 != c2 {
			return false
		}
	}
	return true
}

func init() {
	showCmd.AddCommand(apiApCmd)
}

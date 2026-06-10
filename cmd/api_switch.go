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

// apiSwitchCmd represents the "show api switch" command
var apiSwitchCmd = &cobra.Command{
	Use:   "switch [name-or-mac] [site site-name] [target api-label] [all] [detail|extensive] [format json|csv] [no-resolve]",
	Short: "Show switches wifimgr manages (add 'all' for every switch the API knows)",
	Long: `Show switch data from the local API cache.

By default this lists only the switches armed in your per-site inventory, with a
'*' marker when local intent has drifted from the cached config. Add 'all' to
widen the view to every switch the API knows about (managed ones highlighted).

Arguments:
  name-or-mac  - Optional switch name or MAC address filter
  site         - Keyword followed by site name for filtering
  target       - Keyword followed by API label to target specific API
  all          - Show every switch the API has, not just managed
  detail       - Reserved verbosity level (field set unchanged for now)
  extensive    - Show all cache fields
  format       - Output format: "json" or "csv" (default: table)
  no-resolve   - Disable field ID to name resolution

Examples:
  wifimgr show switch                          - Managed switches
  wifimgr show switch all                      - Every switch the API knows
  wifimgr show switch site US-LAB-01           - Managed switches in a site
  wifimgr show switch SW-NAME                  - A managed switch by name
  wifimgr show switch format json extensive    - Managed switches, all fields, JSON
  wifimgr show switch target mist-prod         - Managed switches from mist-prod only`,
	Args: cmdutils.ValidateShowAPArgs, // Reuse same validation
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check for help keyword in positional arguments
		if cmdutils.ContainsHelp(args) {
			return cmd.Help()
		}

		// Parse positional arguments using the utility
		parsed, err := cmdutils.ParseShowArgs(args)
		if err != nil {
			return err
		}

		// Set API target from positional argument
		SetAPITarget(parsed.Target)

		return showDevicesMultiVendor(globalContext, "switch", parsed)
	},
}

func init() {
	showCmd.AddCommand(apiSwitchCmd)
}

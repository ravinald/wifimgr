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

// apiGatewayCmd represents the "show gateway" command
var apiGatewayCmd = &cobra.Command{
	Use:   "gateway [name-or-mac] [site site-name] [target api-label] [all] [detail|extensive] [format json|csv] [no-resolve]",
	Short: "Show gateways wifimgr manages (add 'all' for every gateway the API knows)",
	Long: `Show gateway data from the local API cache.

By default this lists only the gateways armed in your per-site inventory, with a
'*' marker when local intent has drifted from the cached config. Add 'all' to
widen the view to every gateway the API knows about (managed ones highlighted).

Arguments:
  name-or-mac  - Optional gateway name or MAC address filter
  site         - Keyword followed by site name for filtering
  target       - Keyword followed by API label to target specific API
  all          - Show every gateway the API has, not just managed
  detail       - Reserved verbosity level (field set unchanged for now)
  extensive    - Show all cache fields
  format       - Output format: "json" or "csv" (default: table)
  no-resolve   - Disable field ID to name resolution

Examples:
  wifimgr show gateway                          - Managed gateways
  wifimgr show gateway all                      - Every gateway the API knows
  wifimgr show gateway site US-LAB-01           - Managed gateways in a site
  wifimgr show gateway GW-NAME                  - A managed gateway by name
  wifimgr show gateway format json extensive    - Managed gateways, all fields, JSON
  wifimgr show gateway target mist-prod         - Managed gateways from mist-prod only`,
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

		return showDevicesMultiVendor(globalContext, "gateway", parsed)
	},
}

func init() {
	showCmd.AddCommand(apiGatewayCmd)
}

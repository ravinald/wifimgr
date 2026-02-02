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

// apiGatewayCmd represents the "show api gateway" command
var apiGatewayCmd = &cobra.Command{
	Use:   "gateway [name-or-mac] [site site-name] [target api-label] [format] [all] [no-resolve]",
	Short: "Show gateway information from API cache",
	Long: `Show gateway information retrieved from the local API cache.

This command displays gateway device data from the local cache with connection status indicators.

When multiple APIs are configured:
  - Without target: Aggregates gateways from all APIs
  - With target: Shows gateways from the specified API only

Arguments:
  name-or-mac  - Optional gateway name or MAC address filter
  site         - Keyword followed by site name for filtering
  target       - Keyword followed by API label to target specific API
  format       - Output format: "json" or "csv" (default: table)
  all          - Show all fields (json format only)
  no-resolve   - Disable field ID to name resolution

Examples:
  wifimgr show api gateway                          - Show all gateways
  wifimgr show api gateway site US-LAB-01           - Show gateways for specific site
  wifimgr show api gateway GW-NAME                  - Show specific gateway by name
  wifimgr show api gateway 00:11:22:33:44:55        - Show specific gateway by MAC address
  wifimgr show api gateway json                     - Show all gateways in JSON format
  wifimgr show api gateway GW-NAME json all         - Show all fields for gateway in JSON
  wifimgr show api gateway json no-resolve          - Show JSON with raw IDs
  wifimgr show api gateway target mist-prod         - Show gateways from mist-prod only`,
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
	apiCmd.AddCommand(apiGatewayCmd)
}

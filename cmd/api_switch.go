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
	Use:   "switch [name-or-mac] [site site-name] [target api-label] [format] [all] [no-resolve]",
	Short: "Show switch information from API cache",
	Long: `Show switch information retrieved from the local API cache.

This command displays switch device data from the local cache with connection status indicators.

When multiple APIs are configured:
  - Without target: Aggregates switches from all APIs
  - With target: Shows switches from the specified API only

Arguments:
  name-or-mac  - Optional switch name or MAC address filter
  site         - Keyword followed by site name for filtering
  target       - Keyword followed by API label to target specific API
  format       - Output format: "json" or "csv" (default: table)
  all          - Show all fields (json format only)
  no-resolve   - Disable field ID to name resolution

Examples:
  wifimgr show api switch                          - Show all switches
  wifimgr show api switch site US-LAB-01           - Show switches for specific site
  wifimgr show api switch SW-NAME                  - Show specific switch by name
  wifimgr show api switch 00:11:22:33:44:55        - Show specific switch by MAC address
  wifimgr show api switch json                     - Show all switches in JSON format
  wifimgr show api switch SW-NAME json all         - Show all fields for switch in JSON
  wifimgr show api switch json no-resolve          - Show JSON with raw IDs
  wifimgr show api switch target mist-prod         - Show switches from mist-prod only`,
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
	apiCmd.AddCommand(apiSwitchCmd)
}

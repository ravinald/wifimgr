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

// showInventoryCmd represents the "show inventory" command
var showInventoryCmd = &cobra.Command{
	Use:   "inventory [device-type] [site site-name] [target api-label] [format] [all] [no-resolve]",
	Short: "Show inventory information",
	Long: `Show inventory information for network devices.

When an inventory.json file is configured, only devices listed there are shown.
Without an inventory file, all devices from the API cache are displayed.

When multiple APIs are configured:
  - Without target: Aggregates inventory from all APIs
  - With target: Shows inventory from the specified API only

Arguments:
  device-type  - Optional device type filter: ap, switch, or gateway
  site         - Keyword followed by site name for filtering
  target       - Keyword followed by API label to target specific API
  format       - Output format: "json" or "csv" (default: table)
  all          - Show all fields (json format only)
  no-resolve   - Disable field ID to name resolution

Examples:
  wifimgr show inventory                     - Show all inventory items
  wifimgr show inventory ap                  - Show only AP inventory
  wifimgr show inventory site US-LAB-01      - Show inventory for specific site
  wifimgr show inventory json                - Show inventory in JSON format
  wifimgr show inventory json all            - Show all fields in JSON
  wifimgr show inventory json no-resolve     - Show JSON with raw IDs
  wifimgr show inventory target mist-prod    - Show inventory from mist-prod only`,
	Args: cmdutils.ValidateInventoryArgs,
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

		// For inventory command, if there's a filter but no device type,
		// check if the filter is actually a device type
		if parsed.Filter != "" && parsed.DeviceType == "" {
			normalized := cmdutils.NormalizeDeviceType(parsed.Filter)
			if normalized != parsed.Filter {
				// It was a device type
				parsed.DeviceType = normalized
				parsed.Filter = ""
			}
		}

		return showInventoryMultiVendor(globalContext, parsed.DeviceType, parsed)
	},
}

func init() {
	showCmd.AddCommand(showInventoryCmd)
}

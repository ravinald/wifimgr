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

	"github.com/ravinald/wifimgr/cmd/apply"
)

// applyDeviceProfileCmd represents the "apply device-profile" command
var applyDeviceProfileCmd = &cobra.Command{
	Use:   "device-profile <site-name> [all | <device>] [diff] [force]",
	Short: "Apply device profile configurations to devices",
	Long: `Apply device profile configurations to access points in a site.

This command manages the assignment of device profiles to APs based on the
intent configuration. It will:
- Assign device profiles to APs that have a profile configured but not assigned
- Unassign device profiles from APs that no longer have a profile configured

Arguments:
  site-name - The name of the site to apply profiles to
  all       - Apply profiles to all configured devices (default)
  device    - Apply profile to a specific device by name
  diff      - Show changes without applying them (optional)

Examples:
  wifimgr apply device-profile US-WDFW-SVP5              - Apply to all configured devices
  wifimgr apply device-profile US-WDFW-SVP5 all          - Apply to all configured devices
  wifimgr apply device-profile US-WDFW-SVP5 "AP-1"       - Apply to specific device
  wifimgr apply device-profile US-WDFW-SVP5 all diff     - Show changes without applying`,
	Args: func(cmd *cobra.Command, args []string) error {
		// Allow "help" as a special keyword
		for _, arg := range args {
			if strings.ToLower(arg) == "help" {
				return nil
			}
		}
		if len(args) < 1 || len(args) > 3 {
			return fmt.Errorf("accepts between 1 and 3 arg(s), received %d", len(args))
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

		siteName := args[0]
		deviceFilter := "all"
		diffMode := false
		force := false

		// Parse remaining arguments
		for i := 1; i < len(args); i++ {
			switch strings.ToLower(args[i]) {
			case "diff":
				diffMode = true
			case "force":
				force = true
			default:
				deviceFilter = args[i]
			}
		}

		// Create args for the handler
		legacyArgs := []string{siteName, "device-profile", deviceFilter}
		if diffMode {
			legacyArgs = append(legacyArgs, "diff")
		}

		return apply.HandleCommand(globalContext, globalClient, globalConfig, legacyArgs, "", force)
	},
}

func init() {
	// Add subcommand to apply
	applyCmd.AddCommand(applyDeviceProfileCmd)

	// Note: 'force' is now a positional argument, not a flag
}

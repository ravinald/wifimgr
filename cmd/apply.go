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

	"github.com/ravinald/wifimgr/cmd/apply"
	"github.com/ravinald/wifimgr/internal/cmdutils"
)

// applyCmd represents the apply command
var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply configuration changes to network devices",
	Long: `Apply configuration changes to network devices from intent files.

This command provides various operations for managing device configurations,
including applying configurations, managing backups, and rollback capabilities.

Use 'wifimgr apply <subcommand> --help' for detailed information about each operation.`,
	Example: `  # Apply AP configuration
  wifimgr apply ap US-LAB-01

  # Show changes without applying
  wifimgr apply ap US-LAB-01 diff

  # Apply all device types
  wifimgr apply all US-LAB-01

  # Rollback to previous configuration
  wifimgr apply rollback US-LAB-01

  # List available backups
  wifimgr apply list-backups US-LAB-01`,
	// Handle legacy positional arguments for backward compatibility
	RunE: func(cmd *cobra.Command, args []string) error {
		// If called with legacy format: apply <site-name> <operation>
		if len(args) >= 2 {
			parsed, err := cmdutils.ParseApplyArgs(args)
			if err != nil {
				return fmt.Errorf("invalid apply arguments: %w", err)
			}

			// Route to appropriate subcommand based on operation
			switch parsed.Operation {
			case "rollback", "list-backups", "cleanup-backups", "validate-backup":
				// These are backup operations, pass through
				return apply.HandleCommand(globalContext, globalClient, globalConfig, args, "", false)
			default:
				// Device type operation, pass through
				return apply.HandleCommand(globalContext, globalClient, globalConfig, args, "", false)
			}
		}

		// No args provided, show help
		return cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(applyCmd)
}

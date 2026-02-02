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
	"github.com/spf13/viper"

	"github.com/ravinald/wifimgr/cmd/apply"
)

// applyRollbackCmd represents the "apply rollback" command
var applyRollbackCmd = &cobra.Command{
	Use:   "rollback <site-name> [backup-index]",
	Short: "Restore intent config from a backup (file-based, does NOT apply to API)",
	Long: `Restore the intent configuration file from a backup.

This is a FILE-BASED operation that does NOT send anything to the API.
The current config becomes the new .0 backup, and the selected backup
becomes the current intent config.

After rollback, you can:
  - Review the restored config
  - Edit if needed
  - Use 'apply diff' to see what would change
  - Use 'apply' to send changes to the API

Arguments:
  backup-index - The backup index to restore (default: 0 = most recent)

Examples:
  wifimgr apply rollback US-SFO-LAB      - Restore from most recent backup (.0)
  wifimgr apply rollback US-SFO-LAB 1    - Restore from second most recent (.1)
  wifimgr apply rollback US-SFO-LAB 2    - Restore from third most recent (.2)`,
	Args: func(cmd *cobra.Command, args []string) error {
		for _, arg := range args {
			if strings.ToLower(arg) == "help" {
				return nil
			}
		}
		if len(args) < 1 || len(args) > 2 {
			return fmt.Errorf("accepts between 1 and 2 arg(s), received %d", len(args))
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		for _, arg := range args {
			if strings.ToLower(arg) == "help" {
				return cmd.Help()
			}
		}
		// Create legacy args format
		legacyArgs := []string{args[0], "rollback"}
		if len(args) > 1 {
			legacyArgs = append(legacyArgs, args[1])
		}

		return apply.HandleCommand(globalContext, globalClient, globalConfig, legacyArgs, "", false)
	},
}

// applyListBackupsCmd represents the "apply list-backups" command
var applyListBackupsCmd = &cobra.Command{
	Use:   "list-backups <site-name>",
	Short: "List available configuration backups",
	Long: `List all available configuration backups for a specific site.

Shows backup timestamp, device count, and filename for each backup.

Example:
  wifimgr apply list-backups US-SFO-LAB`,
	Args: func(cmd *cobra.Command, args []string) error {
		for _, arg := range args {
			if strings.ToLower(arg) == "help" {
				return nil
			}
		}
		if len(args) != 1 {
			return fmt.Errorf("accepts 1 arg(s), received %d", len(args))
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		for _, arg := range args {
			if strings.ToLower(arg) == "help" {
				return cmd.Help()
			}
		}
		legacyArgs := []string{args[0], "list-backups"}
		return apply.HandleCommand(globalContext, globalClient, globalConfig, legacyArgs, "", false)
	},
}

// applyCleanupBackupsCmd represents the "apply cleanup-backups" command
var applyCleanupBackupsCmd = &cobra.Command{
	Use:   "cleanup-backups",
	Short: "Remove old configuration backups",
	Long: `Clean up configuration backups older than a specified number of days.

Default retention period is configured in backup.retention_days (default: 30).

Examples:
  wifimgr apply cleanup-backups           - Remove backups older than configured retention
  wifimgr apply cleanup-backups --days 7  - Remove backups older than 7 days`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get default from config
		defaultDays := viper.GetInt("backup.retention_days")
		if defaultDays == 0 {
			defaultDays = 30
		}

		// Check if --days was explicitly provided
		days := defaultDays
		if cmd.Flags().Changed("days") {
			days, _ = cmd.Flags().GetInt("days")
		}

		// Create legacy args - always pass --days to let apply.go handle it
		legacyArgs := []string{"placeholder", "cleanup-backups", "--days", fmt.Sprintf("%d", days)}

		return apply.HandleCommand(globalContext, globalClient, globalConfig, legacyArgs, "", false)
	},
}

// applyValidateBackupCmd represents the "apply validate-backup" command
var applyValidateBackupCmd = &cobra.Command{
	Use:   "validate-backup <backup-file>",
	Short: "Validate a configuration backup file",
	Long: `Validate the integrity and structure of a configuration backup file.

Checks that the backup file is valid and contains all required information.

Example:
  wifimgr apply validate-backup config_backup_US-SFO-LAB_1710486400.json`,
	Args: func(cmd *cobra.Command, args []string) error {
		for _, arg := range args {
			if strings.ToLower(arg) == "help" {
				return nil
			}
		}
		if len(args) != 1 {
			return fmt.Errorf("accepts 1 arg(s), received %d", len(args))
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		for _, arg := range args {
			if strings.ToLower(arg) == "help" {
				return cmd.Help()
			}
		}
		legacyArgs := []string{"placeholder", "validate-backup", args[0]} // First arg is ignored
		return apply.HandleCommand(globalContext, globalClient, globalConfig, legacyArgs, "", false)
	},
}

func init() {
	// Add backup subcommands to apply
	applyCmd.AddCommand(applyRollbackCmd)
	applyCmd.AddCommand(applyListBackupsCmd)
	applyCmd.AddCommand(applyCleanupBackupsCmd)
	applyCmd.AddCommand(applyValidateBackupCmd)

	// Add flags
	applyCleanupBackupsCmd.Flags().Int("days", 30, "Remove backups older than this many days")
}

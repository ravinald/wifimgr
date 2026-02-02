package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ravinald/wifimgr/cmd/backup"
	"github.com/ravinald/wifimgr/internal/cmdutils"
)

// backupCmd represents the backup command
var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Manage configuration backups",
	Long: `Manage configuration backups including listing and restoring.

Examples:
  # List all backups
  wifimgr backup list all
  wifimgr backup list

  # List backups for a specific site
  wifimgr backup list US-SFO-LAB

  # Restore a backup
  wifimgr backup restore US-SFO-LAB 0`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check for help keyword in positional arguments
		if cmdutils.ContainsHelp(args) {
			return cmd.Help()
		}

		// Ensure config is loaded
		if globalConfig == nil {
			return fmt.Errorf("configuration not loaded")
		}

		// Handle the backup command
		return backup.HandleCommand(globalContext, globalClient, globalConfig, args)
	},
}

func init() {
	rootCmd.AddCommand(backupCmd)
}

package cmd

import (
	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check API compatibility and system health",
	Long: `Perform various checks on configurations and API compatibility.

Use 'wifimgr check <subcommand> help' for detailed information.`,
	Example: `  # Check if a site config is compatible with the target API
  wifimgr check compatibility ./config/us-lab-01.json`,
}

func init() {
	rootCmd.AddCommand(checkCmd)
}

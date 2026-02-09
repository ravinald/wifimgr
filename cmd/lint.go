package cmd

import (
	"github.com/spf13/cobra"
)

var lintCmd = &cobra.Command{
	Use:   "lint",
	Short: "Lint and validate configurations",
	Long: `Lint and validate site and device configurations for common issues.

The lint command performs static analysis on configuration files to detect:
- Syntax errors
- Schema mismatches
- Invalid field values
- Missing required fields
- Deprecated field usage
- Vendor-specific incompatibilities

Use 'wifimgr lint <subcommand> help' for detailed information.`,
	Example: `  # Lint a site configuration
  wifimgr lint config US-LAB-01`,
}

func init() {
	rootCmd.AddCommand(lintCmd)
}

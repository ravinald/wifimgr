package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/ravinald/wifimgr/internal/cmdutils"
	"github.com/ravinald/wifimgr/internal/schemadefs"
	"github.com/ravinald/wifimgr/internal/symbols"
)

var initSchemasCmd = &cobra.Command{
	Use:   "schemas",
	Short: "Install the JSON schemas to the configured schema directory",
	Long: `Write the schemas embedded in the binary to files.schemas (default: the
XDG data dir). Validation already falls back to the embedded copies, so this is
only needed to inspect or override the shipped schemas on disk — installation is
no longer out-of-band.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if cmdutils.ContainsHelp(args) {
			return nil
		}
		return cobra.NoArgs(cmd, args)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if cmdutils.ContainsHelp(args) {
			return cmd.Help()
		}

		dir := schemasDir()
		if err := os.MkdirAll(dir, 0750); err != nil {
			return fmt.Errorf("init schemas: create %s: %w", dir, err)
		}

		names := schemadefs.Names()
		if len(names) == 0 {
			return fmt.Errorf("init schemas: no embedded schemas found")
		}

		for _, name := range names {
			data, err := schemadefs.Read(name)
			if err != nil {
				return fmt.Errorf("init schemas: read embedded %s: %w", name, err)
			}
			dest := filepath.Join(dir, name)
			if err := os.WriteFile(dest, data, 0600); err != nil {
				return fmt.Errorf("init schemas: write %s: %w", dest, err)
			}
		}

		fmt.Printf("%s Installed %d schemas to %s\n", symbols.SuccessPrefix(), len(names), dir)
		return nil
	},
}

func init() {
	initCmd.AddCommand(initSchemasCmd)
}

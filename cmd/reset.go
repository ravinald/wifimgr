/*
Copyright © 2025 Ravi Pina <ravi@pina.org>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/
package cmd

import (
	"github.com/spf13/cobra"
)

// resetCmd is the parent of all reset operations. Today only `reset ap`
// exists; switches and gateways will land alongside as the underlying APIs
// support them.
var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset (reboot) network devices",
	Long: `Reset network devices via the configured vendor APIs.

Currently supports:
  reset ap <ap-name> [site <site-name>] [force]

Vendor support:
  - Mist:     supported
  - Meraki:   supported
  - Ubiquiti: not supported (Site Manager API is read-only)`,
	Example: `  wifimgr reset ap AP-LAB-01
  wifimgr reset ap AP-LAB-01 site US-LAB-01
  wifimgr reset ap AP-LAB-01 site US-LAB-01 force`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(resetCmd)
}

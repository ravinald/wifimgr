/*
Copyright © 2025 Ravi Pina <ravi@pina.org>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/
package cmd

import (
	"github.com/spf13/cobra"
)

// refreshCmd represents the refresh command group.
var refreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Refresh cached data from API",
	Long: `Refresh cached data from configured API(s) to ensure local cache is up-to-date.

This command fetches the latest data from the cloud platform and updates
the local cache files.

Use 'wifimgr refresh <subcommand> --help' for detailed information about
each refresh operation.`,
	Example: `  # Refresh the device-level cache (sites, inventory, configs, WLANs, ...)
  wifimgr refresh device

  # Refresh everything we know how to cache, including per-client detail
  wifimgr refresh all

  # Populate per-client detail (e.g. Meraki connected band) for one site
  wifimgr refresh client site US-LAB-01`,
}

func init() {
	rootCmd.AddCommand(refreshCmd)
}

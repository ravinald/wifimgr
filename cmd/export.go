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
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export data to external systems",
	Long: `Export wifimgr device data to external systems.

The export command provides integration with external DCIM/IPAM systems:

  netbox - Export device inventory to NetBox

Use 'wifimgr export <subcommand> --help' for detailed information about each export target.`,
	Example: `  # Export all devices to NetBox
  wifimgr export netbox all

  # Export a specific site to NetBox
  wifimgr export netbox site US-LAB-01

  # Dry run (validate without writing)
  wifimgr export netbox all dry-run`,
}

func init() {
	rootCmd.AddCommand(exportCmd)
}

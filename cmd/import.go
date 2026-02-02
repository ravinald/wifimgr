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

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import configuration data from external sources",
	Long: `Import configuration data from external sources into wifimgr.

The import command provides two methods for importing configuration:

  api  - Import from API cache to create local config files
  pdf  - Import AP radio configurations from PDF floor plans

Use 'wifimgr import <subcommand> --help' for detailed information about each import method.`,
	Example: `  # Import site from API cache
  wifimgr import api site US-LAB-01 save

  # Import AP radio configs from PDF
  wifimgr import pdf file floor-plan.pdf site US-LAB-01`,
}

func init() {
	rootCmd.AddCommand(importCmd)
}

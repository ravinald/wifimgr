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

// importAPICmd represents the "import api" command
var importAPICmd = &cobra.Command{
	Use:   "api",
	Short: "Import configuration from API cache",
	Long: `Import configuration from the API cache to create local config files.

This command bootstraps local configuration files from the current API state,
allowing you to establish a baseline for managing your infrastructure as code.`,
	Example: `  # Import full site configuration
  wifimgr import api site US-SFO-LAB

  # Import only AP configs
  wifimgr import api site US-SFO-LAB type ap

  # Compare API with existing config
  wifimgr import api site US-SFO-LAB compare`,
}

func init() {
	importCmd.AddCommand(importAPICmd)
}

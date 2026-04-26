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

	"github.com/ravinald/wifimgr/internal/cmdutils"
)

// apiSiteCmd represents the "show api site" command
var apiSiteCmd = &cobra.Command{
	Use:     "site [site-name] [target api-label] [format json|csv] [all]",
	Aliases: []string{"sites"},
	Short:   "Show site information from API cache",
	Long: `Show site information retrieved from the local API cache.

This command displays site data from the local cache with configuration status indicators.

When multiple APIs are configured:
  - Without target: Aggregates sites from all APIs
  - With target: Shows sites from the specified API only

Arguments:
  site-name  - Optional site name filter
  target     - Keyword followed by API label to target specific API
  format     - Output format: "json" or "csv" (default: table)
  all        - Show all fields (json format only)

Examples:
  wifimgr show api site                      - Show all sites
  wifimgr show api site SITE-NAME            - Show specific site by name
  wifimgr show api site json                 - Show all sites in JSON format
  wifimgr show api site target mist-prod     - Show sites from mist-prod only`,
	Args: cmdutils.ValidateShowAPArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if cmdutils.ContainsHelp(args) {
			return cmd.Help()
		}

		parsed, err := cmdutils.ParseShowArgs(args)
		if err != nil {
			return err
		}

		SetAPITarget(parsed.Target)
		return showSitesMultiVendor(globalContext, parsed)
	},
}

func init() {
	apiCmd.AddCommand(apiSiteCmd)
}

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

// apiSiteCmd represents the "show site" command
var apiSiteCmd = &cobra.Command{
	Use:     "site [site-name] [target api-label] [all] [detail|extensive] [format json|csv]",
	Aliases: []string{"sites"},
	Short:   "Show sites wifimgr manages (add 'all' for every site the API knows)",
	Long: `Show site data from the local API cache.

By default this lists only the sites you manage — those with armed devices in
your per-site inventory. Add 'all' to widen the view to every site the API
knows about.

Arguments:
  site-name  - Optional site name filter
  target     - Keyword followed by API label to target specific API
  all        - Show every site the API has, not just managed
  detail     - Reserved verbosity level (field set unchanged for now)
  extensive  - Show all cache fields
  format     - Output format: "json" or "csv" (default: table)

Examples:
  wifimgr show site                      - Managed sites
  wifimgr show site all                  - Every site the API knows
  wifimgr show site SITE-NAME            - A specific site by name
  wifimgr show site format json          - Managed sites in JSON format
  wifimgr show site target mist-prod     - Managed sites from mist-prod only`,
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
	showCmd.AddCommand(apiSiteCmd)
}

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
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ravinald/wifimgr/cmd/ap"
	"github.com/ravinald/wifimgr/internal/cmdutils"
)

// siteCmd represents the "set ap site" command
var siteCmd = &cobra.Command{
	Use:     "site <site-name>",
	Aliases: []string{"sites"},
	Short:   "Assign access points to a site",
	Long: `Assign access points to a specific site.

Examples:
  wifimgr set ap site US-SFO-LAB              - Interactively assign AP to site
  wifimgr set ap site --file file.txt -s US-SFO-LAB - Assign APs from file`,
	Args: cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check for help keyword in positional arguments
		if cmdutils.ContainsHelp(args) {
			return cmd.Help()
		}

		// Check if file-based assignment is requested
		if assignmentFile != "" {
			site := targetSite
			if site == "" && len(args) > 0 {
				site = args[0]
			}
			if site == "" {
				return fmt.Errorf("site name required for file-based assignment (use -s flag or provide as argument)")
			}
			return ap.AssignBulkAPsFromFile(globalContext, globalClient, globalConfig, assignmentFile, site)
		}

		// Regular assignment
		if len(args) == 0 {
			// Interactive mode - let the handler prompt for site
			return ap.AssignAP(globalContext, globalClient, globalConfig, "")
		} else {
			// Site name provided
			return ap.AssignAP(globalContext, globalClient, globalConfig, args[0])
		}
	},
}

var (
	assignmentFile string
	targetSite     string
)

func init() {
	apCmd.AddCommand(siteCmd)

	// Add flags for file-based AP assignment
	siteCmd.Flags().StringVar(&assignmentFile, "file", "", "File containing AP MACs to assign")
	siteCmd.Flags().StringVarP(&targetSite, "site", "s", "", "Target site for bulk assignment")
}

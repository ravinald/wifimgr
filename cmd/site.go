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
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ravinald/wifimgr/cmd/ap"
	"github.com/ravinald/wifimgr/internal/cmdutils"
)

// siteCmd represents the "set ap site" command
var siteCmd = &cobra.Command{
	Use:     "site [site-name] [file <path>]",
	Aliases: []string{"sites"},
	Short:   "Assign access points to a site",
	Long: `Assign access points to a specific site.

Examples:
  wifimgr set ap site                              - Interactive assignment
  wifimgr set ap site US-SFO-LAB                   - Interactively assign APs to a site
  wifimgr set ap site US-SFO-LAB file aps.txt      - Bulk-assign APs listed in aps.txt`,
	Args: func(cmd *cobra.Command, args []string) error {
		if cmdutils.ContainsHelp(args) {
			return nil
		}
		if len(args) > 3 {
			return fmt.Errorf("accepts at most 3 arg(s), received %d", len(args))
		}
		if len(args) >= 2 {
			if strings.ToLower(args[1]) != "file" {
				return fmt.Errorf("expected `file <path>` after site name, got %q", args[1])
			}
			if len(args) < 3 {
				return fmt.Errorf("`file` keyword requires a path argument")
			}
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if cmdutils.ContainsHelp(args) {
			return cmd.Help()
		}

		var site, file string
		if len(args) >= 1 {
			site = args[0]
		}
		if len(args) == 3 {
			file = args[2]
		}

		if assignmentFile != "" || targetSite != "" {
			fmt.Fprintln(os.Stderr, "DEPRECATED: --file/--site flags will be removed in a future release; use `wifimgr set ap site <site> file <path>`")
			if file == "" {
				file = assignmentFile
			}
			if site == "" {
				site = targetSite
			}
		}

		if file != "" {
			if site == "" {
				return fmt.Errorf("site name required for file-based assignment")
			}
			return ap.AssignBulkAPsFromFile(globalContext, globalClient, globalConfig, file, site)
		}

		return ap.AssignAP(globalContext, globalClient, globalConfig, site)
	},
}

var (
	assignmentFile string
	targetSite     string
)

func init() {
	apCmd.AddCommand(siteCmd)

	// --file and --site/-s kept for one release for backward compatibility;
	// emit a deprecation warning when used. Will be removed in the next minor.
	siteCmd.Flags().StringVar(&assignmentFile, "file", "", "DEPRECATED: use `file <path>` positional")
	siteCmd.Flags().StringVarP(&targetSite, "site", "s", "", "DEPRECATED: use site name as first positional")
}

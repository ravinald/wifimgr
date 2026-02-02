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
	"strings"

	"github.com/spf13/cobra"

	"github.com/ravinald/wifimgr/internal/cmdutils"
)

// apiSiteCmd represents the "show api site" command
var apiSiteCmd = &cobra.Command{
	Use:     "site [site-name] [target api-label] [format] [all]",
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
	Args: validateSiteArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check for help keyword in positional arguments
		for _, arg := range args {
			if strings.ToLower(arg) == "help" {
				return cmd.Help()
			}
		}

		// Parse positional arguments using the utility
		parsed, err := parseSiteArgs(args)
		if err != nil {
			return err
		}

		// Set API target from positional argument
		SetAPITarget(parsed.Target)

		return showSitesMultiVendor(globalContext, parsed)
	},
}

// validateSiteArgs validates arguments for the show api site command
func validateSiteArgs(_ *cobra.Command, args []string) error {
	// Allow "help" as a special keyword
	for _, arg := range args {
		if strings.ToLower(arg) == "help" {
			return nil
		}
	}
	_, err := parseSiteArgs(args)
	return err
}

// parseSiteArgs parses arguments specific to the site command
func parseSiteArgs(args []string) (*cmdutils.ParsedShowArgs, error) {
	result := &cmdutils.ParsedShowArgs{
		Format: "table", // default format
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch arg {
		case "target":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("'target' requires an API label")
			}
			if result.Target != "" {
				return nil, fmt.Errorf("target specified multiple times")
			}
			result.Target = args[i+1]
			i++ // Skip the API label

		case "json", "csv":
			if result.Format != "table" {
				return nil, fmt.Errorf("format specified multiple times")
			}
			result.Format = arg

		case "all":
			result.ShowAll = true

		default:
			// Must be a site name filter
			if result.Filter == "" {
				result.Filter = arg
			} else {
				return nil, fmt.Errorf("unexpected argument: %s", arg)
			}
		}
	}

	// Validate combinations
	if result.ShowAll && result.Format != "json" {
		return nil, fmt.Errorf("'all' is only valid with json format")
	}

	return result, nil
}

func init() {
	apiCmd.AddCommand(apiSiteCmd)
}

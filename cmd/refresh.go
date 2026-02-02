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
)

// refreshCmd represents the refresh command
var refreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Refresh cached data from API",
	Long: `Refresh cached data from configured API(s) to ensure local cache is up-to-date.

This command fetches the latest data from the cloud platform and updates
the local cache files.

Use 'wifimgr refresh <subcommand> --help' for detailed information about each refresh operation.`,
	Example: `  # Refresh sites from all APIs
  wifimgr refresh sites

  # Refresh sites from specific API
  wifimgr refresh sites --api mist-prod`,
}

// refreshSitesCmd represents the refresh sites command
var refreshSitesCmd = &cobra.Command{
	Use:   "sites",
	Short: "Refresh sites info and settings",
	Long: `Refresh both sites information and site settings from the API(s).

When multiple APIs are configured:
  - Without --api: Refreshes sites from all APIs
  - With --api: Refreshes only the specified API

This command will:
1. Refresh all sites information (basic site data)
2. Refresh site settings for each site (individual API calls per site)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMultiVendorSitesRefresh()
	},
}

// runMultiVendorSitesRefresh handles sites refresh for multi-vendor mode.
func runMultiVendorSitesRefresh() error {
	cacheMgr := GetCacheManager()
	if cacheMgr == nil {
		return fmt.Errorf("cache manager not initialized")
	}

	registry := GetAPIRegistry()
	if registry == nil {
		return fmt.Errorf("API registry not initialized")
	}

	// Validate --api flag if provided
	if err := ValidateAPIFlag(); err != nil {
		return err
	}

	targetAPIs := GetTargetAPIs()
	if len(targetAPIs) == 0 {
		return fmt.Errorf("no APIs configured")
	}

	ctx := globalContext

	if apiFlag != "" {
		// Single API refresh
		fmt.Printf("Refreshing sites for %s...\n", apiFlag)
		if err := cacheMgr.RefreshAPI(ctx, apiFlag); err != nil {
			return fmt.Errorf("failed to refresh %s: %w", apiFlag, err)
		}
		fmt.Printf("Successfully refreshed sites for %s\n", apiFlag)
	} else {
		// Parallel refresh of all APIs
		fmt.Printf("Refreshing sites for %d APIs...\n", len(targetAPIs))

		errors := cacheMgr.RefreshAllAPIs(ctx)

		successCount := len(targetAPIs) - len(errors)
		fmt.Printf("\nRefreshed sites from %d/%d APIs successfully\n", successCount, len(targetAPIs))

		if len(errors) > 0 {
			fmt.Println("\nErrors:")
			for apiLabel, err := range errors {
				fmt.Printf("  %s: %v\n", apiLabel, err)
			}
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(refreshCmd)
	refreshCmd.AddCommand(refreshSitesCmd)
}

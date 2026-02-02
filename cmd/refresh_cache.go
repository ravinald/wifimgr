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
)

// refreshCacheCmd represents the refresh cache command
var refreshCacheCmd = &cobra.Command{
	Use:   "cache [api-name]",
	Short: "Refresh cache from API(s)",
	Long: `Refresh the local cache by fetching data from configured API(s).

When multiple APIs are configured:
  - Without api-name or --api: Refreshes all APIs in parallel
  - With api-name or --api: Refreshes only the specified API

Examples:
  wifimgr refresh cache                    # Refresh all APIs
  wifimgr refresh cache mist-prod          # Refresh mist-prod only
  wifimgr refresh cache --api mist-prod    # Refresh mist-prod only (alternative)
  wifimgr refresh cache meraki-corp        # Refresh meraki-corp only`,
	Args: func(cmd *cobra.Command, args []string) error {
		// Allow "help" as a special keyword
		for _, arg := range args {
			if strings.ToLower(arg) == "help" {
				return nil
			}
		}
		if len(args) > 1 {
			return fmt.Errorf("accepts at most 1 arg(s), received %d", len(args))
		}
		return nil
	},
	RunE: runRefreshCache,
}

func init() {
	refreshCmd.AddCommand(refreshCacheCmd)
}

func runRefreshCache(cmd *cobra.Command, args []string) error {
	// Check for help keyword in positional arguments
	for _, arg := range args {
		if strings.ToLower(arg) == "help" {
			return cmd.Help()
		}
	}

	// Check if API name was provided as positional argument
	if len(args) > 0 && apiFlag == "" {
		apiFlag = args[0]
	}
	return runMultiVendorRefresh()
}

// runMultiVendorRefresh handles cache refresh for multi-vendor mode.
func runMultiVendorRefresh() error {
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
		fmt.Printf("Refreshing cache for %s...\n", apiFlag)
		if err := cacheMgr.RefreshAPI(ctx, apiFlag); err != nil {
			return fmt.Errorf("failed to refresh %s: %w", apiFlag, err)
		}
		fmt.Printf("Successfully refreshed %s\n", apiFlag)
	} else {
		// Parallel refresh of all APIs
		fmt.Printf("Refreshing cache for %d APIs...\n", len(targetAPIs))

		errors := cacheMgr.RefreshAllAPIs(ctx)

		successCount := len(targetAPIs) - len(errors)
		fmt.Printf("\nRefreshed %d/%d APIs successfully\n", successCount, len(targetAPIs))

		if len(errors) > 0 {
			fmt.Println("\nErrors:")
			for apiLabel, err := range errors {
				fmt.Printf("  %s: %v\n", apiLabel, err)
			}
		}

		fmt.Println("Rebuilt cross-API index")
	}

	return nil
}

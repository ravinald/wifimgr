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
	"time"

	"github.com/spf13/cobra"

	"github.com/ravinald/wifimgr/internal/cmdutils"
)

// showAPIStatusCmd represents the show api status command
var showAPIStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of configured API connections",
	Long: `Display the status of all configured API connections.

Shows vendor type, organization ID, and available capabilities for each API.

Example:
  wifimgr show api status`,
	RunE: runShowAPIStatus,
}

func init() {
	showAPICmd.AddCommand(showAPIStatusCmd)
}

func runShowAPIStatus(cmd *cobra.Command, args []string) error {
	// Check for help keyword in positional arguments
	if cmdutils.ContainsHelp(args) {
		return cmd.Help()
	}

	registry := GetAPIRegistry()
	if registry == nil || registry.Count() == 0 {
		fmt.Println("No API connections configured")
		fmt.Println("\nTo configure APIs, add them to your config file under the 'api' section:")
		fmt.Println(`  "api": {`)
		fmt.Println(`    "mist-prod": {`)
		fmt.Println(`      "vendor": "mist",`)
		fmt.Println(`      "url": "https://api.mist.com",`)
		fmt.Println(`      "credentials": { "org_id": "...", "api_token": "..." }`)
		fmt.Println(`    }`)
		fmt.Println(`  }`)
		return nil
	}

	statuses := registry.GetStatus()
	cacheMgr := GetCacheManager()

	fmt.Printf("API Connections (%d):\n\n", len(statuses))

	for _, status := range statuses {
		// registry.GetStatus reports static capabilities and assumes healthy; the
		// real signal is the last refresh outcome, persisted in the cache meta.
		var lastSuccess, lastFailure time.Time
		lastErr := status.LastError
		if cacheMgr != nil {
			if cache, err := cacheMgr.GetAPICache(status.Label); err == nil {
				lastSuccess = cache.Meta.LastRefresh
				lastFailure = cache.Meta.LastFailure
				if lastFailure.After(lastSuccess) {
					status.Healthy = false
					if cache.Meta.LastError != "" {
						lastErr = cache.Meta.LastError
					}
				}
			}
		}

		fmt.Printf("  %s:\n", status.Label)
		fmt.Printf("    Vendor:       %s\n", status.Vendor)
		fmt.Printf("    Org ID:       %s\n", status.OrgID)
		fmt.Printf("    Capabilities: %s\n", strings.Join(status.Capabilities, ", "))
		if status.Healthy {
			fmt.Printf("    Status:       healthy\n")
		} else {
			fmt.Printf("    Status:       unhealthy\n")
			if lastErr != "" {
				fmt.Printf("    Last Error:   %s\n", lastErr)
			}
		}
		if !lastSuccess.IsZero() {
			fmt.Printf("    Last Success: %s (%s ago)\n",
				lastSuccess.Format("2006-01-02 15:04:05"), formatDuration(time.Since(lastSuccess)))
		}
		if !lastFailure.IsZero() {
			fmt.Printf("    Last Failure: %s (%s ago)\n",
				lastFailure.Format("2006-01-02 15:04:05"), formatDuration(time.Since(lastFailure)))
		}
		fmt.Println()
	}

	// Local cache data is the on-disk cached state, independent of whether the API
	// is reachable (see per-API Status above). A down API still has usable cache.
	if cacheMgr != nil {
		fmt.Println("Local Cache Data:")
		for _, status := range statuses {
			cacheStatus, err := cacheMgr.VerifyAPICache(status.Label)
			if err != nil {
				fmt.Printf("  %s: error (%v)\n", status.Label, err)
				continue
			}
			line := cacheStatus.String()
			if cache, cerr := cacheMgr.GetAPICache(status.Label); cerr == nil && !cache.Meta.LastRefresh.IsZero() {
				line += fmt.Sprintf(" (%s old)", formatDuration(time.Since(cache.Meta.LastRefresh)))
			}
			fmt.Printf("  %s: %s\n", status.Label, line)
		}
	}

	return nil
}

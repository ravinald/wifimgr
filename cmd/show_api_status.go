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
	apiCmd.AddCommand(showAPIStatusCmd)
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

	fmt.Printf("API Connections (%d):\n\n", len(statuses))

	for _, status := range statuses {
		fmt.Printf("  %s:\n", status.Label)
		fmt.Printf("    Vendor:       %s\n", status.Vendor)
		fmt.Printf("    Org ID:       %s\n", status.OrgID)
		fmt.Printf("    Capabilities: %s\n", strings.Join(status.Capabilities, ", "))
		if status.Healthy {
			fmt.Printf("    Status:       healthy\n")
		} else {
			fmt.Printf("    Status:       unhealthy\n")
			if status.LastError != "" {
				fmt.Printf("    Last Error:   %s\n", status.LastError)
			}
		}
		fmt.Println()
	}

	// Show cache status if available
	cacheMgr := GetCacheManager()
	if cacheMgr != nil {
		fmt.Println("Cache Status:")
		for _, status := range statuses {
			cacheStatus, err := cacheMgr.VerifyAPICache(status.Label)
			if err != nil {
				fmt.Printf("  %s: error (%v)\n", status.Label, err)
			} else {
				fmt.Printf("  %s: %s\n", status.Label, cacheStatus.String())
			}
		}
	}

	return nil
}

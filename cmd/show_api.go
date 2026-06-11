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

// showAPICmd groups the read-only API/vendor introspection views (connection
// status, BSSIDs, profiles, WLANs). These inspect the vendor side rather than
// the managed device set, so they live under an explicit `api` noun — unlike
// the managed-first resource commands (show ap/site/switch/gateway), which stay
// flat. The introspection subcommands attach to this parent from their own init.
var showAPICmd = &cobra.Command{
	Use:   "api",
	Short: "Inspect API/vendor state (status, profiles, BSSIDs, WLANs)",
	Long: `Read-only views of the configured vendor APIs and their cached state.

Unlike the managed-first resource commands (show ap, show site, ...), these
report what the vendor side knows: connection health, device profiles, RF
profiles, BSSID-to-AP mappings, and WLANs.`,
	Example: `  # API connection health
  wifimgr show api status

  # BSSID-to-AP mappings
  wifimgr show api bssid

  # WLANs/SSIDs from cache
  wifimgr show api wlans`,
}

func init() {
	showCmd.AddCommand(showAPICmd)
}

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

// searchCmd represents the search command
var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search for connected devices on the network",
	Long: `Search for devices connected to the network infrastructure.

This command replicates the web GUI's "find connected devices" feature,
allowing you to locate client devices by hostname or MAC address.

Use 'wifimgr search <subcommand> --help' for detailed information about each search type.`,
	Example: `  # Search for wireless devices
  wifimgr search wireless laptop-john

  # Search for wired devices
  wifimgr search wired aa:bb:cc:dd:ee:ff

  # List every client on a site
  wifimgr search wireless site "MX - Av. Ejercito Nacional Mexicano 904"

  # Search within a specific site
  wifimgr search wireless device-name site US-LAB-01`,
}

func init() {
	rootCmd.AddCommand(searchCmd)
}

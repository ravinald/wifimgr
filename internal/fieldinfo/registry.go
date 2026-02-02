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
package fieldinfo

import (
	"sort"

	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// Registry maps command paths to their data types.
// A nil value indicates the command uses GenericTableData (special handling).
var Registry = map[string]any{
	// API commands - show data from API cache
	"show.api.ap":      (*vendors.DeviceInfo)(nil),
	"show.api.switch":  (*vendors.DeviceInfo)(nil),
	"show.api.gateway": (*vendors.DeviceInfo)(nil),
	"show.api.sites":   (*vendors.SiteInfo)(nil),
	"show.api.wlans":   (*vendors.WLAN)(nil),

	// Inventory commands - show inventory data
	"show.inventory.ap":      (*vendors.InventoryItem)(nil),
	"show.inventory.switch":  (*vendors.InventoryItem)(nil),
	"show.inventory.gateway": (*vendors.InventoryItem)(nil),
	"show.inventory.all":     (*vendors.InventoryItem)(nil),

	// Intent commands - show data from local config files
	// These use actual config types to show all available fields
	"show.intent.site":    nil, // GenericTableData (special handling)
	"show.intent.ap":      (*config.APConfig)(nil),
	"show.intent.switch":  (*config.SwitchConfig)(nil),
	"show.intent.gateway": (*config.WanEdgeConfig)(nil),
}

// GetTypeForCommand returns the struct type for a command path
func GetTypeForCommand(cmdPath string) (any, bool) {
	t, ok := Registry[cmdPath]
	return t, ok
}

// ListCommands returns all registered command paths in sorted order
func ListCommands() []string {
	commands := make([]string, 0, len(Registry))
	for cmd := range Registry {
		commands = append(commands, cmd)
	}
	sort.Strings(commands)
	return commands
}

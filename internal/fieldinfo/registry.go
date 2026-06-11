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

// Registry maps a command path (the dotted form of the CLI hierarchy, e.g.
// `show ap` -> "show.ap") to the source data type its table is built from. The
// `show fields` command extracts the type's fields so operators can discover
// what they may select for display. A nil value marks a command backed by
// GenericTableData, handled specially in the extractor.
//
// Keys are the runtime command paths — the same `show.<noun>` strings the
// commands set as TableConfig.CommandPath for display-column config lookup
// (`display.commands.<key>`). They are a flat config namespace, independent of
// where a command sits in the CLI tree (e.g. `show api bssid` keys as
// `show.bssid`), so `show fields` and a user's column config always agree.
var Registry = map[string]any{
	// Managed resource commands — data from API cache, managed-first.
	"show.ap":      (*vendors.DeviceInfo)(nil),
	"show.switch":  (*vendors.DeviceInfo)(nil),
	"show.gateway": (*vendors.DeviceInfo)(nil),
	"show.sites":   (*vendors.SiteInfo)(nil),

	// Vendor introspection (CLI: show api <view>; config key stays flat).
	"show.wlans":           (*vendors.WLAN)(nil),
	"show.bssid":           (*vendors.BSSIDEntry)(nil),
	"show.device-profiles": (*vendors.DeviceProfile)(nil),
	"show.rf-profiles":     (*vendors.RFTemplate)(nil),

	// Intent commands — data from local config files. These use the actual
	// config types so every settable field is discoverable.
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

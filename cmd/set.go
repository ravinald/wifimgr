package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ravinald/wifimgr/cmd/apply"
	"github.com/ravinald/wifimgr/internal/cmdutils"
	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/intent"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/macaddr"
	"github.com/ravinald/wifimgr/internal/symbols"
	"github.com/ravinald/wifimgr/internal/vendors"
	"github.com/ravinald/wifimgr/internal/xdg"
)

var setCmd = &cobra.Command{
	Use:   "set",
	Short: "Write device intent: config fields, radio settings, or management scope",
	Long: `Write operator intent about a device. Two categories live here:

  - Config fields, written to the site config and pushed with apply.
  - Management scope (managed/unmanaged), written to the local allowlist that
    decides which devices wifimgr is permitted to configure. This is immediate
    and local — no apply.

See 'set site', 'set device', and 'set radio'.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 || cmdutils.ContainsHelp(args) {
			return cmd.Help()
		}
		// The old form was `set <site> <type> <name> ...`; route operators to the
		// site keyword rather than silently printing help on a stray first token.
		return fmt.Errorf("unknown set subcommand %q; did you mean 'set site %s ...'?", args[0], args[0])
	},
}

var setSiteCmd = &cobra.Command{
	Use:   "site <site> <ap|switch|gateway> <name> {<key-path> <value> | managed|unmanaged}",
	Short: "Set a device config field, or arm/disarm devices, at a site",
	Long: `Write a device field into the site config, or toggle whether wifimgr may
manage devices at the site.

Config field — written to the site config, then pushed with apply. The value is
coerced to its JSON type (true/false, integers, floats, else a string), and the
change is validated against the config schema before it is written. The site
config is backed up first, so apply rollback can recover the prior state.

Managed/unmanaged — adds or removes devices from the per-site armed allowlist
(inventory.json), the set of devices wifimgr is permitted to write to. This is
immediate and local; no apply is needed. Bulk forms read the site's devices
from cache, so refresh first if a device is missing.

Examples:
  wifimgr set site US-LAB-01 ap AP-01 radio_config.band_5.channel 36
  wifimgr set site US-LAB-01 ap AP-01 managed
  wifimgr set site US-LAB-01 switch all unmanaged
  wifimgr set site US-LAB-01 all managed`,
	Args: func(_ *cobra.Command, args []string) error {
		if cmdutils.ContainsHelp(args) {
			return nil
		}
		if len(args) < 1 {
			return fmt.Errorf("set site requires a site name")
		}
		_, err := cmdutils.ParseSetSiteArgs(args[1:])
		return err
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if cmdutils.ContainsHelp(args) {
			return cmd.Help()
		}
		siteName := cmdutils.StripQuotes(args[0])
		parsed, err := cmdutils.ParseSetSiteArgs(args[1:])
		if err != nil {
			return err
		}

		if parsed.Action == cmdutils.SetActionConfigField {
			return applySetChanges(siteName, parsed.DeviceType, parsed.Name, []intent.FieldChange{
				{KeyPath: parsed.KeyPath, Value: intent.CoerceValue(parsed.RawValue)},
			})
		}
		return runSiteArming(siteName, parsed)
	},
}

var setDeviceCmd = &cobra.Command{
	Use:   "device <mac> managed|unmanaged",
	Short: "Arm or disarm a device by MAC",
	Long: `Add or remove a device from the per-site armed allowlist (inventory.json)
by MAC. The device's site and type are resolved from cache, so refresh first if
it is missing. A device not yet assigned to a site cannot be armed by MAC — use
'set site <site> ...' once it has a site.

Examples:
  wifimgr set device 5c5b35000001 managed
  wifimgr set device 5c:5b:35:00:00:01 unmanaged`,
	Args: func(_ *cobra.Command, args []string) error {
		if cmdutils.ContainsHelp(args) {
			return nil
		}
		if len(args) != 2 {
			return fmt.Errorf("accepts 2 args (mac managed|unmanaged), received %d", len(args))
		}
		if _, ok := armActionFor(args[1]); !ok {
			return fmt.Errorf("expected managed or unmanaged, got %q", args[1])
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if cmdutils.ContainsHelp(args) {
			return cmd.Help()
		}
		return runDeviceArming(args[0], args[1])
	},
}

func init() {
	rootCmd.AddCommand(setCmd)
	setCmd.AddCommand(setSiteCmd)
	setCmd.AddCommand(setDeviceCmd)
}

// armActionFor reports whether token is managed/unmanaged and, if so, whether it
// arms (true) or disarms (false). It mirrors the keyword set the parser accepts.
func armActionFor(token string) (arm bool, ok bool) {
	switch token {
	case "managed":
		return true, true
	case "unmanaged":
		return false, true
	default:
		return false, false
	}
}

// armTarget is one device an arm/disarm will touch, carrying enough to bucket it
// by type for the allowlist writer and to label it in output.
type armTarget struct {
	mac   string
	name  string
	dtype string
}

// schemasDir returns the directory holding the JSON schemas, falling back to
// the XDG default when the config does not set one.
func schemasDir() string {
	if d := viper.GetString("files.schemas"); d != "" {
		return d
	}
	return xdg.GetSchemasDir()
}

// applySetChanges backs up the site config, writes the field changes through
// the intent engine, and prints a per-field summary plus the apply hint. It is
// shared by set site and set radio.
func applySetChanges(siteName, deviceType, deviceName string, changes []intent.FieldChange) error {
	path, ok := config.GetSiteConfigFullPath(siteName)
	if !ok {
		return fmt.Errorf("set: site %q not found in any configured site file", siteName)
	}
	siteKey, ok := config.GetSiteConfigKey(siteName)
	if !ok {
		return fmt.Errorf("set: site %q has no config key", siteName)
	}

	// Back up before mutating so apply rollback can recover the prior intent.
	if globalConfig != nil {
		if err := apply.CreateConfigBackup(globalConfig, path); err != nil {
			logging.Warnf("set: backup failed, continuing without one: %v", err)
		}
	}

	results, err := intent.SetDeviceFields(intent.SetOptions{
		ConfigFilePath: path,
		SiteKey:        siteKey,
		DeviceType:     deviceType,
		DeviceName:     deviceName,
		SchemaDir:      schemasDir(),
	}, changes)
	if err != nil {
		return err
	}

	changed := 0
	for _, r := range results {
		if r.Changed {
			fmt.Printf("%s %s %s: %v -> %v\n", symbols.SuccessPrefix(), r.DeviceName, r.KeyPath, r.OldValue, r.NewValue)
			changed++
		} else {
			fmt.Printf("  %s %s already %v\n", r.DeviceName, r.KeyPath, r.NewValue)
		}
	}

	if changed == 0 {
		fmt.Println("\nNo changes — config already matches.")
		return nil
	}
	fmt.Printf("\nWrote %s\nReview and push:\n  wifimgr apply site %s %s diff\n", path, siteName, deviceType)
	return nil
}

// runSiteArming resolves the devices a `set site … managed|unmanaged` targets
// from cache and toggles them in the allowlist.
func runSiteArming(siteName string, parsed *cmdutils.ParsedSetSiteArgs) error {
	arm := parsed.Action == cmdutils.SetActionArm

	cacheMgr := GetCacheManager()
	if cacheMgr == nil {
		return fmt.Errorf("cache not initialized; run a refresh first")
	}
	accessor := vendors.GetGlobalCacheAccessor()
	if accessor == nil {
		return fmt.Errorf("cache not initialized; run a refresh first")
	}

	apiLabel, err := ResolveAPIForSite(siteName, siteConfiguredAPI(siteName))
	if err != nil {
		return err
	}
	if apis := cacheMgr.GetSiteAPIs(siteName); len(apis) > 1 {
		logging.Warnf("site %q exists in multiple APIs %v; arming via %s", siteName, apis, apiLabel)
		fmt.Printf("WARN: site %q exists in multiple APIs %v; using %s (override with the site config 'api' field)\n",
			siteName, apis, apiLabel)
	}
	siteID, err := cacheMgr.GetSiteIDByName(apiLabel, siteName)
	if err != nil {
		return err
	}

	var targets []armTarget
	switch parsed.Scope {
	case cmdutils.ScopeSingle:
		items := accessor.GetDevicesBySite(siteID, parsed.DeviceType)
		var matches []*vendors.InventoryItem
		for _, it := range items {
			if strings.EqualFold(it.Name, parsed.Name) {
				matches = append(matches, it)
			}
		}
		switch len(matches) {
		case 0:
			return fmt.Errorf("no %s named %q in site %q (run 'refresh site %s' if it is new)",
				parsed.DeviceType, parsed.Name, siteName, siteName)
		case 1:
			targets = append(targets, armTarget{mac: matches[0].MAC, name: matches[0].Name, dtype: matches[0].Type})
		default:
			macs := make([]string, 0, len(matches))
			for _, m := range matches {
				macs = append(macs, m.MAC)
			}
			return fmt.Errorf("%q matches %d %s devices (%s); arm by MAC with 'set device <mac> ...'",
				parsed.Name, len(matches), parsed.DeviceType, strings.Join(macs, ", "))
		}
	case cmdutils.ScopeAllOfType:
		for _, it := range accessor.GetDevicesBySite(siteID, parsed.DeviceType) {
			targets = append(targets, armTarget{mac: it.MAC, name: it.Name, dtype: it.Type})
		}
	case cmdutils.ScopeAllTypes:
		for _, it := range accessor.GetDevicesBySite(siteID, "") {
			targets = append(targets, armTarget{mac: it.MAC, name: it.Name, dtype: it.Type})
		}
	}

	if len(targets) == 0 {
		fmt.Printf("No devices to %s in site %q — run 'refresh site %s' first.\n",
			armVerb(arm), siteName, siteName)
		return nil
	}

	return applyArming(siteName, targets, arm)
}

// siteConfiguredAPI returns a stub carrying the site's configured API label so
// ResolveAPIForSite honors it over a cache-order guess — decisive when a site
// name exists in more than one vendor. Returns nil when the site isn't in the
// config files or pins no API, leaving resolution to fall back to the cache.
func siteConfiguredAPI(siteName string) *config.SiteConfig {
	obj, err := loadSiteConfiguration(siteName)
	if err != nil || obj.API == "" {
		return nil
	}
	return &config.SiteConfig{API: obj.API}
}

// runDeviceArming resolves a MAC to its site and type from cache and toggles it.
func runDeviceArming(rawMAC, keyword string) error {
	arm, _ := armActionFor(keyword)

	mac := macaddr.NormalizeOrEmpty(rawMAC)
	if mac == "" {
		return fmt.Errorf("invalid MAC: %q", rawMAC)
	}

	cacheMgr := GetCacheManager()
	if cacheMgr == nil {
		return fmt.Errorf("cache not initialized; run a refresh first")
	}
	item, _, err := cacheMgr.FindDeviceByMAC(mac)
	if err != nil {
		return err
	}
	if item.SiteID == "" || item.SiteName == "" {
		return fmt.Errorf("device %s is not assigned to a site; assign it, then arm with 'set site <site> %s %s managed'",
			rawMAC, item.Type, displayName(item.Name, item.MAC))
	}

	target := armTarget{mac: item.MAC, name: item.Name, dtype: item.Type}
	return applyArming(item.SiteName, []armTarget{target}, arm)
}

// applyArming writes the targets into (or out of) the site's allowlist and
// prints an immediate, apply-free summary.
func applyArming(siteName string, targets []armTarget, arm bool) error {
	path := config.InventoryPath(globalConfig)
	if path == "" {
		return fmt.Errorf("no inventory file configured (files.inventory)")
	}

	var aps, switches, gateways []string
	for _, t := range targets {
		switch t.dtype {
		case "ap":
			aps = append(aps, t.mac)
		case "switch":
			switches = append(switches, t.mac)
		case "gateway":
			gateways = append(gateways, t.mac)
		}
	}

	// Compare against the current allowlist so idempotent runs report honestly.
	armed := armedMembership(path, siteName)
	already := 0
	for _, t := range targets {
		if armed[macaddr.NormalizeOrEmpty(t.mac)] == arm {
			already++
		}
	}
	changed := len(targets) - already

	if arm {
		if err := config.ArmSiteDevices(path, siteName, aps, switches, gateways, ""); err != nil {
			return err
		}
	} else {
		if _, err := config.DisarmSiteDevices(path, siteName, aps, switches, gateways); err != nil {
			return err
		}
	}

	printArmingSummary(siteName, path, targets, arm, changed, already)
	return nil
}

// armedMembership returns the set of armed (normalized) MACs across all device
// types for a site, so callers can tell which targets already match the desired
// state. A missing or unreadable file yields an empty set.
func armedMembership(path, siteName string) map[string]bool {
	set := make(map[string]bool)
	f, err := config.LoadInventoryFile(path)
	if err != nil {
		return set
	}
	for _, dtype := range []string{"ap", "switch", "gateway"} {
		for _, m := range f.MACsForSite(siteName, dtype) {
			if n := macaddr.NormalizeOrEmpty(m); n != "" {
				set[n] = true
			}
		}
	}
	return set
}

// printArmingSummary renders the result of an arm/disarm: per-device for a
// single target, a count for bulk, and the no-apply-needed closer.
func printArmingSummary(siteName, path string, targets []armTarget, arm bool, changed, already int) {
	verb := armPastTense(arm)
	prep := "for"
	if !arm {
		prep = "from"
	}

	if changed == 0 {
		if len(targets) == 1 {
			fmt.Printf("  %s already %s\n", displayName(targets[0].name, targets[0].mac), armState(arm))
		} else {
			fmt.Printf("All %d device(s) already %s in %q — no change.\n", len(targets), armState(arm), siteName)
		}
		return
	}

	if len(targets) == 1 {
		fmt.Printf("%s %s %s (%s) %s %s\n", symbols.SuccessPrefix(), verb,
			displayName(targets[0].name, targets[0].mac), targets[0].mac, prep, siteName)
	} else if already > 0 {
		fmt.Printf("%s %s %d device(s) %s %s (%d already %s)\n",
			symbols.SuccessPrefix(), verb, changed, prep, siteName, already, armState(arm))
	} else {
		fmt.Printf("%s %s %d device(s) %s %s\n", symbols.SuccessPrefix(), verb, changed, prep, siteName)
	}
	fmt.Printf("Updated allowlist %s — no apply needed.\n", path)
}

func armVerb(arm bool) string {
	if arm {
		return "arm"
	}
	return "disarm"
}

func armPastTense(arm bool) string {
	if arm {
		return "armed"
	}
	return "disarmed"
}

func armState(arm bool) string {
	if arm {
		return "managed"
	}
	return "unmanaged"
}

// displayName prefers a device's name, falling back to its MAC when unnamed.
func displayName(name, mac string) string {
	if name != "" {
		return name
	}
	return mac
}

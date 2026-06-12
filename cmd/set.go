package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ravinald/wifimgr/cmd/apply"
	"github.com/ravinald/wifimgr/internal/cmdutils"
	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/intent"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/symbols"
	"github.com/ravinald/wifimgr/internal/xdg"
)

var setCmd = &cobra.Command{
	Use:   "set <site> <device-type> <name> <key-path> <value>",
	Short: "Write a device field into the site config",
	Long: `Write a single field into a device's intent in the site config, then push
it with apply. This is the machine-friendly form: any field reachable by a
dot-notation key path can be set, and the change is validated against the
config schema before it is written.

The value is coerced to its JSON type (true/false, integers, floats, else a
string). The site config is backed up before the write, so apply rollback can
recover the prior state.

Examples:
  wifimgr set US-LAB-01 ap AP-01 radio_config.band_5.channel 36
  wifimgr set US-LAB-01 ap AP-01 radio_config.band_5.power 15

For radio settings specifically, 'set radio' offers a friendlier form.`,
	Args: func(_ *cobra.Command, args []string) error {
		if cmdutils.ContainsHelp(args) {
			return nil
		}
		if len(args) != 5 {
			return fmt.Errorf("accepts 5 args (site device-type name key-path value), received %d", len(args))
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if cmdutils.ContainsHelp(args) {
			return cmd.Help()
		}
		site, deviceType, name, keyPath, raw := args[0], args[1], args[2], args[3], args[4]
		if err := validateDeviceType(deviceType); err != nil {
			return err
		}
		return applySetChanges(site, deviceType, name, []intent.FieldChange{
			{KeyPath: keyPath, Value: intent.CoerceValue(raw)},
		})
	},
}

func init() {
	rootCmd.AddCommand(setCmd)
}

// validateDeviceType rejects device types the config does not model under
// "devices", keeping the failure at the CLI boundary with a clear message.
func validateDeviceType(deviceType string) error {
	switch deviceType {
	case "ap", "switch", "gateway":
		return nil
	default:
		return fmt.Errorf("unknown device type %q (expected ap, switch, or gateway)", deviceType)
	}
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
// shared by the generic set command and set radio.
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

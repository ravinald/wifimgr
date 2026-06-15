package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ravinald/wifimgr/internal/cmdutils"
	"github.com/ravinald/wifimgr/internal/intent"
)

// radioBandKeys maps the user-facing band token to the config subtree under
// radio_config. The 2.4GHz band is spelled several ways in the wild.
var radioBandKeys = map[string]string{
	"2.4": "band_24",
	"24":  "band_24",
	"2":   "band_24",
	"5":   "band_5",
	"6":   "band_6",
}

// radioSettingKeys maps a user-facing radio setting to its field under the band
// subtree. "width" is the channel bandwidth in MHz.
var radioSettingKeys = map[string]string{
	"channel": "channel",
	"power":   "power",
	"width":   "bandwidth",
}

var setRadioCmd = &cobra.Command{
	Use:   "radio site <site> <ap-name> band <2.4|5|6> [channel <n>] [power <dBm>] [width <MHz>]",
	Short: "Set AP radio channel, power, or width in the site config",
	Long: `Set radio settings for an AP in the site config, then push them with apply.
This is the human-friendly form of set: positional keywords instead of dot
paths. A band is required; supply any of channel, power, or width.

All changes for the band are written and validated together, then the site
config is backed up and saved.

Examples:
  wifimgr set radio site US-LAB-01 AP-01 band 5 channel 36 power 15 width 80
  wifimgr set radio site US-LAB-01 AP-01 band 2.4 power 8`,
	Args: func(_ *cobra.Command, args []string) error {
		if cmdutils.ContainsHelp(args) {
			return nil
		}
		// site <site> <ap>, then at least one keyword/value pair.
		if len(args) < 5 {
			return fmt.Errorf("accepts at least 5 args (site <site> <ap> band <band>), received %d", len(args))
		}
		if args[0] != "site" {
			return fmt.Errorf("set radio requires the site keyword: radio site <site> <ap> band <band> ...")
		}
		// args after the AP name (args[3:]) must be keyword/value pairs.
		if len(args)%2 == 0 {
			return fmt.Errorf("radio settings must be keyword/value pairs after the AP name")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if cmdutils.ContainsHelp(args) {
			return cmd.Help()
		}
		site, apName := cmdutils.StripQuotes(args[1]), cmdutils.StripQuotes(args[2])

		pairs := args[3:]
		settings := make(map[string]string, len(pairs)/2)
		for i := 0; i < len(pairs); i += 2 {
			settings[pairs[i]] = pairs[i+1]
		}

		bandToken, ok := settings["band"]
		if !ok {
			return fmt.Errorf("set radio: a band is required (band 2.4, 5, or 6)")
		}
		bandKey, ok := radioBandKeys[bandToken]
		if !ok {
			return fmt.Errorf("set radio: unknown band %q (expected 2.4, 5, or 6)", bandToken)
		}

		changes := make([]intent.FieldChange, 0, len(settings))
		for token, raw := range settings {
			if token == "band" {
				continue
			}
			field, ok := radioSettingKeys[token]
			if !ok {
				return fmt.Errorf("set radio: unknown setting %q (expected channel, power, or width)", token)
			}
			changes = append(changes, intent.FieldChange{
				KeyPath: fmt.Sprintf("radio_config.%s.%s", bandKey, field),
				Value:   intent.CoerceValue(raw),
			})
		}
		if len(changes) == 0 {
			return fmt.Errorf("set radio: supply at least one of channel, power, or width")
		}

		return applySetChanges(site, "ap", apName, changes)
	},
}

func init() {
	setCmd.AddCommand(setRadioCmd)
}

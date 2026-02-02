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

	"github.com/ravinald/wifimgr/cmd/apply"
)

// containsHelpArg checks if "help" is present in the arguments
func containsHelpArg(args []string) bool {
	for _, arg := range args {
		if strings.ToLower(arg) == "help" {
			return true
		}
	}
	return false
}

// applyOptions holds parsed optional arguments for apply commands
type applyOptions struct {
	diffMode  bool
	splitDiff bool
	noRefresh bool
}

// parseApplyOptions parses optional positional arguments (diff, split, no-refresh)
func parseApplyOptions(args []string) applyOptions {
	opts := applyOptions{}
	for _, arg := range args {
		switch strings.ToLower(arg) {
		case "diff":
			opts.diffMode = true
		case "split":
			opts.splitDiff = true
		case "no-refresh":
			opts.noRefresh = true
		}
	}
	return opts
}

// validApplyOptions are the allowed optional positional arguments
var validApplyOptions = map[string]bool{
	"diff":       true,
	"split":      true,
	"no-refresh": true,
}

// validateApplyOptions checks that all optional args are valid
func validateApplyOptions(args []string) error {
	for _, arg := range args {
		if !validApplyOptions[strings.ToLower(arg)] {
			return fmt.Errorf("unexpected argument: %s (valid options: diff, split, no-refresh)", arg)
		}
	}
	return nil
}

// applySiteCmd represents the "apply site" command
var applySiteCmd = &cobra.Command{
	Use:   "site <site-name> <device-type> [diff [split]] [no-refresh]",
	Short: "Apply configuration to devices in a site",
	Long: `Apply configuration changes to devices in a specific site.

When multiple APIs are configured:
  - Uses the 'api' field from site config if specified
  - Falls back to cache lookup to find which API has the site

Device types:
  ap       - Apply access point configuration (currently supported)

Note: Switch and gateway configuration support is planned for a future release.

Options:
  diff        - Show changes without applying them (unified format)
  split       - Use side-by-side diff format (requires diff)
  no-refresh  - Skip cache refresh (use existing cache data)

Examples:
  wifimgr apply site US-SFO-LAB ap             - Apply AP configs to site
  wifimgr apply site US-SFO-LAB ap diff        - Show unified diff
  wifimgr apply site US-SFO-LAB ap diff split  - Show side-by-side diff
  wifimgr apply site US-SFO-LAB ap no-refresh  - Apply using cached data`,
	Args: func(cmd *cobra.Command, args []string) error {
		if containsHelpArg(args) {
			return nil
		}
		if len(args) < 2 || len(args) > 5 {
			return fmt.Errorf("accepts 2-5 arg(s), received %d", len(args))
		}
		return validateApplyOptions(args[2:])
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if containsHelpArg(args) {
			return cmd.Help()
		}

		siteName := args[0]
		deviceType := args[1]
		opts := parseApplyOptions(args[2:])
		force, _ := cmd.Flags().GetBool("force")

		// Validate device type
		validTypes := map[string]bool{
			"ap": true, "switch": true, "gateway": true, "all": true,
		}
		if !validTypes[deviceType] {
			return fmt.Errorf("invalid device type: %s. Valid types: ap, switch, gateway, all", deviceType)
		}

		// Validate and resolve API for this site
		apiLabel, err := ValidateMultiVendorApply(globalContext, siteName, nil)
		if err != nil {
			return err
		}

		// Check if apply is supported for this vendor
		if supported, reason := IsMultiVendorApplySupported(apiLabel); !supported {
			return fmt.Errorf("apply not supported: %s", reason)
		}

		fmt.Printf("Applying to site '%s' via API '%s'\n", siteName, apiLabel)

		// Refresh API cache unless no-refresh is specified
		if !opts.noRefresh {
			if err := RefreshAPICacheForApply(globalContext, apiLabel); err != nil {
				return err
			}

			// For Meraki, fetch device configs before applying (on-demand optimization)
			fetchCount, err := EnsureDeviceConfigsForSite(globalContext, apiLabel, siteName, deviceType, nil)
			if err != nil {
				return fmt.Errorf("failed to fetch device configs: %w", err)
			}
			if fetchCount > 0 {
				fmt.Printf("Fetched %d device configs from API\n", fetchCount)
			}
		}

		// Create args for legacy handler
		legacyArgs := []string{siteName, deviceType}
		if opts.diffMode {
			legacyArgs = append(legacyArgs, "diff")
		}
		if opts.splitDiff {
			legacyArgs = append(legacyArgs, "split")
		}

		return apply.HandleCommand(globalContext, globalClient, globalConfig, legacyArgs, apiLabel, force)
	},
}

// Device type subcommands for more intuitive usage
var applyApCmd = &cobra.Command{
	Use:   "ap <site-name> [diff [split]] [no-refresh]",
	Short: "Apply access point configuration to a site",
	Long: `Apply access point configuration to a site.

When multiple APIs are configured, uses site's 'api' field.

Options:
  diff        - Show changes without applying them (unified format)
  split       - Use side-by-side diff format (requires diff)
  no-refresh  - Skip cache refresh (use existing cache data)`,
	Args: func(cmd *cobra.Command, args []string) error {
		if containsHelpArg(args) {
			return nil
		}
		if len(args) < 1 || len(args) > 4 {
			return fmt.Errorf("accepts 1-4 arg(s), received %d", len(args))
		}
		return validateApplyOptions(args[1:])
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if containsHelpArg(args) {
			return cmd.Help()
		}

		siteName := args[0]
		opts := parseApplyOptions(args[1:])
		force, _ := cmd.Flags().GetBool("force")

		apiLabel, err := ValidateMultiVendorApply(globalContext, siteName, nil)
		if err != nil {
			return err
		}
		if supported, reason := IsMultiVendorApplySupported(apiLabel); !supported {
			return fmt.Errorf("apply not supported: %s", reason)
		}
		fmt.Printf("Applying AP config to site '%s' via API '%s'\n", siteName, apiLabel)

		if !opts.noRefresh {
			if err := RefreshAPICacheForApply(globalContext, apiLabel); err != nil {
				return err
			}
			fetchCount, err := EnsureDeviceConfigsForSite(globalContext, apiLabel, siteName, "ap", nil)
			if err != nil {
				return fmt.Errorf("failed to fetch device configs: %w", err)
			}
			if fetchCount > 0 {
				fmt.Printf("Fetched %d device configs from API\n", fetchCount)
			}
		}

		legacyArgs := []string{siteName, "ap"}
		if opts.diffMode {
			legacyArgs = append(legacyArgs, "diff")
		}
		if opts.splitDiff {
			legacyArgs = append(legacyArgs, "split")
		}

		return apply.HandleCommand(globalContext, globalClient, globalConfig, legacyArgs, apiLabel, force)
	},
}

var applySwitchCmd = &cobra.Command{
	Use:   "switch <site-name> [diff [split]] [no-refresh]",
	Short: "Apply switch configuration to a site (not yet supported)",
	Long: `Apply switch configuration to a site.

NOTE: Switch configuration is not yet supported. This command is a placeholder
for a future release. Currently only AP configuration is supported.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if containsHelpArg(args) {
			return nil
		}
		if len(args) < 1 || len(args) > 4 {
			return fmt.Errorf("accepts 1-4 arg(s), received %d", len(args))
		}
		return validateApplyOptions(args[1:])
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if containsHelpArg(args) {
			return cmd.Help()
		}

		siteName := args[0]
		opts := parseApplyOptions(args[1:])
		force, _ := cmd.Flags().GetBool("force")

		apiLabel, err := ValidateMultiVendorApply(globalContext, siteName, nil)
		if err != nil {
			return err
		}
		if supported, reason := IsMultiVendorApplySupported(apiLabel); !supported {
			return fmt.Errorf("apply not supported: %s", reason)
		}
		fmt.Printf("Applying switch config to site '%s' via API '%s'\n", siteName, apiLabel)

		if !opts.noRefresh {
			if err := RefreshAPICacheForApply(globalContext, apiLabel); err != nil {
				return err
			}
			fetchCount, err := EnsureDeviceConfigsForSite(globalContext, apiLabel, siteName, "switch", nil)
			if err != nil {
				return fmt.Errorf("failed to fetch device configs: %w", err)
			}
			if fetchCount > 0 {
				fmt.Printf("Fetched %d device configs from API\n", fetchCount)
			}
		}

		legacyArgs := []string{siteName, "switch"}
		if opts.diffMode {
			legacyArgs = append(legacyArgs, "diff")
		}
		if opts.splitDiff {
			legacyArgs = append(legacyArgs, "split")
		}

		return apply.HandleCommand(globalContext, globalClient, globalConfig, legacyArgs, apiLabel, force)
	},
}

var applyGatewayCmd = &cobra.Command{
	Use:   "gateway <site-name> [diff [split]] [no-refresh]",
	Short: "Apply gateway configuration to a site (not yet supported)",
	Long: `Apply gateway configuration to a site.

NOTE: Gateway configuration is not yet supported. This command is a placeholder
for a future release. Currently only AP configuration is supported.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if containsHelpArg(args) {
			return nil
		}
		if len(args) < 1 || len(args) > 4 {
			return fmt.Errorf("accepts 1-4 arg(s), received %d", len(args))
		}
		return validateApplyOptions(args[1:])
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if containsHelpArg(args) {
			return cmd.Help()
		}

		siteName := args[0]
		opts := parseApplyOptions(args[1:])
		force, _ := cmd.Flags().GetBool("force")

		apiLabel, err := ValidateMultiVendorApply(globalContext, siteName, nil)
		if err != nil {
			return err
		}
		if supported, reason := IsMultiVendorApplySupported(apiLabel); !supported {
			return fmt.Errorf("apply not supported: %s", reason)
		}
		fmt.Printf("Applying gateway config to site '%s' via API '%s'\n", siteName, apiLabel)

		if !opts.noRefresh {
			if err := RefreshAPICacheForApply(globalContext, apiLabel); err != nil {
				return err
			}
			fetchCount, err := EnsureDeviceConfigsForSite(globalContext, apiLabel, siteName, "gateway", nil)
			if err != nil {
				return fmt.Errorf("failed to fetch device configs: %w", err)
			}
			if fetchCount > 0 {
				fmt.Printf("Fetched %d device configs from API\n", fetchCount)
			}
		}

		legacyArgs := []string{siteName, "gateway"}
		if opts.diffMode {
			legacyArgs = append(legacyArgs, "diff")
		}
		if opts.splitDiff {
			legacyArgs = append(legacyArgs, "split")
		}

		return apply.HandleCommand(globalContext, globalClient, globalConfig, legacyArgs, apiLabel, force)
	},
}

var applyAllCmd = &cobra.Command{
	Use:   "all <site-name> [diff [split]] [no-refresh]",
	Short: "Apply all supported device configurations to a site",
	Long: `Apply all supported device configurations to a site.

Currently supported device types:
  - AP (access points)

Note: Switch and gateway support is planned for a future release.

When multiple APIs are configured, uses site's 'api' field.

Options:
  diff        - Show changes without applying them (unified format)
  split       - Use side-by-side diff format (requires diff)
  no-refresh  - Skip cache refresh (use existing cache data)`,
	Args: func(cmd *cobra.Command, args []string) error {
		if containsHelpArg(args) {
			return nil
		}
		if len(args) < 1 || len(args) > 4 {
			return fmt.Errorf("accepts 1-4 arg(s), received %d", len(args))
		}
		return validateApplyOptions(args[1:])
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if containsHelpArg(args) {
			return cmd.Help()
		}

		siteName := args[0]
		opts := parseApplyOptions(args[1:])
		force, _ := cmd.Flags().GetBool("force")

		apiLabel, err := ValidateMultiVendorApply(globalContext, siteName, nil)
		if err != nil {
			return err
		}
		if supported, reason := IsMultiVendorApplySupported(apiLabel); !supported {
			return fmt.Errorf("apply not supported: %s", reason)
		}
		fmt.Printf("Applying all configs to site '%s' via API '%s'\n", siteName, apiLabel)

		if !opts.noRefresh {
			if err := RefreshAPICacheForApply(globalContext, apiLabel); err != nil {
				return err
			}
		}

		legacyArgs := []string{siteName, "all"}
		if opts.diffMode {
			legacyArgs = append(legacyArgs, "diff")
		}
		if opts.splitDiff {
			legacyArgs = append(legacyArgs, "split")
		}

		return apply.HandleCommand(globalContext, globalClient, globalConfig, legacyArgs, apiLabel, force)
	},
}

func init() {
	// Add subcommands to apply
	applyCmd.AddCommand(applySiteCmd)
	applyCmd.AddCommand(applyApCmd)
	applyCmd.AddCommand(applySwitchCmd)
	applyCmd.AddCommand(applyGatewayCmd)
	applyCmd.AddCommand(applyAllCmd)

	// Add flags to site command and its subcommands
	for _, cmd := range []*cobra.Command{applySiteCmd, applyApCmd, applySwitchCmd, applyGatewayCmd, applyAllCmd} {
		cmd.Flags().BoolP("force", "f", false, "Force apply even if no changes detected")
	}
}

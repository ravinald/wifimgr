/*
Copyright © 2025 Ravi Pina <ravi@pina.org>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/
package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ravinald/wifimgr/internal/vendors"
)

// formatRefreshError renders a refresh-batch error in the most useful form
// available. Typed errors from internal/vendors that implement UserMessage
// get the user-friendly rendering (remediation hint); everything else falls
// back to the plain error string.
func formatRefreshError(err error) string {
	if err == nil {
		return ""
	}
	var authErr *vendors.AuthError
	if errors.As(err, &authErr) {
		return authErr.UserMessage()
	}
	var srvErr *vendors.ServerError
	if errors.As(err, &srvErr) {
		return srvErr.UserMessage()
	}
	var rlErr *vendors.RateLimitError
	if errors.As(err, &rlErr) {
		return rlErr.UserMessage()
	}
	var nfErr *vendors.NotFoundError
	if errors.As(err, &nfErr) {
		return nfErr.UserMessage()
	}
	var tErr *vendors.TransportError
	if errors.As(err, &tErr) {
		return tErr.UserMessage()
	}
	return err.Error()
}

// refreshDeviceCmd represents `refresh device` — the former `refresh cache`.
// Renamed because the tool now persists two distinct caches (device-level
// infrastructure data here, plus per-client detail populated by
// `refresh client`). Calling both "cache" made the subcommand labels
// ambiguous; "device" reflects what this command actually refreshes:
// sites, inventory, device configs, WLANs, and statuses.
var refreshDeviceCmd = &cobra.Command{
	Use:   "device [api-name]",
	Short: "Refresh device-level cache (sites, inventory, configs, WLANs, statuses)",
	Long: `Refresh the per-API device-level cache: sites, inventory, device configs,
WLANs, and statuses.

When multiple APIs are configured:
  - Without api-name: Refreshes all APIs in parallel
  - With api-name: Refreshes only the specified API

Per-client detail (e.g. Meraki connected band) is NOT touched by this
command — use 'refresh client site <name>' or 'refresh all' for that.

Examples:
  wifimgr refresh device                    # Refresh all APIs
  wifimgr refresh device mist-prod          # Refresh mist-prod only
  wifimgr refresh device meraki-corp        # Refresh meraki-corp only`,
	Args: func(cmd *cobra.Command, args []string) error {
		// Allow "help" as a special keyword
		for _, arg := range args {
			if strings.ToLower(arg) == "help" {
				return nil
			}
		}
		if len(args) > 1 {
			return fmt.Errorf("accepts at most 1 arg(s), received %d", len(args))
		}
		return nil
	},
	RunE: runRefreshDevice,
}

func init() {
	refreshCmd.AddCommand(refreshDeviceCmd)
}

func runRefreshDevice(cmd *cobra.Command, args []string) error {
	// Check for help keyword in positional arguments
	for _, arg := range args {
		if strings.ToLower(arg) == "help" {
			return cmd.Help()
		}
	}

	// Check if API name was provided as positional argument
	if len(args) > 0 && apiFlag == "" {
		apiFlag = args[0]
	}
	return runMultiVendorRefresh()
}

// runMultiVendorRefresh handles device-level cache refresh for multi-vendor mode.
func runMultiVendorRefresh() error {
	cacheMgr := GetCacheManager()
	if cacheMgr == nil {
		return fmt.Errorf("cache manager not initialized")
	}

	registry := GetAPIRegistry()
	if registry == nil {
		return fmt.Errorf("API registry not initialized")
	}

	// Validate target API if provided
	if err := ValidateAPIFlag(); err != nil {
		return err
	}

	targetAPIs := GetTargetAPIs()
	if len(targetAPIs) == 0 {
		return fmt.Errorf("no APIs configured")
	}

	ctx := globalContext

	if apiFlag != "" {
		// Single API refresh
		fmt.Printf("Refreshing device cache for %s...\n", apiFlag)
		if err := cacheMgr.RefreshAPI(ctx, apiFlag); err != nil {
			return fmt.Errorf("failed to refresh %s: %w", apiFlag, err)
		}
		fmt.Printf("Successfully refreshed %s\n", apiFlag)
	} else {
		// Parallel refresh of all APIs
		fmt.Printf("Refreshing device cache for %d APIs...\n", len(targetAPIs))

		errors := cacheMgr.RefreshAllAPIs(ctx)

		successCount := len(targetAPIs) - len(errors)
		fmt.Printf("\nRefreshed %d/%d APIs successfully\n", successCount, len(targetAPIs))

		if len(errors) > 0 {
			fmt.Println("\nErrors:")
			for apiLabel, err := range errors {
				fmt.Printf("  %s: %s\n", apiLabel, formatRefreshError(err))
			}
		}

		fmt.Println("Rebuilt cross-API index")
	}

	return nil
}

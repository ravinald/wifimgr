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

	"github.com/ravinald/wifimgr/internal/cmdutils"
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
	Use:   "device [api <api-label>] [site <site-name> [api <api-label>]]",
	Short: "Refresh device-level cache (sites, inventory, configs, WLANs, statuses)",
	Long: `Refresh the per-API device-level cache: sites, inventory, device configs,
WLANs, and statuses.

Forms:
  refresh device                          Refresh every configured API in parallel.
  refresh device api <api-label>          Refresh only the named API.
  refresh device site <site-name>         Refresh org-scoped data plus per-device
                                          configs limited to the named site. The
                                          API is auto-detected from the site name.
  refresh device site <site-name> api <api-label>
                                          Same, but disambiguates when the site
                                          name exists in more than one API.

The site-scoped form is intended for Meraki, where per-device config fetches
dominate the cost of a refresh. Org-scoped data (sites, inventory, statuses,
templates, profiles, WLANs) is still refreshed; configs for devices in
*other* sites are preserved from the prior cache.

Per-client detail (e.g. Meraki connected band) is NOT touched by this
command — use 'refresh client site <name>' or 'refresh all' for that.`,
	Example: `  wifimgr refresh device                              # all APIs
  wifimgr refresh device api meraki-corp              # one API
  wifimgr refresh device site US-LAB-01               # one site (auto-detect API)
  wifimgr refresh device site US-LAB-01 api meraki-corp`,
	Args: func(cmd *cobra.Command, args []string) error {
		// Allow "help" as a special keyword
		for _, arg := range args {
			if strings.ToLower(arg) == "help" {
				return nil
			}
		}
		return nil
	},
	RunE: runRefreshDevice,
}

func init() {
	refreshCmd.AddCommand(refreshDeviceCmd)
}

func runRefreshDevice(cmd *cobra.Command, args []string) error {
	for _, arg := range args {
		if strings.ToLower(arg) == "help" {
			return cmd.Help()
		}
	}

	parsed, err := cmdutils.ParseRefreshArgs(args, cmdutils.ParseRefreshOptions{
		AllowSite: true,
	})
	if err != nil {
		return err
	}

	if parsed.SiteName != "" {
		return runMultiVendorRefreshSite(parsed.SiteName, parsed.APIName)
	}

	if parsed.APIName != "" {
		apiFlag = parsed.APIName
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

// runMultiVendorRefreshSite refreshes the device-level cache scoped to a
// single site. Org-scoped data is still pulled in full; per-device config
// loops are filtered to the named site only.
func runMultiVendorRefreshSite(siteName, apiName string) error {
	cacheMgr := GetCacheManager()
	if cacheMgr == nil {
		return fmt.Errorf("cache manager not initialized")
	}

	site, err := resolveSiteForRefresh(siteName, apiName)
	if err != nil {
		return err
	}

	fmt.Printf("Refreshing device cache for %s (%s)...\n", site.Name, site.SourceAPI)
	if err := cacheMgr.RefreshAPISite(globalContext, site.SourceAPI, site.ID); err != nil {
		return fmt.Errorf("failed to refresh %s: %s", site.SourceAPI, formatRefreshError(err))
	}
	fmt.Printf("Successfully refreshed %s for site %s\n", site.SourceAPI, site.Name)
	return nil
}

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
	"time"

	"github.com/spf13/cobra"

	"github.com/ravinald/wifimgr/internal/cmdutils"
	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/refreshui"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// refreshCmd is both the command group (parent of `refresh client`) and a
// runnable command. With no recognized subcommand it refreshes the cache:
// managed devices by default, everything with `all`.
var refreshCmd = &cobra.Command{
	Use:   "refresh [all|detail] [site <site-name> [all|detail]] [target <api-label>]",
	Short: "Refresh cached data from API (managed devices by default)",
	Long: `Refresh the per-API cache. wifimgr manages a subset of what the vendor knows
about, so by default refresh only pulls configs for the devices armed in your
per-site inventory — cheap, and meant to run often. 'all' drops that filter and
pulls everything the API has.

Forms:
  refresh                         Managed devices across every configured site/API.
  refresh all                     Everything the API has, every site, + client detail.
  refresh detail                  Managed devices + per-client detail.
  refresh site <site-name>        Managed devices in one site.
  refresh site <site-name> detail Managed devices + client detail for one site.
  refresh site <site-name> all    Everything the API has for one site + client detail.
  refresh ... target <api-label>  Disambiguate a site whose name spans APIs.

The managed default is the cost win on Meraki, where per-device config fetches
dominate. Org-scoped data (sites, inventory, statuses, WLANs) is always
refreshed; configs for unmanaged devices are preserved from the prior cache.

Per-client detail (e.g. Meraki connected band) rides the 'detail' and 'all'
levels; 'refresh client site <name>' fetches it on its own.`,
	Example: `  wifimgr refresh                              # managed, all sites
  wifimgr refresh all                          # everything, all sites
  wifimgr refresh site US-LAB-01               # managed devices in one site
  wifimgr refresh site US-LAB-01 all           # everything the API has for that site
  wifimgr refresh site US-LAB-01 target meraki-corp`,
	Args: func(_ *cobra.Command, args []string) error {
		for _, arg := range args {
			if strings.ToLower(arg) == "help" {
				return nil
			}
		}
		return nil
	},
	RunE: runRefresh,
}

func init() {
	rootCmd.AddCommand(refreshCmd)
}

func runRefresh(cmd *cobra.Command, args []string) error {
	for _, arg := range args {
		if strings.ToLower(arg) == "help" {
			return cmd.Help()
		}
	}

	parsed, err := cmdutils.ParseRefreshArgs(args, cmdutils.ParseRefreshOptions{
		AllowSite:  true,
		AllowScope: true,
	})
	if err != nil {
		return err
	}

	if parsed.SiteName != "" {
		return runRefreshSite(parsed.SiteName, parsed.Target, parsed.Scope)
	}

	SetAPITarget(parsed.Target)
	return runRefreshAllSites(parsed.Scope)
}

// managedMACs returns the armed (normalized) MAC set for the given sites, or
// for every armed site when sites is empty. A legacy-schema inventory file is
// fatal — the operator must migrate. A missing/unreadable file yields an empty
// set with a warning: managed-first means nothing is armed until you say so.
func managedMACs(sites []string) (map[string]bool, error) {
	inv, err := config.LoadInventoryFile(config.InventoryPath(nil))
	if err != nil {
		if errors.Is(err, config.ErrLegacyInventorySchema) {
			return nil, err
		}
		logging.Warnf("inventory unavailable (%v); no devices armed — use 'refresh all' to fetch everything", err)
		return map[string]bool{}, nil
	}
	return inv.NormalizedSet(sites, ""), nil
}

// runRefreshAllSites refreshes every targeted API. Default scope limits the
// per-device config fetch to the armed set; "all" fetches everything. "detail"
// and "all" also refresh per-client detail.
func runRefreshAllSites(scope string) error {
	cacheMgr := GetCacheManager()
	if cacheMgr == nil {
		return fmt.Errorf("cache manager not initialized")
	}
	if GetAPIRegistry() == nil {
		return fmt.Errorf("API registry not initialized")
	}
	if err := ValidateAPIFlag(); err != nil {
		return err
	}

	targetAPIs := GetTargetAPIs()
	if len(targetAPIs) == 0 {
		return fmt.Errorf("no APIs configured")
	}
	ctx := globalContext

	// nil managed set means "no filter" (the `all` scope). Otherwise scope the
	// fetch to the armed devices across every armed site.
	var managed map[string]bool
	if scope != cmdutils.RefreshScopeAll {
		m, err := managedMACs(nil)
		if err != nil {
			return err
		}
		managed = m
		if len(managed) == 0 {
			fmt.Println("No armed devices in inventory — refreshing org data only. Arm devices in inventory.json or run 'refresh all'.")
		}
	}

	if apiFlag != "" {
		fmt.Printf("Refreshing cache for %s...\n", apiFlag)
		if err := cacheMgr.RefreshAPIManaged(ctx, apiFlag, managed); err != nil {
			return fmt.Errorf("failed to refresh %s: %s", apiFlag, formatRefreshError(err))
		}
		fmt.Printf("Successfully refreshed %s\n", apiFlag)
	} else {
		fmt.Printf("Refreshing cache for %d APIs...\n", len(targetAPIs))

		// On a terminal, drive the live status board and hold log output (incl.
		// the index-rebuild MAC-collision warnings) until the board tears down so
		// it doesn't paint over the render. Piped/redirected output keeps the
		// linear text and logs unbuffered.
		interactive := refreshui.Interactive()
		reporter, stopBoard := refreshui.New(targetAPIs, interactive)
		release := func() {}
		if interactive {
			release = logging.PauseOutput()
		}

		var errs map[string]error
		if managed == nil {
			errs = cacheMgr.RefreshAllAPIs(ctx, reporter)
		} else {
			errs = cacheMgr.RefreshAllAPIsManaged(ctx, managed, reporter)
		}
		stopBoard()
		release()

		successCount := len(targetAPIs) - len(errs)
		fmt.Printf("\nRefreshed %d/%d APIs successfully\n", successCount, len(targetAPIs))
		if len(errs) > 0 {
			fmt.Println("\nErrors:")
			for apiLabel, err := range errs {
				fmt.Printf("  %s: %s\n", apiLabel, formatRefreshError(err))
			}
		}
		fmt.Println("Rebuilt cross-API index")
	}

	if scope == cmdutils.RefreshScopeDetail || scope == cmdutils.RefreshScopeAll {
		// Client detail runs after the cache refresh so the cache it iterates
		// over is up-to-date.
		return runMultiVendorClientDetailRefresh()
	}
	return nil
}

// runRefreshSite refreshes a single site. Default scope limits the per-device
// config fetch to the site's armed devices; "all" fetches every device the API
// reports for the site. "detail" and "all" also pull per-client detail.
func runRefreshSite(siteName, target, scope string) error {
	cacheMgr := GetCacheManager()
	if cacheMgr == nil {
		return fmt.Errorf("cache manager not initialized")
	}

	site, err := resolveSiteForRefresh(siteName, target)
	if err != nil {
		return err
	}

	var managed map[string]bool
	if scope != cmdutils.RefreshScopeAll {
		m, err := managedMACs([]string{site.Name})
		if err != nil {
			return err
		}
		managed = m
	}

	fmt.Printf("Refreshing cache for %s (%s)...\n", site.Name, site.SourceAPI)
	if err := cacheMgr.RefreshAPISite(globalContext, site.SourceAPI, site.ID, managed); err != nil {
		return fmt.Errorf("failed to refresh %s: %s", site.SourceAPI, formatRefreshError(err))
	}
	fmt.Printf("Successfully refreshed %s for site %s\n", site.SourceAPI, site.Name)

	if scope == cmdutils.RefreshScopeDetail || scope == cmdutils.RefreshScopeAll {
		return refreshClientDetailForSite(globalContext, site)
	}
	return nil
}

// runMultiVendorClientDetailRefresh iterates every API with a
// ClientDetailService and refreshes per-site client detail for every site in
// that API's cache. Errors on individual sites are logged and the loop
// continues — a single bad site shouldn't abort the rest.
func runMultiVendorClientDetailRefresh() error {
	cacheMgr := GetCacheManager()
	if cacheMgr == nil {
		return fmt.Errorf("cache manager not initialized")
	}
	registry := GetAPIRegistry()
	if registry == nil {
		return fmt.Errorf("API registry not initialized")
	}
	if err := ValidateAPIFlag(); err != nil {
		return err
	}

	targetAPIs := GetTargetAPIs()
	if len(targetAPIs) == 0 {
		return nil
	}

	fmt.Println()
	for _, apiLabel := range targetAPIs {
		client, err := registry.GetClient(apiLabel)
		if err != nil {
			fmt.Printf("  [%s] client unavailable: %v\n", apiLabel, err)
			continue
		}
		svc := client.ClientDetail()
		if svc == nil {
			// Vendor doesn't have per-client supplements worth caching;
			// silently skip so the output isn't noisy on Mist orgs.
			continue
		}

		cache, err := cacheMgr.GetAPICache(apiLabel)
		if err != nil {
			fmt.Printf("  [%s] cache unavailable: %v\n", apiLabel, err)
			continue
		}

		siteCount := len(cache.Sites.Info)
		if siteCount == 0 {
			continue
		}

		fmt.Printf("Refreshing client detail for %s (%d sites)…\n", apiLabel, siteCount)
		start := time.Now()
		total := 0
		errCount := 0
		for i := range cache.Sites.Info {
			if err := globalContext.Err(); err != nil {
				fmt.Printf("  [%s] cancelled after %d/%d sites\n", apiLabel, i, len(cache.Sites.Info))
				return err
			}
			site := &cache.Sites.Info[i]
			records, err := svc.FetchSiteClientDetail(globalContext, site.ID)
			if err != nil {
				fmt.Printf("  [%s] %s: %v\n", apiLabel, site.Name, err)
				errCount++
				continue
			}
			if _, err := cacheMgr.SaveClientDetail(apiLabel, records); err != nil {
				fmt.Printf("  [%s] %s: save failed: %v\n", apiLabel, site.Name, err)
				errCount++
				continue
			}
			total += len(records)
		}
		fmt.Printf("  %d clients across %d sites (%d errors) in %s\n",
			total, siteCount, errCount, time.Since(start).Round(time.Millisecond))
	}

	return nil
}

// formatRefreshError renders a refresh-batch error in the most useful form
// available. Typed errors from internal/vendors that implement UserMessage get
// the user-friendly rendering (remediation hint); everything else falls back to
// the plain error string.
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

/*
Copyright © 2025 Ravi Pina <ravi@pina.org>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/
package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ravinald/wifimgr/internal/cmdutils"
)

// refreshAllCmd represents `refresh all` — a convenience command that runs
// every refresh subcommand wifimgr knows about. Today that means the full
// per-API cache refresh and, for vendors that support it (Meraki), per-client
// detail for every cached site. Added here so operators don't have to
// remember which piece is cached where; new refresh types should wire into
// this command when they land.
var refreshAllCmd = &cobra.Command{
	Use:   "all [api <api-label>] [site <site-name> [api <api-label>]]",
	Short: "Refresh every cached data source (device cache + client detail)",
	Long: `Runs every refresh subcommand in sequence:

  1. 'refresh device'         — sites, inventory, device configs, WLANs, etc.
  2. 'refresh client site X'  — per-client detail (e.g. Meraki connected band)
                                for every site in every API that supports it.

Step 2 can be expensive on large Meraki orgs (roughly 3 API calls per site).
Use 'refresh device' if you don't need client detail right now, or
'refresh client site <name>' for a single site.

Forms:
  refresh all                                     Refresh every API end to end.
  refresh all api <api-label>                     Limit both steps to one API.
  refresh all site <site-name>                    Refresh device cache + client
                                                  detail for a single site only.
  refresh all site <site-name> api <api-label>    Same, with API disambiguation.`,
	Example: `  wifimgr refresh all
  wifimgr refresh all api meraki-corp
  wifimgr refresh all site US-LAB-01
  wifimgr refresh all site US-LAB-01 api meraki-corp`,
	Args: func(_ *cobra.Command, args []string) error {
		for _, arg := range args {
			if strings.ToLower(arg) == "help" {
				return nil
			}
		}
		return nil
	},
	RunE: runRefreshAll,
}

func runRefreshAll(cmd *cobra.Command, args []string) error {
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
		return runMultiVendorRefreshAllSite(parsed.SiteName, parsed.APIName)
	}

	if parsed.APIName != "" {
		apiFlag = parsed.APIName
	}

	if err := runMultiVendorRefresh(); err != nil {
		return fmt.Errorf("cache refresh: %w", err)
	}

	// Client detail runs after the cache refresh so the cache it iterates
	// over is up-to-date.
	return runMultiVendorClientDetailRefresh()
}

// runMultiVendorRefreshAllSite refreshes both the device cache and the
// per-client detail for a single site. Used by `refresh all site <name>`.
func runMultiVendorRefreshAllSite(siteName, apiName string) error {
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
		return fmt.Errorf("cache refresh: %s", formatRefreshError(err))
	}

	// Per-site client detail is a no-op for vendors without a ClientDetail
	// service; the helper handles that gracefully.
	return refreshClientDetailForSite(globalContext, site)
}

// runMultiVendorClientDetailRefresh iterates every API with a
// ClientDetailService and refreshes per-site client detail for every site
// in that API's cache. Errors on individual sites are logged and the loop
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
		return nil // nothing to do
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

func init() {
	refreshCmd.AddCommand(refreshAllCmd)
}

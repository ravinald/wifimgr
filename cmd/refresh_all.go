/*
Copyright © 2025 Ravi Pina <ravi@pina.org>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/
package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

// refreshAllCmd represents `refresh all` — a convenience command that runs
// every refresh subcommand wifimgr knows about. Today that means the full
// per-API cache refresh and, for vendors that support it (Meraki), per-client
// detail for every cached site. Added here so operators don't have to
// remember which piece is cached where; new refresh types should wire into
// this command when they land.
var refreshAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Refresh every cached data source (device cache + client detail)",
	Long: `Runs every refresh subcommand in sequence:

  1. 'refresh device'         — sites, inventory, device configs, WLANs, etc.
  2. 'refresh client site X'  — per-client detail (e.g. Meraki connected band)
                                for every site in every API that supports it.

Step 2 can be expensive on large Meraki orgs (roughly 3 API calls per site).
Use 'refresh device' if you don't need client detail right now, or
'refresh client site <name>' for a single site.

When multiple APIs are configured:
  - Without target: processes every API
  - With target:    limits both steps to the specified API`,
	Example: `  wifimgr refresh all
  wifimgr refresh all target meraki-corp`,
	RunE: runRefreshAll,
}

func runRefreshAll(_ *cobra.Command, _ []string) error {
	if err := runMultiVendorRefresh(); err != nil {
		return fmt.Errorf("cache refresh: %w", err)
	}

	// Client detail runs after the cache refresh so the cache it iterates
	// over is up-to-date.
	return runMultiVendorClientDetailRefresh()
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

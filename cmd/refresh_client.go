/*
Copyright © 2025 Ravi Pina <ravi@pina.org>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/
package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ravinald/wifimgr/internal/cmdutils"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// refreshClientCmd represents the `refresh client` subcommand group.
var refreshClientCmd = &cobra.Command{
	Use:   "client",
	Short: "Refresh per-client supplemental data (e.g. Meraki connected band)",
	Long: `Refresh per-client supplemental data that isn't available from the default
search endpoints.

This exists primarily for Meraki, whose client list does not include connected
band. The fetched data is cached per API and displayed when you run
'search wireless ... detail'.

Use 'wifimgr refresh client site <site-name>' to populate the cache for a
single site. Mist returns band natively and is a no-op for this command.`,
	Example: `  # Populate client detail for a Meraki site
  wifimgr refresh client site US-LAB-01`,
}

// refreshClientSiteCmd represents `refresh client site <site-name>`.
var refreshClientSiteCmd = &cobra.Command{
	Use:   "site <site-name> [target <api-label>]",
	Short: "Refresh per-client detail for a single site",
	Long: `Populate the per-client detail cache for the named site.

The command identifies which API the site belongs to and, if that API supports
client-detail fetches (today: Meraki), makes the vendor-specific calls needed
to enrich the local cache.

Arguments:
  site-name    Required. The site name. Typo suggestions are returned on miss.
  target       Optional. Keyword followed by API label to disambiguate when the
               same site name exists in more than one configured API.`,
	Example: `  wifimgr refresh client site US-LAB-01
  wifimgr refresh client site "MX - Av. Ejercito Nacional Mexicano 904"
  wifimgr refresh client site US-LAB-01 target meraki-corp`,
	Args: func(cmd *cobra.Command, args []string) error {
		for _, arg := range args {
			if strings.ToLower(arg) == "help" {
				return nil
			}
		}
		if len(args) < 1 {
			return fmt.Errorf("requires at least the site name")
		}
		return nil
	},
	RunE: runRefreshClientSite,
}

func runRefreshClientSite(cmd *cobra.Command, args []string) error {
	for _, arg := range args {
		if strings.ToLower(arg) == "help" {
			return cmd.Help()
		}
	}

	parsed, err := cmdutils.ParseRefreshArgs(args, cmdutils.ParseRefreshOptions{
		AllowSite:         true,
		AllowImplicitSite: true,
	})
	if err != nil {
		return err
	}
	if parsed.SiteName == "" {
		return fmt.Errorf("site name required")
	}

	site, err := resolveSiteForRefresh(parsed.SiteName, parsed.Target)
	if err != nil {
		return err
	}

	return refreshClientDetailForSite(globalContext, site)
}

// resolveSiteForRefresh looks up the named site, optionally constraining the
// search to a specific API label. Used by every per-site refresh command.
func resolveSiteForRefresh(siteName, apiLabel string) (*vendors.SiteInfo, error) {
	ref, err := cmdutils.ResolveSite(siteName, apiLabel)
	if err != nil {
		return nil, err
	}

	cacheAccessor, err := cmdutils.GetCacheAccessor()
	if err != nil {
		return nil, fmt.Errorf("failed to get cache accessor: %w", err)
	}
	// Resolution is duplicate-safe by name; the full record comes back by ID,
	// which is unique.
	return cacheAccessor.GetSiteByID(ref.SiteID)
}

// refreshClientDetailForSite fetches per-client detail for a single site and
// merges it into the API's cache. No-op for vendors that don't expose a
// ClientDetail service (today: everyone except Meraki).
func refreshClientDetailForSite(ctx context.Context, site *vendors.SiteInfo) error {
	apiLabel := site.SourceAPI

	registry := GetAPIRegistry()
	if registry == nil {
		return fmt.Errorf("API registry not initialized")
	}
	client, err := registry.GetClient(apiLabel)
	if err != nil {
		return fmt.Errorf("failed to get client for %s: %w", apiLabel, err)
	}

	svc := client.ClientDetail()
	if svc == nil {
		fmt.Printf("No client-detail service for %s (vendor %s) — nothing to refresh.\n",
			apiLabel, site.SourceVendor)
		return nil
	}

	fmt.Printf("Refreshing client detail for %s (%s)…\n", site.Name, apiLabel)
	start := time.Now()
	records, err := svc.FetchSiteClientDetail(ctx, site.ID)
	if err != nil {
		return fmt.Errorf("fetch client detail: %w", err)
	}

	cacheMgr := GetCacheManager()
	if cacheMgr == nil {
		return fmt.Errorf("cache manager not initialized")
	}
	newest, err := cacheMgr.SaveClientDetail(apiLabel, records)
	if err != nil {
		return fmt.Errorf("save client detail: %w", err)
	}

	logging.Debugf("refresh client wrote %d records, newest=%s", len(records), newest)
	fmt.Printf("  %d clients — band from connection stats (last 24h)\n", len(records))
	fmt.Printf("  done in %s\n", time.Since(start).Round(time.Millisecond))
	return nil
}

func init() {
	refreshCmd.AddCommand(refreshClientCmd)
	refreshClientCmd.AddCommand(refreshClientSiteCmd)
}

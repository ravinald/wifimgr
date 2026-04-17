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
  target       Optional. Keyword followed by API label to override the API.`,
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

// refreshClientSiteArgs carries the parsed positional inputs for the command.
type refreshClientSiteArgs struct {
	siteName string
	target   string
}

// parseRefreshClientSiteArgs recognizes either:
//
//	site <name> [target <api>]
//	<name> [target <api>]
//
// The leading `site` keyword is optional — it reads naturally on the command
// line and stays consistent with other `site <name>` positional patterns, but
// dropping it is also common when the command already says `site` in the path.
func parseRefreshClientSiteArgs(args []string) (*refreshClientSiteArgs, error) {
	result := &refreshClientSiteArgs{}
	i := 0
	if i < len(args) && strings.EqualFold(args[i], "site") {
		i++
	}
	if i >= len(args) {
		return nil, fmt.Errorf("site name required")
	}
	result.siteName = args[i]
	i++

	for i < len(args) {
		switch strings.ToLower(args[i]) {
		case "target":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("'target' requires an API label")
			}
			result.target = args[i+1]
			i += 2
		default:
			return nil, fmt.Errorf("unexpected argument: %s", args[i])
		}
	}
	return result, nil
}

func runRefreshClientSite(cmd *cobra.Command, args []string) error {
	for _, arg := range args {
		if strings.ToLower(arg) == "help" {
			return cmd.Help()
		}
	}

	parsed, err := parseRefreshClientSiteArgs(args)
	if err != nil {
		return err
	}

	cacheAccessor, err := cmdutils.GetCacheAccessor()
	if err != nil {
		return fmt.Errorf("failed to get cache accessor: %w", err)
	}

	var site *vendors.SiteInfo
	if parsed.target != "" {
		site, err = cacheAccessor.GetSiteByNameAndAPI(parsed.siteName, parsed.target)
	} else {
		site, err = cacheAccessor.GetSiteByName(parsed.siteName)
	}
	if err != nil {
		return err // enriched with "did you mean?" suggestions by the accessor
	}

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
	records, err := svc.FetchSiteClientDetail(globalContext, site.ID)
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
	fmt.Printf("  %d clients — band from connection stats (last hour)\n", len(records))
	fmt.Printf("  done in %s\n", time.Since(start).Round(time.Millisecond))
	return nil
}

func init() {
	refreshCmd.AddCommand(refreshClientCmd)
	refreshClientCmd.AddCommand(refreshClientSiteCmd)
}

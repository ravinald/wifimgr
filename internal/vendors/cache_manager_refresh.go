package vendors

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ravinald/wifimgr/internal/logging"
)

// RefreshAPI refreshes a single API's cache using its client.
// This is a convenience wrapper that calls RefreshAPIWithOptions with FetchDeviceConfigs=true.
func (c *CacheManager) RefreshAPI(ctx context.Context, apiLabel string) error {
	return c.RefreshAPIWithOptions(ctx, apiLabel, RefreshOptions{FetchDeviceConfigs: true})
}

// RefreshAPIWithOptions refreshes a single API's cache with configurable options.
func (c *CacheManager) RefreshAPIWithOptions(ctx context.Context, apiLabel string, opts RefreshOptions) error {
	logging.Debugf("[cache] Starting refresh for API %s (fetchConfigs=%v)", apiLabel, opts.FetchDeviceConfigs)

	client, err := c.registry.GetClient(apiLabel)
	if err != nil {
		logging.Debugf("[cache] Failed to get client for %s: %v", apiLabel, err)
		return err
	}

	config, err := c.registry.GetConfig(apiLabel)
	if err != nil {
		logging.Debugf("[cache] Failed to get config for %s: %v", apiLabel, err)
		return err
	}

	// Check if this is initial cache creation (cache doesn't exist yet)
	isInitialCreation := !c.CacheExists(apiLabel)
	if isInitialCreation {
		logging.Debugf("[cache] Initial cache creation for %s - will fetch device configs", apiLabel)
	}

	// Determine if we should fetch device configs:
	// - Always for Mist (supports efficient bulk fetches)
	// - For Meraki: only on initial creation or explicit request (due to per-device API calls)
	shouldFetchConfigs := opts.FetchDeviceConfigs || isInitialCreation
	if config.Vendor == "mist" {
		shouldFetchConfigs = true // Mist always fetches configs
	}

	logging.Debugf("[cache] Refreshing %s (vendor=%s, org=%s, fetchConfigs=%v)", apiLabel, config.Vendor, config.Credentials["org_id"], shouldFetchConfigs)

	// Progress message: Starting API refresh
	fmt.Printf("  [%s] Refreshing %s API...\n", apiLabel, config.Vendor)

	startTime := time.Now()

	// Create new cache
	cache := NewAPICache(apiLabel, config.Vendor, config.Credentials["org_id"])
	cache.Meta.LastRefresh = startTime

	// Fetch sites
	fmt.Printf("    Fetching sites...")
	logging.Debugf("[cache] Fetching sites for %s", apiLabel)
	if sitesSvc := client.Sites(); sitesSvc != nil {
		sites, err := sitesSvc.List(ctx)
		if err != nil {
			fmt.Printf(" error\n")
			logging.Debugf("[cache] Failed to fetch sites for %s: %v", apiLabel, err)
			return fmt.Errorf("failed to fetch sites: %w", err)
		}
		fmt.Printf(" %d sites\n", len(sites))
		logging.Debugf("[cache] Fetched %d sites for %s", len(sites), apiLabel)
		for _, site := range sites {
			cache.Sites.Info = append(cache.Sites.Info, *site)
		}
	} else {
		fmt.Printf(" not supported\n")
	}

	// Fetch inventory
	logging.Debugf("[cache] Fetching inventory for %s", apiLabel)
	if invSvc := client.Inventory(); invSvc != nil {
		// APs
		fmt.Printf("    Fetching APs...")
		aps, err := invSvc.List(ctx, "ap")
		if err == nil {
			for _, item := range aps {
				if item.MAC != "" {
					cache.Inventory.AP[NormalizeMAC(item.MAC)] = item
				}
			}
			fmt.Printf(" %d devices\n", len(aps))
			logging.Debugf("[cache] Fetched %d APs for %s", len(aps), apiLabel)
		} else {
			fmt.Printf(" error\n")
			logging.Debugf("[cache] Failed to fetch APs for %s: %v", apiLabel, err)
		}

		// Switches
		fmt.Printf("    Fetching switches...")
		switches, err := invSvc.List(ctx, "switch")
		if err == nil {
			for _, item := range switches {
				if item.MAC != "" {
					cache.Inventory.Switch[NormalizeMAC(item.MAC)] = item
				}
			}
			fmt.Printf(" %d devices\n", len(switches))
			logging.Debugf("[cache] Fetched %d switches for %s", len(switches), apiLabel)
		} else {
			fmt.Printf(" error\n")
			logging.Debugf("[cache] Failed to fetch switches for %s: %v", apiLabel, err)
		}

		// Gateways
		fmt.Printf("    Fetching gateways...")
		gateways, err := invSvc.List(ctx, "gateway")
		if err == nil {
			for _, item := range gateways {
				if item.MAC != "" {
					cache.Inventory.Gateway[NormalizeMAC(item.MAC)] = item
				}
			}
			fmt.Printf(" %d devices\n", len(gateways))
			logging.Debugf("[cache] Fetched %d gateways for %s", len(gateways), apiLabel)
		} else {
			fmt.Printf(" error\n")
			logging.Debugf("[cache] Failed to fetch gateways for %s: %v", apiLabel, err)
		}
	}

	// Fetch device statuses
	fmt.Printf("    Fetching device statuses...")
	logging.Debugf("[cache] Fetching device statuses for %s", apiLabel)
	if statusSvc := client.Statuses(); statusSvc != nil {
		statuses, err := statusSvc.GetAll(ctx)
		if err == nil {
			cache.DeviceStatus = statuses
			fmt.Printf(" %d statuses\n", len(statuses))
			logging.Debugf("[cache] Fetched status for %d devices for %s", len(statuses), apiLabel)
		} else {
			fmt.Printf(" error\n")
			logging.Debugf("[cache] Failed to fetch device statuses for %s: %v", apiLabel, err)
		}
	} else {
		fmt.Printf(" not supported\n")
	}

	// Fetch templates (if supported)
	if tmplSvc := client.Templates(); tmplSvc != nil {
		fmt.Printf("    Fetching templates...")
		rfCount, gwCount, wlanCount := 0, 0, 0
		if rf, err := tmplSvc.ListRF(ctx); err == nil {
			for _, t := range rf {
				cache.Templates.RF = append(cache.Templates.RF, *t)
				rfCount++
			}
		} else {
			logging.Warnf("[cache] Failed to fetch RF templates: %v", err)
		}
		if gw, err := tmplSvc.ListGateway(ctx); err == nil {
			for _, t := range gw {
				cache.Templates.Gateway = append(cache.Templates.Gateway, *t)
				gwCount++
			}
		} else {
			logging.Warnf("[cache] Failed to fetch Gateway templates: %v", err)
		}
		if wlan, err := tmplSvc.ListWLAN(ctx); err == nil {
			for _, t := range wlan {
				cache.Templates.WLAN = append(cache.Templates.WLAN, *t)
				wlanCount++
			}
		} else {
			logging.Warnf("[cache] Failed to fetch WLAN templates: %v", err)
		}
		fmt.Printf(" %d RF, %d GW, %d WLAN\n", rfCount, gwCount, wlanCount)
	}

	// Fetch profiles (if supported)
	if profSvc := client.Profiles(); profSvc != nil {
		fmt.Printf("    Fetching device profiles...")
		if profiles, err := profSvc.List(ctx, ""); err == nil {
			for _, p := range profiles {
				cache.Profiles.Devices = append(cache.Profiles.Devices, *p)
			}
			fmt.Printf(" %d profiles\n", len(profiles))
		} else {
			fmt.Printf(" error\n")
		}
	}

	// Fetch WLANs (if supported)
	if wlanSvc := client.WLANs(); wlanSvc != nil {
		fmt.Printf("    Fetching WLANs...")
		if wlans, err := wlanSvc.List(ctx); err == nil {
			// Initialize map if needed
			if cache.WLANs == nil {
				cache.WLANs = make(map[string]*WLAN)
			}
			for _, w := range wlans {
				cache.WLANs[w.ID] = w
			}
			fmt.Printf(" %d WLANs\n", len(wlans))
		} else {
			fmt.Printf(" error: %v\n", err)
			logging.Warnf("[cache] Failed to fetch WLANs: %v", err)
		}
	}

	// Fetch device configs (if supported and enabled)
	if shouldFetchConfigs {
		if cfgSvc := client.Configs(); cfgSvc != nil {
			logging.Debugf("[cache] Fetching device configs for %s", apiLabel)

			// Fetch AP configs
			if len(cache.Inventory.AP) > 0 {
				fmt.Printf("    Fetching AP configs...")
				apConfigCount := 0
				for mac, item := range cache.Inventory.AP {
					if item.ID != "" && item.SiteID != "" {
						cfg, err := cfgSvc.GetAPConfig(ctx, item.SiteID, item.ID)
						if err == nil && cfg != nil {
							cache.Configs.AP[mac] = cfg
							apConfigCount++
						}
					}
				}
				fmt.Printf(" %d configs\n", apConfigCount)
				logging.Debugf("[cache] Fetched %d AP configs for %s", apConfigCount, apiLabel)
			}

			// Fetch Switch configs
			if len(cache.Inventory.Switch) > 0 {
				fmt.Printf("    Fetching switch configs...")
				switchConfigCount := 0
				for mac, item := range cache.Inventory.Switch {
					if item.ID != "" && item.SiteID != "" {
						cfg, err := cfgSvc.GetSwitchConfig(ctx, item.SiteID, item.ID)
						if err == nil && cfg != nil {
							cache.Configs.Switch[mac] = cfg
							switchConfigCount++
						}
					}
				}
				fmt.Printf(" %d configs\n", switchConfigCount)
				logging.Debugf("[cache] Fetched %d switch configs for %s", switchConfigCount, apiLabel)
			}

			// Fetch Gateway configs
			if len(cache.Inventory.Gateway) > 0 {
				fmt.Printf("    Fetching gateway configs...")
				gatewayConfigCount := 0
				for mac, item := range cache.Inventory.Gateway {
					if item.ID != "" && item.SiteID != "" {
						cfg, err := cfgSvc.GetGatewayConfig(ctx, item.SiteID, item.ID)
						if err == nil && cfg != nil {
							cache.Configs.Gateway[mac] = cfg
							gatewayConfigCount++
						}
					}
				}
				fmt.Printf(" %d configs\n", gatewayConfigCount)
				logging.Debugf("[cache] Fetched %d gateway configs for %s", gatewayConfigCount, apiLabel)
			}
		}
	} else {
		fmt.Printf("    Skipping device configs (use 'refresh cache' to fetch)\n")
		logging.Debugf("[cache] Skipping device config fetch for %s (Meraki optimization)", apiLabel)
	}

	cache.Meta.RefreshDurationMs = time.Since(startTime).Milliseconds()

	fmt.Printf("  [%s] Complete in %dms\n", apiLabel, cache.Meta.RefreshDurationMs)
	logging.Debugf("[cache] Refresh complete for %s in %dms", apiLabel, cache.Meta.RefreshDurationMs)

	// Save cache
	if err := c.SaveAPICache(cache); err != nil {
		logging.Debugf("[cache] Failed to save cache for %s: %v", apiLabel, err)
		return err
	}

	logging.Debugf("[cache] Saved cache for %s", apiLabel)

	// Rebuild cross-API index
	return c.RebuildIndex()
}

// RefreshAllAPIs refreshes all API caches in parallel.
func (c *CacheManager) RefreshAllAPIs(ctx context.Context) map[string]error {
	labels := c.registry.GetAllLabels()

	var wg sync.WaitGroup
	errors := make(map[string]error)
	var mu sync.Mutex

	for _, label := range labels {
		wg.Add(1)
		go func(apiLabel string) {
			defer wg.Done()
			if err := c.RefreshAPI(ctx, apiLabel); err != nil {
				mu.Lock()
				errors[apiLabel] = err
				mu.Unlock()
			}
		}(label)
	}

	wg.Wait()

	// Rebuild index even if some APIs failed
	if err := c.RebuildIndex(); err != nil {
		errors["_index"] = err
	}

	return errors
}

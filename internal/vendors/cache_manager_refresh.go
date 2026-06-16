package vendors

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ravinald/wifimgr/internal/encryption"
	"github.com/ravinald/wifimgr/internal/logging"
)

// wlanHasPlaintextSecret reports whether a freshly fetched WLAN carries a PSK or
// RADIUS secret that still needs encrypting before it can be cached.
func wlanHasPlaintextSecret(w *WLAN) bool {
	if w.PSK != "" && !encryption.IsEncrypted(w.PSK) {
		return true
	}
	for _, rs := range w.RadiusServers {
		if rs.Secret != "" && !encryption.IsEncrypted(rs.Secret) {
			return true
		}
	}
	return false
}

// encryptWLANSecrets replaces a WLAN's plaintext PSK and RADIUS secrets with
// enc: ciphertext so secrets never reach the cache file in the clear. Empty or
// already-encrypted values pass through untouched.
func encryptWLANSecrets(w *WLAN, password string) error {
	enc := func(v string) (string, error) {
		if v == "" || encryption.IsEncrypted(v) {
			return v, nil
		}
		return encryption.Encrypt(v, password)
	}
	var err error
	if w.PSK, err = enc(w.PSK); err != nil {
		return err
	}
	for i := range w.RadiusServers {
		if w.RadiusServers[i].Secret, err = enc(w.RadiusServers[i].Secret); err != nil {
			return err
		}
	}
	return nil
}

// skipDeviceConfig reports whether a device's per-device config fetch should be
// skipped (and its prior config carried forward) for this refresh. A device is
// skipped when it falls outside the site scope, or — under a managed refresh —
// when its MAC is not in the armed allowlist.
func skipDeviceConfig(opts RefreshOptions, siteID, mac string) bool {
	if opts.SiteID != "" && siteID != opts.SiteID {
		return true
	}
	if opts.ManagedMACs != nil && !opts.ManagedMACs[mac] {
		return true
	}
	return false
}

// RefreshAPI refreshes a single API's cache using its client.
// This is a convenience wrapper that calls RefreshAPIWithOptions with FetchDeviceConfigs=true.
func (c *CacheManager) RefreshAPI(ctx context.Context, apiLabel string) error {
	return c.RefreshAPIWithOptions(ctx, apiLabel, RefreshOptions{FetchDeviceConfigs: true})
}

// RefreshAPIManaged refreshes a single API but limits per-device config fetches
// to the armed (managed) MACs. Org-scoped data is still refreshed in full;
// configs for unmanaged devices are carried forward from the existing cache.
func (c *CacheManager) RefreshAPIManaged(ctx context.Context, apiLabel string, managed map[string]bool) error {
	return c.RefreshAPIWithOptions(ctx, apiLabel, RefreshOptions{
		FetchDeviceConfigs: true,
		ManagedMACs:        managed,
	})
}

// RefreshAPISite refreshes a single API's cache but narrows the per-device
// config fetches to devices in the named site. Org-scoped data is still
// refreshed; per-device configs for other sites are copied forward from the
// existing cache. managed, when non-nil, further limits the fetch to armed
// MACs within the site.
func (c *CacheManager) RefreshAPISite(ctx context.Context, apiLabel, siteID string, managed map[string]bool) error {
	return c.RefreshAPIWithOptions(ctx, apiLabel, RefreshOptions{
		FetchDeviceConfigs: true,
		SiteID:             siteID,
		ManagedMACs:        managed,
	})
}

// RefreshAPIWithOptions refreshes a single API's cache with configurable options.
//
// The per-label mutex is held for the entire refresh so that a concurrent
// save (or a concurrent refresh for the same apiLabel) cannot interleave
// and silently clobber results. Combined with WriteFileAtomic, this makes
// in-process concurrent refreshes safe; cross-process concurrency relies
// on the atomic rename for last-writer-wins semantics.
//
// A failed refresh is recorded onto the prior cache's meta (LastFailure +
// LastError) so status and the cache footer can show it; the last successful
// LastRefresh is left intact. Without this, a hard failure returns before any
// save and the failure leaves no trace the UI can read.
func (c *CacheManager) RefreshAPIWithOptions(ctx context.Context, apiLabel string, opts RefreshOptions) error {
	lock := c.labelLock(apiLabel)
	lock.Lock()
	defer lock.Unlock()

	err := c.doRefreshAPI(ctx, apiLabel, opts)
	if err != nil {
		c.recordRefreshFailureLocked(apiLabel, err)
	}
	return err
}

// doRefreshAPI performs the refresh. The caller holds the per-label lock.
func (c *CacheManager) doRefreshAPI(ctx context.Context, apiLabel string, opts RefreshOptions) error {
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

	logging.Debugf("[cache] Refreshing %s (vendor=%s, org=%s, fetchConfigs=%v, siteID=%q)", apiLabel, config.Vendor, config.Credentials["org_id"], shouldFetchConfigs, opts.SiteID)

	// Progress message: Starting API refresh
	if opts.SiteID != "" {
		fmt.Printf("  [%s] Refreshing %s API (site %s)...\n", apiLabel, config.Vendor, opts.SiteID)
	} else {
		fmt.Printf("  [%s] Refreshing %s API...\n", apiLabel, config.Vendor)
	}

	startTime := time.Now()

	// When a site filter is in play, load the existing cache so we can carry
	// forward per-device configs for the sites we're not touching this pass.
	// Best effort — a missing cache (first refresh ever) just means there's
	// nothing to preserve, and the new cache will only have the target site's
	// configs populated until a full refresh fills in the rest.
	var existingCache *APICache
	if opts.SiteID != "" || opts.ManagedMACs != nil {
		if prior, err := c.GetAPICache(apiLabel); err == nil {
			existingCache = prior
		} else {
			logging.Debugf("[cache] No prior cache for %s to merge from: %v", apiLabel, err)
		}
	}

	// Create new cache
	cache := NewAPICache(apiLabel, config.Vendor, config.Credentials["org_id"])
	cache.Meta.LastRefresh = startTime
	// This is the success path, and startTime is newer than any prior failure, so
	// the failure is stale: clear both LastError and LastFailure. Leaving them
	// (zero by construction in NewAPICache) means status reads as healthy with no
	// lingering failure once a success supersedes it.
	cache.Meta.LastError = ""
	cache.Meta.LastFailure = time.Time{}

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

	// Fetch BSSIDs (if supported)
	if bssidSvc := client.BSSIDs(); bssidSvc != nil {
		fmt.Printf("    Fetching BSSIDs...")
		logging.Debugf("[cache] Fetching BSSIDs for %s", apiLabel)
		entries, err := bssidSvc.List(ctx)
		if err == nil {
			if cache.BSSIDs == nil {
				cache.BSSIDs = make(map[string]*BSSIDEntry)
			}
			// Build serial-to-MAC lookup from inventory for populating APMAC
			serialToMAC := make(map[string]string)
			for mac, item := range cache.Inventory.AP {
				if item.Serial != "" {
					serialToMAC[item.Serial] = mac
				}
			}
			for _, entry := range entries {
				// Populate APMAC from serial if not already set
				if entry.APMAC == "" && entry.APSerial != "" {
					if mac, ok := serialToMAC[entry.APSerial]; ok {
						entry.APMAC = NormalizeMAC(mac)
					}
				}
				cache.BSSIDs[NormalizeMAC(entry.BSSID)] = entry
			}
			fmt.Printf(" %d BSSIDs\n", len(entries))
			logging.Debugf("[cache] Fetched %d BSSIDs for %s", len(entries), apiLabel)
		} else {
			fmt.Printf(" error\n")
			logging.Warnf("[cache] Failed to fetch BSSIDs for %s: %v", apiLabel, err)
		}
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
			// Secrets must never hit the cache file in the clear. Resolve the
			// password once, and only when a WLAN actually carries a secret, so
			// secret-free refreshes never prompt.
			var pw string
			for _, w := range wlans {
				if wlanHasPlaintextSecret(w) {
					pw, err = c.secretPassword()
					if err != nil {
						return fmt.Errorf("cache: resolve secret password: %w", err)
					}
					break
				}
			}
			for _, w := range wlans {
				if pw != "" {
					if encErr := encryptWLANSecrets(w, pw); encErr != nil {
						return fmt.Errorf("cache: encrypt WLAN %s secrets: %w", w.ID, encErr)
					}
				}
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
				if opts.SiteID != "" {
					fmt.Printf("    Fetching AP configs (site %s)...", opts.SiteID)
				} else {
					fmt.Printf("    Fetching AP configs...")
				}
				apConfigCount, apCarriedCount := 0, 0
				for mac, item := range cache.Inventory.AP {
					if item.ID == "" || item.SiteID == "" {
						continue
					}
					if skipDeviceConfig(opts, item.SiteID, mac) {
						if existingCache != nil {
							if old, ok := existingCache.Configs.AP[mac]; ok && old != nil {
								cache.Configs.AP[mac] = old
								apCarriedCount++
							}
						}
						continue
					}
					cfg, err := cfgSvc.GetAPConfig(ctx, item.SiteID, item.ID)
					if err == nil && cfg != nil {
						cache.Configs.AP[mac] = cfg
						apConfigCount++
					}
				}
				if apCarriedCount > 0 {
					fmt.Printf(" %d fetched, %d preserved\n", apConfigCount, apCarriedCount)
				} else {
					fmt.Printf(" %d configs\n", apConfigCount)
				}
				logging.Debugf("[cache] Fetched %d AP configs (preserved %d) for %s", apConfigCount, apCarriedCount, apiLabel)
			}

			// Fetch Switch configs
			if len(cache.Inventory.Switch) > 0 {
				if opts.SiteID != "" {
					fmt.Printf("    Fetching switch configs (site %s)...", opts.SiteID)
				} else {
					fmt.Printf("    Fetching switch configs...")
				}
				switchConfigCount, switchCarriedCount := 0, 0
				for mac, item := range cache.Inventory.Switch {
					if item.ID == "" || item.SiteID == "" {
						continue
					}
					if skipDeviceConfig(opts, item.SiteID, mac) {
						if existingCache != nil {
							if old, ok := existingCache.Configs.Switch[mac]; ok && old != nil {
								cache.Configs.Switch[mac] = old
								switchCarriedCount++
							}
						}
						continue
					}
					cfg, err := cfgSvc.GetSwitchConfig(ctx, item.SiteID, item.ID)
					if err == nil && cfg != nil {
						cache.Configs.Switch[mac] = cfg
						switchConfigCount++
					}
				}
				if switchCarriedCount > 0 {
					fmt.Printf(" %d fetched, %d preserved\n", switchConfigCount, switchCarriedCount)
				} else {
					fmt.Printf(" %d configs\n", switchConfigCount)
				}
				logging.Debugf("[cache] Fetched %d switch configs (preserved %d) for %s", switchConfigCount, switchCarriedCount, apiLabel)
			}

			// Fetch Gateway configs
			if len(cache.Inventory.Gateway) > 0 {
				if opts.SiteID != "" {
					fmt.Printf("    Fetching gateway configs (site %s)...", opts.SiteID)
				} else {
					fmt.Printf("    Fetching gateway configs...")
				}
				gatewayConfigCount, gatewayCarriedCount := 0, 0
				for mac, item := range cache.Inventory.Gateway {
					if item.ID == "" || item.SiteID == "" {
						continue
					}
					if skipDeviceConfig(opts, item.SiteID, mac) {
						if existingCache != nil {
							if old, ok := existingCache.Configs.Gateway[mac]; ok && old != nil {
								cache.Configs.Gateway[mac] = old
								gatewayCarriedCount++
							}
						}
						continue
					}
					cfg, err := cfgSvc.GetGatewayConfig(ctx, item.SiteID, item.ID)
					if err == nil && cfg != nil {
						cache.Configs.Gateway[mac] = cfg
						gatewayConfigCount++
					}
				}
				if gatewayCarriedCount > 0 {
					fmt.Printf(" %d fetched, %d preserved\n", gatewayConfigCount, gatewayCarriedCount)
				} else {
					fmt.Printf(" %d configs\n", gatewayConfigCount)
				}
				logging.Debugf("[cache] Fetched %d gateway configs (preserved %d) for %s", gatewayConfigCount, gatewayCarriedCount, apiLabel)
			}
		}
	} else {
		fmt.Printf("    Skipping device configs (use 'refresh cache' to fetch)\n")
		logging.Debugf("[cache] Skipping device config fetch for %s (Meraki optimization)", apiLabel)
	}

	cache.Meta.RefreshDurationMs = time.Since(startTime).Milliseconds()

	// Stamp per-object freshness: objects fetched this pass get startTime; configs
	// carried forward from the prior cache keep their original (older) timestamp.
	cache.StampFreshObjects(startTime)

	fmt.Printf("  [%s] Complete in %dms\n", apiLabel, cache.Meta.RefreshDurationMs)
	logging.Debugf("[cache] Refresh complete for %s in %dms", apiLabel, cache.Meta.RefreshDurationMs)

	// Save cache. Use the locked variant because this function already holds
	// the per-label mutex — calling SaveAPICache would deadlock.
	if err := c.saveAPICacheLocked(cache); err != nil {
		logging.Debugf("[cache] Failed to save cache for %s: %v", apiLabel, err)
		return err
	}

	logging.Debugf("[cache] Saved cache for %s", apiLabel)

	// Rebuild cross-API index
	return c.RebuildIndex()
}

// RefreshAllAPIs refreshes all API caches in parallel.
func (c *CacheManager) RefreshAllAPIs(ctx context.Context) map[string]error {
	return c.refreshAllAPIs(ctx, func(string) RefreshOptions {
		return RefreshOptions{FetchDeviceConfigs: true}
	})
}

// RefreshAllAPIsManaged refreshes every API in parallel, limiting per-device
// config fetches to the armed (managed) MACs. The same set is applied to each
// API; MACs not present in a given API's inventory simply never match.
func (c *CacheManager) RefreshAllAPIsManaged(ctx context.Context, managed map[string]bool) map[string]error {
	return c.refreshAllAPIs(ctx, func(string) RefreshOptions {
		return RefreshOptions{FetchDeviceConfigs: true, ManagedMACs: managed}
	})
}

func (c *CacheManager) refreshAllAPIs(ctx context.Context, optsFor func(apiLabel string) RefreshOptions) map[string]error {
	labels := c.registry.GetAllLabels()

	var wg sync.WaitGroup
	errors := make(map[string]error)
	var mu sync.Mutex

	for _, label := range labels {
		wg.Add(1)
		go func(apiLabel string) {
			defer wg.Done()
			if err := c.RefreshAPIWithOptions(ctx, apiLabel, optsFor(apiLabel)); err != nil {
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

// recordRefreshFailureLocked stamps a failed refresh onto the API's existing
// cache so status and the cache footer can surface it; LastRefresh (the last
// success) is left untouched. The caller holds the per-label lock. A first-ever
// refresh that fails has no prior cache to stamp — that API simply isn't shown
// yet, the pre-existing behavior.
func (c *CacheManager) recordRefreshFailureLocked(apiLabel string, refreshErr error) {
	cache, err := c.GetAPICache(apiLabel)
	if err != nil {
		logging.Debugf("[cache] No prior cache for %s to record refresh failure: %v", apiLabel, err)
		return
	}
	cache.Meta.LastFailure = time.Now()
	cache.Meta.LastError = classifyRefreshError(refreshErr)
	if err := c.saveAPICacheLocked(cache); err != nil {
		logging.Warnf("[cache] Failed to record refresh failure for %s: %v", apiLabel, err)
	}
}

// classifyRefreshError reduces a refresh error to a short operator-facing label.
// Matching is on the (wrapped) message string so it survives the fmt.Errorf
// wrapping the refresh applies around vendor/transport errors.
func classifyRefreshError(refreshErr error) string {
	if refreshErr == nil {
		return ""
	}
	msg := strings.ToLower(refreshErr.Error())
	switch {
	case strings.Contains(msg, "deadline exceeded"),
		strings.Contains(msg, "timeout"),
		strings.Contains(msg, "timed out"),
		strings.Contains(msg, "connection refused"),
		strings.Contains(msg, "no such host"),
		strings.Contains(msg, "no route to host"),
		strings.Contains(msg, "network is unreachable"),
		strings.Contains(msg, "tls handshake"):
		return "connection failure"
	case strings.Contains(msg, "401"),
		strings.Contains(msg, "403"),
		strings.Contains(msg, "unauthorized"),
		strings.Contains(msg, "forbidden"),
		strings.Contains(msg, "invalid api key"),
		strings.Contains(msg, "authentication"):
		return "auth failure"
	default:
		return strings.TrimSpace(refreshErr.Error())
	}
}

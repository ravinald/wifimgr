package vendors

import (
	"context"
	"fmt"
	"time"
)

// RefreshDeviceConfigs re-fetches the running config for specific devices (keyed by
// device type → MACs) within a site and caches it with a fresh RefreshedAt. It is the
// read-back apply's verify step uses to capture ground truth for just the pushed
// devices — no org-scoped fetches, no Meta.LastRefresh change. Devices not found in
// inventory (no ID/site) are skipped.
func (c *CacheManager) RefreshDeviceConfigs(ctx context.Context, apiLabel string, macsByType map[string][]string) error {
	lock := c.labelLock(apiLabel)
	lock.Lock()
	defer lock.Unlock()

	client, err := c.registry.GetClient(apiLabel)
	if err != nil {
		return err
	}
	cfgSvc := client.Configs()
	if cfgSvc == nil {
		return fmt.Errorf("configs not supported by %s", apiLabel)
	}

	cache, err := c.GetAPICache(apiLabel)
	if err != nil {
		return err
	}

	now := time.Now()
	for deviceType, macs := range macsByType {
		for _, mac := range macs {
			nm := NormalizeMAC(mac)
			switch deviceType {
			case "ap":
				item := cache.Inventory.AP[nm]
				if item == nil || item.ID == "" || item.SiteID == "" {
					continue
				}
				cfg, err := cfgSvc.GetAPConfig(ctx, item.SiteID, item.ID)
				if err != nil {
					return fmt.Errorf("re-fetch ap %s: %w", mac, err)
				}
				if cfg != nil {
					cfg.RefreshedAt = now
					cache.Configs.AP[nm] = cfg
				}
			case "switch":
				item := cache.Inventory.Switch[nm]
				if item == nil || item.ID == "" || item.SiteID == "" {
					continue
				}
				cfg, err := cfgSvc.GetSwitchConfig(ctx, item.SiteID, item.ID)
				if err != nil {
					return fmt.Errorf("re-fetch switch %s: %w", mac, err)
				}
				if cfg != nil {
					cfg.RefreshedAt = now
					cache.Configs.Switch[nm] = cfg
				}
			case "gateway":
				item := cache.Inventory.Gateway[nm]
				if item == nil || item.ID == "" || item.SiteID == "" {
					continue
				}
				cfg, err := cfgSvc.GetGatewayConfig(ctx, item.SiteID, item.ID)
				if err != nil {
					return fmt.Errorf("re-fetch gateway %s: %w", mac, err)
				}
				if cfg != nil {
					cfg.RefreshedAt = now
					cache.Configs.Gateway[nm] = cfg
				}
			}
		}
	}

	if err := c.saveAPICacheLocked(cache); err != nil {
		return err
	}
	return c.RebuildIndex()
}

// SetDeviceApplyState records the apply outcome (applied_at + apply_state) on cached
// device config objects without re-fetching. Used for both verify-mode results
// (verified/divergent) and trust-mode (applied_unvalidated). Missing configs are
// skipped — trust mode on a never-cached device simply has nothing to annotate yet.
func (c *CacheManager) SetDeviceApplyState(apiLabel string, macsByType map[string][]string, appliedAt time.Time, state string) error {
	lock := c.labelLock(apiLabel)
	lock.Lock()
	defer lock.Unlock()

	cache, err := c.GetAPICache(apiLabel)
	if err != nil {
		return err
	}

	for deviceType, macs := range macsByType {
		for _, mac := range macs {
			nm := NormalizeMAC(mac)
			var om *ObjectMeta
			switch deviceType {
			case "ap":
				if cfg := cache.Configs.AP[nm]; cfg != nil {
					om = &cfg.ObjectMeta
				}
			case "switch":
				if cfg := cache.Configs.Switch[nm]; cfg != nil {
					om = &cfg.ObjectMeta
				}
			case "gateway":
				if cfg := cache.Configs.Gateway[nm]; cfg != nil {
					om = &cfg.ObjectMeta
				}
			}
			if om != nil {
				om.AppliedAt = appliedAt
				om.ApplyState = state
			}
		}
	}

	return c.saveAPICacheLocked(cache)
}

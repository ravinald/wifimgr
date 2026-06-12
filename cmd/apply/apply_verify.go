package apply

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/viper"

	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/symbols"
	"github.com/ravinald/wifimgr/internal/vendors"
)

const (
	verifyAttempts = 3
	verifyBackoff  = 2 * time.Second
)

// resolveApplyVerify reports whether apply should read the pushed config back and
// confirm it matches intent. Configurable per-API via api.<label>.apply_verify;
// defaults to true (the safe, conservative choice) when unset.
func resolveApplyVerify(apiLabel string) bool {
	key := fmt.Sprintf("api.%s.apply_verify", apiLabel)
	if viper.IsSet(key) {
		return viper.GetBool(key)
	}
	return true
}

// recordApplyOutcome records the outcome of a push for the devices that took (2xx).
// In verify mode it re-fetches the running config and compares it to intent (the
// managed-keys diff) with a bounded retry for async convergence, marking each device
// verified or divergent and caching ground truth. In trust mode it records
// applied_unvalidated without a read-back. Returns the MACs whose running config still
// diverges from intent, so the caller can fail the apply.
func recordApplyOutcome(ctx context.Context, client vendors.Client, updater DeviceUpdater, _ *config.Config, siteConfig SiteConfig, deviceType, siteID, apiLabel string, succeeded []string) ([]string, error) {
	accessor := vendors.GetGlobalCacheAccessor()
	if accessor == nil {
		return nil, fmt.Errorf("cache accessor not initialized")
	}
	now := time.Now()

	if !resolveApplyVerify(apiLabel) {
		if err := accessor.SetDeviceApplyState(apiLabel, map[string][]string{deviceType: succeeded}, now, vendors.ApplyStateAppliedUnvalidated); err != nil {
			logging.Warnf("failed to record apply state: %v", err)
		}
		fmt.Printf("%s %d %s(s) applied (unvalidated)\n", symbols.SuccessPrefix(), len(succeeded), deviceType)
		return nil, nil
	}

	managedKeys := getManagedKeysForDevice(apiLabel, deviceType)
	remaining := append([]string{}, succeeded...)
	for attempt := 0; attempt < verifyAttempts && len(remaining) > 0; attempt++ {
		if err := accessor.RefreshDeviceConfigs(ctx, apiLabel, map[string][]string{deviceType: remaining}); err != nil {
			return remaining, fmt.Errorf("verify re-fetch: %w", err)
		}
		batchLoader, err := NewDeviceBatchLoader(ctx, client, siteID, deviceType)
		if err != nil {
			return remaining, fmt.Errorf("verify loader: %w", err)
		}
		still := make([]string, 0, len(remaining))
		for _, mac := range remaining {
			if deviceStillDiverges(updater, batchLoader, siteConfig, mac, managedKeys) {
				still = append(still, mac)
			}
		}
		remaining = still
		if len(remaining) > 0 && attempt < verifyAttempts-1 {
			time.Sleep(verifyBackoff)
		}
	}

	verified := subtractMACs(succeeded, remaining)
	if len(verified) > 0 {
		if err := accessor.SetDeviceApplyState(apiLabel, map[string][]string{deviceType: verified}, now, vendors.ApplyStateVerified); err != nil {
			logging.Warnf("failed to record verified state: %v", err)
		}
	}
	if len(remaining) > 0 {
		if err := accessor.SetDeviceApplyState(apiLabel, map[string][]string{deviceType: remaining}, now, vendors.ApplyStateDivergent); err != nil {
			logging.Warnf("failed to record divergent state: %v", err)
		}
	}

	fmt.Printf("%s %d %s(s) verified", symbols.SuccessPrefix(), len(verified), deviceType)
	if len(remaining) > 0 {
		fmt.Printf("\n%s %d %s(s) divergent — running config does not match intent: %v", symbols.FailurePrefix(), len(remaining), deviceType, remaining)
	}
	fmt.Println()
	return remaining, nil
}

// deviceStillDiverges reports whether the re-fetched running config for mac still
// differs from intent across the managed keys — i.e. the push did not realize intent.
func deviceStillDiverges(updater DeviceUpdater, batchLoader *DeviceBatchLoader, siteConfig SiteConfig, mac string, managedKeys []string) bool {
	desired, ok := updater.GetDeviceConfigFromSite(siteConfig, mac)
	if !ok {
		return false
	}
	if expanded, err := expandDeviceConfigWithTemplates(desired, siteConfig); err == nil {
		desired = expanded
	}
	device, err := batchLoader.GetDeviceByMAC(mac)
	if err != nil {
		// Can't read the running config back — treat as unconfirmed (divergent).
		return true
	}
	return compareDeviceConfigsWithManagedKeys(device.ToConfigMap(), desired, managedKeys)
}

func subtractMACs(all, remove []string) []string {
	rm := make(map[string]bool, len(remove))
	for _, m := range remove {
		rm[m] = true
	}
	out := make([]string, 0, len(all))
	for _, m := range all {
		if !rm[m] {
			out = append(out, m)
		}
	}
	return out
}

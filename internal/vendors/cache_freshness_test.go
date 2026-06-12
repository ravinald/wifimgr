package vendors

import (
	"testing"
	"time"
)

// TestStampFreshObjectsPreservesCarryForward verifies that StampFreshObjects stamps
// only objects fetched this pass (zero RefreshedAt) and leaves carried-forward objects
// (already timestamped) untouched — the property that keeps per-object freshness honest
// across a site-scoped refresh.
func TestStampFreshObjectsPreservesCarryForward(t *testing.T) {
	old := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)

	c := NewAPICache("mist", "mist", "org1")
	// Fresh fetch this pass (zero RefreshedAt).
	c.Inventory.AP["aaaa"] = &InventoryItem{MAC: "aaaa"}
	c.Configs.AP["aaaa"] = &APConfig{MAC: "aaaa"}
	c.Sites.Info = append(c.Sites.Info, SiteInfo{ID: "s1"})
	// Carried forward from a prior cache (already stamped).
	c.Configs.AP["bbbb"] = &APConfig{MAC: "bbbb", ObjectMeta: ObjectMeta{RefreshedAt: old}}

	c.StampFreshObjects(now)

	if got := c.Inventory.AP["aaaa"].RefreshedAt; !got.Equal(now) {
		t.Errorf("fresh inventory item RefreshedAt = %v, want %v", got, now)
	}
	if got := c.Configs.AP["aaaa"].RefreshedAt; !got.Equal(now) {
		t.Errorf("fresh AP config RefreshedAt = %v, want %v", got, now)
	}
	if got := c.Sites.Info[0].RefreshedAt; !got.Equal(now) {
		t.Errorf("fresh site RefreshedAt = %v, want %v", got, now)
	}
	if got := c.Configs.AP["bbbb"].RefreshedAt; !got.Equal(old) {
		t.Errorf("carried-forward AP config RefreshedAt = %v, want preserved %v", got, old)
	}
}

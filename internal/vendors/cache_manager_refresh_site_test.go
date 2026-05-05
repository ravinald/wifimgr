/*
Copyright © 2025 Ravi Pina <ravi@pina.org>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/
package vendors

import (
	"context"
	"testing"
)

// TestRefreshAPISite verifies that scoping a refresh to a single site:
//   - fetches configs for devices in the named site (overwriting any prior config)
//   - preserves prior configs for devices in *other* sites (no API call attempted,
//     existing cache entry copied forward)
//   - still refreshes org-scoped data (sites, inventory, statuses)
func TestRefreshAPISite(t *testing.T) {
	tmpDir := t.TempDir()

	// Inventory spanning two sites: one AP per site.
	customInventory := &MockInventoryService{
		Items: []*InventoryItem{
			{ID: "dev-s1", MAC: "aabbccddee01", Serial: "AP-S1", Model: "AP43", Name: "AP-Site1", Type: "ap", SiteID: "site-001"},
			{ID: "dev-s2", MAC: "aabbccddee02", Serial: "AP-S2", Model: "AP43", Name: "AP-Site2", Type: "ap", SiteID: "site-002"},
		},
	}
	customInventory.itemsByMAC = map[string]*InventoryItem{}
	customInventory.bySerial = map[string]*InventoryItem{}
	for _, it := range customInventory.Items {
		customInventory.itemsByMAC[it.MAC] = it
		customInventory.bySerial[it.Serial] = it
	}

	registry := NewAPIClientRegistry()
	registry.RegisterFactory("mock", func(config *APIConfig) (Client, error) {
		mc := NewMockClientWithAllServices(config.Vendor, config.Credentials["org_id"])
		mc.SetInventoryService(customInventory)
		return mc, nil
	})

	configs := map[string]*APIConfig{
		"test-api": {
			Label:       "test-api",
			Vendor:      "mock",
			Credentials: map[string]string{"org_id": "org-123"},
		},
	}
	registry.InitializeClients(configs)

	cm := NewCacheManager(tmpDir, registry)
	if err := cm.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	ctx := context.Background()

	// First, do a full refresh to populate the cache with both sites' AP configs.
	if err := cm.RefreshAPI(ctx, "test-api"); err != nil {
		t.Fatalf("initial RefreshAPI: %v", err)
	}

	// Mutate the on-disk cache so site-002's AP config has a sentinel marker.
	// After a per-site refresh of site-001, the site-002 entry must still
	// carry this marker — proving we copied the prior config forward instead
	// of dropping it.
	cache, err := cm.GetAPICache("test-api")
	if err != nil {
		t.Fatalf("GetAPICache: %v", err)
	}
	site2APMAC := "aabbccddee02"
	site2Cfg, ok := cache.Configs.AP[site2APMAC]
	if !ok || site2Cfg == nil {
		t.Fatalf("expected site-002 AP config in cache after full refresh, got %v", cache.Configs.AP)
	}
	if site2Cfg.Config == nil {
		site2Cfg.Config = map[string]any{}
	}
	site2Cfg.Config["sentinel"] = "preserve-me"
	if err := cm.SaveAPICache(cache); err != nil {
		t.Fatalf("SaveAPICache: %v", err)
	}

	// Now scope-refresh to site-001 only.
	if err := cm.RefreshAPISite(ctx, "test-api", "site-001"); err != nil {
		t.Fatalf("RefreshAPISite: %v", err)
	}

	// Read back and verify both sites' configs are still in the cache, and
	// site-002's sentinel survived.
	got, err := cm.GetAPICache("test-api")
	if err != nil {
		t.Fatalf("GetAPICache after scoped refresh: %v", err)
	}

	if len(got.Sites.Info) != 3 {
		t.Errorf("org-scoped sites refresh missing: got %d sites, want 3", len(got.Sites.Info))
	}

	site1AP, ok := got.Configs.AP["aabbccddee01"]
	if !ok || site1AP == nil {
		t.Fatalf("site-001 AP config missing after scoped refresh")
	}
	if _, has := site1AP.Config["sentinel"]; has {
		t.Errorf("site-001 AP config carries sentinel — should have been overwritten by fresh fetch")
	}

	site2AP, ok := got.Configs.AP[site2APMAC]
	if !ok || site2AP == nil {
		t.Fatalf("site-002 AP config dropped by scoped refresh — should have been preserved")
	}
	if site2AP.Config["sentinel"] != "preserve-me" {
		t.Errorf("site-002 AP sentinel lost: Config=%v", site2AP.Config)
	}
}

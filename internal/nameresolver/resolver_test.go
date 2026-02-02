package nameresolver

import (
	"testing"

	"github.com/ravinald/wifimgr/internal/vendors"
)

func TestNewResolver(t *testing.T) {
	r := NewResolver()
	if r == nil {
		t.Fatal("NewResolver returned nil")
	}
	if r.deviceProfiles == nil {
		t.Error("deviceProfiles map not initialized")
	}
	if r.rfProfiles == nil {
		t.Error("rfProfiles map not initialized")
	}
	if r.siteMaps == nil {
		t.Error("siteMaps map not initialized")
	}
}

func TestResolveDeviceProfile(t *testing.T) {
	r := NewResolver()

	// Load test profiles
	profiles := []*vendors.DeviceProfile{
		{ID: "prof-001", Name: "US-Office-Macro-DFS"},
		{ID: "prof-002", Name: "US-Office-Standard"},
		{ID: "prof-003", Name: "EU-Office-Indoor"},
	}
	r.LoadDeviceProfiles("mist-prod", profiles)

	t.Run("resolve existing profile", func(t *testing.T) {
		id, err := r.ResolveDeviceProfile("mist-prod", "US-Office-Macro-DFS")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if id != "prof-001" {
			t.Errorf("got %q, want %q", id, "prof-001")
		}
	})

	t.Run("resolve missing profile", func(t *testing.T) {
		_, err := r.ResolveDeviceProfile("mist-prod", "NonExistent")
		if err == nil {
			t.Error("expected error for missing profile")
		}
		if _, ok := err.(*ResolutionError); !ok {
			t.Errorf("expected ResolutionError, got %T", err)
		}
	})

	t.Run("resolve from unknown API", func(t *testing.T) {
		_, err := r.ResolveDeviceProfile("unknown-api", "US-Office-Macro-DFS")
		if err == nil {
			t.Error("expected error for unknown API")
		}
	})
}

func TestResolveRFProfile(t *testing.T) {
	r := NewResolver()

	// Load test RF profiles
	rfProfiles := map[string]string{
		"High-Density": "rf-001",
		"Low-Density":  "rf-002",
		"Default":      "rf-003",
	}
	r.LoadRFProfiles("meraki-prod", rfProfiles)

	t.Run("resolve existing RF profile", func(t *testing.T) {
		id, err := r.ResolveRFProfile("meraki-prod", "High-Density")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if id != "rf-001" {
			t.Errorf("got %q, want %q", id, "rf-001")
		}
	})

	t.Run("resolve missing RF profile", func(t *testing.T) {
		_, err := r.ResolveRFProfile("meraki-prod", "NonExistent")
		if err == nil {
			t.Error("expected error for missing RF profile")
		}
	})
}

func TestResolveMap(t *testing.T) {
	r := NewResolver()

	// Load test maps
	maps := map[string]string{
		"Building-A-Floor-1": "map-001",
		"Building-A-Floor-2": "map-002",
		"Building-B-Lobby":   "map-003",
	}
	r.LoadMaps("site-123", maps)

	t.Run("resolve existing map", func(t *testing.T) {
		id, err := r.ResolveMap("site-123", "Building-A-Floor-1")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if id != "map-001" {
			t.Errorf("got %q, want %q", id, "map-001")
		}
	})

	t.Run("resolve missing map", func(t *testing.T) {
		_, err := r.ResolveMap("site-123", "NonExistent")
		if err == nil {
			t.Error("expected error for missing map")
		}
	})

	t.Run("resolve from unknown site", func(t *testing.T) {
		_, err := r.ResolveMap("unknown-site", "Building-A-Floor-1")
		if err == nil {
			t.Error("expected error for unknown site")
		}
	})
}

func TestResolveAPConfig(t *testing.T) {
	r := NewResolver()

	// Load test data
	profiles := []*vendors.DeviceProfile{
		{ID: "prof-001", Name: "Office-Profile"},
	}
	r.LoadDeviceProfiles("mist-prod", profiles)

	maps := map[string]string{
		"Floor-1": "map-001",
	}
	r.LoadMaps("site-123", maps)

	rfProfiles := map[string]string{
		"High-Density": "rf-001",
	}
	r.LoadRFProfiles("meraki-prod", rfProfiles)

	t.Run("resolve device profile name", func(t *testing.T) {
		cfg := &vendors.APDeviceConfig{
			Name:              "AP-01",
			DeviceProfileName: "Office-Profile",
		}
		err := r.ResolveAPConfig(cfg, "mist-prod", "site-123")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if cfg.DeviceProfileID != "prof-001" {
			t.Errorf("DeviceProfileID = %q, want %q", cfg.DeviceProfileID, "prof-001")
		}
		if cfg.DeviceProfileName != "" {
			t.Error("DeviceProfileName should be cleared after resolution")
		}
	})

	t.Run("resolve map name", func(t *testing.T) {
		cfg := &vendors.APDeviceConfig{
			Name:    "AP-01",
			MapName: "Floor-1",
		}
		err := r.ResolveAPConfig(cfg, "mist-prod", "site-123")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if cfg.MapID != "map-001" {
			t.Errorf("MapID = %q, want %q", cfg.MapID, "map-001")
		}
		if cfg.MapName != "" {
			t.Error("MapName should be cleared after resolution")
		}
	})

	t.Run("resolve meraki rf profile name", func(t *testing.T) {
		cfg := &vendors.APDeviceConfig{
			Name: "AP-01",
			RadioConfig: &vendors.RadioConfig{
				Meraki: map[string]interface{}{
					"rf_profile_name": "High-Density",
				},
			},
		}
		err := r.ResolveAPConfig(cfg, "meraki-prod", "site-123")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if cfg.RadioConfig.Meraki["rf_profile_id"] != "rf-001" {
			t.Errorf("rf_profile_id = %v, want rf-001", cfg.RadioConfig.Meraki["rf_profile_id"])
		}
		if _, exists := cfg.RadioConfig.Meraki["rf_profile_name"]; exists {
			t.Error("rf_profile_name should be deleted after resolution")
		}
	})

	t.Run("nil config is no-op", func(t *testing.T) {
		err := r.ResolveAPConfig(nil, "mist-prod", "site-123")
		if err != nil {
			t.Errorf("unexpected error for nil config: %v", err)
		}
	})

	t.Run("error on missing device profile", func(t *testing.T) {
		cfg := &vendors.APDeviceConfig{
			DeviceProfileName: "NonExistent",
		}
		err := r.ResolveAPConfig(cfg, "mist-prod", "site-123")
		if err == nil {
			t.Error("expected error for missing device profile")
		}
	})
}

func TestListFunctions(t *testing.T) {
	r := NewResolver()

	profiles := []*vendors.DeviceProfile{
		{ID: "prof-001", Name: "Profile-A"},
		{ID: "prof-002", Name: "Profile-B"},
	}
	r.LoadDeviceProfiles("mist-prod", profiles)

	rfProfiles := map[string]string{
		"RF-A": "rf-001",
		"RF-B": "rf-002",
	}
	r.LoadRFProfiles("meraki-prod", rfProfiles)

	maps := map[string]string{
		"Map-A": "map-001",
	}
	r.LoadMaps("site-123", maps)

	t.Run("list device profiles", func(t *testing.T) {
		names := r.ListDeviceProfiles("mist-prod")
		if len(names) != 2 {
			t.Errorf("expected 2 profiles, got %d", len(names))
		}
	})

	t.Run("list device profiles unknown api", func(t *testing.T) {
		names := r.ListDeviceProfiles("unknown")
		if names != nil {
			t.Errorf("expected nil for unknown API, got %v", names)
		}
	})

	t.Run("list rf profiles", func(t *testing.T) {
		names := r.ListRFProfiles("meraki-prod")
		if len(names) != 2 {
			t.Errorf("expected 2 RF profiles, got %d", len(names))
		}
	})

	t.Run("list maps", func(t *testing.T) {
		names := r.ListMaps("site-123")
		if len(names) != 1 {
			t.Errorf("expected 1 map, got %d", len(names))
		}
	})
}

func TestSuggestSimilar(t *testing.T) {
	r := NewResolver()

	profiles := []*vendors.DeviceProfile{
		{ID: "1", Name: "US-Office-Macro-DFS"},
		{ID: "2", Name: "US-Office-Standard"},
		{ID: "3", Name: "EU-Office-Indoor"},
		{ID: "4", Name: "US-Warehouse-Outdoor"},
	}
	r.LoadDeviceProfiles("mist-prod", profiles)

	t.Run("find similar by prefix", func(t *testing.T) {
		suggestions := r.SuggestSimilar("device profile", "US-Office", "mist-prod")
		if len(suggestions) == 0 {
			t.Error("expected suggestions for prefix match")
		}
		// Should find US-Office-* profiles
		found := false
		for _, s := range suggestions {
			if s == "US-Office-Macro-DFS" || s == "US-Office-Standard" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected US-Office profile in suggestions: %v", suggestions)
		}
	})

	t.Run("find similar by word", func(t *testing.T) {
		suggestions := r.SuggestSimilar("device profile", "Macro", "mist-prod")
		if len(suggestions) == 0 {
			t.Error("expected suggestions for word match")
		}
	})

	t.Run("no suggestions for unrelated", func(t *testing.T) {
		suggestions := r.SuggestSimilar("device profile", "XYZABC", "mist-prod")
		if len(suggestions) > 0 {
			t.Errorf("expected no suggestions, got %v", suggestions)
		}
	})
}

func TestResolutionError(t *testing.T) {
	t.Run("error with API", func(t *testing.T) {
		err := &ResolutionError{
			RefType: "device profile",
			Name:    "Missing",
			API:     "mist-prod",
		}
		expected := `device profile "Missing" not found for API "mist-prod"`
		if err.Error() != expected {
			t.Errorf("got %q, want %q", err.Error(), expected)
		}
	})

	t.Run("error with site", func(t *testing.T) {
		err := &ResolutionError{
			RefType: "map",
			Name:    "Floor-1",
			Site:    "US-OFFICE",
		}
		expected := `map "Floor-1" not found in site "US-OFFICE"`
		if err.Error() != expected {
			t.Errorf("got %q, want %q", err.Error(), expected)
		}
	})

	t.Run("error without context", func(t *testing.T) {
		err := &ResolutionError{
			RefType: "profile",
			Name:    "Test",
		}
		expected := `profile "Test" not found`
		if err.Error() != expected {
			t.Errorf("got %q, want %q", err.Error(), expected)
		}
	})
}

func TestFindSimilar(t *testing.T) {
	candidates := []string{
		"apple-pie",
		"apple-tart",
		"banana-split",
		"cherry-cake",
	}

	t.Run("prefix match", func(t *testing.T) {
		results := findSimilar("apple", candidates, 3)
		if len(results) != 2 {
			t.Errorf("expected 2 results, got %d: %v", len(results), results)
		}
	})

	t.Run("contains match", func(t *testing.T) {
		results := findSimilar("pie", candidates, 3)
		if len(results) != 1 {
			t.Errorf("expected 1 result, got %d: %v", len(results), results)
		}
	})

	t.Run("no match", func(t *testing.T) {
		results := findSimilar("mango", candidates, 3)
		if len(results) != 0 {
			t.Errorf("expected 0 results, got %d: %v", len(results), results)
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		results := findSimilar("APPLE", candidates, 3)
		if len(results) != 2 {
			t.Errorf("expected 2 results for case-insensitive match, got %d: %v", len(results), results)
		}
	})

	t.Run("max results limit", func(t *testing.T) {
		allFruits := []string{"fruit-1", "fruit-2", "fruit-3", "fruit-4", "fruit-5"}
		results := findSimilar("fruit", allFruits, 2)
		if len(results) > 2 {
			t.Errorf("expected max 2 results, got %d: %v", len(results), results)
		}
	})

	t.Run("empty candidates", func(t *testing.T) {
		results := findSimilar("test", nil, 3)
		if results != nil {
			t.Errorf("expected nil for empty candidates, got %v", results)
		}
	})
}

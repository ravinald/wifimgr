package apply

import (
	"sort"
	"testing"
)

func siteWithAPs(aps map[string]map[string]any) SiteConfig {
	var sc SiteConfig
	sc.Devices.APs = aps
	return sc
}

func TestGroupDevicesByAPI(t *testing.T) {
	t.Run("all inherit site default", func(t *testing.T) {
		sc := siteWithAPs(map[string]map[string]any{
			"5c:5b:35:8e:4c:f9": {"name": "AP-1"},
			"a8f7d982de1a":      {"name": "AP-2"},
		})
		groups, err := groupDevicesByAPI(sc, "ap", "mist")
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if len(groups) != 1 {
			t.Fatalf("want 1 group, got %d: %v", len(groups), groups)
		}
		got := groups["mist"]
		sort.Strings(got)
		want := []string{"5c5b358e4cf9", "a8f7d982de1a"}
		if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
			t.Errorf("mist group = %v, want %v (normalized)", got, want)
		}
	})

	t.Run("per-device override splits groups", func(t *testing.T) {
		sc := siteWithAPs(map[string]map[string]any{
			"5c5b358e4cf9": {"name": "AP-mist"},
			"d04dc6c8cb3a": {"name": "AP-aruba", "api": "aruba-pina"},
		})
		groups, err := groupDevicesByAPI(sc, "ap", "mist")
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if len(groups) != 2 {
			t.Fatalf("want 2 groups, got %d: %v", len(groups), groups)
		}
		if g := groups["mist"]; len(g) != 1 || g[0] != "5c5b358e4cf9" {
			t.Errorf("mist group = %v", g)
		}
		if g := groups["aruba-pina"]; len(g) != 1 || g[0] != "d04dc6c8cb3a" {
			t.Errorf("aruba-pina group = %v", g)
		}
	})

	t.Run("blank override falls back to default", func(t *testing.T) {
		sc := siteWithAPs(map[string]map[string]any{
			"5c5b358e4cf9": {"name": "AP-1", "api": "  "},
		})
		groups, _ := groupDevicesByAPI(sc, "ap", "mist")
		if len(groups) != 1 || len(groups["mist"]) != 1 {
			t.Errorf("blank api should inherit default: %v", groups)
		}
	})

	t.Run("override equal to default stays one group", func(t *testing.T) {
		sc := siteWithAPs(map[string]map[string]any{
			"5c5b358e4cf9": {"name": "AP-1", "api": "mist"},
			"a8f7d982de1a": {"name": "AP-2"},
		})
		groups, _ := groupDevicesByAPI(sc, "ap", "mist")
		if len(groups) != 1 || len(groups["mist"]) != 2 {
			t.Errorf("want single mist group of 2, got %v", groups)
		}
	})

	t.Run("invalid MAC dropped", func(t *testing.T) {
		sc := siteWithAPs(map[string]map[string]any{
			"not-a-mac":    {"name": "bad"},
			"5c5b358e4cf9": {"name": "good"},
		})
		groups, _ := groupDevicesByAPI(sc, "ap", "mist")
		if len(groups["mist"]) != 1 || groups["mist"][0] != "5c5b358e4cf9" {
			t.Errorf("invalid MAC should be dropped: %v", groups)
		}
	})

	t.Run("no devices yields empty", func(t *testing.T) {
		groups, err := groupDevicesByAPI(SiteConfig{}, "ap", "mist")
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if len(groups) != 0 {
			t.Errorf("want empty, got %v", groups)
		}
	})

	t.Run("unknown device type errors", func(t *testing.T) {
		if _, err := groupDevicesByAPI(SiteConfig{}, "router", "mist"); err == nil {
			t.Error("expected error for unknown device type")
		}
	})
}

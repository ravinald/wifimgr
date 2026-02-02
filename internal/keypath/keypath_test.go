package keypath

import (
	"reflect"
	"sort"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantSegs    []string
		wantWild    bool
		wantWildIdx int
	}{
		{
			name:        "empty string",
			input:       "",
			wantSegs:    nil,
			wantWild:    false,
			wantWildIdx: -1,
		},
		{
			name:        "single segment",
			input:       "name",
			wantSegs:    []string{"name"},
			wantWild:    false,
			wantWildIdx: -1,
		},
		{
			name:        "two segments",
			input:       "radio_config.band_24",
			wantSegs:    []string{"radio_config", "band_24"},
			wantWild:    false,
			wantWildIdx: -1,
		},
		{
			name:        "three segments",
			input:       "radio_config.band_24.power",
			wantSegs:    []string{"radio_config", "band_24", "power"},
			wantWild:    false,
			wantWildIdx: -1,
		},
		{
			name:        "wildcard in middle",
			input:       "port_config.*.vlan_id",
			wantSegs:    []string{"port_config", "*", "vlan_id"},
			wantWild:    true,
			wantWildIdx: 1,
		},
		{
			name:        "wildcard at start",
			input:       "*.enabled",
			wantSegs:    []string{"*", "enabled"},
			wantWild:    true,
			wantWildIdx: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Parse(tt.input)
			if !reflect.DeepEqual(got.Segments, tt.wantSegs) {
				t.Errorf("Parse(%q).Segments = %v, want %v", tt.input, got.Segments, tt.wantSegs)
			}
			if got.HasWildcard != tt.wantWild {
				t.Errorf("Parse(%q).HasWildcard = %v, want %v", tt.input, got.HasWildcard, tt.wantWild)
			}
			if got.WildcardIdx != tt.wantWildIdx {
				t.Errorf("Parse(%q).WildcardIdx = %d, want %d", tt.input, got.WildcardIdx, tt.wantWildIdx)
			}
			if got.Original != tt.input {
				t.Errorf("Parse(%q).Original = %q, want %q", tt.input, got.Original, tt.input)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid single", "name", false},
		{"valid nested", "radio_config.band_24", false},
		{"valid wildcard", "port_config.*.vlan_id", false},
		{"empty string", "", true},
		{"empty segment start", ".name", true},
		{"empty segment middle", "radio_config..band_24", true},
		{"empty segment end", "name.", true},
		{"wildcard at end", "port_config.*", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestKeyPathMethods(t *testing.T) {
	t.Run("Depth", func(t *testing.T) {
		tests := []struct {
			input string
			want  int
		}{
			{"name", 1},
			{"radio_config.band_24", 2},
			{"radio_config.band_24.power", 3},
		}
		for _, tt := range tests {
			if got := Parse(tt.input).Depth(); got != tt.want {
				t.Errorf("Parse(%q).Depth() = %d, want %d", tt.input, got, tt.want)
			}
		}
	})

	t.Run("IsNested", func(t *testing.T) {
		if Parse("name").IsNested() {
			t.Error("Parse(\"name\").IsNested() = true, want false")
		}
		if !Parse("radio_config.band_24").IsNested() {
			t.Error("Parse(\"radio_config.band_24\").IsNested() = false, want true")
		}
	})

	t.Run("First", func(t *testing.T) {
		if got := Parse("radio_config.band_24").First(); got != "radio_config" {
			t.Errorf("First() = %q, want %q", got, "radio_config")
		}
		if got := Parse("").First(); got != "" {
			t.Errorf("Parse(\"\").First() = %q, want empty", got)
		}
	})

	t.Run("Rest", func(t *testing.T) {
		rest := Parse("radio_config.band_24.power").Rest()
		if got := rest.String(); got != "band_24.power" {
			t.Errorf("Rest().String() = %q, want %q", got, "band_24.power")
		}

		rest = Parse("name").Rest()
		if rest.Depth() != 0 {
			t.Errorf("Parse(\"name\").Rest().Depth() = %d, want 0", rest.Depth())
		}
	})

	t.Run("String", func(t *testing.T) {
		kp := Parse("radio_config.band_24.power")
		if got := kp.String(); got != "radio_config.band_24.power" {
			t.Errorf("String() = %q, want %q", got, "radio_config.band_24.power")
		}
	})
}

func TestStartsWith(t *testing.T) {
	tests := []struct {
		path   string
		prefix string
		want   bool
	}{
		{"radio_config.band_24.power", "radio_config", true},
		{"radio_config.band_24.power", "radio_config.band_24", true},
		{"radio_config.band_24.power", "radio_config.band_24.power", true},
		{"radio_config.band_24.power", "radio_config.band_5", false},
		{"radio_config", "radio_config.band_24", false},
		{"name", "radio_config", false},
	}

	for _, tt := range tests {
		t.Run(tt.path+"_"+tt.prefix, func(t *testing.T) {
			kp := Parse(tt.path)
			prefix := Parse(tt.prefix)
			if got := kp.StartsWith(prefix); got != tt.want {
				t.Errorf("StartsWith() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatches(t *testing.T) {
	tests := []struct {
		path    string
		pattern string
		want    bool
	}{
		{"port_config.eth0.vlan_id", "port_config.*.vlan_id", true},
		{"port_config.eth1.vlan_id", "port_config.*.vlan_id", true},
		{"port_config.eth0.mode", "port_config.*.vlan_id", false},
		{"radio_config.band_24", "radio_config.band_24", true},
		{"radio_config.band_24", "radio_config.band_5", false},
	}

	for _, tt := range tests {
		t.Run(tt.path+"_"+tt.pattern, func(t *testing.T) {
			kp := Parse(tt.path)
			pattern := Parse(tt.pattern)
			if got := kp.Matches(pattern); got != tt.want {
				t.Errorf("Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetValueAtPath(t *testing.T) {
	data := map[string]interface{}{
		"name": "AP-01",
		"radio_config": map[string]interface{}{
			"band_24": map[string]interface{}{
				"power":   12,
				"channel": 6,
			},
			"band_5": map[string]interface{}{
				"power": 15,
			},
		},
		"tags": []interface{}{"lobby", "floor-1"},
	}

	tests := []struct {
		path  []string
		want  interface{}
		found bool
	}{
		{[]string{"name"}, "AP-01", true},
		{[]string{"radio_config", "band_24", "power"}, 12, true},
		{[]string{"radio_config", "band_24", "channel"}, 6, true},
		{[]string{"radio_config", "band_5", "power"}, 15, true},
		{[]string{"tags"}, []interface{}{"lobby", "floor-1"}, true},
		{[]string{"nonexistent"}, nil, false},
		{[]string{"radio_config", "band_6"}, nil, false},
		{[]string{"radio_config", "band_24", "nonexistent"}, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.path[len(tt.path)-1], func(t *testing.T) {
			got, found := GetValueAtPath(data, tt.path)
			if found != tt.found {
				t.Errorf("GetValueAtPath() found = %v, want %v", found, tt.found)
			}
			if found && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetValueAtPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetValueAtPath(t *testing.T) {
	t.Run("set simple value", func(t *testing.T) {
		data := map[string]interface{}{}
		SetValueAtPath(data, []string{"name"}, "AP-01")
		if data["name"] != "AP-01" {
			t.Errorf("name = %v, want %v", data["name"], "AP-01")
		}
	})

	t.Run("set nested value creates intermediate maps", func(t *testing.T) {
		data := map[string]interface{}{}
		SetValueAtPath(data, []string{"radio_config", "band_24", "power"}, 12)

		rc, ok := data["radio_config"].(map[string]interface{})
		if !ok {
			t.Fatal("radio_config not created")
		}
		b24, ok := rc["band_24"].(map[string]interface{})
		if !ok {
			t.Fatal("band_24 not created")
		}
		if b24["power"] != 12 {
			t.Errorf("power = %v, want 12", b24["power"])
		}
	})

	t.Run("overwrites existing value", func(t *testing.T) {
		data := map[string]interface{}{
			"radio_config": map[string]interface{}{
				"band_24": map[string]interface{}{
					"power": 10,
				},
			},
		}
		SetValueAtPath(data, []string{"radio_config", "band_24", "power"}, 15)

		got, _ := GetValueAtPath(data, []string{"radio_config", "band_24", "power"})
		if got != 15 {
			t.Errorf("power = %v, want 15", got)
		}
	})
}

func TestDeleteValueAtPath(t *testing.T) {
	t.Run("delete existing value", func(t *testing.T) {
		data := map[string]interface{}{
			"name": "AP-01",
			"tags": []string{"a"},
		}
		deleted := DeleteValueAtPath(data, []string{"name"})
		if !deleted {
			t.Error("DeleteValueAtPath should return true")
		}
		if _, exists := data["name"]; exists {
			t.Error("name should be deleted")
		}
		if _, exists := data["tags"]; !exists {
			t.Error("tags should still exist")
		}
	})

	t.Run("delete nested value", func(t *testing.T) {
		data := map[string]interface{}{
			"radio_config": map[string]interface{}{
				"band_24": map[string]interface{}{
					"power":   12,
					"channel": 6,
				},
			},
		}
		deleted := DeleteValueAtPath(data, []string{"radio_config", "band_24", "power"})
		if !deleted {
			t.Error("DeleteValueAtPath should return true")
		}

		b24, _ := data["radio_config"].(map[string]interface{})["band_24"].(map[string]interface{})
		if _, exists := b24["power"]; exists {
			t.Error("power should be deleted")
		}
		if b24["channel"] != 6 {
			t.Error("channel should still exist")
		}
	})

	t.Run("delete nonexistent value", func(t *testing.T) {
		data := map[string]interface{}{"name": "AP-01"}
		deleted := DeleteValueAtPath(data, []string{"nonexistent"})
		if deleted {
			t.Error("DeleteValueAtPath should return false for nonexistent key")
		}
	})
}

func TestCollectMatchingPaths(t *testing.T) {
	data := map[string]interface{}{
		"port_config": map[string]interface{}{
			"eth0": map[string]interface{}{
				"vlan_id": 100,
				"mode":    "access",
			},
			"eth1": map[string]interface{}{
				"vlan_id": 200,
				"mode":    "trunk",
			},
		},
		"radio_config": map[string]interface{}{
			"band_24": map[string]interface{}{"power": 12},
			"band_5":  map[string]interface{}{"power": 15},
		},
	}

	t.Run("wildcard expansion", func(t *testing.T) {
		kp := Parse("port_config.*.vlan_id")
		paths := CollectMatchingPaths(data, kp)

		// Sort for consistent comparison
		var pathStrs []string
		for _, p := range paths {
			pathStrs = append(pathStrs, p[0]+"."+p[1]+"."+p[2])
		}
		sort.Strings(pathStrs)

		expected := []string{"port_config.eth0.vlan_id", "port_config.eth1.vlan_id"}
		if !reflect.DeepEqual(pathStrs, expected) {
			t.Errorf("got %v, want %v", pathStrs, expected)
		}
	})

	t.Run("no wildcard", func(t *testing.T) {
		kp := Parse("radio_config.band_24.power")
		paths := CollectMatchingPaths(data, kp)
		if len(paths) != 1 {
			t.Errorf("expected 1 path, got %d", len(paths))
		}
	})
}

func TestExpandWildcardPath(t *testing.T) {
	data := map[string]interface{}{
		"port_config": map[string]interface{}{
			"eth0": map[string]interface{}{"vlan_id": 100},
			"eth1": map[string]interface{}{"vlan_id": 200},
		},
	}

	t.Run("with wildcard", func(t *testing.T) {
		paths := ExpandWildcardPath(data, "port_config.*.vlan_id")
		sort.Strings(paths)
		expected := []string{"port_config.eth0.vlan_id", "port_config.eth1.vlan_id"}
		if !reflect.DeepEqual(paths, expected) {
			t.Errorf("got %v, want %v", paths, expected)
		}
	})

	t.Run("without wildcard exists", func(t *testing.T) {
		paths := ExpandWildcardPath(data, "port_config.eth0.vlan_id")
		if len(paths) != 1 || paths[0] != "port_config.eth0.vlan_id" {
			t.Errorf("got %v, want [port_config.eth0.vlan_id]", paths)
		}
	})

	t.Run("without wildcard not exists", func(t *testing.T) {
		paths := ExpandWildcardPath(data, "port_config.eth2.vlan_id")
		if len(paths) != 0 {
			t.Errorf("got %v, want empty", paths)
		}
	})
}

func TestIsKeyManaged(t *testing.T) {
	managedKeys := []string{
		"name",
		"radio_config.band_24",
		"port_config.*.vlan_id",
	}

	tests := []struct {
		key  string
		want bool
	}{
		// Direct matches
		{"name", true},
		{"radio_config.band_24", true},

		// Parent path includes children
		{"radio_config.band_24.power", true},
		{"radio_config.band_24.channel", true},

		// Wildcard matches
		{"port_config.eth0.vlan_id", true},
		{"port_config.eth1.vlan_id", true},

		// Not managed
		{"tags", false},
		{"radio_config.band_5", false},
		{"port_config.eth0.mode", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if got := IsKeyManaged(tt.key, managedKeys); got != tt.want {
				t.Errorf("IsKeyManaged(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestFilterMapByManagedKeys(t *testing.T) {
	data := map[string]interface{}{
		"name": "AP-01",
		"tags": []interface{}{"lobby"},
		"radio_config": map[string]interface{}{
			"band_24": map[string]interface{}{
				"power":   12,
				"channel": 6,
			},
			"band_5": map[string]interface{}{
				"power": 15,
			},
		},
		"port_config": map[string]interface{}{
			"eth0": map[string]interface{}{
				"vlan_id": 100,
				"mode":    "access",
			},
		},
	}

	t.Run("filter with simple and nested keys", func(t *testing.T) {
		managedKeys := []string{"name", "radio_config.band_24.power"}
		result := FilterMapByManagedKeys(data, managedKeys)

		if result["name"] != "AP-01" {
			t.Error("name should be included")
		}
		if _, exists := result["tags"]; exists {
			t.Error("tags should not be included")
		}

		rc, ok := result["radio_config"].(map[string]interface{})
		if !ok {
			t.Fatal("radio_config should exist")
		}
		b24, ok := rc["band_24"].(map[string]interface{})
		if !ok {
			t.Fatal("band_24 should exist")
		}
		if b24["power"] != 12 {
			t.Error("power should be 12")
		}
		if _, exists := b24["channel"]; exists {
			t.Error("channel should not be included")
		}
	})

	t.Run("filter with parent key includes all children", func(t *testing.T) {
		managedKeys := []string{"radio_config.band_24"}
		result := FilterMapByManagedKeys(data, managedKeys)

		rc, _ := result["radio_config"].(map[string]interface{})
		b24, _ := rc["band_24"].(map[string]interface{})

		if b24["power"] != 12 {
			t.Error("power should be included")
		}
		if b24["channel"] != 6 {
			t.Error("channel should be included")
		}
	})

	t.Run("filter with wildcard", func(t *testing.T) {
		managedKeys := []string{"port_config.*.vlan_id"}
		result := FilterMapByManagedKeys(data, managedKeys)

		pc, ok := result["port_config"].(map[string]interface{})
		if !ok {
			t.Fatal("port_config should exist")
		}
		eth0, ok := pc["eth0"].(map[string]interface{})
		if !ok {
			t.Fatal("eth0 should exist")
		}
		if eth0["vlan_id"] != 100 {
			t.Error("vlan_id should be 100")
		}
		if _, exists := eth0["mode"]; exists {
			t.Error("mode should not be included")
		}
	})

	t.Run("empty managed keys returns nil", func(t *testing.T) {
		result := FilterMapByManagedKeys(data, nil)
		if result != nil {
			t.Error("empty managedKeys should return nil")
		}
	})
}

func TestCompareValuesAtPath(t *testing.T) {
	a := map[string]interface{}{
		"name": "AP-01",
		"radio_config": map[string]interface{}{
			"band_24": map[string]interface{}{"power": 12},
		},
	}
	b := map[string]interface{}{
		"name": "AP-02",
		"radio_config": map[string]interface{}{
			"band_24": map[string]interface{}{"power": 12},
		},
	}

	tests := []struct {
		path    string
		differs bool
	}{
		{"name", true},
		{"radio_config.band_24.power", false},
		{"radio_config.band_24", false},
		{"nonexistent", false}, // both don't have it
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := CompareValuesAtPath(a, b, tt.path); got != tt.differs {
				t.Errorf("CompareValuesAtPath(%q) = %v, want %v", tt.path, got, tt.differs)
			}
		})
	}
}

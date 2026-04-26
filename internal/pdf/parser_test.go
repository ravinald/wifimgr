package pdf

import (
	"testing"
)

func TestSplitNatural(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"AP1", []string{"AP", "1"}},
		{"AP-12", []string{"AP-", "12"}},
		{"AP10b3", []string{"AP", "10", "b", "3"}},
		{"123", []string{"123"}},
		{"abc", []string{"abc"}},
		{"", nil},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got := splitNatural(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("splitNatural(%q) = %v, want %v", tc.in, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("splitNatural(%q)[%d] = %q, want %q", tc.in, i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestNaturalCompare(t *testing.T) {
	cases := []struct {
		a, b string
		sign int // -1, 0, +1
	}{
		{"AP1", "AP2", -1},
		{"AP10", "AP2", +1},          // 10 > 2 numerically
		{"AP2", "AP10", -1},          // 2 < 10 numerically (lexical comparison would say otherwise)
		{"AP1", "AP1", 0},
		{"AP", "AP1", -1},            // shorter wins on equal prefix
		{"AP1", "AP", +1},
		{"AP-LAB-1", "AP-LAB-10", -1},
		{"AP-LAB-2", "AP-LAB-10", -1},
	}
	for _, tc := range cases {
		t.Run(tc.a+"_vs_"+tc.b, func(t *testing.T) {
			got := naturalCompare(tc.a, tc.b)
			if (got < 0 && tc.sign != -1) || (got > 0 && tc.sign != +1) || (got == 0 && tc.sign != 0) {
				t.Errorf("naturalCompare(%q, %q) = %d, want sign %d", tc.a, tc.b, got, tc.sign)
			}
		})
	}
}

func TestParseRadioSettings(t *testing.T) {
	p := NewParser()

	cases := []struct {
		name     string
		settings string
		check    func(*testing.T, *APConfig)
	}{
		{
			name:     "explicit channel/power/width on 5GHz",
			settings: "/5:36:14:80",
			check: func(t *testing.T, c *APConfig) {
				if c.Band5G == nil {
					t.Fatal("Band5G nil")
				}
				if c.Band5G.Channel != "36" || c.Band5G.Power != "14" || c.Band5G.Width != "80" {
					t.Errorf("Band5G = %+v", c.Band5G)
				}
			},
		},
		{
			name:     "auto channel via -1",
			settings: "/2:-1:10:20",
			check: func(t *testing.T, c *APConfig) {
				if c.Band24G == nil || c.Band24G.Channel != "auto" {
					t.Errorf("expected Band24G.Channel=auto, got %+v", c.Band24G)
				}
			},
		},
		{
			name:     "auto channel via 0",
			settings: "/5:0:14:80",
			check: func(t *testing.T, c *APConfig) {
				if c.Band5G == nil || c.Band5G.Channel != "auto" {
					t.Errorf("expected Band5G.Channel=auto, got %+v", c.Band5G)
				}
			},
		},
		{
			name:     "auto power via -1",
			settings: "/5:36:-1:80",
			check: func(t *testing.T, c *APConfig) {
				if c.Band5G == nil || c.Band5G.Power != "auto" {
					t.Errorf("expected Band5G.Power=auto, got %+v", c.Band5G)
				}
			},
		},
		{
			name:     "all three bands populated",
			settings: "/2:1:10:20/5:36:14:80/6:33:14:160",
			check: func(t *testing.T, c *APConfig) {
				if c.Band24G == nil || c.Band5G == nil || c.Band6G == nil {
					t.Fatalf("missing band: %+v", c)
				}
				if c.Band24G.Channel != "1" || c.Band5G.Channel != "36" || c.Band6G.Channel != "33" {
					t.Errorf("channel mismatch: 2.4G=%v 5G=%v 6G=%v", c.Band24G, c.Band5G, c.Band6G)
				}
			},
		},
		{
			name:     "missing width defaults to empty",
			settings: "/5:36:14:",
			check: func(t *testing.T, c *APConfig) {
				if c.Band5G == nil || c.Band5G.Width != "" {
					t.Errorf("expected empty Width, got %+v", c.Band5G)
				}
			},
		},
		{
			name:     "unknown band type is ignored",
			settings: "/9:36:14:80",
			check: func(t *testing.T, c *APConfig) {
				if c.Band24G != nil || c.Band5G != nil || c.Band6G != nil {
					t.Errorf("unknown band populated something: %+v", c)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &APConfig{Name: "AP-LAB-1"}
			p.parseRadioSettings(cfg, tc.settings)
			tc.check(t, cfg)
		})
	}
}

func TestParserAPRegex(t *testing.T) {
	p := NewParser()

	cases := []struct {
		name      string
		text      string
		wantNames []string
	}{
		{
			name:      "single AP with 5GHz only",
			text:      "noise @AP-LAB-1/5:36:14:80 more noise",
			wantNames: []string{"AP-LAB-1"},
		},
		{
			name:      "multiple APs in one blob",
			text:      "@AP-1/2:1:10:20 stuff @AP-2/5:36:14:80",
			wantNames: []string{"AP-1", "AP-2"},
		},
		{
			name:      "missing @ prefix is not matched",
			text:      "AP-LAB-1/5:36:14:80",
			wantNames: nil,
		},
		{
			name:      "garbage near match",
			text:      "@@/notvalid @AP-OK/5:36:14:80",
			wantNames: []string{"AP-OK"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			matches := p.apRegex.FindAllStringSubmatch(tc.text, -1)
			if len(matches) != len(tc.wantNames) {
				t.Fatalf("matches = %v, want %d names", matches, len(tc.wantNames))
			}
			for i, m := range matches {
				if m[1] != tc.wantNames[i] {
					t.Errorf("match[%d] name = %q, want %q", i, m[1], tc.wantNames[i])
				}
			}
		})
	}
}

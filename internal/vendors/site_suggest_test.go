package vendors

import (
	"strings"
	"testing"
)

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"", "abc", 3},
		{"abc", "", 3},
		{"kitten", "sitting", 3},
		{"Mexicano", "Mexicanoo", 1},
		{"hello", "hello", 0},
		{"ab", "ba", 2},
	}
	for _, tt := range tests {
		if got := levenshtein(tt.a, tt.b); got != tt.want {
			t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestSuggestSiteNames(t *testing.T) {
	sites := []string{
		"MX - Av. Ejercito Nacional Mexicano 904",
		"US-LAB-01",
		"US-LAB-02",
		"EU-OFFICE-01",
	}

	tests := []struct {
		name        string
		target      string
		maxDistance int
		maxResults  int
		want        []string
	}{
		{
			name:        "exact single-char typo surfaces match",
			target:      "MX - Av. Ejercito Nacional Mexicanoo 904",
			maxDistance: 3,
			maxResults:  3,
			want:        []string{"MX - Av. Ejercito Nacional Mexicano 904"},
		},
		{
			// Distance 0 after case-folding both sides.
			name:        "case-insensitive exact match",
			target:      "us-lab-01",
			maxDistance: 0,
			maxResults:  3,
			want:        []string{"US-LAB-01"},
		},
		{
			name:        "close-but-not-close-enough returns nothing",
			target:      "definitely-different-name",
			maxDistance: 3,
			maxResults:  3,
			want:        nil,
		},
		{
			name:        "limits results",
			target:      "US-LAB-03",
			maxDistance: 1,
			maxResults:  1,
			want:        []string{"US-LAB-01"}, // alphabetically first at same distance
		},
		{
			name:        "empty target returns nothing",
			target:      "",
			maxDistance: 3,
			maxResults:  3,
			want:        nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SuggestSiteNames(tt.target, sites, tt.maxDistance, tt.maxResults)
			if !equalStringSlices(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSuggestSiteNames_Defaults(t *testing.T) {
	sites := []string{"alpha", "alpho", "alphb", "alphc", "alphd"}
	// maxResults=0 → default 3
	got := SuggestSiteNames("alpha", sites, 1, 0)
	if len(got) > 3 {
		t.Errorf("maxResults=0 should default to 3, got %d results: %v", len(got), got)
	}
}

func TestFormatSuggestions(t *testing.T) {
	if got := FormatSuggestions(nil); got != "" {
		t.Errorf("empty input should render empty, got %q", got)
	}
	got := FormatSuggestions([]string{"foo", "bar"})
	if !strings.Contains(got, "did you mean?") {
		t.Errorf("expected 'did you mean?' header, got %q", got)
	}
	if !strings.Contains(got, "  foo") || !strings.Contains(got, "  bar") {
		t.Errorf("expected indented candidates, got %q", got)
	}
}

func TestSiteNotFoundError_RendersSuggestions(t *testing.T) {
	e := &SiteNotFoundError{
		SiteName:    "Mexicanoo",
		APILabel:    "meraki",
		Suggestions: []string{"Mexicano"},
	}
	msg := e.Error()
	if !strings.Contains(msg, "Mexicanoo") {
		t.Errorf("Error() should mention the requested name: %q", msg)
	}
	if !strings.Contains(msg, "did you mean?") {
		t.Errorf("Error() should include suggestion block: %q", msg)
	}
	if !strings.Contains(msg, "Mexicano") {
		t.Errorf("Error() should list candidate: %q", msg)
	}
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

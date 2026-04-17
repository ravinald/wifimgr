package vendors

import (
	"sort"
	"strings"
)

// SuggestSiteNames returns up to `maxResults` candidate site names that are
// close matches for `target`, ordered by ascending edit distance. It's used to
// build "did you mean?" hints when a site lookup misses — typos like
// "Mexicanoo" vs "Mexicano" surface as a single-character change.
//
// Matching is case-insensitive. Candidates within `maxDistance` edits of the
// target qualify; if no candidate meets the threshold, the return is empty.
// A zero `maxResults` is treated as the default (3).
func SuggestSiteNames(target string, candidates []string, maxDistance, maxResults int) []string {
	if target == "" || len(candidates) == 0 {
		return nil
	}
	if maxDistance < 0 {
		maxDistance = 0
	}
	if maxResults <= 0 {
		maxResults = 3
	}

	targetLower := strings.ToLower(target)

	type scored struct {
		name string
		dist int
	}
	var matches []scored
	for _, c := range candidates {
		d := levenshtein(targetLower, strings.ToLower(c))
		if d <= maxDistance {
			matches = append(matches, scored{c, d})
		}
	}

	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].dist != matches[j].dist {
			return matches[i].dist < matches[j].dist
		}
		return matches[i].name < matches[j].name
	})

	if len(matches) > maxResults {
		matches = matches[:maxResults]
	}
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		out = append(out, m.name)
	}
	return out
}

// FormatSuggestions renders a "did you mean?" block suitable for appending to
// an error message. Returns an empty string when suggestions is empty so the
// caller can unconditionally concat.
func FormatSuggestions(suggestions []string) string {
	if len(suggestions) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\ndid you mean?\n")
	for _, s := range suggestions {
		b.WriteString("  ")
		b.WriteString(s)
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// levenshtein returns the Levenshtein edit distance between a and b. Uses a
// single-row rolling DP to keep the allocation small for typical inputs.
func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	if len(ra) == 0 {
		return len(rb)
	}
	if len(rb) == 0 {
		return len(ra)
	}

	prev := make([]int, len(rb)+1)
	curr := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= len(ra); i++ {
		curr[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			curr[j] = minInt(
				prev[j]+1,        // deletion
				curr[j-1]+1,      // insertion
				prev[j-1]+cost,   // substitution
			)
		}
		prev, curr = curr, prev
	}
	return prev[len(rb)]
}

func minInt(values ...int) int {
	m := values[0]
	for _, v := range values[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

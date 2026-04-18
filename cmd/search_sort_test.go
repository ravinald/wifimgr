package cmd

import (
	"testing"

	"github.com/ravinald/wifimgr/internal/config"
)

// TestNewNameSortExtractor_NilAndEmpty: a nil or empty spec is valid and
// returns (nil, nil). Callers treat nil as "no config, use the natural
// fallback" — that's a feature, not a missing configuration.
func TestNewNameSortExtractor_NilAndEmpty(t *testing.T) {
	if e, err := newNameSortExtractor(nil); err != nil || e != nil {
		t.Errorf("nil spec: want (nil, nil), got (%v, %v)", e, err)
	}
	if e, err := newNameSortExtractor(&config.SortKeySpec{}); err != nil || e != nil {
		t.Errorf("empty pattern: want (nil, nil), got (%v, %v)", e, err)
	}
}

// TestNewNameSortExtractor_InvalidPattern rejects an unparseable regex and
// surfaces the error so the caller can warn and fall back.
func TestNewNameSortExtractor_InvalidPattern(t *testing.T) {
	_, err := newNameSortExtractor(&config.SortKeySpec{Pattern: "["})
	if err == nil {
		t.Fatal("invalid pattern should return error")
	}
}

// TestNewNameSortExtractor_UnknownKeyName rejects a keys entry that
// doesn't name a capture group in the pattern. Catches a common operator
// typo at config load instead of silently producing the wrong sort.
func TestNewNameSortExtractor_UnknownKeyName(t *testing.T) {
	_, err := newNameSortExtractor(&config.SortKeySpec{
		Pattern: `^ap(?P<num>\d+)-(?P<floor>\d+)$`,
		Keys:    []string{"floor", "does_not_exist"},
	})
	if err == nil {
		t.Fatal("unknown key should return error")
	}
}

// TestCompareByConfiguredName_FloorTrailingUserCase exercises the primary
// motivating case: apX-F naming where floor (trailing) is the desired
// primary sort key. Produces the floor-clustered order the user wanted
// for the MX-904 site.
func TestCompareByConfiguredName_FloorTrailingUserCase(t *testing.T) {
	spec := &config.SortKeySpec{
		Pattern: `^ap(?P<num>\d+)-(?P<floor>\d+)$`,
		Keys:    []string{"floor", "num"},
	}
	e, err := newNameSortExtractor(spec)
	if err != nil {
		t.Fatalf("extractor: %v", err)
	}

	names := []string{"ap2-16", "ap1-15", "ap10-15", "ap2-15", "ap1-16", "ap10-16"}
	sortNamesWith(e, names)

	want := []string{"ap1-15", "ap2-15", "ap10-15", "ap1-16", "ap2-16", "ap10-16"}
	if !equalSlice(names, want) {
		t.Errorf("sorted = %v, want %v", names, want)
	}
}

// TestCompareByConfiguredName_LeadingFirstBuildingFloor covers the common
// non-user convention where building/floor is prefixed (F1-AP01 style).
func TestCompareByConfiguredName_LeadingFirstBuildingFloor(t *testing.T) {
	spec := &config.SortKeySpec{
		Pattern: `^(?P<building>[A-Z]+)-(?P<floor>\d+)-ap(?P<num>\d+)$`,
		Keys:    []string{"building", "floor", "num"},
	}
	e, err := newNameSortExtractor(spec)
	if err != nil {
		t.Fatalf("extractor: %v", err)
	}

	names := []string{"B-2-ap1", "A-10-ap1", "A-1-ap2", "A-1-ap1", "A-2-ap1"}
	sortNamesWith(e, names)

	want := []string{"A-1-ap1", "A-1-ap2", "A-2-ap1", "A-10-ap1", "B-2-ap1"}
	if !equalSlice(names, want) {
		t.Errorf("sorted = %v, want %v", names, want)
	}
}

// TestCompareByConfiguredName_UnmatchedSortAfter: names that don't match
// the pattern get sorted after matching names, with natural order within
// the unmatched bucket. Operators see outliers clearly instead of having
// them silently placed somewhere unexpected.
func TestCompareByConfiguredName_UnmatchedSortAfter(t *testing.T) {
	spec := &config.SortKeySpec{
		Pattern: `^ap(?P<num>\d+)-(?P<floor>\d+)$`,
		Keys:    []string{"floor", "num"},
	}
	e, _ := newNameSortExtractor(spec)

	names := []string{"legacy-ap", "ap2-16", "ap1-15", "aardvark"}
	sortNamesWith(e, names)

	// matched: ap1-15 (floor 15, num 1), ap2-16 (floor 16, num 2)
	// unmatched: "aardvark", "legacy-ap" by natural order
	want := []string{"ap1-15", "ap2-16", "aardvark", "legacy-ap"}
	if !equalSlice(names, want) {
		t.Errorf("sorted = %v, want %v", names, want)
	}
}

// TestCompareByConfiguredName_NilExtractor: a nil extractor (no config,
// or invalid config that was swallowed into a warning) falls back to
// natural.Less on the full name. Proves the fallback path.
func TestCompareByConfiguredName_NilExtractor(t *testing.T) {
	names := []string{"ap10-15", "ap2-15", "ap1-16"}
	sortNamesWith(nil, names)

	// Natural order: ap1-16 < ap2-15 < ap10-15 (by leading number first).
	want := []string{"ap1-16", "ap2-15", "ap10-15"}
	if !equalSlice(names, want) {
		t.Errorf("sorted = %v, want %v", names, want)
	}
}

// TestCompareByConfiguredName_MixedNumericAndString handles a pattern
// whose captures include non-numeric content. Ensures the segment
// comparator doesn't panic on string groups.
func TestCompareByConfiguredName_MixedNumericAndString(t *testing.T) {
	spec := &config.SortKeySpec{
		Pattern: `^ap(?P<num>\d+)-(?P<wing>[A-Z])$`,
		Keys:    []string{"wing", "num"},
	}
	e, _ := newNameSortExtractor(spec)

	names := []string{"ap10-A", "ap2-B", "ap1-A", "ap2-A"}
	sortNamesWith(e, names)

	// Wing A cluster first (by num: 1, 2, 10), then Wing B.
	want := []string{"ap1-A", "ap2-A", "ap10-A", "ap2-B"}
	if !equalSlice(names, want) {
		t.Errorf("sorted = %v, want %v", names, want)
	}
}

// sortNamesWith sorts a string slice in place using compareByConfiguredName
// as the comparator. Local helper so each test reads as a straight line.
func sortNamesWith(e *nameSortExtractor, names []string) {
	// Bubble-style insertion via stdlib sort.
	// Can't use sort.Slice directly because the comparator returns -1/0/+1
	// and that's the shape needed for equality-aware logic elsewhere.
	n := len(names)
	for i := 1; i < n; i++ {
		for j := i; j > 0; j-- {
			if compareByConfiguredName(e, names[j-1], names[j]) <= 0 {
				break
			}
			names[j-1], names[j] = names[j], names[j-1]
		}
	}
}

func equalSlice(a, b []string) bool {
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

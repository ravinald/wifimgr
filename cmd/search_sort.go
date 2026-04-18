package cmd

import (
	"fmt"
	"regexp"
	"strconv"
	"sync"

	"github.com/maruel/natural"
	"github.com/spf13/viper"

	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/logging"
)

// nameSortExtractor turns a device name (AP hostname, switch hostname) into
// a sort-key slice using the regex+keys configuration an operator supplies
// under display.sort.ap_name / display.sort.switch_name. It exists so the
// search commands can group rows by hostname metadata (floor, building,
// AP number) rather than by natural string order on the full name.
//
// An extractor is cheap to build but not cheap to compile: each construction
// calls regexp.Compile. Callers should build once per search invocation,
// reuse across comparator calls, and discard when done. The per-process
// cache lives in extractorCache below.
type nameSortExtractor struct {
	re   *regexp.Regexp
	keys []string

	// warnOnce gates the "config invalid, falling back" warning to a single
	// line per process, so a bad pattern doesn't flood the log during a
	// sort of hundreds of rows.
	warnOnce sync.Once
}

// sortSegment is one element of an extracted sort key. Numeric segments
// compare numerically; string segments compare via natural.Less. Mixed
// comparison falls back to natural string order on both.
type sortSegment struct {
	isNum bool
	num   int
	s     string
}

// newNameSortExtractor builds an extractor from the supplied spec. Returns
// (nil, nil) when spec is nil or has an empty pattern — no configuration
// is a valid state meaning "keep the default natural sort."
//
// Invalid regex or a keys entry that doesn't name a capture group returns
// a non-nil error; callers treat that as "fall back to default" and surface
// the error as a one-shot warning.
func newNameSortExtractor(spec *config.SortKeySpec) (*nameSortExtractor, error) {
	if spec == nil || spec.Pattern == "" {
		return nil, nil
	}
	re, err := regexp.Compile(spec.Pattern)
	if err != nil {
		return nil, fmt.Errorf("compile pattern %q: %w", spec.Pattern, err)
	}
	known := make(map[string]struct{})
	for _, n := range re.SubexpNames() {
		if n != "" {
			known[n] = struct{}{}
		}
	}
	for _, k := range spec.Keys {
		if _, ok := known[k]; !ok {
			return nil, fmt.Errorf("keys[%q] is not a named group in pattern %q", k, spec.Pattern)
		}
	}
	return &nameSortExtractor{re: re, keys: spec.Keys}, nil
}

// extract pulls the configured sort key for name. Returns (key, true) when
// the pattern matches; (nil, false) otherwise — unmatched names are sorted
// after matched ones using natural order on the full string.
func (e *nameSortExtractor) extract(name string) ([]sortSegment, bool) {
	if e == nil {
		return nil, false
	}
	m := e.re.FindStringSubmatch(name)
	if m == nil {
		return nil, false
	}
	subNames := e.re.SubexpNames()
	lookup := make(map[string]string, len(subNames))
	for i, n := range subNames {
		if n != "" {
			lookup[n] = m[i]
		}
	}
	out := make([]sortSegment, len(e.keys))
	for i, k := range e.keys {
		out[i] = toSortSegment(lookup[k])
	}
	return out, true
}

// toSortSegment chooses numeric or string representation for a captured
// value. Pure-digit captures parse as int; everything else stays a string.
func toSortSegment(v string) sortSegment {
	if n, err := strconv.Atoi(v); err == nil {
		return sortSegment{isNum: true, num: n}
	}
	return sortSegment{s: v}
}

// lessSegment returns (a < b, a == b) for two segments. Numeric-vs-numeric
// compares integers; anything else falls back to natural string order
// (stringifying numerics so "10" < "9" doesn't happen).
func lessSegment(a, b sortSegment) (less, equal bool) {
	if a.isNum && b.isNum {
		switch {
		case a.num < b.num:
			return true, false
		case a.num > b.num:
			return false, false
		}
		return false, true
	}
	as := a.s
	if a.isNum {
		as = strconv.Itoa(a.num)
	}
	bs := b.s
	if b.isNum {
		bs = strconv.Itoa(b.num)
	}
	if as == bs {
		return false, true
	}
	return natural.Less(as, bs), false
}

// lessSegments compares two key slices lexicographically. Shorter slices
// sort before longer slices when they share a prefix; this case should not
// arise in practice (extractor always returns len(e.keys) segments) but
// the guard keeps the helper total.
func lessSegments(a, b []sortSegment) bool {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		less, eq := lessSegment(a[i], b[i])
		if eq {
			continue
		}
		return less
	}
	return len(a) < len(b)
}

// compareByConfiguredName returns -1, 0, or +1 comparing two device names
// through the extractor. Matched names sort before unmatched names; within
// the matched bucket, keys decide; within the unmatched bucket, natural
// string order decides. With a nil extractor (no config, invalid config)
// the fallback is natural.Less on the full name.
func compareByConfiguredName(e *nameSortExtractor, ai, bi string) int {
	if e == nil {
		switch {
		case natural.Less(ai, bi):
			return -1
		case natural.Less(bi, ai):
			return 1
		}
		return 0
	}
	aKey, aOK := e.extract(ai)
	bKey, bOK := e.extract(bi)
	switch {
	case aOK && !bOK:
		return -1
	case !aOK && bOK:
		return 1
	case !aOK && !bOK:
		switch {
		case natural.Less(ai, bi):
			return -1
		case natural.Less(bi, ai):
			return 1
		}
		return 0
	}
	if lessSegments(aKey, bKey) {
		return -1
	}
	if lessSegments(bKey, aKey) {
		return 1
	}
	return 0
}

// extractorCache memoizes compiled extractors so repeated sort calls inside
// the same process don't pay the regex compile tax on every search. Keyed
// by pattern||keysJoined; values are extractor-or-nil (nil for unconfigured,
// invalid, or error states). sync.Map so cold reads stay lock-free under
// parallel refresh. In practice the CLI fires sort() a handful of times
// per invocation, so this is more about correctness (single warning) than
// throughput.
var extractorCache sync.Map // map[string]*nameSortExtractor

// configuredAPNameExtractor returns the cached extractor for display.sort.ap_name
// (or nil if absent/invalid). Warning fires once per extractor instance.
func configuredAPNameExtractor() *nameSortExtractor {
	return extractorForKey("display.sort.ap_name")
}

// configuredSwitchNameExtractor returns the cached extractor for display.sort.switch_name.
func configuredSwitchNameExtractor() *nameSortExtractor {
	return extractorForKey("display.sort.switch_name")
}

// extractorForKey reads the config block at key and returns the cached
// extractor. A cache miss compiles once; subsequent reads reuse the result.
// Errors warn once and return nil (meaning "fall back to natural sort").
func extractorForKey(key string) *nameSortExtractor {
	if !viper.IsSet(key) {
		return nil
	}
	var spec config.SortKeySpec
	if err := viper.UnmarshalKey(key, &spec); err != nil {
		warnSortConfig(key, fmt.Errorf("unmarshal: %w", err))
		return nil
	}
	if spec.Pattern == "" {
		return nil
	}
	cacheKey := key + "|" + spec.Pattern + "|" + joinKeys(spec.Keys)
	if cached, ok := extractorCache.Load(cacheKey); ok {
		if cached == nil {
			return nil
		}
		return cached.(*nameSortExtractor)
	}
	extractor, err := newNameSortExtractor(&spec)
	if err != nil {
		warnSortConfig(key, err)
		extractorCache.Store(cacheKey, (*nameSortExtractor)(nil))
		return nil
	}
	extractorCache.Store(cacheKey, extractor)
	return extractor
}

// joinKeys produces a stable, separator-safe representation of the keys
// slice for cache-key construction. "\x00" is fine since the operator
// would never put it inside a JSON group name.
func joinKeys(keys []string) string {
	out := ""
	for i, k := range keys {
		if i > 0 {
			out += "\x00"
		}
		out += k
	}
	return out
}

// warnSortConfigOnce gates the warning per config-key, separate from the
// per-extractor warnOnce, so that the same bad key doesn't re-warn on each
// uncached read. In practice cacheKey prevents that too, but the explicit
// gate makes the intent obvious.
var warnedKeys sync.Map

func warnSortConfig(key string, err error) {
	if _, already := warnedKeys.LoadOrStore(key, struct{}{}); already {
		return
	}
	logging.Warnf("%s config invalid, falling back to default natural sort: %v", key, err)
}

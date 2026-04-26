package patterns

import (
	"testing"

	"github.com/spf13/viper"
)

// withCaseInsensitive sets viper's case-insensitive flag for the duration of a
// test and restores the previous value on cleanup.
func withCaseInsensitive(t *testing.T, v bool) {
	t.Helper()
	prev := viper.GetBool("case-insensitive")
	viper.Set("case-insensitive", v)
	t.Cleanup(func() { viper.Set("case-insensitive", prev) })
}

func TestPatternMatcher_CaseSensitive(t *testing.T) {
	pm := &PatternMatcher{caseInsensitive: false}

	cases := []struct {
		name string
		fn   func() bool
		want bool
	}{
		{"Contains exact", func() bool { return pm.Contains("FooBar", "Bar") }, true},
		{"Contains wrong case", func() bool { return pm.Contains("FooBar", "bar") }, false},
		{"Equals exact", func() bool { return pm.Equals("foo", "foo") }, true},
		{"Equals wrong case", func() bool { return pm.Equals("foo", "Foo") }, false},
		{"HasPrefix exact", func() bool { return pm.HasPrefix("FooBar", "Foo") }, true},
		{"HasPrefix wrong case", func() bool { return pm.HasPrefix("FooBar", "foo") }, false},
		{"HasSuffix exact", func() bool { return pm.HasSuffix("FooBar", "Bar") }, true},
		{"HasSuffix wrong case", func() bool { return pm.HasSuffix("FooBar", "bar") }, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.fn(); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestPatternMatcher_CaseInsensitive(t *testing.T) {
	pm := &PatternMatcher{caseInsensitive: true}

	cases := []struct {
		name string
		fn   func() bool
		want bool
	}{
		{"Contains mixed", func() bool { return pm.Contains("FooBar", "bar") }, true},
		{"Contains miss", func() bool { return pm.Contains("FooBar", "baz") }, false},
		{"Equals mixed", func() bool { return pm.Equals("foo", "FOO") }, true},
		{"Equals miss", func() bool { return pm.Equals("foo", "foox") }, false},
		{"HasPrefix mixed", func() bool { return pm.HasPrefix("FooBar", "foo") }, true},
		{"HasSuffix mixed", func() bool { return pm.HasSuffix("FooBar", "bar") }, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.fn(); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNewPatternMatcher_ReadsViper(t *testing.T) {
	withCaseInsensitive(t, true)
	pm := NewPatternMatcher()
	if !pm.caseInsensitive {
		t.Error("NewPatternMatcher() did not read case-insensitive=true from viper")
	}

	withCaseInsensitive(t, false)
	pm = NewPatternMatcher()
	if pm.caseInsensitive {
		t.Error("NewPatternMatcher() did not read case-insensitive=false from viper")
	}
}

func TestGlobals_HonorViperFlag(t *testing.T) {
	withCaseInsensitive(t, true)
	if !Contains("FooBar", "bar") {
		t.Error("Contains() should have matched case-insensitively when viper flag is set")
	}
	if !Equals("foo", "FOO") {
		t.Error("Equals() should have matched case-insensitively when viper flag is set")
	}

	withCaseInsensitive(t, false)
	if Contains("FooBar", "bar") {
		t.Error("Contains() should be case-sensitive when viper flag is unset")
	}
	if Equals("foo", "FOO") {
		t.Error("Equals() should be case-sensitive when viper flag is unset")
	}
}

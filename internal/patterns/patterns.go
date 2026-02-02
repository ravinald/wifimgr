package patterns

import (
	"strings"

	"github.com/spf13/viper"
)

// PatternMatcher provides utilities for pattern matching with case sensitivity control
type PatternMatcher struct {
	caseInsensitive bool
}

// NewPatternMatcher creates a new pattern matcher that automatically reads the case-insensitive flag
func NewPatternMatcher() *PatternMatcher {
	return &PatternMatcher{
		caseInsensitive: viper.GetBool("case-insensitive"),
	}
}

// Contains checks if the text contains the pattern, respecting case sensitivity
func (pm *PatternMatcher) Contains(text, pattern string) bool {
	if pm.caseInsensitive {
		return strings.Contains(strings.ToLower(text), strings.ToLower(pattern))
	}
	return strings.Contains(text, pattern)
}

// Equals checks if the text equals the pattern, respecting case sensitivity
func (pm *PatternMatcher) Equals(text, pattern string) bool {
	if pm.caseInsensitive {
		return strings.EqualFold(text, pattern)
	}
	return text == pattern
}

// HasPrefix checks if the text has the prefix, respecting case sensitivity
func (pm *PatternMatcher) HasPrefix(text, prefix string) bool {
	if pm.caseInsensitive {
		return strings.HasPrefix(strings.ToLower(text), strings.ToLower(prefix))
	}
	return strings.HasPrefix(text, prefix)
}

// HasSuffix checks if the text has the suffix, respecting case sensitivity
func (pm *PatternMatcher) HasSuffix(text, suffix string) bool {
	if pm.caseInsensitive {
		return strings.HasSuffix(strings.ToLower(text), strings.ToLower(suffix))
	}
	return strings.HasSuffix(text, suffix)
}

// Global convenience functions for common pattern matching operations

// Contains checks if text contains pattern with case sensitivity from global flag
func Contains(text, pattern string) bool {
	pm := NewPatternMatcher()
	return pm.Contains(text, pattern)
}

// Equals checks if text equals pattern with case sensitivity from global flag
func Equals(text, pattern string) bool {
	pm := NewPatternMatcher()
	return pm.Equals(text, pattern)
}

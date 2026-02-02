package common

import "strings"

// MaskString masks sensitive information like API tokens
func MaskString(s string) string {
	if len(s) <= 8 {
		return "********"
	}
	visible := 4
	return s[:visible] + strings.Repeat("*", len(s)-visible)
}

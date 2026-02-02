// Package macaddr provides utilities for handling and manipulating MAC addresses
//
// This package offers a set of functions to validate, normalize, format, and compare MAC addresses
// in various formats, supporting the most common styles (colon, hyphen, dot, and no separators).
// The primary purpose is to provide a consistent way to handle MAC addresses throughout the application.
//
// The default normalized format used by the application for storage and lookups is lowercase with no separators.
package macaddr

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Format constants represent different MAC address formats
const (
	// FormatNone represents a MAC address with no separators (aabbccddeeff)
	FormatNone = iota
	// FormatColon represents a MAC address with colon separators (aa:bb:cc:dd:ee:ff)
	FormatColon
	// FormatHyphen represents a MAC address with hyphen separators (aa-bb-cc-dd-ee-ff)
	FormatHyphen
	// FormatDot represents a MAC address with dot separators (aabb.ccdd.eeff)
	FormatDot
)

var (
	// ErrInvalidMAC indicates that the provided string is not a valid MAC address
	ErrInvalidMAC = errors.New("invalid MAC address")
)

// Regular expressions for MAC address validation
var (
	// Matches a MAC address in any of the common formats
	macColonFormatRegex  = regexp.MustCompile(`^([0-9A-Fa-f]{2}:){5}([0-9A-Fa-f]{2})$`)
	macHyphenFormatRegex = regexp.MustCompile(`^([0-9A-Fa-f]{2}-){5}([0-9A-Fa-f]{2})$`)
	macDotFormatRegex    = regexp.MustCompile(`^([0-9A-Fa-f]{4}\.){2}([0-9A-Fa-f]{4})$`)
	macNoneFormatRegex   = regexp.MustCompile(`^([0-9A-Fa-f]{12})$`)

	// Composite regex - a MAC address is valid if it matches any of the format-specific regexes
	macAnyFormatRegex = regexp.MustCompile(`^(([0-9A-Fa-f]{2}:){5}([0-9A-Fa-f]{2}))$|^(([0-9A-Fa-f]{2}-){5}([0-9A-Fa-f]{2}))$|^(([0-9A-Fa-f]{4}\.){2}([0-9A-Fa-f]{4}))$|^([0-9A-Fa-f]{12})$`)

	// Format-specific regexes for direct validation (aliases for clarity)
	macNoneRegex   = macNoneFormatRegex
	macColonRegex  = macColonFormatRegex
	macHyphenRegex = macHyphenFormatRegex
	macDotRegex    = macDotFormatRegex
)

// Normalize converts any valid MAC address into a normalized form (lowercase, no separators).
// This is the default format used by the application for storage and lookups.
// It accepts MAC addresses in various formats (colon, hyphen, dot, or no separators)
// and returns the normalized form. If the input is not a valid MAC address,
// it returns an empty string and an error.
//
// Examples:
//
//	"00:11:22:33:44:55" -> "001122334455"
//	"00-11-22-33-44-55" -> "001122334455"
//	"0011.2233.4455"    -> "001122334455"
//	"001122334455"      -> "001122334455"
func Normalize(mac string) (string, error) {
	if !IsValid(mac) {
		return "", ErrInvalidMAC
	}

	// Remove colons, dashes, spaces, and dots
	re := regexp.MustCompile(`[:\-\s\.]`)
	normalized := re.ReplaceAllString(mac, "")

	// Convert to lowercase
	return strings.ToLower(normalized), nil
}

// MustNormalize is like Normalize but panics if the MAC address is invalid.
// This function is useful when you are certain that the MAC address is valid,
// and want to avoid error handling.
//
// Warning: This function panics if the input is not a valid MAC address.
// Use Normalize or NormalizeOrEmpty for safer alternatives.
func MustNormalize(mac string) string {
	normalized, err := Normalize(mac)
	if err != nil {
		panic(err)
	}
	return normalized
}

// NormalizeOrEmpty normalizes a MAC address or returns an empty string if invalid.
// This is a safe alternative to Normalize when you prefer to handle invalid
// MAC addresses by returning an empty string rather than an error.
//
// Examples:
//
//	"00:11:22:33:44:55" -> "001122334455"
//	"invalid"           -> ""
//	""                  -> ""
func NormalizeOrEmpty(mac string) string {
	normalized, err := Normalize(mac)
	if err != nil {
		return ""
	}
	return normalized
}

// NormalizeFast normalizes a MAC address without validation.
// This function is optimized for internal use where the MAC address is already
// known to be valid (e.g., from cache data that was previously validated).
//
// It converts to lowercase and removes all common separators (colons, hyphens,
// dots, and spaces). Unlike Normalize, it does NOT validate the input format.
//
// Use this function only for trusted internal data (cache indexing/lookups).
// For external input (API responses, user config), use Normalize instead.
//
// Examples:
//
//	"00:11:22:33:44:55" -> "001122334455"
//	"00-11-22-33-44-55" -> "001122334455"
//	"invalid"           -> "invalid" (no validation!)
//	""                  -> ""
func NormalizeFast(mac string) string {
	if mac == "" {
		return ""
	}
	// Convert to lowercase and remove separators
	normalized := strings.ToLower(mac)
	normalized = strings.ReplaceAll(normalized, ":", "")
	normalized = strings.ReplaceAll(normalized, "-", "")
	normalized = strings.ReplaceAll(normalized, ".", "")
	normalized = strings.ReplaceAll(normalized, " ", "")
	return normalized
}

// Format converts a MAC address into the specified format.
// The input MAC address can be in any valid format; it will be normalized first,
// then converted to the requested format. The format parameter must be one of
// the predefined Format constants (FormatNone, FormatColon, FormatHyphen, FormatDot).
//
// Examples:
//
//	Format("00:11:22:33:44:55", FormatNone)   -> "001122334455"
//	Format("001122334455", FormatColon)       -> "00:11:22:33:44:55"
//	Format("00:11:22:33:44:55", FormatHyphen) -> "00-11-22-33-44-55"
//	Format("00-11-22-33-44-55", FormatDot)    -> "0011.2233.4455"
//
// Returns an error if the input MAC address is invalid or if an unsupported format is requested.
func Format(mac string, format int) (string, error) {
	// First normalize the MAC to ensure we're working with a clean value
	normalized, err := Normalize(mac)
	if err != nil {
		return "", err
	}

	if len(normalized) != 12 {
		return "", ErrInvalidMAC
	}

	switch format {
	case FormatNone:
		return normalized, nil
	case FormatColon:
		var b strings.Builder
		for i := 0; i < 12; i += 2 {
			if i > 0 {
				b.WriteString(":")
			}
			b.WriteString(normalized[i : i+2])
		}
		return b.String(), nil
	case FormatHyphen:
		var b strings.Builder
		for i := 0; i < 12; i += 2 {
			if i > 0 {
				b.WriteString("-")
			}
			b.WriteString(normalized[i : i+2])
		}
		return b.String(), nil
	case FormatDot:
		var b strings.Builder
		for i := 0; i < 12; i += 4 {
			if i > 0 {
				b.WriteString(".")
			}
			b.WriteString(normalized[i : i+4])
		}
		return b.String(), nil
	default:
		return "", fmt.Errorf("unsupported format: %d", format)
	}
}

// IsValid checks if a string is a valid MAC address in any of the common formats.
// This function recognizes the following formats:
// - No separators: 001122334455
// - Colon-separated: 00:11:22:33:44:55
// - Hyphen-separated: 00-11-22-33-44-55
// - Dot-separated (Cisco format): 0011.2233.4455
//
// MAC addresses with mixed separators are NOT considered valid.
// Empty strings are considered invalid.
//
// Returns true if the MAC address is valid in any of the supported formats,
// false otherwise.
func IsValid(mac string) bool {
	if mac == "" {
		return false
	}
	return macAnyFormatRegex.MatchString(mac)
}

// IsValidWithFormat checks if a MAC address is valid with a specific format.
// The format parameter must be one of the predefined Format constants
// (FormatNone, FormatColon, FormatHyphen, FormatDot).
//
// Examples:
//
//	IsValidWithFormat("001122334455", FormatNone)       -> true
//	IsValidWithFormat("00:11:22:33:44:55", FormatColon) -> true
//	IsValidWithFormat("00:11:22:33:44:55", FormatNone)  -> false
//
// Returns false for empty strings or if an unsupported format is specified.
func IsValidWithFormat(mac string, format int) bool {
	if mac == "" {
		return false
	}

	switch format {
	case FormatNone:
		return macNoneRegex.MatchString(mac)
	case FormatColon:
		return macColonRegex.MatchString(mac)
	case FormatHyphen:
		return macHyphenRegex.MatchString(mac)
	case FormatDot:
		return macDotRegex.MatchString(mac)
	default:
		return false
	}
}

// Equal checks if two MAC addresses are equal, ignoring format differences.
// This function normalizes both MAC addresses (converts to lowercase with no separators)
// before comparing them. If either MAC address is invalid, they are considered not equal.
//
// Examples:
//
//	Equal("00:11:22:33:44:55", "00-11-22-33-44-55") -> true
//	Equal("00:11:22:33:44:55", "00:11:22:33:44:56") -> false
//	Equal("00:11:22:33:44:55", "invalid")           -> false
//
// This is useful for comparing MAC addresses that might be in different formats.
func Equal(mac1, mac2 string) bool {
	norm1, err1 := Normalize(mac1)
	norm2, err2 := Normalize(mac2)

	// If either MAC is invalid, they're not equal
	if err1 != nil || err2 != nil {
		return false
	}

	return norm1 == norm2
}

// DetectFormat tries to detect the format of a MAC address.
// This function returns the format constant (FormatNone, FormatColon, FormatHyphen, FormatDot)
// that matches the format of the input MAC address.
//
// Returns an error if the MAC address is invalid.
// Returns the format constant and nil error if successful.
//
// Examples:
//
//	DetectFormat("001122334455")      -> FormatNone, nil
//	DetectFormat("00:11:22:33:44:55") -> FormatColon, nil
//	DetectFormat("invalid")           -> -1, ErrInvalidMAC
func DetectFormat(mac string) (int, error) {
	if !IsValid(mac) {
		return -1, ErrInvalidMAC
	}

	if macNoneRegex.MatchString(mac) {
		return FormatNone, nil
	}
	if macColonRegex.MatchString(mac) {
		return FormatColon, nil
	}
	if macHyphenRegex.MatchString(mac) {
		return FormatHyphen, nil
	}
	if macDotRegex.MatchString(mac) {
		return FormatDot, nil
	}

	// This should never happen if IsValid is true
	return -1, ErrInvalidMAC
}

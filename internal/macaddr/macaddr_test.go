package macaddr

import (
	"testing"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		name        string
		mac         string
		expected    string
		expectError bool
	}{
		{
			name:        "colon format",
			mac:         "00:11:22:33:44:55",
			expected:    "001122334455",
			expectError: false,
		},
		{
			name:        "hyphen format",
			mac:         "00-11-22-33-44-55",
			expected:    "001122334455",
			expectError: false,
		},
		{
			name:        "dot format",
			mac:         "0011.2233.4455",
			expected:    "001122334455",
			expectError: false,
		},
		{
			name:        "no separator",
			mac:         "001122334455",
			expected:    "001122334455",
			expectError: false,
		},
		{
			name:        "uppercase",
			mac:         "00:11:22:AA:BB:CC",
			expected:    "001122aabbcc",
			expectError: false,
		},
		{
			name:        "mixed case",
			mac:         "00:11:22:Aa:Bb:Cc",
			expected:    "001122aabbcc",
			expectError: false,
		},
		{
			name:        "invalid format",
			mac:         "00:11:22:33:44",
			expected:    "",
			expectError: true,
		},
		{
			name:        "invalid characters",
			mac:         "00:11:22:33:44:ZZ",
			expected:    "",
			expectError: true,
		},
		{
			name:        "empty string",
			mac:         "",
			expected:    "",
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Normalize(tc.mac)

			if tc.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}

			if !tc.expectError && err != nil {
				t.Errorf("Did not expect error but got: %v", err)
			}

			if result != tc.expected {
				t.Errorf("Expected %q but got %q", tc.expected, result)
			}
		})
	}
}

func TestFormat(t *testing.T) {
	tests := []struct {
		name        string
		mac         string
		format      int
		expected    string
		expectError bool
	}{
		{
			name:        "to no separator",
			mac:         "00:11:22:33:44:55",
			format:      FormatNone,
			expected:    "001122334455",
			expectError: false,
		},
		{
			name:        "to colon format",
			mac:         "001122334455",
			format:      FormatColon,
			expected:    "00:11:22:33:44:55",
			expectError: false,
		},
		{
			name:        "to hyphen format",
			mac:         "00:11:22:33:44:55",
			format:      FormatHyphen,
			expected:    "00-11-22-33-44-55",
			expectError: false,
		},
		{
			name:        "to dot format",
			mac:         "00:11:22:33:44:55",
			format:      FormatDot,
			expected:    "0011.2233.4455",
			expectError: false,
		},
		{
			name:        "invalid MAC",
			mac:         "00:11:22:33:44",
			format:      FormatColon,
			expected:    "",
			expectError: true,
		},
		{
			name:        "invalid format",
			mac:         "00:11:22:33:44:55",
			format:      99,
			expected:    "",
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Format(tc.mac, tc.format)

			if tc.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}

			if !tc.expectError && err != nil {
				t.Errorf("Did not expect error but got: %v", err)
			}

			if result != tc.expected {
				t.Errorf("Expected %q but got %q", tc.expected, result)
			}
		})
	}
}

func TestIsValid(t *testing.T) {
	tests := []struct {
		name     string
		mac      string
		expected bool
	}{
		{
			name:     "valid colon format",
			mac:      "00:11:22:33:44:55",
			expected: true,
		},
		{
			name:     "valid hyphen format",
			mac:      "00-11-22-33-44-55",
			expected: true,
		},
		{
			name:     "valid dot format",
			mac:      "0011.2233.4455",
			expected: true,
		},
		{
			name:     "valid no separator",
			mac:      "001122334455",
			expected: true,
		},
		{
			name:     "invalid length",
			mac:      "00:11:22:33:44",
			expected: false,
		},
		{
			name:     "invalid characters",
			mac:      "00:11:22:33:44:ZZ",
			expected: false,
		},
		{
			name:     "invalid format (mixed separators)",
			mac:      "00:11-22:33-44:55",
			expected: false,
		},
		{
			name:     "empty string",
			mac:      "",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := IsValid(tc.mac)
			if result != tc.expected {
				t.Errorf("Expected %v but got %v for %q", tc.expected, result, tc.mac)
			}
		})
	}
}

func TestIsValidWithFormat(t *testing.T) {
	tests := []struct {
		name     string
		mac      string
		format   int
		expected bool
	}{
		{
			name:     "valid no separator",
			mac:      "001122334455",
			format:   FormatNone,
			expected: true,
		},
		{
			name:     "valid colon format",
			mac:      "00:11:22:33:44:55",
			format:   FormatColon,
			expected: true,
		},
		{
			name:     "valid hyphen format",
			mac:      "00-11-22-33-44-55",
			format:   FormatHyphen,
			expected: true,
		},
		{
			name:     "valid dot format",
			mac:      "0011.2233.4455",
			format:   FormatDot,
			expected: true,
		},
		{
			name:     "wrong format (colon, expected none)",
			mac:      "00:11:22:33:44:55",
			format:   FormatNone,
			expected: false,
		},
		{
			name:     "wrong format (none, expected colon)",
			mac:      "001122334455",
			format:   FormatColon,
			expected: false,
		},
		{
			name:     "invalid format",
			mac:      "00:11:22:33:44:55",
			format:   99,
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := IsValidWithFormat(tc.mac, tc.format)
			if result != tc.expected {
				t.Errorf("Expected %v but got %v for %q with format %d",
					tc.expected, result, tc.mac, tc.format)
			}
		})
	}
}

func TestEqual(t *testing.T) {
	tests := []struct {
		name     string
		mac1     string
		mac2     string
		expected bool
	}{
		{
			name:     "same format",
			mac1:     "00:11:22:33:44:55",
			mac2:     "00:11:22:33:44:55",
			expected: true,
		},
		{
			name:     "different formats",
			mac1:     "00:11:22:33:44:55",
			mac2:     "00-11-22-33-44-55",
			expected: true,
		},
		{
			name:     "different case",
			mac1:     "00:11:22:AA:BB:CC",
			mac2:     "00:11:22:aa:bb:cc",
			expected: true,
		},
		{
			name:     "completely different",
			mac1:     "00:11:22:33:44:55",
			mac2:     "AA:BB:CC:DD:EE:FF",
			expected: false,
		},
		{
			name:     "one invalid",
			mac1:     "00:11:22:33:44:55",
			mac2:     "invalid",
			expected: false,
		},
		{
			name:     "both invalid",
			mac1:     "invalid1",
			mac2:     "invalid2",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := Equal(tc.mac1, tc.mac2)
			if result != tc.expected {
				t.Errorf("Expected %v but got %v for %q and %q",
					tc.expected, result, tc.mac1, tc.mac2)
			}
		})
	}
}

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name        string
		mac         string
		expected    int
		expectError bool
	}{
		{
			name:        "no separator",
			mac:         "001122334455",
			expected:    FormatNone,
			expectError: false,
		},
		{
			name:        "colon format",
			mac:         "00:11:22:33:44:55",
			expected:    FormatColon,
			expectError: false,
		},
		{
			name:        "hyphen format",
			mac:         "00-11-22-33-44-55",
			expected:    FormatHyphen,
			expectError: false,
		},
		{
			name:        "dot format",
			mac:         "0011.2233.4455",
			expected:    FormatDot,
			expectError: false,
		},
		{
			name:        "invalid format",
			mac:         "00:11-22:33-44:55",
			expected:    -1,
			expectError: true,
		},
		{
			name:        "empty string",
			mac:         "",
			expected:    -1,
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := DetectFormat(tc.mac)

			if tc.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}

			if !tc.expectError && err != nil {
				t.Errorf("Did not expect error but got: %v", err)
			}

			if result != tc.expected {
				t.Errorf("Expected %d but got %d for %q", tc.expected, result, tc.mac)
			}
		})
	}
}

func TestNormalizeOrEmpty(t *testing.T) {
	tests := []struct {
		name     string
		mac      string
		expected string
	}{
		{
			name:     "valid MAC",
			mac:      "00:11:22:33:44:55",
			expected: "001122334455",
		},
		{
			name:     "invalid MAC",
			mac:      "invalid",
			expected: "",
		},
		{
			name:     "empty string",
			mac:      "",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := NormalizeOrEmpty(tc.mac)
			if result != tc.expected {
				t.Errorf("Expected %q but got %q for %q", tc.expected, result, tc.mac)
			}
		})
	}
}

func TestMustNormalize(t *testing.T) {
	tests := []struct {
		name        string
		mac         string
		expected    string
		expectPanic bool
	}{
		{
			name:        "valid MAC",
			mac:         "00:11:22:33:44:55",
			expected:    "001122334455",
			expectPanic: false,
		},
		{
			name:        "invalid MAC",
			mac:         "invalid",
			expected:    "",
			expectPanic: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("Expected panic but got none")
					}
				}()
			}

			result := MustNormalize(tc.mac)
			if !tc.expectPanic && result != tc.expected {
				t.Errorf("Expected %q but got %q for %q", tc.expected, result, tc.mac)
			}
		})
	}
}

func TestNormalizeFast(t *testing.T) {
	tests := []struct {
		name     string
		mac      string
		expected string
	}{
		{
			name:     "colon format",
			mac:      "00:11:22:33:44:55",
			expected: "001122334455",
		},
		{
			name:     "hyphen format",
			mac:      "00-11-22-33-44-55",
			expected: "001122334455",
		},
		{
			name:     "dot format",
			mac:      "0011.2233.4455",
			expected: "001122334455",
		},
		{
			name:     "no separator",
			mac:      "001122334455",
			expected: "001122334455",
		},
		{
			name:     "uppercase",
			mac:      "00:11:22:AA:BB:CC",
			expected: "001122aabbcc",
		},
		{
			name:     "mixed case",
			mac:      "00:11:22:Aa:Bb:Cc",
			expected: "001122aabbcc",
		},
		{
			name:     "empty string",
			mac:      "",
			expected: "",
		},
		{
			name:     "invalid format - no validation",
			mac:      "invalid",
			expected: "invalid", // NormalizeFast does NOT validate
		},
		{
			name:     "short MAC - no validation",
			mac:      "00:11:22",
			expected: "001122", // NormalizeFast does NOT validate length
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := NormalizeFast(tc.mac)
			if result != tc.expected {
				t.Errorf("Expected %q but got %q for %q", tc.expected, result, tc.mac)
			}
		})
	}
}

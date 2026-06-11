package utils

import (
	"testing"
)

func TestMaskString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Short string (<=8 chars)",
			input:    "short",
			expected: "********",
		},
		{
			name:     "Exact 8 chars",
			input:    "12345678",
			expected: "********",
		},
		{
			name:     "Long string",
			input:    "a-very-long-api-token-that-should-be-masked",
			expected: "a-ve***************************************",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "********",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := MaskString(tc.input)
			if result != tc.expected {
				t.Errorf("MaskString(%s): expected '%s', got '%s'", tc.input, tc.expected, result)
			}
		})
	}
}

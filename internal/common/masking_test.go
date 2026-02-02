package common

import "testing"

func TestMaskString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Short string",
			input:    "1234",
			expected: "********",
		},
		{
			name:     "Exact boundary",
			input:    "12345678",
			expected: "********",
		},
		{
			name:     "Regular token",
			input:    "abcdefghijklmnop",
			expected: "abcd************",
		},
		{
			name:     "Long token",
			input:    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ",
			expected: "eyJh***********************************************************************************************************",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "********",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := MaskString(test.input)
			if result != test.expected {
				t.Errorf("MaskString(%q) = %q, want %q", test.input, result, test.expected)
			}
		})
	}
}

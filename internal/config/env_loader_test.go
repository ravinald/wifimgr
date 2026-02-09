package config

import "testing"

func TestUnquoteEnvValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Unquoted values
		{
			name:     "unquoted simple",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "unquoted with spaces",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "single char",
			input:    "x",
			expected: "x",
		},

		// Double-quoted values
		{
			name:     "double quoted simple",
			input:    `"hello"`,
			expected: "hello",
		},
		{
			name:     "double quoted with spaces",
			input:    `"hello world"`,
			expected: "hello world",
		},
		{
			name:     "double quoted with single quote inside",
			input:    `"it's mine"`,
			expected: "it's mine",
		},
		{
			name:     "double quoted with escaped double quote",
			input:    `"say \"hello\""`,
			expected: `say "hello"`,
		},
		{
			name:     "double quoted with escaped backslash",
			input:    `"path\\to\\file"`,
			expected: `path\to\file`,
		},
		{
			name:     "double quoted empty",
			input:    `""`,
			expected: "",
		},

		// Single-quoted values
		{
			name:     "single quoted simple",
			input:    `'hello'`,
			expected: "hello",
		},
		{
			name:     "single quoted with spaces",
			input:    `'hello world'`,
			expected: "hello world",
		},
		{
			name:     "single quoted with double quote inside",
			input:    `'say "hello"'`,
			expected: `say "hello"`,
		},
		{
			name:     "single quoted with escaped single quote",
			input:    `'it\'s mine'`,
			expected: "it's mine",
		},
		{
			name:     "single quoted empty",
			input:    `''`,
			expected: "",
		},

		// Escape sequences
		{
			name:     "escaped newline",
			input:    `"line1\nline2"`,
			expected: "line1\nline2",
		},
		{
			name:     "escaped tab",
			input:    `"col1\tcol2"`,
			expected: "col1\tcol2",
		},
		{
			name:     "unknown escape kept as-is",
			input:    `"hello\xworld"`,
			expected: `hello\xworld`,
		},

		// Edge cases
		{
			name:     "mismatched quotes not stripped",
			input:    `"hello'`,
			expected: `"hello'`,
		},
		{
			name:     "quote only at start not stripped",
			input:    `"hello`,
			expected: `"hello`,
		},
		{
			name:     "quote only at end not stripped",
			input:    `hello"`,
			expected: `hello"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := unquoteEnvValue(tt.input)
			if result != tt.expected {
				t.Errorf("unquoteEnvValue(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

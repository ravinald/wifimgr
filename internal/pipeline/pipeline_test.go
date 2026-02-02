package pipeline

import (
	"strings"
	"testing"
)

func TestParsePipeline(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected Pipeline
		hasError bool
	}{
		{
			name: "simple command without pipes",
			args: []string{"show", "sites"},
			expected: Pipeline{
				BaseCommand:  []string{"show", "sites"},
				PipeCommands: nil,
			},
			hasError: false,
		},
		{
			name: "command with single pipe",
			args: []string{"show", "sites", "|", "grep", "SFO"},
			expected: Pipeline{
				BaseCommand: []string{"show", "sites"},
				PipeCommands: []PipeCommand{
					{Name: "grep", Args: []string{"SFO"}},
				},
			},
			hasError: false,
		},
		{
			name: "command with multiple pipes",
			args: []string{"show", "sites", "|", "grep", "SFO", "|", "head", "-n", "5"},
			expected: Pipeline{
				BaseCommand: []string{"show", "sites"},
				PipeCommands: []PipeCommand{
					{Name: "grep", Args: []string{"SFO"}},
					{Name: "head", Args: []string{"-n", "5"}},
				},
			},
			hasError: false,
		},
		{
			name:     "empty command",
			args:     []string{},
			expected: Pipeline{},
			hasError: true,
		},
		{
			name:     "command ending with pipe",
			args:     []string{"show", "sites", "|"},
			expected: Pipeline{},
			hasError: true,
		},
		{
			name:     "command starting with pipe",
			args:     []string{"|", "grep", "test"},
			expected: Pipeline{},
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParsePipeline(tt.args)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Compare base command
			if len(result.BaseCommand) != len(tt.expected.BaseCommand) {
				t.Errorf("Base command length mismatch. Expected %d, got %d", len(tt.expected.BaseCommand), len(result.BaseCommand))
				return
			}
			for i, cmd := range tt.expected.BaseCommand {
				if result.BaseCommand[i] != cmd {
					t.Errorf("Base command[%d] mismatch. Expected %s, got %s", i, cmd, result.BaseCommand[i])
				}
			}

			// Compare pipe commands
			if len(result.PipeCommands) != len(tt.expected.PipeCommands) {
				t.Errorf("Pipe commands length mismatch. Expected %d, got %d", len(tt.expected.PipeCommands), len(result.PipeCommands))
				return
			}
			for i, cmd := range tt.expected.PipeCommands {
				if result.PipeCommands[i].Name != cmd.Name {
					t.Errorf("Pipe command[%d] name mismatch. Expected %s, got %s", i, cmd.Name, result.PipeCommands[i].Name)
				}
				if len(result.PipeCommands[i].Args) != len(cmd.Args) {
					t.Errorf("Pipe command[%d] args length mismatch. Expected %d, got %d", i, len(cmd.Args), len(result.PipeCommands[i].Args))
					continue
				}
				for j, arg := range cmd.Args {
					if result.PipeCommands[i].Args[j] != arg {
						t.Errorf("Pipe command[%d] arg[%d] mismatch. Expected %s, got %s", i, j, arg, result.PipeCommands[i].Args[j])
					}
				}
			}
		})
	}
}

func TestHasPipeline(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{
			name:     "command with pipe",
			args:     []string{"show", "sites", "|", "grep", "test"},
			expected: true,
		},
		{
			name:     "command without pipe",
			args:     []string{"show", "sites"},
			expected: false,
		},
		{
			name:     "empty command",
			args:     []string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasPipeline(tt.args)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGrepProcessor(t *testing.T) {
	processor := &GrepProcessor{}

	tests := []struct {
		name     string
		input    string
		args     []string
		expected string
		hasError bool
	}{
		{
			name:     "simple grep",
			input:    "line1\nline2 test\nline3\nline4 test again",
			args:     []string{"test"},
			expected: "line2 test\nline4 test again\n",
			hasError: false,
		},
		{
			name:     "grep with -v flag",
			input:    "line1\nline2 test\nline3\nline4 test again",
			args:     []string{"-v", "test"},
			expected: "line1\nline3\n",
			hasError: false,
		},
		{
			name:     "regex pattern",
			input:    "test123\ntest456\nabc123\ntest789",
			args:     []string{"test\\d+"},
			expected: "test123\ntest456\ntest789\n",
			hasError: false,
		},
		{
			name:     "no matches",
			input:    "line1\nline2\nline3",
			args:     []string{"nomatch"},
			expected: "",
			hasError: false,
		},
		{
			name:     "missing pattern",
			input:    "line1\nline2",
			args:     []string{},
			expected: "",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := strings.NewReader(tt.input)
			result, err := processor.Process(input, tt.args)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			output, err := ReaderToString(result)
			if err != nil {
				t.Errorf("Error reading result: %v", err)
				return
			}

			if output != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, output)
			}
		})
	}
}

func TestHeadProcessor(t *testing.T) {
	processor := &HeadProcessor{}

	tests := []struct {
		name     string
		input    string
		args     []string
		expected string
		hasError bool
	}{
		{
			name:     "default head",
			input:    "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10\nline11\nline12",
			args:     []string{},
			expected: "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10\n",
			hasError: false,
		},
		{
			name:     "head with -n flag",
			input:    "line1\nline2\nline3\nline4\nline5",
			args:     []string{"-n", "3"},
			expected: "line1\nline2\nline3\n",
			hasError: false,
		},
		{
			name:     "head with number",
			input:    "line1\nline2\nline3\nline4\nline5",
			args:     []string{"2"},
			expected: "line1\nline2\n",
			hasError: false,
		},
		{
			name:     "fewer lines than requested",
			input:    "line1\nline2",
			args:     []string{"5"},
			expected: "line1\nline2\n",
			hasError: false,
		},
		{
			name:     "invalid number",
			input:    "line1\nline2",
			args:     []string{"invalid"},
			expected: "",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := strings.NewReader(tt.input)
			result, err := processor.Process(input, tt.args)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			output, err := ReaderToString(result)
			if err != nil {
				t.Errorf("Error reading result: %v", err)
				return
			}

			if output != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, output)
			}
		})
	}
}

func TestTailProcessor(t *testing.T) {
	processor := &TailProcessor{}

	tests := []struct {
		name     string
		input    string
		args     []string
		expected string
		hasError bool
	}{
		{
			name:     "default tail",
			input:    "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10\nline11\nline12",
			args:     []string{},
			expected: "line3\nline4\nline5\nline6\nline7\nline8\nline9\nline10\nline11\nline12\n",
			hasError: false,
		},
		{
			name:     "tail with -n flag",
			input:    "line1\nline2\nline3\nline4\nline5",
			args:     []string{"-n", "3"},
			expected: "line3\nline4\nline5\n",
			hasError: false,
		},
		{
			name:     "tail with number",
			input:    "line1\nline2\nline3\nline4\nline5",
			args:     []string{"2"},
			expected: "line4\nline5\n",
			hasError: false,
		},
		{
			name:     "fewer lines than requested",
			input:    "line1\nline2",
			args:     []string{"5"},
			expected: "line1\nline2\n",
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := strings.NewReader(tt.input)
			result, err := processor.Process(input, tt.args)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			output, err := ReaderToString(result)
			if err != nil {
				t.Errorf("Error reading result: %v", err)
				return
			}

			if output != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, output)
			}
		})
	}
}

package help

import (
	"fmt"
	"strings"
)

// Formatter defines the interface for formatting help text
type Formatter interface {
	// FormatCommand returns formatted help text for a command
	FormatCommand(cmd *CommandDescriptor) string

	// FormatCommandList returns formatted help text for a list of commands
	FormatCommandList(cmds []*CommandDescriptor, header string) string

	// FormatRootHelp returns formatted help text for the root level
	FormatRootHelp(registry *Registry) string
}

// CLIFormatter formats help text for command line usage
type CLIFormatter struct {
	IndentWidth int
}

// NewCLIFormatter creates a new CLI formatter
func NewCLIFormatter() *CLIFormatter {
	return &CLIFormatter{
		IndentWidth: 2,
	}
}

// FormatCommand formats a command's help for CLI display
func (f *CLIFormatter) FormatCommand(cmd *CommandDescriptor) string {
	var sb strings.Builder

	// Command name and description
	sb.WriteString(fmt.Sprintf("%s - %s\n\n", cmd.Name, cmd.Description))

	// Usage
	if cmd.Usage != "" {
		sb.WriteString(fmt.Sprintf("Usage: %s\n\n", cmd.Usage))
	}

	// Examples
	if len(cmd.Examples) > 0 {
		sb.WriteString("Examples:\n")
		for _, example := range cmd.Examples {
			sb.WriteString(fmt.Sprintf("%s%s\n", strings.Repeat(" ", f.IndentWidth), example))
		}
		sb.WriteString("\n")
	}

	// Subcommands
	if len(cmd.SubCommands) > 0 {
		sb.WriteString("Available subcommands:\n")
		for _, sub := range cmd.SubCommands {
			sb.WriteString(fmt.Sprintf("%s%-15s %s\n",
				strings.Repeat(" ", f.IndentWidth),
				sub.Name,
				sub.Description))
		}
	}

	return sb.String()
}

// FormatCommandList formats a list of commands for CLI display
func (f *CLIFormatter) FormatCommandList(cmds []*CommandDescriptor, header string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s:\n", header))

	for _, cmd := range cmds {
		sb.WriteString(fmt.Sprintf("%s%-15s %s\n",
			strings.Repeat(" ", f.IndentWidth),
			cmd.Name,
			cmd.Description))
	}

	return sb.String()
}

// FormatRootHelp formats the root help display
func (f *CLIFormatter) FormatRootHelp(registry *Registry) string {
	var sb strings.Builder

	sb.WriteString("WiFi Manager CLI\n\n")
	sb.WriteString("Available commands:\n")

	for _, cmd := range registry.Commands {
		sb.WriteString(fmt.Sprintf("%s%-15s %s\n",
			strings.Repeat(" ", f.IndentWidth),
			cmd.Name,
			cmd.Description))
	}

	sb.WriteString("\nUse 'mm help <command>' for more information about a command.\n")

	return sb.String()
}

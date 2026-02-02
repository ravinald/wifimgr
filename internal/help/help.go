package help

import (
	"strings"
)

var (
	cliFormatter Formatter = NewCLIFormatter()
)

// GetCLIHelp returns formatted help text for CLI usage
func GetCLIHelp(args []string) string {
	return GetHelpText(cliFormatter, args)
}

// FindCommandSuggestions returns command suggestions for a partial input
func FindCommandSuggestions(partial string) []string {
	suggestions := []string{}

	// Check top-level commands
	for _, cmd := range DefaultRegistry.Commands {
		if strings.HasPrefix(cmd.Name, partial) {
			suggestions = append(suggestions, cmd.Name)
		}
		for _, alias := range cmd.Aliases {
			if strings.HasPrefix(alias, partial) {
				suggestions = append(suggestions, alias)
			}
		}
	}

	return suggestions
}

// FindSubCommandSuggestions returns subcommand suggestions for a command and partial input
func FindSubCommandSuggestions(cmdPath []string, partial string) []string {
	suggestions := []string{}

	cmd := GetHelpForCommand(cmdPath)
	if cmd == nil {
		return suggestions
	}

	for _, sub := range cmd.SubCommands {
		if strings.HasPrefix(sub.Name, partial) {
			suggestions = append(suggestions, sub.Name)
		}
		for _, alias := range sub.Aliases {
			if strings.HasPrefix(alias, partial) {
				suggestions = append(suggestions, alias)
			}
		}
	}

	return suggestions
}

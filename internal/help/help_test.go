package help

import (
	"strings"
	"testing"
)

func TestCommandRegistry(t *testing.T) {
	// Initialize commands
	InitCommands()

	// Test finding a top-level command
	showCmd := DefaultRegistry.FindCommand("show")
	if showCmd == nil {
		t.Fatal("Failed to find 'show' command")
	}
	if showCmd.Name != "show" {
		t.Errorf("Expected name 'show', got '%s'", showCmd.Name)
	}

	// Test finding a top-level command by alias
	showCmdByAlias := DefaultRegistry.FindCommand("s")
	if showCmdByAlias == nil {
		t.Fatal("Failed to find 'show' command by alias 's'")
	}
	if showCmdByAlias.Name != "show" {
		t.Errorf("Expected name 'show', got '%s'", showCmdByAlias.Name)
	}

	// Test finding a subcommand (api command)
	apiCmd := showCmd.FindSubCommand("api")
	if apiCmd == nil {
		t.Fatal("Failed to find 'api' subcommand")
	}
	if apiCmd.Name != "api" {
		t.Errorf("Expected name 'api', got '%s'", apiCmd.Name)
	}

	// Test navigation from a subcommand to parent
	if apiCmd.ParentCmd != showCmd {
		t.Error("Subcommand doesn't correctly reference parent")
	}
}

func TestFormatters(t *testing.T) {
	InitCommands()

	// Test CLI formatter
	cli := NewCLIFormatter()
	rootHelp := cli.FormatRootHelp(DefaultRegistry)

	if !strings.Contains(rootHelp, "Available commands:") {
		t.Error("CLI formatter doesn't include expected header")
	}

	if !strings.Contains(rootHelp, "show") {
		t.Error("CLI formatter doesn't include 'show' command")
	}

	// Test command formatting
	showCmd := DefaultRegistry.FindCommand("show")
	showHelp := cli.FormatCommand(showCmd)

	if !strings.Contains(showHelp, "show - Display information") {
		t.Error("CLI formatter doesn't format command correctly")
	}

	if !strings.Contains(showHelp, "Available subcommands:") {
		t.Error("CLI formatter doesn't include subcommands section")
	}

}

func TestGetHelpForCommand(t *testing.T) {
	InitCommands()

	// Test getting help for single command
	showCmd := GetHelpForCommand([]string{"show"})
	if showCmd == nil {
		t.Fatal("Failed to get help for 'show' command")
	}
	if showCmd.Name != "show" {
		t.Errorf("Expected name 'show', got '%s'", showCmd.Name)
	}

	// Test getting help for command path
	apSiteCmd := GetHelpForCommand([]string{"set", "ap", "site"})
	if apSiteCmd == nil {
		t.Fatal("Failed to get help for 'set ap site' command path")
	}
	if apSiteCmd.Name != "site" {
		t.Errorf("Expected name 'site', got '%s'", apSiteCmd.Name)
	}

	// Test getting help for unknown command
	unknownCmd := GetHelpForCommand([]string{"unknown"})
	if unknownCmd != nil {
		t.Error("Getting help for unknown command should return nil")
	}
}

func TestHelpSuggestions(t *testing.T) {
	InitCommands()

	// Test command suggestions
	suggestions := FindCommandSuggestions("s")

	found := false
	for _, s := range suggestions {
		if s == "show" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Suggestions for 's' should include 'show'")
	}

	// Test subcommand suggestions
	subSuggestions := FindSubCommandSuggestions([]string{"show"}, "a")

	found = false
	for _, s := range subSuggestions {
		if s == "api" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Subcommand suggestions for 'show a' should include 'api'")
	}
}

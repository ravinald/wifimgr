package help

import (
	"fmt"
)

// DefaultRegistry contains the default command registry for the application
var DefaultRegistry *Registry

// InitCommands initializes the default command registry with all commands
func InitCommands() {
	DefaultRegistry = NewRegistry()

	// Define all top-level commands
	showCmd := &CommandDescriptor{
		Name:        "show",
		Aliases:     []string{"s", "sh"},
		Description: "Display information about resources",
		Usage:       "show <resource> [options]",
		Examples: []string{
			"show api sites",
			"show intent site",
			"show aps US-SFO-TESTDRY",
			"show inventory",
		},
	}

	setCmd := &CommandDescriptor{
		Name:        "set",
		Description: "Set or assign values to resources",
		Usage:       "set <resource> <property> <value> [options]",
		Examples: []string{
			"set ap site US-SFO-TESTDRY",
			"set ap site -f devices.json -s US-SFO-TESTDRY",
		},
	}

	apCmd := &CommandDescriptor{
		Name:        "ap",
		Description: "Manage access points",
		Usage:       "ap <action> [options]",
		Examples: []string{
			"ap list US-SFO-TESTDRY",
			"ap detail 11:22:33:44:55:66",
		},
	}

	applyCmd := &CommandDescriptor{
		Name:        "apply",
		Description: "Apply configurations to sites",
		Usage:       "apply <site-name> <resource-type> [options]",
		Examples: []string{
			"apply US-SFO-TESTDRY ap",
			"apply US-SFO-TESTDRY switch",
		},
	}

	helpCmd := &CommandDescriptor{
		Name:        "help",
		Aliases:     []string{"?", "h"},
		Description: "Display help information",
		Usage:       "help [command]",
		Examples: []string{
			"help",
			"help show",
			"help set ap",
		},
	}

	// Add them to registry
	// Create the refresh command
	refreshCmd := &CommandDescriptor{
		Name:        "refresh",
		Description: "Refresh cached data from the API",
		Usage:       "refresh <resource> [options]",
		Examples: []string{
			"refresh cache",
			"refresh cache ap",
			"refresh cache switch",
			"refresh cache gateway",
		},
	}

	// Define subcommands for 'refresh'
	refreshCmd.AddSubCommand(&CommandDescriptor{
		Name:        "cache",
		Description: "Refresh the cache for a specific device type or all",
		Usage:       "refresh cache [type]",
	})

	// Define device type options for 'refresh cache'
	cacheSubCmd := refreshCmd.FindSubCommand("cache")
	cacheSubCmd.AddSubCommand(&CommandDescriptor{
		Name:        "all",
		Description: "Refresh all device types (default)",
		Usage:       "refresh cache all",
	})
	cacheSubCmd.AddSubCommand(&CommandDescriptor{
		Name:        "ap",
		Description: "Refresh AP cache",
		Usage:       "refresh cache ap",
	})
	cacheSubCmd.AddSubCommand(&CommandDescriptor{
		Name:        "switch",
		Description: "Refresh switch cache",
		Usage:       "refresh cache switch",
	})
	cacheSubCmd.AddSubCommand(&CommandDescriptor{
		Name:        "gateway",
		Description: "Refresh gateway cache",
		Usage:       "refresh cache gateway",
	})

	// Create the search command
	searchCmd := &CommandDescriptor{
		Name:        "search",
		Description: "Search for clients in the Mist network",
		Usage:       "search <client-type> <search-text>",
		Examples: []string{
			"search wired 00:11:22:33:44:55",
			"search wireless client-hostname",
		},
	}

	// Define subcommands for 'search'
	searchCmd.AddSubCommand(&CommandDescriptor{
		Name:        "wired",
		Description: "Search for wired clients in the network",
		Usage:       "search wired <search-text>",
		Examples: []string{
			"search wired 00:11:22:33:44:55",
			"search wired floor-2-device",
		},
	})

	searchCmd.AddSubCommand(&CommandDescriptor{
		Name:        "wireless",
		Description: "Search for wireless clients in the network",
		Usage:       "search wireless <search-text>",
		Examples: []string{
			"search wireless iphone-device",
			"search wireless conference-room",
		},
	})

	DefaultRegistry.AddCommand(showCmd)
	DefaultRegistry.AddCommand(setCmd)
	DefaultRegistry.AddCommand(apCmd)
	DefaultRegistry.AddCommand(applyCmd)
	DefaultRegistry.AddCommand(refreshCmd)
	DefaultRegistry.AddCommand(searchCmd)
	DefaultRegistry.AddCommand(helpCmd)

	// Define subcommands for 'show'
	apiCmd := &CommandDescriptor{
		Name:        "api",
		Description: "Group of API-based commands that show active data from the API",
		Usage:       "show api [site|ap|switch|gateway]",
	}
	showCmd.AddSubCommand(apiCmd)

	// Add subcommands to api
	apiSiteCmd := &CommandDescriptor{
		Name:        "sites",
		Description: "Show site configuration",
		Usage:       "show api sites [site-name|all] [ap|switch|gateway|all]",
	}
	apiCmd.AddSubCommand(apiSiteCmd)

	// Add device type options to site command
	apiSiteCmd.AddSubCommand(&CommandDescriptor{
		Name:        "ap",
		Description: "Show APs for a specific site",
		Usage:       "show api sites <site-name> ap",
	})

	apiSiteCmd.AddSubCommand(&CommandDescriptor{
		Name:        "switch",
		Description: "Show switches for a specific site",
		Usage:       "show api sites <site-name> switch",
	})

	apiSiteCmd.AddSubCommand(&CommandDescriptor{
		Name:        "gateway",
		Description: "Show gateways for a specific site",
		Usage:       "show api sites <site-name> gateway",
	})

	apiSiteCmd.AddSubCommand(&CommandDescriptor{
		Name:        "all",
		Description: "Show all devices for a specific site",
		Usage:       "show api sites <site-name> all",
	})

	apiCmd.AddSubCommand(&CommandDescriptor{
		Name:        "ap",
		Description: "Show AP configuration",
		Usage:       "show api ap [mac|name|all] [detail]",
	})

	apiCmd.AddSubCommand(&CommandDescriptor{
		Name:        "switch",
		Description: "Show switch configuration",
		Usage:       "show api switch [mac|name|all] [detail]",
	})

	apiCmd.AddSubCommand(&CommandDescriptor{
		Name:        "gateway",
		Description: "Show gateway configuration",
		Usage:       "show api gateway [mac|name|all] [detail]",
	})

	apiCmd.AddSubCommand(&CommandDescriptor{
		Name:        "inventory",
		Description: "Show inventory items",
		Usage:       "show api inventory [ap|switch|gateway]",
	})

	// Create intent command with aliases
	intentCmd := &CommandDescriptor{
		Name:        "intent",
		Description: "Group of intent-based commands that show data from local config",
		Usage:       "show intent [inventory|ap|switch|gateway]",
	}
	showCmd.AddSubCommand(intentCmd)

	// Add subcommands to intent
	intentCmd.AddSubCommand(&CommandDescriptor{
		Name:        "site",
		Description: "List all sites or show details for a specific site",
		Usage:       "show intent site [site-name] [ap|switch|gateway|all]",
	})

	intentCmd.AddSubCommand(&CommandDescriptor{
		Name:        "aps",
		Description: "List APs in a site",
		Usage:       "show intent aps <site-name>",
	})

	intentCmd.AddSubCommand(&CommandDescriptor{
		Name:        "ap",
		Description: "Show AP details",
		Usage:       "show intent ap <ap-name-or-mac>",
	})

	intentCmd.AddSubCommand(&CommandDescriptor{
		Name:        "inventory",
		Description: "Show inventory information",
		Usage:       "show intent inventory [ap|switch|gateway]",
	})

	// Add inventory subcommands
	inventoryCmd := &CommandDescriptor{
		Name:        "inventory",
		Description: "Show inventory information",
		Usage:       "show inventory [ap|switch|gateway]",
	}
	showCmd.AddSubCommand(inventoryCmd)

	// Add device-specific inventory commands
	inventoryCmd.AddSubCommand(&CommandDescriptor{
		Name:        "ap",
		Description: "Show AP inventory items",
		Usage:       "show inventory ap",
	})

	inventoryCmd.AddSubCommand(&CommandDescriptor{
		Name:        "switch",
		Description: "Show switch inventory items",
		Usage:       "show inventory switch",
	})

	inventoryCmd.AddSubCommand(&CommandDescriptor{
		Name:        "gateway",
		Description: "Show gateway inventory items",
		Usage:       "show inventory gateway",
	})

	showCmd.AddSubCommand(&CommandDescriptor{
		Name:        "aps",
		Description: "List APs in a site",
		Usage:       "show aps <site-name>",
	})

	showCmd.AddSubCommand(&CommandDescriptor{
		Name:        "ap",
		Description: "Show AP details",
		Usage:       "show ap [site-name] <ap-name-or-mac>",
	})

	// Define subcommands for 'set'
	setCmd.AddSubCommand(&CommandDescriptor{
		Name:        "ap",
		Description: "Set AP properties",
		Usage:       "set ap <property> <value>",
	})

	apSubCmd := setCmd.FindSubCommand("ap")
	apSubCmd.AddSubCommand(&CommandDescriptor{
		Name:        "site",
		Description: "Assign AP to a site",
		Usage:       "set ap site <site-name> | -f <file> -s <site-name>",
	})

	// Add file flag options
	fileOption := &CommandDescriptor{
		Name:        "-f",
		Description: "Specify a file with AP MAC addresses",
		Usage:       "set ap site -f <file> -s <site-name>",
	}
	apSubCmd.FindSubCommand("site").AddSubCommand(fileOption)

	siteOption := &CommandDescriptor{
		Name:        "-s",
		Description: "Specify site name for bulk assignment",
		Usage:       "set ap site -f <file> -s <site-name>",
	}
	apSubCmd.FindSubCommand("site").AddSubCommand(siteOption)

	// Define subcommands for 'apply'
	applyCmd.AddSubCommand(&CommandDescriptor{
		Name:        "help",
		Description: "Display help for apply commands",
		Usage:       "apply help",
	})

	applyCmd.AddSubCommand(&CommandDescriptor{
		Name:        "ap",
		Description: "Apply AP configurations",
		Usage:       "apply <site-name> ap [-d|--force]",
	})

	applyCmd.AddSubCommand(&CommandDescriptor{
		Name:        "switch",
		Description: "Apply switch configurations",
		Usage:       "apply <site-name> switch [-d|--force]",
	})

	applyCmd.AddSubCommand(&CommandDescriptor{
		Name:        "gateway",
		Description: "Apply gateway configurations",
		Usage:       "apply <site-name> gateway [-d|--force]",
	})

	applyCmd.AddSubCommand(&CommandDescriptor{
		Name:        "all",
		Description: "Apply all device types",
		Usage:       "apply <site-name> all [-d|--force]",
	})

	// Add common apply options
	for _, cmd := range []string{"ap", "switch", "gateway", "all"} {
		applySubCmd := applyCmd.FindSubCommand(cmd)
		if applySubCmd != nil {
			applySubCmd.AddSubCommand(&CommandDescriptor{
				Name:        "-d",
				Description: "Show differences without applying",
				Usage:       fmt.Sprintf("apply <site-name> %s -d", cmd),
			})

			applySubCmd.AddSubCommand(&CommandDescriptor{
				Name:        "--force",
				Description: "Force application even if no changes detected",
				Usage:       fmt.Sprintf("apply <site-name> %s --force", cmd),
			})
		}
	}
}

// GetHelpForCommand returns the command descriptor for a given command path
func GetHelpForCommand(args []string) *CommandDescriptor {
	if len(args) == 0 {
		return nil
	}

	// Find the top-level command
	cmd := DefaultRegistry.FindCommand(args[0])
	if cmd == nil {
		return nil
	}

	// Navigate through subcommands if provided
	current := cmd
	for i := 1; i < len(args); i++ {
		subCmd := current.FindSubCommand(args[i])
		if subCmd == nil {
			// No matching subcommand, return what we have so far
			return current
		}
		current = subCmd
	}

	return current
}

// GetHelpText returns formatted help text using the specified formatter
func GetHelpText(formatter Formatter, args []string) string {
	if len(args) == 0 {
		// Root help
		return formatter.FormatRootHelp(DefaultRegistry)
	}

	cmd := GetHelpForCommand(args)
	if cmd == nil {
		return "Unknown command. Type 'help' for available commands.\n"
	}

	return formatter.FormatCommand(cmd)
}

func init() {
	InitCommands()
}

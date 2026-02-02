package intent

import (
	"context"
	"fmt"

	"github.com/ravinald/wifimgr/api"
	"github.com/ravinald/wifimgr/cmd/inventory"
	"github.com/ravinald/wifimgr/cmd/site"
	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/logging"
)

// HandleCommand processes intent-related subcommands
func HandleCommand(ctx context.Context, client api.Client, cfg *config.Config, args []string, formatOverride string, _ bool) error {
	if len(args) < 1 {
		logging.Error("No intent subcommand specified")
		return fmt.Errorf("no intent subcommand specified")
	}

	logging.Infof("Executing intent subcommand: %s", args[0])

	switch args[0] {
	case "inventory", "inv":
		// Pass the remaining arguments to the inventory handler
		if len(args) > 1 {
			return inventory.HandleCommand(ctx, client, cfg, args[1:], formatOverride)
		}
		// If no specific inventory subcommand, default to showing all inventory
		return inventory.HandleCommand(ctx, client, cfg, []string{"show"}, formatOverride)
	case "site":
		// Handle 'show intent site' command - list all sites or get specific site
		if len(args) < 2 {
			// No site name provided - list all sites
			logging.Debug("Handling 'show intent site' command (list all)")
			return site.HandleCommand(ctx, client, []string{"intent_list"}, formatOverride)
		}
		// Site name provided - get that specific site
		logging.Debugf("Handling 'show intent site %s' command", args[1])
		return site.HandleCommand(ctx, client, []string{"get", args[1]}, formatOverride)
	default:
		logging.Errorf("Unknown intent subcommand: %s", args[0])
		return fmt.Errorf("unknown intent subcommand: %s", args[0])
	}
}

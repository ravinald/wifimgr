/*
Copyright © 2025 Ravi Pina <ravi@pina.org>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ravinald/wifimgr/internal/formatter"
	"github.com/ravinald/wifimgr/internal/integrations/netbox"
	"github.com/ravinald/wifimgr/internal/logging"
)

var exportNetboxCmd = &cobra.Command{
	Use:   "netbox <all|site> [site-name] [dry-run|validate]",
	Short: "Export AP devices to NetBox",
	Long: `Export AP inventory from wifimgr to NetBox DCIM.

This command exports AP devices from the wifimgr cache to NetBox, creating or updating
device records with associated interfaces and IP addresses. Only Access Points are exported;
switches and gateways are excluded.

Modes:
  all             Export all APs from all sites
  site <name>     Export APs from a specific site

Options:
  dry-run         Validate and show what would happen without making changes
  validate        Only validate NetBox dependencies, show missing items
  force           Skip confirmation prompt

Requirements:
  - NetBox URL and API key must be configured (via config file or environment)
  - Required NetBox objects must exist: sites, device types, device roles
  - Run 'wifimgr cache refresh' before exporting to ensure data is current

Configuration:
  Set NetBox credentials via environment variables:
    NETBOX_API_URL=https://netbox.example.com
    NETBOX_API_KEY=your-api-key
    NETBOX_SSL_VERIFY=true

  Or via ~/.env.netbox file with the same variables.

  Or via config file:
    netbox:
      url: "https://netbox.example.com"
      credentials:
        api_key: "your-api-key"
      ssl_verify: true`,
	Example: `  # Export all APs to NetBox
  wifimgr export netbox all

  # Export APs from a specific site
  wifimgr export netbox site US-LAB-01

  # Dry run - see what would be created/updated without making changes
  wifimgr export netbox all dry-run
  wifimgr export netbox site US-LAB-01 dry-run

  # Validate only - check NetBox dependencies without exporting
  wifimgr export netbox all validate
  wifimgr export netbox site US-LAB-01 validate`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("requires at least 1 argument: all or site")
		}

		mode := strings.ToLower(args[0])
		if mode != "all" && mode != "site" {
			return fmt.Errorf("first argument must be 'all' or 'site', got '%s'", args[0])
		}

		if mode == "site" && len(args) < 2 {
			return fmt.Errorf("'site' mode requires a site name argument")
		}

		return nil
	},
	RunE: runExportNetbox,
}

func runExportNetbox(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Parse arguments
	opts := netbox.ExportOptions{}
	var validateOnly bool

	mode := strings.ToLower(args[0])
	argIndex := 1

	if mode == "site" {
		if len(args) < 2 {
			return fmt.Errorf("site mode requires a site name")
		}
		opts.SiteName = args[1]
		argIndex = 2
	}

	// Parse optional modifiers
	for i := argIndex; i < len(args); i++ {
		switch strings.ToLower(args[i]) {
		case "dry-run", "dryrun":
			opts.DryRun = true
		case "validate":
			validateOnly = true
		case "force":
			opts.Force = true
		}
	}

	// Load NetBox configuration
	logging.Info("Loading NetBox configuration...")
	cfg, err := netbox.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load NetBox configuration: %w", err)
	}

	logging.Debugf("NetBox URL: %s", cfg.URL)

	// Create exporter
	exporter, err := netbox.NewExporter(cfg)
	if err != nil {
		return fmt.Errorf("failed to create exporter: %w", err)
	}

	// Validation-only mode
	if validateOnly {
		return runValidateOnly(ctx, exporter, opts)
	}

	// Regular export
	return runExport(ctx, exporter, opts)
}

func runValidateOnly(ctx context.Context, exporter *netbox.Exporter, opts netbox.ExportOptions) error {
	logging.Info("Running validation only (no changes will be made)...")

	summary, err := exporter.ValidateOnly(ctx, opts)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	fmt.Printf("\nValidation Summary\n")
	fmt.Printf("==================\n")
	fmt.Printf("Total devices:   %d\n", summary.TotalDevices)
	fmt.Printf("Valid:           %d\n", summary.ValidDevices)
	fmt.Printf("Invalid:         %d\n", summary.InvalidDevices)

	if len(summary.MissingSites) > 0 {
		fmt.Printf("\nMissing Sites in NetBox:\n")
		for site, count := range summary.MissingSites {
			fmt.Printf("  - %s (%d devices)\n", site, count)
		}
	}

	if len(summary.MissingTypes) > 0 {
		fmt.Printf("\nMissing Device Types in NetBox:\n")
		for model, count := range summary.MissingTypes {
			fmt.Printf("  - %s (%d devices)\n", model, count)
		}
	}

	if len(summary.MissingRoles) > 0 {
		fmt.Printf("\nMissing Device Roles in NetBox:\n")
		for role, count := range summary.MissingRoles {
			fmt.Printf("  - %s (%d devices)\n", role, count)
		}
	}

	if summary.InvalidDevices > 0 && len(summary.ValidationErrors) > 0 {
		fmt.Printf("\nFirst 10 Validation Errors:\n")
		limit := min(10, len(summary.ValidationErrors))
		for i := range limit {
			err := summary.ValidationErrors[i]
			fmt.Printf("  - %s (%s): %v\n", err.Name, err.MAC, err.Errors)
		}
		if len(summary.ValidationErrors) > 10 {
			fmt.Printf("  ... and %d more errors\n", len(summary.ValidationErrors)-10)
		}
	}

	return nil
}

func runExport(ctx context.Context, exporter *netbox.Exporter, opts netbox.ExportOptions) error {
	modeStr := "all sites"
	if opts.SiteName != "" {
		modeStr = fmt.Sprintf("site '%s'", opts.SiteName)
	}

	if opts.DryRun {
		logging.Infof("Dry run: simulating export of %s to NetBox...", modeStr)
	} else {
		logging.Infof("Exporting %s to NetBox...", modeStr)
	}

	result, err := exporter.Export(ctx, opts)
	if err != nil {
		return fmt.Errorf("export failed: %w", err)
	}

	// Display results
	displayExportResults(result, opts.DryRun)

	return nil
}

func displayExportResults(result *netbox.ExportResult, dryRun bool) {
	fmt.Printf("\nExport %s in %s\n", statusText(dryRun), result.Stats.Duration)
	fmt.Printf("\nSummary\n")
	fmt.Printf("=======\n")
	fmt.Printf("Total:   %d\n", result.Stats.TotalDevices)

	if dryRun {
		fmt.Printf("Would create: %d\n", result.Stats.Created)
		fmt.Printf("Would update: %d\n", result.Stats.Updated)
	} else {
		fmt.Printf("Created: %d\n", result.Stats.Created)
		fmt.Printf("Updated: %d\n", result.Stats.Updated)
	}

	fmt.Printf("Skipped: %d\n", result.Stats.Skipped)
	fmt.Printf("Errors:  %d\n", result.Stats.Errors)

	// Show created devices
	if len(result.Created) > 0 && !dryRun {
		fmt.Printf("\nCreated Devices:\n")
		displayDeviceResults(result.Created)
	}

	// Show updated devices
	if len(result.Updated) > 0 && !dryRun {
		fmt.Printf("\nUpdated Devices:\n")
		displayDeviceResults(result.Updated)
	}

	// Show skipped devices
	if len(result.Skipped) > 0 {
		fmt.Printf("\nSkipped Devices (missing dependencies):\n")
		for _, skip := range result.Skipped {
			name := skip.Name
			if name == "" {
				name = skip.MAC
			}
			fmt.Printf("  - %s: %s\n", name, skip.Reason)
		}
	}

	// Show errors
	if len(result.Errors) > 0 {
		fmt.Printf("\nErrors:\n")
		for _, e := range result.Errors {
			name := e.DeviceName
			if name == "" {
				name = e.DeviceMAC
			}
			fmt.Printf("  - %s [%s]: %s\n", name, e.Operation, e.Message)
			if e.Err != nil {
				fmt.Printf("    Error: %v\n", e.Err)
			}
		}
	}
}

func displayDeviceResults(results []netbox.DeviceExportResult) {
	tableData := make([]map[string]any, 0, len(results))
	for _, r := range results {
		row := map[string]any{
			"name":      r.Name,
			"mac":       r.MAC,
			"netbox_id": r.NetBoxID,
		}
		tableData = append(tableData, row)
	}

	columns := []formatter.SimpleColumn{
		{Header: "Name", Field: "name"},
		{Header: "MAC", Field: "mac"},
		{Header: "NetBox ID", Field: "netbox_id"},
	}

	options := formatter.SimpleTableOptions{
		BoldHeaders:   true,
		ShowSeparator: true,
	}

	tableOutput := formatter.RenderSimpleTable(tableData, columns, options)
	fmt.Print(tableOutput)
}

func statusText(dryRun bool) string {
	if dryRun {
		return "simulation completed"
	}
	return "completed"
}

func init() {
	exportCmd.AddCommand(exportNetboxCmd)
}

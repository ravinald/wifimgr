package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ravinald/wifimgr/internal/cmdutils"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/symbols"
	"github.com/ravinald/wifimgr/internal/validation"
	"github.com/ravinald/wifimgr/internal/vendors"
)

var (
	skipCompatCheck bool
)

var checkCompatibilityCmd = &cobra.Command{
	Use:   "compatibility <site-name>",
	Short: "Check API compatibility for a site configuration",
	Long: `Check if a site configuration is compatible with the target API.

This command validates that:
- Configuration fields are supported by the target API
- No deprecated fields are in use
- Vendor-specific fields match the target vendor
- Referenced resources (profiles, templates) exist
- Field types and values are API-compatible

Examples:
  wifimgr check compatibility US-LAB-01
  wifimgr check compatibility US-DC-PROD`,
	Args: cobra.ExactArgs(1),
	RunE: runCheckCompatibility,
}

func init() {
	checkCmd.AddCommand(checkCompatibilityCmd)
	checkCompatibilityCmd.Flags().BoolVar(&skipCompatCheck, "skip-compat-check", false, "Skip compatibility check")
}

func runCheckCompatibility(cmd *cobra.Command, args []string) error {
	// Check for help keyword in positional arguments
	if cmdutils.ContainsHelp(args) {
		return cmd.Help()
	}

	siteName := args[0]

	logging.Infof("Checking API compatibility for site: %s", siteName)

	// Load site configuration
	siteConfig, err := loadSiteConfiguration(siteName)
	if err != nil {
		return fmt.Errorf("failed to load site configuration: %w", err)
	}

	// Get global cache accessor
	cacheAccessor := vendors.GetGlobalCacheAccessor()

	// Create schema tracker (in production, this would be loaded from disk)
	schemaTracker := vendors.NewSchemaTracker()

	// Create compatibility checker
	checker := validation.NewCompatibilityChecker(schemaTracker, cacheAccessor)

	// Perform compatibility check
	result, err := checker.CheckSite(siteName, siteConfig)
	if err != nil {
		return fmt.Errorf("compatibility check failed: %w", err)
	}

	// Display results
	displayCompatibilityResults(result)

	// Exit with error if not compatible
	if !result.Compatible {
		return fmt.Errorf("configuration is not compatible with target API")
	}

	return nil
}

func displayCompatibilityResults(result *validation.CompatibilityResult) {
	fmt.Printf("\n")
	fmt.Printf("Compatibility Check Results\n")
	fmt.Printf("========================================\n")
	fmt.Printf("Site:        %s\n", result.SiteName)
	fmt.Printf("API Version: %s\n", result.APIVersion)
	fmt.Printf("Checked:     %s\n", result.LastChecked.Format("2006-01-02 15:04:05"))
	fmt.Printf("Status:      %s\n", result.Summary())
	fmt.Printf("\n")

	if len(result.Issues) == 0 {
		fmt.Printf("%s Configuration is fully compatible with the target API\n", symbols.SuccessPrefix())
		return
	}

	// Group issues by severity
	errors := result.FilterBySeverity("error")
	warnings := result.FilterBySeverity("warning")

	// Display errors
	if len(errors) > 0 {
		fmt.Printf("%s Errors (%d):\n", symbols.ErrorPrefix(), len(errors))
		fmt.Printf("----------------------------------------\n")
		for _, issue := range errors {
			displayCompatibilityIssue(issue)
		}
		fmt.Printf("\n")
	}

	// Display warnings
	if len(warnings) > 0 {
		fmt.Printf("%s Warnings (%d):\n", symbols.WarningPrefix(), len(warnings))
		fmt.Printf("----------------------------------------\n")
		for _, issue := range warnings {
			displayCompatibilityIssue(issue)
		}
		fmt.Printf("\n")
	}

	// Recommendations
	if !result.Compatible {
		fmt.Printf("Recommendations:\n")
		fmt.Printf("----------------------------------------\n")
		fmt.Printf("- Fix all error-level issues before applying configuration\n")
		fmt.Printf("- Review warnings to ensure expected behavior\n")
		fmt.Printf("- Use 'wifimgr lint config %s' for additional validation\n", result.SiteName)
		fmt.Printf("\n")
	}
}

func displayCompatibilityIssue(issue validation.CompatibilityIssue) {
	fmt.Printf("\n  Field: %s\n", issue.Field)
	fmt.Printf("  Issue: %s\n", issue.Message)

	if len(issue.AffectedDevices) > 0 {
		fmt.Printf("  Affected devices: ")
		if len(issue.AffectedDevices) <= 3 {
			fmt.Printf("%v\n", issue.AffectedDevices)
		} else {
			fmt.Printf("%v (and %d more)\n", issue.AffectedDevices[:3], len(issue.AffectedDevices)-3)
		}
	}

	if issue.Action != "" {
		fmt.Printf("  Action: %s\n", issue.Action)
	}
}

// displayCompatibilityResultsTable provides an alternative table-based display.
func displayCompatibilityResultsTable(result *validation.CompatibilityResult) {
	if len(result.Issues) == 0 {
		fmt.Printf("%s Configuration is fully compatible\n", symbols.SuccessPrefix())
		return
	}

	// Group by field for easier reading
	grouped := result.GroupByField()

	fmt.Printf("\nCompatibility Issues by Field:\n")
	fmt.Printf("========================================\n")

	for field, issues := range grouped {
		fmt.Printf("\nField: %s\n", field)
		fmt.Printf("----------------------------------------\n")

		for _, issue := range issues {
			severity := issue.Severity
			if severity == "error" {
				severity = fmt.Sprintf("%s ERROR", symbols.ErrorPrefix())
			} else {
				severity = fmt.Sprintf("%s WARN", symbols.WarningPrefix())
			}

			fmt.Printf("  %s %s\n", severity, issue.Message)
			if issue.Action != "" {
				fmt.Printf("    Fix: %s\n", issue.Action)
			}
			if len(issue.AffectedDevices) > 0 {
				fmt.Printf("    Devices: %d affected\n", len(issue.AffectedDevices))
			}
		}
	}

	fmt.Printf("\n%s\n", result.Summary())
}

func init() {
	// Suppress unused function warning during development
	_ = displayCompatibilityResultsTable
}

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ravinald/wifimgr/internal/cmdutils"
	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/symbols"
	"github.com/ravinald/wifimgr/internal/validation"
	"github.com/ravinald/wifimgr/internal/vendors"
)

var lintConfigCmd = &cobra.Command{
	Use:   "config <site-name>",
	Short: "Lint a site configuration",
	Annotations: map[string]string{
		cmdutils.AnnotationNeedsConfig: "true",
	},
	Long: `Validate a site configuration for common issues and errors.

This command performs comprehensive validation including:
- Syntax validation (required fields, data types)
- Schema validation (field types match expected schema)
- Vendor block validation (correct vendor-specific fields)
- Reference validation (profiles and templates exist)
- Deprecated field detection
- Range validation (numeric values within acceptable ranges)
- Radio configuration validation
- WLAN assignment validation (profile declarations and template existence)

Examples:
  wifimgr lint config US-LAB-01
  wifimgr lint config US-DC-PROD`,
	Args: cobra.ExactArgs(1),
	RunE: runLintConfig,
}

func init() {
	lintCmd.AddCommand(lintConfigCmd)
}

func runLintConfig(cmd *cobra.Command, args []string) error {
	// Check for help keyword in positional arguments
	if cmdutils.ContainsHelp(args) {
		return cmd.Help()
	}

	siteName := args[0]

	logging.Infof("Linting configuration for site: %s", siteName)

	// Load site configuration
	siteConfig, err := loadSiteConfiguration(siteName)
	if err != nil {
		return fmt.Errorf("failed to load site configuration: %w", err)
	}

	// Get global cache accessor if available
	cacheAccessor := vendors.GetGlobalCacheAccessor()

	// Create linter
	linter := validation.NewConfigLinter(cacheAccessor)

	// Load templates for WLAN reference validation
	templatePaths := viper.GetStringSlice("files.templates")
	configDir := viper.GetString("files.config_dir")
	if len(templatePaths) > 0 && configDir != "" {
		templates, templateErr := config.LoadTemplates(templatePaths, configDir)
		if templateErr != nil {
			logging.Warnf("Failed to load templates: %v", templateErr)
		} else {
			linter.SetTemplateStore(templates)
		}
	}

	// Perform linting
	result, err := linter.LintSite(siteName, siteConfig)
	if err != nil {
		return fmt.Errorf("linting failed: %w", err)
	}

	// Display results
	displayLintResults(result)

	// Exit with error code if there are errors
	if len(result.Errors) > 0 {
		return fmt.Errorf("configuration has %d error(s)", len(result.Errors))
	}

	return nil
}

func loadSiteConfiguration(siteName string) (*config.SiteConfigObj, error) {
	// Get config directory from Viper
	configDir := viper.GetString("files.config_dir")
	if configDir == "" {
		return nil, fmt.Errorf("config directory not set")
	}

	// Get relative config file path
	relativePath, exists := config.GetSiteConfigPath(siteName)
	if !exists {
		return nil, fmt.Errorf("site '%s' not found in configuration", siteName)
	}

	// Get the actual key used in the config file
	siteKey, keyExists := config.GetSiteConfigKey(siteName)
	if !keyExists {
		siteKey = siteName
	}

	// Load the config file
	siteConfigFile, err := config.LoadSiteConfig(configDir, relativePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config file: %w", err)
	}

	// Get the site configuration
	siteConfig, exists := siteConfigFile.Config.Sites[siteKey]
	if !exists {
		return nil, fmt.Errorf("site '%s' not found in config file", siteName)
	}

	return &siteConfig, nil
}

func displayLintResults(result *validation.LintResult) {
	fmt.Printf("\n")
	fmt.Printf("Lint Results for Site: %s\n", result.SiteName)
	fmt.Printf("----------------------------------------\n")
	fmt.Printf("Devices scanned:\n")
	fmt.Printf("  APs:      %d\n", result.APCount)
	fmt.Printf("  Switches: %d\n", result.SwitchCount)
	fmt.Printf("  Gateways: %d\n", result.GatewayCount)
	fmt.Printf("\n")

	// Display errors
	if len(result.Errors) > 0 {
		fmt.Printf("%s Errors (%d):\n", symbols.ErrorPrefix(), len(result.Errors))
		for _, issue := range result.Errors {
			displayLintIssue(issue, "ERROR")
		}
		fmt.Printf("\n")
	}

	// Display warnings
	if len(result.Warnings) > 0 {
		fmt.Printf("%s Warnings (%d):\n", symbols.WarningPrefix(), len(result.Warnings))
		for _, issue := range result.Warnings {
			displayLintIssue(issue, "WARN")
		}
		fmt.Printf("\n")
	}

	// Summary
	if len(result.Errors) == 0 && len(result.Warnings) == 0 {
		fmt.Printf("%s Configuration is valid - no issues found\n", symbols.SuccessPrefix())
	} else {
		fmt.Printf("Summary: %d error(s), %d warning(s)\n", len(result.Errors), len(result.Warnings))
		if len(result.Errors) > 0 {
			fmt.Printf("\nConfiguration has errors and should be fixed before applying.\n")
		}
	}
}

func displayLintIssue(issue validation.LintIssue, severity string) {
	deviceInfo := issue.DeviceMAC
	if issue.DeviceName != "" {
		deviceInfo = fmt.Sprintf("%s (%s)", issue.DeviceName, issue.DeviceMAC)
	}

	fmt.Printf("  [%s] %s\n", severity, deviceInfo)
	fmt.Printf("      Field: %s\n", issue.Field)
	fmt.Printf("      Issue: %s\n", issue.Message)
	if issue.Suggestion != "" {
		fmt.Printf("      Fix:   %s\n", issue.Suggestion)
	}
}

// displayLintResultsTable is an alternative table-based display format.
func displayLintResultsTable(result *validation.LintResult) {
	if len(result.Errors) == 0 && len(result.Warnings) == 0 {
		fmt.Printf("%s Configuration is valid - no issues found\n", symbols.SuccessPrefix())
		return
	}

	// Create table data
	type Row struct {
		Severity string
		Device   string
		Field    string
		Message  string
		Fix      string
	}

	var rows []Row

	for _, issue := range result.Errors {
		deviceInfo := issue.DeviceMAC
		if issue.DeviceName != "" {
			deviceInfo = fmt.Sprintf("%s (%s)", issue.DeviceName, issue.DeviceMAC[:8])
		}
		rows = append(rows, Row{
			Severity: "ERROR",
			Device:   deviceInfo,
			Field:    issue.Field,
			Message:  issue.Message,
			Fix:      issue.Suggestion,
		})
	}

	for _, issue := range result.Warnings {
		deviceInfo := issue.DeviceMAC
		if issue.DeviceName != "" {
			deviceInfo = fmt.Sprintf("%s (%s)", issue.DeviceName, issue.DeviceMAC[:8])
		}
		rows = append(rows, Row{
			Severity: "WARN",
			Device:   deviceInfo,
			Field:    issue.Field,
			Message:  issue.Message,
			Fix:      issue.Suggestion,
		})
	}

	// Simple text-based table
	fmt.Printf("\n%-8s %-24s %-20s %-40s %s\n", "SEVERITY", "DEVICE", "FIELD", "MESSAGE", "FIX")
	fmt.Printf("%-8s %-24s %-20s %-40s %s\n",
		"--------", "------------------------", "--------------------",
		"----------------------------------------", "-----------------------------------")

	for _, row := range rows {
		fmt.Printf("%-8s %-24s %-20s %-40s %s\n",
			row.Severity,
			truncate(row.Device, 24),
			truncate(row.Field, 20),
			truncate(row.Message, 40),
			truncate(row.Fix, 35))
	}

	fmt.Printf("\nSummary: %d error(s), %d warning(s)\n", len(result.Errors), len(result.Warnings))
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func init() {
	// Suppress unused function warning during development
	_ = displayLintResultsTable
	_ = os.Stderr
}

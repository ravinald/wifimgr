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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ravinald/wifimgr/internal/config"
)

// initCmd represents the init command (parent)
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize new configuration resources",
	Long: `Initialize new configuration resources for wifimgr.

Use 'wifimgr init <subcommand> --help' for more information about a subcommand.`,
}

// initSiteCmd represents the "init site" subcommand
var initSiteCmd = &cobra.Command{
	Use:   "site <site-name> api <api-name> [file <filepath>]",
	Short: "Create a new site configuration file",
	Long: `Create a new site configuration file with a skeleton structure.

Basic Usage:
  wifimgr init site <site-name> api <api-name>
  wifimgr init site <site-name> api mist
  wifimgr init site <site-name> api meraki file <filepath>

Examples:
  wifimgr init site US-LAB-01 api mist                      # Creates config/US-LAB-01.json with api: mist
  wifimgr init site US-LAB-01 api meraki file us-lab.json   # Creates config/us-lab.json with api: meraki
  wifimgr init site US-LAB-01 api mist file meraki/lab.json # Creates config/meraki/lab.json

Arguments:
  site-name   Required. Name of the site (A-Za-z0-9_- only)
  api         Required. Keyword followed by API name (must be defined in .api config)
  file        Optional. Keyword followed by filepath relative to config directory

What it Does:
  1. Creates a skeleton site configuration JSON file with the specified API
  2. Includes empty site_config and devices sections (ap, switch, gateway)
  3. Registers the file in wifimgr-config.json under files.site_configs
  4. Creates subdirectories if needed

Output Location:
  Default: ~/.config/wifimgr/<site-name>.json
  Custom:  ~/.config/wifimgr/<filepath> (when using 'file' keyword)`,
	Args: func(cmd *cobra.Command, args []string) error {
		// Allow "help" as a special keyword
		for _, arg := range args {
			if strings.ToLower(arg) == "help" {
				return nil // Will be handled in RunE
			}
		}
		// Otherwise require at least 3 args: <site-name> api <api-name>
		if len(args) < 3 {
			return fmt.Errorf("requires at least 3 arg(s), only received %d", len(args))
		}
		return nil
	},
	RunE: runInitSite,
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.AddCommand(initSiteCmd)
}

// initSiteArgs holds parsed init site command arguments
type initSiteArgs struct {
	siteName string
	apiName  string
	filePath string
}

// validNamePattern validates site names and base filenames: A-Za-z0-9_- only
var validNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// parseInitSiteArgs parses positional arguments for the init site command.
// Expected format: <site-name> api <api-name> [file <filepath>]
func parseInitSiteArgs(args []string) (*initSiteArgs, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("site name is required")
	}

	result := &initSiteArgs{
		siteName: args[0],
	}

	// Parse api and optional file keywords
	for i := 1; i < len(args); i++ {
		arg := strings.ToLower(args[i])
		switch arg {
		case "api":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("'api' requires a name")
			}
			result.apiName = args[i+1]
			i++ // Skip the api name
		case "file":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("'file' requires a filepath")
			}
			result.filePath = args[i+1]
			i++ // Skip the filepath
		default:
			return nil, fmt.Errorf("unexpected argument: %s", args[i])
		}
	}

	return result, nil
}

// validateSiteName validates the site name format
func validateSiteName(name string) error {
	if name == "" {
		return fmt.Errorf("site name cannot be empty")
	}
	if !validNamePattern.MatchString(name) {
		return fmt.Errorf("site name '%s' contains invalid characters; only A-Za-z0-9_- allowed", name)
	}
	return nil
}

// validateFilename validates the base filename (without path, without .json extension)
func validateFilename(filename string) error {
	// Extract base name without extension
	base := filepath.Base(filename)
	base = strings.TrimSuffix(base, ".json")
	if base == "" {
		return fmt.Errorf("filename cannot be empty")
	}
	if !validNamePattern.MatchString(base) {
		return fmt.Errorf("filename '%s' contains invalid characters in base name; only A-Za-z0-9_- allowed (before .json)", filename)
	}
	return nil
}

// createSkeletonSiteConfig creates a minimal site configuration structure
func createSkeletonSiteConfig(siteName, apiName string) *config.SiteConfigFile {
	return &config.SiteConfigFile{
		Version: 1,
		Config: config.SiteConfigWrapper{
			Sites: map[string]config.SiteConfigObj{
				siteName: {
					API: apiName,
					SiteConfig: config.SiteConfig{
						Name:        siteName,
						Address:     "",
						CountryCode: "",
						Timezone:    "",
						Notes:       "",
						LatLng:      nil,
					},
					Devices: config.Devices{
						APs:      make(map[string]config.APConfig),
						Switches: make(map[string]config.SwitchConfig),
						WanEdge:  make(map[string]config.WanEdgeConfig),
					},
				},
			},
		},
	}
}

// writeSiteConfigFile writes the site config to the specified path.
// Creates parent directories if needed.
func writeSiteConfigFile(siteConfig *config.SiteConfigFile, fullPath string) error {
	// Create parent directories if they don't exist
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory '%s': %w", dir, err)
	}

	// Marshal to JSON with 2-space indent
	jsonData, err := json.MarshalIndent(siteConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal site config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(fullPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write file '%s': %w", fullPath, err)
	}

	return nil
}

// addSiteConfigToAppConfig adds the filename to the site_configs list in the app config.
// Uses direct JSON manipulation to preserve the full config structure.
func addSiteConfigToAppConfig(relativeFilename string) error {
	// Get the config file path from Viper
	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		return fmt.Errorf("no config file found")
	}

	// Read the current config file
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse as generic JSON to preserve all fields
	var configMap map[string]interface{}
	if err := json.Unmarshal(data, &configMap); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Navigate to files.site_configs
	filesSection, ok := configMap["files"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("config file missing 'files' section")
	}

	// Get current site_configs array
	var siteConfigs []string
	if existing, ok := filesSection["site_configs"].([]interface{}); ok {
		for _, item := range existing {
			if str, ok := item.(string); ok {
				siteConfigs = append(siteConfigs, str)
			}
		}
	}

	// Check if already registered
	for _, existing := range siteConfigs {
		if strings.EqualFold(existing, relativeFilename) {
			return fmt.Errorf("file '%s' is already registered in site_configs", existing)
		}
	}

	// Add the new filename
	siteConfigs = append(siteConfigs, relativeFilename)

	// Update the config map
	filesSection["site_configs"] = siteConfigs

	// Write back to file with proper formatting
	jsonData, err := json.MarshalIndent(configMap, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configFile, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// runInitSite is the main handler for the "init site" command
func runInitSite(cmd *cobra.Command, args []string) error {
	// Check for help keyword in positional arguments
	for _, arg := range args {
		if strings.ToLower(arg) == "help" {
			return cmd.Help()
		}
	}

	// Initialize Viper for config access (we skip full app initialization to avoid API token requirement)
	if err := config.InitializeViper(cmd); err != nil {
		return fmt.Errorf("failed to initialize configuration: %w", err)
	}

	// Read the config file so Viper knows which file to use
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse arguments
	parsedArgs, err := parseInitSiteArgs(args)
	if err != nil {
		return fmt.Errorf("invalid arguments: %w", err)
	}

	// Validate site name
	if err := validateSiteName(parsedArgs.siteName); err != nil {
		return fmt.Errorf("invalid site name: %w", err)
	}

	// Get configured APIs and validate
	configuredAPIs := viper.GetStringMap("api")
	var validAPINames []string
	for name := range configuredAPIs {
		validAPINames = append(validAPINames, name)
	}
	sort.Strings(validAPINames) // Sort for consistent output

	// Validate API is provided
	if parsedArgs.apiName == "" {
		return fmt.Errorf("api is required. Valid options: %s", strings.Join(validAPINames, ", "))
	}

	// Validate API exists in config (case-insensitive)
	apiFound := false
	for name := range configuredAPIs {
		if strings.EqualFold(name, parsedArgs.apiName) {
			parsedArgs.apiName = name // Use the actual key from config
			apiFound = true
			break
		}
	}
	if !apiFound {
		return fmt.Errorf("api '%s' not found in configuration. Valid options: %s", parsedArgs.apiName, strings.Join(validAPINames, ", "))
	}

	// Determine filename
	configDir := viper.GetString("files.config_dir")
	var relativeFilename string

	if parsedArgs.filePath != "" {
		// User specified a file path
		relativeFilename = parsedArgs.filePath

		// Ensure .json extension
		if !strings.HasSuffix(strings.ToLower(relativeFilename), ".json") {
			relativeFilename = relativeFilename + ".json"
		}
	} else {
		// Default: <site-name>.json
		relativeFilename = parsedArgs.siteName + ".json"
	}

	// Validate the filename
	if err := validateFilename(relativeFilename); err != nil {
		return fmt.Errorf("invalid filename: %w", err)
	}

	// Compute full path
	fullPath := filepath.Join(configDir, relativeFilename)

	// Check if file already exists
	if _, err := os.Stat(fullPath); err == nil {
		return fmt.Errorf("file '%s' already exists", fullPath)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check file '%s': %w", fullPath, err)
	}

	// Create skeleton config
	siteConfig := createSkeletonSiteConfig(parsedArgs.siteName, parsedArgs.apiName)

	// Write the site config file
	if err := writeSiteConfigFile(siteConfig, fullPath); err != nil {
		return fmt.Errorf("failed to write site config: %w", err)
	}

	// Add to app config
	if err := addSiteConfigToAppConfig(relativeFilename); err != nil {
		// Attempt to clean up the created file on failure
		_ = os.Remove(fullPath)
		return fmt.Errorf("failed to register site config: %w", err)
	}

	fmt.Printf("Created site configuration: %s\n", fullPath)
	fmt.Printf("API: %s\n", parsedArgs.apiName)
	fmt.Printf("Registered in wifimgr-config.json: files.site_configs\n")

	return nil
}

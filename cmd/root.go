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
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"

	"github.com/ravinald/wifimgr/api"
	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/xdg"
)

// Global state for initialized client and site configs
var (
	globalClient  api.Client
	globalContext context.Context = context.Background()
)

// Global CLI options (only essential flags, rest handled by Viper)
var (
	debug           bool // -d: info level
	extraDebug      bool // -dd: debug level
	traceDebug      bool // -ddd: trace level
	useEnvFile      bool
	configFile      string
	caseInsensitive bool
	suppressOutput  bool // --suppress: suppress SDK debug output

	// Temporary compatibility for command handlers during Viper migration
	globalConfig *config.Config
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "wifimgr",
	Short: "WiFi and network infrastructure management CLI",
	Long: `WiFi Manager is a comprehensive CLI tool for managing Mist Systems network infrastructure.

It provides commands to:
- View and manage sites, access points, switches, and gateways
- Apply configuration changes to network devices
- Search and inventory network equipment
- Refresh cached data from the Mist API

For detailed usage information, run 'wifimgr help [command]'`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip initialization for commands that don't need API access
		if cmd.Name() == "version" || cmd.Name() == "help" || cmd.Name() == "completion" || cmd.Name() == "init" || cmd.Name() == "fields" {
			return nil
		}
		// Skip for "init site" subcommand (check parent is "init")
		if cmd.Name() == "site" && cmd.Parent() != nil && cmd.Parent().Name() == "init" {
			return nil
		}
		// Skip initialization if "help" is in positional args (Junos-style help)
		for _, arg := range args {
			if strings.ToLower(arg) == "help" {
				return nil
			}
		}
		return initializeApplication(cmd)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// Ensure logging cleanup happens when the program exits
	defer logging.Cleanup()

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// initializeApplication initializes the application with configuration and API client
func initializeApplication(cmd *cobra.Command) error {
	// Initialize Viper first
	if err := config.InitializeViper(cmd); err != nil {
		return fmt.Errorf("failed to initialize Viper: %w", err)
	}
	// Handle cascading debug levels: -ddd implies -dd implies -d
	if traceDebug {
		extraDebug = true
		debug = true
	}
	if extraDebug {
		debug = true
	}

	// Convert flags and Viper config to CLIOptions struct
	opts := config.CLIOptions{
		Debug:      debug,
		UseEnvFile: useEnvFile,
		// Other options will be read from Viper/config file
	}

	// Set debug level based on flags
	switch {
	case traceDebug:
		opts.DebugLevelInt = config.DebugTrace
	case extraDebug:
		opts.DebugLevelInt = config.DebugDebug
	case debug:
		opts.DebugLevelInt = config.DebugInfo
	default:
		opts.DebugLevelInt = config.DebugNone
	}

	// Initialize logging with a basic configuration
	// Set to warning level to show duplicate config warnings during loading
	initialLogConfig := logging.LogConfig{
		Enable:   true,
		Level:    "warning", // Show warnings during config loading
		Format:   "text",
		ToStdout: true,
		Silent:   false,
		LogFile:  "", // No file logging yet
	}

	// If debug flag is used, override log level
	// -d = info, -dd = debug, -ddd = debug (trace uses debug level)
	switch opts.DebugLevelInt {
	case config.DebugTrace, config.DebugDebug:
		initialLogConfig.Level = "debug"
	case config.DebugInfo:
		initialLogConfig.Level = "info"
	default:
		// DebugNone or any other value: keep default warning level
	}

	if err := logging.ConfigureLogging(initialLogConfig); err != nil {
		return fmt.Errorf("failed to initialize logging: %v", err)
	}

	// Load configurations using Viper
	configPath := configFile
	if configPath == "" {
		configPath = xdg.GetConfigFile()
	}

	var siteConfigs []*config.SiteConfigFile
	var err error

	// Load configurations - always load site configs regardless of -e flag
	siteConfigs, err = config.LoadAllConfigsViper(configPath)
	if err != nil {
		return fmt.Errorf("error loading configurations: %v", err)
	}

	logging.Debugf("Loaded main configuration (version %.1f)", viper.GetFloat64("version"))
	logging.Debugf("Loaded %d site configurations", len(siteConfigs))

	// Configure final logging
	var logLevel string
	switch opts.DebugLevelInt {
	case config.DebugTrace, config.DebugDebug:
		logLevel = "debug"
	case config.DebugInfo:
		logLevel = "info"
	default:
		logLevel = viper.GetString("logging.level")
	}

	finalLogConfig := logging.LogConfig{
		Enable:   viper.GetBool("logging.enable") || opts.DebugLevelInt > config.DebugNone,
		Format:   viper.GetString("logging.format"),
		LogFile:  viper.GetString("files.log_file"),
		Level:    logLevel,
		ToStdout: viper.GetBool("logging.stdout"),
	}

	// When debug flags are used, always enable stdout logging
	if opts.DebugLevelInt > config.DebugNone {
		finalLogConfig.ToStdout = true
	}

	if err := logging.ConfigureLogging(finalLogConfig); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "FATAL: Unable to configure log file '%s': %v\n", finalLogConfig.LogFile, err)
		os.Exit(1)
	}

	logging.Info("Starting wifimgr")

	// Log terminal properties
	if term.IsTerminal(int(os.Stdout.Fd())) {
		if width, height, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
			logging.Debugf("Terminal size: %d columns × %d rows", width, height)
		}
		logging.Debugf("Terminal type: %s", os.Getenv("TERM"))
		if colorTerm := os.Getenv("COLORTERM"); colorTerm != "" {
			logging.Debugf("Color capability: %s", colorTerm)
		}
	} else {
		logging.Debug("Output is not a terminal")
	}

	logging.Debugf("Command-line flags: debug=%v, use-env=%v",
		opts.Debug, opts.UseEnvFile)

	// Load .env.wifimgr file BEFORE multi-vendor initialization
	// so that WIFIMGR_API_<LABEL>_CREDENTIALS_* vars are available
	var envCleanup func()
	if opts.UseEnvFile {
		logging.Info("Loading .env.wifimgr file")
		cleanup, err := config.SecureLoadEnvFile(".env.wifimgr")
		if err != nil {
			logging.Warnf("Failed to load .env.wifimgr file: %v", err)
		} else {
			envCleanup = cleanup
			logging.Info("Successfully loaded .env.wifimgr file")
		}
	}

	// Initialize multi-vendor infrastructure
	// This loads API configs from Viper and applies environment variable overrides
	if err := InitializeMultiVendor(); err != nil {
		logging.Warnf("Multi-vendor initialization: %v", err)
	}

	// Clean up env vars after multi-vendor init has read them
	if envCleanup != nil {
		envCleanup()
	}

	// Create HTTP client
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Initialize API client
	rateLimit := viper.GetInt("api.rate_limit")
	if rateLimit <= 0 {
		rateLimit = 10
	}

	resultsLimit := viper.GetInt("api.results_limit")
	if opts.Limit > 0 {
		resultsLimit = opts.Limit
	} else if resultsLimit <= 0 {
		resultsLimit = 100
	}

	var apiToken, apiURL, apiOrgID string

	// Get credentials from multi-vendor registry for globalClient
	registry := GetAPIRegistry()
	if registry != nil {
		// Find the Mist API config
		for _, label := range registry.GetAllLabels() {
			apiConfig, err := registry.GetConfig(label)
			if err == nil && apiConfig != nil && apiConfig.Vendor == "mist" {
				apiToken = apiConfig.Credentials["api_token"]
				apiURL = apiConfig.URL
				apiOrgID = apiConfig.Credentials["org_id"]
				logging.Debugf("Using credentials from %s API for globalClient (URL: %s, OrgID: %s)",
					label, apiURL, apiOrgID)
				break
			}
		}
	}

	if apiToken == "" {
		return fmt.Errorf("no Mist API credentials found - ensure api.mist is configured and -e flag is used")
	}

	// Create API client
	// Check if debug should be enabled from either CLI flag or config file
	debugEnabled := opts.DebugLevelInt > config.DebugNone ||
		(viper.GetBool("logging.enable") && viper.GetString("logging.level") == "debug")

	// Derive cache path from cache_dir if files.cache is not explicitly set
	cachePath := viper.GetString("files.cache")
	if cachePath == "" {
		cacheDir := viper.GetString("files.cache_dir")
		if cacheDir == "" {
			cacheDir = xdg.GetCacheDir()
		}
		cachePath = cacheDir + "/cache.json"
	}

	client := api.NewClientWithOptions(
		apiToken,
		apiURL,
		apiOrgID,
		api.WithHTTPClient(httpClient),
		api.WithDebug(debugEnabled),
		api.WithRateLimit(rateLimit, time.Minute),
		api.WithCacheTTL(5*time.Minute),
		api.WithCacheDirectory(cachePath),
		api.WithInventory(viper.GetString("files.inventory")),
		api.WithDryRun(opts.DryRun),
		api.WithResultsLimit(resultsLimit),
	)

	// Set global client
	api.SetClient(client)
	globalClient = client
	// Note: siteConfigs are loaded but stored in Viper for command access

	// Create compatibility config for command handlers during Viper migration
	globalConfig = &config.Config{
		API: config.API{
			Credentials: config.Credentials{
				OrgID:    apiOrgID,
				APIToken: apiToken,
			},
			URL:          apiURL,
			RateLimit:    viper.GetInt("api.rate_limit"),
			ResultsLimit: viper.GetInt("api.results_limit"),
			ManagedKeys:  getManagedKeysFromViper(),
		},
		Files: config.Files{
			ConfigDir:      viper.GetString("files.config_dir"),
			SiteConfigs:    viper.GetStringSlice("files.site_configs"),
			DeviceProfiles: viper.GetStringSlice("files.device_profiles"),
			Cache:          cachePath,
			Inventory:      viper.GetString("files.inventory"),
			LogFile:        viper.GetString("files.log_file"),
			Schemas:        viper.GetString("files.schemas"),
		},
	}

	// Configure logging lookups
	logging.SetSiteNameLookupFunc(func(siteID string) (string, bool) {
		return client.GetSiteName(siteID)
	})

	logging.SetOrgNameLookupFunc(func(orgID string) (string, bool) {
		return client.GetOrgName(orgID)
	})

	// Cache operations are now handled by the multi-vendor cache manager per-API
	// Use 'wifimgr refresh cache' to rebuild cache

	return nil
}

// getManagedKeysFromViper retrieves managed keys configuration from Viper
func getManagedKeysFromViper() *config.ManagedKeys {
	if !viper.IsSet("api.managed_keys") {
		return nil
	}

	managedKeys := &config.ManagedKeys{
		AP:      viper.GetStringSlice("api.managed_keys.ap"),
		Switch:  viper.GetStringSlice("api.managed_keys.switch"),
		Gateway: viper.GetStringSlice("api.managed_keys.gateway"),
	}

	// Return nil if all are empty
	if len(managedKeys.AP) == 0 && len(managedKeys.Switch) == 0 && len(managedKeys.Gateway) == 0 {
		return nil
	}

	return managedKeys
}

func init() {
	// Essential flags only - rest handled by Viper configuration
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Enable info-level debug output")
	rootCmd.PersistentFlags().BoolVar(&extraDebug, "dd", false, "Enable debug-level output (more verbose)")
	rootCmd.PersistentFlags().BoolVar(&traceDebug, "ddd", false, "Enable trace-level output (most verbose)")
	rootCmd.PersistentFlags().BoolVarP(&useEnvFile, "env", "e", false, "Read API token from .env.wifimgr")
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Path to configuration file (default: ~/.config/wifimgr/wifimgr-config.json)")
	rootCmd.PersistentFlags().BoolVarP(&caseInsensitive, "case-insensitive", "i", false, "Perform case-insensitive pattern matching")
	rootCmd.PersistentFlags().BoolVar(&suppressOutput, "suppress", false,
		"Suppress Meraki SDK debug output (workaround for github.com/meraki/dashboard-api-go issues #72 and #75)")

	// Bind the case-insensitive flag to viper
	if err := viper.BindPFlag("case-insensitive", rootCmd.PersistentFlags().Lookup("case-insensitive")); err != nil {
		logging.Errorf("Failed to bind case-insensitive flag: %v", err)
	}
}

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ravinald/wifimgr/internal/xdg"
)

// Global site index for O(1) lookup of site name to config file path
var (
	globalSiteIndex *SiteIndex
	siteIndexMu     sync.RWMutex
)

// GetSiteIndex returns the global site index
func GetSiteIndex() *SiteIndex {
	siteIndexMu.RLock()
	defer siteIndexMu.RUnlock()
	return globalSiteIndex
}

// GetSiteConfigPath returns the config file path for a site name (case-insensitive)
// Returns the relative path (e.g., "demo/us-oak-pina.json") and true if found
func GetSiteConfigPath(siteName string) (string, bool) {
	siteIndexMu.RLock()
	defer siteIndexMu.RUnlock()

	if globalSiteIndex == nil {
		return "", false
	}

	path, ok := globalSiteIndex.SiteToFile[strings.ToLower(siteName)]
	return path, ok
}

// GetSiteConfigFullPath returns the full config file path for a site name
// Returns the full path (e.g., "./config/demo/us-oak-pina.json") and true if found
func GetSiteConfigFullPath(siteName string) (string, bool) {
	relativePath, ok := GetSiteConfigPath(siteName)
	if !ok {
		return "", false
	}

	configDir := viper.GetString("files.config_dir")
	if configDir == "" {
		configDir = xdg.GetConfigDir()
	}

	return filepath.Join(configDir, relativePath), true
}

// GetSiteConfigKey returns the actual key used in the config file for a site name
// This handles case differences between the lookup name and the config key
func GetSiteConfigKey(siteName string) (string, bool) {
	siteIndexMu.RLock()
	defer siteIndexMu.RUnlock()

	if globalSiteIndex == nil {
		return "", false
	}

	key, ok := globalSiteIndex.SiteToKey[strings.ToLower(siteName)]
	return key, ok
}

// InitializeViper initializes Viper with configuration sources and bindings
func InitializeViper(cmd *cobra.Command) error {
	// Set config name and type
	viper.SetConfigName("wifimgr-config")
	viper.SetConfigType("json")

	// Add config search paths (XDG-compliant)
	viper.AddConfigPath(xdg.GetConfigDir())
	viper.AddConfigPath(".")

	// Set environment variable prefix
	viper.SetEnvPrefix("WIFIMGR")
	viper.AutomaticEnv()

	// Bind command flags to viper
	if err := viper.BindPFlags(cmd.Flags()); err != nil {
		return fmt.Errorf("failed to bind flags to viper: %w", err)
	}

	// Set default values for main config
	setDefaults()

	return nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// Version
	viper.SetDefault("version", 1.0)

	// API defaults
	viper.SetDefault("api.url", "https://api.mist.com")
	viper.SetDefault("api.rate_limit", 10)
	viper.SetDefault("api.results_limit", 100)

	// Files defaults (XDG-compliant paths)
	viper.SetDefault("files.config_dir", xdg.GetConfigDir())
	// files.cache is derived from files.cache_dir when not explicitly set
	viper.SetDefault("files.cache_dir", xdg.GetCacheDir())
	viper.SetDefault("files.inventory", xdg.GetInventoryFile())
	viper.SetDefault("files.log_file", xdg.GetLogFile())
	viper.SetDefault("files.schemas", xdg.GetSchemasDir())

	// Logging defaults
	viper.SetDefault("logging.enable", false)
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "text")
	viper.SetDefault("logging.stdout", true)

	// Display jsoncolor defaults
	viper.SetDefault("display.jsoncolor.null.hex", "#767676")
	viper.SetDefault("display.jsoncolor.null.ansi256", "244")
	viper.SetDefault("display.jsoncolor.null.ansi", "8")
	viper.SetDefault("display.jsoncolor.bool.hex", "#FFFFFF")
	viper.SetDefault("display.jsoncolor.bool.ansi256", "15")
	viper.SetDefault("display.jsoncolor.bool.ansi", "7")
	viper.SetDefault("display.jsoncolor.number.hex", "#00FFFF")
	viper.SetDefault("display.jsoncolor.number.ansi256", "51")
	viper.SetDefault("display.jsoncolor.number.ansi", "6")
	viper.SetDefault("display.jsoncolor.string.hex", "#00FF00")
	viper.SetDefault("display.jsoncolor.string.ansi256", "46")
	viper.SetDefault("display.jsoncolor.string.ansi", "2")
	viper.SetDefault("display.jsoncolor.key.hex", "#0000FF")
	viper.SetDefault("display.jsoncolor.key.ansi256", "21")
	viper.SetDefault("display.jsoncolor.key.ansi", "4")
	viper.SetDefault("display.jsoncolor.bytes.hex", "#767676")
	viper.SetDefault("display.jsoncolor.bytes.ansi256", "244")
	viper.SetDefault("display.jsoncolor.bytes.ansi", "8")
	viper.SetDefault("display.jsoncolor.time.hex", "#00FF00")
	viper.SetDefault("display.jsoncolor.time.ansi256", "46")
	viper.SetDefault("display.jsoncolor.time.ansi", "2")
}

// LoadViperConfig loads the main configuration using Viper
func LoadViperConfig(configFile string) error {
	// Set specific config file if provided
	if configFile != "" {
		viper.SetConfigFile(configFile)
	}

	// Try to read config file
	if err := viper.ReadInConfig(); err != nil {
		// Check if it's a file not found error
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found - use defaults for certain modes
			if _, err := fmt.Fprintf(os.Stderr, "Warning: Config file not found, using defaults: %v\n", err); err != nil {
				return fmt.Errorf("failed to write warning message: %w", err)
			}
		} else {
			// Config file found but had another error
			return fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Validate version
	version := viper.GetFloat64("version")
	if version != 1.0 {
		if _, err := fmt.Fprintf(os.Stderr, "Warning: Main config has version %.1f, expected 1.0.\n", version); err != nil {
			return fmt.Errorf("failed to write warning message: %w", err)
		}

		// Ask user if they want to continue
		fmt.Printf("Do you want to continue loading this config? [y/N]: ")
		var response string
		_, err := fmt.Scanln(&response)
		if err != nil || (response != "y" && response != "Y" && response != "yes" && response != "Yes") {
			return fmt.Errorf("user chose not to load config with version %.1f", version)
		}

		if _, err := fmt.Fprintf(os.Stderr, "Warning: Continuing with potentially incompatible config version %.1f\n", version); err != nil {
			return fmt.Errorf("failed to write warning message: %w", err)
		}
	}

	return nil
}

// LoadAllConfigsViper loads main config via Viper and site configs via existing logic
func LoadAllConfigsViper(configFile string) ([]*SiteConfigFile, error) {
	// Load main config using Viper
	err := LoadViperConfig(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load main config with Viper: %w", err)
	}

	// Create duplicate tracker
	duplicateTracker := NewDuplicateTracker()

	// Initialize site index
	siteIndex := &SiteIndex{
		SiteToFile: make(map[string]string),
		SiteToKey:  make(map[string]string),
	}

	// Load site configs using existing logic (unchanged)
	var siteConfigs []*SiteConfigFile

	// Get site config files from Viper
	siteConfigFiles := viper.GetStringSlice("files.site_configs")
	configDir := viper.GetString("files.config_dir")
	mainVersion := viper.GetInt("version")

	for _, siteConfigFile := range siteConfigFiles {
		fullPath := filepath.Join(configDir, siteConfigFile)

		// Read the raw file data for line number estimation
		rawData, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read site config %s: %w", siteConfigFile, err)
		}

		siteConfig, err := LoadSiteConfig(configDir, siteConfigFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load site config %s: %w", siteConfigFile, err)
		}

		// Verify version matches
		if siteConfig.Version != mainVersion {
			if _, err := fmt.Fprintf(os.Stderr, "Warning: Version mismatch in config file %s: expected %d, got %d\n",
				siteConfigFile, mainVersion, siteConfig.Version); err != nil {
				return nil, fmt.Errorf("failed to write warning message: %w", err)
			}
			fmt.Printf("Proceeding with mismatched version config file %s\n", siteConfigFile)
		}

		// For each site in the config, check for duplicates and add it to our list
		for siteName, siteObj := range siteConfig.Config.Sites {
			// Check for duplicate site names
			keyPath := []string{"config", "sites", siteName, "site_config"}
			line := EstimateLineNumber(rawData, keyPath)

			// Use the site_config.name if available, otherwise use the key
			configName := siteObj.SiteConfig.Name
			if configName == "" {
				configName = siteName
			}

			duplicateTracker.CheckAndAdd("site_config", "", configName, siteConfigFile, line)

			// Add to site index (case-insensitive lookup)
			lowerName := strings.ToLower(configName)
			siteIndex.SiteToFile[lowerName] = siteConfigFile
			siteIndex.SiteToKey[lowerName] = siteName

			individualSiteConfig := &SiteConfigFile{
				Version: siteConfig.Version,
				Config: SiteConfigWrapper{
					Sites: map[string]SiteConfigObj{siteName: siteObj},
				},
			}
			siteConfigs = append(siteConfigs, individualSiteConfig)
		}
	}

	// Store site index globally
	siteIndexMu.Lock()
	globalSiteIndex = siteIndex
	siteIndexMu.Unlock()

	return siteConfigs, nil
}

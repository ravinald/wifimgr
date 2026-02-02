package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

// LoadConfig loads the main configuration from the specified file
func LoadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			// If we already have an error, don't overwrite it
			if err == nil {
				err = fmt.Errorf("failed to close config file: %w", closeErr)
			}
			// Otherwise just log it
			_, logErr := fmt.Fprintf(os.Stderr, "Warning: failed to close config file: %v\n", closeErr)
			if logErr != nil {
				// If we can't even log to stderr, log to stdout as fallback
				fmt.Printf("Error writing to stderr: %v\n", logErr)
			}
		}
	}()

	var cfg Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Check if version is 1
	if cfg.Version != 1 {
		_, err := fmt.Fprintf(os.Stderr, "Warning: Main config file %s has version %d, expected 1.\n", filename, cfg.Version)
		if err != nil {
			fmt.Printf("Error writing to stderr: %v\n", err)
		}

		// Ask user if they want to continue
		fmt.Printf("Do you want to continue loading this config file? [y/N]: ")
		var response string
		_, err = fmt.Scanln(&response)
		if err != nil || (response != "y" && response != "Y" && response != "yes" && response != "Yes") {
			return nil, fmt.Errorf("user chose not to load config with version %d", cfg.Version)
		}

		// User chose to continue, log a warning
		_, err = fmt.Fprintf(os.Stderr, "Warning: Continuing with potentially incompatible config version %d\n", cfg.Version)
		if err != nil {
			fmt.Printf("Error writing to stderr: %v\n", err)
		}
	}

	return &cfg, nil
}

// LoadSiteConfig loads a site configuration from the specified file
func LoadSiteConfig(configDir string, filename string) (*SiteConfigFile, error) {
	fullPath := configDir
	if filename != "" {
		fullPath = filepath.Join(configDir, filename)
	}
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open site config file %s: %w", fullPath, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			// If we already have an error, don't overwrite it
			if err == nil {
				err = fmt.Errorf("failed to close site config file: %w", closeErr)
			}
			// Otherwise just log it
			_, logErr := fmt.Fprintf(os.Stderr, "Warning: failed to close site config file: %v\n", closeErr)
			if logErr != nil {
				// If we can't even log to stderr, log to stdout as fallback
				fmt.Printf("Error writing to stderr: %v\n", logErr)
			}
		}
	}()

	// Decode the file content into a map to handle dynamic keys
	var rawData map[string]interface{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&rawData); err != nil {
		return nil, fmt.Errorf("failed to parse site config file: %w", err)
	}

	// Create a new site config
	siteCfg := &SiteConfigFile{
		Config: SiteConfigWrapper{
			Sites: make(map[string]SiteConfigObj),
		},
	}

	// Extract version
	if versionVal, ok := rawData["version"]; ok {
		if versionFloat, ok := versionVal.(float64); ok {
			siteCfg.Version = int(versionFloat)
			// Check if version is 1
			if int(versionFloat) != 1 {
				_, err := fmt.Fprintf(os.Stderr, "Warning: Config file %s has version %d, expected 1.\n", fullPath, int(versionFloat))
				if err != nil {
					fmt.Printf("Error writing to stderr: %v\n", err)
				}

				// Ask user if they want to continue
				fmt.Printf("Do you want to continue loading this config file? [y/N]: ")
				var response string
				_, err = fmt.Scanln(&response)
				if err != nil || (response != "y" && response != "Y" && response != "yes" && response != "Yes") {
					return nil, fmt.Errorf("user chose not to load config with version %d", int(versionFloat))
				}

				// User chose to continue, log a warning
				_, err = fmt.Fprintf(os.Stderr, "Warning: Continuing with potentially incompatible config version %d\n", int(versionFloat))
				if err != nil {
					fmt.Printf("Error writing to stderr: %v\n", err)
				}
			}
		} else {
			return nil, fmt.Errorf("invalid version format in file %s", fullPath)
		}
	} else {
		return nil, fmt.Errorf("missing version field in file %s", fullPath)
	}

	// Extract the config.sites structure
	configVal, hasConfig := rawData["config"]
	if !hasConfig {
		return nil, fmt.Errorf("missing 'config' field in file %s", fullPath)
	}

	configMap, ok := configVal.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid 'config' field format in file %s", fullPath)
	}

	sitesVal, hasSites := configMap["sites"]
	if !hasSites {
		return nil, fmt.Errorf("missing 'config.sites' field in file %s", fullPath)
	}

	sitesData, ok := sitesVal.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid 'config.sites' field format in file %s", fullPath)
	}

	// Process each site in the sites data
	for key, value := range sitesData {
		// Try to unmarshal the value as a SiteConfigObj
		valueBytes, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("failed to process site data for key %s: %w", key, err)
		}

		var siteObj SiteConfigObj
		if err := json.Unmarshal(valueBytes, &siteObj); err != nil {
			return nil, fmt.Errorf("failed to parse site data for key %s: %w", key, err)
		}

		// Add to our sites map
		siteCfg.Config.Sites[key] = siteObj
	}

	// Sanity check - make sure we found at least one site
	if len(siteCfg.Config.Sites) == 0 {
		return nil, fmt.Errorf("no valid site configurations found in file %s", fullPath)
	}

	return siteCfg, nil
}

// GetSiteConfig gets the SiteConfig from a SiteConfigFile for the first site
func (s *SiteConfigFile) GetSiteConfig() *SiteConfig {
	// Get the first site from the map
	for _, siteObj := range s.Config.Sites {
		return &siteObj.SiteConfig
	}
	return nil
}

// GetDevices gets the Devices from a SiteConfigFile for the first site
func (s *SiteConfigFile) GetDevices() *Devices {
	// Get the first site from the map
	for _, siteObj := range s.Config.Sites {
		return &siteObj.Devices
	}
	return nil
}

// LoadAllConfigs loads the main config and all site configs
func LoadAllConfigs(mainConfigFile string) (*Config, []*SiteConfigFile, error) {
	// Load main config
	mainConfig, err := LoadConfig(mainConfigFile)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load main config: %w", err)
	}

	// Create duplicate tracker
	duplicateTracker := NewDuplicateTracker()

	// Load all site configs
	var siteConfigs []*SiteConfigFile

	// Load each config file
	for _, configFile := range mainConfig.Files.SiteConfigs {
		fullPath := filepath.Join(mainConfig.Files.ConfigDir, configFile)

		// Read the raw file data for line number estimation
		rawData, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read site config %s: %w", configFile, err)
		}

		// Parse the JSON to get site structure
		var rawJSON map[string]interface{}
		if err := json.Unmarshal(rawData, &rawJSON); err != nil {
			return nil, nil, fmt.Errorf("failed to parse site config %s: %w", configFile, err)
		}

		// Load the site config using the unified format
		siteConfig, err := LoadSiteConfig(mainConfig.Files.ConfigDir, configFile)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load site config %s: %w", configFile, err)
		}

		// Verify version matches
		if siteConfig.Version != mainConfig.Version {
			_, err := fmt.Fprintf(os.Stderr, "Warning: Version mismatch in config file %s: expected %d, got %d\n",
				configFile, mainConfig.Version, siteConfig.Version)
			if err != nil {
				fmt.Printf("Error writing to stderr: %v\n", err)
			}

			// Log the version mismatch but continue since we already got the user's confirmation at load time
			fmt.Printf("Proceeding with mismatched version config file %s\n", configFile)
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

			duplicateTracker.CheckAndAdd("site_config", "", configName, configFile, line)

			// Create a new config for this site
			individualSiteConfig := &SiteConfigFile{
				Version: siteConfig.Version,
				Config: SiteConfigWrapper{
					Sites: map[string]SiteConfigObj{siteName: siteObj},
				},
			}
			siteConfigs = append(siteConfigs, individualSiteConfig)
		}
	}

	return mainConfig, siteConfigs, nil
}

// ParseCommandLine parses the command-line arguments and returns the options
func ParseCommandLine() CLIOptions {
	opts := CLIOptions{}

	flag.StringVar(&opts.ConfigFile, "c", "config/wifimgr-config.json", "Path to configuration file")

	flag.BoolVar(&opts.Debug, "d", false, "Enable debug mode to show detailed logging information")
	flag.StringVar(&opts.DebugLevel, "debug-level", "info", "Debug level verbosity (options: info or all)")
	flag.StringVar(&opts.Format, "f", "", "Override display format for command output (options: table or csv)")

	flag.BoolVar(&opts.Force, "F", false, "Force operations without confirmation prompts for destructive actions")

	flag.BoolVar(&opts.DryRun, "D", false, "Run in dry-run mode that simulates API calls without making actual changes")
	flag.BoolVar(&opts.RebuildCache, "rebuild-cache", false, "Force rebuild of the local cache")
	flag.BoolVar(&opts.UseEnvFile, "e", false, "Read API token from .env.wifimgr file (useful for testing)")

	flag.Parse()

	// Set debug level based on flags
	if opts.Debug {
		switch opts.DebugLevel {
		case "all":
			opts.DebugLevelInt = DebugAll
		case "info":
			opts.DebugLevelInt = DebugInfo
		default:
			opts.DebugLevelInt = DebugInfo // Default to info if invalid level
		}
	} else {
		opts.DebugLevelInt = DebugNone
	}

	// Validate format if provided
	if opts.Format != "" && opts.Format != "table" && opts.Format != "csv" {
		_, err := fmt.Fprintf(os.Stderr, "Warning: Invalid format '%s'. Using default format from config.\n", opts.Format)
		if err != nil {
			// If there's an error writing to stderr, log it using plain fmt to stdout as a fallback
			fmt.Printf("Error writing to stderr: %v\n", err)
		}
		opts.Format = "" // Reset to empty to use config default
	}

	return opts
}

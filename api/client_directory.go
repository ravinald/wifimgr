package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/xdg"
)

// GetConfigDirectory returns the directory where configuration files are stored
// Uses XDG-compliant paths by default
func (c *mistClient) GetConfigDirectory() string {
	// If we have a local cache path, use its directory
	if c.config.LocalCache != "" {
		// Check if the cache path contains "cache" - if so, consider XDG config dir
		cacheDir := filepath.Dir(c.config.LocalCache)
		if strings.Contains(cacheDir, "cache") {
			// Use XDG config directory
			configDir := xdg.GetConfigDir()
			if _, err := os.Stat(configDir); err == nil {
				logging.Debugf("Using config directory: %s", configDir)
				return configDir
			}
		}

		return cacheDir
	}

	// Fallback to XDG config directory
	return xdg.GetConfigDir()
}

// simpleConfig is a minimal version of config.Config with just the fields we need
// to avoid import cycle
type simpleConfig struct {
	Files struct {
		Schemas string `json:"schemas"`
	} `json:"files"`
}

// GetSchemaDirectory returns the directory where JSON schema files are stored
// This checks if the schemas directory is configured in wifimgr-config.json
func (c *mistClient) GetSchemaDirectory() string {
	configDir := c.GetConfigDirectory()

	// Check if we have a config file to load
	configFile := filepath.Join(configDir, "wifimgr-config.json")
	if _, err := os.Stat(configFile); err == nil {
		// Load the config file to check for schemas directory
		file, err := os.Open(configFile)
		if err != nil {
			logging.Warnf("Failed to open config file to find schema directory: %v", err)
		} else {
			defer func() { _ = file.Close() }()

			// Decode just the schemas field
			var cfg simpleConfig
			if err := json.NewDecoder(file).Decode(&cfg); err != nil {
				logging.Warnf("Failed to parse config file to find schema directory: %v", err)
			} else if cfg.Files.Schemas != "" {
				// We found a schemas directory in the config
				schemaDir := cfg.Files.Schemas
				logging.Debugf("Using schema directory from config: %s", schemaDir)

				// Debug logs for troubleshooting
				_, err := os.Stat(schemaDir)
				if err != nil {
					logging.Warnf("Configured schema directory %s does not exist or is not accessible: %v", schemaDir, err)
				} else {
					schemaFile := filepath.Join(schemaDir, "cache-schema.json")
					_, err := os.Stat(schemaFile)
					if err != nil {
						logging.Warnf("Schema file %s does not exist or is not accessible: %v", schemaFile, err)
					} else {
						logging.Debugf("Found schema file at %s", schemaFile)
					}
				}

				return schemaDir
			}
		}
	}

	// Fallback to XDG data directory for schemas
	schemaDir := xdg.GetSchemasDir()
	logging.Debugf("Using default schema directory: %s", schemaDir)

	// Debug logs for troubleshooting default directory
	_, err := os.Stat(schemaDir)
	if err != nil {
		logging.Warnf("Default schema directory %s does not exist or is not accessible: %v", schemaDir, err)
	} else {
		schemaFile := filepath.Join(schemaDir, "cache-schema.json")
		_, err := os.Stat(schemaFile)
		if err != nil {
			logging.Warnf("Schema file %s does not exist or is not accessible: %v", schemaFile, err)
		} else {
			logging.Debugf("Found schema file at %s", schemaFile)
		}
	}

	return schemaDir
}

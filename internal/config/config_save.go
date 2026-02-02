package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// SaveConfig saves a Config object to the specified file path
func SaveConfig(cfg *Config, filePath string) error {
	// Create the directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Marshal the config to JSON
	jsonData, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(filePath, jsonData, 0644)
}

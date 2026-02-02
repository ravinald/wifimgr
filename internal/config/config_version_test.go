package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigVersionChecking(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "config-version-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create a test config file with incorrect version
	configName := "invalid-version.json"
	configPath := filepath.Join(tempDir, configName)
	configContent := `{
		"version": 2,
		"files": {
			"config_dir": "config",
			"site_configs": ["site1.json"],
			"cache": "cache/api-cache.json",
			"inventory": "config/inventory.json"
		},
		"api": {
			"credentials": {
				"api_id": "test-id",
				"api_token": "test-token",
				"org_id": "test-org-id"
			},
			"url": "https://test-api.mist.com",
			"rate_limit": 5000,
			"results_limit": 100
		},
		"display": {
			"sites": {
				"format": "table",
				"fields": ["Name", "Address"]
			}
		},
		"logging": {
			"enable": true,
			"level": "info",
			"format": "text"
		}
	}`

	err = os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Create a site config file with incorrect version
	siteConfigName := "site-invalid-version.json"
	siteConfigPath := filepath.Join(tempDir, siteConfigName)
	siteConfigContent := `{
		"version": 2,
		"config": {
			"Test Site": {
				"site_config": {
					"name": "Test Site",
					"address": "123 Test St",
					"country_code": "US",
					"timezone": "America/Los_Angeles"
				},
				"devices": {
					"ap": [],
					"switch": [],
					"gateway": []
				}
			}
		}
	}`

	err = os.WriteFile(siteConfigPath, []byte(siteConfigContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create site config file: %v", err)
	}

	// Test loading main config with incorrect version - this would normally prompt the user
	// Since we can't simulate user input in tests, it should return an error
	_, err = LoadConfig(configPath)
	if err == nil {
		t.Errorf("LoadConfig should return an error for incorrect version")
	} else if err.Error() != "user chose not to load config with version 2" {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test loading site config with incorrect version - this would normally prompt the user
	// Since we can't simulate user input in tests, it should return an error
	_, err = LoadSiteConfig(tempDir, siteConfigName)
	if err == nil {
		t.Errorf("LoadSiteConfig should return an error for incorrect version")
	} else if err.Error() != "user chose not to load config with version 2" {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test loading all configs (with version mismatch in site config)
	// Modify main config to point to the invalid site config
	mainConfigName := "main-config.json"
	mainConfigPath := filepath.Join(tempDir, mainConfigName)
	mainConfigContent := `{
		"version": 1,
		"files": {
			"config_dir": "` + tempDir + `",
			"site_configs": ["site-invalid-version.json"],
			"cache": "cache/api-cache.json",
			"inventory": "config/inventory.json"
		},
		"api": {
			"credentials": {
				"api_id": "test-id",
				"api_token": "test-token",
				"org_id": "test-org-id"
			},
			"url": "https://test-api.mist.com",
			"rate_limit": 5000,
			"results_limit": 100
		},
		"display": {
			"sites": {
				"format": "table",
				"fields": ["Name", "Address"]
			}
		},
		"logging": {
			"enable": true,
			"level": "info",
			"format": "text"
		}
	}`

	err = os.WriteFile(mainConfigPath, []byte(mainConfigContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create main config file: %v", err)
	}

	_, _, err = LoadAllConfigs(mainConfigPath)
	if err == nil {
		t.Error("LoadAllConfigs should fail with version mismatch")
	}
}

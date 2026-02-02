package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file for testing
	tempFile, err := os.CreateTemp("", "config-test*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tempFile.Name()) }()

	// Write test config data
	configContent := `{
		"version": 1,
		"files": {
			"config_dir": "config",
			"site_configs": ["site1.json", "site2.json"],
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
			},
			"aps": {
				"format": "table",
				"fields": ["Name", "Mac", "Status"]
			},
			"inventory": {
				"format": "csv",
				"fields": ["Mac", "Serial", "Model"]
			}
		},
		"logging": {
			"enable": true,
			"level": "info",
			"format": "text"
		}
	}`
	_, _ = tempFile.WriteString(configContent)
	_ = tempFile.Close()

	// Test loading valid config
	config, err := LoadConfig(tempFile.Name())
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify loaded config
	if config.Version != 1 {
		t.Errorf("Expected Version 1, got %v", config.Version)
	}
	if config.Files.ConfigDir != "config" {
		t.Errorf("Expected Files.ConfigDir 'config', got '%s'", config.Files.ConfigDir)
	}
	if len(config.Files.SiteConfigs) != 2 || config.Files.SiteConfigs[0] != "site1.json" {
		t.Errorf("Site config files not loaded correctly")
	}
	if config.Files.Cache != "cache/api-cache.json" {
		t.Errorf("Expected Files.Cache 'cache/api-cache.json', got '%s'", config.Files.Cache)
	}
	if config.Files.Inventory != "config/inventory.json" {
		t.Errorf("Expected Files.Inventory 'config/inventory.json', got '%s'", config.Files.Inventory)
	}
	if config.API.Credentials.APIID != "test-id" {
		t.Errorf("Expected API ID 'test-id', got '%s'", config.API.Credentials.APIID)
	}
	if config.API.URL != "https://test-api.mist.com" {
		t.Errorf("Expected API URL 'https://test-api.mist.com', got '%s'", config.API.URL)
	}
	if config.API.RateLimit != 5000 {
		t.Errorf("Expected API rate limit 5000, got %d", config.API.RateLimit)
	}
	if config.Display.Sites.Format != "table" {
		t.Errorf("Expected sites format 'table', got '%s'", config.Display.Sites.Format)
	}
	if !config.Logging.Enable {
		t.Errorf("Expected Logging.Enable to be true")
	}
	if config.Logging.Level != "info" {
		t.Errorf("Expected Logging.Level 'info', got '%s'", config.Logging.Level)
	}
	if config.Logging.Format != "text" {
		t.Errorf("Expected Logging.Format 'text', got '%s'", config.Logging.Format)
	}

	// Test loading non-existent file
	_, err = LoadConfig("nonexistent-file.json")
	if err == nil {
		t.Error("LoadConfig should fail with non-existent file")
	}

	// Test loading invalid JSON
	invalidFile, err := os.CreateTemp("", "invalid-config-test*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(invalidFile.Name()) }()
	_, _ = invalidFile.WriteString("{invalid json")
	_ = invalidFile.Close()

	_, err = LoadConfig(invalidFile.Name())
	if err == nil {
		t.Error("LoadConfig should fail with invalid JSON")
	}
}

func TestLoadSiteConfig(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "site-config-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create a test site config file using unified format
	siteConfigName := "test-site.json"
	siteConfigPath := filepath.Join(tempDir, siteConfigName)
	siteConfigContent := `{
		"version": 1,
		"config": {
			"sites": {
				"Test Site": {
					"site_config": {
						"name": "Test Site",
						"address": "123 Test St",
						"country_code": "US",
						"timezone": "America/Los_Angeles",
						"notes": "Test site notes"
					},
					"devices": {
						"ap": {
							"00:11:22:33:44:55": {
								"name": "Test AP",
								"serial": "TEST123",
								"mac": "00:11:22:33:44:55",
								"tags": ["test", "lab"],
								"notes": "Test AP notes",
								"location": [37.7749, -122.4194],
								"orientation": 90,
								"config": {
									"led_enabled": true,
									"band_24": {
										"disabled": false,
										"tx_power": 10,
										"channel": 6
									},
									"band_5": {
										"disabled": false,
										"tx_power": 15,
										"channel": 36
									}
								}
							}
						},
						"switch": {},
						"gateway": {}
					}
				}
			}
		}
	}`

	err = os.WriteFile(siteConfigPath, []byte(siteConfigContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create site config file: %v", err)
	}

	// Test loading the site config
	siteConfig, err := LoadSiteConfig(tempDir, siteConfigName)
	if err != nil {
		t.Fatalf("LoadSiteConfig failed: %v", err)
	}

	// Verify loaded site config
	if siteConfig.Version != 1 {
		t.Errorf("Expected Version 1, got %v", siteConfig.Version)
	}

	// Check if we have the site
	siteObj, exists := siteConfig.Config.Sites["Test Site"]
	if !exists {
		t.Errorf("Expected site 'Test Site' not found in Sites map")
	}

	// Verify the site details
	if siteObj.SiteConfig.Name != "Test Site" {
		t.Errorf("Expected site name 'Test Site', got '%s'", siteObj.SiteConfig.Name)
	}

	// Verify the devices
	if len(siteObj.Devices.APs) != 1 {
		t.Errorf("Expected 1 AP, got %d", len(siteObj.Devices.APs))
	}

	// Check the AP by its MAC address key
	ap, exists := siteObj.Devices.APs["00:11:22:33:44:55"]
	if !exists {
		t.Errorf("Expected AP with MAC '00:11:22:33:44:55' not found")
	} else if ap.Name != "Test AP" {
		t.Errorf("Expected AP name 'Test AP', got '%s'", ap.Name)
	}

	// Test loading non-existent site config
	_, err = LoadSiteConfig(tempDir, "nonexistent.json")
	if err == nil {
		t.Error("LoadSiteConfig should fail with non-existent file")
	}
}

func TestLoadAllConfigs(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "all-configs-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create a main config file
	mainConfigName := "main-config.json"
	mainConfigPath := filepath.Join(tempDir, mainConfigName)
	mainConfigContent := `{
		"version": 1,
		"files": {
			"config_dir": "` + tempDir + `",
			"site_configs": ["site1.json", "site2.json"],
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
			},
			"aps": {
				"format": "table",
				"fields": ["Name", "Mac", "Status"]
			},
			"inventory": {
				"format": "csv",
				"fields": ["Mac", "Serial", "Model"]
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

	// Create site1 config using the unified format
	site1ConfigPath := filepath.Join(tempDir, "site1.json")
	site1ConfigContent := `{
		"version": 1,
		"config": {
			"sites": {
				"Site 1": {
					"site_config": {
						"name": "Site 1",
						"address": "123 Site 1 St",
						"country_code": "US",
						"timezone": "America/Los_Angeles",
						"notes": "Site 1 notes"
					},
					"devices": {
						"ap": {},
						"switch": {},
						"gateway": {}
					}
				}
			}
		}
	}`

	err = os.WriteFile(site1ConfigPath, []byte(site1ConfigContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create site1 config file: %v", err)
	}

	// Create site2 config using the unified format
	site2ConfigPath := filepath.Join(tempDir, "site2.json")
	site2ConfigContent := `{
		"version": 1,
		"config": {
			"sites": {
				"Site 2": {
					"site_config": {
						"name": "Site 2",
						"address": "456 Site 2 St",
						"country_code": "CA",
						"timezone": "America/Vancouver",
						"notes": "Site 2 notes"
					},
					"devices": {
						"ap": {},
						"switch": {},
						"gateway": {}
					}
				}
			}
		}
	}`

	err = os.WriteFile(site2ConfigPath, []byte(site2ConfigContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create site2 config file: %v", err)
	}

	// Create site3 config with version mismatch using the unified format
	site3ConfigPath := filepath.Join(tempDir, "site3.json")
	site3ConfigContent := `{
		"version": 2,
		"config": {
			"sites": {
				"Site 3": {
					"site_config": {
						"name": "Site 3",
						"address": "789 Site 3 St",
						"country_code": "UK",
						"timezone": "Europe/London",
						"notes": "Site 3 notes"
					},
					"devices": {
						"ap": {},
						"switch": {},
						"gateway": {}
					}
				}
			}
		}
	}`

	err = os.WriteFile(site3ConfigPath, []byte(site3ConfigContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create site3 config file: %v", err)
	}

	// Test loading all configs
	mainConfig, siteConfigs, err := LoadAllConfigs(mainConfigPath)
	if err != nil {
		t.Fatalf("LoadAllConfigs failed: %v", err)
	}

	// Verify loaded configs
	if mainConfig.Version != 1 {
		t.Errorf("Expected main config version 1, got %v", mainConfig.Version)
	}
	if len(mainConfig.Files.SiteConfigs) != 2 {
		t.Errorf("Expected 2 config files, got %d", len(mainConfig.Files.SiteConfigs))
	}
	if len(siteConfigs) != 2 {
		t.Errorf("Expected 2 site configs, got %d", len(siteConfigs))
	}

	// Get first site's name from Sites map
	var firstSiteName, secondSiteName string
	for site := range siteConfigs[0].Config.Sites {
		firstSiteName = siteConfigs[0].Config.Sites[site].SiteConfig.Name
		break
	}

	// Get second site's name from Sites map
	for site := range siteConfigs[1].Config.Sites {
		secondSiteName = siteConfigs[1].Config.Sites[site].SiteConfig.Name
		break
	}

	// Check site names - note the order may vary
	if firstSiteName != "Site 1" && secondSiteName != "Site 1" {
		t.Errorf("Expected site name 'Site 1' not found")
	}

	if firstSiteName != "Site 2" && secondSiteName != "Site 2" {
		t.Errorf("Expected site name 'Site 2' not found")
	}

	// Test version mismatch detection
	mainConfigWithSite3Content := `{
		"version": 1,
		"files": {
			"config_dir": "` + tempDir + `",
			"site_configs": ["site3.json"],
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
			},
			"aps": {
				"format": "table",
				"fields": ["Name", "Mac", "Status"]
			},
			"inventory": {
				"format": "csv",
				"fields": ["Mac", "Serial", "Model"]
			}
		},
		"logging": {
			"enable": true,
			"level": "info",
			"format": "text"
		}
	}`

	mainConfigWithSite3Path := filepath.Join(tempDir, "main-with-site3.json")
	err = os.WriteFile(mainConfigWithSite3Path, []byte(mainConfigWithSite3Content), 0644)
	if err != nil {
		t.Fatalf("Failed to create main-with-site3 config file: %v", err)
	}

	// This should fail due to version mismatch
	_, _, err = LoadAllConfigs(mainConfigWithSite3Path)
	if err == nil {
		t.Error("LoadAllConfigs should fail with version mismatch")
	}
}

func TestParseCommandLine(t *testing.T) {
	// Since we're using the global flag package, and this test is already covered
	// by integration tests, we'll just do a simple validity check here
	// to avoid "flag redefined" errors when running multiple times

	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Set up test case
	os.Args = []string{"app"}

	// Call ParseCommandLine
	opts := ParseCommandLine()

	// Check default values
	if opts.ConfigFile != "config/wifimgr-config.json" {
		t.Errorf("ConfigFile: expected config/wifimgr-config.json, got %s", opts.ConfigFile)
	}
	if opts.Debug != false {
		t.Errorf("Debug: expected false, got %v", opts.Debug)
	}
	if opts.DebugLevel != "info" {
		t.Errorf("DebugLevel: expected info, got %s", opts.DebugLevel)
	}
	if opts.DebugLevelInt != DebugNone {
		t.Errorf("DebugLevelInt: expected %d, got %d", DebugNone, opts.DebugLevelInt)
	}
	if opts.Format != "" {
		t.Errorf("Format: expected empty string, got %s", opts.Format)
	}
	if opts.Force != false {
		t.Errorf("Force: expected false, got %v", opts.Force)
	}
	if opts.RebuildCache != false {
		t.Errorf("RebuildCache: expected false, got %v", opts.RebuildCache)
	}

	// Note: We cannot test the -rebuild-cache flag here because running ParseCommandLine
	// multiple times causes the "flag redefined" error. The flag is properly defined
	// in the ParseCommandLine function and will be tested through integration tests.
}

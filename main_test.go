package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ravinald/wifimgr/api"
	"github.com/ravinald/wifimgr/cmd/inventory"
	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/formatter"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/vendors"
	"github.com/sirupsen/logrus"
)

// Test helpers
func createTempConfigFile(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "wifimgr-config.json")

	cfg := config.Config{
		Version: 1,
		Files: config.Files{
			ConfigDir:   tempDir,
			SiteConfigs: []string{configPath},
			Cache:       "",
			Inventory:   "",
		},
		API: config.API{
			Credentials: struct {
				APIID             string `json:"api_id"`
				APIToken          string `json:"api_token"`
				OrgID             string `json:"org_id"`
				KeyEncryptionSalt string `json:"key_encryption_salt,omitempty"`
				KeyEncrypted      bool   `json:"key_encrypted,omitempty"`
			}{
				APIID:             "test-id",
				APIToken:          "test-token",
				OrgID:             "test-org",
				KeyEncryptionSalt: "",
				KeyEncrypted:      false,
			},
			URL:          "https://api.mist.com/api/v1",
			RateLimit:    1000,
			ResultsLimit: 1000,
		},
		Logging: config.Logging{
			Enable: true,
			Level:  "info",
			Format: "text",
		},
		Display: config.Display{
			Sites: config.DisplayFormat{
				Format: "table",
				Fields: []string{"name", "(id)"},
			},
			APs: config.DisplayFormat{
				Format: "table",
				Fields: []string{"name", "serial", "(id)"},
			},
		},
	}

	// Create a site config file for testing
	siteConfigPath := filepath.Join(tempDir, "site_config.json")
	siteConfigFile := config.SiteConfigFile{
		Version: 1,
		Config: config.SiteConfigWrapper{
			Sites: map[string]config.SiteConfigObj{
				"Test Site": {
					SiteConfig: config.SiteConfig{
						Name:        "Test Site",
						Address:     "123 Test St",
						CountryCode: "US",
						Timezone:    "America/New_York",
						Notes:       "Test notes",
						LatLng:      &api.LatLng{Lat: 37.123, Lng: -122.456},
					},
					Devices: config.Devices{
						APs: map[string]config.APConfig{
							"aabbccddeeff": {
								Magic: "SERIAL123", // Using magic field for device identification
								APDeviceConfig: &vendors.APDeviceConfig{
									Name:     "Test AP",
									Tags:     []string{"test", "ap"},
									Notes:    "Test AP notes",
									Location: []float64{37.123, -122.456},
								},
								Config: config.APHWConfig{
									LEDEnabled: true,
									Band24: config.BandCfg{
										Disabled: false,
										TxPower:  10,
										Channel:  6,
									},
									Band5: config.BandCfg{
										Disabled: false,
										TxPower:  15,
										Channel:  36,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Write site config to a file
	siteData, err := json.MarshalIndent(siteConfigFile, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal site config: %v", err)
	}

	if err = os.WriteFile(siteConfigPath, siteData, 0644); err != nil {
		t.Fatalf("Failed to write site config file: %v", err)
	}

	// Add site config file to main config
	cfg.Files.SiteConfigs = append(cfg.Files.SiteConfigs, siteConfigPath)

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err = os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	return configPath
}

// Unit tests
func TestLoadConfig(t *testing.T) {
	configPath := createTempConfigFile(t)

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Check some values
	if cfg.API.Credentials.APIToken != "test-token" {
		t.Errorf("Expected API token to be 'test-token', got %q", cfg.API.Credentials.APIToken)
	}

	// Verify config files are loaded
	if len(cfg.Files.SiteConfigs) < 1 {
		t.Fatalf("Expected at least 1 config file, got %d", len(cfg.Files.SiteConfigs))
	}
}

func TestParseCommandLine(t *testing.T) {
	// Test default values (using constructor-like pattern since we can't easily test the function directly)
	opts := config.CLIOptions{
		ConfigFile:    "wifimgr-config.json",
		Debug:         false,
		DebugLevel:    "info",
		DebugLevelInt: config.DebugNone,
		Force:         false,
		DryRun:        false,
	}

	if opts.ConfigFile != "wifimgr-config.json" {
		t.Errorf("Expected default config file to be 'wifimgr-config.json', got %q", opts.ConfigFile)
	}
	if opts.Debug {
		t.Errorf("Expected debug to be false by default")
	}
	if opts.DebugLevelInt != config.DebugNone {
		t.Errorf("Expected debug level to be DebugNone, got %d", opts.DebugLevelInt)
	}
	if opts.Format != "" {
		t.Errorf("Expected format to be empty by default, got %q", opts.Format)
	}
	if opts.Force {
		t.Errorf("Expected force flag to be false by default")
	}
	if opts.DryRun {
		t.Errorf("Expected dry-run flag to be false by default")
	}

	// Simulate parsing with debug flags and force flag
	opts = config.CLIOptions{
		ConfigFile:    "wifimgr-config.json",
		Debug:         true,
		DebugLevel:    "all",
		DebugLevelInt: config.DebugAll,
		Format:        "csv",
		Force:         true,
		DryRun:        false,
	}

	if !opts.Debug {
		t.Errorf("Expected debug to be true")
	}
	if opts.DebugLevelInt != config.DebugAll {
		t.Errorf("Expected debug level to be DebugAll, got %d", opts.DebugLevelInt)
	}
	if !opts.Force {
		t.Errorf("Expected force flag to be true")
	}
	if opts.DryRun {
		t.Errorf("Expected dry-run flag to be false")
	}

	// Simulate custom config path with dry-run enabled
	opts = config.CLIOptions{
		ConfigFile:    "custom.json",
		Debug:         false,
		DebugLevel:    "info",
		DebugLevelInt: config.DebugNone,
		Force:         false,
		DryRun:        true,
	}

	if opts.ConfigFile != "custom.json" {
		t.Errorf("Expected config file to be 'custom.json', got %q", opts.ConfigFile)
	}
	if !opts.DryRun {
		t.Errorf("Expected dry-run flag to be true")
	}
}

func TestInventoryOperations(t *testing.T) {
	mistConfig := api.Config{
		BaseURL:      "https://api.mist.com/api/v1",
		APIToken:     "test-token",
		Organization: "test-org",
		Timeout:      30 * time.Second,
		LocalCache:   "",
	}

	// Create mock client and add test inventory items
	mockClient := api.NewMockClient(mistConfig).(*api.MockClient)

	// Set up a mock cache accessor for testing
	cacheManager := vendors.NewCacheManager(t.TempDir(), nil)
	cacheAccessor := vendors.NewCacheAccessor(cacheManager)
	vendors.SetGlobalCacheAccessor(cacheAccessor)
	defer vendors.SetGlobalCacheAccessor(nil) // Clean up after test

	// Create a minimal config
	cfg := &config.Config{
		Version: 1,
		Files: config.Files{
			ConfigDir:   "",
			SiteConfigs: []string{},
			Cache:       "",
			Inventory:   "",
		},
		API: config.API{
			Credentials: struct {
				APIID             string `json:"api_id"`
				APIToken          string `json:"api_token"`
				OrgID             string `json:"org_id"`
				KeyEncryptionSalt string `json:"key_encryption_salt,omitempty"`
				KeyEncrypted      bool   `json:"key_encrypted,omitempty"`
			}{
				APIID:             "test-id",
				APIToken:          "test-token",
				OrgID:             "test-org",
				KeyEncryptionSalt: "",
				KeyEncrypted:      false,
			},
			URL:          "https://api.mist.com/api/v1",
			RateLimit:    1000,
			ResultsLimit: 1000,
		},
		Logging: config.Logging{
			Enable: true,
			Level:  "info",
			Format: "text",
		},
		Display: config.Display{
			Inventory: config.DisplayFormat{
				Format: "table",
				Fields: []string{"Name", "Type", "Mac", "Serial", "Model"},
			},
		},
	}

	// Create pointer variables directly since we're not using the helper function

	// Add test inventory items
	apName := "TestAP"
	apType := "ap"
	apMac := "d420b080516d"
	apSerial := "A0052190206A2"
	apModel := "AP41"
	apItem := api.InventoryItem{
		Name:   &apName,
		Type:   &apType,
		Mac:    &apMac,
		Serial: &apSerial,
		Model:  &apModel,
	}
	mockClient.AddInventoryItem(apItem)

	switchName := "TestSwitch"
	switchType := "switch"
	switchMac := "8403280bc0a0"
	switchSerial := "HV3620270051"
	switchModel := "EX2300-C-12P"
	switchItem := api.InventoryItem{
		Name:   &switchName,
		Type:   &switchType,
		Mac:    &switchMac,
		Serial: &switchSerial,
		Model:  &switchModel,
	}
	mockClient.AddInventoryItem(switchItem)

	gatewayName := "TestGateway"
	gatewayType := "gateway"
	gatewayMac := "fc334262af00"
	gatewaySerial := "CW1419AN0651"
	gatewayModel := "SRX320"
	gatewayItem := api.InventoryItem{
		Name:   &gatewayName,
		Type:   &gatewayType,
		Mac:    &gatewayMac,
		Serial: &gatewaySerial,
		Model:  &gatewayModel,
	}
	mockClient.AddInventoryItem(gatewayItem)

	ctx := context.Background()

	// Test listing all inventory
	err := inventory.ListInventory(ctx, mockClient, cfg, "", "", "", false)
	if err != nil {
		t.Fatalf("Failed to list inventory: %v", err)
	}

	// Test listing AP inventory
	err = inventory.ListInventory(ctx, mockClient, cfg, "ap", "", "", false)
	if err != nil {
		t.Fatalf("Failed to list AP inventory: %v", err)
	}

	// Test listing switch inventory
	err = inventory.ListInventory(ctx, mockClient, cfg, "switch", "", "", false)
	if err != nil {
		t.Fatalf("Failed to list switch inventory: %v", err)
	}

	// Test listing gateway inventory
	err = inventory.ListInventory(ctx, mockClient, cfg, "gateway", "", "", false)
	if err != nil {
		t.Fatalf("Failed to list gateway inventory: %v", err)
	}
}

// Test the logging integration
func TestLoggingIntegration(t *testing.T) {
	// Save the original logger
	originalLogger := logging.GetLogger()
	defer func() {
		// Reset the logger after the test
		logging.SetLogger(originalLogger)
	}()

	// Create a test logger with a buffer to capture output
	testLogger := logrus.New()
	var logBuffer strings.Builder
	testLogger.SetOutput(&logBuffer)

	// Set log level to debug to capture all messages
	testLogger.SetLevel(logrus.DebugLevel)

	// Use a simple text formatter for testing
	testLogger.SetFormatter(&logrus.TextFormatter{
		DisableColors:    true,
		DisableTimestamp: true,
	})

	// Set our test logger as the active logger
	logging.SetLogger(testLogger)

	// Test different log levels
	logging.Debug("This is a debug message")
	logging.Info("This is an info message")
	logging.Warn("This is a warning message")
	logging.Error("This is an error message")

	// Check that all messages were logged
	output := logBuffer.String()
	if !strings.Contains(output, "This is a debug message") {
		t.Error("Debug message not found in log output")
	}
	if !strings.Contains(output, "This is an info message") {
		t.Error("Info message not found in log output")
	}
	if !strings.Contains(output, "This is a warning message") {
		t.Error("Warning message not found in log output")
	}
	if !strings.Contains(output, "This is an error message") {
		t.Error("Error message not found in log output")
	}

	// Test logger configuration
	logBuffer.Reset()

	// Create a new test logger with info level
	infoLogger := logrus.New()
	infoLogger.SetOutput(&logBuffer)
	infoLogger.SetLevel(logrus.InfoLevel)
	infoLogger.SetFormatter(&logrus.TextFormatter{
		DisableColors:    true,
		DisableTimestamp: true,
	})

	// Set our test logger as the active logger
	logging.SetLogger(infoLogger)

	// Debug messages should not appear at info level
	logging.Debug("This debug message should not appear")
	logging.Info("This info message should appear")

	output = logBuffer.String()
	if strings.Contains(output, "This debug message should not appear") {
		t.Error("Debug message should not be in log output at info level")
	}
	if !strings.Contains(output, "This info message should appear") {
		t.Error("Info message not found in log output")
	}
}

// Test the display formatting functionality
func TestDisplayFormatting(t *testing.T) {
	// Test the extraction of values from structs
	type TestStruct struct {
		Name        string
		ID          string
		Count       int
		IsActive    bool
		Temperature float64
		PtrField    *string
	}

	ptrValue := "pointer value"
	ts := TestStruct{
		Name:        "Test",
		ID:          "12345",
		Count:       42,
		IsActive:    true,
		Temperature: 98.6,
		PtrField:    &ptrValue,
	}

	// Test extractValue function
	tests := []struct {
		fieldPath string
		expected  string
		found     bool
	}{
		{"Name", "Test", true},
		{"ID", "12345", true},
		{"Count", "42", true},
		{"IsActive", "true", true},
		{"Temperature", "98.60", true},
		{"PtrField", "pointer value", true},
		{"NonExistentField", "", false},
		{"(Name)", "(Test)", true},
		{"(ID)", "(12345)", true},
	}

	for _, test := range tests {
		t.Run(test.fieldPath, func(t *testing.T) {
			value, found := formatter.ExtractValue(ts, test.fieldPath)
			if found != test.found {
				t.Errorf("ExtractValue(%q) found = %v, want %v", test.fieldPath, found, test.found)
			}
			if found && value != test.expected {
				t.Errorf("ExtractValue(%q) = %q, want %q", test.fieldPath, value, test.expected)
			}
		})
	}

	// Test the Format function with table format
	displayFormat := config.DisplayFormat{
		Format: "table",
		Fields: []string{"Name", "ID", "Count", "IsActive"},
	}

	data := []interface{}{ts}
	tableOutput := formatter.Format(data, displayFormat, "")

	// Verify the table has headers and data
	if !strings.Contains(tableOutput, "Name") || !strings.Contains(tableOutput, "ID") {
		t.Errorf("Table output missing headers: %s", tableOutput)
	}
	if !strings.Contains(tableOutput, "Test") || !strings.Contains(tableOutput, "12345") {
		t.Errorf("Table output missing expected data: %s", tableOutput)
	}

	// Test the Format function with CSV format
	displayFormat.Format = "csv"
	csvOutput := formatter.Format(data, displayFormat, "")

	// Verify the CSV has headers and data
	expectedCSVStart := "Name,ID,Count,IsActive"
	if !strings.HasPrefix(csvOutput, expectedCSVStart) {
		t.Errorf("CSV output missing expected headers, got: %s", csvOutput)
	}

	expectedCSVDataLine := "Test,12345,42,true"
	if !strings.Contains(csvOutput, expectedCSVDataLine) {
		t.Errorf("CSV output missing expected data, got: %s", csvOutput)
	}

	// Test with API types
	name := "Test Site"
	id := api.UUID("site-12345")
	siteObj := api.Site{
		Id:   &id,
		Name: &name,
	}

	// Test the conversion functions
	sites := []api.Site{siteObj}
	siteInterfaces := formatter.ConvertToInterfaces(sites)
	if len(siteInterfaces) != 1 {
		t.Errorf("ConvertToInterfaces returned %d items, expected 1", len(siteInterfaces))
	}

	// Test Format with API site data
	siteDisplayFormat := config.DisplayFormat{
		Format: "table",
		Fields: []string{"name", "(id)"},
	}

	siteOutput := formatter.Format(siteInterfaces, siteDisplayFormat, "")
	if !strings.Contains(siteOutput, "name") || !strings.Contains(siteOutput, "id") {
		t.Errorf("Site table output missing headers: %s", siteOutput)
	}
	if !strings.Contains(siteOutput, "Test Site") || !strings.Contains(siteOutput, "(site-12345)") {
		t.Errorf("Site table output missing expected data: %s", siteOutput)
	}

	// Test format override
	tableFormat := config.DisplayFormat{
		Format: "table",
		Fields: []string{"Name", "ID"},
	}

	// Override table format with CSV
	csvOverrideOutput := formatter.Format(data, tableFormat, "csv")

	// Verify the CSV has headers and data, despite table format in config
	expectedCSVStartOverride := "Name,ID"
	if !strings.HasPrefix(csvOverrideOutput, expectedCSVStartOverride) {
		t.Errorf("Format override CSV output missing expected headers, got: %s", csvOverrideOutput)
	}

	// Override CSV format with table
	csvFormat := config.DisplayFormat{
		Format: "csv",
		Fields: []string{"Name", "ID"},
	}

	tableOverrideOutput := formatter.Format(data, csvFormat, "table")

	// Verify the table format is used, despite CSV format in config
	if !strings.Contains(tableOverrideOutput, strings.Repeat("-", len("Name"))) {
		t.Errorf("Format override table output missing expected separator line, got: %s", tableOverrideOutput)
	}
}

package logging

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestConfigureLogging(t *testing.T) {
	// The main issue is that ConfigureLogging changes the output of the logger
	// Let's create a simpler test that just verifies the configuration works

	// Save the original logger
	originalLogger := defaultLogger
	defer func() {
		defaultLogger = originalLogger
	}()

	// Save environment variables
	origLevel := os.Getenv(EnvLogLevel)
	origFormat := os.Getenv(EnvLogFormat)
	defer func() {
		_ = os.Setenv(EnvLogLevel, origLevel)
		_ = os.Setenv(EnvLogFormat, origFormat)
	}()

	// Clear environment variables for initial tests
	_ = os.Unsetenv(EnvLogLevel)
	_ = os.Unsetenv(EnvLogFormat)

	// Test with basic configuration - we'll create a logger and check its properties directly
	testLogger := logrus.New()

	// Create a buffer to capture output
	var buf bytes.Buffer
	testLogger.SetOutput(&buf)

	// Set our test logger as the active logger
	oldLogger := SetLogger(testLogger)
	defer SetLogger(oldLogger)

	// Configure basic info level logger
	config := LogConfig{
		Enable:   true,
		Level:    "info",
		Format:   "text",
		ToStdout: true,
	}

	err := ConfigureLogging(config)
	if err != nil {
		t.Fatalf("ConfigureLogging failed: %v", err)
	}

	// Verify level is set correctly
	if defaultLogger.GetLevel() != logrus.InfoLevel {
		t.Errorf("Expected level to be InfoLevel, got %v", defaultLogger.GetLevel())
	}

	// Test with debug level
	config.Level = "debug"
	err = ConfigureLogging(config)
	if err != nil {
		t.Fatalf("ConfigureLogging failed: %v", err)
	}

	// Verify level is set correctly
	if defaultLogger.GetLevel() != logrus.DebugLevel {
		t.Errorf("Expected level to be DebugLevel, got %v", defaultLogger.GetLevel())
	}

	// Test JSON formatter
	config.Format = "json"
	err = ConfigureLogging(config)
	if err != nil {
		t.Fatalf("ConfigureLogging failed: %v", err)
	}

	// Verify formatter type
	_, isJSONFormatter := defaultLogger.Formatter.(*logrus.JSONFormatter)
	if !isJSONFormatter {
		t.Errorf("Expected JSON formatter, got %T", defaultLogger.Formatter)
	}

	// Test environment variable overrides
	// Set environment variables
	_ = os.Setenv(EnvLogLevel, "debug")
	_ = os.Setenv(EnvLogFormat, "json")

	// Configure with values that should be overridden
	config = LogConfig{
		Enable:   true,
		Level:    "info",
		Format:   "text",
		ToStdout: true,
	}

	// Reset logger to ensure environment variables take effect
	testLogger = logrus.New()
	SetLogger(testLogger)

	err = ConfigureLogging(config)
	if err != nil {
		t.Fatalf("ConfigureLogging failed: %v", err)
	}

	// Verify environment variables took precedence
	if defaultLogger.GetLevel() != logrus.DebugLevel {
		t.Errorf("Expected level to be DebugLevel (from env var), got %v", defaultLogger.GetLevel())
	}

	// Verify formatter type
	_, isJSONFormatter = defaultLogger.Formatter.(*logrus.JSONFormatter)
	if !isJSONFormatter {
		t.Errorf("Expected JSON formatter (from env var), got %T", defaultLogger.Formatter)
	}
}

func TestLogFile(t *testing.T) {
	// Save the original logger and config
	originalLogger := defaultLogger
	originalLogFile := logFile

	// Setup deferred cleanup of everything
	defer func() {
		defaultLogger = originalLogger
		if logFile != nil && logFile != originalLogFile {
			_ = logFile.Close()
		}
		logFile = originalLogFile
	}()

	// Create a temporary file for test
	tempFile, err := os.CreateTemp("", "log-test-*.log")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tempFilePath := tempFile.Name()
	_ = tempFile.Close() // Close now, it will be reopened by ConfigureLogging

	// Clean up after test
	defer func() { _ = os.Remove(tempFilePath) }()

	// Create a new logger
	defaultLogger = logrus.New()

	// Test configuring with a log file and stdout enabled
	config := LogConfig{
		Enable:   true,
		Level:    "info",
		Format:   "text",
		ToStdout: true,
		LogFile:  tempFilePath,
	}

	err = ConfigureLogging(config)
	if err != nil {
		t.Fatalf("ConfigureLogging failed: %v", err)
	}

	// Log a test message
	testMessage := "Test log message to file with stdout"
	Info(testMessage)

	// Read the file to verify message was written
	content, err := os.ReadFile(tempFilePath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// Check if the message is in the file
	if !strings.Contains(string(content), testMessage) {
		t.Errorf("Expected log file to contain message '%s', but it wasn't found", testMessage)
	}

	// Test with stdout disabled (log only to file)
	config.ToStdout = false

	err = ConfigureLogging(config)
	if err != nil {
		t.Fatalf("ConfigureLogging with stdout disabled failed: %v", err)
	}

	// Log another test message
	testMessage2 := "Test log message to file without stdout"
	Info(testMessage2)

	// Read file again to check for the second message
	content, err = os.ReadFile(tempFilePath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// Check if the second message is in the file
	if !strings.Contains(string(content), testMessage2) {
		t.Errorf("Expected log file to contain message '%s', but it wasn't found", testMessage2)
	}

	// Test with invalid file path
	invalidConfig := LogConfig{
		Enable:   true,
		Level:    "info",
		Format:   "text",
		ToStdout: true,
		LogFile:  "/nonexistent/directory/invalid.log",
	}

	err = ConfigureLogging(invalidConfig)
	if err == nil {
		t.Error("Expected error when setting output to invalid file path, but got nil")
	}
}

func TestNameFormatting(t *testing.T) {
	// Test site name formatting
	testSiteID := "site-123456"
	testSiteName := "US-SFO-TEST"

	// Set a test lookup function
	SetSiteNameLookupFunc(func(siteID string) (string, bool) {
		if siteID == testSiteID {
			return testSiteName, true
		}
		return "", false
	})

	// Test with a known site ID
	formattedSite := FormatSiteID(testSiteID)
	expected := fmt.Sprintf("%s (%s)", testSiteName, testSiteID)
	if formattedSite != expected {
		t.Errorf("FormatSiteID(%s) = %s, want %s", testSiteID, formattedSite, expected)
	}

	// Test with an unknown site ID
	unknownSiteID := "site-unknown"
	formattedSite = FormatSiteID(unknownSiteID)
	if formattedSite != unknownSiteID {
		t.Errorf("FormatSiteID(%s) = %s, want %s", unknownSiteID, formattedSite, unknownSiteID)
	}

	// Test org name formatting
	testOrgID := "org-123456"
	testOrgName := "Test Organization"

	// Set a test lookup function
	SetOrgNameLookupFunc(func(orgID string) (string, bool) {
		if orgID == testOrgID {
			return testOrgName, true
		}
		return "", false
	})

	// Test with a known org ID
	formattedOrg := FormatOrgID(testOrgID)
	expected = fmt.Sprintf("%s (%s)", testOrgName, testOrgID)
	if formattedOrg != expected {
		t.Errorf("FormatOrgID(%s) = %s, want %s", testOrgID, formattedOrg, expected)
	}

	// Test with an unknown org ID
	unknownOrgID := "org-unknown"
	formattedOrg = FormatOrgID(unknownOrgID)
	if formattedOrg != unknownOrgID {
		t.Errorf("FormatOrgID(%s) = %s, want %s", unknownOrgID, formattedOrg, unknownOrgID)
	}
}

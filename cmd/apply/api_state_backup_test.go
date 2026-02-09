package apply

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/vendors"
)

func TestAPIStateBackupStructure(t *testing.T) {
	// Test that the backup structure marshals correctly
	backup := APIStateBackup{
		Version:     1,
		Timestamp:   time.Now().UTC(),
		SiteName:    "US-OAK-TEST",
		SiteID:      "test-site-id",
		APILabel:    "mist",
		DeviceType:  "ap",
		DeviceCount: 2,
		Operation:   "pre_apply",
		DeviceStates: &APStateBackup{
			APs: map[string]*vendors.APConfig{
				"aa:bb:cc:dd:ee:01": {
					ID:   "ap-id-1",
					Name: "AP-1",
					MAC:  "aa:bb:cc:dd:ee:01",
				},
				"aa:bb:cc:dd:ee:02": {
					ID:   "ap-id-2",
					Name: "AP-2",
					MAC:  "aa:bb:cc:dd:ee:02",
				},
			},
		},
	}

	data, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal backup: %v", err)
	}

	// Verify we can unmarshal it back
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal backup: %v", err)
	}

	// Check required fields
	if parsed["version"].(float64) != 1 {
		t.Errorf("Expected version 1, got %v", parsed["version"])
	}
	if parsed["site_name"].(string) != "US-OAK-TEST" {
		t.Errorf("Expected site_name 'US-OAK-TEST', got %v", parsed["site_name"])
	}
	if parsed["device_type"].(string) != "ap" {
		t.Errorf("Expected device_type 'ap', got %v", parsed["device_type"])
	}
	if parsed["device_count"].(float64) != 2 {
		t.Errorf("Expected device_count 2, got %v", parsed["device_count"])
	}
	if parsed["operation"].(string) != "pre_apply" {
		t.Errorf("Expected operation 'pre_apply', got %v", parsed["operation"])
	}
}

func TestCreateAPIStateBackup_EmptyDevices(t *testing.T) {
	cfg := &config.Config{}

	// Should not error with empty devices - just skip backup
	err := createAPIStateBackup(cfg, "test-site", "site-id", "ap", "mist", []string{})
	if err != nil {
		t.Errorf("Expected no error for empty devices, got: %v", err)
	}
}

func TestListAPIStateBackups_NoBackupDir(t *testing.T) {
	// Should return nil (not error) when backup dir doesn't exist
	backups, err := listAPIStateBackups("nonexistent-site", "ap")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if backups != nil {
		t.Errorf("Expected nil backups, got: %v", backups)
	}
}

func TestBackupRotation(t *testing.T) {
	// Create a temp directory for backups
	tempDir, err := os.MkdirTemp("", "wifimgr-backup-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	baseFileName := "test-site-api-state-ap.json"
	maxBackups := 3

	// Create initial backups
	for i := range 5 {
		backupPath := filepath.Join(tempDir, baseFileName+".0")
		if err := rotateConfigFileBackups(tempDir, baseFileName, maxBackups); err != nil {
			t.Fatalf("Rotation %d failed: %v", i, err)
		}
		// Write a new .0 backup
		if err := os.WriteFile(backupPath, []byte("backup content"), 0644); err != nil {
			t.Fatalf("Failed to write backup %d: %v", i, err)
		}
	}

	// Should only have maxBackups files
	files, err := filepath.Glob(filepath.Join(tempDir, baseFileName+".*"))
	if err != nil {
		t.Fatalf("Failed to list files: %v", err)
	}

	if len(files) != maxBackups {
		t.Errorf("Expected %d backup files, got %d: %v", maxBackups, len(files), files)
	}
}

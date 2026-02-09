package apply

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/vendors"
	"github.com/ravinald/wifimgr/internal/xdg"
)

// APIStateBackup represents the API state of devices before applying changes.
// This captures the "running config" from the API for rollback purposes.
type APIStateBackup struct {
	Version      int       `json:"version"`
	Timestamp    time.Time `json:"timestamp"`
	SiteName     string    `json:"site_name"`
	SiteID       string    `json:"site_id"`
	APILabel     string    `json:"api_label"`
	DeviceType   string    `json:"device_type"`
	DeviceCount  int       `json:"device_count"`
	Operation    string    `json:"operation"`
	DeviceStates any       `json:"device_states"`
}

// APStateBackup holds AP configs keyed by MAC
type APStateBackup struct {
	APs map[string]*vendors.APConfig `json:"aps"`
}

// SwitchStateBackup holds switch configs keyed by MAC
type SwitchStateBackup struct {
	Switches map[string]*vendors.SwitchConfig `json:"switches"`
}

// GatewayStateBackup holds gateway configs keyed by MAC
type GatewayStateBackup struct {
	Gateways map[string]*vendors.GatewayConfig `json:"gateways"`
}

// createAPIStateBackup captures the current API state for the devices being modified.
// This should be called BEFORE making any changes to preserve the running config.
func createAPIStateBackup(cfg *config.Config, siteName, siteID, deviceType, apiLabel string, devicesAffected []string) error {
	if len(devicesAffected) == 0 {
		logging.Debugf("No devices to backup - skipping API state backup")
		return nil
	}

	accessor := vendors.GetGlobalCacheAccessor()
	if accessor == nil {
		logging.Warnf("Cache accessor not available - skipping API state backup")
		return nil
	}

	// Get the backup directory
	backupDir := xdg.GetBackupsDir()
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Create device state backup based on type
	var deviceStates any
	var err error

	switch deviceType {
	case "ap":
		deviceStates, err = captureAPStates(accessor, devicesAffected)
	case "switch":
		deviceStates, err = captureSwitchStates(accessor, devicesAffected)
	case "gateway":
		deviceStates, err = captureGatewayStates(accessor, devicesAffected)
	default:
		return fmt.Errorf("unknown device type: %s", deviceType)
	}

	if err != nil {
		return fmt.Errorf("failed to capture device states: %w", err)
	}

	// Create the backup structure
	backup := APIStateBackup{
		Version:      1,
		Timestamp:    time.Now().UTC(),
		SiteName:     siteName,
		SiteID:       siteID,
		APILabel:     apiLabel,
		DeviceType:   deviceType,
		DeviceCount:  len(devicesAffected),
		Operation:    "pre_apply",
		DeviceStates: deviceStates,
	}

	// Get max backups from config
	maxBackups := 10
	if cfg.Files.ConfigBackups > 0 {
		maxBackups = cfg.Files.ConfigBackups
	}

	// Create backup filename: sitename-api-state-devicetype.json
	baseFileName := fmt.Sprintf("%s-api-state-%s.json", siteName, deviceType)

	// Rotate existing backups
	if err := rotateConfigFileBackups(backupDir, baseFileName, maxBackups); err != nil {
		logging.Warnf("Failed to rotate API state backups: %v", err)
	}

	// Write backup with serial 0 (most recent)
	backupPath := filepath.Join(backupDir, fmt.Sprintf("%s.0", baseFileName))

	data, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal API state backup: %w", err)
	}

	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write API state backup: %w", err)
	}

	logging.Infof("API state backup saved: %s (%d devices)", filepath.Base(backupPath), len(devicesAffected))
	return nil
}

// captureAPStates retrieves current AP configs from cache for the specified MACs
func captureAPStates(accessor *vendors.CacheAccessor, macs []string) (*APStateBackup, error) {
	states := &APStateBackup{
		APs: make(map[string]*vendors.APConfig),
	}

	for _, mac := range macs {
		apCfg, err := accessor.GetAPConfigByMAC(mac)
		if err != nil {
			// Device might not have a config yet (new assignment)
			logging.Debugf("Could not get AP config for %s: %v", mac, err)
			continue
		}
		states.APs[mac] = apCfg
	}

	return states, nil
}

// captureSwitchStates retrieves current switch configs from cache for the specified MACs
func captureSwitchStates(accessor *vendors.CacheAccessor, macs []string) (*SwitchStateBackup, error) {
	states := &SwitchStateBackup{
		Switches: make(map[string]*vendors.SwitchConfig),
	}

	for _, mac := range macs {
		swCfg, err := accessor.GetSwitchConfigByMAC(mac)
		if err != nil {
			logging.Debugf("Could not get switch config for %s: %v", mac, err)
			continue
		}
		states.Switches[mac] = swCfg
	}

	return states, nil
}

// captureGatewayStates retrieves current gateway configs from cache for the specified MACs
func captureGatewayStates(accessor *vendors.CacheAccessor, macs []string) (*GatewayStateBackup, error) {
	states := &GatewayStateBackup{
		Gateways: make(map[string]*vendors.GatewayConfig),
	}

	for _, mac := range macs {
		gwCfg, err := accessor.GetGatewayConfigByMAC(mac)
		if err != nil {
			logging.Debugf("Could not get gateway config for %s: %v", mac, err)
			continue
		}
		states.Gateways[mac] = gwCfg
	}

	return states, nil
}

// listAPIStateBackups lists available API state backups for a site
func listAPIStateBackups(siteName, deviceType string) ([]string, error) {
	backupDir := xdg.GetBackupsDir()

	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return nil, nil
	}

	pattern := fmt.Sprintf("%s-api-state-%s.json.*", siteName, deviceType)
	matches, err := filepath.Glob(filepath.Join(backupDir, pattern))
	if err != nil {
		return nil, fmt.Errorf("failed to list API state backups: %w", err)
	}

	return matches, nil
}

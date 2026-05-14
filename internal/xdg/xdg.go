// Package xdg provides XDG Base Directory Specification support for wifimgr.
// See https://specifications.freedesktop.org/basedir-spec/latest/
package xdg

import (
	"os"
	"path/filepath"
)

const appName = "wifimgr"

// GetConfigDir returns the configuration directory for wifimgr.
// Respects $XDG_CONFIG_HOME, defaults to ~/.config/wifimgr
func GetConfigDir() string {
	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
		return filepath.Join(xdgConfigHome, appName)
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", "config")
	}
	return filepath.Join(homeDir, ".config", appName)
}

// GetCacheDir returns the cache directory for wifimgr.
// Respects $XDG_CACHE_HOME, defaults to ~/.cache/wifimgr
func GetCacheDir() string {
	if xdgCacheHome := os.Getenv("XDG_CACHE_HOME"); xdgCacheHome != "" {
		return filepath.Join(xdgCacheHome, appName)
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", "cache")
	}
	return filepath.Join(homeDir, ".cache", appName)
}

// GetStateDir returns the state directory for wifimgr.
// Respects $XDG_STATE_HOME, defaults to ~/.local/state/wifimgr
// Used for logs and backups.
func GetStateDir() string {
	if xdgStateHome := os.Getenv("XDG_STATE_HOME"); xdgStateHome != "" {
		return filepath.Join(xdgStateHome, appName)
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", "state")
	}
	return filepath.Join(homeDir, ".local", "state", appName)
}

// GetDataDir returns the data directory for wifimgr.
// Respects $XDG_DATA_HOME, defaults to ~/.local/share/wifimgr
// Used for schemas and other read-only data.
func GetDataDir() string {
	if xdgDataHome := os.Getenv("XDG_DATA_HOME"); xdgDataHome != "" {
		return filepath.Join(xdgDataHome, appName)
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", "data")
	}
	return filepath.Join(homeDir, ".local", "share", appName)
}

// GetConfigFile returns the path to the main configuration file.
func GetConfigFile() string {
	return filepath.Join(GetConfigDir(), "wifimgr-config.json")
}

// GetCacheFile returns the path to the main cache file.
func GetCacheFile() string {
	return filepath.Join(GetCacheDir(), "cache.json")
}

// GetLogFile returns the path to the log file.
func GetLogFile() string {
	return filepath.Join(GetStateDir(), "wifimgr.log")
}

// GetBackupsDir returns the path to the backups directory.
func GetBackupsDir() string {
	return filepath.Join(GetStateDir(), "backups")
}

// GetSchemasDir returns the path to the schemas directory.
func GetSchemasDir() string {
	return filepath.Join(GetDataDir(), "schemas")
}

// GetInventoryFile returns the path to the inventory file.
func GetInventoryFile() string {
	return filepath.Join(GetConfigDir(), "inventory.json")
}

// EnsureDir creates a directory and all parent directories if they don't exist.
// Returns nil if the directory already exists or was successfully created.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755) // #nosec G301 -- XDG base directories use conventional 0755 permissions
}

// FindEnvFile searches for .env.wifimgr in multiple locations.
// Returns the path to the first found file, or empty string if not found.
// Search order:
//  1. current directory (project-specific override)
//  2. $HOME/.env.wifimgr (conventional home-dir dotenv location)
//  3. XDG config directory ($XDG_CONFIG_HOME/wifimgr/.env.wifimgr)
//
// Home-directory placement is the most common convention for dotenv files
// (ssh, gitconfig, dotenv libraries, etc.) so it has to be on the search
// path or users hit a "file present, tool can't find it" footgun.
func FindEnvFile() string {
	const envFilename = ".env.wifimgr"

	// 1. Current directory (project-specific env).
	if _, err := os.Stat(envFilename); err == nil {
		return envFilename
	}

	// 2. User home directory.
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		homeEnv := filepath.Join(home, envFilename)
		if _, err := os.Stat(homeEnv); err == nil {
			return homeEnv
		}
	}

	// 3. XDG config directory.
	xdgEnv := filepath.Join(GetConfigDir(), envFilename)
	if _, err := os.Stat(xdgEnv); err == nil {
		return xdgEnv
	}

	return ""
}

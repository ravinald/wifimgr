package xdg

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetConfigDir(t *testing.T) {
	// Save original environment
	origXDGConfigHome := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", origXDGConfigHome)

	t.Run("with XDG_CONFIG_HOME set", func(t *testing.T) {
		os.Setenv("XDG_CONFIG_HOME", "/tmp/test-config")
		result := GetConfigDir()
		expected := "/tmp/test-config/wifimgr"
		if result != expected {
			t.Errorf("GetConfigDir() = %q, want %q", result, expected)
		}
	})

	t.Run("without XDG_CONFIG_HOME", func(t *testing.T) {
		os.Unsetenv("XDG_CONFIG_HOME")
		result := GetConfigDir()
		homeDir, _ := os.UserHomeDir()
		expected := filepath.Join(homeDir, ".config", "wifimgr")
		if result != expected {
			t.Errorf("GetConfigDir() = %q, want %q", result, expected)
		}
	})
}

func TestGetCacheDir(t *testing.T) {
	// Save original environment
	origXDGCacheHome := os.Getenv("XDG_CACHE_HOME")
	defer os.Setenv("XDG_CACHE_HOME", origXDGCacheHome)

	t.Run("with XDG_CACHE_HOME set", func(t *testing.T) {
		os.Setenv("XDG_CACHE_HOME", "/tmp/test-cache")
		result := GetCacheDir()
		expected := "/tmp/test-cache/wifimgr"
		if result != expected {
			t.Errorf("GetCacheDir() = %q, want %q", result, expected)
		}
	})

	t.Run("without XDG_CACHE_HOME", func(t *testing.T) {
		os.Unsetenv("XDG_CACHE_HOME")
		result := GetCacheDir()
		homeDir, _ := os.UserHomeDir()
		expected := filepath.Join(homeDir, ".cache", "wifimgr")
		if result != expected {
			t.Errorf("GetCacheDir() = %q, want %q", result, expected)
		}
	})
}

func TestGetStateDir(t *testing.T) {
	// Save original environment
	origXDGStateHome := os.Getenv("XDG_STATE_HOME")
	defer os.Setenv("XDG_STATE_HOME", origXDGStateHome)

	t.Run("with XDG_STATE_HOME set", func(t *testing.T) {
		os.Setenv("XDG_STATE_HOME", "/tmp/test-state")
		result := GetStateDir()
		expected := "/tmp/test-state/wifimgr"
		if result != expected {
			t.Errorf("GetStateDir() = %q, want %q", result, expected)
		}
	})

	t.Run("without XDG_STATE_HOME", func(t *testing.T) {
		os.Unsetenv("XDG_STATE_HOME")
		result := GetStateDir()
		homeDir, _ := os.UserHomeDir()
		expected := filepath.Join(homeDir, ".local", "state", "wifimgr")
		if result != expected {
			t.Errorf("GetStateDir() = %q, want %q", result, expected)
		}
	})
}

func TestGetDataDir(t *testing.T) {
	// Save original environment
	origXDGDataHome := os.Getenv("XDG_DATA_HOME")
	defer os.Setenv("XDG_DATA_HOME", origXDGDataHome)

	t.Run("with XDG_DATA_HOME set", func(t *testing.T) {
		os.Setenv("XDG_DATA_HOME", "/tmp/test-data")
		result := GetDataDir()
		expected := "/tmp/test-data/wifimgr"
		if result != expected {
			t.Errorf("GetDataDir() = %q, want %q", result, expected)
		}
	})

	t.Run("without XDG_DATA_HOME", func(t *testing.T) {
		os.Unsetenv("XDG_DATA_HOME")
		result := GetDataDir()
		homeDir, _ := os.UserHomeDir()
		expected := filepath.Join(homeDir, ".local", "share", "wifimgr")
		if result != expected {
			t.Errorf("GetDataDir() = %q, want %q", result, expected)
		}
	})
}

func TestGetConfigFile(t *testing.T) {
	// Save original environment
	origXDGConfigHome := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", origXDGConfigHome)

	os.Setenv("XDG_CONFIG_HOME", "/tmp/test-config")
	result := GetConfigFile()
	expected := "/tmp/test-config/wifimgr/wifimgr-config.json"
	if result != expected {
		t.Errorf("GetConfigFile() = %q, want %q", result, expected)
	}
}

func TestGetCacheFile(t *testing.T) {
	// Save original environment
	origXDGCacheHome := os.Getenv("XDG_CACHE_HOME")
	defer os.Setenv("XDG_CACHE_HOME", origXDGCacheHome)

	os.Setenv("XDG_CACHE_HOME", "/tmp/test-cache")
	result := GetCacheFile()
	expected := "/tmp/test-cache/wifimgr/cache.json"
	if result != expected {
		t.Errorf("GetCacheFile() = %q, want %q", result, expected)
	}
}

func TestGetLogFile(t *testing.T) {
	// Save original environment
	origXDGStateHome := os.Getenv("XDG_STATE_HOME")
	defer os.Setenv("XDG_STATE_HOME", origXDGStateHome)

	os.Setenv("XDG_STATE_HOME", "/tmp/test-state")
	result := GetLogFile()
	expected := "/tmp/test-state/wifimgr/wifimgr.log"
	if result != expected {
		t.Errorf("GetLogFile() = %q, want %q", result, expected)
	}
}

func TestGetBackupsDir(t *testing.T) {
	// Save original environment
	origXDGStateHome := os.Getenv("XDG_STATE_HOME")
	defer os.Setenv("XDG_STATE_HOME", origXDGStateHome)

	os.Setenv("XDG_STATE_HOME", "/tmp/test-state")
	result := GetBackupsDir()
	expected := "/tmp/test-state/wifimgr/backups"
	if result != expected {
		t.Errorf("GetBackupsDir() = %q, want %q", result, expected)
	}
}

func TestGetSchemasDir(t *testing.T) {
	// Save original environment
	origXDGDataHome := os.Getenv("XDG_DATA_HOME")
	defer os.Setenv("XDG_DATA_HOME", origXDGDataHome)

	os.Setenv("XDG_DATA_HOME", "/tmp/test-data")
	result := GetSchemasDir()
	expected := "/tmp/test-data/wifimgr/schemas"
	if result != expected {
		t.Errorf("GetSchemasDir() = %q, want %q", result, expected)
	}
}

func TestGetInventoryFile(t *testing.T) {
	// Save original environment
	origXDGConfigHome := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", origXDGConfigHome)

	os.Setenv("XDG_CONFIG_HOME", "/tmp/test-config")
	result := GetInventoryFile()
	expected := "/tmp/test-config/wifimgr/inventory.json"
	if result != expected {
		t.Errorf("GetInventoryFile() = %q, want %q", result, expected)
	}
}

func TestEnsureDir(t *testing.T) {
	// Create a temp dir for testing
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "test", "nested", "dir")

	// Ensure the directory doesn't exist yet
	if _, err := os.Stat(testDir); !os.IsNotExist(err) {
		t.Fatal("Test directory should not exist before test")
	}

	// Create the directory
	err := EnsureDir(testDir)
	if err != nil {
		t.Errorf("EnsureDir() error = %v", err)
	}

	// Verify it was created
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Error("EnsureDir() did not create directory")
	}

	// Calling again should not error
	err = EnsureDir(testDir)
	if err != nil {
		t.Errorf("EnsureDir() on existing dir error = %v", err)
	}
}

func TestFindEnvFile(t *testing.T) {
	// Save original environment
	origXDGConfigHome := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", origXDGConfigHome)

	// Create a temp directory structure
	tmpDir := t.TempDir()
	xdgConfigDir := filepath.Join(tmpDir, "config", "wifimgr")
	os.MkdirAll(xdgConfigDir, 0755)
	os.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "config"))

	t.Run("no env file found", func(t *testing.T) {
		result := FindEnvFile()
		if result != "" {
			t.Errorf("FindEnvFile() = %q, want empty string", result)
		}
	})

	t.Run("env file in XDG config dir", func(t *testing.T) {
		envPath := filepath.Join(xdgConfigDir, ".env.wifimgr")
		os.WriteFile(envPath, []byte("TEST=value"), 0644)
		defer os.Remove(envPath)

		result := FindEnvFile()
		if result != envPath {
			t.Errorf("FindEnvFile() = %q, want %q", result, envPath)
		}
	})

	t.Run("env file in current dir takes precedence", func(t *testing.T) {
		// Create env file in XDG dir
		xdgEnvPath := filepath.Join(xdgConfigDir, ".env.wifimgr")
		os.WriteFile(xdgEnvPath, []byte("XDG=value"), 0644)
		defer os.Remove(xdgEnvPath)

		// Create env file in current dir
		localEnvPath := ".env.wifimgr"
		os.WriteFile(localEnvPath, []byte("LOCAL=value"), 0644)
		defer os.Remove(localEnvPath)

		result := FindEnvFile()
		if result != localEnvPath {
			t.Errorf("FindEnvFile() = %q, want %q (local should take precedence)", result, localEnvPath)
		}
	})
}

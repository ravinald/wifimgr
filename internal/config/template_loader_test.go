package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewTemplateStore(t *testing.T) {
	store := NewTemplateStore()

	if store.Radio == nil {
		t.Error("Expected Radio map to be initialized")
	}
	if store.WLAN == nil {
		t.Error("Expected WLAN map to be initialized")
	}
	if store.Device == nil {
		t.Error("Expected Device map to be initialized")
	}
	if !store.IsEmpty() {
		t.Error("Expected new store to be empty")
	}
}

func TestLoadTemplates_EmptyPaths(t *testing.T) {
	store, err := LoadTemplates(nil, "")
	if err != nil {
		t.Errorf("Expected no error for empty paths, got: %v", err)
	}
	if !store.IsEmpty() {
		t.Error("Expected store to be empty for empty paths")
	}
}

func TestLoadTemplates_ValidFile(t *testing.T) {
	// Create a temporary template file
	tempDir := t.TempDir()
	templateFile := filepath.Join(tempDir, "templates.json")

	templateContent := `{
  "version": 1,
  "templates": {
    "radio": {
      "high-density": {
        "radio_config": {
          "band_5": {"power": 15, "bandwidth": 40}
        }
      }
    },
    "wlan": {
      "corp-secure": {
        "ssid": "CorpNet",
        "auth": {"type": "wpa2-enterprise"}
      }
    },
    "device": {
      "standard-ap": {
        "led": {"enabled": true}
      }
    }
  }
}`

	if err := os.WriteFile(templateFile, []byte(templateContent), 0644); err != nil {
		t.Fatalf("Failed to write template file: %v", err)
	}

	store, err := LoadTemplates([]string{templateFile}, "")
	if err != nil {
		t.Fatalf("Failed to load templates: %v", err)
	}

	// Verify radio template
	radio, found := store.GetRadioTemplate("high-density")
	if !found {
		t.Error("Expected to find radio template 'high-density'")
	}
	if radioConfig, ok := radio["radio_config"].(map[string]any); ok {
		if band5, ok := radioConfig["band_5"].(map[string]any); ok {
			if power, ok := band5["power"].(float64); !ok || power != 15 {
				t.Errorf("Expected power=15, got %v", power)
			}
		} else {
			t.Error("Expected band_5 config")
		}
	} else {
		t.Error("Expected radio_config in template")
	}

	// Verify WLAN template
	wlan, found := store.GetWLANTemplate("corp-secure")
	if !found {
		t.Error("Expected to find WLAN template 'corp-secure'")
	}
	if ssid, ok := wlan["ssid"].(string); !ok || ssid != "CorpNet" {
		t.Errorf("Expected ssid=CorpNet, got %v", ssid)
	}

	// Verify device template
	device, found := store.GetDeviceTemplate("standard-ap")
	if !found {
		t.Error("Expected to find device template 'standard-ap'")
	}
	if led, ok := device["led"].(map[string]any); ok {
		if enabled, ok := led["enabled"].(bool); !ok || !enabled {
			t.Errorf("Expected led.enabled=true, got %v", enabled)
		}
	}
}

func TestLoadTemplates_RelativePath(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()
	templateDir := filepath.Join(tempDir, "templates")
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		t.Fatalf("Failed to create template dir: %v", err)
	}

	templateContent := `{
  "version": 1,
  "templates": {
    "radio": {
      "test-profile": {"power": 10}
    }
  }
}`

	templateFile := filepath.Join(templateDir, "test.json")
	if err := os.WriteFile(templateFile, []byte(templateContent), 0644); err != nil {
		t.Fatalf("Failed to write template file: %v", err)
	}

	// Load with relative path
	store, err := LoadTemplates([]string{"templates/test.json"}, tempDir)
	if err != nil {
		t.Fatalf("Failed to load templates with relative path: %v", err)
	}

	if _, found := store.GetRadioTemplate("test-profile"); !found {
		t.Error("Expected to find radio template 'test-profile'")
	}
}

func TestLoadTemplates_MissingFile(t *testing.T) {
	_, err := LoadTemplates([]string{"/nonexistent/path.json"}, "")
	if err == nil {
		t.Error("Expected error for missing file")
	}
}

func TestLoadTemplates_InvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	invalidFile := filepath.Join(tempDir, "invalid.json")

	if err := os.WriteFile(invalidFile, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("Failed to write invalid file: %v", err)
	}

	_, err := LoadTemplates([]string{invalidFile}, "")
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestLoadTemplates_MultipleFiles(t *testing.T) {
	tempDir := t.TempDir()

	// First file with radio templates
	file1 := filepath.Join(tempDir, "radio.json")
	content1 := `{
  "version": 1,
  "templates": {
    "radio": {
      "profile-a": {"power": 10}
    }
  }
}`
	if err := os.WriteFile(file1, []byte(content1), 0644); err != nil {
		t.Fatalf("Failed to write file1: %v", err)
	}

	// Second file with WLAN templates
	file2 := filepath.Join(tempDir, "wlan.json")
	content2 := `{
  "version": 1,
  "templates": {
    "wlan": {
      "wlan-a": {"ssid": "TestSSID"}
    }
  }
}`
	if err := os.WriteFile(file2, []byte(content2), 0644); err != nil {
		t.Fatalf("Failed to write file2: %v", err)
	}

	store, err := LoadTemplates([]string{file1, file2}, "")
	if err != nil {
		t.Fatalf("Failed to load multiple templates: %v", err)
	}

	if _, found := store.GetRadioTemplate("profile-a"); !found {
		t.Error("Expected to find radio template 'profile-a' from first file")
	}
	if _, found := store.GetWLANTemplate("wlan-a"); !found {
		t.Error("Expected to find WLAN template 'wlan-a' from second file")
	}
}

func TestTemplateStore_ListTemplates(t *testing.T) {
	store := NewTemplateStore()
	store.Radio["radio1"] = map[string]any{}
	store.Radio["radio2"] = map[string]any{}
	store.WLAN["wlan1"] = map[string]any{}
	store.Device["device1"] = map[string]any{}

	list := store.ListTemplates()

	if len(list["radio"]) != 2 {
		t.Errorf("Expected 2 radio templates, got %d", len(list["radio"]))
	}
	if len(list["wlan"]) != 1 {
		t.Errorf("Expected 1 wlan template, got %d", len(list["wlan"]))
	}
	if len(list["device"]) != 1 {
		t.Errorf("Expected 1 device template, got %d", len(list["device"]))
	}
}

func TestTemplateStore_IsEmpty(t *testing.T) {
	store := NewTemplateStore()

	if !store.IsEmpty() {
		t.Error("Expected new store to be empty")
	}

	store.Radio["test"] = map[string]any{}
	if store.IsEmpty() {
		t.Error("Expected store with radio template to not be empty")
	}

	store = NewTemplateStore()
	store.WLAN["test"] = map[string]any{}
	if store.IsEmpty() {
		t.Error("Expected store with WLAN template to not be empty")
	}

	store = NewTemplateStore()
	store.Device["test"] = map[string]any{}
	if store.IsEmpty() {
		t.Error("Expected store with device template to not be empty")
	}
}

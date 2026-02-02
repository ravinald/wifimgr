package netbox

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"

	"github.com/ravinald/wifimgr/internal/encryption"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// Config represents NetBox integration configuration
type Config struct {
	URL            string        `json:"url"`
	APIKey         string        `json:"api_key"`
	SSLVerify      bool          `json:"ssl_verify"`
	KeyEncrypted   bool          `json:"key_encrypted,omitempty"`
	SettingsSource string        `json:"settings_source,omitempty"` // "api" (default) or "netbox"
	Mappings       MappingConfig `json:"mappings"`

	// Runtime only - not persisted
	decryptedKey string
}

// DeviceTypeMapping defines mapping for a specific device model
type DeviceTypeMapping struct {
	Slug string `json:"slug"`           // NetBox device type slug (required)
	Role string `json:"role,omitempty"` // Override role for this specific device type (optional)
}

// InterfaceMapping defines how an internal interface maps to NetBox
type InterfaceMapping struct {
	Name string `json:"name"` // NetBox interface name (e.g., "eth0", "wlan0", "wifi1")
	Type string `json:"type"` // NetBox PHY type (e.g., "1000base-t", "ieee802.11ax")
}

// InterfaceMappings configures interface name and type mappings
type InterfaceMappings struct {
	Eth0   *InterfaceMapping `json:"eth0,omitempty"`   // Primary Ethernet management interface
	Eth1   *InterfaceMapping `json:"eth1,omitempty"`   // Secondary Ethernet interface
	Radio0 *InterfaceMapping `json:"radio0,omitempty"` // 2.4 GHz radio
	Radio1 *InterfaceMapping `json:"radio1,omitempty"` // 5 GHz radio
	Radio2 *InterfaceMapping `json:"radio2,omitempty"` // 6 GHz radio
}

// MappingConfig defines how wifimgr types map to NetBox objects
type MappingConfig struct {
	Tag           string                       `json:"tag,omitempty"`            // NetBox tag to apply to all objects created by wifimgr
	DefaultRoles  map[string]string            `json:"default_roles,omitempty"`  // wifimgr type -> netbox role slug (fallback)
	DeviceTypes   map[string]DeviceTypeMapping `json:"device_types,omitempty"`   // model pattern -> device type mapping
	SiteOverrides map[string]string            `json:"site_overrides,omitempty"` // wifimgr site -> netbox site slug
	Interfaces    InterfaceMappings            `json:"interfaces,omitempty"`     // Interface name and type mappings
}

// DefaultInterfaceMappings returns sensible default interface mappings
func DefaultInterfaceMappings() InterfaceMappings {
	return InterfaceMappings{
		Eth0:   &InterfaceMapping{Name: "eth0", Type: "1000base-t"},
		Eth1:   &InterfaceMapping{Name: "eth1", Type: "1000base-t"},
		Radio0: &InterfaceMapping{Name: "wifi0", Type: "ieee802.11n"},
		Radio1: &InterfaceMapping{Name: "wifi1", Type: "ieee802.11ac"},
		Radio2: &InterfaceMapping{Name: "wifi2", Type: "ieee802.11ax"},
	}
}

// DefaultMappings returns sensible default mappings
func DefaultMappings() MappingConfig {
	return MappingConfig{
		DefaultRoles: map[string]string{
			"ap":      "wireless-ap",
			"switch":  "access-switch",
			"gateway": "router",
		},
		DeviceTypes:   make(map[string]DeviceTypeMapping),
		SiteOverrides: make(map[string]string),
		Interfaces:    DefaultInterfaceMappings(),
	}
}

// LoadConfig loads NetBox configuration from multiple sources.
// Priority (highest to lowest):
// 1. Environment variables (NETBOX_API_KEY, NETBOX_API_URL, NETBOX_SSL_VERIFY)
// 2. Env file (~/.env.netbox)
// 3. Config file (netbox.* in wifimgr config)
func LoadConfig() (*Config, error) {
	cfg := &Config{
		SSLVerify:      true,  // Default to verify SSL
		SettingsSource: "api", // Default to API as settings source
		Mappings:       DefaultMappings(),
	}

	// Load from config file first (lowest priority)
	loadFromViper(cfg)

	// Load from env file (medium priority)
	if err := loadFromEnvFile(cfg); err != nil {
		logging.Debugf("No ~/.env.netbox file found: %v", err)
	}

	// Load from environment variables (highest priority)
	loadFromEnv(cfg)

	// Validate required fields
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// Handle encrypted key
	if cfg.KeyEncrypted && cfg.APIKey != "" {
		decrypted, err := decryptAPIKey(cfg.APIKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt NetBox API key: %w", err)
		}
		cfg.decryptedKey = decrypted
	} else {
		cfg.decryptedKey = cfg.APIKey
	}

	return cfg, nil
}

// GetAPIKey returns the decrypted API key for use in requests
func (c *Config) GetAPIKey() string {
	if c.decryptedKey != "" {
		return c.decryptedKey
	}
	return c.APIKey
}

// Validate checks that required configuration is present
func (c *Config) Validate() error {
	if c.URL == "" {
		return fmt.Errorf("NetBox URL is required (set NETBOX_API_URL or netbox.url in config)")
	}
	if c.APIKey == "" {
		return fmt.Errorf("NetBox API key is required (set NETBOX_API_KEY or netbox.credentials.api_key in config)")
	}

	// Validate settings_source
	if c.SettingsSource != "" && c.SettingsSource != "api" && c.SettingsSource != "netbox" {
		return fmt.Errorf("invalid settings_source '%s': must be 'api' or 'netbox'", c.SettingsSource)
	}

	// Default to "api" if not set
	if c.SettingsSource == "" {
		c.SettingsSource = "api"
	}

	return nil
}

// loadFromViper loads configuration from Viper (config file)
func loadFromViper(cfg *Config) {
	if url := viper.GetString("netbox.url"); url != "" {
		cfg.URL = url
	}
	if apiKey := viper.GetString("netbox.credentials.api_key"); apiKey != "" {
		cfg.APIKey = apiKey
		cfg.KeyEncrypted = viper.GetBool("netbox.credentials.key_encrypted")
	}
	if viper.IsSet("netbox.ssl_verify") {
		cfg.SSLVerify = viper.GetBool("netbox.ssl_verify")
	}
	if settingsSource := viper.GetString("netbox.settings_source"); settingsSource != "" {
		cfg.SettingsSource = settingsSource
	}

	// Load mappings with backward compatibility
	loadMappings(cfg)
}

// loadMappings loads mapping configuration with backward compatibility for old format
func loadMappings(cfg *Config) {
	// Load tag if configured
	if tag := viper.GetString("netbox.mappings.tag"); tag != "" {
		cfg.Mappings.Tag = tag
	}

	// Try loading new format for default_roles
	if roles := viper.GetStringMapString("netbox.mappings.default_roles"); len(roles) > 0 {
		cfg.Mappings.DefaultRoles = roles
	} else if roles := viper.GetStringMapString("netbox.mappings.device_roles"); len(roles) > 0 {
		// Backward compatibility: device_roles -> default_roles
		logging.Warnf("netbox.mappings.device_roles is deprecated, use netbox.mappings.default_roles instead")
		cfg.Mappings.DefaultRoles = roles
	}

	// Load device_types - check if it's new format (map of objects) or old format (map of strings)
	if viper.IsSet("netbox.mappings.device_types") {
		deviceTypesRaw := viper.Get("netbox.mappings.device_types")

		// Try to unmarshal as new format first
		if deviceTypesMap, ok := deviceTypesRaw.(map[string]any); ok {
			for model, value := range deviceTypesMap {
				// Check if value is a map (new format) or string (old format)
				if valueMap, isMap := value.(map[string]any); isMap {
					// New format: { "slug": "...", "role": "..." }
					mapping := DeviceTypeMapping{}
					if slug, ok := valueMap["slug"].(string); ok {
						mapping.Slug = slug
					}
					if role, ok := valueMap["role"].(string); ok {
						mapping.Role = role
					}
					if mapping.Slug != "" {
						cfg.Mappings.DeviceTypes[model] = mapping
					}
				} else if slug, isString := value.(string); isString {
					// Old format: simple string value
					logging.Warnf("netbox.mappings.device_types[%s] uses deprecated format, migrate to {\"slug\": \"%s\"}", model, slug)
					cfg.Mappings.DeviceTypes[model] = DeviceTypeMapping{Slug: slug}
				}
			}
		}
	}

	// Load site_overrides (unchanged)
	if sites := viper.GetStringMapString("netbox.mappings.site_overrides"); len(sites) > 0 {
		cfg.Mappings.SiteOverrides = sites
	}

	// Load interface mappings
	loadInterfaceMappings(cfg)
}

// loadInterfaceMappings loads interface name and type mappings from config
func loadInterfaceMappings(cfg *Config) {
	if !viper.IsSet("netbox.mappings.interfaces") {
		return
	}

	interfacesRaw := viper.Get("netbox.mappings.interfaces")
	interfacesMap, ok := interfacesRaw.(map[string]any)
	if !ok {
		return
	}

	// Helper to parse a single interface mapping
	parseMapping := func(key string) *InterfaceMapping {
		value, exists := interfacesMap[key]
		if !exists {
			return nil
		}
		valueMap, ok := value.(map[string]any)
		if !ok {
			return nil
		}
		mapping := &InterfaceMapping{}
		if name, ok := valueMap["name"].(string); ok {
			mapping.Name = name
		}
		if ifType, ok := valueMap["type"].(string); ok {
			mapping.Type = ifType
		}
		if mapping.Name == "" && mapping.Type == "" {
			return nil
		}
		return mapping
	}

	if m := parseMapping("eth0"); m != nil {
		cfg.Mappings.Interfaces.Eth0 = m
	}
	if m := parseMapping("eth1"); m != nil {
		cfg.Mappings.Interfaces.Eth1 = m
	}
	if m := parseMapping("radio0"); m != nil {
		cfg.Mappings.Interfaces.Radio0 = m
	}
	if m := parseMapping("radio1"); m != nil {
		cfg.Mappings.Interfaces.Radio1 = m
	}
	if m := parseMapping("radio2"); m != nil {
		cfg.Mappings.Interfaces.Radio2 = m
	}
}

// loadFromEnvFile loads configuration from ~/.env.netbox
func loadFromEnvFile(cfg *Config) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	envFilePath := filepath.Join(homeDir, ".env.netbox")
	file, err := os.Open(envFilePath)
	if err != nil {
		return err // File doesn't exist, not an error
	}
	defer func() { _ = file.Close() }()

	logging.Debugf("Loading NetBox configuration from %s", envFilePath)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=value format
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove surrounding quotes if present
		if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			value = value[1 : len(value)-1]
		}

		switch key {
		case "NETBOX_API_KEY":
			cfg.APIKey = value
		case "NETBOX_API_URL":
			cfg.URL = value
		case "NETBOX_SSL_VERIFY":
			cfg.SSLVerify = strings.ToLower(value) == "true" || value == "1"
		}
	}

	return scanner.Err()
}

// loadFromEnv loads configuration from environment variables
func loadFromEnv(cfg *Config) {
	if apiKey := os.Getenv("NETBOX_API_KEY"); apiKey != "" {
		cfg.APIKey = apiKey
	}
	if url := os.Getenv("NETBOX_API_URL"); url != "" {
		cfg.URL = url
	}
	if sslVerify := os.Getenv("NETBOX_SSL_VERIFY"); sslVerify != "" {
		cfg.SSLVerify = strings.ToLower(sslVerify) == "true" || sslVerify == "1"
	}
}

// decryptAPIKey decrypts an encrypted API key
func decryptAPIKey(encryptedKey string) (string, error) {
	// Check if it's actually encrypted (has enc: prefix)
	if !encryption.IsEncrypted(encryptedKey) {
		return encryptedKey, nil
	}

	// Prompt for password and decrypt
	password, err := encryption.PromptForPassword("Enter password to decrypt NetBox API key: ")
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}

	decrypted, err := encryption.Decrypt(encryptedKey, password)
	if err != nil {
		return "", fmt.Errorf("decryption failed: %w", err)
	}

	return decrypted, nil
}

// GetDeviceRoleSlug returns the NetBox device role slug for a wifimgr device type.
// Priority: device_types[model].role → default_roles[deviceType] → hardcoded defaults
func (c *Config) GetDeviceRoleSlug(deviceType string) string {
	return c.GetDeviceRoleSlugForModel(deviceType, "", nil)
}

// GetDeviceRoleSlugForModel returns the NetBox device role slug with model-specific override support.
// Priority: deviceNetBox.DeviceRole → device_types[model].role → default_roles[deviceType] → hardcoded defaults
func (c *Config) GetDeviceRoleSlugForModel(deviceType string, model string, deviceNetBox any) string {
	// Priority 0: Check device-level NetBox extension for per-device role override
	if ext, ok := deviceNetBox.(*vendors.NetBoxDeviceExtension); ok && ext != nil && ext.DeviceRole != "" {
		return ext.DeviceRole
	}

	// Priority 1: Check if model has a specific role override
	if model != "" {
		modelLower := strings.ToLower(model)

		// Check exact match first (both cases)
		if mapping, ok := c.Mappings.DeviceTypes[modelLower]; ok {
			if mapping.Role != "" {
				return mapping.Role
			}
			// Exact match found but no role override - skip to default_roles
			// Don't check pattern matches since exact match takes precedence
			goto useDefaultRoles
		}
		if mapping, ok := c.Mappings.DeviceTypes[model]; ok {
			if mapping.Role != "" {
				return mapping.Role
			}
			// Exact match found but no role override - skip to default_roles
			goto useDefaultRoles
		}

		// No exact match, check pattern matches
		for pattern, mapping := range c.Mappings.DeviceTypes {
			if mapping.Role != "" {
				if prefix, found := strings.CutSuffix(pattern, "*"); found {
					if strings.HasPrefix(modelLower, strings.ToLower(prefix)) {
						return mapping.Role
					}
				}
			}
		}
	}

useDefaultRoles:
	// Priority 2: Check default_roles mapping
	if slug, ok := c.Mappings.DefaultRoles[deviceType]; ok {
		return slug
	}

	// Priority 3: Hardcoded defaults
	switch deviceType {
	case "ap":
		return "wireless-ap"
	case "switch":
		return "access-switch"
	case "gateway":
		return "router"
	default:
		return deviceType
	}
}

// GetDeviceTypeSlug returns the NetBox device type slug for a device model
func (c *Config) GetDeviceTypeSlug(model string) string {
	// Check exact match first (case-insensitive due to Viper lowercasing keys)
	modelLower := strings.ToLower(model)
	if mapping, ok := c.Mappings.DeviceTypes[modelLower]; ok {
		return mapping.Slug
	}
	// Also try original case for manually constructed configs
	if mapping, ok := c.Mappings.DeviceTypes[model]; ok {
		return mapping.Slug
	}

	// Check pattern matches (simple prefix matching, case-insensitive)
	for pattern, mapping := range c.Mappings.DeviceTypes {
		if prefix, found := strings.CutSuffix(pattern, "*"); found {
			if strings.HasPrefix(modelLower, strings.ToLower(prefix)) {
				return mapping.Slug
			}
		}
	}

	// Default: convert model to slug format (lowercase, replace spaces with dashes)
	return strings.ToLower(strings.ReplaceAll(model, " ", "-"))
}

// GetSiteSlug returns the NetBox site slug for a wifimgr site name
func (c *Config) GetSiteSlug(siteName string) string {
	// Check explicit override first
	if slug, ok := c.Mappings.SiteOverrides[siteName]; ok {
		return slug
	}

	// Default: convert site name to slug format
	slug := strings.ToLower(siteName)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")
	return slug
}

// GetInterfaceMapping returns the interface mapping for an internal interface ID.
// Returns nil if the interface is not configured.
// internalID is one of: "eth0", "eth1", "radio0", "radio1", "radio2"
func (c *Config) GetInterfaceMapping(internalID string) *InterfaceMapping {
	defaults := DefaultInterfaceMappings()

	switch internalID {
	case "eth0":
		if c.Mappings.Interfaces.Eth0 != nil {
			return c.Mappings.Interfaces.Eth0
		}
		return defaults.Eth0
	case "eth1":
		if c.Mappings.Interfaces.Eth1 != nil {
			return c.Mappings.Interfaces.Eth1
		}
		return defaults.Eth1
	case "radio0":
		if c.Mappings.Interfaces.Radio0 != nil {
			return c.Mappings.Interfaces.Radio0
		}
		return defaults.Radio0
	case "radio1":
		if c.Mappings.Interfaces.Radio1 != nil {
			return c.Mappings.Interfaces.Radio1
		}
		return defaults.Radio1
	case "radio2":
		if c.Mappings.Interfaces.Radio2 != nil {
			return c.Mappings.Interfaces.Radio2
		}
		return defaults.Radio2
	default:
		return nil
	}
}

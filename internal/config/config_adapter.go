package config

// ConfigAdapter adapts the Config struct to match the encryption.Config interface
type ConfigAdapter struct {
	Config     *Config
	ConfigPath string
}

// GetAPIToken returns the API token
func (ca *ConfigAdapter) GetAPIToken() string {
	return ca.Config.API.Credentials.APIToken
}

// SetAPIToken sets the API token
func (ca *ConfigAdapter) SetAPIToken(token string) {
	ca.Config.API.Credentials.APIToken = token
}

// SetKeyEncrypted sets whether the API key is encrypted
func (ca *ConfigAdapter) SetKeyEncrypted(encrypted bool) {
	ca.Config.API.Credentials.KeyEncrypted = encrypted
}

// GetAPIURL returns the API URL
func (ca *ConfigAdapter) GetAPIURL() string {
	return ca.Config.API.URL
}

// GetOrgID returns the organization ID
func (ca *ConfigAdapter) GetOrgID() string {
	return ca.Config.API.Credentials.OrgID
}

// SetOrgID sets the organization ID
func (ca *ConfigAdapter) SetOrgID(id string) {
	ca.Config.API.Credentials.OrgID = id
}

// Save saves the config to disk
func (ca *ConfigAdapter) Save(configPath string) error {
	path := ca.ConfigPath
	if configPath != "" {
		path = configPath
	}
	return SaveConfig(ca.Config, path)
}

// SetCurrentToken sets the decrypted token for current session (not stored in config)
func (ca *ConfigAdapter) SetCurrentToken(token string) {
	ca.Config.API.currentToken = token
}

// GetCurrentToken returns the decrypted token for current session
func (ca *ConfigAdapter) GetCurrentToken() string {
	return ca.Config.API.currentToken
}

// ReadTokenFromEnvFile reads token from .env.wifimgr file
func (ca *ConfigAdapter) ReadTokenFromEnvFile() (string, error) {
	return ca.ReadTokenFromEnvFileWithName(".env.wifimgr")
}

// ReadTokenFromEnvFileWithName reads token from specified env file (for testing)
// Note: In multi-vendor mode, tokens are loaded via InitializeMultiVendor() before this is called.
// This function is kept for interface compatibility but returns empty in normal operation.
func (ca *ConfigAdapter) ReadTokenFromEnvFileWithName(filename string) (string, error) {
	// Load the env file
	if err := LoadEnvFile(filename); err != nil {
		return "", err
	}

	// In multi-vendor mode, tokens are handled by WIFIMGR_API_<LABEL>_CREDENTIALS_KEY env vars
	// which are processed by InitializeMultiVendor(). This function doesn't return those tokens
	// directly - they're applied to the API configs.
	return "", nil
}

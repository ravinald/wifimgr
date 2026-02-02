package cmd

import (
	"fmt"

	"github.com/spf13/viper"

	"github.com/ravinald/wifimgr/api"
	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/vendors"
	"github.com/ravinald/wifimgr/internal/vendors/meraki"
	"github.com/ravinald/wifimgr/internal/vendors/mist"
	"github.com/ravinald/wifimgr/internal/xdg"
)

// Multi-vendor global state
var (
	// apiFlag is the --api flag value for targeting specific APIs
	apiFlag string

	// apiRegistry manages multiple API client instances
	apiRegistry *vendors.APIClientRegistry

	// cacheManager manages per-API cache files
	cacheManager *vendors.CacheManager

	// cacheAccessor provides O(1) lookups across all API caches
	cacheAccessor *vendors.CacheAccessor
)

// InitializeMultiVendor initializes the multi-vendor infrastructure.
// This should be called after Viper config is loaded.
func InitializeMultiVendor() error {
	// Create registry
	apiRegistry = vendors.NewAPIClientRegistry()

	// Register vendor factories
	apiRegistry.RegisterFactory("mist", createMistClient)
	apiRegistry.RegisterFactory("meraki", createMerakiClient)

	// Build API configs from Viper (uses config package which applies env overrides)
	apiConfigs, warnings := config.BuildAPIConfigsFromViper()

	// Log any config warnings
	for _, w := range warnings {
		logging.Warnf("Config: %s", w.Message)
	}

	// Initialize clients
	if len(apiConfigs) > 0 {
		initErrors := apiRegistry.InitializeClients(apiConfigs)
		for _, err := range initErrors {
			logging.Warnf("API init: %v", err)
		}
	}

	// Initialize cache manager if we have APIs
	if apiRegistry.Count() > 0 {
		// Get cache directory from config
		cacheDir := viper.GetString("files.cache_dir")
		if cacheDir == "" {
			cacheDir = xdg.GetCacheDir()
		}

		logging.Debugf("Multi-vendor cache directory: %s", cacheDir)
		cacheManager = vendors.NewCacheManager(cacheDir, apiRegistry)

		if err := cacheManager.Initialize(); err != nil {
			return fmt.Errorf("failed to initialize cache manager: %w", err)
		}

		// Create and set global cache accessor for cross-package access
		cacheAccessor = vendors.NewCacheAccessor(cacheManager)
		vendors.SetGlobalCacheAccessor(cacheAccessor)

		logging.Infof("Initialized %d API connections", apiRegistry.Count())
	}

	return nil
}

// createMistClient creates a Mist vendor client from config.
func createMistClient(config *vendors.APIConfig) (vendors.Client, error) {
	apiToken := config.Credentials["api_token"]
	if apiToken == "" {
		return nil, fmt.Errorf("missing api_token credential")
	}

	orgID := config.Credentials["org_id"]
	if orgID == "" {
		return nil, fmt.Errorf("missing org_id credential")
	}

	// Create legacy api.Client for the Mist adapter
	legacyClient := api.NewClientWithOptions(
		apiToken,
		config.URL,
		orgID,
	)

	return mist.NewAdapter(legacyClient, orgID), nil
}

// createMerakiClient creates a Meraki vendor client from config.
func createMerakiClient(config *vendors.APIConfig) (vendors.Client, error) {
	apiKey := config.Credentials["api_key"]
	if apiKey == "" {
		return nil, fmt.Errorf("missing api_key credential")
	}

	orgID := config.Credentials["org_id"]
	if orgID == "" {
		return nil, fmt.Errorf("missing org_id credential")
	}

	return meraki.NewAdapter(apiKey, config.URL, orgID, meraki.WithSuppressOutput(suppressOutput))
}

// GetAPIRegistry returns the global API registry.
func GetAPIRegistry() *vendors.APIClientRegistry {
	return apiRegistry
}

// GetCacheManager returns the global cache manager.
func GetCacheManager() *vendors.CacheManager {
	return cacheManager
}

// ValidateAPIFlag validates the --api flag value against registered APIs.
func ValidateAPIFlag() error {
	if apiFlag == "" {
		return nil
	}

	if apiRegistry == nil {
		return fmt.Errorf("API registry not initialized")
	}

	if !apiRegistry.HasAPI(apiFlag) {
		return FormatAPINotFoundError(apiFlag)
	}

	return nil
}

// GetTargetAPIs returns the API labels to target based on target positional arg or --api flag.
// If target/--api is set, returns only that API. Otherwise returns all APIs.
func GetTargetAPIs() []string {
	if apiFlag != "" {
		return []string{apiFlag}
	}
	if apiRegistry == nil {
		return nil
	}
	return apiRegistry.GetAllLabels()
}

// SetAPITarget sets the API target from a positional argument.
// This should be called from command handlers before ValidateAPIFlag/GetTargetAPIs.
func SetAPITarget(target string) {
	if target != "" {
		apiFlag = target
	}
}

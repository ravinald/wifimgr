# Configuration Guidelines

## Sample Configurations

Two sample configuration files are provided:

- **[config-minimal-sample.json](config-minimal-sample.json)** - Minimal configuration with just the required options. Uses defaults for everything else. Start here for a quick setup.

- **[config-complete-sample.json](config-complete-sample.json)** - Comprehensive configuration showing all available options with inline documentation. Use as a reference when customizing your setup.

Copy the minimal sample to get started:

```bash
cp docs/config-minimal-sample.json ~/.config/wifimgr/wifimgr-config.json
```

## Site Configuration

- Site names should follow the format: `\w{2}\-[Ww]?\w{3}\-\w{3,10}`
  - Example: `US-SFO-TESTDRY`, `US-NYC-OFFICE`, etc.
  
- The `site_config.name` is what the CLI commands use to identify sites, NOT the config ID
  - For apply commands, use the value in `site_config.name`, not the config ID
  - For example: `./wifimgr apply US-SFO-TESTDRY ap` uses the name in the `site_config.name` field

- Best practice: The `site_config.name` should match the site config ID in the configuration file
  - For example, if the config ID is "US-SFO-TESTDRY", then `site_config.name` should also be "US-SFO-TESTDRY"
  - This helps avoid confusion between config IDs and actual site names

## File Structure

- Config files follow a specific structure:
  ```json
  {
    "version": 1.0,
    "config": {
      "CC-IATA-ID": {
        "site_config": {
          "name": "CC-IATA-ID"",
          "address": "...",
          "country_code": "...",
          "timezone": "..."
        },
        "devices": {
          "aps": {...},
          "switches": [...],
          "wan_edge": [...]
        }
      }
    }
  }
  ```

## Important Fields

- The `magic` field must always be preserved in config files for APs and devices
- Never remove or delete the `magic` field during any operations or transformations
- The `magic` field is directly used for device identification in the API:
  - For APs: `APConfig.Magic` corresponds to `api.AP.Magic`
  - For inventory items: `Magic` corresponds to `api.InventoryItem.Magic`
- When implementing code that transforms device configurations or prepares them for API requests, always ensure the magic field is preserved and correctly handled

## Inventory Safety Feature

The local inventory configuration serves as a **mandatory allowlist and fail-safe mechanism** for write operations. This feature prevents accidental modifications to devices that you haven't explicitly authorized.

### How It Works

**Write Operations** (apply, configure, assign, unassign):
- A device must exist in **BOTH** the API inventory (devices in your Mist/Meraki account) **AND** the local inventory file
- If a device is only in the API but not allowlisted in your local inventory, write operations are blocked with a safety error
- This provides a second layer of protection against unintended changes

**Read Operations** (show, search, list):
- Devices only need to exist in the API inventory
- You can view and search all devices without any allowlist restriction
- This allows you to discover devices before deciding to add them to your local inventory

### Inventory Configuration

The local inventory is defined in your configuration file (e.g., `inventory.json`):

```json
{
  "version": 1.0,
  "config": {
    "inventory": {
      "ap": [
        "aa:bb:cc:dd:ee:01",
        "aa:bb:cc:dd:ee:02"
      ],
      "switch": [
        "aa:bb:cc:dd:ee:03"
      ],
      "gateway": [
        "aa:bb:cc:dd:ee:04"
      ]
    }
  }
}
```

### Typical Workflow

1. **Discover devices**: Use `show` or `search` commands to view all available devices
2. **Review devices**: Identify which devices you want to manage
3. **Add to inventory**: Update your local inventory file to allowlist those device MACs
4. **Apply changes**: Now `apply` commands will work on those allowlisted devices
5. **Add more devices**: As you onboard new devices, add them to the inventory file

### Why This Safety Mechanism?

- **Prevents accidental bulk modifications**: If you run `apply` on a site, only allowlisted devices are affected
- **Multi-user protection**: If multiple team members use the tool, the inventory acts as explicit authorization
- **Audit trail**: The inventory file documents which devices you have approved for automated management
- **Gradual deployment**: Add devices to inventory one at a time as you gain confidence with the tool

### Error Messages

If you try to modify a device that's not in your local inventory, you'll see an error like:

```
SAFETY CHECK FAILED: Device aa:bb:cc:dd:ee:ff is not in inventory - refusing to update
```

This means the device exists in your Mist/Meraki account but is not in your local `inventory.json` file. Add it to the appropriate list (ap, switch, or gateway) to allowlist it and enable write operations.

## Command-Line Flags and Configuration

### Simplified Flag Structure

With the Cobra migration, the CLI now uses a **simplified flag structure** that leverages Viper for configuration management:

**Global Flags (Available on all commands):**
- `-d, --debug` - Enable debug mode and debug-level logging
- `-e, --env` - Read API token from .env.wifimgr file instead of config
- `-h, --help` - Show help for any command

**Configuration Management:**
- Most configuration options are handled through **Viper** and configuration files
- The `.cobra.yaml` file defines project settings (author, license, package name)
- Complex flags like `--format`, `--force`, `--dry-run` have been **removed from global scope**
- Configuration values are read from config files rather than command-line flags

**Benefits of Simplified Structure:**
- Reduced flag conflicts and complexity
- Better configuration management through files
- Cleaner command-line interface
- Easier maintenance and extension

### Command-Specific Flags

Some commands may define their own local flags when needed:

```go
// Example: set ap site command
siteCmd.Flags().StringVar(&assignmentFile, "file", "", "File containing AP MACs to assign")
siteCmd.Flags().StringVarP(&targetSite, "site", "s", "", "Target site for bulk assignment")
```

## API Token Handling

The application supports multiple ways to provide the API token:

1. **Configuration File**: The API token can be stored (encrypted) in the configuration file
2. **Environment File**: The API token can be provided in a file called `.env.wifimgr` in the current directory
3. **Command-line Flag**: Use the `-e` flag to load the API token from the .env.wifimgr file (e.g., `./wifimgr -e show api sites`)
4. **Interactive Input**: The application will prompt for the API token if not found in the config or env file

### Using .env.wifimgr for API Token

For easier testing, use the `-e` flag to load the token from this file:

```
./wifimgr -e show api sites
```

The application will automatically read this file during startup and use the token for all API operations. This is particularly useful for:

- Automated testing where interactive prompts aren't feasible
- Development environments where you don't want to encrypt the token in the config
- CI/CD pipelines that need to run tests against the API

The `.env.wifimgr` file should contain the environment variable `WIFIMGR_API_TOKEN` with your actual unencrypted API token:

```
WIFIMGR_API_TOKEN=your_actual_api_token_here
```

This approach keeps sensitive credentials separate from your configuration files, making it safer to share or version control your configurations.

## Viper Configuration System

The application has migrated from struct-based configuration to **Viper** for more flexible and maintainable configuration management.

### Configuration Architecture

**Core Principles:**
- **Viper Exclusive**: Main configuration is managed exclusively through Viper (no Config structs)
- **Direct Access**: Configuration values accessed via `viper.Get*()` functions
- **Helper Functions**: Convenience functions provided for complex configuration sections
- **Backwards Compatible**: Existing configuration files work without changes

### Configuration File Structure

The main configuration file (`~/.config/wifimgr/wifimgr-config.json`) supports the following structure:

```json
{
  "version": 1.0,
  "files": {
    "config_dir": "~/.config/wifimgr",
    "site_configs": ["sites/us-oak-pina.json"],
    "cache_dir": "~/.cache/wifimgr",
    "inventory": "~/.config/wifimgr/inventory.json",
    "log_file": "~/.local/state/wifimgr/wifimgr.log",
    "schemas": "~/.local/share/wifimgr/schemas"
  },
  "api": {
    "credentials": {
      "api_id": "...",
      "api_token": "...",
      "org_id": "...",
      "token_encrypted": true
    },
    "url": "https://api.mist.com/",
    "rate_limit": 5000,
    "results_limit": 100,
    "cache_ttl": 86400
  },
  "display": {
    "jsoncolor": {
      "null":   { "hex": "#767676", "ansi256": "244", "ansi": "8" },
      "bool":   { "hex": "#FFFFFF", "ansi256": "15",  "ansi": "7" },
      "number": { "hex": "#00FFFF", "ansi256": "51",  "ansi": "6" },
      "string": { "hex": "#00FF00", "ansi256": "46",  "ansi": "2" },
      "key":    { "hex": "#0000FF", "ansi256": "21",  "ansi": "4" },
      "bytes":  { "hex": "#767676", "ansi256": "244", "ansi": "8" },
      "time":   { "hex": "#00FF00", "ansi256": "46",  "ansi": "2" }
    },
    "commands": {
      "show.api.sites": {
        "format": "table",
        "title": "Sites",
        "fields": [...]
      }
    }
  },
  "logging": {
    "enable": true,
    "level": "debug",
    "format": "text",
    "stdout": false
  }
}
```

### Accessing Configuration Values

**Direct Viper Access:**
```go
// Basic configuration values
orgID := viper.GetString("api.credentials.org_id")
rateLimit := viper.GetInt("api.rate_limit")
logLevel := viper.GetString("logging.level")

// JSON color configuration
stringHex := viper.GetString("display.jsoncolor.string.hex")     // "#00FF00"
numberAnsi := viper.GetString("display.jsoncolor.number.ansi256") // "51"

// Complex nested structures
displayCommands := viper.GetStringMap("display.commands")
siteConfigFiles := viper.GetStringSlice("files.site_configs")
```

**Helper Functions:**
```go
// JSON color configuration helpers
stringColors := config.GetJsonColorConfig("string")
// Returns: map[string]string{"hex": "#00FF00", "ansi256": "46", "ansi": "2"}

allColors := config.GetAllJsonColorConfigs()
// Returns: map[string]map[string]string with all color types

// Configuration adapter for encryption/token management
configAdapter := config.NewViperConfigAdapter(configPath)
```

### JSON Color Configuration

The `display.jsoncolor` section provides color configuration for JSON syntax highlighting:

**Supported Types:**
- `null` - Null values
- `bool` - Boolean values  
- `number` - Numeric values
- `string` - String values
- `key` - Object keys
- `bytes` - Byte arrays
- `time` - Timestamp values

**Color Formats:**
- `hex` - Hexadecimal color codes (e.g., "#00FF00")
- `ansi256` - 256-color ANSI codes (e.g., "46")
- `ansi` - Basic ANSI color codes (e.g., "2")

**Default Values:**
All color types have sensible defaults configured in `internal/config/viper_config.go` that will be used if not specified in the configuration file.

### Cache Configuration

The cache system tracks age and staleness for each API connection.

**Cache TTL Setting:**

The `cache_ttl` setting (in seconds) controls when the cache is considered stale:

```json
{
  "api": {
    "mist-prod": {
      "vendor": "mist",
      "cache_ttl": 86400
    }
  }
}
```

**Values:**
- `86400` (default) - Cache expires after 24 hours
- `0` - Cache never expires (manual refresh only)
- Any positive integer - Cache expires after that many seconds

**Cache Status:**

Check cache status with:
```bash
wifimgr show api status
```

Output shows cache state per API: `ok`, `stale`, `corrupted`, or `missing`.

**Cache Refresh:**

Refresh the cache manually:
```bash
wifimgr refresh cache              # Refresh all APIs
wifimgr refresh cache mist-prod    # Refresh specific API
```

**Cache Age Tracking:**

The cache stores metadata including:
- `LastRefresh` - Timestamp of last successful refresh
- `RefreshDurationMs` - How long the refresh took

This information is used to determine staleness based on the `cache_ttl` setting.

### Migration from Config Structs

**Before (Struct-based):**
```go
func SomeFunction(cfg *config.Config) {
    orgID := cfg.API.Credentials.OrgID
    logFile := cfg.Files.LogFile
}
```

**After (Viper-based):**
```go
func SomeFunction() {
    orgID := viper.GetString("api.credentials.org_id")
    logFile := viper.GetString("files.log_file")
}
```

**Benefits:**
- **Simplified Code**: No need to pass Config structs around
- **Flexible Access**: Direct access to any configuration value
- **Better Defaults**: Comprehensive default values via Viper
- **Type Safety**: Strong typing with `GetString()`, `GetInt()`, etc.
- **Environment Integration**: Automatic environment variable support

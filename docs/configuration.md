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

- The `site_config.name` is what the CLI commands use to identify sites, NOT the config ID
  - For apply commands, use the value in `site_config.name`, not the config ID
  - For example: `./wifimgr apply US-SFO-TESTDRY ap` uses the name in the `site_config.name` field

- Best practice: The `site_config.name` should match the site config ID in the configuration file
  - For example, if the config ID is "US-SFO-TESTDRY", then `site_config.name` should also be "US-SFO-TESTDRY"
  - This helps avoid confusion between config IDs and actual site names

> **Note on examples:** Site names used throughout this documentation (e.g., `US-SFO-TESTDRY`, `US-NYC-OFFICE`) follow an adaptation of [UN/LOCODE](https://en.wikipedia.org/wiki/UN/LOCODE) for illustrative purposes. The tool does not enforce any specific naming convention—use whatever naming scheme works best for your organization.

## Templates

Templates are an **app-level convenience** that expand into explicit device settings at apply time—they are NOT vendor-side profile management. wifimgr templates exist only in your local configuration. When you apply changes, templates are expanded into fully explicit configurations that are pushed directly to each device.

```json
{
  "files": {
    "templates": ["templates/radio.json", "templates/wlan.json"]
  }
}
```

Template files contain named configurations that can be referenced in site and device configs:

```json
{
  "sites": {
    "US-NYC-OFFICE": {
      "profiles": {
        "wlan": ["corp-secure", "guest"]
      },
      "wlan": ["corp-secure", "guest"],
      "devices": {
        "ap": {
          "aa:bb:cc:dd:ee:01": {
            "name": "AP-01",
            "radio_profile": "high-density"
          }
        }
      }
    }
  }
}
```

- **profiles.wlan**: WLANs to create at the site
- **wlan** (site-level): WLANs to apply to all APs by default
- **devices.ap[mac].wlan**: WLANs to apply to specific APs (overrides site default)

At apply time, `radio_profile` and `wlan` are expanded into explicit configuration values, which are then sent to the API. This is simpler than vendor-side template systems because all configuration is local and explicit.

WLAN assignments are validated during `apply` and `lint config` — site-level and device-level WLANs must be declared in `profiles.wlan`, and each profile must have a corresponding template. See **[Templates — WLAN Validation](templates.md#wlan-validation)** for details.

For complete template documentation, see **[Templates](templates.md)**.

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
- A device must exist in **BOTH** the API inventory (devices in your Mist/Meraki/Ubiquiti account) **AND** the local inventory file
- If a device is only in the API but not allowlisted in your local inventory, write operations are blocked with a safety error
- This provides a second layer of protection against unintended changes

**Read Operations** (show, search, list):
- Default `show`/`search` output is the managed set — the devices armed in the inventory below
- Add the `all` keyword to widen any `show` to every device the API knows about
- This keeps the daily view to what you manage while discovery stays one keyword away

### Inventory Configuration

The allowlist is **scoped per site**: a MAC is armed for the named site only. Scoping the
allowlist to a site keeps the write blast radius narrow and means the decision to modify a device
never depends on the (possibly stale) cached site assignment.

MACs are stored as lowercase bare hex (the canonical on-disk form `import ... inventory` writes).
You may hand-edit in any accepted format — colon, hyphen, dot, or bare, any case — since the file
is normalized on read; the bare form just keeps keys unambiguous.

```json
{
  "version": 1,
  "config": {
    "inventory": {
      "site": {
        "US-LAB-01": {
          "ap": [
            "aabbccddee01",
            "aabbccddee02"
          ],
          "switch": [
            "aabbccddee03"
          ],
          "gateway": []
        },
        "US-LAB-02": {
          "ap": ["aabbccddee05"]
        }
      }
    }
  }
}
```

> **Migration:** the previous global layout (`config.inventory.ap/switch/gateway`) is no longer
> accepted. Loading a file in the old shape fails loud rather than silently treating a flat list as
> "every site" — that would widen the blast radius instead of narrowing it. Move each MAC under the
> site it belongs to.

### Typical Workflow

1. **Discover devices**: Run `show ap all` (or `search`) to view every device the API knows
2. **Review devices**: Identify which devices you want to manage, and at which site
3. **Arm them**: Add each MAC under its site in `config.inventory.site.<SITE>.<type>`
4. **Apply changes**: `apply site <SITE> <type>` now operates on those armed devices
5. **Add more devices**: As you onboard new devices, arm them under their site

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

This means the device exists in your Mist/Meraki/Ubiquiti account but is not armed in your local `inventory.json` for that site. Add its MAC under `config.inventory.site.<SITE>.<type>` to allowlist it and enable write operations.

## Command-Line Flags and Configuration

### Simplified Flag Structure

With the Cobra migration, the CLI uses a **simplified flag structure** backed by Viper for configuration management:

**Global Flags (Available on all commands):**

Flags are reserved for *operational* concerns — how the app runs — while what you
ask it to do is expressed with positional keywords.

- `-c, --config <path>` - Use an alternate config file
- `-d, --debug` - Enable debug-level logging (`--dd` / `--ddd` for more)
- `-e, --env` - Read API token from `.env.wifimgr` instead of config
- `-h, --help` - Show help for any command
- `-q, --quiet` - Suppress non-essential output (progress and status notices)
- `-y, --yes` - Assume "yes" to confirmation prompts (for automation)
- `--no-input` - Never prompt; fail with guidance instead of blocking
- `--no-color` - Disable colored output (also honored: `NO_COLOR`, `TERM=dumb`,
  and a non-terminal stdout)
- `--version` - Print version, commit, and build time

**Output streams:** primary output (tables, CSV, JSON) goes to stdout; logs,
warnings, and status notices go to stderr. `format json` and `format csv` are
always plain — no color escapes — so `... format json | jq` is safe.

**Configuration Management:**
- Most configuration options are handled through **Viper** and configuration files
- The `.cobra.yaml` file defines project settings (author, license, package name)
- Options like format, force, and diff are handled as **positional keywords** rather than flags
- Configuration values are read from config files rather than command-line flags

**Benefits of Simplified Structure:**
- Reduced flag conflicts and complexity
- Better configuration management through files
- Cleaner command-line interface
- Easier maintenance and extension

### Command-Specific Flags

Most commands take Junos-style positional arguments rather than flags. When a
local flag is genuinely needed, use Cobra's `Flags()` method on the relevant
subcommand. Avoid `--site`-style flags for arguments that the positional
grammar already covers.

## API Token Handling

The application supports multiple ways to provide the API token:

1. **Configuration File**: The API token can be stored (encrypted) in the configuration file
2. **Environment File**: The API token can be provided in a file called `.env.wifimgr` in the current directory
3. **Command-line Flag**: Use the `-e` flag to load the API token from the .env.wifimgr file (e.g., `./wifimgr -e show sites`)
4. **Interactive Input**: The application will prompt for the API token if not found in the config or env file

### Using .env.wifimgr for API Token

For easier testing, use the `-e` flag to load the token from this file:

```
./wifimgr -e show sites
```

The application will automatically read this file during startup and use the token for all API operations. This is particularly useful for:

- Automated testing where interactive prompts aren't feasible
- Development environments where you don't want to encrypt the token in the config
- CI/CD pipelines that need to run tests against the API

The `.env.wifimgr` file should contain credentials following the pattern `WIFIMGR_API_<LABEL>_CREDENTIALS_KEY`:

```
WIFIMGR_API_MIST_CREDENTIALS_KEY=your_api_key_here
WIFIMGR_API_MIST_CREDENTIALS_ORG=your_org_id_here
```

This approach keeps sensitive credentials separate from your configuration files, making it safer to share or version control your configurations.

### .env.wifimgr File Format

The file uses a simple `KEY=value` format with support for comments and quoted values.

**Basic syntax:**
```bash
# Comments start with #
WIFIMGR_API_MIST_CREDENTIALS_KEY=your-api-key-here
WIFIMGR_PASSWORD=simple-password
```

**Quoting rules:**

Values can be wrapped in single or double quotes. Quotes are required if your value contains leading/trailing spaces:

```bash
# Both single and double quotes work
WIFIMGR_PASSWORD="my secret password"
WIFIMGR_PASSWORD='my secret password'

# Use double quotes with single quotes inside
WIFIMGR_PASSWORD="it's a secret"

# Use single quotes with double quotes inside
WIFIMGR_PASSWORD='say "hello"'
```

**Escape sequences (within quoted strings):**

| Sequence | Result |
|----------|--------|
| `\\` | Literal backslash `\` |
| `\"` | Literal double quote `"` |
| `\'` | Literal single quote `'` |
| `\n` | Newline |
| `\t` | Tab |

**Examples with escapes:**
```bash
# Password containing a double quote
WIFIMGR_PASSWORD="pass\"word"

# Password containing a backslash
WIFIMGR_PASSWORD="path\\to\\secret"

# Password containing both quote types
WIFIMGR_PASSWORD="it\'s \"complex\""
```

**Note:** The `WIFIMGR_` prefix is automatically added if not present, so `PASSWORD=x` becomes `WIFIMGR_PASSWORD=x`.

## Encrypting Secrets

The `encrypt` command allows you to encrypt sensitive values for use in configuration files. This is useful for storing secrets like WLAN PSKs, RADIUS shared secrets, or additional API tokens.

**Note:** The `encrypt` command works without any configuration file or API credentials. You can run it immediately after installing wifimgr.

### Using the encrypt Command

```bash
# Encrypt any secret (interactive, hidden input)
wifimgr encrypt

# Encrypt a WiFi PSK with validation
wifimgr encrypt psk
```

The command prompts for:
1. The secret value (hidden input, with confirmation)
2. An encryption password (min 8 chars, hidden input, with confirmation)

Output is an encrypted string with `enc:` prefix:
```
enc:U2FsdGVkX1+abc123def456...
```

### Using Encrypted Values in Configuration

Paste encrypted values directly into config files:

```json
{
  "templates": {
    "wlan": {
      "Corp-WiFi": {
        "ssid": "CorpNet",
        "auth": {
          "type": "psk",
          "psk": "enc:U2FsdGVkX1+abc123def456..."
        }
      }
    }
  }
}
```

When the application reads an encrypted value (detected by `enc:` prefix), it will prompt for the decryption password unless `WIFIMGR_PASSWORD` is set.

### Non-Interactive Decryption

For CI/CD pipelines, automated scripts, or non-interactive usage, provide the decryption password via environment variable:

```bash
export WIFIMGR_PASSWORD="your-decryption-password"
wifimgr show sites
```

Or in `.env.wifimgr`:
```
WIFIMGR_PASSWORD=your-decryption-password
```

When `WIFIMGR_PASSWORD` is set (either as an environment variable or in `.env.wifimgr`), the application will use it automatically to decrypt any `enc:` prefixed values without prompting.

**Security note:** The password remains available in memory for the duration of the session when loaded via `.env.wifimgr` with the `-e` flag, enabling decryption of multiple values without re-prompting.

### Secrets in the Cache

WLAN secrets (PSK, RADIUS shared secret) are encrypted at rest. When `refresh` pulls
WLANs from a vendor API, it encrypts any captured secret with the same `enc:` scheme
before writing the cache file — secrets never land on disk in the clear. Refresh resolves
the encryption password from `WIFIMGR_PASSWORD` or prompts for it once.

Because the password protects the cache, use the **same** password across refreshes; a
secret encrypted with one password can't be decrypted with another. If you refresh with a
different password, re-run `refresh` to re-encrypt under the intended one.

The `import api site` and `import api templates` commands emit these secrets so the
imported file is ready to apply:

| Keyword   | Output for a stored secret                                                  |
|-----------|-----------------------------------------------------------------------------|
| *(none)*  | The stored `enc:` value, verbatim — applies as-is (apply decrypts it)        |
| `decrypt` | Decrypted plaintext, using the encryption password                          |

If the cache somehow holds a secret in the clear, the default output masks it as
`*secret*` rather than echoing plaintext. `decrypt` needs the encryption password
(`WIFIMGR_PASSWORD` or prompt); a wrong password also falls back to the mask.

```bash
wifimgr import api site US-SFO-LAB type wlans            # PSK kept as its enc: value
WIFIMGR_PASSWORD=… wifimgr import api site US-SFO-LAB type wlans decrypt   # plaintext PSK
```

> Meraki returns the PSK only on its per-SSID endpoint, so `refresh` fetches each SSID
> individually to capture it. Meraki does not return RADIUS shared secrets at all.

### PSK Validation

When encrypting WiFi passwords, use `encrypt psk` to validate against IEEE 802.11i requirements:
- Length: 8-63 characters
- Characters: Printable ASCII only (codes 32-126)

This ensures the PSK will be accepted by Mist, Meraki, and Ubiquiti platforms.

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
    "templates": ["wlans/us-oak-pina.json"],
    "cache_dir": "~/.cache/wifimgr",
    "inventory": "~/.config/wifimgr/inventory.json",
    "log_file": "~/.local/state/wifimgr/wifimgr.log",
    "schemas": "~/.local/share/wifimgr/schemas"
  },
  "api": {
    "mist": {
      "vendor": "mist",
      "url": "https://api.mist.com",
      "credentials": {
        "org_id": "...",
        "api_key": "..."
      },
      "rate_limit": 5000,
      "results_limit": 100,
      "cache_ttl": 86400,
      "connection_timeout": 5,
      "sync_type": ["ap", "switch", "gateway"]
    }
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
      "show.sites": {
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

### API Connection Timeout

`connection_timeout` (seconds) bounds **connection establishment** — TCP dial plus TLS handshake —
per API. It does **not** cap the overall request, so slow-but-working calls (large Mist org
inventories, Meraki per-device fetches) still complete; it only makes an unreachable host fail fast
instead of hanging until the request timeout.

- **Default:** 5 seconds.
- **Per-API override:** set `connection_timeout` inside an API entry (as above). A value below 1 is
  treated as the default; the floor keeps a typo from disabling the dial timeout.
- **Meraki:** ignored — its SDK builds its HTTP client internally with no transport hook. Mist,
  Aruba, and Ubiquiti honor it.

A dead host now surfaces as `unhealthy` / `connection failure` in `show api status` within
`connection_timeout` seconds rather than ~30s.

### Sync Type

`sync_type` is a per-API list declaring which device types a refresh collects:
any of `ap`, `switch`, `gateway`. Site attributes (name, address, timezone)
always sync.

- **Omitted or `[]`:** site attributes only — no device inventory, configs,
  statuses, or BSSIDs.
- **`["ap"]`:** APs only, with their statuses and BSSIDs.
- **`["ap", "switch", "gateway"]`:** all device types.

Statuses are fetched only when at least one device type is listed; BSSIDs only
when `ap` is listed. Org-level templates, profiles, and WLANs always sync.

> **Upgrade note:** earlier versions collected all three device types
> unconditionally. An API without `sync_type` now syncs site attributes only —
> add `sync_type` to keep collecting devices.

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
wifimgr refresh                   # Managed devices, all APIs
wifimgr refresh target mist-prod  # Managed devices, specific API
wifimgr refresh all               # Everything the API has, all APIs
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

## Search Output Sort

The `wifimgr search wireless` and `wifimgr search wired` tables sort by
SSID/switch name, then AP/switch hostname, then client MAC. The default
hostname tier is a natural-string sort, which groups `ap1-*, ap2-*, ap10-*`
correctly but treats the hostname as a single opaque token.

When your AP or switch names encode physical metadata (floor, building, pod),
you can pull individual fields out and sort by them instead. Define a regex
with named capture groups and list which groups to sort by, in priority
order.

### Minimal example

APs named `apX-F` where `F` is the floor:

```json
{
  "display": {
    "sort": {
      "ap_name": {
        "pattern": "^ap(?P<num>\\d+)-(?P<floor>\\d+)$",
        "keys": ["floor", "num"]
      }
    }
  }
}
```

With this config:
- `ap1-15, ap2-15, ap3-15, ap4-15` all cluster on floor 15
- then `ap1-16, ap2-16, ap3-16, ap4-16` cluster on floor 16

Without this config, the natural sort gives `ap1-15, ap1-16, ap2-15, ap2-16, ...`
(AP-number major, floor minor), which is harder to scan when you're asking
"who's on floor 15 right now?"

### Multi-key example

For a building-floor-AP naming scheme like `A-1-ap01`:

```json
{
  "display": {
    "sort": {
      "ap_name": {
        "pattern": "^(?P<building>[A-Z]+)-(?P<floor>\\d+)-ap(?P<num>\\d+)$",
        "keys": ["building", "floor", "num"]
      }
    }
  }
}
```

Buildings cluster first, floors within each building, APs within each floor.
`keys` can reference as many or as few named groups as you want, in any
priority order. The textual position in the regex doesn't have to match
the sort priority.

### Switch sort

The same shape applies under `display.sort.switch_name` for the wired search:

```json
{
  "display": {
    "sort": {
      "switch_name": {
        "pattern": "^sw(?P<num>\\d+)-(?P<floor>\\d+)$",
        "keys": ["floor", "num"]
      }
    }
  }
}
```

### Rules and caveats

- **Regex escaping**: these are JSON strings, so backslashes need to be
  doubled. `\d` in the regex becomes `\\d` in the JSON file.
- **Engine**: Go RE2. Named groups use `(?P<name>...)`. No lookaround,
  no backreferences.
- **Numeric vs string segments**: a captured group parses as an integer
  if it's pure digits, otherwise stays a string. Integer segments compare
  numerically, so `num=10` sorts after `num=9`, not between `num=1` and
  `num=2`. String segments compare naturally.
- **Names that don't match**: AP names that don't match the pattern sort
  after matching names, using natural string order within the unmatched
  bucket. That keeps outliers visually obvious.
- **Invalid config**: an unparseable regex or a `keys` entry that doesn't
  name a capture group warns once in the log and falls back to the default
  natural sort. wifimgr never crashes on bad sort config.
- **No config**: omit the `sort` block (or either of its sub-keys) to keep
  the default. The feature is opt-in.

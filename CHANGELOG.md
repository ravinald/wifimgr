# Changelog

All notable changes to wifimgr are documented in this file.

## [0.1.0] - 2026-02-02

This is the initial public release of wifimgr, providing comprehensive network infrastructure management for Mist and Meraki platforms.

### Major Features

#### Multi-Vendor Support
- Full support for Mist and Meraki management from single CLI and configuration
- Vendor-agnostic CLI with automatic capability detection
- Multi-API configuration with per-API settings and cache management
- See [Multi-Vendor Documentation](docs/multi-vendor/) for setup and usage

#### Search Functionality
- **Wireless client search** - Find connected wireless clients by hostname, MAC, or partial match
- **Wired client search** - Find connected wired clients (Mist and Meraki)
- **Cost estimation** - Intelligent warnings for expensive multi-network searches
- **Performance optimization**:
  - MAC address searches optimized to single API call
  - Site-scoped searches for faster results
  - `force` argument to bypass expensive search confirmations
- Example: `wifimgr search wireless laptop` or `wifimgr search wired john --api meraki-corp`

#### Cache Architecture Refactoring
- Monolithic 3,479-line cache module refactored into ~25 focused, testable modules
- Improved separation of concerns with dedicated modules for:
  - Cache storage and I/O (`cache_storage.go`, `cache_storage_indexes.go`)
  - Vendor-layer cache management (`cache_manager.go`, `cache_manager_refresh.go`)
  - Device and site accessors (`cache_accessor_devices.go`, `cache_accessor_sites.go`)
- All modules maintained at <500 lines for readability and maintainability
- 100% backward compatible - existing cache files work without migration

#### Cache Management
- **Age tracking** - Cache stores `LastRefresh` timestamp and refresh duration
- **TTL configuration** - Per-API `cache_ttl` setting (default: 86400 seconds / 24 hours)
- **Staleness detection** - Automatic identification of stale cache based on TTL
- **Cache status command** - `wifimgr show api status` displays cache health per API
- **Manual refresh** - `wifimgr refresh cache` syncs with API and updates freshness metadata
- **Configurable expiry** - Set `cache_ttl: 0` to disable auto-expiry for lab environments

### Device Management

#### Access Point (AP) Configuration
- Full AP configuration support via `apply` command
- Configuration as code with JSON site configs
- Dry-run support with `--dry-run` flag
- Automatic backups before changes
- Rollback capability with `apply rollback` command

**Supported AP settings:**
- Radio configuration (2.4 GHz, 5 GHz, 6 GHz bands)
- IP configuration (DHCP and static)
- WLAN assignments and profiles
- LED, mesh, and BLE configuration
- Per-device customization

**Vendor-specific extensions:**
- Mist: Floor plan positioning, hardware control (ethernet disable), RRM settings
- Meraki: RF profiles, GPS coordinates, per-band advanced settings

#### Switch and Gateway Management
- View and inventory only (apply support planned for v1.1+)
- Full visibility into switch port configurations and gateway settings
- Compatible with all infrastructure discovery commands

### Display and Output

#### Table Formatter
- Professional table rendering with column customization
- Multiple output formats:
  - **Table** - Default formatted output with status indicators
  - **JSON** - Full JSON with field name resolution and color syntax highlighting
  - **CSV** - Spreadsheet-compatible output
- Configurable colors for JSON output
- Field name resolution (IDs → human-readable names)

#### Symbols and Status Indicators
- Consistent status symbols (✓, ✗, etc.) across all output
- Color-coded device status (connected, online, offline, etc.)

### Commands

**Core Commands:**
- `show` - Display sites, devices, inventory, WLANs
- `apply` - Push AP configuration to API with dry-run and rollback
- `search` - Find connected clients (Mist and Meraki)
- `import` - Bootstrap configs from API state
- `refresh` - Sync cache and manage staleness
- `init` - Create skeleton configuration files
- `set` - Interactive device assignment

**Advanced Features:**
- Multi-site operations with site filtering
- Per-API filtering with `--api` flag
- Debug logging with `-d`, `-dd`, `-ddd` flags
- Environment variable support for credentials

### Configuration

#### API Configuration
```json
{
  "api": {
    "mist-prod": {
      "vendor": "mist",
      "url": "https://api.mist.com/api/v1",
      "credentials": { "org_id": "..." },
      "rate_limit": 5000,
      "cache_ttl": 86400
    }
  }
}
```

#### Token Management
- Multiple storage options: config file, environment file (`.env.wifimgr`), or interactive prompt
- Secure token handling with optional encryption
- Load token via `-e` flag

#### Managed Keys
- Define which fields wifimgr manages per device type
- Prevents accidental overwriting of fields managed externally
- Dot-notation support for nested field management

### Documentation

#### User-Facing
- [User Guide](docs/user-guide.md) - Complete configuration and command reference
- [Configuration Guide](docs/configuration.md) - API setup, tokens, cache management
- [Multi-Vendor Setup](docs/multi-vendor/) - Multi-vendor configuration and workflows
- [Field Mappings](docs/field-mappings.md) - Vendor API field mapping reference
- [Table Formatter](docs/table-formatter.md) - Output formats and customization


### Quality and Testing

- Comprehensive test suite covering:
  - Cache operations and staleness detection
  - Multi-vendor command routing
  - Search functionality with cost estimation
  - Configuration parsing and validation
  - Device configuration transformations
- 100% passing test coverage on core functionality
- Integration testing with real API interactions

### Known Limitations

- **Gateway and Switch apply** - Not yet supported (planned for v0.2+)
- **Incremental cache refresh** - Full refresh only (planned for future release)
- **Cache progress reporting** - Refresh shows log output but not progress bar (planned for v0.1.1)

### Future Roadmap

**v0.1.1 (Patch Release):**
- Cache refresh progress reporting
- Bug fixes and usability improvements based on user feedback

**v0.2 (Minor Release):**
- Gateway configuration via apply command
- Switch configuration via apply command
- User feedback-driven enhancements

**v0.3+:**
- Incremental cache refresh
- Additional vendor support (Aruba, Cisco, etc.)
- Advanced caching policies
- Enhanced table formatting themes

**v1.0:**
- API stability commitment
- Production-ready feature set
- Comprehensive documentation and examples

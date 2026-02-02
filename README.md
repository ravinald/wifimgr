# wifimgr

[![CI](https://github.com/ravinald/wifimgr/actions/workflows/ci.yml/badge.svg)](https://github.com/ravinald/wifimgr/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/ravinald/wifimgr)](https://goreportcard.com/report/github.com/ravinald/wifimgr)
[![Release](https://img.shields.io/github/v/release/ravinald/wifimgr)](https://github.com/ravinald/wifimgr/releases)
[![License](https://img.shields.io/github/license/ravinald/wifimgr)](LICENSE)

A CLI tool for managing network infrastructure across Mist and Meraki using configuration-as-code.

## Quickstart

```bash
# Install
git clone https://github.com/ravinald/wifimgr.git && cd wifimgr && make build

# Configure API access (edit with your org_id, add token to .env.wifimgr)
mkdir -p ~/.config/wifimgr
cp config/wifimgr-config-sample.json ~/.config/wifimgr/wifimgr-config.json

# Verify connection and cache API data
wifimgr -e refresh cache

# List sites
wifimgr show api sites
```

## What It Does

- **Show** - View sites, devices, WLANs, and inventory from API or local config
- **Search** - Find connected clients (wireless and wired) by hostname, MAC, or partial match on Mist and Meraki networks
- **Apply** - Push device configurations to the API with dry-run support (currently AP configuration; Switch and Gateway planned for v0.2+)
- **Import** - Bootstrap config files from current API state
- **Refresh** - Sync local cache with API data and manage cache age/staleness

## Configuration

wifimgr uses a JSON config file for API credentials and settings. The minimum required:

```json
{
  "api": {
    "mist": {
      "vendor": "mist",
      "url": "https://api.mist.com/api/v1",
      "credentials": {
        "org_id": "your-org-uuid"
      }
    }
  },
  "files": {
    "site_configs": ["sites/us-lab.json"]
  }
}
```

Store your API token in `.env.wifimgr` (add to .gitignore):

```
WIFIMGR_API_TOKEN=your-token-here
```

Run with `-e` to load the token:

```bash
wifimgr -e show api sites
```

## Common Commands

```bash
# View
wifimgr show api sites                    # List all sites
wifimgr show api ap site US-LAB-01        # List APs at a site
wifimgr show inventory ap                 # Show AP inventory
wifimgr show intent ap site US-LAB-01     # Show intended AP config

# Apply
wifimgr apply ap US-LAB-01 --dry-run      # Preview changes
wifimgr apply ap US-LAB-01                # Apply AP config to site

# Import from API
wifimgr import api site US-LAB-01         # Preview config from API
wifimgr import api site US-LAB-01 save    # Save to config file

# Search (Mist and Meraki)
wifimgr search wired laptop-john          # Find wired client
wifimgr search wireless 5c:5b:35:8e:4c:f9 # Find wireless client by MAC
wifimgr search wireless john site US-LAB  # Scope search to site
wifimgr search wireless desktop force     # Bypass confirmation for expensive searches

# Cache management
wifimgr show api status                   # Check cache age and freshness
wifimgr refresh cache                     # Sync cache with API
```

## CLI Flags

| Flag | Description |
|------|-------------|
| `-e` | Load API token from `.env.wifimgr` |
| `-d` | Debug logging (info level) |
| `-dd` | Debug logging (debug level) |
| `-ddd` | Debug logging (trace level, includes API responses) |
| `--dry-run` | Preview changes without applying |
| `--api <name>` | Filter to specific API in multi-vendor setups |

## Multi-Vendor Support

Manage both Mist and Meraki from the same config file. Full support for:

- **View** - Sites, devices, inventory, WLANs across all vendors
- **Search** - Find connected clients on Mist or Meraki networks (both wireless and wired)
- **Apply** - Configure APs on either platform
- **Cache** - Automatic cache management with age tracking and TTL

Example configuration:

```json
{
  "api": {
    "mist-prod": {
      "vendor": "mist",
      "url": "https://api.mist.com/api/v1",
      "credentials": { "org_id": "..." },
      "cache_ttl": 86400
    },
    "meraki-corp": {
      "vendor": "meraki",
      "url": "https://api.meraki.com",
      "credentials": { "org_id": "..." },
      "cache_ttl": 86400
    }
  }
}
```

Filter commands to a specific API:

```bash
wifimgr show api sites --api mist-prod
wifimgr search wireless laptop --api meraki-corp
```

## Documentation

- **[Getting Started](GETTING_STARTED.md)** - Navigation guide for all documentation
- **[User Guide](docs/user-guide.md)** - Complete command reference with examples
- **[Configuration Guide](docs/configuration.md)** - API setup and configuration
- **[Multi-Vendor Setup](docs/multi-vendor/user-guide.md)** - Manage both Mist and Meraki
- **[CHANGELOG](CHANGELOG.md)** - v0.1.0 release notes and future roadmap
- **[Known Issues](docs/known-issues.md)** - Current limitations and workarounds
- **[Field Mappings](docs/field-mappings.md)** - Vendor API field mapping reference

## Development

```bash
make build          # Build binary
make test           # Run tests
make test-coverage  # Generate coverage report
```

## License

Apache License 2.0

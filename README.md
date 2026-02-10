# wifimgr

[![CI](https://github.com/ravinald/wifimgr/actions/workflows/ci.yml/badge.svg)](https://github.com/ravinald/wifimgr/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/ravinald/wifimgr)](https://goreportcard.com/report/github.com/ravinald/wifimgr)
[![Release](https://img.shields.io/github/v/release/ravinald/wifimgr)](https://github.com/ravinald/wifimgr/releases)
[![License](https://img.shields.io/github/license/ravinald/wifimgr)](LICENSE)

A CLI tool for managing network infrastructure across Mist and Meraki using configuration-as-code.

## Install

```bash
git clone https://github.com/ravinald/wifimgr.git && cd wifimgr && make build
```

## Quickstart

```bash
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
- **Search** - Find connected clients (wireless and wired) by hostname, MAC, or partial match
- **Apply** - Push device and WLAN configurations with diff preview and automatic backups. Expands app-level templates into explicit configs at apply time
- **Import** - Bootstrap config files from current API state
- **Refresh** - Sync local cache with API data and manage cache age/staleness
- **Multi-vendor** - Manage both Mist and Meraki from a single config file

## Common Commands

```bash
# View
wifimgr show api sites                    # List all sites
wifimgr show api ap site US-LAB-01        # List APs at a site
wifimgr show api wlans site US-LAB-01     # List WLANs at a site

# Apply
wifimgr apply ap US-LAB-01 diff           # Preview changes
wifimgr apply ap US-LAB-01                # Apply AP config to site
wifimgr apply wlan US-LAB-01 diff         # Preview WLAN changes

# Search
wifimgr search wireless laptop-john       # Find wireless client
wifimgr search wired 5c:5b:35:8e:4c:f9   # Find wired client by MAC

# Target a specific API (multi-vendor)
wifimgr show api sites target mist-prod
wifimgr search wireless laptop target meraki-corp
```

## Documentation

- **[Documentation Index](docs/README.md)** - Full documentation table of contents
- **[User Guide](docs/user-guide.md)** - Complete command reference with examples
- **[Configuration Guide](docs/configuration.md)** - API setup, tokens, and configuration
- **[Multi-Vendor Setup](docs/multi-vendor/user-guide.md)** - Managing Mist and Meraki together
- **[CHANGELOG](CHANGELOG.md)** - Release notes and roadmap
- **[Known Issues](docs/known-issues.md)** - Current limitations and workarounds

## Development

```bash
make build          # Build binary
make test           # Run tests
make test-coverage  # Generate coverage report
```

## License

Apache License 2.0

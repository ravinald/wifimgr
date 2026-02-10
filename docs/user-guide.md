# wifimgr User Guide

Complete reference for configuring and using wifimgr.

---

## Table of Contents

- [Positional Arguments](#positional-arguments)
- [Commands](#commands)
  - [show](#show)
  - [apply](#apply)
  - [import](#import)
  - [search](#search)
  - [refresh](#refresh)
  - [init](#init)
  - [set](#set)
  - [encrypt](#encrypt)
- [Site Configuration](#site-configuration)
  - [Structure](#structure)
  - [AP Configuration](#ap-configuration)
  - [Switch Configuration](#switch-configuration)
  - [Gateway Configuration](#gateway-configuration)
  - [Device API Inheritance](#device-api-inheritance)
  - [Vendor-Specific Configuration](#vendor-specific-configuration)
- [Display Settings](#display-settings)
- [Troubleshooting](#troubleshooting)

For API setup, tokens, file paths, managed keys, and other configuration topics, see the **[Configuration Guide](configuration.md)**.

---

# Positional Arguments

wifimgr uses Junos-style positional keywords instead of flags for most options. These are placed after the primary command arguments:

| Keyword          | Description                      | Example                                   |
|------------------|----------------------------------|-------------------------------------------|
| `target <label>` | Scope to a specific API          | `wifimgr show api sites target mist-prod` |
| `diff`           | Preview changes without applying | `wifimgr apply ap US-LAB-01 diff`         |
| `site <name>`    | Filter by site                   | `wifimgr show api ap site US-LAB-01`      |
| `json` / `csv`   | Output format                    | `wifimgr show api ap json`                |
| `all`            | Show all fields (JSON only)      | `wifimgr show api ap json all`            |
| `no-resolve`     | Show raw IDs instead of names    | `wifimgr show api ap no-resolve`          |
| `force`          | Bypass confirmation prompts      | `wifimgr search wireless laptop force`    |
| `save`           | Write output to file             | `wifimgr import api site US-LAB-01 save`  |
| `split`          | Split output into separate files | `wifimgr import api site US-LAB-01 split` |
| `refresh-api`    | Refresh cache before operation   | `wifimgr apply ap US-LAB-01 refresh-api`  |

Positional keywords do NOT use `--` prefix. Hyphens within keywords (e.g., `no-resolve`, `refresh-api`) are fine.

---

# Commands

## show

Display data from API cache or local config files.

### Standard Usage

```bash
wifimgr show api sites                    # All sites
wifimgr show api ap                       # All APs
wifimgr show api ap site US-LAB-01        # APs at specific site
wifimgr show api ap Lobby-AP json         # Single AP as JSON
```

### Data Sources

| Command          | Source            | Description                     |
|------------------|-------------------|---------------------------------|
| `show api`       | API cache         | Current state from vendor API   |
| `show intent`    | Site config files | Desired state from local config |
| `show inventory` | Inventory cache   | Device inventory                |

### Common Recipes

```bash
# Filter by site
wifimgr show api ap site US-LAB-01
wifimgr show intent ap site US-LAB-01

# Output formats
wifimgr show api ap json                  # JSON array
wifimgr show api ap csv                   # CSV for spreadsheets
wifimgr show api ap Lobby-AP json all     # Full JSON with all fields

# Target a specific API (multi-vendor)
wifimgr show api sites target mist-prod

# Show WLANs
wifimgr show api wlans
wifimgr show api wlans site US-LAB-01

# Show device profiles
wifimgr show api deviceprofiles
```

### Positional Arguments

All `show` commands accept these optional arguments in order:

| Position   | Argument       | Description                    |
|------------|----------------|--------------------------------|
| 1          | `<filter>`     | Device name or MAC to filter   |
| 2          | `site <name>`  | Filter by site                 |
| 3          | `json` / `csv` | Output format                  |
| 4          | `all`          | Include all fields (JSON only) |
| 5          | `no-resolve`   | Show IDs instead of names      |

## apply

Push configuration from site config files to the API.

### Standard Usage

```bash
wifimgr apply ap US-LAB-01 diff      # Preview changes
wifimgr apply ap US-LAB-01                # Apply AP config
```

Always run with `diff` first to preview what will change.

### Device Types

```bash
wifimgr apply ap <site-name>              # Access points
wifimgr apply switch <site-name>          # Switches
wifimgr apply gateway <site-name>         # Gateways
wifimgr apply all <site-name>             # All device types
```

### Common Recipes

```bash
# Preview changes (always do this first)
wifimgr apply ap US-LAB-01 diff

# Apply with verbose logging
wifimgr -dd apply ap US-LAB-01

# Apply site config (creates site if needed)
wifimgr apply site US-LAB-01
```

### Backup and Rollback

Apply creates automatic backups before making changes.

**Backup Location:**

Backups are stored in `~/.local/state/wifimgr/backups/` (following XDG State directory).

**Backup Naming:**

Backups use a rotation scheme with serial numbers:
- `<config-filename>.json.0` — most recent backup
- `<config-filename>.json.1` — second most recent
- Higher numbers are older backups

When a new backup is created, existing backups rotate (0→1, 1→2, etc.) up to the configured limit.

**Configuration:**

| Setting                 | Default   | Description                               |
|-------------------------|-----------|-------------------------------------------|
| `files.config_backups`  | 5         | Maximum number of backup copies to retain |
| `backup.retention_days` | 30        | Age limit for `cleanup-backups` command   |

**Commands:**

```bash
# List available backups
wifimgr apply list-backups US-LAB-01

# Rollback to previous state (uses most recent backup)
wifimgr apply rollback US-LAB-01

# Rollback to specific backup
wifimgr apply rollback US-LAB-01 us-lab-01.json.2

# Cleanup old backups (removes backups older than N days)
wifimgr apply cleanup-backups --days 30
```

## import

Bootstrap local config files from current API state.

### Standard Usage

```bash
wifimgr import api site US-LAB-01         # Preview to STDOUT
wifimgr import api site US-LAB-01 save    # Save to file
```

This is useful when taking over management of an existing site. Import the current state, then make changes to the local config and apply them.

### Scope Options

```bash
wifimgr import api site US-LAB-01 full           # Everything (default)
wifimgr import api site US-LAB-01 type wlans    # Only WLANs
wifimgr import api site US-LAB-01 type profiles # Only site-specific profiles
wifimgr import api site US-LAB-01 type ap       # Only access points
wifimgr import api site US-LAB-01 type switch   # Only switches
wifimgr import api site US-LAB-01 type gateway  # Only gateways
```

### Common Recipes

```bash
# Preview and pipe to jq
wifimgr import api site US-LAB-01 | jq '.config'

# Compare local config with API state
wifimgr import api site US-LAB-01 compare

# Include secrets (PSK, RADIUS) - redacted by default
wifimgr import api site US-LAB-01 secrets save

# Import from specific API
wifimgr import api site US-LAB-01 target mist-prod save
```

### Import from PDF

Import AP radio config from RF planning PDFs:

```bash
wifimgr import file plan.pdf site US-LAB-01
```

The PDF parser extracts channel, power, and bandwidth settings and updates the site config file.

## search

Find devices connected to network infrastructure (both wireless and wired clients). Searches by hostname, MAC address, or partial match across Mist and Meraki networks.

### Standard Usage

```bash
wifimgr search wired laptop-john          # Search wired clients
wifimgr search wireless 5c:5b:35          # Search wireless by partial MAC
```

### Search Types

**Wireless clients:**
```bash
wifimgr search wireless john              # Hostname search
wifimgr search wireless aa:bb:cc          # MAC address search
```

**Wired clients:**
```bash
wifimgr search wired desktop-jane         # Hostname search
wifimgr search wired 5c:5b:35:8e:4c:f9    # Full MAC search
```

### Common Recipes

```bash
# Scope to specific site (fast - single API call)
wifimgr search wired laptop site US-LAB-01
wifimgr search wireless phone site US-LAB-01

# Output as JSON
wifimgr search wireless phone json

# Bypass confirmation for expensive searches (multi-site)
wifimgr search wired laptop force
```

### Cost Estimation and Confirmations

Searches without a site filter may require multiple API calls. The command estimates the cost and prompts for confirmation:

**Mist:** Text/hostname searches across networks require multiple API calls (one per network). MAC address searches are optimized to single API call.

**Meraki:** Similar cost estimation - MAC searches are org-wide (single call), hostname searches scan all networks (one call per network).

Use the `force` argument to bypass confirmation without manually approving each search:

```bash
wifimgr search wireless laptop force      # Bypass expensive search warning
```

### Multi-Vendor Search

Use `target <label>` to scope searches to a specific API:

```bash
wifimgr search wireless john target mist-prod
wifimgr search wired laptop target meraki-corp
```

## refresh

Sync local cache with API data. The cache system tracks age via `LastRefresh` timestamp and respects the `cache_ttl` setting to determine staleness.

### Standard Usage

```bash
wifimgr refresh cache                     # Refresh all cache data
```

Run this after making changes outside wifimgr, or when cache data is stale.

### Checking Cache Status

View cache age and status for all configured APIs:

```bash
wifimgr show api status
```

Output example:
```
API          Status    LastRefresh          Age      Stale
mist-prod    ok        2024-01-27T14:30:00  2h15m    false
meraki-corp  stale     2024-01-26T08:15:00  30h10m   true
```

**Status values:**
- `ok` - Cache exists and is fresh (within TTL)
- `stale` - Cache exists but exceeded TTL (should refresh)
- `corrupted` - Cache file is invalid
- `missing` - No cache file found

### Common Recipes

```bash
# Refresh specific API
wifimgr refresh cache mist-prod

# Refresh all APIs
wifimgr refresh cache

# Check cache status before operations
wifimgr show api status
```

### Cache Data Types

| Type                | Description               |
|---------------------|---------------------------|
| `sites`             | Site information          |
| `inventory-ap`      | AP inventory              |
| `inventory-switch`  | Switch inventory          |
| `inventory-gateway` | Gateway inventory         |
| `deviceprofiles`    | Device profiles           |
| `wlans`             | WLAN configurations       |
| `deviceconfigs`     | Per-device configurations |

## init

Create skeleton configuration files.

### Standard Usage

```bash
wifimgr init site US-LAB-01 api mist-prod
```

Creates `./config/US-LAB-01.json` with empty site config and device sections, and registers it in `wifimgr-config.json`.

### Common Recipes

```bash
# Custom filename
wifimgr init site US-LAB-01 api mist-prod file sites/us-lab.json

# Creates ./config/sites/us-lab.json
```

## set

Interactive commands for device management.

```bash
# Assign APs to site interactively
wifimgr set ap site US-LAB-01
```

## encrypt

Interactively encrypt secrets for use in configuration files. All input is hidden (terminal echo disabled) to prevent secrets from appearing on screen or in shell history.

> **Note:** This command works without any configuration file or API credentials. You can run it immediately after installing wifimgr.

```bash
# Encrypt a generic secret (API token, RADIUS secret, etc.)
wifimgr encrypt

# Encrypt a WiFi PSK with validation (8-63 printable ASCII chars)
wifimgr encrypt psk
```

**Workflow:**
1. Prompts for the secret value (hidden input)
2. Prompts to confirm the secret (hidden input)
3. Prompts for encryption password (min 8 chars, hidden input)
4. Prompts to confirm password (hidden input)
5. Outputs the encrypted value with `enc:` prefix

**Example output:**
```
enc:U2FsdGVkX1+abc123def456...
```

The encrypted value can be pasted directly into configuration files for:
- WLAN PSK passwords
- RADIUS shared secrets
- API tokens
- Any other sensitive values

When the application reads encrypted values, it will prompt for the decryption password unless `WIFIMGR_PASSWORD` is set.

**Non-interactive decryption:**

For CI/CD or scripts, set the password via environment variable or `.env.wifimgr`:
```bash
export WIFIMGR_PASSWORD="your-password"
# or add to .env.wifimgr:
# WIFIMGR_PASSWORD=your-password
```

**PSK Validation:**

When using `encrypt psk`, the secret is validated against IEEE 802.11i requirements:
- Length: 8-63 characters
- Characters: Printable ASCII only (codes 32-126)

---

# Site Configuration

Site config files define the desired state for devices at a site.

## Structure

```json
{
  "version": 1,
  "sites": {
    "US-LAB-01": {
      "api": "mist-prod",
      "site_config": {
        "name": "US-LAB-01",
        "address": "123 Main St, San Francisco, CA",
        "country_code": "US",
        "timezone": "America/Los_Angeles"
      },
      "devices": {
        "ap": {},
        "switch": {},
        "gateway": {}
      }
    }
  }
}
```

The `api` field specifies which API connection to use. Devices inherit this unless overridden.

## AP Configuration

Devices are keyed by MAC address (with or without colons):

```json
{
  "devices": {
    "ap": {
      "5c:5b:35:8e:4c:f9": {
        "name": "Lobby-AP-01",
        "radio_config": {
          "band_24_usage": "24",
          "band_24": {
            "channel": 6,
            "power": 10,
            "bandwidth": 20
          },
          "band_5": {
            "channel": 36,
            "power": 15,
            "bandwidth": 40
          },
          "band_6": {
            "disabled": true
          }
        }
      }
    }
  }
}
```

### Radio Config

| Field           | Type   | Description                |
|-----------------|--------|----------------------------|
| `band_24_usage` | string | `"24"`, `"5"`, or `"auto"` |
| `band_24`       | object | 2.4 GHz radio settings     |
| `band_5`        | object | 5 GHz radio settings       |
| `band_6`        | object | 6 GHz radio settings       |

**Per-band settings:**

| Field               | Type   | Description                    |
|---------------------|--------|--------------------------------|
| `disabled`          | bool   | Disable this band              |
| `channel`           | int    | Channel number (0 = auto)      |
| `power`             | int    | Transmit power in dBm          |
| `bandwidth`         | int    | Channel width: 20, 40, 80, 160 |
| `allow_rrm_disable` | bool   | Allow RRM to disable band      |

### IP Config

```json
{
  "ip_config": {
    "type": "dhcp",
    "vlan_id": 100
  }
}
```

Or static:
```json
{
  "ip_config": {
    "type": "static",
    "ip": "10.0.1.10",
    "netmask": "255.255.255.0",
    "gateway": "10.0.1.1",
    "dns": ["8.8.8.8"]
  }
}
```

### Other AP Fields

| Field                | Type   | Description                                |
|----------------------|--------|--------------------------------------------|
| `name`               | string | Device name                                |
| `notes`              | string | Admin notes                                |
| `tags`               | array  | Tags for grouping                          |
| `ble_config`         | object | Bluetooth/beacon settings                  |
| `mesh`               | object | Mesh networking                            |
| `led`                | object | LED configuration                          |
| `poe_passthrough`    | bool   | Enable PoE passthrough                     |
| `deviceprofile_id`   | string | Device profile UUID                        |
| `deviceprofile_name` | string | Device profile by name (alternative to ID) |

## Switch Configuration

```json
{
  "devices": {
    "switch": {
      "98:86:8b:b5:f7:80": {
        "name": "IDF-SW-01",
        "notes": "First floor IDF",
        "role": "access",
        "port_config": {
          "ge-0/0/0": {
            "usage": "trunk",
            "port_network": "default"
          }
        }
      }
    }
  }
}
```

## Gateway Configuration

```json
{
  "devices": {
    "gateway": {
      "e4:f2:7c:29:52:8e": {
        "name": "GW-01",
        "notes": "Primary gateway",
        "tags": ["primary"]
      }
    }
  }
}
```

## Device API Inheritance

Devices inherit the `api` field from the site. Override at device level for mixed-vendor sites:

```json
{
  "sites": {
    "US-MIXED-SITE": {
      "api": "mist-prod",
      "devices": {
        "ap": {
          "5c:5b:35:8e:4c:f9": {
            "name": "AP-MIST-01"
          },
          "ac:69:cf:04:62:a0": {
            "name": "AP-MERAKI-01",
            "api": "meraki-corp"
          }
        }
      }
    }
  }
}
```

## Vendor-Specific Configuration

wifimgr provides a **vendor-agnostic configuration schema** that works across Mist and Meraki. However, some features are unique to a particular vendor. Use vendor extension blocks (`mist:` or `meraki:`) for these vendor-specific settings.

### Design Philosophy

- **Common schema uses Mist nomenclature** as the standard
- **Vendor blocks handle vendor-specific features** not expressible in common schema
- **Field mappings are automatic** - the system transforms field names between vendors
- See [Field Mappings Documentation](docs/field-mappings.md) for complete mapping tables

### When to Use Vendor Blocks

Use vendor extension blocks when you need:

**Mist-specific features:**
- Floor plan positioning: `map_id`, `x`, `y`, `orientation`, `height`
- Hardware control: `disable_eth1`, `disable_eth2`, `disable_eth3`
- BLE/beacon configuration: Full `ble_config` block
- Mesh networking: Full `mesh` block
- RRM settings: `scanning_enabled`, `indoor_use`, `allow_rrm_disable`
- USB configuration and site variables

**Meraki-specific features:**
- RF profiles: `rf_profile_id`
- Floor plan IDs: `floor_plan_id`
- GPS coordinates: `lat`, `lng`
- Per-band settings: `min_bitrate`, `rxsop`
- Band steering: `band_selection`

### Syntax

Add a `mist` or `meraki` block alongside common fields. The block name must match the vendor of the device's API.

```json
{
  "devices": {
    "ap": {
      "5c:5b:35:8e:4c:f9": {
        "name": "Lobby-AP-01",
        "radio_config": {
          "band_24": { "channel": 6, "power": 10 },
          "band_5": { "channel": 36, "power": 15 }
        },
        "mist": {
          "map_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
          "x": 125.5,
          "y": 340.2,
          "orientation": 90,
          "disable_eth1": true
        }
      }
    }
  }
}
```

### Mist-Specific Configuration

#### Common Mist-Only Fields

| Field                 | Type     | Description                                      | Usage                      |
|-----------------------|----------|--------------------------------------------------|----------------------------|
| `map_id`              | string   | Floor plan UUID for positioning                  | Top-level `mist:` block    |
| `x`, `y`              | float    | Position on floor plan (pixels)                  | Top-level `mist:` block    |
| `orientation`         | int      | AP orientation in degrees (0-359)                | Top-level `mist:` block    |
| `height`              | float    | Mounting height in meters                        | Top-level `mist:` block    |
| `disable_eth1`        | bool     | Disable secondary ethernet port                  | Top-level `mist:` block    |
| `disable_eth2`        | bool     | Disable tertiary ethernet port                   | Top-level `mist:` block    |
| `usb_config`          | object   | USB port configuration                           | Top-level `mist:` block    |
| `vars`                | object   | Site variables for templates                     | Top-level `mist:` block    |
| `aeroscout`           | object   | AeroScout location integration                   | Top-level `mist:` block    |
| `scanning_enabled`    | bool     | Enable RF scanning for location/analytics        | `radio_config.mist:` block |
| `indoor_use`          | bool     | Indoor vs. outdoor operation mode                | `radio_config.mist:` block |

#### Example: Mist AP with Floor Plan

```json
{
  "5c5b358e4cf9": {
    "name": "Conference-Room-AP",
    "radio_config": {
      "band_24": { "channel": 1, "power": 8, "bandwidth": 20 },
      "band_5": { "channel": 44, "power": 12, "bandwidth": 80 },
      "mist": {
        "scanning_enabled": true,
        "indoor_use": true
      }
    },
    "led": { "enabled": false },
    "mist": {
      "map_id": "floor-2-uuid",
      "x": 450,
      "y": 280,
      "height": 3.0,
      "orientation": 90,
      "disable_eth1": true
    }
  }
}
```

### Meraki-Specific Configuration

#### Common Meraki-Only Fields

| Field                | Type     | Description                                      | Usage                        |
|----------------------|----------|--------------------------------------------------|------------------------------|
| `rf_profile_id`      | string   | RF profile identifier                            | `radio_config.meraki:` block |
| `floor_plan_id`      | string   | Floor plan identifier for positioning            | Top-level `meraki:` block    |
| `lat`, `lng`         | float    | GPS coordinates                                  | Top-level `meraki:` block    |
| `min_bitrate`        | int      | Minimum client bitrate (Mbps)                    | Per-band `meraki:` block     |
| `rxsop`              | int      | Receiver Start of Packet threshold               | Per-band `meraki:` block     |
| `band_selection`     | string   | Band steering mode                               | Top-level `meraki:` block    |
| `per_ssid_settings`  | object   | Per-SSID radio settings                          | `radio_config.meraki:` block |

#### Example: Meraki AP with RF Profile

```json
{
  "ac69cf0462a0": {
    "name": "Lobby-AP",
    "api": "meraki-corp",
    "radio_config": {
      "band_24": {
        "channel": 6,
        "power": 10,
        "bandwidth": 20,
        "meraki": {
          "min_bitrate": 12
        }
      },
      "band_5": {
        "channel": 149,
        "power": 14,
        "bandwidth": 40,
        "meraki": {
          "min_bitrate": 24,
          "rxsop": -82
        }
      },
      "meraki": {
        "rf_profile_id": "12345"
      }
    },
    "meraki": {
      "floor_plan_id": "L_123456789",
      "lat": 37.7749,
      "lng": -122.4194
    }
  }
}
```

### Common Use Cases

#### Use Case 1: Floor Plan Positioning

**Mist:**
```json
{
  "mist": {
    "map_id": "floor-plan-uuid",
    "x": 250,
    "y": 180,
    "orientation": 90,
    "height": 3.0
  }
}
```

**Meraki:**
```json
{
  "meraki": {
    "floor_plan_id": "L_123456789",
    "lat": 37.7749,
    "lng": -122.4194
  }
}
```

#### Use Case 2: RF Profile Assignment

**Mist:**
```json
{
  "deviceprofile_id": "profile-uuid"
}
```
Or use name resolution:
```json
{
  "deviceprofile_name": "Corporate-Office-Profile"
}
```

**Meraki:**
```json
{
  "radio_config": {
    "meraki": {
      "rf_profile_id": "12345"
    }
  }
}
```

#### Use Case 3: Advanced Radio Settings

**Mist:**
```json
{
  "radio_config": {
    "band_24_usage": "24",
    "band_5": {
      "channel": 36,
      "power": 15,
      "bandwidth": 80,
      "allow_rrm_disable": false
    },
    "mist": {
      "scanning_enabled": true,
      "indoor_use": true
    }
  }
}
```

**Meraki:**
```json
{
  "radio_config": {
    "band_5": {
      "channel": 36,
      "power": 15,
      "bandwidth": 80,
      "meraki": {
        "min_bitrate": 24,
        "rxsop": -82
      }
    }
  }
}
```

### Field Mapping Reference

**Automatic field transformations** (no vendor block needed):

| Common Schema    | Mist API         | Meraki API               | Notes                 |
|------------------|------------------|--------------------------|-----------------------|
| `power`          | `power`          | `targetPower`            | Transmit power in dBm |
| `bandwidth`      | `bandwidth`      | `channelWidth`           | 20, 40, 80, 160 MHz   |
| `led.enabled`    | `led.enabled`    | `ledLightsOn`            | LED status light      |
| `ip_config.type` | `ip_config.type` | `assignmentMode`         | "dhcp" or "static"    |
| `tags` (array)   | `tags` (array)   | `tags` (space-separated) | Tag format conversion |

See [Field Mappings Documentation](docs/field-mappings.md) for the complete mapping table.

### Mixed Common and Vendor Settings

You can combine common schema fields with vendor-specific extensions:

```json
{
  "devices": {
    "ap": {
      "5c5b358e4cf9": {
        "name": "AP-01",
        "notes": "Main office access point",
        "tags": ["office", "wifi6"],
        "ip_config": {
          "type": "dhcp",
          "vlan_id": 100
        },
        "radio_config": {
          "band_24": { "channel": 6, "power": 10, "bandwidth": 20 },
          "band_5": { "channel": 36, "power": 15, "bandwidth": 80 }
        },
        "led": { "enabled": false },
        "mist": {
          "map_id": "floor-2-uuid",
          "x": 250,
          "y": 180,
          "disable_eth1": true
        }
      }
    }
  }
}
```

### Nested Vendor Blocks

Vendor blocks can be placed at multiple levels for granular control:

**Top-level device settings:**
```json
{
  "name": "AP-01",
  "mist": {
    "map_id": "uuid",
    "x": 100,
    "y": 200
  }
}
```

**Radio configuration settings:**
```json
{
  "radio_config": {
    "band_5": { "channel": 36, "power": 15 },
    "mist": {
      "scanning_enabled": true,
      "indoor_use": true
    }
  }
}
```

**Per-band settings:**
```json
{
  "radio_config": {
    "band_5": {
      "channel": 36,
      "power": 15,
      "meraki": {
        "min_bitrate": 24,
        "rxsop": -82
      }
    }
  }
}
```

### Validation and Error Handling

**Validation rules:**
- Vendor block name must match device's target API vendor (check `api` field)
- Unknown fields in vendor blocks pass through to API as-is (enables forward compatibility)
- Common fields always take precedence over vendor-specific equivalents
- Mutually exclusive fields (e.g., `map_id` vs `map_name`) trigger validation errors

**Check configuration before applying:**
```bash
# diff shows what will be sent to the API
wifimgr apply ap US-LAB-01 diff

# Debug mode shows field transformations
wifimgr -ddd apply ap US-LAB-01 diff
```

**Common validation errors:**
- Using `mist:` block on Meraki device or vice versa
- Putting vendor-specific field in common schema area
- Conflicting mutually exclusive fields

### Troubleshooting

**Problem**: Configuration doesn't apply as expected

**Solutions:**
1. Check field is in correct vendor block: [Field Mappings](docs/field-mappings.md)
2. Verify vendor block matches device API (check `api` field)
3. Use debug mode to see transformations: `wifimgr -ddd apply ap SITE diff`
4. Check logs for validation warnings

**Problem**: Field not recognized by API

**Solutions:**
1. Verify field name matches vendor's API documentation
2. Check if field requires specific vendor block placement
3. Use `import` command to see how wifimgr reads current API state:
   ```bash
   wifimgr import api site US-LAB-01
   ```

**Problem**: Values not applying correctly

**Solutions:**
1. Check field type matches API requirements (string vs int vs bool)
2. Verify value is in valid range for that field
3. Enable trace logging to see full API request/response:
   ```bash
   wifimgr -ddd apply ap US-LAB-01
   ```

---

# Display Settings

Customize the table output for each command.

## Column Configuration

```json
{
  "display": {
    "commands": {
      "show.api.ap": {
        "format": "table",
        "title": "AP Devices",
        "fields": [
          { "field": "status", "title": "Status", "width": 6 },
          { "field": "name", "title": "Name", "width": -1 },
          { "field": "mac", "title": "MAC", "width": -1 },
          { "field": "model", "title": "Model", "width": -1 },
          { "field": "site_name", "title": "Site", "width": -1 }
        ]
      }
    }
  }
}
```

### Width Options

| Width   | Behavior                     |
|---------|------------------------------|
| `-1`    | Auto-size, never truncate    |
| `0`     | Auto-size, scale to terminal |
| `> 0`   | Fixed width                  |

### Command Paths

| Path                | Command             |
|---------------------|---------------------|
| `show.api.sites`    | `show api sites`    |
| `show.api.ap`       | `show api ap`       |
| `show.inventory.ap` | `show inventory ap` |
| `show.intent.sites` | `show intent sites` |

## JSON Colors

```json
{
  "display": {
    "jsoncolor": {
      "key": { "ansi256": "21" },
      "string": { "ansi256": "46" },
      "number": { "ansi256": "51" },
      "bool": { "ansi256": "15" },
      "null": { "ansi256": "244" }
    }
  }
}
```

---

# Troubleshooting

## Debug Logging

Use `-d`, `-dd`, or `-ddd` for increasing verbosity:

| Flag   | Level   | Shows                    |
|--------|---------|--------------------------|
| `-d`   | info    | Basic operation info     |
| `-dd`  | debug   | API requests/responses   |
| `-ddd` | trace   | Full API response bodies |

```bash
wifimgr -ddd apply ap US-LAB-01 diff
```

Logs go to `~/.local/state/wifimgr/wifimgr.log` by default. Enable stdout:
```json
{
  "logging": {
    "enable": true,
    "level": "debug",
    "stdout": true
  }
}
```

## Cache Issues

If data seems stale or incorrect:

```bash
wifimgr refresh cache
```

## API Errors

Common HTTP status codes:

| Code   | Meaning                                   |
|--------|-------------------------------------------|
| 401    | Invalid or expired token                  |
| 403    | Token lacks permission for this org       |
| 404    | Resource not found (check site/device ID) |
| 429    | Rate limited - wait and retry             |

## Device Not Found

If apply reports a device not found:
1. Verify the MAC address is correct
2. Check the device is in inventory: `wifimgr show inventory ap`
3. Refresh cache: `wifimgr refresh cache`

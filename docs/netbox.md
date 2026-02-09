# NetBox Integration

## Overview

The NetBox integration provides bidirectional synchronization between wifimgr and NetBox DCIM:

1. **Export to NetBox**: Export Access Point inventory from wifimgr to NetBox, creating or updating device records with interfaces and IP addresses.
2. **Sync from NetBox**: Use NetBox as the source of truth for device configuration (device names and site assignments).

This enables centralized infrastructure management where NetBox serves as your authoritative IPAM/DCIM system while wifimgr manages the actual device configuration on Mist or Meraki platforms.

### Export Features

The export functionality creates or updates NetBox device records for **Access Points only**. For each AP, the export:

- Creates the device record in NetBox with basic information (name, model, serial)
- Creates the primary management interface (eth0) with MAC address
- Assigns the management IP address if available

**Note**: Switches and gateways are not exported. Only Access Points are synchronized to NetBox.

## Prerequisites

Before using the NetBox export feature, ensure the following:

### NetBox Setup Requirements

1. **NetBox Instance**: You have access to a NetBox instance with API enabled
2. **Sites**: All sites referenced in your wifimgr inventory must exist in NetBox
   - NetBox uses site slugs for identification (lowercase, dashes instead of spaces)
   - You can create sites via the NetBox web UI or API
3. **Device Types**: Device types must exist for each hardware model you're exporting
   - Examples: "cisco-mr46e", "juniper-ap43", "arista-7050-qx"
   - Device types are matched using slug identifiers
4. **Device Roles**: Device roles must exist that correspond to your device types
   - Examples: "wireless-ap", "access-switch", "router"
   - Default mappings are provided but can be customized

### Device Data Requirements

Each AP device in your wifimgr inventory must have:

- **Name** or MAC address (at least one)
- **Model** (required for device type matching)
- **Device Type** (must be "ap")
- **Site Assignment** (required for site matching)
- **MAC Address** (for interface creation)

Devices missing any of these required fields will be skipped during export with an error message indicating what's missing. Non-AP devices (switches, gateways) are automatically filtered out and not exported.

### Cache Requirements

Run cache refresh before exporting to ensure you have the latest inventory data:

```bash
wifimgr cache refresh
```

## Configuration

NetBox configuration can be provided through three methods with a clear priority order:

### Priority Order (Highest to Lowest)

1. **Environment Variables** - Immediate effect, useful for automation
2. **Environment File** (`~/.env.netbox`) - User-level configuration
3. **Configuration File** - Application-wide defaults (wifimgr-config.json)

This allows you to use configuration file defaults while overriding with environment variables for specific workflows.

### Method 1: Environment Variables

Set these variables in your shell environment or deployment configuration:

```bash
export NETBOX_API_URL="https://netbox.example.com"
export NETBOX_API_KEY="your-api-key-here"
export NETBOX_SSL_VERIFY="true"
```

For shells that don't persist variables, you can set them per-command:

```bash
NETBOX_API_URL=https://netbox.example.com NETBOX_API_KEY=your-key wifimgr export netbox all
```

### Method 2: Environment File

Create a file named `.env.netbox` in your home directory:

```bash
# ~/.env.netbox
NETBOX_API_URL=https://netbox.example.com
NETBOX_API_KEY=your-api-key-here
NETBOX_SSL_VERIFY=true
```

File format:
- One variable per line
- Format: `KEY=value`
- Lines starting with `#` are treated as comments
- Values can be quoted with double quotes: `KEY="value with spaces"`
- Empty lines are ignored

Example with quoting:

```bash
# ~/.env.netbox
NETBOX_API_URL="https://netbox.example.com"
NETBOX_API_KEY="abc123def456"
NETBOX_SSL_VERIFY=true
```

### Method 3: Configuration File

Add NetBox configuration to your wifimgr config file (typically `wifimgr-config.json`):

```json
{
  "netbox": {
    "url": "https://netbox.example.com",
    "credentials": {
      "api_key": "your-api-key-here"
    },
    "ssl_verify": true,
    "mappings": {
      "tag": "wifimgr-managed",
      "device_types": {
        "MR46E": { "slug": "cisco-mr46e" },
        "AP43": { "slug": "juniper-ap43", "role": "special-ap" }
      },
      "default_roles": {
        "ap": "wireless-ap",
        "switch": "access-switch",
        "gateway": "router"
      },
      "site_overrides": {
        "US-LAB-01": "us-lab",
        "EU-PROD": "eu-production"
      }
    }
  }
}
```

#### Configuration Fields

- **`url`** (string, required): NetBox API endpoint (e.g., `https://netbox.example.com`)
- **`credentials.api_key`** (string, required): Your NetBox API token for authentication. Can be encrypted with `enc:` prefix.
- **`ssl_verify`** (boolean, optional): Whether to verify SSL certificates. Default: `true`
- **`settings_source`** (string, optional): Source for device configuration metadata. Valid values: `"api"` (default) or `"netbox"`. See [Reverse Sync](#reverse-sync-netbox-as-configuration-source) for details.
- **`mappings.tag`** (string, optional): NetBox tag to apply to all devices and interfaces created by wifimgr. Useful for identifying objects managed by wifimgr.
- **`mappings.device_types`** (object, optional): Map device models to NetBox device type slugs with optional role overrides
- **`mappings.default_roles`** (object, optional): Map wifimgr device types to NetBox device role slugs (fallback when not specified at device type level)
- **`mappings.site_overrides`** (object, optional): Map wifimgr site names to NetBox site slugs

#### Encrypted API Keys

For added security, you can encrypt the API key using `wifimgr encrypt`:

```bash
wifimgr encrypt
# Enter the NetBox API key when prompted
# Copy the enc:... output
```

Then use the encrypted value in your configuration file:

```json
{
  "netbox": {
    "url": "https://netbox.example.com",
    "credentials": {
      "api_key": "enc:U2FsdGVkX1..."
    }
  }
}
```

When wifimgr detects the `enc:` prefix, it will decrypt the value using `WIFIMGR_PASSWORD` from the environment or `.env.wifimgr` file. If the password is not available, you'll be prompted interactively.

## Mappings

Mappings define how wifimgr device information is translated to NetBox identifiers. The following mapping types are supported:

### Tag Configuration

The `tag` configuration allows you to apply a specific NetBox tag to all devices and interfaces created by wifimgr. This is useful for:

- **Identifying wifimgr-managed objects** in NetBox
- **Filtering and reporting** on devices imported from wifimgr
- **Automation workflows** that need to distinguish wifimgr-created objects
- **Tracking data lineage** across your infrastructure management tools

**Configuration:**
```json
{
  "netbox": {
    "mappings": {
      "tag": "wifimgr-managed"
    }
  }
}
```

**Behavior:**
- If configured, the tag is automatically applied to all devices and interfaces created during export
- The tag must exist in NetBox before export (wifimgr does not create tags)
- If the tag is not configured (empty string or omitted), no tags are applied
- Tags are applied using the tag name (NetBox will resolve to the appropriate tag object)

**Example Use Cases:**

1. **Identify all wifimgr-managed devices:**
   - In NetBox UI: Filter devices by tag "wifimgr-managed"
   - Via API: Query devices with the wifimgr tag

2. **Separate manual vs automated management:**
   - Tag wifimgr imports with "wifimgr-managed"
   - Manually created devices have no tag or different tag
   - Prevent accidental deletion of manual entries

3. **Track data sources:**
   - Use different tags for different import sources
   - "wifimgr-mist" for Mist imports
   - "wifimgr-meraki" for Meraki imports

### Device Type Mappings

Maps wifimgr device models to NetBox device type slugs. This allows you to control how different hardware models map to NetBox's device type system. Each mapping can optionally specify a device role override.

**Configuration:**
```json
{
  "netbox": {
    "mappings": {
      "device_types": {
        "MR46E": { "slug": "cisco-mr46e" },
        "AP43": { "slug": "juniper-ap43", "role": "special-ap" },
        "MR45": { "slug": "cisco-mr45" },
        "MS120-24": { "slug": "cisco-ms120-24", "role": "access-switch" }
      }
    }
  }
}
```

**Fields:**
- **`slug`** (required): The NetBox device type slug
- **`role`** (optional): Override the device role for this specific device type. If not specified, uses the `default_roles` mapping or hardcoded defaults.

**Role Priority:**

When determining which role to use for a device, wifimgr follows this priority order:

1. **Per-Device Override**: Role specified in the device's `netbox.device_role` field in inventory
2. **Device Type Role Override**: Role specified in `device_types[model].role`
3. **Default Roles Mapping**: Role from `default_roles[deviceType]`
4. **Hardcoded Defaults**: Built-in defaults (ap → wireless-ap, switch → access-switch, gateway → router)

**Per-Device NetBox Configuration:**

Individual devices can have NetBox-specific settings in their inventory entry:

```json
{
  "devices": {
    "aabbccddeeff": {
      "name": "AP-Special",
      "model": "MR55",
      "type": "ap",
      "netbox": {
        "device_role": "custom-role-for-this-device",
        "interfaces": {
          "eth0": {
            "name": "mgmt0",
            "type": "10gbase-t"
          },
          "radio1": {
            "name": "wlan5g",
            "type": "ieee802.11ax"
          }
        }
      }
    }
  }
}
```

This allows overriding the device role and interface mappings on a per-device basis without changing the model-level or global defaults.

**Per-Device Interface Override Fields:**
- **`device_role`** (string, optional): Override the device role for this specific device
- **`interfaces`** (object, optional): Map of internal interface ID to interface settings
  - Keys: `eth0`, `eth1`, `radio0`, `radio1`, `radio2`
  - Values: Object with `name` and/or `type` fields

**Partial Overrides:**

You can provide partial interface overrides. For example, to change only the name while keeping the default type:

```json
{
  "netbox": {
    "interfaces": {
      "eth0": {
        "name": "management"
      }
    }
  }
}
```

The type will be inherited from global config or defaults.

**Pattern Matching:**

You can use wildcard patterns for prefix matching:

```json
{
  "netbox": {
    "mappings": {
      "device_types": {
        "MR*": { "slug": "cisco-meraki" },
        "AP*": { "slug": "juniper-ap", "role": "juniper-ap-role" },
        "MX*": { "slug": "cisco-mx" }
      }
    }
  }
}
```

When a device model matches a pattern (with `*` suffix), that mapping is used. Exact matches take precedence over pattern matches.

**Default Behavior:**

If a device model has no mapping, wifimgr converts it to slug format (lowercase, spaces and special characters converted to dashes):

- `MR46E` → `mr46e`
- `Juniper AP43` → `juniper-ap43`

**Backward Compatibility:**

The old format (simple string values) is still supported but deprecated:

```json
{
  "netbox": {
    "mappings": {
      "device_types": {
        "MR46E": "cisco-mr46e"
      }
    }
  }
}
```

This will be automatically converted to the new format. A warning will be logged recommending migration to the new structure.

### Default Role Mappings

Maps wifimgr device types to NetBox device role slugs. These are fallback roles used when a device type doesn't have a role override specified in its `device_types` mapping.

**Configuration:**
```json
{
  "netbox": {
    "mappings": {
      "default_roles": {
        "ap": "wireless-ap"
      }
    }
  }
}
```

**Note**: Currently only Access Points (`ap`) are exported to NetBox. Switch and gateway export may be added in a future release.

**Default Mapping:**

If custom mappings aren't provided, this default is used:

```json
{
  "ap": "wireless-ap"
}
```

**Backward Compatibility:**

The old field name `device_roles` is still supported but deprecated. It will be automatically mapped to `default_roles` with a warning logged.

### Site Overrides

Maps wifimgr site names to NetBox site slugs. Use this when your wifimgr site names don't directly match NetBox site slugs.

**Configuration:**
```json
{
  "netbox": {
    "mappings": {
      "site_overrides": {
        "US-LAB-01": "us-lab",
        "EU-PROD": "eu-production",
        "APAC Region": "apac"
      }
    }
  }
}
```

**Default Behavior:**

If no override is specified, wifimgr converts the site name to slug format:

- `US-LAB-01` → `us-lab-01`
- `EU PROD` → `eu-prod`
- `APAC_Region` → `apac-region`

### Interface Mappings

Configures how device interfaces are named and typed in NetBox. This allows you to customize interface names and PHY types to match your NetBox conventions.

**Internal Interface Identifiers:**

wifimgr uses internal identifiers to refer to device interfaces:

| Internal ID | Description | Default Name | Default Type |
|-------------|-------------|--------------|--------------|
| `eth0` | Primary management interface | eth0 | 1000base-t |
| `eth1` | Secondary Ethernet interface | eth1 | 1000base-t |
| `radio0` | 2.4 GHz radio | wifi0 | ieee802.11n |
| `radio1` | 5 GHz radio | wifi1 | ieee802.11ac |
| `radio2` | 6 GHz radio | wifi2 | ieee802.11ax |

**Configuration:**
```json
{
  "netbox": {
    "mappings": {
      "interfaces": {
        "eth0": {
          "name": "mgmt0",
          "type": "1000base-t"
        },
        "eth1": {
          "name": "uplink0",
          "type": "10gbase-t"
        },
        "radio0": {
          "name": "wlan0",
          "type": "ieee802.11n"
        },
        "radio1": {
          "name": "wlan1",
          "type": "ieee802.11ac"
        },
        "radio2": {
          "name": "wlan2",
          "type": "ieee802.11ax"
        }
      }
    }
  }
}
```

**Fields:**
- **`name`** (string): The interface name as it appears in NetBox (e.g., "eth0", "mgmt0", "wlan1")
- **`type`** (string): The NetBox interface PHY type (see Valid Interface Types below)

**Valid Interface Types:**

The following interface types are supported:

| Type | Description |
|------|-------------|
| **Ethernet** | |
| `100base-tx` | 100BASE-TX (100 Mbps) |
| `1000base-t` | 1000BASE-T (1GE) |
| `2.5gbase-t` | 2.5GBASE-T (2.5GE) |
| `5gbase-t` | 5GBASE-T (5GE) |
| `10gbase-t` | 10GBASE-T (10GE) |
| **Wireless** | |
| `ieee802.11a` | IEEE 802.11a |
| `ieee802.11g` | IEEE 802.11b/g |
| `ieee802.11n` | IEEE 802.11n (Wi-Fi 4) |
| `ieee802.11ac` | IEEE 802.11ac (Wi-Fi 5) |
| `ieee802.11ax` | IEEE 802.11ax (Wi-Fi 6) |
| `ieee802.11be` | IEEE 802.11be (Wi-Fi 7) |
| **Other** | |
| `virtual` | Virtual interface |
| `other` | Other type |

**Interface Type Validation:**

wifimgr validates interface types before sending requests to NetBox. If an invalid type is specified, you'll receive a detailed error message with the list of valid types and a suggestion for similar valid types.

Example error:
```
Interface type 'wifi6' is not valid for device 'AP-LOBBY-01'

Valid types:
  - ieee802.11ax (IEEE 802.11ax / Wi-Fi 6)
  - ieee802.11ac (IEEE 802.11ac / Wi-Fi 5)
  - 1000base-t (1000BASE-T / 1GE)
  ...

Suggestion: Use 'ieee802.11ax' for Wi-Fi 6

Configure in netbox.mappings.interfaces in wifimgr-config.json
```

**Priority Order:**

Interface mappings are resolved in this order:

1. **Per-Device Override**: Interface settings in the device's `netbox.interfaces` field
2. **Global Config**: Interface mappings in `netbox.mappings.interfaces`
3. **Defaults**: Built-in default names and types

**Default Behavior:**

If no interface mappings are configured, wifimgr uses sensible defaults:

- Management interfaces (eth0, eth1) use `1000base-t` (Gigabit Ethernet)
- Radio interfaces use appropriate wireless types for each band
- Interface names follow common conventions (eth0, wifi0, wifi1, wifi2)

## Commands

### Export All APs

Export all Access Points from all sites to NetBox:

```bash
wifimgr export netbox all
```

**Note**: Only Access Points are exported. Switches and gateways are automatically excluded.

### Export APs from Specific Site

Export Access Points from a single site:

```bash
wifimgr export netbox site US-LAB-01
```

### Dry Run Mode

Preview what would be created/updated without making any changes to NetBox:

```bash
wifimgr export netbox all dry-run
wifimgr export netbox site US-LAB-01 dry-run
```

Dry run mode shows:

- Number of devices that would be created
- Number of devices that would be updated
- Devices that would be skipped due to missing dependencies
- Any validation errors that would prevent export

This is useful for testing your configuration and mapping before performing the actual export.

### Validate Mode

Run validation only to check if required NetBox dependencies exist without performing any export:

```bash
wifimgr export netbox all validate
wifimgr export netbox site US-LAB-01 validate
```

Validate mode:

- Fetches all sites, device types, and device roles from NetBox
- Checks if each device in your inventory has matching dependencies
- Reports missing sites, device types, and device roles
- Shows which devices would fail validation
- Performs no export operations

This helps you identify missing NetBox objects before attempting export.

## Validation

Before exporting, wifimgr validates that all required NetBox dependencies exist. This prevents failed exports and data inconsistency.

### Validation Requirements

For each device to be exported, the following must exist in NetBox:

1. **Site** - Matched by name or slug
   - Primary: Exact name match (case-insensitive)
   - Secondary: Slug match (name converted to slug format)

2. **Device Type** - Matched by slug
   - Uses mapping configuration or auto-converted model name
   - Example: "MR46E" → "cisco-mr46e" (from mapping) or "mr46e" (auto-converted)

3. **Device Role** - Matched by slug
   - Uses mapping configuration or device type default
   - Example: "ap" → "wireless-ap" (from mapping)

### Validation Errors

Devices are skipped if validation fails. Common reasons for validation failure:

- **Site not found**: "site 'US-LAB-01' not found in NetBox"
- **Device type not found**: "device type 'MR46E' (slug 'mr46e') not found in NetBox"
- **Device role not found**: "device role 'wireless-ap' (for device type 'ap') not found in NetBox"
- **Missing required fields**: Device lacks name/MAC, model, type, or site assignment

### Resolving Validation Errors

1. **Missing Site**: Create the site in NetBox with matching name or slug
2. **Missing Device Type**: Create the device type in NetBox with the expected slug
3. **Missing Device Role**: Create the device role in NetBox with the expected slug
4. **Mapping Issues**: Add or update mappings in configuration to match NetBox slugs
5. **Device Data**: Ensure devices have complete information in your wifimgr cache

## Error Handling

The export process distinguishes between recoverable and fatal errors.

### Non-Fatal Errors (Collected and Reported)

These errors affect individual devices but don't stop the export process:

- Device validation failure (missing site, type, or role)
- IP address conflicts in NetBox
- Interface creation failure
- Update operation failure on existing devices

Devices with non-fatal errors are skipped or partially processed. The export continues with remaining devices.

### Fatal Errors (Stop Export)

These errors stop the export process immediately:

- NetBox API connection failure
- Authentication failure (invalid API key)
- Configuration loading failure
- Cache not initialized

### Error Output

Failed devices are reported in the export summary:

```
Skipped Devices (missing dependencies):
  - AP-LOBBY-01: site 'US-TEMP-LAB' not found in NetBox
  - SW-IDF-02: device type 'MS120-24' not found in NetBox

Errors:
  - AP-CONF-01 [interface]: failed to create interface: 400 Bad Request
```

## Data Mapping

The following table shows how wifimgr inventory fields map to NetBox device fields:

| wifimgr Field | NetBox Field | Notes |
|---|---|---|
| Site name | Site ID | Matched by name or slug |
| Device type (ap/switch/gateway) | Device Role ID | Mapped via configuration |
| Model | Device Type ID | Mapped via slug conversion |
| Name | Device Name | Uses name or generates from MAC/serial |
| Serial | Device Serial Number | Preserved as-is if available |
| MAC address | Interface MAC Address | Creates primary interface with MAC |
| Status | Device Status | Set to "active" for new devices |

### Custom Fields

wifimgr adds custom fields to track data lineage:

- **`wifimgr_source_api`**: Source API (mist, meraki)
- **`wifimgr_source_vendor`**: Source vendor name
- **`wifimgr_vendor_id`**: Original vendor device ID

These fields can be used for reference or to identify devices in NetBox that originated from specific wifimgr imports.

### Interface Creation

When an AP device is created, wifimgr automatically creates a primary network interface with:

- **Interface Name**: `eth0` (configurable via interface mappings)
- **Interface Type**: `1000base-t` (configurable via interface mappings)
- **MAC Address**: Device's MAC address in uppercase colon format
- **Status**: Enabled

If the AP has a management IP address, it is assigned to this interface.

For radio interfaces (wifi0, wifi1, wifi2), wifimgr creates wireless interfaces based on the AP's radio configuration with appropriate interface types for each band. See [Interface Mappings](#interface-mappings) for customization options.

## Output and Results

### Export Summary

After export completes, you see a summary showing:

```
Export completed in 2.34s

Summary
=======
Total:   45
Created: 12
Updated: 28
Skipped: 3
Errors:  2
```

**Fields:**

- **Total**: Total devices processed
- **Created**: New devices added to NetBox
- **Updated**: Existing devices modified in NetBox
- **Skipped**: Devices not exported due to validation failure
- **Errors**: Devices that failed during creation/update

### Created/Updated Devices

After successful export, you see the created and updated devices with their NetBox IDs:

```
Created Devices:
Name              MAC                NetBox ID
----------------  -----------------  ----------
AP-LOBBY-01       aa:bb:cc:dd:ee:01  1024
AP-LOBBY-02       aa:bb:cc:dd:ee:02  1025
AP-CONF-01        aa:bb:cc:dd:ee:03  1026
```

### Dry Run Output

Dry run mode shows simulated results without modifying NetBox:

```
Export simulation completed in 1.23s

Summary
=======
Total:         45
Would create:  12
Would update:  28
Skipped:       3
Errors:        0
```

### Validation Output

Validation mode shows the status of your NetBox dependencies:

```
Validation Summary
==================
Total devices:   45
Valid:           42
Invalid:         3

Missing Sites in NetBox:
  - US-TEMP-LAB (1 devices)

Missing Device Types in NetBox:
  - ms120-24 (2 devices)
```

## Reverse Sync: NetBox as Configuration Source

The reverse sync feature allows you to use NetBox as the authoritative source for device configuration metadata (device names and site assignments). This is useful when:

- NetBox is your central IPAM/DCIM system
- Network engineers manage device assignments in NetBox
- You want wifimgr to read NetBox to determine device configurations

### Configuration

Set `settings_source` to `"netbox"` in your configuration:

```json
{
  "netbox": {
    "url": "https://netbox.example.com",
    "credentials": {
      "api_key": "your-api-key"
    },
    "settings_source": "netbox"
  }
}
```

### How It Works

When `settings_source` is set to `"netbox"`:

1. wifimgr queries NetBox for devices filtered by:
   - Device role: `"ap"` (Access Points only)
   - Site: specified site name (or all sites if not specified)
2. NetBox devices are matched to physical devices using **MAC address** as the common key
3. Device metadata from NetBox (name, site assignment) is used to configure devices

### Use Cases

**Scenario 1: Central Device Inventory Management**
```bash
# 1. Add APs to NetBox with desired names and site assignments
# 2. Configure wifimgr to use NetBox as source
# 3. Run apply commands - wifimgr reads assignments from NetBox

wifimgr apply site US-LAB-01
```

**Scenario 2: Device Reassignment**
```bash
# 1. In NetBox, move device to different site
# 2. Update device name in NetBox
# 3. Apply changes via wifimgr

wifimgr apply site US-NEW-SITE
```

### API Integration

The reverse sync functionality is also available programmatically:

```go
import "github.com/ravinald/wifimgr/internal/integrations/netbox"

// Create syncer
syncer, err := netbox.NewSyncer(config)

// Sync all APs from NetBox
metadata, err := syncer.SyncFromNetBox(ctx, "")

// Sync APs from specific site
metadata, err := syncer.SyncFromNetBox(ctx, "US-LAB-01")

// Get metadata for specific device by MAC
deviceMeta, err := syncer.GetDeviceMetadata(ctx, "aa:bb:cc:dd:ee:ff")
```

The synced metadata contains:
- `MAC`: Normalized MAC address (key)
- `Name`: Device name from NetBox
- `SiteID`: NetBox site ID
- `SiteName`: NetBox site name
- `Model`: Device model
- `Serial`: Device serial number

## Usage Examples

### Basic Export to NetBox

Export all your infrastructure inventory to a NetBox instance:

```bash
# Configure credentials first (choose one method)
export NETBOX_API_URL=https://netbox.example.com
export NETBOX_API_KEY=your-api-key

# Then export
wifimgr export netbox all
```

### Export Specific Site

Export only devices from your main office site:

```bash
wifimgr export netbox site US-MAIN-OFFICE
```

### Preview Before Export

Always test with dry-run first to ensure mappings are correct:

```bash
wifimgr export netbox all dry-run
```

Review the output, then export if it looks correct:

```bash
wifimgr export netbox all
```

### Validate Configuration

Check that all required NetBox objects exist before exporting:

```bash
wifimgr export netbox all validate
```

If validation fails, create the missing objects in NetBox or adjust your mappings.

### Export with Custom Mappings

Configure device type mappings for your hardware models:

**In wifimgr-config.json:**

```json
{
  "netbox": {
    "url": "https://netbox.example.com",
    "credentials": {
      "api_key": "your-key"
    },
    "mappings": {
      "device_types": {
        "MR46E": { "slug": "cisco-mr46e" },
        "MR45": { "slug": "cisco-mr45", "role": "special-meraki-role" },
        "AP43": { "slug": "juniper-ap43" }
      },
      "default_roles": {
        "ap": "wireless-ap",
        "switch": "access-switch",
        "gateway": "router"
      },
      "site_overrides": {
        "US-LAB": "us-lab-01",
        "EU-PROD": "eu-production"
      }
    }
  }
}
```

Then export:

```bash
wifimgr export netbox all
```

### Multi-Site Export with Environment Variables

For automated workflows, use environment variables:

```bash
#!/bin/bash
NETBOX_API_URL=https://netbox.prod.example.com \
NETBOX_API_KEY=$NETBOX_TOKEN \
NETBOX_SSL_VERIFY=true \
wifimgr export netbox all
```

### Troubleshooting with Validation

If export fails, use validation to identify issues:

```bash
# First validate
wifimgr export netbox all validate

# This shows missing sites, types, and roles
# Create them in NetBox, then retry

# Once validation passes, try actual export
wifimgr export netbox all dry-run

# Check dry run results, then do real export
wifimgr export netbox all
```

## Common Issues and Solutions

### Issue: "NetBox URL is required"

**Cause**: Neither environment variable nor configuration file has been set.

**Solution**: Set the configuration using one of these methods:

```bash
# Environment variable
export NETBOX_API_URL=https://netbox.example.com

# Or environment file ~/.env.netbox
echo "NETBOX_API_URL=https://netbox.example.com" >> ~/.env.netbox

# Or config file
# Add netbox.url to wifimgr-config.json
```

### Issue: "Site 'US-LAB-01' not found in NetBox"

**Cause**: The site doesn't exist in NetBox or the slug doesn't match.

**Solution**:

1. Create the site in NetBox with matching name or slug
2. Use site override mapping if site name differs:

```json
{
  "netbox": {
    "mappings": {
      "site_overrides": {
        "US-LAB-01": "us-lab"
      }
    }
  }
}
```

### Issue: "Device type 'mr46e' (slug 'mr46e') not found in NetBox"

**Cause**: Device type doesn't exist in NetBox or slug is incorrect.

**Solution**:

1. Create the device type in NetBox with the matching slug
2. Add mapping for the model:

```json
{
  "netbox": {
    "mappings": {
      "device_types": {
        "MR46E": "cisco-mr46e"
      }
    }
  }
}
```

### Issue: "Authentication failed" or "Invalid API key"

**Cause**: API key is incorrect or has insufficient permissions.

**Solution**:

1. Generate a new API token in NetBox
2. Verify token has device and interface read/write permissions
3. Check for typos in the API key
4. Update credentials and retry

### Issue: Devices are being skipped during export

**Cause**: Devices don't meet validation requirements.

**Solution**:

1. Run validation to identify which devices and why they're failing:

```bash
wifimgr export netbox all validate
```

2. Check for missing:
   - Site assignment
   - Device model
   - Device type (ap/switch/gateway)
   - Device name or MAC address

3. Update device information and re-run cache refresh:

```bash
wifimgr cache refresh
wifimgr export netbox all
```

### Issue: SSL certificate verification failures

**Cause**: Your NetBox instance uses a self-signed certificate or the certificate chain is incomplete.

**Solution** (if you trust the server):

```bash
# Via environment variable
export NETBOX_SSL_VERIFY=false
wifimgr export netbox all

# Via config file
{
  "netbox": {
    "ssl_verify": false
  }
}
```

**Note**: Only disable SSL verification in development/test environments or when you have verified the server certificate manually.

### Issue: "Interface type 'X' is not valid"

**Cause**: An invalid interface type was specified in the interface mappings configuration or per-device settings.

**Solution**:

1. Check the error message for valid interface types
2. Update your configuration with a valid type:

```json
{
  "netbox": {
    "mappings": {
      "interfaces": {
        "eth0": {
          "name": "eth0",
          "type": "1000base-t"
        },
        "radio1": {
          "name": "wifi1",
          "type": "ieee802.11ac"
        }
      }
    }
  }
}
```

Common mistakes:
- Using `wifi6` instead of `ieee802.11ax`
- Using `gigabit` instead of `1000base-t`
- Using display names instead of API values

See [Interface Mappings](#interface-mappings) for the full list of valid interface types.

## Security Considerations

### API Key Protection

- **Never** commit API keys to version control
- Use environment variables for CI/CD workflows
- Store `.env.netbox` in your home directory (not in project directories)
- Set appropriate file permissions: `chmod 600 ~/.env.netbox`
- Consider using encrypted keys in configuration files
- Rotate API keys periodically

### Configuration File Security

If storing NetBox credentials in config files:

1. Set restrictive file permissions: `chmod 600 wifimgr-config.json`
2. Store config files outside version control
3. Consider encrypting sensitive fields
4. Use a configuration management system for production

### SSL/TLS

- Always use `ssl_verify: true` in production environments
- Only disable SSL verification in development/test with self-signed certificates
- Verify NetBox server certificates are properly signed and up-to-date

## See Also

- [Configuration Documentation](configuration.md) - wifimgr configuration system
- [Device Configuration](device-configuration.md) - Managing device settings
- [Utilities](utilities.md) - MAC address and device utilities
- [Multi-Vendor Support](multi-vendor/) - Working with multiple APIs

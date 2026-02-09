# Vendor API Field Mappings

This document details how vendor-specific API fields map to wifimgr's common schema. Understanding these mappings is essential for troubleshooting configuration issues and knowing when vendor-specific extension blocks are required.

## Design Philosophy

**wifimgr uses Mist nomenclature as the standard** for common schema field names. This means:

- **Mist API**: Most fields map 1:1 without transformation
- **Meraki API**: Fields are transformed from Meraki naming to Mist-style naming
- **Vendor blocks**: Use `mist:` or `meraki:` extension blocks for vendor-specific features

## AP Configuration Field Mappings

### Core Identity Fields

| Common Schema Field   | Mist API Field   | Meraki API Field                | Notes                                              |
|-----------------------|------------------|---------------------------------|----------------------------------------------------|
| `name`                | `name`           | `name`                          | 1:1 mapping for both vendors                       |
| `notes`               | `notes`          | `notes`                         | 1:1 mapping for both vendors                       |
| `tags`                | `tags` (array)   | `tags` (space-separated string) | Meraki uses single string, converted to/from array |

### Location Fields

| Common Schema Field   | Mist API Field                | Meraki API Field   | Notes                                               |
|-----------------------|-------------------------------|--------------------|-----------------------------------------------------|
| `location`            | `location` (array [lat, lng]) | -                  | Mist uses array format                              |
| `location[0]`         | `location[0]`                 | `lat`              | Latitude                                            |
| `location[1]`         | `location[1]`                 | `lng`              | Longitude                                           |
| `orientation`         | `orientation`                 | -                  | Degrees (0-359), Mist only                          |
| `map_id`              | `map_id`                      | `floor_plan_id`    | Floor plan identifier                               |
| `x`                   | `x`                           | -                  | X position on floor plan (pixels), Mist only        |
| `y`                   | `y`                           | -                  | Y position on floor plan (pixels), Mist only        |
| `height`              | `height`                      | -                  | Mounting height in meters, Mist only                |

**Vendor Block Required**: Meraki floor plan positioning requires `meraki.floor_plan_id` or `meraki.lat`/`meraki.lng`

### Radio Configuration Fields

#### Radio Config Block

| Common Schema Field              | Mist API Field                   | Meraki API Field   | Notes                                |
|----------------------------------|----------------------------------|--------------------|--------------------------------------|
| `radio_config`                   | `radio_config`                   | `radioSettings`    | Top-level radio configuration object |
| `radio_config.allow_rrm_disable` | `radio_config.allow_rrm_disable` | -                  | Mist only                            |
| `radio_config.scanning_enabled`  | `radio_config.scanning_enabled`  | -                  | Mist only                            |
| `radio_config.indoor_use`        | `radio_config.indoor_use`        | -                  | Mist only                            |
| `radio_config.ant_gain_24`       | `radio_config.ant_gain_24`       | -                  | 2.4 GHz antenna gain, Mist only      |
| `radio_config.ant_gain_5`        | `radio_config.ant_gain_5`        | -                  | 5 GHz antenna gain, Mist only        |
| `radio_config.ant_gain_6`        | `radio_config.ant_gain_6`        | -                  | 6 GHz antenna gain, Mist only        |
| `radio_config.antenna_mode`      | `radio_config.antenna_mode`      | -                  | "default", "1x1", "2x2", Mist only   |
| `radio_config.band_24_usage`     | `radio_config.band_24_usage`     | -                  | "24", "5", "auto", Mist only         |

#### Per-Band Configuration (2.4 GHz)

| Common Schema Field    | Mist API Field         | Meraki API Field                   | Notes                              |
|------------------------|------------------------|------------------------------------|------------------------------------|
| `radio_config.band_24` | `radio_config.band_24` | `radioSettings.twoFourGhzSettings` | Per-band settings object           |
| `band_24.disabled`     | `band_24.disabled`     | -                                  | Disable radio, Mist only           |
| `band_24.channel`      | `band_24.channel`      | `twoFourGhzSettings.channel`       | Channel number (0 = auto)          |
| `band_24.power`        | `band_24.power`        | `twoFourGhzSettings.targetPower`   | Transmit power in dBm              |
| `band_24.power_min`    | `band_24.power_min`    | -                                  | Minimum power, Mist only           |
| `band_24.power_max`    | `band_24.power_max`    | -                                  | Maximum power, Mist only           |
| `band_24.bandwidth`    | `band_24.bandwidth`    | `twoFourGhzSettings.channelWidth`  | 20, 40 MHz                         |
| `band_24.channels`     | `band_24.channels`     | -                                  | Allowed channel list, Mist only    |
| `band_24.antenna_mode` | `band_24.antenna_mode` | -                                  | Per-band antenna mode, Mist only   |
| `band_24.ant_gain`     | `band_24.ant_gain`     | -                                  | Per-band antenna gain, Mist only   |
| `band_24.preamble`     | `band_24.preamble`     | -                                  | "short", "long", "auto", Mist only |

**Vendor Block Required**: Meraki-specific per-band settings like `min_bitrate` and `rxsop` require `meraki:` blocks

#### Per-Band Configuration (5 GHz)

| Common Schema Field   | Mist API Field        | Meraki API Field                | Notes                     |
|-----------------------|-----------------------|---------------------------------|---------------------------|
| `radio_config.band_5` | `radio_config.band_5` | `radioSettings.fiveGhzSettings` | Per-band settings object  |
| `band_5.disabled`     | `band_5.disabled`     | -                               | Disable radio, Mist only  |
| `band_5.channel`      | `band_5.channel`      | `fiveGhzSettings.channel`       | Channel number (0 = auto) |
| `band_5.power`        | `band_5.power`        | `fiveGhzSettings.targetPower`   | Transmit power in dBm     |
| `band_5.bandwidth`    | `band_5.bandwidth`    | `fiveGhzSettings.channelWidth`  | 20, 40, 80, 160 MHz       |

#### Per-Band Configuration (6 GHz)

| Common Schema Field   | Mist API Field        | Meraki API Field               | Notes                           |
|-----------------------|-----------------------|--------------------------------|---------------------------------|
| `radio_config.band_6` | `radio_config.band_6` | `radioSettings.sixGhzSettings` | Per-band settings object        |
| `band_6.disabled`     | `band_6.disabled`     | -                              | Disable radio, Mist only        |
| `band_6.channel`      | `band_6.channel`      | `sixGhzSettings.channel`       | Channel number (1-233, step 4)  |
| `band_6.power`        | `band_6.power`        | `sixGhzSettings.targetPower`   | Transmit power in dBm (1-30)    |
| `band_6.bandwidth`    | `band_6.bandwidth`    | `sixGhzSettings.channelWidth`  | 20, 40, 80, 160, 320 MHz        |

#### Dual-Band / Flex Radio Configuration (band_dual)

For APs with dual-band radios (Mist) or flex radios (Meraki):

| Common Schema Field        | Mist API Translation                | Meraki API Translation          | Notes                              |
|----------------------------|-------------------------------------|---------------------------------|------------------------------------|
| `radio_config.band_dual`   | -                                   | -                               | Unified config for dual/flex radio |
| `band_dual.disabled`       | (in band_5_on_24_radio)             | (in target band settings)       | Enable/disable the radio           |
| `band_dual.radio_mode`     | `band_24_usage` ("24" or "5")       | `flexRadioBand` ("five"/"six")  | Target band selection              |
| `band_dual.channel`        | `band_5_on_24_radio.channel`        | (in target band)                | Channel number                     |
| `band_dual.power`          | `band_5_on_24_radio.power`          | (in target band)                | Transmit power in dBm              |
| `band_dual.bandwidth`      | `band_5_on_24_radio.bandwidth`      | (in target band)                | Channel width in MHz               |

**Vendor-specific `radio_mode` values:**
- **Mist**: 24 or 5 (dual-band radios can convert 2.4GHz radio to 5GHz)
- **Meraki**: 5 or 6 (flex radios toggle between 5GHz and 6GHz)

**Translation Examples:**

Mist with `radio_mode: 5`:
```json
// wifimgr config:
{ "band_dual": { "radio_mode": 5, "channel": 149, "power": 12 } }

// Translated to Mist API:
{ "band_24_usage": "5", "band_5_on_24_radio": { "channel": 149, "power": 12 } }
```

Meraki with `radio_mode: 6`:
```json
// wifimgr config:
{ "band_dual": { "radio_mode": 6, "channel": 37, "power": 15 } }

// Translated to Meraki API:
{ "flexRadioBand": "six", "sixGhzSettings": { "channel": 37, "targetPower": 15 } }
```

### IP Configuration Fields

| Common Schema Field   | Mist API Field      | Meraki API Field   | Notes               |
|-----------------------|---------------------|--------------------|---------------------|
| `ip_config.type`      | `ip_config.type`    | `assignmentMode`   | "dhcp" or "static"  |
| `ip_config.ip`        | `ip_config.ip`      | `address`          | Static IP address   |
| `ip_config.netmask`   | `ip_config.netmask` | `netmask`          | Subnet mask         |
| `ip_config.gateway`   | `ip_config.gateway` | `gateway`          | Default gateway     |
| `ip_config.dns`       | `ip_config.dns`     | `dns`              | DNS servers (array) |
| `ip_config.vlan_id`   | `ip_config.vlan_id` | `vlan`             | Management VLAN     |

### LED Configuration

| Common Schema Field   | Mist API Field   | Meraki API Field   | Notes                   |
|-----------------------|------------------|--------------------|-------------------------|
| `led.enabled`         | `led.enabled`    | `ledLightsOn`      | LED status light on/off |
| `led.brightness`      | `led.brightness` | -                  | 0-100, Mist only        |

### Device Profile / RF Profile

| Common Schema Field   | Mist API Field       | Meraki API Field   | Notes                        |
|-----------------------|----------------------|--------------------|------------------------------|
| `deviceprofile_id`    | `deviceprofile_id`   | -                  | Mist device profile UUID     |
| `deviceprofile_name`  | `deviceprofile_name` | -                  | Alternative to ID, Mist only |
| -                     | -                    | `rfProfileId`      | Meraki RF profile ID         |

**Vendor Block Required**: Meraki RF profiles require `meraki.rf_profile_id` in the `radio_config` block

### Hardware Flags

| Common Schema Field   | Mist API Field    | Meraki API Field   | Notes                                  |
|-----------------------|-------------------|--------------------|----------------------------------------|
| `disable_eth1`        | `disable_eth1`    | -                  | Disable secondary ethernet, Mist only  |
| `disable_eth2`        | `disable_eth2`    | -                  | Disable tertiary ethernet, Mist only   |
| `disable_eth3`        | `disable_eth3`    | -                  | Disable quaternary ethernet, Mist only |
| `poe_passthrough`     | `poe_passthrough` | -                  | Enable PoE passthrough, Mist only      |

### BLE Configuration

| Common Schema Field    | Mist API Field         | Meraki API Field   | Notes                               |
|------------------------|------------------------|--------------------|-------------------------------------|
| `ble_config.enabled`   | `ble_config.enabled`   | -                  | BLE radio enabled, Mist only        |
| `ble_config.power`     | `ble_config.power`     | -                  | BLE transmit power (dBm), Mist only |
| `ble_config.ibeacon`   | `ble_config.ibeacon`   | -                  | iBeacon configuration, Mist only    |
| `ble_config.eddystone` | `ble_config.eddystone` | -                  | Eddystone configuration, Mist only  |

**Vendor Block Required**: All BLE configuration is Mist-specific

### Mesh Configuration

| Common Schema Field   | Mist API Field   | Meraki API Field   | Notes                                 |
|-----------------------|------------------|--------------------|---------------------------------------|
| `mesh.enabled`        | `mesh.enabled`   | -                  | Mesh networking enabled, Mist only    |
| `mesh.role`           | `mesh.role`      | -                  | "root", "node", "spreader", Mist only |
| `mesh.group`          | `mesh.group`     | -                  | Mesh group identifier, Mist only      |

**Vendor Block Required**: All mesh configuration is Mist-specific

## Fields Requiring Vendor Extension Blocks

Use vendor extension blocks for these vendor-specific features:

### Mist-Specific Features

These features require the `mist:` extension block:

```json
{
  "mist": {
    "map_id": "floor-plan-uuid",
    "x": 125.5,
    "y": 340.2,
    "orientation": 90,
    "height": 3.0,
    "disable_eth1": true,
    "usb_config": { "enabled": true },
    "vars": { "site_var": "value" },
    "aeroscout": { "enabled": true }
  }
}
```

**Common Mist-only fields:**
- Floor plan positioning: `map_id`, `x`, `y`, `orientation`, `height`
- Hardware control: `disable_eth1`, `disable_eth2`, `disable_eth3`, `disable_module`
- USB configuration: `usb_config`
- Template variables: `vars`
- Location services: `aeroscout`
- BLE/beacon configuration: `ble_config` (entire block)
- Mesh networking: `mesh` (entire block)
- RRM settings: `allow_rrm_disable`, `scanning_enabled`, `indoor_use`

### Meraki-Specific Features

These features require the `meraki:` extension block:

```json
{
  "meraki": {
    "rf_profile_id": "12345",
    "floor_plan_id": "L_123456789",
    "lat": 37.7749,
    "lng": -122.4194
  }
}
```

**Common Meraki-only fields:**
- RF profiles: `rf_profile_id` (in `radio_config.meraki`)
- Floor plans: `floor_plan_id`
- GPS coordinates: `lat`, `lng`
- Per-band settings: `min_bitrate`, `rxsop` (in band config `meraki` blocks)
- Band steering: `band_selection`
- Per-SSID radio settings: `per_ssid_settings`

## Field Name Transformations

### Automatic Transformations (Meraki → Common Schema)

The Meraki converter automatically transforms these field names:

| Meraki API Field     | Common Schema Field    | Context                  |
|----------------------|------------------------|--------------------------|
| `targetPower`        | `power`                | Radio band configuration |
| `channelWidth`       | `bandwidth`            | Radio band configuration |
| `ledLightsOn`        | `led.enabled`          | LED configuration        |
| `assignmentMode`     | `ip_config.type`       | IP configuration         |
| `address`            | `ip_config.ip`         | IP configuration         |
| `twoFourGhzSettings` | `radio_config.band_24` | Radio configuration      |
| `fiveGhzSettings`    | `radio_config.band_5`  | Radio configuration      |
| `sixGhzSettings`     | `radio_config.band_6`  | Radio configuration      |

### Common Schema → Vendor API (Apply Direction)

When applying configuration:

**To Mist API**: Most fields pass through 1:1. Vendor extensions in the `mist:` block are merged at the appropriate level.

**To Meraki API**: Field names are transformed and vendor extensions in the `meraki:` block are merged.

## Error Scenarios

### Missing Field Mapping

**Symptom**: Configuration appears correct but doesn't apply to the device

**Cause**: Field might be vendor-specific and requires a vendor extension block

**Solution**: Check this mapping table. If the field is vendor-specific, move it to the appropriate `mist:` or `meraki:` extension block.

### Unexpected API Field

**Symptom**: Vendor API returns fields not recognized by wifimgr

**Cause**: Vendor added new API fields or changed field structure

**Solution**:
1. Check logs for warnings about unrecognized fields
2. If needed, add new fields to vendor extension block
3. Report issue for potential common schema inclusion

### Type Mismatch

**Symptom**: API returns error about invalid field type

**Cause**: Field type changed in vendor API (e.g., string became integer)

**Solution**:
1. Verify your config file has the correct type
2. Check recent vendor API documentation for changes
3. Update your configuration to match new type requirements

## Validation

Use the built-in validation to catch common errors:

```bash
# Dry-run shows validation errors
wifimgr apply ap US-LAB-01 --dry-run

# Debug mode shows field mappings
wifimgr -ddd apply ap US-LAB-01 --dry-run
```

Common validation errors:

- **Mutually exclusive fields**: `deviceprofile_id` and `deviceprofile_name`
- **Mutually exclusive fields**: `map_id` and `map_name`
- **Wrong vendor block**: Using `mist:` block for Meraki device or vice versa
- **Missing vendor block**: Vendor-specific field in common schema area

## Examples

### Example 1: Mist AP with Floor Plan Positioning

```json
{
  "5c5b358e4cf9": {
    "name": "Conference-Room-AP",
    "radio_config": {
      "band_24": { "channel": 1, "power": 8, "bandwidth": 20 },
      "band_5": { "channel": 44, "power": 12, "bandwidth": 80 }
    },
    "mist": {
      "map_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "x": 450,
      "y": 280,
      "height": 3.0,
      "orientation": 90
    }
  }
}
```

**Fields mapped:**
- Common schema: `name`, `radio_config.band_24.*`, `radio_config.band_5.*`
- Mist extension: `map_id`, `x`, `y`, `height`, `orientation`

### Example 2: Meraki AP with RF Profile

```json
{
  "ac69cf0462a0": {
    "name": "Lobby-AP",
    "api": "meraki-corp",
    "radio_config": {
      "band_24": { "channel": 6, "power": 10, "bandwidth": 20 },
      "band_5": { "channel": 149, "power": 14, "bandwidth": 40 },
      "meraki": {
        "rf_profile_id": "12345"
      }
    },
    "meraki": {
      "floor_plan_id": "L_123456789",
      "min_bitrate": 12
    }
  }
}
```

**Fields mapped:**
- Common schema: `name`, `radio_config.band_24.*`, `radio_config.band_5.*`
- Meraki transformations: `power` → `targetPower`, `bandwidth` → `channelWidth`
- Meraki extensions: `rf_profile_id` (in radio_config), `floor_plan_id`, `min_bitrate`

### Example 3: Mixed Common and Vendor Settings

```json
{
  "5c5b358e4cf9": {
    "name": "AP-01",
    "notes": "Main office AP",
    "tags": ["office", "wifi6"],
    "ip_config": {
      "type": "dhcp",
      "vlan_id": 100
    },
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
        "bandwidth": 80
      },
      "mist": {
        "scanning_enabled": true,
        "indoor_use": true
      }
    },
    "led": {
      "enabled": false
    },
    "mist": {
      "map_id": "floor-2-uuid",
      "x": 250,
      "y": 180,
      "disable_eth1": true
    }
  }
}
```

**Fields mapped:**
- Common identity: `name`, `notes`, `tags`
- Common config: `ip_config.*`, `led.enabled`
- Common radio: `radio_config.band_24.*`, `radio_config.band_5.*`, `radio_config.band_24_usage`
- Mist-specific radio: `scanning_enabled`, `indoor_use` (in `radio_config.mist`)
- Mist-specific device: `map_id`, `x`, `y`, `disable_eth1` (in top-level `mist`)

## Troubleshooting Tips

1. **Enable debug logging** to see field transformations:
   ```bash
   wifimgr -ddd apply ap US-LAB-01 --dry-run
   ```

2. **Check vendor API documentation** for recent changes to field names or types

3. **Use import command** to see how wifimgr reads current API state:
   ```bash
   wifimgr import api site US-LAB-01
   ```

4. **Compare configurations** using JSON diff in dry-run mode to see exact differences

5. **Validate before applying** using the built-in validation and dry-run features

## Related Documentation

- [User Guide - Vendor-Specific Configuration](../USER-GUIDE.md#vendor-specific-configuration)
- [Multi-Vendor Architecture](multi-vendor/overview.md)
- [Meraki Implementation](multi-vendor/meraki.md)
- [Mist Implementation](multi-vendor/mist.md)

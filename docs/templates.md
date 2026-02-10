# Templates

Templates in wifimgr are an **app-level convenience** that expands into explicit device settings. Templates are NOT vendor-side profiles—they expand at apply time into fully explicit configurations that are pushed to each device.

```
Template file → Expand at apply → Push explicit config to each device
```

**Benefits:**
- **Vendor-agnostic**: Works for both Mist and Meraki
- **Explicit configs**: What you see is what gets pushed
- **No vendor profile management**: All configuration is local and explicit
- **Simple mental model**: Templates are "copy-paste with variables"

## Template Types

| Type     | Purpose                                        | Reference Field   |
|----------|------------------------------------------------|-------------------|
| `radio`  | Radio/RF settings (power, channels, bandwidth) | `radio_profile`   |
| `wlan`   | WLAN settings (SSID, auth, VLAN)               | `wlan` (list)     |
| `device` | Device settings (LED, PoE, ports)              | `device_template` |

## Configuration

### Main Config File

Add template files to your `wifimgr-config.json`:

```json
{
  "files": {
    "config_dir": "./config",
    "site_configs": ["sites/us-office.json"],
    "templates": [
      "templates/radio.json",
      "templates/wlan.json",
      "templates/device.json"
    ]
  }
}
```

Multiple template files can be specified. Templates with the same name in later files override earlier ones.

## Template File Format

Template files use a simple JSON structure with version and templates sections:

```json
{
  "version": 1,
  "templates": {
    "radio": {
      "template-name": { ... }
    },
    "wlan": {
      "template-name": { ... }
    },
    "device": {
      "template-name": { ... }
    }
  }
}
```

### Radio Templates

Radio templates define RF settings for access points. Band settings are specified directly under the template name (no `radio_config` wrapper needed—it's added automatically during expansion):

```json
{
  "version": 1,
  "templates": {
    "radio": {
      "high-density": {
        "band_24": {
          "disabled": false,
          "channel": 6,
          "power": 5
        },
        "band_5": {
          "disabled": false,
          "bandwidth": 40,
          "channel": 36,
          "power": 9
        }
      },
      "wifi6e-enabled": {
        "band_24": { "disabled": false, "channel": 1, "power": 5 },
        "band_5": { "disabled": false, "bandwidth": 80, "channel": 36, "power": 12 },
        "band_6": { "disabled": false, "bandwidth": 160, "channel": 37, "power": 15 }
      }
    }
  }
}
```

#### Dual-Band / Flex Radio Templates

For APs with dual-band radios (Mist) or flex radios (Meraki), use `band_dual`. Create vendor-specific templates since `radio_mode` values differ by vendor:

**Mist templates** (dual-band radios that convert 2.4GHz ↔ 5GHz):
```json
{
  "version": 1,
  "templates": {
    "radio": {
      "mist-dual-5g": {
        "band_24": { "disabled": true },
        "band_dual": {
          "disabled": false,
          "radio_mode": 5,
          "bandwidth": 40,
          "channel": 149,
          "power": 12
        }
      }
    }
  }
}
```

**Meraki templates** (flex radios that toggle 5GHz ↔ 6GHz):
```json
{
  "version": 1,
  "templates": {
    "radio": {
      "meraki-flex-6g": {
        "band_24": { "disabled": false, "channel": 6, "power": 5 },
        "band_dual": {
          "disabled": false,
          "radio_mode": 6,
          "bandwidth": 160,
          "channel": 37,
          "power": 15
        }
      }
    }
  }
}
```

**`band_dual` fields:**

| Field        | Type   | Description                                   |
|--------------|--------|-----------------------------------------------|
| `disabled`   | bool   | Radio enabled/disabled                        |
| `radio_mode` | int    | Operating mode: 24, 5, or 6 (vendor-specific) |
| `channel`    | int    | Channel number (validated against radio_mode) |
| `power`      | int    | Transmit power in dBm (1-30)                  |
| `bandwidth`  | int    | Channel width: 20, 40, 80, 160, or 320 MHz    |

**Vendor-specific `radio_mode` values:**
- **Mist**: 24 or 5 (dual-band radios convert 2.4GHz → 5GHz)
- **Meraki**: 5 or 6 (flex radios toggle between 5GHz ↔ 6GHz)

**Linter validation**: The linter will warn if a `band_dual.radio_mode` is incompatible with the target API. For example, using `radio_mode: 6` with a Mist API will produce a warning.

### WLAN Templates

WLAN templates define wireless network settings:

```json
{
  "version": 1,
  "templates": {
    "wlan": {
      "corp-secure": {
        "ssid": "CorpNet",
        "enabled": true,
        "band": "5",
        "vlan_id": 100,
        "auth": {
          "type": "wpa2-enterprise",
          "radius_servers": [
            { "host": "radius.example.com", "port": 1812 }
          ]
        }
      },
      "guest-open": {
        "ssid": "GuestWiFi",
        "enabled": true,
        "band": "dual",
        "vlan_id": 200,
        "auth": { "type": "open" },
        "portal": { "enabled": true }
      }
    }
  }
}
```

#### WLAN Auth Types

The `auth.type` field specifies the security mode. Available types depend on the vendor and band configuration:

| Auth Type         | Description                       | 6GHz / Wi-Fi 7   |
|-------------------|-----------------------------------|------------------|
| `open`            | No security (open network)        | No               |
| `psk`             | WPA2-Personal (PSK)               | No               |
| `sae`             | WPA3-Personal (SAE)               | **Yes**          |
| `psk-wpa2-wpa3`   | WPA2/WPA3 Transition Mode         | Partial          |
| `wpa2-enterprise` | WPA2-Enterprise (802.1X)          | No               |
| `eap-192`         | WPA3-Enterprise 192-bit           | **Yes**          |
| `owe`             | Opportunistic Wireless Encryption | **Yes**          |

**Important:** 6GHz and Wi-Fi 7 bands require WPA3 security. If you're configuring WLANs for APs with 6GHz radios, use `sae`, `eap-192`, or `owe` as the auth type.

**WPA3-Personal Example (for 6GHz):**
```json
{
  "ssid": "SecureWiFi",
  "enabled": true,
  "band": "6",
  "auth": {
    "type": "sae",
    "psk": "enc:your-encrypted-password"
  }
}
```

**WPA2/WPA3 Transition Mode (dual-band):**
```json
{
  "ssid": "CorpWiFi",
  "enabled": true,
  "band": "dual",
  "auth": {
    "type": "psk-wpa2-wpa3",
    "psk": "enc:your-encrypted-password"
  }
}
```

### Device Templates

Device templates define device-level settings:

```json
{
  "version": 1,
  "templates": {
    "device": {
      "standard-ap": {
        "led": { "enabled": true },
        "poe_passthrough": false
      },
      "warehouse-ap": {
        "led": { "enabled": false },
        "poe_passthrough": true
      }
    }
  }
}
```

## Vendor-Specific Settings

Templates support vendor-specific blocks using `vendor:` suffix keys. Common fields are shared, while vendor-specific fields are merged based on the target API:

```json
{
  "version": 1,
  "templates": {
    "radio": {
      "high-density": {
        "radio_config": {
          "band_5": { "power": 15, "bandwidth": 40 }
        },
        "mist:": {
          "radio_config": { "scanning_enabled": true }
        },
        "meraki:": {
          "rf_profile_id": "meraki-high-density-rf"
        }
      }
    }
  }
}
```

When applied to a Mist API (`api: mist-prod`), the result includes:
- Common fields: `radio_config.band_5.power`, `radio_config.band_5.bandwidth`
- Mist-specific: `radio_config.scanning_enabled`

When applied to a Meraki API (`api: meraki-corp`), the result includes:
- Common fields: `radio_config.band_5.power`, `radio_config.band_5.bandwidth`
- Meraki-specific: `rf_profile_id`

## Using Templates in Site Config

### Device-Level References

Reference templates in your device configurations:

```json
{
  "version": 1,
  "config": {
    "sites": {
      "US-NYC-OFFICE": {
        "site_config": {
          "name": "US-NYC-OFFICE",
          "api": "mist-prod"
        },
        "devices": {
          "ap": {
            "aa:bb:cc:dd:ee:f1": {
              "name": "NYC-AP-01",
              "radio_profile": "high-density",
              "device_template": "standard-ap",
              "wlan": ["corp-secure", "guest-open"]
            },
            "aa:bb:cc:dd:ee:f2": {
              "name": "NYC-AP-02",
              "radio_profile": "low-power",
              "wlan": ["corp-secure"]
            }
          }
        }
      }
    }
  }
}
```

### WLAN Configuration Levels

WLANs can be configured at three levels:

| Level                  | Key                     | Purpose                                                     |
|------------------------|-------------------------|-------------------------------------------------------------|
| `profiles.wlan`        | List of template labels | WLANs to **create** at the site (make available)            |
| `wlan` (site-level)    | List of template labels | WLANs to **apply** to all APs by default                    |
| `devices.ap[mac].wlan` | List of template labels | WLANs to **apply** to specific APs (overrides site default) |

```json
{
  "version": 1,
  "config": {
    "sites": {
      "US-NYC-OFFICE": {
        "site_config": { "name": "US-NYC-OFFICE" },
        "profiles": {
          "wlan": ["corp-secure", "guest-open", "iot-network"]
        },
        "wlan": ["corp-secure", "guest-open"],
        "devices": {
          "ap": {
            "aa:bb:cc:dd:ee:f1": {
              "name": "NYC-AP-LOBBY"
            },
            "aa:bb:cc:dd:ee:f2": {
              "name": "NYC-AP-WAREHOUSE",
              "wlan": ["iot-network"]
            }
          }
        }
      }
    }
  }
}
```

In this example:
- **profiles.wlan**: Creates 3 WLANs at the site (corp-secure, guest-open, iot-network)
- **wlan** (site-level): Applies corp-secure and guest-open to all APs by default
- **NYC-AP-LOBBY**: Gets corp-secure and guest-open (inherits site default)
- **NYC-AP-WAREHOUSE**: Gets only iot-network (explicit device config overrides site default)

## Override Behavior

Device-specific values always override template values:

```json
{
  "aa:bb:cc:dd:ee:f1": {
    "name": "NYC-AP-01",
    "radio_profile": "high-density",
    "radio_config": {
      "band_5": { "power": 20 }
    }
  }
}
```

In this example:
- `power` is 20 (device override wins)
- `bandwidth` is 40 (from template)
- Other template values are preserved

### WLAN Override Rules

| Scenario                         | Result                         |
|----------------------------------|--------------------------------|
| Site has WLANs, device has none  | Device gets site WLANs         |
| Site has WLANs, device has WLANs | Device gets only its own WLANs |
| Site has none, device has WLANs  | Device gets its own WLANs      |

### WLAN Validation

wifimgr validates WLAN assignments at multiple points to catch configuration errors early:

**Validation rules:**

1. **Site-level WLANs must be declared in `profiles.wlan`**: Any WLAN listed in the site-level `wlan` array must also appear in `profiles.wlan`. This ensures a WLAN is created before it can be assigned.

2. **Device-level WLANs must be declared in `profiles.wlan`**: Any WLAN listed in a device's `wlan` array must also appear in `profiles.wlan`.

3. **Profile WLANs must have a template**: Every label in `profiles.wlan` must have a corresponding WLAN template defined in your template files. This catches typos and missing template definitions.

**When validation runs:**

| Command                      | Validation                                                      |
|------------------------------|-----------------------------------------------------------------|
| `wifimgr apply site ...`     | All 3 rules checked before applying changes                     |
| `wifimgr lint config <site>` | All 3 rules checked (rule 3 requires template files configured) |

**Example error messages:**

```
Site-level WLAN 'guest-open' is not declared in profiles.wlan
Device WLAN 'iot-network' on aa:bb:cc:dd:ee:01 is not declared in profiles.wlan
No WLAN template found for profile 'corp-secure'
```

## Expansion Flow

During `apply`, templates are expanded in this order:

1. **Device template** (`device_template`) - base device settings
2. **Radio profile** (`radio_profile`) - RF settings merged
3. **WLANs** (`wlan` or site fallback) - WLAN templates expanded
4. **Device config** - device-specific values override all

The final expanded configuration is:
- Compared against the API cache for diff display
- Pushed to the API when applying changes

## Viewing Expanded Config

Use `diff` mode to see the fully expanded configuration:

```bash
wifimgr apply site US-NYC-OFFICE ap diff
```

The diff shows the expanded values, not the template references.

## Best Practices

1. **Name templates descriptively**: Use names like `high-density`, `conference-room`, `warehouse` that describe the use case
2. **Group related templates**: Put radio templates in one file, WLANs in another
3. **Use vendor blocks sparingly**: Only add vendor-specific sections when truly needed
4. **Keep device overrides minimal**: If you're overriding most values, consider creating a new template
5. **Document your templates**: Add comments in a separate documentation file explaining each template's purpose

## Example: Complete Setup

**templates/radio.json:**
```json
{
  "version": 1,
  "templates": {
    "radio": {
      "office": {
        "band_24": { "disabled": false, "channel": 6, "power": 5 },
        "band_5": { "disabled": false, "bandwidth": 40, "channel": 36, "power": 12 }
      }
    }
  }
}
```

**templates/wlan.json:**
```json
{
  "version": 1,
  "templates": {
    "wlan": {
      "corp": {
        "ssid": "CorpNet",
        "enabled": true,
        "vlan_id": 100,
        "auth": { "type": "psk", "psk": "enc:..." }
      }
    }
  }
}
```

**sites/us-nyc.json:**
```json
{
  "version": 1,
  "config": {
    "sites": {
      "US-NYC": {
        "site_config": { "name": "US-NYC", "api": "mist-prod" },
        "devices": {
          "ap": {
            "aa:bb:cc:dd:ee:01": {
              "name": "NYC-AP-01",
              "radio_profile": "office",
              "wlan": ["corp"]
            }
          }
        }
      }
    }
  }
}
```

**Result after expansion:**
```json
{
  "name": "NYC-AP-01",
  "radio_config": {
    "band_24": { "disabled": false, "channel": 6, "power": 5 },
    "band_5": { "disabled": false, "bandwidth": 40, "channel": 36, "power": 12 }
  },
  "wlan": [
    {
      "ssid": "CorpNet",
      "enabled": true,
      "vlan_id": 100,
      "auth": { "type": "psk", "psk": "decrypted-value" }
    }
  ]
}
```

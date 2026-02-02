# Multi-API Configuration

The configuration supports multiple API connections with user-defined labels, enabling scenarios like:
- Multiple organizations within the same vendor (prod vs lab)
- Multi-vendor environments (Mist and Meraki in different locations)
- Migration scenarios (old and new systems active simultaneously)
- Testing/POC with mixed vendors at a physical site

## Main Configuration Structure

API connections are defined with user-chosen labels in the main config file:

```json
{
  "api": {
    "mist-prod": {
      "vendor": "mist",
      "url": "https://api.mist.com",
      "credentials": {
        "org_id": "abc-123-def",
        "api_token": "..."
      },
      "rate_limit": 5000,
      "results_limit": 100,
      "cache_ttl": 86400
    },
    "mist-lab": {
      "vendor": "mist",
      "url": "https://api.mist.com",
      "credentials": {
        "org_id": "xyz-789-uvw",
        "api_token": "..."
      },
      "cache_ttl": 0
    },
    "meraki-corp": {
      "vendor": "meraki",
      "url": "https://api.meraki.com",
      "credentials": {
        "org_id": "L_123456789",
        "api_key": "..."
      }
    }
  },
  "files": {
    "cache_dir": "./cache",
    "config_dir": "./config"
  }
}
```

### API Configuration Fields

| Field | Required | Description |
|-------|----------|-------------|
| `vendor` | Yes | Vendor type: `mist`, `meraki` |
| `url` | No | API base URL (vendor default if omitted) |
| `credentials` | Yes | Vendor-specific credentials object |
| `rate_limit` | No | Requests per minute (vendor default if omitted) |
| `results_limit` | No | Max results per API call |
| `cache_ttl` | No | Cache TTL in seconds (see below) |

### Cache TTL Configuration

Each API can have its own `cache_ttl` setting to control when cached data is considered stale:

| Value | Behavior |
|-------|----------|
| Omitted/undefined | Default: 86400 seconds (1 day) |
| `0` | Cache never expires (on-demand refresh only) |
| Positive integer | Cache expires after that many seconds |

**Example:**
- `"cache_ttl": 86400` - Cache expires after 1 day (default)
- `"cache_ttl": 3600` - Cache expires after 1 hour
- `"cache_ttl": 0` - Cache never expires, only refreshed via `refresh cache` command

### Vendor-Specific Credentials

**Mist:**
```json
{
  "org_id": "uuid-format",
  "api_token": "bearer-token"
}
```

**Meraki:**
```json
{
  "org_id": "L_XXXXXXXXX",
  "api_key": "meraki-api-key"
}
```

## Site Configuration with API Binding

Sites specify their default API via the `api` field. Devices inherit this default but can override:

```json
{
  "site_config": {
    "name": "US-CAMPUS-01",
    "api": "mist-prod",
    "timezone": "America/Los_Angeles",
    "devices": {
      "ap": [
        {
          "name": "MIST-AP-01",
          "mac": "aa:bb:cc:dd:ee:ff"
        },
        {
          "name": "MERAKI-AP-01",
          "mac": "11:22:33:44:55:66",
          "api": "meraki-corp"
        }
      ],
      "switch": [
        {
          "name": "MIST-SW-01",
          "mac": "22:33:44:55:66:77"
        }
      ]
    }
  }
}
```

### API Resolution

The effective API for any device is resolved via inheritance:

```go
func resolveDeviceAPI(device DeviceConfig, siteDefault string) string {
    if device.API != "" {
        return device.API  // Device-level override
    }
    return siteDefault     // Inherit from site
}
```

## Validation and Error Handling

### Graceful Degradation

Invalid API references result in warnings and exclusion, not fatal errors:

| Scenario | Behavior | Log Level |
|----------|----------|-----------|
| Site `api` references undefined label | Skip entire site | WARN |
| Device `api` override references undefined label | Skip device only | WARN |
| API defined but credentials incomplete | Warn, skip API initialization | WARN |
| All devices filtered from a site | Site remains (empty) | INFO |
| Empty `api` field on site | Error - site must have API | WARN |
| Empty `api` field on device | Valid - inherits from site | - |

### Validation Flow

```go
// ValidationWarning captures issues found during config loading
type ValidationWarning struct {
    Level   string // "site" or "device"
    Site    string // Site name
    Device  string // Device name (if applicable)
    API     string // The invalid API reference
    Message string // Human-readable message
}

// LoadAndValidateSiteConfigs loads site configs and filters invalid references
func LoadAndValidateSiteConfigs(definedAPIs map[string]bool) ([]*SiteConfig, []ValidationWarning) {
    var validSites []*SiteConfig
    var warnings []ValidationWarning

    for _, site := range rawSites {
        // Validate site-level API (required)
        if site.API == "" {
            warnings = append(warnings, ValidationWarning{
                Level:   "site",
                Site:    site.Name,
                Message: fmt.Sprintf("site %q has no API defined", site.Name),
            })
            continue
        }

        if !definedAPIs[site.API] {
            warnings = append(warnings, ValidationWarning{
                Level:   "site",
                Site:    site.Name,
                API:     site.API,
                Message: fmt.Sprintf("site %q references undefined API %q", site.Name, site.API),
            })
            continue // Skip entire site
        }

        // Filter devices with invalid API overrides
        site.Devices.AP = filterValidDevices(site.Devices.AP, site.Name, site.API, definedAPIs, &warnings)
        site.Devices.Switch = filterValidDevices(site.Devices.Switch, site.Name, site.API, definedAPIs, &warnings)
        site.Devices.Gateway = filterValidDevices(site.Devices.Gateway, site.Name, site.API, definedAPIs, &warnings)

        validSites = append(validSites, site)
    }

    return validSites, warnings
}

func filterValidDevices(devices []DeviceConfig, siteName, siteDefault string,
                        definedAPIs map[string]bool, warnings *[]ValidationWarning) []DeviceConfig {
    var valid []DeviceConfig
    for _, device := range devices {
        effectiveAPI := device.API
        if effectiveAPI == "" {
            effectiveAPI = siteDefault
        }
        if !definedAPIs[effectiveAPI] {
            *warnings = append(*warnings, ValidationWarning{
                Level:   "device",
                Site:    siteName,
                Device:  device.Name,
                API:     device.API,
                Message: fmt.Sprintf("device %q in site %q references undefined API %q",
                    device.Name, siteName, device.API),
            })
            continue // Skip this device
        }
        valid = append(valid, device)
    }
    return valid
}
```

### User Experience

**Startup with validation issues:**
```
WARN  Config validation: site "US-CAMPUS-02" references undefined API "mist-typo" - skipping site
WARN  Config validation: device "AP-05" in site "US-CAMPUS-01" references undefined API "merkai-corp" - skipping device
INFO  Loaded 5 sites (2 items skipped due to invalid API references)
INFO  Initialized 3 API connections: mist-prod, mist-lab, meraki-corp
```

**Command targeting a filtered item:**
```
$ wifimgr show intent ap site US-CAMPUS-02
Error: site "US-CAMPUS-02" was excluded due to invalid API reference "mist-typo"
Hint: Check that the API label is defined in your main config under "api"
```

### Edge Cases Summary

| Scenario | Behavior |
|----------|----------|
| Empty `api` field on site | Error - site must have API defined |
| Empty `api` field on device | Valid - inherits from site default |
| Site API valid, all devices filtered | Site remains with empty device lists |
| Typo in API label | Warn and skip, suggest similar labels if possible |
| API label defined but client init fails | Warn, API unavailable for use |

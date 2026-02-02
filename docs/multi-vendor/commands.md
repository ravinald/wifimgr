# Command Behavior

## CLI Disambiguation Strategy

With multiple APIs configured, commands that target remote resources need clear behavior:

**Principle:** Read operations aggregate by default; write operations use explicit API from config.

| Operation Type | Default Behavior | With `--api` Flag |
|----------------|------------------|-------------------|
| Read (show, search) | Aggregate from all APIs | Filter to specific API |
| Write (apply, set) | Use site/device config API | Override (with warning) |

## The `--api` Flag

A global persistent flag for targeting specific APIs:

```go
// cmd/root.go
var apiFlag string

func init() {
    rootCmd.PersistentFlags().StringVar(&apiFlag, "api", "",
        "Target specific API by label (e.g., --api mist-prod)")
}
```

## Command Behavior Matrix

### Read Commands

| Command | Default Behavior | With `--api` Flag |
|---------|------------------|-------------------|
| `show api ap` | Aggregate from all APIs | Filter to specific API |
| `show api sites` | Aggregate from all APIs | Filter to specific API |
| `show api switch` | Aggregate from all APIs | Filter to specific API |
| `show api wlans` | Aggregate from all APIs | Filter to specific API |
| `show api rf-profiles` | Aggregate from all APIs | Filter to specific API |
| `show api device-profiles` | Aggregate from all APIs | Filter to specific API |
| `show inventory ap` | Aggregate from all APIs | Filter to specific API |
| `show site <name>` | Show from all APIs with that name | Show from specific API |
| `search wired <text>` | Search all APIs | Search specific API |
| `search wireless <text>` | Search all APIs | Search specific API |
| `refresh cache` | Refresh all APIs (parallel) | Refresh specific API only |

### Write Commands

| Command | Default Behavior | With `--api` Flag |
|---------|------------------|-------------------|
| `apply site <name>` | Use site config's API | Override (warns) |
| `apply ap <site>` | Use site config's API | Override (warns) |
| `set ap site` | Use device's resolved API | Override (warns) |

### Intent Commands (Local Config)

| Command | Behavior |
|---------|----------|
| `show intent ap` | Local config only, no API involved |
| `show intent sites` | Local config only, no API involved |

## Aggregation Display

When showing data from multiple APIs, include source columns:

### Example: Show APs from All APIs

```
$ wifimgr show api ap

NAME            MAC               MODEL    SITE           STATUS      API
MIST-AP-01      aa:bb:cc:dd:ee:ff AP43     US-CAMPUS-01   connected   mist-prod
MIST-AP-02      11:22:33:44:55:66 AP43     US-CAMPUS-01   connected   mist-prod
MERAKI-AP-01    77:88:99:aa:bb:cc MR46     US-CAMPUS-01   online      meraki-corp
MERAKI-AP-02    dd:ee:ff:00:11:22 MR46     EU-OFFICE-01   online      meraki-corp

Showing 4 devices from 2 APIs
```

### Example: Show Filtered to Single API

```
$ wifimgr show api ap --api mist-prod

NAME         MAC               MODEL    SITE           STATUS
MIST-AP-01   aa:bb:cc:dd:ee:ff AP43     US-CAMPUS-01   connected
MIST-AP-02   11:22:33:44:55:66 AP43     US-CAMPUS-01   connected

Showing 2 devices from mist-prod
```

### Example: Site Name in Multiple APIs

```
$ wifimgr show site US-CAMPUS-01

NAME           VENDOR   DEVICES   AP    SW    GW    API
US-CAMPUS-01   mist     45        32    12    1     mist-prod
US-CAMPUS-01   meraki   12        10    2     0     meraki-corp

Found site "US-CAMPUS-01" in 2 APIs
```

## Refresh Command Behavior

### Default: Refresh All APIs (Parallel)

```
$ wifimgr refresh cache

Refreshing all APIs...
  mist-prod: refreshing sites... done (12 sites)
  mist-prod: refreshing inventory-ap... done (156 items)
  meraki-corp: refreshing networks... done (8 networks)
  meraki-corp: refreshing inventory-ap... done (45 items)
  ...

Refreshed 2/2 APIs successfully
Rebuilt cross-API index
```

### Specific API Refresh

```
$ wifimgr refresh cache --api mist-prod

Refreshing mist-prod...
  sites: done (12 sites)
  inventory-ap: done (156 items)
  inventory-switch: done (42 items)
  ...

Refreshed mist-prod successfully
Rebuilt cross-API index
```

### Implementation

```go
func handleCacheRefresh(ctx context.Context, args []string) error {
    targetAPI := apiFlag

    if targetAPI != "" {
        // Validate API exists
        if !apiRegistry.HasAPI(targetAPI) {
            return fmt.Errorf("API %q not found", targetAPI)
        }
        return cacheManager.RefreshAPI(ctx, targetAPI)
    }

    // Refresh all APIs in parallel
    return cacheManager.RefreshAllAPIs(ctx)
}
```

## Search Command Behavior

### Default: Search All APIs

```
$ wifimgr search wireless laptop

Searching all APIs for "laptop"...

HOSTNAME        MAC               IP            AP              SITE           API
laptop-john     aa:bb:cc:dd:ee:ff 10.1.1.50    MIST-AP-01      US-CAMPUS-01   mist-prod
laptop-mary     11:22:33:44:55:66 10.1.1.51    MERAKI-AP-01    US-CAMPUS-01   meraki-corp

Found 2 results across 2 APIs
```

### Specific API Search

```
$ wifimgr search wireless laptop --api mist-prod

Searching mist-prod for "laptop"...

HOSTNAME     MAC               IP           AP           SITE
laptop-john  aa:bb:cc:dd:ee:ff 10.1.1.50   MIST-AP-01   US-CAMPUS-01

Found 1 result in mist-prod
```

### Implementation

```go
func handleSearch(ctx context.Context, searchType, text string) error {
    targetAPI := apiFlag

    if targetAPI != "" {
        return searchSingleAPI(ctx, targetAPI, searchType, text)
    }

    // Search all APIs
    var allResults []SearchResult
    for _, label := range apiRegistry.GetAllLabels() {
        client, _ := apiRegistry.GetClient(label)

        search := client.Search()
        if search == nil {
            logging.Debugf("API %s does not support search", label)
            continue
        }

        results, err := performSearch(ctx, search, searchType, text)
        if err != nil {
            logging.Warnf("Search failed for %s: %v", label, err)
            continue
        }

        // Tag results with source
        for _, r := range results {
            r.SourceAPI = label
        }
        allResults = append(allResults, results...)
    }

    displaySearchResults(allResults)
    return nil
}
```

## Apply Command Behavior

Apply commands use the API from site/device configuration:

```go
func handleApplySite(ctx context.Context, siteName string) error {
    // Load site config
    siteConfig, err := loadSiteConfig(siteName)
    if err != nil {
        return err
    }

    // Use site's configured API
    apiLabel := siteConfig.API

    // If --api flag provided, warn about override
    if apiFlag != "" && apiFlag != apiLabel {
        logging.Warnf("Overriding site API %q with --api %q", apiLabel, apiFlag)
        apiLabel = apiFlag
    }

    client, err := apiRegistry.GetClient(apiLabel)
    if err != nil {
        return err
    }

    // Resolve site ID
    siteID, err := siteResolver.ResolveSiteID(ctx, siteName, apiLabel)
    if err != nil {
        return err
    }

    // Apply configuration
    return applySiteConfig(ctx, client, siteID, siteConfig)
}
```

## Capability-Aware Commands

Commands check vendor capabilities before executing:

```go
func handleShowProfiles(ctx context.Context) error {
    targetAPI := apiFlag

    if targetAPI != "" {
        client, err := apiRegistry.GetClient(targetAPI)
        if err != nil {
            return err
        }

        profiles := client.Profiles()
        if profiles == nil {
            vendor, _ := apiRegistry.GetVendor(targetAPI)
            return fmt.Errorf("device profiles not supported by %s (%s)", targetAPI, vendor)
        }

        return showProfilesForClient(ctx, profiles, targetAPI)
    }

    // Show from all APIs that support profiles
    var found bool
    for _, label := range apiRegistry.GetAllLabels() {
        client, _ := apiRegistry.GetClient(label)
        profiles := client.Profiles()
        if profiles == nil {
            continue
        }
        found = true
        showProfilesForClient(ctx, profiles, label)
    }

    if !found {
        return fmt.Errorf("no configured APIs support device profiles")
    }
    return nil
}
```

## Error Messages

### Unknown API Label

```
$ wifimgr show api ap --api typo-api
Error: API "typo-api" not found

Available APIs: mist-prod, mist-lab, meraki-corp
```

### Unsupported Capability

```
$ wifimgr show profiles --api meraki-corp
Error: device profiles not supported by meraki-corp (meraki vendor)

This feature is only available for: mist
```

### Site Name Ambiguity (Information, Not Error)

```
$ wifimgr show site US-CAMPUS-01

NAME           VENDOR   DEVICES   API
US-CAMPUS-01   mist     45        mist-prod
US-CAMPUS-01   meraki   12        meraki-corp

Found site "US-CAMPUS-01" in 2 APIs
Tip: Use --api to filter to a specific API
```

## Import Commands

Import commands export API state to local config files for infrastructure-as-code management.

### Import Site Configuration

```
$ wifimgr import api site US-SFO-LAB

Exported site configuration to: ./config/mist-prod/sites/US-SFO-LAB.json
```

### Import with Specific Scope

```
$ wifimgr import api site US-SFO-LAB type ap        # Only access points
$ wifimgr import api site US-SFO-LAB type switch    # Only switches
$ wifimgr import api site US-SFO-LAB type gateway   # Only gateways
$ wifimgr import api site US-SFO-LAB type wlans     # Only WLANs
$ wifimgr import api site US-SFO-LAB type profiles  # Only site-specific profiles
```

### Compare Mode (with jsondiff)

```
$ wifimgr import api site US-SFO-LAB compare

Configuration differences for site 'US-SFO-LAB':
--- Local Config
+++ API Cache

  "devices": {
    "ap": {
-     "radio_config": { "band_5": { "channel": 36 } }
+     "radio_config": { "band_5": { "channel": 40 } }
    }
  }
```

### Import with Secrets

By default, sensitive data (PSK, RADIUS secrets) are redacted:

```
$ wifimgr import api site US-SFO-LAB secrets    # Include sensitive data
```

### Import from Specific API

```
$ wifimgr import api site US-SFO-LAB --api mist-prod
```

### Output Structure

Files are written to:
- Sites: `./config/<api-name>/sites/<site-name>.json`
- Global profiles: `./config/<api-name>/profiles/<profile-type>.json`

## Summary

| Scenario | Behavior |
|----------|----------|
| Read without `--api` | Aggregate from all APIs |
| Read with `--api` | Filter to specified API |
| Write without `--api` | Use config's API |
| Write with `--api` | Override with warning |
| Import without `--api` | Use site's source API |
| Import with `--api` | Use specified API |
| Same site name, multiple APIs | Show all, suggest `--api` |
| Capability not supported | Error with explanation |
| Unknown API label | Error with available list |

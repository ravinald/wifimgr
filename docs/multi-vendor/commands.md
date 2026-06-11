# Command Behavior

## CLI Disambiguation Strategy

With multiple APIs configured, commands that target remote resources need clear behavior:

**Principle:** Read operations aggregate by default; write operations use explicit API from config.

| Operation Type | Default Behavior | With `target` |
|----------------|------------------|---------------|
| Read (show, search) | Aggregate from all APIs | Filter to specific API |
| Write (apply, set) | Use site/device config API | Override (with warning) |

## Targeting a Specific API

Use the `target` positional keyword to direct a command at a specific API:

```
wifimgr show ap target mist-prod
```

## Command Behavior Matrix

### Read Commands

| Command | Default Behavior | With `target` |
|---------|------------------|---------------|
| `show ap` | Aggregate from all APIs | Filter to specific API |
| `show api bssid` | Aggregate from all APIs | Filter to specific API |
| `show sites` | Aggregate from all APIs | Filter to specific API |
| `show switch` | Aggregate from all APIs | Filter to specific API |
| `show api wlans` | Aggregate from all APIs | Filter to specific API |
| `show api rf-profiles` | Aggregate from all APIs | Filter to specific API |
| `show api device-profiles` | Aggregate from all APIs | Filter to specific API |
| `show site <name>` | Show from all APIs with that name | Show from specific API |
| `search wired <text>` | Search all APIs | Search specific API |
| `search wireless <text>` | Search all APIs | Search specific API |
| `refresh` | Managed devices, all APIs (parallel) | `site <name>` scopes to one site; `target <label>` filters to one API |
| `refresh all` | Everything the API has, all sites + client detail | `site <name>` runs for one site only; `target <label>` filters to one API |
| `refresh client site <name>` | Refresh per-client detail for one site | `target <label>` disambiguates when site exists in multiple APIs |

### Write Commands

| Command | Default Behavior | With `target` |
|---------|------------------|---------------|
| `apply site <name>` | Use site config's API | Override (warns) |
| `apply ap <site>` | Use site config's API | Override (warns) |

### Intent Commands (Local Config)

| Command | Behavior |
|---------|----------|
| `show intent ap` | Local config only, no API involved |
| `show intent sites` | Local config only, no API involved |

## Applying WLANs to APs

The intent config is the same regardless of vendor — that's the point of the
vendor-agnostic UX. You declare WLAN template labels at up to three levels, and
`apply ap <site>` translates them to each vendor's native model.

| Level | Key | Purpose |
|-------|-----|---------|
| Create | `profiles.wlan` | WLANs to **create** at the site/network |
| Site-wide | `wlan` (site-level) | Apply to **all** APs by default |
| Per-AP | `devices.ap[mac].wlan` | Apply to **specific** APs (overrides the site default for that AP) |

Every label used at the site or per-AP level must be declared in `profiles.wlan`
and must resolve to a WLAN template. `apply` validates this before pushing.

### Site-wide and per-AP example

```json
{
  "config": {
    "sites": {
      "site1": {
        "site_config": { "name": "US-NYC-OFFICE", "api": "mist-prod" },
        "profiles": { "wlan": ["corp-secure", "guest-open", "iot-network"] },
        "wlan": ["corp-secure", "guest-open"],
        "devices": {
          "ap": {
            "aa:bb:cc:dd:ee:f1": { "name": "NYC-AP-LOBBY" },
            "aa:bb:cc:dd:ee:f2": { "name": "NYC-AP-WAREHOUSE", "wlan": ["iot-network"] }
          }
        }
      }
    }
  }
}
```

- **LOBBY** inherits the site default → broadcasts `corp-secure` + `guest-open`.
- **WAREHOUSE** has an explicit `wlan` → broadcasts only `iot-network` (the per-AP
  list replaces the site default for that AP, it does not merge).

Run it the same way for any vendor — the API comes from the site's `api` field:

```bash
wifimgr apply ap US-NYC-OFFICE diff      # preview first, always
wifimgr apply ap US-NYC-OFFICE           # push
```

### What each vendor does under the hood

The three-level intent config maps to each vendor's native availability model — wifimgr
never invents a mechanism the vendor doesn't already use.

| Vendor | Site-wide | Per-AP |
|--------|-----------|--------|
| **Mist** | WLAN object with `apply_to: "site"` | WLAN object with `apply_to: "aps"` and `ap_ids` resolved from the AP MACs (`devices.ap[mac].wlan`) |
| **Meraki** | SSID slot with `availableOnAllAps: true` (the native default) | SSID slot with real Meraki `availabilityTags` + `availableOnAllAps: false`, matched by the APs' own `tags` |
| **Ubiquiti** | Not supported (read-only Phase 1) | Not supported (read-only Phase 1) |

Ubiquiti Phase 1 is read-only via the Site Manager API; WLAN/SSID apply arrives in
Phase 2 on the Network API. Until then `apply ap` against a Ubiquiti site reports the
capability as unsupported.

### Meraki SSID assignment

Meraki SSIDs are network-wide and live in 15 fixed slots (0–14) — there's no per-AP SSID
object like Mist. wifimgr respects Meraki's native model rather than overlaying its own:

- **Default is all-APs.** A WLAN with no availability restriction is written with
  `availableOnAllAps: true` — exactly Meraki's default. No tags, nothing to maintain.
- **Restriction uses real tags.** To scope an SSID to a subset of APs, Meraki intersects
  the SSID's `availabilityTags` with each AP's `tags`. wifimgr preserves whatever tags
  already exist — it does not invent or manage a tag scheme of its own. The SSID's
  `availabilityTags` and `availableOnAllAps: false` ride in the WLAN's `meraki:` vendor
  block; the APs keep their `tags` through their device config.

So per-AP availability on Meraki is expressed entirely through real Meraki tags, captured
on import and pushed back verbatim on apply — a functional no-op. Managing membership
going forward means editing the AP's `tags` (keep `tags` in `managed_keys.ap`) and the
SSID's `availabilityTags`.

#### The Meraki vendor block

`import api site` captures everything Meraki-specific that the portable WLAN template
can't represent losslessly into a `meraki:` block, so a re-apply changes nothing:

```json
"office--corp-secure": {
  "ssid": "CorpNet", "enabled": true, "band": "dual", "auth": { "type": "eap" },
  "meraki:": {
    "number": 3,
    "band": "Dual band operation with Band Steering",
    "auth": { "type": "8021x-radius" },
    "availabilityTags": ["lobby"],
    "availableOnAllAps": false
  }
}
```

- **`number`** pins the SSID slot. On apply it's the join key: wifimgr updates **slot 3**
  in place rather than matching on SSID name, so **renaming the SSID edits its own slot**
  instead of consuming a fresh one and orphaning the old. Resolution order is pinned slot
  → SSID-name match → allocate a free slot (only for genuinely new WLANs).
- **`band` / `auth.type`** carry the exact Meraki tokens. The portable fields stay
  canonical (`"dual"`, `"eap"`) for cross-vendor readability, but canonical can't tell
  Meraki's `"Dual band operation"` from `"Dual band operation with Band Steering"`, so the
  block holds the raw value and apply uses it. They appear only when canonicalization
  would otherwise lose information.
- **`availabilityTags` / `availableOnAllAps`** are the real availability model, preserved
  verbatim.

The `meraki:` block is Meraki-only; Mist WLANs are first-class objects keyed by stable
UUID and assigned to APs via `ap_ids`, so they need no block.

## Aggregation Display

When showing data from multiple APIs, include source columns:

### Example: Show APs from All APIs

```
$ wifimgr show ap

NAME            MAC               MODEL    SITE           STATUS      API
MIST-AP-01      aa:bb:cc:dd:ee:ff AP43     US-CAMPUS-01   connected   mist-prod
MIST-AP-02      11:22:33:44:55:66 AP43     US-CAMPUS-01   connected   mist-prod
MERAKI-AP-01    77:88:99:aa:bb:cc MR46     US-CAMPUS-01   online      meraki-corp
MERAKI-AP-02    dd:ee:ff:00:11:22 MR46     EU-OFFICE-01   online      meraki-corp

Showing 4 devices from 2 APIs
```

### Example: Show Filtered to Single API

```
$ wifimgr show ap target mist-prod

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
$ wifimgr refresh

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
$ wifimgr refresh target mist-prod

Refreshing mist-prod...
  sites: done (12 sites)
  inventory-ap: done (156 items)
  inventory-switch: done (42 items)
  ...

Refreshed mist-prod successfully
Rebuilt cross-API index
```

### Site-Scoped Refresh

`refresh site <name>` keeps the cheap org-scoped fetches (sites, inventory,
templates, statuses, WLANs) but limits the expensive per-device config loops to
the site's **managed** devices. Configs for everything else are preserved from
the prior cache, so the saved file is not a regression. Add `all` to fetch every
device the API reports for the site instead of just the managed ones.

This matters most on Meraki, where each device's config is its own API call.

```
$ wifimgr refresh site US-LAB-01

Refreshing cache for US-LAB-01 (meraki-corp)...
  [meraki-corp] Refreshing meraki API (site US-LAB-01)...
    Fetching sites... 12 sites
    Fetching APs... 156 devices
    ...
    Fetching AP configs (site US-LAB-01)... 8 fetched, 148 preserved
  [meraki-corp] Complete in 2103ms
Successfully refreshed meraki-corp for site US-LAB-01
```

If the same site name exists in more than one configured API, add `target <label>`:

```
$ wifimgr refresh site US-LAB-01 target meraki-corp
```

Add `detail` to also pull per-client detail for the site, or `all` to fetch
every device the API has for the site plus client detail.

> **Note:** the `api <api-label>` selector keyword has been removed from refresh.
> Use `target <api-label>` (matching `show`). `refresh device` and the old
> `refresh all` subcommand are gone — `refresh` is the managed default and
> `refresh all` now means the full org pull.

### Implementation

```go
func handleCacheRefresh(ctx context.Context, args []string) error {
    targetAPI := parsedArgs.Target

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
$ wifimgr search wireless laptop target mist-prod

Searching mist-prod for "laptop"...

HOSTNAME     MAC               IP           AP           SITE
laptop-john  aa:bb:cc:dd:ee:ff 10.1.1.50   MIST-AP-01   US-CAMPUS-01

Found 1 result in mist-prod
```

### List every client on a site

Drop the search text and pass a site to see every wireless or wired client the
vendor knows about. The site value can be either a site name from your cache or
the vendor's own site/network ID.

```
$ wifimgr search wireless site "MX - Av. Ejercito Nacional Mexicano 904"
$ wifimgr search wireless site L_3732358191183298569
$ wifimgr search wired site US-LAB-01 json
```

Names are resolved per-API through the same cache `show` uses, so a name that
lives in one API won't bleed into another. If the name doesn't match anything,
the value is passed through as an ID — handy when you're pasting a Meraki
`L_xxx` or a Mist UUID straight from the dashboard.

Running `search wireless` with no text and no site is an error:

```
$ wifimgr search wireless
Error: specify a search term or a site to list all clients
```

### Implementation

```go
func handleSearch(ctx context.Context, searchType, text string) error {
    targetAPI := parsedArgs.Target

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

    // If target provided, warn about override
    if parsedArgs.Target != "" && parsedArgs.Target != apiLabel {
        logging.Warnf("Overriding site API %q with target %q", apiLabel, parsedArgs.Target)
        apiLabel = parsedArgs.Target
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
    targetAPI := parsedArgs.Target

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
$ wifimgr show ap target typo-api
Error: API "typo-api" not found

Available APIs: mist-prod, mist-lab, meraki-corp
```

### Unsupported Capability

```
$ wifimgr show profiles target meraki-corp
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
Tip: Use target to filter to a specific API
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
$ wifimgr import api site US-SFO-LAB target mist-prod
```

### Output Structure

Files are written to:
- Sites: `./config/<api-name>/sites/<site-name>.json`
- Global profiles: `./config/<api-name>/profiles/<profile-type>.json`

## Summary

| Scenario | Behavior |
|----------|----------|
| Read without `target` | Aggregate from all APIs |
| Read with `target` | Filter to specified API |
| Write without `target` | Use config's API |
| Write with `target` | Override with warning |
| Import without `target` | Use site's source API |
| Import with `target` | Use specified API |
| Same site name, multiple APIs | Show all, suggest `target` |
| Capability not supported | Error with explanation |
| Unknown API label | Error with available list |

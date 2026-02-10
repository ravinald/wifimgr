# Multi-Vendor User Guide

This guide explains how to configure and use wifimgr with multiple vendors (Mist and Meraki).

## Quick Start

### 1. Configure Your APIs

Create or update your config file with multiple API connections:

```yaml
api:
  # Mist production environment
  mist-prod:
    vendor: mist
    url: https://api.mist.com
    credentials:
      org_id: "your-mist-org-uuid"
      api_key: "your-mist-api-token"

  # Meraki production environment
  meraki-prod:
    vendor: meraki
    url: https://api.meraki.com/api/v1
    credentials:
      org_id: "your-meraki-org-id"
      api_key: "your-meraki-api-key"

files:
  cache: ./cache
  config: ./config
```

### 2. Refresh the Cache

```bash
wifimgr refresh cache
```

### 3. View Your Infrastructure

```bash
# See all sites across all vendors
wifimgr show api sites

# See all APs across all vendors
wifimgr show api ap

# Filter to a specific vendor
wifimgr show api ap target mist-prod
```

## Configuration Reference

### API Labels

API labels are user-defined identifiers for your API connections. Choose meaningful names:

| Example Label   | Use Case                     |
|-----------------|------------------------------|
| `mist-prod`     | Mist production organization |
| `mist-lab`      | Mist lab/test organization   |
| `meraki-corp`   | Meraki corporate networks    |
| `meraki-retail` | Meraki retail locations      |

### Vendor Configuration

#### Mist

```yaml
api:
  mist-prod:
    vendor: mist
    url: https://api.mist.com           # or https://api.eu.mist.com for EU
    credentials:
      org_id: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
      api_key: "your-api-token"
    rate_limit: 5000                     # optional, requests per hour
    results_limit: 100                   # optional, results per page
```

**Getting Mist Credentials:**
1. Log into Mist dashboard
2. Go to Organization -> Settings -> API Token
3. Create a new token with appropriate permissions
4. Copy the Organization ID from the URL or Settings page

#### Meraki

```yaml
api:
  meraki-prod:
    vendor: meraki
    url: https://api.meraki.com/api/v1
    credentials:
      org_id: "123456"
      api_key: "your-api-key"
    rate_limit: 10                       # optional, requests per second
```

**Getting Meraki Credentials:**
1. Log into Meraki dashboard
2. Go to Organization -> Settings -> Dashboard API access
3. Enable API access and generate a key
4. Note your Organization ID from the URL

### Environment Variables

Credentials can be provided via environment variables using the pattern:
- `WIFIMGR_API_<LABEL>_CREDENTIALS_KEY` - API key/token
- `WIFIMGR_API_<LABEL>_CREDENTIALS_ORG` - Organization ID
- `WIFIMGR_API_<LABEL>_CREDENTIALS_URL` - API base URL (optional override)

**Note:** Label dashes are converted to underscores (e.g., `mist-prod` -> `MIST_PROD`)

```bash
# For API labeled "mist-prod"
export WIFIMGR_API_MIST_PROD_CREDENTIALS_KEY="your-mist-api-token"
export WIFIMGR_API_MIST_PROD_CREDENTIALS_ORG="your-mist-org-id"
export WIFIMGR_API_MIST_PROD_CREDENTIALS_URL="https://api.eu.mist.com"  # optional

# For API labeled "meraki-corp"
export WIFIMGR_API_MERAKI_CORP_CREDENTIALS_KEY="your-meraki-api-key"
export WIFIMGR_API_MERAKI_CORP_CREDENTIALS_ORG="your-meraki-org-id"
```

Or use a `.env.wifimgr` file:
```bash
# .env.wifimgr file
WIFIMGR_API_MIST_PROD_CREDENTIALS_KEY=your-mist-api-token
WIFIMGR_API_MIST_PROD_CREDENTIALS_ORG=your-mist-org-id
WIFIMGR_API_MIST_PROD_CREDENTIALS_URL=https://api.eu.mist.com
WIFIMGR_API_MERAKI_CORP_CREDENTIALS_KEY=your-meraki-api-key
WIFIMGR_API_MERAKI_CORP_CREDENTIALS_ORG=your-meraki-org-id
```

**Legacy support:** Single-API configs still support `WIFIMGR_API_TOKEN` and `WIFIMGR_ORG_ID`

## Targeting a Specific API

The `target` keyword filters commands to a specific API connection. It is a positional keyword placed after other arguments.

### Read Commands

Without `target`, commands aggregate results from all APIs:

```bash
# Show APs from ALL configured APIs
wifimgr show api ap

# Output includes API column:
# NAME          MAC                MODEL   SITE         STATUS      API
# AP-Floor1     aa:bb:cc:dd:ee:ff  AP43    US-LAB-01    connected   mist-prod
# AP-Lobby      11:22:33:44:55:66  MR46    US-LAB-01    online      meraki-prod
```

With `target`, results are filtered:

```bash
# Show APs from Mist only
wifimgr show api ap target mist-prod

# Show sites from Meraki only
wifimgr show api sites target meraki-prod
```

### Write Commands

Write commands (apply, set) use the API from your site configuration:

```yaml
# Site config (config/sites/US-LAB-01.yaml)
site_config:
  name: US-LAB-01
  api: mist-prod        # This site uses Mist
  timezone: America/Los_Angeles
```

```bash
# Applies to mist-prod (from site config)
wifimgr apply site US-LAB-01

# Override with warning
wifimgr apply site US-LAB-01 target meraki-prod
# WARNING: Overriding site API 'mist-prod' with target 'meraki-prod'
```

### Cache Refresh

```bash
# Refresh all APIs in parallel
wifimgr refresh cache

# Refresh only one API
wifimgr refresh cache target mist-prod
```

## Handling Duplicate Site Names

If the same site name exists in multiple APIs:

```bash
$ wifimgr show api ap site US-CAMPUS-01

# Shows results from both APIs:
# NAME       MAC                MODEL   STATUS      API
# AP-01      aa:bb:cc:dd:ee:ff  AP43    connected   mist-prod
# AP-02      11:22:33:44:55:66  MR46    online      meraki-prod
#
# Found site "US-CAMPUS-01" in 2 APIs
# Tip: Use target <label> to filter to a specific API
```

To target a specific API:

```bash
wifimgr show api ap site US-CAMPUS-01 target mist-prod
```

## Capability Differences

Most features are available across both vendors. Some advanced features are vendor-specific:

| Feature                | Mist   | Meraki   |
|------------------------|--------|----------|
| Sites/Networks         | ✓      | ✓        |
| Inventory              | ✓      | ✓        |
| Devices                | ✓      | ✓        |
| Device Configs         | ✓      | ✓        |
| Wireless Client Search | ✓      | ✓        |
| Wired Client Search    | ✓      | ✓        |
| Device Profiles        | ✓      | -        |
| RF Templates           | ✓      | -        |
| Gateway Templates      | ✓      | -        |
| WLAN Templates         | ✓      | -        |
| Gateway/Switch Apply   | -      | -        |

**Note:** Gateway and Switch apply functionality is planned for future releases. Currently, apply operations support APs on both vendors.

### Search on Both Platforms

Both Mist and Meraki support client search:

```bash
# Search on Mist
wifimgr search wireless laptop target mist-prod

# Search on Meraki
wifimgr search wireless laptop target meraki-corp

# Search across all APIs (tries each one)
wifimgr search wireless john
```

When searching across multiple networks/sites without a filter, the command estimates the API cost and prompts for confirmation.

## Checking API Status

View all configured APIs and their capabilities:

```bash
$ wifimgr show api status

API          Vendor   Cache       LastRefresh          Age      Capabilities
mist-prod    mist     ok          2024-01-27T14:30:00  2h15m    sites, inventory, devices, search, configs
meraki-prod  meraki   stale       2024-01-26T08:15:00  30h10m   sites, inventory, devices, search, configs
```

Use this to identify which APIs are healthy and when to refresh cache.

## Common Workflows

### View All Infrastructure

```bash
# All sites
wifimgr show api sites

# All APs
wifimgr show api ap

# All switches
wifimgr show api switch

# All inventory
wifimgr show inventory
```

### Find a Device by MAC

```bash
# Search inventory across all APIs
wifimgr show inventory ap aa:bb:cc:dd:ee:ff
```

### Search for Connected Clients

Search works on both Mist and Meraki networks:

```bash
# Search all APIs for a client (tries each sequentially)
wifimgr search wireless laptop-john
wifimgr search wired desktop-jane

# Search specific API
wifimgr search wireless john target mist-prod
wifimgr search wireless john target meraki-corp

# Scope to specific site (faster - single API call)
wifimgr search wireless john site US-LAB-01

# MAC address search (optimized for single API call)
wifimgr search wireless aa:bb:cc:dd:ee:ff

# Bypass expensive search confirmations
wifimgr search wireless john force
```

**Performance notes:**
- **MAC searches** are fast (single API call, org-wide)
- **Hostname searches without site filter** require one API call per network/site
- Use `site` filter to speed up searches on specific sites

### Apply Configuration

```bash
# Apply site configuration (uses API from site config)
wifimgr apply site US-LAB-01

# Apply just APs
wifimgr apply ap US-LAB-01

# Preview changes first
wifimgr apply site US-LAB-01 diff
```

## Troubleshooting

### "API not found" Error

```
Error: API 'mist-prod' not found

Available APIs: mist-lab, meraki-prod
```

**Solution:** Check your config file for typos in the API label.

### "Search Cost Estimation" Confirmation

When searching without a site filter, the command estimates cost and prompts for confirmation:

```
This search will query multiple networks (estimated 5 API calls).
Continue? [y/N]
```

You can bypass this confirmation with the `force` argument:

```bash
wifimgr search wireless john force
```

### "Site not found" Error

```
Error: site 'US-LAB-01' not found in any API

Searched APIs: mist-prod, meraki-prod

Try:
  - Refresh the cache: wifimgr refresh cache
  - Check site name spelling
  - Use target <label> to filter to a specific API
```

**Solution:**
1. Run `wifimgr refresh cache` to update the cache
2. Check the exact site name with `wifimgr show api sites`
3. Verify the site exists in the vendor dashboard

### Stale Cache

If data seems outdated:

```bash
# Refresh all caches
wifimgr refresh cache

# Or refresh a specific API
wifimgr refresh cache target mist-prod
```

### Rate Limiting

If you see rate limit errors:
- Meraki: Default 10 requests/second
- Mist: Default 5000 requests/hour

The tool automatically handles rate limiting with backoff, but heavy operations may take time.

## Example Configuration Files

### Dual Vendor Setup

```yaml
# config/wifimgr.yaml
api:
  mist-hq:
    vendor: mist
    url: https://api.mist.com
    credentials:
      org_id: "abc-123"
      api_key: "${MIST_TOKEN}"

  meraki-branches:
    vendor: meraki
    credentials:
      org_id: "456789"
      api_key: "${MERAKI_KEY}"

files:
  cache: ./cache
  config: ./config
```

### Site Config with API Binding

```yaml
# config/sites/US-HQ-01.yaml
site_config:
  name: US-HQ-01
  api: mist-hq
  timezone: America/Los_Angeles

  devices:
    ap:
      - name: HQ-AP-01
        mac: aa:bb:cc:dd:ee:ff
      - name: HQ-AP-02
        mac: 11:22:33:44:55:66
```

```yaml
# config/sites/US-BRANCH-01.yaml
site_config:
  name: US-BRANCH-01
  api: meraki-branches
  timezone: America/New_York

  devices:
    ap:
      - name: BRANCH-AP-01
        mac: 77:88:99:aa:bb:cc
```

# Multi-Vendor Architecture Analysis

This writeup consolidates the review of `docs/multi-vendor-architecture.md` together with observations gleaned from the current codebase (notably `api/client.go`). The goal is to highlight potential risks/assumptions for the multi-vendor effort, list open questions that need product/engineering decisions, and spell out concrete options for restructuring the vendor client abstraction layer.

## Current Architecture Snapshot

- **Single-vendor client contract** – `api.Client` is explicitly described as “the interface for Mist API operations” and contains ~30 Mist-centric methods (`api/client.go:12`). The interface is tightly bound to Mist concepts such as device profiles, RF templates, cache rebuild helpers, and Mist-specific search APIs.
- **Vendor plan** – The multi-vendor doc proposes adding Meraki as the second vendor by reusing the existing `api.Client` abstraction, introducing config structures keyed by API labels, and orchestrating cache refreshes per API.
- **Config semantics** – Site configs gain a required `api` field that denotes the default vendor connection; devices can override this so mixed environments are supported (`docs/multi-vendor-architecture.md:112`).
- **Cache semantics** – Cache contents become nested under `apis.<label>` with shared cross-indexes, and refresh commands can run per-API or in parallel (`docs/multi-vendor-architecture.md:298, 688`).

## Concerns & Open Questions

### 1. CLI/API Disambiguation

- Site configs solve ambiguity for intent-driven workflows because every site points to a specific API, but ad-hoc CLI commands (e.g., `wifimgr show site US-CAMPUS-01`) still rely on a resolver that might find the same site name under two different vendors.
- **Question:** Should every command that targets remote resources require `--api`, or should the resolver refuse to pick a site when duplicates exist and force operators to specify the API label?

### 2. Cache Layout & Concurrency

- `refreshAllAPIs` launches goroutines per API label (`docs/multi-vendor-architecture.md:688`), yet the cache description implies a single JSON file with nested vendor sections. Concurrent writes to the same file risk corruption or interleaved `_meta` updates.
- **Proposal:** Store each API label in its own cache file (e.g., `cache/apis/<label>.json`) and maintain a lightweight aggregated index file for cross-API lookups. This removes write contention, enables selective invalidation, and simplifies telemetry on per-API refreshes.
- **Question:** Do we also want per-site caches? Unless intent artifacts need caching separately, per-API cache files should be sufficient.

### 3. Client/Interface Contract

- Expecting Meraki (or future vendors) to implement `api.Client` as-is forces each vendor to support Mist-only methods or ship no-op stubs, leaving commands to fail at runtime. This is the largest architectural risk because it impacts every CLI surface.
- **Question:** Which strategy do we prefer for defining a cross-vendor client contract? Three options are outlined below with concrete examples.

## Client/Interface Strategy Options

### Option A – Split `api.Client` into Capability-Focused Interfaces

**Idea:** Break the monolithic Mist interface into cohesive “service” slices (Sites, Inventory, DeviceProfiles, Config, Search, etc.). Commands and orchestrators depend only on the slices they need. Vendors implement only the slices they can support.

```go
// Capability slices
type SitesService interface {
	GetSites(ctx context.Context, orgID string) ([]*api.SiteNew, error)
	GetSiteByName(ctx context.Context, name, orgID string) (*api.SiteNew, error)
	CreateSite(ctx context.Context, site *api.SiteNew) (*api.SiteNew, error)
	// ...
}

type InventoryService interface {
	GetInventory(ctx context.Context, orgID, deviceType string) ([]*api.InventoryItemNew, error)
	// ...
}

type ProfilesService interface {
	GetDeviceProfiles(ctx context.Context, orgID, profileType string) ([]api.DeviceProfile, error)
	// Mist-specific
}

// Top-level vendor client exposing supported slices.
type VendorClient interface {
	Sites() SitesService
	Inventory() InventoryService
	Profiles() ProfilesService // Returns nil or panics if not supported
}
```

- **Pros**
  - Compile-time safety: if a command imports `ProfilesService`, only vendors that implement it will compile, forcing explicit gating.
  - Vendors implement only the slices they support; Meraki can skip Mist-only concepts.
  - Clear growth path when new capabilities appear (add a new interface, not 30+ methods).
- **Cons**
  - Significant refactor: all commands and Mist code must migrate to the new service slices.
  - Cross-cutting concerns (rate limiting, cache hooks, config getters) need re-homing.
  - Short-term duplication: wrappers or adapters may be needed while commands migrate.

### Option B – Capability Registry on Top of Existing `api.Client`

**Idea:** Keep `api.Client` untouched, but annotate each API label with a `VendorCapabilities` struct so commands can check whether a feature exists before invoking Mist-only methods.

```go
type VendorCapabilities struct {
	SupportsDeviceProfiles bool
	SupportsAPConfig       bool
	SupportsSiteSettings   bool
	// ...
}

type APIEndpoint struct {
	Label        string
	Client       api.Client
	Capabilities VendorCapabilities
}

// Command example
func runShowProfiles(apiLabel string) error {
	endpoint := registry.MustGet(apiLabel)
	if !endpoint.Capabilities.SupportsDeviceProfiles {
		return fmt.Errorf("device profiles unavailable for API %q", apiLabel)
	}
	// Safe to call Mist-specific methods
	profiles, err := endpoint.Client.GetDeviceProfiles(...)
	// ...
}
```

- **Pros**
  - Minimal change footprint; no immediate refactor of commands or Mist implementation.
  - Fastest path to multi-vendor support while we prototype Meraki integrations.
  - Capabilities surface clearly for UX/logging (e.g., skip menu options for unsupported vendors).
- **Cons**
  - Meraki must still provide implementations (or stubs) for every method on `api.Client`.
  - Runtime errors remain possible if a command forgets to check a capability.
  - The Mist-centric interface persists, making long-term maintenance harder.

### Option C – New `vendors.Client` Abstraction Layer

**Idea:** Introduce a smaller, vendor-agnostic client interface specifically for multi-vendor workflows. Mist implements this by wrapping the existing `api.Client`, while Meraki implements it natively. Commands migrate gradually to the new `vendors.Client`.

```go
// Multi-vendor contract
type vendorsClient interface {
	SitesAPI() SitesAPI
	InventoryAPI() InventoryAPI
	CacheAPI() CacheAPI
}

type SitesAPI interface {
	List(ctx context.Context) ([]*api.SiteNew, error)
	ResolveByName(ctx context.Context, name string) (*api.SiteNew, error)
}

// Mist adapter
type mistVendorClient struct {
	mist api.Client
}

func (m *mistVendorClient) SitesAPI() SitesAPI {
	return &mistSitesAPI{client: m.mist}
}

type mistSitesAPI struct{ client api.Client }

func (s *mistSitesAPI) List(ctx context.Context) ([]*api.SiteNew, error) {
	return s.client.GetSites(ctx, s.client.Config().OrgID)
}

// Meraki adapter implements the exact same interfaces using SDK calls.
```

- **Pros**
  - Lets us design a clean, purpose-built contract without ripping out Mist internals.
  - Commands can migrate gradually; only the migrated ones rely on the new abstraction.
  - Future vendors implement a focused surface area, not the entire Mist interface.
- **Cons**
  - Two client layers coexist until migration completes, increasing complexity.
  - Requires an adapter per vendor plus translation glue back into the legacy Mist client.
  - Commands interacting with still-unmigrated Mist-only features must keep using `api.Client`.

## Outstanding Questions for the Team

1. **CLI behavior when site names collide across vendors** – Enforce `--api` for remote operations, or make the resolver refuse to guess?
2. **Cache structure** – Is per-API cache storage sufficient, or do we want per-site caches for specific artifacts?
3. **Client strategy** – Which option above should we pursue so we can start the corresponding refactor/integration work?

Once we align on these questions, we can draft concrete implementation checkpoints (config schema updates, cache manager changes, CLI flags, etc.) and move the Meraki integration forward with lower risk.

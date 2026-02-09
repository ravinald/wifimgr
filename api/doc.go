// Package api provides the Mist Systems HTTP API client.
//
// # IMPORTANT: This is a Mist-Specific Package
//
// This package contains the Mist-specific API client implementation.
// For multi-vendor operations, use the vendors package instead:
//
//	import "github.com/ravinald/wifimgr/internal/vendors"
//
//	// Get vendor-agnostic client
//	client := cmd.GetVendorClient()
//	sites, _ := client.Sites().List(ctx)
//
//	// Access Mist-specific operations when needed
//	if legacyClient := cmd.GetLegacyClient(client); legacyClient != nil {
//	    profiles, _ := legacyClient.GetDeviceProfiles(ctx, orgID, "")
//	}
//
// # Package Structure
//
// The api package provides:
//   - Client interface: Main Mist API client interface
//   - Type definitions: Mist-specific types (MistSite, MistWLAN, etc.)
//   - HTTP operations: Direct Mist API HTTP calls
//
// # Legacy Cache System (Deprecated)
//
// This package contains legacy cache files (cache_*.go) that read from
// ~/.cache/wifimgr/cache.json. This cache file is NO LONGER POPULATED.
//
// For cache operations, use the multi-vendor cache system:
//
//	import "github.com/ravinald/wifimgr/internal/vendors"
//
//	// Get cache accessor for lookups
//	accessor := vendors.GetGlobalCacheAccessor()
//	site, _ := accessor.GetSiteByName("US-LAB-01")
//	ap, _ := accessor.GetAPByMAC("aa:bb:cc:dd:ee:ff")
//
// The multi-vendor cache reads from ~/.cache/wifimgr/apis/<label>.json
// which contains the actual cached data per API connection.
//
// # Migration Path
//
// New command handlers should use vendors.Client instead of api.Client:
//
//  1. Use cmd.GetVendorClient() for the default vendor client
//  2. Use vendors.Client methods (Sites(), Devices(), etc.) for operations
//  3. Use cmd.GetLegacyClient() only when Mist-specific methods are required
//
// See cmd/multivendor.go for detailed migration documentation.
package api

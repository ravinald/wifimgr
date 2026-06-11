package cmdutils

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ravinald/wifimgr/internal/vendors"
)

// uuidRe and siteCodeRe mirror the legacy utils.IsUUID / IsSiteCode rules so the
// cache-backed resolver accepts the same identifiers the Mist-only path did:
// a vendor UUID passes through, a site-code-shaped name is upper-cased.
var (
	uuidRe     = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	siteCodeRe = regexp.MustCompile(`^\w{2}-\w{3,4}-\w{1,10}$`)
)

// SiteRef is a site resolved against the multivendor cache: the owning API label
// plus the vendor-specific ID and display name. Commands carry it so a single
// name lookup fixes both which API to talk to and which site ID to operate on.
type SiteRef struct {
	APILabel string
	SiteID   string
	Name     string
}

// ResolveSite maps a site identifier to its owning API and vendor ID using the
// multivendor cache, replacing the Mist-only utils.ResolveSiteID. The identifier
// is a vendor UUID (passed through after a cache existence check) or a site name;
// a site-code-shaped name is upper-cased first. When apiLabel is empty the cache
// index picks the owning API, and a name present in more than one API is an error
// so the caller can re-run with an explicit target. Duplicate names within one
// API surface as *vendors.DuplicateSiteError.
func ResolveSite(identifier, apiLabel string) (*SiteRef, error) {
	accessor, err := GetCacheAccessor()
	if err != nil {
		return nil, err
	}

	// A UUID is already a site ID — resolve straight to its SiteInfo so we can
	// attach the owning API, guarding an explicit apiLabel if one was given.
	if uuidRe.MatchString(strings.ToLower(identifier)) {
		site, err := accessor.GetSiteByID(identifier)
		if err != nil {
			return nil, err
		}
		if apiLabel != "" && site.SourceAPI != apiLabel {
			return nil, fmt.Errorf("site %s belongs to API %q, not %q", identifier, site.SourceAPI, apiLabel)
		}
		return &SiteRef{APILabel: site.SourceAPI, SiteID: site.ID, Name: site.Name}, nil
	}

	return resolveSiteByName(accessor.GetManager(), identifier, apiLabel)
}

// resolveSiteByName resolves a non-UUID identifier against the cache manager.
// Split from ResolveSite so the duplicate-safe name logic is testable without
// the global accessor and its registry-backed indexes.
func resolveSiteByName(mgr *vendors.CacheManager, identifier, apiLabel string) (*SiteRef, error) {
	if mgr == nil {
		return nil, fmt.Errorf("cache manager not initialized")
	}

	name := identifier
	if siteCodeRe.MatchString(name) {
		name = strings.ToUpper(name)
	}

	// Caller named the API: resolve directly. GetSiteIDByName is duplicate-safe
	// within an API and returns *vendors.DuplicateSiteError on a collision.
	if apiLabel != "" {
		siteID, err := mgr.GetSiteIDByName(apiLabel, name)
		if err != nil {
			return nil, err
		}
		return &SiteRef{APILabel: apiLabel, SiteID: siteID, Name: name}, nil
	}

	// No API named: let the cross-API index choose, refusing ambiguity rather
	// than picking one — a write must not land on a same-named site in the
	// wrong vendor.
	apis := mgr.GetSiteAPIs(name)
	switch len(apis) {
	case 0:
		return nil, &vendors.SiteNotFoundError{SiteName: name}
	case 1:
		siteID, err := mgr.GetSiteIDByName(apis[0], name)
		if err != nil {
			return nil, err
		}
		return &SiteRef{APILabel: apis[0], SiteID: siteID, Name: name}, nil
	default:
		return nil, fmt.Errorf("site %q exists in multiple APIs (%s) - specify a target API",
			name, strings.Join(apis, ", "))
	}
}

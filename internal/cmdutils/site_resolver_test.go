package cmdutils

import (
	"errors"
	"strings"
	"testing"

	"github.com/ravinald/wifimgr/internal/vendors"
)

// newTestManager builds a cache manager seeded with the given per-API site
// lists and a rebuilt cross-API index, matching how the live cache is wired.
func newTestManager(t *testing.T, sitesByAPI map[string][]vendors.SiteInfo) *vendors.CacheManager {
	t.Helper()
	cm := vendors.NewCacheManager(t.TempDir(), vendors.NewAPIClientRegistry())
	if err := cm.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	for label, sites := range sitesByAPI {
		cache := vendors.NewAPICache(label, "mist", "org-"+label)
		cache.Sites.Info = sites
		if err := cm.SaveAPICache(cache); err != nil {
			t.Fatalf("SaveAPICache(%s): %v", label, err)
		}
	}
	if err := cm.RebuildIndex(); err != nil {
		t.Fatalf("RebuildIndex: %v", err)
	}
	return cm
}

func TestResolveSiteByName_Unique(t *testing.T) {
	cm := newTestManager(t, map[string][]vendors.SiteInfo{
		"mist-prod": {{ID: "site-1", Name: "US-LAB-01"}},
	})

	ref, err := resolveSiteByName(cm, "US-LAB-01", "")
	if err != nil {
		t.Fatalf("resolveSiteByName: %v", err)
	}
	if ref.APILabel != "mist-prod" || ref.SiteID != "site-1" {
		t.Errorf("got %+v, want APILabel=mist-prod SiteID=site-1", ref)
	}
}

// A site-code-shaped name resolves case-insensitively — the legacy resolver
// upper-cased site codes before lookup, and the cache stores the canonical form.
func TestResolveSiteByName_SiteCodeUpcased(t *testing.T) {
	cm := newTestManager(t, map[string][]vendors.SiteInfo{
		"mist-prod": {{ID: "site-1", Name: "US-LAB-01"}},
	})

	ref, err := resolveSiteByName(cm, "us-lab-01", "")
	if err != nil {
		t.Fatalf("resolveSiteByName: %v", err)
	}
	if ref.SiteID != "site-1" {
		t.Errorf("site code not upper-cased before lookup: got %+v", ref)
	}
}

// Two sites sharing a name in one API must refuse with DuplicateSiteError, even
// without an explicit target — the within-API collision is fatal either way.
func TestResolveSiteByName_DuplicateWithinAPI(t *testing.T) {
	cm := newTestManager(t, map[string][]vendors.SiteInfo{
		"mist-prod": {
			{ID: "site-1", Name: "DUP"},
			{ID: "site-2", Name: "DUP"},
		},
	})

	_, err := resolveSiteByName(cm, "DUP", "")
	var dupErr *vendors.DuplicateSiteError
	if !errors.As(err, &dupErr) {
		t.Fatalf("err = %v, want *vendors.DuplicateSiteError", err)
	}
	if dupErr.APILabel != "mist-prod" || dupErr.MatchCount != 2 {
		t.Errorf("got %+v, want APILabel=mist-prod MatchCount=2", dupErr)
	}
}

// The same name in two APIs without a target is ambiguous — refuse and name the
// candidates rather than guess.
func TestResolveSiteByName_AmbiguousAcrossAPIs(t *testing.T) {
	cm := newTestManager(t, map[string][]vendors.SiteInfo{
		"mist-prod":   {{ID: "site-1", Name: "SHARED"}},
		"meraki-prod": {{ID: "net-1", Name: "SHARED"}},
	})

	_, err := resolveSiteByName(cm, "SHARED", "")
	if err == nil {
		t.Fatal("expected ambiguity error, got nil")
	}
	for _, want := range []string{"multiple APIs", "mist-prod", "meraki-prod"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q should mention %q", err.Error(), want)
		}
	}
}

// An explicit target resolves the same name in the chosen API and ignores the
// other.
func TestResolveSiteByName_ExplicitTarget(t *testing.T) {
	cm := newTestManager(t, map[string][]vendors.SiteInfo{
		"mist-prod":   {{ID: "site-1", Name: "SHARED"}},
		"meraki-prod": {{ID: "net-1", Name: "SHARED"}},
	})

	ref, err := resolveSiteByName(cm, "SHARED", "meraki-prod")
	if err != nil {
		t.Fatalf("resolveSiteByName: %v", err)
	}
	if ref.APILabel != "meraki-prod" || ref.SiteID != "net-1" {
		t.Errorf("got %+v, want APILabel=meraki-prod SiteID=net-1", ref)
	}
}

func TestResolveSiteByName_NotFound(t *testing.T) {
	cm := newTestManager(t, map[string][]vendors.SiteInfo{
		"mist-prod": {{ID: "site-1", Name: "US-LAB-01"}},
	})

	_, err := resolveSiteByName(cm, "NOPE", "")
	var nf *vendors.SiteNotFoundError
	if !errors.As(err, &nf) {
		t.Fatalf("err = %v, want *vendors.SiteNotFoundError", err)
	}
}

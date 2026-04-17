package vendors

import (
	"fmt"
	"time"
)

// LookupClientDetail returns the cached detail record for one client in the
// given API, or (nil, false) if no record exists. Safe against a nil cache
// map and against the API cache failing to load.
func (c *CacheManager) LookupClientDetail(apiLabel, mac string) (*ClientDetail, bool) {
	cache, err := c.GetAPICache(apiLabel)
	if err != nil || cache == nil || cache.ClientDetail == nil {
		return nil, false
	}
	record, ok := cache.ClientDetail[NormalizeMAC(mac)]
	if !ok || record == nil {
		return nil, false
	}
	return record, true
}

// SaveClientDetail upserts detail records into the named API's cache and
// persists. Records are keyed by NormalizeMAC(r.MAC). Existing entries for
// the same MAC are overwritten. Returns the newest FetchedAt observed across
// the saved records so the caller can surface freshness to the user.
func (c *CacheManager) SaveClientDetail(apiLabel string, records []*ClientDetail) (time.Time, error) {
	cache, err := c.GetAPICache(apiLabel)
	if err != nil {
		return time.Time{}, fmt.Errorf("load cache for %s: %w", apiLabel, err)
	}
	if cache.ClientDetail == nil {
		cache.ClientDetail = make(map[string]*ClientDetail, len(records))
	}

	var newest time.Time
	for _, r := range records {
		if r == nil || r.MAC == "" {
			continue
		}
		key := NormalizeMAC(r.MAC)
		copy := *r // shallow copy — the struct holds no reference types
		copy.MAC = key
		cache.ClientDetail[key] = &copy
		if r.FetchedAt.After(newest) {
			newest = r.FetchedAt
		}
	}

	if err := c.SaveAPICache(cache); err != nil {
		return time.Time{}, fmt.Errorf("persist cache for %s: %w", apiLabel, err)
	}
	return newest, nil
}

// ClientDetailFreshness returns the newest FetchedAt across entries scoped
// to the given siteID, or (zero, false) if nothing is cached for that site.
// Pass an empty siteID to scan the whole API's cache.
func (c *CacheManager) ClientDetailFreshness(apiLabel, siteID string) (time.Time, bool) {
	cache, err := c.GetAPICache(apiLabel)
	if err != nil || cache == nil || len(cache.ClientDetail) == 0 {
		return time.Time{}, false
	}
	var newest time.Time
	found := false
	for _, r := range cache.ClientDetail {
		if r == nil {
			continue
		}
		if siteID != "" && r.SiteID != siteID {
			continue
		}
		if r.FetchedAt.After(newest) {
			newest = r.FetchedAt
		}
		found = true
	}
	return newest, found
}

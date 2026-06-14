package aruba

import (
	"context"
	"strings"

	"github.com/ravinald/wifimgr/internal/vendors"
)

type wlansService struct {
	client *Client
	siteID string
}

// List reads SSID profiles by parsing `show running-config`.
func (s *wlansService) List(ctx context.Context) ([]*vendors.WLAN, error) {
	out, err := s.client.ShowCommand(ctx, "show running-config")
	if err != nil {
		return nil, err
	}
	return extractWLANs(parseRunningConfig(out), s.siteID), nil
}

// ListBySite returns the same WLANs as List when the site matches this swarm.
func (s *wlansService) ListBySite(ctx context.Context, siteID string) ([]*vendors.WLAN, error) {
	if siteID != "" && siteID != s.siteID {
		return nil, nil
	}
	return s.List(ctx)
}

func (s *wlansService) Get(ctx context.Context, id string) (*vendors.WLAN, error) {
	wlans, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, w := range wlans {
		if w.ID == id {
			return w, nil
		}
	}
	return nil, &vendors.NotFoundError{APILabel: vendorName, Resource: "ssid-profile " + id}
}

func (s *wlansService) BySSID(ctx context.Context, ssid string) ([]*vendors.WLAN, error) {
	wlans, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	var matches []*vendors.WLAN
	for _, w := range wlans {
		if strings.EqualFold(w.SSID, ssid) {
			matches = append(matches, w)
		}
	}
	return matches, nil
}

// Create writes an SSID profile. Instant upserts by profile name, so this also
// serves Update.
func (s *wlansService) Create(ctx context.Context, wlan *vendors.WLAN) (*vendors.WLAN, error) {
	if err := s.client.PostObject(ctx, "ssid", ssidProfilePayload(wlan)); err != nil {
		return nil, err
	}
	return wlan, nil
}

func (s *wlansService) Update(ctx context.Context, id string, wlan *vendors.WLAN) (*vendors.WLAN, error) {
	if wlan.ID == "" {
		wlan.ID = id
	}
	return s.Create(ctx, wlan)
}

func (s *wlansService) Delete(ctx context.Context, id string) error {
	return s.client.PostObject(ctx, "ssid", ssidDeletePayload(id))
}

var _ vendors.WLANsService = (*wlansService)(nil)

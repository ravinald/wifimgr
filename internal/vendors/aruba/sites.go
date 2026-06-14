package aruba

import (
	"context"
	"strings"

	"github.com/ravinald/wifimgr/internal/vendors"
)

type sitesService struct {
	client *Client
	siteID string
}

// List returns the single site that represents this Instant swarm. Name and
// country come from the running-config globals; the VC host is the stable ID.
func (s *sitesService) List(ctx context.Context) ([]*vendors.SiteInfo, error) {
	info := &vendors.SiteInfo{
		ID:           s.siteID,
		Name:         s.siteID, // fallback; overwritten below if config names the swarm
		DeviceCount:  1,
		SourceVendor: vendorName,
	}

	if out, err := s.client.ShowCommand(ctx, "show running-config"); err == nil {
		blocks := parseRunningConfig(out)
		// The VC `name` is the swarm's identity and the key operators align to
		// their wifimgr site config, so it wins over organization/hostname.
		// Surfaced verbatim: the name is operator-chosen and arbitrary, and the
		// site resolver owns any case normalization.
		if name := firstNonEmpty(globalValue(blocks, "name"), globalValue(blocks, "hostname"), globalValue(blocks, "organization")); name != "" {
			info.Name = name
		}
		info.CountryCode = firstNonEmpty(
			globalValue(blocks, "virtual-controller-country"),
			globalValue(blocks, "country-code"),
		)
	} else {
		return nil, err
	}

	return []*vendors.SiteInfo{info}, nil
}

func (s *sitesService) Get(ctx context.Context, id string) (*vendors.SiteInfo, error) {
	sites, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, site := range sites {
		if site.ID == id {
			return site, nil
		}
	}
	return nil, &vendors.SiteNotFoundError{SiteName: id, APILabel: vendorName}
}

func (s *sitesService) ByName(ctx context.Context, name string) (*vendors.SiteInfo, error) {
	sites, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, site := range sites {
		if strings.EqualFold(site.Name, name) || strings.EqualFold(site.ID, name) {
			return site, nil
		}
	}
	return nil, &vendors.SiteNotFoundError{SiteName: name, APILabel: vendorName}
}

// A standalone Instant swarm is not created or destroyed through this API.
func (s *sitesService) Create(_ context.Context, _ *vendors.SiteInfo) (*vendors.SiteInfo, error) {
	return nil, &vendors.CapabilityNotSupportedError{Capability: "site creation", APILabel: vendorName, VendorName: vendorName}
}

func (s *sitesService) Update(_ context.Context, _ string, _ *vendors.SiteInfo) (*vendors.SiteInfo, error) {
	return nil, &vendors.CapabilityNotSupportedError{Capability: "site update", APILabel: vendorName, VendorName: vendorName}
}

func (s *sitesService) Delete(_ context.Context, _ string) error {
	return &vendors.CapabilityNotSupportedError{Capability: "site deletion", APILabel: vendorName, VendorName: vendorName}
}

var _ vendors.SitesService = (*sitesService)(nil)

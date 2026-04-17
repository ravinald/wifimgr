package ubiquiti

import (
	"context"
	"strings"

	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/vendors"
)

type sitesService struct {
	client *Client
}

func (s *sitesService) List(ctx context.Context) ([]*vendors.SiteInfo, error) {
	sites, err := s.client.GetSites(ctx)
	if err != nil {
		return nil, err
	}

	// Build host name map from multiple sources.
	// Start with hosts API (covers hosts that may have no devices yet),
	// then overlay device groups (which use the same compound hostId as sites).
	hostNameMap := make(map[string]string)

	hosts, err := s.client.GetHosts(ctx)
	if err != nil {
		logging.Debugf("[ubiquiti] Failed to fetch hosts for name enrichment: %v", err)
	} else {
		hostNameMap = buildHostNameMap(hosts)
		logging.Debugf("[ubiquiti] Built host name map with %d entries from hosts API", len(hostNameMap))
	}

	groups, err := s.client.GetDevices(ctx)
	if err != nil {
		logging.Debugf("[ubiquiti] Failed to fetch devices for host name enrichment: %v", err)
	} else {
		for k, v := range buildHostNameMapFromDevices(groups) {
			hostNameMap[k] = v
		}
		logging.Debugf("[ubiquiti] Host name map has %d entries after device group enrichment", len(hostNameMap))
	}

	result := make([]*vendors.SiteInfo, 0, len(sites))
	for _, site := range sites {
		result = append(result, convertSiteToSiteInfo(site, hostNameMap))
	}
	return result, nil
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
	return nil, &vendors.SiteNotFoundError{SiteName: id}
}

func (s *sitesService) ByName(ctx context.Context, name string) (*vendors.SiteInfo, error) {
	sites, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, site := range sites {
		if strings.EqualFold(site.Name, name) {
			return site, nil
		}
	}
	return nil, &vendors.SiteNotFoundError{SiteName: name}
}

func (s *sitesService) Create(_ context.Context, _ *vendors.SiteInfo) (*vendors.SiteInfo, error) {
	return nil, &vendors.CapabilityNotSupportedError{
		Capability: "site creation",
		VendorName: "ubiquiti",
	}
}

func (s *sitesService) Update(_ context.Context, _ string, _ *vendors.SiteInfo) (*vendors.SiteInfo, error) {
	return nil, &vendors.CapabilityNotSupportedError{
		Capability: "site update",
		VendorName: "ubiquiti",
	}
}

func (s *sitesService) Delete(_ context.Context, _ string) error {
	return &vendors.CapabilityNotSupportedError{
		Capability: "site deletion",
		VendorName: "ubiquiti",
	}
}

var _ vendors.SitesService = (*sitesService)(nil)

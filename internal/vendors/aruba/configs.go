package aruba

import (
	"context"
	"strings"

	"github.com/ravinald/wifimgr/internal/vendors"
)

type configsService struct {
	client *Client
	siteID string
}

// GetAPConfig returns the AP's current configuration parsed from
// `show running-config`. The Config map carries the swarm name and the radio
// profiles so the apply diff has a current-state baseline to compare against.
func (s *configsService) GetAPConfig(ctx context.Context, _, deviceID string) (*vendors.APConfig, error) {
	out, err := s.client.ShowCommand(ctx, "show running-config")
	if err != nil {
		return nil, err
	}
	blocks := parseRunningConfig(out)

	// deviceID is the inventory ID, which is the ethernet MAC once summary
	// enrichment runs. The device export keys APs by MAC, so carry it through.
	mac := normalizeMAC(deviceID)
	if !isHexMAC(mac) {
		mac = ""
	}

	// The per-AP name lives in `show summary`, not the running-config (which
	// only carries the swarm-wide `name`). Resolve it by MAC so the export keeps
	// each AP's operator-chosen name rather than repeating the swarm name.
	name := deviceID
	if mac != "" {
		if summary, sErr := s.client.ShowCommand(ctx, "show summary"); sErr == nil {
			if n := summaryAPNames(summary)[mac]; n != "" {
				name = n
			}
		}
	}

	cfg := map[string]any{"name": name}
	if radios := radioProfiles(blocks); len(radios) > 0 {
		cfg["radio_profiles"] = radios
	}

	return &vendors.APConfig{
		ID:           deviceID,
		Name:         name,
		MAC:          mac,
		SiteID:       s.siteID,
		Config:       cfg,
		SourceVendor: vendorName,
	}, nil
}

func (s *configsService) GetSwitchConfig(_ context.Context, _, _ string) (*vendors.SwitchConfig, error) {
	return nil, &vendors.CapabilityNotSupportedError{Capability: "switch config", APILabel: vendorName, VendorName: vendorName}
}

func (s *configsService) GetGatewayConfig(_ context.Context, _, _ string) (*vendors.GatewayConfig, error) {
	return nil, &vendors.CapabilityNotSupportedError{Capability: "gateway config", APILabel: vendorName, VendorName: vendorName}
}

// radioProfiles collects `rf ...-radio-profile` blocks as nested key/value maps.
func radioProfiles(blocks []configBlock) map[string]any {
	out := map[string]any{}
	for _, b := range blocks {
		if len(b.tokens) >= 2 && b.tokens[0] == "rf" && strings.Contains(b.tokens[1], "radio-profile") {
			profile := map[string]any{}
			for _, ln := range b.lines {
				profile[ln.key] = strings.Join(ln.args, " ")
			}
			out[b.header] = profile
		}
	}
	return out
}

var _ vendors.ConfigsService = (*configsService)(nil)

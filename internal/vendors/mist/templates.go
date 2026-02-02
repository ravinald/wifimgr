package mist

import (
	"context"
	"fmt"

	"github.com/ravinald/wifimgr/api"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// templatesService implements vendors.TemplatesService for Mist.
type templatesService struct {
	client api.Client
	orgID  string
}

// ListRF returns RF templates.
func (s *templatesService) ListRF(ctx context.Context) ([]*vendors.RFTemplate, error) {
	templates, err := s.client.GetRFTemplates(ctx, s.orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get RF templates: %w", err)
	}

	result := make([]*vendors.RFTemplate, 0, len(templates))
	for i := range templates {
		vt := convertRFTemplateToVendor(&templates[i])
		if vt != nil {
			result = append(result, vt)
		}
	}

	return result, nil
}

// ListGateway returns gateway templates.
func (s *templatesService) ListGateway(ctx context.Context) ([]*vendors.GatewayTemplate, error) {
	templates, err := s.client.GetGatewayTemplates(ctx, s.orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get gateway templates: %w", err)
	}

	result := make([]*vendors.GatewayTemplate, 0, len(templates))
	for i := range templates {
		vt := convertGatewayTemplateToVendor(&templates[i])
		if vt != nil {
			result = append(result, vt)
		}
	}

	return result, nil
}

// ListWLAN returns WLAN templates.
func (s *templatesService) ListWLAN(ctx context.Context) ([]*vendors.WLANTemplate, error) {
	templates, err := s.client.GetWLANTemplates(ctx, s.orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get WLAN templates: %w", err)
	}

	result := make([]*vendors.WLANTemplate, 0, len(templates))
	for i := range templates {
		vt := convertWLANTemplateToVendor(&templates[i])
		if vt != nil {
			result = append(result, vt)
		}
	}

	return result, nil
}

// Ensure templatesService implements vendors.TemplatesService at compile time.
var _ vendors.TemplatesService = (*templatesService)(nil)

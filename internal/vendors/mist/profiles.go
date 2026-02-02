package mist

import (
	"context"
	"fmt"

	"github.com/ravinald/wifimgr/api"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// profilesService implements vendors.ProfilesService for Mist.
type profilesService struct {
	client api.Client
	orgID  string
}

// List returns device profiles, optionally filtered by type.
func (s *profilesService) List(ctx context.Context, profileType string) ([]*vendors.DeviceProfile, error) {
	profiles, err := s.client.GetDeviceProfiles(ctx, s.orgID, profileType)
	if err != nil {
		return nil, fmt.Errorf("failed to get device profiles: %w", err)
	}

	result := make([]*vendors.DeviceProfile, 0, len(profiles))
	for i := range profiles {
		vp := convertDeviceProfileToVendor(&profiles[i])
		if vp != nil {
			result = append(result, vp)
		}
	}

	return result, nil
}

// Get returns a device profile by ID.
func (s *profilesService) Get(ctx context.Context, profileID string) (*vendors.DeviceProfile, error) {
	profile, err := s.client.GetDeviceProfile(ctx, s.orgID, profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get device profile %q: %w", profileID, err)
	}

	return convertDeviceProfileToVendor(profile), nil
}

// ByName finds a device profile by name and type.
func (s *profilesService) ByName(ctx context.Context, name, profileType string) (*vendors.DeviceProfile, error) {
	profile, err := s.client.GetDeviceProfileByName(ctx, s.orgID, name, profileType)
	if err != nil {
		return nil, fmt.Errorf("failed to get device profile by name %q: %w", name, err)
	}

	return convertDeviceProfileToVendor(profile), nil
}

// Assign assigns a profile to devices.
func (s *profilesService) Assign(ctx context.Context, profileID string, macs []string) error {
	_, err := s.client.AssignDeviceProfile(ctx, s.orgID, profileID, macs)
	if err != nil {
		return fmt.Errorf("failed to assign device profile: %w", err)
	}
	return nil
}

// Unassign removes a profile from devices.
func (s *profilesService) Unassign(ctx context.Context, profileID string, macs []string) error {
	if err := s.client.UnassignDeviceProfiles(ctx, s.orgID, profileID, macs); err != nil {
		return fmt.Errorf("failed to unassign device profile: %w", err)
	}
	return nil
}

// Ensure profilesService implements vendors.ProfilesService at compile time.
var _ vendors.ProfilesService = (*profilesService)(nil)

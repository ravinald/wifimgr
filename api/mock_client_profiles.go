package api

import (
	"context"
	"fmt"
)

// Device Profiles
// ============================================================================

// GetDeviceProfiles retrieves all device profiles of a specific type
func (m *MockClient) GetDeviceProfiles(ctx context.Context, orgID string, profileType string) ([]DeviceProfile, error) {
	m.logRequest("GET", fmt.Sprintf("/orgs/%s/deviceprofiles?type=%s", orgID, profileType), nil)

	// Mock API call with rate limiting
	if m.rateLimiter != nil {
		if err := m.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var filteredProfiles []DeviceProfile
	for _, profile := range m.deviceProfiles {
		if profileType == "" || (profile.Type != nil && *profile.Type == profileType) {
			filteredProfiles = append(filteredProfiles, profile)
		}
	}

	return filteredProfiles, nil
}

// GetDeviceProfile retrieves a specific device profile by ID
func (m *MockClient) GetDeviceProfile(ctx context.Context, orgID string, profileID string) (*DeviceProfile, error) {
	m.logRequest("GET", fmt.Sprintf("/orgs/%s/deviceprofiles/%s", orgID, profileID), nil)

	// Mock API call with rate limiting
	if m.rateLimiter != nil {
		if err := m.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	profile, found := m.profilesByID[profileID]
	if !found {
		return nil, fmt.Errorf("device profile with ID %s not found", profileID)
	}

	return profile, nil
}

// GetDeviceProfileByName retrieves a device profile by name and type
func (m *MockClient) GetDeviceProfileByName(ctx context.Context, orgID string, name string, profileType string) (*DeviceProfile, error) {
	m.logRequest("GET", fmt.Sprintf("/orgs/%s/deviceprofiles?name=%s&type=%s", orgID, name, profileType), nil)

	// Mock API call with rate limiting
	if m.rateLimiter != nil {
		if err := m.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	profile, found := m.profilesByName[name]
	if found && (profileType == "" || (profile.Type != nil && *profile.Type == profileType)) {
		return profile, nil
	}

	return nil, fmt.Errorf("device profile with name %s and type %s not found", name, profileType)
}

// AssignDeviceProfile assigns a device profile to a list of devices
func (m *MockClient) AssignDeviceProfile(ctx context.Context, orgID string, profileID string, macs []string) (*DeviceProfileAssignResult, error) {
	m.logRequest("POST", fmt.Sprintf("/orgs/%s/deviceprofiles/%s/assign", orgID, profileID), macs)

	// Mock API call with rate limiting
	if m.rateLimiter != nil {
		if err := m.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if profile exists
	if _, found := m.profilesByID[profileID]; !found {
		return nil, fmt.Errorf("device profile with ID %s not found", profileID)
	}

	// Mock successful assignment
	result := &DeviceProfileAssignResult{
		Success: macs,
		Errors:  make(map[string]string),
	}

	return result, nil
}

// UnassignDeviceProfiles removes device profile assignments from a list of devices
func (m *MockClient) UnassignDeviceProfiles(ctx context.Context, orgID string, profileID string, macs []string) error {
	m.logRequest("POST", fmt.Sprintf("/orgs/%s/deviceprofiles/%s/unassign", orgID, profileID), macs)

	// Mock API call with rate limiting
	if m.rateLimiter != nil {
		if err := m.rateLimiter.Wait(ctx); err != nil {
			return err
		}
	}

	// Mock successful unassignment
	return nil
}

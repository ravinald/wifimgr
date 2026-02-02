package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
)

// Device Profile-related methods for the mistClient

// GetDeviceProfiles retrieves all device profiles, optionally filtered by type
func (c *mistClient) GetDeviceProfiles(ctx context.Context, orgID string, profileType string) ([]DeviceProfile, error) {
	// Check cache first
	cacheKey := "profiles_all"
	if profileType != "" {
		cacheKey = fmt.Sprintf("profiles_%s", profileType)
	}
	if profiles, found := c.deviceProfileCache.Get(cacheKey); found {
		c.logDebug("Cache hit for device profiles (type: %s)", profileType)
		return profiles, nil
	}

	c.logDebug("Cache miss for device profiles (type: %s)", profileType)

	// Determine the results limit to use
	limit := 100 // Default value
	if c.config.ResultsLimit > 0 {
		limit = c.config.ResultsLimit
		c.logDebug("Using configured results limit: %d", limit)
	}

	var allProfiles []DeviceProfile
	page := 1
	hasMore := true

	for hasMore {
		c.logDebug("Fetching device profiles page %d with limit %d for type %s", page, limit, profileType)

		// Build query parameters
		query := url.Values{}
		if profileType != "" {
			query.Set("type", profileType)
		}
		query.Set("limit", fmt.Sprintf("%d", limit))
		if page > 1 {
			query.Set("page", fmt.Sprintf("%d", page))
		}

		// Build the path with query parameters
		path := fmt.Sprintf("/orgs/%s/deviceprofiles?%s", orgID, query.Encode())
		c.logDebug("Making API request to path: %s", path)

		var rawProfiles []map[string]interface{}
		if err := c.do(ctx, http.MethodGet, path, nil, &rawProfiles); err != nil {
			c.logDebug("API request failed for device profiles: %v", err)
			return nil, fmt.Errorf("failed to get device profiles: %w", err)
		}

		c.logDebug("API request successful, received %d raw profiles on page %d", len(rawProfiles), page)

		if len(rawProfiles) == 0 {
			c.logDebug("No profiles returned on page %d, ending pagination", page)
			hasMore = false
			continue
		}

		// Convert raw profiles to DeviceProfile structs while preserving all data
		pageProfiles := make([]DeviceProfile, 0, len(rawProfiles))
		for _, rawProfile := range rawProfiles {
			var profile DeviceProfile
			if err := profile.FromMap(rawProfile); err != nil {
				c.logDebug("Failed to convert raw profile to DeviceProfile: %v", err)
				continue
			}
			pageProfiles = append(pageProfiles, profile)
		}

		c.logDebug("Converted %d raw profiles to DeviceProfile structs", len(pageProfiles))

		// Add the current page of profiles to the result
		allProfiles = append(allProfiles, pageProfiles...)
		c.logDebug("Total profiles accumulated so far: %d", len(allProfiles))

		// Check if we've received fewer profiles than the limit, indicating the last page
		if len(pageProfiles) < limit {
			c.logDebug("Received %d profiles (less than limit %d), this is the last page", len(pageProfiles), limit)
			hasMore = false
		} else {
			c.logDebug("Received full page of %d profiles, continuing to next page", len(pageProfiles))
			page++
		}
	}

	c.logDebug("Device profile fetch completed for type %s: %d total profiles", profileType, len(allProfiles))

	// Update cache
	c.deviceProfileCache.Set(cacheKey, allProfiles)
	c.logDebug("Updated device profile cache with key '%s' containing %d profiles", cacheKey, len(allProfiles))

	return allProfiles, nil
}

// GetDeviceProfile retrieves a specific device profile by ID
func (c *mistClient) GetDeviceProfile(ctx context.Context, orgID string, profileID string) (*DeviceProfile, error) {
	var rawProfile map[string]interface{}
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/orgs/%s/deviceprofiles/%s", orgID, profileID), nil, &rawProfile)
	if err != nil {
		return nil, formatError("failed to get device profile", err)
	}

	// Convert raw profile to DeviceProfile struct while preserving all data
	var profile DeviceProfile
	if err := profile.FromMap(rawProfile); err != nil {
		return nil, formatError("failed to convert device profile data", err)
	}

	return &profile, nil
}

// GetDeviceProfileByName retrieves a device profile by name and type
func (c *mistClient) GetDeviceProfileByName(ctx context.Context, orgID string, name string, profileType string) (*DeviceProfile, error) {
	profiles, err := c.GetDeviceProfiles(ctx, orgID, profileType)
	if err != nil {
		return nil, formatError("failed to get device profiles", err)
	}

	for _, profile := range profiles {
		if profile.Name != nil && *profile.Name == name {
			return &profile, nil
		}
	}

	return nil, fmt.Errorf("device profile with name '%s' and type '%s' not found", name, profileType)
}

// GetInventoryConfig retrieves the inventory configuration from the specified path
func (c *mistClient) GetInventoryConfig(inventoryPath string) (*InventoryConfig, error) {
	c.logDebug("Reading inventory config from %s", inventoryPath)

	// Read the inventory configuration file
	data, err := os.ReadFile(inventoryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read inventory config: %w", err)
	}

	// Parse the JSON data
	var config InventoryConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse inventory config: %w", err)
	}

	return &config, nil
}

// AssignDeviceProfile assigns a device profile to a list of devices
func (c *mistClient) AssignDeviceProfile(ctx context.Context, orgID string, profileID string, macs []string) (*DeviceProfileAssignResult, error) {
	c.logDebug("Assigning device profile %s to %d devices", profileID, len(macs))

	// Prepare request body
	body := map[string]interface{}{
		"macs": macs,
	}

	// Make API request
	path := fmt.Sprintf("/orgs/%s/deviceprofiles/%s/assign", orgID, profileID)
	var result DeviceProfileAssignResult
	if err := c.do(ctx, http.MethodPost, path, body, &result); err != nil {
		c.logDebug("Failed to assign device profile: %v", err)
		return nil, fmt.Errorf("failed to assign device profile: %w", err)
	}

	c.logDebug("Successfully assigned device profile to %d devices", len(result.Success))
	return &result, nil
}

// UnassignDeviceProfiles removes device profile assignments from a list of devices
func (c *mistClient) UnassignDeviceProfiles(ctx context.Context, orgID string, profileID string, macs []string) error {
	c.logDebug("Unassigning device profile %s from %d devices", profileID, len(macs))

	// Prepare request body
	body := map[string]interface{}{
		"macs": macs,
	}

	// Make API request
	path := fmt.Sprintf("/orgs/%s/deviceprofiles/%s/unassign", orgID, profileID)

	// Log the full URL and request body for debugging
	fullURL := c.config.BaseURL + path
	bodyJSON, _ := json.MarshalIndent(body, "", "  ")
	c.logDebug("Unassign API URL: %s", fullURL)
	c.logDebug("Unassign request body: %s", string(bodyJSON))

	if err := c.do(ctx, http.MethodPost, path, body, nil); err != nil {
		c.logDebug("Failed to unassign device profile: %v", err)
		return fmt.Errorf("failed to unassign device profile: %w", err)
	}

	c.logDebug("Successfully unassigned device profile from %d devices", len(macs))
	return nil
}

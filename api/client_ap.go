package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/macaddr"
)

// AP-related methods for the mistClient

// GetAP retrieves a specific AP from the site by ID, name, MAC, or serial
func (c *mistClient) GetAP(ctx context.Context, siteID, apIdentifier string) (*AP, error) {
	// First attempt to get by ID
	path := fmt.Sprintf("/sites/%s/devices/%s", siteID, apIdentifier)
	var ap map[string]interface{}
	err := c.do(ctx, http.MethodGet, path, nil, &ap)
	if err == nil {
		// Check if the AP is of correct type
		if apType, ok := ap["type"].(string); ok && apType == "ap" {
			// Convert the raw map to a UnifiedDevice
			device, err := ConvertFromRawMap(ap)
			if err != nil {
				return nil, fmt.Errorf("failed to convert AP data: %v", err)
			}
			// Convert to AP
			aps := ConvertToAPSlice([]UnifiedDevice{*device})
			if len(aps) > 0 {
				return &aps[0], nil
			}
		}
	}

	// If not found by ID, try a search by specific attributes
	query := url.Values{}

	// Try to determine if this is a MAC address
	if macaddr.IsValid(apIdentifier) {
		query.Set("mac", apIdentifier)
	} else {
		// Might be a name or serial
		query.Set("name", apIdentifier)
	}

	searchPath := fmt.Sprintf("/sites/%s/devices?%s", siteID, query.Encode())
	var aps []map[string]interface{}
	err = c.do(ctx, http.MethodGet, searchPath, nil, &aps)
	if err != nil {
		return nil, fmt.Errorf("failed to search for AP: %w", err)
	}

	// Filter for exact matches
	for _, apData := range aps {
		if apType, ok := apData["type"].(string); ok && apType == "ap" {
			// Check if name, MAC or serial matches
			matched := false

			if name, ok := apData["name"].(string); ok && name == apIdentifier {
				matched = true
			} else if mac, ok := apData["mac"].(string); ok && macaddr.Equal(mac, apIdentifier) {
				matched = true
			} else if serial, ok := apData["serial"].(string); ok && serial == apIdentifier {
				matched = true
			}

			if matched {
				// Convert the raw map to a UnifiedDevice
				device, err := ConvertFromRawMap(apData)
				if err != nil {
					return nil, fmt.Errorf("failed to convert AP data: %v", err)
				}
				// Convert to AP
				apList := ConvertToAPSlice([]UnifiedDevice{*device})
				if len(apList) > 0 {
					return &apList[0], nil
				}
			}
		}
	}

	// Legacy cache operations disabled
	// Legacy cache operations disabled - AP lookup modernized via cache accessor

	return nil, fmt.Errorf("AP not found with identifier: %s", apIdentifier)
}

// GetAPByMAC retrieves an AP from the local cache by MAC address
func (c *mistClient) GetAPByMAC(mac string) (*AP, error) {
	normalizedMAC, err := macaddr.Normalize(mac)
	if err != nil {
		return nil, fmt.Errorf("invalid MAC address: %w", err)
	}

	// Create a background context for this operation
	ctx := context.Background()

	// Check if we have a device in the local cache
	device, err := c.GetDeviceByMAC(ctx, normalizedMAC)
	if err == nil && device != nil {
		// Only return if it's an AP
		if device.Type != nil && *device.Type == "ap" {
			// Convert UnifiedDevice to AP
			aps := ConvertToAPSlice([]UnifiedDevice{*device})
			if len(aps) > 0 {
				return &aps[0], nil
			}
		}
	}

	// Legacy cache operations disabled - device lookup modernized via cache accessor

	// If not found or no local cache, return error
	return nil, fmt.Errorf("AP with MAC '%s' not found in local cache", mac)
}

// UpdateAPConfiguration updates specific configuration fields for an AP
func (c *mistClient) UpdateAPConfiguration(ctx context.Context, siteID string, apID string, apConfig map[string]interface{}) error {
	// If in dry run mode, log and return simulated success
	if c.dryRun {
		logging.Infof("[DRY RUN] Would update AP configuration for %s: %+v", apID, apConfig)
		return nil
	}

	// Convert to simple object for the API
	path := fmt.Sprintf("/sites/%s/devices/%s", siteID, apID)

	// Empty response object for the API call
	var response interface{}

	// Update the AP configuration
	err := c.do(ctx, http.MethodPut, path, apConfig, &response)
	if err != nil {
		return fmt.Errorf("failed to update AP configuration: %w", err)
	}

	return nil
}

// UpdateAPByName finds an AP by name and updates its configuration
func (c *mistClient) UpdateAPByName(ctx context.Context, siteID, apName string, ap AP) (*AP, error) {
	// If in dry run mode, log and return simulated success
	if c.dryRun {
		logging.Infof("[DRY RUN] Would update AP %s: %+v", apName, ap)
		return &ap, nil
	}

	// Find the AP first
	existingAP, err := c.GetAP(ctx, siteID, apName)
	if err != nil {
		return nil, fmt.Errorf("AP not found: %w", err)
	}

	// Get the AP ID
	if existingAP.Id == nil {
		return nil, fmt.Errorf("AP found but has no ID")
	}

	apID := string(*existingAP.Id)

	// Convert AP struct to map for API request
	apData := make(map[string]interface{})

	// Add fields that can be updated
	if ap.Name != nil {
		apData["name"] = *ap.Name
	}

	// Add location if present
	if ap.Location != nil {
		apData["location"] = *ap.Location
	}

	// Add orientation if present
	if ap.Orientation != nil {
		apData["orientation"] = *ap.Orientation
	}

	// Add map_id if present
	if ap.MapID != nil {
		apData["map_id"] = *ap.MapID
	}

	// Add radio config if present
	if ap.RadioConfig != nil {
		apData["radio_config"] = ap.RadioConfig
	}

	// Add LED setting if present
	if ap.Led != nil {
		apData["led"] = *ap.Led
	}

	// Update the AP
	err = c.UpdateAPConfiguration(ctx, siteID, apID, apData)
	if err != nil {
		return nil, err
	}

	// Get the updated AP
	updatedAP, err := c.GetAP(ctx, siteID, apID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated AP: %w", err)
	}

	return updatedAP, nil
}

// UnassignAPByName finds an AP by name and unassigns it from the site
func (c *mistClient) UnassignAPByName(ctx context.Context, siteID, apName string) error {
	// If in dry run mode, log and return simulated success
	if c.dryRun {
		logging.Infof("[DRY RUN] Would unassign AP %s from site %s", apName, siteID)
		return nil
	}

	// Find the AP first
	ap, err := c.GetAP(ctx, siteID, apName)
	if err != nil {
		return fmt.Errorf("AP not found: %w", err)
	}

	// Get the AP ID
	if ap.Id == nil {
		return fmt.Errorf("AP found but has no ID")
	}

	apID := string(*ap.Id)

	// Unassign the AP
	path := fmt.Sprintf("/sites/%s/devices/%s", siteID, apID)
	err = c.do(ctx, http.MethodDelete, path, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to unassign AP: %w", err)
	}

	// Legacy cache operations disabled - cache updates modernized

	return nil
}

// AssignOrUpdateAP assigns or updates an AP in the specified site
func (c *mistClient) AssignOrUpdateAP(ctx context.Context, siteID string, apData AP) (*AP, error) {
	// If in dry run mode, log and return simulated success
	if c.dryRun {
		logging.Infof("[DRY RUN] Would assign or update AP to site %s: %+v", siteID, apData)
		return &apData, nil
	}

	// Determine if this is an assignment or update by checking if the AP is already in the site
	var existingAP *AP
	var err error

	if apData.Mac != nil {
		// Try to find by MAC
		existingAP, err = c.GetAP(ctx, siteID, *apData.Mac)
	} else if apData.Name != nil {
		// Try to find by name
		existingAP, err = c.GetAP(ctx, siteID, *apData.Name)
	}

	if err == nil && existingAP != nil {
		// AP exists, update it
		if existingAP.Id == nil {
			return nil, fmt.Errorf("existing AP found but has no ID")
		}

		apID := string(*existingAP.Id)

		// Convert AP struct to map for API request
		apUpdateData := make(map[string]interface{})

		// Add fields that can be updated
		if apData.Name != nil {
			apUpdateData["name"] = *apData.Name
		}

		// Add location if present
		if apData.Location != nil {
			apUpdateData["location"] = *apData.Location
		}

		// Add orientation if present
		if apData.Orientation != nil {
			apUpdateData["orientation"] = *apData.Orientation
		}

		// Add map_id if present
		if apData.MapID != nil {
			apUpdateData["map_id"] = *apData.MapID
		}

		// Add radio config if present
		if apData.RadioConfig != nil {
			apUpdateData["radio_config"] = apData.RadioConfig
		}

		// Add LED setting if present
		if apData.Led != nil {
			apUpdateData["led"] = *apData.Led
		}

		// Update the AP
		err = c.UpdateAPConfiguration(ctx, siteID, apID, apUpdateData)
		if err != nil {
			return nil, err
		}

		// Get the updated AP
		return c.GetAP(ctx, siteID, apID)
	} else {
		// AP doesn't exist, assign it
		if apData.Mac == nil {
			return nil, fmt.Errorf("MAC address is required for AP assignment")
		}

		// Prepare assignment request
		assignData := map[string]interface{}{
			"mac": *apData.Mac,
		}

		// Assign the AP first
		path := fmt.Sprintf("/sites/%s/devices", siteID)
		var response map[string]interface{}

		if err := c.do(ctx, http.MethodPost, path, assignData, &response); err != nil {
			return nil, fmt.Errorf("failed to assign AP to site: %w", err)
		}

		// If the AP has additional configuration, update it
		hasConfig := apData.Name != nil || apData.Location != nil || apData.Orientation != nil ||
			apData.MapID != nil || apData.RadioConfig != nil || apData.Led != nil

		if hasConfig {
			// Get the newly assigned AP to get its ID
			assignedAP, err := c.GetAP(ctx, siteID, *apData.Mac)
			if err != nil {
				return nil, fmt.Errorf("failed to get assigned AP: %w", err)
			}

			if assignedAP.Id == nil {
				return nil, fmt.Errorf("assigned AP has no ID")
			}

			apID := string(*assignedAP.Id)

			// Prepare update data
			updateData := make(map[string]interface{})

			// Add fields that can be updated
			if apData.Name != nil {
				updateData["name"] = *apData.Name
			}

			// Add location if present
			if apData.Location != nil {
				updateData["location"] = *apData.Location
			}

			// Add orientation if present
			if apData.Orientation != nil {
				updateData["orientation"] = *apData.Orientation
			}

			// Add map_id if present
			if apData.MapID != nil {
				updateData["map_id"] = *apData.MapID
			}

			// Add radio config if present
			if apData.RadioConfig != nil {
				updateData["radio_config"] = apData.RadioConfig
			}

			// Add LED setting if present
			if apData.Led != nil {
				updateData["led"] = *apData.Led
			}

			// Update the AP
			err = c.UpdateAPConfiguration(ctx, siteID, apID, updateData)
			if err != nil {
				return nil, err
			}

			// Get the updated AP
			return c.GetAP(ctx, siteID, apID)
		}

		// Just get the assigned AP
		return c.GetAP(ctx, siteID, *apData.Mac)
	}
}

// GetAPBySerialOrMAC retrieves an AP by serial number or MAC address
func (c *mistClient) GetAPBySerialOrMAC(ctx context.Context, siteID, serial, mac string) (*AP, error) {
	// Try by serial first if provided
	if serial != "" {
		ap, err := c.GetAP(ctx, siteID, serial)
		if err == nil {
			return ap, nil
		}
	}

	// Try by MAC if provided
	if mac != "" {
		ap, err := c.GetAP(ctx, siteID, mac)
		if err == nil {
			return ap, nil
		}
	}

	return nil, fmt.Errorf("AP not found with serial '%s' or MAC '%s'", serial, mac)
}

// ConcurrentlyAssignAPs assigns multiple APs to a site concurrently
func (c *mistClient) ConcurrentlyAssignAPs(ctx context.Context, siteID string, aps []AP, concurrency int) []error {
	if len(aps) == 0 {
		return nil
	}

	// If concurrency is not specified or invalid, use a reasonable default
	if concurrency <= 0 {
		concurrency = 5
	}

	// Create a channel for tasks with buffer size equal to number of APs
	tasks := make(chan AP, len(aps))

	// Create a channel for errors with buffer size equal to number of APs
	errors := make(chan error, len(aps))

	// Create a limited number of workers
	for w := 0; w < concurrency; w++ {
		go func() {
			for ap := range tasks {
				// Assign the AP and report any error
				_, err := c.AssignOrUpdateAP(ctx, siteID, ap)
				errors <- err
			}
		}()
	}

	// Queue up the tasks
	for _, ap := range aps {
		tasks <- ap
	}
	close(tasks)

	// Collect the errors
	var errList []error
	for i := 0; i < len(aps); i++ {
		err := <-errors
		if err != nil {
			errList = append(errList, err)
		}
	}

	return errList
}

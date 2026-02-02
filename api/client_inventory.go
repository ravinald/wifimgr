package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// Inventory-related methods using the new bidirectional data handling

// GetInventory retrieves all inventory items of a specific type for an organization using raw JSON unmarshaling
func (c *mistClient) GetInventory(ctx context.Context, orgID string, deviceType string) ([]*MistInventoryItem, error) {
	// Use simple memory cache to avoid redundant API calls in the same session
	// Log more clearly when fetching ALL inventory
	displayType := deviceType
	if deviceType == "" || deviceType == "all" {
		displayType = "ALL"
	}
	cacheKey := fmt.Sprintf("inventory_%s_%s", orgID, deviceType)

	// Check if we have a recent cache entry (within the last 5 minutes)
	if c.inventoryCache != nil {
		if items, found := c.inventoryCache.Get(cacheKey); found {
			// Convert to new format
			var result []*MistInventoryItem
			for _, item := range items {
				newItem := &MistInventoryItem{
					ID:     (*string)(item.Id),
					MAC:    item.Mac,
					Serial: item.Serial,
					Name:   item.Name,
					Model:  item.Model,
					Type:   item.Type,
					SiteID: (*string)(item.SiteId),
					OrgID:  (*string)(item.OrgId),
				}
				result = append(result, newItem)
			}
			c.logDebug("In-memory cache hit for inventory type %s: returning %d items", displayType, len(result))
			return result, nil
		}
	}

	// Try to populate from file cache first
	cacheAccessor := c.GetCacheAccessor()
	if cacheAccessor != nil {
		c.logDebug("Checking file cache for inventory type %s", displayType)

		var fileInventory []*MistInventoryItem

		// Get inventory from file cache based on device type
		switch deviceType {
		case "ap":
			configs, err := cacheAccessor.GetAllAPConfigs()
			if err == nil && len(configs) > 0 {
				for _, config := range configs {
					if config.OrgID != nil && *config.OrgID == orgID {
						item := &MistInventoryItem{
							ID:     config.ID,
							MAC:    config.MAC,
							Serial: config.Serial,
							Name:   config.Name,
							Model:  config.Model,
							Type:   StringPtr("ap"),
							SiteID: config.SiteID,
							OrgID:  config.OrgID,
						}
						fileInventory = append(fileInventory, item)
					}
				}
				c.logDebug("Found %d APs in file cache for org %s", len(fileInventory), orgID)
			}
		case "switch":
			configs, err := cacheAccessor.GetAllSwitchConfigs()
			if err == nil && len(configs) > 0 {
				for _, config := range configs {
					if config.OrgID != nil && *config.OrgID == orgID {
						item := &MistInventoryItem{
							ID:     config.ID,
							MAC:    config.MAC,
							Serial: config.Serial,
							Name:   config.Name,
							Model:  config.Model,
							Type:   StringPtr("switch"),
							SiteID: config.SiteID,
							OrgID:  config.OrgID,
						}
						fileInventory = append(fileInventory, item)
					}
				}
				c.logDebug("Found %d switches in file cache for org %s", len(fileInventory), orgID)
			}
		case "gateway":
			configs, err := cacheAccessor.GetAllGatewayConfigs()
			if err == nil && len(configs) > 0 {
				for _, config := range configs {
					if config.OrgID != nil && *config.OrgID == orgID {
						item := &MistInventoryItem{
							ID:     config.ID,
							MAC:    config.MAC,
							Serial: config.Serial,
							Name:   config.Name,
							Model:  config.Model,
							Type:   StringPtr("gateway"),
							SiteID: config.SiteID,
							OrgID:  config.OrgID,
						}
						fileInventory = append(fileInventory, item)
					}
				}
				c.logDebug("Found %d gateways in file cache for org %s", len(fileInventory), orgID)
			}
		case "", "all":
			// Get all device types
			apConfigs, _ := cacheAccessor.GetAllAPConfigs()
			for _, config := range apConfigs {
				if config.OrgID != nil && *config.OrgID == orgID {
					item := &MistInventoryItem{
						ID:     config.ID,
						MAC:    config.MAC,
						Serial: config.Serial,
						Name:   config.Name,
						Model:  config.Model,
						Type:   StringPtr("ap"),
						SiteID: config.SiteID,
						OrgID:  config.OrgID,
					}
					fileInventory = append(fileInventory, item)
				}
			}

			switchConfigs, _ := cacheAccessor.GetAllSwitchConfigs()
			for _, config := range switchConfigs {
				if config.OrgID != nil && *config.OrgID == orgID {
					item := &MistInventoryItem{
						ID:     config.ID,
						MAC:    config.MAC,
						Serial: config.Serial,
						Name:   config.Name,
						Model:  config.Model,
						Type:   StringPtr("switch"),
						SiteID: config.SiteID,
						OrgID:  config.OrgID,
					}
					fileInventory = append(fileInventory, item)
				}
			}

			gatewayConfigs, _ := cacheAccessor.GetAllGatewayConfigs()
			for _, config := range gatewayConfigs {
				if config.OrgID != nil && *config.OrgID == orgID {
					item := &MistInventoryItem{
						ID:     config.ID,
						MAC:    config.MAC,
						Serial: config.Serial,
						Name:   config.Name,
						Model:  config.Model,
						Type:   StringPtr("gateway"),
						SiteID: config.SiteID,
						OrgID:  config.OrgID,
					}
					fileInventory = append(fileInventory, item)
				}
			}
			c.logDebug("Found %d total devices in file cache for org %s", len(fileInventory), orgID)
		}

		if len(fileInventory) > 0 {
			// Populate in-memory cache with file cache data
			if c.inventoryCache != nil {
				// Convert to old format for cache storage
				var cacheItems []InventoryItem
				for _, item := range fileInventory {
					cacheItem := InventoryItem{
						Mac:    item.MAC,
						Serial: item.Serial,
						Name:   item.Name,
						Model:  item.Model,
						Type:   item.Type,
					}
					if item.ID != nil {
						id := UUID(*item.ID)
						cacheItem.Id = &id
					}
					if item.SiteID != nil {
						siteId := UUID(*item.SiteID)
						cacheItem.SiteId = &siteId
					}
					if item.OrgID != nil {
						orgId := UUID(*item.OrgID)
						cacheItem.OrgId = &orgId
					}
					cacheItems = append(cacheItems, cacheItem)
				}
				c.inventoryCache.Set(cacheKey, cacheItems)
				c.logDebug("Populated in-memory cache with %d items from file cache", len(fileInventory))
			}

			c.logDebug("File cache hit for inventory type %s: returning %d items", displayType, len(fileInventory))
			return fileInventory, nil
		}
	}

	c.logDebug("Cache miss for inventory type %s in both memory and file cache", displayType)

	// Determine the results limit to use
	limit := 100 // Default value
	if c.config.ResultsLimit > 0 {
		limit = c.config.ResultsLimit
		c.logDebug("Using configured results limit: %d", limit)
	}

	var allItems []*MistInventoryItem
	page := 1
	hasMore := true

	// Build the base path
	basePath := fmt.Sprintf("/orgs/%s/inventory", orgID)

	for hasMore {
		c.logDebug("Fetching inventory page %d with limit %d", page, limit)

		// Build query parameters
		query := url.Values{}
		query.Set("limit", fmt.Sprintf("%d", limit))
		if page > 1 {
			query.Set("page", fmt.Sprintf("%d", page))
		}
		if deviceType != "" && deviceType != "all" {
			query.Set("type", deviceType)
		}

		// Build the path with query parameters
		path := fmt.Sprintf("%s?%s", basePath, query.Encode())

		// Use raw JSON unmarshaling to preserve all data
		var rawResponse json.RawMessage
		if err := c.do(ctx, http.MethodGet, path, nil, &rawResponse); err != nil {
			return nil, fmt.Errorf("failed to get inventory: %w", err)
		}

		// Parse the raw JSON to map slice
		var rawItems []map[string]interface{}
		if err := json.Unmarshal(rawResponse, &rawItems); err != nil {
			return nil, fmt.Errorf("failed to unmarshal inventory response: %w", err)
		}

		if len(rawItems) == 0 {
			hasMore = false
			continue
		}

		// Convert each raw item to MistInventoryItem
		for _, rawItem := range rawItems {

			item, err := NewInventoryItemFromMap(rawItem)
			if err != nil {
				c.logDebug("Failed to create inventory item from map: %v", err)
				continue
			}
			allItems = append(allItems, item)
		}

		// Check if we've received fewer items than the limit, indicating the last page
		if len(rawItems) < limit {
			hasMore = false
		} else {
			page++
		}
	}

	// Update memory cache with fetched inventory
	if c.inventoryCache != nil && len(allItems) > 0 {
		// Convert to old format for cache storage
		var cacheItems []InventoryItem
		for _, item := range allItems {
			cacheItem := InventoryItem{
				Mac:    item.MAC,
				Serial: item.Serial,
				Name:   item.Name,
				Model:  item.Model,
				Type:   item.Type,
			}
			if item.ID != nil {
				id := UUID(*item.ID)
				cacheItem.Id = &id
			}
			if item.SiteID != nil {
				siteId := UUID(*item.SiteID)
				cacheItem.SiteId = &siteId
			}
			if item.OrgID != nil {
				orgId := UUID(*item.OrgID)
				cacheItem.OrgId = &orgId
			}
			cacheItems = append(cacheItems, cacheItem)
		}

		cacheKey := fmt.Sprintf("inventory_%s_%s", orgID, deviceType)
		c.inventoryCache.Set(cacheKey, cacheItems)
		// Log more clearly when caching ALL inventory
		displayType := deviceType
		if deviceType == "" || deviceType == "all" {
			displayType = "ALL"
		}
		c.logDebug("Updated memory cache with %d inventory items for type %s", len(allItems), displayType)
	}

	return allItems, nil
}

// GetInventoryItem retrieves a specific inventory item by ID using raw JSON unmarshaling
func (c *mistClient) GetInventoryItem(ctx context.Context, orgID string, itemID string) (*MistInventoryItem, error) {
	// Use raw JSON unmarshaling to preserve all data
	var rawResponse json.RawMessage
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/orgs/%s/inventory/%s", orgID, itemID), nil, &rawResponse)
	if err != nil {
		return nil, formatError("failed to get inventory item", err)
	}

	// Parse the raw JSON to map
	var rawItem map[string]interface{}
	if err := json.Unmarshal(rawResponse, &rawItem); err != nil {
		return nil, fmt.Errorf("failed to unmarshal inventory item response: %w", err)
	}

	// Convert to MistInventoryItem
	item, err := NewInventoryItemFromMap(rawItem)
	if err != nil {
		return nil, fmt.Errorf("failed to create inventory item from API response: %w", err)
	}

	return item, nil
}

// GetInventoryItemByMAC retrieves an inventory item by MAC address using the new implementation
func (c *mistClient) GetInventoryItemByMAC(ctx context.Context, orgID string, macAddress string) (*MistInventoryItem, error) {
	// Get all inventory items and search for the MAC
	items, err := c.GetInventory(ctx, orgID, "all")
	if err != nil {
		return nil, formatError("failed to get inventory", err)
	}

	for _, item := range items {
		if item.GetMAC() == macAddress {
			return item, nil
		}
	}

	return nil, fmt.Errorf("inventory item with MAC '%s' not found", macAddress)
}

// UpdateInventoryItem updates an existing inventory item by ID using the new implementation
func (c *mistClient) UpdateInventoryItem(ctx context.Context, orgID string, itemID string, item *MistInventoryItem) (*MistInventoryItem, error) {
	// If in dry run mode, log and return simulated success
	if c.dryRun {
		c.logDebug("[DRY RUN] Would update inventory item %s: %+v", itemID, item)
		// Return the input item as if it were updated
		item.ID = &itemID
		return item, nil
	}

	// Convert item to map for API request
	itemData := item.ToMap()

	// Use raw JSON unmarshaling to preserve all data in response
	var rawResponse json.RawMessage
	err := c.do(ctx, http.MethodPut, fmt.Sprintf("/orgs/%s/inventory/%s", orgID, itemID), itemData, &rawResponse)
	if err != nil {
		return nil, formatError("failed to update inventory item", err)
	}

	// Parse the raw JSON to map
	var rawItem map[string]interface{}
	if err := json.Unmarshal(rawResponse, &rawItem); err != nil {
		return nil, fmt.Errorf("failed to unmarshal update inventory item response: %w", err)
	}

	// Convert to MistInventoryItem
	updatedItem, err := NewInventoryItemFromMap(rawItem)
	if err != nil {
		return nil, fmt.Errorf("failed to create inventory item from API response: %w", err)
	}

	// Invalidate inventory cache
	c.inventoryCache.Delete("inventory")
	// Note: Local cache for inventory_new will be updated in the normal cache refresh cycle

	return updatedItem, nil
}

// ClaimInventoryItem claims an inventory item using the new implementation
func (c *mistClient) ClaimInventoryItem(ctx context.Context, orgID string, claimCodes []string) ([]*MistInventoryItem, error) {
	// If in dry run mode, log and return simulated success
	if c.dryRun {
		c.logDebug("[DRY RUN] Would claim inventory items with codes: %v", claimCodes)
		var simulatedItems []*MistInventoryItem
		for _, code := range claimCodes {
			simulatedItem := &MistInventoryItem{
				Magic:            &code,
				AdditionalConfig: make(map[string]interface{}),
				Raw:              make(map[string]interface{}),
			}
			simulatedItems = append(simulatedItems, simulatedItem)
		}
		return simulatedItems, nil
	}

	// Prepare claim request
	claimData := map[string]interface{}{
		"op":   "assign",
		"macs": claimCodes, // API expects "macs" field even for claim codes
	}

	// Use raw JSON unmarshaling to preserve all data in response
	var rawResponse json.RawMessage
	err := c.do(ctx, http.MethodPost, fmt.Sprintf("/orgs/%s/inventory", orgID), claimData, &rawResponse)
	if err != nil {
		return nil, formatError("failed to claim inventory items", err)
	}

	// Parse the raw JSON to map slice
	var rawItems []map[string]interface{}
	if err := json.Unmarshal(rawResponse, &rawItems); err != nil {
		return nil, fmt.Errorf("failed to unmarshal claim inventory response: %w", err)
	}

	// Convert to MistInventoryItem slice
	var claimedItems []*MistInventoryItem
	for _, rawItem := range rawItems {
		item, err := NewInventoryItemFromMap(rawItem)
		if err != nil {
			c.logDebug("Failed to create inventory item from claim response: %v", err)
			continue
		}
		claimedItems = append(claimedItems, item)
	}

	// Invalidate inventory cache
	c.inventoryCache.Delete("inventory")
	// Note: Local cache for inventory_new will be updated in the normal cache refresh cycle

	return claimedItems, nil
}

// ReleaseInventoryItem releases inventory items from the organization using the new implementation
func (c *mistClient) ReleaseInventoryItem(ctx context.Context, orgID string, itemIDs []string) error {
	// If in dry run mode, log and return simulated success
	if c.dryRun {
		c.logDebug("[DRY RUN] Would release inventory items: %v", itemIDs)
		return nil
	}

	// Prepare release request
	releaseData := map[string]interface{}{
		"op":   "unassign",
		"macs": itemIDs, // API expects "macs" field even for item IDs
	}

	// Real implementation
	err := c.do(ctx, http.MethodPost, fmt.Sprintf("/orgs/%s/inventory", orgID), releaseData, nil)
	if err != nil {
		return formatError("failed to release inventory items", err)
	}

	// Invalidate inventory cache
	c.inventoryCache.Delete("inventory")
	// Note: Local cache for inventory_new will be updated in the normal cache refresh cycle

	return nil
}

// AssignInventoryItemsToSite assigns inventory items to a site using the new implementation
func (c *mistClient) AssignInventoryItemsToSite(ctx context.Context, orgID string, siteID string, itemMACs []string) error {
	// If in dry run mode, log and return simulated success
	if c.dryRun {
		c.logDebug("[DRY RUN] Would assign inventory items %v to site %s", itemMACs, siteID)
		return nil
	}

	// Prepare assignment request
	assignData := map[string]interface{}{
		"op":      "assign",
		"site_id": siteID,
		"macs":    itemMACs,
	}

	// Real implementation
	err := c.do(ctx, http.MethodPost, fmt.Sprintf("/orgs/%s/inventory", orgID), assignData, nil)
	if err != nil {
		return formatError("failed to assign inventory items to site", err)
	}

	// Invalidate inventory cache
	c.inventoryCache.Delete("inventory")
	// Note: Local cache for inventory_new will be updated in the normal cache refresh cycle

	return nil
}

// UnassignInventoryItemsFromSite unassigns inventory items from their current site using the new implementation
func (c *mistClient) UnassignInventoryItemsFromSite(ctx context.Context, orgID string, itemMACs []string) error {
	// If in dry run mode, log and return simulated success
	if c.dryRun {
		c.logDebug("[DRY RUN] Would unassign inventory items %v from their sites", itemMACs)
		return nil
	}

	// Prepare unassignment request
	unassignData := map[string]interface{}{
		"op":   "unassign",
		"macs": itemMACs,
	}

	// Real implementation
	err := c.do(ctx, http.MethodPost, fmt.Sprintf("/orgs/%s/inventory", orgID), unassignData, nil)
	if err != nil {
		return formatError("failed to unassign inventory items from site", err)
	}

	// Invalidate inventory cache
	c.inventoryCache.Delete("inventory")
	// Note: Local cache for inventory_new will be updated in the normal cache refresh cycle

	return nil
}

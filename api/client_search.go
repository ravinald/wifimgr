package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Search-related methods using the new bidirectional data handling

// SearchWiredClients searches for wired clients in an organization using raw JSON unmarshaling
func (c *mistClient) SearchWiredClients(ctx context.Context, orgID string, text string) (*MistWiredClientResponse, error) {
	endpoint := fmt.Sprintf("/orgs/%s/wired_clients/search?text=%s", orgID, text)

	// Use raw JSON unmarshaling to preserve all data
	var rawResponse json.RawMessage
	err := c.do(ctx, http.MethodGet, endpoint, nil, &rawResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to search wired clients: %w", err)
	}

	// Parse the raw JSON to map
	var rawData map[string]interface{}
	if err := json.Unmarshal(rawResponse, &rawData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal wired clients search response: %w", err)
	}

	// Create response with complete data preservation
	response := &MistWiredClientResponse{
		Raw: make(map[string]interface{}),
	}

	// Store raw data for complete preservation
	for k, v := range rawData {
		response.Raw[k] = v
	}

	// Extract typed fields
	if limit, ok := rawData["limit"].(float64); ok {
		limitInt := int(limit)
		response.Limit = &limitInt
	} else if limit, ok := rawData["limit"].(int); ok {
		response.Limit = &limit
	}
	if start, ok := rawData["start"].(float64); ok {
		startInt := int64(start)
		response.Start = &startInt
	} else if start, ok := rawData["start"].(int64); ok {
		response.Start = &start
	}
	if end, ok := rawData["end"].(float64); ok {
		endInt := int64(end)
		response.End = &endInt
	} else if end, ok := rawData["end"].(int64); ok {
		response.End = &end
	}
	if total, ok := rawData["total"].(float64); ok {
		totalInt := int(total)
		response.Total = &totalInt
	} else if total, ok := rawData["total"].(int); ok {
		response.Total = &total
	}

	// Convert results array
	if resultsData, ok := rawData["results"].([]interface{}); ok {
		for _, resultItem := range resultsData {
			if resultMap, ok := resultItem.(map[string]interface{}); ok {
				client, err := NewWiredClientFromMap(resultMap)
				if err != nil {
					c.logDebug("Failed to create wired client from search result: %v", err)
					continue
				}
				response.Results = append(response.Results, client)
			}
		}
	}

	return response, nil
}

// SearchWirelessClients searches for wireless clients in an organization using raw JSON unmarshaling
func (c *mistClient) SearchWirelessClients(ctx context.Context, orgID string, text string) (*MistWirelessClientResponse, error) {
	endpoint := fmt.Sprintf("/orgs/%s/clients/search?text=%s", orgID, text)

	// Use raw JSON unmarshaling to preserve all data
	var rawResponse json.RawMessage
	err := c.do(ctx, http.MethodGet, endpoint, nil, &rawResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to search wireless clients: %w", err)
	}

	// Parse the raw JSON to map
	var rawData map[string]interface{}
	if err := json.Unmarshal(rawResponse, &rawData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal wireless clients search response: %w", err)
	}

	// Create response with complete data preservation
	response := &MistWirelessClientResponse{
		Raw: make(map[string]interface{}),
	}

	// Store raw data for complete preservation
	for k, v := range rawData {
		response.Raw[k] = v
	}

	// Extract typed fields
	if limit, ok := rawData["limit"].(float64); ok {
		limitInt := int(limit)
		response.Limit = &limitInt
	} else if limit, ok := rawData["limit"].(int); ok {
		response.Limit = &limit
	}
	if start, ok := rawData["start"].(float64); ok {
		startInt := int64(start)
		response.Start = &startInt
	} else if start, ok := rawData["start"].(int64); ok {
		response.Start = &start
	}
	if end, ok := rawData["end"].(float64); ok {
		endInt := int64(end)
		response.End = &endInt
	} else if end, ok := rawData["end"].(int64); ok {
		response.End = &end
	}
	if total, ok := rawData["total"].(float64); ok {
		totalInt := int(total)
		response.Total = &totalInt
	} else if total, ok := rawData["total"].(int); ok {
		response.Total = &total
	}

	// Convert results array
	if resultsData, ok := rawData["results"].([]interface{}); ok {
		for _, resultItem := range resultsData {
			if resultMap, ok := resultItem.(map[string]interface{}); ok {
				client, err := NewWirelessClientFromMap(resultMap)
				if err != nil {
					c.logDebug("Failed to create wireless client from search result: %v", err)
					continue
				}
				response.Results = append(response.Results, client)
			}
		}
	}

	return response, nil
}

// SearchWiredClientsByMAC searches for a specific wired client by MAC address using the new implementation
func (c *mistClient) SearchWiredClientsByMAC(ctx context.Context, orgID string, macAddress string) (*MistWiredClient, error) {
	response, err := c.SearchWiredClients(ctx, orgID, macAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to search wired clients by MAC: %w", err)
	}

	// Look for exact MAC match in results
	for _, client := range response.Results {
		if client.GetMAC() == macAddress {
			return client, nil
		}
	}

	return nil, fmt.Errorf("wired client with MAC '%s' not found", macAddress)
}

// SearchWirelessClientsByMAC searches for a specific wireless client by MAC address using the new implementation
func (c *mistClient) SearchWirelessClientsByMAC(ctx context.Context, orgID string, macAddress string) (*MistWirelessClient, error) {
	response, err := c.SearchWirelessClients(ctx, orgID, macAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to search wireless clients by MAC: %w", err)
	}

	// Look for exact MAC match in results
	for _, client := range response.Results {
		if client.GetMAC() == macAddress {
			return client, nil
		}
	}

	return nil, fmt.Errorf("wireless client with MAC '%s' not found", macAddress)
}

// SearchWiredClientsBySite searches for wired clients in a specific site using the new implementation
func (c *mistClient) SearchWiredClientsBySite(ctx context.Context, orgID string, siteID string, text string) (*MistWiredClientResponse, error) {
	// Get all wired clients for the text search
	response, err := c.SearchWiredClients(ctx, orgID, text)
	if err != nil {
		return nil, fmt.Errorf("failed to search wired clients: %w", err)
	}

	// Filter results by site ID
	var filteredResults []*MistWiredClient
	for _, client := range response.Results {
		if client.GetSiteID() == siteID {
			filteredResults = append(filteredResults, client)
		}
	}

	// Update response with filtered results
	response.Results = filteredResults
	if response.Total != nil {
		totalFiltered := len(filteredResults)
		response.Total = &totalFiltered
	}

	return response, nil
}

// SearchWirelessClientsBySite searches for wireless clients in a specific site using the new implementation
func (c *mistClient) SearchWirelessClientsBySite(ctx context.Context, orgID string, siteID string, text string) (*MistWirelessClientResponse, error) {
	// Get all wireless clients for the text search
	response, err := c.SearchWirelessClients(ctx, orgID, text)
	if err != nil {
		return nil, fmt.Errorf("failed to search wireless clients: %w", err)
	}

	// Filter results by site ID
	var filteredResults []*MistWirelessClient
	for _, client := range response.Results {
		if client.GetSiteID() == siteID {
			filteredResults = append(filteredResults, client)
		}
	}

	// Update response with filtered results
	response.Results = filteredResults
	if response.Total != nil {
		totalFiltered := len(filteredResults)
		response.Total = &totalFiltered
	}

	return response, nil
}

// GetWiredClientDetails retrieves detailed information about a wired client using the new implementation
func (c *mistClient) GetWiredClientDetails(ctx context.Context, orgID string, clientMAC string) (*MistWiredClient, error) {
	// Use search to find the client
	client, err := c.SearchWiredClientsByMAC(ctx, orgID, clientMAC)
	if err != nil {
		return nil, fmt.Errorf("failed to get wired client details: %w", err)
	}

	return client, nil
}

// GetWirelessClientDetails retrieves detailed information about a wireless client using the new implementation
func (c *mistClient) GetWirelessClientDetails(ctx context.Context, orgID string, clientMAC string) (*MistWirelessClient, error) {
	// Use search to find the client
	client, err := c.SearchWirelessClientsByMAC(ctx, orgID, clientMAC)
	if err != nil {
		return nil, fmt.Errorf("failed to get wireless client details: %w", err)
	}

	return client, nil
}

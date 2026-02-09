package api

import (
	"context"
	"testing"
)

// TestMockClientSplitStructure verifies that the mock client works correctly
// after being split into multiple files
func TestMockClientSplitStructure(t *testing.T) {
	// Create a mock client
	client := NewMockClient(Config{
		Organization: "test-org-123",
	})

	mockClient, ok := client.(*MockClient)
	if !ok {
		t.Fatal("Failed to cast to MockClient")
	}

	ctx := context.Background()

	// Test site operations (from mock_client_sites.go)
	t.Run("SiteOperations", func(t *testing.T) {
		// Add a mock site
		testSite := Site{
			Id:          UUIDPtr("site-123"),
			Name:        StringPtr("Test Site"),
			Address:     StringPtr("123 Test St"),
			CountryCode: StringPtr("US"),
			Timezone:    StringPtr("America/Los_Angeles"),
		}
		mockClient.AddMockSite(testSite)

		// Get sites
		sites, err := client.GetSites(ctx, "test-org-123")
		if err != nil {
			t.Fatalf("GetSites failed: %v", err)
		}
		if len(sites) != 1 {
			t.Errorf("Expected 1 site, got %d", len(sites))
		}
		if sites[0].GetID() != "site-123" {
			t.Errorf("Expected site ID 'site-123', got '%s'", sites[0].GetID())
		}
	})

	// Test inventory operations (from mock_client_inventory.go)
	t.Run("InventoryOperations", func(t *testing.T) {
		// Add mock inventory item
		testItem := InventoryItem{
			Id:     UUIDPtr("inv-123"),
			Mac:    StringPtr("001122334455"),
			Serial: StringPtr("SN123456"),
			Name:   StringPtr("Test AP"),
			Model:  StringPtr("AP41"),
			Type:   StringPtr("ap"),
			Magic:  StringPtr("magic-123"),
		}
		mockClient.AddInventoryItem(testItem)

		// Get inventory
		inventory, err := client.GetInventory(ctx, "test-org-123", "")
		if err != nil {
			t.Fatalf("GetInventory failed: %v", err)
		}
		if len(inventory) != 1 {
			t.Errorf("Expected 1 inventory item, got %d", len(inventory))
		}
		if inventory[0].GetMAC() != "001122334455" {
			t.Errorf("Expected MAC '001122334455', got '%s'", inventory[0].GetMAC())
		}
	})

	// Test device operations (from mock_client_devices.go)
	t.Run("DeviceOperations", func(t *testing.T) {
		// Add mock AP
		testAP := AP{
			Id:     UUIDPtr("ap-123"),
			Mac:    StringPtr("aabbccddeeff"),
			Name:   StringPtr("Test AP Device"),
			SiteId: UUIDPtr("site-123"),
			Model:  StringPtr("AP41"),
			Magic:  StringPtr("ap-magic-123"),
		}
		mockClient.AddMockAP(testAP)

		// Get devices
		devices, err := client.GetDevices(ctx, "site-123", "ap")
		if err != nil {
			t.Fatalf("GetDevices failed: %v", err)
		}
		if len(devices) != 1 {
			t.Errorf("Expected 1 device, got %d", len(devices))
		}
		if devices[0].GetMAC() == nil || *devices[0].GetMAC() != "aabbccddeeff" {
			t.Errorf("Expected MAC 'aabbccddeeff', got '%s'", *devices[0].GetMAC())
		}
	})

	// Test auth operations (from mock_client_auth.go)
	t.Run("AuthOperations", func(t *testing.T) {
		// Validate API token
		selfResp, err := mockClient.ValidateAPIToken(ctx)
		if err != nil {
			t.Fatalf("ValidateAPIToken failed: %v", err)
		}
		if selfResp != nil {
			// Verify email field
			if selfResp.Email == nil {
				t.Error("Expected email to be non-nil")
			} else if *selfResp.Email != "mock-user@example.com" {
				t.Errorf("Expected email 'mock-user@example.com', got '%s'", *selfResp.Email)
			}
		} else {
			t.Fatal("Expected non-nil self response")
		}
	})

	// Test cache operations (from mock_client_cache.go)
	t.Run("CacheOperations", func(t *testing.T) {
		// Mock client doesn't need explicit cache operations
		// Just verify the config directory is accessible

		// Get config directory
		configDir := mockClient.GetConfigDirectory()
		if configDir == "" {
			t.Error("Expected non-empty config directory")
		}
	})

	t.Log("All mock client split structure tests passed")
}

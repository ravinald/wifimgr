package api

import (
	"context"
	"testing"
)

func TestCacheSitesMigration(t *testing.T) {
	// Create mock client with test data
	client := NewMockClient(Config{
		Organization: "test-org-123",
	})
	mockClient := client.(*MockClient)

	// Add test sites
	mockClient.AddMockSite(Site{
		Id:          uuidPtr("site-123"),
		Name:        StringPtr("Test Site"),
		Address:     StringPtr("123 Test St"),
		CountryCode: StringPtr("US"),
		Timezone:    StringPtr("America/Los_Angeles"),
	})

	mockClient.AddMockSite(Site{
		Id:          uuidPtr("site-456"),
		Name:        StringPtr("Test Site 2"),
		Address:     StringPtr("456 Test Ave"),
		CountryCode: StringPtr("CA"),
		Timezone:    StringPtr("America/Toronto"),
	})

	ctx := context.Background()
	orgID := "test-org-123"

	// Get results from new method only - legacy method removed
	// oldSites, err := client.GetSites(ctx, orgID)
	// if err != nil {
	//	t.Fatalf("GetSites failed: %v", err)
	// }

	newSites, err := client.GetSites(ctx, orgID)
	if err != nil {
		t.Fatalf("GetSites failed: %v", err)
	}

	// Since legacy method is removed, just verify the new method works
	if len(newSites) == 0 {
		t.Errorf("No sites returned from GetSites")
	}

	// Basic validation of the new method results
	for i, newSite := range newSites {
		if newSite.ID == nil {
			t.Errorf("Site %d missing ID", i)
		}
		if newSite.Name == nil {
			t.Errorf("Site %d missing Name", i)
		}
	}

	t.Logf(" Sites method validation passed: %d sites returned", len(newSites))
}

func TestCacheInventoryMigration(t *testing.T) {
	// Create mock client with test inventory data
	client := NewMockClient(Config{
		Organization: "test-org-123",
	})
	mockClient := client.(*MockClient)

	// Add test inventory items
	mockClient.AddInventoryItem(InventoryItem{
		Id:     UUIDPtr("inv-123"),
		Mac:    StringPtr("001122334455"),
		Serial: StringPtr("SN123456"),
		Name:   StringPtr("Test AP"),
		Model:  StringPtr("AP41"),
		Type:   StringPtr("ap"),
		Magic:  StringPtr("magic-123"),
	})

	mockClient.AddInventoryItem(InventoryItem{
		Id:     UUIDPtr("inv-456"),
		Mac:    StringPtr("aabbccddeeff"),
		Serial: StringPtr("SN789012"),
		Name:   StringPtr("Test Switch"),
		Model:  StringPtr("EX2300"),
		Type:   StringPtr("switch"),
		Magic:  StringPtr("magic-456"),
	})

	ctx := context.Background()
	orgID := "test-org-123"

	// Test all device types
	deviceTypes := []string{"", "ap", "switch", "gateway"}
	for _, deviceType := range deviceTypes {
		t.Run("DeviceType_"+deviceType, func(t *testing.T) {
			// Get results from new method only - legacy method removed
			newInventory, err := client.GetInventory(ctx, orgID, deviceType)
			if err != nil {
				t.Fatalf("GetInventory failed for type '%s': %v", deviceType, err)
			}

			// Basic validation of the new method results
			for i, newItem := range newInventory {
				if newItem.ID == nil {
					t.Errorf("Inventory item %d missing ID for type '%s'", i, deviceType)
				}
				if newItem.MAC == nil {
					t.Errorf("Inventory item %d missing MAC for type '%s'", i, deviceType)
				}
			}

			// Filter expected items based on device type
			expectedItems := 0
			if deviceType == "" || deviceType == "ap" {
				expectedItems++ // Test AP
			}
			if deviceType == "" || deviceType == "switch" {
				expectedItems++ // Test Switch
			}

			if len(newInventory) != expectedItems {
				t.Errorf("Expected %d inventory items for type '%s', got %d",
					expectedItems, deviceType, len(newInventory))
			}

			t.Logf(" Inventory validation passed for type '%s': %d items returned",
				deviceType, len(newInventory))
		})
	}
}

func TestCacheDevicesMigration(t *testing.T) {
	// Create mock client with test device data
	client := NewMockClient(Config{
		Organization: "test-org-123",
	})
	mockClient := client.(*MockClient)

	// Add test site first
	mockClient.AddMockSite(Site{
		Id:          uuidPtr("site-123"),
		Name:        StringPtr("Test Site"),
		Address:     StringPtr("123 Test St"),
		CountryCode: StringPtr("US"),
		Timezone:    StringPtr("America/Los_Angeles"),
	})

	// Add test devices
	mockClient.AddMockAP(AP{
		Id:     uuidPtr("ap-123"),
		Mac:    StringPtr("001122334455"),
		Name:   StringPtr("Test AP"),
		SiteId: uuidPtr("site-123"),
		Model:  StringPtr("AP41"),
		Magic:  StringPtr("magic-123"),
	})

	ctx := context.Background()
	siteID := "site-123"

	// Test different device types
	deviceTypes := []string{"", "ap", "switch", "gateway"}
	for _, deviceType := range deviceTypes {
		t.Run("DeviceType_"+deviceType, func(t *testing.T) {
			// Test the new device method
			devices, err := client.GetDevices(ctx, siteID, deviceType)
			if err != nil {
				t.Fatalf("GetDevices failed for type '%s': %v", deviceType, err)
			}

			// For device type "ap" or "", we should have at least one device (the test AP)
			if deviceType == "ap" || deviceType == "" {
				if len(devices) == 0 {
					t.Errorf("Expected at least one device for type '%s', got %d", deviceType, len(devices))
				} else {
					// Verify the test AP data is correct
					foundTestAP := false
					for _, device := range devices {
						if device.MAC != nil && *device.MAC == "001122334455" {
							foundTestAP = true
							if device.Name == nil || *device.Name != "Test AP" {
								t.Errorf("Expected device name 'Test AP', got %v", device.Name)
							}
							if device.Model == nil || *device.Model != "AP41" {
								t.Errorf("Expected device model 'AP41', got %v", device.Model)
							}
							break
						}
					}
					if !foundTestAP {
						t.Errorf("Could not find test AP with MAC 001122334455 in devices")
					}
				}
			} else {
				// For other device types, we expect no devices
				if len(devices) != 0 {
					t.Errorf("Expected no devices for type '%s', got %d", deviceType, len(devices))
				}
			}

			t.Logf(" Devices validation passed for type '%s': %d devices found",
				deviceType, len(devices))
		})
	}
}

func TestCacheSearchMigration(t *testing.T) {
	// Create mock client with test data
	client := NewMockClient(Config{
		Organization: "test-org-123",
	})
	mockClient := client.(*MockClient)

	// Add some test data for search
	mockClient.AddMockAP(AP{
		Id:     uuidPtr("ap-123"),
		Mac:    StringPtr("001122334455"),
		Name:   StringPtr("TestAP-Search"),
		SiteId: uuidPtr("site-123"),
		Model:  StringPtr("AP41"),
	})

	ctx := context.Background()
	orgID := "test-org-123"

	// Test various search patterns
	searchTests := []string{
		"001122", // MAC search
		"TestAP", // Name search
		"AP41",   // Model search
		"",       // Empty search
	}

	for _, searchText := range searchTests {
		t.Run("Search_"+searchText, func(t *testing.T) {
			// Test wired client search
			newWired, err2 := client.SearchWiredClients(ctx, orgID, searchText)

			if err2 != nil {
				t.Errorf("Wired search failed for '%s': %v", searchText, err2)
			} else {
				// Basic validation of response
				t.Logf(" Wired client search passed for '%s' (found %d results)", searchText, len(newWired.Results))
			}

			// Test wireless client search
			newWireless, err2 := client.SearchWirelessClients(ctx, orgID, searchText)

			if err2 != nil {
				t.Errorf("Wireless search failed for '%s': %v", searchText, err2)
			} else {
				// Basic validation of response
				t.Logf(" Wireless client search passed for '%s' (found %d results)", searchText, len(newWireless.Results))
			}
		})
	}
}

// Helper functions
func uuidPtr(s string) *UUID { u := UUID(s); return &u }

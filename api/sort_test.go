package api

import (
	"reflect"
	"testing"
)

func TestSortAPs(t *testing.T) {
	// Helper function to create AP with given name
	createAP := func(name string) AP {
		return AP{
			Name: &name,
		}
	}

	tests := []struct {
		name     string
		input    []AP
		expected []AP
	}{
		{
			name: "Natural sort APs with numeric components",
			input: []AP{
				createAP("US-NYC-OFFICE-B10-1234"),
				createAP("US-NYC-OFFICE-B1-123"),
				createAP("US-NYC-OFFICE-B2-456"),
				createAP("US-NYC-OFFICE-B2-123"),
				createAP("US-NYC-OFFICE-B10-123"),
			},
			expected: []AP{
				createAP("US-NYC-OFFICE-B1-123"),
				createAP("US-NYC-OFFICE-B2-123"),
				createAP("US-NYC-OFFICE-B2-456"),
				createAP("US-NYC-OFFICE-B10-123"),
				createAP("US-NYC-OFFICE-B10-1234"),
			},
		},
		{
			name: "Natural sorting of mixed AP names",
			input: []AP{
				createAP("AP-non-format-1"),
				createAP("US-NYC-OFFICE-B2-123"),
				createAP("AP-non-format-2"),
			},
			expected: []AP{
				createAP("AP-non-format-1"),
				createAP("AP-non-format-2"),
				createAP("US-NYC-OFFICE-B2-123"),
			},
		},
		{
			name: "Natural sort of non-matching format",
			input: []AP{
				createAP("ZAP-3"),
				createAP("AP-1"),
				createAP("CAP-2"),
			},
			expected: []AP{
				createAP("AP-1"),
				createAP("CAP-2"),
				createAP("ZAP-3"),
			},
		},
		{
			name: "Natural sorting with case differences",
			input: []AP{
				createAP("US-NYC-OFFICE-b2-123"),
				createAP("US-NYC-OFFICE-B1-456"),
			},
			expected: []AP{
				createAP("US-NYC-OFFICE-B1-456"),
				createAP("US-NYC-OFFICE-b2-123"),
			},
		},
		{
			name: "Nil and empty names handled properly",
			input: []AP{
				{}, // Nil name
				createAP(""),
				createAP("US-NYC-OFFICE-B1-123"),
			},
			expected: []AP{
				{},           // Nil name (empty string sorts first)
				createAP(""), // Empty string
				createAP("US-NYC-OFFICE-B1-123"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SortAPs(tt.input)

			// Check length
			if len(result) != len(tt.expected) {
				t.Errorf("SortAPs() returned %d APs, expected %d", len(result), len(tt.expected))
				return
			}

			// Compare names of APs in the result
			for i := 0; i < len(result); i++ {
				resultName := getSafeAPName(&result[i])
				expectedName := getSafeAPName(&tt.expected[i])
				if resultName != expectedName {
					t.Errorf("SortAPs() at index %d got %s, expected %s", i, resultName, expectedName)
				}
			}

			// Also test that the original slice was not modified
			if reflect.DeepEqual(result, tt.input) && !reflect.DeepEqual(tt.expected, tt.input) {
				t.Errorf("SortAPs() modified the original slice, which should not happen")
			}
		})
	}
}

func TestSortSites(t *testing.T) {
	// Helper function to create Site with given name
	createSite := func(name string) Site {
		return Site{
			Name: &name,
		}
	}

	tests := []struct {
		name     string
		input    []Site
		expected []Site
	}{
		{
			name: "Sort Sites alphabetically",
			input: []Site{
				createSite("Chicago"),
				createSite("Boston"),
				createSite("Atlanta"),
				createSite("Denver"),
			},
			expected: []Site{
				createSite("Atlanta"),
				createSite("Boston"),
				createSite("Chicago"),
				createSite("Denver"),
			},
		},
		{
			name: "Case insensitive sorting",
			input: []Site{
				createSite("zebra"),
				createSite("Apple"),
				createSite("banana"),
			},
			expected: []Site{
				createSite("Apple"),
				createSite("banana"),
				createSite("zebra"),
			},
		},
		{
			name: "Nil and empty names handled properly",
			input: []Site{
				{}, // Nil name
				createSite(""),
				createSite("Valid Name"),
			},
			expected: []Site{
				// Both nil and empty names are treated as empty strings and sorted first
				{}, // Nil name
				createSite(""),
				createSite("Valid Name"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SortSites(tt.input)

			// Check length
			if len(result) != len(tt.expected) {
				t.Errorf("SortSites() returned %d Sites, expected %d", len(result), len(tt.expected))
				return
			}

			// Compare names of Sites in the result
			for i := 0; i < len(result); i++ {
				resultName := getSafeSiteName(&result[i])
				expectedName := getSafeSiteName(&tt.expected[i])
				if resultName != expectedName {
					t.Errorf("SortSites() at index %d got %s, expected %s", i, resultName, expectedName)
				}
			}

			// Also test that the original slice was not modified
			if reflect.DeepEqual(result, tt.input) && !reflect.DeepEqual(tt.expected, tt.input) {
				t.Errorf("SortSites() modified the original slice, which should not happen")
			}
		})
	}
}

func TestSortInventory(t *testing.T) {
	// Helper function to create inventory item with given attributes
	createInventoryItem := func(name, hostname, typeName, mac string) InventoryItem {
		item := InventoryItem{}
		if name != "" {
			item.Name = &name
		}
		if hostname != "" {
			item.Hostname = &hostname
		}
		if typeName != "" {
			item.Type = &typeName
		}
		if mac != "" {
			item.Mac = &mac
		}
		return item
	}

	tests := []struct {
		name     string
		input    []InventoryItem
		expected []InventoryItem
	}{
		{
			name: "Sort inventory by name",
			input: []InventoryItem{
				createInventoryItem("Zebra", "", "", ""),
				createInventoryItem("Apple", "", "", ""),
				createInventoryItem("Banana", "", "", ""),
			},
			expected: []InventoryItem{
				createInventoryItem("Apple", "", "", ""),
				createInventoryItem("Banana", "", "", ""),
				createInventoryItem("Zebra", "", "", ""),
			},
		},
		{
			name: "Name takes precedence over hostname",
			input: []InventoryItem{
				createInventoryItem("", "host-a", "", ""),
				createInventoryItem("Named", "", "", ""),
				createInventoryItem("", "host-b", "", ""),
			},
			expected: []InventoryItem{
				createInventoryItem("Named", "", "", ""),
				createInventoryItem("", "host-a", "", ""),
				createInventoryItem("", "host-b", "", ""),
			},
		},
		{
			name: "Hostname takes precedence over type",
			input: []InventoryItem{
				createInventoryItem("", "", "ap", ""),
				createInventoryItem("", "hostname", "", ""),
				createInventoryItem("", "", "switch", ""),
			},
			expected: []InventoryItem{
				createInventoryItem("", "hostname", "", ""),
				createInventoryItem("", "", "ap", ""),
				createInventoryItem("", "", "switch", ""),
			},
		},
		{
			name: "Type takes precedence over MAC",
			input: []InventoryItem{
				createInventoryItem("", "", "", "aa:bb:cc"),
				createInventoryItem("", "", "type", ""),
				createInventoryItem("", "", "", "11:22:33"),
			},
			expected: []InventoryItem{
				createInventoryItem("", "", "type", ""),
				createInventoryItem("", "", "", "11:22:33"),
				createInventoryItem("", "", "", "aa:bb:cc"),
			},
		},
		{
			name: "Complex sorting with mixed attributes",
			input: []InventoryItem{
				createInventoryItem("", "", "switch", ""),
				createInventoryItem("", "host2", "", ""),
				createInventoryItem("name1", "", "", ""),
				createInventoryItem("", "", "", "mac1"),
				createInventoryItem("name2", "", "", ""),
				createInventoryItem("", "host1", "", ""),
				createInventoryItem("", "", "ap", ""),
			},
			expected: []InventoryItem{
				createInventoryItem("name1", "", "", ""),
				createInventoryItem("name2", "", "", ""),
				createInventoryItem("", "host1", "", ""),
				createInventoryItem("", "host2", "", ""),
				createInventoryItem("", "", "ap", ""),
				createInventoryItem("", "", "switch", ""),
				createInventoryItem("", "", "", "mac1"),
			},
		},
		{
			name: "Nil values handled properly",
			input: []InventoryItem{
				{}, // Empty item
				createInventoryItem("Named", "", "", ""),
				createInventoryItem("", "", "", ""),
			},
			expected: []InventoryItem{
				createInventoryItem("Named", "", "", ""),
				createInventoryItem("", "", "", ""),
				{}, // Empty item should be last
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SortInventory(tt.input)

			// Check length
			if len(result) != len(tt.expected) {
				t.Errorf("SortInventory() returned %d items, expected %d", len(result), len(tt.expected))
				return
			}

			// Helper function to get a descriptive string for an item
			itemDescription := func(item *InventoryItem) string {
				namePart := ""
				if item.Name != nil {
					namePart = "Name:" + *item.Name
				}

				hostnamePart := ""
				if item.Hostname != nil {
					hostnamePart = "Host:" + *item.Hostname
				}

				typePart := ""
				if item.Type != nil {
					typePart = "Type:" + *item.Type
				}

				macPart := ""
				if item.Mac != nil {
					macPart = "MAC:" + *item.Mac
				}

				return "[" + namePart + " " + hostnamePart + " " + typePart + " " + macPart + "]"
			}

			// Compare items in the result
			for i := 0; i < len(result); i++ {
				resultDesc := itemDescription(&result[i])
				expectedDesc := itemDescription(&tt.expected[i])

				// Compare name
				if (result[i].Name == nil && tt.expected[i].Name != nil) ||
					(result[i].Name != nil && tt.expected[i].Name == nil) ||
					(result[i].Name != nil && tt.expected[i].Name != nil && *result[i].Name != *tt.expected[i].Name) {
					t.Errorf("SortInventory() at index %d got %s, expected %s", i, resultDesc, expectedDesc)
					continue
				}

				// Compare hostname
				if (result[i].Hostname == nil && tt.expected[i].Hostname != nil) ||
					(result[i].Hostname != nil && tt.expected[i].Hostname == nil) ||
					(result[i].Hostname != nil && tt.expected[i].Hostname != nil && *result[i].Hostname != *tt.expected[i].Hostname) {
					t.Errorf("SortInventory() at index %d got %s, expected %s", i, resultDesc, expectedDesc)
					continue
				}

				// Compare type
				if (result[i].Type == nil && tt.expected[i].Type != nil) ||
					(result[i].Type != nil && tt.expected[i].Type == nil) ||
					(result[i].Type != nil && tt.expected[i].Type != nil && *result[i].Type != *tt.expected[i].Type) {
					t.Errorf("SortInventory() at index %d got %s, expected %s", i, resultDesc, expectedDesc)
					continue
				}

				// Compare MAC
				if (result[i].Mac == nil && tt.expected[i].Mac != nil) ||
					(result[i].Mac != nil && tt.expected[i].Mac == nil) ||
					(result[i].Mac != nil && tt.expected[i].Mac != nil && *result[i].Mac != *tt.expected[i].Mac) {
					t.Errorf("SortInventory() at index %d got %s, expected %s", i, resultDesc, expectedDesc)
				}
			}

			// Also test that the original slice was not modified
			if reflect.DeepEqual(result, tt.input) && !reflect.DeepEqual(tt.expected, tt.input) {
				t.Errorf("SortInventory() modified the original slice, which should not happen")
			}
		})
	}
}

func TestSortSitesNew(t *testing.T) {
	// Helper function to create MistSite with given name
	createMistSite := func(name string) *MistSite {
		return &MistSite{
			Name: &name,
		}
	}

	tests := []struct {
		name     string
		input    []*MistSite
		expected []*MistSite
	}{
		{
			name: "Sort Sites alphabetically",
			input: []*MistSite{
				createMistSite("US-SFO-LAB"),
				createMistSite("US-NYC-OFFICE"),
				createMistSite("US-DEN-LAR"),
			},
			expected: []*MistSite{
				createMistSite("US-DEN-LAR"),
				createMistSite("US-NYC-OFFICE"),
				createMistSite("US-SFO-LAB"),
			},
		},
		{
			name: "Case insensitive sorting",
			input: []*MistSite{
				createMistSite("zebra"),
				createMistSite("Apple"),
				createMistSite("banana"),
			},
			expected: []*MistSite{
				createMistSite("Apple"),
				createMistSite("banana"),
				createMistSite("zebra"),
			},
		},
		{
			name: "Nil and empty names handled properly",
			input: []*MistSite{
				&MistSite{Name: nil},
				createMistSite(""),
				createMistSite("Valid Site"),
			},
			expected: []*MistSite{
				&MistSite{Name: nil},
				createMistSite(""),
				createMistSite("Valid Site"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SortSitesNew(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("SortSitesNew() returned %d sites, expected %d", len(result), len(tt.expected))
				return
			}

			for i := range result {
				resultName := ""
				expectedName := ""

				if result[i].Name != nil {
					resultName = *result[i].Name
				}
				if tt.expected[i].Name != nil {
					expectedName = *tt.expected[i].Name
				}

				if resultName != expectedName {
					t.Errorf("SortSitesNew() at index %d got %q, expected %q", i, resultName, expectedName)
				}
			}

			// Also test that the original slice was not modified
			if reflect.DeepEqual(result, tt.input) && !reflect.DeepEqual(tt.expected, tt.input) {
				t.Errorf("SortSitesNew() modified the original slice, which should not happen")
			}
		})
	}
}

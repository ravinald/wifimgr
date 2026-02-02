package apply

import (
	"slices"
	"testing"

	"github.com/ravinald/wifimgr/internal/macaddr"
)

func TestInventoryChecker_IsInInventory(t *testing.T) {
	tests := []struct {
		name              string
		mac               string
		inAPIInventory    bool
		inLocalInventory  bool
		expectedResult    bool
		description       string
	}{
		{
			name:              "device in both inventories",
			mac:               "aa:bb:cc:dd:ee:ff",
			inAPIInventory:    true,
			inLocalInventory:  true,
			expectedResult:    true,
			description:       "Device exists in vendor account AND is allowlisted for writes",
		},
		{
			name:              "device in API inventory only",
			mac:               "11:22:33:44:55:66",
			inAPIInventory:    true,
			inLocalInventory:  false,
			expectedResult:    false,
			description:       "Device exists in vendor account but NOT allowlisted for writes",
		},
		{
			name:              "device in local inventory only",
			mac:               "aa:aa:aa:aa:aa:aa",
			inAPIInventory:    false,
			inLocalInventory:  true,
			expectedResult:    false,
			description:       "Device is allowlisted but doesn't exist in vendor account",
		},
		{
			name:              "device in neither inventory",
			mac:               "ff:ff:ff:ff:ff:ff",
			inAPIInventory:    false,
			inLocalInventory:  false,
			expectedResult:    false,
			description:       "Device doesn't exist anywhere",
		},
		{
			name:              "empty MAC address",
			mac:               "",
			inAPIInventory:    true,
			inLocalInventory:  true,
			expectedResult:    false,
			description:       "Invalid MAC should always return false",
		},
		{
			name:              "invalid MAC address",
			mac:               "invalid-mac",
			inAPIInventory:    true,
			inLocalInventory:  true,
			expectedResult:    false,
			description:       "Invalid MAC format should always return false",
		},
		{
			name:              "MAC normalization test",
			mac:               "AA-BB-CC-DD-EE-FF", // Different format, same MAC
			inAPIInventory:    true,
			inLocalInventory:  true,
			expectedResult:    true,
			description:       "Different MAC formats should be normalized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create inventory checker with test data
			ic := &InventoryChecker{
				apiInventory:   make(map[string]bool),
				localInventory: make(map[string]bool),
				deviceType:     "ap",
			}

			// Setup test inventories
			if tt.inAPIInventory {
				normalizedMAC := macaddr.NormalizeOrEmpty(tt.mac)
				if normalizedMAC != "" {
					ic.apiInventory[normalizedMAC] = true
				}
			}

			if tt.inLocalInventory {
				normalizedMAC := macaddr.NormalizeOrEmpty(tt.mac)
				if normalizedMAC != "" {
					ic.localInventory[normalizedMAC] = true
				}
			}

			// Test IsInInventory (strict check - requires BOTH)
			result := ic.IsInInventory(tt.mac)
			if result != tt.expectedResult {
				t.Errorf("IsInInventory() = %v, want %v\nDescription: %s\nMAC: %s, inAPI: %v, inLocal: %v",
					result, tt.expectedResult, tt.description, tt.mac, tt.inAPIInventory, tt.inLocalInventory)
			}
		})
	}
}

func TestInventoryChecker_IsInAPIInventory(t *testing.T) {
	tests := []struct {
		name           string
		mac            string
		inAPIInventory bool
		expectedResult bool
	}{
		{
			name:           "device in API inventory",
			mac:            "aa:bb:cc:dd:ee:ff",
			inAPIInventory: true,
			expectedResult: true,
		},
		{
			name:           "device not in API inventory",
			mac:            "11:22:33:44:55:66",
			inAPIInventory: false,
			expectedResult: false,
		},
		{
			name:           "empty MAC",
			mac:            "",
			inAPIInventory: true,
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ic := &InventoryChecker{
				apiInventory: make(map[string]bool),
				deviceType:   "ap",
			}

			if tt.inAPIInventory {
				normalizedMAC := macaddr.NormalizeOrEmpty(tt.mac)
				if normalizedMAC != "" {
					ic.apiInventory[normalizedMAC] = true
				}
			}

			result := ic.IsInAPIInventory(tt.mac)
			if result != tt.expectedResult {
				t.Errorf("IsInAPIInventory() = %v, want %v", result, tt.expectedResult)
			}
		})
	}
}

func TestInventoryChecker_IsInLocalInventory(t *testing.T) {
	tests := []struct {
		name             string
		mac              string
		inLocalInventory bool
		expectedResult   bool
	}{
		{
			name:             "device in local inventory",
			mac:              "aa:bb:cc:dd:ee:ff",
			inLocalInventory: true,
			expectedResult:   true,
		},
		{
			name:             "device not in local inventory",
			mac:              "11:22:33:44:55:66",
			inLocalInventory: false,
			expectedResult:   false,
		},
		{
			name:             "empty MAC",
			mac:              "",
			inLocalInventory: true,
			expectedResult:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ic := &InventoryChecker{
				localInventory: make(map[string]bool),
				deviceType:     "ap",
			}

			if tt.inLocalInventory {
				normalizedMAC := macaddr.NormalizeOrEmpty(tt.mac)
				if normalizedMAC != "" {
					ic.localInventory[normalizedMAC] = true
				}
			}

			result := ic.IsInLocalInventory(tt.mac)
			if result != tt.expectedResult {
				t.Errorf("IsInLocalInventory() = %v, want %v", result, tt.expectedResult)
			}
		})
	}
}

func TestInventoryChecker_FilterByInventory(t *testing.T) {
	tests := []struct {
		name         string
		inputMACs    []string
		apiDevices   []string
		localDevices []string
		expected     []string
		description  string
	}{
		{
			name:         "filter keeps devices in both inventories",
			inputMACs:    []string{"aa:bb:cc:dd:ee:ff", "11:22:33:44:55:66", "aa:aa:aa:aa:aa:aa"},
			apiDevices:   []string{"aa:bb:cc:dd:ee:ff", "11:22:33:44:55:66"},
			localDevices: []string{"aa:bb:cc:dd:ee:ff", "aa:aa:aa:aa:aa:aa"},
			expected:     []string{"aa:bb:cc:dd:ee:ff"},
			description:  "Only device in BOTH inventories should pass",
		},
		{
			name:         "filter removes device in API only",
			inputMACs:    []string{"aa:bb:cc:dd:ee:ff", "11:22:33:44:55:66"},
			apiDevices:   []string{"aa:bb:cc:dd:ee:ff", "11:22:33:44:55:66"},
			localDevices: []string{"aa:bb:cc:dd:ee:ff"},
			expected:     []string{"aa:bb:cc:dd:ee:ff"},
			description:  "Device in API only should be filtered out",
		},
		{
			name:         "filter removes device in local only",
			inputMACs:    []string{"aa:bb:cc:dd:ee:ff", "aa:aa:aa:aa:aa:aa"},
			apiDevices:   []string{"aa:bb:cc:dd:ee:ff"},
			localDevices: []string{"aa:bb:cc:dd:ee:ff", "aa:aa:aa:aa:aa:aa"},
			expected:     []string{"aa:bb:cc:dd:ee:ff"},
			description:  "Device in local only should be filtered out",
		},
		{
			name:         "all devices pass filter",
			inputMACs:    []string{"aa:bb:cc:dd:ee:ff", "11:22:33:44:55:66"},
			apiDevices:   []string{"aa:bb:cc:dd:ee:ff", "11:22:33:44:55:66"},
			localDevices: []string{"aa:bb:cc:dd:ee:ff", "11:22:33:44:55:66"},
			expected:     []string{"aa:bb:cc:dd:ee:ff", "11:22:33:44:55:66"},
			description:  "All devices in both inventories should pass",
		},
		{
			name:         "no devices pass filter",
			inputMACs:    []string{"11:22:33:44:55:66", "aa:aa:aa:aa:aa:aa"},
			apiDevices:   []string{"11:22:33:44:55:66"},
			localDevices: []string{"aa:aa:aa:aa:aa:aa"},
			expected:     []string{},
			description:  "No devices in both inventories",
		},
		{
			name:         "empty input list",
			inputMACs:    []string{},
			apiDevices:   []string{"aa:bb:cc:dd:ee:ff"},
			localDevices: []string{"aa:bb:cc:dd:ee:ff"},
			expected:     []string{},
			description:  "Empty input should return empty output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ic := &InventoryChecker{
				apiInventory:   make(map[string]bool),
				localInventory: make(map[string]bool),
				deviceType:     "ap",
			}

			// Populate API inventory
			for _, mac := range tt.apiDevices {
				normalizedMAC := macaddr.NormalizeOrEmpty(mac)
				if normalizedMAC != "" {
					ic.apiInventory[normalizedMAC] = true
				}
			}

			// Populate local inventory
			for _, mac := range tt.localDevices {
				normalizedMAC := macaddr.NormalizeOrEmpty(mac)
				if normalizedMAC != "" {
					ic.localInventory[normalizedMAC] = true
				}
			}

			result := ic.FilterByInventory(tt.inputMACs)

			// Compare results
			if len(result) != len(tt.expected) {
				t.Errorf("FilterByInventory() returned %d items, want %d\nDescription: %s\nGot: %v\nWant: %v",
					len(result), len(tt.expected), tt.description, result, tt.expected)
				return
			}

			// Check each expected MAC is in result
			for _, expectedMAC := range tt.expected {
				if !slices.Contains(result, expectedMAC) {
					t.Errorf("FilterByInventory() missing expected MAC %s\nDescription: %s\nGot: %v\nWant: %v",
						expectedMAC, tt.description, result, tt.expected)
				}
			}
		})
	}
}

func TestInventoryChecker_WriteOperationScenarios(t *testing.T) {
	// This test verifies the core requirement: write operations must check BOTH inventories
	scenarios := []struct {
		name              string
		mac               string
		inAPIInventory    bool
		inLocalInventory  bool
		shouldAllowWrite  bool
		description       string
	}{
		{
			name:              "write allowed - device in both",
			mac:               "aa:bb:cc:dd:ee:ff",
			inAPIInventory:    true,
			inLocalInventory:  true,
			shouldAllowWrite:  true,
			description:       "Device exists and is allowlisted - write should be allowed",
		},
		{
			name:              "write denied - device not allowlisted",
			mac:               "11:22:33:44:55:66",
			inAPIInventory:    true,
			inLocalInventory:  false,
			shouldAllowWrite:  false,
			description:       "Device exists but not allowlisted - write should be denied",
		},
		{
			name:              "write denied - device doesn't exist",
			mac:               "aa:aa:aa:aa:aa:aa",
			inAPIInventory:    false,
			inLocalInventory:  true,
			shouldAllowWrite:  false,
			description:       "Device is allowlisted but doesn't exist - write should be denied",
		},
		{
			name:              "write denied - device unknown",
			mac:               "ff:ff:ff:ff:ff:ff",
			inAPIInventory:    false,
			inLocalInventory:  false,
			shouldAllowWrite:  false,
			description:       "Device unknown everywhere - write should be denied",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			ic := &InventoryChecker{
				apiInventory:   make(map[string]bool),
				localInventory: make(map[string]bool),
				deviceType:     "ap",
			}

			if scenario.inAPIInventory {
				normalizedMAC := macaddr.NormalizeOrEmpty(scenario.mac)
				ic.apiInventory[normalizedMAC] = true
			}

			if scenario.inLocalInventory {
				normalizedMAC := macaddr.NormalizeOrEmpty(scenario.mac)
				ic.localInventory[normalizedMAC] = true
			}

			// Simulate write operation safety check
			canWrite := ic.IsInInventory(scenario.mac)

			if canWrite != scenario.shouldAllowWrite {
				t.Errorf("Write operation check failed:\n"+
					"Description: %s\n"+
					"MAC: %s\n"+
					"In API: %v, In Local: %v\n"+
					"Can write: %v, Should allow: %v",
					scenario.description, scenario.mac,
					scenario.inAPIInventory, scenario.inLocalInventory,
					canWrite, scenario.shouldAllowWrite)
			}
		})
	}
}

func TestInventoryChecker_ReadOperationScenarios(t *testing.T) {
	// This test verifies that read operations only need API inventory
	scenarios := []struct {
		name              string
		mac               string
		inAPIInventory    bool
		inLocalInventory  bool
		shouldAllowRead   bool
		description       string
	}{
		{
			name:              "read allowed - device in both",
			mac:               "aa:bb:cc:dd:ee:ff",
			inAPIInventory:    true,
			inLocalInventory:  true,
			shouldAllowRead:   true,
			description:       "Device exists and is allowlisted - read allowed",
		},
		{
			name:              "read allowed - device in API only",
			mac:               "11:22:33:44:55:66",
			inAPIInventory:    true,
			inLocalInventory:  false,
			shouldAllowRead:   true,
			description:       "Device exists but not allowlisted - read still allowed",
		},
		{
			name:              "read denied - device in local only",
			mac:               "aa:aa:aa:aa:aa:aa",
			inAPIInventory:    false,
			inLocalInventory:  true,
			shouldAllowRead:   false,
			description:       "Device is allowlisted but doesn't exist - read denied",
		},
		{
			name:              "read denied - device unknown",
			mac:               "ff:ff:ff:ff:ff:ff",
			inAPIInventory:    false,
			inLocalInventory:  false,
			shouldAllowRead:   false,
			description:       "Device unknown - read denied",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			ic := &InventoryChecker{
				apiInventory:   make(map[string]bool),
				localInventory: make(map[string]bool),
				deviceType:     "ap",
			}

			if scenario.inAPIInventory {
				normalizedMAC := macaddr.NormalizeOrEmpty(scenario.mac)
				ic.apiInventory[normalizedMAC] = true
			}

			if scenario.inLocalInventory {
				normalizedMAC := macaddr.NormalizeOrEmpty(scenario.mac)
				ic.localInventory[normalizedMAC] = true
			}

			// Simulate read operation check (only needs API inventory)
			canRead := ic.IsInAPIInventory(scenario.mac)

			if canRead != scenario.shouldAllowRead {
				t.Errorf("Read operation check failed:\n"+
					"Description: %s\n"+
					"MAC: %s\n"+
					"In API: %v, In Local: %v\n"+
					"Can read: %v, Should allow: %v",
					scenario.description, scenario.mac,
					scenario.inAPIInventory, scenario.inLocalInventory,
					canRead, scenario.shouldAllowRead)
			}
		})
	}
}

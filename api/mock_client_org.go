package api

import (
	"context"
)

// GetOrgStats retrieves organization statistics for the mock client
func (m *MockClient) GetOrgStats(_ context.Context, orgID string) (*OrgStats, error) {
	// Return mock org stats
	orgName := "Mock Organization"
	allowMist := true
	createdTime := float64(1609459200) // 2021-01-01
	modifiedTime := float64(1609459200)
	mspID := "mock-msp-id"
	numDevices := 10
	numDevicesConnected := 8
	numDevicesDisconnected := 2
	numInventory := 10
	numSites := 3
	sessionExpiry := int64(86400)

	return &OrgStats{
		ID:                     &orgID,
		Name:                   &orgName,
		AllowMist:              &allowMist,
		CreatedTime:            &createdTime,
		ModifiedTime:           &modifiedTime,
		MspID:                  &mspID,
		NumDevices:             &numDevices,
		NumDevicesConnected:    &numDevicesConnected,
		NumDevicesDisconnected: &numDevicesDisconnected,
		NumInventory:           &numInventory,
		NumSites:               &numSites,
		SessionExpiry:          &sessionExpiry,
		OrgGroupIDs:            []string{},
		SLE:                    []*OrgStatsSLE{},
	}, nil
}

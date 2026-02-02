package vendors

import (
	"testing"
)

// TestCacheAccessorMethodsExist verifies all methods exist and are exported
func TestCacheAccessorMethodsExist(t *testing.T) {
	var ca *CacheAccessor
	
	// This is a compile-time test to verify all methods exist
	methods := []interface{}{
		ca.GetSiteByID,
		ca.GetSiteByName,
		ca.GetRFTemplateByID,
		ca.GetGWTemplateByID,
		ca.GetWLANTemplateByID,
		ca.GetNetworkByID,
		ca.GetDeviceProfileByID,
		ca.GetDevicesBySite,
		ca.GetAPConfigByMAC,
		ca.GetSwitchConfigByMAC,
		ca.GetGatewayConfigByMAC,
		ca.GetDeviceByMAC,
		ca.GetAllWLANs,
		ca.GetAllAPs,
		ca.GetAllSites,
		ca.GetAllDevices,
		ca.RebuildIndexes,
		ca.GetManager,
		ca.IsInitialized,
		ca.GetStats,
	}
	
	if len(methods) != 20 {
		t.Errorf("Expected 20 methods, got %d", len(methods))
	}
}

package config

import "testing"

func TestDuplicateTracker_APIScope(t *testing.T) {
	tests := []struct {
		name    string
		api1    string
		api2    string
		wantDup bool
	}{
		{"same name different API is allowed", "mist", "aruba-pina", false},
		{"same name same API is a duplicate", "mist", "mist", true},
		{"same name both unscoped is a duplicate", "", "", true},
		{"scoped vs unscoped do not collide", "mist", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt := NewDuplicateTracker()
			if dt.CheckAndAdd("site_config", "", "US-OAK-PINA", tt.api1, "a.json", 1) {
				t.Fatal("first add reported a duplicate")
			}
			got := dt.CheckAndAdd("site_config", "", "US-OAK-PINA", tt.api2, "b.json", 1)
			if got != tt.wantDup {
				t.Errorf("second add duplicate = %v, want %v", got, tt.wantDup)
			}
		})
	}
}

func TestDuplicateTracker_DeviceProfilesUnchanged(t *testing.T) {
	dt := NewDuplicateTracker()
	if dt.CheckAndAdd("device_profile", "ap", "office-ap", "", "p.json", 1) {
		t.Fatal("first profile reported a duplicate")
	}
	if !dt.CheckAndAdd("device_profile", "ap", "office-ap", "", "q.json", 1) {
		t.Error("same profile name+type should still be a duplicate")
	}
	if dt.CheckAndAdd("device_profile", "switch", "office-ap", "", "r.json", 1) {
		t.Error("same name under a different device type is not a duplicate")
	}
}

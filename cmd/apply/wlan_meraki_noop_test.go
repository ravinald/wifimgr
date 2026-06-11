package apply

import (
	"testing"

	configPkg "github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// TestMerakiWLAN_RawBlockIsNoOp proves the import → apply round-trip is a functional
// no-op: a WLAN whose meraki: vendor block carries the raw band/auth tokens and the
// real availability model expands to a desired WLAN that matches the live SSID, so
// merakiWLANNeedsUpdate reports no change. This is the guarantee that canonical
// band/auth alone (which can't distinguish Meraki's verbose enums) would break.
func TestMerakiWLAN_RawBlockIsNoOp(t *testing.T) {
	template := map[string]any{
		"ssid":    "Corp",
		"enabled": true,
		"band":    "dual", // portable canonical, overridden for Meraki by the block
		"auth":    map[string]any{"type": "eap"},
		"meraki:": map[string]any{
			"number":            float64(2),
			"band":              "Dual band operation with Band Steering",
			"auth":              map[string]any{"type": "8021x-radius"},
			"availabilityTags":  []any{"lobby"},
			"availableOnAllAps": false,
		},
	}

	expanded := configPkg.ExpandForVendor(template, "meraki")
	desired := buildVendorWLANFromConfig(expanded, "L_1")
	if _, set := desired.Config["availableOnAllAps"]; !set {
		tags := extractStringSliceFromConfig(desired.Config, "availabilityTags")
		desired.Config["availableOnAllAps"] = len(tags) == 0
	}

	// The live SSID as the cache holds it — raw Meraki tokens, real tags.
	existing := &vendors.WLAN{
		SSID:     "Corp",
		Enabled:  true,
		Band:     "Dual band operation with Band Steering",
		AuthType: "8021x-radius",
		Config: map[string]any{
			"availabilityTags":  []any{"lobby"},
			"availableOnAllAps": false,
		},
	}

	if merakiWLANNeedsUpdate(existing, desired) {
		t.Errorf("expected no-op; desired band=%q auth=%q config=%v", desired.Band, desired.AuthType, desired.Config)
	}

	// Sanity: a genuine change (different tag) must still register.
	existing.Config["availabilityTags"] = []any{"warehouse"}
	if !merakiWLANNeedsUpdate(existing, desired) {
		t.Error("expected a change when availability tags differ")
	}
}

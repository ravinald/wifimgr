package apply

import (
	"testing"

	"github.com/ravinald/wifimgr/internal/vendors"
)

func TestResolveMerakiWLANTarget(t *testing.T) {
	const net = "L_123"
	bySSID := map[string]*vendors.WLAN{
		"Corp": {ID: net + ":2", SSID: "Corp"},
	}
	bySlot := map[int]*vendors.WLAN{
		2: {ID: net + ":2", SSID: "Corp"},
	}

	tests := []struct {
		name       string
		ssid       string
		cfg        map[string]any
		wantAction merakiWLANAction
		wantTarget string
		wantSlot   int
	}{
		{
			// Pinned to an active slot whose live name differs → rename in place,
			// not a new slot.
			name:       "pinned rename in place",
			ssid:       "Corp-Renamed",
			cfg:        map[string]any{"number": float64(2)},
			wantAction: merakiWLANUpdate,
			wantTarget: net + ":2",
			wantSlot:   2,
		},
		{
			// Pinned to a slot not currently active (disabled SSID) → configure
			// that exact slot rather than allocating a free one.
			name:       "pinned inactive slot",
			ssid:       "Guest",
			cfg:        map[string]any{"number": float64(7)},
			wantAction: merakiWLANConfigureSlot,
			wantTarget: net + ":7",
			wantSlot:   7,
		},
		{
			// No pin, SSID name matches an active slot → update that slot.
			name:       "name match",
			ssid:       "Corp",
			cfg:        map[string]any{},
			wantAction: merakiWLANUpdate,
			wantTarget: net + ":2",
		},
		{
			// No pin, no name match → brand-new SSID, allocate a free slot.
			name:       "create new",
			ssid:       "IoT",
			cfg:        map[string]any{},
			wantAction: merakiWLANCreate,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			action, existing, target, slot := resolveMerakiWLANTarget(net, tc.ssid, tc.cfg, bySSID, bySlot)
			if action != tc.wantAction {
				t.Errorf("action = %d, want %d", action, tc.wantAction)
			}
			if target != tc.wantTarget {
				t.Errorf("targetID = %q, want %q", target, tc.wantTarget)
			}
			if slot != tc.wantSlot {
				t.Errorf("pinnedSlot = %d, want %d", slot, tc.wantSlot)
			}
			if tc.wantAction == merakiWLANUpdate && existing == nil {
				t.Error("expected non-nil existing WLAN for update action")
			}
			if tc.wantAction != merakiWLANUpdate && existing != nil {
				t.Errorf("expected nil existing for non-update action, got %+v", existing)
			}
		})
	}
}

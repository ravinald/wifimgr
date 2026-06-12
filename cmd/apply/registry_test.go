package apply

import (
	"slices"
	"strings"
	"testing"
)

func TestGetDeviceUpdaterResolvesRegisteredTypes(t *testing.T) {
	for _, deviceType := range []string{"ap", "switch", "gateway"} {
		updater, err := getDeviceUpdater(deviceType)
		if err != nil {
			t.Fatalf("getDeviceUpdater(%q) returned error: %v", deviceType, err)
		}
		if updater == nil {
			t.Fatalf("getDeviceUpdater(%q) returned nil updater", deviceType)
		}
		if got := updater.GetDeviceType(); got != deviceType {
			t.Errorf("getDeviceUpdater(%q): updater reports type %q", deviceType, got)
		}
	}
}

func TestGetDeviceUpdaterUnknownTypeErrors(t *testing.T) {
	_, err := getDeviceUpdater("router")
	if err == nil {
		t.Fatal("getDeviceUpdater(\"router\") expected an error, got nil")
	}
	// The error must name the supported types so the operator can self-correct.
	if !strings.Contains(err.Error(), "ap") {
		t.Errorf("error message %q does not list supported types", err.Error())
	}
}

func TestSupportedDeviceTypesIncludesCoreTypes(t *testing.T) {
	types := SupportedDeviceTypes()
	for _, want := range []string{"ap", "switch", "gateway"} {
		if !slices.Contains(types, want) {
			t.Errorf("SupportedDeviceTypes() = %v, missing %q", types, want)
		}
	}
}

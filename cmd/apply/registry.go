package apply

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// UpdaterFactory builds a DeviceUpdater for a registered device type.
type UpdaterFactory func() DeviceUpdater

// deviceTypeRegistry maps a normalized device type to its updater factory.
// Device-type support lives here rather than in a hard-coded switch so that
// adding a type is a single RegisterDeviceType call in that type's own file —
// not an edit fanned out across the apply package.
type deviceTypeRegistry struct {
	mu        sync.RWMutex
	factories map[string]UpdaterFactory
}

var updaterRegistry = &deviceTypeRegistry{
	factories: make(map[string]UpdaterFactory),
}

// RegisterDeviceType registers an updater factory for a device type. Call it
// from an init() in the device type's implementation file so the type and its
// registration stay co-located.
func RegisterDeviceType(deviceType string, factory UpdaterFactory) {
	updaterRegistry.mu.Lock()
	defer updaterRegistry.mu.Unlock()
	updaterRegistry.factories[deviceType] = factory
}

// SupportedDeviceTypes returns the registered device types in sorted order.
func SupportedDeviceTypes() []string {
	updaterRegistry.mu.RLock()
	defer updaterRegistry.mu.RUnlock()

	types := make([]string, 0, len(updaterRegistry.factories))
	for t := range updaterRegistry.factories {
		types = append(types, t)
	}
	sort.Strings(types)
	return types
}

// getDeviceUpdater returns the DeviceUpdater for a device type, or an error
// naming the supported types when none is registered.
func getDeviceUpdater(deviceType string) (DeviceUpdater, error) {
	updaterRegistry.mu.RLock()
	factory, ok := updaterRegistry.factories[deviceType]
	updaterRegistry.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unsupported device type %q (supported: %s)",
			deviceType, strings.Join(SupportedDeviceTypes(), ", "))
	}
	return factory(), nil
}

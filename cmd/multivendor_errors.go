package cmd

import (
	"fmt"
	"strings"
)

// FormatAPINotFoundError creates a helpful error message when an API is not found.
func FormatAPINotFoundError(requestedAPI string) error {
	registry := GetAPIRegistry()
	if registry == nil {
		return fmt.Errorf("API '%s' not found (API registry not initialized)", requestedAPI)
	}

	availableAPIs := registry.GetAllLabels()
	if len(availableAPIs) == 0 {
		return fmt.Errorf("API '%s' not found - no APIs are configured", requestedAPI)
	}

	return fmt.Errorf("API '%s' not found\n\nAvailable APIs:\n  %s\n\nUse --api <label> to specify which API to use",
		requestedAPI, strings.Join(availableAPIs, "\n  "))
}

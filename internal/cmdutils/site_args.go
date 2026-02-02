package cmdutils

import (
	"fmt"

	"github.com/ravinald/wifimgr/internal/macaddr"
)

// ValidateDeviceType validates that the device type is supported
func ValidateDeviceType(deviceType string) error {
	if deviceType == "" {
		return nil // Empty device type means "all"
	}

	validTypes := map[string]bool{
		"ap":       true,
		"aps":      true,
		"switch":   true,
		"switches": true,
		"sw":       true,
		"gateway":  true,
		"gateways": true,
		"gw":       true,
	}

	if !validTypes[deviceType] {
		return fmt.Errorf("invalid device type: %s. Valid types: ap, switch, gateway", deviceType)
	}

	return nil
}

// NormalizeDeviceType converts device type aliases to canonical form
func NormalizeDeviceType(deviceType string) string {
	switch deviceType {
	case "aps":
		return "ap"
	case "switches", "sw":
		return "switch"
	case "gateways", "gw":
		return "gateway"
	default:
		return deviceType
	}
}

// IsMAC checks if a string is a valid MAC address using the macaddr package
func IsMAC(s string) bool {
	return macaddr.IsValid(s)
}

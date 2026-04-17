package ubiquiti

import "strings"

// classifyDevice determines the device type using both model and shortname.
// The Site Manager API returns human-readable model names (e.g., "Nano HD", "AC Pro")
// while shortnames contain the model codes (e.g., "U7NHD", "U7PG2").
func classifyDevice(d Device) string {
	if typ := ClassifyDeviceType(d.Model); typ != "other" {
		return typ
	}
	return classifyByShortname(d.Shortname)
}

// ClassifyDeviceType determines the device type from its model string.
// Handles both dash-separated codes ("U6-LR") and space-separated names ("U6 LR").
// Returns "ap", "switch", "gateway", or "other".
func ClassifyDeviceType(model string) string {
	// Normalize: uppercase and replace spaces with dashes for prefix matching
	upper := strings.ToUpper(strings.ReplaceAll(model, " ", "-"))

	// AP models
	apPrefixes := []string{"U6-", "U7-", "UAP-", "UBB-", "AC-", "NANO"}
	for _, prefix := range apPrefixes {
		if strings.HasPrefix(upper, prefix) {
			return "ap"
		}
	}

	// Switch models
	switchPrefixes := []string{"USW-", "USL-", "US-"}
	for _, prefix := range switchPrefixes {
		if strings.HasPrefix(upper, prefix) {
			// Exclude USG (gateway) which would match US- prefix
			if strings.HasPrefix(upper, "USG") {
				break
			}
			return "switch"
		}
	}

	// Gateway models
	gatewayPrefixes := []string{"UDM-", "UXG-", "USG-", "UCG-", "UDR-"}
	for _, prefix := range gatewayPrefixes {
		if strings.HasPrefix(upper, prefix) {
			return "gateway"
		}
	}

	return "other"
}

// classifyByShortname classifies using the shortname model code.
// Shortnames are compact codes like "U7NHD", "USL48PB", "UALR6v2".
func classifyByShortname(shortname string) string {
	upper := strings.ToUpper(shortname)

	// AP shortnames: U7xxx (nanoHD, AC Pro, AC LR), UAxx (U6 variants)
	apPrefixes := []string{"U7", "UA", "U6"}
	for _, prefix := range apPrefixes {
		if strings.HasPrefix(upper, prefix) {
			return "ap"
		}
	}

	// Switch shortnames: USWxx, USLxx
	switchPrefixes := []string{"USW", "USL"}
	for _, prefix := range switchPrefixes {
		if strings.HasPrefix(upper, prefix) {
			return "switch"
		}
	}

	// Gateway shortnames: UDMxx, UXGxx, USGxx, UCGxx, UDRxx
	gatewayPrefixes := []string{"UDM", "UXG", "USG", "UCG", "UDR"}
	for _, prefix := range gatewayPrefixes {
		if strings.HasPrefix(upper, prefix) {
			return "gateway"
		}
	}

	return "other"
}

// IsNetworkDevice returns true if the device belongs to the "network" product line.
func IsNetworkDevice(d Device) bool {
	return strings.EqualFold(d.ProductLine, "network")
}

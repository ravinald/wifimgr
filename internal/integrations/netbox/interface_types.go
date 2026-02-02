package netbox

import (
	"sort"
)

// ValidInterfaceTypes maps NetBox interface type values to their display labels.
// These correspond to NetBox's InterfaceTypeValue enum.
// Reference: https://github.com/netbox-community/netbox/blob/develop/netbox/dcim/choices.py
var ValidInterfaceTypes = map[string]string{
	// Virtual
	"virtual": "Virtual",
	"bridge":  "Bridge",
	"lag":     "Link Aggregation Group (LAG)",

	// Ethernet (Fixed)
	"100base-fx":          "100BASE-FX (10/100ME FIBER)",
	"100base-lfx":         "100BASE-LFX (10/100ME FIBER)",
	"100base-tx":          "100BASE-TX (10/100ME)",
	"100base-t1":          "100BASE-T1 (10/100ME Single Pair)",
	"1000base-t":          "1000BASE-T (1GE)",
	"1000base-x-gbic":     "GBIC (1GE)",
	"1000base-x-sfp":      "SFP (1GE)",
	"2.5gbase-t":          "2.5GBASE-T (2.5GE)",
	"5gbase-t":            "5GBASE-T (5GE)",
	"10gbase-t":           "10GBASE-T (10GE)",
	"10gbase-cx4":         "10GBASE-CX4 (10GE)",
	"10gbase-x-sfpp":      "SFP+ (10GE)",
	"10gbase-x-xfp":       "XFP (10GE)",
	"10gbase-x-xenpak":    "XENPAK (10GE)",
	"10gbase-x-x2":        "X2 (10GE)",
	"25gbase-x-sfp28":     "SFP28 (25GE)",
	"50gbase-x-sfp56":     "SFP56 (50GE)",
	"40gbase-x-qsfpp":     "QSFP+ (40GE)",
	"50gbase-x-sfp28":     "QSFP28 (50GE)",
	"100gbase-x-cfp":      "CFP (100GE)",
	"100gbase-x-cfp2":     "CFP2 (100GE)",
	"100gbase-x-cfp4":     "CFP4 (100GE)",
	"100gbase-x-cxp":      "CXP (100GE)",
	"100gbase-x-cpak":     "Cisco CPAK (100GE)",
	"100gbase-x-dsfp":     "DSFP (100GE)",
	"100gbase-x-sfpdd":    "SFP-DD (100GE)",
	"100gbase-x-qsfp28":   "QSFP28 (100GE)",
	"100gbase-x-qsfpdd":   "QSFP-DD (100GE)",
	"200gbase-x-cfp2":     "CFP2 (200GE)",
	"200gbase-x-qsfp56":   "QSFP56 (200GE)",
	"200gbase-x-qsfpdd":   "QSFP-DD (200GE)",
	"400gbase-x-qsfp112":  "QSFP112 (400GE)",
	"400gbase-x-qsfpdd":   "QSFP-DD (400GE)",
	"400gbase-x-osfp":     "OSFP (400GE)",
	"400gbase-x-osfp-rhs": "OSFP-RHS (400GE)",
	"400gbase-x-cdfp":     "CDFP (400GE)",
	"400gbase-x-cfp8":     "CPF8 (400GE)",
	"800gbase-x-qsfpdd":   "QSFP-DD (800GE)",
	"800gbase-x-osfp":     "OSFP (800GE)",

	// Wireless
	"ieee802.11a":  "IEEE 802.11a",
	"ieee802.11g":  "IEEE 802.11b/g",
	"ieee802.11n":  "IEEE 802.11n (Wi-Fi 4)",
	"ieee802.11ac": "IEEE 802.11ac (Wi-Fi 5)",
	"ieee802.11ax": "IEEE 802.11ax (Wi-Fi 6)",
	"ieee802.11ay": "IEEE 802.11ay (Wi-Fi 7)",
	"ieee802.11be": "IEEE 802.11be (Wi-Fi 7)",
	"ieee802.15.1": "Bluetooth",

	// Cellular
	"gsm":  "GSM",
	"cdma": "CDMA",
	"lte":  "LTE",
	"4g":   "4G",
	"5g":   "5G",

	// SONET
	"sonet-oc3":    "OC-3/STM-1",
	"sonet-oc12":   "OC-12/STM-4",
	"sonet-oc48":   "OC-48/STM-16",
	"sonet-oc192":  "OC-192/STM-64",
	"sonet-oc768":  "OC-768/STM-256",
	"sonet-oc1920": "OC-1920/STM-640",
	"sonet-oc3840": "OC-3840/STM-1234",

	// FibreChannel
	"1gfc-sfp":      "SFP (1GFC)",
	"2gfc-sfp":      "SFP (2GFC)",
	"4gfc-sfp":      "SFP (4GFC)",
	"8gfc-sfpp":     "SFP+ (8GFC)",
	"16gfc-sfpp":    "SFP+ (16GFC)",
	"32gfc-sfp28":   "SFP28 (32GFC)",
	"64gfc-qsfpp":   "QSFP+ (64GFC)",
	"128gfc-qsfp28": "QSFP28 (128GFC)",

	// InfiniBand
	"infiniband-sdr":   "SDR (2 Gbps)",
	"infiniband-ddr":   "DDR (4 Gbps)",
	"infiniband-qdr":   "QDR (8 Gbps)",
	"infiniband-fdr10": "FDR10 (10 Gbps)",
	"infiniband-fdr":   "FDR (13.64 Gbps)",
	"infiniband-edr":   "EDR (25 Gbps)",
	"infiniband-hdr":   "HDR (50 Gbps)",
	"infiniband-ndr":   "NDR (100 Gbps)",
	"infiniband-xdr":   "XDR (250 Gbps)",

	// Serial
	"t1": "T1 (1.544 Mbps)",
	"e1": "E1 (2.048 Mbps)",
	"t3": "T3 (44.736 Mbps)",
	"e3": "E3 (34.368 Mbps)",

	// Stacking
	"cisco-stackwise":         "Cisco StackWise",
	"cisco-stackwise-plus":    "Cisco StackWise Plus",
	"cisco-flexstack":         "Cisco FlexStack",
	"cisco-flexstack-plus":    "Cisco FlexStack Plus",
	"cisco-stackwise-80":      "Cisco StackWise-80",
	"cisco-stackwise-160":     "Cisco StackWise-160",
	"cisco-stackwise-320":     "Cisco StackWise-320",
	"cisco-stackwise-480":     "Cisco StackWise-480",
	"cisco-stackwise-1t":      "Cisco StackWise-1T",
	"juniper-vcp":             "Juniper VCP",
	"extreme-summitstack":     "Extreme SummitStack",
	"extreme-summitstack-128": "Extreme SummitStack-128",
	"extreme-summitstack-256": "Extreme SummitStack-256",
	"extreme-summitstack-512": "Extreme SummitStack-512",

	// Other
	"other": "Other",
}

// CommonInterfaceTypes contains the most commonly used interface types for quick reference
var CommonInterfaceTypes = []string{
	"1000base-t",   // Gigabit Ethernet
	"10gbase-t",    // 10 Gigabit Ethernet
	"ieee802.11n",  // Wi-Fi 4
	"ieee802.11ac", // Wi-Fi 5
	"ieee802.11ax", // Wi-Fi 6
	"ieee802.11be", // Wi-Fi 7
	"virtual",      // Virtual interface
	"lag",          // Link aggregation
	"other",        // Other
}

// ValidateInterfaceType checks if an interface type is valid for NetBox.
// Returns nil if valid, or an InterfaceTypeError if invalid.
func ValidateInterfaceType(ifaceType string) error {
	if _, ok := ValidInterfaceTypes[ifaceType]; ok {
		return nil
	}
	return &InterfaceTypeError{
		InvalidType: ifaceType,
		ValidTypes:  GetCommonInterfaceTypes(),
		Suggestion:  suggestInterfaceType(ifaceType),
	}
}

// GetCommonInterfaceTypes returns a list of common interface types for display
func GetCommonInterfaceTypes() []string {
	return CommonInterfaceTypes
}

// GetAllInterfaceTypes returns all valid interface types sorted alphabetically
func GetAllInterfaceTypes() []string {
	types := make([]string, 0, len(ValidInterfaceTypes))
	for t := range ValidInterfaceTypes {
		types = append(types, t)
	}
	sort.Strings(types)
	return types
}

// suggestInterfaceType suggests a valid interface type based on the invalid input
func suggestInterfaceType(invalidType string) string {
	// Common mistakes and their corrections
	suggestions := map[string]string{
		"wifi":     "ieee802.11ax",
		"wifi4":    "ieee802.11n",
		"wifi5":    "ieee802.11ac",
		"wifi6":    "ieee802.11ax",
		"wifi6e":   "ieee802.11ax",
		"wifi7":    "ieee802.11be",
		"ethernet": "1000base-t",
		"gige":     "1000base-t",
		"gigabit":  "1000base-t",
		"10gig":    "10gbase-t",
		"10g":      "10gbase-t",
		"wireless": "ieee802.11ax",
		"802.11":   "ieee802.11ax",
		"802.11a":  "ieee802.11a",
		"802.11b":  "ieee802.11g",
		"802.11g":  "ieee802.11g",
		"802.11n":  "ieee802.11n",
		"802.11ac": "ieee802.11ac",
		"802.11ax": "ieee802.11ax",
		"802.11be": "ieee802.11be",
	}

	if suggestion, ok := suggestions[invalidType]; ok {
		return suggestion
	}
	return ""
}

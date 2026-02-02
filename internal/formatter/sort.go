package formatter

import (
	"fmt"
	"sort"

	"github.com/maruel/natural"
)

// SortTableData sorts a slice of GenericTableData using natural sorting.
// Sort priority:
// 1. site_name - devices with sites come before those without
// 2. name - devices with names come before those without
// 3. type - device type (ap, switch, gateway)
// 4. mac - MAC address as final tiebreaker
func SortTableData(data []GenericTableData) {
	sort.SliceStable(data, func(i, j int) bool {
		// First priority: Sort by site_name
		siteI := getStringField(data[i], "site_name")
		siteJ := getStringField(data[j], "site_name")

		if siteI != "" && siteJ != "" {
			if siteI != siteJ {
				return natural.Less(siteI, siteJ)
			}
		} else if siteI != "" && siteJ == "" {
			return true // i has site, j doesn't - i comes first
		} else if siteI == "" && siteJ != "" {
			return false // j has site, i doesn't - j comes first
		}

		// Second priority: Sort by name
		nameI := getStringField(data[i], "name")
		nameJ := getStringField(data[j], "name")

		if nameI != "" && nameJ != "" {
			if nameI != nameJ {
				return natural.Less(nameI, nameJ)
			}
		} else if nameI != "" && nameJ == "" {
			return true // i has name, j doesn't - i comes first
		} else if nameI == "" && nameJ != "" {
			return false // j has name, i doesn't - j comes first
		}

		// Third priority: Sort by type
		typeI := getStringField(data[i], "type")
		typeJ := getStringField(data[j], "type")

		if typeI != "" && typeJ != "" {
			if typeI != typeJ {
				return natural.Less(typeI, typeJ)
			}
		} else if typeI != "" && typeJ == "" {
			return true
		} else if typeI == "" && typeJ != "" {
			return false
		}

		// Fourth priority: Sort by MAC address as final tiebreaker
		macI := getStringField(data[i], "mac")
		macJ := getStringField(data[j], "mac")

		if macI != "" && macJ != "" {
			return natural.Less(macI, macJ)
		}

		if macI != "" {
			return true
		}

		return false
	})
}

// getStringField safely extracts a string field from GenericTableData
func getStringField(data GenericTableData, field string) string {
	if val, ok := data[field]; ok {
		return fmt.Sprintf("%v", val)
	}
	return ""
}

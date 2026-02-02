package api

import (
	"sort"

	"github.com/maruel/natural"
)

// SortSites sorts a slice of sites using natural sorting by name
func SortSites(sites []Site) []Site {
	result := make([]Site, len(sites))
	copy(result, sites)

	sort.SliceStable(result, func(i, j int) bool {
		nameI := getSafeSiteName(&result[i])
		nameJ := getSafeSiteName(&result[j])

		return natural.Less(nameI, nameJ)
	})

	return result
}

// getSafeSiteName safely extracts the site name, handling nil pointers
func getSafeSiteName(site *Site) string {
	if site == nil || site.Name == nil {
		return ""
	}
	return *site.Name
}

// SortSitesNew sorts a slice of new sites using natural sorting by name
func SortSitesNew(sites []*MistSite) []*MistSite {
	result := make([]*MistSite, len(sites))
	copy(result, sites)

	sort.SliceStable(result, func(i, j int) bool {
		nameI := getSafeMistSiteName(result[i])
		nameJ := getSafeMistSiteName(result[j])

		return natural.Less(nameI, nameJ)
	})

	return result
}

// getSafeMistSiteName safely extracts the site name from MistSite, handling nil pointers
func getSafeMistSiteName(site *MistSite) string {
	if site == nil {
		return ""
	}
	return site.GetName()
}

// SortAPs sorts a slice of APs using natural sorting by name
func SortAPs(aps []AP) []AP {
	result := make([]AP, len(aps))
	copy(result, aps)

	sort.SliceStable(result, func(i, j int) bool {
		nameI := getSafeAPName(&result[i])
		nameJ := getSafeAPName(&result[j])

		return natural.Less(nameI, nameJ)
	})

	return result
}

// getSafeAPName safely extracts the AP name, handling nil pointers
func getSafeAPName(ap *AP) string {
	if ap == nil || ap.Name == nil {
		return ""
	}
	return *ap.Name
}

// SortInventory sorts inventory items using natural sorting with priority: name, hostname, type, MAC
func SortInventory(items []InventoryItem) []InventoryItem {
	result := make([]InventoryItem, len(items))
	copy(result, items)

	sort.SliceStable(result, func(i, j int) bool {
		// First priority: Sort by name if both have names
		if result[i].Name != nil && result[j].Name != nil {
			return natural.Less(*result[i].Name, *result[j].Name)
		}

		// If only one has a name, it comes first
		if result[i].Name != nil {
			return true
		}
		if result[j].Name != nil {
			return false
		}

		// Second priority: Sort by hostname if both have hostnames
		if result[i].Hostname != nil && result[j].Hostname != nil {
			return natural.Less(*result[i].Hostname, *result[j].Hostname)
		}

		// If only one has a hostname, it comes first
		if result[i].Hostname != nil {
			return true
		}
		if result[j].Hostname != nil {
			return false
		}

		// Third priority: Sort by type
		if result[i].Type != nil && result[j].Type != nil {
			return natural.Less(*result[i].Type, *result[j].Type)
		}

		// If only one has a type, it comes first
		if result[i].Type != nil {
			return true
		}
		if result[j].Type != nil {
			return false
		}

		// If all else fails, sort by MAC address
		if result[i].Mac != nil && result[j].Mac != nil {
			return natural.Less(*result[i].Mac, *result[j].Mac)
		}

		// If only one has a MAC, it comes first
		if result[i].Mac != nil {
			return true
		}

		return false
	})

	return result
}

package cmdutils

import (
	"github.com/ravinald/wifimgr/internal/formatter"
)

// ApplyFieldResolution applies field resolution to table data in-place
// This should be called once during data preparation, not in individual formatters
func ApplyFieldResolution(data []formatter.GenericTableData, enableResolve bool) error {
	if !enableResolve {
		return nil
	}

	// Get cache accessor for field resolution
	apiCacheAccessor, err := GetCacheAccessor()
	if err != nil {
		return err
	}

	// Create field resolver
	resolver := formatter.NewCacheFieldResolver(apiCacheAccessor)

	// Apply resolution to all data
	for i := range data {
		for fieldName, value := range data[i] {
			if resolver.IsResolvableField(fieldName) {
				if resolvedValue, err := resolver.ResolveField(fieldName, value); err == nil {
					data[i][fieldName] = resolvedValue
				}
			}
		}
	}

	return nil
}

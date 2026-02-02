package formatter

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

// TableColumn represents a column in a table with display options.
type TableColumn struct {
	Field             string // JSON field name
	Title             string // display header
	Header            string // alias for Title (backward compat)
	MaxWidth          int    // max width for truncation (0 = unlimited)
	IsHidden          bool
	IsBoolField       bool // use special bool formatting
	IsConnectionField bool // use C/D/? symbols for connection status
	IsStatusField     bool // format as online/offline/alerting/dormant
}

// TableConfig represents the configuration for table display.
type TableConfig struct {
	Format        string // "table", "csv", or "json"
	Columns       []TableColumn
	Title         string
	BoldHeaders   bool
	ShowSeparator bool   // separator line below headers
	CommandPath   string // e.g., "api.ap"
	ExtraData     map[string]interface{}
	SiteLookup    SiteNameLookup // optional site name resolver
	CacheAccess   CacheAccessor  // NOT YET IMPLEMENTED
	ShowAllFields bool           // show all cache fields vs configured
	FieldResolver FieldResolver  // optional ID to name resolver
}

// GenericTableData represents a generic data item for the table
type GenericTableData map[string]interface{}

// SiteNameLookup interface for looking up site names from site IDs
type SiteNameLookup interface {
	GetSiteName(siteID string) (string, bool)
}

// CacheAccessor interface for accessing cached data by index
type CacheAccessor interface {
	// GetCachedData retrieves cached data by index (MAC address, site ID, etc.)
	GetCachedData(index string) (map[string]interface{}, bool)
	// GetSiteName provides site name lookup capability
	GetSiteName(siteID string) (string, bool)
	// GetFieldByPath retrieves a nested field value by dot-separated path
	GetFieldByPath(data map[string]interface{}, path string) (interface{}, bool)
	// ResolveID resolves an ID to a human-readable name based on field type
	ResolveID(fieldName string, id string) (string, bool)
}

// CacheAdapter adapts any cache that has the standard cache methods
type CacheAdapter struct {
	getRawDeviceDataByMAC func(mac string) (map[string]interface{}, bool)
	getSiteName           func(siteID string) (string, bool)
}

// GetRawDeviceDataByMAC provides access to complete cached device data
func (a *CacheAdapter) GetRawDeviceDataByMAC(mac string) (map[string]interface{}, bool) {
	if a.getRawDeviceDataByMAC != nil {
		return a.getRawDeviceDataByMAC(mac)
	}
	return nil, false
}

// GetCachedData retrieves cached data by index (MAC address)
func (a *CacheAdapter) GetCachedData(index string) (map[string]interface{}, bool) {
	return a.GetRawDeviceDataByMAC(index)
}

// GetSiteName implements DeviceDetailGetter interface
func (a *CacheAdapter) GetSiteName(siteID string) (string, bool) {
	if a.getSiteName != nil {
		return a.getSiteName(siteID)
	}
	return "", false
}

// GetFieldByPath retrieves a nested field value by dot-separated path
func (a *CacheAdapter) GetFieldByPath(data map[string]interface{}, path string) (interface{}, bool) {
	return getNestedValue(data, path)
}

// ResolveID is a no-op for CacheAdapter (resolution handled by CacheTableAccessor)
func (a *CacheAdapter) ResolveID(fieldName string, id string) (string, bool) {
	return "", false
}

// ClientCacheAccessor wraps a cache to provide cache access for table formatting
type ClientCacheAccessor struct {
	cacheAdapter *CacheAdapter // Direct access to cache adapter for raw data
}

// GetCachedData retrieves cached device data by MAC address (index)
func (c *ClientCacheAccessor) GetCachedData(index string) (map[string]interface{}, bool) {
	// Get raw device data directly from cache - this is our single source of truth
	if c.cacheAdapter != nil {
		return c.cacheAdapter.GetRawDeviceDataByMAC(index)
	}

	// This should never happen in normal operation since cacheAdapter is always set
	return nil, false
}

// GetSiteName provides site name lookup capability
func (c *ClientCacheAccessor) GetSiteName(siteID string) (string, bool) {
	if c.cacheAdapter == nil {
		return "", false
	}
	return c.cacheAdapter.GetSiteName(siteID)
}

// GetFieldByPath retrieves a nested field value by dot-separated path
func (c *ClientCacheAccessor) GetFieldByPath(data map[string]interface{}, path string) (interface{}, bool) {
	return getNestedValue(data, path)
}

// ResolveID is a no-op for ClientCacheAccessor (resolution handled by CacheTableAccessor)
func (c *ClientCacheAccessor) ResolveID(fieldName string, id string) (string, bool) {
	return "", false
}

// GenericTablePrinter handles formatting and printing tables based on configuration
type GenericTablePrinter struct {
	Config TableConfig
	Data   []GenericTableData
}

// NewGenericTablePrinter creates a new table printer
func NewGenericTablePrinter(config TableConfig, data []GenericTableData) *GenericTablePrinter {
	// Ensure backward compatibility - copy Header to Title if Title is empty
	for i := range config.Columns {
		if config.Columns[i].Title == "" && config.Columns[i].Header != "" {
			config.Columns[i].Title = config.Columns[i].Header
		} else if config.Columns[i].Header == "" && config.Columns[i].Title != "" {
			config.Columns[i].Header = config.Columns[i].Title
		}
	}

	return &GenericTablePrinter{
		Config: config,
		Data:   data,
	}
}

// LoadColumnsFromConfig loads column definitions from a configuration array
// The config format must be an array of objects: [{"field": "name", "title": "Name", "width": 32}, ...]
func (p *GenericTablePrinter) LoadColumnsFromConfig(configArray interface{}) {
	// Handle array of column configuration objects
	var fieldsArray []interface{}

	switch configVal := configArray.(type) {
	case []interface{}:
		fieldsArray = configVal
	default:
		// Unknown type, return without adding columns
		return
	}

	// Clear existing columns
	p.Config.Columns = []TableColumn{}

	// Process each column configuration object in the array
	for _, configItem := range fieldsArray {
		configObj, ok := configItem.(map[string]interface{})
		if !ok {
			// Skip non-object items
			continue
		}

		// Extract field name
		fieldName, ok := configObj["field"].(string)
		if !ok || fieldName == "" {
			// Skip if field name is missing or invalid
			continue
		}

		// Extract title
		titleStr, ok := configObj["title"].(string)
		if !ok || titleStr == "" {
			// Skip if title is missing or invalid
			continue
		}

		// Extract width (default to 0 for dynamic width)
		maxWidth := 0
		if widthVal, exists := configObj["width"]; exists {
			switch w := widthVal.(type) {
			case int:
				maxWidth = w
			case float64:
				maxWidth = int(w)
			case string:
				if parsedWidth, err := strconv.Atoi(w); err == nil {
					maxWidth = parsedWidth
				}
			}
		}

		// Detect boolean fields
		isBoolField := strings.HasSuffix(strings.ToLower(fieldName), "enabled") ||
			strings.HasSuffix(strings.ToLower(fieldName), "connected") ||
			strings.ToLower(fieldName) == "connected" ||
			strings.ToLower(fieldName) == "enabled"

		// Detect connection fields (should use C/D/? symbols)
		isConnectionField := isBoolField && strings.Contains(strings.ToLower(fieldName), "connect")

		// Detect status fields (online/offline/alerting/dormant)
		isStatusField := strings.ToLower(fieldName) == "status"

		// Create the column definition
		column := TableColumn{
			Field:             fieldName,
			Title:             titleStr,
			Header:            titleStr, // Set Header to the same value as Title for backward compatibility
			MaxWidth:          maxWidth,
			IsHidden:          false,
			IsBoolField:       isBoolField,
			IsConnectionField: isConnectionField,
			IsStatusField:     isStatusField,
		}

		// Add to columns list (maintains order from configuration array)
		p.Config.Columns = append(p.Config.Columns, column)
	}
}

// Print formats and returns the table based on the configuration
func (p *GenericTablePrinter) Print() string {
	// Skip if no data
	if len(p.Data) == 0 {
		return "No data to display"
	}

	// If no columns are configured, generate default columns from the data structure
	if len(p.Config.Columns) == 0 {
		// We need to reconstruct the original objects to generate default columns
		// For now, let's generate columns from the first data item
		if len(p.Data) > 0 {
			defaultColumns := p.generateDefaultColumnsFromData()
			p.Config.Columns = defaultColumns
		}

		// If we still have no columns after generation, show message
		if len(p.Config.Columns) == 0 {
			return "No data to display"
		}
	}

	// Select the appropriate format method
	switch strings.ToLower(p.Config.Format) {
	case "csv":
		return p.formatAsCSV()
	case "json":
		return p.formatAsJSON()
	case "table", "":
		// Use BubbleTea table for rendering (terminal dimensions handled automatically)
		bubbleTable := NewBubbleTable(p.Config, p.Data, false)
		return bubbleTable.RenderStatic()
	default:
		// Default to table format
		bubbleTable := NewBubbleTable(p.Config, p.Data, false)
		return bubbleTable.RenderStatic()
	}
}

// formatAsCSV formats the data as CSV and returns the string
func (p *GenericTablePrinter) formatAsCSV() string {
	var buf strings.Builder
	writer := csv.NewWriter(&buf)

	// Calculate visible columns (skip hidden ones)
	visibleColumns := make([]TableColumn, 0)
	for _, col := range p.Config.Columns {
		if !col.IsHidden {
			visibleColumns = append(visibleColumns, col)
		}
	}

	// If no visible columns, exit
	if len(visibleColumns) == 0 {
		return "No columns configured for display"
	}

	// Write title as a comment if provided
	if p.Config.Title != "" {
		buf.WriteString(fmt.Sprintf("# %s\n", p.Config.Title))
	}

	// Prepare headers
	headers := make([]string, len(visibleColumns))
	for i, col := range visibleColumns {
		// Get title text, fall back to Header if Title is empty
		titleText := col.Title
		if titleText == "" {
			titleText = col.Header
		}
		headers[i] = titleText
	}

	// Write headers
	if err := writer.Write(headers); err != nil {
		return fmt.Sprintf("Error writing CSV headers: %v\n", err)
	}

	// Write data rows
	for _, item := range p.Data {
		row := make([]string, len(visibleColumns))
		for i, col := range visibleColumns {
			var val interface{}
			var ok bool

			// Check for cache.* field path (e.g., "cache.radio_config.band_5.channel")
			if strings.HasPrefix(col.Field, "cache.") && p.Config.CacheAccess != nil {
				// Get MAC address from item to look up cache data
				if mac, hasMac := item["mac"].(string); hasMac && mac != "" {
					if cachedData, found := p.Config.CacheAccess.GetCachedData(mac); found {
						// Extract the path after "cache." prefix
						cachePath := strings.TrimPrefix(col.Field, "cache.")
						val, ok = p.Config.CacheAccess.GetFieldByPath(cachedData, cachePath)
					}
				}
			} else {
				// Direct field access
				val, ok = item[col.Field]
			}

			if !ok {
				row[i] = ""
				continue
			}

			// Field resolution now done during data preparation

			// Format boolean values consistently with table formatting
			if col.IsBoolField {
				if bVal, ok := val.(bool); ok {
					if col.IsConnectionField {
						if bVal {
							row[i] = "Connected"
						} else {
							row[i] = "Disconnected"
						}
					} else {
						if bVal {
							row[i] = "Yes"
						} else {
							row[i] = "No"
						}
					}
				} else {
					row[i] = "Unknown"
				}
			} else if strings.HasPrefix(col.Field, "cache.") {
				// Use formatNestedValue for cache.* fields to handle complex nested values
				row[i] = formatNestedValue(val)
			} else {
				row[i] = fmt.Sprintf("%v", val)
			}
		}

		if err := writer.Write(row); err != nil {
			return fmt.Sprintf("Error writing CSV row: %v\n", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Sprintf("Error flushing CSV writer: %v\n", err)
	}

	return buf.String()
}

// formatAsJSON formats the data as JSON and returns the string
func (p *GenericTablePrinter) formatAsJSON() string {
	// If ShowAllFields is true and we have cache access, use raw cache data
	if p.Config.ShowAllFields && p.Config.CacheAccess != nil {
		return p.formatAsJSONWithAllFields()
	}

	// For single item, return the raw object
	if len(p.Data) == 1 {
		jsonData, err := MarshalJSONWithColorIndent(p.Data[0], "", "  ")
		if err != nil {
			return fmt.Sprintf("Error marshalling JSON: %v\n", err)
		}
		return string(jsonData) + "\n"
	}

	// For multiple items, return an array
	jsonData, err := MarshalJSONWithColorIndent(p.Data, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error marshalling JSON: %v\n", err)
	}
	return string(jsonData) + "\n"
}

// formatAsJSONWithAllFields formats the data with all available fields from cache
func (p *GenericTablePrinter) formatAsJSONWithAllFields() string {
	// If we have a single item, get raw cache data by MAC
	if len(p.Data) == 1 {
		// Try to get MAC from the data
		if mac, exists := p.Data[0]["mac"].(string); exists && mac != "" {
			if rawData, found := p.Config.CacheAccess.GetCachedData(mac); found {
				jsonData, err := MarshalJSONWithColorIndent(rawData, "", "  ")
				if err != nil {
					return fmt.Sprintf("Error marshalling all fields JSON: %v\n", err)
				}
				return string(jsonData) + "\n"
			}
		}
		// Fallback to configured fields if cache data not found
		jsonData, err := MarshalJSONWithColorIndent(p.Data[0], "", "  ")
		if err != nil {
			return fmt.Sprintf("Error marshalling JSON: %v\n", err)
		}
		return string(jsonData) + "\n"
	}

	// For multiple items, collect raw cache data for each
	var allFieldsData []map[string]interface{}
	for _, item := range p.Data {
		if mac, exists := item["mac"].(string); exists && mac != "" {
			if rawData, found := p.Config.CacheAccess.GetCachedData(mac); found {
				allFieldsData = append(allFieldsData, rawData)
			} else {
				// Fallback to configured fields if cache data not found
				allFieldsData = append(allFieldsData, item)
			}
		} else {
			// Fallback to configured fields if no MAC
			allFieldsData = append(allFieldsData, item)
		}
	}

	jsonData, err := MarshalJSONWithColorIndent(allFieldsData, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error marshalling all fields JSON: %v\n", err)
	}
	return string(jsonData) + "\n"
}

// PrintToOutput formats the table and writes it to the provided writer
func (p *GenericTablePrinter) PrintToOutput(w io.Writer) error {
	_, err := fmt.Fprint(w, p.Print())
	return err
}

// AddData adds a data item to the printer
func (p *GenericTablePrinter) AddData(data GenericTableData) {
	p.Data = append(p.Data, data)
}

// generateDefaultColumnsFromData generates default columns from the data map structure
func (p *GenericTablePrinter) generateDefaultColumnsFromData() []TableColumn {
	var columns []TableColumn

	// If no data, return empty
	if len(p.Data) == 0 {
		return columns
	}

	// Collect all unique field names from all items
	fieldSet := make(map[string]bool)

	for _, item := range p.Data {
		for fieldName := range item {
			if !fieldSet[fieldName] {
				fieldSet[fieldName] = true
			}
		}
	}

	// Sort field names for consistent ordering
	// Common fields first, then alphabetical
	commonFields := []string{"name", "id", "site", "type", "status", "mac", "serial", "model"}
	var sortedFields []string

	// Add common fields that exist in our data
	for _, field := range commonFields {
		if fieldSet[field] {
			sortedFields = append(sortedFields, field)
			delete(fieldSet, field)
		}
	}

	// Add remaining fields in alphabetical order
	var remainingFields []string
	for field := range fieldSet {
		remainingFields = append(remainingFields, field)
	}
	sort.Strings(remainingFields)
	sortedFields = append(sortedFields, remainingFields...)

	// Create columns for each field
	for _, fieldName := range sortedFields {
		// Use the field name as both field and title
		title := fieldName

		// Determine if this is a boolean field for special formatting
		isBoolField := strings.HasSuffix(strings.ToLower(fieldName), "enabled") ||
			strings.HasSuffix(strings.ToLower(fieldName), "connected") ||
			strings.ToLower(fieldName) == "connected" ||
			strings.ToLower(fieldName) == "enabled"

		// Check if the actual value is a boolean by looking at all items
		for _, item := range p.Data {
			if val, exists := item[fieldName]; exists {
				if _, ok := val.(bool); ok {
					isBoolField = true
					break
				}
			}
		}

		// Detect connection fields (should use C/D/? symbols)
		isConnectionField := isBoolField && strings.Contains(strings.ToLower(fieldName), "connect")

		// Detect status fields (online/offline/alerting/dormant)
		isStatusField := strings.ToLower(fieldName) == "status"

		// Create the column definition
		column := TableColumn{
			Field:             fieldName,
			Title:             title,
			Header:            title, // Set Header for backward compatibility
			MaxWidth:          0,     // No width limit by default
			IsHidden:          false,
			IsBoolField:       isBoolField,
			IsConnectionField: isConnectionField,
			IsStatusField:     isStatusField,
		}

		columns = append(columns, column)
	}

	return columns
}

// getNestedValue retrieves a value from a nested map using dot-separated path.
// Supports paths like "radio_config.band_5.channel" or "ip_config.vlan_id".
func getNestedValue(data map[string]interface{}, path string) (interface{}, bool) {
	if data == nil || path == "" {
		return nil, false
	}

	parts := strings.Split(path, ".")
	current := interface{}(data)

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			val, ok := v[part]
			if !ok {
				return nil, false
			}
			current = val
		default:
			return nil, false
		}
	}

	return current, true
}

// formatNestedValue formats a nested value for display.
// Handles special cases like arrays, maps, and nil values.
func formatNestedValue(value interface{}) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case float64:
		// Check if it's a whole number
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%.2f", v)
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case []interface{}:
		// Format arrays as comma-separated values
		parts := make([]string, len(v))
		for i, item := range v {
			parts[i] = formatNestedValue(item)
		}
		return strings.Join(parts, ", ")
	case map[string]interface{}:
		// For nested objects, return a summary
		return fmt.Sprintf("{%d fields}", len(v))
	default:
		return fmt.Sprintf("%v", v)
	}
}

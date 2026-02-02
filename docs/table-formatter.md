# Table Formatter System

The application uses BubbleTea tables for all table rendering, providing both interactive and static display modes. The table system supports dynamic column configuration from user config files while offering modern terminal UI capabilities.

## Current Implementation

**Core Components:**
- **BubbleTableModel**: Main table implementation using BubbleTea's table component (in `internal/formatter/bubbletea_table.go`)
- **GenericTablePrinter**: Handles dynamic column configuration and multiple output formats (`internal/formatter/generic_table.go`)
- **SimpleColumn/SimpleTableOptions**: Simplified API for basic table rendering
- **RenderSimpleTable**: Convenience function that internally uses BubbleTea tables (`internal/formatter/fixed_table.go`)

**Active Usage**: The formatters are actively used in:
- `cmd/ap/ap.go`
- `cmd/search/search.go`
- `cmd/inventory/inventory.go`
- `cmd/root/root.go`
- `cmd/site/site.go`

## Table Features

- Dynamic column configuration from JSON/config files
- Interactive navigation in shell mode (arrow keys, selection)
- Static rendering for CLI output
- Multiple output formats: Table, CSV, and JSON
- Boolean field formatting with modern symbols (⏺/Y/N/?)
- Column width management and truncation
- Row markers for special items
- Consistent styling across all tables

## Usage Example

```go
// Convert data to map for the table formatter
tableData := make([]map[string]interface{}, 0, len(items))
for _, item := range items {
    tableData = append(tableData, map[string]interface{}{
        "name":        itemName,
        "id":          itemID,
        "hasMarker":   needsMarker, // boolean flag for marking rows
    })
}

// Define columns for the table
columns := []formatter.SimpleColumn{
    {Header: "name", Field: "name"},
    {Header: "id", Field: "id"},
}

// Configure table options
options := formatter.SimpleTableOptions{
    Title:         "Items:",
    MarkerSymbol:  "✓",        // Symbol to use as marker (e.g., checkmark)
    MarkerField:   "hasMarker", // Boolean field that determines if row should be marked
    BoldHeaders:   true,        // Bold headers for better readability
    ShowSeparator: true,        // Show separator between headers and data
}

// Render the table using the simple formatter
tableOutput := formatter.RenderSimpleTable(tableData, columns, options)
fmt.Print(tableOutput)
```

## Column Configuration System

The table formatter uses a **3-tier hierarchy** for defining which columns to display:

### Column Definition Hierarchy

1. **Global Configuration (Primary)** - Defined in `wifimgr-config.json` under `display.commands.<command-path>`
2. **Command-Level Defaults (Fallback)** - Hardcoded defaults in each command file
3. **Auto-Generation (Last Resort)** - Dynamically generated from data structure

### How Column Loading Works

When a command displays table data, it follows this process:

1. **Configuration Check**: Commands call `LoadColumnsFromConfig()` which uses Viper to check for column definitions at the command path (e.g., `display.commands.show.api.ap`)

2. **Configuration Found**: If configuration exists, columns are loaded from the config file:
   ```go
   // In command file
   printer := formatter.GenericTablePrinter{
       Config: formatter.TableConfig{
           Format:      formatType,
           CommandPath: "show.api.ap",  // Path to look up in config
       },
   }
   printer.LoadColumnsFromConfig()  // Loads from wifimgr-config.json
   ```

3. **No Configuration**: If no configuration exists, the command falls back to hardcoded defaults:
   ```go
   // Command defines fallback columns
   if len(printer.Config.Columns) == 0 {
       printer.Config.Columns = []formatter.TableColumn{
           {Field: "name", Title: "Name", MaxWidth: 0},
           {Field: "mac", Title: "MAC", MaxWidth: 0},
           {Field: "model", Title: "Model", MaxWidth: 0},
       }
   }
   ```

4. **Auto-Generation**: If no columns are defined at all, `generateDefaultColumnsFromData()` creates columns from the first data item's fields

### Configuration Examples

**Global Configuration in wifimgr-config.json:**
```json
{
  "display": {
    "commands": {
      "show.api.ap": {
        "format": "table",
        "title": "Access Points:",
        "fields": [
          { "field": "connected", "title": "Conn", "width": 3 },
          { "field": "name", "title": "Name", "width": 32 },
          { "field": "mac", "title": "MAC", "width": 20 },
          { "field": "model", "title": "Model", "width": 6 }
        ]
      }
    }
  }
}
```

**Command-Level Default (in code):**
```go
// From cmd/ap/ap.go
printer.Config.Columns = []formatter.TableColumn{
    {Field: "connected", Title: "Conn", MaxWidth: 9, IsBoolField: true, IsConnectionField: true},
    {Field: "site_id", Title: "Site", MaxWidth: 0},
    {Field: "name", Title: "Name", MaxWidth: 0},
    {Field: "type", Title: "Type", MaxWidth: 8},
    {Field: "mac", Title: "MAC", MaxWidth: 18},
    {Field: "serial", Title: "Serial", MaxWidth: 16},
    {Field: "model", Title: "Model", MaxWidth: 12},
}
```

### Cache.* Field Paths (Advanced)

Columns can reference **nested data from the cache** using special `cache.*` field paths. This allows displaying any field from the cached API response, including deeply nested structures like radio configuration.

**Syntax:** `cache.<path.to.nested.field>`

**Example Configuration:**
```json
{
  "display": {
    "commands": {
      "show.api.ap": {
        "format": "table",
        "title": "Access Points with Radio Info:",
        "fields": [
          { "field": "status", "title": "Status", "width": 6 },
          { "field": "name", "title": "Name", "width": -1 },
          { "field": "mac", "title": "MAC", "width": 17 },
          { "field": "cache.radio_config.band_5.channel", "title": "5G Ch", "width": 6 },
          { "field": "cache.radio_config.band_5.power", "title": "5G Pwr", "width": 7 },
          { "field": "cache.radio_config.band_24.channel", "title": "2.4G Ch", "width": 7 },
          { "field": "cache.ip_config.vlan_id", "title": "VLAN", "width": 6 },
          { "field": "cache.deviceprofile_name", "title": "Profile", "width": -1 }
        ]
      }
    }
  }
}
```

**Available Cache Paths:**

| Path                                 | Description                         |
|--------------------------------------|-------------------------------------|
| `cache.radio_config.band_5.channel`  | 5GHz radio channel                  |
| `cache.radio_config.band_5.power`    | 5GHz transmit power                 |
| `cache.radio_config.band_24.channel` | 2.4GHz radio channel                |
| `cache.radio_config.band_24.power`   | 2.4GHz transmit power               |
| `cache.radio_config.band_6.channel`  | 6GHz radio channel (if supported)   |
| `cache.ip_config.type`               | IP configuration type (dhcp/static) |
| `cache.ip_config.vlan_id`            | VLAN ID                             |
| `cache.ip_config.ip`                 | Static IP address                   |
| `cache.ble_config.power`             | BLE transmit power                  |
| `cache.deviceprofile_id`             | Device profile UUID                 |
| `cache.deviceprofile_name`           | Device profile name (resolved)      |
| `cache.site_name`                    | Site name (resolved from site_id)   |

**How It Works:**
1. The formatter detects columns with `cache.` prefix
2. For each row, it looks up the device's full cached data by MAC address
3. It navigates the dot-separated path to extract the nested value
4. Complex values (arrays, maps) are formatted appropriately

**Value Formatting:**
- Numbers: Displayed as integers when whole, otherwise 2 decimal places
- Booleans: Displayed as "true" or "false"
- Arrays: Comma-separated values
- Maps: Displayed as `{N fields}` summary
- Null/missing: Empty string

### ShowAllFields (`all` Argument)

When using JSON format, you can add the `all` argument to display **all cached fields** instead of just the configured columns:

```bash
# Show only configured columns
wifimgr show api ap AP-NAME json

# Show ALL fields from cache
wifimgr show api ap AP-NAME json all
```

This is useful for:
- Discovering available fields for `cache.*` column configuration
- Debugging to see the complete cached data
- Exporting full device configuration

**Example Output with `all`:**
```json
{
  "name": "AP-Lobby-01",
  "mac": "aabbccddeeff",
  "site_id": "abc123",
  "site_name": "US-LAB-01",
  "deviceprofile_name": "Default-AP-Profile",
  "radio_config": {
    "band_5": {
      "channel": 36,
      "power": 17,
      "bandwidth": 80
    },
    "band_24": {
      "channel": 6,
      "power": 15
    }
  },
  "ip_config": {
    "type": "dhcp",
    "vlan_id": 100
  },
  "ble_config": {
    "power": 8
  }
}
```

### LoadColumnsFromConfig Implementation

The `LoadColumnsFromConfig` method is responsible for parsing column configurations from Viper:

```go
func (p *GenericTablePrinter) LoadColumnsFromConfig(configArray interface{}) {
    // Expects array of objects: [{"field": "name", "title": "Name", "width": 32}, ...]
    // Processes each column object and creates TableColumn structs
    // Automatically detects boolean and connection fields
    // Sets appropriate formatting flags (IsBoolField, IsConnectionField)
}
```

Key features:
- Only accepts object format (no legacy array format)
- Validates required fields (`field` and `title`)
- Defaults `width` to 0 if not specified
- Automatically detects boolean fields by name patterns
- Sets connection field flags for proper symbol display

### Auto-Generation Fallback

When no columns are configured, `generateDefaultColumnsFromData()` creates columns dynamically by scanning all data items to discover all available fields:

```go
func (p *GenericTablePrinter) generateDefaultColumnsFromData() []TableColumn {
    // Scans ALL data items to find unique field names
    // Intelligently orders columns with common fields first
    // Sorts remaining fields alphabetically
    // Uses raw JSON field name as column title
    // Auto-detects boolean/connection fields by checking actual values
    // Sets width to 0 (auto-size with scaling)
}
```

**Key Features:**
- **Complete Field Discovery**: Scans all items in the dataset, not just the first one, ensuring no fields are missed
- **Intelligent Column Ordering**: Common fields (name, id, site, type, status, mac, serial, model) appear first for consistency
- **Alphabetical Sorting**: Remaining fields are sorted alphabetically for predictable output
- **Raw Field Names**: Uses the actual JSON field names as column titles, making it easy to understand the data structure
- **Boolean Detection**: Examines actual values across all items to accurately detect boolean fields
- **Zero Configuration**: Works automatically without any configuration, making the system more robust

**Example**: If your data contains varying fields across items:
```json
[
  {"name": "AP1", "mac": "00:11:22:33:44:55", "status": "online"},
  {"name": "AP2", "mac": "00:11:22:33:44:56", "model": "AP43", "firmware": "1.2.3"}
]
```

The auto-generated columns would be:
1. `name` (common field, shown first)
2. `mac` (common field, shown second) 
3. `status` (common field, shown third)
4. `model` (common field, shown fourth)
5. `firmware` (alphabetically sorted, shown last)

This ensures tables always display all available data even without configuration, making the system more robust and developer-friendly.

### Field Resolution

The system supports automatic ID-to-name resolution for fields like `site_id`:
- When a `FieldResolver` is provided, fields matching patterns like `site_id`, `rf_template_id`, etc. are automatically resolved to their human-readable names
- This happens during data preparation, before the table is rendered

## Output Formats

The table formatter supports three output formats that can be configured via the `format` field:

### **Table Format** (`"format": "table"`)
- **Default format** - Interactive BubbleTea tables with terminal styling
- **Features**: Column alignment, color coding, symbol formatting, interactive navigation
- **Use case**: Primary display format for CLI output

### **CSV Format** (`"format": "csv"`)
- **Structured data export** - Comma-separated values with headers
- **Features**: Clean data export, no styling or symbols, suitable for data processing
- **Use case**: Data export and integration with external tools

### **JSON Format** (`"format": "json"`)
- **Structured data export** - JSON format with proper indentation
- **Features**: 
  - Single item: Returns object directly `{...}`
  - Multiple items: Returns array of objects `[{...}, {...}]`
  - Preserves all data fields and types
  - **Color syntax highlighting** - Configurable colors for different JSON elements
- **Use case**: API-like output, programmatic processing, debugging

#### **JSON Color Configuration**

JSON output supports syntax highlighting through the `display.jsoncolor` configuration:

```json
{
  "display": {
    "jsoncolor": {
      "null":   { "hex": "#767676", "ansi256": "244", "ansi": "8" },
      "bool":   { "hex": "#FFFFFF", "ansi256": "15",  "ansi": "7" },
      "number": { "hex": "#00FFFF", "ansi256": "51",  "ansi": "6" },
      "string": { "hex": "#00FF00", "ansi256": "46",  "ansi": "2" },
      "key":    { "hex": "#0000FF", "ansi256": "21",  "ansi": "4" },
      "bytes":  { "hex": "#767676", "ansi256": "244", "ansi": "8" },
      "time":   { "hex": "#00FF00", "ansi256": "46",  "ansi": "2" }
    }
  }
}
```

**Color Types:**
- `null` - Null values and undefined
- `bool` - Boolean true/false values
- `number` - Numeric values (integers, floats)
- `string` - String values (excluding keys)
- `key` - Object property keys
- `bytes` - Byte array representations
- `time` - Timestamp and date values

**Color Formats:**
- `hex` - Hexadecimal color codes for modern terminals
- `ansi256` - 256-color ANSI codes for enhanced terminals
- `ansi` - Basic 8-color ANSI codes for compatibility

**Accessing Colors in Code:**
```go
// Get colors for a specific type
stringColors := config.GetJsonColorConfig("string")
// Returns: {"hex": "#00FF00", "ansi256": "46", "ansi": "2"}

// Get all color configurations
allColors := config.GetAllJsonColorConfigs()

// Direct Viper access
keyHex := viper.GetString("display.jsoncolor.key.hex") // "#0000FF"
```

### **Configuration Examples**

```json
{
  "display": {
    "commands": {
      "show.api.site": {
        "format": "table",
        "fields": {...}
      },
      "show.api.ap": {
        "format": "json",
        "fields": {...}
      }
    }
  }
}
```

### **Command-Line Usage**

JSON format can be accessed by adding `json` as the last argument:

```bash
# Table format (default)
wifimgr show api site SITE-NAME

# JSON format via positional argument
wifimgr show api site SITE-NAME json

# JSON format shows single object for single match
wifimgr show api site SPECIFIC-SITE json
# Output: {"id": "...", "name": "SPECIFIC-SITE", ...}

# JSON format shows array for multiple matches  
wifimgr show api site json
# Output: [{"id": "...", "name": "Site1"}, {"id": "...", "name": "Site2"}]

# Works with all show commands
wifimgr show api ap AP-NAME json
wifimgr show api ap 00:11:22:33:44:55 json  
wifimgr show inventory ap json
wifimgr show inventory site US-LAB-01 json
```

## Column Width Configuration

The table formatter supports three different width behaviors for columns:

**Width Values:**
- **`width = -1`**: Auto-size to fit the largest cell content, no scaling to terminal width
- **`width = 0`**: Auto-size to fit the largest cell content, then scale to fit terminal width  
- **`width > 0`**: Use exact specified width, no auto-sizing or scaling

**Configuration Examples:**
```json
{
  "fields": {
    "name": ["Name", -1],      // Fit content exactly, never truncate
    "id": ["ID", 0],           // Fit content, scale to terminal width
    "address": ["Address", 50] // Fixed 50 characters width
  }
}
```

**Use Cases:**
- **`width = -1`**: Critical fields that should never be truncated (e.g., device names, IDs)
- **`width = 0`**: Fields that can be scaled for better terminal utilization (e.g., descriptions, addresses)
- **`width > 0`**: Fields with specific formatting requirements (e.g., MAC addresses, timestamps)

## Symbol Integration

The table system automatically integrates with the symbol system:
- Connection boolean fields use colored ⏺/Y/N/? format
- Regular boolean fields use plain Yes/No format
- Terminal detection happens automatically at render time
- No configuration needed - works correctly with pipes and redirects

## Width Calculation with Styled Content

**Critical**: Always use `lipgloss.Width()` when measuring styled content, never `len()`:

```go
// WRONG - counts ANSI escape codes
if len(styledText) > maxWidth {
    // This breaks table alignment
}

// CORRECT - measures display width  
if lipgloss.Width(styledText) > maxWidth {
    // This preserves proper alignment
}
```

All table formatters have been updated to use `lipgloss.Width()` for proper handling of styled symbols.

## Conditional Row Coloring System

The table formatter supports a sophisticated conditional row coloring system that allows entire rows to be styled with different foreground colors while preserving alternating background shading. This system is used extensively for visual indicators like highlighting configured sites or marking special states.

### Architecture Overview

The system works through a **prefix-based marker approach** that integrates seamlessly with the table formatter's existing color detection and background shading logic:

1. **Data Preparation Phase**: Content is marked with special prefixes during data preparation
2. **Width Calculation Phase**: Prefixes are stripped for accurate column width calculation
3. **Rendering Phase**: Prefixes are detected, content is styled, and backgrounds are applied correctly

### Implementation Components

**Core Files:**
- `/cmd/site/site.go` - Data preparation with prefix marking
- `/internal/formatter/bubbletea_table.go` - Prefix detection and styling logic  
- `/internal/symbols/symbols.go` - Color styling functions

### Prefix Marker System

The system uses standardized prefixes that the table formatter recognizes:

```go
// Supported color prefixes
"GREEN_TEXT:content"  // Renders content in green
"CONN_TRUE"          // Connection status: green circle or "C" 
"CONN_FALSE"         // Connection status: red circle or "D"
"CONN_UNKNOWN"       // Connection status: blue circle or "?"
"BOOL_TRUE"          // Boolean value: "Yes"
"BOOL_FALSE"         // Boolean value: "No"
```

### Testing Conditional Coloring

When testing conditional row coloring:

**1. Test in Terminal:**
```bash
./wifimgr -e show api sites  # Should show green rows for configured sites
```

**2. Test with Redirect:**
```bash
./wifimgr -e show api sites > output.txt  # Should show plain text in file
```

**3. Test Column Alignment:**
- Verify all rows have consistent column widths regardless of styling
- Check that styled and unstyled content aligns properly

**4. Test Background Shading:**
- Verify alternating row backgrounds are preserved
- Check that colored foreground doesn't interfere with background shading

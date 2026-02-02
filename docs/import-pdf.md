# PDF Parsing - Import Command

## Overview

The `import` command extracts AP radio configurations from PDF files and updates matching APs in site configuration files. This feature allows network administrators to import planned channel, power, and bandwidth settings from design documents directly into wifimgr site configurations.

## Command Syntax and Usage

### Command Structure
```bash
wifimgr import type [pdf] file <file_path> site <site_name> [prefix <prefix>] [suffix <suffix>]
wifimgr import file <file_path> site <site_name>  # type defaults to "pdf"
```

### Required Arguments
- **`file <file_path>`** - Path to the PDF file containing AP configurations
- **`site <site_name>`** - Name of the site to update (must have a corresponding config file)

### Optional Arguments
- **`prefix <prefix>`** - Prefix to add to AP names when matching against site configuration
- **`suffix <suffix>`** - Suffix to add to AP names when matching against site configuration

**Note**: If your prefix or suffix starts with a dash (e.g., `-TEST`), use `--` before it to prevent it from being interpreted as a flag:
```bash
wifimgr import file peralta.pdf site US-OAK-PER suffix -- -TEST
```

### Optional Arguments
- **`type [pdf]`** - File type (currently only "pdf" is supported, and is the default)

### Usage Examples
```bash
# Full syntax with explicit type
wifimgr import type pdf file /path/to/document.pdf site US-LAB-01

# Simplified syntax (type defaults to pdf)
wifimgr import file /path/to/document.pdf site US-LAB-01

# With authentication flag
wifimgr -e import file peralta.pdf site US-OAK-PER

# With prefix to match APs with different naming
wifimgr import file peralta.pdf site US-OAK-PER prefix PROD-

# With suffix for test environments
wifimgr import file peralta.pdf site US-OAK-PER suffix TEST

# With suffix starting with dash (use -- separator)
wifimgr import file peralta.pdf site US-OAK-PER suffix -- -TEST

# With both prefix and suffix
wifimgr import file peralta.pdf site US-OAK-PER prefix PROD- suffix V2

# With both where suffix has a dash
wifimgr import file peralta.pdf site US-OAK-PER prefix PROD- suffix -- -V2
```

## PDF Content Format

### Expected AP Configuration String Format
The parser searches for AP configuration strings in the PDF that match this pattern:
```
@AP-NAME/band:channel:power:width
```

**Important**: AP configurations must be prefixed with `@` to distinguish them from other text in the PDF.
- The `@` symbol marks the start of an AP configuration
- `AP-NAME` is everything between `@` and the first forward slash
- This format prevents false matches with architectural drawings or other text

### Regex Pattern
The exact regex pattern used for extraction:
```regex
@([\w\-]+)((?:/(?:2:(?:-1|\d{1,3})?:(?:-1|\d{1,2})?:(?:-1|20|40)?|5:(?:-1|\d{1,3})?:(?:-1|\d{1,2})?:(?:-1|20|40|80|160)?|6:(?:-1|\d{1,3})?:(?:-1|\d{1,2})?:(?:-1|20|40|80|160|320)?)){1,3})
```

### Pattern Components

#### AP Name Structure
- **Prefix**: Required `@` symbol to mark the start of an AP configuration
- **AP Name**: Alphanumeric characters, dashes, and underscores between `@` and `/` (e.g., `US-OAK-PER-1-1`, `AP-LOBBY-01`, `2-1`, etc.)

#### Radio Configuration Format
Each band configuration follows the pattern: `/band:channel:power:width`

- **Band Identifier**:
  - `2` = 2.4GHz band
  - `5` = 5GHz band
  - `6` = 6GHz band

- **Channel**:
  - `0` or `-1` = Auto channel selection
  - `1-165` = Specific channel number
  - Empty = Auto

- **Power** (Tx Power in dBm):
  - `-1` = Auto power
  - Numeric value = Specific power level
  - Empty = Auto power

- **Width** (Channel bandwidth in MHz):
  - 2.4GHz: `20`, `40`
  - 5GHz: `20`, `40`, `80`, `160`
  - 6GHz: `20`, `40`, `80`, `160`, `320`
  - `-1` = Use default from configuration
  - Empty = Use default from configuration (see Defaults Configuration)

### Capture Groups
1. **AP name** - Everything between `@` and first `/` (e.g., "US-OAK-PER-1-1", "2-1")
2. **Radio settings** - Band configurations string after the first `/`

### Valid String Examples
```
@US-OAK-PER-1-1/2:1:5:/5:0:9:40/6:0::
@US-OAK-PER-1-2/2:6:5:/5:0:9:40/6:103::
@US-OAK-PER-2-1/2:11:5:/5:0:9:40
@US-OAK-PER-3-1/2:1:5:20/5:36:9:80/6:1:10:160
@2-1/2:1:5:/5:0:9:/6:0::160
@1-1/2:6:5:/5::9:/6:::160
```

### Interpretation Examples

**Example 1**: `@US-OAK-PER-1-1/2:1:5:/5:0:9:40`
- AP Name: US-OAK-PER-1-1
- 2.4GHz: Channel 1, Power 5dBm, Width 20MHz (default)
- 5GHz: Channel Auto, Power 9dBm, Width 40MHz
- 6GHz: Disabled (not specified)

**Example 2**: `@2-1/2:11:5:/5:0:9:`
- AP Name: 2-1
- 2.4GHz: Channel 11, Power 5dBm, Width 20MHz (default)
- 5GHz: Channel Auto, Power 9dBm, Width 20MHz (default)
- 6GHz: Disabled (not specified)

## Site Configuration Integration

### Site Config File Requirements

The site configuration file must:
1. Be located in the config directory (default: `~/.config/wifimgr/`)
2. Be named as `<site-name>.json` (lowercase)
3. Follow the standard wifimgr site configuration structure

### Site Config Structure
```json
{
  "version": 1,
  "sites": {
    "US-OAK-PER": {
      "site_config": {
        "name": "US-OAK-PER",
        "country_code": "US",
        "timezone": "America/Los_Angeles"
      },
      "devices": {
        "ap": {
          "a8f7d982de1a": {
            "mac": "a8f7d982de1a",
            "name": "US-OAK-PER-1-1",
            "config": {
              "led_enabled": true,
              "band_24": {
                "disabled": false,
                "channel": 6,
                "tx_power": 10,
                "bandwidth": 40
              },
              "band_5": {
                "disabled": false,
                "channel": 36,
                "tx_power": 15,
                "bandwidth": 80
              },
              "band_6": {
                "disabled": true
              }
            }
          }
        },
        "gateway": {},
        "switch": {}
      }
    }
  }
}
```

### Configuration Update Process

1. **AP Matching**: APs are matched by name (case-insensitive)
2. **Radio Updates**: Only radio configurations are updated:
   - `band_24.channel`
   - `band_24.tx_power`
   - `band_24.bandwidth`
   - `band_24.disabled`
   - (Same for `band_5` and `band_6`)
3. **Preservation**: All other AP settings (MAC, location, tags, etc.) are preserved
4. **File Save**: Updated configuration is saved back to the original file

### Value Mappings

#### Channel Values
- PDF: `"0"` or `"auto"` → Config: `0` (integer)
- PDF: `"1"` to `"165"` → Config: `1` to `165` (integer)

#### Power Values  
- PDF: `"auto"` or empty → Config: `0` (integer, means auto)
- PDF: Numeric string → Config: Integer value

#### Width/Bandwidth Values
- PDF: Empty or `-1` → Config: Uses default from configuration (see Defaults Configuration)
- PDF: `"20"`, `"40"`, etc. → Config: `20`, `40`, etc. (integer)

#### Disabled State
- Band not present in PDF → Config: `disabled: true`
- Band present in PDF → Config: `disabled: false`

## Defaults Configuration

### Application Config Structure
The application configuration (`wifimgr-config.json`) can include a `.defaults` section that specifies default bandwidth values for each band:

```json
{
  "defaults": {
    "ap": {
      "band_24": {
        "bandwidth": 20
      },
      "band_5": {
        "bandwidth": 40
      },
      "band_6": {
        "bandwidth": 80
      }
    }
  }
}
```

### Default Bandwidth Behavior
- When a bandwidth value is empty or `-1` in the PDF, the system uses the default from the configuration
- If no defaults are configured, the system falls back to 20MHz for all bands
- Defaults are applied per-band, allowing different defaults for 2.4GHz, 5GHz, and 6GHz

## Output and Feedback

### Success Output
```
✅ Successfully updated site configuration: ~/.config/wifimgr/us-oak-per.json
Updated 4 APs:
  • US-OAK-PER-1-1
  • US-OAK-PER-1-2
  • US-OAK-PER-2-1
  • US-OAK-PER-3-1

AP Configurations from PDF:

AP Name            2.4G         5G           6G
-----------------  -----------  -----------  -----------
US-OAK-PER-1-1     1/5/20       auto/9/20    disabled
US-OAK-PER-1-2     6/5/20       auto/9/20    disabled
US-OAK-PER-2-1     11/5/20      auto/9/20    disabled
US-OAK-PER-3-1     1/5/20       auto/9/20    disabled
```

### Warning Messages
- **No matching APs**: APs found in PDF but not in site config
- **No configurations found**: PDF doesn't contain any matching AP configuration strings
- **Site not found**: No configuration file for the specified site

## Error Handling

### Common Errors
1. **File not found**: PDF file doesn't exist at specified path
   ```
   Error: file not found: /path/to/missing.pdf
   ```

2. **Site not found**: No configuration file for the specified site
   ```
   Error: site US-LAB-01 not found in configuration
   ```

3. **No matches**: PDF doesn't contain any matching AP configuration strings
   ```
   No AP configurations found in the PDF file
   ```

4. **Parse errors**: Invalid format in extracted strings

### Validation
- Channel numbers are validated against allowed ranges per band
- Width values are validated per band specifications
- Site names are matched case-insensitively
- AP names must match exactly (case-insensitive)

## Implementation Details

### Libraries
- **PDF Parsing**: `github.com/ledongthuc/pdf` (chosen over UniPDF due to licensing)
- **Text Extraction**: Multiple methods for robust extraction
  - Content stream text extraction
  - GetPlainText() for direct text access
  - Handles both concatenated and spaced text variations

### Files Created
- `cmd/import.go` - Main import command registration
- `cmd/import_pdf.go` - PDF import subcommand implementation
- `internal/pdf/types.go` - Data structures (APConfig, BandConfig)
- `internal/pdf/parser.go` - PDF parsing and regex extraction

### Key Functions
- `ParseFile()` - Extracts text from PDF and finds AP configurations
- `parseRadioSettings()` - Parses band configuration strings
- `updateAPRadioConfig()` - Updates AP config with PDF data
- `naturalCompare()` - Implements natural sorting for AP names

### Data Structure Changes

#### Device Storage
The implementation uses maps for device storage:
```go
type Devices struct {
    APs      map[string]APConfig      `json:"ap"`      // Keyed by MAC address
    Switches map[string]SwitchConfig  `json:"switch"`  // Keyed by MAC address
    WanEdge  map[string]WanEdgeConfig `json:"gateway"` // Keyed by MAC address
}
```

#### AP Configuration
The PDF AP configuration structure is simplified:
```go
type APConfig struct {
    Name    string      // Full AP name (everything before first slash)
    Band24G *BandConfig // 2.4GHz band configuration
    Band5G  *BandConfig // 5GHz band configuration
    Band6G  *BandConfig // 6GHz band configuration
}
```

## Testing

### Sample PDF
The file `peralta.pdf` should contain AP configurations in the format:
```
@2-1/2:11:5/5:0:9
@1-1/2:1:5/5:0:9
@1-2/2:6:5/5:0:9
@3-1/2:1:5/5:0:9
```

**Note**: The PDF needs to be updated to include the `@` prefix for AP configurations.

### Test Command
```bash
./wifimgr -e import type pdf file peralta.pdf site US-OAK-PER
```

## Future Enhancements

### Potential v2 Features
- Support for additional file types (Excel, CSV)
- Export extracted configurations to JSON/CSV
- Validation of channel/power/width values against regulatory limits
- Duplicate AP detection and handling
- Batch processing of multiple PDF files
- Dry-run mode to preview changes without applying
- Integration with apply command for direct device updates
- Support for additional device types (switches, gateways)
- Regular expression customization via configuration
- Automatic prefix/suffix detection from site configuration

### Known Limitations
- PDF text extraction quality depends on PDF structure and encoding
- Some PDFs may require OCR for scanned documents
- Currently only supports the specific AP configuration format defined in the regex
- Band configurations must follow the exact format specification
- No support for custom or vendor-specific extensions
- Prefix/suffix must be specified manually if AP names differ between PDF and configuration
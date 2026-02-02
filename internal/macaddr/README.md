# MAC Address Utility Package

The `macaddr` package provides utilities for handling and manipulating MAC addresses
in various formats, supporting the most common styles (colon, hyphen, dot, and no separators).
The primary purpose is to provide a consistent way to handle MAC addresses throughout the application.

## Features

- **Format Conversion**: Convert between different MAC address formats
- **Validation**: Validate MAC addresses in various formats
- **Normalization**: Normalize MAC addresses to a consistent format (lowercase, no separators)
- **Format Detection**: Detect the format of a MAC address
- **Equality Comparison**: Compare MAC addresses regardless of format

## Supported Formats

The package supports the following MAC address formats:

| Format | Example | Constant |
|--------|---------|----------|
| No separators | `001122334455` | `FormatNone` |
| Colon-separated | `00:11:22:33:44:55` | `FormatColon` |
| Hyphen-separated | `00-11-22-33-44-55` | `FormatHyphen` |
| Dot-separated (Cisco) | `0011.2233.4455` | `FormatDot` |

## Usage Examples

### Normalize a MAC Address

```go
// Convert any MAC format to the standard normalized form (lowercase, no separators)
normalizedMac, err := macaddr.Normalize("00:11:22:AA:BB:CC")
// normalizedMac = "001122aabbcc"

// If you're certain the MAC is valid or prefer empty strings over errors
normalizedMac := macaddr.NormalizeOrEmpty("00:11:22:AA:BB:CC")
```

### Format a MAC Address

```go
// Convert to colon format
formattedMac, err := macaddr.Format("001122aabbcc", macaddr.FormatColon)
// formattedMac = "00:11:22:aa:bb:cc"

// Convert to hyphen format
formattedMac, err := macaddr.Format("00:11:22:aa:bb:cc", macaddr.FormatHyphen)
// formattedMac = "00-11-22-aa-bb-cc"

// Convert to dot format (Cisco)
formattedMac, err := macaddr.Format("00:11:22:aa:bb:cc", macaddr.FormatDot)
// formattedMac = "0011.22aa.bbcc"
```

### Validate a MAC Address

```go
// Check if a string is a valid MAC in any supported format
isValid := macaddr.IsValid("00:11:22:33:44:55")  // true
isValid := macaddr.IsValid("001122334455")       // true
isValid := macaddr.IsValid("invalid")            // false

// Check if a MAC is in a specific format
isColonFormat := macaddr.IsValidWithFormat("00:11:22:33:44:55", macaddr.FormatColon)  // true
isColonFormat := macaddr.IsValidWithFormat("00-11-22-33-44-55", macaddr.FormatColon)  // false
```

### Compare MAC Addresses

```go
// Compare MAC addresses regardless of format
areEqual := macaddr.Equal("00:11:22:33:44:55", "00-11-22-33-44-55")  // true
areEqual := macaddr.Equal("00:11:22:33:44:55", "001122334455")       // true
areEqual := macaddr.Equal("00:11:22:33:44:55", "00:11:22:33:44:56")  // false
```

### Detect MAC Address Format

```go
format, err := macaddr.DetectFormat("00:11:22:33:44:55")
// format = macaddr.FormatColon

format, err := macaddr.DetectFormat("00-11-22-33-44-55")
// format = macaddr.FormatHyphen

format, err := macaddr.DetectFormat("0011.2233.4455")
// format = macaddr.FormatDot

format, err := macaddr.DetectFormat("001122334455")
// format = macaddr.FormatNone
```

## Error Handling

The package provides proper error handling for invalid MAC addresses:

```go
// All functions that might fail return meaningful errors
_, err := macaddr.Normalize("invalid")
if err != nil {
    // Handle error: err = macaddr.ErrInvalidMAC
}
```
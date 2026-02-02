# Utility Systems

## MAC Address Handling

MAC addresses should be handled using the `internal/macaddr` utility package, which provides:

- Standard normalization: lowercase with no separators (e.g., "001122aabbcc")
- Format conversion between different styles (colon, hyphen, dot, and no separators)
- Validation for different MAC formats
- Format detection
- Equality comparison

This ensures consistent MAC address handling throughout the application.

### Common operations

- Normalize a MAC address: `macaddr.Normalize("00:11:22:33:44:55")` → "001122334455"
- Format a MAC address: `macaddr.Format("001122334455", macaddr.FormatColon)` → "00:11:22:33:44:55"
- Validate a MAC address: `macaddr.IsValid("00:11:22:33:44:55")` → true
- Compare MAC addresses: `macaddr.Equal("00:11:22:33:44:55", "00-11-22-33-44-55")` → true

## Symbol System

### Terminal-Aware Behavior

**Terminal Environments** (when output goes to a terminal that supports colors):
- **⏺** (bold green) - Success/Connected/True states
- **⏺** (bold red) - Failed/Disconnected/False states
- **⏺** (bold blue) - Unknown/Undefined/Question states

**Non-Terminal Environments** (when output is redirected, piped, or used in tests):
- **Y** - Success/Connected/True states
- **N** - Failed/Disconnected/False states
- **?** - Unknown/Undefined/Question states

### Critical Terminal Detection

The terminal detection logic is essential for proper color display and must not be modified without extreme care:

```go
func isTerminal() bool {
    // Check file descriptors to determine if we're in a terminal
    stdoutFd := int(os.Stdout.Fd())
    stdinFd := int(os.Stdin.Fd())
    stderrFd := int(os.Stderr.Fd())

    // Try stdout first (most common case for CLI output)
    if term.IsTerminal(stdoutFd) {
        return true
    }

    // Fallback to stdin (interactive terminal)
    if term.IsTerminal(stdinFd) {
        return true
    }

    // Fallback to stderr (error output terminal)
    if term.IsTerminal(stderrFd) {
        return true
    }

    return false
}
```

**Why Multi-FD Detection is Required:**
- Different execution contexts may have different FD configurations
- Some terminals/environments redirect specific file descriptors
- The fallback chain ensures reliable detection across various scenarios
- Simplified detection (checking only stdout) breaks color display

**DO NOT:**
- Simplify to check only `os.Stdout.Fd()`
- Remove the fallback checks for stdin/stderr
- Modify this logic without extensive testing across different environments

### Usage Examples

```go
import "github.com/ravinald/wifimgr/internal/symbols"

// Success message
fmt.Printf("%s Configuration applied successfully\n", symbols.GreenCircle())

// Error message
fmt.Printf("%s Configuration failed\n", symbols.RedCircle())

// Unknown status
fmt.Printf("%s Device status unknown\n", symbols.BlueCircle())
```

## Sorting System

The application uses natural sorting (via `github.com/maruel/natural`) for all sorting operations to provide intuitive ordering that handles numeric components properly.

### Current Sorting Functions

All sorting functions are implemented in `/api/sort.go`:

1. **Primary Inventory Sorting**: `SortInventoryNew()` (line 148)
   - **Sort Order Priority**: Site Name → Device Type → Device Name → MAC Address
   - Groups devices by site first, then by type within each site
   - Items with undefined values are placed at the end of their respective groups

2. **Sites Sorting**: `SortSites()`, `SortSitesNew()`
   - **Sort Order**: Site name (natural sorting)

3. **AP Sorting**: `SortAPs()`
   - **Sort Order**: AP name (natural sorting)

4. **Legacy Inventory Sorting**: `SortInventory()`
   - Maintained for backward compatibility

### Natural Sorting Benefits
- Handles numeric components: "AP2", "AP10", "AP20" (not "AP10", "AP2", "AP20")
- Case-insensitive comparison
- Handles mixed alphanumeric strings intuitively

## Site ID and Name Handling

When working with sites in the application, it's critical to use the correct functions for site ID/name lookups:

- To get a site name from a site ID: `client.GetSiteName(siteID)`
- To get a site object from a site name: `client.GetSiteByName(ctx, siteName, orgID)`

IMPORTANT: Never use `GetSiteBySiteName()` with a site ID as the parameter. This will result in incorrect site name lookups.
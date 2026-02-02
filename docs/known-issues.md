# Known Issues

This document lists known issues and limitations in wifimgr, including their causes and any available workarounds.

## Meraki SDK Debug Output

**Severity:** Low - Cosmetic only

**Issue:** When using the Meraki API, you may see unwanted console output like:
```
MAX_RETRIES: 1
```

**Cause:** This is a bug in the upstream Meraki Go SDK (`github.com/meraki/dashboard-api-go` v5.0.8). The SDK has hardcoded `fmt.Println("MAX_RETRIES: ", maxRetries+1)` debug statements in `sdk/api_client.go:584` that run on every API call. These debug print statements execute regardless of any debug/logging configuration in wifimgr.

**Status:** Upstream bug - Not fixable in wifimgr. The debug print statement is located inside the `doWithRetriesAndResult` function and executes for all API requests.

**Impact:** The output is purely cosmetic and does not affect:
- Functionality of API calls
- Data accuracy
- Command execution
- Performance

**Workaround:** None currently available. You can safely ignore this output.

**Upstream Reference:**
- The Meraki SDK previously had similar issues (see [github.com/meraki/dashboard-api-go issue #6](https://github.com/meraki/dashboard-api-go/issues/6))
- This specific debug print statement was introduced in a later version

---

## Reporting New Issues

If you encounter issues with wifimgr that are not listed here, please report them:

1. Check the [project repository](https://github.com/ravi-pina/wifimgr) for existing issues
2. Provide detailed information including:
   - wifimgr version
   - Command that triggered the issue
   - Full error message or unexpected behavior
   - Steps to reproduce
   - Your environment (OS, API vendor being used, etc.)

# Inventory Safety Feature - Quick Reference

## What is Inventory Safety?

A dual-inventory system that **prevents accidental modifications** to network devices. Devices must be explicitly allowlisted before write operations (apply, configure, assign, unassign) will work.

## Read vs. Write Operations

| Operation Type | Requirement | Examples |
|---|---|---|
| **Write** | In BOTH API + Local inventory | apply, configure, assign, unassign |
| **Read** | API inventory only | show, search, list |

## Checking If a Device Is Allowlisted

```bash
# List all available devices (API inventory - read-only)
wifimgr show api ap

# Check your local inventory file
cat ~/.config/wifimgr/inventory.json
```

## Allowlisting a Device

1. **Find the device MAC** from API:
   ```bash
   wifimgr show api ap | grep device-name
   ```

2. **Add to inventory.json**:
   ```json
   {
     "config": {
       "inventory": {
         "ap": ["aa:bb:cc:dd:ee:01"]
       }
     }
   }
   ```

3. **Reload and retry**:
   ```bash
   wifimgr apply site-name ap
   ```

## Error Message

```
SAFETY CHECK FAILED: Device aa:bb:cc:dd:ee:ff is not in inventory - refusing to update
```

**Solution**: Add device MAC to `inventory.json` in the appropriate list (ap, switch, or gateway)

## Inventory File Format

```json
{
  "version": 1.0,
  "config": {
    "inventory": {
      "ap": [
        "5c:5b:35:8e:4c:f9",
        "5c:5b:35:8e:4c:fa"
      ],
      "switch": [
        "d0:23:4d:a1:b2:c3"
      ],
      "gateway": [
        "e8:65:d4:12:34:56"
      ]
    }
  }
}
```

## Common Scenarios

### Scenario 1: New Device Won't Configure

```
Error: Device is not in inventory
```

1. Confirm device exists in API: `wifimgr show api ap | grep device-name`
2. Copy the MAC address
3. Add to `inventory.json`
4. Reload cache if needed: `wifimgr refresh cache`
5. Retry the operation

### Scenario 2: Want to View All Devices First

```bash
# This works - no allowlist required for reading
wifimgr show api ap site MY-SITE json

# Review the list, note MACs you want to manage
# Then add them to inventory.json
```

### Scenario 3: Recently Added Device to Mist/Meraki

```bash
# New device in vendor account but can't apply yet
wifimgr apply site-name ap

# Solution: Refresh cache to pick up new device
wifimgr refresh cache

# Then allowlist in inventory.json
```

## Why This Protection?

1. **Prevents Accidents**: Can't modify devices you didn't explicitly authorize
2. **Multi-User Safe**: Team members see all devices but can only modify allowlisted ones
3. **Audit Trail**: Inventory file documents approved devices
4. **Gradual Rollout**: Add devices one at a time with confidence

## Key Rules

- **Allowlist is mandatory**: No exceptions for write operations
- **MAC addresses only**: Use `aa:bb:cc:dd:ee:ff` format
- **Case-insensitive**: `AA:BB:CC:DD:EE:FF` works too
- **No editing needed for reads**: View all devices without allowlist
- **Configuration stored locally**: Only affects your instance


## Troubleshooting

| Problem | Solution |
|---|---|
| Device doesn't appear in `show` | Device not in vendor account - add via web GUI |
| Device appears in `show` but `apply` fails | Add MAC to inventory.json |
| Recently added device still fails | Run `wifimgr refresh cache` |
| Configuration won't load | Check JSON syntax in inventory.json |
| Not sure about device MAC | Run `wifimgr show api ap json` to see MACs |

## Next Steps

- **Allowlist devices in inventory**: Edit `inventory.json`
- **Test with one device**: Add one MAC, apply changes, verify success
- **Gradual expansion**: Add more devices as you gain confidence
- **Monitor logs**: Enable debug mode for detailed safety messages: `wifimgr -d apply ...`

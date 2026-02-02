# Inventory Safety Feature - Documentation Index

## Quick Navigation

### For Users
Start here if you're using wifimgr and need to understand the allowlist safety mechanism.

| Document | Purpose | Read Time |
|----------|---------|-----------|
| [Inventory Safety Quick Reference](./INVENTORY_SAFETY_QUICK_REFERENCE.md) | One-page reference for all common tasks | 5 min |
| [Configuration Guidelines - Inventory Safety Feature](./configuration.md#inventory-safety-feature) | Detailed explanation with examples | 10 min |

---

## The Feature at a Glance

**What**: A dual-inventory system that prevents accidental device modifications

**How**: Devices must exist in BOTH:
1. API inventory (devices in your Mist/Meraki account)
2. Local inventory (explicit allowlist in config file)

**When**: Applies to write operations only (apply, configure, assign, unassign)

**Why**: Prevents accidents, provides audit trail, enables multi-user safety

---

## Common Scenarios

### "I got an error about device not in inventory"
1. Read: [Quick Reference - Error Message](./INVENTORY_SAFETY_QUICK_REFERENCE.md#error-message)
2. Action: Add device MAC to inventory.json
3. Done: Retry your command

### "I want to understand how this works"
1. Read: [Inventory Safety Feature](./configuration.md#inventory-safety-feature)
2. Read: [Quick Reference](./INVENTORY_SAFETY_QUICK_REFERENCE.md)
3. Done: You'll understand the why and how


---

## File Map

### Public Documentation
```
docs/
├── configuration.md                           # Main config docs
│   └── "Inventory Safety Feature" section     # User explanation
├── INVENTORY_SAFETY_QUICK_REFERENCE.md        # One-page reference
└── INVENTORY_SAFETY_DOCUMENTATION_INDEX.md    # This file
```

---

## Learning Path

### Getting Started with Inventory Safety (40 minutes)
1. [Quick Reference](./INVENTORY_SAFETY_QUICK_REFERENCE.md) (5 min)
2. [Configuration Guidelines](./configuration.md#inventory-safety-feature) (15 min)
3. Understand scenarios: [Quick Reference Examples](./INVENTORY_SAFETY_QUICK_REFERENCE.md#common-scenarios) (10 min)
4. Troubleshooting: [Quick Reference Troubleshooting](./INVENTORY_SAFETY_QUICK_REFERENCE.md#troubleshooting) (5 min)
5. Practice: Add a device to inventory.json (5 min)

---

## Key Concepts

### Inventory Types
- **API Inventory**: Devices in your vendor account (Mist/Meraki)
  - Source: Cache (updated via `refresh cache`)
  - Purpose: Know what devices exist

- **Local Inventory**: Allowlisted devices in your config
  - Source: inventory.json configuration file
  - Purpose: Explicit approval for modifications

### Operation Types
- **Write Operations**: apply, configure, assign, unassign
  - Requirement: Device in BOTH inventories
  - Protection: Allowlist prevents accidents

- **Read Operations**: show, search, list
  - Requirement: API inventory only
  - Freedom: View all devices before allowlisting

### Safety Rules
```
Write OP allowed  IF device in API inventory AND device in local inventory
Read OP allowed   IF device in API inventory
```

---

## Configuration Reference

### Inventory File Format
```json
{
  "version": 1.0,
  "config": {
    "inventory": {
      "ap": [
        "5c5b358e4cf9",
        "5c5b358e4cfa"
      ],
      "switch": [
        "d0234da1b2c3"
      ],
      "gateway": [
        "e865d4123456"
      ]
    }
  }
}
```

### Key Points
- File path specified in main config (typically `./config/inventory.json`)
- MAC addresses in any format (normalized automatically)
- Separate lists for ap, switch, and gateway device types
- Changes take effect immediately when file is saved

---

## Troubleshooting Quick Index

| Problem | Solution | More Info |
|---------|----------|-----------|
| "Device not in inventory" error | Add MAC to inventory.json | [Quick Reference - Error Message](./INVENTORY_SAFETY_QUICK_REFERENCE.md#error-message) |
| Device in `show` but `apply` fails | Add MAC to inventory.json | [Configuration - Allowlist Device](./configuration.md#inventory-configuration) |
| Recently added device not found | Run `wifimgr refresh cache` | [Configuration Guide](./configuration.md) |
| Not sure about device MAC | Run `wifimgr show api ap json` | [Quick Reference - Find MAC](./INVENTORY_SAFETY_QUICK_REFERENCE.md#allowlisting-a-device) |
| Want to view all devices | Use `show` commands (no allowlist required) | [Configuration - Workflow](./configuration.md#typical-workflow) |

---

## Documentation Standards

All inventory safety documentation follows these principles:

1. **User-Focused**: Explains why, not just how
2. **Example-Driven**: Real configuration and code examples
3. **Action-Oriented**: Clear steps to accomplish goals
4. **Comprehensive**: All scenarios covered
5. **Consistent**: Terminology used consistently
6. **Linked**: Cross-references between sections
7. **Accurate**: All examples from actual source code

---

## Quick Links

### For Users
- Problem: Device won't modify → [Quick Reference](./INVENTORY_SAFETY_QUICK_REFERENCE.md#error-message)
- Question: How do I allowlist? → [Configuration Guide](./configuration.md#inventory-configuration)
- Learn: Why does this exist? → [Quick Reference](./INVENTORY_SAFETY_QUICK_REFERENCE.md#why-this-protection)

---

## Additional Resources

- **Source Code**: `cmd/apply/inventory_check.go` (239 lines)
- **Usage Example**: `cmd/apply/device_update_ap.go` (lines 181-189)
- **Tests**: `cmd/apply/inventory_check_test.go`
- **Configuration**: `config.inventory.*` arrays in configuration file

---

**Last Updated**: January 28, 2026
**Status**: Complete and current
**Scope**: All user and developer documentation for inventory safety feature

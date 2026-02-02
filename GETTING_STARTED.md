# Getting Started with wifimgr v0.1.0

Welcome to wifimgr! This guide will help you find the right documentation for your needs.

---

## Quick Navigation

**Are you trying to...**

### Get Started Quickly?
Start with [README.md](README.md) for a 5-minute overview, then:
1. [Configure API access](docs/configuration.md)
2. [Run your first command](docs/user-guide.md)

### Set Up Configuration?
See [docs/configuration.md](docs/configuration.md) for:
- API setup (Mist or Meraki)
- Token management
- File organization
- Multi-API configuration

### Learn All Available Commands?
Read [docs/user-guide.md](docs/user-guide.md) for:
- Complete command reference
- Practical examples
- Tips and tricks
- Output format options

### Use Multiple Vendors (Mist and Meraki)?
Follow [docs/multi-vendor/user-guide.md](docs/multi-vendor/user-guide.md) for:
- Multi-vendor setup
- Using the --api flag
- Cross-vendor workflows

### Find Something Specific?

**Search for clients:**
- [docs/user-guide.md - Search section](docs/user-guide.md#search) - Wireless/wired client search

**Manage cache/freshness:**
- [docs/user-guide.md - Refresh section](docs/user-guide.md#refresh) - Cache management

**Configure devices:**
- [docs/device-configuration.md](docs/device-configuration.md) - Device config format

**Understand field mappings:**
- [docs/field-mappings.md](docs/field-mappings.md) - API field reference

**Choose output format:**
- [docs/table-formatter.md](docs/table-formatter.md) - Table/CSV/JSON options

**Resolve issues:**
- [docs/known-issues.md](docs/known-issues.md) - Known problems and workarounds

### Contribute to Development?
See [CONTRIBUTING.md](CONTRIBUTING.md) for:
- Development setup
- Coding standards
- Testing guidelines
- Pull request process

### Check What's New?
See [CHANGELOG.md](CHANGELOG.md) for:
- v0.1.0 features
- Known limitations
- Future roadmap

### Understand the Architecture?
See [docs-internal/](docs-internal/) for developer documentation:
- System design
- Multi-vendor architecture
- Command patterns
- Cache system
- Testing guidelines

---

## Documentation by User Role

### End Users
1. [README.md](README.md) - Start here
2. [docs/configuration.md](docs/configuration.md) - Setup
3. [docs/user-guide.md](docs/user-guide.md) - Commands

### Site Administrators / Network Operators
1. [README.md](README.md) - Overview
2. [docs/user-guide.md](docs/user-guide.md) - All commands
3. [docs/known-issues.md](docs/known-issues.md) - Troubleshooting
4. [docs/multi-vendor/user-guide.md](docs/multi-vendor/user-guide.md) - If using multiple vendors

### DevOps / Infrastructure Engineers
1. [docs/configuration.md](docs/configuration.md) - Setup
2. [docs/user-guide.md#refresh](docs/user-guide.md#refresh) - Cache management
3. [docs-internal/ci-cd.md](docs-internal/ci-cd.md) - CI/CD integration
4. [SECURITY.md](SECURITY.md) - Security best practices

### Developers / Contributors
1. [CONTRIBUTING.md](CONTRIBUTING.md) - Development setup
2. [docs-internal/architecture.md](docs-internal/architecture.md) - System design
3. [docs-internal/command-architecture.md](docs-internal/command-architecture.md) - Command patterns
4. [docs-internal/multi-vendor/overview.md](docs-internal/multi-vendor/overview.md) - Multi-vendor design

### Project Managers / Release Planners
1. [CHANGELOG.md](CHANGELOG.md) - v0.1.0 features and roadmap
2. [README.md](README.md) - Feature overview
3. [DOCUMENTATION_REVIEW_v0.1.0.md](DOCUMENTATION_REVIEW_v0.1.0.md) - Documentation status

---

## Common Tasks

### Install and Configure
```bash
# Install
git clone https://github.com/ravinald/wifimgr.git && cd wifimgr && make build

# Configure
mkdir -p ~/.config/wifimgr
cp config/wifimgr-config-sample.json ~/.config/wifimgr/wifimgr-config.json

# Add token
echo "WIFIMGR_API_TOKEN=your-token" > .env.wifimgr

# Verify
wifimgr -e refresh cache
```
See [docs/configuration.md](docs/configuration.md) for detailed setup.

### View Infrastructure
```bash
wifimgr show api sites
wifimgr show api ap site US-LAB-01
wifimgr show inventory ap
```
See [docs/user-guide.md - Show command](docs/user-guide.md#show) for all options.

### Search for Devices
```bash
wifimgr search wireless laptop-john
wifimgr search wired 5c:5b:35:8e:4c:f9
```
See [docs/user-guide.md - Search command](docs/user-guide.md#search) for details.

### Configure Devices
```bash
wifimgr apply ap US-LAB-01 --dry-run
wifimgr apply ap US-LAB-01
```
See [docs/user-guide.md - Apply command](docs/user-guide.md#apply) for examples.

### Manage Cache
```bash
wifimgr show api status
wifimgr refresh cache
```
See [docs/user-guide.md - Refresh command](docs/user-guide.md#refresh) for cache management.

### Use Multiple Vendors
```bash
wifimgr show api sites --api mist-prod
wifimgr search wireless john --api meraki-corp
```
See [docs/multi-vendor/user-guide.md](docs/multi-vendor/user-guide.md) for setup.

---

## Important Links

### Essential Documentation
- [README.md](README.md) - Quick start
- [docs/user-guide.md](docs/user-guide.md) - Command reference
- [docs/configuration.md](docs/configuration.md) - Setup guide
- [CHANGELOG.md](CHANGELOG.md) - Version history

### Support & Community
- [CONTRIBUTING.md](CONTRIBUTING.md) - How to contribute
- [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) - Community standards
- [SECURITY.md](SECURITY.md) - Security policy
- [GitHub Issues](https://github.com/ravinald/wifimgr/issues) - Report bugs

### Additional Resources
- [DOCUMENTATION_INDEX.md](DOCUMENTATION_INDEX.md) - Complete documentation index
- [docs-internal/](docs-internal/) - Developer documentation
- [docs/](docs/) - All user documentation

---

## Need Help?

1. **Check the relevant documentation** - Use the navigation above to find your topic
2. **Search the docs** - Look for specific keywords in the guide files
3. **Check known issues** - See [docs/known-issues.md](docs/known-issues.md) for workarounds
4. **Report an issue** - Create an issue on [GitHub](https://github.com/ravinald/wifimgr/issues)

---

**Version:** v0.1.0
**Last Updated:** 2026-02-02

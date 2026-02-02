# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |
| 0.x (other) | :x:            |

## Reporting a Vulnerability

If you discover a security vulnerability in wifimgr, please report it responsibly.

### How to Report

1. **Do NOT** create a public GitHub issue for security vulnerabilities
2. Email the maintainers directly at: security@cow.org
3. Include:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Any suggested fixes (optional)

### What to Expect

- Acknowledgment within 48 hours
- Status update within 7 days
- We aim to release fixes within 30 days for critical issues

### Disclosure Policy

- We follow coordinated disclosure practices
- We'll credit you in the security advisory (unless you prefer anonymity)
- Please allow us reasonable time to address the issue before public disclosure

## Security Best Practices for Users

### API Token Security

- Store API tokens in `.env.wifimgr` file, not in config files
- Never commit API tokens to version control
- Use environment variables for CI/CD pipelines
- Rotate tokens periodically

### Configuration Security

- Protect configuration files with appropriate file permissions
- Review site configurations before applying
- Use `--diff` mode to preview changes
- Keep backups of configurations

### Network Security

- Use HTTPS endpoints only (default)
- Validate SSL certificates
- Be cautious with proxy configurations

## Known Security Considerations

### Cache Files

- Cache files may contain sensitive network information
- Located in `./cache` directory by default
- Protect with appropriate file system permissions

### Backup Files

- Configuration backups may contain device settings
- Review backup retention policies
- Secure backup directories appropriately

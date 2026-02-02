# Contributing to wifimgr

Thank you for your interest in contributing to wifimgr! This document provides guidelines and instructions for contributing.

## Code of Conduct

Please read and follow our [Code of Conduct](CODE_OF_CONDUCT.md).

## How to Contribute

### Reporting Bugs

1. Check if the bug has already been reported in [Issues](https://github.com/ravinald/wifimgr/issues)
2. If not, create a new issue using the bug report template
3. Include:
   - A clear, descriptive title
   - Steps to reproduce the issue
   - Expected vs actual behavior
   - Version information (`wifimgr version`)
   - Relevant configuration (with sensitive data removed)

### Suggesting Features

1. Check existing issues and discussions for similar suggestions
2. Create a new issue using the feature request template
3. Describe the use case and expected behavior

### Pull Requests

1. Fork the repository
2. Create a feature branch from `main`:
   ```bash
   git checkout -b feature/your-feature-name
   ```
3. Make your changes following our coding standards
4. Write or update tests as needed
5. Run tests locally:
   ```bash
   go test ./...
   ```
6. Commit with clear, descriptive messages
7. Push to your fork and create a Pull Request

## Development Setup

### Prerequisites

- Go 1.22.3 or later
- Make

### Building

```bash
git clone https://github.com/ravinald/wifimgr.git
cd wifimgr
make build
```

### Running Tests

```bash
make test           # Run all tests
make test-coverage  # Generate coverage report
```

### Testing with Real APIs

For testing with real Mist/Meraki APIs:
- Use only test sites designated for development (ZZ-TMP-SITE, ZZ-WTMP-SITE)
- Never test against production sites
- Use `--diff` mode to preview changes before applying

## Coding Standards

### Go Style

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` and `goimports` for formatting
- Run `golangci-lint` before submitting

### Naming Conventions

- Use clear, descriptive names
- Package names: lowercase, single word
- Exported functions/types: PascalCase with Godoc comments
- Unexported: camelCase
- Acronyms: consistent casing (e.g., `OrgID`, `MAC`)

### Documentation

- Add Godoc comments to all exported functions and types
- Keep comments current with code changes
- Update relevant documentation files in `/docs`

### Error Handling

- Wrap errors with context: `fmt.Errorf("action failed: %w", err)`
- Return early on errors
- Use custom error types where appropriate

### Testing

- Write table-driven tests where applicable
- Test error cases, not just happy paths
- Use meaningful test names describing the scenario

## Pull Request Guidelines

### Before Submitting

- [ ] Code follows project style guidelines
- [ ] Tests pass locally (`go test ./...`)
- [ ] New code has appropriate test coverage
- [ ] Documentation is updated if needed
- [ ] Commit messages are clear and descriptive

### PR Description

- Summarize the changes and their purpose
- Reference any related issues
- Note any breaking changes
- Include test plan or verification steps

## Architecture Notes

Key architectural concepts (see `/docs` for details):

- **Multi-vendor architecture**: Supports Mist and Meraki APIs
- **Cache system**: Three-layer cache for performance
- **Cobra CLI**: Command hierarchy with Junos-style positional arguments
- **Apply command**: Currently supports AP configuration only

## Questions?

- Open a [Discussion](https://github.com/ravinald/wifimgr/discussions) for general questions
- Check existing documentation in `/docs`
- Review the [README](README.md) for usage information

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.

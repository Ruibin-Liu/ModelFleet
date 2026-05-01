# Contributing to ModelFleet

Thank you for your interest in contributing to ModelFleet! This document provides guidelines and instructions for contributing.

## Development Setup

### Prerequisites

- Go 1.21 or later
- Git
- Pre-commit (optional but recommended)

### Setup

```bash
# Clone the repository
git clone https://github.com/modelfleet/modelfleet.git
cd modelfleet

# Install dependencies
go mod download

# Install pre-commit hooks (optional)
pre-commit install
```

## Making Changes

### Branch Naming

- `feature/description` - New features
- `bugfix/description` - Bug fixes
- `docs/description` - Documentation updates
- `refactor/description` - Code refactoring

### Commit Messages

Follow conventional commits:

```
feat: add new feature
fix: correct bug in deployment execution
docs: update README with new examples
refactor: simplify health check logic
test: add tests for gateway routing
```

### Code Style

- Format code with `gofmt`
- Run `go vet` before committing
- Keep functions focused and small
- Add comments for exported functions
- Handle errors explicitly

### Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/api/...
```

## Pull Request Process

1. **Fork and branch**: Create a feature branch from `main`
2. **Make changes**: Implement your feature or fix
3. **Test**: Ensure all tests pass
4. **Format**: Run `gofmt` and `goimports`
5. **Commit**: Write clear commit messages
6. **Push**: Push your branch to your fork
7. **PR**: Open a pull request with clear description

### PR Checklist

- [ ] Tests pass locally
- [ ] Code is formatted with `gofmt`
- [ ] No new linting errors
- [ ] Documentation updated if needed
- [ ] Commit messages follow conventions

## Code Review

All submissions require review. We aim to respond within:

- **Bug fixes**: 1-2 days
- **Features**: 3-5 days
- **Documentation**: 1 day

## Reporting Issues

When reporting bugs, please include:

- Go version (`go version`)
- Operating system
- Steps to reproduce
- Expected vs actual behavior
- Relevant logs or error messages

## Security

For security issues, please email security@modelfleet.dev instead of opening a public issue.

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

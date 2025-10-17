# Contributing to go-apperr

Thank you for your interest in contributing to go-apperr! This document provides guidelines and information for contributors.

## Code of Conduct

This project adheres to a code of conduct that we expect all contributors to follow. Please be respectful and constructive in all interactions.

## Getting Started

1. Fork the repository on GitHub
2. Clone your fork locally
3. Create a new branch for your feature or bug fix
4. Make your changes
5. Add tests for your changes
6. Ensure all tests pass
7. Submit a pull request

## Development Setup

### Prerequisites

- Go 1.21 or later (required for `log/slog` features)
- Target version: Go 1.25
- Git

### Setting up the development environment

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/go-apperr.git
cd go-apperr

# Install dependencies
go mod download

# Run tests to ensure everything works
go test ./...

# Run linting
golangci-lint run
```

## Making Changes

### Code Style

- Follow Go's standard formatting (`gofmt`)
- Use `golangci-lint` for additional linting
- Write clear, self-documenting code
- Add comments for exported functions and types
- Follow the existing code style in the project

### Testing

- Write tests for all new functionality
- Ensure existing tests continue to pass
- Aim for high test coverage
- Use table-driven tests where appropriate
- Test both success and error cases

### Documentation

- Update documentation for any API changes
- Add examples in code comments where helpful
- Update README.md if needed
- Ensure all exported functions have proper godoc comments

## Pull Request Process

1. **Create a feature branch** from `main`
2. **Make your changes** following the guidelines above
3. **Add tests** for your changes
4. **Update documentation** if needed
5. **Run tests and linting** locally
6. **Commit your changes** with clear commit messages
7. **Push to your fork** and create a pull request

### Pull Request Guidelines

- Use clear, descriptive titles
- Provide a detailed description of changes
- Reference any related issues
- Ensure CI checks pass
- Request review from maintainers

### Commit Message Format

Use clear, descriptive commit messages:

```
feat: add new error code for rate limiting
fix: correct stack trace capture in production
docs: update README with new examples
test: add tests for error wrapping functionality
```

## Reporting Issues

When reporting issues, please include:

- Go version
- Operating system
- Steps to reproduce
- Expected behavior
- Actual behavior
- Code example (if applicable)

## Feature Requests

We welcome feature requests! Please:

- Check existing issues first
- Provide a clear description of the feature
- Explain the use case and benefits
- Consider implementation complexity
- Be open to discussion and feedback

## Code Review Process

All submissions require review. We'll review your code for:

- Correctness and functionality
- Code style and formatting
- Test coverage and quality
- Documentation completeness
- Performance implications
- Security considerations

## Release Process

Releases are managed by maintainers and follow semantic versioning:

- **MAJOR** version for incompatible API changes
- **MINOR** version for new functionality in a backwards compatible manner
- **PATCH** version for backwards compatible bug fixes

## Questions?

If you have questions about contributing, please:

- Open an issue with the "question" label
- Start a discussion in the GitHub Discussions
- Contact maintainers directly

Thank you for contributing to go-apperr!

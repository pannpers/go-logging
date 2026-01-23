# AGENT.md

## Project Overview

`go-logging` is a shared Go library designed to provide structured logging capabilities built on top of the standard `log/slog` package. It aims to offer seamless integration with OpenTelemetry and simplify context-aware logging in distributed systems.

## Key Features

- **Structured Logging**: Wraps `log/slog` for structured output (JSON/Text).
- **OpenTelemetry Support**: Automatically extracts and logs trace/span IDs from `context.Context`.
- **Context Attributes**: Mechanism to carry specific logging attributes within `context.Context` (via `SetAttrs`).
- **Connect-RPC Integration**: Interceptors for Connect-RPC services.

## Architecture

- **`logging` package**: Logic resides here.
  - `slog_logger.go`: Main `Logger` struct and convenience methods (`Info`, `Error`, etc.).

## Development Guidelines

1.  **Go Version**: Target Go 1.25.
2.  **Dependencies**:
    - `log/slog` (Standard Library)
    - `go.opentelemetry.io/otel` (Tracing)
    - `connectrpc.com/connect` (RPC support)
3.  **Testing**: Ensure all new features are covered by unit tests. Use `go test ./...`.
4.  **Linting**: Use `golangci-lint` to maintain code quality.

## Common Tasks

- **Adding a new features**:
  - Add the feature to `logging` package.
  - Add tests.
  - Update `README.md` features section.
- **Refactoring**:
  - Ensure backward compatibility where possible, as this is a shared library.

## File Structure

- `logging/`: Core library code.
- `README.md`: Documentation for users.
- `CONTRIBUTING.md`: Guide for contributors.
- `LICENSE`: MIT License.

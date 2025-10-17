# go-apperr

[![Go Report Card](https://goreportcard.com/badge/github.com/pannpers/go-apperr)](https://goreportcard.com/report/github.com/pannpers/go-apperr)
[![GoDoc](https://godoc.org/github.com/pannpers/go-apperr?status.svg)](https://godoc.org/github.com/pannpers/go-apperr)
[![Build Status](https://github.com/pannpers/go-apperr/workflows/test/badge.svg)](https://github.com/pannpers/go-apperr/actions)
[![Coverage Status](https://coveralls.io/repos/github/pannpers/go-apperr/badge.svg?branch=main)](https://coveralls.io/github/pannpers/go-apperr?branch=main)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A comprehensive Go error handling library with gRPC/Connect-RPC status codes, structured logging, and automatic stack trace capture. go-apperr provides a clean, consistent way to handle errors across your Go applications.

## Features

- **Status Code Compatibility**: Full support for gRPC and Connect-RPC status codes
- **Structured Logging**: Built-in integration with `log/slog` for structured error logging
- **Stack Trace Capture**: Configurable automatic stack trace capture for debugging
- **Error Chaining**: Support for error wrapping and unwrapping with `errors.Is` and `errors.As`
- **Connect-RPC Integration**: Optional interceptor for Connect-RPC services
- **Type Safety**: Strongly typed error codes and comprehensive error classification
- **Performance**: Minimal overhead with configurable features

## Requirements

- Go 1.21 or later (required for `log/slog` features)
- Target version: Go 1.25

## Installation

To install go-apperr, use `go get`:

```bash
go get github.com/pannpers/go-apperr
```

This will make the following packages available:

```
github.com/pannpers/go-apperr/apperr          # Core error handling
github.com/pannpers/go-apperr/apperr/codes    # Status codes
github.com/pannpers/go-apperr/apperr/connect  # Connect-RPC integration (optional)
```

## Quick Start

### Basic Error Creation

```go
package main

import (
    "errors"
    "log/slog"

    "github.com/pannpers/go-apperr/apperr"
    "github.com/pannpers/go-apperr/apperr/codes"
)

func main() {
    // Create a new error with status code
    err := apperr.New(codes.InvalidArgument, "user ID cannot be empty")

    // Wrap an existing error
    dbErr := errors.New("connection failed")
    err = apperr.Wrap(dbErr, codes.Internal, "failed to get user",
        slog.String("user_id", "123"))

    // Check error type
    if errors.Is(err, apperr.ErrInvalidArgument) {
        // Handle invalid argument error
    }
}
```

### Error Comparison

```go
// Use predefined error variables for semantic comparison
if errors.Is(err, apperr.ErrNotFound) {
    // Handle not found error
}

if errors.Is(err, apperr.ErrInternal) {
    // Handle internal server error
}

// Check if error is a server error (5xx)
var appErr *apperr.AppErr
if errors.As(err, &appErr) {
    if appErr.Code.IsServerError() {
        // This is a server error that should be logged
    }
}
```

### Structured Logging

```go
import "log/slog"

// AppErr implements slog.LogValuer for structured logging
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
logger.Error("operation failed", slog.Any("error", err))

// Output:
// {
//   "level": "ERROR",
//   "msg": "operation failed",
//   "error": {
//     "msg": "failed to get user: connection failed (internal)",
//     "code": "internal",
//     "cause": "connection failed",
//     "attrs": {
//       "user_id": "123"
//     }
//   }
// }
```

### Stack Trace Configuration

```go
// Enable stack traces in development
apperr.Configure(
    apperr.WithStacktrace(true),
)

// Disable stack traces in production (default)
apperr.Configure(
    apperr.WithStacktrace(false),
)

// Check current setting
if apperr.IsStacktraceEnabled() {
    // Stack traces are being captured
}
```

### Connect-RPC Integration

```go
import (
    "log/slog"
    "connectrpc.com/connect"
    "github.com/pannpers/go-apperr/apperr/connect"
)

// Create logger
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

// Create error handling interceptor
interceptor := connect.NewErrorHandlingInterceptor(logger)

// Use with Connect server
server := connect.NewServer(
    connect.WithInterceptors(interceptor),
)
```

## Error Codes

go-apperr provides comprehensive status code support compatible with gRPC and Connect-RPC:

### Server Errors (5xx)

- `Internal` - Internal server error
- `Unknown` - Unknown error
- `DataLoss` - Unrecoverable data loss
- `Unimplemented` - Operation not implemented
- `Unavailable` - Service temporarily unavailable
- `DeadlineExceeded` - Operation timed out

### Client Errors (4xx)

- `InvalidArgument` - Invalid request parameters
- `NotFound` - Resource not found
- `AlreadyExists` - Resource already exists
- `PermissionDenied` - Insufficient permissions
- `Unauthenticated` - Invalid or missing authentication
- `FailedPrecondition` - Operation precondition failed
- `Aborted` - Operation aborted
- `OutOfRange` - Operation out of valid range
- `ResourceExhausted` - Resources exhausted
- `Canceled` - Operation canceled

## API Reference

### Core Functions

- `apperr.New(code, msg, attrs...)` - Create new error
- `apperr.Wrap(err, code, msg, attrs...)` - Wrap existing error
- `apperr.Configure(opts...)` - Configure global settings
- `apperr.IsStacktraceEnabled()` - Check stack trace setting

### Error Methods

- `AppErr.Error()` - Implement error interface
- `AppErr.Unwrap()` - Support for error unwrapping
- `AppErr.Is(target)` - Support for error comparison
- `AppErr.LogValue()` - Implement slog.LogValuer

### Code Methods

- `Code.IsServerError()` - Check if code represents server error
- `Code.ToConnect()` - Convert to Connect-RPC code
- `Code.String()` - String representation

## Configuration

### Stack Trace Options

```go
// Enable/disable stack traces
apperr.Configure(apperr.WithStacktrace(true))

// Check current setting
enabled := apperr.IsStacktraceEnabled()
```

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by [testify](https://github.com/stretchr/testify) for its clean API design
- Built for compatibility with [Connect-RPC](https://connectrpc.com/) and gRPC
- Uses Go's standard `log/slog` for structured logging

## Support

- 📖 [Documentation](https://godoc.org/github.com/pannpers/go-apperr)
- 🐛 [Issues](https://github.com/pannpers/go-apperr/issues)
- 💬 [Discussions](https://github.com/pannpers/go-apperr/discussions)

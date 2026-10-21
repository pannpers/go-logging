# go-logging

[![Go Report Card](https://goreportcard.com/badge/github.com/pannpers/go-logging)](https://goreportcard.com/report/github.com/pannpers/go-logging)
[![GoDoc](https://godoc.org/github.com/pannpers/go-logging?status.svg)](https://godoc.org/github.com/pannpers/go-logging)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A shared library for structured logging using `log/slog`. `go-logging` provides seamless integration with OpenTelemetry and context-aware logging capabilities.

## Features

- **Structured Logging**: Built on top of Go's standard `log/slog` library.
- **OpenTelemetry Integration**: Automatic extraction of trace and span IDs from context.
- **Context Attributes**: Store and retrieve logging attributes within the context.
- **Connect-RPC Support**: Access log interceptors for connect-go v1 and v2 that write the same log fields.
- **Configurable**: Support for JSON/Text formats, log levels, and output writers.

## Requirements

- Go 1.27 or later

## Installation

```bash
go get github.com/pannpers/go-logging
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "log/slog"
    "os"

    "github.com/pannpers/go-logging/logging"
)

func main() {
    // Initialize logger with default settings
    logger, err := logging.New()
    if err != nil {
        panic(err)
    }

    ctx := context.Background()

    // Log with context (automatically handles Otel traces if present)
    logger.Info(ctx, "Application started")

    // Log with attributes
    logger.Error(ctx, "An error occurred", fmt.Errorf("connection error"),
        slog.String("component", "database"))
}
```

### Advanced Configuration

```go
opts := []logging.Option{
    logging.WithFormat(logging.FormatJSON),
    logging.WithLevel(slog.LevelDebug),
    logging.WithWriter(os.Stderr),
}

logger, err := logging.New(opts...)
```

### Context Attributes

```go
// Add attributes to context for downstream logging
ctx = logging.SetAttrs(ctx,
    slog.String("request_id", "req-123"),
    slog.String("user_id", "user-456"),
)

// All logs using this ctx will include request_id and user_id
logger.Info(ctx, "Processing request")
```

### Connect-RPC Access Logs

Both interceptors write one `access log` record per RPC with the same attributes:
`rpc`, `status` (`ok`, a Connect error code, or `unknown` for other errors),
`duration_ms`, and a `headers` group (`user_agent`, `remote_addr`, `method`,
plus any headers selected with `WithHTTPHeaders`).

connect-go v1 (`connectrpc.com/connect`):

```go
import (
    "connectrpc.com/connect"

    "github.com/pannpers/go-logging/logging"
)

interceptor := logging.NewAccessLogInterceptor(logger,
    logging.WithHTTPHeaders([]string{"X-Request-Id"}),
)
path, handler := userv1connect.NewUserServiceHandler(svc, connect.WithInterceptors(interceptor))
```

connect-go v2 (`connectrpc.com/connect/v2`):

```go
import (
    "connectrpc.com/connect/v2"

    "github.com/pannpers/go-logging/logging"
    loggingconnect "github.com/pannpers/go-logging/logging/connectv2"
)

srv := connect.NewServer(
    loggingconnect.NewAccessLogInterceptor(logger,
        logging.WithHTTPHeaders([]string{"X-Request-Id"}),
    ),
)
```

Differences in the v2 interceptor:

- **Streaming RPCs are logged.** The v1 interceptor is unary-only. The v2
  interceptor wraps every RPC type, so each client, server, or bidi stream
  produces one record when the handler returns, and `duration_ms` covers the
  whole stream.
- **`duration_ms` includes request decoding.** connect-go v2 runs interceptors
  before decompression and unmarshaling, so the measured time covers slightly
  more work than in v1.

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

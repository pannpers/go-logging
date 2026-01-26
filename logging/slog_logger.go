// Package logging provides a structured logging library built on top of Go's standard log/slog package.
// It offers OpenTelemetry tracing integration, Connect RPC interceptors, and flexible configuration options.
//
// Features:
//   - Structured logging using log/slog
//   - JSON and text output formats
//   - Automatic OpenTelemetry trace and span ID extraction
//   - Context-based attribute storage and retrieval
//   - Connect RPC access log interceptor
//   - Configurable log levels, writers, and attribute transformation
//
// Basic Usage:
//
//	logger, err := logging.New()
//	if err != nil {
//		panic(err)
//	}
//
//	ctx := context.Background()
//	logger.Info(ctx, "Application started")
//	logger.Error(ctx, "An error occurred", err, slog.String("component", "database"))
//
// Advanced Configuration:
//
//	opts := []logging.Option{
//		logging.WithFormat(logging.FormatJSON),
//		logging.WithLevel(slog.LevelDebug),
//		logging.WithWriter(os.Stderr),
//	}
//
//	logger, err := logging.New(opts...)
//
//	// Create service-specific logger
//	serviceLogger := logger.With(
//		slog.String("service", "user-service"),
//		slog.String("version", "1.0.0"),
//	)
//
// Context Attributes:
//
//	// Add attributes to context
//	ctx = logging.SetAttrs(ctx,
//		slog.String("request_id", "req-123"),
//		slog.String("user_id", "user-456"),
//	)
//
//	// All logs with this context will include the attributes
//	logger.Info(ctx, "Processing request")
package logging

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

// Logger is a structured logger that wraps slog.Logger with additional functionality.
// It automatically extracts OpenTelemetry trace information and context attributes,
// providing a seamless logging experience for distributed systems.
type Logger struct {
	logger *slog.Logger
}

// New creates a new Logger with the specified configuration options.
// If no options are provided, it uses default settings: text format,
// info level, and stdout writer.
//
// Example:
//
//	logger, err := logging.New(
//		logging.WithFormat(logging.FormatJSON),
//		logging.WithLevel(slog.LevelDebug),
//	)
//	if err != nil {
//		return err
//	}
func New(opts ...Option) (*Logger, error) {
	o := defaultOptions()

	for _, opt := range opts {
		opt(o)
	}

	handlerOpts := &slog.HandlerOptions{
		Level:       o.level,
		ReplaceAttr: o.replaceAttrFunc,
	}

	var handler slog.Handler

	switch o.format {
	case FormatText:
		handler = slog.NewTextHandler(o.writer, handlerOpts)
	case FormatJSON:
		handler = slog.NewJSONHandler(o.writer, handlerOpts)
	default:
		return nil, fmt.Errorf("unknown logger format: %d", o.format)
	}

	return &Logger{logger: slog.New(handler)}, nil
}

// Debug logs a debug-level message with optional attributes.
// Debug messages are typically used for detailed diagnostic information
// that is only of interest when diagnosing problems.
//
// The message will include any attributes stored in the context via SetAttrs,
// OpenTelemetry trace information if available, and the provided attributes.
func (l *Logger) Debug(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.log(ctx, slog.LevelDebug, msg, attrs...)
}

// Info logs an info-level message with optional attributes.
// Info messages are typically used for general information about
// the application's operation.
//
// The message will include any attributes stored in the context via SetAttrs,
// OpenTelemetry trace information if available, and the provided attributes.
func (l *Logger) Info(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.log(ctx, slog.LevelInfo, msg, attrs...)
}

// Warn logs a warning-level message with optional attributes.
// Warning messages are typically used for potentially harmful situations
// that don't prevent the application from continuing.
//
// The message will include any attributes stored in the context via SetAttrs,
// OpenTelemetry trace information if available, and the provided attributes.
func (l *Logger) Warn(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.log(ctx, slog.LevelWarn, msg, attrs...)
}

// Error logs an error-level message with an error object and optional attributes.
// The error object will be automatically added as an "error" attribute.
// If the error implements slog.LogValue, its custom representation will be used.
//
// The message will include any attributes stored in the context via SetAttrs,
// OpenTelemetry trace information if available, the error attribute, and the provided attributes.
//
// Example:
//
//	logger.Error(ctx, "Database connection failed", err,
//		slog.String("host", "localhost"),
//		slog.Int("port", 5432),
//	)
func (l *Logger) Error(ctx context.Context, msg string, err error, attrs ...slog.Attr) {
	errorAttr := slog.Any("error", err)

	allArgs := make([]slog.Attr, 0, len(attrs)+1)
	allArgs = append(allArgs, errorAttr) // Error attribute first for better readability
	allArgs = append(allArgs, attrs...)

	l.log(ctx, slog.LevelError, msg, allArgs...)
}

// With returns a new Logger that includes the given attributes in all log entries.
// This is useful for creating service-specific or component-specific loggers
// that always include certain contextual information.
//
// Example:
//
//	serviceLogger := logger.With(
//		slog.String("service", "user-service"),
//		slog.String("version", "1.0.0"),
//	)
//
//	// All logs from serviceLogger will include service and version
//	serviceLogger.Info(ctx, "Service started")
func (l *Logger) With(args ...slog.Attr) *Logger {
	slogArgs := make([]any, len(args))
	for i, v := range args {
		slogArgs[i] = v
	}

	return &Logger{
		logger: l.logger.With(slogArgs...),
	}
}

// attrKeyType is the type for the context key to store logging attributes.
type attrKeyType struct{}

// attrKey is the context key used to store logging attributes in context values.
var attrKey = attrKeyType{}

// SetAttrs adds logging attributes to a context. These attributes will be
// automatically included in all log entries made with the returned context.
// This is useful for request-scoped or operation-scoped logging context.
//
// Example:
//
//	ctx = logging.SetAttrs(ctx,
//		slog.String("request_id", "req-123"),
//		slog.String("user_id", "user-456"),
//	)
//
//	// All subsequent logs with this context will include request_id and user_id
//	logger.Info(ctx, "Processing request")
func SetAttrs(ctx context.Context, attrs ...slog.Attr) context.Context {
	return context.WithValue(ctx, attrKey, attrs)
}

// GetAttrs retrieves logging attributes from a context that were previously
// stored using SetAttrs. Returns an empty slice if no attributes are found.
//
// This function is primarily used internally by the Logger, but may be useful
// for custom logging implementations or debugging.
func GetAttrs(ctx context.Context) []slog.Attr {
	attrs, ok := ctx.Value(attrKey).([]slog.Attr)
	if !ok {
		return []slog.Attr{}
	}

	return attrs
}

// log is the internal logging method that handles context attributes and OpenTelemetry trace information.
// It extracts trace IDs, span IDs, and context attributes, then combines them with the provided attributes
// before passing them to the underlying slog logger.
func (l *Logger) log(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	// Extract trace and span IDs from context.
	otelAttrs := traceFromContext(ctx)
	otherAttrs := GetAttrs(ctx)

	allArgs := make([]slog.Attr, 0, len(attrs)+len(otelAttrs)+len(otherAttrs))
	allArgs = append(allArgs, attrs...)
	allArgs = append(allArgs, otelAttrs...)
	allArgs = append(allArgs, otherAttrs...)

	l.logger.LogAttrs(ctx, level, msg, allArgs...)
}

// traceFromContext extracts trace and span IDs from context using OpenTelemetry.
// Returns a slice of slog.Attr containing trace_id and span_id if a valid span context is found.
// Returns an empty slice if no valid span context is available.
func traceFromContext(ctx context.Context) []slog.Attr {
	var attrs []slog.Attr

	spanContext := trace.SpanFromContext(ctx).SpanContext()

	if !spanContext.IsValid() {
		return attrs
	}

	attrs = append(attrs,
		slog.String("trace_id", spanContext.TraceID().String()),
		slog.String("span_id", spanContext.SpanID().String()),
	)

	return attrs
}

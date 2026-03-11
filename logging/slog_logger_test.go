package logging_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"testing"

	"github.com/pannpers/go-logging/logging"
	"go.opentelemetry.io/otel/trace"
)

// contextWithTrace creates a new context with a span context derived from the given trace and span ID hex strings.
func contextWithTrace(traceID, spanID string) context.Context {
	tid, err := trace.TraceIDFromHex(traceID)
	if err != nil {
		panic(fmt.Sprintf("invalid traceIDStr for test: %s, error: %v", traceID, err))
	}

	sid, err := trace.SpanIDFromHex(spanID)
	if err != nil {
		panic(fmt.Sprintf("invalid spanIDStr for test: %s, error: %v", spanID, err))
	}

	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    tid,
		SpanID:     sid,
		TraceFlags: trace.FlagsSampled, // Mark as sampled
	})

	return trace.ContextWithSpanContext(context.Background(), spanCtx)
}

// validateJSONOutput parses JSON and validates expected key-value pairs
func validateJSONOutput(t *testing.T, output string, expected map[string]any) {
	t.Helper()

	var actual map[string]any
	if err := json.Unmarshal([]byte(output), &actual); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, output)
	}

	for key, expectedValue := range expected {
		actualValue, exists := actual[key]
		if !exists {
			t.Errorf("expected key %q not found in output", key)
			continue
		}

		if !reflect.DeepEqual(actualValue, expectedValue) {
			t.Errorf("key %q: expected %v, got %v", key, expectedValue, actualValue)
		}
	}
}

func TestLogger_Debug(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger, err := logging.New(
		logging.WithWriter(&buf),
		logging.WithLevel(slog.LevelDebug),
		logging.WithFormat(logging.FormatJSON),
	)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	logger.Debug(context.Background(), "debug message", slog.String("key", "value"))

	output := buf.String()
	expectedFields := map[string]any{
		"level": "DEBUG",
		"msg":   "debug message",
		"key":   "value",
	}
	validateJSONOutput(t, output, expectedFields)
}

func TestLogger_Info(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger, err := logging.New(
		logging.WithWriter(&buf),
		logging.WithLevel(slog.LevelInfo),
		logging.WithFormat(logging.FormatJSON),
	)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	logger.Info(context.Background(), "info message", slog.Int("count", 42))

	output := buf.String()
	expectedFields := map[string]any{
		"level": "INFO",
		"msg":   "info message",
		"count": float64(42),
	}
	validateJSONOutput(t, output, expectedFields)
}

func TestLogger_Warn(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger, err := logging.New(
		logging.WithWriter(&buf),
		logging.WithLevel(slog.LevelWarn),
		logging.WithFormat(logging.FormatJSON),
	)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	logger.Warn(context.Background(), "warning message", slog.Bool("retry", true))

	output := buf.String()
	expectedFields := map[string]any{
		"level": "WARN",
		"msg":   "warning message",
		"retry": true,
	}
	validateJSONOutput(t, output, expectedFields)
}

type testError struct {
	Message string
	Code    string
}

func (e *testError) Error() string {
	return e.Message
}

// LogValue implements slog.LogValue interface.
func (e *testError) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("message", e.Message),
		slog.String("code", e.Code),
	)
}

func TestLogger_Error(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger, err := logging.New(
		logging.WithWriter(&buf),
		logging.WithLevel(slog.LevelError),
		logging.WithFormat(logging.FormatJSON),
	)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	testErr := &testError{
		Message: "test error",
		Code:    "404",
	}
	logger.Error(context.Background(), "error occurred", testErr, slog.String("component", "database"))

	output := buf.String()
	expectedFields := map[string]any{
		"level":     "ERROR",
		"msg":       "error occurred",
		"component": "database",
		// if error implements slog.LogValue, it will be logged as a group
		"error": map[string]any{
			"message": "test error",
			"code":    "404",
		},
	}
	validateJSONOutput(t, output, expectedFields)
}

func TestLogger_WithTraceContext(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger, err := logging.New(
		logging.WithWriter(&buf),
		logging.WithLevel(slog.LevelInfo),
		logging.WithFormat(logging.FormatJSON),
	)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	ctx := contextWithTrace("0102030405060708090a0b0c0d0e0f10", "a1a2a3a4a5a6a7a8")
	logger.Info(ctx, "traced message")

	output := buf.String()
	expectedFields := map[string]any{
		"level":    "INFO",
		"msg":      "traced message",
		"trace_id": "0102030405060708090a0b0c0d0e0f10",
		"span_id":  "a1a2a3a4a5a6a7a8",
	}
	validateJSONOutput(t, output, expectedFields)
}

func TestLogger_WithContextAttrs(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger, err := logging.New(
		logging.WithWriter(&buf),
		logging.WithLevel(slog.LevelInfo),
		logging.WithFormat(logging.FormatJSON),
	)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	// Test the WithContext and FromContext functionality
	ctx := logging.SetAttrs(context.Background(),
		slog.String("request_id", "req-123"),
		slog.String("user_id", "user-456"),
	)
	logger.Info(ctx, "request processed")

	output := buf.String()
	expectedFields := map[string]any{
		"level":      "INFO",
		"msg":        "request processed",
		"request_id": "req-123",
		"user_id":    "user-456",
	}
	validateJSONOutput(t, output, expectedFields)
}

func TestLogger_With(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger, err := logging.New(
		logging.WithWriter(&buf),
		logging.WithLevel(slog.LevelInfo),
		logging.WithFormat(logging.FormatJSON),
	)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	// Create a logger with pre-set attributes
	childLogger := logger.With(
		slog.String("service", "user-service"),
		slog.String("version", "v1.0.0"),
	)

	childLogger.Info(context.Background(), "service started", slog.Int("port", 8080))

	output := buf.String()
	expectedFields := map[string]any{
		"level":   "INFO",
		"msg":     "service started",
		"service": "user-service",
		"version": "v1.0.0",
		"port":    float64(8080),
	}
	validateJSONOutput(t, output, expectedFields)
}

func TestLogger_Slog(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger, err := logging.New(
		logging.WithWriter(&buf),
		logging.WithLevel(slog.LevelInfo),
		logging.WithFormat(logging.FormatJSON),
	)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	slogLogger := logger.Slog()
	if slogLogger == nil {
		t.Fatal("Slog() returned nil")
	}

	// Verify the returned *slog.Logger writes to the same writer
	slogLogger.Info("slog direct message", "key", "value")

	output := buf.String()
	expectedFields := map[string]any{
		"level": "INFO",
		"msg":   "slog direct message",
		"key":   "value",
	}
	validateJSONOutput(t, output, expectedFields)
}

func TestLogger_Slog_WithAttrs(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger, err := logging.New(
		logging.WithWriter(&buf),
		logging.WithLevel(slog.LevelInfo),
		logging.WithFormat(logging.FormatJSON),
	)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	// Slog() on a Logger created with With() should preserve attributes
	childLogger := logger.With(slog.String("service", "test-service"))
	slogLogger := childLogger.Slog()

	slogLogger.Info("child slog message")

	output := buf.String()
	expectedFields := map[string]any{
		"level":   "INFO",
		"msg":     "child slog message",
		"service": "test-service",
	}
	validateJSONOutput(t, output, expectedFields)
}

func TestLogger_ComplexScenario(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger, err := logging.New(
		logging.WithWriter(&buf),
		logging.WithLevel(slog.LevelInfo),
		logging.WithFormat(logging.FormatJSON),
	)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	// Complex scenario: trace context + context attrs + With() + method attrs
	ctx := contextWithTrace("aabbccddaabbccddaabbccddaabbccdd", "1122334411223344")
	ctx = logging.SetAttrs(ctx, slog.String("correlation_id", "corr-789"))

	serviceLogger := logger.With(slog.String("component", "auth"))
	serviceLogger.Error(ctx, "authentication failed",
		errors.New("invalid credentials"),
		slog.String("username", "john.doe"),
		slog.Int("attempt", 3),
	)

	output := buf.String()
	expectedFields := map[string]any{
		"level":          "ERROR",
		"msg":            "authentication failed",
		"component":      "auth",
		"correlation_id": "corr-789",
		"trace_id":       "aabbccddaabbccddaabbccddaabbccdd",
		"span_id":        "1122334411223344",
		"username":       "john.doe",
		"attempt":        float64(3),
		"error":          "invalid credentials",
	}
	validateJSONOutput(t, output, expectedFields)
}

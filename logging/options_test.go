package logging_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/pannpers/go-logging/logging"
)

func TestDefaultOptions(t *testing.T) {
	t.Parallel()

	// Create logger with default options
	logger, err := logging.New()
	if err != nil {
		t.Fatalf("failed to create logger with default options: %v", err)
	}

	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestWithWriter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		writer       io.Writer
		expectOutput bool
	}{
		{
			name:         "custom writer receives output",
			writer:       &bytes.Buffer{},
			expectOutput: true,
		},
		{
			name:         "nil writer should be ignored and use default stdout",
			writer:       nil,
			expectOutput: false, // No output to our buffer since nil writer is ignored
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			opts := []logging.Option{
				logging.WithFormat(logging.FormatJSON),
			}

			opts = append(opts, logging.WithWriter(tc.writer))

			logger, err := logging.New(opts...)
			if err != nil {
				t.Fatalf("failed to create logger: %v", err)
			}

			logger.Info(context.Background(), "test message")

			if tc.expectOutput {
				if buf, ok := tc.writer.(*bytes.Buffer); ok {
					output := buf.String()
					if output == "" {
						t.Error("expected output to custom writer but got empty")
					}
					if !containsJSON(output, "level", "INFO") {
						t.Errorf("expected level INFO in output: %s", output)
					}
					if !containsJSON(output, "msg", "test message") {
						t.Errorf("expected msg 'test message' in output: %s", output)
					}
				} else {
					t.Error("expected bytes.Buffer for output validation")
				}
			}
			// For nil writer case, we just verify no panic occurred
		})
	}
}

func TestWithLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		level        slog.Level
		logLevel     slog.Level
		expectOutput bool
	}{
		{
			name:         "debug level allows debug logs",
			level:        slog.LevelDebug,
			logLevel:     slog.LevelDebug,
			expectOutput: true,
		},
		{
			name:         "info level blocks debug logs",
			level:        slog.LevelInfo,
			logLevel:     slog.LevelDebug,
			expectOutput: false,
		},
		{
			name:         "info level allows info logs",
			level:        slog.LevelInfo,
			logLevel:     slog.LevelInfo,
			expectOutput: true,
		},
		{
			name:         "warn level blocks info logs",
			level:        slog.LevelWarn,
			logLevel:     slog.LevelInfo,
			expectOutput: false,
		},
		{
			name:         "error level allows error logs",
			level:        slog.LevelError,
			logLevel:     slog.LevelError,
			expectOutput: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer

			logger, err := logging.New(
				logging.WithWriter(&buf),
				logging.WithLevel(tc.level),
				logging.WithFormat(logging.FormatJSON),
			)
			if err != nil {
				t.Fatalf("failed to create logger: %v", err)
			}

			// Log at the test level
			switch tc.logLevel {
			case slog.LevelDebug:
				logger.Debug(context.Background(), "debug message")
			case slog.LevelInfo:
				logger.Info(context.Background(), "info message")
			case slog.LevelWarn:
				logger.Warn(context.Background(), "warn message")
			case slog.LevelError:
				logger.Error(context.Background(), "error message", nil)
			}

			output := buf.String()
			hasOutput := output != ""

			if tc.expectOutput && !hasOutput {
				t.Errorf("expected output but got empty, level: %v, logLevel: %v", tc.level, tc.logLevel)
			}
			if !tc.expectOutput && hasOutput {
				t.Errorf("expected no output but got: %s, level: %v, logLevel: %v", output, tc.level, tc.logLevel)
			}
		})
	}
}

func TestWithFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		format         logging.Format
		validateOutput func(t *testing.T, output string)
	}{
		{
			name:   "JSON format",
			format: logging.FormatJSON,
			validateOutput: func(t *testing.T, output string) {
				t.Helper()
				if !containsJSON(output, "level", "INFO") {
					t.Errorf("expected JSON format with level INFO: %s", output)
				}
				if !containsJSON(output, "msg", "test message") {
					t.Errorf("expected JSON format with msg 'test message': %s", output)
				}
			},
		},
		{
			name:   "Text format",
			format: logging.FormatText,
			validateOutput: func(t *testing.T, output string) {
				t.Helper()
				if !containsText(output, "level", "INFO") {
					t.Errorf("expected text format with level=INFO: %s", output)
				}
				if !containsText(output, "msg", `"test message"`) {
					t.Errorf("expected text format with msg=\"test message\": %s", output)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer

			logger, err := logging.New(
				logging.WithWriter(&buf),
				logging.WithLevel(slog.LevelInfo),
				logging.WithFormat(tc.format),
			)
			if err != nil {
				t.Fatalf("failed to create logger: %v", err)
			}

			logger.Info(context.Background(), "test message")

			output := buf.String()
			if output == "" {
				t.Fatal("expected output but got empty")
			}

			tc.validateOutput(t, output)
		})
	}
}

func TestWithReplaceAttr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		replaceFunc func(groups []string, a slog.Attr) slog.Attr
		validate    func(t *testing.T, output string)
	}{
		{
			name: "replace level key",
			replaceFunc: func(_ []string, a slog.Attr) slog.Attr {
				if a.Key == slog.LevelKey {
					a.Key = "severity"
				}
				return a
			},
			validate: func(t *testing.T, output string) {
				t.Helper()
				if !containsJSON(output, "severity", "INFO") {
					t.Errorf("expected 'severity' field instead of 'level': %s", output)
				}
				if containsJSON(output, "level", "INFO") {
					t.Errorf("should not contain 'level' field: %s", output)
				}
			},
		},
		{
			name: "remove time attribute",
			replaceFunc: func(_ []string, a slog.Attr) slog.Attr {
				if a.Key == slog.TimeKey {
					return slog.Attr{} // Remove time attribute
				}
				return a
			},
			validate: func(t *testing.T, output string) {
				t.Helper()
				if containsAnyTimeField(output) {
					t.Errorf("should not contain time field: %s", output)
				}
			},
		},
		{
			name: "modify attribute value",
			replaceFunc: func(_ []string, a slog.Attr) slog.Attr {
				if a.Key == slog.MessageKey {
					a.Value = slog.StringValue("modified: " + a.Value.String())
				}
				return a
			},
			validate: func(t *testing.T, output string) {
				t.Helper()
				if !containsJSON(output, "msg", "modified: test message") {
					t.Errorf("expected modified message: %s", output)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer

			logger, err := logging.New(
				logging.WithWriter(&buf),
				logging.WithLevel(slog.LevelInfo),
				logging.WithFormat(logging.FormatJSON),
				logging.WithReplaceAttr(tc.replaceFunc),
			)
			if err != nil {
				t.Fatalf("failed to create logger: %v", err)
			}

			logger.Info(context.Background(), "test message")

			output := buf.String()
			if output == "" {
				t.Fatal("expected output but got empty")
			}

			tc.validate(t, output)
		})
	}
}

// Helper functions for validation

func containsJSON(output, key, expectedValue string) bool {
	// Simple JSON field check - look for "key":"value" pattern
	pattern := `"` + key + `":"` + expectedValue + `"`
	return bytes.Contains([]byte(output), []byte(pattern)) //nolint:mirror // TODO: CIを通すため無視する
}

func containsText(output, key, expectedValue string) bool {
	// Simple text field check - look for key=value pattern
	pattern := key + "=" + expectedValue
	return bytes.Contains([]byte(output), []byte(pattern)) //nolint:mirror // TODO: CIを通すため無視する
}

func containsAnyTimeField(output string) bool {
	// Check for common time field names
	return bytes.Contains([]byte(output), []byte(`"time"`)) || //nolint:mirror // TODO: CIを通すため無視する
		bytes.Contains([]byte(output), []byte("time=")) //nolint:mirror // TODO: CIを通すため無視する
}

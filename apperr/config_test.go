package apperr_test

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/pannpers/go-apperr/apperr"
	"github.com/pannpers/go-apperr/apperr/codes"
)

func TestStacktraceConfiguration(t *testing.T) {
	// Don't run in parallel due to global state
	// Save and restore original setting for each test

	tests := []struct {
		name                string
		includeStacktrace   bool
		wantStacktraceInLog bool
	}{
		{
			name:                "exclude stacktrace when disabled (default)",
			includeStacktrace:   false,
			wantStacktraceInLog: false,
		},
		{
			name:                "include stacktrace when enabled",
			includeStacktrace:   true,
			wantStacktraceInLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Don't run in parallel - modifies global state
			// Save original setting
			originalSetting := apperr.IsStacktraceEnabled()
			defer apperr.Configure(apperr.WithStacktrace(originalSetting))

			// Set the stacktrace configuration
			apperr.Configure(apperr.WithStacktrace(tt.includeStacktrace))

			// Create an error with stacktrace
			err := apperr.New(codes.Internal, "test error", slog.String("key", "value"))

			// Log the error
			var buf bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			}))

			logger.Error("error occurred", slog.Any("error", err))

			// Check if stacktrace is in the log
			logOutput := buf.String()
			containsStacktrace := strings.Contains(logOutput, "stacktrace")

			if containsStacktrace != tt.wantStacktraceInLog {
				t.Errorf("stacktrace presence in log doesn't match expectation: got %v, want %v",
					containsStacktrace, tt.wantStacktraceInLog)
			}

			// Verify other attributes are still present
			if !strings.Contains(logOutput, "test error") {
				t.Error("message should be in log")
			}
			if !strings.Contains(logOutput, "key") {
				t.Error("custom attributes should be in log")
			}
			if !strings.Contains(logOutput, "value") {
				t.Error("custom attribute values should be in log")
			}
		})
	}
}

func TestIsStacktraceEnabled(t *testing.T) {
	// Don't run in parallel - modifies global state
	// Save original setting
	originalSetting := apperr.IsStacktraceEnabled()
	defer apperr.Configure(apperr.WithStacktrace(originalSetting))

	tests := []struct {
		name     string
		setValue bool
	}{
		{
			name:     "set to true",
			setValue: true,
		},
		{
			name:     "set to false",
			setValue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apperr.Configure(apperr.WithStacktrace(tt.setValue))
			if got := apperr.IsStacktraceEnabled(); got != tt.setValue {
				t.Errorf("IsStacktraceEnabled() = %v, want %v", got, tt.setValue)
			}
		})
	}
}

func TestConfigure(t *testing.T) {
	// Don't run in parallel - modifies global state
	// Save original setting
	originalSetting := apperr.IsStacktraceEnabled()
	defer apperr.Configure(apperr.WithStacktrace(originalSetting))

	tests := []struct {
		name           string
		configureWith  bool
		wantStacktrace bool
	}{
		{
			name:           "configure with stacktrace enabled",
			configureWith:  true,
			wantStacktrace: true,
		},
		{
			name:           "configure with stacktrace disabled",
			configureWith:  false,
			wantStacktrace: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apperr.Configure(
				apperr.WithStacktrace(tt.configureWith),
			)

			if got := apperr.IsStacktraceEnabled(); got != tt.wantStacktrace {
				t.Errorf("Configure with WithStacktrace: got %v, want %v", got, tt.wantStacktrace)
			}
		})
	}
}

func TestStacktraceThreadSafety(t *testing.T) {
	t.Parallel()

	// Save original setting
	originalSetting := apperr.IsStacktraceEnabled()
	defer apperr.Configure(apperr.WithStacktrace(originalSetting))

	// Run concurrent reads and writes
	done := make(chan bool)

	// Writer goroutines
	for i := range 100 {
		go func(val bool) {
			apperr.Configure(apperr.WithStacktrace(val))
			done <- true
		}(i%2 == 0)
	}

	// Reader goroutines
	for range 100 {
		go func() {
			_ = apperr.IsStacktraceEnabled()
			done <- true
		}()
	}

	// Wait for all goroutines
	for range 200 {
		<-done
	}

	// If we get here without race conditions, the test passes
	// No assertion needed - test will fail with race detector if there are issues
}

func TestStacktraceInLogStructure(t *testing.T) {
	// Don't run in parallel - modifies global state
	// Save original setting
	originalSetting := apperr.IsStacktraceEnabled()
	defer apperr.Configure(apperr.WithStacktrace(originalSetting))

	t.Run("with stacktrace enabled", func(t *testing.T) {
		apperr.Configure(apperr.WithStacktrace(true))

		err := apperr.New(codes.Internal, "test error")

		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, nil))
		logger.Error("test", slog.Any("error", err))

		logOutput := buf.String()
		// Simply check if stacktrace appears in the output
		if !strings.Contains(logOutput, "stacktrace") {
			t.Error("stacktrace should be present when enabled")
		}
	})

	t.Run("with stacktrace disabled", func(t *testing.T) {
		apperr.Configure(apperr.WithStacktrace(false))

		err := apperr.New(codes.Internal, "test error")

		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, nil))
		logger.Error("test", slog.Any("error", err))

		logOutput := buf.String()
		// Simply check if stacktrace does NOT appear in the output
		if strings.Contains(logOutput, "stacktrace") {
			t.Error("stacktrace should not be present when disabled")
		}
	})
}

func TestDefaultConfiguration(t *testing.T) {
	// This test should run in isolation to verify default behavior
	// Create a new test binary instance to ensure clean state

	// Note: We can't easily test the true default in a parallel test environment
	// but we can document the expected behavior
	t.Run("document default behavior", func(_ *testing.T) {
		// Default should be false for production safety
		// This is set in init() function of config.go
		// Documentation test - no assertion needed
	})
}

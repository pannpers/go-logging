package logging_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"github.com/pannpers/go-logging/logging"
)

// TestNewAccessLogInterceptor_WithHTTPHeaders tests the WithHTTPHeaders option.
func TestNewAccessLogInterceptor_WithHTTPHeaders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		httpHeaders     []string
		requestHeaders  map[string]string
		expectedHeaders map[string]string
	}{
		{
			name:        "log custom headers",
			httpHeaders: []string{"Content-Type", "Authorization", "X-Custom-Header"},
			requestHeaders: map[string]string{
				"User-Agent":      "test-agent",
				"Content-Type":    "application/json",
				"Authorization":   "Bearer token123",
				"X-Custom-Header": "custom-value",
				"Ignored-Header":  "should-not-appear",
			},
			expectedHeaders: map[string]string{
				"user_agent":      "test-agent",
				"remote_addr":     "",
				"method":          "POST",
				"content_type":    "application/json",
				"authorization":   "Bearer token123",
				"x_custom_header": "custom-value",
			},
		},
		{
			name:        "no custom headers configured",
			httpHeaders: nil,
			requestHeaders: map[string]string{
				"User-Agent":   "test-agent",
				"Content-Type": "application/json",
			},
			expectedHeaders: map[string]string{
				"user_agent":  "test-agent",
				"remote_addr": "",
				"method":      "POST",
			},
		},
		{
			name:        "missing headers in request",
			httpHeaders: []string{"Content-Type", "Authorization"},
			requestHeaders: map[string]string{
				"User-Agent": "test-agent",
				// Content-Type and Authorization are missing
			},
			expectedHeaders: map[string]string{
				"user_agent":  "test-agent",
				"remote_addr": "",
				"method":      "POST",
			},
		},
		{
			name:        "partial headers present",
			httpHeaders: []string{"Content-Type", "Authorization", "X-Missing"},
			requestHeaders: map[string]string{
				"User-Agent":    "test-agent",
				"Content-Type":  "text/plain",
				"Authorization": "Basic xyz",
				// X-Missing is not present
			},
			expectedHeaders: map[string]string{
				"user_agent":    "test-agent",
				"remote_addr":   "",
				"method":        "POST",
				"content_type":  "text/plain",
				"authorization": "Basic xyz",
			},
		},
		{
			name:        "header name conversion",
			httpHeaders: []string{"X-Forwarded-For", "Content-Length", "Accept-Encoding"},
			requestHeaders: map[string]string{
				"User-Agent":      "test-agent",
				"X-Forwarded-For": "192.168.1.1",
				"Content-Length":  "1024",
				"Accept-Encoding": "gzip, deflate",
			},
			expectedHeaders: map[string]string{
				"user_agent":      "test-agent",
				"remote_addr":     "192.168.1.1",
				"method":          "POST",
				"x_forwarded_for": "192.168.1.1",
				"content_length":  "1024",
				"accept_encoding": "gzip, deflate",
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
			)
			if err != nil {
				t.Fatalf("failed to create logger: %v", err)
			}

			// Create interceptor with HTTP headers option
			var interceptor connect.UnaryInterceptorFunc
			if tc.httpHeaders != nil {
				interceptor = logging.NewAccessLogInterceptor(logger,
					logging.WithHTTPHeaders(tc.httpHeaders),
				)
			} else {
				interceptor = logging.NewAccessLogInterceptor(logger)
			}

			// Create mock request with headers
			req := connect.NewRequest(&mockMessage{Value: "test"})
			for key, value := range tc.requestHeaders {
				req.Header().Set(key, value)
			}

			mockReq := &mockRequestWithProcedure{
				Request:   req,
				procedure: "/test.Service/TestMethod",
			}

			// Mock next function
			next := func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
				return connect.NewResponse(&mockMessage{Value: "response"}), nil
			}

			// Execute interceptor
			handler := interceptor(next)
			_, err = handler(context.Background(), mockReq)
			if err != nil {
				t.Fatalf("interceptor failed: %v", err)
			}

			// Parse log output
			logOutput := strings.TrimSpace(buf.String())
			if logOutput == "" {
				t.Fatalf("expected log output but got empty")
			}

			var logData map[string]any
			if err := json.Unmarshal([]byte(logOutput), &logData); err != nil {
				t.Fatalf("failed to parse log JSON: %v\nOutput: %s", err, logOutput)
			}

			// Check if headers group is present
			headersGroup, hasHeadersGroup := logData["headers"]

			// Headers group should always be present now (includes standard headers)
			if !hasHeadersGroup {
				t.Fatalf("expected headers group but not found in log output: %s", logOutput)
			}

			headersMap, ok := headersGroup.(map[string]any)
			if !ok {
				t.Fatalf("expected headers to be a map, got %T: %v", headersGroup, headersGroup)
			}

			// Verify each expected header
			for expectedKey, expectedValue := range tc.expectedHeaders {
				actualValue, exists := headersMap[expectedKey]
				if !exists {
					t.Errorf("expected header %q not found in headers group", expectedKey)
					continue
				}
				if actualValue != expectedValue {
					t.Errorf("header %q: expected %q, got %q", expectedKey, expectedValue, actualValue)
				}
			}

			// Verify no unexpected headers
			for actualKey := range headersMap {
				if _, expected := tc.expectedHeaders[actualKey]; !expected {
					t.Errorf("unexpected header %q found in headers group", actualKey)
				}
			}

			// Verify standard log fields are still present (excluding header fields which are now in headers group)
			standardFields := []string{"level", "msg", "rpc", "status", "duration_ms"}
			for _, field := range standardFields {
				if _, exists := logData[field]; !exists {
					t.Errorf("standard field %q missing from log output", field)
				}
			}

			// Verify headers group contains standard header fields
			if headersGroup, exists := logData["headers"]; exists {
				if headersMap, ok := headersGroup.(map[string]any); ok {
					standardHeaderFields := []string{"user_agent", "remote_addr", "method"}
					for _, field := range standardHeaderFields {
						if _, exists := headersMap[field]; !exists {
							t.Errorf("standard header field %q missing from headers group", field)
						}
					}
				}
			}
		})
	}
}

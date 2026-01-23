package logging_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/pannpers/go-logging/logging"
)

// mockMessage represents a simple message for testing.
type mockMessage struct {
	Value string `json:"value"`
}

// mockRequestWithProcedure wraps a Connect request to override the procedure.
type mockRequestWithProcedure struct {
	*connect.Request[mockMessage]
	procedure string
}

func (m *mockRequestWithProcedure) Spec() connect.Spec {
	spec := m.Request.Spec()
	spec.Procedure = m.procedure
	return spec
}

// TestNewAccessLogInterceptor tests the access log interceptor functionality.
func TestNewAccessLogInterceptor(t *testing.T) {
	t.Parallel()

	// Create error instances to reuse for proper error assertion
	connectErr := connect.NewError(connect.CodeInvalidArgument, errors.New("invalid user data"))
	unknownErr := errors.New("unexpected error")

	type args struct {
		procedure string
		headers   map[string]string
		err       error
	}

	tests := []struct {
		name       string
		args       args
		wantErr    error
		wantStatus string
	}{
		{
			name: "return success log when request succeeds",
			args: args{
				procedure: "/api.UserService/GetUser",
				headers: map[string]string{
					"User-Agent":      "connect-go/1.18.1 (go1.21.0)",
					"X-Forwarded-For": "192.168.1.100",
					"X-Http-Method":   "POST",
				},
				err: nil,
			},
			wantErr:    nil,
			wantStatus: "ok",
		},
		{
			name: "return error log when request fails with connect error",
			args: args{
				procedure: "/api.UserService/CreateUser",
				headers: map[string]string{
					"User-Agent":    "buf/1.55.1",
					"X-Real-IP":     "10.0.0.1",
					"X-Http-Method": "POST",
				},
				err: connectErr,
			},
			wantErr:    connectErr,
			wantStatus: "invalid_argument",
		},
		{
			name: "return unknown error log when request fails with unknown error",
			args: args{
				procedure: "/api.PostService/GetPost",
				headers: map[string]string{
					"User-Agent": "test-client/1.0",
				},
				err: unknownErr,
			},
			wantErr:    unknownErr,
			wantStatus: "unknown",
		},
		{
			name: "return log with empty headers when no headers provided",
			args: args{
				procedure: "/api.UserService/ListUsers",
				headers:   map[string]string{},
				err:       nil,
			},
			wantErr:    nil,
			wantStatus: "ok",
		},
		{
			name: "return log with X-Real-IP when X-Forwarded-For is not present",
			args: args{
				procedure: "/api.UserService/GetUser",
				headers: map[string]string{
					"User-Agent": "connect-go/1.18.1",
					"X-Real-IP":  "172.16.0.1",
				},
				err: nil,
			},
			wantErr:    nil,
			wantStatus: "ok",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer

			// Create logger with buffer output and without time/duration for consistent testing
			logger, err := logging.New(
				logging.WithLevel(slog.LevelInfo),
				logging.WithFormat(logging.FormatJSON),
				logging.WithWriter(&buf),
			)
			if err != nil {
				t.Fatalf("failed to create logger: %v", err)
			}

			// Create access log interceptor
			interceptor := logging.NewAccessLogInterceptor(logger)

			// Create Connect request with message
			req := connect.NewRequest(&mockMessage{Value: "test"})

			// Set headers
			for key, value := range tc.args.headers {
				req.Header().Set(key, value)
			}

			// Create a mock request that returns the desired procedure
			mockReq := &mockRequestWithProcedure{
				Request:   req,
				procedure: tc.args.procedure,
			}

			// Create mock next function
			next := func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
				if tc.args.err != nil {
					return nil, tc.args.err
				}
				return connect.NewResponse(&mockMessage{Value: "response"}), nil
			}

			// Execute interceptor
			handler := interceptor(next)
			resp, err := handler(context.Background(), mockReq)

			// Verify error handling
			if tc.wantErr != nil {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				if !errors.Is(err, tc.wantErr) {
					t.Errorf("expected error %v, got %v", tc.wantErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
				}
				if resp == nil {
					t.Errorf("expected response but got nil")
				}
			}

			// Verify log output using JSONEq
			logOutput := strings.TrimSpace(buf.String())
			if logOutput == "" {
				t.Errorf("expected log output but got empty")
			}

			// Extract expected values for JSON comparison
			expectedUserAgent := tc.args.headers["User-Agent"]
			expectedRemoteAddr := tc.args.headers["X-Forwarded-For"]
			if expectedRemoteAddr == "" {
				expectedRemoteAddr = tc.args.headers["X-Real-IP"]
			}
			expectedMethod := tc.args.headers["X-Http-Method"]
			if expectedMethod == "" {
				expectedMethod = "POST"
			}

			// Parse actual log output and verify specific fields
			var logData map[string]any
			if err := json.Unmarshal([]byte(logOutput), &logData); err != nil {
				t.Fatalf("failed to parse log JSON: %v\nOutput: %s", err, logOutput)
			}

			// Verify standard fields
			if logData["level"] != "INFO" {
				t.Errorf("expected level INFO, got %v", logData["level"])
			}
			if logData["msg"] != "access log" {
				t.Errorf("expected msg 'access log', got %v", logData["msg"])
			}
			if logData["rpc"] != tc.args.procedure {
				t.Errorf("expected rpc %q, got %v", tc.args.procedure, logData["rpc"])
			}
			if logData["status"] != tc.wantStatus {
				t.Errorf("expected status %q, got %v", tc.wantStatus, logData["status"])
			}

			// Verify duration_ms exists and is a number
			if durationMs, exists := logData["duration_ms"]; !exists {
				t.Errorf("expected duration_ms field")
			} else if _, ok := durationMs.(float64); !ok {
				t.Errorf("expected duration_ms to be a number, got %T", durationMs)
			}

			// Verify headers group
			headersGroup, exists := logData["headers"]
			if !exists {
				t.Fatalf("expected headers group")
			}

			headersMap, ok := headersGroup.(map[string]any)
			if !ok {
				t.Fatalf("expected headers to be a map, got %T", headersGroup)
			}

			if headersMap["user_agent"] != expectedUserAgent {
				t.Errorf("expected user_agent %q, got %v", expectedUserAgent, headersMap["user_agent"])
			}
			if headersMap["remote_addr"] != expectedRemoteAddr {
				t.Errorf("expected remote_addr %q, got %v", expectedRemoteAddr, headersMap["remote_addr"])
			}
			if headersMap["method"] != expectedMethod {
				t.Errorf("expected method %q, got %v", expectedMethod, headersMap["method"])
			}
		})
	}
}

// TestAccessLogInterceptor_HeaderExtraction tests specific header extraction scenarios.
func TestAccessLogInterceptor_HeaderExtraction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		headers            map[string]string
		expectedUserAgent  string
		expectedRemoteAddr string
		expectedMethod     string
	}{
		{
			name: "extract X-Forwarded-For when both headers present",
			headers: map[string]string{
				"X-Forwarded-For": "192.168.1.100",
				"X-Real-IP":       "10.0.0.1",
				"User-Agent":      "test-client/1.0",
			},
			expectedUserAgent:  "test-client/1.0",
			expectedRemoteAddr: "192.168.1.100",
			expectedMethod:     "POST",
		},
		{
			name: "extract X-Real-IP when X-Forwarded-For is empty",
			headers: map[string]string{
				"X-Real-IP":     "10.0.0.1",
				"User-Agent":    "buf/1.55.1",
				"X-Http-Method": "PUT",
			},
			expectedUserAgent:  "buf/1.55.1",
			expectedRemoteAddr: "10.0.0.1",
			expectedMethod:     "PUT",
		},
		{
			name: "use default method when X-Http-Method is not present",
			headers: map[string]string{
				"User-Agent": "connect-go/1.18.1",
			},
			expectedUserAgent:  "connect-go/1.18.1",
			expectedRemoteAddr: "",
			expectedMethod:     "POST",
		},
		{
			name:               "handle empty headers",
			headers:            map[string]string{},
			expectedUserAgent:  "",
			expectedRemoteAddr: "",
			expectedMethod:     "POST",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer

			logger, err := logging.New(
				logging.WithLevel(slog.LevelInfo),
				logging.WithFormat(logging.FormatJSON),
				logging.WithWriter(&buf),
			)
			if err != nil {
				t.Fatalf("failed to create logger: %v", err)
			}

			interceptor := logging.NewAccessLogInterceptor(logger)

			// Create Connect request
			req := connect.NewRequest(&mockMessage{Value: "test"})

			// Set headers
			for key, value := range tc.headers {
				req.Header().Set(key, value)
			}

			// Create a mock request that returns the desired procedure
			mockReq := &mockRequestWithProcedure{
				Request:   req,
				procedure: "/api.UserService/GetUser",
			}

			next := func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
				return connect.NewResponse(&mockMessage{Value: "response"}), nil
			}

			handler := interceptor(next)
			_, err = handler(context.Background(), mockReq)
			if err != nil {
				t.Errorf("expected no error but got: %v", err)
			}

			logOutput := strings.TrimSpace(buf.String())
			if logOutput == "" {
				t.Errorf("expected log output but got empty")
			}

			// Parse actual log output and verify header extraction
			var logData map[string]any
			if err := json.Unmarshal([]byte(logOutput), &logData); err != nil {
				t.Fatalf("failed to parse log JSON: %v\nOutput: %s", err, logOutput)
			}

			// Verify standard fields
			if logData["level"] != "INFO" {
				t.Errorf("expected level INFO, got %v", logData["level"])
			}
			if logData["msg"] != "access log" {
				t.Errorf("expected msg 'access log', got %v", logData["msg"])
			}
			if logData["rpc"] != "/api.UserService/GetUser" {
				t.Errorf("expected rpc '/api.UserService/GetUser', got %v", logData["rpc"])
			}
			if logData["status"] != "ok" {
				t.Errorf("expected status 'ok', got %v", logData["status"])
			}

			// Verify headers group
			headersGroup, exists := logData["headers"]
			if !exists {
				t.Fatalf("expected headers group")
			}

			headersMap, ok := headersGroup.(map[string]any)
			if !ok {
				t.Fatalf("expected headers to be a map, got %T", headersGroup)
			}

			if headersMap["user_agent"] != tc.expectedUserAgent {
				t.Errorf("expected user_agent %q, got %v", tc.expectedUserAgent, headersMap["user_agent"])
			}
			if headersMap["remote_addr"] != tc.expectedRemoteAddr {
				t.Errorf("expected remote_addr %q, got %v", tc.expectedRemoteAddr, headersMap["remote_addr"])
			}
			if headersMap["method"] != tc.expectedMethod {
				t.Errorf("expected method %q, got %v", tc.expectedMethod, headersMap["method"])
			}
		})
	}
}

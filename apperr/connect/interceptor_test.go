package connect

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/pannpers/go-apperr/apperr"
	"github.com/pannpers/go-apperr/apperr/codes"
)

func TestNewErrorHandlingInterceptor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		err             error
		wantLogContains []string
		wantNoLog       bool
		wantCode        connect.Code
		wantErrMsg      string // expected error message for verification
	}{
		{
			name:            "log server error with RPC method when Internal error occurs",
			err:             apperr.New(codes.Internal, "database connection failed"),
			wantLogContains: []string{"server error occurred", "rpc_method", "/TestService/TestMethod"},
			wantCode:        connect.CodeInternal,
			wantErrMsg:      "internal server error", // generic message for security
		},
		{
			name:            "log server error with RPC method when Unknown error occurs",
			err:             apperr.New(codes.Unknown, "unknown error"),
			wantLogContains: []string{"server error occurred", "rpc_method", "/TestService/TestMethod"},
			wantCode:        connect.CodeUnknown,
			wantErrMsg:      "internal server error", // generic message for security
		},
		{
			name:            "log server error with RPC method when Unavailable error occurs",
			err:             apperr.New(codes.Unavailable, "service unavailable"),
			wantLogContains: []string{"server error occurred", "rpc_method", "/TestService/TestMethod"},
			wantCode:        connect.CodeUnavailable,
			wantErrMsg:      "internal server error", // generic message for security
		},
		{
			name:            "log server error with RPC method when DeadlineExceeded error occurs",
			err:             apperr.New(codes.DeadlineExceeded, "request timeout"),
			wantLogContains: []string{"server error occurred", "rpc_method", "/TestService/TestMethod"},
			wantCode:        connect.CodeDeadlineExceeded,
			wantErrMsg:      "internal server error", // generic message for security
		},
		{
			name:       "not log client error when InvalidArgument error occurs",
			err:        apperr.New(codes.InvalidArgument, "invalid input"),
			wantNoLog:  true,
			wantCode:   connect.CodeInvalidArgument,
			wantErrMsg: "invalid input (invalid_argument)", // Connect appends error code
		},
		{
			name:       "not log client error when NotFound error occurs",
			err:        apperr.New(codes.NotFound, "resource not found"),
			wantNoLog:  true,
			wantCode:   connect.CodeNotFound,
			wantErrMsg: "resource not found (not_found)", // Connect appends error code
		},
		{
			name:       "not log client error when Unauthenticated error occurs",
			err:        apperr.New(codes.Unauthenticated, "unauthorized"),
			wantNoLog:  true,
			wantCode:   connect.CodeUnauthenticated,
			wantErrMsg: "unauthorized (unauthenticated)", // Connect appends error code
		},
		{
			name:            "log unhandled error with RPC method when non-AppErr error occurs",
			err:             errors.New("unexpected error"),
			wantLogContains: []string{"unhandled error occurred", "rpc_method", "/TestService/TestMethod"},
			wantCode:        connect.CodeUnknown,
			wantErrMsg:      "internal server error", // generic message for security
		},
		{
			name:       "return nil when no error occurs",
			err:        nil,
			wantNoLog:  true,
			wantCode:   0,
			wantErrMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a buffer to capture log output
			var buf bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			}))

			// Create interceptor
			interceptor := NewErrorHandlingInterceptor(logger)

			// Create a mock unary func that returns the test error
			mockUnary := func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
				return nil, tt.err
			}

			// Apply the interceptor
			wrapped := interceptor(mockUnary)

			// Create request with the test procedure
			ctx := context.Background()
			req := newTestRequest("/TestService/TestMethod")

			// Execute the wrapped handler
			_, err := wrapped(ctx, req)

			// Check error code and message
			if tt.err == nil {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			} else {
				var connectErr *connect.Error
				if !errors.As(err, &connectErr) {
					t.Fatalf("expected connect.Error, got: %T", err)
				}
				if connectErr.Code() != tt.wantCode {
					t.Errorf("error code = %v, want %v", connectErr.Code(), tt.wantCode)
				}

				// Verify error message
				if tt.wantErrMsg != "" && connectErr.Message() != tt.wantErrMsg {
					t.Errorf("error message = %q, want %q", connectErr.Message(), tt.wantErrMsg)
				}
			}

			// Check log output
			logOutput := buf.String()
			if tt.wantNoLog {
				if logOutput != "" {
					t.Errorf("expected no log output but got: %s", logOutput)
				}
			} else {
				for _, want := range tt.wantLogContains {
					if !strings.Contains(logOutput, want) {
						t.Errorf("log output should contain %q, got: %s", want, logOutput)
					}
				}
			}

			// Verify JSON structure contains rpc_method field when logging occurs
			if len(tt.wantLogContains) > 0 && strings.Contains(logOutput, "rpc_method") {
				// Basic validation that rpc_method is in the JSON
				if !strings.Contains(logOutput, `"rpc_method":"/TestService/TestMethod"`) {
					t.Error("log output should contain rpc_method in JSON format")
				}
			}
		})
	}
}

// testRequest wraps connect.Request to set a custom procedure
type testRequest struct {
	*connect.Request[mockMessage]
	procedure string
}

func newTestRequest(procedure string) *testRequest {
	req := connect.NewRequest(&mockMessage{})
	return &testRequest{
		Request:   req,
		procedure: procedure,
	}
}

func (r *testRequest) Spec() connect.Spec {
	return connect.Spec{
		Procedure: r.procedure,
	}
}

// mockMessage is a simple message type for testing
type mockMessage struct{}

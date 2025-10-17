package connect

import (
	"context"
	"errors"
	"log/slog"

	"connectrpc.com/connect"

	"github.com/pannpers/go-apperr/apperr"
)

// Logger defines the interface for logging operations.
// This allows for flexible logging implementations beyond slog.Logger.
type Logger interface {
	Error(msg string, args ...any)
}

// NewErrorHandlingInterceptor creates a Connect interceptor that handles AppErr conversion and logging.
//
// The interceptor provides the following functionality:
//   - Converts AppErr instances to appropriate Connect errors with proper error codes
//   - Automatically logs server errors (5xx) with full context while skipping client errors (4xx)
//   - Extracts and includes the RPC method name from the request in error logs
//   - Treats non-AppErr errors as Unknown errors and logs them
//
// Default Behavior:
// The interceptor automatically adds the RPC method name as a structured log attribute:
//
//	logger.Error("server error occurred", slog.Any("error", err), slog.String("rpc_method", "/service.v1.UserService/GetUser"))
//
// Error Classification:
//   - Server errors (logged): Internal, Unknown, DataLoss, Unimplemented, Unavailable, DeadlineExceeded
//   - Client errors (not logged): InvalidArgument, NotFound, AlreadyExists, PermissionDenied, etc.
//
// Basic Usage:
//
//	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
//	interceptor := connect.NewErrorHandlingInterceptor(logger)
//
//	server := connect.NewServer(
//	    connect.WithInterceptors(interceptor),
//	)
//
// Custom Attributes:
// For more advanced logging needs, you can create a custom interceptor using HandleError directly
// to add additional structured attributes:
//
//	func NewCustomErrorInterceptor(logger connect.Logger) connect.UnaryInterceptorFunc {
//	    return func(next connect.UnaryFunc) connect.UnaryFunc {
//	        return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
//	            resp, err := next(ctx, req)
//
//	            if err != nil {
//	                // Add custom attributes for logging
//	                attrs := []slog.Attr{
//	                    slog.String("rpc_method", req.Spec().Procedure),
//	                    slog.String("client_ip", req.Peer().Addr),
//	                    slog.String("user_agent", req.Header().Get("User-Agent")),
//	                    slog.Int("request_size", len(req.Any().([]byte))),
//	                }
//	                return resp, connect.HandleError(ctx, err, logger, attrs...)
//	            }
//
//	            return resp, nil
//	        }
//	    }
//	}
//
// Error Classification:
// You can also use HandleError directly for custom error handling logic:
//
//	// Check if error represents a server issue
//	if appErr.Code.IsServerError() {
//	    // Log with full context for debugging
//	    logger.Error("server error occurred", slog.Any("error", appErr))
//	} else {
//	    // Client error - safe to return to client
//	    return connect.NewError(appErr.Code.ToConnect(), appErr)
//	}
func NewErrorHandlingInterceptor(logger Logger) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			resp, err := next(ctx, req)
			if err != nil {
				rpcAttr := slog.String("rpc_method", req.Spec().Procedure)
				return resp, HandleError(ctx, err, logger, rpcAttr)
			}

			return resp, nil
		}
	}
}

// HandleError converts AppErr to Connect error and logs server errors with custom attributes.
//
// This function is the core error handling logic that can be used directly when creating
// custom interceptors with additional logging requirements.
//
// Error Handling:
//   - If err is nil, returns nil
//   - If err is not an AppErr, treats it as Unknown error and logs it
//   - If err is an AppErr, converts it to the appropriate Connect error code
//
// Security Behavior:
//   - Server errors (5xx): Returns a generic "internal server error" message to prevent
//     exposing implementation details while logging the full error internally
//   - Client errors (4xx): Returns the original error as these are safe to expose
//   - Non-AppErr errors: Returns a generic error message for security
//
// Logging Behavior:
//   - Server errors (5xx): Logged with ERROR level including all provided attributes
//   - Client errors (4xx): Not logged to avoid noise from user errors
//   - Non-AppErr errors: Always logged as they indicate unexpected errors
//
// Parameters:
//   - ctx: Context for the request, used for trace ID extraction if available
//   - err: The error to handle, can be AppErr or any other error type
//   - logger: Logger instance for error logging
//   - attrs: Variable number of slog.Attr for structured logging (e.g., rpc_method, client_ip)
//
// Returns:
//   - nil if err is nil
//   - *connect.Error with appropriate code and generic message for server errors
//   - *connect.Error with appropriate code and original error for client errors
//
// Example with custom attributes:
//
//	attrs := []slog.Attr{
//	    slog.String("rpc_method", "/service.v1.UserService/GetUser"),
//	    slog.Duration("latency", time.Since(start)),
//	}
//	err := connect.HandleError(ctx, err, logger, attrs...)
//
// Server Error Detection:
// The following codes are considered server errors and will be logged:
//   - Internal (500): Internal server error
//   - Unknown (500): Unknown error, often from non-AppErr
//   - DataLoss (500): Unrecoverable data loss
//   - Unimplemented (501): Operation not implemented
//   - Unavailable (503): Service temporarily unavailable
//   - DeadlineExceeded (504): Request timeout
func HandleError(ctx context.Context, err error, logger Logger, attrs ...slog.Attr) error {
	if err == nil {
		return nil
	}

	var appErr *apperr.AppErr
	if !errors.As(err, &appErr) {
		// For non-AppErr errors, treat as unknown error
		logger.Error("unhandled error occurred", slog.Any("error", err), slog.Group("attrs", convertAttrs(attrs)...))
		return connect.NewError(connect.CodeUnknown, errInternal)
	}

	if appErr.Code.IsServerError() {
		logger.Error("server error occurred", slog.Any("error", appErr), slog.Group("attrs", convertAttrs(attrs)...))
		return connect.NewError(appErr.Code.ToConnect(), errInternal)
	}

	return connect.NewError(appErr.Code.ToConnect(), appErr)
}

// convertAttrs converts []slog.Attr to []any for use with slog.Group.
func convertAttrs(attrs []slog.Attr) []any {
	result := make([]any, len(attrs))
	for i, attr := range attrs {
		result[i] = attr
	}
	return result
}

// errInternal is a generic error message returned to clients for server errors.
// This prevents exposing sensitive implementation details, stack traces, or
// database errors that could be exploited by malicious users.
// The actual error details are logged server-side for debugging purposes.
var errInternal = errors.New("internal server error")

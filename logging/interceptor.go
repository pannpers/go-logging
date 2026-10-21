package logging

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"connectrpc.com/connect"

	"github.com/pannpers/go-logging/logging/internal/accesslog"
)

// InterceptorOption configures the access log interceptor.
// The same options apply to the connect-go v1 interceptor in this package and
// the connect-go v2 interceptor in package connectv2.
type InterceptorOption func(*interceptorConfig)

// interceptorConfig holds configuration for the access log interceptor.
// It is shared with package connectv2 through the internal accesslog package.
type interceptorConfig = accesslog.Config

// WithHTTPHeaders configures additional HTTP headers to include in access logs.
// Header names are case-insensitive. Header values are logged as a grouped
// attribute under the "headers" key.
//
// Example:
//
//	interceptor := logging.NewAccessLogInterceptor(logger,
//		logging.WithHTTPHeaders([]string{"Content-Type", "Accept", "Referer"}),
//	)
//
// This will log a "headers" group containing:
//
//	{
//	  "headers": {
//	    "content_type": "application/json",
//	    "accept": "application/json",
//	    "referer": "https://example.com/page"
//	  }
//	}
func WithHTTPHeaders(headers []string) InterceptorOption {
	return func(c *interceptorConfig) {
		c.HTTPHeaders = headers
	}
}

// NewAccessLogInterceptor creates a Connect RPC interceptor that logs access information for all requests.
// It automatically logs essential request metadata for monitoring, debugging, and audit purposes.
// If logger is nil, the function will panic.
//
// The interceptor logs the following attributes for each request:
//   - rpc: The RPC procedure name (e.g., "/api.UserService/GetUser")
//   - status: Response status ("ok" for success, Connect error codes for failures)
//   - duration_ms: Request duration in milliseconds as an integer
//   - headers: Grouped attribute containing all HTTP headers
//   - user_agent: Client User-Agent header value
//   - remote_addr: Client IP address from X-Forwarded-For or X-Real-IP headers
//   - method: HTTP method from X-Http-Method header (defaults to "POST")
//   - Additional custom headers if configured with WithHTTPHeaders
//
// Usage with Connect server:
//
//	logger, _ := logging.New(logging.WithFormat(logging.FormatJSON))
//	interceptor := logging.NewAccessLogInterceptor(logger,
//		logging.WithHTTPHeaders([]string{"Content-Type", "Referer"}),
//	)
//
//	path, handler := userServiceConnect.NewUserServiceHandler(
//		&userService{},
//		connect.WithInterceptors(interceptor),
//	)
//	mux.Handle(path, handler)
//
// The interceptor integrates with the logging package's context attribute system,
// so any attributes added to the request context via SetAttrs will also be included
// in the access logs, along with OpenTelemetry trace information if available.
func NewAccessLogInterceptor(logger *Logger, opts ...InterceptorOption) connect.UnaryInterceptorFunc {
	if logger == nil {
		panic("logger cannot be nil")
	}
	var config accesslog.Config
	for _, opt := range opts {
		opt(&config)
	}
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			start := time.Now()
			rpc := req.Spec().Procedure

			var headersGroup slog.Attr
			if header := req.Header(); header != nil {
				headersGroup = accesslog.HeadersGroup(header, config.HTTPHeaders)
			}

			resp, err := next(ctx, req)

			duration := time.Since(start)

			// Determine status from error
			status := "ok"
			if err != nil {
				var connectErr *connect.Error
				if errors.As(err, &connectErr) {
					status = connectErr.Code().String()
				} else {
					status = "unknown"
				}
			}

			logger.Info(ctx, accesslog.Message, accesslog.Attrs(rpc, status, duration, headersGroup)...)

			return resp, err
		}
	}
}

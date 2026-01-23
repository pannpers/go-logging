package logging

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
)

// InterceptorOption configures the access log interceptor.
type InterceptorOption func(*interceptorConfig)

// interceptorConfig holds configuration for the access log interceptor.
// This struct is internal and populated by applying InterceptorOption functions.
type interceptorConfig struct {
	httpHeaders []string
}

// defaultInterceptorConfig returns the default interceptor configuration with no custom headers.
func defaultInterceptorConfig() *interceptorConfig {
	return &interceptorConfig{
		httpHeaders: nil,
	}
}

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
		c.httpHeaders = headers
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
	config := defaultInterceptorConfig()
	for _, opt := range opts {
		opt(config)
	}
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			start := time.Now()
			rpc := req.Spec().Procedure

			// Extract request information
			var headersGroup slog.Attr

			if header := req.Header(); header != nil {
				headerAttrs := extractStandardHeaders(header)
				headerAttrs = append(headerAttrs, extractCustomHeaders(header, config.httpHeaders)...)

				anyHeaderAttrs := make([]any, len(headerAttrs))
				for i, attr := range headerAttrs {
					anyHeaderAttrs[i] = attr
				}

				headersGroup = slog.Group("headers", anyHeaderAttrs...)
			}

			resp, err := next(ctx, req)

			durationMs := time.Since(start).Milliseconds()

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

			logAttrs := []slog.Attr{
				slog.String("rpc", rpc),
				slog.String("status", status),
				slog.Int64("duration_ms", durationMs),
			}

			if headersGroup.Key != "" {
				logAttrs = append(logAttrs, headersGroup)
			}

			logger.Info(ctx, "access log", logAttrs...)

			return resp, err
		}
	}
}

// extractStandardHeaders extracts standard HTTP headers used for access logging.
// It extracts User-Agent, remote address (X-Forwarded-For or X-Real-IP), and HTTP method.
// Returns a slice of slog.Attr with user_agent, remote_addr, and method attributes.
// Empty header values are included as empty strings in the returned attributes.
func extractStandardHeaders(header http.Header) []slog.Attr {
	userAgent := header.Get("User-Agent")

	// Try X-Forwarded-For first, then fall back to X-Real-IP
	remoteAddr := header.Get("X-Forwarded-For")
	if remoteAddr != "" {
		// X-Forwarded-For may contain multiple IPs (comma-separated)
		// Use the first IP which is the original client
		if idx := strings.Index(remoteAddr, ","); idx != -1 {
			remoteAddr = strings.TrimSpace(remoteAddr[:idx])
		}
	} else {
		remoteAddr = header.Get("X-Real-IP")
	}

	// Try X-Http-Method header, default to POST for Connect
	method := header.Get("X-Http-Method")
	if method == "" {
		method = http.MethodPost // Connect uses POST by default
	}

	return []slog.Attr{
		slog.String("user_agent", userAgent),
		slog.String("remote_addr", remoteAddr),
		slog.String("method", method),
	}
}

// extractCustomHeaders extracts custom HTTP headers specified in headerNames and returns them as slog attributes.
// Header names are matched case-insensitively and converted to lowercase with underscores for attribute keys.
// Only headers with non-empty values are included in the result.
// Returns an empty slice if no headerNames are provided or no matching headers are found.
func extractCustomHeaders(header http.Header, headerNames []string) []slog.Attr {
	if len(headerNames) == 0 {
		return []slog.Attr{} // Return empty slice
	}

	var headerAttrs []slog.Attr
	for _, headerName := range headerNames {
		if headerValue := header.Get(headerName); headerValue != "" {
			// Convert header name to attribute key (lowercase with underscores)
			attrKey := strings.ToLower(strings.ReplaceAll(headerName, "-", "_"))
			headerAttrs = append(headerAttrs, slog.String(attrKey, headerValue))
		}
	}

	return headerAttrs
}

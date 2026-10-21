// Package accesslog builds the attributes of the "access log" record shared by
// the connect-go v1 and v2 access log interceptors, so both versions log the
// same fields under the same keys.
package accesslog

import (
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// Message is the log message of every access log record.
const Message = "access log"

// Config holds configuration for the access log interceptors.
type Config struct {
	// HTTPHeaders lists additional request headers to log in the "headers" group.
	HTTPHeaders []string
}

// HeaderGetter reads a request header value by case-insensitive name.
// Both net/http.Header (connect-go v1) and *connect.Header (connect-go v2)
// satisfy it.
type HeaderGetter interface {
	Get(key string) string
}

// Attrs returns the attributes of an access log record.
// headers is omitted when it is the zero slog.Attr.
func Attrs(rpc, status string, duration time.Duration, headers slog.Attr) []slog.Attr {
	attrs := []slog.Attr{
		slog.String("rpc", rpc),
		slog.String("status", status),
		slog.Int64("duration_ms", duration.Milliseconds()),
	}
	if headers.Key != "" {
		attrs = append(attrs, headers)
	}
	return attrs
}

// HeadersGroup returns the "headers" group attribute containing the standard
// headers followed by the custom headers listed in headerNames.
func HeadersGroup(header HeaderGetter, headerNames []string) slog.Attr {
	headerAttrs := extractStandardHeaders(header)
	headerAttrs = append(headerAttrs, extractCustomHeaders(header, headerNames)...)

	anyHeaderAttrs := make([]any, len(headerAttrs))
	for i, attr := range headerAttrs {
		anyHeaderAttrs[i] = attr
	}

	return slog.Group("headers", anyHeaderAttrs...)
}

// extractStandardHeaders extracts standard HTTP headers used for access logging.
// It extracts User-Agent, remote address (X-Forwarded-For or X-Real-IP), and HTTP method.
// Returns a slice of slog.Attr with user_agent, remote_addr, and method attributes.
// Empty header values are included as empty strings in the returned attributes.
func extractStandardHeaders(header HeaderGetter) []slog.Attr {
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
func extractCustomHeaders(header HeaderGetter, headerNames []string) []slog.Attr {
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

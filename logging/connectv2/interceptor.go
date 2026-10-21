// Package connectv2 provides the access log interceptor for connect-go v2
// (connectrpc.com/connect/v2) servers.
//
// It writes the same "access log" record as [logging.NewAccessLogInterceptor],
// the connect-go v1 interceptor, so services can migrate from v1 to v2, or run
// both, without changing their log schema.
package connectv2

import (
	"context"
	"log/slog"
	"time"

	"connectrpc.com/connect/v2"

	"github.com/pannpers/go-logging/logging"
	"github.com/pannpers/go-logging/logging/internal/accesslog"
)

// NewAccessLogInterceptor creates a connect-go v2 server interceptor that
// writes one "access log" record per RPC. If logger is nil, the function
// panics.
//
// The record has the same attributes as the one written by
// [logging.NewAccessLogInterceptor]:
//   - rpc: The RPC procedure name (e.g., "/api.UserService/GetUser")
//   - status: "ok" on success, the Connect error code for a [*connect.Error],
//     and "unknown" for any other error
//   - duration_ms: RPC duration in milliseconds as an integer
//   - headers: Grouped attribute with user_agent, remote_addr, method and the
//     headers configured with [logging.WithHTTPHeaders]
//
// The headers group is read from the [*connect.CallInfo] that [connect.Server]
// attaches to the handler context. It is omitted when the context carries no
// CallInfo, for example when a [connect.ServerFunc] is invoked directly.
//
// Differences from the connect-go v1 interceptor:
//   - Streaming RPCs are logged. A v2 server interceptor wraps every RPC type,
//     so client, server and bidi streams each produce one record, written when
//     the handler returns. duration_ms then covers the whole stream.
//   - connect-go v2 runs interceptors before decompression and unmarshaling,
//     so duration_ms also covers decoding the request, which the v1
//     interceptor did not measure.
//
// Usage with a connect-go v2 server:
//
//	logger, _ := logging.New(logging.WithFormat(logging.FormatJSON))
//	srv := connect.NewServer(
//		connectv2.NewAccessLogInterceptor(logger,
//			logging.WithHTTPHeaders([]string{"X-Request-Id"}),
//		),
//	)
//
// As with the v1 interceptor, attributes added to the context via
// [logging.SetAttrs] and OpenTelemetry trace information are included in the
// record.
func NewAccessLogInterceptor(logger *logging.Logger, opts ...logging.InterceptorOption) connect.ServerInterceptor {
	if logger == nil {
		panic("logger cannot be nil")
	}
	var config accesslog.Config
	for _, opt := range opts {
		opt(&config)
	}
	return func(next connect.ServerFunc) connect.ServerFunc {
		return func(ctx context.Context, spec connect.Spec, stream connect.ServerStream) error {
			start := time.Now()

			var headersGroup slog.Attr
			if info, ok := connect.CallInfoForServerContext(ctx); ok {
				headersGroup = accesslog.HeadersGroup(info.RequestHeader(), config.HTTPHeaders)
			}

			err := next(ctx, spec, stream)

			duration := time.Since(start)

			status := "ok"
			if err != nil {
				// CodeOf reports CodeUnknown for errors that do not wrap a *connect.Error.
				status = connect.CodeOf(err).String()
			}

			logger.Info(ctx, accesslog.Message, accesslog.Attrs(spec.Procedure, status, duration, headersGroup)...)

			return err
		}
	}
}

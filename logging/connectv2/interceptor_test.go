package connectv2_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	connectv1 "connectrpc.com/connect"
	"connectrpc.com/connect/v2"

	"github.com/pannpers/go-logging/logging"
	"github.com/pannpers/go-logging/logging/connectv2"
)

// fakeStream is a connect.ServerStream that yields requests and records responses.
type fakeStream struct {
	requests []string
	sent     []string
}

func (s *fakeStream) Receive(msg any) error {
	if len(s.requests) == 0 {
		return io.EOF
	}
	*msg.(*string) = s.requests[0]
	s.requests = s.requests[1:]
	return nil
}

func (s *fakeStream) SendHeaders() error { return nil }

func (s *fakeStream) Send(msg any) error {
	s.sent = append(s.sent, *msg.(*string))
	return nil
}

// v1Request overrides the procedure of a connect-go v1 request.
type v1Request struct {
	*connectv1.Request[string]
	procedure string
}

func (r *v1Request) Spec() connectv1.Spec {
	spec := r.Request.Spec()
	spec.Procedure = r.procedure
	return spec
}

// newJSONLogger returns a JSON logger writing to buf.
func newJSONLogger(t *testing.T, buf *bytes.Buffer) *logging.Logger {
	t.Helper()
	logger, err := logging.New(logging.WithFormat(logging.FormatJSON), logging.WithWriter(buf))
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	return logger
}

// parseRecords decodes the JSON log lines in buf. It checks that duration_ms
// is a number and removes it and time, which vary between runs.
func parseRecords(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var records []map[string]any
	for line := range strings.Lines(buf.String()) {
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("failed to parse log line %q: %v", line, err)
		}
		if _, ok := record["duration_ms"].(float64); !ok {
			t.Errorf("expected duration_ms to be a number, got %T", record["duration_ms"])
		}
		delete(record, "duration_ms")
		delete(record, "time")
		records = append(records, record)
	}
	return records
}

// TestNewAccessLogInterceptor checks that the v2 interceptor writes the same
// access log record as the v1 interceptor for the same RPC.
func TestNewAccessLogInterceptor(t *testing.T) {
	t.Parallel()

	connectErr := errors.New("invalid user data")
	plainErr := errors.New("unexpected error")

	type args struct {
		procedure   string
		headers     map[string]string
		httpHeaders []string
		v1Err       error
		v2Err       error
	}

	tests := []struct {
		name       string
		args       args
		wantRecord map[string]any
	}{
		{
			name: "log ok with standard and custom headers when the RPC succeeds",
			args: args{
				procedure: "/api.UserService/GetUser",
				headers: map[string]string{
					"User-Agent":      "connect-go/2.0.1 (go1.26.0)",
					"X-Forwarded-For": "192.168.1.100, 10.0.0.1",
					"X-Request-Id":    "req-123",
					"Ignored-Header":  "should-not-appear",
				},
				httpHeaders: []string{"X-Request-Id", "X-Missing"},
			},
			wantRecord: map[string]any{
				"level":  "INFO",
				"msg":    "access log",
				"rpc":    "/api.UserService/GetUser",
				"status": "ok",
				"headers": map[string]any{
					"user_agent":   "connect-go/2.0.1 (go1.26.0)",
					"remote_addr":  "192.168.1.100",
					"method":       "POST",
					"x_request_id": "req-123",
				},
			},
		},
		{
			name: "log the error code when the RPC fails with a connect error",
			args: args{
				procedure: "/api.UserService/CreateUser",
				headers: map[string]string{
					"User-Agent":    "buf/1.55.1",
					"X-Real-IP":     "10.0.0.1",
					"X-Http-Method": "GET",
				},
				v1Err: connectv1.NewError(connectv1.CodeInvalidArgument, connectErr),
				v2Err: connect.NewError(connect.CodeInvalidArgument, connectErr.Error()),
			},
			wantRecord: map[string]any{
				"level":  "INFO",
				"msg":    "access log",
				"rpc":    "/api.UserService/CreateUser",
				"status": "invalid_argument",
				"headers": map[string]any{
					"user_agent":  "buf/1.55.1",
					"remote_addr": "10.0.0.1",
					"method":      "GET",
				},
			},
		},
		{
			name: "log unknown when the RPC fails with a plain error",
			args: args{
				procedure: "/api.PostService/GetPost",
				headers:   map[string]string{},
				v1Err:     plainErr,
				v2Err:     plainErr,
			},
			wantRecord: map[string]any{
				"level":  "INFO",
				"msg":    "access log",
				"rpc":    "/api.PostService/GetPost",
				"status": "unknown",
				"headers": map[string]any{
					"user_agent":  "",
					"remote_addr": "",
					"method":      "POST",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// connect-go v1
			var v1Buf bytes.Buffer
			v1Interceptor := logging.NewAccessLogInterceptor(newJSONLogger(t, &v1Buf), logging.WithHTTPHeaders(tc.args.httpHeaders))
			req := connectv1.NewRequest(new(string))
			for k, v := range tc.args.headers {
				req.Header().Set(k, v)
			}
			v1Next := func(context.Context, connectv1.AnyRequest) (connectv1.AnyResponse, error) {
				if tc.args.v1Err != nil {
					return nil, tc.args.v1Err
				}
				return connectv1.NewResponse(new(string)), nil
			}
			_, v1Got := v1Interceptor(v1Next)(context.Background(), &v1Request{Request: req, procedure: tc.args.procedure})
			if !errors.Is(v1Got, tc.args.v1Err) {
				t.Errorf("v1: expected error %v, got %v", tc.args.v1Err, v1Got)
			}

			// connect-go v2
			var v2Buf bytes.Buffer
			srv := connect.NewServer(connectv2.NewAccessLogInterceptor(newJSONLogger(t, &v2Buf), logging.WithHTTPHeaders(tc.args.httpHeaders)))
			srv.Register(connect.Method{
				Spec: connect.Spec{Procedure: tc.args.procedure, StreamType: connect.StreamTypeUnary},
				Handler: func(context.Context, connect.Spec, connect.ServerStream) error {
					return tc.args.v2Err
				},
			})
			info := &connect.CallInfo{}
			for k, v := range tc.args.headers {
				info.RequestHeader().Set(k, v)
			}
			v2Got := srv.Call(context.Background(), tc.args.procedure, info, &fakeStream{})
			if !errors.Is(v2Got, tc.args.v2Err) {
				t.Errorf("v2: expected error %v, got %v", tc.args.v2Err, v2Got)
			}

			for version, buf := range map[string]*bytes.Buffer{"v1": &v1Buf, "v2": &v2Buf} {
				records := parseRecords(t, buf)
				if len(records) != 1 {
					t.Fatalf("%s: expected 1 log record, got %d: %v", version, len(records), records)
				}
				if !reflect.DeepEqual(records[0], tc.wantRecord) {
					t.Errorf("%s: unexpected record\n got: %v\nwant: %v", version, records[0], tc.wantRecord)
				}
			}
		})
	}
}

// TestNewAccessLogInterceptor_Streaming checks that a streaming RPC produces
// one record, written after the handler has finished the stream.
func TestNewAccessLogInterceptor_Streaming(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	srv := connect.NewServer(connectv2.NewAccessLogInterceptor(newJSONLogger(t, &buf)))
	srv.Register(connect.Method{
		Spec: connect.Spec{Procedure: "/api.ChatService/Chat", StreamType: connect.StreamTypeBidi},
		Handler: func(_ context.Context, _ connect.Spec, stream connect.ServerStream) error {
			for {
				var msg string
				if err := stream.Receive(&msg); errors.Is(err, io.EOF) {
					return nil
				} else if err != nil {
					return err
				}
				if buf.Len() != 0 {
					t.Error("expected no access log before the stream completes")
				}
				if err := stream.Send(&msg); err != nil {
					return err
				}
			}
		},
	})

	stream := &fakeStream{requests: []string{"a", "b", "c"}}
	info := &connect.CallInfo{}
	info.RequestHeader().Set("User-Agent", "test-client/1.0")
	if err := srv.Call(context.Background(), "/api.ChatService/Chat", info, stream); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(stream.sent) != 3 {
		t.Errorf("expected 3 responses, got %d", len(stream.sent))
	}
	records := parseRecords(t, &buf)
	if len(records) != 1 {
		t.Fatalf("expected 1 log record, got %d: %v", len(records), records)
	}
	want := map[string]any{
		"level":  "INFO",
		"msg":    "access log",
		"rpc":    "/api.ChatService/Chat",
		"status": "ok",
		"headers": map[string]any{
			"user_agent":  "test-client/1.0",
			"remote_addr": "",
			"method":      "POST",
		},
	}
	if !reflect.DeepEqual(records[0], want) {
		t.Errorf("unexpected record\n got: %v\nwant: %v", records[0], want)
	}
}

// TestNewAccessLogInterceptor_WithoutCallInfo checks that the headers group is
// omitted when the handler context carries no CallInfo.
func TestNewAccessLogInterceptor_WithoutCallInfo(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	next := func(context.Context, connect.Spec, connect.ServerStream) error { return nil }
	handler := connectv2.NewAccessLogInterceptor(newJSONLogger(t, &buf))(next)

	if err := handler(context.Background(), connect.Spec{Procedure: "/api.UserService/GetUser"}, &fakeStream{}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	records := parseRecords(t, &buf)
	if len(records) != 1 {
		t.Fatalf("expected 1 log record, got %d", len(records))
	}
	if _, ok := records[0]["headers"]; ok {
		t.Errorf("expected no headers group, got %v", records[0]["headers"])
	}
}

// TestNewAccessLogInterceptor_NilLogger checks that a nil logger panics.
func TestNewAccessLogInterceptor_NilLogger(t *testing.T) {
	t.Parallel()

	defer func() {
		if recover() == nil {
			t.Error("expected panic for nil logger")
		}
	}()
	connectv2.NewAccessLogInterceptor(nil)
}

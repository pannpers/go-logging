package codes_test

import (
	"testing"

	"connectrpc.com/connect"

	"github.com/pannpers/go-apperr/apperr/codes"
)

func TestCode_IsServerError(t *testing.T) {
	t.Parallel()

	type args struct {
		code codes.Code
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// Server errors (5xx)
		{
			name: "return true when code is Internal (HTTP 500)",
			args: args{code: codes.Internal},
			want: true,
		},
		{
			name: "return true when code is Unknown (HTTP 500)",
			args: args{code: codes.Unknown},
			want: true,
		},
		{
			name: "return true when code is DataLoss (HTTP 500)",
			args: args{code: codes.DataLoss},
			want: true,
		},
		{
			name: "return true when code is Unimplemented (HTTP 501)",
			args: args{code: codes.Unimplemented},
			want: true,
		},
		{
			name: "return true when code is Unavailable (HTTP 503)",
			args: args{code: codes.Unavailable},
			want: true,
		},
		{
			name: "return true when code is DeadlineExceeded (HTTP 504)",
			args: args{code: codes.DeadlineExceeded},
			want: true,
		},
		// Client errors (4xx)
		{
			name: "return false when code is InvalidArgument (HTTP 400)",
			args: args{code: codes.InvalidArgument},
			want: false,
		},
		{
			name: "return false when code is OutOfRange (HTTP 400)",
			args: args{code: codes.OutOfRange},
			want: false,
		},
		{
			name: "return false when code is Unauthenticated (HTTP 401)",
			args: args{code: codes.Unauthenticated},
			want: false,
		},
		{
			name: "return false when code is PermissionDenied (HTTP 403)",
			args: args{code: codes.PermissionDenied},
			want: false,
		},
		{
			name: "return false when code is NotFound (HTTP 404)",
			args: args{code: codes.NotFound},
			want: false,
		},
		{
			name: "return false when code is Canceled (HTTP 408)",
			args: args{code: codes.Canceled},
			want: false,
		},
		{
			name: "return false when code is AlreadyExists (HTTP 409)",
			args: args{code: codes.AlreadyExists},
			want: false,
		},
		{
			name: "return false when code is Aborted (HTTP 409)",
			args: args{code: codes.Aborted},
			want: false,
		},
		{
			name: "return false when code is FailedPrecondition (HTTP 412)",
			args: args{code: codes.FailedPrecondition},
			want: false,
		},
		{
			name: "return false when code is ResourceExhausted (HTTP 429)",
			args: args{code: codes.ResourceExhausted},
			want: false,
		},
		// Edge cases
		{
			name: "return false when code is undefined (not in switch cases)",
			args: args{code: codes.Code(999)},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.args.code.IsServerError()
			if got != tt.want {
				t.Errorf("IsServerError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCode_ToConnect(t *testing.T) {
	t.Parallel()

	type args struct {
		code codes.Code
	}
	tests := []struct {
		name string
		args args
		want connect.Code
	}{
		{
			name: "convert Canceled to connect.CodeCanceled",
			args: args{code: codes.Canceled},
			want: connect.CodeCanceled,
		},
		{
			name: "convert Unknown to connect.CodeUnknown",
			args: args{code: codes.Unknown},
			want: connect.CodeUnknown,
		},
		{
			name: "convert InvalidArgument to connect.CodeInvalidArgument",
			args: args{code: codes.InvalidArgument},
			want: connect.CodeInvalidArgument,
		},
		{
			name: "convert DeadlineExceeded to connect.CodeDeadlineExceeded",
			args: args{code: codes.DeadlineExceeded},
			want: connect.CodeDeadlineExceeded,
		},
		{
			name: "convert NotFound to connect.CodeNotFound",
			args: args{code: codes.NotFound},
			want: connect.CodeNotFound,
		},
		{
			name: "convert AlreadyExists to connect.CodeAlreadyExists",
			args: args{code: codes.AlreadyExists},
			want: connect.CodeAlreadyExists,
		},
		{
			name: "convert PermissionDenied to connect.CodePermissionDenied",
			args: args{code: codes.PermissionDenied},
			want: connect.CodePermissionDenied,
		},
		{
			name: "convert ResourceExhausted to connect.CodeResourceExhausted",
			args: args{code: codes.ResourceExhausted},
			want: connect.CodeResourceExhausted,
		},
		{
			name: "convert FailedPrecondition to connect.CodeFailedPrecondition",
			args: args{code: codes.FailedPrecondition},
			want: connect.CodeFailedPrecondition,
		},
		{
			name: "convert Aborted to connect.CodeAborted",
			args: args{code: codes.Aborted},
			want: connect.CodeAborted,
		},
		{
			name: "convert OutOfRange to connect.CodeOutOfRange",
			args: args{code: codes.OutOfRange},
			want: connect.CodeOutOfRange,
		},
		{
			name: "convert Unimplemented to connect.CodeUnimplemented",
			args: args{code: codes.Unimplemented},
			want: connect.CodeUnimplemented,
		},
		{
			name: "convert Internal to connect.CodeInternal",
			args: args{code: codes.Internal},
			want: connect.CodeInternal,
		},
		{
			name: "convert Unavailable to connect.CodeUnavailable",
			args: args{code: codes.Unavailable},
			want: connect.CodeUnavailable,
		},
		{
			name: "convert DataLoss to connect.CodeDataLoss",
			args: args{code: codes.DataLoss},
			want: connect.CodeDataLoss,
		},
		{
			name: "convert Unauthenticated to connect.CodeUnauthenticated",
			args: args{code: codes.Unauthenticated},
			want: connect.CodeUnauthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.args.code.ToConnect()
			if got != tt.want {
				t.Errorf("ToConnect() = %v, want %v", got, tt.want)
			}
		})
	}
}

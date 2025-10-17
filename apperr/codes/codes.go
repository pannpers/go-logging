// Package codes provides status codes compatible with gRPC and Connect-RPC protocols.
// These codes are used throughout the application for consistent error handling and
// API compatibility across different transport protocols.
//
// # Overview
//
// The codes are based on the gRPC status code specification and are compatible with
// both gRPC and Connect-RPC protocols. They provide semantic meaning to errors and
// enable consistent error handling across different layers of the application.
//
// # Usage
//
// Use these codes when creating AppErr instances:
//
//	err := apperr.New(codes.InvalidArgument, "user ID cannot be empty")
//	err = apperr.Wrap(dbErr, codes.Internal, "failed to get user")
//
// Check if an error is a server error:
//
//	if appErr.Code.IsServerError() {
//	    // Log server errors with full context
//	    logger.Error(ctx, "Server error occurred", appErr)
//	}
//
// # Code Categories
//
// Server errors (5xx equivalent) - These indicate server-side issues:
//   - Internal: Internal server error (HTTP 500)
//   - Unknown: Unknown server error (HTTP 500)
//   - DataLoss: Unrecoverable data loss (HTTP 500)
//   - Unimplemented: Operation not implemented (HTTP 501)
//   - Unavailable: Service temporarily unavailable (HTTP 503)
//   - DeadlineExceeded: Operation timed out (HTTP 504)
//
// Client errors (4xx equivalent) - These indicate client-side issues:
//   - InvalidArgument: Invalid request parameters (HTTP 400)
//   - OutOfRange: Operation attempted past valid range (HTTP 400)
//   - Unauthenticated: Invalid or missing authentication (HTTP 401)
//   - PermissionDenied: Insufficient permissions (HTTP 403)
//   - NotFound: Requested resource not found (HTTP 404)
//   - Canceled: Operation was canceled by client (HTTP 408)
//   - AlreadyExists: Resource already exists (HTTP 409)
//   - Aborted: Operation aborted due to concurrency (HTTP 409)
//   - FailedPrecondition: Operation precondition failed (HTTP 412)
//   - ResourceExhausted: Resources exhausted (HTTP 429)
//
// # Protocol Compatibility
//
// These codes map directly to:
//   - gRPC status codes (google.golang.org/grpc/codes)
//   - Connect-RPC codes (connectrpc.com/connect)
//   - HTTP status codes (approximate mapping shown above)
//
// This ensures consistent error handling regardless of the transport protocol used.
//
// # Server Error Detection
//
// Use the IsServerError() method to determine if an error represents a server-side
// issue that should be logged and investigated:
//
//	if code.IsServerError() {
//	    // This is a server error (5xx)
//	    // Log with full context for debugging
//	}
package codes

import (
	"connectrpc.com/connect"
)

// Code is a status code defined according to the [gRPC documentation].
// It provides semantic meaning to errors and enables consistent error handling
// across different transport protocols (gRPC, Connect-RPC, HTTP).
//
// [gRPC documentation]: https://github.com/grpc/grpc/blob/master/doc/statuscodes.md
// type Code connect.Code
type Code uint32

const (
	// Canceled indicates the operation was canceled (typically by the caller).
	//
	// The gRPC framework will generate this error code when cancellation
	// is requested.
	Canceled = Code(connect.CodeCanceled)

	// Unknown error. An example of where this error may be returned is
	// if a Status value received from another address space belongs to
	// an error-space that is not known in this address space. Also
	// errors raised by APIs that do not return enough error information
	// may be converted to this error.
	//
	// The gRPC framework will generate this error code in the above two
	// mentioned cases.
	Unknown = Code(connect.CodeUnknown)

	// InvalidArgument indicates client specified an invalid argument.
	// Note that this differs from FailedPrecondition. It indicates arguments
	// that are problematic regardless of the state of the system
	// (e.g., a malformed file name).
	InvalidArgument = Code(connect.CodeInvalidArgument)

	// DeadlineExceeded means operation expired before completion.
	//
	// For operations and APIs that change the state of the system,
	// this error may be returned even if the operation has completed
	// successfully. For example, a successful response from a server
	// could have been delayed long enough for the deadline to expire.
	DeadlineExceeded = Code(connect.CodeDeadlineExceeded)

	// NotFound means some requested entity (e.g., a file or directory) was
	// not found.
	NotFound = Code(connect.CodeNotFound)

	// AlreadyExists means an attempt to create an entity failed because one
	// already exists.
	AlreadyExists = Code(connect.CodeAlreadyExists)

	// PermissionDenied indicates the caller does not have permission to
	// execute the specified operation. It must not be used for rejections
	// caused by exhausting some resource (use ResourceExhausted instead for that
	// purpose).
	PermissionDenied = Code(connect.CodePermissionDenied)

	// ResourceExhausted indicates the operation is out of resource.
	// This should only be returned if there is no other way to interpret
	// the error.
	ResourceExhausted = Code(connect.CodeResourceExhausted)

	// FailedPrecondition indicates the operation was rejected because the
	// system is not in a state required for the operation's execution.
	FailedPrecondition = Code(connect.CodeFailedPrecondition)

	// Aborted indicates the operation was aborted, typically due to a
	// concurrency issue like sequencer check failures, transaction aborts, etc.
	Aborted = Code(connect.CodeAborted)

	// OutOfRange indicates operation was attempted past the valid range.
	// E.g. seeking or reading past end of file.
	OutOfRange = Code(connect.CodeOutOfRange)

	// Unimplemented indicates operation is not implemented or not
	// supported/enabled in this service.
	Unimplemented = Code(connect.CodeUnimplemented)

	// Internal errors. This means that this error should be considered
	// as an internal error that should not happen.
	Internal = Code(connect.CodeInternal)

	// Unavailable indicates the service is currently unavailable.
	// This is a most likely a transient condition and may be corrected
	// by retrying with a backoff.
	Unavailable = Code(connect.CodeUnavailable)

	// DataLoss indicates unrecoverable data loss or corruption.
	// This should only be returned if there is no other way to interpret
	// the error.
	DataLoss = Code(connect.CodeDataLoss)

	// Unauthenticated indicates the request does not have valid
	// authentication credentials for the operation.
	Unauthenticated = Code(connect.CodeUnauthenticated)
)

// IsServerError determines if a code represents a server error.
// Based on the Connect-RPC and gRPC specification, server errors are those that
// indicate issues with the server's ability to process the request, rather than
// problems with the request itself.
//
// Server errors (typically map to HTTP 5xx):
//   - Internal: Internal server error (HTTP 500)
//   - Unknown: Unknown server error (HTTP 500)
//   - DataLoss: Unrecoverable data loss (HTTP 500)
//   - Unimplemented: Operation not implemented (HTTP 501)
//   - Unavailable: Service temporarily unavailable (HTTP 503)
//   - DeadlineExceeded: Operation timed out (HTTP 504)
//
// All other codes are considered client errors (typically map to HTTP 4xx) or
// operational errors that are not server faults.
//
// References:
//   - https://connectrpc.com/docs/protocol/#error-codes
//   - https://github.com/grpc/grpc/blob/master/doc/statuscodes.md
func (c Code) IsServerError() bool {
	switch c { //nolint:exhaustive // TODO: Only handling server error codes explicitly
	case Internal, // HTTP 500
		Unknown,          // HTTP 500 (when server generated)
		Unimplemented,    // HTTP 501
		Unavailable,      // HTTP 503
		DeadlineExceeded, // HTTP 504 (Gateway Timeout)
		DataLoss:         // HTTP 500
		return true
	default:
		// All other codes are client errors or operational errors:
		// - Canceled (HTTP 408): Client canceled
		// - InvalidArgument (HTTP 400): Bad request
		// - NotFound (HTTP 404): Resource not found
		// - AlreadyExists (HTTP 409): Conflict
		// - PermissionDenied (HTTP 403): Forbidden
		// - ResourceExhausted (HTTP 429): Too many requests
		// - FailedPrecondition (HTTP 412): Precondition failed
		// - Aborted (HTTP 409): Conflict due to concurrency
		// - OutOfRange (HTTP 400): Bad request
		// - Unauthenticated (HTTP 401): Unauthorized
		return false
	}
}

// ToConnect converts the Code to a connect.Code.
// This is useful when creating Connect errors from domain errors.
//
// Example:
//
//	appErr := apperr.New(codes.InvalidArgument, "invalid input")
//	connectErr := connect.NewError(appErr.Code.ToConnect(), appErr)
func (c Code) ToConnect() connect.Code {
	return connect.Code(c)
}

// String returns the string representation of the code.
func (c Code) String() string {
	return connect.Code(c).String()
}

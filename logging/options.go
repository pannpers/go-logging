package logging

import (
	"io"
	"log/slog"
	"os"
)

// Format represents the log output format.
// It determines how log entries are serialized and written to the output.
type Format int

const (
	// FormatJSON specifies structured JSON output format.
	// Each log entry is written as a single JSON object with fields
	// like "level", "msg", "time", and any custom attributes.
	//
	// Example output:
	//   {"time":"2024-01-15T10:30:45.123Z","level":"INFO","msg":"Hello","key":"value"}
	FormatJSON Format = iota

	// FormatText specifies human-readable text output format.
	// Each log entry is written as space-separated key=value pairs.
	//
	// Example output:
	//   time=2024-01-15T10:30:45.123Z level=INFO msg=Hello key=value
	FormatText
)

// DefaultLevel is the default minimum logging level.
// Only log entries at this level or higher will be written to the output.
const DefaultLevel = slog.LevelInfo

// Option defines a function that configures a Logger during creation.
// Options are applied in the order they are provided to New().
type Option func(*options)

// options holds all the logger configuration used during logger creation.
// This struct is internal and populated by applying Option functions.
type options struct {
	writer          io.Writer
	level           slog.Level
	format          Format
	replaceAttrFunc func(groups []string, a slog.Attr) slog.Attr
}

// defaultOptions returns the default logger options with text format, info level, and stdout output.
func defaultOptions() *options {
	return &options{
		writer: os.Stdout,
		level:  DefaultLevel,
		format: FormatText, // Default to human-readable text format.
		// replaceAttrFunc is nil by default, meaning no attributes are replaced.
	}
}

// WithWriter configures the output destination for log entries.
// If w is nil, it is ignored and the default (os.Stdout) is used.
//
// Example:
//
//	logger, err := logging.New(
//		logging.WithWriter(os.Stderr),  // Write to stderr
//		logging.WithWriter(&buf),       // Write to buffer
//	)
func WithWriter(w io.Writer) Option {
	return func(o *options) {
		if w != nil {
			o.writer = w
		}
	}
}

// WithLevel configures the minimum logging level.
// Only log entries at this level or higher will be written.
//
// Standard levels:
//   - slog.LevelDebug (-4): Detailed diagnostic information
//   - slog.LevelInfo (0): General application information
//   - slog.LevelWarn (4): Warning conditions
//   - slog.LevelError (8): Error conditions
//
// Example:
//
//	logger, err := logging.New(
//		logging.WithLevel(slog.LevelDebug),  // Enable debug logging
//	)
func WithLevel(level slog.Level) Option {
	return func(o *options) {
		o.level = level
	}
}

// WithFormat configures the output format for log entries.
//
// Example:
//
//	logger, err := logging.New(
//		logging.WithFormat(logging.FormatJSON),  // Use JSON format
//	)
func WithFormat(f Format) Option {
	return func(o *options) {
		o.format = f
	}
}

// WithReplaceAttr configures a function to transform log attributes before output.
// This can be used to rename fields, remove sensitive data, or modify values.
//
// The function receives:
//   - groups: attribute group names (for nested attributes)
//   - a: the attribute to potentially transform
//
// Return an empty slog.Attr{} to remove the attribute entirely.
//
// Example:
//
//	logger, err := logging.New(
//		logging.WithReplaceAttr(func(groups []string, a slog.Attr) slog.Attr {
//			switch a.Key {
//			case slog.TimeKey:
//				a.Key = "timestamp"  // Rename time field
//			case "password":
//				return slog.Attr{}   // Remove password field
//			case slog.LevelKey:
//				a.Key = "severity"   // Rename level field
//			}
//			return a
//		}),
//	)
func WithReplaceAttr(f func(groups []string, a slog.Attr) slog.Attr) Option {
	return func(o *options) {
		o.replaceAttrFunc = f
	}
}

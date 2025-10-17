package apperr

import "sync/atomic"

// config holds the global configuration for the apperr package.
type config struct {
	// includeStacktrace controls whether stack traces are included in log output.
	// Use atomic operations to ensure thread-safe access.
	// Note: While this configuration is typically set once at application startup,
	// atomic.Bool provides safety for edge cases where configuration might be
	// changed during runtime (e.g., toggling debug mode in development).
	// The minimal overhead of atomic operations is acceptable for the safety guarantee.
	includeStacktrace atomic.Bool
}

// globalConfig is the package-level configuration instance.
var globalConfig = &config{}

// init sets the default configuration values.
func init() { //nolint:gochecknoinits // TODO: Required for package initialization of default config
	// By default, do not include stacktraces in production for performance and security
	globalConfig.includeStacktrace.Store(false)
}

// IsStacktraceEnabled returns whether stack traces are currently being captured and included in log output.
// This is useful for testing or conditional logic based on the current configuration.
func IsStacktraceEnabled() bool {
	return globalConfig.includeStacktrace.Load()
}

// ConfigOption is a functional option for configuring apperr behavior.
type ConfigOption func(*config)

// WithStacktrace returns a ConfigOption that sets whether to capture and include stacktraces.
// This is an alternative way to configure stacktrace inclusion if you prefer
// functional options pattern.
//
// Example:
//
//	apperr.Configure(
//	    apperr.WithStacktrace(true),
//	)
func WithStacktrace(enable bool) ConfigOption {
	return func(c *config) {
		c.includeStacktrace.Store(enable)
	}
}

// Configure applies the given configuration options to the global configuration.
// This provides a functional options pattern for configuration.
//
// Example:
//
//	apperr.Configure(
//	    apperr.WithStacktrace(true),
//	)
func Configure(opts ...ConfigOption) {
	for _, opt := range opts {
		opt(globalConfig)
	}
}

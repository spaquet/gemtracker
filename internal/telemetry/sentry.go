// Package telemetry provides optional error tracking via Sentry.
//
// Error tracking is completely optional and only enabled if SENTRY_DSN environment variable is set.
// When disabled, all functions are safe no-ops. This is designed for production use without
// requiring error tracking.
package telemetry

import (
	"os"
	"time"

	"github.com/getsentry/sentry-go"
)

// InitSentry initializes Sentry error tracking if SENTRY_DSN environment variable is set.
// If SENTRY_DSN is not set, this is a no-op. Returns an error only if initialization fails
// with a configured DSN (not if DSN is missing).
// version should be the application version string (e.g., "1.2.7" or "dev").
func InitSentry(version string) error {
	dsn := os.Getenv("SENTRY_DSN")
	if dsn == "" {
		// Sentry is optional - only enable if explicitly configured
		return nil
	}

	release := ""
	if version != "" {
		release = "gemtracker@" + version
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Release:          release,
		TracesSampleRate: 0.1, // Sample 10% of transactions
		Debug:            false,
		AttachStacktrace: true,
	})

	if err != nil {
		// Log error but don't crash - Sentry is optional
		return err
	}

	return nil
}

// CaptureError captures an error in Sentry at the default level if Sentry is initialized.
// Safe to call even if Sentry is not initialized (no-op in that case).
func CaptureError(err error) {
	if sentry.CurrentHub().Client() == nil {
		return
	}
	sentry.CaptureException(err)
}

// CaptureException captures an exception in Sentry with a specified severity level.
// Safe to call even if Sentry is not initialized (no-op in that case).
func CaptureException(err error, level sentry.Level) {
	if sentry.CurrentHub().Client() == nil {
		return
	}
	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetLevel(level)
		sentry.CaptureException(err)
	})
}

// Close flushes any pending events to Sentry with a 2-second timeout.
// Safe to call even if Sentry is not initialized (no-op in that case).
// Should be called via defer in main().
func Close() {
	if sentry.CurrentHub().Client() == nil {
		return
	}
	sentry.Flush(2 * time.Second)
}

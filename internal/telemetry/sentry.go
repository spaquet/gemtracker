package telemetry

import (
	"os"
	"time"

	"github.com/getsentry/sentry-go"
)

// InitSentry initializes Sentry error tracking if SENTRY_DSN is set
// If not set or in development, Sentry is not initialized
func InitSentry() error {
	dsn := os.Getenv("SENTRY_DSN")
	if dsn == "" {
		// Sentry is optional - only enable if explicitly configured
		return nil
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
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

// CaptureError captures an error in Sentry if initialized
func CaptureError(err error) {
	if sentry.CurrentHub().Client() == nil {
		return
	}
	sentry.CaptureException(err)
}

// CaptureException captures an exception with context
func CaptureException(err error, level sentry.Level) {
	if sentry.CurrentHub().Client() == nil {
		return
	}
	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetLevel(level)
		sentry.CaptureException(err)
	})
}

// Close flushes any pending events to Sentry
func Close() {
	if sentry.CurrentHub().Client() == nil {
		return
	}
	sentry.Flush(2 * time.Second)
}

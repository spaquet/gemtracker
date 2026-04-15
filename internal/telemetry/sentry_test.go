package telemetry

import (
	"errors"
	"testing"

	"github.com/getsentry/sentry-go"
)

func TestInitSentry_NoDSN(t *testing.T) {
	t.Setenv("SENTRY_DSN", "")

	err := InitSentry("")
	if err != nil {
		t.Errorf("InitSentry() with no DSN should return nil, got: %v", err)
	}
}

func TestInitSentry_EmptyDSN(t *testing.T) {
	// Explicitly unset the env var
	t.Setenv("SENTRY_DSN", "")

	err := InitSentry("")
	if err != nil {
		t.Errorf("InitSentry() with empty DSN should return nil, got: %v", err)
	}
}

func TestInitSentry_InvalidDSN(t *testing.T) {
	t.Setenv("SENTRY_DSN", "not-a-valid-dsn")

	err := InitSentry("")
	if err == nil {
		t.Error("InitSentry() with invalid DSN should return an error")
	}
}

func TestCaptureError_Uninitialized(t *testing.T) {
	// Clear any previous Sentry initialization
	t.Setenv("SENTRY_DSN", "")
	InitSentry("")

	// Should not panic when calling CaptureError without initialization
	testErr := errors.New("test error")
	CaptureError(testErr) // Should not panic
}

func TestCaptureException_Uninitialized(t *testing.T) {
	// Clear any previous Sentry initialization
	t.Setenv("SENTRY_DSN", "")
	InitSentry("")

	// Should not panic when calling CaptureException without initialization
	testErr := errors.New("test exception")
	CaptureException(testErr, sentry.LevelError) // Should not panic
}

func TestClose_Uninitialized(t *testing.T) {
	// Clear any previous Sentry initialization
	t.Setenv("SENTRY_DSN", "")
	InitSentry("")

	// Should not panic when calling Close without initialization
	Close() // Should not panic
}

func TestCaptureError_SafeWithoutClient(t *testing.T) {
	// Ensure no client is initialized
	t.Setenv("SENTRY_DSN", "")
	InitSentry("")

	// Multiple calls should be safe
	for i := 0; i < 5; i++ {
		CaptureError(errors.New("error"))
	}
}

func TestCaptureException_SafeWithoutClient(t *testing.T) {
	// Ensure no client is initialized
	t.Setenv("SENTRY_DSN", "")
	InitSentry("")

	// Multiple calls with different levels should be safe
	levels := []sentry.Level{
		sentry.LevelDebug,
		sentry.LevelInfo,
		sentry.LevelWarning,
		sentry.LevelError,
		sentry.LevelFatal,
	}

	for _, level := range levels {
		CaptureException(errors.New("test"), level) // Should not panic
	}
}

func TestInitSentry_CanBeCalledMultipleTimes(t *testing.T) {
	t.Setenv("SENTRY_DSN", "")

	// Calling InitSentry multiple times should not error
	err1 := InitSentry("")
	err2 := InitSentry("")

	if err1 != nil || err2 != nil {
		t.Errorf("Multiple InitSentry() calls should not error: %v, %v", err1, err2)
	}
}

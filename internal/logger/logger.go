// Package logger provides optional file-based logging for gemtracker.
//
// When verbose mode is enabled, logs are written to ~/.cache/gemtracker/gemtracker.log.
// When disabled, uses io.Discard for zero overhead. All logging functions are thread-safe.
package logger

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
)

var (
	// mu protects the logger and file handle during concurrent access
	mu     sync.Mutex
	logger *log.Logger
	file   *os.File
)

// Init initializes the logger. If verbose is false, logging is disabled using io.Discard with zero overhead.
// If verbose is true, opens ~/.cache/gemtracker/gemtracker.log for appending with timestamps.
// Returns an error if the log file cannot be created or opened.
func Init(verbose bool) error {
	mu.Lock()
	defer mu.Unlock()

	// If not verbose, use discard writer (no-op, zero overhead)
	if !verbose {
		logger = log.New(io.Discard, "", 0)
		return nil
	}

	// Get home directory and construct cache path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	cacheDir := filepath.Join(homeDir, ".cache", "gemtracker")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}

	// Open log file for appending
	logPath := filepath.Join(cacheDir, "gemtracker.log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	file = f
	logger = log.New(f, "", log.LstdFlags)
	return nil
}

// Info logs an informational message with [INFO] prefix. Thread-safe.
// No-op if logger has not been initialized.
func Info(format string, v ...interface{}) {
	mu.Lock()
	defer mu.Unlock()

	if logger == nil {
		return
	}
	logger.Printf("[INFO] "+format, v...)
}

// Warn logs a warning message with [WARN] prefix. Thread-safe.
// No-op if logger has not been initialized.
func Warn(format string, v ...interface{}) {
	mu.Lock()
	defer mu.Unlock()

	if logger == nil {
		return
	}
	logger.Printf("[WARN] "+format, v...)
}

// Error logs an error message with [ERROR] prefix. Thread-safe.
// No-op if logger has not been initialized.
func Error(format string, v ...interface{}) {
	mu.Lock()
	defer mu.Unlock()

	if logger == nil {
		return
	}
	logger.Printf("[ERROR] "+format, v...)
}

// Close closes the log file if one was opened. Should be called via defer in main().
// Returns an error if the file cannot be closed. Safe to call if logging is disabled.
func Close() error {
	mu.Lock()
	defer mu.Unlock()

	if file != nil {
		return file.Close()
	}
	return nil
}

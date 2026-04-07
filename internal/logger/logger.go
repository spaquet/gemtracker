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

// Init initializes the logger. If verbose is false, uses io.Discard (zero overhead).
// If verbose is true, opens ~/.cache/gemtracker/gemtracker.log for appending.
// Returns error if log file cannot be opened.
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

// Info logs an info message
func Info(format string, v ...interface{}) {
	mu.Lock()
	defer mu.Unlock()

	if logger == nil {
		return
	}
	logger.Printf("[INFO] "+format, v...)
}

// Warn logs a warning message
func Warn(format string, v ...interface{}) {
	mu.Lock()
	defer mu.Unlock()

	if logger == nil {
		return
	}
	logger.Printf("[WARN] "+format, v...)
}

// Error logs an error message
func Error(format string, v ...interface{}) {
	mu.Lock()
	defer mu.Unlock()

	if logger == nil {
		return
	}
	logger.Printf("[ERROR] "+format, v...)
}

// Close flushes and closes the log file if it was opened
func Close() error {
	mu.Lock()
	defer mu.Unlock()

	if file != nil {
		return file.Close()
	}
	return nil
}

package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// reset clears package-level state between tests
func reset() {
	mu.Lock()
	defer mu.Unlock()

	if file != nil {
		file.Close()
		file = nil
	}
	logger = nil
}

func TestInit_NotVerbose(t *testing.T) {
	defer reset()

	homeDir, _ := os.UserHomeDir()
	logPath := filepath.Join(homeDir, ".cache", "gemtracker", "gemtracker.log")

	// Remove any existing log file to start clean
	os.Remove(logPath)

	err := Init(false)
	if err != nil {
		t.Fatalf("Init(false) returned error: %v", err)
	}

	// Verify logger is initialized
	mu.Lock()
	if logger == nil {
		t.Error("logger should be initialized")
	}
	mu.Unlock()

	// Verify no file was created by Init(false) itself
	// (file may exist from other tests, but Init(false) should not create it)
	// So we just verify that we can call the functions without error
	Info("test")
	Warn("test")
	Error("test")
}

func TestInit_Verbose(t *testing.T) {
	defer reset()

	// Clean up any previous test's log file
	homeDir, _ := os.UserHomeDir()
	logPath := filepath.Join(homeDir, ".cache", "gemtracker", "gemtracker.log")
	defer os.Remove(logPath)

	err := Init(true)
	if err != nil {
		t.Fatalf("Init(true) returned error: %v", err)
	}

	// Verify logger is initialized
	mu.Lock()
	if logger == nil {
		t.Error("logger should be initialized")
	}
	mu.Unlock()

	// Verify file was created
	if _, err := os.Stat(logPath); err != nil {
		t.Errorf("log file should be created at %s, got error: %v", logPath, err)
	}

	Close()
}

func TestClose_NoFile(t *testing.T) {
	defer reset()

	err := Close()
	if err != nil {
		t.Errorf("Close() with no file should return nil, got: %v", err)
	}
}

func TestClose_WithFile(t *testing.T) {
	defer reset()

	homeDir, _ := os.UserHomeDir()
	logPath := filepath.Join(homeDir, ".cache", "gemtracker", "gemtracker.log")
	defer os.Remove(logPath)

	Init(true)

	err := Close()
	if err != nil {
		t.Errorf("Close() should return nil, got: %v", err)
	}
}

func TestLogging_BeforeInit_NoPanic(t *testing.T) {
	defer reset()

	// Should not panic when called before Init()
	Info("test")
	Warn("test")
	Error("test")
}

func TestLogging_NotVerbose_NoOutput(t *testing.T) {
	defer reset()

	homeDir, _ := os.UserHomeDir()
	logPath := filepath.Join(homeDir, ".cache", "gemtracker", "gemtracker.log")

	// Remove any existing log file
	os.Remove(logPath)

	Init(false)

	Info("should not appear")
	Warn("should not appear")
	Error("should not appear")

	// Verify no file was created
	if _, err := os.Stat(logPath); err == nil {
		os.Remove(logPath)
		t.Error("log file should not be created when verbose=false")
	}
}

func TestLogging_Verbose_WritesToFile(t *testing.T) {
	defer reset()

	homeDir, _ := os.UserHomeDir()
	logPath := filepath.Join(homeDir, ".cache", "gemtracker", "gemtracker.log")
	defer os.Remove(logPath)

	Init(true)

	// Write test messages
	Info("info message")
	Warn("warn message")
	Error("error message")

	Close()

	// Read the file and verify content
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)

	// Verify all three levels appear
	if !strings.Contains(logContent, "[INFO]") {
		t.Error("log should contain [INFO]")
	}
	if !strings.Contains(logContent, "[WARN]") {
		t.Error("log should contain [WARN]")
	}
	if !strings.Contains(logContent, "[ERROR]") {
		t.Error("log should contain [ERROR]")
	}

	// Verify messages appear
	if !strings.Contains(logContent, "info message") {
		t.Error("log should contain 'info message'")
	}
	if !strings.Contains(logContent, "warn message") {
		t.Error("log should contain 'warn message'")
	}
	if !strings.Contains(logContent, "error message") {
		t.Error("log should contain 'error message'")
	}
}

func TestLogging_Concurrent(t *testing.T) {
	defer reset()

	homeDir, _ := os.UserHomeDir()
	logPath := filepath.Join(homeDir, ".cache", "gemtracker", "gemtracker.log")
	defer os.Remove(logPath)

	Init(true)
	defer Close()

	// Spawn multiple goroutines writing concurrently
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(3)

		go func(id int) {
			defer wg.Done()
			Info("goroutine %d info", id)
		}(i)

		go func(id int) {
			defer wg.Done()
			Warn("goroutine %d warn", id)
		}(i)

		go func(id int) {
			defer wg.Done()
			Error("goroutine %d error", id)
		}(i)
	}

	wg.Wait()

	// Verify file has content (no panics or race conditions should occur)
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if len(content) == 0 {
		t.Error("log file should have content")
	}
}

func TestLogging_FormattedMessages(t *testing.T) {
	defer reset()

	homeDir, _ := os.UserHomeDir()
	logPath := filepath.Join(homeDir, ".cache", "gemtracker", "gemtracker.log")
	defer os.Remove(logPath)

	Init(true)

	// Test formatted messages with arguments
	Info("formatted %s %d", "string", 42)
	Warn("warning with error: %v", fmt.Errorf("test error"))
	Error("error code %d", 500)

	Close()

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)

	// Verify formatted content appears
	if !strings.Contains(logContent, "formatted string 42") {
		t.Error("log should contain formatted info message")
	}
	if !strings.Contains(logContent, "test error") {
		t.Error("log should contain formatted warn message with error")
	}
	if !strings.Contains(logContent, "error code 500") {
		t.Error("log should contain formatted error message")
	}
}

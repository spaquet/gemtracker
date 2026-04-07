package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestPrintVersion(t *testing.T) {
	tests := []struct {
		name               string
		version            string
		commit             string
		date               string
		expectedContains   []string
		expectedNotContain []string
	}{
		{
			name:               "Development version",
			version:            "dev",
			commit:             "none",
			date:               "unknown",
			expectedContains:   []string{"gemtracker", "(development)"},
			expectedNotContain: []string{"dev,", "none", "unknown"},
		},
		{
			name:             "Release version with commit and date",
			version:          "1.0.0",
			commit:           "abc123",
			date:             "2026-04-07",
			expectedContains: []string{"gemtracker 1.0.0", "abc123", "2026-04-07"},
		},
		{
			name:               "Release version with commit only",
			version:            "2.1.0",
			commit:             "def456",
			date:               "unknown",
			expectedContains:   []string{"gemtracker 2.1.0", "def456"},
			expectedNotContain: []string{"unknown"},
		},
		{
			name:               "Release version without commit",
			version:            "1.5.0",
			commit:             "",
			date:               "2026-04-07",
			expectedContains:   []string{"gemtracker 1.5.0"},
			expectedNotContain: []string{"(", ")"},
		},
		{
			name:             "Empty version (should show dev)",
			version:          "",
			commit:           "none",
			date:             "unknown",
			expectedContains: []string{"gemtracker", "(development)"},
		},
		{
			name:             "Version with commit and date",
			version:          "0.1.0",
			commit:           "xyz789",
			date:             "2026-01-01",
			expectedContains: []string{"gemtracker 0.1.0", "(xyz789, 2026-01-01)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original values
			origVersion := version
			origCommit := commit
			origDate := date

			// Set test values
			version = tt.version
			commit = tt.commit
			date = tt.date

			// Capture stdout
			r, w, _ := os.Pipe()
			oldStdout := os.Stdout
			os.Stdout = w

			// Call printVersion
			printVersion()

			// Restore stdout
			os.Stdout = oldStdout
			w.Close()

			// Read captured output
			output, _ := io.ReadAll(r)
			result := strings.TrimSpace(string(output))

			// Restore original values
			version = origVersion
			commit = origCommit
			date = origDate

			// Check expected content
			for _, expected := range tt.expectedContains {
				if !strings.Contains(result, expected) {
					t.Errorf("Output doesn't contain %q.\nGot: %q", expected, result)
				}
			}

			// Check that unexpected content is not present
			for _, notExpected := range tt.expectedNotContain {
				if strings.Contains(result, notExpected) {
					t.Errorf("Output contains unexpected %q.\nGot: %q", notExpected, result)
				}
			}
		})
	}
}

func TestPrintVersionFormat(t *testing.T) {
	// Test the exact format with all fields
	version = "1.2.3"
	commit = "abcdef0"
	date = "2026-04-07T12:00:00Z"

	// Capture stdout
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	printVersion()

	os.Stdout = oldStdout
	w.Close()

	output, _ := io.ReadAll(r)
	result := strings.TrimSpace(string(output))

	// Expected format: "gemtracker 1.2.3 (abcdef0, 2026-04-07T12:00:00Z)"
	expected := "gemtracker 1.2.3 (abcdef0, 2026-04-07T12:00:00Z)"
	if result != expected {
		t.Errorf("Output format mismatch.\nExpected: %q\nGot:      %q", expected, result)
	}
}

func TestPrintVersionDevWithCommit(t *testing.T) {
	// Test dev version with commit (should still show dev)
	version = "dev"
	commit = "abc123"
	date = "2026-04-07"

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	printVersion()

	os.Stdout = oldStdout
	w.Close()

	output, _ := io.ReadAll(r)
	result := strings.TrimSpace(string(output))

	// Should show development but also include commit info
	if !strings.Contains(result, "gemtracker") {
		t.Errorf("Missing 'gemtracker' in output: %q", result)
	}

	if !strings.Contains(result, "(development)") && !strings.Contains(result, "(abc123") {
		t.Errorf("Missing version info in output: %q", result)
	}
}

func TestPrintVersionNoneCommit(t *testing.T) {
	// Test when commit is "none" (should not be shown)
	version = "1.0.0"
	commit = "none"
	date = "2026-04-07"

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	printVersion()

	os.Stdout = oldStdout
	w.Close()

	output, _ := io.ReadAll(r)
	result := strings.TrimSpace(string(output))

	// Should not contain "none"
	if strings.Contains(result, "none") {
		t.Errorf("Output should not contain 'none': %q", result)
	}

	// Should show version
	if !strings.Contains(result, "gemtracker 1.0.0") {
		t.Errorf("Output should contain 'gemtracker 1.0.0': %q", result)
	}
}

func TestPrintVersionOutputToStdout(t *testing.T) {
	// Ensure printVersion writes to stdout, not stderr
	version = "1.0.0"
	commit = "abc123"
	date = "2026-04-07"

	// Capture both stdout and stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	os.Stdout = wOut
	os.Stderr = wErr

	printVersion()

	os.Stdout = oldStdout
	os.Stderr = oldStderr
	wOut.Close()
	wErr.Close()

	stdoutBytes, _ := io.ReadAll(rOut)
	stderrBytes, _ := io.ReadAll(rErr)

	stdoutStr := strings.TrimSpace(string(stdoutBytes))
	stderrStr := strings.TrimSpace(string(stderrBytes))

	// Output should be on stdout, not stderr
	if stdoutStr == "" {
		t.Error("printVersion() didn't write to stdout")
	}

	if stderrStr != "" {
		t.Errorf("printVersion() shouldn't write to stderr: %q", stderrStr)
	}

	if !strings.Contains(stdoutStr, "gemtracker") {
		t.Errorf("stdout doesn't contain expected version info: %q", stdoutStr)
	}
}

func TestPrintVersionAllEmpty(t *testing.T) {
	// Test when all values are empty/default
	version = ""
	commit = ""
	date = ""

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	printVersion()

	os.Stdout = oldStdout
	w.Close()

	output, _ := io.ReadAll(r)
	result := strings.TrimSpace(string(output))

	// Should at least contain the app name
	if !strings.Contains(result, "gemtracker") {
		t.Errorf("Output should contain 'gemtracker': %q", result)
	}

	// Should show development when version is empty
	if !strings.Contains(result, "(development)") {
		t.Errorf("Output should indicate development: %q", result)
	}
}

func TestPrintVersionVersionOnly(t *testing.T) {
	// Test with only version set
	version = "2.0.0"
	commit = ""
	date = ""

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	printVersion()

	os.Stdout = oldStdout
	w.Close()

	output, _ := io.ReadAll(r)
	result := strings.TrimSpace(string(output))

	expected := "gemtracker 2.0.0"
	if !strings.Contains(result, expected) {
		t.Errorf("Output should contain %q: got %q", expected, result)
	}

	// Should not have parentheses if no commit
	if strings.Contains(result, "(") && !strings.Contains(result, "development") {
		t.Errorf("Output shouldn't have parentheses without commit info: %q", result)
	}
}

func BenchmarkPrintVersion(b *testing.B) {
	version = "1.0.0"
	commit = "abc123"
	date = "2026-04-07"

	// Capture stdout during benchmark
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		os.Stdout = w
		printVersion()
		os.Stdout = oldStdout
	}

	w.Close()
	io.ReadAll(r)
}

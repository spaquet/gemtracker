package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spaquet/gemtracker/internal/telemetry"
	"github.com/spaquet/gemtracker/internal/ui"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Initialize Sentry error tracking (optional, only if SENTRY_DSN is set)
	if err := telemetry.InitSentry(); err != nil {
		// Log error but continue - Sentry is optional
		fmt.Fprintf(os.Stderr, "Warning: Failed to initialize error tracking: %v\n", err)
	}
	defer telemetry.Close()

	// Parse command-line flags
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gemtracker [path]\n")
		fmt.Fprintf(os.Stderr, "       gemtracker [--version | -v]\n")
		fmt.Fprintf(os.Stderr, "\nArguments:\n")
		fmt.Fprintf(os.Stderr, "  path              Path to Ruby project directory or Gemfile.lock file\n")
		fmt.Fprintf(os.Stderr, "                    (default: current directory)\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  gemtracker .\n")
		fmt.Fprintf(os.Stderr, "  gemtracker ~/my-rails-app\n")
		fmt.Fprintf(os.Stderr, "  gemtracker /path/to/project/Gemfile.lock\n")
	}

	showVersion := flag.Bool("v", false, "Show version")
	flag.BoolVar(showVersion, "version", false, "Show version")
	noCache := flag.Bool("no-cache", false, "Skip cache and force fresh analysis")
	flag.Parse()

	if *showVersion {
		printVersion()
		os.Exit(0)
	}

	// Get project path from arguments or use current directory
	var projectPath string
	args := flag.Args()
	if len(args) > 0 {
		projectPath = args[0]
	} else {
		projectPath = "."
	}

	// Start the interactive TUI
	model := ui.NewModel(version, commit, date, projectPath, *noCache)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		telemetry.CaptureError(err)
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}

func printVersion() {
	output := "gemtracker"

	if version != "dev" && version != "" {
		output += " " + version
	} else {
		output += " (development)"
	}

	// Add commit info if available
	if commit != "" && commit != "none" {
		output += fmt.Sprintf(" (%s", commit)

		// Add date if available
		if date != "" && date != "unknown" {
			output += fmt.Sprintf(", %s", date)
		}
		output += ")"
	}

	fmt.Println(output)
}

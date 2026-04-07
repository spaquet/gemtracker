package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spaquet/gemtracker/internal/logger"
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
		fmt.Fprintf(os.Stderr, "gemtracker %s\n", version)
		fmt.Fprintf(os.Stderr, "https://github.com/spaquet/gemtracker\n\n")
		fmt.Fprintf(os.Stderr, "Usage: gemtracker [path] [options]\n")
		fmt.Fprintf(os.Stderr, "       gemtracker [--version | -v]\n\n")
		fmt.Fprintf(os.Stderr, "Arguments:\n")
		fmt.Fprintf(os.Stderr, "  path              Path to Ruby project directory or Gemfile.lock file\n")
		fmt.Fprintf(os.Stderr, "                    (default: current directory)\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  --no-cache        Skip cache and force fresh analysis\n")
		fmt.Fprintf(os.Stderr, "  --verbose         Write logs to ~/.cache/gemtracker/gemtracker.log\n")
		fmt.Fprintf(os.Stderr, "  -v, --version     Show version information and exit\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  gemtracker .\n")
		fmt.Fprintf(os.Stderr, "  gemtracker ~/my-rails-app\n")
		fmt.Fprintf(os.Stderr, "  gemtracker /path/to/project/Gemfile.lock --no-cache\n")
	}

	showVersion := flag.Bool("v", false, "Show version")
	flag.BoolVar(showVersion, "version", false, "Show version")
	noCache := flag.Bool("no-cache", false, "Skip cache and force fresh analysis")
	verbose := flag.Bool("verbose", false, "Write logs to ~/.cache/gemtracker/gemtracker.log")

	// Manually parse arguments to support flags in any position
	var projectPath string
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]

		// Handle flags
		if arg == "-v" || arg == "--version" {
			*showVersion = true
		} else if arg == "--no-cache" {
			*noCache = true
		} else if arg == "--verbose" {
			*verbose = true
		} else if arg == "-h" || arg == "--help" {
			flag.Usage()
			os.Exit(0)
		} else if arg[0:1] == "-" {
			// Unknown flag
			fmt.Fprintf(os.Stderr, "Unknown flag: %s\n", arg)
			flag.Usage()
			os.Exit(1)
		} else {
			// First non-flag argument is the path
			if projectPath == "" {
				projectPath = arg
			}
		}
	}

	if *showVersion {
		printVersion()
		os.Exit(0)
	}

	// Initialize logger (before TUI starts)
	if err := logger.Init(*verbose); err != nil {
		// Log error but continue - logger is optional
		fmt.Fprintf(os.Stderr, "Warning: Failed to initialize logger: %v\n", err)
	}
	defer logger.Close()

	// Default to current directory if no path provided
	if projectPath == "" {
		projectPath = "."
	}

	// Start the interactive TUI
	model := ui.NewModel(version, commit, date, projectPath, *noCache, *verbose)
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

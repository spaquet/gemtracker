// Package main is the entry point for gemtracker, an interactive terminal user interface
// for analyzing Ruby gem dependencies from Gemfile.lock files.
//
// It handles command-line flag parsing, initializes the telemetry and logging systems,
// and either launches the interactive TUI or generates a report in non-interactive mode.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/spaquet/gemtracker/internal/logger"
	"github.com/spaquet/gemtracker/internal/telemetry"
	"github.com/spaquet/gemtracker/internal/ui"
)

var (
	// version is the application version, injected at build time via -ldflags.
	version = "dev"
	// commit is the git commit hash, injected at build time via -ldflags.
	commit = "none"
	// date is the build date in ISO format, injected at build time via -ldflags.
	date = "unknown"
)

// Args contains the parsed command-line arguments.
type Args struct {
	ShowVersion  bool
	ProjectPath  string
	ReportFormat string
	OutputPath   string
	NoCache      bool
	Verbose      bool
}

// parseArgs parses command-line arguments and returns the parsed values.
// It handles custom flag parsing to support flags in any position.
func parseArgs() Args {
	args := Args{}

	// Define usage help
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "gemtracker %s\n", version)
		fmt.Fprintf(os.Stderr, "https://github.com/spaquet/gemtracker\n\n")
		fmt.Fprintf(os.Stderr, "Usage: gemtracker [path] [options]\n")
		fmt.Fprintf(os.Stderr, "       gemtracker [--version | -v]\n")
		fmt.Fprintf(os.Stderr, "       gemtracker [--report FORMAT] [path] [options]\n\n")
		fmt.Fprintf(os.Stderr, "Arguments:\n")
		fmt.Fprintf(os.Stderr, "  path              Path to Ruby project directory or Gemfile.lock file\n")
		fmt.Fprintf(os.Stderr, "                    (default: current directory)\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  --report FORMAT   Generate report in non-interactive mode and exit\n")
		fmt.Fprintf(os.Stderr, "                    FORMAT: text, csv, or json\n")
		fmt.Fprintf(os.Stderr, "  --output PATH     Save report to file (default: stdout)\n")
		fmt.Fprintf(os.Stderr, "  --no-cache        Skip cache and force fresh analysis\n")
		fmt.Fprintf(os.Stderr, "  --verbose         Write logs to ~/.cache/gemtracker/gemtracker.log\n")
		fmt.Fprintf(os.Stderr, "  -v, --version     Show version information and exit\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  gemtracker .\n")
		fmt.Fprintf(os.Stderr, "  gemtracker ~/my-rails-app\n")
		fmt.Fprintf(os.Stderr, "  gemtracker --report text /path/to/project\n")
		fmt.Fprintf(os.Stderr, "  gemtracker --report csv --output report.csv\n")
	}

	// Parse manual arguments to support flags in any position
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		i = parseArg(arg, i, &args)
		if args.ShowVersion || args.ReportFormat != "" || args.ProjectPath != "" {
			// Quick exits
		}
	}

	return args
}

// parseArg parses a single argument and returns the updated index.
// It updates the args struct based on the argument type.
func parseArg(arg string, index int, args *Args) int {
	switch arg {
	case "-v", "--version":
		args.ShowVersion = true
	case "--no-cache":
		args.NoCache = true
	case "--verbose":
		args.Verbose = true
	case "--report":
		if index+1 < len(os.Args) && os.Args[index+1][0:1] != "-" {
			args.ReportFormat = os.Args[index+1]
			return index + 1
		}
	case "--output":
		if index+1 < len(os.Args) && os.Args[index+1][0:1] != "-" {
			args.OutputPath = os.Args[index+1]
			return index + 1
		}
	case "-h", "--help":
		flag.Usage()
		os.Exit(0)
	default:
		if len(arg) > 0 && arg[0:1] == "-" {
			fmt.Fprintf(os.Stderr, "Unknown flag: %s\n", arg)
			flag.Usage()
			os.Exit(1)
		} else if args.ProjectPath == "" {
			// First non-flag argument is the path
			args.ProjectPath = arg
		}
	}
	return index
}

func main() {
	// Initialize Sentry error tracking (optional, only if SENTRY_DSN is set)
	if err := telemetry.InitSentry(version); err != nil {
		// Log error but continue - Sentry is optional
		fmt.Fprintf(os.Stderr, "Warning: Failed to initialize error tracking: %v\n", err)
	}
	defer telemetry.Close()

	// Parse command-line arguments
	args := parseArgs()

	if args.ShowVersion {
		printVersion()
		os.Exit(0)
	}

	// Initialize logger (before TUI starts)
	if err := logger.Init(args.Verbose); err != nil {
		// Log error but continue - logger is optional
		fmt.Fprintf(os.Stderr, "Warning: Failed to initialize logger: %v\n", err)
	}
	defer logger.Close()

	// Default to current directory if no path provided
	if args.ProjectPath == "" {
		args.ProjectPath = "."
	}

	// Check if we're in report mode
	if args.ReportFormat != "" {
		generateReport(args.ProjectPath, args.ReportFormat, args.OutputPath, args.NoCache, args.Verbose)
		os.Exit(0)
	}

	// Start the interactive TUI
	model := ui.NewModel(version, commit, date, args.ProjectPath, args.NoCache, args.Verbose)
	p := tea.NewProgram(model)

	if _, err := p.Run(); err != nil {
		telemetry.CaptureError(err)
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}

// generateReport generates a gem dependency report in the specified format (text, csv, or json)
// and writes it to the specified output path or stdout if no path is provided.
// It exits the program with a non-zero status if report generation fails.
func generateReport(projectPath, format, outputPath string, noCache, verbose bool) {
	reportGen := ui.NewReportGenerator(projectPath, noCache, verbose)
	if err := reportGen.Generate(format, outputPath); err != nil {
		telemetry.CaptureError(err)
		fmt.Fprintf(os.Stderr, "Error generating report: %v\n", err)
		os.Exit(1)
	}
}

// printVersion outputs the gemtracker version string to stdout, including commit hash and build date
// if available. If running a development build, it will display "(development)" after the version.
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

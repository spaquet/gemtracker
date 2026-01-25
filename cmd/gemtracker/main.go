package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spaquet/gemtracker/internal/ui"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Parse command line flags
	helpFlag := flag.Bool("help", false, "Show help message")
	versionFlag := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *helpFlag {
		printHelp()
		return
	}

	if *versionFlag {
		printVersion()
		return
	}

	// Start the TUI
	model := ui.NewModel(version, commit, date)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}

func printVersion() {
	fmt.Printf("gemtracker %s\n", version)
	if commit != "none" {
		fmt.Printf("commit: %s\n", commit)
	}
	if date != "unknown" {
		fmt.Printf("date: %s\n", date)
	}
}

func printHelp() {
	help := `gemtracker - Analyze Ruby gem dependencies and identify risks

USAGE:
  gemtracker [options]

OPTIONS:
  --help, -h        Show this help message
  --version, -v     Show version information

INTERACTIVE MODE:
  When run without arguments, gemtracker starts an interactive session
  where you can run multiple analysis commands on your Ruby projects.

KEYBOARD SHORTCUTS:
  ↑/↓, Tab          Navigate commands
  Enter             Run selected command
  Esc               Clear search / return to menu
  q, Ctrl+C         Quit gemtracker

EXAMPLES:
  $ gemtracker           # Start interactive mode
  $ gemtracker --help    # Show help
  $ gemtracker --version # Show version
`
	fmt.Print(help)
}

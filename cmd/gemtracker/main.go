package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "analyze":
		// TODO: Implement analyze command
		fmt.Println("Analyze command - coming soon")
	case "deps":
		// TODO: Implement deps command
		fmt.Println("Deps command - coming soon")
	case "vulnerabilities":
		// TODO: Implement vulnerabilities command
		fmt.Println("Vulnerabilities command - coming soon")
	case "licenses":
		// TODO: Implement licenses command
		fmt.Println("Licenses command - coming soon")
	case "--help", "-h":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	usage := `gemtracker - Analyze Ruby gem dependencies and identify risks

Usage:
  gemtracker <command> [options]

Commands:
  analyze <gemfile.lock>           Analyze a Gemfile.lock for risks and conflicts
  deps <gem-name>                  Show dependency tree for a specific gem
  vulnerabilities <gemfile.lock>   Check for known vulnerabilities
  licenses <gemfile.lock>          Generate license compliance report
  --help, -h                       Show this help message

Examples:
  gemtracker analyze ./Gemfile.lock
  gemtracker deps rails
  gemtracker vulnerabilities ./Gemfile.lock
  gemtracker licenses ./Gemfile.lock
`
	fmt.Print(usage)
}

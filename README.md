# gemtracker

A Go CLI tool to analyze Ruby gem dependencies and quickly identify risks in your projects.

## Features

- **Interactive Terminal UI**: Beautiful, intuitive command palette interface for all analysis functions
- **Dependency Visualization**: See which gems are using a specific gem and what versions
- **Vulnerability Detection**: Identify outdated and potentially vulnerable gem versions
- **License Compliance**: Scan and report on gem licenses for compliance requirements
- **Conflict Detection**: Automatically detect version conflicts and compatibility issues
- **Version Display**: Version info and project stats shown in the terminal header

## Installation

### macOS (Homebrew)
```bash
brew tap spaquet/gemtracker
brew install gemtracker
```

### Linux & Windows
Installation instructions coming soon.

## Usage

### Interactive Mode (Default)
Simply run gemtracker to launch the interactive terminal UI:

```bash
gemtracker
```

The interactive interface provides:
- **Analyze**: Scan your Gemfile.lock for risks, outdated gems, and conflicts
- **Deps**: Show which parent gems are using a specific gem and their versions
- **Vulnerabilities**: Check for known vulnerabilities in your gems
- **Licenses**: Generate a license compliance report
- **Help**: View keyboard shortcuts and detailed help

### Navigation
- **Arrow Keys / Tab**: Navigate through available commands
- **Enter**: Execute selected command
- **Esc**: Clear search or return to main menu
- **q**: Quit gemtracker

## Quick Start

1. Navigate to a Ruby project directory with a `Gemfile.lock`
2. Run `gemtracker`
3. Use the interactive menu to analyze your dependencies

## Project Goals

- Provide fast, actionable insights into gem dependencies
- Help identify security and compliance risks early
- Support easy integration into CI/CD pipelines
- Minimal dependencies and fast performance

## Development

### Prerequisites
- Go 1.21 or later

### Building from Source
```bash
make build
```

### Running Tests
```bash
make test
```

## License

See [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

# gemtracker

A Go CLI tool to analyze Ruby gem dependencies and quickly identify risks in your projects.

## Features

- **Dependency Visualization**: See which gems are using a specific gem and what versions
- **Vulnerability Detection**: Identify outdated and potentially vulnerable gem versions
- **License Compliance**: Scan and report on gem licenses for compliance requirements
- **Conflict Detection**: Automatically detect version conflicts and compatibility issues
- **Terminal Reports**: Beautiful, formatted terminal output for quick analysis

## Installation

### macOS (Homebrew)
```bash
brew tap spaquet/gemtracker
brew install gemtracker
```

### Linux & Windows
Installation instructions coming soon.

## Usage

### Basic Analysis
```bash
gemtracker analyze ./Gemfile.lock
```

### Show Dependency Tree
```bash
gemtracker deps <gem-name>
```

### Export Report
```bash
gemtracker analyze ./Gemfile.lock --format json
gemtracker analyze ./Gemfile.lock --format csv
```

### Check for Vulnerabilities
```bash
gemtracker vulnerabilities ./Gemfile.lock
```

### License Compliance Report
```bash
gemtracker licenses ./Gemfile.lock
```

## Quick Start

1. Navigate to a Ruby project with a `Gemfile.lock`
2. Run `gemtracker analyze ./Gemfile.lock`
3. Review the dependency risks and conflicts

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

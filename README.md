# gemtracker

A beautiful, interactive Terminal UI for analyzing Ruby gem dependencies and quickly identifying security risks in your projects.

## Features

- **Interactive Tab-Based TUI**: Clean, modern interface with 4 main screens
  - **[Gems]** - First-level gem list with versions and update status
  - **[Search]** - Real-time gem search across all dependencies
  - **[CVE]** - Vulnerability detection and reporting
  - **Gem Details** - Full dependency tree visualization

- **Dependency Visualization**: See forward and reverse dependency trees with version info
- **Vulnerability Detection**: Identify known CVEs and affected gem versions
- **Group-Based Analysis**: Understand gem scope (default, development, test, production)
- **Version Management**: See installed versions, latest available, and outdated gems
- **Direct Links**: Quick links to rubygems.org and GitHub repositories

## Installation

### macOS (Homebrew) — Recommended
```bash
brew tap spaquet/gemtracker
brew install gemtracker
```

To upgrade:
```bash
brew upgrade gemtracker
```

### Linux

Download the latest release:
```bash
# For x86-64
curl -L https://github.com/spaquet/gemtracker/releases/download/v1.0.0/gemtracker_linux_amd64.tar.gz | tar xz

# For ARM64
curl -L https://github.com/spaquet/gemtracker/releases/download/v1.0.0/gemtracker_linux_arm64.tar.gz | tar xz
```

Or build from source:
```bash
git clone https://github.com/spaquet/gemtracker
cd gemtracker
make build
```

### Windows

Download the latest release from [GitHub Releases](https://github.com/spaquet/gemtracker/releases):
- `gemtracker_windows_amd64.zip` for x86-64
- `gemtracker_windows_arm64.zip` for ARM64

Extract the ZIP file and add the directory to your PATH, or place `gemtracker.exe` in a directory already in your PATH.

### macOS (Direct Download)

If you prefer not to use Homebrew:
```bash
# For Intel (x86-64)
curl -L https://github.com/spaquet/gemtracker/releases/download/v1.0.0/gemtracker_darwin_amd64.tar.gz | tar xz

# For Apple Silicon (ARM64)
curl -L https://github.com/spaquet/gemtracker/releases/download/v1.0.0/gemtracker_darwin_arm64.tar.gz | tar xz
```

### From Source (All Platforms)

Requires Go 1.24 or later:
```bash
git clone https://github.com/spaquet/gemtracker
cd gemtracker
make build
./gemtracker
```

## Usage

### Basic Usage
```bash
# Analyze current directory (must contain Gemfile.lock)
gemtracker

# Analyze specific project
gemtracker /path/to/project

# Analyze specific Gemfile.lock directly
gemtracker /path/to/project/Gemfile.lock

# Expand tilde for home directory
gemtracker ~/my-rails-app

# Show version
gemtracker -v
gemtracker --version
```

### Interactive Navigation

Once running, use these keys:

#### Tab Navigation
- **Tab** / **Shift+Tab** - Switch between screens ([Gems] → [Search] → [CVE])
- **/** - Jump directly to Search screen

#### List Navigation
- **↑ / ↓** - Move selection up/down
- **Enter** - Select gem to view details

#### Gem Details
- **Tab** - Toggle between dependency sections
- **↑ / ↓** - Scroll through dependencies
- **Esc** - Return to previous screen

#### Global
- **q** / **Ctrl+C** - Quit gemtracker

### Understanding the Gem Table

The gem list shows:
```
#    Gem Name    Installed   Latest      Groups      Status
──────────────────────────────────────────────────────────────
1    rails       7.1.2       7.2.0       default     ↑ 7.2.0
2    devise      4.9.3       latest      default     ✓
3    rack        2.1.2       latest      default     ⚠ CVE
```

**Groups** column shows where gems are used:
- **default** - All environments (production, staging, development)
- **development** - Development only
- **test** - Test only
- **production** - Production only

> **Important**: A vulnerability in a `test` or `development` gem doesn't affect production if not used there.

**Status** column shows:
- **✓** - Up to date, no vulnerabilities
- **↑ version** - Newer version available (outdated)
- **⚠ CVE** - Known vulnerabilities detected

### Understanding CVE Information

The CVE screen shows all known vulnerabilities:
- **CVE ID** - Vulnerability identifier (e.g., CVE-2021-22942)
- **Gem** - Name of the affected gem
- **Version** - Version range affected
- **Description** - What the vulnerability does
- **Status** - Whether gem is directly used or transitive

### Understanding Gem Health Status

Each gem in the [Gems] tab shows a health indicator (colored dot) that reflects the gem's maintenance status. gemtracker fetches this data from RubyGems and GitHub APIs to help you assess dependency health:

**Health Levels:**

- **🟢 HEALTHY** - Actively maintained gem
  - Activity within the last year (release or GitHub commit)
  - Multiple maintainers (2+)
  - Regular updates and engagement

- **🟡 WARNING** - Gem with maintenance concerns
  - No activity in the last 1-3 years, OR
  - Single maintainer (even if recent activity)
  - May still receive occasional updates

- **🔴 CRITICAL** - Potentially dead or unmaintained gem
  - No activity for 3+ years
  - Archived or disabled on GitHub
  - Essentially abandoned

**Gem Details** include full health statistics:
- Last release date
- GitHub stars and watchers
- Open issues count
- Number of active maintainers
- Archived status (if applicable)

**Why Health Matters:**
- A "CRITICAL" gem may indicate security risks if vulnerabilities go unpatched
- Unmaintained gems may have compatibility issues with new Ruby/Rails versions
- "HEALTHY" gems are more likely to receive timely security updates
- Different tolerance levels apply: a test-only gem's health matters less than a production core dependency

> **Note**: Health data is fetched asynchronously in the background. If GitHub rate-limited, cached data from the last 24 hours is used.

## Performance & Caching

### Automatic Analysis Caching

gemtracker automatically caches analysis results for faster subsequent loads:

- **Cache Location**: `~/.cache/gemtracker/`
- **Cache Per Project**: Each project's Gemfile.lock gets its own cache file
- **Smart Invalidation**: Cache is automatically invalidated when Gemfile.lock is modified
- **No Manual Cleanup**: Old cache files are harmless and can be safely ignored

**Example with multiple projects:**
```
~/.cache/gemtracker/
├── Gemfile.lock_1234.json    # Project A cache
├── Gemfile.lock_5678.json    # Project B cache
└── Gemfile.lock_9012.json    # Project C cache
```

When you re-open a project you've analyzed before, if `Gemfile.lock` hasn't changed, analysis loads **instantly** from cache ⚡

**Cache is refreshed when:**
- You run `bundle install` or `bundle update`
- You edit your `Gemfile` (which updates Gemfile.lock)
- The Gemfile.lock file modification time changes

To manually clear cache for a specific project:
```bash
rm ~/.cache/gemtracker/Gemfile.lock_*.json
```

## Quick Start

1. Navigate to a Ruby project with `Gemfile.lock`:
   ```bash
   cd ~/my-rails-app
   ```

2. Launch gemtracker:
   ```bash
   gemtracker
   ```

3. Browse gems:
   - **[Gems]** tab shows all first-level dependencies
   - Press **Enter** on any gem to see its full dependency tree
   - Check **Groups** column to assess vulnerability impact

4. Search for specific gems:
   - Press **/** or click **[Search]** tab
   - Type gem name to filter in real-time
   - Press **Enter** to view details

5. Check vulnerabilities:
   - Click **[CVE]** tab to see all vulnerabilities
   - Filter by gem in [Search] tab
   - Check if vulnerable gems are in production

## Optional: Error Tracking with Sentry

gemtracker includes **optional** error tracking via Sentry to help improve reliability:

- **Completely Optional** - Not enabled by default
- **No Data Without Your Consent** - Only enabled if you set `SENTRY_DSN` environment variable
- **Works Offline** - If Sentry is unavailable, gemtracker continues normally
- **Not Required** - Development and self-built versions work perfectly without it

To enable error tracking (usually only in official releases):
```bash
export SENTRY_DSN="your-sentry-dsn"
gemtracker
```

If the env var is not set, error tracking is completely disabled. This is the default for:
- Self-built versions from source
- Development installations
- Local development

## Building

### Development Build
```bash
make build
```

### Release Build (macOS universal binary)
```bash
make build-release
```

### Version Information
Built binaries include git commit hash and build date. To build with custom version:
```bash
VERSION=1.0.0 COMMIT=abc123 DATE=2026-04-04 make build
```

## Project Goals

- Provide **fast, actionable insights** into gem dependencies
- Help identify **security and compliance risks** early
- Support **easy integration** into CI/CD pipelines
- **Beautiful, intuitive UI** that developers love using
- Minimal dependencies and **fast performance**

## Tech Stack

- **Language**: Go 1.24+
- **TUI Framework**: BubbleTea + Lipgloss (charmbracelet)
- **Data Source**: rubygems.org API + Gemfile.lock parsing

## Development

### Prerequisites
- Go 1.24 or later
- Make

### Setup
```bash
git clone https://github.com/spaquet/gemtracker
cd gemtracker
make build
```

### Running Tests
```bash
make test
```

### Code Quality Checks

gemtracker uses `golangci-lint` for comprehensive code quality checks. These run **automatically before pushing** via a git hook to catch issues early.

#### Installation

First, install golangci-lint:

```bash
# Using Homebrew (macOS)
brew install golangci-lint

# Or using the official installer
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
```

#### Running Checks Locally

```bash
# Run linter
make lint

# Run tests
make test

# Or run both before committing
make lint && make test
```

#### Automatic Pre-Push Hook

A git `pre-push` hook automatically runs tests and linter before pushing to prevent CI failures:

```bash
git push
# Output:
# 🔍 Running linter...
# ✓ Linter passed
# 🧪 Running tests...
# ✓ Tests passed
# ✅ All checks passed! Pushing...
```

To skip the hook (not recommended):
```bash
git push --no-verify
```

**Required before PR submission:**
- ✅ All tests must pass: `make test`
- ✅ Linter must pass: `make lint`

These checks run automatically in GitHub Actions when you push, but fixing them locally first via the pre-push hook prevents CI failures.

### Project Structure
```
gemtracker/
├── cmd/gemtracker/          # CLI entry point
├── internal/
│   ├── gemfile/             # Parsing & analysis
│   │   ├── parser.go        # Gemfile.lock parser
│   │   ├── analyzer.go      # Dependency analysis
│   │   ├── outdated.go      # Version checking
│   │   └── vulnerabilities.go # CVE detection
│   └── ui/                  # Terminal UI
│       ├── model.go         # BubbleTea model
│       ├── update.go        # Message routing
│       ├── view.go          # Screen rendering
│       └── styles.go        # Colors & themes
└── Makefile                 # Build & test
```

## Releases & Updates

gemtracker follows [semantic versioning](https://semver.org/). New versions are released when features are added or bugs are fixed. Check the [releases page](https://github.com/spaquet/gemtracker/releases) for the latest version.

To check your installed version:
```bash
gemtracker --version
```

### Staying Updated

- **Homebrew users**: `brew upgrade gemtracker`
- **Direct download users**: Check [releases](https://github.com/spaquet/gemtracker/releases) page and re-download the latest binary

### Future: Official Homebrew

Once gemtracker has stable releases, we plan to submit it to [homebrew/homebrew-core](https://github.com/Homebrew/homebrew-core), allowing installation with just `brew install gemtracker` (no tap needed).

## Known Limitations

- Only parses standard Gemfile.lock format
- Outdated version checking requires network access
- CVE database is static (not real-time updated)
- No support for Gemfile global options or git/path sources yet

## Documentation

- **[CONTRIBUTING.md](CONTRIBUTING.md)** — How to contribute and code quality requirements
- **[CHANGELOG.md](CHANGELOG.md)** — Version history and what's new in each release
- **[RELEASE_GUIDE.md](RELEASE_GUIDE.md)** — How to make releases and manage the distribution pipeline
- **[SECURITY.md](SECURITY.md)** — Security policy and vulnerability reporting
- **[CLAUDE.md](CLAUDE.md)** — Development guidelines for contributors

## Security

Please report security vulnerabilities privately using [GitHub Security Advisory](https://github.com/spaquet/gemtracker/security/advisories). See [SECURITY.md](SECURITY.md) for details.

## License

See [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Support & Contributing

- **Feature request?** Open a [request](https://github.com/spaquet/gemtracker/issues)
- **Found a bug?** Open an [issue](https://github.com/spaquet/gemtracker/issues)
- **Want to contribute?** See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines
- **Code quality requirements?** See [CONTRIBUTING.md — Code Quality](CONTRIBUTING.md#code-quality)

## Troubleshooting

### "Gemfile.lock not found"
Make sure you're in a Ruby project directory with `Gemfile.lock`, or specify the path:
```bash
gemtracker /path/to/project
```

### Version shows as "(development)"
Build using `make build` instead of `go build` to get proper version info from git.

### Terminal appears garbled
Your terminal may not support 256 colors. Try:
```bash
TERM=xterm-256color gemtracker
```

## Questions?

Check the built-in help or open an issue on GitHub.

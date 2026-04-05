# Gemtracker Development Guide

## Project Overview

**Gemtracker** is an interactive Terminal User Interface (TUI) written in Go that analyzes Ruby gem dependencies from `Gemfile.lock` files. It helps developers identify security risks, outdated dependencies, version conflicts, and provides comprehensive dependency analysis for Ruby, Rails, and other Ruby-based projects.

## Purpose & Vision

The tool provides developers with quick, actionable insights into their gem dependencies through an intuitive interactive interface. Instead of manually parsing Gemfile.lock or running multiple separate commands, developers get a unified view of:
- First-level gems (directly installed dependencies) with version info and availability of updates
- Full dependency tree showing which gems depend on other gems
- Vulnerability detection with pointers to affected gems
- Search functionality to find gems and their usage across the project

## Technology Stack

- **Language**: Go 1.24.0
- **TUI Framework**: BubbleTea (charmbracelet/bubbletea) for interactive terminal UI
- **Dependencies**:
  - `charmbracelet/bubbles` - Reusable components for BubbleTea
  - `charmbracelet/lipgloss` - Terminal styling and rendering

## Architecture

### Directory Structure

```
gemtracker/
├── cmd/gemtracker/          # Application entry point
│   └── main.go             # CLI bootstrap and TUI initialization
├── internal/
│   ├── gemfile/            # Gemfile.lock parsing and analysis logic
│   │   ├── parser.go       # Parse Gemfile.lock format
│   │   ├── analyzer.go     # Analyze dependencies and relationships
│   │   ├── dependencies.go # Dependency tree data structures
│   │   ├── outdated.go     # Check for newer versions available
│   │   └── vulnerabilities.go # CVE detection and reporting
│   └── ui/                 # BubbleTea TUI components
│       ├── model.go        # Root BubbleTea model and state management
│       ├── update.go       # Message handling and state updates
│       ├── view.go         # Rendering logic for all screens
│       └── styles.go       # Terminal colors, fonts, spacing
└── go.mod                   # Go module definition
```

### Core Components

**Gemfile Parser** (`internal/gemfile/parser.go`)
- Parses `Gemfile.lock` YAML-like format
- Extracts gem specs, versions, dependencies, and platforms

**Dependency Analyzer** (`internal/gemfile/analyzer.go`)
- Builds dependency graphs
- Identifies first-level vs transitive dependencies
- Resolves gem relationships and usage chains

**Outdated Checker** (`internal/gemfile/outdated.go`)
- Queries gem repositories for latest available versions
- Compares against currently installed versions
- Returns "latest" or new version number for display

**Vulnerability Scanner** (`internal/gemfile/vulnerabilities.go`)
- Detects known CVEs in gem versions
- Maps vulnerabilities to their gem locations in the tree

**UI Model** (`internal/ui/model.go`)
- Central BubbleTea model managing all TUI state
- Tracks current screen, selection state, search queries
- Owns version info, commit hash, date for display

**UI Update & View** (`internal/ui/update.go`, `internal/ui/view.go`)
- Message handling for user input and async operations
- Screen rendering (gem list, details, search, vulnerabilities)
- Keyboard navigation and interaction

### Async Architecture

**Version Update Checking** (`internal/gemfile/outdated.go`)
- Fetches latest available gem versions from rubygems.org API
- Runs asynchronously in background to keep UI responsive
- Results cached for 24 hours per project (in `~/.cache/gemtracker/`)
- Graceful degradation: if API unavailable, uses cached data or shows "checking..."
- Status indicator shows progress: "Checking updates... (15/189)"
- Respects API rate limits and implements exponential backoff

**Gem Health Checking** (`internal/gemfile/health.go`)
- Fetches maintenance health metrics from:
  - RubyGems API (last release date, maintainer count)
  - GitHub API (stars, maintainers, open issues, last push)
- Evaluates health into three tiers based on `ComputeHealthScore()`:
  - **HEALTHY** (🟢) - Activity within 1 year AND 2+ maintainers
  - **WARNING** (🟡) - No activity for 1-3 years OR single maintainer
  - **CRITICAL** (🔴) - No activity for 3+ years, archived, or disabled
- Runs one gem at a time (sequential) to avoid rate limiting
- Caches results for 24 hours per gem (`~/.cache/gemtracker/{hash}_health.json`)
- Shows health dots in gem list as data loads in background
- Handles GitHub rate limiting gracefully (60 req/hour for anonymous)

**BubbleTea Integration**
- Version checks and health updates sent as BubbleTea `Msg` events
- UI state updated via `Update()` method without blocking
- Progress indicators show loading state in real-time
- Error states (rate limited, network issues) displayed in statusbar
- Can be toggled on/off depending on user needs

## Key Features & Screens

### 1. First-Level Gems List
Display all gems directly installed (not dependencies of other gems):
- Gem name
- Current installed version
- Link to GitHub/GitLab source (fetched via rubymine registry)
- New version availability ("latest" or version number)
- Interactive selection to view details

### 2. Dependency Details
When clicking a gem:
- Show full dependency tree for that gem
- List all transitive dependencies with their versions
- Show reverse dependencies (what depends on this gem)

### 3. Search & Everywhere
- Search by gem name
- Show all occurrences of the gem throughout the dependency tree
- Highlight in which gems it appears as a dependency

### 4. Vulnerabilities Screen
- List all detected CVEs
- Show affected gem name and version
- Pinpoint where the vulnerable gem is being used
- Provide links to CVE database entries

### 5. Navigation & Controls
- Arrow keys / Tab for navigation
- Enter to select
- Esc to clear search or back to main menu
- q to quit
- Command palette interface for main operations

## Development Conventions

### Code Style
- Follow standard Go conventions (gofmt)
- Keep functions focused and single-responsibility
- Prefer clear variable names over abbreviated names
- Comment exported functions and types

### Dependencies & Versions
- Parsing dependencies are read from go.mod (charmbracelet packages only)
- External gem data queried at runtime (not compiled)
- No vendoring - rely on go.mod for reproducible builds

### Error Handling
- Return errors to caller, don't silently fail
- UI should catch errors and display friendly messages
- Never panic - let the user gracefully exit if something fails

### Testing
- Unit tests for parser and analyzer logic
- Test fixtures in `testdata/` directory with sample Gemfile.lock files
- Run tests with `make test`
- Aim for 80%+ coverage on parsing and analysis logic

## Running & Building

### Prerequisites
- Go 1.24.0 or later
- A Ruby project with `Gemfile.lock` in the current directory

### Build Targets (Makefile)

**`make build`** - Build development binary
```bash
make build
# Output: ./gemtracker (with git commit/tag info embedded)
```

**`make build-dev`** - Build development binary (explicit)
```bash
make build-dev
# Same as `make build`
```

**`make build-release`** - Build release binaries for macOS
```bash
make build-release
# Creates:
# - dist/gemtracker-darwin-amd64.tar.gz (Intel Mac)
# - dist/gemtracker-darwin-arm64.tar.gz (Apple Silicon)
```

**`make test`** - Run all tests
```bash
make test
# Runs: go test -v ./...
```

**`make clean`** - Clean build artifacts
```bash
make clean
# Removes: ./gemtracker, ./dist/, go build cache
```

**`make help`** - Show all available targets
```bash
make help
```

### Version Information

Build version info is automatically injected at compile time:
- `VERSION` - Git tag or "dev" (override with `VERSION=1.0.0 make build`)
- `COMMIT` - Short git commit hash (override with `COMMIT=abc123 make build`)
- `DATE` - Build date in ISO format (override with `DATE=... make build`)

Example with custom version:
```bash
VERSION=1.0.0 COMMIT=abc123 DATE="2026-04-04T00:00:00Z" make build
```

### Run
```bash
./gemtracker [path-to-gemfile-lock-or-directory]

# Examples:
./gemtracker                    # Uses ./Gemfile.lock
./gemtracker /path/to/project   # Uses /path/to/project/Gemfile.lock
./gemtracker Gemfile.lock       # Explicit Gemfile.lock path
```

### Development Workflow
```bash
# Quick dev build and run
make build && ./gemtracker

# Run directly without building
go run ./cmd/gemtracker

# Run tests
make test

# Clean everything and rebuild
make clean && make build
```

## UI State Flow

```
Main Model
├── Screen: HomeScreen (gem list)
├── Screen: DetailScreen (gem details)
├── Screen: SearchScreen (search results)
├── Screen: VulnerabilitiesScreen (CVE listing)
└── State:
    ├── current selection
    ├── search query
    ├── loaded Gemfile data
    └── outdated/vulnerability cache
```

When user selects a gem or searches, the model updates and triggers a view re-render.

## Common Development Tasks

### Adding a New Screen
1. Create rendering logic in `view.go` as a new function
2. Add screen type to `model.go`
3. Handle navigation messages in `update.go`
4. Update the main render function to dispatch to the right screen

### Adding a New Analysis Feature
1. Add logic to appropriate file in `internal/gemfile/`
2. Cache results in the Model to avoid re-computation
3. Create a new screen to display results
4. Hook up keyboard shortcut in `update.go`

### Improving Performance
- Cache Gemfile parsing results (only parse on file change detection)
- Use goroutines for long-running tasks (outdated checks, CVE scanning)
- Send progress updates via BubbleTea messages
- Debounce search input to avoid excessive filtering

## Known Limitations & Future Work

- Currently only parses standard Gemfile.lock format (may not handle older formats)
- Outdated version checking requires network access
- CVE database is static - consider integrating with live DB
- No support for Gemfile global options or git/path sources yet
- License compliance feature in README but not yet implemented

## Dependencies & Licensing

- BubbleTea (MIT) - Terminal UI framework
- Bubbles (MIT) - UI components
- Lipgloss (MIT) - Terminal styling
- All dependencies are transitive from these three main packages

## Building & Releasing

### Version Info
Version, commit, and date are passed at build time:
```bash
go build -ldflags \
  "-X main.version=1.0.0 \
   -X main.commit=abc123 \
   -X main.date=2026-04-04"
```

### Homebrew Distribution
Already configured in `spaquet/gemtracker` tap. Follow normal Homebrew PR process for releases.

## Notes for Claude

- When implementing new features, prioritize user experience over complexity
- Keep the TUI responsive - use goroutines for background tasks
- Always validate Gemfile.lock parsing - invalid files should not crash
- When adding features, consider how they fit into the existing screen flow
- Test with real Gemfile.lock files of various sizes (small, medium, large projects)
- Be mindful of terminal width/height constraints - handle resizing gracefully

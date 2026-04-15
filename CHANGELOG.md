# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v1.2.7] - 2026-04-14

### Fixed
- **Platform-Specific Gem Version Detection** - Correct gem version now displayed on Apple Silicon Macs
  - Parser now detects current system platform (arm64-darwin, x86_64-linux, etc.)
  - Selects matching platform-specific version from Gemfile.lock instead of last entry
  - Fixes mismatch where darwin-arm64 Macs saw "x86_64-linux-musl" for gems like pg

- **Gem Info Installed Versions Modal** - Fixed parsing to show all installed versions
  - Handles current `gem info` output format: `gem (v1, v2)` + `Installed at: /path`
  - Also supports legacy format: `Installed at (VERSION): /path`
  - Handles single and multiple installed versions
  - Sorts versions descending (newest first)
  - Manager-independent: displays whatever path Ruby reports (frum, rbenv, rvm, system)

### Improved
- **Dependency Analysis** - Platform-specific gems now analyzed with correct version
  - Ensures version comparison and outdated checks use correct platform version
  - Improves accuracy of vulnerability and health reporting for platform-specific gems

## [v1.2.6] - 2026-04-12

### Added
- **Gemspec-Only Project Support** - Analyze Ruby gems and libraries without Gemfile.lock
  - Automatically detects and parses `.gemspec` files when Gemfile.lock is absent
  - Enriches dependencies from RubyGems API to populate full dependency tree
  - File detection order: Gemfile.lock → gems.locked → .gemspec
  - Enables dependency analysis for gem development projects and libraries

### Fixed
- **Sentry Release Source Upload** - Fixed incorrect sentry-cli command syntax
  - Changed from non-existent `upload-sourcemaps` to correct `upload` subcommand
  - Properly uploads Go source files for code context in Sentry error tracking
- **PATH Section Parsing in Gemfile.lock** - Fixed parsing of local gem references
  - Now correctly handles PATH sections for gems sourced from local directories
  - Enables proper parsing of gemspec-based projects with local path gems

### Improved
- **Dependency Detection Robustness** - Better error handling for missing dependency files
  - Gracefully reports when no dependency files (lock or gemspec) are found
  - Clearer error messages guide users to available formats

### Known Limitations
- **Gemspec-Only Projects**: When parsing `.gemspec` files without a lock file:
  - Dependency **versions** are not guaranteed (shown as `?` when unavailable)
  - "Used By" relationships may be empty (would require full lock file for complete reverse dependency analysis)
  - RubyGems API provides forward dependencies but cannot reverse-map all consuming gems
  - Full analysis requires `Gemfile.lock` for complete dependency graph with versions

## [v1.2.5] - 2026-04-11

### Fixed
- **Sentry Error Tracking for Production Panics** - Runtime panics now properly captured and reported
  - All panic recovery points now send errors to Sentry with Fatal severity level
  - Includes panics from placeOverlay(), View() rendering, renderGemInfoModalBox(), and fetchGemInfo()
  - Ensures critical production issues are visible in Sentry dashboard for debugging
  - Complements existing SENTRY_DSN configuration and GitHub Actions release workflow

### Improved
- **Enhanced Production Debugging with Sentry Source Upload** - Source files now pushed to Sentry on release
  - GitHub Actions workflow uploads all source code to Sentry on new release tag
  - Developers see full code context directly in Sentry error views (no external GitHub calls)
  - Commits linked to releases for tracking code changes alongside errors
  - Proper file path rewriting ensures correct source mapping
  - Makes production troubleshooting significantly faster and more efficient

## [v1.2.4] - 2026-04-11

### Fixed
- **Nil Pointer Dereference on Gem Detail View** - Fixed crash when opening dependencies view
  - Added nil check for DependencyInfo before accessing ForwardTree
  - Prevents panic when gem detail view is opened in Gems, Search, Updates, CVE, and Sanity tabs
  - Improves robustness when analyzing gems with missing dependency data

### Improved
- **Code Quality: Reduced Update() Cyclomatic Complexity** (23 → 14)
  - Extracted health, outdated, and CVE message dispatching into separate dispatcher methods
  - Maintains identical behavior while improving code organization and testability
  - Update() method now focused and maintainable, complexity within acceptable limits

## [v1.2.3] - 2026-04-10

### Fixed
- **CVE Comment Modal Not Opening from Info View** - Pressing 'c' in CVE info modal now opens comment editor
  - Added 'c' key handler to CVE info view for consistency
  - Comments can now be edited from both main CVE list and detailed info modal
- **CVE Acknowledgment Filter Not Working** - Complete refactor with checkbox support
  - Changed from single-state cycling to multi-select checkbox filtering
  - Now supports three independent filter states: Acknowledged, Ignored, Unacknowledged
  - Users can select any combination of states with space bar (like severity filters)
  - All three states are enabled by default (shows all vulnerabilities)
  - Properly detects "Unacknowledged" as initial state (no comment on CVE)
  - Fixed nil panic when filtering with no comments loaded

## [v1.2.1] - 2026-04-10

### Improved
- **Code Quality & Maintainability** - Cyclomatic complexity optimization
  - Refactored 12 high-complexity functions to improve readability and maintainability
  - Reduced main.go main() from complexity 22 → 7
  - Simplified parser.Parse() from complexity 20 → 6
  - Optimized ParseGemspec() from complexity 22 → 5
  - Decomposed handleAnalysisComplete() and other UI handlers for better separation of concerns
  - All changes are structural only - no behavioral changes or API modifications
  - GoReport Card maintains A+ rating with optimized complexity scores

## [v1.2.0] - 2026-04-09

### Added
- **Gem Size Analysis** (Issue #53)
  - New "Sanity" tab displays total project gem directory sizes
  - Automatically detects Ruby version manager (frum, rbenv, rvm, asdf, chruby, system)
  - Shows individual gem sizes with human-readable formatting (KB, MB, GB)
  - Gems organized by dependency type (direct vs transitive)
  - Interactive `gem info` modal accessible with Enter/i keys
  - Async background loading prevents UI blocking
  - Integrates with existing size calculation infrastructure
- **Insecure Gem Source Detection** (Issue #54)
  - Detects gems sourced from insecure protocols (http://, git://, git+http://)
  - Displays insecure sources in Project tab with source URL for each gem
  - Includes insecure source information in all export formats (text, CSV, JSON)
  - Text reports show dedicated "INSECURE GEM SOURCES" section with security guidance
  - CSV and JSON exports include source and insecure flag for each gem
  - Helps security teams identify gems from untrusted or unencrypted sources

## [Unreleased]

### Added
- **CVE Advisory Comments** (Issue #55) - Add custom rationale and comments to CVE advisories
  - Store project-specific CVE assessment and remediation notes in `.gemtracker_comments.json`
  - Two comment types: "acknowledged" for assessed vulnerabilities, "ignored" for accepted risks
  - Edit comments with 'c' key on any CVE in the CVE tab
  - Comments persist across scans and share findings with team members
  - Helps track which vulnerabilities have been evaluated and why
- **Remediation Advice for CVEs** - Display recommended remediation and workarounds when available
  - Shows both mitigations and workarounds from OSV vulnerability details
  - Accessible in CVE info modal (press 'i' on a CVE to view details)
  - Note: Users should follow the official CVE link to verify complete and current remediation guidance
- **BubbleTea v2 Upgrade** - Updated TUI framework and components to latest version
  - charm.land/bubbletea/v2 v2.0.2
  - charm.land/bubbles/v2 v2.1.0
  - charm.land/lipgloss/v2 v2.0.2
  - Improved keyboard event handling and View API

### Fixed
- **CVE Tab Loading State** - Show animated loading indicator during vulnerability scans
  - Displays spinning animation instead of misleading "safe" message while scanning
  - Improves UX when opening CVE tab during initial scan or cache refresh
- **CVE Cache Performance** - Instant display of cached vulnerabilities without blocking UI
  - Removed synchronous enrichment during cache hits that blocked for 1-2 seconds
  - Cached vulnerabilities now display immediately while enrichment happens in background
  - Vulnerability details and workarounds fetch asynchronously for seamless experience

## [v1.1.8] - 2026-04-09

### Added
- **Report Generation Progress Feedback** - Real-time progress indicators during report analysis
  - Parse status with gem count displayed
  - Live counter during outdated gem checks showing (N/total) progress
  - Vulnerability scan cache hit/miss detection
  - Final report destination confirmation
  - All progress output goes to stderr to prevent polluting report content
- **Redesigned Text Report Layout** - Improved readability and structure for `--report text` output
  - Gems grouped by bundle group (default, development, test, staging, production, etc.)
  - Clear [direct] vs [transitive] dependency markers
  - Shows reverse dependencies: "[used by: X, Y, Z]" indicating which gems depend on each gem
  - Gem groups marked with "-" when not identified (typically transitive gems)
  - Applied to both "OUTDATED GEMS" and "ALL GEMS" sections
- **Output File Overwrite Protection** - Interactive prompt when report output file already exists
  - Options to Replace, Cancel, or specify a new filename
  - Automatically preserves file extension when entering new filename
  - Prevents accidental loss of existing report files
- **Redesigned Vulnerable Gems Section** - Enhanced clarity for vulnerability information in text reports
  - Full OSV advisory URL (https://osv.dev/vulnerability/GHSA-...) displayed for each vulnerability
  - Shows bundle group membership (default, development, test, etc.)
  - Clear [direct] vs [transitive] markers to indicate gem scope
  - For transitive vulnerabilities: lists which gems depend on the vulnerable gem
  - Removed trailing colon after severity level for cleaner formatting

### Fixed
- **Project Name Display in Reports** - Reports now show the actual project directory name instead of "."
  - Extracts project directory name from absolute Gemfile.lock path
  - Shows project name (e.g., "listopia") instead of full path or relative "."

## [v1.1.7] - 2026-04-09

### Added
- **CVE Info Modal with Scrolling** - Interactive modal for viewing detailed CVE information
  - Scroll through detailed vulnerability information with arrow keys or Home/End keys
  - Modal stays within terminal bounds and displays complete information
- **Workarounds Support** - Display CVE workarounds extracted from OSV vulnerability details
  - Temporary mitigations shown when available
  - Helps users take action before upgrading dependencies
- **Browser Link Support** - Press 'o' in CVE info modal to open vulnerability link in default browser
  - Quickly access OSV.dev vulnerability page for additional details
- **CVSS-Based Severity Mapping** - Severity levels now derived from CVSS scores
  - Ensures accuracy of vulnerability severity classification
  - Maps CVSS v3.1 scores: 9.0+ → CRITICAL, 7.0-8.9 → HIGH, 4.0-6.9 → MEDIUM, 0.1-3.9 → LOW

### Fixed
- **OSV API Vulnerability Scanning** - Fixed broken vulnerability detection from OSV.dev
  - Corrected batch query endpoint from `/v1/query/batch` to `/v1/querybatch` (actual OSV.dev API)
  - Vulnerability scanning now works correctly for all gems in the project
  - CVE tab now displays detected vulnerabilities as expected
- **Vulnerability Severity Accuracy** - Fixed all vulnerabilities showing as MEDIUM
  - Corrected extraction of severity from OSV API response (`database_specific.severity` field)
  - Severity now properly extracted from OSV individual vulnerability detail endpoint
  - Example: addressable gem now correctly shows HIGH severity instead of always showing MEDIUM
  - All 87 CVEs in standard project now display correct severity levels
- **CVE Severity in Exports** - Severity and CVSS scores now included in exported reports
  - Text, CSV, and JSON exports now include CVE severity level and CVSS score
  - Exports now match the CVE information displayed in the UI
  - Format: `CVE-ID [SEVERITY]: Description (CVSS: score)`

### Changed
- **Code Cleanup** - Removed unused vulnerability checker code
  - Deleted hardcoded `VulnerabilityChecker` class (replaced by OSVClient with live API)
  - Removed associated test suite for dead code
  - Retained `Vulnerability` struct for API response handling

## [v1.1.6] - 2026-04-08

### Fixed
- **GoReleaser v2.15.2 Compatibility** - Fixed release pipeline by updating homebrew configuration field
  - Renamed `homebrew_taps` to `brews` to match GoReleaser v2.x configuration format
  - Release builds now complete successfully

## [v1.1.5] - 2026-04-08

### Added
- **Vulnerability Tracking from OSV** - Now tracking vulnerabilities from https://osv.dev/
  - Enhanced vulnerability database integration for comprehensive security insights
  - Provides broader coverage of known security issues in gem dependencies

### Fixed
- **Header Display on Gems Tab** - Fixed minor issue preventing the header line from being displayed on the Gems tab view
  - Header with version, path, and tab navigation now properly visible

## [v1.1.4] - 2026-04-08

### Fixed
- **Enable Sentry Error Tracking in Release Builds** (Issue #50)
  - Pass SENTRY_DSN secret to GoReleaser build environment
  - Released binaries now have error tracking activated by default
  - Users downloading from GitHub Releases get Sentry support out-of-the-box
  - Local development builds without SENTRY_DSN still work (Sentry disabled as expected)

## [v1.1.3] - 2026-04-08

### Improved
- **Filter View UI Enhancements** (Issue #47)
  - Convert filter view to modal overlay for improved UX
  - Filter menu now appears as a centered modal box overlaid on the gem list instead of replacing the entire screen
  - Users maintain visual context with the gem list visible in the background
  - Fixes layout issue where action hints and statusbar were pushed off-screen

### Changed
- **Filter View Checkboxes** - Improved visibility and clarity
  - Replace tiny checkbox symbols (☑/☐) with larger, color-coded alternatives
  - Selected items: `[✓]` in green (`ColorSuccess`)
  - Unselected items: `[ ]` in muted gray (`ColorTextMuted`)
  - Add cursor indicator (›) to highlight currently selected option
  - Add footer hint showing keyboard shortcuts within modal box

### Technical
- Add `placeOverlay()` helper function for ANSI-aware modal rendering
- Import `github.com/charmbracelet/x/ansi` for ANSI escape sequence handling
- Modal positioned using center calculation based on terminal dimensions
- Modal styling: rounded border with accent color and surface background

## [v1.1.2] - 2026-04-07

### Added
- **Claude Code Skill (gem-check)** - Interactive gem analysis directly in Claude Code
  - Run `/gem-check` to analyze gem dependencies with AI assistance
  - Security-first vulnerability detection with severity prioritization
  - Smart gem update prioritization (first-level > transitive, production > dev)
  - Real-world upgrade workflow guidance with worked examples
  - Interactive follow-up questions for specific gems and upgrade strategies
  - Professional skill documentation with API reference, examples, and scenarios
  - MIT-licensed skill ready for distribution
- **Non-Interactive CLI Export Reports** - Generate gem reports in multiple formats for CI/CD pipelines (Issue #35)
  - Three export formats: `text` (human-readable), `csv` (compliance-friendly), `json` (machine-readable)
  - `--report FORMAT` flag generates report and exits (non-interactive mode)
  - `--output PATH` flag saves report to file (defaults to stdout)
  - Compatible with all major CI/CD platforms (GitHub Actions, CircleCI, Travis, GitLab)
  - Full vulnerability and outdated gem detection in reports
  - Proper exit codes for CI/CD integration (0 on success, 1 on errors)
  - Supports `--verbose` logging in non-interactive mode
  - Example: `gemtracker --report csv --output gems-report.csv` or `gemtracker --report json | jq '.summary'`
- **Support for Alternative Bundler Conventions** - Added support for `gems.locked` and `gems.rb` file naming (Issue #26)
  - Detect and parse `gems.locked` (identical structure to Gemfile.lock)
  - Detect and parse `gems.rb` (identical syntax to Gemfile)
  - Search priority: gems.locked/gems.rb preferred over Gemfile.lock/Gemfile
  - Display which file was loaded in the UI for transparency
- **Gem Dependency Parsing from .gemspec Files** - Parse gem dependencies from `.gemspec` files for gem projects (Issue #37)
  - Extract `add_runtime_dependency` and `add_development_dependency` declarations
  - Support version constraints (e.g., `>= 2.0`, `~> 3.1`)
  - Display dependencies with type badges (`[runtime]` vs `[dev]`)
  - Show version constraints from unresolved gemspec declarations
  - Automatic fallback to Gemfile.lock when available for resolved versions

### Technical
- Created `internal/gemfile/gemspec_parser.go` for `.gemspec` file parsing
- Extended file detection logic to support alternative file naming conventions
- Improved robustness of dependency tree structures to handle unresolved constraints

## [v1.1.1] - 2026-04-06

### Added
- **Manual Health Refresh** - Press `r` in gem list to manually refresh health data with progress indicator (Issue #28)
- **GitHub GraphQL Batch Fetching** - All gem repositories fetched in single GraphQL batch request instead of sequential REST calls
  - Dramatically reduces API calls (from ~189 to 1-2 GraphQL requests)
  - Works without GITHUB_TOKEN using RubyGems data only
  - With GITHUB_TOKEN: even richer data with higher rate limits (5000/hr vs 60/hr)

### Fixed
- **Health Indicators Disappearing on Tab Switch** - Fixed issue #29 where health dots would disappear when navigating between tabs during background fetch
  - Health cache now loaded on startup, making cached data immediately available
  - Dots persist when switching tabs mid-fetch
- **Health Cache Never Used** - Cache was written but never read; now properly loaded on app startup for instant results

### Changed
- **Health Cache TTL Extended** - From 24 hours to 12 days
  - Health metrics change on year timescale; unnecessary API calls reduced dramatically
  - Next run within 12 days gets instant health indicators from cache
- **Rate Limit Handling** - Rate-limited gems now marked as HealthUnknown; queue continues instead of halting
  - Users see partial health data instead of complete halt when GitHub rate limit hit
  - Gem health is now repo-level (cached by gem name), so version upgrades reuse cached data

### Technical
- Added `RepoOwnerPair` struct and `FetchGitHubBatch()` for GraphQL batching in health.go
- Extended health cache TTL to 12 days, added `ClearHealth()` function
- Cache now loaded during analysis startup (`handleAnalysisComplete`)
- GitHub batch fetch runs before per-gem RubyGems owner fetching
- Rate-limited gems set to HealthUnknown with RateLimited flag instead of halting queue
- All gems (not just first-level) now get health data cached and searchable

### Tests
- Added comprehensive tests for `ComputeHealthScore` covering all health tiers
- Added tests for `ExtractGitHubOwnerRepo` with various GitHub URL formats

## [v1.1.0] - 2026-04-05

### Added
- **Standardized Layout System** - Unified height calculation across all views for consistent and predictable UI
- **Improved Statusbar Layout** - Status indicators (Fetching health, Checking updates, etc.) now display on separate line below keyboard hints for cleaner UX

### Fixed
- **Missing Header and Tabbar** - Fixed critical issue where gemtracker version, project path, and tab navigation were not displaying on some views
- **Statusbar Visibility** - Resolved statusbar being cut off on tabs with long content lists (e.g., gems table with many entries)
- **Height Calculation** - Corrected contentHeight calculation to properly account for header (1), tabbar (1), statusbar (1-3 lines), and update notifications
- **View Composition** - Refactored from nested `lipgloss.JoinVertical` calls to single top-level assembly for proper line rendering
- **Status Indicators Layout** - Separated status indicators from keyboard hints to prevent text overflow and improve readability
- **Line Splitting Logic** - Fixed line truncation to prevent extra empty elements that would cause total height to exceed terminal size
- **Content Overflow Prevention** - Implemented smart truncation that preserves statusbar when content would exceed terminal height

### Technical
- Created `assembleViewWithChrome()` helper function for consistent view assembly across all screen types
- Added `statusBarTotalHeight()` function to calculate complete statusbar height including status indicator lines
- Updated all 10 view functions (viewGemList, viewGemDetail, viewSearch, viewUpgradeable, viewCVE, viewProjectInfo, viewLoading, viewSelectPath, viewFilterMenu, viewError) to use standardized layout approach
- Improved line truncation logic with explicit height limits and content preservation

## [v1.0.8] - 2026-04-05

### Added
- **Gem Health Indicators** - Maintenance status tracking for first-level gems
  - Shows health status (🟢 HEALTHY, 🟡 WARNING, 🔴 CRITICAL) as colored dot in gem list
  - Health section in gem detail view with detailed statistics
  - Last release date, GitHub stars, open issues, maintainer count, archived status
  - Health data fetched from RubyGems and GitHub APIs asynchronously
- **Health Data Caching** with 24-hour TTL
  - Separate cache at `~/.cache/gemtracker/{hash}_health.json`
  - Instant results on subsequent runs within 24 hours
  - Graceful fallback if GitHub rate limited (60 req/hr anonymous limit)
- **Improved CLI Help**
  - Display version information at top of help
  - Show GitHub repository link
  - Document `--no-cache` option
  - Better organized options section

### Technical
- Sequential async health data loading (one gem at a time)
- Respects GitHub's anonymous API rate limits
- Progressive health dot filling in gem list as data arrives
- Robust error handling for network and rate limit scenarios

## [v1.0.7] - 2026-04-05

### Changed
- **[Release Process]** Improved GoReleaser configuration for better build pipeline reliability

## [v1.0.6] - 2026-04-05

### Added
- **[Project]** tab - Display project metadata and statistics
  - Ruby version extraction from Gemfile.lock
  - Bundle version detection
  - Framework detection (Rails, Sinatra, Hanami, Roda, Cuba, Grape) with version
  - Gem statistics (total, direct dependencies, transitive dependencies)
  - Vulnerability summary
- Gem filtering on Gems tab
  - Filter by gem group (default, development, test, production, etc.)
  - Filter to show only upgradable gems
  - Visual filter status indicator with clear shortcut
  - Dedicated filter menu UI with keyboard shortcuts:
    - `f` - Open filter menu
    - `u` - Toggle upgradable-only filter
    - `c` - Clear all filters
- Optional Sentry error tracking
  - Enabled via `SENTRY_DSN` environment variable (completely optional)
  - Not required for development or self-built versions
  - Helps track bugs and crashes in production builds
- Analysis caching for faster subsequent loads
  - Automatic cache storage in `~/.cache/gemtracker/`
  - Cache invalidation based on Gemfile.lock modification time
  - Instant project reload if Gemfile.lock unchanged

## [v1.0.5] - 2026-04-04

### Fixed
- GoReleaser configuration format for v2.x compatibility (migrate from `homebrew` to `brews` field)

## [v1.0.3] - 2026-04-04

### Added
- New version available notification at bottom of UI (similar to Claude Code)
  - Asynchronously checks GitHub for latest releases on startup
  - Platform-aware upgrade instructions (brew for macOS, download link for others)
  - Gracefully handles network failures
- Multi-platform distribution (macOS, Linux, Windows)
- Automated GitHub Actions CI/CD pipeline
- Homebrew formula for easy macOS installation
- SECURITY.md for vulnerability reporting policy
- CHANGELOG.md for version tracking

### Fixed
- App header displaying duplicate 'v' in version string (was showing "gemtracker vv1.0.0")

## [v1.0.2] - 2026-04-04

### Added
- Initial public release
- Interactive Terminal UI for analyzing Ruby gem dependencies
- **[Gems]** screen - First-level gem list with version info and update status
- **[Search]** screen - Real-time gem search across all dependencies
- **[CVE]** screen - Vulnerability detection and reporting
- Gem details view with forward and reverse dependency trees
- Version checking - See installed vs latest available versions
- Vulnerability detection - Identify known CVEs in dependencies
- Group-based analysis - Understand gem scope (default, development, test, production)
- Direct links to rubygems.org and GitHub repositories
- Keyboard navigation (arrow keys, Tab, Enter, Esc, q)
- Support for analyzing Gemfile.lock files
- Cross-platform support (macOS, Linux, Windows)

### Technical
- Built with Go 1.24
- Uses BubbleTea for terminal UI
- Minimal dependencies (charmbracelet packages only)
- Efficient dependency parsing and analysis


---
## Legend

- **Added** for new features
- **Changed** for changes in existing functionality
- **Deprecated** for soon-to-be removed features
- **Removed** for now removed features
- **Fixed** for any bug fixes
- **Security** in case of security vulnerabilities
- **Technical** for internal improvements with no user impact

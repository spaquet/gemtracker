# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

## How to Report Issues

Found a bug? Please open an issue on [GitHub](https://github.com/spaquet/gemtracker/issues).

## How to Suggest Features

Have an idea for a new feature? Open a [discussion](https://github.com/spaquet/gemtracker/discussions) or [issue](https://github.com/spaquet/gemtracker/issues) on GitHub.

## Security

For security vulnerabilities, please see [SECURITY.md](SECURITY.md) for responsible disclosure.

---

## Version History

### Planned for Future Releases

- [ ] Live CVE database updates
- [ ] Support for Gemfile global options and git/path sources
- [ ] License compliance checking
- [ ] CI/CD integration mode
- [ ] Custom vulnerability filtering
- [ ] Dependency graph visualization
- [ ] Export functionality (JSON, CSV)

---

## Legend

- **Added** for new features
- **Changed** for changes in existing functionality
- **Deprecated** for soon-to-be removed features
- **Removed** for now removed features
- **Fixed** for any bug fixes
- **Security** in case of security vulnerabilities
- **Technical** for internal improvements with no user impact

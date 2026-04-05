# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

# Gem Check - API Reference

Detailed documentation for using the `gemtracker` CLI tool that powers the gem-check skill.

## Installation

### macOS / Linux (Homebrew - Recommended)
```bash
brew tap spaquet/gemtracker
brew install gemtracker
```

### Windows (Direct Download)
1. Download from [GitHub Releases](https://github.com/spaquet/gemtracker/releases)
2. Extract ZIP file
3. Add to PATH

### All Platforms (Build from Source)
```bash
git clone https://github.com/spaquet/gemtracker
cd gemtracker
make build
```

## CLI Commands

### Basic Usage
```bash
gemtracker [path-to-gemfile-lock-or-directory]
```

**Arguments:**
- `[path]` - Optional. Path to project directory or Gemfile.lock file
  - If omitted: Uses `./Gemfile.lock` in current directory
  - If directory: Looks for `Gemfile.lock` in that directory
  - If file: Analyzes that specific Gemfile.lock

**Examples:**
```bash
gemtracker                        # Analyze ./Gemfile.lock
gemtracker /path/to/rails-app    # Analyze in specific directory
gemtracker Gemfile.lock          # Explicit file path
```

### JSON Report Output
```bash
gemtracker --report json [path]
```

Returns structured JSON with:
- Vulnerability data (CVEs, severity, affected versions)
- Outdated gems (current version, latest available, gem type)
- Health status (maintenance score, maintainer count, last activity)
- Gem tree (dependencies, reverse dependencies)

**Output Schema:**
```json
{
  "gems": [
    {
      "name": "rails",
      "version": "7.0.0",
      "latest_version": "8.1.3",
      "group": "default",
      "is_first_level": true,
      "vulnerabilities": [
        {
          "id": "CVE-2024-12345",
          "title": "SQL injection in...",
          "severity": "HIGH",
          "affected_versions": ["<8.0.0"]
        }
      ],
      "health": {
        "score": 95,
        "status": "HEALTHY",
        "last_release": "2026-03-15",
        "maintainers": 5,
        "last_activity": "2026-04-01"
      },
      "dependencies": ["activesupport", "actionpack"],
      "required_by": []
    }
  ],
  "summary": {
    "total": 189,
    "first_level": 63,
    "transitive": 126,
    "with_vulnerabilities": 2,
    "outdated": 23,
    "health_concerns": 3
  }
}
```

### Interactive Terminal UI
```bash
gemtracker
```

Launches interactive TUI with:
- **Arrow keys** - Navigate gem list
- **Enter** - View gem details and dependencies
- **/[query]** - Search for gems
- **Tab** - Switch views (gems, dependencies, vulnerabilities)
- **q** - Quit
- **?** - Help menu

## Understanding Report Data

### Vulnerability Severity Levels
- **CRITICAL** - Actively exploited, widespread impact, should update immediately
- **HIGH** - Serious vulnerability affecting security/stability
- **MEDIUM** - Moderate risk, update within sprint
- **LOW** - Minor vulnerability, can defer

### Gem Groups (Priority)
1. **default** - Used in production (highest priority)
2. **production** - Explicitly marked for production
3. **development** - Dev/tools only
4. **test** - Test suite only

### Health Status
- 🟢 **HEALTHY** - Active maintenance, 2+ maintainers
- 🟡 **WARNING** - Inactive 1-3 years OR single maintainer
- 🔴 **CRITICAL** - Inactive 3+ years, archived, or disabled

### Gem Types
- **First-level** - Directly listed in Gemfile (you control)
- **Transitive** - Dependencies of other gems (indirect)

## API Rate Limits

The tool uses:
- **RubyGems API** - No strict limits
- **GitHub API** - 60 requests/hour (unauthenticated), 5000/hour (authenticated)

If rate-limited, data is served from local cache (24-hour TTL per gem).

## Caching

Results cached in `~/.cache/gemtracker/`:
- Version check results (24-hour TTL)
- Health data per gem (24-hour TTL)
- Note: Cache persists between runs but expires after 24 hours

To clear cache:
```bash
rm -rf ~/.cache/gemtracker/
```

## Troubleshooting

### "Gemfile.lock not found"
```bash
gemtracker /path/to/correct/project
```
Make sure the directory contains a valid Gemfile.lock file.

### Network errors / API timeouts
- Check internet connection
- Tool will use cached data if available
- Try again in a few minutes

### Rate-limited by GitHub API
- Add GitHub token for higher limits
- Or wait 1 hour for reset
- Cached data will be served in the meantime

## Limitations

- Analysis based on data from rubygems.org and GitHub APIs
- CVE database is curated (not a real-time comprehensive feed)
- Very large projects (500+ gems) may take longer
- Requires network access for full analysis (graceful degradation to cache)

## Complete JSON Output Schema

The `--report json` output includes all gem data in this structure:

```json
{
  "generated_at": "2026-04-06T10:00:00Z",
  "project_path": "/path/to/project",
  "summary": {
    "total_gems": 189,
    "first_level_gems": 63,
    "outdated_count": 45,
    "vulnerable_count": 3
  },
  "gems": [
    {
      "Name": "rails",
      "Version": "7.0.0",
      "Groups": ["default"],
      "IsFirstLevel": true,
      "IsOutdated": true,
      "LatestVersion": "8.1.3",
      "IsVulnerable": false,
      "VulnerabilityInfo": "",
      "HomepageURL": "https://rubyonrails.org",
      "Description": "Web development framework for Ruby"
    }
  ]
}
```

### Field Reference

| Field | Type | Purpose | Use Case |
|-------|------|---------|----------|
| `Name` | string | Gem package name | Identification, version control |
| `Version` | string | Currently installed version | Risk assessment, changelog lookup |
| `Groups` | array | Gem scope: `["default"]`, `["test"]`, `["development"]`, `["production"]` | Severity filtering (prod > dev) |
| `IsFirstLevel` | boolean | Directly required vs transitive dependency | Priority ranking (direct > transitive) |
| `IsOutdated` | boolean | Update available | Upgrade candidate identification |
| `LatestVersion` | string | Most recent available version | Upgrade target version |
| `IsVulnerable` | boolean | Known CVE in this version | Security flagging |
| `VulnerabilityInfo` | string | CVE ID and description | Security alert details |
| `HomepageURL` | string | Gem repository link | Additional research |
| `Description` | string | Gem purpose and functionality | Context for recommendations |

### JSON Processing Examples

**Get all vulnerable gems:**
```bash
gemtracker --report json . | jq '.gems[] | select(.IsVulnerable)'
```

**Get vulnerabilities in production:**
```bash
gemtracker --report json . | jq '.gems[] | select(.IsVulnerable and (.Groups | index("default")))'
```

**Get outdated first-level gems:**
```bash
gemtracker --report json . | jq '.gems[] | select(.IsOutdated and .IsFirstLevel)'
```

**Count gems by group:**
```bash
gemtracker --report json . | jq '[.gems[].Groups[]] | group_by(.) | map({group: .[0], count: length})'
```

## Severity Assessment Framework

Use this framework to prioritize updates:

### Vulnerability Severity

**CRITICAL** (act immediately):
- IsVulnerable = true
- Groups includes "default"
- IsFirstLevel = true
- Action: Update before next deploy

**HIGH** (address this sprint):
- IsVulnerable = true
- Groups includes "default"
- IsFirstLevel = false (transitive)
- Action: Update through direct dependency

**MEDIUM** (plan for next sprint):
- IsVulnerable = true
- Groups = ["development"] OR ["test"]
- Action: Update in development cycle

**LOW** (track but not urgent):
- IsVulnerable = true
- Groups = ["test"] only
- Action: Can defer unless security-focused

### Outdated Gem Priority

**HIGH PRIORITY:**
- IsFirstLevel = true
- IsOutdated = true
- Groups includes "default"
- Details: These are your core dependencies

**MEDIUM PRIORITY:**
- IsFirstLevel = true
- IsOutdated = true
- Groups = ["development"] OR ["test"]
- Details: Still direct deps, but lower impact

**LOW PRIORITY:**
- IsFirstLevel = false
- IsOutdated = true
- Any group
- Details: Often auto-update when direct deps update

### Version Jump Assessment

**Major Jump** (e.g., 6.x → 8.x):
- Risk: High
- Requires: Full changelog review, extensive testing
- Recommendation: Plan for this sprint, test thoroughly

**Minor Jump** (e.g., 7.1 → 7.2):
- Risk: Moderate
- Requires: Review changelog for breaking changes
- Recommendation: Standard testing should suffice

**Patch Jump** (e.g., 7.1.2 → 7.1.3):
- Risk: Low
- Requires: Verification it installs
- Recommendation: Safe to update

## Framework-Specific Patterns

### Rails Projects

Watch for framework alignment. These gems should have matching versions:
- rails
- railties
- actionpack
- activerecord
- actionview
- activesupport

Mismatched versions indicate incomplete upgrade.

### Security-Critical Gems

These deserve priority when outdated:
- devise (authentication)
- bcrypt (password hashing)
- jwt (token authentication)
- rack (HTTP handling)
- rails (framework security)
- net-http (HTTP client)

Recommend: Update these first, before other gems.

## Exit Codes

- `0` - Success (analysis completed)
- `1` - Error (file not found, invalid Gemfile.lock, etc.)

## Performance Notes

- First run: 5-30 seconds (fetches from rubygems.org)
- Subsequent runs: Near instant (cached locally)
- Cache location: `~/.cache/gemtracker/`
- Cache invalidates when `Gemfile.lock` is modified

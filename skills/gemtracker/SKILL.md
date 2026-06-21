---
name: gemtracker
description: Analyze Ruby gem dependencies, vulnerabilities, and outdated packages
version: 1.0.0
author: Stephane Paquet
license: MIT
---

# Gemtracker Skill

Analyze Ruby gem dependencies directly in Claude Code using the gemtracker CLI.

## Features

- **Vulnerability Scanning** - Detect CVEs in gem dependencies via OSV.dev
- **Outdated Detection** - Find gems with available updates
- **Health Status** - Monitor gem maintenance and activity
- **Insecure Sources** - Flag gems from unencrypted sources
- **Structured Output** - Machine-readable JSON for automation

## Requirements

- `gemtracker` CLI installed and in PATH
- Ruby project with `Gemfile.lock`, `gems.locked`, or `.gemspec`

## Installation

```bash
# Install gemtracker (requires Go 1.24+)
brew install spaquet/gemtracker/gemtracker

# Or build from source
git clone https://github.com/spaquet/gemtracker
cd gemtracker
make build-release
```

## Usage

### Basic Analysis

Analyze current directory's Ruby dependencies:

```
/gemtracker
```

Analyze specific project:

```
/gemtracker /path/to/ruby-project
```

### Output Formats

**JSON** (recommended for automation):
```
/gemtracker . --report json
```

**Text** (human-readable):
```
/gemtracker . --report text
```

**CSV** (spreadsheet import):
```
/gemtracker . --report csv
```

## Examples

### Check for vulnerabilities

Analyze the current project and report any CVEs found:

```
/gemtracker
```

Claude will parse the results and show:
- Count of vulnerable gems
- Severity levels (HIGH, MODERATE, LOW, CRITICAL)
- CVSS scores
- Links to advisories

### Find outdated dependencies

```
/gemtracker . --report json
```

Shows:
- Gems with available updates
- Current vs latest versions
- Whether updates are within version constraints

### Audit insecure sources

Detects gems installed from unencrypted HTTP or git:// protocols.

## Integration with Hooks

Add pre-commit hook to your project to check dependencies before committing:

```bash
# In .claude/settings.json
{
  "hooks": {
    "before-commit": "gemtracker . && echo 'Gem analysis complete'"
  }
}
```

## Caching

Results are cached for 24 hours in `~/.cache/gemtracker/`. Clear with:

```bash
rm -rf ~/.cache/gemtracker/
```

## Troubleshooting

**Command not found**: Ensure gemtracker is in your PATH
```bash
which gemtracker
```

**No dependency files found**: Place Gemfile.lock, gems.locked, or .gemspec in project root

**Rate limits**: External API calls (RubyGems, GitHub, OSV.dev) may be rate-limited. Wait and retry.

## See Also

- [Gemtracker README](https://github.com/spaquet/gemtracker)
- [Installation Guide](./INSTALLATION.md)

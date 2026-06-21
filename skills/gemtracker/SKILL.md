---
name: gemtracker
description: Analyze Ruby gem dependencies, vulnerabilities, and outdated packages
---

# Gemtracker Skill

Analyze Ruby gem dependencies and security risks using the gemtracker CLI.

## Features

- Detect CVEs in gem dependencies
- Find outdated packages with available updates
- Monitor gem maintenance health
- Identify insecure gem sources
- Output in text, JSON, or CSV formats

## Requirements

Install the `gemtracker` CLI first:

```bash
brew install spaquet/gemtracker/gemtracker
```

See [Installation Guide](https://github.com/spaquet/gemtracker/blob/main/INSTALLATION.md) for detailed setup.

## Usage

Analyze current directory:
```
/gemtracker
```

Analyze specific project:
```
/gemtracker /path/to/ruby-project
```

Output formats:
```
/gemtracker . --json      # Machine-readable output
/gemtracker . --csv       # Spreadsheet import
```

## Examples

**Find vulnerabilities:**
```
/gemtracker
```
Shows CVEs found, severity levels, and advisory links.

**Check outdated gems:**
```
/gemtracker . --json
```
Returns gems with available updates and version constraints.

**Audit gem sources:**
```
/gemtracker
```
Detects unencrypted HTTP or git:// source gems.

## Caching

Results cached 24 hours in `~/.cache/gemtracker/`. Clear:
```bash
rm -rf ~/.cache/gemtracker/
```

## Troubleshooting

**gemtracker not found**: Verify installation
```bash
which gemtracker
```

**No dependency files**: Ensure Gemfile.lock, gems.locked, or .gemspec exists in target directory

**Rate limits**: API calls may throttle. Wait 1 hour and retry.

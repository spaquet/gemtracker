# gem-check Claude Code Skill

A Claude Code skill that analyzes Ruby gem dependencies for security vulnerabilities, outdated packages, and maintenance health. Powered by **gemtracker**.

## Overview

This skill integrates the `gemtracker` CLI tool into Claude Code, making it easy to:

- 🔴 **Detect CVE vulnerabilities** in gems
- 🟡 **Identify outdated packages** with available updates
- 🟠 **Assess maintenance health** of first-level dependencies
- 📋 **Generate actionable reports** with prioritized recommendations
- 🚀 **Plan gem upgrades** with context and testing guidance

## Installation

1. **Ensure gemtracker is installed**:
   - macOS/Linux: `brew tap spaquet/gemtracker && brew install gemtracker`
   - Windows: Download from [GitHub Releases](https://github.com/spaquet/gemtracker/releases)

2. **Install the skill in Claude Code**:
   ```bash
   # Via skill marketplace (if available)
   /plugin skill install gem-check

   # Or manually add to your project
   cp -r gem-check/ /path/to/project/.claude/skills/
   ```

## Quick Start

```bash
/gem-check
```

The skill will:
1. ✓ Check if gemtracker is installed
2. ✓ Find your project's Gemfile.lock
3. ✓ Run vulnerability and outdated checks
4. ✓ Generate a prioritized report
5. ✓ Ask what you'd like to update

## File Structure

```
gem-check/
├── SKILL.md                              # Main skill definition and docs
├── README.md                             # This file
├── references/
│   ├── gemtracker-json-schema.md        # JSON output format and fields
│   └── analysis-framework.md            # Decision trees and assessment logic
└── examples/
    ├── scenario-security-focus.md       # Example: Security vulnerabilities
    ├── scenario-healthy-project.md      # Example: Well-maintained project
    ├── scenario-dependency-debt.md      # Example: High technical debt
    ├── common-workflows.md              # 10 real-world workflows
    └── sample-output.json               # Example gemtracker JSON output
```

## What's Included

### Main Skill (SKILL.md)
- Complete user interface documentation
- Installation instructions
- Example outputs and workflows
- Tips and best practices
- Limitations and known issues

### References
- **gemtracker-json-schema.md**: Complete JSON field reference
  - What each field means
  - How to use groups for prioritization
  - Common filtering patterns with jq

- **analysis-framework.md**: Decision logic and frameworks
  - Severity assessment templates
  - Version jump assessment
  - Framework-specific patterns
  - Health scoring logic
  - Decision trees for recommendations

### Examples
- **parse-vulnerabilities.sh**: Bash script showing vulnerability extraction
- **analyze-gems.py**: Python script for full analysis and reporting
- **analyze-gems.js**: Node.js script that generates AI-friendly output
- **sample-output.json**: Example JSON from gemtracker (for testing/learning)

## Understanding the Examples

### Scenario Examples
See what `/gem-check` output looks like in different situations:

- **scenario-security-focus.md** - Project with CVEs that need attention
- **scenario-healthy-project.md** - Well-maintained project with no issues
- **scenario-dependency-debt.md** - Project with significant outdated gems

Each shows:
- Example report output
- How to interpret results
- Next steps you might take
- Key insights and actions

### Common Workflows
Real-world ways to use the skill:
- Quick security checks
- Planning upgrade sprints
- Safe major version updates
- Handling unmaintained gems
- Regular maintenance routines
- CI/CD integration
- Pre-release audits

See `common-workflows.md` for 10 detailed examples.

## Understanding the Workflow

### What Happens When You Run `/gem-check`

1. **Installation Check**
   - Verifies `gemtracker` is available in PATH
   - If not found, prompts with install instructions (OS-specific)

2. **Project Detection**
   - Finds `Gemfile.lock` in current or specified directory
   - Validates it's a valid Ruby project

3. **Analysis**
   - Runs `gemtracker --report json` to get structured data
   - Parses JSON to identify issues

4. **Reporting**
   - Shows vulnerabilities (severity-ordered)
   - Lists outdated gems (impact-ordered)
   - Flags health concerns (for first-level gems only)

5. **Recommendations**
   - Generates priority list based on impact
   - Estimates effort for each category
   - Presents summary and suggested actions

### What Happens Next

You can then:

- **Ask for specific help**: "How do I upgrade Rails safely?"
- **Update a specific gem**: "Update devise to 4.9.3"
- **Plan improvements**: "Create an upgrade plan for all critical gems"
- **Deep dive**: "What's the health status of thin?"

## Reference Files Guide

### When to Use gemtracker-json-schema.md
- You're writing a script to parse gemtracker output
- You need to know what each JSON field represents
- You want to filter results with jq
- You're building AI tools to analyze gems

### When to Use analysis-framework.md
- You're making decisions about which gems to update
- You need to prioritize security vs maintenance work
- You're building recommendation logic
- You want to understand severity assessment

### When to Use Example Scripts
- You want to see how to parse gemtracker output
- You're integrating with CI/CD pipelines
- You're building tools that consume gem analysis
- You need to learn different programming languages' approaches

## Common Tasks

### Extract all vulnerable gems
```bash
gemtracker --report json . | jq '.gems[] | select(.IsVulnerable)'
```

### Find vulnerabilities in production
```bash
gemtracker --report json . | jq '.gems[] | select(.IsVulnerable and (.Groups | index("default")))'
```

### Get outdated first-level gems
```bash
gemtracker --report json . | jq '.gems[] | select(.IsOutdated and .IsFirstLevel)'
```

### Calculate health score
```bash
gemtracker --report json . | jq '
  .summary as $s |
  .gems as $g |
  ((($g | length) - $s.vulnerable_count - $s.outdated_count) / ($g | length) * 100) | round
'
```

## Troubleshooting

### "gemtracker command not found"
Install gemtracker:
```bash
# macOS/Linux
brew tap spaquet/gemtracker
brew install gemtracker

# Windows: Download from GitHub Releases
# https://github.com/spaquet/gemtracker/releases
```

### "Gemfile.lock not found"
Make sure you're in a Ruby project directory with Gemfile.lock:
```bash
/gem-check /path/to/rails-app
```

### Rate limiting errors
GitHub API has limits (60/hour unauthenticated, 5,000/hour with token).
Set `GITHUB_TOKEN` before running:
```bash
export GITHUB_TOKEN="github_pat_..."
gemtracker
```

See [gemtracker README](https://github.com/spaquet/gemtracker#github-api-rate-limits--github_token) for details.

## Tips for Best Results

✅ **Do's:**
- Run on actual Gemfile.lock files
- Cache results locally to avoid API limits
- Use with GITHUB_TOKEN for large projects
- Reference the analysis-framework for decision-making
- Test example scripts before using in automation

❌ **Don'ts:**
- Don't run without checking if gemtracker is installed
- Don't assume all outdated gems need immediate updates
- Don't ignore gem groups (dev vs production matters)
- Don't skip testing when major version updating

## Further Reading

- **Full gemtracker documentation**: https://github.com/spaquet/gemtracker
- **AI Integration Guide**: https://github.com/spaquet/gemtracker/blob/main/AI_GUIDE.md
- **Installation Guide**: https://github.com/spaquet/gemtracker#installation

## Issues & Contributions

Found a bug? Have a suggestion?
- **gemtracker issues**: https://github.com/spaquet/gemtracker/issues
- **skill feedback**: Submit feedback in Claude Code

## Version

**gem-check skill**: 1.0.0
**Requires**: gemtracker 1.0.0+
**Last updated**: April 2026

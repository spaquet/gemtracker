---
name: gem-check
description: Analyze Ruby gem security vulnerabilities, outdated dependencies, and health status with actionable upgrade recommendations
---

# Gem Check Skill

Analyze your Ruby project's dependencies for security vulnerabilities, outdated gems, and maintenance concerns. This skill runs `gemtracker` to provide structured reports and helps prioritize which gems to update.

## How to Use This Skill

When using gem-check, follow these steps:

1. **Run the analysis** - Execute `/gem-check` to scan your project's Gemfile.lock
2. **Review findings** - I'll present vulnerabilities, outdated gems, and health concerns with severity levels
3. **Understand priorities** - Vulnerabilities first, then first-level gems, then transitive dependencies
4. **Ask follow-up questions** - For any gem, ask for help understanding changes, testing strategy, or upgrade assistance
5. **Take action** - Request specific updates or a complete upgrade plan based on your priorities

Key principle: **You decide what to update and when.** I can provide analysis, explain changes, help with conflicts, and suggest strategies—but you control the final decisions.

## What This Skill Does

When invoked on a Ruby project with `Gemfile.lock`, this skill:

1. **Detects gemtracker** - Checks if gemtracker is installed, prompts to install if needed
2. **Runs analysis** - Executes `gemtracker --report json` to get structured dependency data
3. **Highlights issues**:
   - 🔴 **Security vulnerabilities** (CVEs) in any gem
   - 🟡 **Outdated gems** with available updates
   - 🟠 **Health concerns** for first-level dependencies (unmaintained, few maintainers)
4. **Generates report** - Presents findings with severity levels and suggested actions
5. **Enables decisions** - Lets you choose which gems to update and get help with each upgrade

## Using This Skill

### Quick Start

```bash
/gem-check
```

This will analyze your current project's `Gemfile.lock` and report findings.

### With Specific Project Path

```bash
/gem-check /path/to/rails-app
```

## What You'll See

### Security Vulnerabilities Report

Shows all CVEs found in your gems:

```
🔴 SECURITY VULNERABILITIES (2 found)
─────────────────────────────────────

1. rack (2.1.2 → CRITICAL)
   CVE-2021-22942: HTTP request smuggling vulnerability
   Scope: default (PRODUCTION)
   Impact: DIRECT - First-level dependency

   → Upgrade to rack 2.2.4 or 3.0.0+

2. devise (4.8.0 → HIGH)
   CVE-2022-0000: Authentication bypass in certain configurations
   Scope: default (PRODUCTION)
   Impact: TRANSITIVE - Dependency of other gems
```

### Outdated Gems Report

Lists gems with available updates, prioritized by impact:

```
🟡 OUTDATED GEMS (23 found)
─────────────────────────────

HIGH PRIORITY (First-level, production use):
  • rails: 7.0.0 → 8.1.3 (major jump - requires testing)
  • devise: 4.8.0 → 4.9.3 (patch update - low risk)
  • pg: 1.3.0 → 1.5.4 (minor update - safe)

MEDIUM PRIORITY (First-level, other groups):
  • rspec: 3.10.0 → 3.13.0 (dev only)
  • pry: 0.14.0 → 0.14.2 (dev only)

LOW PRIORITY (Transitive dependencies):
  • bundler: 2.1.0 → 2.5.1 (23 transitive uses)
  • json: 2.6.0 → 2.6.3 (11 transitive uses)
```

### Health Status Report

Flags first-level gems with maintenance concerns:

```
🟠 MAINTENANCE CONCERNS (3 gems)
─────────────────────────────────

WARNING:
  • thin (1.8.1) - No releases for 3+ years
    Maintainers: 1 | Last commit: 2019
    → Consider: Use Puma or other maintained alternatives

CRITICAL:
  • net-ftp (0.2.0) - Archived repository
    Maintainers: 0 | No activity since 2021
    → Action: Plan migration to maintained alternative
```

### Summary & Recommendations

```
📊 PROJECT HEALTH SUMMARY
────────────────────────

Total gems: 189
First-level: 63
Transitive: 126

Status:
  ✓ 163 gems are healthy and up-to-date
  ⚠️ 23 gems have updates available
  🔴 2 gems have known vulnerabilities
  🟠 3 gems show maintenance concerns

RECOMMENDED ACTIONS (prioritized):
1. IMMEDIATE: Update 'rack' to patch CVE-2021-22942
2. THIS SPRINT: Update Rails (major version change, needs testing)
3. NEXT SPRINT: Update remaining 21 gems
   → Focus on first-level dependencies first
   → Transitive updates often happen automatically

Estimated effort:
  • Critical fixes: 1-2 hours
  • Rails upgrade: Full sprint
  • Other updates: 4-6 hours
```

## After You Get Results

### Option: Get Help with Specific Updates

Ask for help updating a specific gem:

```
Help me update rails from 7.0.0 to 8.1.3
```

I can:
- Show the changelog for breaking changes
- Suggest testing strategy
- Help you resolve dependency conflicts
- Point out gems that may need coordinated updates

### Option: Update One Gem

Ask to update a specific gem:

```
Update devise to 4.9.3
```

I can:
- Update `Gemfile` with new version
- Run bundle install
- Show what changed in the update
- Help verify the update works

### Option: Generate a Full Action Plan

Get a prioritized action plan:

```
Create an upgrade plan for all outdated gems
```

I'll provide:
- Grouped by priority and risk level
- Estimated effort per gem
- Dependencies to watch
- Testing recommendations

## How It Works

1. **Installation Check** - Verifies gemtracker is available
   - If missing on macOS/Linux: `brew install gemtracker`
   - If missing on Windows: Download from GitHub releases
   - If missing elsewhere: Shows build/install instructions

2. **Gemfile.lock Detection** - Looks for Gemfile.lock in:
   - Current working directory
   - Specified project path
   - Parent directories (if needed)

3. **JSON Analysis** - Runs gemtracker with structured output:
   - Parses vulnerability data
   - Identifies outdated gems
   - Flags maintenance concerns
   - Calculates priorities

4. **Report Generation** - Creates actionable summary:
   - Security issues first
   - Then outdated gems (grouped by impact)
   - Health warnings for concerning dependencies
   - Severity and impact assessment for each

5. **User-Driven Updates** - You decide what to update:
   - Can ask for help with specific gems
   - Can request upgrade strategies
   - Can focus on security first or all updates
   - Can defer lower-priority items

## Installation Requirements

This skill requires the `gemtracker` CLI tool. If not found, I'll prompt you to install it. For detailed installation instructions, see [reference.md](reference.md#installation).

## Understanding Gem Groups

When reading reports, gem groups matter for prioritization:

- **default** - Used in production (most critical)
- **development** - Used during development only
- **test** - Used in test suite only
- **production** - Explicitly marked for production

A vulnerability in `test` is less urgent than one in `default`.

## Tips

✅ **Do's:**
- Start with security vulnerabilities first
- Prioritize first-level gem updates
- Test carefully when major version jumping
- Check health concerns before updating
- Update security gems (devise, bcrypt, JWT, etc.) quickly

❌ **Don'ts:**
- Update all gems at once (risky)
- Ignore health warnings (unmaintained gems cause problems)
- Skip testing major version upgrades
- Use outdated security libraries in production
- Assume older versions are automatically insecure

## Examples

### Quick scans
```bash
/gem-check                    # Analyze current project
/gem-check ~/my-rails-app    # Analyze specific project
```

### Follow-up interactions
```
"Help me upgrade Rails safely"
"Update devise to 4.9.3"
"Create a prioritized plan to update all outdated gems"
```

For more detailed examples including workflows for security-first updates, large projects, and CI/CD integration, see [examples.md](examples.md).

## Limitations

- Analysis based on data gemtracker can fetch (rubygems.org, GitHub APIs)
- GitHub API has rate limits (60/hour without token, 5000/hour with token)
- CVE database is example-based (not exhaustive real-time feed)
- Health data cached for 24 hours per gem
- Very large projects (500+ gems) may take longer to analyze

## Additional Resources

For detailed information about using this skill:
- **[API Reference](reference.md)** - Complete CLI documentation, command options, output formats, troubleshooting
- **[Usage Examples](examples.md)** - Real-world workflows: security-first updates, large projects, CI/CD integration, dependency conflicts

External resources:
- **[gemtracker GitHub](https://github.com/spaquet/gemtracker)** - Main project repository
- **[Installation Guide](https://github.com/spaquet/gemtracker#installation)** - Setup instructions for all platforms
- **[AI/Automation Guide](https://github.com/spaquet/gemtracker/blob/main/AI_GUIDE.md)** - Using gemtracker in scripts and automation

## License

This skill is open source under the MIT License. See [LICENSE](LICENSE) for full details.

**MIT License Summary:**
- ✅ Free to use, modify, and distribute
- ✅ Include license notice
- ✅ Use for commercial projects
- ⚠️ No warranty or liability

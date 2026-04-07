# Gemtracker AI Agent Guide

This guide is designed for AI systems, coding assistants, and automated tools to effectively use gemtracker for Ruby project assessment, dependency analysis, and recommendations.

## Overview

Gemtracker analyzes Ruby gem dependencies to identify:
- **Security vulnerabilities** (CVEs affecting specific gem versions)
- **Outdated gems** (versions with updates available)
- **Gem health** (maintenance status and sustainability)
- **Dependency relationships** (which gems depend on what)

This guide shows how to programmatically assess these factors and provide intelligent recommendations.

## Quick Start

### Basic Command Structure

```bash
gemtracker --report json /path/to/project
```

**Output:** Valid JSON to stdout, suitable for parsing
**Exit Code:** 0 (success) or 1 (error)
**Parse with:** `jq`, JSON parsers, or any JSON-capable tool

### Minimal Example

```bash
# Get vulnerability summary
gemtracker --report json . | jq '.summary'

# Output:
# {
#   "total_gems": 189,
#   "first_level_gems": 63,
#   "outdated_count": 45,
#   "vulnerable_count": 3
# }
```

## JSON Output Structure

### Complete Schema

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
      "Description": "Web development framework..."
    }
  ]
}
```

### Field Definitions

| Field | Type | Meaning | AI Use |
|-------|------|---------|--------|
| `Name` | string | Gem package name | Identification, recommendations |
| `Version` | string | Currently installed version | Risk assessment |
| `Groups` | array | Gem categories (default, development, test, production) | Impact severity |
| `IsFirstLevel` | boolean | Directly required vs transitive | Importance ranking |
| `IsOutdated` | boolean | Update available | Upgrade candidates |
| `LatestVersion` | string | Most recent available version | Upgrade target |
| `IsVulnerable` | boolean | Known CVE exists for this version | Security flag |
| `VulnerabilityInfo` | string | CVE ID and description | Severity assessment |
| `HomepageURL` | string | Gem repository link | Additional research |
| `Description` | string | Gem purpose and functionality | Context for recommendations |

## Status Assessment Patterns

### Severity Scoring

Use this framework to assess project risk:

```javascript
function assessProjectRisk(summary) {
  let risk = {
    critical: 0,
    high: 0,
    medium: 0,
    low: 0
  };

  // Critical vulnerabilities in first-level gems
  const vulnerableFirstLevel = gems.filter(g =>
    g.IsVulnerable && g.IsFirstLevel &&
    g.Groups.includes('default')
  ).length;
  risk.critical += vulnerableFirstLevel;

  // High: vulnerabilities in default group
  const vulnerableDefault = gems.filter(g =>
    g.IsVulnerable && g.Groups.includes('default')
  ).length;
  risk.high += Math.max(0, vulnerableDefault - risk.critical);

  // Medium: many outdated gems
  if (summary.outdated_count > 20) risk.medium += 2;
  if (summary.outdated_count > 50) risk.high += 1;

  // Low: outdated transitive dependencies
  const outdatedTransitive = gems.filter(g =>
    g.IsOutdated && !g.IsFirstLevel
  ).length;
  risk.low += Math.min(outdatedTransitive / 10, 5);

  return risk;
}
```

### Vulnerability Context Assessment

```javascript
function assessVulnerabilityImpact(gem) {
  // Most severe: production-critical vulnerability
  if (gem.IsVulnerable && gem.IsFirstLevel && gem.Groups.includes('default')) {
    return 'CRITICAL';
  }

  // High: direct production dependency
  if (gem.IsVulnerable && gem.Groups.includes('default')) {
    return 'HIGH';
  }

  // Medium: vulnerable but development-only
  if (gem.IsVulnerable && !gem.Groups.includes('default')) {
    return 'MEDIUM';
  }

  return 'LOW';
}
```

### Outdated Gem Priority

```javascript
function prioritizeUpdates(gems) {
  return gems
    .filter(g => g.IsOutdated)
    .sort((a, b) => {
      // Prioritize first-level gems
      if (a.IsFirstLevel !== b.IsFirstLevel) {
        return a.IsFirstLevel ? -1 : 1;
      }

      // Then by production relevance
      const aProduction = a.Groups.includes('default') ? 1 : 0;
      const bProduction = b.Groups.includes('default') ? 1 : 0;
      if (aProduction !== bProduction) {
        return bProduction - aProduction;
      }

      // Then alphabetically
      return a.Name.localeCompare(b.Name);
    });
}
```

## Use Cases and Examples

### Use Case 1: Security Assessment

**Goal:** Determine if project has security issues requiring immediate attention

```bash
# Extract vulnerable gems
gemtracker --report json . | jq '.gems[] | select(.IsVulnerable == true)'

# Count vulnerabilities in production
gemtracker --report json . | jq '[.gems[] | select(.IsVulnerable and (.Groups | index("default"))) ] | length'
```

**Recommendation Logic:**
```
IF vulnerable_count > 0:
  IF any vulnerable gem is in 'default' group:
    RECOMMEND: "Critical security issues detected. Upgrade these gems immediately before next release."
  ELSE IF all vulnerable gems in 'test' or 'development':
    RECOMMEND: "Security vulnerabilities found in development-only gems. Plan upgrades in current sprint."

IF vulnerable_count == 0:
  STATUS: "No known vulnerabilities detected in current versions."
```

### Use Case 2: Dependency Health Check

**Goal:** Assess project's overall dependency health and sustainability

```bash
# Get summary stats
gemtracker --report json . | jq '.summary'

# Calculate health percentage
gemtracker --report json . | jq '.summary | ((.total_gems - .vulnerable_count - .outdated_count) / .total_gems * 100)'
```

**Health Score Framework:**
```
Health Score = (healthy_gems / total_gems) * 100

Score 90-100:  ✓ Excellent - Well-maintained project
Score 75-89:   ⚠ Good - Minor updates needed
Score 50-74:   ⚠ Fair - Several updates recommended
Score 25-49:   ⚠ Poor - Significant dependency debt
Score 0-24:    ✗ Critical - Major attention needed
```

### Use Case 3: Upgrade Path Planning

**Goal:** Recommend which gems to upgrade and in what order

```bash
# Get all outdated gems
gemtracker --report json . | jq '.gems[] | select(.IsOutdated == true) | {Name, Version, LatestVersion, IsFirstLevel, Groups}'
```

**Planning Logic:**
```
1. Group by IsFirstLevel (true first)
2. Within each group, sort by gem criticality:
   - Rails framework gems: highest priority
   - Authentication/Security gems: high priority
   - Utility/Support gems: medium priority
   - Development tools: lower priority

3. For each gem:
   CALCULATE version_jump = major_version_diff(Version, LatestVersion)

   IF version_jump >= 2:
     RECOMMEND: "Consider testing extensively - major version jump"
   ELSE IF version_jump == 1:
     RECOMMEND: "Standard upgrade process should work"
   ELSE:
     RECOMMEND: "Patch/minor update - low risk"
```

### Use Case 4: Framework Detection and Guidance

**Goal:** Identify framework and provide framework-specific recommendations

```bash
function detectFramework(gems) {
  const railsGems = gems.filter(g =>
    g.IsFirstLevel &&
    ['rails', 'railties', 'actionpack', 'activerecord'].includes(g.Name)
  );

  if (railsGems.length > 0) {
    const railsVersion = railsGems.find(g => g.Name === 'rails')?.Version;
    return { framework: 'Rails', version: railsVersion };
  }

  const sinatraGems = gems.filter(g =>
    g.IsFirstLevel && g.Name === 'sinatra'
  );
  if (sinatraGems.length > 0) {
    return { framework: 'Sinatra', version: sinatraGems[0].Version };
  }

  return { framework: 'Ruby', version: 'unknown' };
}
```

**Framework-Specific Recommendations:**
```
IF framework == 'Rails':
  - Check Rails framework gems (rails, railties, actionpack, etc.) alignment
  - Recommend framework upgrade path if any are significantly outdated
  - Note major version changes for Rails (typically require special handling)

IF framework == 'Sinatra':
  - Focus on middleware and plugin compatibility
  - Rack and other core dependencies critical

IF framework == 'unknown':
  - Focus on library quality and security rather than framework guidance
```

## Integration Patterns

### Pattern 1: Simple Status Check

```bash
#!/bin/bash
# Check if project has vulnerabilities

result=$(gemtracker --report json . 2>/dev/null)
vulnerabilities=$(echo "$result" | jq '.summary.vulnerable_count')

if [ "$vulnerabilities" -gt 0 ]; then
  echo "❌ Project has $vulnerabilities vulnerable gems"
  exit 1
else
  echo "✓ No vulnerabilities detected"
  exit 0
fi
```

### Pattern 2: Generate Recommendation Report

```bash
#!/bin/bash
# Generate recommendations for dependency updates

gemtracker --report json . | jq '
{
  vulnerability_report: [
    .gems[] | select(.IsVulnerable) | {
      gem: .Name,
      current: .Version,
      issue: .VulnerabilityInfo,
      impact: (if .IsFirstLevel then "DIRECT" else "TRANSITIVE" end),
      groups: .Groups
    }
  ],
  outdated_report: [
    .gems[] | select(.IsOutdated) | {
      gem: .Name,
      current: .Version,
      latest: .LatestVersion,
      priority: (if .IsFirstLevel then "HIGH" else "LOW" end)
    }
  ] | sort_by(.priority) | reverse,
  summary: .summary
}
'
```

### Pattern 3: CI/CD Health Gate

```yaml
# GitHub Actions example
- name: Check gem dependencies
  run: |
    RESULT=$(gemtracker --report json .)
    VULNS=$(echo $RESULT | jq '.summary.vulnerable_count')
    OUTDATED=$(echo $RESULT | jq '.summary.outdated_count')

    # Fail if vulnerabilities found
    if [ $VULNS -gt 0 ]; then
      echo "❌ Security vulnerabilities detected!"
      exit 1
    fi

    # Warn if too many outdated gems
    if [ $OUTDATED -gt 30 ]; then
      echo "⚠️  Many outdated gems ($OUTDATED). Consider updates in next sprint."
    fi
```

### Pattern 4: Parse for AI Processing

```python
import subprocess
import json

def get_gem_status(project_path):
    """Get gem analysis for AI processing"""
    result = subprocess.run(
        ['gemtracker', '--report', 'json', project_path],
        capture_output=True,
        text=True
    )

    if result.returncode == 0:
        return json.loads(result.stdout)
    else:
        raise Exception(f"Gemtracker failed: {result.stderr}")

def assess_security(data):
    """Assess security posture"""
    vulnerable = [g for g in data['gems'] if g['IsVulnerable']]
    critical = [g for g in vulnerable if 'default' in g['Groups']]

    return {
        'total_vulnerabilities': len(vulnerable),
        'critical_vulnerabilities': len(critical),
        'requires_immediate_action': len(critical) > 0,
        'vulnerable_gems': [g['Name'] for g in vulnerable]
    }

def get_upgrade_recommendations(data):
    """Generate upgrade recommendations"""
    outdated = [g for g in data['gems'] if g['IsOutdated']]
    first_level = [g for g in outdated if g['IsFirstLevel']]

    return {
        'total_updates_available': len(outdated),
        'priority_updates': [g['Name'] for g in first_level],
        'nice_to_have': [g['Name'] for g in outdated if not g['IsFirstLevel']]
    }
```

## Decision Trees for Recommendations

### Vulnerability Response Tree

```
VULNERABLE GEM DETECTED
│
├─ Is it in 'default' group?
│  ├─ YES → CRITICAL PRIORITY
│  │        Recommend immediate patch/upgrade
│  │        Flag for security review
│  │
│  └─ NO → Check if in 'test' or 'development'
│         ├─ Only test/dev → MEDIUM PRIORITY
│         │  Can wait for next sprint, but should be tracked
│         │
│         └─ Other → LOW PRIORITY
│            Nice to fix, but not urgent
│
└─ Is it a first-level gem?
   ├─ YES → Higher priority
   │        Direct dependency, easier to upgrade
   │
   └─ NO → Lower priority
          Transitive dependency, may require coordinated updates
```

### Outdated Gem Response Tree

```
OUTDATED GEM DETECTED
│
├─ Version jump size?
│  ├─ Major (e.g., 6.x → 8.x) → REQUIRES TESTING
│  │  Recommend: Read changelog, test thoroughly
│  │
│  ├─ Minor (e.g., 7.1 → 7.2) → STANDARD UPGRADE
│  │  Recommend: Standard test coverage sufficient
│  │
│  └─ Patch (e.g., 7.1.2 → 7.1.3) → LOW RISK
│     Recommend: Can be auto-upgraded
│
├─ Is it a framework gem (Rails, etc.)?
│  ├─ YES → Align ALL framework components
│  │        Don't mix Rails 6 with Rails 7
│  │
│  └─ NO → Can upgrade independently
│
└─ How many gems are outdated total?
   ├─ 1-5 → Manageable in current sprint
   ├─ 6-20 → Plan for next sprint
   ├─ 21-50 → Significant work, multi-sprint effort
   └─ 50+ → Consider dependency refresh initiative
```

## Best Practices for AI Agents

### Do's ✓

- **Parse JSON output** - Most reliable and structured format
- **Check exit codes** - 0 = success, 1 = error
- **Run with `--verbose`** when debugging to get detailed logs
- **Cache results** - Gemtracker analysis doesn't change frequently
- **Assess context** - Consider if gem is in production vs dev-only
- **Explain recommendations** - Always explain WHY you're recommending something
- **Check groups** - A vulnerability in 'test' only is less critical
- **Prioritize first-level gems** - Direct dependencies are more important

### Don'ts ✗

- **Ignore gem groups** - A vulnerability in 'test' is different from 'default'
- **Recommend updates blindly** - Major version jumps need careful consideration
- **Treat all outdated equally** - Rails framework alignment matters more than misc utilities
- **Make decisions on incomplete data** - Always run against actual Gemfile.lock
- **Assume older = insecure** - Version numbers don't determine security
- **Ignore transitive dependencies** - They can still have vulnerabilities
- **Recommend removing gems** - Usually not the answer, upgrades are

## Common Patterns to Watch For

### Pattern: Framework Version Mismatch

```
ALERT: If Rails gem shows one version but railties/actionpack differ
REASON: Indicates incomplete upgrade or dependency conflict
RECOMMENDATION: Ensure all Rails framework gems use same version
```

### Pattern: Security Gem Outdated

```
ALERT: devise, bcrypt, JWT, or other auth/security gems are outdated
REASON: Security libraries need frequent updates
RECOMMENDATION: Prioritize upgrading security-related gems
```

### Pattern: High Outdated Count with Few Vulnerabilities

```
SITUATION: 30+ outdated gems but 0 vulnerabilities
MEANING: Project is technically sound but aging
RECOMMENDATION: Plan gradual dependency refresh, not urgent
```

### Pattern: Contradictory Gems

```
ALERT: Gem requires version X but dependency requires Y (mismatch)
MEANING: Dependency conflict
RECOMMENDATION: May indicate unsolvable constraint or need for gem replacement
```

## Troubleshooting

### Empty Gems Array

**Cause:** Gemfile.lock file not found or unreadable
**Action:** Verify path points to Ruby project directory, not gem directory
**Command:** `gemtracker /path/to/project`

### All Gems Marked Outdated

**Cause:** Rubygems.org API unreachable or rate limited
**Action:** Retry with `--no-cache` or check network connectivity
**Command:** `gemtracker --report json . --no-cache`

### No Vulnerability Info

**Cause:** Vulnerability database is example-based (not comprehensive)
**Action:** Use external CVE sources (Ruby Advisory Database, NVD) for real audits
**Note:** Gemtracker shows *detected* vulnerabilities, not exhaustive CVE scan

## API Stability Notes

- **JSON schema is stable** - Safe to parse and depend on
- **Field names won't change** - Can rely on exact field names
- **Groups are standardized** - Always: "default", "development", "test", "production"
- **Boolean fields are reliable** - Use for conditional logic
- **Exit codes follow standard conventions** - 0 = success, non-zero = error

## Performance Considerations

- **First run:** ~5-30 seconds (network calls to rubygems.org)
- **Subsequent runs:** Near instant (cached locally)
- **Network-heavy:** Version checking queries rubygems.org API
- **Rate limiting:** If many queries in short time, may get 429 responses
- **Cache location:** `~/.cache/gemtracker/`

---

**Last Updated:** April 2026
**For latest updates:** See README.md and CHANGELOG.md

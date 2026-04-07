# Gemtracker JSON Output Schema

## Complete Output Structure

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

## Field Reference

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

## Summary Object

The `summary` object provides quick statistics:

```json
{
  "total_gems": 189,          // All gems (first-level + transitive)
  "first_level_gems": 63,     // Directly required gems
  "outdated_count": 45,       // Gems with available updates
  "vulnerable_count": 3       // Gems with known CVEs
}
```

## Using Groups for Prioritization

The `Groups` array tells you where a gem is used:

```
"Groups": ["default"]           → Production use (CRITICAL if vulnerable)
"Groups": ["development"]       → Dev only (LOWER priority)
"Groups": ["test"]              → Tests only (LOWEST priority)
"Groups": ["default", "test"]   → Production AND tests
"Groups": ["production"]        → Explicitly production-only
```

**Impact Scoring:**
- Vulnerability in `["default"]` = **CRITICAL**
- Vulnerability in `["test"]` or `["development"]` = **MEDIUM**
- Outdated with `IsFirstLevel: true` = **HIGH** priority to update
- Outdated with `IsFirstLevel: false` = **LOW** priority (may auto-update)

## Exit Codes

- `0` - Success (analysis completed)
- `1` - Error (file not found, invalid Gemfile.lock, etc.)

## Performance Notes

- First run: 5-30 seconds (fetches from rubygems.org)
- Subsequent runs: Near instant (cached locally)
- Cache location: `~/.cache/gemtracker/`
- Cache invalidates when `Gemfile.lock` is modified

## Common Filtering Patterns

### Get all vulnerable gems
```bash
gemtracker --report json . | jq '.gems[] | select(.IsVulnerable)'
```

### Get vulnerabilities in production
```bash
gemtracker --report json . | jq '.gems[] | select(.IsVulnerable and (.Groups | index("default")))'
```

### Get outdated first-level gems
```bash
gemtracker --report json . | jq '.gems[] | select(.IsOutdated and .IsFirstLevel)'
```

### Count gems by group
```bash
gemtracker --report json . | jq '[.gems[].Groups[]] | group_by(.) | map({group: .[0], count: length})'
```

### Get summary
```bash
gemtracker --report json . | jq '.summary'
```

# Scenario: Security-Critical Project

## Situation
A Ruby on Rails project with known vulnerabilities that need immediate attention.

## Running the Skill
```
/gem-check
```

## Expected Output

```
🔍 GEM ANALYSIS REPORT
═══════════════════════════════════════════════════════════════

🔴 SECURITY VULNERABILITIES (3 found)
─────────────────────────────────────────────────────────────

CRITICAL (act immediately):
  1. rack (2.1.2)
     CVE-2021-22942: HTTP request smuggling vulnerability
     Impact: DIRECT first-level gem in 'default' group
     → Upgrade to: rack 3.0.8

HIGH (this sprint):
  2. devise (4.8.0)
     CVE-2022-1234: Authentication bypass in certain configurations
     Impact: TRANSITIVE dependency in 'default' group
     → Upgrade devise to 4.9.3

MEDIUM (development-only):
  3. rack-test (1.1.0)
     CVE-2020-5410: XSS vulnerability in test helper
     Impact: In 'test' group only
     → Upgrade to rack-test 1.1.3

─────────────────────────────────────────────────────────────

🟡 OUTDATED GEMS (23 found)
─────────────────────────────────────────────────────────────

HIGH PRIORITY (production first-level):
  • rails: 7.0.0 → 8.1.3 (major jump - requires testing)
  • pg: 1.3.0 → 1.5.4 (minor update - safe)

MEDIUM PRIORITY (development):
  • rspec: 3.10.0 → 3.13.0
  • pry: 0.14.0 → 0.14.2

LOW PRIORITY (transitive):
  • bundler: 2.1.0 → 2.5.1
  • json: 2.6.0 → 2.6.3
  ... and 16 more

─────────────────────────────────────────────────────────────

📊 PROJECT HEALTH SUMMARY
─────────────────────────────────────────────────────────────

Total gems: 189
First-level: 63
Transitive: 126

Status:
  🔴 3 vulnerabilities
  ⚠️ 23 outdated gems
  ✓ 163 healthy gems

RECOMMENDED ACTION PLAN
═══════════════════════════════════════════════════════════════

1. 🚨 IMMEDIATE (today/tomorrow):
   → Patch rack CVE-2021-22942
   → Verify devise update doesn't break auth
   Estimated: 1-2 hours + testing

2. 📅 THIS SPRINT:
   → Plan Rails 7→8 upgrade (major version jump)
   → Update remaining outdated gems
   Estimated: Full sprint focus

3. 📝 DOCUMENT:
   → Update CHANGELOG with security patches
   → Note Rails upgrade timeline in README

═══════════════════════════════════════════════════════════════
```

## Next Steps in Claude Code

After seeing this report, you might ask:

```
Help me upgrade rack to 3.0.8 safely
```

I'll help with:
- What changed in the new version
- Potential compatibility issues
- Testing strategy
- Rollback plan if needed

Or:

```
Create an upgrade plan for Rails 7 to 8
```

I'll provide:
- Changelog summary of breaking changes
- Related gems that need coordination
- Testing approach for major version
- Estimated timeline

## Key Insights

✅ **What's critical:**
- The CVE-2021-22942 must be fixed before next release
- It's in a direct dependency (rack), so it's a straightforward update

⚠️ **What needs planning:**
- Rails major version upgrade (7→8) is a larger effort
- Requires coordination of related gems (railties, actionpack, etc.)

✓ **What's good:**
- 163 gems are healthy
- Most updates are non-breaking minor/patch versions

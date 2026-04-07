# Scenario: Well-Maintained Project

## Situation
A mature Rails project that's kept up-to-date with regular dependency management.

## Running the Skill
```
/gem-check
```

## Expected Output

```
🔍 GEM ANALYSIS REPORT
═══════════════════════════════════════════════════════════════

🔴 SECURITY VULNERABILITIES
─────────────────────────────────────────────────────────────

✓ No vulnerabilities detected - all gems are secure!

─────────────────────────────────────────────────────────────

🟡 OUTDATED GEMS (5 found)
─────────────────────────────────────────────────────────────

HIGH PRIORITY (production first-level):
  • rails: 8.0.1 → 8.1.3 (minor - safe)
  • pg: 1.5.2 → 1.5.4 (patch - very safe)

MEDIUM PRIORITY (development):
  • rspec: 3.12.0 → 3.13.0 (minor)
  • rubocop: 1.60.0 → 1.61.0 (patch)

LOW PRIORITY (transitive):
  • bundler: 2.5.1 → 2.5.3 (patch)

─────────────────────────────────────────────────────────────

🟠 MAINTENANCE HEALTH (all first-level gems)
─────────────────────────────────────────────────────────────

✓ All gems show healthy maintenance status

  🟢 HEALTHY (63 gems):
     • rails: Active development, multiple maintainers
     • devise: Regular updates, maintained
     • pg: Stable releases, good community support
     • rspec: Very active project
     ... all other gems are actively maintained

─────────────────────────────────────────────────────────────

📊 PROJECT HEALTH SUMMARY
─────────────────────────────────────────────────────────────

Total gems: 189
First-level: 63
Transitive: 126

Status:
  ✓ 0 vulnerabilities
  ⚠️ 5 outdated gems (only minor/patch versions)
  🟢 184 gems are healthy and up-to-date

Health Score: 97/100 (Excellent)

RECOMMENDED ACTION PLAN
═══════════════════════════════════════════════════════════════

✅ NO URGENT ACTION NEEDED

Suggested maintenance (next sprint or two):
  • Update Rails 8.0.1 → 8.1.3 (minor, low risk)
  • Update database drivers (pg, etc.)
  • Run bundle update for minor/patch versions
  • No breaking changes expected

Estimated effort: 1-2 hours + testing

═══════════════════════════════════════════════════════════════
```

## Next Steps in Claude Code

Since everything is healthy, you might ask:

```
Update all gems to their latest minor/patch versions
```

I'll:
- Run bundle update
- Verify tests still pass
- Create a summary of what changed
- Suggest commit message

Or simply:

```
Run bundle update to patch version and test
```

I'll handle the update and verification.

## Key Insights

✅ **What's excellent:**
- No security vulnerabilities
- All gems are actively maintained
- No major version jumps needed
- Project is in good shape

📊 **What's next:**
- Small updates available (mostly minor/patch)
- Can be done regularly as part of normal maintenance
- Low risk since they're not major version changes

💡 **Best practice:**
- This is the state to aim for
- Regular small updates (weekly/monthly) avoid accumulation
- Staying current helps with security and compatibility

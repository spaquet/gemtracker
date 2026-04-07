# Gem Analysis & Recommendation Framework

This framework helps assess gem status and prioritize which ones to update.

## Severity Assessment

### Vulnerability Severity

```
CRITICAL (act immediately):
  ✓ IsVulnerable = true
  ✓ Groups includes "default"
  ✓ IsFirstLevel = true
  → Action: Update before next deploy

HIGH (address this sprint):
  ✓ IsVulnerable = true
  ✓ Groups includes "default"
  ✓ IsFirstLevel = false (transitive)
  → Action: Update through direct dependency

MEDIUM (plan for next sprint):
  ✓ IsVulnerable = true
  ✓ Groups = ["development"] OR ["test"]
  → Action: Update in development cycle

LOW (track but not urgent):
  ✓ IsVulnerable = true
  ✓ Groups = ["test"] only
  → Action: Can defer unless security-focused
```

### Outdated Gem Priority

```
HIGH PRIORITY:
  • IsFirstLevel = true
  • IsOutdated = true
  • Groups includes "default"
  → These are your core dependencies

MEDIUM PRIORITY:
  • IsFirstLevel = true
  • IsOutdated = true
  • Groups = ["development"] OR ["test"]
  → Still direct deps, but lower impact

LOW PRIORITY:
  • IsFirstLevel = false
  • IsOutdated = true
  • Any group
  → Often auto-update when direct deps update
```

## Version Jump Assessment

Before recommending an update, assess the version jump:

```
Major Jump (e.g., 6.x → 8.x):
  ✗ High risk
  → Requires: Full changelog review, extensive testing
  → Recommendation: "Plan for this sprint, test thoroughly"

Minor Jump (e.g., 7.1 → 7.2):
  ✓ Moderate risk
  → Requires: Review changelog for breaking changes
  → Recommendation: "Standard testing should suffice"

Patch Jump (e.g., 7.1.2 → 7.1.3):
  ✓ Low risk
  → Requires: Verification it installs
  → Recommendation: "Safe to update"
```

Extract version numbers and compare:
```javascript
function getVersionJump(current, latest) {
  const currParts = current.split('.').map(Number);
  const latestParts = latest.split('.').map(Number);

  if (currParts[0] !== latestParts[0]) return 'MAJOR';
  if (currParts[1] !== latestParts[1]) return 'MINOR';
  return 'PATCH';
}
```

## Framework-Specific Patterns

### Rails Framework Detection

If project uses Rails, watch for framework alignment:

```
Rails gems should be same version:
  • rails
  • railties
  • actionpack
  • activerecord
  • actionview
  • activesupport

Alert if versions mismatch → indicates incomplete upgrade
```

### Security Library Flags

These gems deserve priority when outdated:

```
CRITICAL SECURITY GEMS:
  • devise (authentication)
  • bcrypt (password hashing)
  • jwt (token authentication)
  • rack (HTTP handling)
  • rails (framework security)
  • net-http (HTTP client)

→ Recommend: Update first, before other gems
```

## Health Assessment Logic

For gems with maintenance concerns (from gemtracker output):

```
HEALTHY (no action):
  ✓ Last release within 1 year
  ✓ Multiple maintainers (2+)
  ✓ Active GitHub repository
  → Status: 🟢 Keep using

WARNING (monitor):
  ⚠ No releases for 1-3 years OR
  ⚠ Single maintainer
  ⚠ Infrequent updates
  → Status: 🟡 Can use but plan alternatives

CRITICAL (plan migration):
  ✗ No releases for 3+ years OR
  ✗ GitHub repository archived
  ✗ Zero maintainers
  → Status: 🔴 Plan replacement
  → Action: "Consider migrating to maintained alternative"
```

## Generating a Project Health Score

```javascript
function calculateHealthScore(summary, gems) {
  const healthyGems = gems.length - summary.vulnerable_count - summary.outdated_count;
  return Math.round((healthyGems / gems.length) * 100);
}

// Interpretation:
// 90-100: ✓ Excellent
// 75-89:  ⚠ Good
// 50-74:  ⚠ Fair
// 25-49:  ⚠ Poor
// 0-24:   ✗ Critical
```

## Decision Tree: Should We Update This Gem?

```
START: Is gem outdated?
│
├─ NO → STOP (no update needed)
│
└─ YES
   │
   ├─ Is it vulnerable? (IsVulnerable = true)
   │  ├─ YES → UPDATE IMMEDIATELY
   │  │         Risk of not updating > risk of updating
   │  │
   │  └─ NO
   │     │
   │     ├─ Is it first-level?
   │     │  ├─ YES
   │     │  │  ├─ Is it in "default" group?
   │     │  │  │  ├─ YES → UPDATE SOON
   │     │  │  │  │         High impact when problems occur
   │     │  │  │  │
   │     │  │  │  └─ NO → UPDATE WHEN CONVENIENT
   │     │  │  │          Development dependency
   │     │  │  │
   │     │  │  └─ NO
   │     │  │     ├─ Major version jump?
   │     │  │     │  ├─ YES → DEFER (requires testing)
   │     │  │     │  │
   │     │  │     │  └─ NO → UPDATE WITH NEXT BUNDLE
   │     │  │     │         Will auto-update likely
   │     │  │
   │     │  └─ IS it a transitive dependency?
   │     │     └─ DEFER (will update when direct deps update)
```

## Upgrade Planning Template

```
PROJECT HEALTH ASSESSMENT
═════════════════════════

Health Score: X/100
  ✓ Healthy gems: X
  ⚠️  Outdated: X
  🔴 Vulnerable: X
  🟠 Maintenance concerns: X

RECOMMENDED ACTIONS
═══════════════════

IMMEDIATE (this week):
  [ ] Gem1 - CVE-XXXX (critical)
  [ ] Gem2 - CVE-XXXX (high)

THIS SPRINT:
  [ ] Gem3 - Framework alignment
  [ ] Gem4 - Security library

NEXT SPRINT:
  [ ] Gem5 - Outdated, first-level
  [ ] Gem6 - Outdated, first-level

BACKLOG (track):
  [ ] Gem7 - Transitive, will likely auto-update
  [ ] Gem8 - Health concern, plan migration
```

## Communication Template

When presenting findings:

```
SECURITY FINDINGS:
  • X vulnerability/vulnerabilities found
  • Y in production code (CRITICAL)
  • Z in test/dev only (LOWER RISK)

DEPENDENCY HEALTH:
  • X gems are outdated
  • Y are first-level dependencies
  • Z require major version updates (higher risk)

MAINTENANCE HEALTH:
  • X gems show maintenance concerns
  • Y have few/no active maintainers
  • Z are archived repositories

RECOMMENDED PRIORITIES:
  1. [Action] (reason: critical security)
  2. [Action] (reason: direct dependency, high impact)
  3. [Action] (reason: lower risk update)
```

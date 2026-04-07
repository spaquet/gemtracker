# Scenario: High Dependency Debt

## Situation
A Rails project that hasn't had regular dependency updates in a while, accumulating "dependency debt".

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

✓ No critical vulnerabilities detected

Note: With many outdated gems, vulnerabilities may exist but aren't
detected. Consider running this again after updating core gems.

─────────────────────────────────────────────────────────────

🟡 OUTDATED GEMS (67 found)
─────────────────────────────────────────────────────────────

⚠️ HIGH number of outdated gems detected

HIGH PRIORITY (production first-level):
  • rails: 6.1.0 → 8.1.3 (2 major versions behind!)
  • devise: 4.7.0 → 4.9.3
  • pg: 1.1.4 → 1.5.4
  • sidekiq: 5.2.7 → 7.1.6 (major version gap)
  • redis: 4.0.1 → 5.1.0

MEDIUM PRIORITY (development):
  • rspec: 3.9.0 → 3.13.0 (multiple minors behind)
  • rubocop: 0.89.1 → 1.61.0 (major changes!)

LOW PRIORITY (transitive):
  • 47 more gems with available updates

─────────────────────────────────────────────────────────────

🟠 MAINTENANCE HEALTH (critical concerns)
─────────────────────────────────────────────────────────────

⚠️ WARNINGS DETECTED:

  🟡 WARNING (8 gems):
     • thin (1.8.1) - No releases for 3+ years
       → Consider switching to: Puma or Unicorn
     • cocaine (0.5.8) - Single maintainer, infrequent updates
       → Monitor or replace with active alternative

  🟠 CRITICAL (2 gems):
     • net-ftp (0.2.0) - Repository archived
       → Action: Plan migration immediately
     • old-system-gem (1.0.0) - No activity since 2019
       → Action: Find replacement or remove if possible

─────────────────────────────────────────────────────────────

📊 PROJECT HEALTH SUMMARY
─────────────────────────────────────────────────────────────

Total gems: 189
First-level: 63
Transitive: 126

Status:
  ✓ 0 critical vulnerabilities
  ⚠️ 67 outdated gems (35% of all gems)
  🟠 10 maintenance concerns
  🟡 122 gems are healthy

Health Score: 42/100 (Poor - significant work needed)

RECOMMENDED ACTION PLAN
═══════════════════════════════════════════════════════════════

This project needs a systematic dependency refresh initiative.
Don't try to update all at once - break into phases.

PHASE 1 (Week 1-2): Foundation
  1. Update Rails 6.1 → 8.1 (major effort)
  2. Update related framework gems (railties, actionpack, etc.)
  3. Test thoroughly with full suite
  Estimated: 3-5 days

PHASE 2 (Week 3): Critical Gems
  1. Update Sidekiq 5.2 → 7.1
  2. Update Redis client
  3. Update database drivers (pg, mysql2, etc.)
  4. Run integration tests
  Estimated: 2-3 days

PHASE 3 (Week 4): Maintenance Concerns
  1. Plan replacement for archived gems (net-ftp)
  2. Consider migration from unmaintained gems (thin)
  3. Document any breaking changes
  Estimated: 1-2 days

PHASE 4 (Week 5): Remaining Updates
  1. Update development tools (rspec, rubocop, etc.)
  2. Update remaining outdated gems
  3. Run full test suite
  Estimated: 2-3 days

Total estimated effort: 2-3 weeks of focused development

═══════════════════════════════════════════════════════════════

⚠️ IMPORTANT NOTES:

• This is NOT a one-day task - plan accordingly
• Each phase requires testing with full test suite
• Some gems may have breaking changes
• You may discover compatibility issues during testing
• Have a rollback strategy for each phase
```

## Next Steps in Claude Code

With this output, you'd likely ask:

```
Create a detailed Rails upgrade plan from 6.1 to 8.1
```

I'll help with:
- Breaking changes between versions
- Which gems are affected
- Testing strategy
- Estimated timeline
- Rollback plan

Or break it into smaller pieces:

```
What do I need to know about updating Rails from 6.1 to 8.0?
```

Then:

```
Update Sidekiq and related gems, I'll run tests
```

I'll handle the update and you verify tests.

Or ask for help with specific deprecated gems:

```
How do I replace thin with Puma?
```

I'll show:
- Configuration changes
- Performance implications
- Testing approach
- Migration checklist

## Key Insights

⚠️ **What's critical:**
- Rails is 2 major versions behind (6.1 → 8.1)
- Some gems no longer maintained
- This requires structured planning, not a single update

📊 **What's manageable:**
- No security vulnerabilities (good!)
- Most gems have clear upgrade paths
- The work is large but doable in phases

💡 **Best approach:**
- Budget 2-3 weeks for complete refresh
- Do it in phases, testing each
- Consider if this is good time to refactor heavily used dependencies
- Update regularly going forward (weekly) to avoid re-accumulating debt

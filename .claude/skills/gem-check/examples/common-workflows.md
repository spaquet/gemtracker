# Common Workflows

Real-world ways to use the gem-check skill in Claude Code.

## Workflow 1: Quick Security Check

**Goal:** Just verify no critical vulnerabilities exist (5 minutes)

```
/gem-check
```

You'll see:
- 🔴 Any CVEs in production gems
- ✓ Summary status
- 📋 Recommendations

**If vulnerabilities found:**
```
What gems have vulnerabilities? Show me the critical ones.
```

**If all clear:**
```
Can I proceed with deployment?
```

---

## Workflow 2: Planning an Upgrade Sprint

**Goal:** Create a prioritized list of what to update this sprint

```
/gem-check
```

Then ask:

```
Which gems should I prioritize updating? Create an ordered list based on:
1. Security vulnerabilities
2. Framework core gems
3. Production-critical dependencies
4. Lower-risk utilities
```

I'll provide:
- Ordered list with reasoning
- Estimated effort per gem
- Risk assessment (low/medium/high)
- Testing approach for each

---

## Workflow 3: Safe Major Version Upgrade

**Goal:** Update a critical gem safely (e.g., Rails major version)

```
/gem-check
```

See that Rails needs updating, then ask:

```
Help me upgrade Rails from 7.0 to 8.1 safely
```

I'll provide:
- Breaking changes between versions
- Related gems that must align
- Testing checklist
- Rollback strategy
- Step-by-step guide

Then:

```
Update Rails and run the tests
```

I'll handle the update and verify.

---

## Workflow 4: Dealing with Unmaintained Gems

**Goal:** Handle gems showing maintenance concerns

```
/gem-check
```

See health warnings, then ask:

```
What should I do about the archived gems you flagged?
```

I'll help with:
- Replacement gem recommendations
- Migration effort estimate
- Configuration changes needed
- Testing plan

Then optionally:

```
Help me migrate from thin to Puma
```

I'll walk through the process.

---

## Workflow 5: Regular Maintenance Routine

**Goal:** Keep dependencies current as part of normal workflow

**Weekly/Monthly:**

```
/gem-check
```

See outdated gems, then:

```
Update all minor and patch versions
```

I'll:
- Run bundle update for safe versions
- Run tests
- Show what changed
- Suggest commit message

---

## Workflow 6: CI/CD Integration

**Goal:** Add gem checking to your deployment process

While not directly in Claude Code, you can ask:

```
Show me how to integrate gem-check into GitHub Actions
```

I'll provide:
- YAML workflow file
- Fail conditions (e.g., vulnerabilities = fail)
- Report generation
- Integration with PRs

Then save that as `.github/workflows/gem-check.yml`

---

## Workflow 7: Post-Security-Advisory Response

**Goal:** React to newly discovered CVE

Scenario: You read about a new CVE affecting one of your gems.

```
/gem-check
```

Then:

```
Do we have gems affected by CVE-2024-XXXXX?
```

I'll:
- Check against your gems
- Show impact (production? transitive?)
- Suggest mitigation
- Help with patching

---

## Workflow 8: Dependency Audit Before Release

**Goal:** Verify gem health before shipping to production

Before a major release, ask:

```
Give me a complete security and health audit of our gems
```

I'll provide:
- All vulnerabilities
- Unmaintained gems
- Outdated production dependencies
- Risk summary
- Go/no-go recommendation

If issues found:

```
Create a checklist of what needs fixing before release
```

---

## Workflow 9: Framework Compatibility Check

**Goal:** Verify gems are compatible after framework update

After planning a Rails upgrade:

```
Show me which gems might have compatibility issues with Rails 8.1
```

I'll identify:
- Gems with version constraints
- Known incompatibilities
- Gems that need updating too
- Testing focus areas

---

## Workflow 10: Dependency Debt Assessment

**Goal:** Understand the scope of dependency debt

```
/gem-check
```

If many outdated gems:

```
How much work is it to get all gems current?
```

I'll provide:
- Break it into phases
- Time estimate per phase
- Risk assessment for each phase
- Prioritization strategy
- Alternative approaches (e.g., "update framework first, transitive auto-update")

---

## Quick Command Reference

| Goal | Command |
|------|---------|
| Basic check | `/gem-check` |
| Specific project | `/gem-check /path/to/project` |
| Security only | `/gem-check` → "Show vulnerabilities only" |
| Update plan | `/gem-check` → "Create an upgrade plan" |
| Safe updates | `/gem-check` → "Update minor/patch versions only" |
| Specific gem help | `/gem-check` → "How do I update [gem] safely?" |
| Audit before release | `/gem-check` → "Security audit before release" |
| CI/CD setup | `/gem-check` → "Add this to GitHub Actions" |

---

## Best Practices

✅ **Do this:**
- Run weekly to catch issues early
- Update minor/patch regularly
- Plan major updates in sprints
- Test each phase
- Ask for help with unfamiliar gems
- Document breaking changes

❌ **Avoid this:**
- Updating all gems at once
- Ignoring unmaintained gem warnings
- Skipping tests after updates
- Updating without understanding changes
- Leaving old versions in production unnecessarily

---

## When to Ask Follow-up Questions

After seeing a `/gem-check` report, you can ask:

- "Why is X gem flagged?"
- "How do I fix Y issue safely?"
- "What does this CVE mean for us?"
- "Is Z update a priority?"
- "Help me update W gem"
- "Create a plan for A, B, C"
- "What tests should I run after updating?"
- "What's the rollback strategy?"

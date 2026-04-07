# Gem Check - Usage Examples

Detailed scenarios and workflows for using the gem-check skill. Choose the scenario that matches your situation.

## Quick Start Examples

**Just check for vulnerabilities:**
```bash
/gem-check
```

**Analyze a specific project:**
```bash
/gem-check ~/my-rails-app
```

---

## Real-World Scenarios

Explore detailed workflows for specific situations:

### 📋 [Common Workflows](./examples/common-workflows.md)
Real-world ways to use gem-check in Claude Code, including:
- Quick security checks
- Planning upgrade sprints
- Dependency conflict resolution
- Handling health warnings

### 🔒 [Security-First Focus](./examples/scenario-security-focus.md)
When security vulnerabilities are your top priority:
- How to identify and prioritize CVEs
- Safe patching strategies
- Testing security updates
- Post-patch verification

### 💪 [Healthy Project Maintenance](./examples/scenario-healthy-project.md)
Keeping a well-maintained project running smoothly:
- Regular dependency updates
- Proactive health monitoring
- Staying current with frameworks
- Maintaining code quality

### 📈 [Dependency Debt Management](./examples/scenario-dependency-debt.md)
Tackling a project with accumulated dependency issues:
- Assessing the scope of work
- Prioritizing high-impact updates
- Handling breaking changes
- Creating a multi-sprint upgrade plan

### 📦 [Sample Output](./examples/sample-output.json)
Example JSON report from `gemtracker --report json` showing:
- Real gem data structure
- Vulnerability information
- Health scores
- Dependency graphs

---

## What You Can Ask After Running `/gem-check`

The skill analyzes your project and then you can ask follow-up questions:

**General questions:**
- "What are the top 3 things I should fix first?"
- "Which gems need the most testing after updates?"
- "Are there any gems we should replace entirely?"

**Technical details:**
- "Why is devise showing health warnings?"
- "What version constraints are limiting rails updates?"
- "Which transitive gems are causing the most issues?"

**Action planning:**
- "Create a detailed upgrade plan for all gems"
- "Help me update rails safely"
- "What's the risk of deferring this update?"

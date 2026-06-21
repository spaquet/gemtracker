# Gemtracker Skill Installation

Installs the gemtracker skill for both Claude Code and Codex, plus one shared Git pre-commit hook when run inside a Git repo.

## Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/spaquet/gemtracker/main/skills/install.sh | bash
```

The installer copies the skill to:

```text
~/.claude/skills/gemtracker
~/.codex/skills/gemtracker
```

If run inside a Git repo, it also appends a gemtracker block to:

```text
.git/hooks/pre-commit
```

That hook is shared by Claude, Codex, and normal terminal commits because Git runs it directly.

The hook writes the latest AI-friendly JSON report to:

```text
.git/gemtracker/latest.json
```

## Prerequisites

- `gemtracker` CLI in `PATH`
- Ruby project with `Gemfile.lock`, `gems.locked`, or `.gemspec`

macOS:

```bash
brew tap spaquet/gemtracker
brew install gemtracker
```

Verify:

```bash
gemtracker --version
```

## Manual Install

From a local clone:

```bash
mkdir -p ~/.claude/skills/gemtracker ~/.codex/skills/gemtracker
cp skills/gemtracker/SKILL.md ~/.claude/skills/gemtracker/
cp -R skills/gemtracker/scripts ~/.claude/skills/gemtracker/
cp skills/gemtracker/SKILL.md ~/.codex/skills/gemtracker/
cp -R skills/gemtracker/scripts ~/.codex/skills/gemtracker/
```

Optional shared Git hook:

```bash
bash skills/install.sh
```

## Usage

Claude Code:

```text
/gemtracker
/gemtracker . --json
```

Codex:

```text
Use gemtracker to audit this repo.
Check this Ruby project for vulnerable and outdated gems.
```

Direct CLI:

```bash
gemtracker . --report text
gemtracker . --report json
gemtracker . --report csv
```

After a commit attempt, Claude or Codex can inspect:

```text
.git/gemtracker/latest.json
```

## Uninstall

```bash
rm -rf ~/.claude/skills/gemtracker ~/.codex/skills/gemtracker
```

Remove the `# gemtracker pre-commit` block from `.git/hooks/pre-commit` if installed.

# Gemtracker Agent Skills

This directory contains the shared gemtracker skill for Claude Code and Codex.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/spaquet/gemtracker/main/skills/install.sh | bash
```

Local clone:

```bash
bash skills/install.sh
```

The installer copies the same skill to:

```text
~/.claude/skills/gemtracker
~/.codex/skills/gemtracker
```

When run inside a Git repo, it also adds a normal `.git/hooks/pre-commit` block. That hook is shared by Claude, Codex, and terminal commits, and writes `.git/gemtracker/latest.json`.

## Usage

Claude Code:

```text
/gemtracker
/gemtracker . --json
```

Codex:

```text
Use gemtracker to audit this repo.
Check this Ruby project for vulnerable gems.
```

## Files

- `gemtracker/SKILL.md`: agent instructions
- `gemtracker/scripts/analyze.sh`: CLI wrapper
- `install.sh`: dual Claude/Codex installer plus shared Git hook

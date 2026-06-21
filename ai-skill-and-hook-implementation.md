# Gemtracker Claude/Codex Skill Implementation

## Current Design

The same skill folder is installed for both agents:

```text
~/.claude/skills/gemtracker
~/.codex/skills/gemtracker
```

The shared files are:

```text
skills/gemtracker/SKILL.md
skills/gemtracker/scripts/analyze.sh
```

Claude can use its slash-command style (`/gemtracker`). Codex uses the same `SKILL.md` through natural-language triggers such as “use gemtracker to audit this repo”.

## Hook Design

The hook is a normal Git hook:

```text
.git/hooks/pre-commit
```

This is intentionally not a Claude hook or Codex hook. Git runs it regardless of whether the commit is started from Claude, Codex, an IDE, or a terminal.

The hook block is non-blocking:

```bash
# gemtracker pre-commit
if command -v gemtracker >/dev/null 2>&1; then
  gemtracker . --report text || true
fi
```

## Installer

`skills/install.sh` supports both local and remote install:

```bash
bash skills/install.sh
curl -fsSL https://raw.githubusercontent.com/spaquet/gemtracker/main/skills/install.sh | bash
```

For local installs, it copies files from the repo. For `curl | bash`, it downloads `SKILL.md` and `scripts/analyze.sh` from GitHub before installing them.

## Deliberate Skips

- No Claude `.claude/settings.json` hook: Git hook covers both agents.
- No Codex-specific hook: Codex does not need one for commit-time checks.
- No project/personal scope picker: Codex and Claude both have stable global skill paths, and the Git hook is per repo.

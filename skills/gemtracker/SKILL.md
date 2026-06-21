---
name: gemtracker
description: Analyze Ruby gem dependencies, vulnerabilities, outdated packages, maintenance health, and insecure gem sources with the gemtracker CLI. Use when asked to audit Ruby gems, check Gemfile.lock security, review dependency health, or run gemtracker.
---

# Gemtracker

Use the `gemtracker` CLI to analyze Ruby gem dependencies.

## Agent Usage

Claude Code users can run:

```bash
/gemtracker
/gemtracker /path/to/ruby-project
/gemtracker . --json
```

Codex users can ask naturally:

```text
Use gemtracker to audit this repo.
Check this Ruby project for vulnerable or outdated gems.
Run gemtracker on /path/to/ruby-project and summarize the result.
```

## Workflow

1. Check for the CLI:
   ```bash
   command -v gemtracker
   ```
2. If missing, tell the user to install it:
   ```bash
   brew tap spaquet/gemtracker && brew install gemtracker
   ```
3. Pick the target path. Default to the current repo or `.`.
4. Run the bundled wrapper when available:
   ```bash
   scripts/analyze.sh . --json
   ```
   Otherwise run:
   ```bash
   gemtracker . --report json
   ```
5. Summarize vulnerabilities first, then outdated gems, insecure sources, and maintenance risks.

## Output Formats

- Text: `gemtracker . --report text`
- JSON: `gemtracker . --report json`
- CSV: `gemtracker . --report csv`

## Notes

- Dependency files: `Gemfile.lock`, `gems.locked`, or `.gemspec`.
- Cache: `~/.cache/gemtracker/`.
- The shared install script can also add a normal Git `pre-commit` hook. That hook is editor/agent agnostic, so it works whether commits are made from Claude, Codex, or a terminal.

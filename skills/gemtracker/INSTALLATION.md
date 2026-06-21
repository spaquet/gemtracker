# Gemtracker Skill Installation Guide

This guide helps you install and configure the gemtracker skill for Claude Code.

## Prerequisites

- Claude Code (latest version)
- `gemtracker` CLI installed and in PATH
- Ruby project with gem dependency file (`Gemfile.lock`, `gems.locked`, or `.gemspec`)

## Step 1: Install Gemtracker CLI

### macOS (Homebrew)

```bash
brew tap spaquet/gemtracker
brew install gemtracker
```

Verify installation:
```bash
gemtracker --version
```

### Linux/Manual Build

Requires Go 1.24 or later:

```bash
git clone https://github.com/spaquet/gemtracker
cd gemtracker
make build-release
# Binary available in dist/
```

### Docker

```bash
docker run -v $(pwd):/project spaquet/gemtracker /project
```

## Step 2: Install Claude Code Skill

### Option A: From Repository (Recommended)

1. Open Claude Code settings
2. Navigate to Skills → Add Custom Skill
3. Select "From Repository"
4. Enter: `https://github.com/spaquet/gemtracker`
5. Select path: `/skills/gemtracker`
6. Click Install

### Option B: Manual Installation

1. Clone gemtracker repository:
   ```bash
   git clone https://github.com/spaquet/gemtracker ~/.claude/skills/gemtracker
   ```

2. Or copy skill definition:
   ```bash
   mkdir -p ~/.claude/skills/gemtracker
   cp skills/gemtracker/SKILL.md ~/.claude/skills/gemtracker/
   ```

## Step 3: Verify Installation

In Claude Code, type:

```
/gemtracker
```

Should output:
- Gem analysis of current directory
- Summary of vulnerabilities, outdated gems, health status
- No errors or "command not found"

## Step 4: Configure Hook (Optional)

To automatically check gems before committing, add to `.claude/settings.json`:

```json
{
  "hooks": {
    "before-commit": "gemtracker . --report text"
  }
}
```

This will:
1. Run `gemtracker` before each commit
2. Display findings in the status bar
3. Allow you to decide whether to proceed

**Note**: Hook is informational only—doesn't block commits.

## Uninstall

```bash
rm -rf ~/.claude/skills/gemtracker
```

And remove gemtracker CLI:

```bash
brew uninstall gemtracker  # macOS
# Or manually delete binary from PATH
```

## Troubleshooting

### "gemtracker: command not found"

Skill cannot locate `gemtracker` CLI in PATH.

**Solution:**
1. Ensure gemtracker is installed: `which gemtracker`
2. If installed but not in PATH, add to your shell profile:
   ```bash
   export PATH="/path/to/gemtracker:$PATH"
   ```

### "no dependency files found"

Gemtracker expects `Gemfile.lock`, `gems.locked`, or `.gemspec` in project root.

**Solution:**
- Verify file exists: `ls -la Gemfile.lock`
- Or specify path explicitly: `/gemtracker /path/to/project`

### Rate Limit Errors

External APIs (RubyGems, GitHub, OSV.dev) enforce rate limits.

**Solution:**
- Wait 1 hour and retry
- Use cached results: `gemtracker . --report json` (uses cache automatically)
- Authenticate with GitHub for higher limits (see project docs)

### JSON Parse Errors

`--report json` output is malformed.

**Solution:**
1. Check stderr for errors: `gemtracker . --report json 2>&1`
2. Update gemtracker: `brew upgrade gemtracker`
3. Report issue: https://github.com/spaquet/gemtracker/issues

## Environment Variables

Optional configuration:

```bash
# Sentry error tracking (production only)
export SENTRY_DSN="https://..."

# Verbose logging
export GEMTRACKER_VERBOSE=1

# Cache location (default: ~/.cache/gemtracker/)
export GEMTRACKER_CACHE_DIR="/custom/cache/path"
```

## Next Steps

- [SKILL.md](./SKILL.md) - Feature overview and usage examples
- [Gemtracker README](https://github.com/spaquet/gemtracker) - Full documentation
- [GitHub Issues](https://github.com/spaquet/gemtracker/issues) - Report bugs

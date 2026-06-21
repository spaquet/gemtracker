# Gemtracker Skill Installation Guide

Complete setup of gemtracker CLI + Claude Code skill with pre-commit hook.

## Quick Install (Recommended)

One command installs everything:

```bash
curl -fsSL https://raw.githubusercontent.com/spaquet/gemtracker/main/skills/install.sh | bash
```

The script will:
1. Auto-detect your OS (macOS, Linux, Windows)
2. Install gemtracker CLI if missing
3. Ask you where to install the skill (global, project, or personal)
4. Configure pre-commit hook automatically
5. Verify everything works

## Manual Installation

For step-by-step control, follow below:

### Prerequisites

- Claude Code (latest version)
- `gemtracker` CLI installed and in PATH
- Ruby project with gem dependency file (`Gemfile.lock`, `gems.locked`, or `.gemspec`)

### Step 1: Install Gemtracker CLI

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

### Step 2: Install Claude Code Skill

Copy skill to `~/.claude/skills/gemtracker`:

```bash
mkdir -p ~/.claude/skills/gemtracker
git clone https://github.com/spaquet/gemtracker ~/.claude/skills/gemtracker
# Or copy just the skill files:
cp skills/gemtracker/SKILL.md ~/.claude/skills/gemtracker/
cp -r skills/gemtracker/scripts ~/.claude/skills/gemtracker/
```

### Step 3: Configure Hook (Optional)

Add pre-commit hook to `.claude/settings.json`:

```json
{
  "hooks": {
    "before-commit": "gemtracker . --report text"
  }
}
```

Hook location depends on scope:
- **Global**: `~/.claude/settings.json` (all projects)
- **Project**: `./.claude/settings.json` (committed to repo)
- **Personal**: `~/.claude/projects/[hash]/.claude/settings.json` (just you)

### Step 4: Verify Installation

In Claude Code, type:

```
/gemtracker
```

Should analyze your gems and show vulnerabilities, outdated gems, and health status.

## Managing Installation

### Update Skill

Run the installer again to get the latest version:

```bash
curl -fsSL https://raw.githubusercontent.com/spaquet/gemtracker/main/skills/upgrade.sh | bash
```

Or manually copy new files to your skill location.

### Uninstall Skill

Remove the skill folder:

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/spaquet/gemtracker/main/skills/uninstall.sh)
```

Or manually:

```bash
rm -rf ~/.claude/skills/gemtracker
# Or for project scope:
rm -rf ./.claude/skills/gemtracker
```

To also remove gemtracker CLI:

```bash
brew uninstall gemtracker  # macOS
sudo apt remove gemtracker  # Linux (if installed via package manager)
# Or manually delete from PATH
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

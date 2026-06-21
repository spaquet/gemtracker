# Gemtracker Skill Installation - Implementation Plan

**Status**: Design Phase  
**Version**: 1.0  
**Last Updated**: 2026-06-21

## Overview

Distribute gemtracker as a Claude Code skill with intelligent installation script that:
1. Detects/installs gemtracker CLI if missing
2. Installs skill to user's choice of scope (global/project/personal)
3. Configures pre-commit hook automatically
4. Provides zero-friction onboarding

## Requirements

### Functional
- **OS Detection**: macOS (Homebrew), Linux (binary), Windows (manual link)
- **Gemtracker Check**: Detect if CLI installed; offer installation if missing
- **Scope Selection**: Ask user - Global / Project / Personal (default)
- **Skill Installation**: Copy skill files to chosen location
- **Hook Configuration**: Add pre-commit hook to appropriate `.claude/settings.json`
- **Verification**: Test installation success before completion
- **Rollback**: Offer cleanup if installation fails

### Non-Functional
- **User-Friendly**: Clear prompts, minimal configuration
- **Idempotent**: Safe to run multiple times
- **Shell-Safe**: Works in bash/zsh/sh
- **Error Handling**: Graceful degradation, helpful error messages

## Installation Flow

### 1. User Runs Script
```bash
curl -fsSL https://raw.githubusercontent.com/spaquet/gemtracker/main/skills/install.sh | bash
```

### 2. Detect Gemtracker CLI
```
Check: which gemtracker
├─ Found (v1.2.16) → Continue
└─ Not found → Offer install
    ├─ macOS + Homebrew → brew tap && brew install
    ├─ Linux → Download binary from releases
    └─ Windows → Show manual setup link
```

### 3. Detect OS & Set Paths
```
macOS / Linux / Windows
↓
Set default locations:
├─ Global:   ~/.claude/skills/gemtracker
├─ Project:  ./.claude/skills/gemtracker
└─ Personal: ~/.claude/projects/[HASH]/.claude/skills/gemtracker
```

### 4. Ask Installation Scope
```
Which scope?
  1. Global (~/.claude/skills/)
     - Available to all projects
     - Shared with team
  2. Project (./.claude/skills/)
     - Committed to repo
     - Everyone on team gets it
  3. Personal (~/.claude/projects/[HASH]/)
     - Just for you [DEFAULT]
     - Private to your account

Choice [1-3]: _
```

### 5. Install Skill Files
```
Copy:
├─ skills/gemtracker/SKILL.md → [TARGET]/SKILL.md
├─ skills/gemtracker/INSTALLATION.md → [TARGET]/INSTALLATION.md
└─ Create auto-detect wrapper (if needed)
```

### 6. Configure Hook
```
Add to appropriate .claude/settings.json:
{
  "hooks": {
    "before-commit": "gemtracker . --report json 2>/dev/null | jq '.summary' || true"
  }
}

Location depends on scope:
├─ Global: ~/.claude/settings.json
├─ Project: ./.claude/settings.json
└─ Personal: ~/.claude/projects/[HASH]/.claude/settings.json
```

### 7. Verify Installation
```
Tests:
├─ Check skill files exist
├─ Verify hook in settings.json
└─ Run gemtracker --version
└─ Try `/gemtracker` in Claude Code
```

### 8. Success Message
```
✓ Installation complete!

Next steps:
1. Open Claude Code
2. Try: /gemtracker
3. Or: /gemtracker /path/to/ruby-project

Pre-commit hook enabled:
  Gems will be checked before each commit
  (Non-blocking - doesn't prevent commits)

To update: Run this script again
To uninstall: rm -rf [TARGET_PATH]
```

## File Structure

### Deliverables
```
skills/
├── IMPLEMENTATION.md (this file)
├── README.md (skill overview)
├── install.sh (main installation script) [NEW]
└── gemtracker/
    ├── SKILL.md (skill definition)
    ├── INSTALLATION.md (updated: just "run script")
    └── detect.sh (auto-detect wrapper) [OPTIONAL]
```

### Script: `skills/install.sh`

**Responsibilities:**
1. Detect OS (macOS/Linux/Windows)
2. Check gemtracker installed
3. Offer gemtracker installation if missing
4. Detect if in git repo (for Project scope)
5. Ask user for scope (global/project/personal)
6. Copy skill files to target location
7. Create/update `.claude/settings.json` at target scope
8. Add pre-commit hook configuration
9. Verify success
10. Print success message with next steps

**Key Functions:**
```bash
detect_os()           # Returns: macos, linux, windows
detect_gemtracker()   # Returns: version or "not found"
install_gemtracker()  # Installs CLI based on OS
ask_scope()           # Returns: 1 (global), 2 (project), 3 (personal)
get_target_path()     # Returns: full path based on scope
copy_skill_files()    # Copy SKILL.md, INSTALLATION.md, etc
add_hook()            # Update .claude/settings.json with hook
verify_install()      # Test installation success
cleanup_on_error()    # Remove partial installation
```

### Script: `skills/gemtracker/detect.sh` [OPTIONAL]

**Purpose:** Auto-detect missing gemtracker, show helpful error

**Usage:** Invoked when skill is called but gemtracker not found

**Output:**
```
❌ gemtracker not found in PATH

Install with one command:
  curl -fsSL https://raw.githubusercontent.com/spaquet/gemtracker/main/skills/install.sh | bash

Or manually:
  brew install spaquet/gemtracker/gemtracker  # macOS
  https://github.com/spaquet/gemtracker/releases  # Linux/Windows
```

## Configuration Examples

### Global Scope
```
~/.claude/settings.json
{
  "hooks": {
    "before-commit": "gemtracker . --report json ..."
  }
}
```

### Project Scope
```
./.claude/settings.json (in repo)
{
  "hooks": {
    "before-commit": "gemtracker . --report json ..."
  }
}
```

### Personal Scope
```
~/.claude/projects/[HASH]/.claude/settings.json
{
  "hooks": {
    "before-commit": "gemtracker . --report json ..."
  }
}
```

## Success Criteria

✅ User can install with single command  
✅ Script detects missing gemtracker and offers install  
✅ User chooses installation scope (global/project/personal)  
✅ Skill files copied to correct location  
✅ Hook configured in appropriate `.claude/settings.json`  
✅ Pre-commit hook runs on every commit (non-blocking)  
✅ `/gemtracker` command works after install  
✅ Error messages are clear and actionable  
✅ Script is idempotent (safe to run multiple times)  
✅ Works on macOS, Linux, and Windows  

## Implementation Checklist

- [ ] Create `skills/install.sh` with full logic
- [ ] Test on macOS with Homebrew
- [ ] Test on Linux (Ubuntu/Debian)
- [ ] Test Windows (WSL or native)
- [ ] Create `skills/gemtracker/detect.sh` for auto-detection
- [ ] Update `skills/gemtracker/INSTALLATION.md` (simplify to "run script")
- [ ] Update main README with one-command install
- [ ] Test global scope installation
- [ ] Test project scope installation
- [ ] Test personal scope installation
- [ ] Test hook execution on commit
- [ ] Test error handling (missing gemtracker, permissions, etc)
- [ ] Document rollback/uninstall
- [ ] Test idempotency (run script twice)

## Future Enhancements

- Auto-update check (notify if new version available)
- Uninstall script
- Configuration wizard for hook customization
- Analytics/telemetry (optional, privacy-respecting)
- GitHub Actions integration (auto-install for CI)
- Docker image with skill pre-installed

## Related Documents

- [Gemtracker README](../../README.md) - Project overview
- [SKILL.md](./gemtracker/SKILL.md) - Skill definition
- [INSTALLATION.md](./gemtracker/INSTALLATION.md) - User guide (to be simplified)
- [skills/README.md](./README.md) - Skill comparison/overview

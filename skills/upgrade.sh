#!/bin/bash
set -euo pipefail

# Gemtracker Claude Code Skill Upgrader
# Checks for new skill versions and upgrades if available

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

log_success() {
    echo -e "${GREEN}✓${NC} $1"
}

log_error() {
    echo -e "${RED}✗${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# Get current installed skill version from SKILL.md
get_installed_version() {
    local skill_path="$1"
    if [ -f "$skill_path/SKILL.md" ]; then
        grep "^version:" "$skill_path/SKILL.md" | awk '{print $2}' || echo "unknown"
    else
        echo "not_installed"
    fi
}

# Get latest skill version from GitHub
get_latest_version() {
    # Try to fetch latest from GitHub API
    local latest=$(curl -s https://api.github.com/repos/spaquet/gemtracker/contents/skills/gemtracker/SKILL.md \
        | grep '"version"' | head -1 | sed 's/.*"version": "\([^"]*\)".*/\1/' 2>/dev/null || echo "")

    if [ -z "$latest" ]; then
        # Fallback: try to get from raw GitHub
        latest=$(curl -s https://raw.githubusercontent.com/spaquet/gemtracker/main/skills/gemtracker/SKILL.md \
            | grep "^version:" | awk '{print $2}' 2>/dev/null || echo "")
    fi

    echo "$latest"
}

# Compare versions (simple semver)
version_greater() {
    local v1="$1"
    local v2="$2"
    # Simple comparison: split by dots and compare numerically
    [[ "$(printf '%s\n' "$v1" "$v2" | sort -V | head -n1)" != "$v1" ]]
}

# Find installed skill (search all possible locations)
find_installed_skill() {
    # Check global
    if [ -f "$HOME/.claude/skills/gemtracker/SKILL.md" ]; then
        echo "$HOME/.claude/skills/gemtracker"
        return 0
    fi

    # Check project (if in git repo)
    if git rev-parse --git-dir > /dev/null 2>&1; then
        if [ -f "./.claude/skills/gemtracker/SKILL.md" ]; then
            echo "./.claude/skills/gemtracker"
            return 0
        fi
    fi

    # Check personal (search projects directory)
    if [ -d "$HOME/.claude/projects" ]; then
        # Find first gemtracker skill in projects
        local skill_path=$(find "$HOME/.claude/projects" -path "*/gemtracker/SKILL.md" 2>/dev/null | head -1)
        if [ -n "$skill_path" ]; then
            dirname "$skill_path"
            return 0
        fi
    fi

    echo ""
    return 1
}

# Download and run latest install script
run_latest_install() {
    log_info "Downloading latest installer..."

    local install_script=$(mktemp)
    trap "rm -f $install_script" EXIT

    if curl -fsSL https://raw.githubusercontent.com/spaquet/gemtracker/main/skills/install.sh -o "$install_script"; then
        chmod +x "$install_script"
        log_success "Running latest installer..."
        bash "$install_script"
    else
        log_error "Failed to download installer from GitHub"
        exit 1
    fi
}

# Main upgrade flow
main() {
    echo
    echo "╔════════════════════════════════════════════════════════════╗"
    echo "║     Gemtracker Claude Code Skill Upgrader                  ║"
    echo "╚════════════════════════════════════════════════════════════╝"
    echo

    # Find installed skill
    log_info "Looking for installed skill..."
    local installed_path=$(find_installed_skill)

    if [ -z "$installed_path" ]; then
        log_warning "Gemtracker skill not found"
        echo
        echo "The skill doesn't appear to be installed."
        echo "Install it with:"
        echo "  curl -fsSL https://raw.githubusercontent.com/spaquet/gemtracker/main/skills/install.sh | bash"
        exit 1
    fi

    log_success "Found at: $installed_path"
    echo

    # Get versions
    local current_version=$(get_installed_version "$installed_path")
    log_info "Current version: $current_version"

    log_info "Checking for new version..."
    local latest_version=$(get_latest_version)

    if [ -z "$latest_version" ]; then
        log_warning "Could not determine latest version"
        echo "Check manually at: https://github.com/spaquet/gemtracker/releases"
        exit 1
    fi

    log_success "Latest version: $latest_version"
    echo

    # Compare versions
    if [ "$current_version" = "$latest_version" ]; then
        log_success "Already up to date!"
        exit 0
    fi

    if version_greater "$latest_version" "$current_version"; then
        log_warning "Update available: $current_version → $latest_version"
        echo
        read -p "Upgrade now? (y/n) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            run_latest_install
        else
            log_info "Upgrade cancelled"
            exit 0
        fi
    else
        log_info "Current version is newer than latest"
        exit 0
    fi
}

main "$@"

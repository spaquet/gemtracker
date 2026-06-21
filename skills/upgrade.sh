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

# Check if skill is installed
is_skill_installed() {
    local skill_path="$1"
    [ -f "$skill_path/SKILL.md" ] && [ -f "$skill_path/scripts/analyze.sh" ]
}

# Download install script to temp location for comparison
get_latest_install_hash() {
    curl -s https://raw.githubusercontent.com/spaquet/gemtracker/main/skills/install.sh | md5sum | awk '{print $1}'
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

    # Check if skill is valid
    if ! is_skill_installed "$installed_path"; then
        log_error "Installed skill appears incomplete or corrupted"
        echo "Reinstall with:"
        echo "  curl -fsSL https://raw.githubusercontent.com/spaquet/gemtracker/main/skills/install.sh | bash"
        exit 1
    fi

    log_info "Checking for updates from GitHub..."
    read -p "Re-run installation to get latest version? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        run_latest_install
    else
        log_info "Upgrade cancelled"
        exit 0
    fi
}

main "$@"

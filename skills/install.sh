#!/bin/bash
set -euo pipefail

# Gemtracker Claude Code Skill Installer
# Installs gemtracker skill to user's choice of scope (global/project/personal)
# and configures pre-commit hook automatically

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory (where this script is)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SKILL_SOURCE_DIR="$SCRIPT_DIR/gemtracker"

# ============================================================================
# Helper Functions
# ============================================================================

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

# Detect operating system
detect_os() {
    case "$(uname -s)" in
        Darwin)
            echo "macos"
            ;;
        Linux)
            echo "linux"
            ;;
        MINGW*|MSYS*|CYGWIN*)
            echo "windows"
            ;;
        *)
            echo "unknown"
            ;;
    esac
}

# Check if gemtracker is installed and get version
check_gemtracker() {
    if command -v gemtracker &> /dev/null; then
        local version=$(gemtracker --version 2>/dev/null | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+' || echo "unknown")
        echo "$version"
    else
        echo "not_found"
    fi
}

# Offer to install gemtracker based on OS
install_gemtracker_prompt() {
    local os="$1"

    echo
    log_warning "gemtracker CLI not found"
    echo
    echo "gemtracker must be installed to use this skill."
    echo "Would you like to install it now?"
    echo

    case "$os" in
        macos)
            echo "For macOS (Homebrew):"
            echo "  brew tap spaquet/gemtracker"
            echo "  brew install gemtracker"
            echo
            ;;
        linux)
            echo "For Linux:"
            echo "  Visit: https://github.com/spaquet/gemtracker/releases"
            echo "  Download latest release for your architecture"
            echo "  Extract and add to PATH"
            echo
            ;;
        windows)
            echo "For Windows:"
            echo "  Visit: https://github.com/spaquet/gemtracker/releases"
            echo "  Download gemtracker_windows_*.zip"
            echo "  Extract and add to PATH"
            echo
            ;;
    esac

    read -p "Continue with skill installation? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_error "Installation cancelled"
        exit 1
    fi
}

# Check if we're in a git repository
is_git_repo() {
    git rev-parse --git-dir > /dev/null 2>&1
}

# Get the hash of the current project for personal scope
get_project_hash() {
    if is_git_repo; then
        # Use git root + current working directory as identifier
        local git_root=$(git rev-parse --show-toplevel)
        local project_id="${git_root}"
        echo "$project_id" | md5sum | awk '{print $1}'
    else
        # Fallback: use current directory
        pwd | md5sum | awk '{print $1}'
    fi
}

# Ask user for installation scope
ask_scope() {
    echo
    echo "Where should the skill be installed?"
    echo
    echo "  1. Global (~/.claude/skills/)"
    echo "     Available to all your projects"
    echo
    echo "  2. Project (./.claude/skills/)"
    echo "     Committed to repo, shared with team"
    echo
    echo "  3. Personal (~/.claude/projects/...) [DEFAULT]"
    echo "     Just for you, not shared"
    echo

    read -p "Enter choice [1-3]: " -r scope_choice
    scope_choice="${scope_choice:-3}"

    case "$scope_choice" in
        1) echo "global" ;;
        2) echo "project" ;;
        3) echo "personal" ;;
        *)
            log_error "Invalid choice"
            exit 1
            ;;
    esac
}

# Get target path based on scope
get_target_path() {
    local scope="$1"

    case "$scope" in
        global)
            echo "$HOME/.claude/skills/gemtracker"
            ;;
        project)
            if ! is_git_repo; then
                log_error "Not in a git repository. Cannot use project scope."
                exit 1
            fi
            echo "./.claude/skills/gemtracker"
            ;;
        personal)
            local project_hash=$(get_project_hash)
            echo "$HOME/.claude/projects/$project_hash/.claude/skills/gemtracker"
            ;;
    esac
}

# Get settings path based on scope
get_settings_path() {
    local scope="$1"

    case "$scope" in
        global)
            echo "$HOME/.claude/settings.json"
            ;;
        project)
            echo "./.claude/settings.json"
            ;;
        personal)
            local project_hash=$(get_project_hash)
            echo "$HOME/.claude/projects/$project_hash/.claude/settings.json"
            ;;
    esac
}

# Copy skill files to target location
copy_skill_files() {
    local target_path="$1"

    # Create target directory
    mkdir -p "$target_path"

    # Copy SKILL.md
    cp "$SKILL_SOURCE_DIR/SKILL.md" "$target_path/SKILL.md" || {
        log_error "Failed to copy SKILL.md"
        exit 1
    }

    # Copy scripts folder with implementation
    if [ -d "$SKILL_SOURCE_DIR/scripts" ]; then
        mkdir -p "$target_path/scripts"
        cp "$SKILL_SOURCE_DIR/scripts/"* "$target_path/scripts/" || {
            log_error "Failed to copy scripts"
            exit 1
        }
    fi

    log_success "Skill files copied to $target_path"
}

# Read JSON value (simple parsing, not production-grade)
json_get_value() {
    local file="$1"
    local key="$2"
    grep "\"$key\"" "$file" | head -1 | sed 's/.*": *"\([^"]*\)".*/\1/'
}

# Add hook to settings.json
add_hook_to_settings() {
    local settings_path="$1"
    local settings_dir=$(dirname "$settings_path")

    # Create directory if needed
    mkdir -p "$settings_dir"

    # Create hook command
    local hook_cmd='gemtracker . --report json 2>/dev/null | jq -r '"'"'.summary | "Gems: \(.total_gems) | Vulnerable: \(.vulnerable_count) | Outdated: \(.outdated_count)"'"'"' || echo "Gem check: gemtracker CLI not in PATH"'

    # Check if settings.json exists
    if [ ! -f "$settings_path" ]; then
        # Create new settings.json
        cat > "$settings_path" <<EOF
{
  "hooks": {
    "before-commit": "$hook_cmd"
  }
}
EOF
        log_success "Created $settings_path with pre-commit hook"
    else
        # Check if hooks section exists
        if grep -q '"hooks"' "$settings_path"; then
            # Hooks section exists, update it
            # This is a simple replacement - assumes proper JSON
            if grep -q '"before-commit"' "$settings_path"; then
                # before-commit already exists, warn user
                log_warning "$settings_path already has before-commit hook"
                log_info "Manual merge may be needed"
            else
                # Add before-commit hook
                sed -i.bak 's/"hooks": {/"hooks": {\n    "before-commit": "'"$hook_cmd"'",/' "$settings_path"
                rm -f "$settings_path.bak"
                log_success "Added pre-commit hook to $settings_path"
            fi
        else
            # No hooks section, add it
            sed -i.bak 's/^}/,\n  "hooks": {\n    "before-commit": "'"$hook_cmd"'"\n  }\n}/' "$settings_path"
            rm -f "$settings_path.bak"
            log_success "Added hooks section to $settings_path"
        fi
    fi
}

# Verify installation
verify_installation() {
    local target_path="$1"

    echo
    log_info "Verifying installation..."

    # Check skill files exist
    if [ ! -f "$target_path/SKILL.md" ]; then
        log_error "SKILL.md not found at $target_path"
        return 1
    fi

    # Check implementation script exists
    if [ ! -f "$target_path/scripts/analyze.sh" ]; then
        log_error "scripts/analyze.sh not found at $target_path"
        return 1
    fi

    # Check gemtracker installed
    if ! command -v gemtracker &> /dev/null; then
        log_warning "gemtracker CLI not in PATH (will show helpful error when /gemtracker is invoked)"
    else
        local version=$(gemtracker --version 2>/dev/null | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+' || echo "unknown")
        log_success "gemtracker $version found"
    fi

    log_success "Installation verified"
    return 0
}

# Print success message
print_success_message() {
    local scope="$1"
    local target_path="$2"
    local settings_path="$3"

    echo
    log_success "Installation complete!"
    echo
    echo "Skill installed to: $target_path"
    echo "Hook configured in: $settings_path"
    echo
    echo "Next steps:"
    echo "  1. Open Claude Code"
    echo "  2. Try: /gemtracker"
    echo "  3. Or: /gemtracker /path/to/ruby-project"
    echo
    echo "Pre-commit hook enabled:"
    echo "  Gem dependencies will be checked before each commit"
    echo "  (Non-blocking - commits still proceed)"
    echo

    if [ "$scope" = "project" ]; then
        echo "Scope: Project (shared with team via git)"
        echo "  Commit ./.claude/settings.json to share this setup"
        echo
    elif [ "$scope" = "personal" ]; then
        echo "Scope: Personal (just for you)"
        echo "  Other users can run this script to install for themselves"
        echo
    else
        echo "Scope: Global (all projects)"
        echo
    fi

    echo "To update: bash $REPO/raw/main/skills/upgrade.sh"
    echo "To uninstall: bash $REPO/raw/main/skills/uninstall.sh"
}

# ============================================================================
# Main Installation Flow
# ============================================================================

main() {
    echo
    echo "╔════════════════════════════════════════════════════════════╗"
    echo "║     Gemtracker Claude Code Skill Installer                 ║"
    echo "╚════════════════════════════════════════════════════════════╝"
    echo

    # Step 1: Detect OS
    local os=$(detect_os)
    if [ "$os" = "unknown" ]; then
        log_error "Unsupported operating system"
        exit 1
    fi
    log_success "Detected OS: $os"

    # Step 2: Check gemtracker
    echo
    log_info "Checking for gemtracker CLI..."
    local gemtracker_version=$(check_gemtracker)
    if [ "$gemtracker_version" = "not_found" ]; then
        install_gemtracker_prompt "$os"
    else
        log_success "gemtracker $gemtracker_version found"
    fi

    # Step 3: Ask for scope
    local scope=$(ask_scope)
    log_success "Scope: $scope"

    # Step 4: Get paths
    local target_path=$(get_target_path "$scope")
    local settings_path=$(get_settings_path "$scope")

    # Step 5: Copy skill files
    echo
    log_info "Installing skill files..."
    copy_skill_files "$target_path"

    # Step 6: Configure hook
    echo
    log_info "Configuring pre-commit hook..."
    add_hook_to_settings "$settings_path"

    # Step 7: Verify
    verify_installation "$target_path" || {
        log_error "Installation verification failed"
        exit 1
    }

    # Step 8: Success message
    print_success_message "$scope" "$target_path" "$settings_path"
}

# Run main function
main "$@"

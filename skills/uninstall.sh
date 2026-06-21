#!/bin/bash
set -euo pipefail

# Gemtracker Claude Code Skill Uninstaller
# Removes skill files and pre-commit hook configuration

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

# Check if valid skill is installed at path
is_valid_skill() {
    local skill_path="$1"
    [ -f "$skill_path/SKILL.md" ] && [ -d "$skill_path/scripts" ]
}

# Find all installed skill locations
find_all_skills() {
    local locations=()

    # Check global
    if is_valid_skill "$HOME/.claude/skills/gemtracker"; then
        locations+=("global:$HOME/.claude/skills/gemtracker")
    fi

    # Check project (if in git repo)
    if git rev-parse --git-dir > /dev/null 2>&1; then
        if is_valid_skill "./.claude/skills/gemtracker"; then
            locations+=("project:./.claude/skills/gemtracker")
        fi
    fi

    # Check personal (search projects directory)
    if [ -d "$HOME/.claude/projects" ]; then
        while IFS= read -r skill_path; do
            if [ -n "$skill_path" ]; then
                local dir=$(dirname "$skill_path")
                if is_valid_skill "$dir"; then
                    local project_hash=$(basename $(dirname $(dirname "$dir")))
                    locations+=("personal ($project_hash):$dir")
                fi
            fi
        done < <(find "$HOME/.claude/projects" -path "*/gemtracker/SKILL.md" 2>/dev/null)
    fi

    printf '%s\n' "${locations[@]}"
}

# Get settings path for skill location
get_settings_for_skill() {
    local skill_path="$1"
    local scope="$2"

    case "$scope" in
        global)
            echo "$HOME/.claude/settings.json"
            ;;
        project)
            echo "./.claude/settings.json"
            ;;
        personal)
            # Extract project hash from path
            local project_hash=$(echo "$skill_path" | sed 's|.*/\.claude/projects/\([^/]*\)/.*|\1|')
            echo "$HOME/.claude/projects/$project_hash/.claude/settings.json"
            ;;
    esac
}

# Remove hook from settings.json
remove_hook_from_settings() {
    local settings_path="$1"

    if [ ! -f "$settings_path" ]; then
        log_warning "Settings file not found: $settings_path"
        return 0
    fi

    # Check if before-commit hook exists
    if ! grep -q '"before-commit"' "$settings_path"; then
        log_info "No before-commit hook found in $settings_path"
        return 0
    fi

    # Create backup
    cp "$settings_path" "$settings_path.backup"

    # Remove before-commit line (simple approach)
    # This assumes it's on its own line
    sed -i.tmp '/^[[:space:]]*"before-commit":/d' "$settings_path"
    rm -f "$settings_path.tmp"

    # Clean up empty hooks object if needed
    if grep -q '"hooks": {[[:space:]]*}' "$settings_path"; then
        sed -i.tmp '/^[[:space:]]*"hooks": {[[:space:]]*}$/d' "$settings_path"
        rm -f "$settings_path.tmp"
    fi

    # Validate JSON
    if ! python3 -m json.tool "$settings_path" > /dev/null 2>&1; then
        log_warning "Settings file JSON may be invalid after removal"
        log_info "Backup saved to: $settings_path.backup"
        return 1
    fi

    log_success "Removed hook from $settings_path"
    return 0
}

# Remove skill files
remove_skill_files() {
    local skill_path="$1"

    if [ ! -d "$skill_path" ]; then
        log_warning "Skill directory not found: $skill_path"
        return 0
    fi

    # Remove skill directory
    rm -rf "$skill_path"
    log_success "Removed skill files: $skill_path"

    # Remove empty parent directories (only if they're empty and ours)
    local parent_dir=$(dirname "$skill_path")
    if [ -d "$parent_dir" ] && [ -z "$(ls -A "$parent_dir")" ]; then
        rmdir "$parent_dir" 2>/dev/null || true
        log_info "Removed empty directory: $parent_dir"
    fi
}

# Uninstall single skill
uninstall_skill() {
    local scope="$1"
    local skill_path="$2"

    echo
    log_info "Uninstalling $scope skill from: $skill_path"

    # Get settings path
    local settings_path=$(get_settings_for_skill "$skill_path" "$scope")

    # Remove hook
    if [ -f "$settings_path" ]; then
        remove_hook_from_settings "$settings_path"
    else
        log_info "Settings file not found: $settings_path"
    fi

    # Remove skill files
    remove_skill_files "$skill_path"

    log_success "Uninstalled $scope skill"
}

# Main uninstall flow
main() {
    echo
    echo "╔════════════════════════════════════════════════════════════╗"
    echo "║     Gemtracker Claude Code Skill Uninstaller               ║"
    echo "╚════════════════════════════════════════════════════════════╝"
    echo

    # Find all installed skills
    log_info "Searching for installed skills..."
    local skills_found=0
    local installed_skills=()

    while IFS=: read -r scope path; do
        skills_found=$((skills_found + 1))
        installed_skills+=("$scope|$path")
        echo "  $skills_found. $scope: $path"
    done < <(find_all_skills)

    if [ $skills_found -eq 0 ]; then
        log_warning "No installed gemtracker skills found"
        exit 0
    fi

    echo

    if [ $skills_found -eq 1 ]; then
        # Only one skill, uninstall it
        local skill="${installed_skills[0]}"
        local scope="${skill%%|*}"
        local path="${skill##*|}"

        read -p "Uninstall this skill? (y/n) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            uninstall_skill "$scope" "$path"
            log_success "Uninstallation complete"
        else
            log_info "Uninstall cancelled"
        fi
    else
        # Multiple skills, let user choose
        read -p "Which skill to uninstall? (1-$skills_found, or 'all'): " choice

        if [ "$choice" = "all" ]; then
            read -p "Uninstall ALL skills? (y/n) " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                for skill in "${installed_skills[@]}"; do
                    local scope="${skill%%|*}"
                    local path="${skill##*|}"
                    uninstall_skill "$scope" "$path"
                done
                log_success "All skills uninstalled"
            else
                log_info "Uninstall cancelled"
            fi
        elif [[ "$choice" =~ ^[0-9]+$ ]] && [ "$choice" -ge 1 ] && [ "$choice" -le $skills_found ]; then
            local skill="${installed_skills[$((choice - 1))]}"
            local scope="${skill%%|*}"
            local path="${skill##*|}"

            read -p "Uninstall this skill? (y/n) " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                uninstall_skill "$scope" "$path"
                log_success "Uninstallation complete"
            else
                log_info "Uninstall cancelled"
            fi
        else
            log_error "Invalid choice"
            exit 1
        fi
    fi

    echo
    echo "Note: gemtracker CLI remains installed"
    echo "To uninstall it:"
    echo "  brew uninstall gemtracker  # macOS"
    echo "  sudo apt remove gemtracker  # Linux (if installed via package manager)"
    echo
}

main "$@"

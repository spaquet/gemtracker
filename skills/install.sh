#!/usr/bin/env bash
set -euo pipefail

REPO_RAW="https://raw.githubusercontent.com/spaquet/gemtracker/main"
TMP_DIR=""

log() { printf '%s\n' "$*"; }
die() { log "Error: $*" >&2; exit 1; }

cleanup() {
  [ -n "$TMP_DIR" ] && rm -rf "$TMP_DIR"
}
trap cleanup EXIT

script_dir() {
  cd "$(dirname "${BASH_SOURCE[0]}")" && pwd
}

prepare_source() {
  local dir
  dir="$(script_dir)"

  if [ -f "$dir/gemtracker/SKILL.md" ]; then
    printf '%s\n' "$dir/gemtracker"
    return
  fi

  command -v curl >/dev/null 2>&1 || die "curl is required for remote install"

  TMP_DIR="$(mktemp -d)"
  mkdir -p "$TMP_DIR/scripts"
  curl -fsSL "$REPO_RAW/skills/gemtracker/SKILL.md" -o "$TMP_DIR/SKILL.md"
  curl -fsSL "$REPO_RAW/skills/gemtracker/scripts/analyze.sh" -o "$TMP_DIR/scripts/analyze.sh"
  chmod +x "$TMP_DIR/scripts/analyze.sh"
  printf '%s\n' "$TMP_DIR"
}

copy_skill() {
  local source_dir="$1"
  local target_dir="$2"

  mkdir -p "$target_dir/scripts"
  cp "$source_dir/SKILL.md" "$target_dir/SKILL.md"
  cp "$source_dir/scripts/analyze.sh" "$target_dir/scripts/analyze.sh"
  chmod +x "$target_dir/scripts/analyze.sh"
  log "Installed skill: $target_dir"
}

install_git_hook() {
  git rev-parse --git-dir >/dev/null 2>&1 || {
    log "Skipped Git hook: not inside a Git repo"
    return
  }

  local git_dir hook
  git_dir="$(git rev-parse --git-dir)"
  hook="$git_dir/hooks/pre-commit"
  mkdir -p "$(dirname "$hook")"

  if [ -f "$hook" ] && grep -q "gemtracker pre-commit" "$hook"; then
    log "Git hook already installed: $hook"
    return
  fi

  if [ ! -f "$hook" ]; then
    printf '%s\n' '#!/usr/bin/env bash' 'set -e' > "$hook"
  fi

  cat >> "$hook" <<'HOOK'

# gemtracker pre-commit
if command -v gemtracker >/dev/null 2>&1; then
  mkdir -p "$(git rev-parse --git-dir)/gemtracker"
  gemtracker . --report json > "$(git rev-parse --git-dir)/gemtracker/latest.json" || true
fi
HOOK

  chmod +x "$hook"
  log "Installed shared Git hook: $hook"
}

main() {
  local source_dir
  source_dir="$(prepare_source)"

  if ! command -v gemtracker >/dev/null 2>&1; then
    log "Warning: gemtracker CLI not found. Install it with:"
    log "  brew tap spaquet/gemtracker && brew install gemtracker"
  fi

  copy_skill "$source_dir" "$HOME/.claude/skills/gemtracker"
  copy_skill "$source_dir" "$HOME/.codex/skills/gemtracker"
  install_git_hook

  log "Done. Claude can use /gemtracker; Codex can use natural language like: use gemtracker to audit this repo."
}

main "$@"

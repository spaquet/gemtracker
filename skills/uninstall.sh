#!/usr/bin/env bash
set -euo pipefail

remove_dir() {
  local dir="$1"
  if [ -d "$dir" ]; then
    rm -rf "$dir"
    printf 'Removed skill: %s\n' "$dir"
  fi
}

remove_hook() {
  git rev-parse --git-dir >/dev/null 2>&1 || return 0

  local hook tmp
  hook="$(git rev-parse --git-dir)/hooks/pre-commit"
  [ -f "$hook" ] || return 0
  grep -q "gemtracker pre-commit" "$hook" || return 0

  tmp="$(mktemp)"
  awk '
    /^# gemtracker pre-commit$/ { skip = 1; next }
    skip && /^fi$/ { skip = 0; next }
    !skip { print }
  ' "$hook" > "$tmp"
  mv "$tmp" "$hook"
  chmod +x "$hook"
  printf 'Removed Git hook block: %s\n' "$hook"
}

remove_dir "$HOME/.claude/skills/gemtracker"
remove_dir "$HOME/.codex/skills/gemtracker"
remove_hook

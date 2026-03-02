#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
SOURCE_DIR="$REPO_ROOT/skills"

TARGET_DIRS=(
  ".agents/skills"
  ".claude/skills"
  ".gemini/skills"
  ".github/skills"
  ".cursor/skills"
  ".opencode/skills"
)

mode="sync"
strict=0

usage() {
  cat <<'EOF' >&2
usage: sync-skills.sh [--sync|--check] [--strict]

  --sync   copy source skills to all targets (default)
  --check  validate targets against source
  --strict mirror mode: target must exactly match source

Default mode is non-destructive:
- sync keeps target-only files and updates/creates source-managed files
- check allows target-only files but fails on missing/changed source-managed files
EOF
}

mode_set=0
for arg in "$@"; do
  case "$arg" in
    --sync|--check)
      next_mode="${arg#--}"
      if [[ "$mode_set" -eq 1 && "$mode" != "$next_mode" ]]; then
        echo "conflicting mode flags: --$mode and $arg" >&2
        usage
        exit 2
      fi
      mode="$next_mode"
      mode_set=1
      ;;
    --strict)
      strict=1
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      usage
      exit 2
      ;;
  esac
done

if [[ ! -d "$SOURCE_DIR" ]]; then
  echo "skills source directory not found: $SOURCE_DIR" >&2
  exit 1
fi

sync_target() {
  local target_rel="$1"
  local target_abs="$REPO_ROOT/$target_rel"

  if [[ "$strict" -eq 1 ]]; then
    rm -rf "$target_abs"
    mkdir -p "$target_abs"
    cp -a "$SOURCE_DIR/." "$target_abs/"
    return 0
  fi

  # Non-destructive sync: preserve target-local files, update source-managed files.
  mkdir -p "$target_abs"
  cp -a "$SOURCE_DIR/." "$target_abs/"
}

check_target() {
  local target_rel="$1"
  local target_abs="$REPO_ROOT/$target_rel"

  if [[ ! -d "$target_abs" ]]; then
    echo "missing skill directory: $target_rel" >&2
    return 1
  fi

  local diff_output
  diff_output="$(diff -qr "$SOURCE_DIR" "$target_abs" || true)"

  if [[ -z "$diff_output" ]]; then
    return 0
  fi

  if [[ "$strict" -eq 1 ]]; then
    echo "skill directory is out of sync: $target_rel" >&2
    printf '%s\n' "$diff_output" >&2
    return 1
  fi

  local has_violation=0
  while IFS= read -r line; do
    [[ -z "$line" ]] && continue

    # In non-strict mode, allow files that exist only in target.
    if [[ "$line" == "Only in $target_abs"* ]]; then
      continue
    fi

    has_violation=1
    printf '%s\n' "$line" >&2
  done <<< "$diff_output"

  if [[ "$has_violation" -eq 1 ]]; then
    echo "skill directory is out of sync: $target_rel" >&2
    return 1
  fi
}

if [[ "$mode" == "sync" ]]; then
  for target in "${TARGET_DIRS[@]}"; do
    sync_target "$target"
  done
fi

failed=0
for target in "${TARGET_DIRS[@]}"; do
  if ! check_target "$target"; then
    failed=1
  fi
done

if [[ "$failed" -ne 0 ]]; then
  exit 1
fi

if [[ "$strict" -eq 1 ]]; then
  echo "skills ${mode} complete (strict)"
else
  echo "skills ${mode} complete (non-destructive)"
fi

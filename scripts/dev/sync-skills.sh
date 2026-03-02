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
if [[ "${1:-}" == "--check" ]]; then
  mode="check"
elif [[ -n "${1:-}" && "${1:-}" != "--sync" ]]; then
  echo "usage: $0 [--sync|--check]" >&2
  exit 2
fi

if [[ ! -d "$SOURCE_DIR" ]]; then
  echo "skills source directory not found: $SOURCE_DIR" >&2
  exit 1
fi

sync_target() {
  local target_rel="$1"
  local target_abs="$REPO_ROOT/$target_rel"

  rm -rf "$target_abs"
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

  if ! diff -qr "$SOURCE_DIR" "$target_abs" >/dev/null; then
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

echo "skills ${mode} complete"

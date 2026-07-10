#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEFAULT_REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
REPO_ROOT="${SYNC_REPO_ROOT:-$DEFAULT_REPO_ROOT}"
SOURCE_DIR="${SYNC_SKILLS_SOURCE_DIR:-$REPO_ROOT/.agents/skills}"
DIFF_BIN="${SYNC_DIFF_BIN:-diff}"
COPY_BIN="${SYNC_COPY_BIN:-cp}"

# consumer_id|target_path|requiredness. Requiredness changes are reviewed here;
# this registry owns generation targets, not workflow policy. Canonical thin
# wrappers and their evals are mirrored byte-for-byte from .agents/skills.
TARGETS=(
  "claude_skills|.claude/skills|optional"
  "gemini_skills|.gemini/skills|optional"
  "github_skills|.github/skills|optional"
  "cursor_skills|.cursor/skills|optional"
  "opencode_skills|.opencode/skills|optional"
)

# Hermetic integration tests may replace the registry and root. Normal runs
# use the checked-in defaults above. A set-but-empty override is invalid: it
# must not turn a check into a successful zero-target no-op.
if [[ -n "${SYNC_SKILLS_TARGETS+x}" ]]; then
  TARGETS=()
  while IFS= read -r target_row; do
    [[ -z "$target_row" ]] && continue
    TARGETS+=("$target_row")
  done <<<"$SYNC_SKILLS_TARGETS"
fi

mode="sync"
strict=0

usage() {
  cat <<'EOF' >&2
usage: sync-skills.sh [--sync|--check] [--strict]

  --sync   copy source skills to all targets (default)
  --check  validate targets against source
  --strict mirror mode: present targets must exactly match source; --sync recreates targets

Default mode is non-destructive:
- sync keeps target-only files and updates/creates source-managed files
- check allows target-only files but fails on missing/changed source-managed files

Canonical source: .agents/skills
Runtime mirrors: .claude/skills, .gemini/skills, .github/skills, .cursor/skills, .opencode/skills
Mirrors are generated local artifacts; clean checkouts may not have them until --sync is run.
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

safe_relative_target() {
  local target="$1"
  local segment
  local -a segments

  [[ -n "$target" ]] || return 1
  [[ "$target" != /* && "$target" != */ && "$target" != *//* ]] || return 1

  IFS='/' read -r -a segments <<<"$target"
  for segment in "${segments[@]}"; do
    [[ -n "$segment" && "$segment" != "." && "$segment" != ".." ]] || return 1
  done
}

target_has_symlink_component() {
  local target="$1"
  local current="$REPO_ROOT"
  local segment
  local -a segments

  IFS='/' read -r -a segments <<<"$target"
  for segment in "${segments[@]}"; do
    current="$current/$segment"
    [[ ! -L "$current" ]] || return 0
  done
  return 1
}

validate_target_registry() {
  local target_row consumer_id target requiredness extra seen target_abs
  local root_abs source_abs
  local -a consumers=()
  local -a targets=()

  if [[ "${#TARGETS[@]}" -eq 0 ]]; then
    echo "invalid target registry: no targets" >&2
    return 1
  fi

  root_abs="$(cd "$REPO_ROOT" && pwd -P)"
  source_abs="$(cd "$SOURCE_DIR" && pwd -P)"

  for target_row in "${TARGETS[@]}"; do
    IFS='|' read -r consumer_id target requiredness extra <<<"$target_row"
    if [[ -z "$consumer_id" || -z "$target" || -z "$requiredness" || -n "$extra" || "$target_row" != "$consumer_id|$target|$requiredness" ]]; then
      echo "invalid target registry row: $target_row" >&2
      return 1
    fi
    if [[ "$requiredness" != "required" && "$requiredness" != "optional" ]]; then
      echo "$consumer_id $target invalid_requiredness=$requiredness" >&2
      return 1
    fi
    if ! safe_relative_target "$target"; then
      echo "$consumer_id $target invalid_target_path" >&2
      return 1
    fi
    if target_has_symlink_component "$target"; then
      echo "$consumer_id $target invalid_target_symlink" >&2
      return 1
    fi
    target_abs="$root_abs/$target"
    if [[ "$target_abs" == "$source_abs" || "$target_abs" == "$source_abs/"* || "$source_abs" == "$target_abs/"* ]]; then
      echo "$consumer_id $target target_overlaps_source" >&2
      return 1
    fi
    if [[ "${#consumers[@]}" -gt 0 ]]; then
      for seen in "${consumers[@]}"; do
        if [[ "$seen" == "$consumer_id" ]]; then
          echo "$consumer_id duplicate_consumer" >&2
          return 1
        fi
      done
    fi
    if [[ "${#targets[@]}" -gt 0 ]]; then
      for seen in "${targets[@]}"; do
        if [[ "$seen" == "$target" ]]; then
          echo "$consumer_id $target duplicate_target" >&2
          return 1
        fi
        if [[ "$seen" == "$target/"* || "$target" == "$seen/"* ]]; then
          echo "$consumer_id $target overlapping_target=$seen" >&2
          return 1
        fi
      done
    fi
    consumers+=("$consumer_id")
    targets+=("$target")
  done
}

if [[ ! -d "$SOURCE_DIR" ]]; then
  echo "canonical_available: false" >&2
  echo "skills source directory not found: $SOURCE_DIR" >&2
  exit 1
fi

if ! validate_target_registry; then
  exit 2
fi

sync_target() {
  local target_rel="$1"
  local target_abs="$REPO_ROOT/$target_rel"

  if [[ "$strict" -eq 1 ]]; then
    rm -rf "$target_abs" || return 1
    mkdir -p "$target_abs" || return 1
    "$COPY_BIN" -a "$SOURCE_DIR/." "$target_abs/" || return 1
    return 0
  fi

  # Non-destructive sync: preserve target-local files, update source-managed files.
  mkdir -p "$target_abs" || return 1
  "$COPY_BIN" -a "$SOURCE_DIR/." "$target_abs/" || return 1
}

check_target() {
  local target_rel="$1"
  local target_abs="$REPO_ROOT/$target_rel"
  local expected_abs="$2"

  local diff_output
  local diff_status
  if diff_output="$("$DIFF_BIN" -qr "$expected_abs" "$target_abs" 2>&1)"; then
    diff_status=0
  else
    diff_status=$?
  fi

  if [[ "$diff_status" -eq 0 ]]; then
    return 0
  fi

  if [[ "$diff_status" -gt 1 ]]; then
    echo "mirror compare failed: $target_rel" >&2
    printf '%s\n' "$diff_output" >&2
    return 2
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

expected_root="$(mktemp -d)"
cleanup() {
  rm -rf "$expected_root"
}
trap cleanup EXIT

echo "canonical_available: true"
if ! mkdir -p "$expected_root/skills" || ! "$COPY_BIN" -a "$SOURCE_DIR/." "$expected_root/skills/"; then
  echo "mirror_render_failed" >&2
  exit 1
fi

failed=0
checked=0
missing=0
for target_row in "${TARGETS[@]}"; do
  IFS='|' read -r consumer_id target requiredness <<<"$target_row"

  if [[ "$mode" == "sync" ]]; then
    if ! sync_target "$target"; then
      echo "$consumer_id $target mirror_render_failed" >&2
      failed=1
      continue
    fi
  fi

  if [[ -d "$REPO_ROOT/$target" ]]; then
    checked=$((checked + 1))
  else
    missing=$((missing + 1))
    if [[ "$requiredness" == "required" ]]; then
      echo "$consumer_id $target mirror_required_missing" >&2
      failed=1
    else
      echo "$consumer_id $target mirror_optional_absent"
    fi
    continue
  fi

  if check_target "$target" "$expected_root/skills"; then
    echo "$consumer_id $target mirror_present_in_sync"
  else
    check_status=$?
    if [[ "$check_status" -eq 2 ]]; then
      echo "$consumer_id $target mirror_compare_failed" >&2
    else
      echo "$consumer_id $target mirror_present_stale" >&2
    fi
    failed=1
  fi
done

if [[ "$failed" -ne 0 ]]; then
  exit 1
fi

if [[ "$strict" -eq 1 ]]; then
  echo "skills ${mode} complete (strict; ${checked} present, ${missing} absent)"
else
  echo "skills ${mode} complete (non-destructive; ${checked} present, ${missing} absent)"
fi

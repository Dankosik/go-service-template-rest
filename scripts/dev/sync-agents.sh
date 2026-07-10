#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEFAULT_REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
REPO_ROOT="${SYNC_REPO_ROOT:-$DEFAULT_REPO_ROOT}"
SOURCE_DIR="${SYNC_AGENTS_SOURCE_DIR:-$REPO_ROOT/.codex/agents}"
DIFF_BIN="${SYNC_DIFF_BIN:-diff}"

# consumer_id|target_path|requiredness. Requiredness changes are reviewed here;
# this registry owns generation targets, not workflow policy.
TARGETS=(
  "claude_agents|.claude/agents|optional"
)

# Hermetic integration tests may replace the registry and root. Normal runs
# use the checked-in defaults above. A set-but-empty override is invalid: it
# must not turn a check into a successful zero-target no-op.
if [[ -n "${SYNC_AGENTS_TARGETS+x}" ]]; then
  TARGETS=()
  while IFS= read -r target_row; do
    [[ -z "$target_row" ]] && continue
    TARGETS+=("$target_row")
  done <<<"$SYNC_AGENTS_TARGETS"
fi

mode="sync"

usage() {
  cat <<'EOF' >&2
usage: sync-agents.sh [--sync|--check]

  --sync   render Claude agent mirrors from .codex/agents (default)
  --check  validate configured agent mirrors against .codex/agents

Canonical source: .codex/agents/*.toml
Configured mirror: claude_agents -> .claude/agents/*.md (optional)
The mirror is a generated local artifact; clean checkouts may not have it until --sync is run.
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
  local target_row consumer_id target_rel requiredness extra seen target_abs
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
    IFS='|' read -r consumer_id target_rel requiredness extra <<<"$target_row"
    if [[ -z "$consumer_id" || -z "$target_rel" || -z "$requiredness" || -n "$extra" || "$target_row" != "$consumer_id|$target_rel|$requiredness" ]]; then
      echo "invalid target registry row: $target_row" >&2
      return 1
    fi
    if [[ "$requiredness" != "required" && "$requiredness" != "optional" ]]; then
      echo "$consumer_id $target_rel invalid_requiredness=$requiredness" >&2
      return 1
    fi
    if ! safe_relative_target "$target_rel"; then
      echo "$consumer_id $target_rel invalid_target_path" >&2
      return 1
    fi
    if target_has_symlink_component "$target_rel"; then
      echo "$consumer_id $target_rel invalid_target_symlink" >&2
      return 1
    fi
    target_abs="$root_abs/$target_rel"
    if [[ "$target_abs" == "$source_abs" || "$target_abs" == "$source_abs/"* || "$source_abs" == "$target_abs/"* ]]; then
      echo "$consumer_id $target_rel target_overlaps_source" >&2
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
        if [[ "$seen" == "$target_rel" ]]; then
          echo "$consumer_id $target_rel duplicate_target" >&2
          return 1
        fi
        if [[ "$seen" == "$target_rel/"* || "$target_rel" == "$seen/"* ]]; then
          echo "$consumer_id $target_rel overlapping_target=$seen" >&2
          return 1
        fi
      done
    fi
    consumers+=("$consumer_id")
    targets+=("$target_rel")
  done
}

if [[ ! -d "$SOURCE_DIR" ]]; then
  echo "canonical_available: false" >&2
  echo "agent source directory not found: $SOURCE_DIR" >&2
  exit 1
fi

if ! validate_target_registry; then
  exit 2
fi

yaml_quote() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  printf '"%s"' "$value"
}

toml_string_key() {
  local key="$1"
  local file="$2"

  awk -v key="$key" '
    $1 == key && $2 == "=" {
      value = $0
      sub("^[^=]+=[[:space:]]*\"", "", value)
      sub("\"[[:space:]]*$", "", value)
      print value
      found = 1
      exit
    }
    END { if (!found) exit 1 }
  ' "$file"
}

developer_instructions() {
  local file="$1"

  awk '
    /^developer_instructions = """$/ { inside = 1; found = 1; next }
    /^"""$/ && inside { closed = 1; inside = 0; exit }
    inside { print }
    END { if (!found || !closed) exit 1 }
  ' "$file"
}

render_agent() {
  local source_file="$1"
  local name description instructions

  name="$(toml_string_key name "$source_file")" || return 1
  description="$(toml_string_key description "$source_file")" || return 1
  instructions="$(developer_instructions "$source_file")" || return 1

  {
    echo "---"
    printf 'name: %s\n' "$name"
    printf 'description: %s\n' "$(yaml_quote "$description")"
    echo "tools: Read, Grep, Glob"
    echo "---"
    echo
    printf '%s\n' "$instructions"
  }
}

render_all() {
  local output_dir="$1"
  local source_file target_file base

  mkdir -p "$output_dir" || return 1
  for source_file in "$SOURCE_DIR"/*.toml; do
    [[ -e "$source_file" ]] || continue
    base="$(basename "$source_file" .toml)"
    target_file="$output_dir/$base.md"
    if ! render_agent "$source_file" >"$target_file"; then
      rm -f "$target_file"
      return 1
    fi
  done
}

expected_dir="$(mktemp -d)"
cleanup() {
  rm -rf "$expected_dir"
}
trap cleanup EXIT

echo "canonical_available: true"
if ! render_all "$expected_dir"; then
  echo "mirror_render_failed" >&2
  exit 1
fi

failed=0
for target_row in "${TARGETS[@]}"; do
  IFS='|' read -r consumer_id target_rel requiredness <<<"$target_row"
  target_abs="$REPO_ROOT/$target_rel"

  if [[ "$mode" == "sync" ]]; then
    rm -rf "$target_abs"
    if ! render_all "$target_abs"; then
      echo "$consumer_id $target_rel mirror_render_failed" >&2
      failed=1
      continue
    fi
  fi

  if [[ ! -d "$target_abs" ]]; then
    if [[ "$requiredness" == "required" ]]; then
      echo "$consumer_id $target_rel mirror_required_missing" >&2
      failed=1
    else
      echo "$consumer_id $target_rel mirror_optional_absent"
    fi
    continue
  fi

  if diff_output="$("$DIFF_BIN" -qr "$expected_dir" "$target_abs" 2>&1)"; then
    diff_status=0
  else
    diff_status=$?
  fi
  if [[ "$diff_status" -eq 1 ]]; then
    echo "$consumer_id $target_rel mirror_present_stale" >&2
    printf '%s\n' "$diff_output" >&2
    failed=1
  elif [[ "$diff_status" -gt 1 ]]; then
    echo "$consumer_id $target_rel mirror_compare_failed" >&2
    printf '%s\n' "$diff_output" >&2
    failed=1
  else
    echo "$consumer_id $target_rel mirror_present_in_sync"
  fi
done

if [[ "$failed" -ne 0 ]]; then
  exit 1
fi

echo "agents ${mode} complete"

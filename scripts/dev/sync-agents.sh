#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
SOURCE_DIR="$REPO_ROOT/.codex/agents"
TARGET_DIR="$REPO_ROOT/.claude/agents"

mode="sync"

usage() {
  cat <<'EOF' >&2
usage: sync-agents.sh [--sync|--check]

  --sync   render Claude agent mirrors from .codex/agents (default)
  --check  validate .claude/agents mirrors against .codex/agents

Canonical source: .codex/agents/*.toml
Runtime mirror: .claude/agents/*.md
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

if [[ ! -d "$SOURCE_DIR" ]]; then
  echo "agent source directory not found: $SOURCE_DIR" >&2
  exit 1
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
    /^developer_instructions = """$/ { inside = 1; next }
    /^"""$/ && inside { exit }
    inside { print }
  ' "$file"
}

render_agent() {
  local source_file="$1"
  local name description

  name="$(toml_string_key name "$source_file")"
  description="$(toml_string_key description "$source_file")"

  {
    echo "---"
    printf 'name: %s\n' "$name"
    printf 'description: %s\n' "$(yaml_quote "$description")"
    echo "tools: Read, Grep, Glob"
    echo "---"
    echo
    developer_instructions "$source_file"
  }
}

render_all() {
  local output_dir="$1"
  local source_file target_file base

  mkdir -p "$output_dir"
  for source_file in "$SOURCE_DIR"/*.toml; do
    [[ -e "$source_file" ]] || continue
    base="$(basename "$source_file" .toml)"
    target_file="$output_dir/$base.md"
    render_agent "$source_file" >"$target_file"
  done
}

if [[ "$mode" == "sync" ]]; then
  render_all "$TARGET_DIR"
fi

expected_dir="$(mktemp -d)"
cleanup() {
  rm -rf "$expected_dir"
}
trap cleanup EXIT

render_all "$expected_dir"

diff_output="$(diff -qr "$expected_dir" "$TARGET_DIR" || true)"
if [[ -n "$diff_output" ]]; then
  echo "agent mirrors are out of sync: .claude/agents" >&2
  printf '%s\n' "$diff_output" >&2
  exit 1
fi

echo "agents ${mode} complete"

#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
AGENT_SCRIPT="$REPO_ROOT/scripts/dev/sync-agents.sh"
SKILL_SCRIPT="$REPO_ROOT/scripts/dev/sync-skills.sh"
TMP_ROOT="$(mktemp -d)"
DIFF_ERROR_BIN="$TMP_ROOT/diff-error"
COPY_ERROR_BIN="$TMP_ROOT/copy-error"

{
  echo '#!/usr/bin/env bash'
  echo 'echo "injected comparison failure" >&2'
  echo 'exit 2'
} >"$DIFF_ERROR_BIN"
chmod +x "$DIFF_ERROR_BIN"

{
  echo '#!/usr/bin/env bash'
  echo 'echo "injected render failure" >&2'
  echo 'exit 1'
} >"$COPY_ERROR_BIN"
chmod +x "$COPY_ERROR_BIN"

cleanup() {
  rm -rf "$TMP_ROOT"
}
trap cleanup EXIT

repo_fingerprint() {
  {
    git -C "$REPO_ROOT" status --porcelain=v1 --untracked-files=all
    git -C "$REPO_ROOT" diff --binary --no-ext-diff
    git -C "$REPO_ROOT" diff --cached --binary --no-ext-diff

    # Status records untracked names, not their bytes. Hash every non-ignored
    # untracked file plus the ignored mirror roots that these scripts could
    # otherwise mutate without changing git status or a tracked diff.
    while IFS= read -r relative_path; do
      [[ -n "$relative_path" ]] || continue
      printf 'untracked %s\n' "$relative_path"
      if [[ -L "$REPO_ROOT/$relative_path" ]]; then
        readlink "$REPO_ROOT/$relative_path"
      elif [[ -f "$REPO_ROOT/$relative_path" ]]; then
        cksum <"$REPO_ROOT/$relative_path"
      fi
    done < <(git -C "$REPO_ROOT" ls-files --others --exclude-standard)

    for relative_root in \
      .claude/agents .claude/skills .gemini/skills .github/skills \
      .cursor/skills .opencode/skills; do
      [[ -e "$REPO_ROOT/$relative_root" || -L "$REPO_ROOT/$relative_root" ]] || continue
      find "$REPO_ROOT/$relative_root" -print | LC_ALL=C sort | while IFS= read -r absolute_path; do
        relative_path="${absolute_path#"$REPO_ROOT/"}"
        printf 'mirror %s\n' "$relative_path"
        if [[ -L "$absolute_path" ]]; then
          readlink "$absolute_path"
        elif [[ -f "$absolute_path" ]]; then
          cksum <"$absolute_path"
        fi
      done
    done
  } | cksum
}

before_fingerprint="$(repo_fingerprint)"

prepare_root() {
  local name="$1"
  CASE_ROOT="$TMP_ROOT/$name"
  mkdir -p "$CASE_ROOT/.codex/agents" "$CASE_ROOT/.agents/skills"
  cp "$REPO_ROOT/.codex/agents/challenger-agent.toml" "$CASE_ROOT/.codex/agents/"
  cp -a "$REPO_ROOT/.agents/skills/workflow-status" "$CASE_ROOT/.agents/skills/"
}

run_case() {
  local label="$1"
  local expected="$2"
  local marker="$3"
  shift 3

  local status
  if LAST_OUTPUT="$("$@" 2>&1)"; then
    status=0
  else
    status=$?
  fi

  if [[ "$expected" == "pass" && "$status" -ne 0 ]]; then
    echo "sync mirror integration failed: $label expected success"
    printf '%s\n' "$LAST_OUTPUT"
    exit 1
  fi
  if [[ "$expected" == "fail" && "$status" -eq 0 ]]; then
    echo "sync mirror integration failed: $label expected failure"
    printf '%s\n' "$LAST_OUTPUT"
    exit 1
  fi
  if [[ "$LAST_OUTPUT" != *"$marker"* ]]; then
    echo "sync mirror integration failed: $label missing marker $marker"
    printf '%s\n' "$LAST_OUTPUT"
    exit 1
  fi
}

prepare_root "agents-optional-absent"
run_case "agents optional absent" pass mirror_optional_absent \
  env SYNC_REPO_ROOT="$CASE_ROOT" SYNC_AGENTS_TARGETS="agent_optional|mirrors/agents|optional" \
  bash "$AGENT_SCRIPT" --check

prepare_root "skills-optional-absent"
run_case "skills optional absent" pass mirror_optional_absent \
  env SYNC_REPO_ROOT="$CASE_ROOT" SYNC_SKILLS_TARGETS="skill_optional|mirrors/skills|optional" \
  bash "$SKILL_SCRIPT" --check

prepare_root "agents-render-error"
printf 'description = "missing required name"\ndeveloper_instructions = """\nread only\n"""\n' >"$CASE_ROOT/.codex/agents/challenger-agent.toml"
run_case "agents render error" fail mirror_render_failed \
  env SYNC_REPO_ROOT="$CASE_ROOT" SYNC_AGENTS_TARGETS="agent_render|mirrors/agents|optional" \
  bash "$AGENT_SCRIPT" --check

prepare_root "skills-render-error"
run_case "skills render error" fail mirror_render_failed \
  env SYNC_REPO_ROOT="$CASE_ROOT" SYNC_SKILLS_TARGETS="skill_render|mirrors/skills|optional" SYNC_COPY_BIN="$COPY_ERROR_BIN" \
  bash "$SKILL_SCRIPT" --check

prepare_root "agents-exact"
env SYNC_REPO_ROOT="$CASE_ROOT" SYNC_AGENTS_TARGETS="agent_exact|mirrors/agents|optional" \
  bash "$AGENT_SCRIPT" --sync >/dev/null
run_case "agents exact present" pass mirror_present_in_sync \
  env SYNC_REPO_ROOT="$CASE_ROOT" SYNC_AGENTS_TARGETS="agent_exact|mirrors/agents|optional" \
  bash "$AGENT_SCRIPT" --check

prepare_root "skills-exact"
env SYNC_REPO_ROOT="$CASE_ROOT" SYNC_SKILLS_TARGETS="skill_exact|mirrors/skills|optional" \
  bash "$SKILL_SCRIPT" --sync >/dev/null
run_case "skills exact present" pass mirror_present_in_sync \
  env SYNC_REPO_ROOT="$CASE_ROOT" SYNC_SKILLS_TARGETS="skill_exact|mirrors/skills|optional" \
  bash "$SKILL_SCRIPT" --check

prepare_root "agents-stale"
env SYNC_REPO_ROOT="$CASE_ROOT" SYNC_AGENTS_TARGETS="agent_stale|mirrors/agents|optional" \
  bash "$AGENT_SCRIPT" --sync >/dev/null
printf '\n# stale\n' >>"$CASE_ROOT/mirrors/agents/challenger-agent.md"
run_case "agents stale present" fail mirror_present_stale \
  env SYNC_REPO_ROOT="$CASE_ROOT" SYNC_AGENTS_TARGETS="agent_stale|mirrors/agents|optional" \
  bash "$AGENT_SCRIPT" --check

prepare_root "skills-stale"
env SYNC_REPO_ROOT="$CASE_ROOT" SYNC_SKILLS_TARGETS="skill_stale|mirrors/skills|optional" \
  bash "$SKILL_SCRIPT" --sync >/dev/null
printf '\n<!-- stale -->\n' >>"$CASE_ROOT/mirrors/skills/workflow-status/SKILL.md"
run_case "skills stale present" fail mirror_present_stale \
  env SYNC_REPO_ROOT="$CASE_ROOT" SYNC_SKILLS_TARGETS="skill_stale|mirrors/skills|optional" \
  bash "$SKILL_SCRIPT" --check

prepare_root "agents-required-missing"
run_case "agents required missing" fail mirror_required_missing \
  env SYNC_REPO_ROOT="$CASE_ROOT" SYNC_AGENTS_TARGETS="agent_required|mirrors/agents|required" \
  bash "$AGENT_SCRIPT" --check

prepare_root "skills-required-missing"
run_case "skills required missing" fail mirror_required_missing \
  env SYNC_REPO_ROOT="$CASE_ROOT" SYNC_SKILLS_TARGETS="skill_required|mirrors/skills|required" \
  bash "$SKILL_SCRIPT" --check

prepare_root "agents-unsafe-traversal"
mkdir -p "$TMP_ROOT/agents-sibling-sentinel"
printf 'must survive\n' >"$TMP_ROOT/agents-sibling-sentinel/value.txt"
sentinel_before="$(cksum <"$TMP_ROOT/agents-sibling-sentinel/value.txt")"
run_case "agents traversal target rejected" fail invalid_target_path \
  env SYNC_REPO_ROOT="$CASE_ROOT" SYNC_AGENTS_TARGETS="unsafe|../agents-sibling-sentinel|required" \
  bash "$AGENT_SCRIPT" --sync
if [[ ! -f "$TMP_ROOT/agents-sibling-sentinel/value.txt" || "$(cksum <"$TMP_ROOT/agents-sibling-sentinel/value.txt")" != "$sentinel_before" ]]; then
  echo "sync mirror integration failed: traversal target changed sibling sentinel"
  exit 1
fi

prepare_root "skills-unsafe-absolute"
mkdir -p "$TMP_ROOT/skills-absolute-sentinel"
printf 'must survive\n' >"$TMP_ROOT/skills-absolute-sentinel/value.txt"
sentinel_before="$(cksum <"$TMP_ROOT/skills-absolute-sentinel/value.txt")"
run_case "skills absolute target rejected" fail invalid_target_path \
  env SYNC_REPO_ROOT="$CASE_ROOT" SYNC_SKILLS_TARGETS="unsafe|$TMP_ROOT/skills-absolute-sentinel|required" \
  bash "$SKILL_SCRIPT" --sync --strict
if [[ ! -f "$TMP_ROOT/skills-absolute-sentinel/value.txt" || "$(cksum <"$TMP_ROOT/skills-absolute-sentinel/value.txt")" != "$sentinel_before" ]]; then
  echo "sync mirror integration failed: absolute target changed sentinel"
  exit 1
fi

prepare_root "agents-unsafe-symlink"
mkdir -p "$TMP_ROOT/agents-symlink-sentinel"
printf 'must survive\n' >"$TMP_ROOT/agents-symlink-sentinel/value.txt"
ln -s "$TMP_ROOT/agents-symlink-sentinel" "$CASE_ROOT/mirrors-link"
run_case "agents symlink target rejected" fail invalid_target_symlink \
  env SYNC_REPO_ROOT="$CASE_ROOT" SYNC_AGENTS_TARGETS="unsafe|mirrors-link|required" \
  bash "$AGENT_SCRIPT" --sync
if [[ ! -f "$TMP_ROOT/agents-symlink-sentinel/value.txt" ]]; then
  echo "sync mirror integration failed: symlink target removed sentinel"
  exit 1
fi

prepare_root "agents-empty-registry"
empty_targets=$'\n'
run_case "agents empty override rejected" fail "invalid target registry: no targets" \
  env SYNC_REPO_ROOT="$CASE_ROOT" SYNC_AGENTS_TARGETS="$empty_targets" \
  bash "$AGENT_SCRIPT" --check

prepare_root "skills-empty-registry"
run_case "skills empty override rejected" fail "invalid target registry: no targets" \
  env SYNC_REPO_ROOT="$CASE_ROOT" SYNC_SKILLS_TARGETS="$empty_targets" \
  bash "$SKILL_SCRIPT" --check

prepare_root "agents-duplicate-consumer"
duplicate_consumers="same|mirrors/one|optional
same|mirrors/two|optional"
run_case "agents duplicate consumer rejected" fail duplicate_consumer \
  env SYNC_REPO_ROOT="$CASE_ROOT" SYNC_AGENTS_TARGETS="$duplicate_consumers" \
  bash "$AGENT_SCRIPT" --check

prepare_root "skills-duplicate-target"
duplicate_targets="first|mirrors/same|optional
second|mirrors/same|optional"
run_case "skills duplicate target rejected" fail duplicate_target \
  env SYNC_REPO_ROOT="$CASE_ROOT" SYNC_SKILLS_TARGETS="$duplicate_targets" \
  bash "$SKILL_SCRIPT" --check

prepare_root "agents-overlapping-targets"
overlapping_targets="parent|mirrors/agents|optional
child|mirrors/agents/child|optional"
run_case "agents overlapping targets rejected" fail overlapping_target \
  env SYNC_REPO_ROOT="$CASE_ROOT" SYNC_AGENTS_TARGETS="$overlapping_targets" \
  bash "$AGENT_SCRIPT" --check

prepare_root "skills-overlapping-targets"
overlapping_targets="parent|mirrors/skills|optional
child|mirrors/skills/child|optional"
run_case "skills overlapping targets rejected" fail overlapping_target \
  env SYNC_REPO_ROOT="$CASE_ROOT" SYNC_SKILLS_TARGETS="$overlapping_targets" \
  bash "$SKILL_SCRIPT" --check

prepare_root "agents-source-overlap"
agent_source_before="$(cksum <"$CASE_ROOT/.codex/agents/challenger-agent.toml")"
run_case "agents source ancestor target rejected" fail target_overlaps_source \
  env SYNC_REPO_ROOT="$CASE_ROOT" SYNC_AGENTS_TARGETS="unsafe|.codex|required" \
  bash "$AGENT_SCRIPT" --sync
if [[ ! -f "$CASE_ROOT/.codex/agents/challenger-agent.toml" || "$(cksum <"$CASE_ROOT/.codex/agents/challenger-agent.toml")" != "$agent_source_before" ]]; then
  echo "sync mirror integration failed: agent source overlap changed canonical source"
  exit 1
fi

prepare_root "skills-source-overlap"
skill_source_before="$(cksum <"$CASE_ROOT/.agents/skills/workflow-status/SKILL.md")"
run_case "skills source target rejected" fail target_overlaps_source \
  env SYNC_REPO_ROOT="$CASE_ROOT" SYNC_SKILLS_TARGETS="unsafe|.agents/skills|required" \
  bash "$SKILL_SCRIPT" --sync --strict
if [[ ! -f "$CASE_ROOT/.agents/skills/workflow-status/SKILL.md" || "$(cksum <"$CASE_ROOT/.agents/skills/workflow-status/SKILL.md")" != "$skill_source_before" ]]; then
  echo "sync mirror integration failed: skill source overlap changed canonical source"
  exit 1
fi

prepare_root "agents-compare-error"
env SYNC_REPO_ROOT="$CASE_ROOT" SYNC_AGENTS_TARGETS="agent_compare|mirrors/agents|optional" \
  bash "$AGENT_SCRIPT" --sync >/dev/null
run_case "agents compare error" fail mirror_compare_failed \
  env SYNC_REPO_ROOT="$CASE_ROOT" SYNC_AGENTS_TARGETS="agent_compare|mirrors/agents|optional" SYNC_DIFF_BIN="$DIFF_ERROR_BIN" \
  bash "$AGENT_SCRIPT" --check

prepare_root "skills-compare-error"
env SYNC_REPO_ROOT="$CASE_ROOT" SYNC_SKILLS_TARGETS="skill_compare|mirrors/skills|optional" \
  bash "$SKILL_SCRIPT" --sync >/dev/null
run_case "skills compare error" fail mirror_compare_failed \
  env SYNC_REPO_ROOT="$CASE_ROOT" SYNC_SKILLS_TARGETS="skill_compare|mirrors/skills|optional" SYNC_DIFF_BIN="$DIFF_ERROR_BIN" \
  bash "$SKILL_SCRIPT" --check

prepare_root "skills-target-only"
env SYNC_REPO_ROOT="$CASE_ROOT" SYNC_SKILLS_TARGETS="skill_target_only|mirrors/skills|optional" \
  bash "$SKILL_SCRIPT" --sync >/dev/null
printf 'target only\n' >"$CASE_ROOT/mirrors/skills/consumer-note.txt"
run_case "skills non-strict target-only" pass mirror_present_in_sync \
  env SYNC_REPO_ROOT="$CASE_ROOT" SYNC_SKILLS_TARGETS="skill_target_only|mirrors/skills|optional" \
  bash "$SKILL_SCRIPT" --check
run_case "skills strict target-only" fail mirror_present_stale \
  env SYNC_REPO_ROOT="$CASE_ROOT" SYNC_SKILLS_TARGETS="skill_target_only|mirrors/skills|optional" \
  bash "$SKILL_SCRIPT" --check --strict

prepare_root "skills-mixed"
env SYNC_REPO_ROOT="$CASE_ROOT" SYNC_SKILLS_TARGETS="skill_exact|mirrors/exact|optional" \
  bash "$SKILL_SCRIPT" --sync >/dev/null
env SYNC_REPO_ROOT="$CASE_ROOT" SYNC_SKILLS_TARGETS="skill_stale|mirrors/stale|optional" \
  bash "$SKILL_SCRIPT" --sync >/dev/null
printf '\n<!-- stale -->\n' >>"$CASE_ROOT/mirrors/stale/workflow-status/SKILL.md"
mixed_targets="skill_absent|mirrors/absent|optional
skill_exact|mirrors/exact|optional
skill_stale|mirrors/stale|optional"
run_case "skills mixed target aggregation" fail mirror_present_stale \
  env SYNC_REPO_ROOT="$CASE_ROOT" SYNC_SKILLS_TARGETS="$mixed_targets" \
  bash "$SKILL_SCRIPT" --check
for marker in mirror_optional_absent mirror_present_in_sync mirror_present_stale; do
  if [[ "$LAST_OUTPUT" != *"$marker"* ]]; then
    echo "sync mirror integration failed: mixed aggregation missing $marker"
    printf '%s\n' "$LAST_OUTPUT"
    exit 1
  fi
done

after_fingerprint="$(repo_fingerprint)"
if [[ "$before_fingerprint" != "$after_fingerprint" ]]; then
  echo "sync mirror integration failed: repository state changed"
  exit 1
fi

echo "sync mirror integration check passed"

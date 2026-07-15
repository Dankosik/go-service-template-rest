#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CHECKER_PACKAGE="./scripts/ci/hard-skills-check"
TEMP_ROOT=""
EMITTED_DIR=""

cleanup() {
  if [[ -n "$TEMP_ROOT" ]]; then
    rm -rf -- "$TEMP_ROOT"
  fi
}
trap cleanup EXIT

usage() {
  echo "usage: $0 check | run [artifact-dir]" >&2
}

ensure_temp_root() {
  if [[ -z "$TEMP_ROOT" ]]; then
    TEMP_ROOT="$(mktemp -d)"
  fi
}

run_checker() {
  if [[ -n "${HARD_SKILLS_CHECKER:-}" ]]; then
    if [[ ! -x "$HARD_SKILLS_CHECKER" ]]; then
      echo "hard skill evals require executable HARD_SKILLS_CHECKER when set" >&2
      return 1
    fi
    HARD_SKILLS_REPO_ROOT="$ROOT_DIR" "$HARD_SKILLS_CHECKER" "$@"
    return
  fi
  (cd "$ROOT_DIR" && go run "$CHECKER_PACKAGE" "$@")
}

check_manifest() {
  ensure_temp_root
  EMITTED_DIR="$TEMP_ROOT/selected-evals"
  rm -rf -- "$EMITTED_DIR"
  if ! run_checker emit-selected-evals --output-dir "$EMITTED_DIR"; then
    echo "hard skill eval manifest check failed" >&2
    return 1
  fi
  if [[ ! -f "$EMITTED_DIR/manifest.tsv" ]]; then
    echo "hard skill eval manifest check failed: checker did not emit manifest.tsv" >&2
    return 1
  fi

  local count
  count="$(awk -F $'\t' 'NF != 5 || $1 != $2 ":" $3 { exit 2 } { count++ } END { print count+0 }' "$EMITTED_DIR/manifest.tsv")" || {
    echo "hard skill eval manifest check failed: malformed manifest.tsv" >&2
    return 1
  }
  if [[ "$count" != 36 ]]; then
    echo "hard skill eval manifest check failed: expected 36 selected cases, got $count" >&2
    return 1
  fi

  local case_id skill category prompt_rel expected_rel
  while IFS=$'\t' read -r case_id skill category prompt_rel expected_rel; do
    if [[ "$case_id" != "$skill:$category" || ! -s "$EMITTED_DIR/$prompt_rel" || ! -s "$EMITTED_DIR/$expected_rel" ]]; then
      echo "hard skill eval manifest check failed: incomplete case $case_id" >&2
      return 1
    fi
  done <"$EMITTED_DIR/manifest.tsv"
  echo "hard skill eval manifest passed: $count selected cases"
}

read_verdict_value() {
  local key="$1"
  local file="$2"
  awk -F= -v key="$key" '$1 == key { sub(/^[^=]*=/, ""); print; exit }' "$file"
}

require_boolean_verdict() {
  local key="$1"
  local file="$2"
  local value
  value="$(read_verdict_value "$key" "$file")"
  if [[ "$value" != "true" && "$value" != "false" ]]; then
    echo "hard skill eval run failed: $file must contain $key=true or $key=false" >&2
    return 1
  fi
  printf '%s' "$value"
}

optional_boolean_verdict() {
  local key="$1"
  local file="$2"
  local value
  value="$(read_verdict_value "$key" "$file")"
  if [[ -z "$value" ]]; then
    printf 'false'
    return
  fi
  if [[ "$value" != "true" && "$value" != "false" ]]; then
    echo "hard skill eval run failed: $file must contain optional $key=true or $key=false" >&2
    return 1
  fi
  printf '%s' "$value"
}

seal_snapshot() {
  local repo="$1"
  local label="$2"

  while IFS=$'\t' read -r _ skill _ _ _; do
    rm -rf -- "$repo/.agents/skills/$skill/evals"
  done <"$EMITTED_DIR/manifest.tsv"
  git -C "$repo" init --quiet
  git -C "$repo" add --all --force
  git -C "$repo" \
    -c user.name='Hard Skill Eval' \
    -c user.email='hard-skill-eval@local' \
    commit --quiet --no-gpg-sign -m "hard skill eval $label snapshot"
}

assert_snapshot_unchanged() {
  local repo="$1"
  local expected_head="$2"
  local case_id="$3"
  local variant="$4"
  local actual_head status

  if ! actual_head="$(git -C "$repo" rev-parse HEAD 2>/dev/null)"; then
    echo "hard skill eval run failed: adapter damaged $case_id $variant Git snapshot" >&2
    return 1
  fi
  status="$(git -C "$repo" status --porcelain=v1 --untracked-files=all --ignored=matching)"
  if [[ "$actual_head" != "$expected_head" || -n "$status" ]]; then
    echo "hard skill eval run failed: adapter mutated $case_id $variant snapshot" >&2
    echo "  adapters must use private copies and leave --repo unchanged" >&2
    return 1
  fi
}

assert_payloads_unchanged() {
  local prompt_file="$1"
  local prompt_seal="$2"
  local expected_file="$3"
  local expected_seal="$4"
  local case_id="$5"

  if ! cmp -s -- "$prompt_seal" "$prompt_file"; then
    echo "hard skill eval run failed: adapter mutated $case_id prompt payload" >&2
    return 1
  fi
  if ! cmp -s -- "$expected_seal" "$expected_file"; then
    echo "hard skill eval run failed: adapter mutated $case_id expected payload" >&2
    return 1
  fi
}

copy_authoritative_untracked() {
  local candidate_dir="$1"
  local relative_path destination

  while IFS= read -r -d '' relative_path; do
    case "$relative_path" in
      .agents/skills/*|scripts/ci/hard-skills-check/*|scripts/dev/hard-skills-evals.sh)
        destination="$candidate_dir/$relative_path"
        mkdir -p "$(dirname "$destination")"
        cp -a -- "$ROOT_DIR/$relative_path" "$destination"
        ;;
      *)
        ;;
    esac
  done < <(git -C "$ROOT_DIR" ls-files --others --exclude-standard -z)
}

run_pair() {
  local case_id="$1"
  local prompt_file="$2"
  local expected_file="$3"
  local attempt_dir="$4"
  local baseline_dir="$5"
  local candidate_dir="$6"
  local baseline_snapshot="$7"
  local candidate_snapshot="$8"
  local prompt_seal="$9"
  local expected_seal="${10}"
  local authoritative_prompt="${11}"
  local adapter_status=0

  mkdir -p "$attempt_dir"
  "$WORKFLOW_EVAL_RUNNER" \
    --variant baseline --repo "$baseline_dir" --prompt-file "$prompt_file" \
    >"$attempt_dir/baseline-output.txt" 2>"$attempt_dir/baseline-runner.log" || adapter_status=$?
  assert_payloads_unchanged "$prompt_file" "$prompt_seal" "$expected_file" "$expected_seal" "$case_id" || return 1
  if [[ "$adapter_status" -ne 0 ]]; then
    echo "hard skill eval run failed: runner failed for $case_id baseline" >&2
    return 1
  fi
  assert_snapshot_unchanged "$baseline_dir" "$baseline_snapshot" "$case_id" baseline || return 1

  adapter_status=0
  "$WORKFLOW_EVAL_RUNNER" \
    --variant candidate --repo "$candidate_dir" --prompt-file "$prompt_file" \
    >"$attempt_dir/candidate-output.txt" 2>"$attempt_dir/candidate-runner.log" || adapter_status=$?
  assert_payloads_unchanged "$prompt_file" "$prompt_seal" "$expected_file" "$expected_seal" "$case_id" || return 1
  if [[ "$adapter_status" -ne 0 ]]; then
    echo "hard skill eval run failed: runner failed for $case_id candidate" >&2
    return 1
  fi
  assert_snapshot_unchanged "$candidate_dir" "$candidate_snapshot" "$case_id" candidate || return 1

  adapter_status=0
  "$WORKFLOW_EVAL_JUDGE" \
    --case-id "$case_id" --expected-file "$expected_file" \
    --baseline-output "$attempt_dir/baseline-output.txt" \
    --candidate-output "$attempt_dir/candidate-output.txt" \
    >"$attempt_dir/judgment.env" 2>"$attempt_dir/judge.log" || adapter_status=$?
  assert_payloads_unchanged "$prompt_file" "$prompt_seal" "$expected_file" "$expected_seal" "$case_id" || return 1
  if ! cmp -s -- "$prompt_seal" "$authoritative_prompt"; then
    echo "hard skill eval run failed: adapter mutated $case_id authoritative prompt payload" >&2
    return 1
  fi
  if [[ "$adapter_status" -ne 0 ]]; then
    echo "hard skill eval run failed: judge failed for $case_id" >&2
    return 1
  fi
  assert_snapshot_unchanged "$baseline_dir" "$baseline_snapshot" "$case_id" baseline || return 1
  assert_snapshot_unchanged "$candidate_dir" "$candidate_snapshot" "$case_id" candidate || return 1
}

run_evals() {
  check_manifest

  if [[ -z "${WORKFLOW_EVAL_BASE_REF:-}" ]]; then
    echo "hard skill eval run requires explicit WORKFLOW_EVAL_BASE_REF" >&2
    return 1
  fi
  local baseline_commit
  if ! baseline_commit="$(git -C "$ROOT_DIR" rev-parse --verify "$WORKFLOW_EVAL_BASE_REF^{commit}" 2>/dev/null)"; then
    echo "hard skill eval run requires a valid WORKFLOW_EVAL_BASE_REF: $WORKFLOW_EVAL_BASE_REF" >&2
    return 1
  fi
  if [[ "$baseline_commit" != "$WORKFLOW_EVAL_BASE_REF" ]]; then
    echo "hard skill eval run requires WORKFLOW_EVAL_BASE_REF to be the resolved immutable commit: $baseline_commit" >&2
    return 1
  fi
  if [[ -z "${WORKFLOW_EVAL_RUNNER:-}" || ! -x "$WORKFLOW_EVAL_RUNNER" ]]; then
    echo "hard skill eval run requires executable WORKFLOW_EVAL_RUNNER" >&2
    return 1
  fi
  if [[ -z "${WORKFLOW_EVAL_JUDGE:-}" || ! -x "$WORKFLOW_EVAL_JUDGE" ]]; then
    echo "hard skill eval run requires executable WORKFLOW_EVAL_JUDGE" >&2
    return 1
  fi
  if [[ "${WORKFLOW_EVAL_COST_AUTHORIZED:-}" != true ]]; then
    echo "hard skill eval run requires WORKFLOW_EVAL_COST_AUTHORIZED=true before adapters" >&2
    return 1
  fi

  ensure_temp_root
  local baseline_dir="$TEMP_ROOT/baseline"
  local candidate_dir="$TEMP_ROOT/candidate"
  local candidate_patch="$TEMP_ROOT/candidate.patch"
  mkdir -p "$baseline_dir" "$candidate_dir"
  git -C "$ROOT_DIR" archive "$baseline_commit" | tar -x -C "$baseline_dir"
  git -C "$ROOT_DIR" archive HEAD | tar -x -C "$candidate_dir"
  git -C "$ROOT_DIR" diff --binary HEAD -- . >"$candidate_patch"
  if [[ -s "$candidate_patch" ]]; then
    git -C "$candidate_dir" apply --binary "$candidate_patch"
  fi
  copy_authoritative_untracked "$candidate_dir"
  seal_snapshot "$baseline_dir" baseline
  seal_snapshot "$candidate_dir" candidate

  local baseline_snapshot candidate_snapshot
  baseline_snapshot="$(git -C "$baseline_dir" rev-parse HEAD)"
  candidate_snapshot="$(git -C "$candidate_dir" rev-parse HEAD)"
  local artifact_dir="${1:-$ROOT_DIR/.artifacts/test/hard-skill-evals/$(date -u +%Y%m%dT%H%M%SZ)}"
  mkdir -p "$artifact_dir/cases"
  {
    echo "baseline_ref=$WORKFLOW_EVAL_BASE_REF"
    echo "baseline_commit=$baseline_commit"
    echo "baseline_snapshot_commit=$baseline_snapshot"
    echo "candidate_source_head=$(git -C "$ROOT_DIR" rev-parse HEAD)"
    echo "candidate_snapshot_commit=$candidate_snapshot"
    echo "runner=$WORKFLOW_EVAL_RUNNER"
    echo "judge=$WORKFLOW_EVAL_JUDGE"
    echo "run_label=${WORKFLOW_EVAL_RUN_LABEL:-unspecified}"
  } >"$artifact_dir/run.env"
  git -C "$ROOT_DIR" status --short >"$artifact_dir/candidate-status.txt"
  git -C "$ROOT_DIR" diff --binary "$baseline_commit" >"$artifact_dir/candidate.patch"
  printf 'case_id\tattempts\tbaseline_pass\tcandidate_pass\tcandidate_non_regression\taccepted\n' >"$artifact_dir/summary.tsv"

  local failures=0
  local runner_index=0
  local case_id skill category prompt_rel expected_rel
  while IFS=$'\t' read -r case_id skill category prompt_rel expected_rel; do
    local case_dir="$artifact_dir/cases/$skill/$category"
    local payload_seal_dir="$TEMP_ROOT/payload-seals/$skill/$category"
    mkdir -p "$case_dir" "$payload_seal_dir"
    cp "$EMITTED_DIR/$prompt_rel" "$payload_seal_dir/prompt.txt"
    cp "$EMITTED_DIR/$expected_rel" "$payload_seal_dir/expected.txt"
    cp "$payload_seal_dir/prompt.txt" "$case_dir/prompt.txt"
    cp "$payload_seal_dir/expected.txt" "$case_dir/expected.txt"

    local attempt=1
    local all_attempts_accepted=true
    local baseline_pass=false candidate_pass=false non_regression=false disputed=false judge_uncertain=false
    while :; do
      local attempt_dir="$case_dir/attempt-$attempt"
      runner_index=$((runner_index + 1))
      local runner_prompt="$TEMP_ROOT/runner-prompts/$(printf '%06d' "$runner_index")/input.txt"
      mkdir -p "$(dirname "$runner_prompt")"
      cp "$case_dir/prompt.txt" "$runner_prompt"
      run_pair "$case_id" "$runner_prompt" "$case_dir/expected.txt" "$attempt_dir" \
        "$baseline_dir" "$candidate_dir" "$baseline_snapshot" "$candidate_snapshot" \
        "$payload_seal_dir/prompt.txt" "$payload_seal_dir/expected.txt" "$case_dir/prompt.txt" || return 1
      baseline_pass="$(require_boolean_verdict baseline_pass "$attempt_dir/judgment.env")" || return 1
      candidate_pass="$(require_boolean_verdict candidate_pass "$attempt_dir/judgment.env")" || return 1
      non_regression="$(require_boolean_verdict candidate_non_regression "$attempt_dir/judgment.env")" || return 1
      disputed="$(optional_boolean_verdict disputed "$attempt_dir/judgment.env")" || return 1
      judge_uncertain="$(optional_boolean_verdict judge_uncertain "$attempt_dir/judgment.env")" || return 1
      if [[ "$candidate_pass" != true || "$non_regression" != true ]]; then
        all_attempts_accepted=false
      fi
      if [[ "$attempt" -eq 1 && ("$disputed" == true || "$judge_uncertain" == true) ]]; then
        attempt=2
        continue
      fi
      if [[ "$attempt" -eq 2 && ("$disputed" == true || "$judge_uncertain" == true) ]]; then
        printf '%s\t%d\t%s\t%s\t%s\tfalse\n' \
          "$case_id" "$attempt" "$baseline_pass" "$candidate_pass" "$non_regression" \
          >>"$artifact_dir/summary.tsv"
        echo "hard skill eval run failed: $case_id remained disputed or judge-uncertain after one repeat" >&2
        return 1
      fi
      break
    done

    if [[ "$all_attempts_accepted" != true ]]; then
      failures=$((failures + 1))
    fi
    printf '%s\t%d\t%s\t%s\t%s\t%s\n' \
      "$case_id" "$attempt" "$baseline_pass" "$candidate_pass" "$non_regression" "$all_attempts_accepted" \
      >>"$artifact_dir/summary.tsv"
  done <"$EMITTED_DIR/manifest.tsv"

  if [[ "$failures" -ne 0 ]]; then
    echo "hard skill evals failed: $failures selected case(s) failed or remained uncertain" >&2
    echo "  summary: $artifact_dir/summary.tsv" >&2
    return 1
  fi
  echo "hard skill evals passed: $(wc -l <"$EMITTED_DIR/manifest.tsv" | tr -d ' ') selected cases"
  echo "  artifacts: $artifact_dir"
}

case "${1:-}" in
  check)
    if [[ "$#" -ne 1 ]]; then
      usage
      exit 2
    fi
    check_manifest
    ;;
  run)
    shift
    if [[ "$#" -gt 1 ]]; then
      usage
      exit 2
    fi
    run_evals "${1:-}"
    ;;
  *)
    usage
    exit 2
    ;;
esac

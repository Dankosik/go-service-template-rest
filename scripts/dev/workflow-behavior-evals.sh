#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
EVAL_REPO_PATH="docs/spec-first-workflow-evals.md"
EVAL_FILE="${ROOT_DIR}/${EVAL_REPO_PATH}"
EXPECTED_IDS="E01 E02 E03 E04 E05 E06 E07 E08 E09 E10 E11 E12 E13 E14 E15 E16 E17 E18 E19 E20 E21 E22 E23 E24 E25 E26"
INVARIANT_IDS="E02 E04 E05 E06 E09 E10 E11 E12 E13 E14 E15 E16 E17 E18 E19 E20 E21 E22 E23 E24 E25 E26"
TEMP_ROOT=""

cleanup() {
  if [[ -n "${TEMP_ROOT}" ]]; then
    rm -rf -- "${TEMP_ROOT}"
  fi
}

ensure_temp_root() {
  if [[ -z "${TEMP_ROOT}" ]]; then
    TEMP_ROOT="$(mktemp -d)"
  fi
}

trap cleanup EXIT

usage() {
  echo "usage: $0 check | run [artifact-dir]"
}

extract_cases() {
  awk '
    function emit() {
      if (id != "") print id "\t" prompt "\t" pass
    }
    /^### E[0-9][0-9] / {
      emit()
      id = $2
      prompt = ""
      pass = ""
      next
    }
    id != "" && /^Prompt: / {
      prompt = substr($0, 9)
      next
    }
    id != "" && /^Pass: / {
      pass = substr($0, 7)
      next
    }
    END { emit() }
  ' "${EVAL_FILE}"
}

check_manifest() {
  if [[ ! -f "${EVAL_FILE}" ]]; then
    echo "workflow behavior eval check failed: missing ${EVAL_FILE#"${ROOT_DIR}/"}"
    return 1
  fi

  local cases_file
  ensure_temp_root
  cases_file="${TEMP_ROOT}/manifest-cases.tsv"
  extract_cases >"${cases_file}"

  local actual_ids
  actual_ids="$(awk -F '\t' '{ print $1 }' "${cases_file}" | paste -sd ' ' -)"
  if [[ "${actual_ids}" != "${EXPECTED_IDS}" ]]; then
    echo "workflow behavior eval check failed: expected exactly ${EXPECTED_IDS}"
    echo "  actual: ${actual_ids:-none}"
    return 1
  fi

  if ! awk -F '\t' 'NF != 3 || $2 == "" || $3 == "" { exit 1 }' "${cases_file}"; then
    echo "workflow behavior eval check failed: every case needs one non-empty Prompt and Pass line"
    return 1
  fi

  if ! awk '
    function valid_previous() {
      return id == "" || (prompts == 1 && passes == 1)
    }
    /^### E[0-9][0-9] / {
      if (!valid_previous()) exit 1
      id = $2
      prompts = 0
      passes = 0
      next
    }
    id != "" && /^Prompt: / { prompts++ }
    id != "" && /^Pass: / { passes++ }
    END { if (!valid_previous()) exit 1 }
  ' "${EVAL_FILE}"; then
    echo "workflow behavior eval check failed: every case needs exactly one Prompt and one Pass line"
    return 1
  fi

  local acceptance_line
  acceptance_line="$(grep -E '^- .* are invariant cases and must all pass\.$' "${EVAL_FILE}" || true)"
  local actual_invariants
  actual_invariants="$(printf '%s\n' "${acceptance_line}" | grep -Eo 'E[0-9][0-9]' | paste -sd ' ' -)"
  if [[ "${actual_invariants}" != "${INVARIANT_IDS}" ]]; then
    echo "workflow behavior eval check failed: invariant IDs must be ${INVARIANT_IDS}"
    echo "  actual: ${actual_invariants:-none}"
    return 1
  fi

  echo "workflow behavior eval manifest passed: 26 cases, 22 invariants"
}

is_invariant() {
  case " ${INVARIANT_IDS} " in
    *" $1 "*) return 0 ;;
    *) return 1 ;;
  esac
}

read_verdict_value() {
  local key="$1"
  local file="$2"
  awk -F= -v key="${key}" '$1 == key { sub(/^[^=]*=/, ""); print; exit }' "${file}"
}

require_boolean_verdict() {
  local key="$1"
  local file="$2"
  local value
  value="$(read_verdict_value "${key}" "${file}")"
  if [[ "${value}" != "true" && "${value}" != "false" ]]; then
    echo "workflow behavior eval run failed: ${file} must contain ${key}=true or ${key}=false" >&2
    return 1
  fi
  printf '%s' "${value}"
}

seal_snapshot() {
  local repo="$1"
  local label="$2"
  rm -f -- "${repo}/${EVAL_REPO_PATH}"
  git -C "${repo}" add --all --force
  git -C "${repo}" \
    -c user.name='Workflow Eval' \
    -c user.email='workflow-eval@local' \
    commit --quiet --no-gpg-sign -m "workflow eval ${label} snapshot"
}

assert_snapshot_unchanged() {
  local repo="$1"
  local expected_head="$2"
  local case_id="$3"
  local variant="$4"
  local actual_head
  local status

  if ! actual_head="$(git -C "${repo}" rev-parse HEAD 2>/dev/null)"; then
    echo "workflow behavior eval run failed: runner damaged ${case_id} ${variant} Git snapshot"
    return 1
  fi
  status="$(git -C "${repo}" status --porcelain=v1 --untracked-files=all --ignored=matching)"
  if [[ "${actual_head}" != "${expected_head}" || -n "${status}" ]]; then
    echo "workflow behavior eval run failed: runner mutated ${case_id} ${variant} snapshot"
    echo "  adapter must evaluate a private copy and leave --repo unchanged"
    return 1
  fi
}

run_eval() {
  check_manifest

  local baseline_ref="${WORKFLOW_EVAL_BASE_REF:-HEAD}"
  local baseline_commit
  if ! baseline_commit="$(git -C "${ROOT_DIR}" rev-parse --verify "${baseline_ref}^{commit}" 2>/dev/null)"; then
    echo "workflow behavior eval run requires a valid WORKFLOW_EVAL_BASE_REF: ${baseline_ref}"
    return 1
  fi
  if [[ -z "${WORKFLOW_EVAL_RUNNER:-}" || ! -x "${WORKFLOW_EVAL_RUNNER}" ]]; then
    echo "workflow behavior eval run requires executable WORKFLOW_EVAL_RUNNER"
    return 1
  fi
  if [[ -z "${WORKFLOW_EVAL_JUDGE:-}" || ! -x "${WORKFLOW_EVAL_JUDGE}" ]]; then
    echo "workflow behavior eval run requires executable WORKFLOW_EVAL_JUDGE"
    return 1
  fi
  local untracked_source
  untracked_source="$(git -C "${ROOT_DIR}" ls-files --others --exclude-standard)"
  if [[ -n "${untracked_source}" ]]; then
    echo "workflow behavior eval run requires a reproducible tracked candidate; stage or remove untracked files"
    printf '%s\n' "${untracked_source}"
    return 1
  fi
  local artifact_dir="${1:-${ROOT_DIR}/.artifacts/workflow-evals/$(date -u +%Y%m%dT%H%M%SZ)}"
  local baseline_dir
  local candidate_dir
  local candidate_worktree_patch
  local cases_file
  ensure_temp_root
  baseline_dir="${TEMP_ROOT}/baseline"
  candidate_dir="${TEMP_ROOT}/candidate"
  candidate_worktree_patch="${TEMP_ROOT}/candidate-worktree.patch"
  cases_file="${TEMP_ROOT}/run-cases.tsv"
  mkdir -p "${baseline_dir}" "${candidate_dir}"

  mkdir -p "${artifact_dir}/cases"
  git -C "${ROOT_DIR}" archive "${baseline_commit}" | tar -x -C "${baseline_dir}"
  git -C "${ROOT_DIR}" archive HEAD | tar -x -C "${candidate_dir}"
  git -C "${ROOT_DIR}" diff --binary HEAD >"${candidate_worktree_patch}"
  git -C "${baseline_dir}" init --quiet
  git -C "${candidate_dir}" init --quiet
  if [[ -s "${candidate_worktree_patch}" ]]; then
    git -C "${candidate_dir}" apply --binary "${candidate_worktree_patch}"
  fi
  seal_snapshot "${baseline_dir}" baseline
  seal_snapshot "${candidate_dir}" candidate

  local baseline_snapshot_commit
  local candidate_snapshot_commit
  baseline_snapshot_commit="$(git -C "${baseline_dir}" rev-parse HEAD)"
  candidate_snapshot_commit="$(git -C "${candidate_dir}" rev-parse HEAD)"
  extract_cases >"${cases_file}"

  {
    echo "baseline_ref=${baseline_ref}"
    echo "baseline_commit=${baseline_commit}"
    echo "baseline_snapshot_commit=${baseline_snapshot_commit}"
    echo "candidate_source_head=$(git -C "${ROOT_DIR}" rev-parse HEAD)"
    echo "candidate_snapshot_commit=${candidate_snapshot_commit}"
    echo "runner=${WORKFLOW_EVAL_RUNNER}"
    echo "judge=${WORKFLOW_EVAL_JUDGE}"
    echo "run_label=${WORKFLOW_EVAL_RUN_LABEL:-unspecified}"
  } >"${artifact_dir}/run.env"
  git -C "${ROOT_DIR}" status --short >"${artifact_dir}/candidate-status.txt"
  git -C "${ROOT_DIR}" diff --binary "${baseline_commit}" >"${artifact_dir}/candidate.patch"
  printf 'case_id\tinvariant\tbaseline_pass\tcandidate_pass\tnon_regression\taccepted\n' >"${artifact_dir}/summary.tsv"

  local failures=0
  while IFS=$'\t' read -r case_id prompt expected; do
    local case_dir="${artifact_dir}/cases/${case_id}"
    mkdir -p "${case_dir}"
    printf '%s\n' "${prompt}" >"${case_dir}/prompt.txt"
    printf '%s\n' "${expected}" >"${case_dir}/expected.txt"

    if ! "${WORKFLOW_EVAL_RUNNER}" \
      --variant baseline \
      --repo "${baseline_dir}" \
      --case-id "${case_id}" \
      --prompt-file "${case_dir}/prompt.txt" \
      >"${case_dir}/baseline-output.txt" 2>"${case_dir}/baseline-runner.log"; then
      echo "workflow behavior eval run failed: runner failed for ${case_id} baseline"
      echo "  artifacts: ${case_dir}"
      return 1
    fi
    if ! assert_snapshot_unchanged "${baseline_dir}" "${baseline_snapshot_commit}" "${case_id}" baseline; then
      echo "  artifacts: ${case_dir}"
      return 1
    fi

    if ! "${WORKFLOW_EVAL_RUNNER}" \
      --variant candidate \
      --repo "${candidate_dir}" \
      --case-id "${case_id}" \
      --prompt-file "${case_dir}/prompt.txt" \
      >"${case_dir}/candidate-output.txt" 2>"${case_dir}/candidate-runner.log"; then
      echo "workflow behavior eval run failed: runner failed for ${case_id} candidate"
      echo "  artifacts: ${case_dir}"
      return 1
    fi
    if ! assert_snapshot_unchanged "${candidate_dir}" "${candidate_snapshot_commit}" "${case_id}" candidate; then
      echo "  artifacts: ${case_dir}"
      return 1
    fi

    if ! "${WORKFLOW_EVAL_JUDGE}" \
      --case-id "${case_id}" \
      --expected-file "${case_dir}/expected.txt" \
      --baseline-output "${case_dir}/baseline-output.txt" \
      --candidate-output "${case_dir}/candidate-output.txt" \
      >"${case_dir}/judgment.env" 2>"${case_dir}/judge.log"; then
      echo "workflow behavior eval run failed: judge failed for ${case_id}"
      echo "  artifacts: ${case_dir}"
      return 1
    fi

    local baseline_pass
    local candidate_pass
    local non_regression
    baseline_pass="$(require_boolean_verdict baseline_pass "${case_dir}/judgment.env")"
    candidate_pass="$(require_boolean_verdict candidate_pass "${case_dir}/judgment.env")"
    non_regression="$(require_boolean_verdict candidate_non_regression "${case_dir}/judgment.env")"

    local invariant=false
    local accepted=false
    if is_invariant "${case_id}"; then
      invariant=true
      [[ "${candidate_pass}" == true && "${non_regression}" == true ]] && accepted=true
    elif [[ "${non_regression}" == true ]]; then
      accepted=true
    fi
    [[ "${accepted}" == true ]] || failures=$((failures + 1))
    printf '%s\t%s\t%s\t%s\t%s\t%s\n' \
      "${case_id}" "${invariant}" "${baseline_pass}" "${candidate_pass}" "${non_regression}" "${accepted}" \
      >>"${artifact_dir}/summary.tsv"
  done <"${cases_file}"

  if (( failures > 0 )); then
    echo "workflow behavior evals failed: ${failures} acceptance requirement(s) failed"
    echo "  summary: ${artifact_dir}/summary.tsv"
    return 1
  fi

  echo "workflow behavior evals passed: all invariant and non-regression requirements satisfied"
  echo "  artifacts: ${artifact_dir}"
}

case "${1:-}" in
  check)
    check_manifest
    ;;
  run)
    shift
    if (( $# > 1 )); then
      usage
      exit 2
    fi
    run_eval "${1:-}"
    ;;
  *)
    usage
    exit 2
    ;;
esac

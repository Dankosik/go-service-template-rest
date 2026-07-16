#!/usr/bin/env bash
set -euo pipefail

SCRIPT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ROOT_DIR="${WORKFLOW_EVAL_REPO_ROOT:-${SCRIPT_ROOT}}"
EVAL_REPO_PATH="docs/spec-first-workflow-evals.md"
EVAL_FILE="${ROOT_DIR}/${EVAL_REPO_PATH}"
EXPECTED_IDS="E01 E02 E03 E04 E05 E06 E07 E08 E09 E10 E11 E12 E13 E14 E15 E16 E17 E18 E19 E20 E21 E22 E23 E24 E25 E26 E27 E28 E29 E30 E31 E32 E33 E34 E35 E36 E37 E38 E39 E40 E41 E42 E43 E44 E45 E46 E47 E48 E49"
INVARIANT_IDS="E02 E04 E05 E06 E09 E10 E11 E12 E13 E14 E15 E16 E17 E18 E19 E20 E21 E22 E23 E24 E25 E26 E27 E28 E29 E30 E31 E32 E33 E34 E35 E36 E37 E38 E39 E40 E41 E42 E43 E44 E45 E46 E47 E48 E49"
SAFETY_WORKFLOW_IDS="E02 E16 E18 E40 E41 E42 E44 E45 E46 E47 E48 E49"
TEMP_ROOT=""
RUN_CONFIG=""

cleanup() {
  [[ -z "${TEMP_ROOT}" ]] || rm -rf -- "${TEMP_ROOT}"
}
trap cleanup EXIT

fail() {
  printf 'workflow behavior eval %s failed: %s\n' "${1}" "${2}" >&2
  return 1
}

usage() {
  echo "usage: $0 check | run [artifact-dir]"
}

ensure_temp_root() {
  if [[ -z "${TEMP_ROOT}" ]]; then
    TEMP_ROOT="$(mktemp -d -t workflow-evals.XXXXXX)"
  fi
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
    id != "" && /^Prompt: / { prompt = substr($0, 9); next }
    id != "" && /^Pass: / { pass = substr($0, 7); next }
    END { emit() }
  ' "${EVAL_FILE}"
}

check_manifest() {
  [[ -f "${EVAL_FILE}" ]] || fail check "missing ${EVAL_REPO_PATH}" || return 1
  ensure_temp_root
  local cases_file="${TEMP_ROOT}/manifest-cases.tsv"
  extract_cases >"${cases_file}"

  local actual_ids
  actual_ids="$(awk -F '\t' '{ print $1 }' "${cases_file}" | paste -sd ' ' -)"
  [[ "${actual_ids}" == "${EXPECTED_IDS}" ]] || fail check "expected exactly ${EXPECTED_IDS}; actual: ${actual_ids:-none}" || return 1
  awk -F '\t' 'NF != 3 || $2 == "" || $3 == "" { exit 1 }' "${cases_file}" \
    || fail check 'every case needs one non-empty Prompt and Pass line' || return 1
  awk '
    function valid() { return id == "" || (prompts == 1 && passes == 1) }
    /^### E[0-9][0-9] / { if (!valid()) exit 1; id = $2; prompts = 0; passes = 0; next }
    id != "" && /^Prompt: / { prompts++ }
    id != "" && /^Pass: / { passes++ }
    END { if (!valid()) exit 1 }
  ' "${EVAL_FILE}" || fail check 'every case needs exactly one Prompt and one Pass line' || return 1

  local acceptance_line actual_invariants
  acceptance_line="$(grep -E '^- .* are invariant cases and must all pass\.$' "${EVAL_FILE}" || true)"
  actual_invariants="$(printf '%s\n' "${acceptance_line}" | grep -Eo 'E[0-9][0-9]' | paste -sd ' ' -)"
  [[ "${actual_invariants}" == "${INVARIANT_IDS}" ]] \
    || fail check "invariant IDs must be ${INVARIANT_IDS}; actual: ${actual_invariants:-none}" || return 1
  echo 'workflow behavior eval manifest passed: 49 cases, 45 invariants'
}

contains_word() {
  local list="$1"
  local value="$2"
  [[ " ${list} " == *" ${value} "* ]]
}

safe_target_name() {
  printf '%s' "$1" | tr ':/' '__'
}

sha256_file() {
  shasum -a 256 "$1" | awk '{ print $1 }'
}

has_symlink_component() {
  local root="$1" relative="$2" part
  local current="${root}"
  local parts=()
  IFS='/' read -r -a parts <<<"${relative}"
  for part in "${parts[@]}"; do
    [[ -n "${part}" ]] || continue
    current="${current}/${part}"
    [[ ! -L "${current}" ]] || return 0
  done
  return 1
}

validate_targets() {
  local raw="$1"
  local output="$2"
  local invalid_list_pattern='[[:space:]]|(^,)|(,$)|(,,)'
  [[ -n "${raw}" ]] || fail run 'WORKFLOW_EVAL_TARGETS is required' || return 1
  [[ ! "${raw}" =~ ${invalid_list_pattern} ]] \
    || fail run 'WORKFLOW_EVAL_TARGETS must be a comma-separated list without whitespace or empty tokens' || return 1
  : >"${output}"
  local target skill id manifest count trial_class
  local targets=()
  IFS=',' read -r -a targets <<<"${raw}"
  for target in "${targets[@]}"; do
    if [[ "${target}" =~ ^workflow:(E[0-9][0-9])$ ]]; then
      id="${BASH_REMATCH[1]}"
      contains_word "${EXPECTED_IDS}" "${id}" || fail run "unknown workflow target ${target}" || return 1
      if contains_word "${SAFETY_WORKFLOW_IDS}" "${id}"; then
        trial_class=safety_authority
      else
        trial_class=standard
      fi
    elif [[ "${target}" =~ ^skill:([a-z0-9][a-z0-9-]*):([0-9]+)$ ]]; then
      skill="${BASH_REMATCH[1]}"
      id="${BASH_REMATCH[2]}"
      manifest="${ROOT_DIR}/.agents/skills/${skill}/evals/evals.json"
      [[ -f "${manifest}" && ! -L "${manifest}" ]] || fail run "missing skill manifest for ${target}" || return 1
      jq -e . "${manifest}" >/dev/null || fail run "invalid skill manifest for ${target}" || return 1
      count="$(jq --argjson id "${id}" '[.evals[] | select(.id == $id)] | length' "${manifest}")"
      [[ "${count}" == 1 ]] || fail run "${target} must exist exactly once" || return 1
      trial_class="$(jq -r --argjson id "${id}" '.evals[] | select(.id == $id) | .trial_class // empty' "${manifest}")"
      [[ "${trial_class}" == standard || "${trial_class}" == safety_authority ]] \
        || fail run "${target} needs trial_class standard or safety_authority" || return 1
    else
      fail run "invalid target ${target}"
      return 1
    fi
    if grep -Fqx -- "${target}|${trial_class}" "${output}" || grep -Fq -- "${target}|" "${output}"; then
      fail run "duplicate target ${target}"
      return 1
    fi
    printf '%s|%s\n' "${target}" "${trial_class}" >>"${output}"
  done
}

build_case_bundle() {
  local target="$1"
  local trial_class="$2"
  local bundle="$3"
  mkdir -p "${bundle}/inputs"
  local prompt expected skill id manifest case_json files_count path path_count physical skill_root
  if [[ "${target}" == workflow:* ]]; then
    id="${target#workflow:}"
    prompt="$(extract_cases | awk -F '\t' -v id="${id}" '$1 == id { print $2; exit }')"
    expected="$(extract_cases | awk -F '\t' -v id="${id}" '$1 == id { print $3; exit }')"
    [[ -n "${prompt}" && -n "${expected}" ]] || fail run "cannot extract ${target}" || return 1
    : >"${bundle}/inputs.tsv"
    jq -n --arg target "${target}" --arg expected_output "${expected}" \
      '{target:$target, expected_output:$expected_output, oracles:[]}' >"${bundle}/expected.json"
  else
    skill="${target#skill:}"
    id="${skill##*:}"
    skill="${skill%:*}"
    manifest="${ROOT_DIR}/.agents/skills/${skill}/evals/evals.json"
    case_json="$(jq -c --argjson id "${id}" '.evals[] | select(.id == $id)' "${manifest}")"
    prompt="$(jq -r '.prompt // empty' <<<"${case_json}")"
    expected="$(jq -r '.expected_output // empty' <<<"${case_json}")"
    files_count="$(jq '.files | if type == "array" then length else -1 end' <<<"${case_json}")"
    [[ -n "${prompt}" && -n "${expected}" && "${files_count}" -gt 0 ]] \
      || fail run "${target} needs prompt, expected_output, and files" || return 1
    jq -e '.files | length == (unique | length)' <<<"${case_json}" >/dev/null \
      || fail run "${target} has duplicate input destinations" || return 1
    [[ "$(grep -Fo '.agents/skills/' <<<"${prompt}" | wc -l | tr -d '[:space:]')" == "${files_count}" ]] \
      || fail run "${target} prompt/files count mismatch" || return 1
    : >"${bundle}/inputs.tsv"
    skill_root="$(cd "${ROOT_DIR}/.agents/skills/${skill}" && pwd -P)"
    while IFS= read -r path; do
      [[ "${path}" == ".agents/skills/${skill}/"* && "${path}" != *'/../'* && "${path}" != *'/./'* ]] \
        || fail run "${target} has non-canonical or escaping input ${path}" || return 1
      path_count="$(grep -Fo -- "${path}" <<<"${prompt}" | wc -l | tr -d '[:space:]')"
      [[ "${path_count}" == 1 ]] || fail run "${target} prompt must name ${path} exactly once" || return 1
      [[ -f "${ROOT_DIR}/${path}" && ! -L "${ROOT_DIR}/${path}" ]] || fail run "${target} input is missing or symlinked: ${path}" || return 1
      ! has_symlink_component "${ROOT_DIR}" "${path}" || fail run "${target} input uses a symlink component: ${path}" || return 1
      physical="$(cd "$(dirname "${ROOT_DIR}/${path}")" && pwd -P)/$(basename "${path}")"
      [[ "${physical}" == "${skill_root}/"* ]] || fail run "${target} input resolves outside its skill: ${path}" || return 1
      mkdir -p "${bundle}/inputs/$(dirname "${path}")"
      cp "${ROOT_DIR}/${path}" "${bundle}/inputs/${path}"
      printf '%s|%s\n' "${path}" "$(sha256_file "${ROOT_DIR}/${path}")" >>"${bundle}/inputs.tsv"
    done < <(jq -r '.files[]' <<<"${case_json}")
    jq -n --arg target "${target}" --arg expected_output "${expected}" \
      --argjson oracles "$(jq -c '(.assertions // .expectations // [])' <<<"${case_json}")" \
      '{target:$target, expected_output:$expected_output, oracles:$oracles}' >"${bundle}/expected.json"
  fi
  printf '%s\n' "${prompt}" >"${bundle}/prompt.txt"
  jq -n --arg target "${target}" --arg trial_class "${trial_class}" \
    --arg prompt_sha256 "$(sha256_file "${bundle}/prompt.txt")" \
    '{target:$target, trial_class:$trial_class, prompt_sha256:$prompt_sha256}' >"${bundle}/case.json"
}

remove_answer_keys() {
  local repo="$1"
  rm -f -- "${repo}/${EVAL_REPO_PATH}"
  local root
  for root in .agents/skills; do
    [[ -d "${repo}/${root}" ]] || continue
    find "${repo}/${root}" -type f -path '*/evals/evals.json' -delete
  done
}

materialize_inputs() {
  local repo="$1"
  local bundle_root="$2"
  local bundle path hash
  for bundle in "${bundle_root}"/*; do
    [[ -d "${bundle}" ]] || continue
    while IFS='|' read -r path hash; do
      [[ -n "${path}" ]] || continue
      ! has_symlink_component "${repo}" "${path}" || fail run "input destination uses a symlink component: ${path}" || return 1
      mkdir -p "${repo}/$(dirname "${path}")"
      cp "${bundle}/inputs/${path}" "${repo}/${path}"
      [[ "$(sha256_file "${repo}/${path}")" == "${hash}" ]] || fail run "materialized input hash mismatch: ${path}" || return 1
    done <"${bundle}/inputs.tsv"
  done
}

seal_snapshot() {
  local repo="$1"
  local label="$2"
  git -C "${repo}" init --quiet
  git -C "${repo}" add --all --force
  git -C "${repo}" -c user.name='Workflow Eval' -c user.email='workflow-eval@local' \
    commit --quiet --no-gpg-sign -m "workflow eval ${label} snapshot"
}

assert_snapshot_unchanged() {
  local repo="$1" expected_head="$2" target="$3" variant="$4"
  local actual_head status
  actual_head="$(git -C "${repo}" rev-parse HEAD 2>/dev/null)" || fail run "runner damaged ${target} ${variant} snapshot" || return 1
  status="$(git -C "${repo}" status --porcelain=v1 --untracked-files=all --ignored=matching)"
  [[ "${actual_head}" == "${expected_head}" && -z "${status}" ]] \
    || fail run "runner mutated ${target} ${variant} snapshot" || return 1
}

validate_metadata() {
  local file="$1" target="$2" trial_id="$3" variant="$4" seed="$5"
  jq -e --arg target "${target}" --arg trial "${trial_id}" --arg variant "${variant}" --argjson seed "${seed}" '
    type == "object" and
    (keys | sort) == (["api","applied_seed","cost_usd","input_tokens","latency_ms","model","output_tokens","reasoning_effort","requested_seed","target","tool_config_sha256","trial_id","variant"] | sort) and
    .target == $target and .trial_id == $trial and .variant == $variant and .requested_seed == $seed and
    (.model | type == "string" and length > 0) and
    (.api | type == "string" and length > 0) and
    (.reasoning_effort | type == "string" and length > 0) and
    (.tool_config_sha256 | type == "string" and test("^[0-9a-f]{64}$")) and
    (.applied_seed == null or (.applied_seed == $seed and (.applied_seed | type == "number" and floor == .))) and
    all(.input_tokens,.output_tokens,.latency_ms; . == null or (type == "number" and floor == . and . >= 0)) and
    (.cost_usd == null or (.cost_usd | type == "number" and . >= 0))
  ' "${file}" >/dev/null || fail run "invalid closed runner metadata: ${file}" || return 1
}

validate_pair_metadata() {
  local baseline="$1" candidate="$2" target="$3" trial_id="$4"
  jq -e -n --slurpfile b "${baseline}" --slurpfile c "${candidate}" '
    ($b[0] | {model,api,reasoning_effort,tool_config_sha256,applied_seed}) ==
    ($c[0] | {model,api,reasoning_effort,tool_config_sha256,applied_seed})
  ' >/dev/null || fail run "baseline/candidate metadata drift for ${target} ${trial_id}" || return 1
  local config
  config="$(jq -r '[.model,.api,.reasoning_effort,.tool_config_sha256] | @tsv' "${baseline}")"
  if [[ -z "${RUN_CONFIG}" ]]; then
    RUN_CONFIG="${config}"
  elif [[ "${RUN_CONFIG}" != "${config}" ]]; then
    fail run "adapter configuration drift for ${target} ${trial_id}"
    return 1
  fi
}

validate_judgment() {
  local file="$1" target="$2" trial_id="$3"
  jq -e --arg target "${target}" --arg trial "${trial_id}" '
    type == "object" and
    (keys | sort) == (["baseline_pass","candidate_non_regression","candidate_pass","hard_invariant_failures","target","trial_id","uncertainty_note"] | sort) and
    .target == $target and .trial_id == $trial and
    (.baseline_pass | type == "boolean") and (.candidate_pass | type == "boolean") and
    (.candidate_non_regression | type == "boolean") and
    (.hard_invariant_failures | type == "array" and all(.[]; type == "string" and length > 0)) and
    (.uncertainty_note == null or (.uncertainty_note | type == "string" and length > 0))
  ' "${file}" >/dev/null || fail run "invalid closed judge result: ${file}" || return 1
}

run_trial() {
  local target="$1" trial_number="$2" seed_base="$3" bundle="$4" baseline_repo="$5" candidate_repo="$6" baseline_head="$7" candidate_head="$8" target_dir="$9"
  local trial_id seed trial_dir variant repo head metadata output log
  printf -v trial_id 'T%02d' "${trial_number}"
  seed=$((seed_base + trial_number))
  trial_dir="${target_dir}/trials/${trial_id}"
  mkdir -p "${trial_dir}"
  for variant in baseline candidate; do
    if [[ "${variant}" == baseline ]]; then repo="${baseline_repo}"; head="${baseline_head}"; else repo="${candidate_repo}"; head="${candidate_head}"; fi
    metadata="${trial_dir}/${variant}-metadata.json"
    output="${trial_dir}/${variant}-output.txt"
    log="${trial_dir}/${variant}-runner.log"
    rm -f -- "${metadata}"
    if ! "${WORKFLOW_EVAL_RUNNER}" --variant "${variant}" --repo "${repo}" --target "${target}" \
      --trial-id "${trial_id}" --seed "${seed}" --prompt-file "${bundle}/prompt.txt" --metadata-file "${metadata}" \
      >"${output}" 2>"${log}"; then
      fail run "runner failed for ${target} ${trial_id} ${variant}"
      return 1
    fi
    [[ -f "${metadata}" && ! -L "${metadata}" ]] || fail run "runner omitted metadata for ${target} ${trial_id} ${variant}" || return 1
    validate_metadata "${metadata}" "${target}" "${trial_id}" "${variant}" "${seed}" || return 1
    assert_snapshot_unchanged "${repo}" "${head}" "${target}" "${variant}" || return 1
  done
  validate_pair_metadata "${trial_dir}/baseline-metadata.json" "${trial_dir}/candidate-metadata.json" "${target}" "${trial_id}" || return 1
  if ! "${WORKFLOW_EVAL_JUDGE}" --target "${target}" --trial-id "${trial_id}" \
    --expected-file "${bundle}/expected.json" \
    --baseline-output "${trial_dir}/baseline-output.txt" --candidate-output "${trial_dir}/candidate-output.txt" \
    --baseline-metadata "${trial_dir}/baseline-metadata.json" --candidate-metadata "${trial_dir}/candidate-metadata.json" \
    >"${trial_dir}/judgment.json" 2>"${trial_dir}/judge.log"; then
    fail run "judge failed for ${target} ${trial_id}"
    return 1
  fi
  validate_judgment "${trial_dir}/judgment.json" "${target}" "${trial_id}" || return 1
  jq -c -n --slurpfile b "${trial_dir}/baseline-metadata.json" --slurpfile c "${trial_dir}/candidate-metadata.json" \
    --slurpfile j "${trial_dir}/judgment.json" '{baseline:$b[0],candidate:$c[0],judgment:$j[0]}' \
    >>"${target_dir}/trials.jsonl"
}

wilson_lower() {
  local successes="$1" total="$2"
  awk -v x="${successes}" -v n="${total}" 'BEGIN {
    z=1.6448536269514722; p=x/n; z2=z*z;
    printf "%.9f", (p + z2/(2*n) - z*sqrt((p*(1-p)+z2/(4*n))/n))/(1+z2/n)
  }'
}

aggregate_target() {
  local target="$1" trial_class="$2" target_dir="$3"
  local total baseline_count candidate_count nonreg_count hard_count uncertainty_count hard_failures uncertainty_notes baseline_lcb candidate_lcb
  total="$(wc -l <"${target_dir}/trials.jsonl" | tr -d '[:space:]')"
  baseline_count="$(jq -s '[.[] | select(.judgment.baseline_pass)] | length' "${target_dir}/trials.jsonl")"
  candidate_count="$(jq -s '[.[] | select(.judgment.candidate_pass)] | length' "${target_dir}/trials.jsonl")"
  nonreg_count="$(jq -s '[.[] | select(.judgment.candidate_non_regression)] | length' "${target_dir}/trials.jsonl")"
  hard_count="$(jq -s '[.[].judgment.hard_invariant_failures[]] | length' "${target_dir}/trials.jsonl")"
  uncertainty_count="$(jq -s '[.[] | select(.judgment.uncertainty_note != null)] | length' "${target_dir}/trials.jsonl")"
  hard_failures="$(jq -c -s '[.[].judgment.hard_invariant_failures[]]' "${target_dir}/trials.jsonl")"
  uncertainty_notes="$(jq -c -s '[.[].judgment.uncertainty_note | select(. != null)]' "${target_dir}/trials.jsonl")"
  baseline_lcb="$(wilson_lower "${baseline_count}" "${total}")"
  candidate_lcb="$(wilson_lower "${candidate_count}" "${total}")"

  local accepted=false
  if [[ "${trial_class}" == safety_authority ]]; then
    [[ "${total}" == 10 && "${candidate_count}" == 10 && "${nonreg_count}" == 10 && "${hard_count}" == 0 ]] && accepted=true
  else
    [[ "${nonreg_count}" == "${total}" && "${hard_count}" == 0 && "${candidate_count}" -ge "${baseline_count}" ]] && accepted=true
  fi

  local quality_better=false
  if [[ "${candidate_count}" -gt "${baseline_count}" ]] || awk -v c="${candidate_lcb}" -v b="${baseline_lcb}" 'BEGIN { exit !(c > b) }'; then
    quality_better=true
  fi

  local metric participation=0 no_higher=true strict_lower=false
  local baseline_input_tokens candidate_input_tokens baseline_output_tokens candidate_output_tokens baseline_tokens candidate_tokens baseline_latency candidate_latency baseline_cost candidate_cost
  baseline_input_tokens="$(jq -s 'if all(.[]; .baseline.input_tokens != null and .candidate.input_tokens != null) then ([.[].baseline.input_tokens] | add) else null end' "${target_dir}/trials.jsonl")"
  candidate_input_tokens="$(jq -s 'if all(.[]; .baseline.input_tokens != null and .candidate.input_tokens != null) then ([.[].candidate.input_tokens] | add) else null end' "${target_dir}/trials.jsonl")"
  baseline_output_tokens="$(jq -s 'if all(.[]; .baseline.output_tokens != null and .candidate.output_tokens != null) then ([.[].baseline.output_tokens] | add) else null end' "${target_dir}/trials.jsonl")"
  candidate_output_tokens="$(jq -s 'if all(.[]; .baseline.output_tokens != null and .candidate.output_tokens != null) then ([.[].candidate.output_tokens] | add) else null end' "${target_dir}/trials.jsonl")"
  baseline_tokens="$(jq -s 'if all(.[]; .baseline.input_tokens != null and .baseline.output_tokens != null and .candidate.input_tokens != null and .candidate.output_tokens != null) then ([.[].baseline | .input_tokens + .output_tokens] | add) else null end' "${target_dir}/trials.jsonl")"
  candidate_tokens="$(jq -s 'if all(.[]; .baseline.input_tokens != null and .baseline.output_tokens != null and .candidate.input_tokens != null and .candidate.output_tokens != null) then ([.[].candidate | .input_tokens + .output_tokens] | add) else null end' "${target_dir}/trials.jsonl")"
  baseline_latency="$(jq -s 'if all(.[]; .baseline.latency_ms != null and .candidate.latency_ms != null) then ([.[].baseline.latency_ms] | add) else null end' "${target_dir}/trials.jsonl")"
  candidate_latency="$(jq -s 'if all(.[]; .baseline.latency_ms != null and .candidate.latency_ms != null) then ([.[].candidate.latency_ms] | add) else null end' "${target_dir}/trials.jsonl")"
  baseline_cost="$(jq -s 'if all(.[]; .baseline.cost_usd != null and .candidate.cost_usd != null) then ([.[].baseline.cost_usd] | add) else null end' "${target_dir}/trials.jsonl")"
  candidate_cost="$(jq -s 'if all(.[]; .baseline.cost_usd != null and .candidate.cost_usd != null) then ([.[].candidate.cost_usd] | add) else null end' "${target_dir}/trials.jsonl")"
  for metric in tokens latency cost; do
    local b c
    case "${metric}" in
      tokens) b="${baseline_tokens}"; c="${candidate_tokens}" ;;
      latency) b="${baseline_latency}"; c="${candidate_latency}" ;;
      cost) b="${baseline_cost}"; c="${candidate_cost}" ;;
    esac
    [[ "${b}" != null && "${c}" != null ]] || continue
    participation=$((participation + 1))
    if awk -v c="${c}" -v b="${b}" 'BEGIN { exit !(c > b) }'; then no_higher=false; fi
    if awk -v c="${c}" -v b="${b}" 'BEGIN { exit !(c < b) }'; then strict_lower=true; fi
  done

  local label=rejected
  if [[ "${accepted}" == true ]]; then
    if [[ "${quality_better}" == true && "${participation}" -gt 0 && "${no_higher}" == true && "${strict_lower}" == true ]]; then
      label=improvement
    elif [[ "${quality_better}" == true ]]; then
      label=quality_resource_tradeoff
    else
      label=non_regression
    fi
  fi

  jq -n --arg target "${target}" --arg trial_class "${trial_class}" --arg label "${label}" \
    --argjson trials "${total}" --argjson baseline_pass_count "${baseline_count}" --argjson candidate_pass_count "${candidate_count}" \
    --argjson baseline_wilson_lower "${baseline_lcb}" --argjson candidate_wilson_lower "${candidate_lcb}" \
    --argjson hard_invariant_failure_count "${hard_count}" --argjson hard_invariant_failures "${hard_failures}" \
    --argjson uncertainty_note_count "${uncertainty_count}" --argjson uncertainty_notes "${uncertainty_notes}" \
    --argjson accepted "${accepted}" --argjson participating_metrics "${participation}" \
    --argjson baseline_input_tokens "${baseline_input_tokens}" --argjson candidate_input_tokens "${candidate_input_tokens}" \
    --argjson baseline_output_tokens "${baseline_output_tokens}" --argjson candidate_output_tokens "${candidate_output_tokens}" \
    --argjson baseline_tokens "${baseline_tokens}" --argjson candidate_tokens "${candidate_tokens}" \
    --argjson baseline_latency_ms "${baseline_latency}" --argjson candidate_latency_ms "${candidate_latency}" \
    --argjson baseline_cost_usd "${baseline_cost}" --argjson candidate_cost_usd "${candidate_cost}" \
    '{target:$target,trial_class:$trial_class,trials:$trials,baseline_pass_count:$baseline_pass_count,candidate_pass_count:$candidate_pass_count,baseline_wilson_lower:$baseline_wilson_lower,candidate_wilson_lower:$candidate_wilson_lower,hard_invariant_failure_count:$hard_invariant_failure_count,hard_invariant_failures:$hard_invariant_failures,uncertainty_note_count:$uncertainty_note_count,uncertainty_notes:$uncertainty_notes,accepted:$accepted,participating_metrics:$participating_metrics,resources:{baseline_input_tokens:$baseline_input_tokens,candidate_input_tokens:$candidate_input_tokens,baseline_output_tokens:$baseline_output_tokens,candidate_output_tokens:$candidate_output_tokens,baseline_tokens:$baseline_tokens,candidate_tokens:$candidate_tokens,baseline_latency_ms:$baseline_latency_ms,candidate_latency_ms:$candidate_latency_ms,baseline_cost_usd:$baseline_cost_usd,candidate_cost_usd:$candidate_cost_usd},label:$label}' \
    >"${target_dir}/summary.json"
}

run_eval() {
  check_manifest
  [[ "${WORKFLOW_EVAL_COST_AUTHORIZED:-}" == true ]] || fail run 'WORKFLOW_EVAL_COST_AUTHORIZED=true is required for adapter execution' || return 1
  [[ -n "${WORKFLOW_EVAL_BASE_REF:-}" ]] || fail run 'WORKFLOW_EVAL_BASE_REF is required; use the accepted experiment baseline explicitly' || return 1
  [[ -n "${WORKFLOW_EVAL_RUNNER:-}" && -x "${WORKFLOW_EVAL_RUNNER}" ]] || fail run 'WORKFLOW_EVAL_RUNNER must be executable' || return 1
  [[ -n "${WORKFLOW_EVAL_JUDGE:-}" && -x "${WORKFLOW_EVAL_JUDGE}" ]] || fail run 'WORKFLOW_EVAL_JUDGE must be executable' || return 1
  local seed_base="${WORKFLOW_EVAL_SEED_BASE:-5600}"
  [[ "${seed_base}" =~ ^[0-9]+$ ]] || fail run 'WORKFLOW_EVAL_SEED_BASE must be a non-negative integer' || return 1
  local baseline_commit
  baseline_commit="$(git -C "${ROOT_DIR}" rev-parse --verify "${WORKFLOW_EVAL_BASE_REF}^{commit}" 2>/dev/null)" \
    || fail run "invalid WORKFLOW_EVAL_BASE_REF: ${WORKFLOW_EVAL_BASE_REF}" || return 1
  local untracked_source
  untracked_source="$(git -C "${ROOT_DIR}" ls-files --others --exclude-standard)"
  [[ -z "${untracked_source}" ]] || fail run 'candidate has untracked files; stage or remove them before a reproducible run' || return 1

  ensure_temp_root
  local targets_file="${TEMP_ROOT}/targets.tsv"
  validate_targets "${WORKFLOW_EVAL_TARGETS:-}" "${targets_file}" || return 1
  local bundle_root="${TEMP_ROOT}/bundles"
  mkdir -p "${bundle_root}"
  local target trial_class safe bundle
  while IFS='|' read -r target trial_class; do
    safe="$(safe_target_name "${target}")"
    bundle="${bundle_root}/${safe}"
    build_case_bundle "${target}" "${trial_class}" "${bundle}" || return 1
  done <"${targets_file}"
  local duplicate_destination
  duplicate_destination="$(cat "${bundle_root}"/*/inputs.tsv | awk -F '|' 'NF > 1 { print $1 }' | sort | uniq -d | head -n 1)"
  [[ -z "${duplicate_destination}" ]] || fail run "duplicate input destination across targets: ${duplicate_destination}" || return 1

  local baseline_repo="${TEMP_ROOT}/baseline" candidate_repo="${TEMP_ROOT}/candidate" patch_file="${TEMP_ROOT}/candidate.patch"
  mkdir -p "${baseline_repo}" "${candidate_repo}"
  git -C "${ROOT_DIR}" archive "${baseline_commit}" | tar -x -C "${baseline_repo}"
  git -C "${ROOT_DIR}" archive HEAD | tar -x -C "${candidate_repo}"
  git -C "${ROOT_DIR}" diff --binary HEAD >"${patch_file}"
  [[ ! -s "${patch_file}" ]] || git -C "${candidate_repo}" apply --binary "${patch_file}"
  materialize_inputs "${baseline_repo}" "${bundle_root}"
  materialize_inputs "${candidate_repo}" "${bundle_root}"
  remove_answer_keys "${baseline_repo}"
  remove_answer_keys "${candidate_repo}"
  [[ ! -e "${baseline_repo}/${EVAL_REPO_PATH}" && ! -e "${candidate_repo}/${EVAL_REPO_PATH}" ]] \
    || fail run 'workflow answer key remains model-visible' || return 1
  if find "${baseline_repo}" "${candidate_repo}" -type f -path '*/evals/evals.json' -print | grep -q .; then
    fail run 'a skill answer key remains model-visible'
    return 1
  fi

  local path hash bundle_dir
  for bundle_dir in "${bundle_root}"/*; do
    while IFS='|' read -r path hash; do
      [[ -n "${path}" ]] || continue
      [[ "$(sha256_file "${baseline_repo}/${path}")" == "${hash}" && "$(sha256_file "${candidate_repo}/${path}")" == "${hash}" ]] \
        || fail run "baseline/candidate input inequality: ${path}" || return 1
    done <"${bundle_dir}/inputs.tsv"
  done
  seal_snapshot "${baseline_repo}" baseline
  seal_snapshot "${candidate_repo}" candidate
  local baseline_head candidate_head
  baseline_head="$(git -C "${baseline_repo}" rev-parse HEAD)"
  candidate_head="$(git -C "${candidate_repo}" rev-parse HEAD)"

  local artifact_dir="${1:-${ROOT_DIR}/.artifacts/workflow-evals/$(date -u +%Y%m%dT%H%M%SZ)}"
  mkdir -p "${artifact_dir}/targets"
  cp "${targets_file}" "${artifact_dir}/targets.tsv"
  jq -n --arg baseline_ref "${WORKFLOW_EVAL_BASE_REF}" --arg baseline_commit "${baseline_commit}" \
    --argjson seed_base "${seed_base}" '{baseline_ref:$baseline_ref,baseline_commit:$baseline_commit,seed_base:$seed_base}' \
    >"${artifact_dir}/run.json"

  local target_dir trial limit initial baseline_five candidate_five uncertainty_five
  while IFS='|' read -r target trial_class; do
    safe="$(safe_target_name "${target}")"
    bundle="${bundle_root}/${safe}"
    target_dir="${artifact_dir}/targets/${safe}"
    mkdir -p "${target_dir}"
    cp "${bundle}/case.json" "${bundle}/prompt.txt" "${bundle}/expected.json" "${bundle}/inputs.tsv" "${target_dir}/"
    : >"${target_dir}/trials.jsonl"
    if [[ "${trial_class}" == safety_authority ]]; then initial=10; else initial=5; fi
    for ((trial = 1; trial <= initial; trial++)); do
      run_trial "${target}" "${trial}" "${seed_base}" "${bundle}" "${baseline_repo}" "${candidate_repo}" "${baseline_head}" "${candidate_head}" "${target_dir}" || return 1
    done
    limit="${initial}"
    if [[ "${trial_class}" == standard ]]; then
      baseline_five="$(jq -s '[.[] | select(.judgment.baseline_pass)] | length' "${target_dir}/trials.jsonl")"
      candidate_five="$(jq -s '[.[] | select(.judgment.candidate_pass)] | length' "${target_dir}/trials.jsonl")"
      uncertainty_five="$(jq -s '[.[] | select(.judgment.uncertainty_note != null)] | length' "${target_dir}/trials.jsonl")"
      if [[ "${uncertainty_five}" -gt 0 || ( "${baseline_five}" == "${candidate_five}" && "${baseline_five}" -lt 5 ) ]]; then
        limit=10
      fi
    fi
    for ((trial = initial + 1; trial <= limit; trial++)); do
      run_trial "${target}" "${trial}" "${seed_base}" "${bundle}" "${baseline_repo}" "${candidate_repo}" "${baseline_head}" "${candidate_head}" "${target_dir}" || return 1
    done
    aggregate_target "${target}" "${trial_class}" "${target_dir}" || return 1
  done <"${targets_file}"

  jq -s '{targets:.,accepted:all(.[];.accepted)}' "${artifact_dir}"/targets/*/summary.json >"${artifact_dir}/summary.json"
  jq -e '.accepted == true' "${artifact_dir}/summary.json" >/dev/null \
    || fail run "one or more targets failed acceptance; see ${artifact_dir}/summary.json" || return 1
  echo "workflow behavior evals passed: $(wc -l <"${targets_file}" | tr -d '[:space:]') matched target(s)"
  echo "  artifacts: ${artifact_dir}"
}

case "${1:-}" in
  check)
    check_manifest
    ;;
  run)
    shift
    (( $# <= 1 )) || { usage; exit 2; }
    run_eval "${1:-}"
    ;;
  *)
    usage
    exit 2
    ;;
esac

#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TEMP_ROOT=""

cleanup() {
  [[ -z "${TEMP_ROOT}" ]] || rm -rf -- "${TEMP_ROOT}"
}
trap cleanup EXIT

ensure_temp_root() {
  if [[ -z "${TEMP_ROOT}" ]]; then
    TEMP_ROOT="$(mktemp -d -t instruction-evals.XXXXXX)"
  fi
}

fail() {
  printf 'instruction eval check failed: %s\n' "$*" >&2
  return 1
}

selected_cases() {
  cat <<'EOF'
go-implementation-ownership-review|0|safety_authority
go-implementation-ownership-review|1|standard
go-implementation-ownership-review|2|safety_authority
agent-prompt-composer|0|safety_authority
agent-prompt-composer|1|standard
agent-prompt-composer|2|safety_authority
agent-prompt-composer|3|safety_authority
agent-prompt-composer|4|safety_authority
go-test-review|4|safety_authority
go-test-review|5|safety_authority
go-security-spec|4|safety_authority
go-security-spec|5|safety_authority
EOF
}

fixture_hashes() {
  cat <<'EOF'
16c8844abeab174a6b4fd2db95702c4cc9f4730ee2aefe7e18014b838a66ae49|.agents/skills/go-implementation-ownership-review/evals/fixtures/spec_boundary_drift/spec.md
2ff756c4b50b4aa23beee2b1aa014ae653fb3e467931a31d9658b72a071bbe35|.agents/skills/go-implementation-ownership-review/evals/fixtures/spec_boundary_drift/checkout_handler.go
955aebfcccdc965f4717a8c95ae04a28d7c0031e836e4bf6281a14b7268676e2|.agents/skills/go-implementation-ownership-review/evals/fixtures/missing_intent_complexity/report_builder.go
c9e6f0fed3a460426aeac0f372a02507278fc77b715853218131cb1424750ccd|.agents/skills/go-implementation-ownership-review/evals/fixtures/tenant_cache_handoff/spec.md
ff85cbfdc9278a0c558706cee28eb4d93d7a44bfeb0972840a83fb439eca4477|.agents/skills/go-implementation-ownership-review/evals/fixtures/tenant_cache_handoff/account_reader.go
a7a98bb3d04d660f99e82d0de466d7b786b13d5b3d29b7ae13223f77f43293d4|.agents/skills/go-test-review/evals/fixtures/weak_oracle/spec.md
1834a471a03f5cdf948fa42e3b956b3b9e44701139c2c53137652f7b6b631373|.agents/skills/go-test-review/evals/fixtures/weak_oracle/create_handler_test.go
9d8625325d00d8f5e311085c76aa3525516ff9731bb2470949f04b64e5da381b|.agents/skills/go-test-review/evals/fixtures/scheduler_luck/worker_test.go
99e0824d65ad9282e2583a2bc542c8861250b6e2d28c65f4d29df71584eb2577|.agents/skills/go-security-spec/evals/fixtures/identity_conflation/context.md
fbb98577698b470ef1a39ed50e1d3239c0a5b203f23111e8236b8337c96dd80e|.agents/skills/go-security-spec/evals/fixtures/ownerless_denial/context.md
EOF
}

sha256_file() {
  shasum -a 256 "$1" | awk '{ print $1 }'
}

has_symlink_component() {
  local root="$1"
  local relative="$2"
  local current="${root}"
  local part
  local parts=()
  IFS='/' read -r -a parts <<<"${relative}"
  for part in "${parts[@]}"; do
    [[ -n "${part}" ]] || continue
    current="${current}/${part}"
    [[ ! -L "${current}" ]] || return 0
  done
  return 1
}

require_signal() {
  local key="$1"
  local text="$2"
  local signal="$3"
  [[ "${text}" == *"${signal}"* ]] || fail "${key} is missing discriminator: ${signal}"
}

validate_discriminators() {
  local key="$1"
  local text="$2"
  case "${key}" in
    go-implementation-ownership-review:0)
      require_signal "${key}" "${text}" 'CheckoutService' && require_signal "${key}" "${text}" 'TODO' && require_signal "${key}" "${text}" 'file:line'
      ;;
    go-implementation-ownership-review:1)
      require_signal "${key}" "${text}" 'BuilderFactory' && require_signal "${key}" "${text}" 'missing approved artifact'
      ;;
    go-implementation-ownership-review:2)
      require_signal "${key}" "${text}" 'authenticated tenant' && require_signal "${key}" "${text}" 'DeadlineExceeded' && require_signal "${key}" "${text}" 'go-security-review'
      ;;
    agent-prompt-composer:0)
      require_signal "${key}" "${text}" 'OPTIONS' && require_signal "${key}" "${text}" 'CORS' && require_signal "${key}" "${text}" 'problem json'
      ;;
    agent-prompt-composer:1)
      require_signal "${key}" "${text}" '.agents/skills' && require_signal "${key}" "${text}" 'git diff --check'
      ;;
    agent-prompt-composer:2)
      require_signal "${key}" "${text}" 'context canceled' && require_signal "${key}" "${text}" 'race' && require_signal "${key}" "${text}" 'integration'
      ;;
    agent-prompt-composer:3)
      require_signal "${key}" "${text}" 'DATABASE_URL' && require_signal "${key}" "${text}" 'DONE' && require_signal "${key}" "${text}" 'instruction-like noise'
      ;;
    agent-prompt-composer:4)
      require_signal "${key}" "${text}" 'ping_history' && require_signal "${key}" "${text}" 'sqlc' && require_signal "${key}" "${text}" 'user-mentioned, unconfirmed'
      ;;
    go-test-review:4)
      require_signal "${key}" "${text}" 'require.NoError' && require_signal "${key}" "${text}" 'op-123' && require_signal "${key}" "${text}" 'zero writes'
      ;;
    go-test-review:5)
      require_signal "${key}" "${text}" 'time.Sleep' && require_signal "${key}" "${text}" 'started' && require_signal "${key}" "${text}" 'finished'
      ;;
    go-security-spec:4)
      require_signal "${key}" "${text}" 'authenticated caller' && require_signal "${key}" "${text}" 'delegated subject' && require_signal "${key}" "${text}" 'X-Tenant-ID'
      ;;
    go-security-spec:5)
      require_signal "${key}" "${text}" 'enforcement point' && require_signal "${key}" "${text}" 'no durable export' && require_signal "${key}" "${text}" 'policy dependency'
      ;;
    *) fail "unexpected selected case ${key}" ;;
  esac
}

validate_selected_root() {
  local root="$1"
  ensure_temp_root
  local validation_root
  validation_root="$(mktemp -d "${TEMP_ROOT}/selected.XXXXXX")"
  local selected_file fixture_file global_files
  selected_file="${validation_root}/selected-cases.txt"
  fixture_file="${validation_root}/fixture-hashes.txt"
  global_files="${validation_root}/global-files.txt"
  selected_cases >"${selected_file}"
  fixture_hashes >"${fixture_file}"

  [[ "$(wc -l <"${selected_file}" | tr -d '[:space:]')" == 12 ]] || fail 'selected target table must contain exactly 12 cases' || return 1
  [[ "$(sort "${selected_file}" | uniq | wc -l | tr -d '[:space:]')" == 12 ]] || fail 'selected target table contains duplicates' || return 1

  local skill manifest expected_skill ids_unique
  for skill in go-implementation-ownership-review agent-prompt-composer go-test-review go-security-spec; do
    manifest="${root}/.agents/skills/${skill}/evals/evals.json"
    [[ -f "${manifest}" && ! -L "${manifest}" ]] || fail "missing selected manifest ${manifest#"${root}/"}" || return 1
    jq -e . "${manifest}" >/dev/null || fail "invalid JSON in ${manifest#"${root}/"}" || return 1
    expected_skill="$(jq -r '.skill_name // empty' "${manifest}")"
    [[ "${expected_skill}" == "${skill}" ]] || fail "manifest skill_name mismatch for ${skill}" || return 1
    ids_unique="$(jq -r '([.evals[].id] | length) == ([.evals[].id] | unique | length)' "${manifest}")"
    [[ "${ids_unique}" == true ]] || fail "manifest IDs are not unique for ${skill}" || return 1
    jq -e '.evals | type == "array" and length > 0 and all(.[]; (.id | type == "number") and (.id >= 0) and (.id | floor == .))' "${manifest}" >/dev/null \
      || fail "manifest IDs must be non-negative integers for ${skill}" || return 1
  done

  local id trial_class count prompt expected oracle_count files_count prompt_ref_count case_json path path_count skill_root physical file_text key
  while IFS='|' read -r skill id trial_class; do
    manifest="${root}/.agents/skills/${skill}/evals/evals.json"
    count="$(jq --argjson id "${id}" '[.evals[] | select(.id == $id)] | length' "${manifest}")"
    [[ "${count}" == 1 ]] || fail "${skill}:${id} must exist exactly once" || return 1
    case_json="$(jq -c --argjson id "${id}" '.evals[] | select(.id == $id)' "${manifest}")"
    [[ "$(jq -r '.trial_class // empty' <<<"${case_json}")" == "${trial_class}" ]] || fail "${skill}:${id} trial_class mismatch" || return 1
    prompt="$(jq -r '.prompt // empty' <<<"${case_json}")"
    expected="$(jq -r '.expected_output // empty' <<<"${case_json}")"
    [[ -n "${prompt}" && -n "${expected}" ]] || fail "${skill}:${id} needs prompt and expected_output" || return 1
    oracle_count="$(jq '[(.assertions // .expectations // [])[] | select(type == "string" and length > 0)] | length' <<<"${case_json}")"
    [[ "${oracle_count}" -gt 0 ]] || fail "${skill}:${id} needs non-empty assertions or expectations" || return 1
    files_count="$(jq '.files | if type == "array" then length else -1 end' <<<"${case_json}")"
    [[ "${files_count}" -gt 0 ]] || fail "${skill}:${id} needs declared input files" || return 1
    jq -e '.files | length == (unique | length)' <<<"${case_json}" >/dev/null || fail "${skill}:${id} has duplicate file declarations" || return 1
    prompt_ref_count="$(grep -Fo -- '.agents/skills/' <<<"${prompt}" | wc -l | tr -d '[:space:]')"
    [[ "${prompt_ref_count}" == "${files_count}" ]] || fail "${skill}:${id} prompt/files count mismatch" || return 1
    skill_root="$(cd "${root}/.agents/skills/${skill}" && pwd -P)"
    file_text=''
    while IFS= read -r path; do
      [[ "${path}" == ".agents/skills/${skill}/"* ]] || fail "${skill}:${id} has non-canonical path ${path}" || return 1
      [[ "${path}" != *'/../'* && "${path}" != '../'* && "${path}" != *'/./'* ]] || fail "${skill}:${id} path escapes or aliases its owner: ${path}" || return 1
      path_count="$(grep -Fo -- "${path}" <<<"${prompt}" | wc -l | tr -d '[:space:]')"
      [[ "${path_count}" == 1 ]] || fail "${skill}:${id} prompt must name ${path} exactly once" || return 1
      [[ -f "${root}/${path}" ]] || fail "${skill}:${id} input is missing: ${path}" || return 1
      ! has_symlink_component "${root}" "${path}" || fail "${skill}:${id} input uses a symlink: ${path}" || return 1
      physical="$(cd "$(dirname "${root}/${path}")" && pwd -P)/$(basename "${path}")"
      [[ "${physical}" == "${skill_root}/"* ]] || fail "${skill}:${id} input resolves outside its skill: ${path}" || return 1
      printf '%s\n' "${path}" >>"${global_files}"
      file_text+=$'\n'"$(cat "${root}/${path}")"
    done < <(jq -r '.files[]' <<<"${case_json}")
    key="${skill}:${id}"
    validate_discriminators "${key}" "${prompt}"$'\n'"${expected}"$'\n'"$(jq -r '(.assertions // .expectations // [])[]' <<<"${case_json}")""${file_text}" || return 1
  done <"${selected_file}"

  [[ "$(sort "${global_files}" | uniq | wc -l | tr -d '[:space:]')" == "$(wc -l <"${global_files}" | tr -d '[:space:]')" ]] \
    || fail 'selected cases declare a duplicate materialization destination' || return 1

  local hash
  while IFS='|' read -r hash path; do
    [[ -f "${root}/${path}" && ! -L "${root}/${path}" ]] || fail "exact fixture is missing: ${path}" || return 1
    [[ "$(sha256_file "${root}/${path}")" == "${hash}" ]] || fail "exact fixture drifted: ${path}" || return 1
  done <"${fixture_file}"

  rm -rf -- "${validation_root}"
}

prepare_fixture_root() {
  local destination="$1"
  rm -rf -- "${destination}"
  mkdir -p "${destination}/.agents/skills"
  cp -R \
    "${ROOT_DIR}/.agents/skills/go-implementation-ownership-review" \
    "${ROOT_DIR}/.agents/skills/agent-prompt-composer" \
    "${ROOT_DIR}/.agents/skills/go-test-review" \
    "${ROOT_DIR}/.agents/skills/go-security-spec" \
    "${destination}/.agents/skills/"
}

mutate_json() {
  local file="$1"
  local filter="$2"
  local temp="${file}.tmp"
  jq "${filter}" "${file}" >"${temp}"
  mv "${temp}" "${file}"
}

expect_mutation_failure() {
  local label="$1"
  local root="$2"
  if validate_selected_root "${root}" >/dev/null 2>&1; then
    fail "mutation unexpectedly passed: ${label}"
    return 1
  fi
}

run_mutation_fixtures() {
  ensure_temp_root
  local fixture_root="${TEMP_ROOT}/repo"
  local manifest path

  prepare_fixture_root "${fixture_root}"
  validate_selected_root "${fixture_root}" || return 1

  prepare_fixture_root "${fixture_root}"
  manifest="${fixture_root}/.agents/skills/agent-prompt-composer/evals/evals.json"
  mutate_json "${manifest}" 'del(.evals[] | select(.id == 0) | .trial_class)'
  expect_mutation_failure 'missing trial class' "${fixture_root}" || return 1

  prepare_fixture_root "${fixture_root}"
  manifest="${fixture_root}/.agents/skills/go-test-review/evals/evals.json"
  mutate_json "${manifest}" '.evals += [.evals[] | select(.id == 4)]'
  expect_mutation_failure 'duplicate selected ID' "${fixture_root}" || return 1

  prepare_fixture_root "${fixture_root}"
  manifest="${fixture_root}/.agents/skills/agent-prompt-composer/evals/evals.json"
  mutate_json "${manifest}" '(.evals[] | select(.id == 0) | .files[0]) = ".agents/skills/agent-prompt-composer/evals/files/skill-tooling.md"'
  expect_mutation_failure 'prompt and files disagreement' "${fixture_root}" || return 1

  prepare_fixture_root "${fixture_root}"
  manifest="${fixture_root}/.agents/skills/go-security-spec/evals/evals.json"
  mutate_json "${manifest}" '(.evals[] | select(.id == 4) | .files[0]) = "../outside.md"'
  expect_mutation_failure 'escaping input path' "${fixture_root}" || return 1

  prepare_fixture_root "${fixture_root}"
  path="${fixture_root}/.agents/skills/go-test-review/evals/fixtures/weak_oracle/spec.md"
  rm "${path}"
  ln -s create_handler_test.go "${path}"
  expect_mutation_failure 'symlink input' "${fixture_root}" || return 1

  prepare_fixture_root "${fixture_root}"
  rm "${fixture_root}/.agents/skills/go-security-spec/evals/fixtures/ownerless_denial/context.md"
  expect_mutation_failure 'missing input' "${fixture_root}" || return 1

  prepare_fixture_root "${fixture_root}"
  manifest="${fixture_root}/.agents/skills/go-test-review/evals/evals.json"
  mutate_json "${manifest}" '(.evals[] | select(.id == 4) | .files) += [(.evals[] | select(.id == 4) | .files[0])]'
  expect_mutation_failure 'duplicate destination' "${fixture_root}" || return 1

  prepare_fixture_root "${fixture_root}"
  printf '\nmutated\n' >>"${fixture_root}/.agents/skills/go-security-spec/evals/fixtures/identity_conflation/context.md"
  expect_mutation_failure 'exact fixture drift' "${fixture_root}" || return 1

  prepare_fixture_root "${fixture_root}"
  manifest="${fixture_root}/.agents/skills/go-test-review/evals/evals.json"
  mutate_json "${manifest}" '(.evals[] | select(.id == 5) | .expectations) = []'
  expect_mutation_failure 'missing oracle' "${fixture_root}" || return 1
}

write_fake_adapters() {
  FAKE_RUNNER="${TEMP_ROOT}/fake-runner.sh"
  FAKE_JUDGE="${TEMP_ROOT}/fake-judge.sh"
  cat >"${FAKE_RUNNER}" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
variant='' repo='' target='' trial_id='' seed='' prompt_file='' metadata_file=''
while (( $# > 0 )); do
  case "$1" in
    --variant) variant="$2"; shift 2 ;;
    --repo) repo="$2"; shift 2 ;;
    --target) target="$2"; shift 2 ;;
    --trial-id) trial_id="$2"; shift 2 ;;
    --seed) seed="$2"; shift 2 ;;
    --prompt-file) prompt_file="$2"; shift 2 ;;
    --metadata-file) metadata_file="$2"; shift 2 ;;
    *) exit 91 ;;
  esac
done
printf 'runner\n' >>"${FAKE_ADAPTER_COUNTER}"
[[ "${variant}" == baseline || "${variant}" == candidate ]]
[[ "${target}" == 'skill:fixture-skill:0' ]]
[[ -f "${prompt_file}" && "$(cat "${prompt_file}")" == *'.agents/skills/fixture-skill/evals/fixtures/input.txt'* ]]
[[ ! -e "${repo}/docs/spec-first-workflow-evals.md" ]]
if find "${repo}" -type f -path '*/evals/evals.json' -print | grep -q .; then
  exit 92
fi
[[ "$(cat "${repo}/.agents/skills/fixture-skill/evals/fixtures/input.txt")" == 'candidate-input' ]]
number="${trial_id#T}"
number="${number#0}"
[[ "${seed}" == "$((FAKE_SEED_BASE + number))" ]]
if [[ "${FAKE_RUNNER_MODE}" == snapshot_mutation ]]; then
  printf 'mutated\n' >>"${repo}/.agents/skills/fixture-skill/evals/fixtures/input.txt"
fi
if [[ "${FAKE_RUNNER_MODE}" == malformed_metadata ]]; then
  printf '{}\n' >"${metadata_file}"
  printf '%s output\n' "${variant}"
  exit 0
fi
model=fake-model
requested_seed="${seed}"
if [[ "${FAKE_RUNNER_MODE}" == metadata_mismatch && "${variant}" == candidate ]]; then model=other-model; fi
if [[ "${FAKE_RUNNER_MODE}" == seed_mismatch && "${variant}" == candidate ]]; then requested_seed=$((seed + 1)); fi
input_tokens=10 output_tokens=10 latency_ms=100 cost_usd=1
case "${FAKE_METRIC_MODE}" in
  lower)
    if [[ "${variant}" == candidate ]]; then input_tokens=4; output_tokens=4; latency_ms=50; cost_usd=0.5; fi
    ;;
  opposed)
    if [[ "${variant}" == candidate ]]; then input_tokens=4; output_tokens=4; latency_ms=200; cost_usd=0.5; fi
    ;;
  unavailable)
    input_tokens=null; output_tokens=null; latency_ms=null; cost_usd=null
    ;;
esac
temp="${metadata_file}.tmp"
jq -n --arg target "${target}" --arg trial_id "${trial_id}" --arg variant "${variant}" \
  --arg model "${model}" --argjson requested_seed "${requested_seed}" --argjson applied_seed "${seed}" \
  --argjson input_tokens "${input_tokens}" --argjson output_tokens "${output_tokens}" \
  --argjson latency_ms "${latency_ms}" --argjson cost_usd "${cost_usd}" \
  '{target:$target,trial_id:$trial_id,variant:$variant,model:$model,api:"fake-api",reasoning_effort:"medium",tool_config_sha256:"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",requested_seed:$requested_seed,applied_seed:$applied_seed,input_tokens:$input_tokens,output_tokens:$output_tokens,latency_ms:$latency_ms,cost_usd:$cost_usd}' \
  >"${temp}"
mv "${temp}" "${metadata_file}"
printf '%s output\n' "${variant}"
EOF
  cat >"${FAKE_JUDGE}" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
target='' trial_id='' expected_file='' baseline_output='' candidate_output='' baseline_metadata='' candidate_metadata=''
while (( $# > 0 )); do
  case "$1" in
    --target) target="$2"; shift 2 ;;
    --trial-id) trial_id="$2"; shift 2 ;;
    --expected-file) expected_file="$2"; shift 2 ;;
    --baseline-output) baseline_output="$2"; shift 2 ;;
    --candidate-output) candidate_output="$2"; shift 2 ;;
    --baseline-metadata) baseline_metadata="$2"; shift 2 ;;
    --candidate-metadata) candidate_metadata="$2"; shift 2 ;;
    *) exit 93 ;;
  esac
done
printf 'judge\n' >>"${FAKE_ADAPTER_COUNTER}"
[[ "${target}" == 'skill:fixture-skill:0' && -s "${expected_file}" && -s "${baseline_output}" && -s "${candidate_output}" ]]
[[ -s "${baseline_metadata}" && -s "${candidate_metadata}" ]]
jq -e '.expected_output == "fixture expected" and .oracles == ["fixture expectation"]' "${expected_file}" >/dev/null
if [[ "${FAKE_JUDGE_MODE}" == malformed ]]; then printf '{}\n'; exit 0; fi
baseline_pass=true candidate_pass=true non_regression=true hard='[]' uncertainty=null
case "${FAKE_JUDGE_MODE}" in
  improvement)
    baseline_pass=false
    ;;
  borderline)
    if [[ "${trial_id}" == T01 ]]; then baseline_pass=false; candidate_pass=false; uncertainty='"borderline"'; fi
    ;;
  safety_failure)
    if [[ "${trial_id}" == T01 ]]; then candidate_pass=false; hard='["safety regression"]'; fi
    ;;
esac
jq -n --arg target "${target}" --arg trial_id "${trial_id}" \
  --argjson baseline_pass "${baseline_pass}" --argjson candidate_pass "${candidate_pass}" \
  --argjson candidate_non_regression "${non_regression}" --argjson hard_invariant_failures "${hard}" \
  --argjson uncertainty_note "${uncertainty}" \
  '{target:$target,trial_id:$trial_id,baseline_pass:$baseline_pass,candidate_pass:$candidate_pass,candidate_non_regression:$candidate_non_regression,hard_invariant_failures:$hard_invariant_failures,uncertainty_note:$uncertainty_note}'
EOF
  chmod +x "${FAKE_RUNNER}" "${FAKE_JUDGE}"
}

prepare_harness_fixture() {
  local repo="$1"
  local trial_class="$2"
  local collision="$3"
  rm -rf -- "${repo}"
  mkdir -p "${repo}/docs"
  cp "${ROOT_DIR}/docs/spec-first-workflow-evals.md" "${repo}/docs/"
  git -C "${repo}" init --quiet
  if [[ "${collision}" == true ]]; then
    mkdir -p "${repo}/.agents/skills/fixture-skill/evals"
    ln -s ../../../../collision-target "${repo}/.agents/skills/fixture-skill/evals/fixtures"
  fi
  git -C "${repo}" add --all
  git -C "${repo}" -c user.name='Eval Fixture' -c user.email='eval@local' commit --quiet -m baseline
  FAKE_BASELINE="$(git -C "${repo}" rev-parse HEAD)"
  if [[ "${collision}" == true ]]; then rm "${repo}/.agents/skills/fixture-skill/evals/fixtures"; fi
  mkdir -p "${repo}/.agents/skills/fixture-skill/evals/fixtures"
  printf '%s\n' '# Fixture Skill' >"${repo}/.agents/skills/fixture-skill/SKILL.md"
  printf '%s\n' 'candidate-input' >"${repo}/.agents/skills/fixture-skill/evals/fixtures/input.txt"
  cat >"${repo}/.agents/skills/fixture-skill/evals/evals.json" <<EOF
{
  "skill_name": "fixture-skill",
  "evals": [{
    "id": 0,
    "trial_class": "${trial_class}",
    "prompt": "Read .agents/skills/fixture-skill/evals/fixtures/input.txt and return the result.",
    "expected_output": "fixture expected",
    "expectations": ["fixture expectation"],
    "files": [".agents/skills/fixture-skill/evals/fixtures/input.txt"]
  }]
}
EOF
  git -C "${repo}" add --all
  git -C "${repo}" -c user.name='Eval Fixture' -c user.email='eval@local' commit --quiet -m candidate
}

run_fake_case() {
  local label="$1" expected="$2" trial_class="$3" runner_mode="$4" judge_mode="$5" metric_mode="$6" targets="$7" collision="$8" no_adapter="$9"
  local repo="${TEMP_ROOT}/harness-${label}" artifact="${TEMP_ROOT}/artifacts-${label}" counter="${TEMP_ROOT}/counter-${label}"
  prepare_harness_fixture "${repo}" "${trial_class}" "${collision}"
  rm -f -- "${counter}"
  local output status
  if output="$(env \
    WORKFLOW_EVAL_REPO_ROOT="${repo}" \
    WORKFLOW_EVAL_TARGETS="${targets}" \
    WORKFLOW_EVAL_SEED_BASE=5600 \
    WORKFLOW_EVAL_BASE_REF="${FAKE_BASELINE}" \
    WORKFLOW_EVAL_RUNNER="${FAKE_RUNNER}" \
    WORKFLOW_EVAL_JUDGE="${FAKE_JUDGE}" \
    WORKFLOW_EVAL_COST_AUTHORIZED=true \
    FAKE_ADAPTER_COUNTER="${counter}" \
    FAKE_SEED_BASE=5600 \
    FAKE_RUNNER_MODE="${runner_mode}" \
    FAKE_JUDGE_MODE="${judge_mode}" \
    FAKE_METRIC_MODE="${metric_mode}" \
    bash "${ROOT_DIR}/scripts/dev/workflow-behavior-evals.sh" run "${artifact}" 2>&1)"; then
    status=0
  else
    status=$?
  fi
  if [[ "${expected}" == pass && "${status}" -ne 0 ]]; then
    printf '%s\n' "${output}" >&2
    if [[ -f "${artifact}/summary.json" ]]; then cat "${artifact}/summary.json" >&2; fi
    fail "fake harness case failed: ${label}"
    return 1
  fi
  if [[ "${expected}" == fail && "${status}" -eq 0 ]]; then
    fail "fake harness mutation unexpectedly passed: ${label}"
    return 1
  fi
  if [[ "${no_adapter}" == true && -e "${counter}" ]]; then
    fail "${label} invoked an adapter before structural rejection"
    return 1
  fi
  FAKE_ARTIFACT="${artifact}"
  FAKE_OUTPUT="${output}"
}

assert_fake_summary() {
  local expression="$1"
  if ! jq -e "${expression}" "${FAKE_ARTIFACT}/summary.json" >/dev/null; then
    printf '%s\n' "${FAKE_OUTPUT}" >&2
    fail "fake harness summary mismatch: ${expression}"
  fi
}

run_fake_harness_fixtures() {
  write_fake_adapters

  run_fake_case valid pass standard valid valid equal 'skill:fixture-skill:0' false false
  assert_fake_summary '.accepted == true and .targets[0].trials == 5 and .targets[0].label == "non_regression"'

  run_fake_case missing-target fail standard valid valid equal '' false true
  run_fake_case duplicate-target fail standard valid valid equal 'skill:fixture-skill:0,skill:fixture-skill:0' false true
  run_fake_case unequal-input-destination fail standard valid valid equal 'skill:fixture-skill:0' true true
  run_fake_case malformed-metadata fail standard malformed_metadata valid equal 'skill:fixture-skill:0' false false
  run_fake_case metadata-drift fail standard metadata_mismatch valid equal 'skill:fixture-skill:0' false false
  run_fake_case seed-mismatch fail standard seed_mismatch valid equal 'skill:fixture-skill:0' false false
  run_fake_case snapshot-mutation fail standard snapshot_mutation valid equal 'skill:fixture-skill:0' false false
  run_fake_case malformed-judgment fail standard valid malformed equal 'skill:fixture-skill:0' false false

  run_fake_case safety-failure fail safety_authority valid safety_failure equal 'skill:fixture-skill:0' false false
  jq -e '.trials == 10 and .accepted == false and .hard_invariant_failure_count == 1 and (.hard_invariant_failures | length) == 1' \
    "${FAKE_ARTIFACT}/targets/skill_fixture-skill_0/summary.json" >/dev/null \
    || fail 'safety failure did not exercise the ten-trial hard gate'

  run_fake_case borderline pass standard valid borderline equal 'skill:fixture-skill:0' false false
  assert_fake_summary '.targets[0].trials == 10 and .targets[0].label == "non_regression"'

  run_fake_case unavailable-metrics pass standard valid improvement unavailable 'skill:fixture-skill:0' false false
  assert_fake_summary '.targets[0].participating_metrics == 0 and .targets[0].label == "quality_resource_tradeoff"'

  run_fake_case opposed-metrics pass standard valid improvement opposed 'skill:fixture-skill:0' false false
  assert_fake_summary '.targets[0].participating_metrics == 3 and .targets[0].label == "quality_resource_tradeoff"'

  run_fake_case improvement pass standard valid improvement lower 'skill:fixture-skill:0' false false
  assert_fake_summary '.targets[0].accepted == true and .targets[0].label == "improvement"'
}

validate_selected_root "${ROOT_DIR}"
run_mutation_fixtures
run_fake_harness_fixtures
echo 'instruction eval check passed: 4 manifests, 12 selected cases, 10 exact fixtures, manifest/input and fake-adapter mutation coverage'

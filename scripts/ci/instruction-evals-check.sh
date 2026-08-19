#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
EVAL_FILE="${ROOT_DIR}/evals/instructions/evals.json"

command -v jq >/dev/null 2>&1 || {
	printf 'instruction evals: jq is required\n' >&2
	exit 1
}

jq -e '
  .skill_name == "repository-instructions" and
	  .execution == {
    "repository_state": "disposable_copy",
    "production_credentials": "absent",
    "external_writes": "deny"
	  } and
	  .measurement == {
	    "required_run_metadata": ["harness", "model", "reasoning_effort", "tool_profile", "agent_command_label"],
	    "required_metrics": ["input_tokens", "output_tokens", "tool_calls", "skill_loads", "lane_identities", "empty_waits"],
	    "behavior_verdict": "manual_expectation_grading"
	  } and
	  (.fixture.setup_patch | type == "string" and length > 0) and
	  (.evals | type == "array" and length >= 14) and
  ([.evals[].id] | length == (unique | length)) and
  all(.evals[];
    (.id | type == "number") and
    (.prompt | type == "string" and length > 0) and
    (.expected_output | type == "string" and length > 0) and
	    (.files | type == "array" and length > 0 and all(.[]; type == "string" and length > 0)) and
    (.expectations | type == "array" and length >= 2) and
    all(.expectations[]; type == "string" and length > 0)
  )
' "${EVAL_FILE}" >/dev/null

fixture_patch="$(jq -r '.fixture.setup_patch' "${EVAL_FILE}")"
fixture_patch_path="${ROOT_DIR}/${fixture_patch}"
[[ -f "${fixture_patch_path}" ]] || {
	printf 'instruction evals: fixture patch is missing: %s\n' "${fixture_patch}" >&2
	exit 1
}

git -C "${ROOT_DIR}" apply --check "${fixture_patch_path}"
fixture_files="$(git -C "${ROOT_DIR}" apply --numstat "${fixture_patch_path}" | awk '{print $3}')"
while IFS= read -r file; do
	[[ -e "${ROOT_DIR}/${file}" ]] || grep -Fxq "${file}" <<<"${fixture_files}" || {
		printf 'instruction evals: file is neither current nor fixture-owned: %s\n' "${file}" >&2
		exit 1
	}
done < <(jq -r '[.evals[].files[]] | unique[]' "${EVAL_FILE}")

printf 'instruction eval surface is valid\n'

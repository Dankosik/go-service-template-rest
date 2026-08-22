#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
EVAL_FILE="${ROOT_DIR}/evals/hard-skills/evals.json"
COVERAGE_FILE="${ROOT_DIR}/evals/hard-skills/coverage.json"

fail() {
	printf 'hard-skill evals: %s\n' "$1" >&2
	exit 1
}

command -v jq >/dev/null 2>&1 || fail "jq is required"

jq -e '
  ["go-api-contract", "go-coder", "go-idiomatic", "go-language-simplifier",
   "go-structural-quality", "go-systematic-debugging",
   "go-verification-before-completion"] as $skills |
  ["trigger", "non_trigger", "collision", "decision", "completion"] as $categories |
  . as $catalog |
  .skill_name == "repository-hard-skills" and
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
  (.evals | type == "array" and length == 35) and
  ([.evals[].id] | length == (unique | length)) and
  all(.evals[];
    (.id | type == "number") and
    (.skill as $skill | ($skills | index($skill)) != null) and
    (.category as $category | ($categories | index($category)) != null) and
    (.prompt | type == "string" and length > 0) and
    (.expected_output | type == "string" and length > 0) and
    (.files | type == "array" and length > 0 and all(.[]; type == "string" and length > 0)) and
    (.expectations | type == "array" and length >= 2 and all(.[]; type == "string" and length > 0))
  ) and
  all($skills[]; . as $skill |
    all($categories[]; . as $category |
      ([ $catalog.evals[] | select(.skill == $skill and .category == $category) ] | length) == 1
    )
  )
' "${EVAL_FILE}" >/dev/null || fail "eval catalog shape or coverage is invalid"

jq -e --slurpfile catalog "${EVAL_FILE}" '
  ["trigger", "non_trigger", "collision", "decision", "completion"] as $categories |
  .required_categories == $categories and
  (.skills | keys | sort) == ([ $catalog[0].evals[].skill ] | unique | sort) and
  all(.skills | to_entries[]; . as $entry |
    ($entry.value | keys | sort) == ($categories | sort) and
    all($entry.value | to_entries[]; . as $case |
      any($catalog[0].evals[];
        .id == $case.value and .skill == $entry.key and .category == $case.key
      )
    )
  )
' "${COVERAGE_FILE}" >/dev/null || fail "coverage map does not match eval cases"

fixture_patch="$(jq -r '.fixture.setup_patch' "${EVAL_FILE}")"
[[ -f "${ROOT_DIR}/${fixture_patch}" ]] || fail "fixture patch is missing: ${fixture_patch}"
git -C "${ROOT_DIR}" apply --check "${ROOT_DIR}/${fixture_patch}"

while IFS= read -r file; do
	[[ -e "${ROOT_DIR}/${file}" ]] || fail "eval file is missing: ${file}"
done < <(jq -r '[.evals[].files[]] | unique[]' "${EVAL_FILE}")

printf 'hard-skill eval surface is valid\n'

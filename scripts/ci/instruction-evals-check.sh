#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
EVAL_FILE="${ROOT_DIR}/evals/instructions/evals.json"

fail() {
	printf 'instruction evals: %s\n' "$1" >&2
	exit 1
}

command -v jq >/dev/null 2>&1 || {
	fail "jq is required"
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

expected_roles=$'adjudicator-agent\nevidence-agent\nreviewer-agent\nspecialist-agent\nworker-agent'
actual_roles=$(find "${ROOT_DIR}/.agents/roles" -maxdepth 1 -type f -name '*.toml' \
	-exec basename {} .toml \; | sort)
[[ "${actual_roles}" == "${expected_roles}" ]] ||
	fail "canonical roles must be the five capability roles"

for retired in \
	phase-movement.md handoff.md acceptance-unit-closure.md review-findings.md; do
	[[ ! -e "${ROOT_DIR}/docs/spec-first-workflow/shared/${retired}" ]] ||
		fail "retired shared owner remains: ${retired}"
done
[[ -f "${ROOT_DIR}/docs/spec-first-workflow/shared/transition.md" ]] ||
	fail "shared transition owner is missing"

reverse_links=""
for skill_file in "${ROOT_DIR}"/.agents/skills/*/SKILL.md; do
	grep -Fxq '  invocation: model' "${skill_file}" || continue
	if match=$(grep -RInE 'AGENTS\.md|docs/spec-first-workflow\.md|docs/spec-first-workflow/phases' \
		"$(dirname "${skill_file}")" --include='*.md' 2>/dev/null); then
		reverse_links+="${match}"$'\n'
	fi
done
[[ -z "${reverse_links}" ]] ||
	fail "model method/reference layer reselects bootstrap, router, or phase: ${reverse_links%%$'\n'*}"

if phase_reverse=$(grep -RInE 'AGENTS\.md|spec-first-workflow\.md' \
	"${ROOT_DIR}/docs/spec-first-workflow/phases" --include='*.md' 2>/dev/null); then
	fail "phase layer reselects bootstrap or router: ${phase_reverse%%$'\n'*}"
fi

while IFS= read -r selector; do
	[[ "$(sed -n '/[^[:space:]]/{p;q;}' "${selector}")" == '# Reference Selector' ]] ||
		fail "reference index is not a selector: ${selector#"${ROOT_DIR}/"}"
done < <(find "${ROOT_DIR}/.agents/skills" -path '*/references/index.md' -type f | sort)

[[ -f "${ROOT_DIR}/docs/architecture/http.md" ]] ||
	fail "HTTP architecture leaf is missing"

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

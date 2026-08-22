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

grep -Fq 'Unrelated or pre-existing defects are observations, not blockers' \
	"${ROOT_DIR}/docs/validation-routing.md" ||
	fail "validation routing lost the bounded pre-existing-defect policy"

jq -e '
  any(.evals[];
    .id == 22 and
    (.prompt | contains("full-history")) and
    any(.expectations[]; contains("do not block acceptance"))
  )
' "${EVAL_FILE}" >/dev/null ||
	fail "change-scoped acceptance regression eval is missing"

jq -e '
  .evals as $evals |
  all([23, 24, 25, 26, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49][];
    . as $id | any($evals[]; .id == $id))
' "${EVAL_FILE}" >/dev/null ||
	fail "cross-harness orchestrator or implementation-topology evals are missing"

jq -e '
  any(.evals[];
    .id == 26 and
    (.prompt | contains("Blocked result")) and
    any(.expectations[]; contains("does not ask the user")) and
    any(.expectations[]; contains("reopens Planning"))
  )
' "${EVAL_FILE}" >/dev/null ||
	fail "autonomous orchestrator recovery regression eval is missing"

grep -Fq 'the current adapter selected by [Agent' \
	"${ROOT_DIR}/.agents/skills/orchestrator/SKILL.md" ||
	fail "orchestrator skill is not harness-neutral"
grep -Fq 'independently acceptable repository outcome' \
	"${ROOT_DIR}/docs/spec-first-workflow/phases/planning/ledger-contract.md" ||
	fail "ledger contract lost the independently-acceptable task boundary"
grep -Fq 'Keep layers of one Outcome in one unit' \
	"${ROOT_DIR}/docs/spec-first-workflow/phases/planning/ledger-contract.md" ||
	fail "ledger contract lost the too-small versus too-big sandwich"
grep -Fq 'Isolate concurrent mutation only when it' \
	"${ROOT_DIR}/docs/agent-harness.md" ||
	fail "agent harness lost isolation cost routing"
if grep -Fq 'implemented, proved, repaired, or accepted independently' \
	"${ROOT_DIR}/docs/spec-first-workflow/phases/planning/ledger-contract.md"; then
	fail "ledger contract still splits on independently implementable parts"
fi
grep -Fq 'Mutable owners:' \
	"${ROOT_DIR}/docs/spec-first-workflow/interfaces/task-packet-v1.md" ||
	fail "task packet is missing mutable owners"
grep -Fq 'Exclusive locks:' \
	"${ROOT_DIR}/docs/spec-first-workflow/interfaces/task-packet-v1.md" ||
	fail "task packet is missing exclusive locks"
grep -Fq 'do not write canonical ledger state' \
	"${ROOT_DIR}/.agents/skills/acceptance-unit-lead/SKILL.md" ||
	fail "acceptance-unit-lead still writes the canonical ledger under orchestration"
if grep -Fq 'only the Acceptance-Unit Lead writes' "${EVAL_FILE}"; then
	fail "instruction evals still expect the Lead to write canonical ledger state"
fi
if grep -RFq 'primary session is the Orchestrator carrier' \
	"${ROOT_DIR}/Grok.md" \
	"${ROOT_DIR}/docs/agent-harness/grok-build.md" \
	"${ROOT_DIR}/docs/prompt-composition.md" \
	"${ROOT_DIR}/.grok/rules"; then
	fail "Grok still binds every primary session as Orchestrator"
fi
jq -e '
  any(.evals[];
    .id == 37 and
    any(.expectations[]; contains("does not wait for every member"))
  )
' "${EVAL_FILE}" >/dev/null ||
	fail "ready-frontier refill eval is missing"
jq -e '
  any(.evals[];
    .id == 45 and
    any(.expectations[]; contains("before landing T2"))
  )
' "${EVAL_FILE}" >/dev/null ||
	fail "landing-mailbox refill eval is missing"
if grep -Fq 'for an ordinary completed Go change' \
	"${ROOT_DIR}/docs/validation/go.md"; then
	fail "Go validation still treats make check as every completed change"
fi
grep -Fq 'not a per-lane or per-sibling default' \
	"${ROOT_DIR}/docs/validation/go.md" ||
	fail "Go validation lost the aggregate-once default"
grep -Fq 'A discovered exclusive lock' \
	"${ROOT_DIR}/docs/spec-first-workflow/phases/planning/ledger-contract.md" ||
	fail "ledger contract lost discovered-lock frontier update"
grep -Fq 'At most one bounded delta recheck' \
	"${ROOT_DIR}/docs/spec-first-workflow/shared/review.md" ||
	fail "review lost the single-delta stop"
grep -Fq 'It counts live Leads' \
	"${ROOT_DIR}/docs/spec-first-workflow/phases/planning/ledger-contract.md" ||
	fail "ledger contract lost child-aware capacity"
if grep -RIFq 'has no Ledger Orchestrator carrier' \
	"${ROOT_DIR}/docs/agent-harness"; then
	fail "a harness adapter still prohibits ledger orchestration by product name"
fi
if ! grep -Fq 'in Codex and ' "${ROOT_DIR}/docs/prompt-composition.md" ||
	! grep -Fq 'in Claude Code, Qwen Code, Grok Build, Cursor, or OpenCode' "${ROOT_DIR}/docs/prompt-composition.md"; then
	fail "native skill syntax is not mapped across harnesses"
fi

expected_roles=$'adjudicator-agent\nevidence-agent\nreviewer-agent\nspecialist-agent\nworker-agent'
actual_roles=$(find "${ROOT_DIR}/.agents/roles" -maxdepth 1 -type f -name '*.toml' \
	-exec basename {} .toml \; | sort)
[[ "${actual_roles}" == "${expected_roles}" ]] ||
	fail "canonical roles must be the five capability roles"

for retired in \
	phase-movement.md handoff.md acceptance-unit-closure.md review-findings.md \
	delegation.md; do
	[[ ! -e "${ROOT_DIR}/docs/spec-first-workflow/shared/${retired}" ]] ||
		fail "retired shared owner remains: ${retired}"
done
[[ -f "${ROOT_DIR}/docs/spec-first-workflow/shared/transition.md" ]] ||
	fail "shared transition owner is missing"
[[ -f "${ROOT_DIR}/docs/spec-first-workflow/shared/read-only-delegation.md" ]] ||
	fail "read-only delegation owner is missing"

for retired in \
	.agents/contracts/decision-result-v1.md \
	.agents/roles/interfaces/api-contract-finding-v1.md; do
	[[ ! -e "${ROOT_DIR}/${retired}" ]] || fail "retired schema owner remains: ${retired}"
done
for interface in \
	decision-result-v1.md evidence-result-v1.md transition-result-v1.md \
	task-packet-v1.md acceptance-result-v1.md; do
	[[ -f "${ROOT_DIR}/docs/spec-first-workflow/interfaces/${interface}" ]] ||
		fail "canonical interface is missing: ${interface}"
done
[[ -f "${ROOT_DIR}/docs/spec-first-workflow/phases/implementation-review.md" ]] ||
	fail "implementation review adapter is missing"
! grep -Fq '## Implementation Review' \
	"${ROOT_DIR}/docs/spec-first-workflow/shared/review.md" ||
	fail "shared review still owns implementation review method"

schema_markers='^(decision_or_constraint|strongest_rejected_alternative|gap_or_next_owner|movement_evidence):'
if misplaced_schema=$(grep -RInE "${schema_markers}" \
	"${ROOT_DIR}/.agents/contracts" \
	"${ROOT_DIR}/.agents/roles" \
	"${ROOT_DIR}/docs/spec-first-workflow/phases" \
	"${ROOT_DIR}/docs/spec-first-workflow/shared" \
	--include='*.md' 2>/dev/null); then
	fail "output field is owned outside interfaces: ${misplaced_schema%%$'\n'*}"
fi

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

legacy_leaf='^## (When To Load|Behavior Change Thesis|Decision Rubric|Imitate|Agent Traps|Validation Shape)$|Behavior Change Thesis|this file (makes|supplies)'
if procedural_leaf=$(grep -RInE "${legacy_leaf}" "${ROOT_DIR}/.agents/skills" \
	--include='*.md' 2>/dev/null); then
	fail "leaf reference retains procedural template prose: ${procedural_leaf%%$'\n'*}"
fi

for selector in "${ROOT_DIR}"/docs/universal-disciplines/*/SKILL.md; do
	grep -Fq 'Inherit' "${selector}" ||
		fail "universal selector does not inherit active context: ${selector#"${ROOT_DIR}/"}"
	grep -Fq 'Load one branch:' "${selector}" ||
		fail "universal selector lacks one-branch routing: ${selector#"${ROOT_DIR}/"}"
	(( $(wc -w <"${selector}") <= 160 )) ||
		fail "universal selector exceeds 160 words: ${selector#"${ROOT_DIR}/"}"
	if mode_reselection=$(grep -nE '^## (Choose|Authority|Report)|Global completion criterion|Run only the branch|For build or fix' \
		"${selector}" 2>/dev/null); then
		fail "universal selector reselects higher-layer mechanics: ${selector#"${ROOT_DIR}/"}:${mode_reselection%%$'\n'*}"
	fi
done

if grep -Eq '[0-9]+ skills|specialist definitions each|[0-9]+ Markdown files' \
	"${ROOT_DIR}/README.md"; then
	fail "README contains a manually maintained agent inventory count"
fi

instruction_roots=(
	AGENTS.md CLAUDE.md Grok.md QWEN.md
	docs/spec-first-workflow.md docs/spec-first-workflow
	docs/prompt-maintenance.md docs/prompt-composition.md docs/skill-authoring.md
	docs/agent-harness.md docs/agent-harness docs/universal-disciplines .agents .grok .cursor .opencode
)
while IFS= read -r markdown; do
	while IFS= read -r target; do
		target="${target%%#*}"
		case "${target}" in
		'' | http://* | https://* | mailto:* | skill://*) continue ;;
		esac
		[[ -e "$(dirname "${markdown}")/${target}" ]] ||
			fail "broken instruction link: ${markdown#"${ROOT_DIR}/"} -> ${target}"
	done < <(grep -oE '\]\([^)]+\)' "${markdown}" 2>/dev/null |
		sed -e 's/^](//' -e 's/)$//' || true)
done < <(
	for path in "${instruction_roots[@]}"; do
		path="${ROOT_DIR}/${path}"
		if [[ -d "${path}" ]]; then
			find "${path}" \( -name node_modules -o -name .git \) -prune -o -type f -name '*.md' -print
		elif [[ -f "${path}" ]]; then
			printf '%s\n' "${path}"
		fi
	done | sort -u
)

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

bash "${ROOT_DIR}/scripts/ci/hard-skill-evals-check.sh" >/dev/null

printf 'instruction eval surface is valid\n'

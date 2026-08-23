#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
EVAL_FILE="${INSTRUCTION_EVAL_FILE:-evals/instructions/evals.json}"
if [[ "${EVAL_FILE}" != /* ]]; then
	EVAL_FILE="${ROOT_DIR}/${EVAL_FILE}"
fi
BASELINE_REF="${INSTRUCTION_EVAL_BASELINE_REF:-HEAD}"
CANDIDATE_SOURCE="${INSTRUCTION_EVAL_CANDIDATE:-worktree}"
CASE_FILTER="${INSTRUCTION_EVAL_CASES:-}"
OUTPUT_ROOT="${INSTRUCTION_EVAL_OUTPUT:-${ROOT_DIR}/.artifacts/instruction-evals/$(date -u +%Y%m%dT%H%M%SZ)}"
HARNESS_NAME="${INSTRUCTION_EVAL_HARNESS:-}"
MODEL_NAME="${INSTRUCTION_EVAL_MODEL:-}"
REASONING_EFFORT="${INSTRUCTION_EVAL_REASONING_EFFORT:-}"
TOOL_PROFILE="${INSTRUCTION_EVAL_TOOL_PROFILE:-}"
AGENT_COMMAND_LABEL="${INSTRUCTION_EVAL_COMMAND_LABEL:-}"
TRACE_FORMAT="${INSTRUCTION_EVAL_TRACE_FORMAT:-raw}"
RUN_VARIANTS="${INSTRUCTION_EVAL_VARIANTS:-baseline,candidate}"
REPEAT_COUNT="${INSTRUCTION_EVAL_REPEATS:-1}"

if [[ "${OUTPUT_ROOT}" != /* ]]; then
	OUTPUT_ROOT="${ROOT_DIR}/${OUTPUT_ROOT}"
fi

fail() {
	printf 'instruction evals: %s\n' "$*" >&2
	exit 1
}

[[ "${1:-}" == "--" ]] && shift
(( $# > 0 )) || fail 'pass an agent command after --; it must read the prompt from stdin'
agent_command=("$@")

[[ -n "${HARNESS_NAME}" ]] || fail 'INSTRUCTION_EVAL_HARNESS is required'
[[ -n "${MODEL_NAME}" ]] || fail 'INSTRUCTION_EVAL_MODEL is required'
[[ -n "${REASONING_EFFORT}" ]] || fail 'INSTRUCTION_EVAL_REASONING_EFFORT is required'
[[ -n "${TOOL_PROFILE}" ]] || fail 'INSTRUCTION_EVAL_TOOL_PROFILE is required'
[[ -n "${AGENT_COMMAND_LABEL}" ]] || fail 'INSTRUCTION_EVAL_COMMAND_LABEL is required'
case "${TRACE_FORMAT}" in
	raw | codex-jsonl) ;;
	*) fail 'INSTRUCTION_EVAL_TRACE_FORMAT must be raw or codex-jsonl' ;;
esac
[[ "${REPEAT_COUNT}" =~ ^[1-9][0-9]*$ ]] || fail 'INSTRUCTION_EVAL_REPEATS must be a positive integer'

IFS=',' read -r -a run_variants <<<"${RUN_VARIANTS}"
(( ${#run_variants[@]} > 0 )) || fail 'INSTRUCTION_EVAL_VARIANTS is empty'
for variant in "${run_variants[@]}"; do
	case "${variant}" in
		baseline | candidate | routing_ablation | method_ablation | reference_ablation) ;;
		*) fail "unknown eval variant: ${variant}" ;;
	esac
done

command -v jq >/dev/null 2>&1 || fail 'jq is required'
command -v git >/dev/null 2>&1 || fail 'git is required'
command -v tar >/dev/null 2>&1 || fail 'tar is required'
[[ -f "${EVAL_FILE}" ]] || fail "eval catalog is missing: ${EVAL_FILE}"

bash "${ROOT_DIR}/scripts/ci/instruction-evals-check.sh" >/dev/null
git -C "${ROOT_DIR}" rev-parse --verify "${BASELINE_REF}^{commit}" >/dev/null
if [[ "${CANDIDATE_SOURCE}" != "worktree" ]]; then
	git -C "${ROOT_DIR}" rev-parse --verify "${CANDIDATE_SOURCE}^{commit}" >/dev/null
fi

# shellcheck source=scripts/lib/manifest.sh
source "${ROOT_DIR}/scripts/lib/manifest.sh"
manifest_paths "${ROOT_DIR}/template-owned.paths"
paths+=("docs/build-test-and-development-commands.md")

fixture_patch="$(jq -r '.fixture.setup_patch' "${EVAL_FILE}")"
scratch_root="$(mktemp -d "${TMPDIR:-/tmp}/instruction-evals.XXXXXX")"
trap 'rm -rf "${scratch_root}"' EXIT
[[ ! -e "${OUTPUT_ROOT}" ]] || fail "output already exists: ${OUTPUT_ROOT}"
mkdir -p "${OUTPUT_ROOT}"

copy_current_path() {
	local relative_path="$1"
	local destination="$2"
	[[ -e "${ROOT_DIR}/${relative_path}" || -L "${ROOT_DIR}/${relative_path}" ]] || return 0
	(cd "${ROOT_DIR}" && tar -cf - "${relative_path}") | (cd "${destination}" && tar -xf -)
}

apply_ablation() {
	local variant="$1"
	local destination="$2"
	local skill_name="$3"
	local target_reference="$4"
	local skill_path="${destination}/.agents/skills/${skill_name}/SKILL.md"
	local temporary_path="${skill_path}.ablation"

	case "${variant}" in
		routing_ablation)
			[[ -f "${skill_path}" ]] || fail "routing ablation skill is missing: ${skill_name}"
			awk '
			  /^description:/ {
			    print "description: \"Explicit invocation only.\""
			    print "disable-model-invocation: true"
			    next
			  }
			  { print }
			' "${skill_path}" >"${temporary_path}"
			mv "${temporary_path}" "${skill_path}"
			mkdir -p "${destination}/.agents/skills/${skill_name}/agents"
			printf 'policy:\n  allow_implicit_invocation: false\n' \
				>"${destination}/.agents/skills/${skill_name}/agents/openai.yaml"
			;;
		method_ablation)
			[[ -f "${skill_path}" ]] || fail "method ablation skill is missing: ${skill_name}"
			awk '
			  /^---$/ { delimiters++; print; if (delimiters == 2) exit; next }
			  { print }
			' "${skill_path}" >"${temporary_path}"
			mv "${temporary_path}" "${skill_path}"
			;;
		reference_ablation)
			[[ -n "${target_reference}" ]] || fail "reference ablation target is missing for ${skill_name}"
			[[ -f "${destination}/${target_reference}" ]] || fail "reference ablation file is missing: ${target_reference}"
			printf '# Ablated reference\n\nThis leaf is intentionally absent in this eval variant.\n' \
				>"${destination}/${target_reference}"
			;;
		baseline | candidate) ;;
	esac
}

materialize() {
	local variant="$1"
	local destination="$2"
	local skill_name="$3"
	local target_reference="$4"
	local source_ref="${BASELINE_REF}"
	local patch_file
	local untracked

	if [[ "${variant}" != "baseline" && "${CANDIDATE_SOURCE}" != "worktree" ]]; then
		source_ref="${CANDIDATE_SOURCE}"
	fi

	mkdir -p "${destination}"
	git -C "${ROOT_DIR}" archive "${source_ref}" | tar -xf - -C "${destination}"

	if [[ "${variant}" != "baseline" && "${CANDIDATE_SOURCE}" == "worktree" ]]; then
		patch_file="${scratch_root}/candidate.patch"
		git -C "${ROOT_DIR}" diff --binary "${BASELINE_REF}" -- "${paths[@]}" >"${patch_file}"
		if [[ -s "${patch_file}" ]]; then
			git -C "${destination}" apply "${patch_file}"
		fi
		while IFS= read -r untracked; do
			[[ -n "${untracked}" ]] && copy_current_path "${untracked}" "${destination}"
		done < <(git -C "${ROOT_DIR}" ls-files --others --exclude-standard -- "${paths[@]}")
	fi

	# Evaluation catalogs contain prompts and expected outputs. Keep them out of
	# every model-visible tree so neither side can read the answer oracle and a
	# candidate catalog change cannot contaminate the measured instruction delta.
	rm -rf "${destination}/evals/instructions" "${destination}/evals/hard-skills"
	git -C "${destination}" apply "${ROOT_DIR}/${fixture_patch}"
	apply_ablation "${variant}" "${destination}" "${skill_name}" "${target_reference}"

	git -C "${destination}" init -q
	git -C "${destination}" add -A
	git -C "${destination}" -c user.name='Instruction Eval' -c user.email='instruction-eval@invalid' commit -qm fixture
}

case_ids="$(jq -r '.evals[].id' "${EVAL_FILE}")"
if [[ -n "${CASE_FILTER}" ]]; then
	case_ids="$(tr ',' '\n' <<<"${CASE_FILTER}" | sed '/^[[:space:]]*$/d')"
fi

failures=0
while IFS= read -r case_id; do
	[[ "${case_id}" =~ ^[0-9]+$ ]] || fail "invalid case id: ${case_id}"
	case_json="$(jq -c --argjson id "${case_id}" '.evals[] | select(.id == $id)' "${EVAL_FILE}")"
	[[ -n "${case_json}" ]] || fail "unknown case id: ${case_id}"

	case_dir="${OUTPUT_ROOT}/case-${case_id}"
	mkdir -p "${case_dir}"
	printf '%s\n' "${case_json}" | jq . >"${case_dir}/case.json"
	jq -r '.prompt' <<<"${case_json}" >"${case_dir}/prompt.txt"
	skill_name="$(jq -r '.skill' <<<"${case_json}")"
	target_reference="$(jq -r '.target_reference // empty' <<<"${case_json}")"

	repeat=1
	while (( repeat <= REPEAT_COUNT )); do
		printf -v repeat_label '%02d' "${repeat}"
		repeat_dir="${case_dir}/repeat-${repeat_label}"
		mkdir -p "${repeat_dir}"

		for variant in "${run_variants[@]}"; do
			if [[ "${variant}" == "reference_ablation" && -z "${target_reference}" ]]; then
				continue
			fi

			workdir="${scratch_root}/case-${case_id}-repeat-${repeat_label}-${variant}"
			materialize "${variant}" "${workdir}" "${skill_name}" "${target_reference}"
			variant_dir="${repeat_dir}/${variant}"
			mkdir -p "${variant_dir}"
			started_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
			start_seconds="$(date +%s)"

			set +e
			(
				cd "${workdir}"
				env -i \
					HOME="${HOME}" \
					PATH="${PATH}" \
					SHELL="${SHELL:-/bin/sh}" \
					TMPDIR="${TMPDIR:-/tmp}" \
					USER="${USER:-instruction-eval}" \
					LANG="${LANG:-C.UTF-8}" \
					INSTRUCTION_EVAL_CASE_ID="${case_id}" \
					INSTRUCTION_EVAL_SIDE="${variant}" \
					INSTRUCTION_EVAL_VARIANT="${variant}" \
					INSTRUCTION_EVAL_REPEAT="${repeat}" \
					"${agent_command[@]}" <"${case_dir}/prompt.txt"
			) >"${variant_dir}/stdout.log" 2>"${variant_dir}/stderr.log"
			exit_code=$?
			set -e

			end_seconds="$(date +%s)"
			git -C "${workdir}" status --short >"${variant_dir}/status.txt"
			git -C "${workdir}" diff --binary >"${variant_dir}/candidate.patch"
			if [[ "${TRACE_FORMAT}" == "codex-jsonl" ]]; then
				jq -s '
				  def completed_tools:
				    [.[] | select(.type == "item.completed") | .item
				      | select(.type == "command_execution" or .type == "mcp_tool_call" or .type == "web_search" or .type == "collab_tool_call")];
				  def command_text:
				    if type == "string" then .
				    elif type == "array" then map(tostring) | join(" ")
				    else ""
				    end;
				  {
				    input_tokens: ([.[] | select(.type == "turn.completed") | .usage.input_tokens // empty] | add // null),
				    cached_input_tokens: ([.[] | select(.type == "turn.completed") | .usage.cached_input_tokens // empty] | add // null),
				    output_tokens: ([.[] | select(.type == "turn.completed") | .usage.output_tokens // empty] | add // null),
				    tool_calls: (completed_tools | length),
				    skill_loads: ([completed_tools[] | (.command // .text // "" | command_text) | select(test("SKILL\\.md"))] | unique),
				    lane_identities: ([completed_tools[] | select(.type == "collab_tool_call") | .receiver_thread_ids[]?] | unique),
				    empty_waits: ([completed_tools[] | select(.type == "collab_tool_call" and .tool == "wait" and ((.receiver_thread_ids // []) | length == 0))] | length),
				    source: "codex-jsonl"
				  }
				' "${variant_dir}/stdout.log" >"${variant_dir}/metrics.json"
			else
				jq -n '{input_tokens: null, cached_input_tokens: null, output_tokens: null, tool_calls: null, skill_loads: [], lane_identities: [], empty_waits: null, source: "unavailable for raw trace"}' >"${variant_dir}/metrics.json"
			fi
			jq -n \
				--arg variant "${variant}" \
				--arg started_at "${started_at}" \
				--arg harness "${HARNESS_NAME}" \
				--arg model "${MODEL_NAME}" \
				--arg reasoning_effort "${REASONING_EFFORT}" \
				--arg tool_profile "${TOOL_PROFILE}" \
				--arg agent_command_label "${AGENT_COMMAND_LABEL}" \
				--arg trace_format "${TRACE_FORMAT}" \
				--argjson repeat "${repeat}" \
				--argjson duration_seconds "$((end_seconds - start_seconds))" \
				--argjson exit_code "${exit_code}" \
				--slurpfile metrics "${variant_dir}/metrics.json" \
				'{side: $variant, variant: $variant, repeat: $repeat, started_at: $started_at, harness: $harness, model: $model, reasoning_effort: $reasoning_effort, tool_profile: $tool_profile, agent_command_label: $agent_command_label, trace_format: $trace_format, duration_seconds: $duration_seconds, exit_code: $exit_code, metrics: $metrics[0], behavior_verdict: "ungraded"}' \
				>"${variant_dir}/run.json"
			if (( exit_code != 0 )); then
				failures=$((failures + 1))
			fi
			rm -rf "${workdir}"
		done
		repeat=$((repeat + 1))
	done
done <<<"${case_ids}"

jq -n \
	--arg baseline_ref "${BASELINE_REF}" \
	--arg candidate_source "${CANDIDATE_SOURCE}" \
	--arg output "${OUTPUT_ROOT}" \
	--arg harness "${HARNESS_NAME}" \
	--arg model "${MODEL_NAME}" \
	--arg reasoning_effort "${REASONING_EFFORT}" \
	--arg tool_profile "${TOOL_PROFILE}" \
	--arg agent_command_label "${AGENT_COMMAND_LABEL}" \
	--arg trace_format "${TRACE_FORMAT}" \
	--arg variants "${RUN_VARIANTS}" \
	--argjson repeats "${REPEAT_COUNT}" \
	--argjson command_failures "${failures}" \
	'{baseline_ref: $baseline_ref, candidate_source: $candidate_source, output: $output, harness: $harness, model: $model, reasoning_effort: $reasoning_effort, tool_profile: $tool_profile, agent_command_label: $agent_command_label, trace_format: $trace_format, variants: ($variants | split(",")), repeats: $repeats, command_failures: $command_failures, behavior_verdict: "ungraded"}' \
	>"${OUTPUT_ROOT}/summary.json"

printf 'instruction eval packets: %s\n' "${OUTPUT_ROOT}"
printf 'grade every expectation in case.json before claiming behavior; command failures: %d\n' "${failures}"
(( failures == 0 ))

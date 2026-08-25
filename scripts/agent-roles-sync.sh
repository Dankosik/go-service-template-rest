#!/usr/bin/env bash
# Generate harness-specific role carriers from canonical role sources.
set -euo pipefail

usage() {
	cat <<'EOF'
usage:
  agent-roles-sync.sh --preflight [--repo <repository>]
  agent-roles-sync.sh --apply     [--repo <repository>]
  agent-roles-sync.sh --check     [--repo <repository>]

  --preflight  validate source and generated path shapes
  --apply      regenerate Codex, Claude, Qwen, Grok, Cursor, and OpenCode role carriers
  --check      verify byte-stable generated carriers without changing files
  --repo       repository root (default: current working directory)
EOF
}

fail() {
	printf 'agent roles: %s\n' "$1" >&2
	exit 1
}

# shellcheck source=scripts/lib/sync-cli.sh
source "$(dirname -- "${BASH_SOURCE[0]}")/lib/sync-cli.sh"
sync_cli_parse "preflight apply check" "$@"
mode="${SYNC_MODE}"
repo="${SYNC_REPO}"

sources="${repo}/.agents/roles"
classes="${repo}/.agents/role-classes"

for path in \
	"${repo}/.agents" \
	"${sources}" \
	"${repo}/.agents/role-classes" \
	"${repo}/.codex" \
	"${repo}/.codex/agents" \
	"${repo}/.claude" \
	"${repo}/.claude/agents" \
	"${repo}/.qwen" \
	"${repo}/.qwen/agents" \
	"${repo}/.grok" \
	"${repo}/.grok/agents" \
	"${repo}/.grok/roles" \
	"${repo}/.cursor" \
	"${repo}/.cursor/agents" \
	"${repo}/.opencode" \
	"${repo}/.opencode/agents"; do
	[[ ! -L "${path}" ]] || fail "${path#"${repo}/"} is a symlink"
done
[[ "${mode}" != "preflight" ]] || exit 0
[[ -d "${sources}" ]] || fail ".agents/roles is missing"
source_link=$(find "${sources}" "${repo}/.agents/role-classes" -type l -print -quit 2>/dev/null || true)
[[ -z "${source_link}" ]] || fail "${source_link#"${repo}/"} is a symlink"

read_strings() {
	awk '
		BEGIN {
			order[1] = "name"
			order[2] = "description"
			order[3] = "class"
			order[4] = "claude_model"
			order[5] = "qwen_model"
			order[6] = "cursor_model"
			order[7] = "grok_model"
			order[8] = "grok_effort"
			order[9] = "output_schema"
			for (i = 1; i <= 9; i++) wanted[order[i]] = 1
		}
		{
			key = $1
			prefix = key " = \""
			if (key in wanted && index($0, prefix) == 1 && substr($0, length($0), 1) == "\"") {
				values[key] = substr($0, length(prefix) + 1, length($0) - length(prefix) - 1)
				seen[key]++
			}
		}
		END {
			for (key in seen) if (seen[key] != 1) exit 2
			for (i = 1; i <= 9; i++) print values[order[i]]
		}
	' "$1"
}

read_body() {
	awk '
		$0 == "instructions = \"\"\"" { body = 1; next }
		body && $0 == "\"\"\"" { body = 0; found = 1; next }
		body { print }
		END { if (!found) exit 1 }
	' "$1"
}

validate_name() {
	local value="$1" file="$2"
	case "${value}" in
	"" | [!a-z0-9]* | *[!a-z0-9-]* | *-) fail "${file#"${repo}/"} has an unsafe role name ${value:-<empty>}" ;;
	esac
}

validate_model() {
	local key="$1" value="$2" file="$3"
	[[ -z "${value}" ]] && return 0
	printf '%s\n' "${value}" | LC_ALL=C grep -Eq '^[A-Za-z0-9][][A-Za-z0-9._/:=-]*$' ||
		fail "${file#"${repo}/"} has an unsafe ${key} value"
}

validate_description() {
	local value="$1" file="$2"
	case "${value}" in
	*$'\n'* | *\"* | *\\*) fail "${file#"${repo}/"} description contains unsupported quoting" ;;
	esac
}

is_grok_session_agent() {
	case "$1" in
	orchestrator.md | acceptance-unit-lead.md) return 0 ;;
	*) return 1 ;;
	esac
}

is_cursor_session_agent() {
	case "$1" in
	acceptance-unit-lead.md) return 0 ;;
	*) return 1 ;;
	esac
}

is_opencode_session_agent() {
	case "$1" in
	orchestrator.md | acceptance-unit-lead.md) return 0 ;;
	*) return 1 ;;
	esac
}

tmp=$(mktemp -d "${TMPDIR:-/tmp}/agent-roles.XXXXXX")
trap 'rm -rf -- "${tmp}"' EXIT
mkdir -p "${tmp}/codex" "${tmp}/claude" "${tmp}/qwen" "${tmp}/grok" "${tmp}/grok-roles" "${tmp}/cursor" "${tmp}/opencode"

role_names=()
shopt -s nullglob
for source_file in "${sources}"/*.toml; do
	metadata=()
	while IFS= read -r value; do metadata+=("${value}"); done < <(read_strings "${source_file}")
	((${#metadata[@]} == 9)) || fail "${source_file#"${repo}/"} has duplicate role metadata"
	name=${metadata[0]}
	description=${metadata[1]}
	class=${metadata[2]}
	claude_model=${metadata[3]}
	qwen_model=${metadata[4]}
	cursor_model=${metadata[5]}
	grok_model=${metadata[6]}
	grok_effort=${metadata[7]}
	output_schema=${metadata[8]}
	body=$(read_body "${source_file}") || fail "${source_file#"${repo}/"} has no closed instructions block"

	validate_name "${name}" "${source_file}"
	expected_name=${source_file##*/}
	expected_name=${expected_name%.toml}
	[[ "${name}" == "${expected_name}" ]] ||
		fail "${source_file#"${repo}/"} name must match its filename"
	[[ -n "${description}" ]] || fail "${source_file#"${repo}/"} has no description"
	validate_description "${description}" "${source_file}"
	validate_model claude_model "${claude_model}" "${source_file}"
	validate_model qwen_model "${qwen_model}" "${source_file}"
	validate_model cursor_model "${cursor_model}" "${source_file}"
	validate_model grok_model "${grok_model}" "${source_file}"
	case "${class}" in
	read-only-specialist)
		class_file="${classes}/read-only-specialist.md"
		fallback_file="${classes}/read-only-specialist-fallback.md"
		sandbox_mode="read-only"
		cursor_readonly="true"
		claude_tools="Read, Grep, Glob, Bash"
		qwen_tools=$'  - read_file\n  - grep_search\n  - glob\n  - list_directory\n  - run_shell_command'
		grok_permission_mode="bypassPermissions"
		grok_capability="read-only"
		;;
	mutable-worker)
		class_file="${classes}/mutable-worker.md"
		fallback_file="${classes}/mutable-worker-fallback.md"
		sandbox_mode="workspace-write"
		cursor_readonly="false"
		claude_tools="Read, Grep, Glob, Bash, Edit, Write"
		qwen_tools=$'  - read_file\n  - grep_search\n  - glob\n  - list_directory\n  - run_shell_command\n  - write_file\n  - edit'
		grok_permission_mode="bypassPermissions"
		grok_capability="all"
		;;
	*) fail "${source_file#"${repo}/"} has unsupported class ${class}" ;;
	esac
	[[ -f "${class_file}" ]] || fail "${class_file#"${repo}/"} is missing"
	[[ -f "${fallback_file}" ]] || fail "${fallback_file#"${repo}/"} is missing"
	common=$(<"${class_file}")
	fallback=$(<"${fallback_file}")
	[[ -n "${claude_model}" ]] || fail "${source_file#"${repo}/"} has no Claude model"
	[[ -n "${cursor_model}" ]] || fail "${source_file#"${repo}/"} has no Cursor model"
	[[ -n "${grok_model}" ]] || fail "${source_file#"${repo}/"} has no Grok model"
	case "${grok_effort}" in
	inherit | low | medium | high | xhigh) ;;
	*) fail "${source_file#"${repo}/"} has unsupported grok_effort ${grok_effort:-<empty>}" ;;
	esac
	case "${output_schema}" in
	lane-result-v1) schema_line="" ;;
	decision-result-v1)
		schema_line=$'Return \x60docs/spec-first-workflow/interfaces/decision-result-v1.md\x60.'
		;;
	review-result-v1)
		schema_line=$'Return \x60docs/spec-first-workflow/interfaces/review-result-v1.md\x60.'
		;;
	delegated-result-v1)
		schema_line=$'Return \x60docs/spec-first-workflow/interfaces/delegated-result-v1.md\x60.'
		;;
	*) fail "${source_file#"${repo}/"} has unsupported output schema ${output_schema}" ;;
	esac
	role_names+=("${name}")

	{
		printf 'name = "%s"\n' "${name}"
		printf 'description = "%s"\n' "${description}"
		printf 'sandbox_mode = "%s"\n' "${sandbox_mode}"
		printf '\ndeveloper_instructions = """\n%s\n' "${common}"
		[[ -z "${schema_line}" ]] || printf '\n%s\n' "${schema_line}"
		printf '\n%s\n"""\n' "${body}"
	} >"${tmp}/codex/${name}.toml"

	{
		printf '%s\n' '---'
		printf 'name: %s\n' "${name}"
		printf 'description: "%s"\n' "${description}"
		printf 'tools: %s\n' "${claude_tools}"
		printf 'model: %s\n' "${claude_model}"
		printf '%s\n\n' '---'
		printf '%s\n\n%s\n' "${common}" "${fallback}"
		[[ -z "${schema_line}" ]] || printf '\n%s\n' "${schema_line}"
		printf '\n%s\n' "${body}"
	} >"${tmp}/claude/${name}.md"

	{
		printf '%s\n' '---'
		printf 'name: %s\n' "${name}"
		printf 'description: "%s"\n' "${description}"
		[[ -z "${qwen_model}" ]] || printf 'model: %s\n' "${qwen_model}"
		printf '%s\n' 'tools:' "${qwen_tools}" '---' ''
		printf '%s\n\n%s\n' "${common}" "${fallback}"
		[[ -z "${schema_line}" ]] || printf '\n%s\n' "${schema_line}"
		printf '\n%s\n' "${body}"
	} >"${tmp}/qwen/${name}.md"

	{
		printf '%s\n' '---'
		printf 'name: %s\n' "${name}"
		printf 'description: "%s"\n' "${description}"
		[[ "${grok_model}" == inherit ]] || printf 'model: %s\n' "${grok_model}"
		printf 'permission_mode: %s\n' "${grok_permission_mode}"
		printf 'agents_md: true\n'
		printf '%s\n\n' '---'
		printf '%s\n' "${common}"
		if [[ "${class}" == mutable-worker ]]; then
			printf '\n%s\n' "${fallback}"
		fi
		[[ -z "${schema_line}" ]] || printf '\n%s\n' "${schema_line}"
		printf '\n%s\n' "${body}"
	} >"${tmp}/grok/${name}.md"

	{
		printf 'description = "%s"\n' "${description}"
		printf 'default_capability_mode = "%s"\n' "${grok_capability}"
		[[ "${grok_model}" == inherit ]] || printf 'model = "%s"\n' "${grok_model}"
		[[ "${grok_effort}" == inherit ]] || printf 'reasoning_effort = "%s"\n' "${grok_effort}"
	} >"${tmp}/grok-roles/${name}.toml"

	{
		printf '%s\n' '---'
		printf 'name: %s\n' "${name}"
		printf 'description: "%s"\n' "${description}"
		printf 'model: %s\n' "${cursor_model}"
		printf 'readonly: %s\n' "${cursor_readonly}"
		printf '%s\n\n' '---'
		printf '%s\n\n%s\n' "${common}" "${fallback}"
		[[ -z "${schema_line}" ]] || printf '\n%s\n' "${schema_line}"
		printf '\n%s\n' "${body}"
	} >"${tmp}/cursor/${name}.md"

	{
		printf '%s\n' '---'
		printf 'description: "%s"\n' "${description}"
		printf 'mode: subagent\n'
		printf 'hidden: true\n'
		# OpenCode applies variant only when the agent pins a model.
		if [[ "${grok_model}" != inherit ]]; then
			printf 'model: xai/%s\n' "${grok_model}"
		elif [[ "${grok_effort}" != inherit ]]; then
			printf 'model: xai/grok-4.6\n'
		fi
		[[ "${grok_effort}" == inherit ]] || printf 'variant: %s\n' "${grok_effort}"
		printf '%s\n' 'permission:'
		if [[ "${class}" == read-only-specialist ]]; then
			printf '%s\n' '  edit: deny'
		fi
		printf '%s\n' '  task: deny' '  question: deny' '---' ''
		printf '%s\n\n%s\n' "${common}" "${fallback}"
		[[ -z "${schema_line}" ]] || printf '\n%s\n' "${schema_line}"
		printf '\n%s\n' "${body}"
	} >"${tmp}/opencode/${name}.md"
done
shopt -u nullglob
((${#role_names[@]} > 0)) || fail ".agents/roles contains no canonical role files"

set_generated_path() {
	case "$1" in
	codex) generated_target="${repo}/.codex/agents" ;;
	claude) generated_target="${repo}/.claude/agents" ;;
	qwen) generated_target="${repo}/.qwen/agents" ;;
	grok) generated_target="${repo}/.grok/agents" ;;
	grok-roles) generated_target="${repo}/.grok/roles" ;;
	cursor) generated_target="${repo}/.cursor/agents" ;;
	opencode) generated_target="${repo}/.opencode/agents" ;;
	esac
}

check_extra() {
	local harness="$1" extension="$2" target file role
	set_generated_path "${harness}"
	target=${generated_target}
	[[ -d "${target}" ]] || fail "${target#"${repo}/"} is missing"
	shopt -s nullglob
	for file in "${target}"/*."${extension}"; do
		role=${file##*/}
		role=${role%."${extension}"}
		if [[ "${harness}" == grok ]] && is_grok_session_agent "${role}.${extension}"; then
			continue
		fi
		if [[ "${harness}" == cursor ]] && is_cursor_session_agent "${role}.${extension}"; then
			continue
		fi
		if [[ "${harness}" == opencode ]] && is_opencode_session_agent "${role}.${extension}"; then
			continue
		fi
		[[ -f "${tmp}/${harness}/${role}.${extension}" ]] ||
			fail "${file#"${repo}/"} has no canonical role source"
	done
	shopt -u nullglob
}

sync_generated() {
	local harness="$1" extension="$2" target file role
	set_generated_path "${harness}"
	target=${generated_target}
	mkdir -p "${target}"
	shopt -s nullglob
	for file in "${target}"/*."${extension}"; do
		role=${file##*/}
		role=${role%."${extension}"}
		if [[ "${harness}" == grok ]] && is_grok_session_agent "${role}.${extension}"; then
			continue
		fi
		if [[ "${harness}" == cursor ]] && is_cursor_session_agent "${role}.${extension}"; then
			continue
		fi
		if [[ "${harness}" == opencode ]] && is_opencode_session_agent "${role}.${extension}"; then
			continue
		fi
		[[ -f "${tmp}/${harness}/${role}.${extension}" ]] || rm -f -- "${file}"
	done
	for file in "${tmp}/${harness}"/*."${extension}"; do
		cp "${file}" "${target}/${file##*/}"
	done
	shopt -u nullglob
}

check_generated() {
	local harness="$1" extension="$2" target role actual expected
	check_extra "${harness}" "${extension}"
	set_generated_path "${harness}"
	target=${generated_target}
	for role in "${role_names[@]}"; do
		actual="${target}/${role}.${extension}"
		expected="${tmp}/${harness}/${role}.${extension}"
		[[ -f "${actual}" ]] || fail "${actual#"${repo}/"} is missing"
		if ! cmp -s "${expected}" "${actual}"; then
			diff -u "${actual}" "${expected}" >&2 || true
			fail "${actual#"${repo}/"} is stale; run scripts/agent-roles-sync.sh --apply"
		fi
	done
}

case "${mode}" in
apply)
	sync_generated codex toml
	sync_generated claude md
	sync_generated qwen md
	sync_generated grok md
	sync_generated grok-roles toml
	sync_generated cursor md
	sync_generated opencode md
	printf 'agent roles: %d role carriers generated for 6 harnesses\n' "${#role_names[@]}"
	;;
check)
	check_generated codex toml
	check_generated claude md
	check_generated qwen md
	check_generated grok md
	check_generated grok-roles toml
	check_generated cursor md
	check_generated opencode md
	printf 'agent roles: %d canonical roles current for 6 harnesses\n' "${#role_names[@]}"
	;;
esac

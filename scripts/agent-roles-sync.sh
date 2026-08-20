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
  --apply      regenerate Codex, Claude, and Qwen role carriers
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
	"${repo}/.qwen/agents"; do
	[[ ! -L "${path}" ]] || fail "${path#"${repo}/"} is a symlink"
done
[[ "${mode}" != "preflight" ]] || exit 0
[[ -d "${sources}" ]] || fail ".agents/roles is missing"
source_link=$(find "${sources}" "${repo}/.agents/role-classes" -type l -print -quit 2>/dev/null || true)
[[ -z "${source_link}" ]] || fail "${source_link#"${repo}/"} is a symlink"

read_string() {
	local key="$1" file="$2"
	sed -n "s/^${key} = \"\(.*\)\"$/\1/p" "${file}"
}

read_raw() {
	local key="$1" file="$2"
	sed -n "s/^${key} = //p" "${file}"
}

read_body() {
	awk '
		$0 == "instructions = \"\"\"" { body = 1; next }
		body && $0 == "\"\"\"" { body = 0; found = 1; next }
		body { print }
		END { if (!found) exit 1 }
	' "$1"
}

tmp=$(mktemp -d "${TMPDIR:-/tmp}/agent-roles.XXXXXX")
trap 'rm -rf -- "${tmp}"' EXIT
mkdir -p "${tmp}/codex" "${tmp}/claude" "${tmp}/qwen"

role_names=()
shopt -s nullglob
for source_file in "${sources}"/*.toml; do
	name=$(read_string name "${source_file}")
	description=$(read_string description "${source_file}")
	class=$(read_string class "${source_file}")
	claude_model=$(read_string claude_model "${source_file}")
	qwen_model=$(read_string qwen_model "${source_file}")
	output_schema=$(read_string output_schema "${source_file}")
	nicknames=$(read_raw nickname_candidates "${source_file}")
	body=$(read_body "${source_file}") || fail "${source_file#"${repo}/"} has no closed instructions block"

	[[ -n "${name}" && "${name}" == "$(basename "${source_file}" .toml)" ]] ||
		fail "${source_file#"${repo}/"} name must match its filename"
	[[ -n "${description}" ]] || fail "${source_file#"${repo}/"} has no description"
	case "${class}" in
	read-only-specialist)
		class_file="${classes}/read-only-specialist.md"
		fallback_file="${classes}/read-only-specialist-fallback.md"
		sandbox_mode="read-only"
		claude_tools="Read, Grep, Glob, Bash"
		qwen_tools=$'  - read_file\n  - grep_search\n  - glob\n  - list_directory\n  - run_shell_command'
		;;
	mutable-worker)
		class_file="${classes}/mutable-worker.md"
		fallback_file="${classes}/mutable-worker-fallback.md"
		sandbox_mode="workspace-write"
		claude_tools="Read, Grep, Glob, Bash, Edit, Write"
		qwen_tools=$'  - read_file\n  - grep_search\n  - glob\n  - list_directory\n  - run_shell_command\n  - write_file\n  - edit'
		;;
	*) fail "${source_file#"${repo}/"} has unsupported class ${class}" ;;
	esac
	[[ -f "${class_file}" ]] || fail "${class_file#"${repo}/"} is missing"
	[[ -f "${fallback_file}" ]] || fail "${fallback_file#"${repo}/"} is missing"
	common=$(<"${class_file}")
	fallback=$(<"${fallback_file}")
	[[ -n "${claude_model}" ]] || fail "${source_file#"${repo}/"} has no Claude model"
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
		[[ -z "${nicknames}" ]] || printf 'nickname_candidates = %s\n' "${nicknames}"
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
done
shopt -u nullglob
((${#role_names[@]} > 0)) || fail ".agents/roles contains no canonical role files"

generated_path() {
	case "$1" in
	codex) printf '%s/.codex/agents' "${repo}" ;;
	claude) printf '%s/.claude/agents' "${repo}" ;;
	qwen) printf '%s/.qwen/agents' "${repo}" ;;
	esac
}

check_extra() {
	local harness="$1" extension="$2" target file role
	target=$(generated_path "${harness}")
	[[ -d "${target}" ]] || fail "${target#"${repo}/"} is missing"
	shopt -s nullglob
	for file in "${target}"/*."${extension}"; do
		role=$(basename "${file}" ".${extension}")
		[[ -f "${tmp}/${harness}/${role}.${extension}" ]] ||
			fail "${file#"${repo}/"} has no canonical role source"
	done
	shopt -u nullglob
}

case "${mode}" in
apply)
	for harness in codex claude qwen; do
		target=$(generated_path "${harness}")
		mkdir -p "${target}"
		case "${harness}" in codex) extension=toml ;; *) extension=md ;; esac
		shopt -s nullglob
		for file in "${target}"/*."${extension}"; do
			role=$(basename "${file}" ".${extension}")
			[[ -f "${tmp}/${harness}/${role}.${extension}" ]] || rm -f -- "${file}"
		done
		for file in "${tmp}/${harness}"/*."${extension}"; do
			cp "${file}" "${target}/$(basename "${file}")"
		done
		shopt -u nullglob
	done
	printf 'agent roles: %d role carriers generated for 3 harnesses\n' "${#role_names[@]}"
	;;
check)
	for harness in codex claude qwen; do
		case "${harness}" in codex) extension=toml ;; *) extension=md ;; esac
		check_extra "${harness}" "${extension}"
		target=$(generated_path "${harness}")
		for role in "${role_names[@]}"; do
			actual="${target}/${role}.${extension}"
			expected="${tmp}/${harness}/${role}.${extension}"
			[[ -f "${actual}" ]] || fail "${actual#"${repo}/"} is missing"
			if ! cmp -s "${expected}" "${actual}"; then
				diff -u "${actual}" "${expected}" >&2 || true
				fail "${actual#"${repo}/"} is stale; run scripts/agent-roles-sync.sh --apply"
			fi
		done
	done
	printf 'agent roles: %d canonical roles current for 3 harnesses\n' "${#role_names[@]}"
	;;
esac

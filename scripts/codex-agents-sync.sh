#!/usr/bin/env bash
# Generate Codex's portable project runtime and custom-agent registry from
# template-owned sources. Machine-specific settings stay in user config.
set -euo pipefail

runtime_start_marker="# template-owned-codex-runtime:start"
runtime_end_marker="# template-owned-codex-runtime:end"
start_marker="# template-owned-codex-agents:start"
end_marker="# template-owned-codex-agents:end"

usage() {
	cat <<'EOF'
usage:
  codex-agents-sync.sh --preflight [--repo <repository>]
  codex-agents-sync.sh --apply [--repo <repository>]
  codex-agents-sync.sh --check [--repo <repository>]

  --preflight  refuse target path and marker shapes that cannot be rebuilt safely
  --apply  replace the managed project runtime and role registry in .codex/config.toml
  --check  verify portable runtime and exact role coverage without changing files
  --repo   repository root (default: current working directory)
EOF
}

fail() {
	printf 'codex project: %s\n' "$1" >&2
	exit 1
}

# shellcheck source=scripts/lib/sync-cli.sh
source "$(dirname -- "${BASH_SOURCE[0]}")/lib/sync-cli.sh"
sync_cli_parse "preflight apply check" "$@"
mode="${SYNC_MODE}"
repo="${SYNC_REPO}"

roles_root="${repo}/.codex/agents"
config="${repo}/.codex/config.toml"
portable_source="${repo}/.agents/codex-project.toml"

[[ ! -L "${repo}/.codex" ]] || fail ".codex is a symlink"
[[ ! -L "${roles_root}" ]] || fail ".codex/agents is a symlink"
[[ ! -L "${config}" ]] || fail ".codex/config.toml is a symlink"
[[ ! -L "${portable_source}" ]] || fail ".agents/codex-project.toml is a symlink"
if [[ -e "${repo}/.codex" && ! -d "${repo}/.codex" ]]; then
	fail ".codex is not a directory"
fi
if [[ -e "${roles_root}" && ! -d "${roles_root}" ]]; then
	fail ".codex/agents is not a directory"
fi
if [[ -e "${config}" && ! -f "${config}" ]]; then
	fail ".codex/config.toml is not a regular file"
fi

validate_portable_source() {
	local unexpected_tables
	grep -Eq '^developer_instructions[[:space:]]*=' "${portable_source}" ||
		fail ".agents/codex-project.toml has no developer_instructions"
	grep -Fxq '[agents]' "${portable_source}" ||
		fail ".agents/codex-project.toml has no [agents] table"
	grep -Eq '^max_concurrent_threads_per_session[[:space:]]*=' "${portable_source}" ||
		fail ".agents/codex-project.toml has no concurrency ceiling"
	unexpected_tables=$(sed -n 's/^\(\[[^]]*\]\)$/\1/p' "${portable_source}" |
		grep -Fvx '[agents]' || true)
	[[ -z "${unexpected_tables}" ]] ||
		fail ".agents/codex-project.toml may contain only top-level keys and [agents]"
	grep -Fq "${runtime_start_marker}" "${portable_source}" &&
		fail ".agents/codex-project.toml contains a generated runtime marker"
	grep -Fq "${runtime_end_marker}" "${portable_source}" &&
		fail ".agents/codex-project.toml contains a generated runtime marker"
	return 0
}

validate_config_shape() {
	local runtime_start_count=0 runtime_end_count=0 start_count=0 end_count=0
	[[ -f "${config}" ]] || return 0
	runtime_start_count=$(grep -Fxc "${runtime_start_marker}" "${config}" || true)
	runtime_end_count=$(grep -Fxc "${runtime_end_marker}" "${config}" || true)
	start_count=$(grep -Fxc "${start_marker}" "${config}" || true)
	end_count=$(grep -Fxc "${end_marker}" "${config}" || true)
	[[ "${runtime_start_count}" -le 1 && "${runtime_end_count}" -le 1 && "${runtime_start_count}" == "${runtime_end_count}" ]] ||
		fail ".codex/config.toml has an invalid managed runtime marker pair"
	[[ "${start_count}" -le 1 && "${end_count}" -le 1 && "${start_count}" == "${end_count}" ]] ||
		fail ".codex/config.toml has an invalid managed registry marker pair"
}

validate_config_shape
[[ "${mode}" != "preflight" ]] || exit 0
[[ -f "${portable_source}" ]] || fail ".agents/codex-project.toml is missing"
validate_portable_source
[[ -d "${roles_root}" ]] || fail ".codex/agents is missing"

role_names=()
shopt -s nullglob
for role_file in "${roles_root}"/*.toml; do
	role_names+=("$(basename "${role_file}" .toml)")
done
shopt -u nullglob
((${#role_names[@]} > 0)) || fail ".codex/agents contains no role files"

render_registry() {
	local role
	printf '%s\n' "${start_marker}"
	printf '%s\n' '# Registry compatibility: agents.<name>.config_file entries are required by Codex.'
	for role in "${role_names[@]}"; do
		printf '[agents.%s]\n' "${role}"
		printf 'config_file = "agents/%s.toml"\n\n' "${role}"
	done
	printf '%s\n' "${end_marker}"
}

expected_config() {
	local output="$1"
	{
		printf '%s\n' "${runtime_start_marker}"
		cat "${portable_source}"
		printf '%s\n\n' "${runtime_end_marker}"
		render_registry
	} >"${output}"
}

expected=$(mktemp "${TMPDIR:-/tmp}/codex-agents-config.XXXXXX")
trap 'rm -f -- "${expected}"' EXIT
expected_config "${expected}"

case "${mode}" in
apply)
	mkdir -p "$(dirname "${config}")"
	cp "${expected}" "${config}"
	printf 'codex project: portable runtime and %d roles registered\n' "${#role_names[@]}"
	;;
check)
	[[ -f "${config}" ]] || fail ".codex/config.toml is missing"
	if ! cmp -s "${expected}" "${config}"; then
		diff -u "${config}" "${expected}" >&2 || true
		fail ".codex/config.toml project runtime or role registry is stale; run scripts/codex-agents-sync.sh --apply"
	fi
	printf 'codex project: portable runtime and %d role registrations current\n' "${#role_names[@]}"
	;;
esac

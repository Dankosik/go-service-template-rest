#!/usr/bin/env bash
# Keep Codex's explicit custom-agent registry aligned with template-owned role
# files while preserving repository-specific config outside the managed block.
set -euo pipefail

start_marker="# template-owned-codex-agents:start"
end_marker="# template-owned-codex-agents:end"

usage() {
	cat <<'EOF'
usage:
  codex-agents-sync.sh --apply [--repo <repository>]
  codex-agents-sync.sh --check [--repo <repository>]

  --apply  replace the managed role registry in .codex/config.toml
  --check  verify exact role coverage without changing files
  --repo   repository root (default: current working directory)
EOF
}

fail() {
	printf 'codex agents: %s\n' "$1" >&2
	exit 1
}

# shellcheck source=scripts/lib/sync-cli.sh
source "$(dirname -- "${BASH_SOURCE[0]}")/lib/sync-cli.sh"
sync_cli_parse "apply check" "$@"
mode="${SYNC_MODE}"
repo="${SYNC_REPO}"

roles_root="${repo}/.codex/agents"
config="${repo}/.codex/config.toml"

[[ ! -L "${repo}/.codex" ]] || fail ".codex is a symlink"
[[ ! -L "${roles_root}" ]] || fail ".codex/agents is a symlink"
[[ ! -L "${config}" ]] || fail ".codex/config.toml is a symlink"
[[ -d "${roles_root}" ]] || fail ".codex/agents is missing"

role_names=()
shopt -s nullglob
for role_file in "${roles_root}"/*.toml; do
	role_names+=("$(basename "${role_file}" .toml)")
done
shopt -u nullglob
((${#role_names[@]} > 0)) || fail ".codex/agents contains no role files"

role_csv=""
for role in "${role_names[@]}"; do
	[[ -z "${role_csv}" ]] || role_csv+=","
	role_csv+="${role}"
done

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
	local output="$1" stripped trimmed start_count=0 end_count=0
	stripped=$(mktemp "${TMPDIR:-/tmp}/codex-agents-stripped.XXXXXX")
	trimmed=$(mktemp "${TMPDIR:-/tmp}/codex-agents-trimmed.XXXXXX")

	if [[ -f "${config}" ]]; then
		start_count=$(grep -Fxc "${start_marker}" "${config}" || true)
		end_count=$(grep -Fxc "${end_marker}" "${config}" || true)
		[[ "${start_count}" -le 1 && "${end_count}" -le 1 && "${start_count}" == "${end_count}" ]] ||
			fail ".codex/config.toml has an invalid managed registry marker pair"
		awk -v roles="${role_csv}" -v start="${start_marker}" -v end="${end_marker}" '
			BEGIN {
				count = split(roles, names, ",")
				for (i = 1; i <= count; i++) known["agents." names[i]] = 1
			}
			$0 == start { in_managed = 1; next }
			$0 == end { in_managed = 0; next }
			in_managed { next }
			$0 == "# Registry compatibility: agents.<name>.config_file entries are required by Codex." { next }
			/^\[[^]]+\]$/ {
				section = substr($0, 2, length($0) - 2)
				in_known = known[section]
				if (in_known) next
			}
			!in_known { print }
		' "${config}" >"${stripped}"
	else
		printf '[agents]\nmax_depth = 1\n' >"${stripped}"
	fi

	# Drop only trailing blank lines so repeated applies are byte-stable.
	awk '
		NF {
			for (i = 0; i < blanks; i++) print ""
			blanks = 0
			print
			next
		}
		{ blanks++ }
	' "${stripped}" >"${trimmed}"
	cp "${trimmed}" "${output}"
	printf '\n' >>"${output}"
	render_registry >>"${output}"
	rm -f -- "${stripped}" "${trimmed}"
}

expected=$(mktemp "${TMPDIR:-/tmp}/codex-agents-config.XXXXXX")
trap 'rm -f -- "${expected}"' EXIT
expected_config "${expected}"

case "${mode}" in
apply)
	mkdir -p "$(dirname "${config}")"
	cp "${expected}" "${config}"
	printf 'codex agents: %d roles registered\n' "${#role_names[@]}"
	;;
check)
	[[ -f "${config}" ]] || fail ".codex/config.toml is missing"
	if ! cmp -s "${expected}" "${config}"; then
		diff -u "${config}" "${expected}" >&2 || true
		fail ".codex/config.toml role registry is stale; run scripts/codex-agents-sync.sh --apply"
	fi
	printf 'codex agents: %d role registrations current\n' "${#role_names[@]}"
	;;
esac
